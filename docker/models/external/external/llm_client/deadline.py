"""Deadline context helpers for llm_client."""

import contextvars
import time

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
