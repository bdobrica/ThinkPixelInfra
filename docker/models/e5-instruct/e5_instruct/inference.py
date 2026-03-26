"""
Inference module for Multi-Lingual E5 Large Instruct embedding model with ZeroMQ communication.

This module provides the core inference functionality for the Multi-Lingual E5 Large Instruct embedding
model using ONNX Runtime. It handles model loading, batch processing, and ZeroMQ-based
worker communication for scalable inference serving.

Classes:
    EmptyBatchError: Custom exception for empty input batches
    InferenceWorker: Worker process for handling inference requests

Functions:
    start_inference_workers: Starts multiple inference worker processes

Key Features:
- ONNX Runtime-based model inference with configurable device placement
- Multi-process worker architecture using ZeroMQ for communication
- Batch processing with automatic empty batch handling
- Integration with preprocessing and postprocessing pipelines
- Support for both dense and sparse vector outputs
- Configurable number of workers for scaling

The inference pipeline:
1. Receives text items via ZeroMQ
2. Preprocesses text using tokenizer
3. Runs ONNX model inference
4. Postprocesses results with offset extraction
5. Returns embedding vectors and metadata
"""

import json
import logging
import time
from functools import lru_cache
from multiprocessing import Process

import onnxruntime as ort
import zmq
from transformers import AutoTokenizer

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
from .search import compute_search_prefix_length

# Setup logging
logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)


class EmptyBatchError(Exception):
    """Custom exception for empty input batches."""

    pass


# Load model and tokenizer globally (to save memory per worker)
@lru_cache(maxsize=None)
def load_model():
    global tokenizer, session, search_prefix_length

    logger.info("Loading model from %s...", MODEL_PATH)
    tokenizer = AutoTokenizer.from_pretrained(MODEL_PATH, use_fast=True)

    # Precompute search prefix lengths for all supported languages to remove them from token lists during postprocessing
    # Added 1 to the length of the prefix to account for missing start token in the tokenized output.
    compute_search_prefix_length(lambda prefix: 1 + len(tokenizer([prefix], add_special_tokens=False).input_ids[0]))

    providers = {
        "cpu": "CPUExecutionProvider",
        "gpu": "CUDAExecutionProvider",
        "cuda": "CUDAExecutionProvider",
    }
    session = ort.InferenceSession(
        MODEL_PATH + "/model.onnx",
        providers=[providers.get(MODEL_DEVICE, "CPUExecutionProvider")],
    )


def _process_task(context: zmq.Context):
    """Worker process for running inference."""
    global tokenizer, session, search_prefix_length

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
            language = data.get("language", "auto")
            mode = data.get("mode", "store")
            chunk_size = data.get("chunk_size", MODEL_CHUNK_SIZE)
            chunk_overlap = data.get("chunk_overlap", MODEL_CHUNK_OVERLAP)

            if not text_items:
                raise EmptyBatchError("No text items provided for inference.")

            logger.debug("Received %s text items for inference.", len(text_items))

            text_items = prepare_text_items(
                text_items,
                language=language,
                mode=mode,
                chunk_size=chunk_size,
                chunk_overlap=chunk_overlap,
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
                batch_input = tokenizer(
                    batch_text,
                    padding=True,
                    truncation=True,
                    return_tensors="np",
                    max_length=512,
                )
                batch_tokens = map(tokenizer.convert_ids_to_tokens, batch_input.input_ids)
                model_output = session.run(
                    None,
                    {
                        "input_ids": batch_input.input_ids.astype("int64"),
                        "attention_mask": batch_input.attention_mask.astype("int64"),
                    },
                )
                batch_input = None
                results.extend(build_results(batch_items, batch_tokens, model_output))
                batch_items = None
                batch_tokens = None
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
    """Process task in a separate worker process."""
    try:
        _process_task(context)
    except Exception as e:
        logger.error("Worker process encountered an error: %s", e)
    finally:
        logger.info("Worker process exiting.")


def inference_server():
    """Start the inference server."""
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
