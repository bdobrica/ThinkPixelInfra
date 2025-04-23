import base64
import logging
import time
import unicodedata
from functools import lru_cache
from struct import pack
from typing import Dict, Iterable, Union

import mmh3
import spacy
from fast_langdetect import detect
from spacy.tokens import Doc
from spacy.vocab import Vocab

from .config import LOG_LEVEL

# Setup logging
logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)

# Define the allowed languages and corresponding spaCy small models.
ALLOWED_LANGUAGES = {
    "en": "en_core_web_sm",
    "fr": "fr_core_news_sm",
    "de": "de_core_news_sm",
    "es": "es_core_news_sm",
    "it": "it_core_news_sm",
    "ro": "ro_core_news_sm",
}


class UnknownLanguage(spacy.language.Language):
    def __init__(self):
        self.vocab = Vocab()

    def _is_punct(self, token: str) -> bool:
        return all(unicodedata.category(item)[0] in "PMSC" for item in token)

    def _get_lemma(self, token: str) -> str:
        token = token.strip().lower()
        while token and unicodedata.category(token[0]) in "PMSC":
            token = token[1:]
        while token and unicodedata.category(token[-1]) in "PMSC":
            token = token[:-1]
        return token or "<unk>"

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
            token.lex.is_punct = self._is_punct(token.text)
        return doc


def _detect_language(text: str) -> str:
    """
    Detects the language of the given text.
    Expects fast_langdetect.detect to return a dict with 'lang' and 'score'.
    """
    start_time = time.perf_counter()
    result = detect(text)
    elapsed_time = time.perf_counter() - start_time
    score = float(result.get("score", 0))
    lang = str(result.get("lang", "unk"))

    logger.debug(
        "Detected language: %s with score: %f (time: %f)",
        lang,
        score,
        elapsed_time,
    )

    if score > 0.33:
        return lang

    return "unk"  # fallback to unknown if score is low


@lru_cache(maxsize=None)
def _load_spacy_model(
    lang: str,
) -> Union[spacy.language.Language, UnknownLanguage]:
    """
    Loads the spaCy model for the given language.
    """
    if lang == "unk":
        return UnknownLanguage()

    if lang not in ALLOWED_LANGUAGES:
        logger.warning("Unsupported language: %s", lang)
        return UnknownLanguage()

    model_name = ALLOWED_LANGUAGES[lang]
    start_time = time.perf_counter()
    try:
        nlp = spacy.load(model_name)
    except OSError:
        raise RuntimeError(f"Failed to load spaCy model: {model_name}")
    elapsed_time = time.perf_counter() - start_time
    logger.debug("Loaded spaCy model: %s (time: %f)", model_name, elapsed_time)

    return nlp


def _build_doc(tokens: Iterable[str]) -> Doc:
    """
    Builds a spaCy Doc object from the given tokens.
    """
    tokens = list(tokens)
    language = _detect_language(" ".join(tokens))
    nlp = _load_spacy_model(language)
    doc = Doc(nlp.vocab, words=tokens)
    doc = nlp(doc)

    if len(doc) != len(tokens):
        raise ValueError("Mismatch between tokens and generated Doc length.")

    return doc


def _build_weighted_tokens(
    tokens: Iterable[str], weights: Iterable[float]
) -> Dict[str, float]:
    """
    Maps tokens to their weights.
    Filters out stop words, punctuation, and zero weights.
    Normalizes the weights.
    """
    doc = _build_doc(tokens)

    res = {}
    total_weight = 0
    for token, weight in zip(doc, weights):
        if token.is_stop or token.is_punct or 0 >= weight:
            continue
        lemma = token.lemma_.lower()
        res[lemma] = res.get(lemma, 0) + weight
        total_weight += weight

    # Normalize weights
    for token in res:
        res[token] /= total_weight
        logger.debug("Token: %s, Weight: %f", token, res[token])

    return res


def _base64_uint(x: int) -> str:
    """
    Converts an unsigned integer to a base64 string.
    """
    return base64.b64encode(pack(">L", x)).decode("utf-8")


def _base64_float(x: float) -> str:
    """
    Converts a float to a base64 string.
    """
    return base64.b64encode(pack(">f", x)).decode("utf-8")


def build_sparse_vector(
    tokens: Iterable[str], weights: Iterable[float]
) -> Dict[str, str]:
    """
    Builds a sparse vector from the given tokens and weights.
    Uses mmh3 to hash the tokens into a fixed-size vector.
    """
    token_weights = _build_weighted_tokens(tokens, weights)

    return dict(
        map(
            lambda x: (_base64_uint(abs(mmh3.hash(x[0]))), _base64_float(x[1])),
            token_weights.items(),
        )
    )
