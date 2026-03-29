"""
Text preprocessing module for preparing text items for embedding generation.

This module provides functions to preprocess text items by splitting them into
manageable chunks while preserving metadata and language information. It handles
the chunking process using the TextSplitter class and enriches metadata with
language detection and chunking information.

Functions:
    prepare_text_items: Split text items into chunks and prepare them for processing
"""

from .language import LanguageModel
from .search import get_search_prefix, get_search_prefix_length
from .text_splitter import TextSplitter


def _prepare_chunk_by_mode(chunk: str, language: str = "auto", mode: str = "store") -> str:
    """
    Prepare a text chunk based on the specified mode.

    For All MPNetV2 models, the chunk is returned as-is without any modifications.
    This function serves as a placeholder for potential future processing based on different
    modes (e.g., "search" vs "store"), but currently does not alter the chunk.

    Args:
        chunk (str): The input text chunk to be prepared.
        language (str): The language code for the text (e.g., 'en', 'fr').
                         If 'auto', the language will be detected automatically.
        mode (str): The processing mode, either "search" or "store".

    Returns:
        str: The prepared text chunk ready for embedding generation.
    """
    if mode == "search":
        return get_search_prefix(language) + chunk
    return chunk


def _remove_prefix_length(language: str = "auto", mode: str = "store") -> int:
    """
    Get the length of the prefix to remove from token lists based on the specified mode.

    This function returns the length of the prefix that should be removed from token lists
    during postprocessing. It uses precomputed prefix lengths for supported languages
    when in "search" mode, and returns 0 for "store" mode.

    Args:
        language (str): The language code for the text (e.g., 'en', 'fr').
                         If 'auto', it will return the length for English instructions.
        mode (str): The mode for preparing the chunk (e.g., "search", "store").

    Returns:
        int: The length of the prefix to remove from token lists.
    """
    if mode == "search":
        return get_search_prefix_length(language)
    return 0


def prepare_text_items(
    text_items: list,
    language: str = "auto",
    mode: str = "store",
    chunk_size: int = 1000,
    chunk_overlap: int = 200,
) -> list:
    """
    Prepare a batch of text items for processing by splitting them into chunks.

    This function takes a list of text items and splits long texts into smaller chunks
    while preserving metadata. Each chunk is enriched with language information and
    character offset data for downstream processing.

    Args:
        text_items (list): List of dictionaries containing text and metadata.
                          Each item should have 'text' and optional 'metadata' keys.
        language (str, optional): Language code for text items. Defaults to "auto" (detected from text).
        mode (str, optional): The mode for preparing the chunk (e.g., "search", "store"). Defaults to "store".
        chunk_size (int, optional): Maximum characters per chunk. Defaults to 1000.
        chunk_overlap (int, optional): Maximum characters to overlap between chunks.
                                     Defaults to 200.

    Returns:
        list: List of processed text items with chunks. Each item contains:
            - text: The chunk text
            - metadata: Original metadata plus extra fields:
                - offset: Character offset in original text
                - language: Detected language code
                - language_model: LanguageModel instance for the text

    Example:
        >>> items = [{"text": "Long text here...", "metadata": {"id": 1}}]
        >>> chunks = prepare_text_items(items, chunk_size=500, chunk_overlap=100)
        >>> print(len(chunks))  # Number of chunks created
    """

    batch = []
    for item in text_items:
        text = item.get("text", "")
        if not text:
            continue

        # Initialize the TextSplitter with the language model
        language_model = LanguageModel(text=text, language=language)
        splitter = TextSplitter(
            language_model=language_model,
            chunk_size=chunk_size,
            chunk_overlap=chunk_overlap,
        )
        batch.extend(
            [
                {
                    **item,
                    "text": _prepare_chunk_by_mode(chunk=chunk, language=language, mode=mode),
                    "metadata": {
                        **item.get("metadata", {}),
                        "extra": {
                            **item.get("metadata", {}).get("extra", {}),
                            "offset": offset,
                            "language": language_model.language,
                            "language_model": language_model,
                            "remove_prefix_length": _remove_prefix_length(language=language, mode=mode),
                        },
                    },
                }
                for offset, chunk in splitter
            ]
        )

    return batch
