# External Embeddings Gateway

The external model service is an embeddings gateway that keeps the same `/infer` request and response contract as the local embedding services, but delegates dense vector generation to a remote provider. The current safe default is OpenAI `text-embedding-3-small` for dense vectors, with locally generated lexical sparse vectors so downstream semantic search code can keep consuming the e5-compatible response shape.

## Behavior

- Dense vectors come from the configured provider.
- Sparse vectors are generated locally from lemmatized tokens and normalized lexical weights.
- `mode` stays split between `search` and `store` for compatibility, even when both modes currently use empty prefixes.
- Language detection, sentence splitting, and sparse-vector preparation run off the event loop in a bounded thread pool.

## Environment Variables

- `LOCAL`: Enable debug logging. Default: `false`
- `MODEL_HTTP_PORT`: HTTP port for the service. Default: `8000`
- `MODEL_CHUNK_SIZE`: Maximum characters per chunk. Default: `1000`. Set to `0` to disable splitting.
- `MODEL_CHUNK_OVERLAP`: Character overlap between chunks. Default: `200`
- `MODEL_PREPROCESS_THREADS`: Thread-pool size for language detection, chunking, and sparse-vector generation. Default: `cpu_count() / 2`
- `MODEL_PROVIDER`: Dense embedding provider. Default: `openai`
- `MODEL_PROVIDER_MODEL`: Dense embedding model name. Default: `text-embedding-3-small`
- `MODEL_PROVIDER_BASE_URL`: Provider base URL. Default: `https://api.openai.com/v1`
- `MODEL_PROVIDER_API_KEY`: Provider API key. Required.
- `MODEL_PROVIDER_BATCH_SIZE`: Number of prepared chunks per embeddings request. Default: `32`
- `MODEL_PROVIDER_TIMEOUT_SECONDS`: Provider request timeout. Default: `30`
- `MODEL_PROVIDER_MAX_CONNECTIONS`: Shared HTTP connection limit. Default: `256`
- `MODEL_PROVIDER_MAX_KEEPALIVE_CONNECTIONS`: Shared keep-alive connection limit. Default: `64`
- `MODEL_PROVIDER_KEEPALIVE_EXPIRY_SECONDS`: Keep-alive expiry. Default: `30`
- `MODEL_PROVIDER_DNS_TTL_SECONDS`: Optional DNS cache TTL. Default: disabled
- `MODEL_PROVIDER_RETRY_MAX_RETRIES`: Retry attempts for retryable provider failures. Default: `2`
- `MODEL_PROVIDER_RETRY_INITIAL_BACKOFF_MS`: Initial retry backoff. Default: `250`
- `MODEL_PROVIDER_RETRY_MAX_BACKOFF_MS`: Maximum retry backoff. Default: `1000`
- `MODEL_PROVIDER_RETRY_JITTER_RATIO`: Backoff jitter ratio. Default: `0.2`
- `MODEL_SPARSE_STRATEGY`: Sparse-vector strategy. Supported values: `lexical`, `off`. Default: `lexical`
- `MODEL_SPARSE_MAX_FEATURES`: Maximum sparse features kept per chunk. Default: `128`
- `MODEL_SPARSE_MIN_TOKEN_LENGTH`: Minimum lemma length kept in sparse vectors. Default: `2`
- `MODEL_SEARCH_PREFIX`: Prefix prepended in `search` mode. Default: empty string
- `MODEL_STORE_PREFIX`: Prefix prepended in `store` mode. Default: empty string
- `MODEL_LANGUAGES`: Allowed languages for spaCy processing. Default: `en`
- `MODEL_LANGUAGE_DETECTION_PATH`: Path to the fastText language detection model. Default: `/app/fasttext/lid.176.bin`

## Example

Example request:
```bash
curl \
    -X POST "http://localhost:8000/infer" \
    -H "Content-Type: application/json" \
    -d '{"text_items": [{"text": "This is a very long piece of text ...", "metadata": {"id": 1, "extra": {"key": "value"}}}], "mode": "store"}' \
| jq
```

Example response:
```json
{
  "results": [
    {
      "text": "This is a very long piece of text ...",
      "offset": 0,
      "dense_vector": "<base64 encoded np.float32 array>",
      "sparse_vector": {
        "<base64 encoded >L packed mmh3 of lemma>": "<base64 encoded >f packed float32 normalized weight>"
      },
      "metadata": {
        "id": 1,
        "extra": {
          "key": "value",
          "language": "en"
        }
      }
    }
  ],
  "latency": 0.083
}
```

## Notes

- The response contract matches the local embedding services: `text`, `offset`, `dense_vector`, `sparse_vector`, `metadata`, and `latency`.
- Dense vectors are returned as base64-encoded big-endian `float32` arrays.
- Sparse-vector keys are MMH3 hashes of normalized lemmas, encoded as base64 big-endian unsigned integers.
- Sparse-vector values are base64 big-endian `float32` weights normalized across the kept lexical features.
- If the fastText model or package is unavailable, language detection falls back to `unk` and chunking degrades safely to whole-text chunks.
