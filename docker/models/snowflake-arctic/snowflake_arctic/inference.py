import base64
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
    MODEL_DEVICE,
    MODEL_NUM_WORKERS,
    MODEL_PATH,
    MODEL_ZMQ_WORKER_ADDR,
)
from .tokens import build_batch_sparse_vectors

# Setup logging
logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)


# Load model and tokenizer globally (to save memory per worker)
@lru_cache(maxsize=None)
def load_model():
    global tokenizer, session

    logger.info("Loading model from %s...", MODEL_PATH)
    tokenizer = AutoTokenizer.from_pretrained(MODEL_PATH)
    providers = {
        "cpu": "CPUExecutionProvider",
        "gpu": "CUDAExecutionProvider",
        "cuda": "CUDAExecutionProvider",
    }
    session = ort.InferenceSession(
        MODEL_PATH + "/model.onnx",
        providers=[providers.get(MODEL_DEVICE, "CPUExecutionProvider")],
    )


def process_task(context: zmq.Context):
    """Worker process for running inference."""
    global tokenizer, session

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

            logger.debug(
                "Received %s text items for inference.", len(text_items)
            )

            batch_text = [item.get("text", "") for item in text_items]

            # Batch encode and process with the model
            logger.debug("Processing %s chunks...", len(batch_text))
            batch_input = tokenizer(
                batch_text, padding=True, truncation=True, return_tensors="np"
            )
            batch_tokens = map(
                tokenizer.convert_ids_to_tokens, batch_input.input_ids
            )
            model_output = session.run(
                None,
                {
                    "input_ids": batch_input.input_ids.astype("int64"),
                    "attention_mask": batch_input.attention_mask.astype(
                        "int64"
                    ),
                },
            )
            batch_dense_vectors = model_output[0]
            batch_weights = model_output[1]

            # Prepare results
            batch_sparse_vectors = build_batch_sparse_vectors(
                batch_tokens,  # type: ignore
                batch_weights,
            )

            results = []
            for text_item, dense_vector, sparse_vector in zip(
                text_items,
                batch_dense_vectors,
                batch_sparse_vectors,
            ):
                results.append(
                    {
                        "text": text_item.get("text", ""),
                        "dense_vector": base64.b64encode(
                            dense_vector.flatten().tobytes()
                        ).decode("utf-8"),
                        "sparse_vector": sparse_vector,
                        "metadata": text_item.get("metadata", {}),
                    }
                )

            logger.debug("Sending %s results...", len(results))
            response = {"results": results}
        except Exception as e:
            logger.exception("Error processing task.")
            response = {"error": str(e)}

        socket.send_multipart(
            [identity, b"", json.dumps(response).encode("utf-8")]
        )


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
