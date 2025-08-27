"""
Post-processing module for MPNetv2 embedding results.

This module provides functions for processing transformer model outputs into
final embedding results. It handles mean pooling of token embeddings and
formatting results for API responses.

Functions:
    mean_pooling: Apply attention-masked mean pooling to token embeddings
    build_results: Convert model outputs to formatted embedding results

Key Features:
- Attention-masked mean pooling for variable-length sequences
- Base64 encoding of dense vectors for efficient transmission
- Metadata preservation throughout the processing pipeline
- Float32 big-endian encoding for consistent vector representation

Example:
    >>> # Process model output into embedding results
    >>> results = build_results(text_items, batch_input, model_output)
    >>> print(f"Generated {len(results)} embeddings")
"""

import base64
from typing import Any, Dict, List, Mapping, Tuple

import numpy as np
import torch
from transformers.modeling_outputs import BaseModelOutputWithPoolingAndCrossAttentions

from .language import LanguageModel


def _mean_pooling(token_embeddings: torch.Tensor, attention_mask: torch.Tensor) -> torch.Tensor:
    """
    Apply attention-masked mean pooling to token embeddings.

    This function performs mean pooling over the sequence dimension of token
    embeddings while taking the attention mask into account. This ensures that
    padding tokens don't contribute to the final sentence representation.

    The implementation:
    1. Expands attention mask to match embedding dimensions
    2. Applies mask to embeddings (zeros out padding tokens)
    3. Computes sum of masked embeddings along sequence dimension
    4. Normalizes by the actual sequence length (non-padding tokens)

    Args:
        token_embeddings (torch.Tensor): Token-level embeddings from transformer
                                        Shape: (batch_size, seq_len, hidden_size)
        attention_mask (torch.Tensor): Attention mask indicating real vs padding tokens
                                     Shape: (batch_size, seq_len)

    Returns:
        torch.Tensor: Mean-pooled sentence embeddings
                     Shape: (batch_size, hidden_size)

    Note:
        Uses a minimum denominator of 1e-9 to prevent division by zero
        for edge cases with empty sequences.
    """
    input_mask_expanded = attention_mask.unsqueeze(-1).expand(token_embeddings.size()).float()
    sum_embeddings = torch.sum(token_embeddings * input_mask_expanded, 1)
    sum_mask = torch.clamp(input_mask_expanded.sum(1), min=1e-9)
    return sum_embeddings / sum_mask


def _process_text_item(zipped_item: Tuple[Dict[str, Any], np.ndarray]) -> Dict[str, Any]:
    """
    Process a single text item and its corresponding dense vector into final result format.

    This function takes a text item with metadata and its computed dense vector,
    then formats them into the standardized output structure. It handles metadata
    extraction, language processing, and vector encoding.

    Processing Steps:
    1. Extract text content and metadata from the text item
    2. Remove and process special metadata fields (offset, language_model)
    3. Convert language model reference to language string
    4. Encode dense vector as base64 string with float32 big-endian format
    5. Return formatted result dictionary

    Args:
        zipped_item (Tuple[Dict[str, Any], np.ndarray]): A tuple containing:
            - text_item (Dict[str, Any]): Text item with content and metadata
            - dense_vector (np.ndarray): Computed embedding vector for the text

    Returns:
        Dict[str, Any]: Formatted result containing:
            - text (str): The processed text content
            - offset (int): Character offset in the original text (default: 0)
            - dense_vector (str): Base64-encoded float32 big-endian vector
            - metadata (dict): Preserved metadata with updated language information

    Metadata Processing:
        - Removes 'offset' and 'language_model' from extra metadata
        - Converts LanguageModel instance to language string code
        - Falls back to unknown language if no valid language model found
        - Preserves all other metadata fields unchanged

    Vector Encoding:
        Dense vectors are flattened, converted to float32 big-endian ('>f4'),
        and base64-encoded for efficient transmission and platform consistency.

    Example:
        >>> text_item = {"text": "Hello", "metadata": {"extra": {"offset": 10}}}
        >>> vector = np.array([[0.1, 0.2, 0.3]])
        >>> result = _process_text_item((text_item, vector))
        >>> print(result["text"])  # "Hello"
        >>> print(result["offset"])  # 10
    """
    text_item, dense_vector = zipped_item
    text: str = text_item.get("text", "")
    metadata: dict = text_item.get("metadata", {})
    extra: dict = metadata.get("extra", {})
    offset: int = extra.pop("offset", 0)
    language_model = extra.pop("language_model", None)

    if isinstance(language_model, LanguageModel):
        extra["language"] = language_model.language
    else:
        extra["language"] = LanguageModel.UNKNOWN_LANGUAGE

    return {
        "text": text,
        "offset": offset,
        "dense_vector": base64.b64encode(dense_vector.flatten().astype(">f4").tobytes()).decode("utf-8"),
        "metadata": metadata,
    }


def build_results(
    text_items: List[dict],
    batch_input: Mapping[str, torch.Tensor],
    model_output: BaseModelOutputWithPoolingAndCrossAttentions,
) -> List[dict]:
    """
    Convert model outputs into formatted embedding results.

    This function processes transformer model outputs through mean pooling
    and formats them into the final API response structure. Each result
    includes the original text, character offset, base64-encoded dense vector,
    and preserved metadata.

    Processing Steps:
    1. Apply mean pooling to model's last hidden state
    2. Convert tensors to numpy arrays on CPU
    3. Encode vectors as base64 strings (float32 big-endian)
    4. Combine with text and metadata information

    Args:
        text_items (List[dict]): Original text items with metadata
        batch_input (Mapping[str, torch.Tensor]): Tokenizer output with attention masks
        model_output (BaseModelOutputWithPoolingAndCrossAttentions): Transformer output

    Returns:
        List[dict]: Formatted embedding results, each containing:
            - text: The processed text chunk
            - offset: Character offset in original text
            - dense_vector: Base64-encoded float32 vector
            - metadata: Original metadata from text item

    Vector Encoding:
        Vectors are encoded as float32 big-endian ('>f4') and base64-encoded
        for efficient transmission and consistent cross-platform representation.

    Example:
        >>> results = build_results(text_items, batch_input, model_output)
        >>> vector = base64.b64decode(results[0]['dense_vector'])
        >>> array = np.frombuffer(vector, dtype='>f4')
    """

    pooled_output = _mean_pooling(
        model_output.last_hidden_state,
        batch_input["attention_mask"],
    )

    vector_batch = pooled_output.cpu().numpy()
    results = list(map(_process_text_item, zip(text_items, vector_batch)))
    return results
