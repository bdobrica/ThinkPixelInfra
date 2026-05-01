"""Local lexical sparse vector generation for the external gateway."""

from __future__ import annotations

import base64
import logging
from struct import pack

import mmh3

from .config import LOG_LEVEL, MODEL_SPARSE_MAX_FEATURES, MODEL_SPARSE_MIN_TOKEN_LENGTH
from .language import LanguageModel

logging.basicConfig(level=LOG_LEVEL)


class SparseVector:
    @staticmethod
    def _base64_uint(value: int) -> str:
        return base64.b64encode(pack(">L", value)).decode("utf-8")

    @staticmethod
    def _base64_float(value: float) -> str:
        return base64.b64encode(pack(">f", value)).decode("utf-8")

    def __init__(self, language_model: LanguageModel, max_features: int = MODEL_SPARSE_MAX_FEATURES) -> None:
        self.language_model = language_model
        self.max_features = max_features

    def _to_weight_map(self) -> dict[str, float]:
        weights: dict[str, float] = {}
        first_positions: dict[str, int] = {}

        for position, token in enumerate(self.language_model.doc):
            lemma = token.lemma_.strip().lower()
            if token.is_stop or token.is_punct or not lemma or lemma == "<unk>":
                continue
            if len(lemma) < MODEL_SPARSE_MIN_TOKEN_LENGTH:
                continue
            weights[lemma] = weights.get(lemma, 0.0) + 1.0
            first_positions.setdefault(lemma, position)

        ranked = sorted(
            ((lemma, count + (1.0 / (1.0 + first_positions[lemma]))) for lemma, count in weights.items()),
            key=lambda item: (-item[1], item[0]),
        )[: self.max_features]

        total_weight = sum(weight for _, weight in ranked)
        if total_weight <= 0:
            return {}

        return {lemma: weight / total_weight for lemma, weight in ranked}

    def to_dict(self) -> dict[str, str]:
        return {
            self._base64_uint(abs(mmh3.hash(token))): self._base64_float(weight)
            for token, weight in self._to_weight_map().items()
        }
