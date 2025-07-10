import base64
import logging
import re
import time
from struct import pack
from typing import Dict, Iterable

import mmh3
from fast_langdetect import detect
from spacy.tokens import Doc

from .config import LOG_LEVEL
from .language import load_spacy_model

logging.basicConfig(level=LOG_LEVEL)


class SparseVector:
    """
    Represents a sparse vector with tokens and their weights.
    """

    LANG_DETECT_MAX_LENGTH = 100

    @staticmethod
    def _base64_uint(x: int) -> str:
        """
        Converts an unsigned integer to a base64 string.
        """
        return base64.b64encode(pack(">L", x)).decode("utf-8")

    @staticmethod
    def _base64_float(x: float) -> str:
        """
        Converts a float to a base64 string.
        """
        return base64.b64encode(pack(">f", x)).decode("utf-8")

    def __init__(self, tokens: Iterable[str], weights: Iterable[float]):
        self.logger = logging.getLogger(__class__.__name__)
        self.tokens = list(tokens)
        self.weights = weights
        self.language = self._detect_language(" ".join(self.tokens))

    def _prepare_text_for_language_detection(self, text: str) -> str:
        text_ = re.sub(r"\s+", " ", text.lower().strip())
        text_ = text_[: self.LANG_DETECT_MAX_LENGTH]
        last_space_pos = text_.rfind(" ")
        if last_space_pos != -1:
            text_ = text_[:last_space_pos]
        return text_

    def _detect_language(self, text: str) -> str:
        """
        Detects the language of the given text.
        Expects fast_langdetect.detect to return a dict with 'lang' and 'score'.
        """
        start_time = time.perf_counter()
        text_ = self._prepare_text_for_language_detection(text)
        result = detect(text_)
        elapsed_time = time.perf_counter() - start_time
        score = float(result.get("score", 0))
        lang = str(result.get("lang", "unk"))

        self.logger.debug(
            "Detected language: %s with score: %f (time: %f)",
            lang,
            score,
            elapsed_time,
        )

        if score > 0.2:
            return lang

        return "unk"  # fallback to unknown if score is low

    def _build_doc(self) -> Doc:
        """
        Builds a spaCy Doc object from the given tokens.
        """
        nlp = load_spacy_model(self.language)
        doc = Doc(nlp.vocab, words=self.tokens)
        doc = nlp(doc)

        if len(doc) != len(self.tokens):
            raise ValueError("Mismatch between tokens and generated Doc length.")

        return doc

    def _to_dict(self) -> Dict[str, float]:
        """
        Maps tokens to their weights.
        Filters out stop words, punctuation, and zero weights.
        Normalizes the weights.
        """
        doc = self._build_doc()

        res = {}
        total_weight = 0
        for token, weight in zip(doc, self.weights):
            if token.is_stop or token.is_punct or 0 >= weight:
                continue
            lemma = token.lemma_.lower()
            res[lemma] = res.get(lemma, 0) + weight
            total_weight += weight

        # Normalize weights
        for token in res:
            res[token] /= total_weight
            self.logger.debug("Token: %s, Weight: %f", token, res[token])

        return res

    def to_dict(self) -> Dict[str, str]:
        """
        Builds a sparse vector from the given tokens and weights.
        Uses mmh3 to hash the tokens into a fixed-size vector.
        All numbers are represented in Big Endian base64 format.
        """
        token_weights = self._to_dict()
        return dict(
            map(
                lambda x: (
                    self._base64_uint(abs(mmh3.hash(x[0]))),
                    self._base64_float(x[1]),
                ),
                token_weights.items(),
            )
        )

    def __repr__(self):
        return f"SparseVector(tokens={self.tokens}, weights={self.weights})"
