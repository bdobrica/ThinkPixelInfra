"""
Test module for mpnetv2 language processing and text splitting functionality.
"""

import pytest
from mpnetv2.language import LanguageModel
from mpnetv2.preprocess import prepare_text_items
from mpnetv2.text_splitter import TextSplitter


def test_language_detection():
    """Test language detection functionality."""
    # Test English text
    text_en = "This is a test sentence in English. It should be detected correctly."
    language_model = LanguageModel(text_en)

    # Should detect English
    assert language_model.language == "en"

    # Test that spaCy model loads
    assert language_model.nlp is not None
    assert language_model.doc is not None


def test_text_splitter():
    """Test text splitting functionality."""
    text = "First sentence. Second sentence. Third sentence. Fourth sentence. Fifth sentence."
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=50, chunk_overlap=20)

    chunks = list(splitter)

    # Should produce multiple chunks
    assert len(chunks) > 1

    # Each chunk should be a tuple of (offset, text)
    for chunk in chunks:
        assert isinstance(chunk, tuple)
        assert len(chunk) == 2
        assert isinstance(chunk[0], int)  # offset
        assert isinstance(chunk[1], str)  # text


def test_prepare_text_items():
    """Test preprocessing functionality."""
    text_items = [
        {
            "text": "This is a long text that should be split into multiple chunks. " * 10,
            "metadata": {"id": 1, "extra": {"source": "test"}},
        },
        {"text": "Short text.", "metadata": {"id": 2}},
    ]

    processed_items = prepare_text_items(text_items, chunk_size=100, chunk_overlap=20)

    # Should produce at least as many items as input (due to chunking)
    assert len(processed_items) >= len(text_items)

    # Each processed item should have the required structure
    for item in processed_items:
        assert "text" in item
        assert "metadata" in item
        assert "extra" in item["metadata"]
        assert "offset" in item["metadata"]["extra"]
        assert "language" in item["metadata"]["extra"]


def test_empty_text_handling():
    """Test handling of empty or invalid text."""
    # Empty text
    text_items = [{"text": "", "metadata": {"id": 1}}]
    processed_items = prepare_text_items(text_items)

    # Should return empty list for empty text
    assert len(processed_items) == 0

    # None text should be handled gracefully
    text_items = [{"metadata": {"id": 1}}]  # No text key
    processed_items = prepare_text_items(text_items)
    assert len(processed_items) == 0


if __name__ == "__main__":
    pytest.main([__file__])
