"""Optional DNS-caching transport internals for llm_client."""

import asyncio
import dataclasses
import ipaddress
import socket
import time
from collections.abc import Sequence
from typing import Any, cast

import httpcore
import httpx
from httpcore._backends.auto import AutoBackend as HttpcoreAutoBackend
from httpx._transports.default import AsyncResponseStream, map_httpcore_exceptions

SocketOptionValue = int | bytes
SocketOption = tuple[int, int, SocketOptionValue]
SocketOptions = Sequence[SocketOption]


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
        socket_options: SocketOptions | None = None,
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
        socket_options: SocketOptions | None = None,
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
        socket_options: SocketOptions | None = None,
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

        response_stream = cast(httpx.AsyncByteStream, AsyncResponseStream(resp.stream))

        return httpx.Response(
            status_code=resp.status,
            headers=resp.headers,
            stream=response_stream,
            extensions=resp.extensions,
        )

    async def aclose(self) -> None:
        await self._pool.aclose()
