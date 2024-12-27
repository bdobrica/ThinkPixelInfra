import logging
import os

from torch.multiprocessing import cpu_count

# Environment Variables
LOCAL: bool = os.getenv("LOCAL", "false").lower() in {
    "true",
    "1",
    "yes",
    "y",
    "on",
}
MODEL_PATH: str = os.getenv("MODEL_PATH", "/app/mpnet-base-v2")
MODEL_DEVICE: str = os.getenv("MODEL_DEVICE", "cpu")
MODEL_ZMQ_CLIENT_ADDR: str = os.getenv(
    "MODEL_ZMQ_CLIENT_ADDR", "ipc:///tmp/mpnetv2.client"
)
MODEL_ZMQ_WORKER_ADDR: str = os.getenv(
    "MODEL_ZMQ_WORKER_ADDR", "ipc:///tmp/mpnetv2.worker"
)
MODEL_HTTP_PORT: int = int(os.getenv("MODEL_HTTP_PORT", 8000))
MODEL_TEXT_MAX_LENGTH: int = int(os.getenv("MODEL_TEXT_MAX_LENGTH", 128))
MODEL_TEXT_SPLIT_OVERLAP: int = int(os.getenv("MODEL_TEXT_SPLIT_OVERLAP", 20))
MODEL_NUM_WORKERS: int = max(
    1, int(os.getenv("MODEL_NUM_WORKERS", cpu_count() // 2))
)

# Logging
LOG_LEVEL: int = logging.DEBUG if LOCAL else logging.WARNING
