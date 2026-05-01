"""FastAPI integration helpers for llm_client."""

import asyncio
from typing import Any

from ..deadline import reset_deadline, set_deadline_after_ms
from ..errors import LLMClientError, LLMDeadlineExceeded, LLMProviderError
from ..types import BaseHTTPMiddleware, HTTPException, Request, Response


class DeadlineMiddleware(BaseHTTPMiddleware):
    """
    Parses a client TTL header and stores an absolute deadline in a ContextVar.

    Example:
        app.add_middleware(
            DeadlineMiddleware,
            header_name="X-Request-Timeout-Ms",
            overhead_ms=100,
        )
    """

    def __init__(
        self,
        app: Any,
        *,
        header_name: str = "X-Request-Timeout-Ms",
        overhead_ms: int = 100,
        max_timeout_ms: int | None = None,
    ) -> None:
        super().__init__(app)
        self.header_name = header_name
        self.overhead_ms = overhead_ms
        self.max_timeout_ms = max_timeout_ms

    async def dispatch(self, request: Request, call_next: Any) -> Response:
        raw = request.headers.get(self.header_name)
        timeout_ms: int | None = None

        if raw is not None:
            try:
                timeout_ms = max(0, int(raw))
            except ValueError:
                if HTTPException is not None:
                    raise HTTPException(
                        status_code=400, detail=f"Invalid {self.header_name}; expected integer milliseconds"
                    )
                raise ValueError(f"Invalid {self.header_name}; expected integer milliseconds")

            if self.max_timeout_ms is not None:
                timeout_ms = min(timeout_ms, self.max_timeout_ms)

        token = set_deadline_after_ms(timeout_ms, overhead_ms=self.overhead_ms)
        try:
            return await call_next(request)
        finally:
            reset_deadline(token)


async def llm_client_from_request(request: Request) -> Any:
    """Dependency helper: app.state.llm must be set in lifespan."""
    client = getattr(request.app.state, "llm", None)
    if client is None:
        raise RuntimeError("app.state.llm is not configured")
    return client


def translate_llm_exception(exc: BaseException) -> None:
    """
    Optional helper for FastAPI endpoints.

    Usage:
        try:
            return await llm.responses.create(...)
        except Exception as exc:
            translate_llm_exception(exc)
            raise
    """
    if HTTPException is None:
        raise exc

    if isinstance(exc, LLMDeadlineExceeded):
        raise HTTPException(status_code=504, detail=str(exc)) from exc

    if isinstance(exc, LLMProviderError):
        if 400 <= exc.status_code < 500:
            raise HTTPException(status_code=exc.status_code, detail=exc.body) from exc
        raise HTTPException(status_code=502, detail=exc.body) from exc

    if isinstance(exc, asyncio.CancelledError):
        raise exc

    if isinstance(exc, LLMClientError):
        raise HTTPException(status_code=502, detail=str(exc)) from exc

    raise exc
