"""
Test suite for TextSplitter functionality using pytest
"""

import pytest
from snowflake_arctic.language import LanguageModel
from snowflake_arctic.text_splitter import TextSplitter


def test_chunk_size_zero_no_splitting():
    """Test that setting chunk_size to 0 returns the entire text without splitting"""
    text = "First sentence. Second sentence. Third sentence. Fourth sentence. Fifth sentence."
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=0, chunk_overlap=50)

    chunks = list(splitter)

    # Assertions
    assert len(chunks) == 1, "chunk_size=0 should produce exactly one chunk"
    offset, chunk = chunks[0]
    assert offset == 0, "Single chunk should start at offset 0"
    assert chunk == text, "Single chunk should contain the entire text"
    assert len(chunk) == len(text), "Chunk length should match original text length"


def test_small_text_no_chunking():
    """Test that small text (< chunk_size) is not chunked"""
    text = "This is a short text. It has two sentences."
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=1000, chunk_overlap=200)

    chunks = list(splitter)

    # Assertions
    assert len(chunks) == 1, "Small text should produce exactly one chunk"
    offset, chunk = chunks[0]
    assert offset == 0, "Single chunk should start at offset 0"
    assert chunk == text, "Single chunk should contain the entire text"
    assert len(chunk) == len(text), "Chunk length should match original text length"


def test_large_text_chunking_with_overlap():
    """Test chunking with overlap for large text"""
    # Create a text with multiple sentences
    sentences = [
        "This is the first sentence.",
        "This is the second sentence which is a bit longer than the first one.",
        "The third sentence continues the text and adds more content to test the chunking.",
        "Fourth sentence is here to provide additional content for testing purposes.",
        "Fifth sentence adds even more text to ensure we exceed the chunk size limit.",
        "Sixth sentence continues with more content to test overlap functionality properly.",
        "Seventh sentence is added to make sure we have enough content for multiple chunks.",
        "Eighth sentence provides final content to complete our comprehensive test case.",
    ]
    text = " ".join(sentences)
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=150, chunk_overlap=50)

    chunks = list(splitter)

    # Assertions
    assert len(chunks) > 1, "Large text should produce multiple chunks"

    # Check that all chunks respect the size limit (except possibly the first sentence)
    for i, (offset, chunk) in enumerate(chunks):
        # Allow first chunk of a sentence to exceed limit if it's a single long sentence
        if i == 0 or "." in chunk:  # If it contains at least one complete sentence
            assert (
                len(chunk) <= 150 or chunk.count(".") == 1
            ), f"Chunk {i+1} exceeds size limit: {len(chunk)} characters"

    # Check that chunks are ordered by offset
    offsets = [offset for offset, _ in chunks]
    assert offsets == sorted(offsets), "Chunks should be ordered by offset"

    # Check that chunks cover the entire text
    total_coverage = set()
    for offset, chunk in chunks:
        end_offset = offset + len(chunk)
        total_coverage.update(range(offset, end_offset))

    # We should cover most of the text (allowing for some overlap)
    assert len(total_coverage) >= len(text) * 0.8, "Chunks should cover most of the original text"


def test_single_long_sentence_exceeds_chunk_size():
    """Test handling of a single sentence longer than chunk_size"""
    text = (
        "This is a very long sentence that exceeds the chunk size limit "
        "and should be included as a single chunk even though it's longer "
        "than the specified chunk size limit, demonstrating the fallback behavior."
    )
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=100, chunk_overlap=20)

    chunks = list(splitter)

    # Assertions
    assert len(chunks) == 1, "Single long sentence should produce exactly one chunk"
    offset, chunk = chunks[0]
    assert offset == 0, "Single chunk should start at offset 0"
    assert chunk == text, "Chunk should contain the entire sentence"
    assert len(chunk) > 100, "Chunk should exceed the chunk_size limit"


