"""Resource facade classes for :mod:`external.llm_client`.

These facades provide OpenAI-compatible resource groupings on top of the shared
:class:`external.llm_client.client.LLMClient` request path.
"""

from typing import TYPE_CHECKING, Any

from .types import DisconnectChecker

if TYPE_CHECKING:
    from .client import LLMClient


class ChatCompletionsResource:
    """Facade for ``/chat/completions`` requests."""

    def __init__(self, client: "LLMClient") -> None:
        """Bind the resource facade to a shared client instance.

        :param client: Client used to perform the underlying HTTP requests.
        """
        self._client = client

    async def create(self, *, disconnect_checker: DisconnectChecker | None = None, **payload: Any) -> dict[str, Any]:
        """Create a chat completion request.

        :param disconnect_checker: Optional async callback used to detect caller disconnects.
        :param payload: JSON request body forwarded to the provider.
        :returns: Parsed JSON response object from the provider.
        """
        return await self._client.request_json(
            "POST",
            "chat/completions",
            json_body=payload,
            disconnect_checker=disconnect_checker,
        )


class ResponsesResource:
    """Facade for ``/responses`` requests."""

    def __init__(self, client: "LLMClient") -> None:
        """Bind the resource facade to a shared client instance.

        :param client: Client used to perform the underlying HTTP requests.
        """
        self._client = client

    async def create(self, *, disconnect_checker: DisconnectChecker | None = None, **payload: Any) -> dict[str, Any]:
        """Create a response API request.

        :param disconnect_checker: Optional async callback used to detect caller disconnects.
        :param payload: JSON request body forwarded to the provider.
        :returns: Parsed JSON response object from the provider.
        """
        return await self._client.request_json(
            "POST",
            "responses",
            json_body=payload,
            disconnect_checker=disconnect_checker,
        )


class EmbeddingsResource:
    """Facade for ``/embeddings`` requests."""

    def __init__(self, client: "LLMClient") -> None:
        """Bind the resource facade to a shared client instance.

        :param client: Client used to perform the underlying HTTP requests.
        """
        self._client = client

    async def create(self, *, disconnect_checker: DisconnectChecker | None = None, **payload: Any) -> dict[str, Any]:
        """Create an embeddings request.

        :param disconnect_checker: Optional async callback used to detect caller disconnects.
        :param payload: JSON request body forwarded to the provider.
        :returns: Parsed JSON response object from the provider.
        """
        return await self._client.request_json(
            "POST",
            "embeddings",
            json_body=payload,
            disconnect_checker=disconnect_checker,
        )
