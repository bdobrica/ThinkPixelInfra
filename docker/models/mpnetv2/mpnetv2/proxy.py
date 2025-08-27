"""
ZMQ proxy server for load balancing between API and inference workers.

This module provides a ZMQ proxy that acts as an intermediary between the
FastAPI HTTP server and the inference worker processes. It implements a
ROUTER-DEALER pattern for efficient load balancing and request distribution.

Architecture:
    Client (FastAPI) -> ROUTER -> DEALER -> Workers (Inference)

The proxy enables:
- Load balancing across multiple inference workers
- Decoupling of API server from worker processes
- Scalable architecture for high-throughput inference

Key Features:
- ROUTER-DEALER proxy pattern for automatic load balancing
- Proper socket configuration with linger settings
- Clean shutdown with resource cleanup
- Comprehensive logging for monitoring

Example:
    >>> from mpnetv2.proxy import proxy_server
    >>> proxy_server()  # Starts proxy and blocks until shutdown
"""

import logging

import zmq

from .config import LOG_LEVEL, MODEL_ZMQ_CLIENT_ADDR, MODEL_ZMQ_WORKER_ADDR

# Setup logging
logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)


def proxy_server():
    """
    Start the ZMQ proxy server for load balancing inference requests.

    Creates and manages a ZMQ proxy that routes requests from the FastAPI
    server to available inference workers. The proxy uses a ROUTER-DEALER
    pattern for automatic load balancing and fair distribution of work.

    Socket Configuration:
        - ROUTER socket: Binds to MODEL_ZMQ_CLIENT_ADDR for API server connections
        - DEALER socket: Binds to MODEL_ZMQ_WORKER_ADDR for worker connections
        - Linger time set to 0 for immediate socket closure on shutdown

    Load Balancing:
        The ROUTER-DEALER pattern automatically distributes requests to
        available workers in a round-robin fashion, providing natural
        load balancing without explicit queue management.

    Lifecycle:
        1. Create ZMQ context and sockets
        2. Bind sockets to configured addresses
        3. Start proxy (blocks until shutdown)
        4. Clean up sockets and context on exit

    Configuration:
        MODEL_ZMQ_CLIENT_ADDR: Address for API server connections
        MODEL_ZMQ_WORKER_ADDR: Address for worker connections

    Note:
        This function blocks until the proxy is shut down and should be
        run in a separate process in production deployments.
    """
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
