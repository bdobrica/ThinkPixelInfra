"""
FastAPI server module for Snowflake Arctic embedding model inference.

This module provides a FastAPI-based HTTP server that exposes endpoints for
text embedding generation using the Snowflake Arctic model. It handles text
preprocessing, communicates with the inference backend via ZeroMQ, and returns
structured embedding results.

The server provides two main endpoints:
- /infer: Generate embeddings for text items
- /ping: Health check endpoint

The inference pipeline:
1. Receives text items via HTTP POST
2. Forwards requests to ZeroMQ inference workers
3. Returns dense and sparse embeddings with metadata

Models:
    Metadata: Metadata structure for text items
    TextItem: Input text item with metadata
    InferenceRequest: Request containing list of text items
    EmbeddingsItem: Output embedding item with vectors and metadata
    InferenceResponse: Response containing embedding results and latency
    SuccessResponse: Simple success response for health checks

Functions:
    fastapi_server: Main function to start the FastAPI server
    infer: Endpoint for text embedding inference
    ping: Health check endpoint
    create_zmq_context: Initialize ZeroMQ context
    ping_zmq_request: Helper for ZeroMQ health checks
"""

import json
import logging
import os
import time
from functools import lru_cache
from typing import Dict, List

import uvicorn
import zmq
import zmq.asyncio
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .config import LOG_LEVEL, MODEL_HTTP_PORT, MODEL_ZMQ_CLIENT_ADDR

# Setup logging
logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)

# Track the status of model readiness
ready = False


class Metadata(BaseModel):
    """
    Metadata structure for text items containing ID and additional fields.

    Attributes:
        id (int): Unique identifier for the text item
        extra (dict): Additional metadata fields (default: empty dict)
    """

    id: int
    extra: dict = {}


class TextItem(BaseModel):
    """
    Input text item for embedding generation.

    Attributes:
        text (str): Text content to be embedded
        metadata (Metadata): Associated metadata including ID and extra fields
    """

    text: str
    metadata: Metadata


class InferenceRequest(BaseModel):
    """
    Request payload for text embedding inference.

    Attributes:
        text_items (List[TextItem]): List of text items to be embedded
    """

    text_items: List[TextItem]


class EmbeddingsItem(BaseModel):
    """
    Embedding result item containing text, vectors, and metadata.

    This model represents a single embedding result with the original text,
    dense and sparse vector representations, character offset information,
    and associated metadata.

    Attributes:
        text (str): Original text that was embedded
        offset (int): Character offset of this text chunk in the original document
        dense_vector (str): Base64-encoded dense embedding vector
        sparse_vector (Dict[str, str]): Sparse vector representation as hash->weight mapping
        metadata (Metadata): Associated metadata including ID and extra fields
    """

    text: str
    offset: int
    dense_vector: str
    sparse_vector: Dict[str, str]
    metadata: Metadata


class InferenceResponse(BaseModel):
    """
    Response payload containing embedding results and performance metrics.

    Attributes:
        results (List[EmbeddingsItem]): List of embedding results for each input text
        latency (float): Request processing time in seconds (default: 0.0)
    """

    results: List[EmbeddingsItem]
    latency: float = 0.0


class SuccessResponse(BaseModel):
    """
    Simple success response for health check endpoints.

    Attributes:
        success (bool): Success status (default: True)
        version (str): Application version from VERSION environment variable
    """

    success: bool = True
    version: str = os.getenv("VERSION", "0.0.0")


@lru_cache(maxsize=None)
def create_zmq_context():
    """
    Initialize and cache the ZeroMQ async context for communication with inference workers.

    This function creates a global ZeroMQ async context that is used throughout
    the application for communicating with the inference backend. The context
    is cached to ensure only one instance exists.

    Global Variables:
        context: ZMQ async context for socket operations
    """
    global context

    # ZMQ Async Context
    context = zmq.asyncio.Context()  # type: ignore
    logger.info("Created ZMQ context.")


async def ping_zmq_request(timeout: float = 1.0):
    """
    Send a health check request to the ZeroMQ inference worker.

    This helper function sends a minimal ping request to the inference backend
    to verify connectivity and worker availability. It's used by the ping endpoint
    to check system health.

    Args:
        timeout (float): Request timeout in seconds (default: 1.0)

    Returns:
        bool: True if ping successful, False otherwise

    Note:
        Creates a temporary REQ socket for the ping operation and closes it
        after receiving the response or timing out.
    """
    global context

    # Create ZMQ socket
    socket: zmq.asyncio.Socket = context.socket(zmq.REQ)
    socket.setsockopt(zmq.LINGER, 0)
    socket.setsockopt(zmq.RCVTIMEO, int(timeout * 1000))
    socket.connect(MODEL_ZMQ_CLIENT_ADDR)

    # Prepare request data
    request_id = b"ping-request-" + os.urandom(4)  # Generate unique request ID
    # Minimal request payload for pinging the model
    request_data = {"text_items": [TextItem(text="ping", metadata=Metadata(id=0)).dict()]}

    message = [request_id, b"", json.dumps(request_data).encode("utf-8")]

    try:
        await socket.send_multipart(message)
        message = await socket.recv_multipart()
        _, response = message[1:3]
        response = json.loads(response.decode("utf-8"))

        socket.close()

        # Consider the request successful if "error" is not in the response
        return "error" not in response
    except Exception as e:
        logger.error(f"ZMQ request failed: {e}")
        socket.close()
        return False


