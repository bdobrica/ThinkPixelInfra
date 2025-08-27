"""
Language processing module for multi-language text analysis and NLP operations.

This module provides the LanguageModel class for handling multi-language text processing
using spaCy and fast-langdetect. It supports automatic language detection, sentence
segmentation, and text preprocessing for over 50 languages.

Classes:
    UnknownLanguage: Placeholder language class for unsupported languages
    LanguageModel: Main language processing class with spaCy model management

Key Features:
- Automatic language detection using fast-langdetect
- Cached spaCy model loading for performance
- Support for 50+ languages with fallback to unknown language handling
- Sentence segmentation and tokenization
- Punctuation detection and normalization
- Efficient text preprocessing for embedding generation

Dependencies:
    - spacy: Core NLP library for language processing
    - fast-langdetect: Fast language detection library
    - Various spaCy language models (downloaded separately)
"""

import logging
import re
import time
import unicodedata
from functools import cached_property, lru_cache
from typing import Dict, Union

import fasttext
import spacy
from spacy.tokens import Doc
from spacy.vocab import Vocab

from .config import (
    LOG_LEVEL,
    MODEL_LANGUAGE_DETECTION_PATH,
    MODEL_LANGUAGES,
    SPACY_LANGUAGE_MODELS,
)

# Setup logging
logging.basicConfig(level=LOG_LEVEL)


class UnknownLanguage(spacy.language.Language):
    """
    A placeholder class for unknown languages.
    It provides basic functionality without any specific language processing.
    """

    def __init__(self):
        self.vocab = Vocab()

    def _is_punct(self, token: str) -> bool:
        # List of Unicode categories that are considered punctuation
        # https://en.wikipedia.org/wiki/Template:General_Category_(Unicode)
        return all(unicodedata.category(item)[0] in "MPSZC" for item in token)

    def _get_lemma(self, token: str) -> str:
        token = token.strip().lower()
        # Remove leading and trailing punctuation characters
        while token and unicodedata.category(token[0])[0] in "MPSZC":
            token = token[1:]
        while token and unicodedata.category(token[-1])[0] in "MPSZC":
            token = token[:-1]
        token = token or "<unk>"
        self.vocab.strings.add(token)
        return token

    def __call__(self, text: Union[str, Doc], **kwargs) -> Doc:
        doc: Doc
        if isinstance(text, str):
            words = text.split()
            lemmas = [self._get_lemma(word) for word in words]
            doc = Doc(self.vocab, words=words, lemmas=lemmas)
        elif isinstance(text, Doc):
            doc = text
        else:
            raise ValueError("Input must be a string or a spaCy Doc.")

        for token in doc:
            lemma = self._get_lemma(token.text)
            token.lemma = self.vocab.strings[lemma]
            token.lex.is_punct = self._is_punct(token.text)
        return doc


class LanguageDetector:
    LANG_DETECT_MAX_LENGTH = 100

    @staticmethod
    @lru_cache(maxsize=None)
    def load():
        return fasttext.load_model(MODEL_LANGUAGE_DETECTION_PATH)

    @staticmethod
    def prepare_text_for_language_detection(text: str) -> str:
        text_ = re.sub(r"\s+", " ", text.lower().strip())
        text_ = text_[: LanguageDetector.LANG_DETECT_MAX_LENGTH]
        last_space_pos = text_.rfind(" ")
        if last_space_pos != -1:
            text_ = text_[:last_space_pos]
        return text_

    @staticmethod
    def detect(text: str) -> Dict[str, Union[str, float]]:
        model = LanguageDetector.load()
        text = LanguageDetector.prepare_text_for_language_detection(text)
        predictions = model.predict(text, k=1)
        if not predictions or not predictions[0]:
            return {}

        lang = predictions[0][0].removeprefix("__label__")
        score = predictions[1][0]
        return {"lang": lang, "score": score}


