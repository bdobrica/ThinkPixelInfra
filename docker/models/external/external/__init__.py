"""External embeddings gateway package."""

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from fastapi import FastAPI


def create_app() -> "FastAPI":
    from .fastapi import create_app as _create_app

    return _create_app()


def fastapi_server() -> None:
    from .fastapi import fastapi_server as _fastapi_server

    _fastapi_server()


__all__ = ["create_app", "fastapi_server"]
