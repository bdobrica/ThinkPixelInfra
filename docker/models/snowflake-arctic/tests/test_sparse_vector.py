import random

from snowflake_arctic.language import LanguageModel
from snowflake_arctic.sparse_vector import SparseVector


# --- SparseVector Tests ---
def test_sparse_vector_stopwords_punct():
    tokens = ["▁the", "▁quick", "▁brown", "▁fox", ".", "▁jumps", "▁over", "▁the", "▁lazy", "▁dog", "!"]
    weights = [1.0] * len(tokens)
    language_model = LanguageModel(" ".join(tokens))
    sv = SparseVector(language_model, tokens, weights)
    d = sv.to_dict()
    # Should filter out stopwords and punctuation
    for k in d:
        assert k != "the"
        assert k != "."
        assert k != "!"


def test_sparse_vector_negative_weights():
    tokens = ["▁word1", "▁word2", "▁word3"]
    weights = [1.0, -1.0, 0.0]
    language_model = LanguageModel(" ".join(tokens))
    sv = SparseVector(language_model, tokens, weights)
    d = sv.to_dict()
    assert "▁word2" not in d and "▁word3" not in d


def test_sparse_vector_multilingual():
    samples = {
        "en": "The quick brown fox jumps over the lazy dog.",
        "fr": "Le renard brun rapide saute par-dessus le chien paresseux.",
        "de": "Der schnelle braune Fuchs springt über den faulen Hund.",
        "es": "El rápido zorro marrón salta sobre el perro perezoso.",
        "it": "La veloce volpe marrone salta sopra il cane pigro.",
        "ro": "Vulpea maro rapidă sare peste câinele leneș.",
    }
    for lang, text in samples.items():
        tokens = [f"▁{token}" for token in text.split()]
        weights = [random.uniform(0.1, 1.0) for _ in tokens]
        language_model = LanguageModel(text)
        sv = SparseVector(language_model, tokens, weights)
        d = sv.to_dict()
        assert isinstance(d, dict)
        assert len(d) == len(set(tokens)) or len(d) == len(set([t.lower() for t in tokens]))
