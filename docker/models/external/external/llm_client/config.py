"""Configuration dataclasses for llm_client."""

import dataclasses
import socket
from collections.abc import Mapping


@dataclasses.dataclass(frozen=True)
class RetryConfig:
    max_retries: int = 2
    initial_backoff_ms: int = 250
    max_backoff_ms: int = 1_000
    jitter_ratio: float = 0.2

    retry_statuses: frozenset[int] = frozenset({408, 409, 425, 429, 500, 502, 503, 504})


@dataclasses.dataclass(frozen=True)
class LLMClientConfig:
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
