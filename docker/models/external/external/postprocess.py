"""Result shaping for the external embeddings gateway."""

from __future__ import annotations

import base64
from typing import Sequence

import numpy as np

from .config import MODEL_SPARSE_STRATEGY
from .language import LanguageModel
from .sparse_vector import SparseVector


def _encode_dense_vector(vector: Sequence[float]) -> str:
    return base64.b64encode(np.asarray(vector, dtype=np.float32).flatten().astype(">f4").tobytes()).decode("utf-8")


def build_results(
    text_items: list[dict], dense_vectors: Sequence[Sequence[float]], sparse_strategy: str = MODEL_SPARSE_STRATEGY
) -> list[dict]:
    if len(text_items) != len(dense_vectors):
        raise ValueError("Prepared text items and dense vector counts do not match.")

    results = []
    for text_item, dense_vector in zip(text_items, dense_vectors):
        metadata = dict(text_item.get("metadata", {}))
        extra = dict(metadata.get("extra", {}))
        metadata["extra"] = extra

        offset = int(extra.pop("offset", 0))
        language = str(extra.get("language", LanguageModel.UNKNOWN_LANGUAGE))
        sparse_text = str(extra.pop("sparse_text", text_item.get("text", "")))
        extra.pop("remove_prefix_length", None)
        extra["language"] = language

        if sparse_strategy in {"bm25", "lexical"}:
            sparse_vector = SparseVector(LanguageModel(text=sparse_text, language=language)).to_dict()
        elif sparse_strategy == "off":
            sparse_vector = {}
        else:
            raise ValueError(f"Unsupported sparse strategy: {sparse_strategy}")

        results.append(
            {
                "text": text_item.get("text", ""),
                "offset": offset,
                "dense_vector": _encode_dense_vector(dense_vector),
                "sparse_vector": sparse_vector,
                "metadata": metadata,
            }
        )

    return results
