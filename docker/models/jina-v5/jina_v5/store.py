_STORE_PREFIX = "Document: "


def get_store_prefix(language: str = "auto") -> str:
    """
    Get the store instruction prefix based on the specified language.

    This function retrieves a predefined instruction prefix for document storage based on
    the provided language code. If the language code is not recognized, it defaults to
    English instructions.

    Args:
        language (str): The language code for the instructions (e.g., 'en', 'fr').
                        If 'auto', it will default to English instructions.

    Returns:
        str: The instruction prefix for document storage in the specified language.
    """
    return _STORE_PREFIX
