# Snowflake Arctic Embedding Model

The [Snowflake Arctic](https://huggingface.co/Snowflake/snowflake-arctic-embed-l-v2.0) Embedding Model is a service that generates embeddings for a given input together with token weights that can be used by a [BM42](https://qdrant.tech/articles/bm42/) like algorithm. It is used by the Search Service to generate embeddings for the search index.

## Environment Variables

- `LOCAL`: If set, the Embedding Model will run locally. Default: `false`
- `MODEL_CHUNK_OVERLAP`: Overlap between text chunks. Default: `200`
- `MODEL_CHUNK_SIZE`: Size of text chunks for processing. Default: `1000`. Set to `0` to disable splitting and return entire text
- `MODEL_DEVICE`: Device to run the model on. The model is saved in ONNX format and optimised for CPU. Default: `cpu`
- `MODEL_HTTP_PORT`: HTTP port for the model. Default: `8000`
- `MODEL_NUM_WORKERS`: Number of workers for the model. Default: cpu count / 2
- `MODEL_PATH`: Path to the model weights. Default: `/app/snowflake-arctic-embed-l-v2`
- `MODEL_ZMQ_CLIENT_ADDR`: ZMQ socket address where FastAPI threads are connecting to. Default: `ipc:///tmp/snowflake-arctic.client`
- `MODEL_ZMQ_WORKER_ADDR`: ZMQ socket address where the Model Workers are connecting to. Default: `ipc:///tmp/snowflake-arctic.worker`

## Weight Files

The script downloads the weights for the model from the specified URL and saves them in the  weights  directory. The weights consist of the following files:

- `model.onnx` : The model configuration file
- `model.onnx_data.??` : The model weights. As this is a large file, it is split into multiple parts of ~200MB each.
- `model.onnx_data.sha256sum` : The SHA256 checksum of the model weights to verify the integrity of the downloaded files and their joining.
- `special_tokens_map.json` : Special tokens map
- `tokenizer.json` : Tokenizer configuration
- `tokenizer_config.json` : Tokenizer configuration

The script creates the  weights  directory if it doesn’t exist and then downloads each file to the appropriate subdirectory.

## Resources

Memory and CPU usage of the container running the model (with only one worker):

```
CONTAINER ID   NAME                                CPU %     MEM USAGE / LIMIT     MEM %     NET I/O       BLOCK I/O   PIDS
ea8fbbc7900f   dreamy_leakey                       0.15%     1.464GiB / 15.54GiB   9.42%     586B / 0B     0B / 0B     16
```

With 4 workers:

```
CONTAINER ID   NAME                                CPU %     MEM USAGE / LIMIT     MEM %     NET I/O       BLOCK I/O   PIDS
68e7ad1b029d   focused_shannon                     0.18%     5.597GiB / 15.54GiB   36.00%    656B / 0B     0B / 0B     34
```


## Measured Latency

- startup: 1.42s
- single predict: 0.07s

## Example

Example request:
```bash
curl \
    -X POST "http://snowflake-arctic:8000/infer" \
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
      "offset": 0,
      "dense_vector": "<base 64 encoded np.float32 array>",
      "sparse_vector": {
        "<base 64 encoded >L packed mmh3 of `long`>": "<base 64 encoded >f packed float32 normalized weight>",
        "<base 64 encoded >L packed mmh3 of `piece`>": "<base 64 encoded >f packed float32 normalized weight>",
        "<base 64 encoded >L packed mmh3 of `text`>": "<base 64 encoded >f packed float32 normalized weight>",
      },
      "metadata": {
        "id": 1,
        "extra": {
          "key": "value"
        }
      }
    }
  ],
  "latency": 0.07278088799648685
}
```

Notes:
- The `results` field is a list of output text items;
- The `text` field is the input text item;
- The `offset` field is the character offset of the chunk in the original text; as now the model splits long texts into chunks, this is useful to identify the position of the chunk in the original text;
- The `dense_vector` field is a base64 encoded numpy array of type `np.float32`; the vector has dimensions of 1024x1;
- The `sparse_vector` field is a dictionary of token integer indices and their float normalized weights, where:
    - the key is the base64 packed big-endian unsigned long pack of absolute value of mmh3 of token; special tokens and punctuation are excluded from the token weights; tokens are converted to their lowercase lemma form; supported languages are English, French, German, Spanish, Italian and Romanian via spacy;
    - the value is a base64 encoded packed big-endian float32; the number represents the summed average attention (for each head) of the token relative to `<s>` token in the input text item; the weights are normalized to 1.0;
- The `metadata` field is optional and can be used to store additional information about the input text item. It is passed through to the output unchanged;
    - The `id` field is an integer identifier for the input text item;
    - The `extra` field is an optional dictionary of additional metadata;
- The `latency` field is the time taken to process the request in seconds.
