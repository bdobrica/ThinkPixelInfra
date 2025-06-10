#!/bin/sh

# Define the base URL
BASE_URL="https://ublo.ro/wp-content/mirror"

# Define the files to download
# Used: `split -b 200M pytorch_model.bin pytorch_model.bin.`
FILES="\
mpnetv2/mpnet-base-v2/config.json \
mpnetv2/mpnet-base-v2/pytorch_model.bin.aa \
mpnetv2/mpnet-base-v2/pytorch_model.bin.ab \
mpnetv2/mpnet-base-v2/pytorch_model.bin.ac \
mpnetv2/mpnet-base-v2/pytorch_model.bin.ad \
mpnetv2/mpnet-base-v2/pytorch_model.bin.ae \
mpnetv2/mpnet-base-v2/pytorch_model.bin.af \
mpnetv2/mpnet-base-v2/pytorch_model.bin.sha256sum \
mpnetv2/mpnet-base-v2/sentencepiece.bpe.model \
mpnetv2/mpnet-base-v2/special_tokens_map.json \
mpnetv2/mpnet-base-v2/tokenizer_config.json \
mpnetv2/mpnet-base-v2/tokenizer.json"

# Create the weights folder if it doesn't exist
WEIGHTS_DIR="weights"
if [ ! -d "$WEIGHTS_DIR" ]; then
    echo "Creating weights directory..."
    mkdir -p "$WEIGHTS_DIR"
fi

# Path to the assembled model file
ASSEMBLED="$WEIGHTS_DIR/mpnetv2/mpnet-base-v2/pytorch_model.bin"

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

    # If we've already assembled, skip any pytorch_model.bin.?? parts
    if [ "$SKIP_PARTS" = true ] && echo "$BASENAME" | grep -Eq '^pytorch_model\.bin\.[[:alnum:]]{2}$'; then
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
if [ "$SKIP_PARTS" = false ]; then
    echo "Joining part files into model.onnx_data..."
    (
        cd "$WEIGHTS_DIR/mpnetv2/mpnet-base-v2" || exit 1
        cat pytorch_model.bin.a* > pytorch_model.bin
        rm pytorch_model.bin.a*
        echo "Verifying checksum..."
        sha256sum -c pytorch_model.bin.sha256sum
    ) || (
        echo "Failed to assemble or verify the model."
        exit 1
    )
else
    echo "Skipping join step—already have $ASSEMBLED."
fi

echo "All files have been downloaded successfully."
