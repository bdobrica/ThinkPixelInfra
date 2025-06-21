import logging
import time
import unicodedata
from functools import lru_cache
from typing import Union

import spacy
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
    """
    A placeholder class for unknown languages.
    It provides basic functionality without any specific language processing.
    """

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
            lemma = self._get_lemma(token.text)
            token.lemma = self.vocab.strings[lemma]
            token.lex.is_punct = self._is_punct(token.text)
        return doc


@lru_cache(maxsize=None)
def load_spacy_model(
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
