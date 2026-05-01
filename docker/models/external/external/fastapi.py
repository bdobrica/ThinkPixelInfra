"""FastAPI server for the external embeddings gateway."""

import asyncio
import logging
import os
import time
from concurrent.futures import ThreadPoolExecutor
from contextlib import asynccontextmanager
from functools import partial
from typing import Dict, List, Literal

import uvicorn
from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel, Field

from .config import (
    LOG_LEVEL,
    MODEL_CHUNK_OVERLAP,
    MODEL_CHUNK_SIZE,
    MODEL_HTTP_PORT,
    MODEL_PREPROCESS_THREADS,
)
from .gateway import create_embeddings_gateway, create_llm_client
from .llm_client.integrations.fastapi import (
    DeadlineMiddleware,
    disconnect_checker_from_request,
    translate_llm_exception,
)
from .postprocess import build_results
from .preprocess import prepare_text_items

logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)


class Metadata(BaseModel):
    id: int
    extra: dict = Field(default_factory=dict)


class TextItem(BaseModel):
    text: str
    metadata: Metadata


class InferenceRequest(BaseModel):
    text_items: List[TextItem]
    language: str = "auto"
    mode: Literal["search", "store"] = "store"
    chunk_size: int = MODEL_CHUNK_SIZE
    chunk_overlap: int = MODEL_CHUNK_OVERLAP


class EmbeddingsItem(BaseModel):
    text: str
    offset: int
    dense_vector: str
    sparse_vector: Dict[str, str]
    metadata: Metadata


class InferenceResponse(BaseModel):
    results: List[EmbeddingsItem]
    latency: float = 0.0


class SuccessResponse(BaseModel):
    success: bool = True
    version: str = os.getenv("VERSION", "0.0.0")


async def _run_in_executor(executor: ThreadPoolExecutor, func, *args):
    loop = asyncio.get_running_loop()
    return await loop.run_in_executor(executor, partial(func, *args))


@asynccontextmanager
async def lifespan(app: FastAPI):
    llm_client = create_llm_client()
    executor = ThreadPoolExecutor(max_workers=MODEL_PREPROCESS_THREADS, thread_name_prefix="external-preprocess")
    app.state.llm = llm_client
    app.state.embeddings_gateway = create_embeddings_gateway(llm_client)
    app.state.preprocess_executor = executor
    app.state.ready = False
    try:
        yield
    finally:
        executor.shutdown(wait=True)
        await llm_client.aclose()


async def infer(payload: InferenceRequest, request: Request) -> InferenceResponse:
    start_time = time.perf_counter()
    text_items = [item.model_dump() for item in payload.text_items]
    disconnect_checker = disconnect_checker_from_request(request)

    try:
        prepared_items = await _run_in_executor(
            request.app.state.preprocess_executor,
            prepare_text_items,
            text_items,
            payload.language,
            payload.mode,
            payload.chunk_size,
            payload.chunk_overlap,
        )
        if not prepared_items:
            return InferenceResponse(results=[], latency=time.perf_counter() - start_time)

        dense_vectors = await request.app.state.embeddings_gateway.embed_texts(
            [item.get("text", "") for item in prepared_items],
            disconnect_checker=disconnect_checker,
        )
        results = await _run_in_executor(
            request.app.state.preprocess_executor,
            build_results,
            prepared_items,
            dense_vectors,
        )
    except Exception as exc:
        translate_llm_exception(exc)
        raise HTTPException(status_code=500, detail=str(exc)) from exc

    request.app.state.ready = True
    return InferenceResponse(
        results=[EmbeddingsItem(**item) for item in results],
        latency=time.perf_counter() - start_time,
    )


async def ping(request: Request) -> SuccessResponse:
    if request.app.state.ready:
        return SuccessResponse()

    try:
        await request.app.state.embeddings_gateway.ping(disconnect_checker=disconnect_checker_from_request(request))
    except Exception as exc:
        translate_llm_exception(exc)
        raise HTTPException(status_code=500, detail=str(exc)) from exc

    request.app.state.ready = True
    return SuccessResponse()


def create_app() -> FastAPI:
    app = FastAPI(lifespan=lifespan)
    app.add_middleware(
        DeadlineMiddleware,
        header_name="X-Request-Timeout-Ms",
        overhead_ms=100,
        max_timeout_ms=120_000,
    )
    _ = app.post("/infer", response_model=InferenceResponse)(infer)
    _ = app.get("/ping", response_model=SuccessResponse)(ping)
    return app


def fastapi_server() -> None:
    uvicorn.run(create_app(), host="0.0.0.0", port=MODEL_HTTP_PORT, log_level=LOG_LEVEL)
