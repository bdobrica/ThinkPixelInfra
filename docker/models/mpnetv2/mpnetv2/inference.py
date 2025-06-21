import base64
import json
import logging
import time
from functools import lru_cache
from typing import Any

import torch
import zmq
from torch.multiprocessing import Process
from transformers import AutoModel, AutoTokenizer

from .config import (
    LOCAL,
    LOG_LEVEL,
    MODEL_DEVICE,
    MODEL_NUM_WORKERS,
    MODEL_PATH,
    MODEL_ZMQ_WORKER_ADDR,
)

# Setup logging
logging.basicConfig(level=LOG_LEVEL)
logger = logging.getLogger(__name__)


# Load model and tokenizer globally (to save memory per worker)
@lru_cache(maxsize=None)
def load_model():
    global device, tokenizer, model

    logger.info("Loading model from %s...", MODEL_PATH)
    device = torch.device(MODEL_DEVICE)
    tokenizer = AutoTokenizer.from_pretrained(MODEL_PATH)
    model = AutoModel.from_pretrained(MODEL_PATH).to(device)


def mean_pooling(token_embeddings: torch.Tensor, attention_mask: Any) -> torch.Tensor:
    """
    Mean Pooling - Take attention mask into account for correct averaging
    :param model_output: Model output
    :param attention_mask: Attention mask
    :return: Mean pooled vector
    """
    input_mask_expanded = attention_mask.unsqueeze(-1).expand(token_embeddings.size()).float()
    sum_embeddings = torch.sum(token_embeddings * input_mask_expanded, 1)
    sum_mask = torch.clamp(input_mask_expanded.sum(1), min=1e-9)
    return sum_embeddings / sum_mask


def process_task(context: zmq.Context):
    """Worker process for running inference."""
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

            logger.debug("Received %s text items for inference.", len(text_items))

            text_batch = [item.get("text", "") for item in text_items]

            # Batch encode and process with the model
            logger.debug("Processing %s chunks...", len(text_batch))
            encoded_input = tokenizer(text_batch, padding=True, truncation=True, return_tensors="pt").to(device)
            with torch.no_grad():
                model_output = model(**encoded_input)
                pooled_output = mean_pooling(
                    model_output.last_hidden_state,
                    encoded_input["attention_mask"],
                )

            # Prepare results
            logger.debug("Processing model output...")
            vector_batch = pooled_output.cpu().numpy()
            logger.debug(
                "Model output shape: %s, dtype: %s",
                vector_batch.shape,
                vector_batch.dtype,
            )
            results = []
            for i, vector in enumerate(vector_batch):
                encoded_vector = base64.b64encode(vector.flatten().astype(">f4").tobytes()).decode("utf-8")
                results.append(
                    {
                        "text": text_items[i].get("text", ""),
                        "dense_vector": encoded_vector,
                        "metadata": text_items[i].get("metadata", {}),
                    }
                )

            logger.debug("Sending %s results...", len(results))
            response = {"results": results}
        except Exception as e:
            logger.exception("Error processing task.")
            response = {"error": str(e)}

        socket.send_multipart([identity, b"", json.dumps(response).encode("utf-8")])


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
