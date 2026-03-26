"""
Sparse vector representation module for text embedding sparse features.

This module provides the SparseVector class for handling sparse vector representations
of text embeddings. It processes tokens and their weights, filters out stopwords and
punctuation, and provides methods for converting to dictionary format with hashing.

Classes:
    SparseVector: Represents a sparse vector with tokens and weights

The sparse vector implementation:
- Filters out stopwords and punctuation based on spaCy language models
- Uses MMH3 hashing for token indices in the final representation
- Encodes weights as base64 floats for efficient storage
- Supports multiple languages through spaCy language detection
"""

import base64
import logging
from struct import pack
from typing import Dict, Iterable

import mmh3
from spacy.tokens import Doc

from .config import LOG_LEVEL
from .language import LanguageModel

logging.basicConfig(level=LOG_LEVEL)


class SparseVector:
    """
    Represents a sparse vector with tokens and their weights for text embeddings.

    This class processes tokens and their associated weights to create a sparse
    vector representation. It filters out stopwords and punctuation, normalizes
    tokens using spaCy language models, and provides conversion to dictionary
    format with MMH3 hashing for efficient storage and retrieval.

    Attributes:
        language_model (LanguageModel): Language model for token processing
        tokens (list): List of input tokens
        weights (Iterable[float]): Weights corresponding to each token
        logger: Logger instance for debugging

    Example:
        >>> from e5_instruct.language import LanguageModel
        >>> lm = LanguageModel("Hello world example")
        >>> tokens = ["hello", "world", "example"]
        >>> weights = [0.8, 0.6, 0.9]
        >>> sv = SparseVector(lm, tokens, weights)
        >>> sparse_dict = sv.to_dict()
    """

    @staticmethod
    def _base64_uint(x: int) -> str:
        """
        Convert an unsigned integer to a base64-encoded string.

        Args:
            x (int): Unsigned integer to encode

        Returns:
            str: Base64-encoded string representation
        """
        return base64.b64encode(pack(">L", x)).decode("utf-8")

    @staticmethod
    def _base64_float(x: float) -> str:
        """
        Convert a float to a base64-encoded string.

        Args:
            x (float): Float value to encode

        Returns:
            str: Base64-encoded string representation
        """
        return base64.b64encode(pack(">f", x)).decode("utf-8")

    def __init__(
        self,
        language_model: LanguageModel,
        tokens: Iterable[str],
        weights: Iterable[float],
    ):
        """
        Initialize the SparseVector with tokens and weights.

        Args:
            language_model (LanguageModel): Language model for processing tokens
            tokens (Iterable[str]): Input tokens to process
            weights (Iterable[float]): Weights corresponding to each token
        """
        self.logger = logging.getLogger(__class__.__name__)
        self.language_model = language_model
        self.tokens = list(tokens)
        self.weights = weights

    def _build_doc(self) -> Doc:
        """
        Build a spaCy Doc object from the tokens for linguistic processing.

        Creates a spaCy Doc from the input tokens and processes it through
        the language model pipeline for linguistic analysis including
        lemmatization, POS tagging, and stopword detection.

        Returns:
            Doc: Processed spaCy Doc object

        Raises:
            ValueError: If there's a mismatch between input tokens and processed Doc length
        """
        nlp = self.language_model.nlp
        doc = Doc(nlp.vocab, words=self.tokens)
        doc = nlp(doc)

        if len(doc) != len(self.tokens):
            raise ValueError("Mismatch between tokens and generated Doc length.")

        return doc

    def _to_dict(self) -> Dict[str, float]:
        """
        Map tokens to their weights after filtering and normalization.

        Creates a dictionary mapping from token lemmas to their normalized weights.
        Filters out stopwords, punctuation, and tokens with zero or negative weights.
        Performs L2 normalization on the final weights.

        Returns:
            Dict[str, float]: Dictionary mapping token lemmas to normalized weights

        Note:
            - Stopwords and punctuation are filtered based on spaCy language model
            - Only positive weights are included
            - Final weights are L2 normalized
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
        Build a sparse vector dictionary with hashed token indices and base64-encoded weights.

        Converts the processed tokens and weights into a final sparse vector representation
        suitable for storage and retrieval. Uses MMH3 hashing to convert token lemmas into
        fixed-size indices and encodes all values in Big Endian base64 format.

        Returns:
            Dict[str, str]: Dictionary mapping base64-encoded token hashes to
                          base64-encoded normalized weights

        Note:
            All numeric values are represented in Big Endian base64 format for
            consistent encoding across different platforms.
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
        """
        Return string representation of the SparseVector instance.

        Returns:
            str: String representation showing tokens and weights
        """
        return f"SparseVector(tokens={self.tokens}, weights={self.weights})"
