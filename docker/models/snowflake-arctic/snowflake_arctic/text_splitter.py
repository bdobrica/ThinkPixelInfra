"""
Text splitting module for chunking natural language text into sentence-based segments.

This module provides the TextSplitter class for intelligently splitting text into
chunks while preserving sentence boundaries and maintaining original formatting.
The splitter supports overlap between chunks and works with multiple languages
through spaCy language models.

Classes:
    TextSplitter: Iterator for splitting text into sentence-based chunks with overlap

Example:
    >>> from snowflake_arctic.language import LanguageModel
    >>> from snowflake_arctic.text_splitter import TextSplitter
    >>>
    >>> text = "First sentence. Second sentence. Third sentence."
    >>> language_model = LanguageModel(text)
    >>> splitter = TextSplitter(language_model, chunk_size=50, chunk_overlap=10)
    >>>
    >>> for offset, chunk in splitter:
    ...     print(f"Offset {offset}: {chunk}")
"""

import logging
from collections.abc import Iterator
from functools import cached_property
from typing import Tuple

from spacy.language import Language

from .config import LOG_LEVEL
from .language import LanguageModel

logging.basicConfig(level=LOG_LEVEL)


class TextSplitter(Iterator):
    """
    Iterator for splitting text into sentence-based chunks with configurable overlap.

    This class intelligently splits text into chunks while preserving sentence boundaries
    and maintaining original formatting (spaces, newlines, tabs). It supports overlap
    between chunks and handles edge cases like single long sentences.

    The chunking algorithm follows these rules:
    1. If chunk_size is 0, return entire text without splitting
    2. If text <= chunk_size, return as single chunk
    3. Split into sentences using spaCy
    4. Build chunks by adding sentences until approaching chunk_size
    5. For subsequent chunks, include overlap from previous sentences
    6. Always include at least one sentence per chunk, even if it exceeds chunk_size

    Args:
        language_model (LanguageModel): Initialized language model for the text
        chunk_size (int): Maximum characters per chunk (default: 1000).
                         Set to 0 to disable splitting and return entire text.
        chunk_overlap (int): Maximum characters to overlap between chunks (default: 200)

    Raises:
        TypeError: If language_model is not an instance of LanguageModel

    Example:
        >>> language_model = LanguageModel("Long text here...")
        >>> splitter = TextSplitter(language_model, chunk_size=500, chunk_overlap=100)
        >>> for offset, chunk in splitter:
        ...     print(f"Chunk at {offset}: {len(chunk)} chars")
        >>>
        >>> # No splitting example
        >>> no_split = TextSplitter(language_model, chunk_size=0)
        >>> for offset, chunk in no_split:
        ...     print(f"Full text at {offset}: {len(chunk)} chars")
    """

    def __init__(self, language_model: LanguageModel, chunk_size: int = 1000, chunk_overlap: int = 200):
        """
        Initialize the TextSplitter with a language model and chunking parameters.

        Args:
            language_model (LanguageModel): Initialized language model for processing text
            chunk_size (int, optional): Maximum characters per chunk. Defaults to 1000.
                                       Set to 0 to disable splitting and return entire text.
            chunk_overlap (int, optional): Maximum characters to overlap between chunks.
                                         Defaults to 200.

        Raises:
            TypeError: If language_model is not an instance of LanguageModel
        """
        self.logger = logging.getLogger(__class__.__name__)

        self.language_model = language_model
        if not isinstance(language_model, LanguageModel):
            raise TypeError("language_model must be an instance of LanguageModel")
        self.chunk_size: int = chunk_size
        self.chunk_overlap: int = chunk_overlap
        self.current_sentence_index = 0

    @cached_property
    def _skip_chunking(self) -> bool:
        """
        Determine if the text requires chunking based on its length and chunking parameters.
        """
        return self.chunk_size < 1 or len(self.language_model.text) <= self.chunk_size

    @cached_property
    def _sentences(self) -> Tuple[Tuple[str, int, int], ...]:
        """
        Splits the text into sentences using the loaded spaCy model.
        Returns tuples of (sentence_text, start_char, end_char) to preserve original spacing.
        """
        try:
            sentences = []
            for sent in self.language_model.doc.sents:
                if sent.text.strip():  # Only include non-empty sentences
                    sentences.append((sent.text, sent.start_char, sent.end_char))
        except ValueError as err:
            self.logger.warning(
                "Failed to split text [%s], length [%d], language [%s] into sentences: %s",
                self.language_model.truncated_text,
                len(self.language_model.text),
                self.language_model.language,
                err,
            )
            # Fallback to entire text as single sentence
            sentences = [(self.language_model.text, 0, len(self.language_model.text))]
        return tuple(sentences)

    def _extract_text_with_separators(self, start_idx: int, end_idx: int) -> str:
        """
        Extract text from start_idx to end_idx sentences, preserving original separators.

        This method extracts the original text with all separators (spaces, newlines, tabs)
        between sentences by using character positions from the spaCy sentence boundaries.

        Args:
            start_idx (int): Starting sentence index (inclusive)
            end_idx (int): Ending sentence index (exclusive)

        Returns:
            str: Text segment with original separators preserved, or empty string if invalid range
        """
        if not self._sentences or start_idx >= len(self._sentences):
            return ""

        end_idx = min(end_idx, len(self._sentences))
        if start_idx >= end_idx:
            return ""

        # Get the character range from first sentence start to last sentence end
        first_sent = self._sentences[start_idx]
        last_sent = self._sentences[end_idx - 1]

        start_char = first_sent[1]  # start_char of first sentence
        end_char = last_sent[2]  # end_char of last sentence

        # Extract the original text with separators
        return self.language_model.text[start_char:end_char]

    def _find_overlap_start(self, current_start: int) -> int:
        """
        Find the starting sentence index for overlap with the previous chunk.

        This method looks backwards from the current position to find sentences that
        can fit within the chunk_overlap limit to provide context between chunks.

        Args:
            current_start (int): Current starting sentence index

        Returns:
            int: Starting sentence index for overlap, or current_start if no overlap possible
        """
        if current_start <= 0:
            return current_start

        # Start from the previous sentence and work backwards
        for overlap_start in range(current_start - 1, -1, -1):
            # Calculate the text length including separators
            overlap_text = self._extract_text_with_separators(overlap_start, current_start)

            if len(overlap_text) <= self.chunk_overlap:
                return overlap_start

        # If no valid overlap found, start from current position
        return current_start

    @cached_property
    def language(self) -> str:
        """
        Detects the language of the text.
        """
        return self.language_model.language

    @cached_property
    def nlp(self) -> Language:
        """
        Loads the spaCy model for the detected language.
        """
        return self.language_model.nlp

    def __iter__(self) -> "TextSplitter":
        """
        Initialize the iterator for chunking text.

        Resets the internal sentence index to start iteration from the beginning.

        Returns:
            TextSplitter: Self reference for iterator protocol
        """
        # Reset start index for iteration
        self.current_sentence_index = 0

        if self._skip_chunking:
            self.logger.info("No chunking needed.")
            return self

        if not self._sentences:
            self.logger.warning("No sentences found in the text.")
            return self

        return self

    def __next__(self) -> Tuple[int, str]:
        """
        Return the next chunk of text with its character offset.

        Implements the core chunking algorithm:
        1. If chunk_size is 0, return the entire text without splitting
        2. If text is smaller than chunk_size, return the entire text
        3. Build chunks by adding sentences until approaching chunk_size
        4. Include overlap from previous chunks when possible
        5. Handle single long sentences that exceed chunk_size

        Returns:
            Tuple[int, str]: A tuple containing:
                - int: Character offset of the chunk in the original text
                - str: The chunk text with preserved separators

        Raises:
            StopIteration: When there are no more chunks to return
        """
        # Check if chunk_size is 0 - no splitting, return entire text
        if self._skip_chunking:
            if self.current_sentence_index > 0:
                raise StopIteration
            self.current_sentence_index = 1
            return 0, self.language_model.text

        if self.current_sentence_index >= len(self._sentences):
            raise StopIteration

        # Find the overlap start position if this is not the first chunk
        if self.current_sentence_index > 0:
            overlap_start = self._find_overlap_start(self.current_sentence_index)
        else:
            overlap_start = 0

        # Build chunk starting from overlap_start
        chunk_start_idx = overlap_start
        chunk_end_idx = overlap_start
        chunk_text = ""

        # Add sentences until we approach chunk_size
        while chunk_end_idx < len(self._sentences):
            # Calculate what the text would be if we include the next sentence
            test_chunk_text = self._extract_text_with_separators(chunk_start_idx, chunk_end_idx + 1)

            # If adding this sentence would exceed chunk_size and we already have at least one sentence
            if len(test_chunk_text) > self.chunk_size and chunk_end_idx > chunk_start_idx:
                break

            # Include this sentence in the chunk
            chunk_text = test_chunk_text
            chunk_end_idx += 1

            # If this was the first sentence and it's already over chunk_size, include it anyway
            if chunk_end_idx == chunk_start_idx + 1 and len(chunk_text) > self.chunk_size:
                break

        # Calculate the character offset for this chunk
        if chunk_start_idx < len(self._sentences):
            chunk_offset = self._sentences[chunk_start_idx][1]
        else:
            chunk_offset = len(self.language_model.text)

        # Update position for next iteration
        # Move to the sentence after the current chunk (ignoring overlap)
        if self.current_sentence_index == 0:
            # For the first chunk, move to the end of the current chunk
            self.current_sentence_index = chunk_end_idx
        else:
            # For subsequent chunks, advance past the overlap region
            # Find the first new sentence that wasn't part of the overlap
            self.current_sentence_index = max(chunk_end_idx, self.current_sentence_index + 1)

        return chunk_offset, chunk_text

    def reset(self) -> "TextSplitter":
        """
        Reset the iterator to start from the beginning.

        This allows the TextSplitter to be reused for multiple iterations
        over the same text without creating a new instance.

        Returns:
            TextSplitter: Self reference for method chaining
        """
        self.current_sentence_index = 0
        return self
