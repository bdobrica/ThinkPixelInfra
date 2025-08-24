"""
ZeroMQ proxy module for routing requests between clients and inference workers.

This module provides a ZeroMQ proxy server that acts as a load balancer between
client requests and inference worker processes. It uses the ROUTER-DEALER pattern
to distribute requests across multiple workers efficiently.

Functions:
    proxy_server: Main proxy server function that starts the ZMQ proxy

Architecture:
- ROUTER socket: Receives requests from FastAPI clients
- DEALER socket: Distributes requests to inference workers
- Load balancing: Automatically distributes work across available workers
- Graceful shutdown: Handles socket cleanup with linger settings

The proxy enables:
- Scalable inference serving with multiple worker processes
- Automatic load balancing without client-side logic
- Decoupling of HTTP API server from inference workers
- Fault tolerance through worker process isolation
"""

import logging

import zmq

from .config import LOG_LEVEL, MODEL_ZMQ_CLIENT_ADDR, MODEL_ZMQ_WORKER_ADDR

# Setup logging
logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)


def proxy_server():
    """Start the ZMQ proxy server."""
    context = zmq.Context()

    client_socket: zmq.Socket = context.socket(zmq.ROUTER)
    client_socket.setsockopt(zmq.LINGER, 0)
    client_socket.bind(MODEL_ZMQ_CLIENT_ADDR)

    worker_socket: zmq.Socket = context.socket(zmq.DEALER)
    worker_socket.setsockopt(zmq.LINGER, 0)
    worker_socket.bind(MODEL_ZMQ_WORKER_ADDR)

    try:
        logger.info(
            "Starting ZMQ proxy server binding %s to %s...",
            MODEL_ZMQ_CLIENT_ADDR,
            MODEL_ZMQ_WORKER_ADDR,
        )
        zmq.proxy(client_socket, worker_socket)
    finally:
        logger.info("Closing ZMQ proxy server sockets and terminating context...")
        client_socket.close()
        worker_socket.close()
        context.term()
