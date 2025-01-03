#!/bin/sh

# Define the base URL
BASE_URL="https://ublo.ro/wp-content/mirror"

# Define the files to download
FILES="\
mpnetv2/mpnet-base-v2/config.json \
mpnetv2/mpnet-base-v2/pytorch_model.bin \
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
    if [ -f "$DEST" ]; then
        echo "File already exists: $DEST"
        continue
    fi
    
    echo "Downloading $URL to $DEST..."
    curl -o "$DEST" "$URL"

    # Check if the download was successful
    if [ $? -ne 0 ]; then
        echo "Failed to download $URL"
        exit 1
    fi

done

echo "All files have been downloaded successfully."
