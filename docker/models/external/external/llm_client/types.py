"""Framework-neutral shared types for :mod:`external.llm_client`."""

from collections.abc import Awaitable, Callable

#: Async callable used to detect whether the original caller has disconnected.
DisconnectChecker = Callable[[], Awaitable[bool]]
