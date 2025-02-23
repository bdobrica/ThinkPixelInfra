# LogSnap

LogSnap is a Go application designed to process API Gateway logs stored in an S3-compatible bucket (such as Minio). It aggregates JSONL log files generated hourly, extracts embedded JSON documents from the log entries, and appends the processed data into a Parquet file. This consolidated Parquet snapshot can then be used for further analysis with Python dataframe tools.

---

## Table of Contents

- [Overview](#overview)
- [Architecture and Workflow](#architecture-and-workflow)
- [Data Structure and File Paths](#data-structure-and-file-paths)
- [Setup and Installation](#setup-and-installation)
- [Running and Debugging](#running-and-debugging)
- [Containerization and Deployment](#containerization-and-deployment)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

LogSnap is built to:
- **Poll the S3 bucket hourly:** The app runs as a cron job, processing logs from the previous hour. This guarantees that the log data for that hour is complete.
- **Process JSONL logs concurrently:** Using Go's goroutines and channels, LogSnap reads and parses log files in parallel, optimizing for I/O-bound tasks.
- **Extract and merge document data:** Each log entry contains a `body` field with JSON-formatted documents. LogSnap extracts these documents and upserts them into an existing snapshot stored in a Parquet file.
- **Simplify data analysis:** The final Parquet file serves as a continuously updated snapshot of documents received via the API Gateway, making it easy to analyze with Python libraries like Pandas or PyArrow.

---

## Architecture and Workflow

1. **Determine the Previous Hour:**  
   When the cron job runs, the application calculates the previous hour (adjusting the date if necessary). This is the time window for which the logs are considered complete.

2. **Fetch Logs from the S3 Bucket:**  
   Using the computed prefix (based on cluster ID, pod name, date, and hour), the app lists and downloads relevant JSONL files.

3. **Process and Extract Data:**  
   - Each JSONL file is read line-by-line.
   - Each line (log entry) is unmarshaled into a Go struct.
   - The `body` field (which contains a JSON array) is parsed to extract individual documents.
   - Documents are upserted into an in-memory snapshot (using document IDs as keys).

4. **Merge with Existing Data:**  
   The snapshot is merged with an existing Parquet file (if present) by appending the new documents.

5. **Write Back to Parquet:**  
   The combined data is written back to the Parquet file, updating the snapshot used for further analysis.

---

## Data Structure and File Paths

### S3 Bucket Path Format

The logs are organized in the bucket using the following structure:

```
s3://<bucket-name>/logs/<cluster-id>/<pod-running-api-gateway-name>/<date-YYYY-mm-dd>/<hour>/requests.jsonl-<unique-suffix>
```

**Example:**
```
s3://dev-thinkpixel-request-logs/logs/dev.thinkpixel.io/api-gateway-5d46b74698-lqzv5/2025-02-14/20/requests.jsonl-object0nVQsy0K
```


### JSONL Log Entry Structure

Each log entry is a JSON object with several fields. The key field for processing is the `body` field, which itself is a JSON array of document objects.

**Example Log Entry:**

```json
{
    "date": "2025-02-15T16:57:35.407183Z",
    "timestamp": "2025-02-15T16:17:14.729904676Z",
    "method": "POST",
    "url": "/store",
    "headers": {
        "Accept": ["*/*"],
        "Content-Type": ["application/json"],
        "User-Agent": ["WordPress/6.7; https://ublo.ro"],
        "X-Forwarded-For": ["135.181.209.167"]
    },
    "remote_addr": "10.42.3.19:3686",
    "body": "[{\"id\":2,\"text\":\"about\\nm\\u0103 \\u0219ti\\u021bi drept Bogd\\u0103nel.\",\"extra\":{\"title\":\"about\",\"type\":\"page\"}}, ...]"
}
```

**Embedded Document Structure**

Each document within the body field typically has the following format:

```
[
    {
        "id": <document-id>,
        "text": "<document-text-content>",
        "extra": {
            "title": "<document-title>",
            "type": "<document-type>",
            ...
        }
    },
    ...
]
```
