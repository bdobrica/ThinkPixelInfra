"""Resource facade classes for llm_client."""

from typing import TYPE_CHECKING, Any

from .types import Request

if TYPE_CHECKING:
    from .client import LLMClient


class ChatCompletionsResource:
    def __init__(self, client: "LLMClient") -> None:
        self._client = client

    async def create(self, *, request: Request | None = None, **payload: Any) -> dict[str, Any]:
        return await self._client.request_json("POST", "chat/completions", json_body=payload, request=request)


class ResponsesResource:
    def __init__(self, client: "LLMClient") -> None:
        self._client = client

    async def create(self, *, request: Request | None = None, **payload: Any) -> dict[str, Any]:
        return await self._client.request_json("POST", "responses", json_body=payload, request=request)


class EmbeddingsResource:
    def __init__(self, client: "LLMClient") -> None:
        self._client = client

    async def create(self, *, request: Request | None = None, **payload: Any) -> dict[str, Any]:
        return await self._client.request_json("POST", "embeddings", json_body=payload, request=request)
