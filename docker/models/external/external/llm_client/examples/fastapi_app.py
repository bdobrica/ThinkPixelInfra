"""Example FastAPI integration for llm_client."""

from contextlib import asynccontextmanager

from fastapi import Depends, FastAPI, Request

from ..client import LLMClient
from ..config import RetryConfig
from ..integrations.fastapi import (
    DeadlineMiddleware,
    disconnect_checker_from_request,
    llm_client_from_request,
    translate_llm_exception,
)


@asynccontextmanager
async def lifespan(app: FastAPI):
    app.state.llm = LLMClient.from_env(
        max_connections=512,
        max_keepalive_connections=128,
        keepalive_expiry_seconds=30,
        dns_ttl_seconds=60,
        retry=RetryConfig(
            max_retries=2,
            initial_backoff_ms=250,
            max_backoff_ms=1000,
            jitter_ratio=0.2,
        ),
    )
    yield
    await app.state.llm.aclose()


app = FastAPI(lifespan=lifespan)
app.add_middleware(
    DeadlineMiddleware,
    header_name="X-Request-Timeout-Ms",
    overhead_ms=100,
    max_timeout_ms=120_000,
)


@app.post("/ask")
async def ask(
    request: Request,
    llm: LLMClient = Depends(llm_client_from_request),
):
    try:
        return await llm.responses.create(
            model="gpt-4.1-mini",
            input="Say hello in one sentence.",
            disconnect_checker=disconnect_checker_from_request(request),
        )
    except Exception as exc:
        translate_llm_exception(exc)
        raise
