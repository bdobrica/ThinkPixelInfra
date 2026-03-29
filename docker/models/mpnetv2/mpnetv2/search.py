from typing import Callable

from .config import SPACY_LANGUAGE_MODELS

_SEARCH_PREFIX_LENGTH = {}


def compute_search_prefix_length(callback: Callable[[str], int]) -> None:
    """
    Compute and store the length of the search instruction prefix for each supported language.
    This function uses a provided callback to compute the length of the search instruction prefix
    for each language and stores the results in a global dictionary for later use during postprocessing.
    The callback should take a string input (the search prefix) and return its length as an integer.

    Args:
        callback (Callable[[str], int]): A function that takes a string input and returns its length as an integer.

    Returns:
        None: This function does not return anything. It updates the global _SEARCH_PREFIX_LENGTH dictionary
              with the computed lengths for each supported language.
    """
    global _SEARCH_PREFIX_LENGTH

    if not _SEARCH_PREFIX_LENGTH:
        _SEARCH_PREFIX_LENGTH.update(
            {language: callback(get_search_prefix(language)) for language in ["auto", *SPACY_LANGUAGE_MODELS.keys()]}
        )


def get_search_prefix(language: str = "auto") -> str:
    """
    Get the search instruction prefix based on the specified language.

    This function retrieves a predefined instruction prefix for search queries based on
    the provided language code. If the language code is not recognized, it defaults to
    English instructions.

    Args:
        language (str): The language code for the instructions (e.g., 'en', 'fr').
                        If 'auto', it will default to English instructions.

    Returns:
        str: The instruction prefix for search queries in the specified language.
    """
    return ""


def get_search_prefix_length(language: str = "auto") -> int:
    """
    Get the length of the search instruction prefix for the specified language.

    This function returns the length of the search instruction prefix, which is useful
    for removing the prefix from token lists during postprocessing. It relies on a
    precomputed dictionary of prefix lengths for supported languages.

    Args:
        language (str): The language code for the instructions (e.g., 'en', 'fr').
                        If 'auto', it will return the length for English instructions.
    Returns:
        int: The length of the search instruction prefix for the specified language.
    """
    if not _SEARCH_PREFIX_LENGTH:
        raise RuntimeError("Search prefix lengths have not been computed. Call compute_search_prefix_length() first.")

    return _SEARCH_PREFIX_LENGTH.get(language, _SEARCH_PREFIX_LENGTH["auto"])
