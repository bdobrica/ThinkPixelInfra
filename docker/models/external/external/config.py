"""Configuration for the external embeddings gateway."""

import logging
import os
from multiprocessing import cpu_count
from typing import List

SPACY_LANGUAGE_MODELS = {
    "en": "en_core_web_sm",
    "fr": "fr_core_news_sm",
    "de": "de_core_news_sm",
    "es": "es_core_news_sm",
    "it": "it_core_news_sm",
    "ro": "ro_core_news_sm",
}


def _parse_bool(name: str, default: bool = False) -> bool:
    return os.getenv(name, str(default)).lower() in {"true", "1", "yes", "y", "on"}


def _parse_optional_float(name: str) -> float | None:
    value = os.getenv(name)
    if value is None or value.strip().lower() in {"", "none", "null"}:
        return None
    return float(value)


LOCAL: bool = _parse_bool("LOCAL", default=False)

MODEL_HTTP_PORT: int = int(os.getenv("MODEL_HTTP_PORT", 8000))
MODEL_CHUNK_SIZE: int = int(os.getenv("MODEL_CHUNK_SIZE", 1000))
MODEL_CHUNK_OVERLAP: int = int(os.getenv("MODEL_CHUNK_OVERLAP", 200))
MODEL_PREPROCESS_THREADS: int = max(1, int(os.getenv("MODEL_PREPROCESS_THREADS", max(1, cpu_count() // 2))))

MODEL_PROVIDER: str = os.getenv("MODEL_PROVIDER", "openai").lower()
MODEL_PROVIDER_MODEL: str = os.getenv("MODEL_PROVIDER_MODEL", "text-embedding-3-small")
MODEL_PROVIDER_BASE_URL: str = os.getenv("MODEL_PROVIDER_BASE_URL", "https://api.openai.com/v1")
MODEL_PROVIDER_API_KEY: str = os.getenv("MODEL_PROVIDER_API_KEY", "")
MODEL_PROVIDER_BATCH_SIZE: int = max(1, int(os.getenv("MODEL_PROVIDER_BATCH_SIZE", 32)))
MODEL_PROVIDER_TIMEOUT_SECONDS: float = float(os.getenv("MODEL_PROVIDER_TIMEOUT_SECONDS", 30.0))
MODEL_PROVIDER_MAX_CONNECTIONS: int = max(1, int(os.getenv("MODEL_PROVIDER_MAX_CONNECTIONS", 256)))
MODEL_PROVIDER_MAX_KEEPALIVE_CONNECTIONS: int = max(
    1,
    int(os.getenv("MODEL_PROVIDER_MAX_KEEPALIVE_CONNECTIONS", 64)),
)
MODEL_PROVIDER_KEEPALIVE_EXPIRY_SECONDS: float = float(os.getenv("MODEL_PROVIDER_KEEPALIVE_EXPIRY_SECONDS", 30.0))
MODEL_PROVIDER_DNS_TTL_SECONDS: float | None = _parse_optional_float("MODEL_PROVIDER_DNS_TTL_SECONDS")
MODEL_PROVIDER_RETRY_MAX_RETRIES: int = max(0, int(os.getenv("MODEL_PROVIDER_RETRY_MAX_RETRIES", 2)))
MODEL_PROVIDER_RETRY_INITIAL_BACKOFF_MS: int = max(
    0,
    int(os.getenv("MODEL_PROVIDER_RETRY_INITIAL_BACKOFF_MS", 250)),
)
MODEL_PROVIDER_RETRY_MAX_BACKOFF_MS: int = max(
    MODEL_PROVIDER_RETRY_INITIAL_BACKOFF_MS,
    int(os.getenv("MODEL_PROVIDER_RETRY_MAX_BACKOFF_MS", 1000)),
)
MODEL_PROVIDER_RETRY_JITTER_RATIO: float = max(0.0, float(os.getenv("MODEL_PROVIDER_RETRY_JITTER_RATIO", 0.2)))

MODEL_SPARSE_STRATEGY: str = os.getenv("MODEL_SPARSE_STRATEGY", "bm25").lower()
MODEL_SPARSE_MAX_FEATURES: int = max(1, int(os.getenv("MODEL_SPARSE_MAX_FEATURES", 128)))
MODEL_SPARSE_MIN_TOKEN_LENGTH: int = max(1, int(os.getenv("MODEL_SPARSE_MIN_TOKEN_LENGTH", 2)))
MODEL_SPARSE_BM25_K1: float = max(0.0, float(os.getenv("MODEL_SPARSE_BM25_K1", 1.2)))
MODEL_SPARSE_BM25_B: float = min(1.0, max(0.0, float(os.getenv("MODEL_SPARSE_BM25_B", 0.75))))
MODEL_SPARSE_BM25_AVGDL: float = max(1.0, float(os.getenv("MODEL_SPARSE_BM25_AVGDL", 64.0)))

MODEL_SEARCH_PREFIX: str = os.getenv("MODEL_SEARCH_PREFIX", "")
MODEL_STORE_PREFIX: str = os.getenv("MODEL_STORE_PREFIX", "")

MODEL_LANGUAGES: List[str] = list(
    filter(
        lambda item: item in SPACY_LANGUAGE_MODELS,
        os.getenv("MODEL_LANGUAGES", "en").split(","),
    )
)
MODEL_LANGUAGE_DETECTION_PATH: str = os.getenv("MODEL_LANGUAGE_DETECTION_PATH", "/app/fasttext/lid.176.bin")
MODEL_PING_TEXT: str = os.getenv("MODEL_PING_TEXT", "health check")

LOG_LEVEL: int = logging.DEBUG if LOCAL else logging.INFO
