# Embedding Model

The Embedding Model is a service that generates embeddings for a given input. It is used by the Search Service to generate embeddings for the search index.

## Environment Variables

- `LOCAL`: If set, the Embedding Model will run locally. Default: `false`
- `MODEL_PATH`: Path to the model weights. Default: `/app/mpnet-base-v2`
- `MODEL_DEVICE`: Device to run the model on. Default: `cpu`
- `MODEL_ZMQ_CLIENT_ADDR`: ZMQ socket address where FastAPI threads are connecting to. Default: `ipc:///tmp/mpnetv2.client`
- `MODEL_ZMQ_WORKER_ADDR`: ZMQ socket address where the Model Workers are connecting to. Default: `ipc:///tmp/mpnetv2.worker`
- `MODEL_HTTP_PORT`: HTTP port for the model. Default: `8000`
- `MODEL_NUM_WORKERS`: Number of workers for the model. Default: cpu count / 2

## Weight Files

The script downloads the weights for the model from the specified URL and saves them in the  weights  directory. The weights consist of the following files:

- `config.json` : The model configuration file
- `pytorch_model.bin` : The model weights
- `sentencepiece.bpe.model` : The SentencePiece BPE model
- `special_tokens_map.json` : Special tokens map
- `tokenizer_config.json` : Tokenizer configuration
- `tokenizer.json` : Tokenizer

The script creates the  weights  directory if it doesn’t exist and then downloads each file to the appropriate subdirectory.

## Resources

```
CONTAINER ID   NAME             CPU %     MEM USAGE / LIMIT     MEM %     NET I/O          BLOCK I/O   PIDS
ab7a97ce0e5d   sleepy_maxwell   0.22%     1.744GiB / 15.54GiB   11.22%    4.07kB / 150kB   0B / 0B     57
```

## Measured Latency

- startup: 6.3s
- single predict: 0.04s

## Example

Example request:
```bash
curl \
    -X POST "http://mpnetv2:8000/infer" \
    -H "Content-Type: application/json" \
    -d '{"text_items": [{"text": "This is a very long piece of text ...", "metadata": {"id": 1, "extra": {"key": "value"}}}]}' \
| jq
```

Example response:
```json
{
  "results": [
    {
      "text": "This is a very long piece of text ...",
      "dense_vector": "<base 64 encoded np.float32 array>",
      "metadata": {
        "id": 1,
        "extra": {
          "key": "value"
        }
      }
    }
  ],
  "latency": 0.05925534199923277
}
```

Notes:
- The `results` field is a list of output text items;
- The `text` field is the input text item;
- The `vector` field is a base64 encoded numpy array of type `np.float32`;
- The `metadata` field is optional and can be used to store additional information about the input text item. It is passed through to the output unchanged;
    - The `id` field is an integer identifier for the input text item;
    - The `extra` field is an optional dictionary of additional metadata;
- The `latency` field is the time taken to process the request in seconds.
