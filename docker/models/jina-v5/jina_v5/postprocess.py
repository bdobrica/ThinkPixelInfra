"""
Postprocessing module for building final embedding results from model outputs.

This module processes the raw outputs from the jina-embeddings-v5-text-small embedding model,
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
from typing import Callable, Dict, Iterable, List, Tuple

import numpy as np

from .config import LOG_LEVEL
from .language import LanguageModel
from .sparse_vector import SparseVector

WeightedToken = Tuple[str, float]

logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)

_PAD_TOKEN = None
_SPECIAL_TOKENS = set()


def load_special_tokens(callable: Callable[[], Dict[str, str | List[str]]]) -> None:
    """
    Load special tokens into a global set for filtering during postprocessing.

    This function takes a callable that returns a list of special tokens (e.g., from the tokenizer)
    and populates the global _SPECIAL_TOKENS set. This allows the postprocessing functions to
    efficiently filter out these tokens when building the final sparse vector representation.

    Args:
        callable (Callable[[], Dict[str, str | List[str]]]): A function that returns a dictionary of special token
            strings.
    Returns:
        None: This function does not return anything. It updates the global _SPECIAL_TOKENS set with the provided
            tokens.
    """
    global _PAD_TOKEN, _SPECIAL_TOKENS

    if not _SPECIAL_TOKENS:
        token_map = callable()
        special_tokens = [[token] if isinstance(token, str) else token for token in token_map.values()]

        _PAD_TOKEN = token_map.get("pad_token")
        _SPECIAL_TOKENS.update(token for tokens in special_tokens for token in tokens)


def _l2_normalize(x: np.ndarray, axis: int = -1, eps: float = 1e-12) -> np.ndarray:
    return x / (np.linalg.norm(x, axis=axis, keepdims=True) + eps)


def _is_partial(x: str, _: np.ndarray) -> bool:
    """
    Check if a token is a partial/continuation token in jina-embeddings-v5-text-small tokenization.

    Partial tokens are identified by not starting with the special character (▁).
    These tokens are typically continuations of words split across multiple tokens.

    Args:
        x (str): Token string to check
        _ (np.ndarray): Weight array (unused but kept for function signature consistency)

    Returns:
        bool: True if token is partial (continuation), False otherwise
    """
    return len(x) > 1 and ord(x[0]) != 288


def _clean_token(x: str, w: np.ndarray) -> WeightedToken:
    """
    Clean a token by removing the jina-embeddings-v5-text-small prefix and convert weight to float.

    Removes the leading Ġ character (Unicode 288) if present and converts
    the numpy weight array to a Python float.

    Args:
        x (str): Raw token string
        w (np.ndarray): Weight array from model output

    Returns:
        WeightedToken: Tuple of (cleaned_token, weight_float)
    """
    if ord(x[0]) == 288 and len(x) > 1:
        return x[1:], w.item()
    return x, w.item()


def _add_tokens(x: WeightedToken, y: WeightedToken) -> WeightedToken:
    """Add two tokens together. Keep weight L2 normalized."""
    return x[0] + y[0], np.sqrt(x[1] * x[1] + y[1] * y[1]).item()


def _keep_token(x: str, _: np.ndarray) -> bool:
    """Check if token should be kept."""
    return x not in _SPECIAL_TOKENS


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

    This function combines the outputs from the jina-embeddings-v5-text-small embedding model
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
        # we don't actually use remove_prefix_length as GPT tokenizers are not stable for prefix length calculation
        _: int = extra.pop("remove_prefix_length", 0)

        if not isinstance(language_model, LanguageModel):
            continue

        # Find the first token index that starts with the special character (Unicode 288)
        first_token_idx = 0
        while first_token_idx < len(tokens) and ord(tokens[first_token_idx][0]) != 288:
            first_token_idx += 1

        # Find the last token index that is not a padding token
        last_token_idx = len(tokens)
        if _PAD_TOKEN and _PAD_TOKEN in tokens:
            last_token_idx = tokens.index(_PAD_TOKEN)

        if first_token_idx > last_token_idx:
            logger.warning("No valid tokens found for text %s. Skipping.", text_item.get("text", "")[:50])
            continue

        # The GPT tokenizer doesn't produce end-of-word tokens, so we use the token embedding together with the sentence
        # embedding to compute sparse weights for all tokens.
        sparse_weights = token_embeddings @ dense_vector.T

        extra["language"] = language_model.language

        # Remove prefix tokens from token list and sparse weights if in search mode to avoid including instruction
        # tokens in the sparse vector representation.
        tokens = tokens[first_token_idx:last_token_idx]
        sparse_weights = sparse_weights[first_token_idx:last_token_idx]

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
