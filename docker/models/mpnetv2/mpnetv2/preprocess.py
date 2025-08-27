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
from .text_splitter import TextSplitter


def prepare_text_items(
    text_items: list,
    chunk_size: int = 1000,
    chunk_overlap: int = 200,
    language: str = "auto",
) -> list:
    """
    Prepare a batch of text items for processing by splitting them into chunks.

    This function takes a list of text items and splits long texts into smaller chunks
    while preserving metadata. Each chunk is enriched with language information and
    character offset data for downstream processing.

    Args:
        text_items (list): List of dictionaries containing text and metadata.
                          Each item should have 'text' and optional 'metadata' keys.
        chunk_size (int, optional): Maximum characters per chunk. Defaults to 1000.
        chunk_overlap (int, optional): Maximum characters to overlap between chunks.
                                     Defaults to 200.
        language (str, optional): Language code for text items. Defaults to "auto" (detected from text).

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
                    "text": chunk,
                    "metadata": {
                        **item.get("metadata", {}),
                        "extra": {
                            **item.get("metadata", {}).get("extra", {}),
                            "offset": offset,
                            "language": language_model.language,
                            "language_model": language_model,
                        },
                    },
                }
                for offset, chunk in splitter
            ]
        )

    return batch