async def infer(request: InferenceRequest) -> InferenceResponse:
    """
    Inference endpoint to generate embeddings for text items.

    This endpoint processes text embedding requests by forwarding them to the
    ZeroMQ inference workers and returning the results. It handles the complete
    pipeline from HTTP request to embedding generation.

    Args:
        request (InferenceRequest): Request containing text items to embed

        Returns:
        InferenceResponse: Response containing:
            - results: List of EmbeddingsItem with text, offset, dense_vector,
                      sparse_vector, and metadata
            - latency: Processing time in seconds    Raises:
        HTTPException: 500 error if inference fails or worker returns error

    Note:
        Each result item now includes the offset field at the top level,
        extracted from the metadata.extra field during postprocessing.
    """
    global context

    # Measure request time
    start_time = time.perf_counter()

    # Create ZMQ socket
    socket: zmq.asyncio.Socket = context.socket(zmq.REQ)
    socket.setsockopt(zmq.LINGER, 0)
    socket.connect(MODEL_ZMQ_CLIENT_ADDR)

    # Prepare request data
    request_id = b"request-" + os.urandom(4)  # Generate unique request ID
    request_data = {"text_items": [item.dict() for item in request.text_items]}

    # Send request to ZMQ Dealer
    message = [request_id, b"", json.dumps(request_data).encode("utf-8")]
    logger.debug("Sending request to ZMQ Dealer: %s", message)
    await socket.send_multipart(message)

    # Wait for response
    logger.debug("Waiting for response...")
    message = await socket.recv_multipart()
    logger.debug("Received response from ZMQ Dealer: %s", message)
    _, response = message[1:3]
    response = json.loads(response.decode("utf-8"))

    socket.close()

    if "error" in response:
        raise HTTPException(status_code=500, detail=response["error"])
    return InferenceResponse(
        results=[EmbeddingsItem(**item) for item in response["results"]],
        latency=time.perf_counter() - start_time,
    )


async def ping() -> SuccessResponse:
    """
    Health check endpoint to verify system availability.

    This endpoint checks the health of the embedding system by attempting
    to communicate with the ZeroMQ inference workers. It implements a simple
    caching mechanism to avoid excessive health checks.

    Returns:
        SuccessResponse: Success response with version information

    Raises:
        HTTPException: 500 error if ping fails within timeout

    Behavior:
        - Returns immediately if already marked as ready
        - Attempts ZMQ ping with 1-second timeout
        - Caches successful ping result to avoid repeated checks
    """
    global ready

    if ready:
        # If already successful, return immediately without additional requests
        return SuccessResponse()

    # Attempt to send ZMQ request until successful or timeout
    timeout = 1.0  # seconds
    start_time = time.perf_counter()

    while time.perf_counter() - start_time < timeout:
        success = await ping_zmq_request(timeout=timeout)
        if success:
            ready = True
            return SuccessResponse()

    # If unsuccessful within the timeout, return an error
    raise HTTPException(status_code=500, detail="Ping failed")


def fastapi_server():
    """
    Start the FastAPI server for Snowflake Arctic embedding inference.

    This function initializes and starts the FastAPI application server with
    the inference and health check endpoints. It sets up the ZeroMQ context
    for backend communication and ensures proper cleanup on shutdown.

    Server Configuration:
        - Host: 0.0.0.0 (all interfaces)
        - Port: Configured via MODEL_HTTP_PORT environment variable
        - Log Level: Configured via LOG_LEVEL environment variable

    Endpoints:
        - POST /infer: Text embedding inference
        - GET /ping: Health check

    Note:
        The server runs until interrupted and properly terminates the ZMQ
        context during shutdown.
    """
    app = FastAPI()
    _ = app.post("/infer")(infer)
    _ = app.get("/ping")(ping)

    create_zmq_context()

    try:
        uvicorn.run(app, host="0.0.0.0", port=MODEL_HTTP_PORT, log_level=LOG_LEVEL)
    finally:
        context.term()
