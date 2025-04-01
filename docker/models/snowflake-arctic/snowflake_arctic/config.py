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
MODEL_ZMQ_CLIENT_ADDR: str = os.getenv(
    "MODEL_ZMQ_CLIENT_ADDR", "ipc:///tmp/snowflake-arctic.client"
)
MODEL_ZMQ_WORKER_ADDR: str = os.getenv(
    "MODEL_ZMQ_WORKER_ADDR", "ipc:///tmp/snowflake-arctic.worker"
)
MODEL_HTTP_PORT: int = int(os.getenv("MODEL_HTTP_PORT", 8000))
MODEL_NUM_WORKERS: int = max(
    1, int(os.getenv("MODEL_NUM_WORKERS", cpu_count() // 2))
)

# Logging
LOG_LEVEL: int = logging.DEBUG if LOCAL else logging.WARNING
