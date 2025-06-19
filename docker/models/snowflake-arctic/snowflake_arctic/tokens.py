from functools import reduce
from typing import Iterable, List, Tuple

import numpy as np

from .sparse_vector import SparseVector

WeightedToken = Tuple[str, float]


def _is_partial(x: str, _: np.ndarray) -> bool:
    """Snowflake-specific check for partial tokens."""
    return len(x) > 1 and ord(x[0]) != 9601


def _clean_token(x: str, w: np.ndarray) -> WeightedToken:
    """Clean token and convert weight to float."""
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


def build_batch_sparse_vectors(tokens: Iterable[List[str]], weights: np.ndarray) -> List[SparseVector]:
    """
    Map tokens to weights.

    Args:
        tokens (List[List[str]]): List of lists of tokens.
        weights (np.ndarray): Tensor of weights.

    Returns:
        List[Dict[str, str]]: List of dictionaries of token weights, with the
        weights packed as base64 strings.
    """
    return list(
        map(
            lambda tw: SparseVector(*zip(*reduce(_reduce_tokens, tw, []))),
            map(lambda item: zip(*item), zip(tokens, weights)),
        )
    )
