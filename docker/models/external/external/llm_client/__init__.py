"""
First-pass OpenAI-compatible async LLM client.

Features:
- AsyncIO/FastAPI friendly; no event-loop creation.
- HTTPX-based shared connection pool owned by the LLMClient instance.
- Base URL/API key from constructor or env vars: LLM_BASE_URL, LLM_API_KEY.
- OpenAI-compatible endpoints:
    POST /chat/completions or /v1/chat/completions depending on base_url
    POST /responses or /v1/responses depending on base_url
    POST /embeddings or /v1/embeddings depending on base_url
- Retry loop with exponential backoff and jitter.
- Deadline/TTL support via ContextVar and optional FastAPI middleware.
- FastAPI disconnect-aware cancellation.

DNS cache note:
- This first pass keeps HTTPX's normal transport.
- The second pass can replace `transport=` with a custom DNS-caching transport.
"""

import asyncio
import contextvars
import dataclasses
import email.utils
import ipaddress
import json
import os
import random
import socket
import time
from collections.abc import Mapping, Sequence
from typing import Any

import httpcore
import httpx
from httpcore._backends.auto import AutoBackend as HttpcoreAutoBackend
from httpx._transports.default import AsyncResponseStream, map_httpcore_exceptions

try:
    from fastapi import HTTPException, Request
    from starlette.middleware.base import BaseHTTPMiddleware
    from starlette.responses import Response
except Exception:  # pragma: no cover - keeps non-FastAPI usage importable
    HTTPException = None  # type: ignore[assignment]
    Request = Any  # type: ignore[misc,assignment]
    BaseHTTPMiddleware = object  # type: ignore[assignment]
    Response = Any  # type: ignore[misc,assignment]


# -----------------------------------------------------------------------------
# Deadline context
# -----------------------------------------------------------------------------

_deadline_monotonic: contextvars.ContextVar[float | None] = contextvars.ContextVar(
    "llm_deadline_monotonic",
    default=None,
)


def set_deadline_after_ms(timeout_ms: int | None, overhead_ms: int = 0) -> contextvars.Token:
    """Set an absolute monotonic deadline from a relative millisecond budget."""
    if timeout_ms is None:
        return _deadline_monotonic.set(None)

    usable_ms = max(0, timeout_ms - overhead_ms)
    return _deadline_monotonic.set(time.monotonic() + usable_ms / 1000.0)


def set_deadline_at(deadline_monotonic: float | None) -> contextvars.Token:
    return _deadline_monotonic.set(deadline_monotonic)


def reset_deadline(token: contextvars.Token) -> None:
    _deadline_monotonic.reset(token)


def current_deadline() -> float | None:
    return _deadline_monotonic.get()


def remaining_budget_seconds() -> float | None:
    deadline = current_deadline()
    if deadline is None:
        return None
    return max(0.0, deadline - time.monotonic())


# -----------------------------------------------------------------------------
# DNS cache transport
# -----------------------------------------------------------------------------


@dataclasses.dataclass
class DNSCacheEntry:
    addresses: list[str]
    expires_at: float
    next_index: int = 0

    def choose(self) -> str:
        if not self.addresses:
            raise RuntimeError("DNS cache entry has no addresses")
        value = self.addresses[self.next_index % len(self.addresses)]
        self.next_index += 1
        return value


