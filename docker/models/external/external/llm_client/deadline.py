"""Deadline context helpers for :mod:`external.llm_client`.

This module centralizes the request deadline state shared across the client and
framework integration layers.
"""

import contextvars
import time

_deadline_monotonic: contextvars.ContextVar[float | None] = contextvars.ContextVar(
    "llm_deadline_monotonic",
    default=None,
)


def set_deadline_after_ms(timeout_ms: int | None, overhead_ms: int = 0) -> contextvars.Token:
    """Set an absolute deadline from a relative timeout budget.

    :param timeout_ms: Requested timeout in milliseconds, or ``None`` to clear the deadline.
    :param overhead_ms: Milliseconds reserved for framework overhead before the request is sent.
    :returns: Context variable token that can be passed to :func:`reset_deadline`.
    """
    if timeout_ms is None:
        return _deadline_monotonic.set(None)

    usable_ms = max(0, timeout_ms - overhead_ms)
    return _deadline_monotonic.set(time.monotonic() + usable_ms / 1000.0)


def set_deadline_at(deadline_monotonic: float | None) -> contextvars.Token:
    """Store an already-resolved monotonic deadline in the current context.

    :param deadline_monotonic: Absolute monotonic deadline, or ``None`` to clear it.
    :returns: Context variable token for later reset.
    """
    return _deadline_monotonic.set(deadline_monotonic)


def reset_deadline(token: contextvars.Token) -> None:
    """Restore the previous deadline context.

    :param token: Token returned by :func:`set_deadline_after_ms` or :func:`set_deadline_at`.
    """
    _deadline_monotonic.reset(token)


def current_deadline() -> float | None:
    """Return the currently active monotonic deadline.

    :returns: Absolute monotonic deadline, or ``None`` if no deadline is active.
    """
    return _deadline_monotonic.get()


def remaining_budget_seconds() -> float | None:
    """Return the remaining deadline budget in seconds.

    :returns: Remaining time until the active deadline, or ``None`` when no deadline exists.
    """
    deadline = current_deadline()
    if deadline is None:
        return None
    return max(0.0, deadline - time.monotonic())
