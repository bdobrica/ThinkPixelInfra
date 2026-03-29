"""
Main entry point for the jina-embeddings-v5-text-small-retrieval: Retrieval-Targeted Embedding Distillation server.

This module orchestrates the complete embedding service by starting and managing
multiple processes: FastAPI HTTP server, ZeroMQ proxy, and inference workers.
It provides a unified entry point for running the embedding service in production.

Functions:
    start_processes: Creates and manages all service processes
    main: Entry point with command-line argument handling

Process Architecture:
- FastAPI Server: Handles HTTP requests and health checks
- ZMQ Proxy: Routes requests between API server and workers
- Inference Workers: Multiple processes running the embedding model
- Process Management: Monitors and gracefully shuts down all processes

The service can run in two modes:
- Production mode: All processes (API + proxy + workers)
- Local mode: FastAPI only (for development and testing)

Usage:
    python -m jina_v5 [--local]
"""

import logging
import multiprocessing
import sys
import time

from .fastapi import fastapi_server
from .inference import inference_server
from .proxy import proxy_server

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Set multiprocessing start method
mp_context = multiprocessing.get_context("fork")


def start_processes():
    # Start the subprocesses
    processes = {
        "proxy_server": mp_context.Process(target=proxy_server, name="proxy_server"),
        "fastapi_server": mp_context.Process(target=fastapi_server, name="fastapi_server"),
        "inference_server": mp_context.Process(target=inference_server, name="inference_server"),
    }

    logger.info("Starting processes...")
    for name, process in processes.items():
        process.start()
        logger.info("Started process %s with PID %s.", name, process.pid)
    time.sleep(1)  # Wait for the servers to start

    try:
        while True:
            for name, process in processes.items():
                if not process.is_alive():  # Process has exited
                    retcode = process.exitcode
                    logger.error("Process %s exited with code %s.", name, retcode)
                    # Terminate the other process
                    for other_name, other_process in processes.items():
                        if other_name != name:
                            logger.warning("Terminating %s...", other_name)
                            other_process.terminate()
                            other_process.join()  # Ensure it has terminated
                    return name, retcode
            time.sleep(1)  # Avoid busy waiting
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received. Terminating processes...")
        for name, process in processes.items():
            process.terminate()
            process.join()
        sys.exit(1)


if __name__ == "__main__":
    script_name, exit_code = start_processes()
    logger.info(
        "Main process: %s exited with code %s. Exiting main process.",
        script_name,
        exit_code,
    )
