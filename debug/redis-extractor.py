import base64
import json
import os

import redis

# Read environment variables
REDIS_HOST = os.getenv("REDIS_HOST", "localhost")
REDIS_PORT = int(os.getenv("REDIS_PORT", 6379))
REDIS_DB = int(os.getenv("REDIS_DB", 0))
REDIS_PASSWORD = os.getenv("REDIS_PASSWORD", None)
OUTPUT_FILE = os.getenv(
    "OUTPUT_FILE", "/data/redis-dump.json"
)  # Ensure writable path


def safe_decode(value):
    """Try decoding as UTF-8, otherwise return base64-encoded binary"""
    try:
        return value.decode("utf-8")  # Decode UTF-8 text values
    except (AttributeError, UnicodeDecodeError):  # Handles binary values
        return base64.b64encode(value).decode(
            "utf-8"
        )  # Encode binary as base64 string


def fetch_redis_data():
    try:
        # Connect to Redis (binary-safe)
        r = redis.Redis(
            host=REDIS_HOST,
            port=REDIS_PORT,
            db=REDIS_DB,
            password=REDIS_PASSWORD,
        )

        # Get all keys
        keys = r.keys("*")
        data = {}

        # Fetch values for each key
        for key in keys:
            key_str = key.decode("utf-8") if isinstance(key, bytes) else key
            data_type = r.type(key_str)

            if data_type == b"string":
                value = r.get(key_str)
                data[key_str] = safe_decode(value)
            elif data_type == b"list":
                data[key_str] = [
                    safe_decode(item) for item in r.lrange(key_str, 0, -1)
                ]
            elif data_type == b"set":
                data[key_str] = [
                    safe_decode(item) for item in r.smembers(key_str)
                ]
            elif data_type == b"hash":
                data[key_str] = {
                    safe_decode(k): safe_decode(v)
                    for k, v in r.hgetall(key_str).items()
                }
            elif data_type == b"zset":
                data[key_str] = [
                    (safe_decode(k), v)
                    for k, v in r.zrange(key_str, 0, -1, withscores=True)
                ]
            else:
                data[key_str] = (
                    f"Unsupported data type: {data_type.decode('utf-8')}"
                )

        # Save data to a file
        with open(OUTPUT_FILE, "w", encoding="utf-8") as f:
            json.dump(data, f, indent=4)

        print(f"Data successfully dumped to {OUTPUT_FILE}")

    except Exception as e:
        print(f"Error: {e}")


if __name__ == "__main__":
    fetch_redis_data()
