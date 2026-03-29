_SEARCH_PREFIX = "Query: "


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
    return _SEARCH_PREFIX
