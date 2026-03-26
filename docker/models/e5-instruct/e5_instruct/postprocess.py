"""
Postprocessing module for building final embedding results from model outputs.

This module processes the raw outputs from the Multi-Language E5 Instruct embedding model,
combining dense vectors, sparse vectors, and metadata into the final result format.
It handles token processing, weight reduction, and encoding of vectors for storage.

Types:
    WeightedToken: Tuple of (token, weight) for sparse vector processing

Functions:
    build_results: Main function to process model outputs into final results

Helper Functions:
    _is_partial: Check if a token is a partial/continuation token
    _clean_token: Clean and convert token with weight
    _add_tokens: Combine two weighted tokens
    _keep_token: Filter out special tokens
    _reduce_tokens: Reduce token/weight pairs into final sparse representation
"""

import base64
import logging
from functools import reduce
from typing import Iterable, List, Tuple

import numpy as np

from .config import LOG_LEVEL
from .language import LanguageModel
from .sparse_vector import SparseVector

WeightedToken = Tuple[str, float]

logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)


def _l2_normalize(x: np.ndarray, axis: int = -1, eps: float = 1e-12) -> np.ndarray:
    return x / (np.linalg.norm(x, axis=axis, keepdims=True) + eps)


def _is_partial(x: str, _: np.ndarray) -> bool:
    """
    Check if a token is a partial/continuation token in Multi-Language E5 Instruct tokenization.

    Partial tokens are identified by not starting with the special character (▁).
    These tokens are typically continuations of words split across multiple tokens.

    Args:
        x (str): Token string to check
        _ (np.ndarray): Weight array (unused but kept for function signature consistency)

    Returns:
        bool: True if token is partial (continuation), False otherwise
    """
    return len(x) > 1 and ord(x[0]) != 9601


def _clean_token(x: str, w: np.ndarray) -> WeightedToken:
    """
    Clean a token by removing the Multi-Language E5 Instruct prefix and convert weight to float.

    Removes the leading ▁ character (Unicode 9601) if present and converts
    the numpy weight array to a Python float.

    Args:
        x (str): Raw token string
        w (np.ndarray): Weight array from model output

    Returns:
        WeightedToken: Tuple of (cleaned_token, weight_float)
    """
    if ord(x[0]) == 9601 and len(x) > 1:
        return x[1:], w.item()
    return x, w.item()


def _add_tokens(x: WeightedToken, y: WeightedToken) -> WeightedToken:
    """Add two tokens together."""
    return x[0] + y[0], x[1] + y[1]


def _keep_token(x: str, _: np.ndarray) -> bool:
    """Check if token should be kept."""
    return x not in {"<s>", "</s>", "<pad>", "<unk>"}


def _reduce_tokens(res: List[WeightedToken], y: Tuple[str, np.ndarray]) -> List[WeightedToken]:
    """Helper function to reduce tokens/weights zips."""
    if _keep_token(*y):
        if _is_partial(*y):
            _y = _add_tokens(res.pop(), _clean_token(*y))
        else:
            _y = _clean_token(*y)
        res.append(_y)
    return res


def build_results(
    text_items: List[dict],
    batch_tokens: Iterable[List[str]],
    model_output: Tuple[np.ndarray, np.ndarray],
) -> List[dict]:
    """
    Build final embedding results from model outputs and text items.

    This function combines the outputs from the Multi-Language E5 Instruct embedding model
    (dense vectors and sparse weights) with the original text items and their metadata
    to produce the final embedding results.

    Args:
        text_items (List[dict]): List of text items with metadata from preprocessing
        batch_tokens (Iterable[List[str]]): Tokenized text for each item
        model_output (Tuple[np.ndarray, np.ndarray]): Tuple containing:
            - Dense vectors array of shape (batch_size, embedding_dim)
            - Sparse weights array of shape (batch_size, vocab_size)

    Returns:
        List[dict]: List of result dictionaries, each containing:
            - text: Original chunk text
            - offset: Character offset extracted from metadata.extra
            - dense_vector: Base64-encoded dense embedding vector
            - sparse_vector: Dictionary representation of sparse vector
            - metadata: Original metadata with language information (offset removed)

    Note:
        Empty input returns empty list. Items without valid LanguageModel
        in metadata are skipped.

    Example:
        >>> items = [{"text": "Hello world", "metadata": {"extra": {"language_model": lm}}}]
        >>> tokens = [["Hello", "world"]]
        >>> dense, sparse = model.inference(items)
        >>> results = build_results(items, tokens, (dense, sparse))
    """
    if not text_items:
        return []

    batch_token_embeddings = _l2_normalize(model_output[0].astype(np.float32), axis=-1)
    batch_dense_vectors = _l2_normalize(model_output[1].astype(np.float32), axis=-1)

    results = []
    for text_item, dense_vector, tokens, token_embeddings in zip(
        text_items,
        batch_dense_vectors,
        batch_tokens,
        batch_token_embeddings,
    ):
        metadata: dict = text_item.get("metadata", {})
        extra: dict = metadata.get("extra", {})
        offset: int = extra.pop("offset", 0)
        language_model = extra.pop("language_model", None)
        remove_prefix_length: int = extra.pop("remove_prefix_length", 0)

        if not isinstance(language_model, LanguageModel):
            continue

        if "</s>" in tokens:
            # Use delimiter token embedding as dense vector for sparse weight calculation
            # This is a heuristic to identify important tokens based on their similarity to the delimiter token.
            delimiter_idx = tokens.index("</s>")
            delimiter_embedding = token_embeddings[delimiter_idx]
            sparse_weights = token_embeddings @ delimiter_embedding
        else:
            # If no delimiter token, use sentence embedding as dense vector for sparse weight calculation
            # This is a fallback heuristic that uses the overall sentence embedding to identify important tokens.
            logger.warning("Delimiter token '</s>' not found. Using sentence embedding for sparse weight calculation.")
            sparse_weights = token_embeddings @ dense_vector

        extra["language"] = language_model.language

        # Remove prefix tokens from token list and sparse weights if in search mode to avoid including instruction
        # tokens in the sparse vector representation.
        if remove_prefix_length > 0:
            tokens = tokens[remove_prefix_length:]
            sparse_weights = sparse_weights[remove_prefix_length:]

        tokens, sparse_weights = zip(
            *reduce(
                _reduce_tokens,
                zip(tokens, sparse_weights),
                [],
            )
        )

        sparse_vector = SparseVector(
            language_model=language_model,
            tokens=tokens,
            weights=sparse_weights,
        )

        results.append(
            {
                "text": text_item.get("text", ""),
                "offset": offset,
                "dense_vector": base64.b64encode(dense_vector.flatten().astype(">f4").tobytes()).decode("utf-8"),
                "sparse_vector": sparse_vector.to_dict(),
                "metadata": metadata,
            }
        )

    return results
