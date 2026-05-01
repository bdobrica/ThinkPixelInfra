from __future__ import annotations

"""Optional DNS-caching transport internals for :mod:`external.llm_client`.

This module wraps version-sensitive :mod:`httpx` and :mod:`httpcore` transport
internals behind a small compatibility layer used only when DNS caching is
explicitly enabled.
"""

import asyncio
import dataclasses
import ipaddress
import socket
import time
from collections.abc import Iterator, Sequence
from contextlib import contextmanager
from typing import Any, cast

import httpcore
import httpx

SocketOptionValue = int | bytes
SocketOption = tuple[int, int, SocketOptionValue]
SocketOptions = Sequence[SocketOption]

_SUPPORTED_HTTPX_VERSION_PREFIXES = ("0.28.",)
_SUPPORTED_HTTPCORE_VERSION_PREFIXES = ("1.0.",)
_TESTED_HTTPX_VERSION = "0.28.1"
_TESTED_HTTPCORE_VERSION = "1.0.9"


def _ensure_supported_transport_versions() -> None:
    """Validate that the runtime transport dependencies match the tested ranges.

    :raises RuntimeError: If the installed ``httpx`` or ``httpcore`` versions fall outside the supported range.
    """
    if not any(httpx.__version__.startswith(prefix) for prefix in _SUPPORTED_HTTPX_VERSION_PREFIXES):
        raise RuntimeError(
            "DNSCachingAsyncHTTPTransport relies on private httpx internals and only supports "
            f"httpx {_SUPPORTED_HTTPX_VERSION_PREFIXES!r}; found {httpx.__version__}. "
            f"Tested version: {_TESTED_HTTPX_VERSION}."
        )

    if not any(httpcore.__version__.startswith(prefix) for prefix in _SUPPORTED_HTTPCORE_VERSION_PREFIXES):
        raise RuntimeError(
            "DNSCachingAsyncHTTPTransport relies on private httpcore internals and only supports "
            f"httpcore {_SUPPORTED_HTTPCORE_VERSION_PREFIXES!r}; found {httpcore.__version__}. "
            f"Tested version: {_TESTED_HTTPCORE_VERSION}."
        )


def _build_default_network_backend() -> httpcore.AsyncNetworkBackend:
    """Create the default async network backend used by the DNS transport.

    :returns: Async network backend instance for :mod:`httpcore`.
    :raises RuntimeError: If the required private ``httpcore`` symbol is unavailable.
    """
    try:
        from httpcore._backends.auto import AutoBackend as HttpcoreAutoBackend
    except ImportError as exc:  # pragma: no cover - version-specific private import
        raise RuntimeError(
            "DNSCachingAsyncHTTPTransport could not import httpcore AutoBackend. "
            f"Tested httpcore version: {_TESTED_HTTPCORE_VERSION}."
        ) from exc

    return cast(httpcore.AsyncNetworkBackend, HttpcoreAutoBackend())


@contextmanager
def _map_private_httpcore_exceptions() -> "Iterator[None]":
    """Expose HTTPX's private httpcore-exception mapper as a context manager.

    :yields: Control while httpcore exceptions are translated to HTTPX exceptions.
    :raises RuntimeError: If the private helper cannot be imported.
    """
    try:
        from httpx._transports.default import map_httpcore_exceptions
    except ImportError as exc:  # pragma: no cover - version-specific private import
        raise RuntimeError(
            "DNSCachingAsyncHTTPTransport could not import httpx exception mapping internals. "
            f"Tested httpx version: {_TESTED_HTTPX_VERSION}."
        ) from exc

    with map_httpcore_exceptions():
        yield


def _build_async_response_stream(stream: httpcore.AsyncByteStream) -> httpx.AsyncByteStream:
    """Wrap an httpcore async stream in the HTTPX response-stream adapter.

    :param stream: Async byte stream returned by :mod:`httpcore`.
    :returns: Async byte stream suitable for :class:`httpx.Response`.
    :raises RuntimeError: If the private adapter cannot be imported.
    """
    try:
        from httpx._transports.default import AsyncResponseStream
    except ImportError as exc:  # pragma: no cover - version-specific private import
        raise RuntimeError(
            "DNSCachingAsyncHTTPTransport could not import httpx AsyncResponseStream. "
            f"Tested httpx version: {_TESTED_HTTPX_VERSION}."
        ) from exc

    return cast(httpx.AsyncByteStream, AsyncResponseStream(stream))


