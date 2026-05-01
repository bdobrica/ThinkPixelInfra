"""Error hierarchy for :mod:`external.llm_client`."""

from collections.abc import Mapping
from typing import Any


class LLMClientError(Exception):
    """Base class for client-side LLM errors."""


class LLMConfigurationError(LLMClientError):
    """Raised when the client configuration is invalid or incomplete."""


class LLMDeadlineExceeded(LLMClientError):
    """Raised when the available deadline budget is exhausted."""


class LLMProviderError(LLMClientError):
    """Raised when the upstream provider returns a non-success response."""

    def __init__(self, status_code: int, body: Any, headers: Mapping[str, str] | None = None):
        """Store the provider response details.

        :param status_code: HTTP status code returned by the provider.
        :param body: Parsed provider response body or fallback text payload.
        :param headers: Optional response headers from the provider.
        """
        self.status_code = status_code
        self.body = body
        self.headers = dict(headers or {})
        super().__init__(f"LLM provider returned HTTP {status_code}: {body!r}")
