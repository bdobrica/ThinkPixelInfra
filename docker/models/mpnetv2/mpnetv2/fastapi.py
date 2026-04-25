"""
FastAPI HTTP server for MPNetv2 text embedding service.

This module provides a REST API interface for the MPNetv2 embedding service,
handling HTTP requests and communicating with inference workers via ZMQ.
The server supports text chunking, multi-language processing, and provides
health check endpoints.

Key Features:
- RESTful API with automatic request/response validation
- Asynchronous ZMQ communication with inference workers
- Configurable text chunking with overlap support
- Multi-language text processing capabilities
- Health check endpoint with model readiness verification
- Comprehensive error handling and logging

API Endpoints:
    POST /infer: Generate embeddings for text items with optional chunking
    GET /ping: Health check endpoint for service monitoring

Request/Response Models:
    - InferenceRequest: Text items with optional chunking parameters
    - InferenceResponse: Embeddings with metadata and latency information
    - TextItem: Individual text with metadata
    - EmbeddingsItem: Text chunk with dense vector embedding and metadata

Example:
    >>> import requests
    >>> response = requests.post('http://localhost:8000/infer', json={
    ...     'text_items': [{'text': 'Hello world', 'metadata': {'id': 1}}],
    ...     'chunk_size': 1000,
    ...     'chunk_overlap': 200
    ... })
    >>> embeddings = response.json()['results']
"""

import json
import logging
import os
import time
from functools import lru_cache
from typing import List, Literal

import uvicorn
import zmq
import zmq.asyncio
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .config import (
    LOG_LEVEL,
    MODEL_CHUNK_OVERLAP,
    MODEL_CHUNK_SIZE,
    MODEL_HTTP_PORT,
    MODEL_ZMQ_CLIENT_ADDR,
)

# Setup logging
logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)

# Track the status of model readiness
ready = False


class Metadata(BaseModel):
    """
    Metadata container for text items and embeddings.

    Contains identification and additional context information that gets
    preserved throughout the processing pipeline.

    Attributes:
        id (int): Unique identifier for the text item
        extra (dict): Additional metadata fields, automatically populated with:
            - offset: Character offset in original text (for chunks)
            - language: Detected language code
            - language_model: LanguageModel instance (internal use)
    """

    id: int
    extra: dict = {}


class TextItem(BaseModel):
    """
    Individual text item for embedding generation.

    Represents a single piece of text with associated metadata that will
    be processed for embedding generation. Long texts may be automatically
    split into chunks based on request parameters.

    Attributes:
        text (str): The text content to generate embeddings for
        metadata (Metadata): Associated metadata for the text item
    """

    text: str
    metadata: Metadata


class InferenceRequest(BaseModel):
    """
    Request model for text embedding inference.

    Contains text items and optional chunking parameters for processing.
    Chunking allows long texts to be split into manageable pieces while
    maintaining context through configurable overlap.

    Attributes:
        text_items (List[TextItem]): List of text items to process
        chunk_size (int): Maximum characters per chunk (default from config)
        chunk_overlap (int): Character overlap between chunks (default from config)

    Example:
        >>> request = InferenceRequest(
        ...     text_items=[TextItem(text="Long text...", metadata=Metadata(id=1))],
        ...     chunk_size=1000,
        ...     chunk_overlap=200
        ... )
    """

    text_items: List[TextItem]
    language: str = "auto"
    mode: Literal["search", "store"] = "store"
    chunk_size: int = MODEL_CHUNK_SIZE
    chunk_overlap: int = MODEL_CHUNK_OVERLAP


class EmbeddingsItem(BaseModel):
    """
    Individual embedding result with text chunk and dense vector.

    Represents the embedding output for a single text chunk, including
    the processed text, its position in the original document, and the
    base64-encoded dense vector embedding.

    Attributes:
        text (str): The text chunk that was embedded
        offset (int): Character offset of this chunk in the original text
        dense_vector (str): Base64-encoded dense vector embedding
        metadata (Metadata): Original metadata plus processing information
    """

    text: str
    offset: int
    dense_vector: str
    metadata: Metadata


class InferenceResponse(BaseModel):
    """
    Response model containing embedding results and performance metrics.

    Contains all embedding results for the processed text items along
    with timing information for performance monitoring.

    Attributes:
        results (List[EmbeddingsItem]): List of embedding results
        latency (float): Request processing time in seconds
    """

    results: List[EmbeddingsItem]
    latency: float = 0.0


class SuccessResponse(BaseModel):
    """
    Simple success response for health checks and status endpoints.

    Attributes:
        success (bool): Always True for successful responses
        version (str): Service version from environment variable
    """

    success: bool = True
    version: str = os.getenv("VERSION", "0.0.0")


