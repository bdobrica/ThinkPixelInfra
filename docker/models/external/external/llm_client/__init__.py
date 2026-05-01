"""Public package exports for llm_client."""

from .client import LLMClient
from .config import LLMClientConfig, RetryConfig
from .deadline import (
    current_deadline,
    remaining_budget_seconds,
    reset_deadline,
    set_deadline_after_ms,
    set_deadline_at,
)
from .errors import (
    LLMClientError,
    LLMConfigurationError,
    LLMDeadlineExceeded,
    LLMProviderError,
)
from .examples import EXAMPLE_FASTAPI_USAGE
from .integrations.fastapi import (
    DeadlineMiddleware,
    llm_client_from_request,
    translate_llm_exception,
)
from .resources import ChatCompletionsResource, EmbeddingsResource, ResponsesResource
from .transports.dns_cache import (
    AsyncDNSCache,
    CachingAsyncNetworkBackend,
    DNSCacheEntry,
    DNSCachingAsyncHTTPTransport,
)

__all__ = [
    "AsyncDNSCache",
    "CachingAsyncNetworkBackend",
    "ChatCompletionsResource",
    "current_deadline",
    "DeadlineMiddleware",
    "DNSCacheEntry",
    "DNSCachingAsyncHTTPTransport",
    "EmbeddingsResource",
    "EXAMPLE_FASTAPI_USAGE",
    "LLMClient",
    "LLMClientConfig",
    "LLMClientError",
    "LLMConfigurationError",
    "LLMDeadlineExceeded",
    "LLMProviderError",
    "llm_client_from_request",
    "remaining_budget_seconds",
    "reset_deadline",
    "ResponsesResource",
    "RetryConfig",
    "set_deadline_after_ms",
    "set_deadline_at",
    "translate_llm_exception",
]
