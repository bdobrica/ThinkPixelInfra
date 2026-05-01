"""Sentence-aware text chunking for the external gateway."""

import logging
from collections.abc import Iterator
from functools import cached_property
from typing import Tuple

from .config import LOG_LEVEL
from .language import LanguageModel

logging.basicConfig(level=LOG_LEVEL)


class TextSplitter(Iterator[Tuple[int, str]]):
    def __init__(self, language_model: LanguageModel, chunk_size: int = 1000, chunk_overlap: int = 200):
        self.logger = logging.getLogger(self.__class__.__name__)
        if not isinstance(language_model, LanguageModel):
            raise TypeError("language_model must be an instance of LanguageModel")
        self.language_model = language_model
        self.chunk_size = chunk_size
        self.chunk_overlap = chunk_overlap
        self.current_sentence_index = 0

    @cached_property
    def _skip_chunking(self) -> bool:
        return self.chunk_size < 1 or len(self.language_model.text) <= self.chunk_size

    @cached_property
    def _sentences(self) -> Tuple[Tuple[str, int, int], ...]:
        if self.language_model.language == LanguageModel.UNKNOWN_LANGUAGE:
            return ((self.language_model.text, 0, len(self.language_model.text)),)

        try:
            sentences = []
            for sent in self.language_model.doc.sents:
                if sent.text.strip():
                    sentences.append((sent.text, sent.start_char, sent.end_char))
        except ValueError as err:
            self.logger.warning(
                "Failed to split text [%s], length [%d], language [%s] into sentences: %s",
                self.language_model.truncated_text,
                len(self.language_model.text),
                self.language_model.language,
                err,
            )
            sentences = [(self.language_model.text, 0, len(self.language_model.text))]
        return tuple(sentences)

    def _extract_text_with_separators(self, start_idx: int, end_idx: int) -> str:
        if not self._sentences or start_idx >= len(self._sentences):
            return ""

        end_idx = min(end_idx, len(self._sentences))
        if start_idx >= end_idx:
            return ""

        start_char = self._sentences[start_idx][1]
        end_char = self._sentences[end_idx - 1][2]
        return self.language_model.text[start_char:end_char]

    def _find_overlap_start(self, current_start: int) -> int:
        if current_start <= 0:
            return current_start

        for overlap_start in range(current_start - 1, -1, -1):
            overlap_text = self._extract_text_with_separators(overlap_start, current_start)
            if len(overlap_text) <= self.chunk_overlap:
                return overlap_start

        return current_start

    def __iter__(self) -> "TextSplitter":
        self.current_sentence_index = 0
        return self

    def __next__(self) -> Tuple[int, str]:
        if self._skip_chunking:
            if self.current_sentence_index > 0:
                raise StopIteration
            self.current_sentence_index = 1
            return 0, self.language_model.text

        if self.current_sentence_index >= len(self._sentences):
            raise StopIteration

        overlap_start = self._find_overlap_start(self.current_sentence_index) if self.current_sentence_index > 0 else 0
        chunk_start_idx = overlap_start
        chunk_end_idx = overlap_start
        chunk_text = ""

        while chunk_end_idx < len(self._sentences):
            test_chunk_text = self._extract_text_with_separators(chunk_start_idx, chunk_end_idx + 1)
            if len(test_chunk_text) > self.chunk_size and chunk_end_idx > chunk_start_idx:
                break

            chunk_text = test_chunk_text
            chunk_end_idx += 1

            if chunk_end_idx == chunk_start_idx + 1 and len(chunk_text) > self.chunk_size:
                break

        chunk_offset = (
            self._sentences[chunk_start_idx][1]
            if chunk_start_idx < len(self._sentences)
            else len(self.language_model.text)
        )
        self.current_sentence_index = (
            chunk_end_idx if self.current_sentence_index == 0 else max(chunk_end_idx, self.current_sentence_index + 1)
        )
        return chunk_offset, chunk_text
