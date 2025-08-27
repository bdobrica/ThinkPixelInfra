"""
MPNetv2 Main Entry Point

This module serves as the main entry point for the MPNetv2 embedding service.
It orchestrates three server processes that work together to provide a distributed
text embedding API with intelligent preprocessing capabilities.

Server Components:
- proxy_server: ZMQ ROUTER-DEALER proxy for load balancing requests
- fastapi_server: HTTP API server providing REST endpoints for inference
- inference_server: Worker pool running MPNetv2 model inference with text preprocessing

The main process monitors all server processes and ensures they run together.
If any process exits unexpectedly, all processes are terminated to maintain
system consistency.

Features:
- Multi-process architecture for scalability and fault isolation
- Automatic process monitoring and cleanup
- Graceful shutdown handling with proper resource cleanup
- Comprehensive logging for debugging and monitoring

Usage:
    python -m mpnetv2

Environment Variables:
    MODEL_HTTP_PORT: HTTP server port (default: 8000)
    MODEL_NUM_WORKERS: Number of inference workers (default: CPU count / 2)
    MODEL_ZMQ_CLIENT_ADDR: ZMQ client socket address
    MODEL_ZMQ_WORKER_ADDR: ZMQ worker socket address
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
    """
    Start and monitor all server processes for the MPNetv2 embedding service.

    This function creates and manages three server processes:
    1. proxy_server: ZMQ proxy for load balancing between API and workers
    2. fastapi_server: HTTP API server for client requests
    3. inference_server: Worker pool for model inference

    The function monitors all processes and terminates the entire system if any
    process exits unexpectedly, ensuring system consistency.

    Returns:
        tuple: (process_name, exit_code) of the first process that exited

    Raises:
        KeyboardInterrupt: Handled gracefully with proper cleanup
    """
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
