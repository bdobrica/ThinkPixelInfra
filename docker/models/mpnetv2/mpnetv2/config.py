"""
Configuration module for MPNetv2 embedding service.

This module centralizes all configuration settings for the MPNetv2 service,
including model parameters, server settings, and text processing options.
All settings are configurable via environment variables with sensible defaults.

Environment Variables:
    LOCAL: Enable local development mode with debug logging (default: false)
    MODEL_PATH: Path to the MPNetv2 model directory (default: /app/mpnet-base-v2)
    MODEL_DEVICE: PyTorch device for model inference (default: cpu)
    MODEL_ZMQ_CLIENT_ADDR: ZMQ client socket address (default: ipc:///tmp/mpnetv2.client)
    MODEL_ZMQ_WORKER_ADDR: ZMQ worker socket address (default: ipc:///tmp/mpnetv2.worker)
    MODEL_HTTP_PORT: HTTP API server port (default: 8000)
    MODEL_NUM_WORKERS: Number of inference worker processes (default: CPU count / 2)
    MODEL_CHUNK_SIZE: Default maximum characters per text chunk (default: 1000)
    MODEL_CHUNK_OVERLAP: Default character overlap between chunks (default: 200)

Text Processing Configuration:
    The chunk size and overlap settings control how long texts are split into
    manageable pieces for embedding generation. Larger chunks capture more
    context but may exceed model limits, while smaller chunks provide more
    granular embeddings but may lose context.

Example:
    >>> from mpnetv2.config import MODEL_PATH, MODEL_CHUNK_SIZE
    >>> print(f"Model: {MODEL_PATH}, Chunk size: {MODEL_CHUNK_SIZE}")
"""

import logging
import os
from typing import List

from torch.multiprocessing import cpu_count

SPACY_LANGUAGE_MODELS = {
    "en": "en_core_web_sm",
    "fr": "fr_core_news_sm",
    "de": "de_core_news_sm",
    "es": "es_core_news_sm",
    "it": "it_core_news_sm",
    "ro": "ro_core_news_sm",
}

# Environment Variables
LOCAL: bool = os.getenv("LOCAL", "false").lower() in {
    "true",
    "1",
    "yes",
    "y",
    "on",
}
MODEL_PATH: str = os.getenv("MODEL_PATH", "/app/mpnet-base-v2")
MODEL_DEVICE: str = os.getenv("MODEL_DEVICE", "cpu")
MODEL_ZMQ_CLIENT_ADDR: str = os.getenv("MODEL_ZMQ_CLIENT_ADDR", "ipc:///tmp/mpnetv2.client")
MODEL_ZMQ_WORKER_ADDR: str = os.getenv("MODEL_ZMQ_WORKER_ADDR", "ipc:///tmp/mpnetv2.worker")
MODEL_HTTP_PORT: int = int(os.getenv("MODEL_HTTP_PORT", 8000))
MODEL_NUM_WORKERS: int = max(1, int(os.getenv("MODEL_NUM_WORKERS", cpu_count() // 2)))
MODEL_CHUNK_SIZE: int = int(os.getenv("MODEL_CHUNK_SIZE", 1000))
MODEL_CHUNK_OVERLAP: int = int(os.getenv("MODEL_CHUNK_OVERLAP", 200))
MODEL_LANGUAGES: List[str] = list(
    filter(
        lambda item: item in SPACY_LANGUAGE_MODELS,
        os.getenv("MODEL_LANGUAGES", "en").split(","),
    )
)
MODEL_LANGUAGE_DETECTION_PATH: str = os.getenv("MODEL_LANGUAGE_DETECTION_PATH", "/app/fasttext/lid.176.bin")

# Logging
LOG_LEVEL: int = logging.DEBUG if LOCAL else logging.WARNING
