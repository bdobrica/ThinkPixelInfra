"""Command entrypoint for the external embeddings gateway."""

import logging

from .fastapi import fastapi_server

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

if __name__ == "__main__":
    fastapi_server()