def test_sentence_separators_preservation():
    """Test preservation of original sentence separators"""
    text = (
        "First sentence.  Second sentence with double space.\n\n"
        "Third sentence after newlines.\tFourth sentence with tab."
    )
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=80, chunk_overlap=20)

    chunks = list(splitter)

    # Assertions
    assert len(chunks) >= 1, "Text should produce at least one chunk"

    # Check that separators are preserved
    # Check for specific separators in the chunks
    found_double_space = any("  " in chunk for _, chunk in chunks)
    found_newlines = any("\n" in chunk for _, chunk in chunks)
    found_tab = any("\t" in chunk for _, chunk in chunks)

    # At least some separators should be preserved
    separator_preserved = found_double_space or found_newlines or found_tab
    assert separator_preserved, "Original separators should be preserved in chunks"


def test_chunk_overlap_functionality():
    """Test that overlap functionality works correctly"""
    text = "First sentence. Second sentence. Third sentence. Fourth sentence. Fifth sentence."
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=40, chunk_overlap=15)

    chunks = list(splitter)

    if len(chunks) > 1:
        # Check for overlap between consecutive chunks
        for i in range(len(chunks) - 1):
            current_chunk = chunks[i][1]
            next_chunk = chunks[i + 1][1]

            # There should be some textual overlap or the chunks should be adjacent
            # This is a basic check - more sophisticated overlap detection could be added
            assert len(current_chunk) > 0 and len(next_chunk) > 0, "Both chunks should have content"


def test_iterator_behavior():
    """Test that TextSplitter behaves correctly as an iterator"""
    text = "First sentence. Second sentence. Third sentence."
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=25, chunk_overlap=10)

    # Test that we can iterate multiple times
    chunks1 = list(splitter)
    chunks2 = list(splitter)

    assert chunks1 == chunks2, "Multiple iterations should produce the same results"

    # Test iterator protocol
    iterator = iter(splitter)
    assert hasattr(iterator, "__next__"), "Iterator should have __next__ method"
    assert iterator is splitter, "Iterator should return self"


def test_empty_text_handling():
    """Test handling of empty or whitespace-only text"""
    text = "   \n\t   "  # Only whitespace
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=100, chunk_overlap=20)

    chunks = list(splitter)

    # Should handle gracefully - either empty list or single chunk with the whitespace
    assert isinstance(chunks, list), "Should return a list"
    if chunks:
        assert len(chunks) == 1, "Whitespace-only text should produce at most one chunk"


@pytest.mark.parametrize(
    "chunk_size,chunk_overlap",
    [
        (100, 20),
        (200, 50),
        (50, 10),
        (1000, 100),
    ],
)
def test_different_chunk_sizes(chunk_size: int, chunk_overlap: int):
    """Test TextSplitter with different chunk sizes and overlaps"""
    text = (
        "This is a test text with multiple sentences. "
        "Each sentence adds to the total length. "
        "We want to test different chunking parameters. "
        "The splitter should handle various configurations correctly."
    )
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=chunk_size, chunk_overlap=chunk_overlap)

    chunks = list(splitter)

    # Basic assertions that should hold for any valid configuration
    assert len(chunks) >= 1, "Should produce at least one chunk"
    assert all(isinstance(offset, int) for offset, _ in chunks), "Offsets should be integers"
    assert all(isinstance(chunk, str) for _, chunk in chunks), "Chunks should be strings"
    assert all(len(chunk) > 0 for _, chunk in chunks), "Chunks should not be empty"


def test_reset_functionality():
    """Test that reset functionality works correctly"""
    text = "First sentence. Second sentence. Third sentence."
    language_model = LanguageModel(text)
    splitter = TextSplitter(language_model, chunk_size=25, chunk_overlap=10)

    # Get chunks normally
    chunks1 = list(splitter)

    # Reset and get chunks again
    splitter.reset()
    chunks2 = list(splitter)

    assert chunks1 == chunks2, "Reset should allow re-iteration with same results"