@lru_cache(maxsize=None)
def create_zmq_context():
    """
    Create and cache ZMQ context for communication with inference workers.

    Uses LRU cache to ensure only one context is created per process,
    which is important for ZMQ resource management and performance.

    Global Variables:
        context: ZMQ async context for socket communication
    """
    global context

    # ZMQ Async Context
    context = zmq.asyncio.Context()  # type: ignore
    logger.info("Created ZMQ context.")


async def ping_zmq_request(timeout: float = 1.0):
    """
    Send a ping request to inference workers to verify service health.

    Sends a minimal inference request with sample texts in multiple languages
    to verify that the inference workers are responsive and the model is loaded.
    This is used by the health check endpoint.

    Args:
        timeout (float): Request timeout in seconds (default: 1.0)

    Returns:
        bool: True if ping successful, False otherwise

    Note:
        Uses sample texts in English, French, German, Spanish, Italian, and
        Romanian to test multi-language support.
    """
    global context

    # Create ZMQ socket
    socket: zmq.asyncio.Socket = context.socket(zmq.REQ)
    socket.setsockopt(zmq.LINGER, 0)
    socket.setsockopt(zmq.RCVTIMEO, int(timeout * 1000))
    socket.connect(MODEL_ZMQ_CLIENT_ADDR)

    # Prepare request data
    request_id = b"ping-request-" + os.urandom(4)  # Generate unique request ID
    # Ping texts, one for every supported language English, French, German, Spanish, Italian, Romaanian
    texts = [
        "The quick brown fox jumps over the lazy dog.",
        "Le renard brun rapide saute par-dessus le chien paresseux.",
        "Der schnelle braune Fuchs springt über den faulen Hund.",
        "El rápido zorro marrón salta sobre el perro perezoso.",
        "La volpe marrone veloce salta sopra il cane pigro.",
        "Vulpea maro rapidă sare peste câinele leneș.",
    ]
    # Minimal request payload for pinging the model
    request_data = {
        "text_items": [TextItem(text=text, metadata=Metadata(id=0)).model_dump() for text in texts],
        "chunk_size": 0,
        "chunk_overlap": 0,
    }

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
    Generate embeddings for text items with automatic chunking and language detection.

    This endpoint processes text items through the complete embedding pipeline:
    1. Automatic language detection for each text item
    2. Intelligent text chunking with sentence boundary preservation
    3. Dense vector embedding generation using MPNetv2
    4. Metadata preservation and enrichment

    The function communicates asynchronously with inference workers via ZMQ,
    allowing for horizontal scaling and load distribution.

    Args:
        request (InferenceRequest): Request containing text items and chunking parameters

    Returns:
        InferenceResponse: Embeddings with metadata and latency information

    Raises:
        HTTPException: 500 error if inference fails or workers are unavailable

    Example:
        >>> request = InferenceRequest(
        ...     text_items=[TextItem(text="Hello world", metadata=Metadata(id=1))],
        ...     chunk_size=1000
        ... )
        >>> response = await infer(request)
        >>> print(f"Generated {len(response.results)} embeddings")
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
    request_data = {
        "text_items": [item.model_dump() for item in request.text_items],
        "language": request.language,
        "mode": request.mode,
        "chunk_size": request.chunk_size,
        "chunk_overlap": request.chunk_overlap,
    }

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
    Health check endpoint for service monitoring and readiness verification.

    This endpoint verifies that the entire embedding service is operational:
    - ZMQ communication with inference workers is functioning
    - Model is loaded and responsive
    - Multi-language processing capabilities are available

    The endpoint uses a caching mechanism to avoid repeated health checks
    once the service is confirmed healthy, improving performance for
    frequent monitoring requests.

    Returns:
        SuccessResponse: Success status and version information

    Raises:
        HTTPException: 500 error if service is not ready or workers are unresponsive

    Note:
        On first successful ping, the service is marked as ready and subsequent
        calls return immediately without additional worker communication.
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
    Start the FastAPI HTTP server for the MPNetv2 embedding service.

    Creates and configures a FastAPI application with embedding and health check
    endpoints, initializes ZMQ communication, and starts the HTTP server.

    The server provides:
    - POST /infer: Text embedding generation with chunking support
    - GET /ping: Health check and readiness verification

    Server Configuration:
    - Host: 0.0.0.0 (accepts connections from all interfaces)
    - Port: Configured via MODEL_HTTP_PORT environment variable
    - Log Level: Configured via LOG_LEVEL setting

    The function handles ZMQ context cleanup on shutdown to ensure
    proper resource management.

    Note:
        This function blocks until the server is shut down. It should be
        run in a separate process in production deployments.
    """
    app = FastAPI()
    _ = app.post("/infer")(infer)
    _ = app.get("/ping")(ping)

    create_zmq_context()

    try:
        uvicorn.run(app, host="0.0.0.0", port=MODEL_HTTP_PORT, log_level=LOG_LEVEL)
    finally:
        context.term()
