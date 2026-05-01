"""Provider-backed dense embedding gateway for the external service."""

import logging
from typing import Any, Iterable, Sequence

from .config import (
    MODEL_PING_TEXT,
    MODEL_PROVIDER,
    MODEL_PROVIDER_API_KEY,
    MODEL_PROVIDER_BASE_URL,
    MODEL_PROVIDER_BATCH_SIZE,
    MODEL_PROVIDER_DNS_TTL_SECONDS,
    MODEL_PROVIDER_KEEPALIVE_EXPIRY_SECONDS,
    MODEL_PROVIDER_MAX_CONNECTIONS,
    MODEL_PROVIDER_MAX_KEEPALIVE_CONNECTIONS,
    MODEL_PROVIDER_MODEL,
    MODEL_PROVIDER_RETRY_INITIAL_BACKOFF_MS,
    MODEL_PROVIDER_RETRY_JITTER_RATIO,
    MODEL_PROVIDER_RETRY_MAX_BACKOFF_MS,
    MODEL_PROVIDER_RETRY_MAX_RETRIES,
    MODEL_PROVIDER_TIMEOUT_SECONDS,
)
from .llm_client.client import LLMClient
from .llm_client.config import RetryConfig
from .llm_client.types import DisconnectChecker

logger = logging.getLogger(__name__)

_OPENAI_MODEL_ALIASES = {
    "text-embeddings-3-small": "text-embedding-3-small",
    "text-embeddings-3-large": "text-embedding-3-large",
}


class EmbeddingsGateway:
    def __init__(self, llm_client: LLMClient, provider: str, model: str, batch_size: int) -> None:
        self.llm_client = llm_client
        self.provider = provider
        self.model = model
        self.batch_size = batch_size

    async def embed_texts(
        self,
        texts: Sequence[str],
        disconnect_checker: DisconnectChecker | None = None,
    ) -> list[list[float]]:
        if self.provider != "openai":
            raise ValueError(f"Unsupported provider: {self.provider}")

        embeddings: list[list[float]] = []
        for batch in _batched(texts, self.batch_size):
            response = await self.llm_client.embeddings.create(
                model=self.model,
                input=batch,
                encoding_format="float",
                disconnect_checker=disconnect_checker,
            )
            embeddings.extend(_extract_embeddings(response, expected_count=len(batch)))
        return embeddings

    async def ping(self, disconnect_checker: DisconnectChecker | None = None) -> None:
        await self.embed_texts([MODEL_PING_TEXT], disconnect_checker=disconnect_checker)


def create_llm_client() -> LLMClient:
    if not MODEL_PROVIDER_API_KEY:
        raise RuntimeError("MODEL_PROVIDER_API_KEY is required.")

    return LLMClient(
        base_url=MODEL_PROVIDER_BASE_URL,
        api_key=MODEL_PROVIDER_API_KEY,
        timeout_seconds=MODEL_PROVIDER_TIMEOUT_SECONDS,
        max_connections=MODEL_PROVIDER_MAX_CONNECTIONS,
        max_keepalive_connections=MODEL_PROVIDER_MAX_KEEPALIVE_CONNECTIONS,
        keepalive_expiry_seconds=MODEL_PROVIDER_KEEPALIVE_EXPIRY_SECONDS,
        dns_ttl_seconds=MODEL_PROVIDER_DNS_TTL_SECONDS,
        retry=RetryConfig(
            max_retries=MODEL_PROVIDER_RETRY_MAX_RETRIES,
            initial_backoff_ms=MODEL_PROVIDER_RETRY_INITIAL_BACKOFF_MS,
            max_backoff_ms=MODEL_PROVIDER_RETRY_MAX_BACKOFF_MS,
            jitter_ratio=MODEL_PROVIDER_RETRY_JITTER_RATIO,
        ),
    )


def create_embeddings_gateway(llm_client: LLMClient) -> EmbeddingsGateway:
    model = MODEL_PROVIDER_MODEL
    if MODEL_PROVIDER == "openai":
        normalized_model = _OPENAI_MODEL_ALIASES.get(model, model)
        if normalized_model != model:
            logger.warning(
                "Normalizing deprecated OpenAI embeddings model alias %s to %s.",
                model,
                normalized_model,
            )
        model = normalized_model

    return EmbeddingsGateway(
        llm_client=llm_client,
        provider=MODEL_PROVIDER,
        model=model,
        batch_size=MODEL_PROVIDER_BATCH_SIZE,
    )


def _batched(items: Sequence[str], batch_size: int) -> Iterable[list[str]]:
    for index in range(0, len(items), batch_size):
        yield list(items[index : index + batch_size])  # noqa: E203


def _extract_embeddings(response: dict[str, Any], expected_count: int) -> list[list[float]]:
    data = response.get("data")
    if not isinstance(data, list):
        raise ValueError("Provider response does not contain a valid `data` list.")

    embeddings: list[list[float] | None] = [None] * expected_count
    for item in data:
        if not isinstance(item, dict):
            raise ValueError("Provider response contains a non-object embedding item.")
        index = item.get("index")
        embedding = item.get("embedding")
        if not isinstance(index, int) or not (0 <= index < expected_count):
            raise ValueError("Provider response returned an invalid embedding index.")
        if not isinstance(embedding, list) or not all(isinstance(value, (int, float)) for value in embedding):
            raise ValueError("Provider response returned an invalid embedding vector.")
        embeddings[index] = [float(value) for value in embedding]

    if any(item is None for item in embeddings):
        raise ValueError("Provider response did not return embeddings for every requested input.")

    return [item for item in embeddings if item is not None]