@dataclasses.dataclass
class DNSCacheEntry:
    """Cached DNS resolution state for a single ``(host, port)`` key.

    :param addresses: Ordered list of IP addresses returned by DNS.
    :param expires_at: Absolute monotonic expiry timestamp.
    :param next_index: Next address index used for round-robin selection.
    """

    addresses: list[str]
    expires_at: float
    next_index: int = 0

    def choose(self) -> str:
        """Return the next address using round-robin selection.

        :returns: Selected IP address.
        :raises RuntimeError: If the cache entry contains no addresses.
        """
        if not self.addresses:
            raise RuntimeError("DNS cache entry has no addresses")
        value = self.addresses[self.next_index % len(self.addresses)]
        self.next_index += 1
        return value


class AsyncDNSCache:
    """Small async DNS cache used by :class:`CachingAsyncNetworkBackend`.

    Important behavior:

    - caches A/AAAA answers for ``dns_ttl_seconds``
    - uses ``asyncio.get_running_loop().getaddrinfo()``
    - does not rewrite HTTP URLs or ``Host`` headers
    - is used only at TCP connect time, while TLS/SNI still uses the original hostname

    :param ttl_seconds: Cache TTL in seconds.
    :param family: Socket address family used during DNS resolution.
    """

    def __init__(self, *, ttl_seconds: float = 60.0, family: int = socket.AF_UNSPEC) -> None:
        """Initialize the async DNS cache.

        :param ttl_seconds: Cache TTL in seconds.
        :param family: Socket address family used during DNS resolution.
        :raises ValueError: If ``ttl_seconds`` is not positive.
        """
        if ttl_seconds <= 0:
            raise ValueError("ttl_seconds must be > 0")
        self.ttl_seconds = ttl_seconds
        self.family = family
        self._entries: dict[tuple[str, int], DNSCacheEntry] = {}
        self._lock = asyncio.Lock()

    async def resolve(self, host: str, port: int) -> str:
        """Resolve and cache a hostname for a specific port.

        :param host: Hostname or IP literal.
        :param port: Destination port.
        :returns: Resolved IP address or the original IP literal.
        :raises OSError: If DNS resolution yields no usable addresses.
        """
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
        """Force a fresh DNS resolution for a host/port pair.

        :param host: Hostname to resolve.
        :param port: Destination port.
        :returns: List of resolved IP addresses.
        :raises OSError: If DNS resolution yields no usable addresses.
        """
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
        """Clear cached DNS entries.

        :param host: Optional hostname filter.
        :param port: Optional port filter used together with ``host``.
        """
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
        """Return a serializable view of the cache contents.

        :returns: Mapping of cache keys to address and expiry metadata.
        """
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
        """Resolve a hostname without consulting or updating the cache.

        :param host: Hostname to resolve.
        :param port: Destination port.
        :returns: Ordered list of unique IP addresses.
        """
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
    """httpcore network backend that applies DNS caching before TCP connect.

    httpcore still receives the original URL host and still performs TLS with the
    original server hostname. This backend only swaps the TCP connect target from
    hostname to a cached IP address.

    :param dns_cache: DNS cache used to resolve hostnames before connect.
    :param backend: Optional underlying httpcore backend implementation.
    """

    def __init__(self, *, dns_cache: AsyncDNSCache, backend: httpcore.AsyncNetworkBackend | None = None) -> None:
        """Initialize the caching network backend.

        :param dns_cache: DNS cache used to resolve hostnames.
        :param backend: Optional httpcore backend used for actual socket operations.
        """
        self.dns_cache = dns_cache
        self.backend = backend or _build_default_network_backend()

    async def connect_tcp(  # type: ignore[override]
        self,
        host: str,
        port: int,
        timeout: float | None = None,
        local_address: str | None = None,
        socket_options: SocketOptions | None = None,
    ) -> httpcore.AsyncNetworkStream:
        """Open a TCP connection after resolving the hostname through the cache.

        :param host: Hostname or IP literal.
        :param port: Destination port.
        :param timeout: Optional connection timeout.
        :param local_address: Optional local bind address.
        :param socket_options: Optional low-level socket options.
        :returns: Async network stream connected to the selected IP address.
        """
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
        socket_options: SocketOptions | None = None,
    ) -> httpcore.AsyncNetworkStream:
        """Delegate Unix socket connections to the wrapped backend.

        :param path: Filesystem path to the Unix socket.
        :param timeout: Optional connection timeout.
        :param socket_options: Optional low-level socket options.
        :returns: Async network stream connected to the Unix socket.
        """
        return await self.backend.connect_unix_socket(path, timeout=timeout, socket_options=socket_options)

    async def sleep(self, seconds: float) -> None:
        """Delegate backend sleep behavior.

        :param seconds: Duration to sleep.
        """
        await self.backend.sleep(seconds)


