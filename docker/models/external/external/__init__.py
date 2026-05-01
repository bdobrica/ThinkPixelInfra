"""External embeddings gateway package."""

from .fastapi import create_app, fastapi_server

__all__ = ["create_app", "fastapi_server"]
