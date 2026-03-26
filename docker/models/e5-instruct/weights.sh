#!/bin/sh

# Define the base URL
BASE_URL="https://ublo.ro/wp-content/mirror"

# Define the files to download
FILES="\
fasttext/lid.176.bin \
e5-instruct/multilingual-e5-large-instruct/config.json \
e5-instruct/multilingual-e5-large-instruct/model.onnx \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.aa \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.ab \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.ac \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.ad \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.ae \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.af \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.ag \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.ah \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.ai \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.aj \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.ak \
e5-instruct/multilingual-e5-large-instruct/model.onnx_data.sha256sum \
e5-instruct/multilingual-e5-large-instruct/sentencepiece.bpe.model \
e5-instruct/multilingual-e5-large-instruct/special_tokens_map.json \
e5-instruct/multilingual-e5-large-instruct/tokenizer.json \
e5-instruct/multilingual-e5-large-instruct/tokenizer_config.json"

# Create the weights folder if it doesn't exist
WEIGHTS_DIR="weights"
if [ ! -d "$WEIGHTS_DIR" ]; then
    echo "Creating weights directory..."
    mkdir -p "$WEIGHTS_DIR"
fi

# Path to the assembled model file
ASSEMBLED="$WEIGHTS_DIR/e5-instruct/multilingual-e5-large-instruct/model.onnx_data"

# If the assembled file already exists, we'll skip downloading parts and joining
if [ -f "$ASSEMBLED" ]; then
    echo "Assembled model exists at $ASSEMBLED; skipping part downloads and join."
    SKIP_PARTS=true
else
    SKIP_PARTS=false
fi

# Download each file
for FILE in $FILES; do
    # Create subdirectories if needed
    DIR="$WEIGHTS_DIR/$(dirname "$FILE")"
    if [ ! -d "$DIR" ]; then
        echo "Creating directory: $DIR"
        mkdir -p "$DIR"
    fi

    # Download the file
    URL="$BASE_URL/$FILE"
    DEST="$WEIGHTS_DIR/$FILE"
    BASENAME=$(basename "$FILE")

    # If we've already assembled, skip any .onnx_data.?? parts
    if [ "$SKIP_PARTS" = true ] && echo "$BASENAME" | grep -Eq '^model\.onnx_data\.[[:alnum:]]{2}$'; then
        echo "Skipping part file (already assembled): $BASENAME"
        continue
    fi

    # Make sure the subdirectory exists
    mkdir -p "$(dirname "$DEST")"

    # If the specific file already exists, skip
    if [ -f "$DEST" ]; then
        echo "File already exists: $DEST"
        continue
    fi

    # Download it
    echo "Downloading $URL to $DEST..."
    curl -fSL "$URL" -o "$DEST"

    # Check if the download was successful
    if [ $? -ne 0 ]; then
        echo "Failed to download $URL"
        exit 1
    fi
done

# If we didn’t already have an assembled model, stitch and verify
# Splitting can be done with the following command:
#   `split -b 200M -a 2 model.onnx_data model.onnx_data.`
if [ "$SKIP_PARTS" = false ]; then
    echo "Joining part files into model.onnx_data..."
    (
        cd "$WEIGHTS_DIR/e5-instruct/multilingual-e5-large-instruct" || exit 1
        cat model.onnx_data.a* > model.onnx_data
        rm model.onnx_data.a*
        echo "Verifying checksum..."
        sha256sum -c model.onnx_data.sha256sum
    ) || (
        echo "Failed to assemble or verify the model."
        exit 1
    )
else
    echo "Skipping join step—already have $ASSEMBLED."
fi

echo "All files have been downloaded successfully."
