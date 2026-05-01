"""Local lexical sparse vector generation for the external gateway."""

import base64
import logging
import math
from struct import pack

import mmh3

from .config import (
    LOG_LEVEL,
    MODEL_SPARSE_BM25_AVGDL,
    MODEL_SPARSE_BM25_B,
    MODEL_SPARSE_BM25_K1,
    MODEL_SPARSE_MAX_FEATURES,
    MODEL_SPARSE_MIN_TOKEN_LENGTH,
)
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

    def _extract_terms(self) -> tuple[dict[str, int], int]:
        term_frequencies: dict[str, int] = {}
        document_length = 0

        for token in self.language_model.doc:
            lemma = token.lemma_.strip().lower()
            if token.is_stop or token.is_punct or not lemma or lemma == "<unk>":
                continue
            if len(lemma) < MODEL_SPARSE_MIN_TOKEN_LENGTH:
                continue
            term_frequencies[lemma] = term_frequencies.get(lemma, 0) + 1
            document_length += 1

        return term_frequencies, document_length

    def _bm25_weight_map(self) -> dict[str, float]:
        term_frequencies, document_length = self._extract_terms()
        if not term_frequencies or document_length == 0:
            return {}

        normalization = MODEL_SPARSE_BM25_K1 * (
            1.0 - MODEL_SPARSE_BM25_B + MODEL_SPARSE_BM25_B * (document_length / MODEL_SPARSE_BM25_AVGDL)
        )
        weights = {
            lemma: ((frequency * (MODEL_SPARSE_BM25_K1 + 1.0)) / (frequency + normalization))
            for lemma, frequency in term_frequencies.items()
        }

        ranked = sorted(weights.items(), key=lambda item: (-item[1], item[0]))[: self.max_features]

        l2_norm = math.sqrt(sum(weight * weight for _, weight in ranked))
        if l2_norm <= 0.0:
            return {}

        return {lemma: weight / l2_norm for lemma, weight in ranked}

    def _to_weight_map(self) -> dict[str, float]:
        return self._bm25_weight_map()

    def to_dict(self) -> dict[str, str]:
        return {
            self._base64_uint(abs(mmh3.hash(token))): self._base64_float(weight)
            for token, weight in self._to_weight_map().items()
        }
