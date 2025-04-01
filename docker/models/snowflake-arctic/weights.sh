#!/bin/sh

# Define the base URL
BASE_URL="https://ublo.ro/wp-content/mirror"

# Define the files to download
FILES="\
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.aa \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.ab \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.ac \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.ad \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.ae \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.af \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.ag \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.ah \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.ai \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.aj \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.ak \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/model.onnx_data.sha256sum \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/special_tokens_map.json \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/tokenizer.json \
snowflake-arctic/snowflake-arctic-embed-l-v2.0/tokenizer_config.json"

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

# Join the files
echo "Joining files..."
(
    cd "$WEIGHTS_DIR/snowflake-arctic/snowflake-arctic-embed-l-v2.0" || exit 1
    cat model.onnx_data.a* > model.onnx_data
    rm model.onnx_data.a*
    sha256sum -c model.onnx_data.sha256sum
) || (
    echo "Failed to verify the joined file"
    exit 1
)

echo "All files have been downloaded successfully."
