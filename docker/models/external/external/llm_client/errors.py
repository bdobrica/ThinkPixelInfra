"""Error types for llm_client."""

from collections.abc import Mapping
from typing import Any


class LLMClientError(Exception):
    """Base class for client-side LLM errors."""


class LLMConfigurationError(LLMClientError):
    """Invalid or missing client configuration."""


class LLMDeadlineExceeded(LLMClientError):
    """The request cannot complete within the current deadline."""


class LLMProviderError(LLMClientError):
    """Non-successful provider response."""

    def __init__(self, status_code: int, body: Any, headers: Mapping[str, str] | None = None):
        self.status_code = status_code
        self.body = body
        self.headers = dict(headers or {})
        super().__init__(f"LLM provider returned HTTP {status_code}: {body!r}")
