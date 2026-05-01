"""Framework-neutral shared types for llm_client."""

from collections.abc import Awaitable, Callable

DisconnectChecker = Callable[[], Awaitable[bool]]
