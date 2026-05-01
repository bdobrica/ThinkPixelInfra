"""Text preprocessing for the external embeddings gateway."""

from __future__ import annotations

from .language import LanguageModel
from .search import get_mode_prefix, get_mode_prefix_length
from .text_splitter import TextSplitter


def _prepare_chunk_by_mode(chunk: str, language: str = "auto", mode: str = "store") -> str:
    return get_mode_prefix(language=language, mode=mode) + chunk


def prepare_text_items(
    text_items: list,
    language: str = "auto",
    mode: str = "store",
    chunk_size: int = 1000,
    chunk_overlap: int = 200,
) -> list:
    batch = []
    for item in text_items:
        text = item.get("text", "")
        if not text:
            continue

        language_model = LanguageModel(text=text, language=language)
        resolved_language = language_model.language
        splitter = TextSplitter(
            language_model=language_model,
            chunk_size=chunk_size,
            chunk_overlap=chunk_overlap,
        )

        for offset, chunk in splitter:
            batch.append(
                {
                    **item,
                    "text": _prepare_chunk_by_mode(chunk=chunk, language=resolved_language, mode=mode),
                    "metadata": {
                        **item.get("metadata", {}),
                        "extra": {
                            **item.get("metadata", {}).get("extra", {}),
                            "offset": offset,
                            "language": resolved_language,
                            "sparse_text": chunk,
                            "remove_prefix_length": get_mode_prefix_length(language=resolved_language, mode=mode),
                        },
                    },
                }
            )

    return batch
