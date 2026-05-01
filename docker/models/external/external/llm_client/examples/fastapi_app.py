"""Example FastAPI application for :mod:`external.llm_client`.

This module is intentionally illustrative and is not imported by the core client
package at runtime.
"""

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
    """Create and dispose of the shared client used by the example app.

    :param app: FastAPI application whose ``state`` will store the shared client.
    :yields: Control back to FastAPI while the example app is serving requests.
    """
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


#: Example FastAPI application wired to :class:`LLMClient`.
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
    """Issue a sample response request through the shared client.

    :param request: Incoming FastAPI request used for disconnect propagation.
    :param llm: Shared client instance injected from application state.
    :returns: Parsed JSON response from the upstream model provider.
    :raises fastapi.HTTPException: If the helper translates an upstream client error.
    """
    try:
        return await llm.responses.create(
            model="gpt-4.1-mini",
            input="Say hello in one sentence.",
            disconnect_checker=disconnect_checker_from_request(request),
        )
    except Exception as exc:
        translate_llm_exception(exc)
        raise