class LanguageModel:
    # Unknown language model instance
    UNKNOWN_LANGUAGE = "unk"

    # Define the allowed languages and corresponding spaCy small models.
    ALLOWED_LANGUAGES = {lang: SPACY_LANGUAGE_MODELS[lang] for lang in MODEL_LANGUAGES if lang in SPACY_LANGUAGE_MODELS}

    @staticmethod
    @lru_cache(maxsize=None)
    def load(lang: str) -> Union[spacy.language.Language, UnknownLanguage]:
        """
        Loads the spaCy model for the given language.
        """
        if lang == LanguageModel.UNKNOWN_LANGUAGE:
            return UnknownLanguage()

        if lang not in LanguageModel.ALLOWED_LANGUAGES:
            raise ValueError(f"Unsupported language: {lang}")

        model_name = LanguageModel.ALLOWED_LANGUAGES[lang]
        try:
            nlp = spacy.load(model_name)
        except OSError:
            raise RuntimeError(f"Failed to load spaCy model: {model_name}")

        return nlp

    def __init__(self, text: str, language: str = "auto"):
        """
        Initializes the LanguageModel with the given text.
        If language is "auto", detects the language and loads the corresponding spaCy model.
        """
        self.text = text
        self.language_ = language
        self.logger = logging.getLogger(self.__class__.__name__)

    @cached_property
    def _detect_language_model(self) -> fasttext.FastText._FastText:
        model = fasttext.load_model(MODEL_LANGUAGE_DETECTION_PATH)
        return model

    def _detect_language(self) -> str:
        """
        Detects the language of the given text.
        Expects fast_langdetect.detect to return a dict with 'lang' and 'score'.
        """
        start_time = time.perf_counter()
        result = LanguageDetector.detect(self.text)
        elapsed_time = time.perf_counter() - start_time

        score = float(result.get("score", 0))
        lang = str(result.get("lang", self.UNKNOWN_LANGUAGE))

        self.logger.debug(
            "Detected language for text [%s]: %s with score: %f (time: %f)",
            self.truncated_text,
            lang,
            score,
            elapsed_time,
        )

        if score < 0.2:
            self.logger.warning(
                "Detected language for text [%s]: %s with low score: %f, falling back to `%s`",
                self.truncated_text,
                lang,
                score,
                self.UNKNOWN_LANGUAGE,
            )
            return self.UNKNOWN_LANGUAGE
        elif lang not in self.ALLOWED_LANGUAGES:
            self.logger.warning(
                "Detected language for text [%s]: %s is not in allowed languages: %s, falling back to `%s`",
                self.truncated_text,
                lang,
                self.ALLOWED_LANGUAGES.keys(),
                self.UNKNOWN_LANGUAGE,
            )
            return self.UNKNOWN_LANGUAGE
        elif lang == self.UNKNOWN_LANGUAGE:
            self.logger.warning(
                "Detected language for text [%s] is set to `%s` as there is no valid language detected.",
                self.truncated_text,
                self.UNKNOWN_LANGUAGE,
            )
        return lang

    @cached_property
    def truncated_text(self) -> str:
        """
        Returns the text truncated to the maximum length for language detection.
        """
        truncated_text = LanguageDetector.prepare_text_for_language_detection(self.text)
        if len(truncated_text) < len(self.text):
            truncated_text += "..."
        return truncated_text

    @cached_property
    def language(self) -> str:
        """
        Detects the language of the text using fast_langdetect.
        """
        if self.language_ == "auto":
            return self._detect_language()

        if self.language_ in self.ALLOWED_LANGUAGES:
            return self.language_
        else:
            self.logger.warning(
                "Provided language `%s` is not in allowed languages: %s, falling back to `%s`",
                self.language_,
                self.ALLOWED_LANGUAGES.keys(),
                self.UNKNOWN_LANGUAGE,
            )
            return self.UNKNOWN_LANGUAGE

    @cached_property
    def nlp(self) -> spacy.language.Language:
        """
        Loads the spaCy model for the detected language.
        """
        start_time = time.perf_counter()
        nlp = self.load(self.language)
        elapsed_time = time.perf_counter() - start_time
        self.logger.debug(
            "Loaded spaCy model: %s (time: %f)",
            self.language,
            elapsed_time,
        )
        return nlp

    @cached_property
    def doc(self) -> Doc:
        """
        Processes the text with the loaded spaCy model.
        Returns a spaCy Doc object.
        """
        if not self.nlp:
            raise RuntimeError("Language model is not loaded.")
        doc = self.nlp(self.text)
        if not doc:
            raise ValueError("Failed to process text with spaCy model.")
        return doc
