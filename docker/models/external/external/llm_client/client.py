"""Core async client implementation for :mod:`external.llm_client`.

The :class:`LLMClient` in this module provides a shared async HTTP client, retry
handling, deadline awareness, and optional DNS-cached transport support for
OpenAI-compatible upstream APIs.
"""

import asyncio
import email.utils
import json
import os
import random
import socket
import time
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any

import httpx

from .config import LLMClientConfig, RetryConfig
from .deadline import (
    current_deadline,
    remaining_budget_seconds,
    reset_deadline,
    set_deadline_at,
)
from .errors import (
    LLMClientError,
    LLMConfigurationError,
    LLMDeadlineExceeded,
    LLMProviderError,
)
from .resources import ChatCompletionsResource, EmbeddingsResource, ResponsesResource
from .types import DisconnectChecker

if TYPE_CHECKING:
    from .transports.dns_cache import AsyncDNSCache


class LLMClient:
    """Async client for OpenAI-compatible chat, response, and embedding APIs.

    :param base_url: Provider base URL, or ``None`` to read ``LLM_BASE_URL``.
    :param api_key: Bearer token, or ``None`` to read ``LLM_API_KEY``.
    :param timeout_seconds: Default request timeout in seconds.
    :param max_connections: Maximum pooled connections for the shared HTTP client.
    :param max_keepalive_connections: Maximum idle keepalive connections.
    :param keepalive_expiry_seconds: Idle keepalive expiry in seconds.
    :param dns_ttl_seconds: Optional DNS cache TTL that enables the custom transport.
    :param dns_family: Address family used for DNS resolution.
    :param retry: Optional retry policy override.
    :param default_headers: Additional headers merged into every outgoing request.
    :param http_client: Optional pre-built :class:`httpx.AsyncClient`.
    :raises LLMConfigurationError: If required configuration is missing.
    """

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
            self._dns_cache, transport = self._build_dns_transport(
                dns_ttl_seconds=dns_ttl_seconds,
                dns_family=dns_family,
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
        """Construct a client using environment-backed defaults.

        :param kwargs: Additional keyword arguments forwarded to :class:`LLMClient`.
        :returns: Configured client instance.
        """
        return cls(**kwargs)

    async def aclose(self) -> None:
        """Close the underlying HTTP client if this instance owns it."""
        if self._owns_http_client:
            await self._client.aclose()

    async def clear_dns_cache(self, host: str | None = None, port: int | None = None) -> None:
        """Clear cached DNS entries managed by the optional DNS transport.

        :param host: Optional hostname filter.
        :param port: Optional port filter used together with ``host``.
        """
        if self._dns_cache is not None:
            await self._dns_cache.clear(host=host, port=port)

    async def refresh_dns(self, host: str, port: int | None = None) -> list[str]:
        """Force-refresh DNS answers for a host.

        :param host: Hostname to refresh.
        :param port: Optional port used as part of the cache key.
        :returns: List of resolved IP addresses.
        :raises LLMConfigurationError: If DNS caching is disabled for this client.
        """
        if self._dns_cache is None:
            raise LLMConfigurationError("DNS cache is disabled. Pass dns_ttl_seconds=... to enable it.")
        resolved_port = port if port is not None else self._default_port_for_base_url()
        return await self._dns_cache.refresh(host, resolved_port)

    async def dns_cache_snapshot(self) -> dict[str, Any]:
        """Return the current DNS cache contents.

        :returns: Mapping of cache keys to cached resolution metadata.
        """
        if self._dns_cache is None:
            return {}
        return await self._dns_cache.snapshot()

    async def __aenter__(self) -> "LLMClient":
        """Enter the async context manager for this client.

        :returns: The current client instance.
        """
        return self

    async def __aexit__(self, exc_type: Any, exc: Any, tb: Any) -> None:
        """Close the client when leaving an async context manager."""
        await self.aclose()

    async def request_json(
        self,
        method: str,
        path: str,
        *,
        json_body: Mapping[str, Any] | None = None,
        disconnect_checker: DisconnectChecker | None = None,
        headers: Mapping[str, str] | None = None,
        timeout_ms: int | None = None,
    ) -> dict[str, Any]:
        """Send a JSON request through the shared HTTP client.

        :param method: HTTP method.
        :param path: Relative provider path.
        :param json_body: Optional JSON request body.
        :param disconnect_checker: Optional async callback for caller disconnect state.
        :param headers: Optional per-request headers.
        :param timeout_ms: Optional deadline override in milliseconds.
        :returns: Parsed JSON object from the upstream provider.
        :raises LLMClientError: If the request cannot be completed successfully.
        :raises asyncio.CancelledError: If the caller disconnects while the request is in flight.
        """
        token = None
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
                disconnect_checker=disconnect_checker,
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
        disconnect_checker: DisconnectChecker | None,
        headers: Mapping[str, str] | None,
    ) -> dict[str, Any]:
        """Execute the retry loop for a JSON request.

        :param method: HTTP method.
        :param path: Relative provider path.
        :param json_body: Optional JSON request body.
        :param disconnect_checker: Optional async callback for caller disconnect state.
        :param headers: Optional per-request headers.
        :returns: Parsed JSON object from the upstream provider.
        :raises LLMClientError: If retries are exhausted or the response cannot be decoded.
        """
        cfg = self.config.retry
        attempt = 0
        last_error: BaseException | None = None

        while True:
            self._raise_if_no_budget()

            if disconnect_checker is not None and await disconnect_checker():
                raise asyncio.CancelledError("Client disconnected before LLM request started")

            try:
                response = await self._send_once(
                    method,
                    path,
                    json_body=json_body,
                    disconnect_checker=disconnect_checker,
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
                raise LLMClientError(str(exc)) from exc

            if attempt >= cfg.max_retries:
                self._raise_final_error(last_error)

            backoff_seconds = retry_after_seconds
            if backoff_seconds is None:
                backoff_seconds = self._compute_backoff_seconds(attempt)

            remaining = remaining_budget_seconds()
            if remaining is not None and remaining <= backoff_seconds:
                raise LLMDeadlineExceeded(
                    f"Not enough remaining deadline budget for retry backoff: "
                    f"remaining={remaining:.3f}s backoff={backoff_seconds:.3f}s"
                ) from last_error

            await self._sleep_with_disconnect_watch(backoff_seconds, disconnect_checker)
            attempt += 1

    async def _send_once(
        self,
        method: str,
        path: str,
        *,
        json_body: Mapping[str, Any] | None,
        disconnect_checker: DisconnectChecker | None,
        headers: Mapping[str, str] | None,
    ) -> httpx.Response:
        """Issue a single upstream HTTP request.

        :param method: HTTP method.
        :param path: Relative provider path.
        :param json_body: Optional JSON request body.
        :param disconnect_checker: Optional async callback for caller disconnect state.
        :param headers: Optional per-request headers.
        :returns: Raw :class:`httpx.Response` from the upstream provider.
        :raises asyncio.CancelledError: If the caller disconnects while waiting on the provider.
        """
        timeout = self._timeout_for_current_budget()
        call_coro = self._client.request(
            method,
            self._normalize_path(path),
            json=json_body,
            headers=headers,
            timeout=timeout,
        )

        if disconnect_checker is None:
            return await call_coro

        provider_task = asyncio.create_task(call_coro)
        disconnect_task = asyncio.create_task(self._wait_for_disconnect(disconnect_checker))

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
                raise asyncio.CancelledError("Client disconnected during LLM request")

            disconnect_task.cancel()
            return await provider_task
        finally:
            for task in (provider_task, disconnect_task):
                if not task.done():
                    task.cancel()

    async def _wait_for_disconnect(
        self,
        disconnect_checker: DisconnectChecker,
        poll_interval_seconds: float = 0.05,
    ) -> bool:
        """Poll an async disconnect checker until it reports a disconnect.

        :param disconnect_checker: Async callback reporting caller disconnect state.
        :param poll_interval_seconds: Delay between disconnect checks.
        :returns: ``True`` once the caller is disconnected.
        """
        while True:
            if await disconnect_checker():
                return True
            await asyncio.sleep(poll_interval_seconds)

    async def _sleep_with_disconnect_watch(
        self,
        seconds: float,
        disconnect_checker: DisconnectChecker | None,
    ) -> None:
        """Sleep for a backoff interval while still honoring disconnect signals.

        :param seconds: Backoff interval in seconds.
        :param disconnect_checker: Optional async callback for caller disconnect state.
        :raises asyncio.CancelledError: If the caller disconnects during backoff.
        """
        if seconds <= 0:
            return

        if disconnect_checker is None:
            await asyncio.sleep(seconds)
            return

        sleep_task = asyncio.create_task(asyncio.sleep(seconds))
        disconnect_task = asyncio.create_task(self._wait_for_disconnect(disconnect_checker))
        try:
            done, _ = await asyncio.wait({sleep_task, disconnect_task}, return_when=asyncio.FIRST_COMPLETED)
            if disconnect_task in done and disconnect_task.result() is True:
                raise asyncio.CancelledError("Client disconnected during LLM retry backoff")
        finally:
            for task in (sleep_task, disconnect_task):
                if not task.done():
                    task.cancel()

    def _timeout_for_current_budget(self) -> httpx.Timeout:
        """Build a timeout object constrained by the active deadline.

        :returns: Timeout object suitable for :mod:`httpx` requests.
        :raises LLMDeadlineExceeded: If no usable deadline budget remains.
        """
        remaining = remaining_budget_seconds()
        if remaining is None:
            return httpx.Timeout(self.config.timeout_seconds)

        if remaining <= 0:
            raise LLMDeadlineExceeded("No remaining deadline budget for LLM request")

        cap = min(self.config.timeout_seconds, remaining)
        return httpx.Timeout(timeout=cap, connect=cap, read=cap, write=cap, pool=cap)

    def _raise_if_no_budget(self) -> None:
        """Raise if the current deadline budget has already been exhausted.

        :raises LLMDeadlineExceeded: If the active deadline has expired.
        """
        remaining = remaining_budget_seconds()
        if remaining is not None and remaining <= 0:
            raise LLMDeadlineExceeded("LLM request deadline exceeded")

    def _should_retry_status(self, status_code: int) -> bool:
        """Return whether a status code is retryable.

        :param status_code: HTTP status code returned by the provider.
        :returns: ``True`` when the status code should trigger the retry policy.
        """
        return status_code in self.config.retry.retry_statuses

    def _compute_backoff_seconds(self, attempt: int) -> float:
        """Compute exponential backoff with jitter for a retry attempt.

        :param attempt: Zero-based retry attempt number.
        :returns: Sleep interval in seconds.
        """
        cfg = self.config.retry
        base_ms = min(cfg.max_backoff_ms, cfg.initial_backoff_ms * (2**attempt))
        jitter = base_ms * cfg.jitter_ratio
        actual_ms = random.uniform(base_ms - jitter, base_ms + jitter) if jitter > 0 else base_ms
        return max(0.0, actual_ms / 1000.0)

    def _parse_retry_after(self, value: str | None) -> float | None:
        """Parse a ``Retry-After`` header into seconds.

        :param value: Header value returned by the provider.
        :returns: Parsed delay in seconds, or ``None`` when parsing fails.
        """
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
        """Raise the final exception after retries are exhausted.

        :param last_error: Last captured exception from the retry loop.
        :raises LLMClientError: If the final error is not already a concrete client exception.
        """
        if last_error is None:
            raise LLMClientError("LLM request failed without a captured error")
        if isinstance(last_error, LLMProviderError):
            raise last_error
        if isinstance(last_error, LLMDeadlineExceeded):
            raise last_error
        raise LLMClientError(str(last_error)) from last_error

    def _decode_json_response(self, response: httpx.Response) -> dict[str, Any]:
        """Decode a successful JSON response into an object.

        :param response: Response returned by :mod:`httpx`.
        :returns: Parsed JSON object.
        :raises LLMClientError: If the payload is invalid JSON or not an object.
        """
        try:
            data = response.json()
        except json.JSONDecodeError as exc:
            raise LLMClientError(f"LLM provider returned invalid JSON: {response.text[:500]!r}") from exc

        if not isinstance(data, dict):
            raise LLMClientError(f"LLM provider returned non-object JSON: {type(data).__name__}")
        return data

    def _decode_error_body(self, response: httpx.Response) -> Any:
        """Decode an error response body with a JSON-first strategy.

        :param response: Response returned by :mod:`httpx`.
        :returns: Parsed JSON body or a truncated text fallback.
        """
        try:
            return response.json()
        except json.JSONDecodeError:
            return response.text[:2_000]

    def _base_headers(self) -> dict[str, str]:
        """Build the base request headers for the shared HTTP client.

        :returns: Headers attached to every outgoing request.
        """
        headers = {
            "Authorization": f"Bearer {self.config.api_key}",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": "openai-compatible-llm-client/0.1",
        }
        headers.update(self.config.default_headers)
        return headers

    def _default_port_for_base_url(self) -> int:
        """Return the implicit port for the configured base URL.

        :returns: Explicit or scheme-derived network port.
        """
        url = httpx.URL(self.config.base_url)
        if url.port is not None:
            return url.port
        return 443 if url.scheme == "https" else 80

    @staticmethod
    def _normalize_base_url(base_url: str) -> str:
        """Normalize a base URL so relative request paths resolve correctly.

        :param base_url: Provider base URL.
        :returns: Base URL guaranteed to end with ``/``.
        """
        return base_url.rstrip("/") + "/"

    @staticmethod
    def _normalize_path(path: str) -> str:
        """Normalize a relative request path.

        :param path: Relative provider path.
        :returns: Path without a leading slash.
        """
        return path.lstrip("/")

    def _build_dns_transport(
        self,
        *,
        dns_ttl_seconds: float,
        dns_family: int,
        limits: httpx.Limits,
    ) -> tuple["AsyncDNSCache", httpx.AsyncBaseTransport]:
        """Construct the optional DNS-cached transport.

        :param dns_ttl_seconds: DNS cache TTL in seconds.
        :param dns_family: Address family used during resolution.
        :param limits: Shared HTTP connection limits.
        :returns: Tuple of DNS cache instance and async transport.
        :raises LLMConfigurationError: If the optional transport dependencies are unavailable.
        """
        try:
            from .transports.dns_cache import (
                AsyncDNSCache,
                DNSCachingAsyncHTTPTransport,
            )
        except ImportError as exc:  # pragma: no cover - depends on optional transport dependencies
            raise LLMConfigurationError(
                "DNS caching transport is unavailable. Disable dns_ttl_seconds or install transport dependencies."
            ) from exc

        dns_cache = AsyncDNSCache(ttl_seconds=dns_ttl_seconds, family=dns_family)
        transport = DNSCachingAsyncHTTPTransport(
            dns_cache=dns_cache,
            limits=limits,
        )
        return dns_cache, transport
