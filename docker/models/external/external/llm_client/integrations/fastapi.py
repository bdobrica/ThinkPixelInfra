"""FastAPI integration helpers for :mod:`external.llm_client`.

This module keeps framework-specific request handling and exception translation
outside of the core client package.
"""

import asyncio
from typing import Any

try:
    from fastapi import HTTPException, Request
    from starlette.middleware.base import BaseHTTPMiddleware
    from starlette.responses import Response
except Exception:  # pragma: no cover - keeps non-FastAPI usage importable
    HTTPException = None  # type: ignore[assignment]
    Request = Any  # type: ignore[misc,assignment]
    BaseHTTPMiddleware = object  # type: ignore[assignment]
    Response = Any  # type: ignore[misc,assignment]

from ..deadline import reset_deadline, set_deadline_after_ms
from ..errors import LLMClientError, LLMDeadlineExceeded, LLMProviderError
from ..types import DisconnectChecker


def disconnect_checker_from_request(request: Request) -> DisconnectChecker:
    """Adapt :class:`fastapi.Request` disconnect state to :data:`DisconnectChecker`.

    :param request: Incoming FastAPI request.
    :returns: Async callable that reports whether the client has disconnected.
    """
    return request.is_disconnected


class DeadlineMiddleware(BaseHTTPMiddleware):
    """Translate a timeout header into the shared deadline context.

    Example::

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
        """Configure deadline parsing middleware.

        :param app: Wrapped ASGI application.
        :param header_name: Request header containing timeout milliseconds.
        :param overhead_ms: Milliseconds reserved before the upstream call begins.
        :param max_timeout_ms: Optional upper bound applied to supplied timeout values.
        """
        super().__init__(app)
        self.header_name = header_name
        self.overhead_ms = overhead_ms
        self.max_timeout_ms = max_timeout_ms

    async def dispatch(self, request: Request, call_next: Any) -> Response:
        """Apply the request deadline while processing a FastAPI request.

        :param request: Current FastAPI request.
        :param call_next: Downstream middleware or route handler.
        :returns: Response returned by the downstream application.
        :raises fastapi.HTTPException: If the timeout header is malformed.
        """
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
    """Resolve the shared client instance from ``app.state``.

    :param request: FastAPI request whose application state stores the client.
    :returns: Client instance stored in ``request.app.state.llm``.
    :raises RuntimeError: If the application state does not expose ``llm``.
    """
    client = getattr(request.app.state, "llm", None)
    if client is None:
        raise RuntimeError("app.state.llm is not configured")
    return client


def translate_llm_exception(exc: BaseException) -> None:
    """Translate client exceptions into FastAPI-friendly HTTP errors.

    :param exc: Original exception raised during client processing.
    :raises fastapi.HTTPException: For client and provider failures that should be surfaced over HTTP.
    :raises BaseException: Re-raises ``exc`` unchanged when translation is not appropriate.
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
