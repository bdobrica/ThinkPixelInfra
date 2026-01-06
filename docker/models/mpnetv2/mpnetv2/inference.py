"""
Inference server module for MPNetv2 text embedding generation.

This module provides the core inference functionality for the MPNetv2 embedding
service, including model loading, text preprocessing, and distributed worker
management. It handles text chunking, language detection, and dense vector
generation using transformer models.

Key Components:
- EmptyBatchError: Custom exception for empty input validation
- Model loading and caching with PyTorch and Transformers
- Multi-process worker pool for parallel inference
- ZMQ-based communication for distributed processing
- Comprehensive text preprocessing pipeline

Features:
- Automatic language detection and spaCy model selection
- Intelligent sentence-based text chunking with overlap
- Mean pooling for transformer output aggregation
- Error handling for edge cases and empty inputs
- Process monitoring and graceful shutdown

Architecture:
    The inference server creates a pool of worker processes, each loading
    the MPNetv2 model independently. Workers communicate via ZMQ REP sockets,
    receiving requests and returning embedding results.

Example:
    >>> from mpnetv2.inference import inference_server
    >>> inference_server()  # Starts worker pool and manages processes
"""

import json
import logging
import time
from functools import lru_cache

import torch
import zmq
from torch.multiprocessing import Process
from transformers import AutoModel, AutoTokenizer

from .config import (
    LOCAL,
    LOG_LEVEL,
    MODEL_BATCH_SIZE,
    MODEL_CHUNK_OVERLAP,
    MODEL_CHUNK_SIZE,
    MODEL_DEVICE,
    MODEL_NUM_WORKERS,
    MODEL_PATH,
    MODEL_ZMQ_WORKER_ADDR,
)
from .postprocess import build_results
from .preprocess import prepare_text_items

# Setup logging
logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)


class EmptyBatchError(Exception):
    """
    Custom exception raised when no valid text items are provided for inference.

    This exception is raised in two scenarios:
    1. When the input contains no text items
    2. When all text items are invalid or empty after preprocessing

    This allows for graceful handling of edge cases where clients send
    empty requests or requests with only whitespace/invalid content.
    """

    pass


# Load model and tokenizer globally (to save memory per worker)
@lru_cache(maxsize=None)
def load_model():
    """
    Load and cache the MPNetv2 model and tokenizer for inference.

    This function loads the transformer model and tokenizer from the configured
    path and moves the model to the specified device (CPU/GPU). The function
    is cached to ensure each worker process loads the model only once.

    Global Variables:
        device (torch.device): PyTorch device for model inference
        tokenizer (AutoTokenizer): Transformer tokenizer for text preprocessing
        model (AutoModel): MPNetv2 transformer model for embedding generation

    Environment Configuration:
        MODEL_PATH: Path to the model directory
        MODEL_DEVICE: Target device for model inference (cpu/cuda)

    Note:
        This function should be called once per worker process to initialize
        the model for inference operations.
    """
    global device, tokenizer, model

    logger.info("Loading model from %s...", MODEL_PATH)
    device = torch.device(MODEL_DEVICE)
    tokenizer = AutoTokenizer.from_pretrained(MODEL_PATH)
    model = AutoModel.from_pretrained(MODEL_PATH).to(device)


