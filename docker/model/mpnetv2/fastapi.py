import json
import logging
import os
import time
from functools import lru_cache
from typing import List

import uvicorn
import zmq
import zmq.asyncio
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .config import LOG_LEVEL, MODEL_HTTP_PORT, MODEL_ZMQ_CLIENT_ADDR

# Setup logging
logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)


class Metadata(BaseModel):
    id: int
    extra: dict = {}


class TextItem(BaseModel):
    text: str
    metadata: Metadata


class InferenceRequest(BaseModel):
    text_items: List[TextItem]


class EmbeddingsItem(BaseModel):
    text: str
    vector: str
    metadata: Metadata


class InferenceResponse(BaseModel):
    results: List[EmbeddingsItem]
    latency: float = 0.0


@lru_cache(maxsize=None)
def create_zmq_context():
    global context

    # ZMQ Async Context
    context = zmq.asyncio.Context()  # type: ignore
    logger.info("Created ZMQ context.")


async def infer(request: InferenceRequest) -> InferenceResponse:
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


def fastapi_server():
    app = FastAPI()
    _ = app.post("/infer")(infer)

    create_zmq_context()

    try:
        uvicorn.run(
            app,
            host="0.0.0.0",
            port=MODEL_HTTP_PORT,
            log_level=LOG_LEVEL,
        )
    finally:
        context.term()
