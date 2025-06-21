import base64
import random

import pytest
from snowflake_arctic.language import ALLOWED_LANGUAGES
from snowflake_arctic.sparse_vector import SparseVector

# Sample sentences for each language
SAMPLE_TEXTS = {
    "en": "The quick brown fox jumps over the lazy dog.",
    "fr": "Le vif renard brun saute par-dessus le chien paresseux.",
    "de": "Der schnelle braune Fuchs springt über den faulen Hund.",
    "es": "El rápido zorro marrón salta sobre el perro perezoso.",
    "it": "La veloce volpe marrone salta sopra il cane pigro.",
    "ro": "Vulpea rapidă maro sare peste câinele leneș.",
}


@pytest.mark.parametrize("lang", list(ALLOWED_LANGUAGES.keys()))
def test_sparse_vector_to_dict(lang):
    text = SAMPLE_TEXTS[lang]
    tokens = text.split()
    random.seed(lang)  # Deterministic weights per language
    weights = [random.uniform(0.1, 1.0) for _ in tokens]
    sv = SparseVector(tokens, weights)
    sv.language = lang  # Force language for test
    result = sv.to_dict()
    assert isinstance(result, dict)
    assert len(result) > 0
    for k, v in result.items():
        # Keys and values should be base64-encoded strings
        assert isinstance(k, str)
        assert isinstance(v, str)
        # Decoding should not raise
        base64.b64decode(k)
        base64.b64decode(v)
