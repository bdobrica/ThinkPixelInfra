import numpy as np
from snowflake_arctic.language import LanguageModel
from snowflake_arctic.postprocess import build_results


# --- build_results Tests ---
def test_build_results_empty():
    results = build_results([], [], (np.array([]), np.array([])))
    assert results == []


def test_build_results_single():
    # build_results expects (items, metadatas, scores)
    items = [
        {
            "text": "The quick brown fox jumps over the lazy dog.",
            "metadata": {
                "id": 1,
                "extra": {"offset": 0, "language_model": LanguageModel("The quick brown fox jumps over the lazy dog.")},
            },
        }
    ]
    tokens = [["▁the", "▁quick", "▁brown", "▁fox", "▁jumps", "▁over", "▁lazy", "▁dog"]]
    np.random.seed(42)  # For reproducibility
    model_output = (np.random.rand(1, 768), np.random.rand(1, 8))  # Mock dense and sparse outputs
    results = build_results(items, tokens, model_output)
    assert isinstance(results, list)
    assert results[0]["offset"] == 0
    assert results[0]["metadata"]["id"] == 1
    assert results[0]["metadata"]["extra"]["language"] == "en"
    assert "language_model" not in results[0]["metadata"]["extra"]


def test_build_results_multiple():
    items = [
        {
            "text": "The quick brown fox jumps over the lazy dog.",
            "metadata": {
                "id": 1,
                "extra": {"offset": 0, "language_model": LanguageModel("The quick brown fox jumps over the lazy dog.")},
            },
        },
        {
            "text": "El rápido zorro marrón salta sobre el perro perezoso.",
            "metadata": {
                "id": 2,
                "extra": {
                    "offset": 0,
                    "language_model": LanguageModel("El rápido zorro marrón salta sobre el perro perezoso."),
                },
            },
        },
        {
            "text": "Der schnelle braune Fuchs springt über den faulen Hund.",
            "metadata": {
                "id": 3,
                "extra": {
                    "offset": 0,
                    "language_model": LanguageModel("Der schnelle braune Fuchs springt über den faulen Hund."),
                },
            },
        },
    ]
    tokens = [
        ["▁the", "▁quick", "▁brown", "▁fox", "▁jumps", "▁over", "▁lazy", "▁dog"],
        ["▁el", "▁rápido", "▁zorro", "▁marrón", "▁salta", "▁sobre", "▁el", "▁perro", "▁perezoso"],
        ["▁der", "▁schnelle", "▁braune", "▁fuchs", "▁springt", "▁über", "▁den", "▁faulen", "▁hund"],
    ]
    np.random.seed(42)  # For reproducibility
    model_output = (
        np.random.rand(3, 768),
        np.random.rand(3, 9),
    )  # Mock dense and sparse outputs
    results = build_results(items, tokens, model_output)
    assert len(results) == 3
    ids = [r["metadata"]["id"] for r in results]
    assert set(ids) == {1, 2, 3}
