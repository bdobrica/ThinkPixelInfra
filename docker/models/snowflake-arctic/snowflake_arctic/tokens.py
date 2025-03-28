import base64
import struct
import unicodedata
from functools import reduce
from typing import Dict, Iterable, List, Tuple

import numpy as np

WeightedToken = Tuple[str, float]


def _is_partial(x: str, _: np.ndarray) -> bool:
    """Snowflake-specific check for partial tokens."""
    return len(x) > 1 and ord(x[0]) != 9601


def _clean_token(x: str, w: np.ndarray) -> WeightedToken:
    """Clean token and convert weight to float."""
    if ord(x[0]) == 9601 and len(x) > 1:
        return x[1:], w.item()
    return x, w.item()


def _keep_token(x: str, _: np.ndarray) -> bool:
    """Check if token should be kept."""
    if x in {"<s>", "</s>", "<pad>", "<unk>"}:
        return False
    # Punctuation, Mark, Symbol, Control
    if all(unicodedata.category(item)[0] in "PMSC" for item in x):
        return False
    return True


def _add_tokens(x: WeightedToken, y: WeightedToken) -> WeightedToken:
    """Add two tokens together."""
    return x[0] + y[0], x[1] + y[1]


def _reduce_tokens(
    res: List[WeightedToken], y: Tuple[str, np.ndarray]
) -> List[WeightedToken]:
    """Helper function to reduce tokens/weights zips."""
    if _keep_token(*y):
        if _is_partial(*y):
            _y = _add_tokens(res.pop(), _clean_token(*y))
        else:
            _y = _clean_token(*y)
        res.append(_y)
    return res


def _compress_weighted_tokens(
    res: Dict[str, float], y: WeightedToken
) -> Dict[str, float]:
    y_token = y[0].lower()
    if y_token in res:
        res[y_token] += y[1]
    else:
        res[y_token] = y[1]

    return res


def _pack(item: Dict[str, float]) -> Dict[str, str]:
    """Pack the token weights into a string."""
    return {
        k: base64.b64encode(struct.pack(">f", v)).decode("utf-8")
        for k, v in item.items()
    }


def map_tokens_to_weights(
    tokens: Iterable[List[str]], weights: np.ndarray
) -> List[Dict[str, str]]:
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
            lambda tw: _pack(
                reduce(
                    _compress_weighted_tokens,
                    reduce(_reduce_tokens, tw, []),
                    {},
                )
            ),
            map(lambda item: zip(*item), zip(tokens, weights)),
        )
    )
