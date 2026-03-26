"""
Multi-Lingual E5 Large Instruct Model Package

This package provides a comprehensive solution for text embedding generation using
the Multi-Lingual E5 Large Instruct model. It includes text preprocessing, chunking,
inference, and postprocessing capabilities with support for multiple languages.

The package exports three main server implementations:
- fastapi_server: HTTP API server for text embedding requests
- inference_server: Core inference server for model processing
- proxy_server: Proxy server for load balancing and routing

Example:
    >>> from e5_instruct import fastapi_server
    >>> # Start the FastAPI server
    >>> fastapi_server()
"""

from .fastapi import fastapi_server
from .inference import inference_server
from .proxy import proxy_server

__all__ = ["fastapi_server", "inference_server", "proxy_server"]
