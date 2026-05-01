"""Configuration objects for :mod:`external.llm_client`.

The dataclasses in this module define the durable configuration surface for the
core client and its retry policy.
"""

import dataclasses
import socket
from collections.abc import Mapping


@dataclasses.dataclass(frozen=True)
class RetryConfig:
    """Retry policy for transient upstream failures.

    :param max_retries: Maximum number of retry attempts after the initial call.
    :param initial_backoff_ms: Initial retry delay in milliseconds.
    :param max_backoff_ms: Upper bound for exponential backoff in milliseconds.
    :param jitter_ratio: Fractional jitter applied to the backoff window.
    :param retry_statuses: HTTP status codes treated as retryable.
    """

    max_retries: int = 2
    initial_backoff_ms: int = 250
    max_backoff_ms: int = 1_000
    jitter_ratio: float = 0.2

    retry_statuses: frozenset[int] = frozenset({408, 409, 425, 429, 500, 502, 503, 504})


@dataclasses.dataclass(frozen=True)
class LLMClientConfig:
    """Resolved configuration for :class:`external.llm_client.client.LLMClient`.

    :param base_url: Normalized provider base URL used for relative API paths.
    :param api_key: Bearer token presented to the upstream provider.
    :param timeout_seconds: Default request timeout budget in seconds.
    :param max_connections: Maximum number of pooled HTTP connections.
    :param max_keepalive_connections: Maximum number of idle keepalive connections.
    :param keepalive_expiry_seconds: Idle connection expiry in seconds.
    :param dns_ttl_seconds: Optional DNS cache TTL in seconds.
    :param dns_family: Socket address family used during DNS resolution.
    :param default_headers: Extra headers attached to every request.
    :param retry: Retry policy applied to transient failures.
    """

    base_url: str
    api_key: str
    timeout_seconds: float = 60.0
    max_connections: int = 256
    max_keepalive_connections: int = 64
    keepalive_expiry_seconds: float = 30.0
    dns_ttl_seconds: float | None = None
    dns_family: int = socket.AF_UNSPEC
    default_headers: Mapping[str, str] = dataclasses.field(default_factory=dict)
    retry: RetryConfig = dataclasses.field(default_factory=RetryConfig)