def _process_task(context: zmq.Context):
    """
    Worker process function for handling inference requests.

    This function runs in each worker process and handles the complete inference
    pipeline for text embedding generation:

    1. Text preprocessing with language detection and chunking
    2. Tokenization and model inference
    3. Result aggregation and formatting
    4. Error handling and response generation

    The worker communicates via ZMQ REP sockets, receiving JSON requests
    and returning JSON responses with embedding results.

    Args:
        context (zmq.Context): ZMQ context for socket communication

    Processing Pipeline:
        1. Receive and parse JSON request
        2. Extract text items and chunking parameters
        3. Preprocess texts (language detection, chunking)
        4. Tokenize text chunks
        5. Generate embeddings with MPNetv2 model
        6. Post-process results (mean pooling, base64 encoding)
        7. Send JSON response with results or error

    Error Handling:
        - EmptyBatchError: Gracefully handle empty input
        - General exceptions: Log error and return error response

    Global Dependencies:
        Uses globally loaded model, tokenizer, and device from load_model()
    """
    global device, tokenizer, model

    load_model()

    logger.info("Worker process started.")
    socket = context.socket(zmq.REP)
    socket.connect(MODEL_ZMQ_WORKER_ADDR)

    while True:
        # Receive request
        identity, _, request = socket.recv_multipart()
        logger.debug("Received request: %s", request)

        try:
            # Parse the request
            data = json.loads(request)
            text_items = data.get("text_items", [])
            chunk_size = data.get("chunk_size", MODEL_CHUNK_SIZE)
            chunk_overlap = data.get("chunk_overlap", MODEL_CHUNK_OVERLAP)
            language = data.get("language", "auto")

            if not text_items:
                raise EmptyBatchError("No text items provided for inference.")

            logger.debug("Received %s text items for inference.", len(text_items))

            text_items = prepare_text_items(
                text_items,
                chunk_size=chunk_size,
                chunk_overlap=chunk_overlap,
                language=language,
            )

            if not text_items:
                raise EmptyBatchError("No valid text items after preparation.")
            logger.debug("Prepared %s text items for inference.", len(text_items))

            results = []

            for index in range(0, len(text_items), MODEL_BATCH_SIZE):
                batch_items = text_items[index : index + MODEL_BATCH_SIZE]  # noqa: E203

                logger.debug("Working with %s text items for inference.", len(batch_items))

                batch_text = [item.get("text", "") for item in batch_items]

                # Batch encode and process with the model
                logger.debug(
                    "Processing %s chunks of %s bytes...",
                    len(batch_text),
                    sum(len(text.encode("utf-8")) for text in batch_text),
                )
                batch_input = tokenizer(batch_text, padding=True, truncation=True, return_tensors="pt").to(device)
                with torch.no_grad():
                    model_output = model(**batch_input)
                results.extend(build_results(batch_items, batch_input, model_output))
                batch_items = None
                batch_input = None
                model_output = None
            logger.debug("Sending %s results...", len(results))
            response = {"results": results}
        except EmptyBatchError as e:
            logger.warning("Empty batch received: %s", e)
            response = {"results": []}
        except Exception as e:
            logger.exception("Error processing task.")
            response = {"error": str(e)}

        socket.send_multipart([identity, b"", json.dumps(response).encode("utf-8")])


def process_task(context: zmq.Context):
    """
    Wrapper function for worker process with exception handling.

    Provides a safe wrapper around the main worker function to ensure
    proper error logging and graceful process termination in case of
    unexpected errors.

    Args:
        context (zmq.Context): ZMQ context for socket communication

    Note:
        This function is the actual target for worker processes, providing
        isolation and proper cleanup even if the worker encounters errors.
    """
    try:
        _process_task(context)
    except Exception as e:
        logger.error("Worker process encountered an error: %s", e)
    finally:
        logger.info("Worker process exiting.")


def inference_server():
    """
    Start and manage the inference server with worker processes.

    This function creates and manages a pool of worker processes for handling
    inference requests. It monitors worker health and provides graceful shutdown
    capabilities.

    Architecture:
        - Creates a ZMQ context for inter-process communication
        - Spawns configured number of worker processes
        - Monitors worker process health continuously
        - Handles shutdown signals and cleanup

    Worker Management:
        - Workers are created as daemon processes (in non-LOCAL mode)
        - Continuous monitoring ensures system reliability
        - Automatic termination if any worker exits unexpectedly
        - Graceful cleanup on shutdown or error

    Configuration:
        MODEL_NUM_WORKERS: Number of worker processes to create
        LOCAL: Development mode flag affecting daemon process behavior

    Signals Handled:
        - KeyboardInterrupt: Graceful shutdown with worker termination
        - Worker exit: Automatic system shutdown for consistency

    Note:
        This function blocks until shutdown and should be run in a
        separate process in production deployments.
    """
    # Setup ZMQ context
    context = zmq.Context()

    # Start worker processes
    workers = []
    for _ in range(MODEL_NUM_WORKERS):
        worker = Process(target=process_task, args=(context,), daemon=not LOCAL)
        worker.start()
        workers.append(worker)

    try:
        while True:
            if not all(worker.is_alive() for worker in workers):
                logger.error("Worker process has exited.")
                raise RuntimeError("Worker process has exited.")
            time.sleep(1)
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received. Terminating workers...")
    except RuntimeError:
        logger.error("Worker process has exited. Terminating workers...")
    finally:
        for worker in workers:
            if worker.is_alive():
                worker.terminate()
                worker.join()
        context.term()
