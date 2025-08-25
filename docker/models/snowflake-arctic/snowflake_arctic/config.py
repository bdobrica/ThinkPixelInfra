"""
Configuration module for Snowflake Arctic embedding model server.

This module defines environment-based configuration variables and constants
used throughout the application. All settings can be overridden via environment
variables, making the application configurable for different deployment environments.

Environment Variables:
    LOCAL (bool): Enable local development mode with debug logging
    MODEL_PATH (str): Path to the Snowflake Arctic embedding model
    MODEL_DEVICE (str): Device to run the model on ('cpu' or 'cuda')
    MODEL_ZMQ_CLIENT_ADDR (str): ZeroMQ client socket address
    MODEL_ZMQ_WORKER_ADDR (str): ZeroMQ worker socket address
    MODEL_HTTP_PORT (int): HTTP server port number
    MODEL_NUM_WORKERS (int): Number of worker processes for model inference

Example:
    >>> import os
    >>> os.environ['LOCAL'] = 'true'
    >>> os.environ['MODEL_DEVICE'] = 'cuda'
    >>> from snowflake_arctic.config import LOCAL, MODEL_DEVICE
    >>> print(LOCAL, MODEL_DEVICE)
    True cuda
"""

import logging
import os
from multiprocessing import cpu_count

# Environment Variables
LOCAL: bool = os.getenv("LOCAL", "false").lower() in {
    "true",
    "1",
    "yes",
    "y",
    "on",
}
MODEL_PATH: str = os.getenv("MODEL_PATH", "/app/snowflake-arctic-embed-l-v2.0")
MODEL_DEVICE: str = os.getenv("MODEL_DEVICE", "cpu")
MODEL_ZMQ_CLIENT_ADDR: str = os.getenv("MODEL_ZMQ_CLIENT_ADDR", "ipc:///tmp/snowflake-arctic.client")
MODEL_ZMQ_WORKER_ADDR: str = os.getenv("MODEL_ZMQ_WORKER_ADDR", "ipc:///tmp/snowflake-arctic.worker")
MODEL_HTTP_PORT: int = int(os.getenv("MODEL_HTTP_PORT", 8000))
MODEL_NUM_WORKERS: int = max(1, int(os.getenv("MODEL_NUM_WORKERS", cpu_count() // 2)))
MODEL_CHUNK_SIZE: int = int(os.getenv("MODEL_CHUNK_SIZE", 1000))
MODEL_CHUNK_OVERLAP: int = int(os.getenv("MODEL_CHUNK_OVERLAP", 200))

# Logging
LOG_LEVEL: int = logging.DEBUG if LOCAL else logging.WARNING
