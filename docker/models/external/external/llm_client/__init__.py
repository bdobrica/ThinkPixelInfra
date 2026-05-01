"""Public package exports for :mod:`external.llm_client`.

This package exposes the minimal stable entrypoint intended for callers that only
need the core client surface.

:exports:
    - :class:`.LLMClient`
    - :class:`.RetryConfig`
    - :class:`.LLMClientError`
    - :class:`.LLMConfigurationError`
    - :class:`.LLMDeadlineExceeded`
    - :class:`.LLMProviderError`
"""

from .client import LLMClient
from .config import RetryConfig
from .errors import (
    LLMClientError,
    LLMConfigurationError,
    LLMDeadlineExceeded,
    LLMProviderError,
)

__all__ = [
    "LLMClient",
    "LLMClientError",
    "LLMConfigurationError",
    "LLMDeadlineExceeded",
    "LLMProviderError",
    "RetryConfig",
]