class AsyncDNSCache:
    """
    Small async DNS cache used by CachingAsyncNetworkBackend.

    Important behavior:
    - Caches A/AAAA results for dns_ttl_seconds.
    - Uses asyncio.get_running_loop().getaddrinfo(), so no new event loop is created.
    - Does not rewrite HTTP URLs or Host headers.
    - Used only at TCP connect time; TLS/SNI remains owned by httpcore and keeps the
      original hostname.
    """

    def __init__(self, *, ttl_seconds: float = 60.0, family: int = socket.AF_UNSPEC) -> None:
        if ttl_seconds <= 0:
            raise ValueError("ttl_seconds must be > 0")
        self.ttl_seconds = ttl_seconds
        self.family = family
        self._entries: dict[tuple[str, int], DNSCacheEntry] = {}
        self._lock = asyncio.Lock()

    async def resolve(self, host: str, port: int) -> str:
        # IP literals do not need DNS. Keep them out of the cache.
        try:
            ipaddress.ip_address(host)
            return host
        except ValueError:
            pass

        key = (host, port)
        now = time.monotonic()

        async with self._lock:
            entry = self._entries.get(key)
            if entry is not None and entry.expires_at > now and entry.addresses:
                return entry.choose()

        addresses = await self._resolve_uncached(host, port)
        if not addresses:
            raise OSError(f"DNS resolution returned no usable addresses for {host}:{port}")

        async with self._lock:
            # Another coroutine may have refreshed this while we were resolving. Prefer
            # the newest valid cache entry if present to avoid stampedes.
            existing = self._entries.get(key)
            now = time.monotonic()
            if existing is not None and existing.expires_at > now and existing.addresses:
                return existing.choose()

            entry = DNSCacheEntry(
                addresses=addresses,
                expires_at=now + self.ttl_seconds,
            )
            self._entries[key] = entry
            return entry.choose()

    async def refresh(self, host: str, port: int) -> list[str]:
        addresses = await self._resolve_uncached(host, port)
        if not addresses:
            raise OSError(f"DNS resolution returned no usable addresses for {host}:{port}")
        async with self._lock:
            self._entries[(host, port)] = DNSCacheEntry(
                addresses=addresses,
                expires_at=time.monotonic() + self.ttl_seconds,
            )
        return addresses

    async def clear(self, host: str | None = None, port: int | None = None) -> None:
        async with self._lock:
            if host is None:
                self._entries.clear()
                return
            if port is None:
                for key in list(self._entries):
                    if key[0] == host:
                        self._entries.pop(key, None)
                return
            self._entries.pop((host, port), None)

    async def snapshot(self) -> dict[str, Any]:
        now = time.monotonic()
        async with self._lock:
            return {
                f"{host}:{port}": {
                    "addresses": list(entry.addresses),
                    "expires_in_seconds": max(0.0, entry.expires_at - now),
                    "next_index": entry.next_index,
                }
                for (host, port), entry in self._entries.items()
            }

    async def _resolve_uncached(self, host: str, port: int) -> list[str]:
        loop = asyncio.get_running_loop()
        infos = await loop.getaddrinfo(
            host,
            port,
            family=self.family,
            type=socket.SOCK_STREAM,
        )

        addresses: list[str] = []
        seen: set[str] = set()
        for family, _type, _proto, _canonname, sockaddr in infos:
            ip = sockaddr[0]
            if ip not in seen:
                seen.add(ip)
                addresses.append(ip)
        return addresses


class CachingAsyncNetworkBackend(httpcore.AsyncNetworkBackend):
    """
    httpcore network backend that caches DNS before TCP connect.

    httpcore still receives the original URL host and still performs TLS with the
    original server hostname. This backend only swaps the TCP connect target from
    hostname -> cached IP address.
    """

    def __init__(self, *, dns_cache: AsyncDNSCache, backend: httpcore.AsyncNetworkBackend | None = None) -> None:
        self.dns_cache = dns_cache
        self.backend = backend or HttpcoreAutoBackend()

    async def connect_tcp(  # type: ignore[override]
        self,
        host: str,
        port: int,
        timeout: float | None = None,
        local_address: str | None = None,
        socket_options: Sequence[tuple[int, int, int | bytes]] | None = None,
    ) -> httpcore.AsyncNetworkStream:
        ip = await self.dns_cache.resolve(host, port)
        return await self.backend.connect_tcp(
            ip,
            port,
            timeout=timeout,
            local_address=local_address,
            socket_options=socket_options,
        )

    async def connect_unix_socket(  # type: ignore[override]
        self,
        path: str,
        timeout: float | None = None,
        socket_options: Sequence[tuple[int, int, int | bytes]] | None = None,
    ) -> httpcore.AsyncNetworkStream:
        return await self.backend.connect_unix_socket(path, timeout=timeout, socket_options=socket_options)

    async def sleep(self, seconds: float) -> None:
        await self.backend.sleep(seconds)


