"""
MPNetv2 Multi-Language Text Embedding Model with Intelligent Text Chunking

This package provides a distributed MPNetv2 text embedding service with advanced
text preprocessing capabilities including automatic language detection, sentence-based
text splitting, and multi-language support.

The service consists of three main components:
- FastAPI server: HTTP API endpoints for inference and health checks
- Inference server: ZMQ-based workers for model inference with text preprocessing
- Proxy server: ZMQ proxy for load balancing between API and workers

Key Features:
- Multi-language support (English, French, German, Spanish, Italian, Romanian)
- Automatic language detection using fast-langdetect
- Intelligent sentence-based text chunking with configurable overlap
- Dense vector embeddings using MPNetv2 transformer model
- Distributed architecture with ZMQ for high throughput
- Comprehensive error handling and logging

Example:
    >>> from mpnetv2 import fastapi_server, inference_server, proxy_server
    >>> # Start servers in separate processes for production deployment
"""

from .fastapi import fastapi_server
from .inference import inference_server
from .proxy import proxy_server

__all__ = ["fastapi_server", "inference_server", "proxy_server"]