class DNSCachingAsyncHTTPTransport(httpx.AsyncBaseTransport):
    """Async HTTPX transport backed by :class:`httpcore.AsyncConnectionPool`.

    This transport mirrors HTTPX's default async transport, but injects a custom
    DNS-aware network backend. It relies on private HTTPX/httpcore APIs and is
    therefore considered an advanced internal transport.

    :param dns_cache: DNS cache used during hostname resolution.
    :param verify: TLS verification setting forwarded to HTTPX.
    :param http1: Whether HTTP/1.1 support is enabled.
    :param http2: Whether HTTP/2 support is enabled.
    :param limits: Optional HTTP connection limits.
    :param retries: Low-level connection retry count used by httpcore.
    :param local_address: Optional local bind address.
    :param socket_options: Optional low-level socket options.
    :raises RuntimeError: If the runtime HTTPX/httpcore versions are unsupported.
    """

    def __init__(
        self,
        *,
        dns_cache: AsyncDNSCache,
        verify: bool = True,
        http1: bool = True,
        http2: bool = False,
        limits: httpx.Limits | None = None,
        retries: int = 0,
        local_address: str | None = None,
        socket_options: SocketOptions | None = None,
    ) -> None:
        """Initialize the DNS-caching async HTTP transport.

        :param dns_cache: DNS cache used during hostname resolution.
        :param verify: TLS verification setting forwarded to HTTPX.
        :param http1: Whether HTTP/1.1 support is enabled.
        :param http2: Whether HTTP/2 support is enabled.
        :param limits: Optional HTTP connection limits.
        :param retries: Low-level connection retry count used by httpcore.
        :param local_address: Optional local bind address.
        :param socket_options: Optional low-level socket options.
        :raises RuntimeError: If the runtime HTTPX/httpcore versions are unsupported.
        """
        _ensure_supported_transport_versions()

        resolved_limits = limits or httpx.Limits()
        self.dns_cache = dns_cache
        self.network_backend = CachingAsyncNetworkBackend(dns_cache=dns_cache)
        self._pool = httpcore.AsyncConnectionPool(
            ssl_context=httpx.create_ssl_context(verify=verify),
            max_connections=resolved_limits.max_connections,
            max_keepalive_connections=resolved_limits.max_keepalive_connections,
            keepalive_expiry=resolved_limits.keepalive_expiry,
            http1=http1,
            http2=http2,
            retries=retries,
            local_address=local_address,
            network_backend=self.network_backend,
            socket_options=socket_options,
        )

    async def handle_async_request(self, request: httpx.Request) -> httpx.Response:
        """Handle a single asynchronous HTTP request.

        :param request: HTTPX request object.
        :returns: HTTPX response wrapping the httpcore response stream.
        :raises TypeError: If the request uses a non-async stream.
        :raises RuntimeError: If the required private HTTPX helpers are unavailable.
        """
        if not isinstance(request.stream, httpx.AsyncByteStream):
            raise TypeError("DNSCachingAsyncHTTPTransport requires an async request stream")

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

        with _map_private_httpcore_exceptions():
            resp = await self._pool.handle_async_request(req)

        response_stream = _build_async_response_stream(resp.stream)

        return httpx.Response(
            status_code=resp.status,
            headers=resp.headers,
            stream=response_stream,
            extensions=resp.extensions,
        )

    async def aclose(self) -> None:
        """Close the underlying httpcore connection pool."""
        await self._pool.aclose()