class DNSCachingAsyncHTTPTransport(httpx.AsyncBaseTransport):
    """
    Async HTTPX transport backed by httpcore.AsyncConnectionPool and DNS cache.

    This intentionally mirrors HTTPX's default async transport, but injects a
    custom httpcore network backend. It uses HTTPX's internal exception mapping;
    pin/test HTTPX versions in production.
    """

    def __init__(
        self,
        *,
        dns_cache: AsyncDNSCache,
        verify: bool = True,
        http1: bool = True,
        http2: bool = False,
        limits: httpx.Limits = httpx.Limits(),
        retries: int = 0,
        local_address: str | None = None,
        socket_options: Sequence[tuple[int, int, int | bytes]] | None = None,
    ) -> None:
        self.dns_cache = dns_cache
        self.network_backend = CachingAsyncNetworkBackend(dns_cache=dns_cache)
        self._pool = httpcore.AsyncConnectionPool(
            ssl_context=httpx.create_ssl_context(verify=verify),
            max_connections=limits.max_connections,
            max_keepalive_connections=limits.max_keepalive_connections,
            keepalive_expiry=limits.keepalive_expiry,
            http1=http1,
            http2=http2,
            retries=retries,
            local_address=local_address,
            network_backend=self.network_backend,
            socket_options=socket_options,
        )

    async def handle_async_request(self, request: httpx.Request) -> httpx.Response:
        assert isinstance(request.stream, httpx.AsyncByteStream)

        req = httpcore.Request(
            method=request.method,
            url=httpcore.URL(
                scheme=request.url.raw_scheme,
                host=request.url.raw_host,
                port=request.url.port,
                target=request.url.raw_path,
            ),
            headers=request.headers.raw,
            content=request.stream,
            extensions=request.extensions,
        )

        with map_httpcore_exceptions():
            resp = await self._pool.handle_async_request(req)

        return httpx.Response(
            status_code=resp.status,
            headers=resp.headers,
            stream=AsyncResponseStream(resp.stream),
            extensions=resp.extensions,
        )

    async def aclose(self) -> None:
        await self._pool.aclose()


# -----------------------------------------------------------------------------
# Config/errors
# -----------------------------------------------------------------------------


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


# -----------------------------------------------------------------------------
# FastAPI middleware
# -----------------------------------------------------------------------------


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
                # Bad TTL header means the caller sent an invalid contract.
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


# -----------------------------------------------------------------------------
# Resource facades
# -----------------------------------------------------------------------------


class ChatCompletionsResource:
    def __init__(self, client: "LLMClient") -> None:
        self._client = client

    async def create(self, *, request: Request | None = None, **payload: Any) -> dict[str, Any]:
        return await self._client.request_json("POST", "chat/completions", json_body=payload, request=request)


class ResponsesResource:
    def __init__(self, client: "LLMClient") -> None:
        self._client = client

    async def create(self, *, request: Request | None = None, **payload: Any) -> dict[str, Any]:
        return await self._client.request_json("POST", "responses", json_body=payload, request=request)


class EmbeddingsResource:
    def __init__(self, client: "LLMClient") -> None:
        self._client = client

    async def create(self, *, request: Request | None = None, **payload: Any) -> dict[str, Any]:
        return await self._client.request_json("POST", "embeddings", json_body=payload, request=request)


# -----------------------------------------------------------------------------
# Client
# -----------------------------------------------------------------------------


