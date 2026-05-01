"""Shared type and optional framework compatibility helpers for llm_client."""

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
