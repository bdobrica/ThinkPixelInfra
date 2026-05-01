"""Mode-specific text preparation helpers for the external gateway."""

from .config import MODEL_SEARCH_PREFIX, MODEL_STORE_PREFIX


def get_mode_prefix(language: str = "auto", mode: str = "store") -> str:
    del language
    if mode == "search":
        return MODEL_SEARCH_PREFIX
    return MODEL_STORE_PREFIX


def get_mode_prefix_length(language: str = "auto", mode: str = "store") -> int:
    return len(get_mode_prefix(language=language, mode=mode))