class LLMClient:
    def __init__(
        self,
        *,
        base_url: str | None = None,
        api_key: str | None = None,
        timeout_seconds: float = 60.0,
        max_connections: int = 256,
        max_keepalive_connections: int = 64,
        keepalive_expiry_seconds: float = 30.0,
        dns_ttl_seconds: float | None = None,
        dns_family: int = socket.AF_UNSPEC,
        retry: RetryConfig | None = None,
        default_headers: Mapping[str, str] | None = None,
        http_client: httpx.AsyncClient | None = None,
    ) -> None:
        base_url = base_url or os.getenv("LLM_BASE_URL")
        api_key = api_key or os.getenv("LLM_API_KEY")

        if not base_url:
            raise LLMConfigurationError("Missing base_url. Pass base_url=... or set LLM_BASE_URL.")
        if not api_key:
            raise LLMConfigurationError("Missing api_key. Pass api_key=... or set LLM_API_KEY.")

        self.config = LLMClientConfig(
            base_url=self._normalize_base_url(base_url),
            api_key=api_key,
            timeout_seconds=timeout_seconds,
            max_connections=max_connections,
            max_keepalive_connections=max_keepalive_connections,
            keepalive_expiry_seconds=keepalive_expiry_seconds,
            dns_ttl_seconds=dns_ttl_seconds,
            dns_family=dns_family,
            default_headers=dict(default_headers or {}),
            retry=retry or RetryConfig(),
        )

        self._owns_http_client = http_client is None
        self._dns_cache: AsyncDNSCache | None = None

        limits = httpx.Limits(
            max_connections=max_connections,
            max_keepalive_connections=max_keepalive_connections,
            keepalive_expiry=keepalive_expiry_seconds,
        )

        transport: httpx.AsyncBaseTransport | None = None
        if http_client is None and dns_ttl_seconds is not None:
            self._dns_cache = AsyncDNSCache(ttl_seconds=dns_ttl_seconds, family=dns_family)
            transport = DNSCachingAsyncHTTPTransport(
                dns_cache=self._dns_cache,
                limits=limits,
            )

        self._client = http_client or httpx.AsyncClient(
            base_url=self.config.base_url,
            timeout=httpx.Timeout(timeout_seconds),
            limits=limits,
            transport=transport,
            headers=self._base_headers(),
        )

        self.chat_completions = ChatCompletionsResource(self)
        self.responses = ResponsesResource(self)
        self.embeddings = EmbeddingsResource(self)

    @classmethod
    def from_env(cls, **kwargs: Any) -> "LLMClient":
        return cls(**kwargs)

    async def aclose(self) -> None:
        if self._owns_http_client:
            await self._client.aclose()

    async def clear_dns_cache(self, host: str | None = None, port: int | None = None) -> None:
        if self._dns_cache is not None:
            await self._dns_cache.clear(host=host, port=port)

    async def refresh_dns(self, host: str, port: int | None = None) -> list[str]:
        if self._dns_cache is None:
            raise LLMConfigurationError("DNS cache is disabled. Pass dns_ttl_seconds=... to enable it.")
        resolved_port = port if port is not None else self._default_port_for_base_url()
        return await self._dns_cache.refresh(host, resolved_port)

    async def dns_cache_snapshot(self) -> dict[str, Any]:
        if self._dns_cache is None:
            return {}
        return await self._dns_cache.snapshot()

    async def __aenter__(self) -> "LLMClient":
        return self

    async def __aexit__(self, exc_type: Any, exc: Any, tb: Any) -> None:
        await self.aclose()

    async def request_json(
        self,
        method: str,
        path: str,
        *,
        json_body: Mapping[str, Any] | None = None,
        request: Request | None = None,
        headers: Mapping[str, str] | None = None,
        timeout_ms: int | None = None,
    ) -> dict[str, Any]:
        """
        Make a deadline-aware, retrying JSON request.

        Args:
            request:
                Optional FastAPI request. If provided, this method cancels/aborts
                the provider request if the caller disconnects.
            timeout_ms:
                Optional per-call TTL. If set, it further constrains or establishes
                the current deadline for this call.
        """
        token: contextvars.Token | None = None
        if timeout_ms is not None:
            new_deadline = time.monotonic() + max(0, timeout_ms) / 1000.0
            old_deadline = current_deadline()
            effective_deadline = min(old_deadline, new_deadline) if old_deadline is not None else new_deadline
            token = set_deadline_at(effective_deadline)

        try:
            return await self._request_json_with_retries(
                method,
                path,
                json_body=json_body,
                request=request,
                headers=headers,
            )
        finally:
            if token is not None:
                reset_deadline(token)

    async def _request_json_with_retries(
        self,
        method: str,
        path: str,
        *,
        json_body: Mapping[str, Any] | None,
        request: Request | None,
        headers: Mapping[str, str] | None,
    ) -> dict[str, Any]:
        cfg = self.config.retry
        attempt = 0
        last_error: BaseException | None = None

        while True:
            self._raise_if_no_budget()

            if request is not None and await request.is_disconnected():
                raise asyncio.CancelledError("FastAPI client disconnected before LLM request started")

            try:
                response = await self._send_once(
                    method,
                    path,
                    json_body=json_body,
                    request=request,
                    headers=headers,
                )

                if 200 <= response.status_code < 300:
                    return self._decode_json_response(response)

                body = self._decode_error_body(response)
                provider_error = LLMProviderError(response.status_code, body, headers=response.headers)

                if not self._should_retry_status(response.status_code):
                    raise provider_error

                last_error = provider_error
                retry_after_seconds = self._parse_retry_after(response.headers.get("retry-after"))

            except asyncio.CancelledError:
                raise
            except httpx.TimeoutException as exc:
                last_error = exc
                retry_after_seconds = None
            except (httpx.ConnectError, httpx.RemoteProtocolError, httpx.ReadError, httpx.PoolTimeout) as exc:
                last_error = exc
                retry_after_seconds = None
            except httpx.HTTPError as exc:
                # Non-retryable HTTPX errors, e.g. invalid URL, too many redirects.
                raise LLMClientError(str(exc)) from exc

            if attempt >= cfg.max_retries:
                self._raise_final_error(last_error)

            backoff_seconds = retry_after_seconds
            if backoff_seconds is None:
                backoff_seconds = self._compute_backoff_seconds(attempt)

            # If there is not enough time to even wait the backoff, fail immediately.
            remaining = remaining_budget_seconds()
            if remaining is not None and remaining <= backoff_seconds:
                raise LLMDeadlineExceeded(
                    f"Not enough remaining deadline budget for retry backoff: "
                    f"remaining={remaining:.3f}s backoff={backoff_seconds:.3f}s"
                ) from last_error

            await self._sleep_with_disconnect_watch(backoff_seconds, request)
            attempt += 1

    async def _send_once(
        self,
        method: str,
        path: str,
        *,
        json_body: Mapping[str, Any] | None,
        request: Request | None,
        headers: Mapping[str, str] | None,
    ) -> httpx.Response:
        timeout = self._timeout_for_current_budget()
        call_coro = self._client.request(
            method,
            self._normalize_path(path),
            json=json_body,
            headers=headers,
            timeout=timeout,
        )

        if request is None:
            return await call_coro

        provider_task = asyncio.create_task(call_coro)
        disconnect_task = asyncio.create_task(self._wait_for_disconnect(request))

        try:
            done, pending = await asyncio.wait(
                {provider_task, disconnect_task},
                return_when=asyncio.FIRST_COMPLETED,
            )

            if disconnect_task in done and disconnect_task.result() is True:
                provider_task.cancel()
                try:
                    await provider_task
                except asyncio.CancelledError:
                    pass
                raise asyncio.CancelledError("FastAPI client disconnected during LLM request")

            disconnect_task.cancel()
            return await provider_task
        finally:
            for task in (provider_task, disconnect_task):
                if not task.done():
                    task.cancel()

    async def _wait_for_disconnect(self, request: Request, poll_interval_seconds: float = 0.05) -> bool:
        while True:
            if await request.is_disconnected():
                return True
            await asyncio.sleep(poll_interval_seconds)

    async def _sleep_with_disconnect_watch(self, seconds: float, request: Request | None) -> None:
        if seconds <= 0:
            return

        if request is None:
            await asyncio.sleep(seconds)
            return

        sleep_task = asyncio.create_task(asyncio.sleep(seconds))
        disconnect_task = asyncio.create_task(self._wait_for_disconnect(request))
        try:
            done, _ = await asyncio.wait({sleep_task, disconnect_task}, return_when=asyncio.FIRST_COMPLETED)
            if disconnect_task in done and disconnect_task.result() is True:
                raise asyncio.CancelledError("FastAPI client disconnected during LLM retry backoff")
        finally:
            for task in (sleep_task, disconnect_task):
                if not task.done():
                    task.cancel()

    def _timeout_for_current_budget(self) -> httpx.Timeout:
        remaining = remaining_budget_seconds()
        if remaining is None:
            return httpx.Timeout(self.config.timeout_seconds)

        if remaining <= 0:
            raise LLMDeadlineExceeded("No remaining deadline budget for LLM request")

        # Apply the remaining deadline as a hard total-ish timeout.
        # HTTPX has phase-specific timeouts, so we set all phases to the same cap.
        cap = min(self.config.timeout_seconds, remaining)
        return httpx.Timeout(timeout=cap, connect=cap, read=cap, write=cap, pool=cap)

    def _raise_if_no_budget(self) -> None:
        remaining = remaining_budget_seconds()
        if remaining is not None and remaining <= 0:
            raise LLMDeadlineExceeded("LLM request deadline exceeded")

    def _should_retry_status(self, status_code: int) -> bool:
        return status_code in self.config.retry.retry_statuses

    def _compute_backoff_seconds(self, attempt: int) -> float:
        cfg = self.config.retry
        base_ms = min(cfg.max_backoff_ms, cfg.initial_backoff_ms * (2**attempt))
        jitter = base_ms * cfg.jitter_ratio
        actual_ms = random.uniform(base_ms - jitter, base_ms + jitter) if jitter > 0 else base_ms
        return max(0.0, actual_ms / 1000.0)

    def _parse_retry_after(self, value: str | None) -> float | None:
        if not value:
            return None

        value = value.strip()
        if value.isdigit():
            return max(0.0, float(value))

        try:
            retry_at = email.utils.parsedate_to_datetime(value)
        except (TypeError, ValueError):
            return None

        return max(0.0, retry_at.timestamp() - time.time())

    def _raise_final_error(self, last_error: BaseException | None) -> None:
        if last_error is None:
            raise LLMClientError("LLM request failed without a captured error")
        if isinstance(last_error, LLMProviderError):
            raise last_error
        if isinstance(last_error, LLMDeadlineExceeded):
            raise last_error
        raise LLMClientError(str(last_error)) from last_error

    def _decode_json_response(self, response: httpx.Response) -> dict[str, Any]:
        try:
            data = response.json()
        except json.JSONDecodeError as exc:
            raise LLMClientError(f"LLM provider returned invalid JSON: {response.text[:500]!r}") from exc

        if not isinstance(data, dict):
            raise LLMClientError(f"LLM provider returned non-object JSON: {type(data).__name__}")
        return data

    def _decode_error_body(self, response: httpx.Response) -> Any:
        try:
            return response.json()
        except json.JSONDecodeError:
            return response.text[:2_000]

    def _base_headers(self) -> dict[str, str]:
        headers = {
            "Authorization": f"Bearer {self.config.api_key}",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": "openai-compatible-llm-client/0.1",
        }
        headers.update(self.config.default_headers)
        return headers

    def _default_port_for_base_url(self) -> int:
        url = httpx.URL(self.config.base_url)
        if url.port is not None:
            return url.port
        return 443 if url.scheme == "https" else 80

    @staticmethod
    def _normalize_base_url(base_url: str) -> str:
        return base_url.rstrip("/") + "/"

    @staticmethod
    def _normalize_path(path: str) -> str:
        return path.lstrip("/")


