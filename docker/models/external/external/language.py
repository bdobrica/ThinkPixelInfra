"""Language detection and NLP helpers for the external gateway."""

import logging
import re
import unicodedata
from functools import cached_property, lru_cache
from typing import Any, Dict, Union

import spacy
from spacy.tokens import Doc
from spacy.vocab import Vocab

try:
    import fasttext
except ImportError:  # pragma: no cover - optional runtime dependency in local dev
    fasttext = None  # type: ignore[assignment]

from .config import (
    LOG_LEVEL,
    MODEL_LANGUAGE_DETECTION_PATH,
    MODEL_LANGUAGES,
    SPACY_LANGUAGE_MODELS,
)

logging.basicConfig(level=LOG_LEVEL)


class UnknownLanguage(spacy.language.Language):
    def __init__(self) -> None:
        self.vocab = Vocab()

    def _is_punct(self, token: str) -> bool:
        return all(unicodedata.category(item)[0] in "MPSZC" for item in token)

    def _get_lemma(self, token: str) -> str:
        token = token.strip().lower()
        while token and unicodedata.category(token[0])[0] in "MPSZC":
            token = token[1:]
        while token and unicodedata.category(token[-1])[0] in "MPSZC":
            token = token[:-1]
        token = token or "<unk>"
        self.vocab.strings.add(token)
        return token

    def __call__(self, text: Union[str, Doc], **kwargs: Any) -> Doc:
        del kwargs
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
    def load() -> Any | None:
        if fasttext is None:
            logging.getLogger(__name__).warning(
                "fasttext package is unavailable; falling back to unknown language.",
            )
            return None
        try:
            return fasttext.load_model(MODEL_LANGUAGE_DETECTION_PATH)
        except Exception:
            logging.getLogger(__name__).warning(
                "Language detection model unavailable at %s; falling back to unknown language.",
                MODEL_LANGUAGE_DETECTION_PATH,
            )
            return None

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
        if model is None:
            return {"lang": LanguageModel.UNKNOWN_LANGUAGE, "score": 0.0}

        text = LanguageDetector.prepare_text_for_language_detection(text)
        predictions = model.predict(text, k=1)
        if not predictions or not predictions[0]:
            return {"lang": LanguageModel.UNKNOWN_LANGUAGE, "score": 0.0}

        lang = predictions[0][0].removeprefix("__label__")
        score = float(predictions[1][0])
        return {"lang": lang, "score": score}


class LanguageModel:
    UNKNOWN_LANGUAGE = "unk"
    ALLOWED_LANGUAGES = {lang: SPACY_LANGUAGE_MODELS[lang] for lang in MODEL_LANGUAGES if lang in SPACY_LANGUAGE_MODELS}

    @staticmethod
    @lru_cache(maxsize=None)
    def load(lang: str) -> Union[spacy.language.Language, UnknownLanguage]:
        if lang == LanguageModel.UNKNOWN_LANGUAGE:
            return UnknownLanguage()

        if lang not in LanguageModel.ALLOWED_LANGUAGES:
            raise ValueError(f"Unsupported language: {lang}")

        model_name = LanguageModel.ALLOWED_LANGUAGES[lang]
        try:
            return spacy.load(model_name)
        except OSError as exc:
            raise RuntimeError(f"Failed to load spaCy model: {model_name}") from exc

    def __init__(self, text: str, language: str = "auto") -> None:
        self.text = text
        self.language_ = language
        self.logger = logging.getLogger(self.__class__.__name__)

    def _detect_language(self) -> str:
        result = LanguageDetector.detect(self.text)
        score = float(result.get("score", 0.0))
        lang = str(result.get("lang", self.UNKNOWN_LANGUAGE))

        if score < 0.2:
            return self.UNKNOWN_LANGUAGE
        if lang not in self.ALLOWED_LANGUAGES:
            return self.UNKNOWN_LANGUAGE
        return lang

    @cached_property
    def truncated_text(self) -> str:
        truncated_text = LanguageDetector.prepare_text_for_language_detection(self.text)
        if len(truncated_text) < len(self.text):
            truncated_text += "..."
        return truncated_text

    @cached_property
    def language(self) -> str:
        if self.language_ == "auto":
            return self._detect_language()

        if self.language_ == self.UNKNOWN_LANGUAGE:
            return self.UNKNOWN_LANGUAGE

        if self.language_ in self.ALLOWED_LANGUAGES:
            return self.language_

        self.logger.warning(
            "Provided language `%s` is unsupported; falling back to `%s`.",
            self.language_,
            self.UNKNOWN_LANGUAGE,
        )
        return self.UNKNOWN_LANGUAGE

    @cached_property
    def nlp(self) -> spacy.language.Language:
        return self.load(self.language)

    @cached_property
    def doc(self) -> Doc:
        doc = self.nlp(self.text)
        if not doc:
            raise ValueError("Failed to process text with spaCy model.")
        return doc