# -----------------------------------------------------------------------------
# FastAPI helpers
# -----------------------------------------------------------------------------


async def llm_client_from_request(request: Request) -> LLMClient:
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
        # Preserve upstream status where reasonable, but convert provider 5xx to 502.
        if 400 <= exc.status_code < 500:
            raise HTTPException(status_code=exc.status_code, detail=exc.body) from exc
        raise HTTPException(status_code=502, detail=exc.body) from exc

    if isinstance(exc, asyncio.CancelledError):
        # Usually the ASGI server will handle this. Kept here for completeness.
        raise exc

    if isinstance(exc, LLMClientError):
        raise HTTPException(status_code=502, detail=str(exc)) from exc

    raise exc


# -----------------------------------------------------------------------------
# Example FastAPI integration
# -----------------------------------------------------------------------------

EXAMPLE_FASTAPI_USAGE = r"""
from contextlib import asynccontextmanager
from fastapi import Depends, FastAPI, Request

from llm_client_first_pass import (
    DeadlineMiddleware,
    LLMClient,
    RetryConfig,
    llm_client_from_request,
    translate_llm_exception,
)

@asynccontextmanager
async def lifespan(app: FastAPI):
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
    try:
        return await llm.responses.create(
            model="gpt-4.1-mini",
            input="Say hello in one sentence.",
            request=request,
        )
    except Exception as exc:
        translate_llm_exception(exc)
        raise
"""
