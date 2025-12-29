#!/bin/bash
# Build script for DLQ reinjection tool

set -e

echo "Building DLQ reinjection tool..."

# Build for current platform
go build -o dlq-reinject cmd/dlq-reinject/main.go

echo "✅ Built: ./dlq-reinject"

# Optionally build for multiple platforms
if [ "$1" == "all" ]; then
    echo "Building for multiple platforms..."

    # Linux AMD64
    GOOS=linux GOARCH=amd64 go build -o dlq-reinject-linux-amd64 cmd/dlq-reinject/main.go
    echo "✅ Built: ./dlq-reinject-linux-amd64"

    # Linux ARM64
    GOOS=linux GOARCH=arm64 go build -o dlq-reinject-linux-arm64 cmd/dlq-reinject/main.go
    echo "✅ Built: ./dlq-reinject-linux-arm64"

    # macOS AMD64
    GOOS=darwin GOARCH=amd64 go build -o dlq-reinject-darwin-amd64 cmd/dlq-reinject/main.go
    echo "✅ Built: ./dlq-reinject-darwin-amd64"

    # macOS ARM64 (M1/M2)
    GOOS=darwin GOARCH=arm64 go build -o dlq-reinject-darwin-arm64 cmd/dlq-reinject/main.go
    echo "✅ Built: ./dlq-reinject-darwin-arm64"
fi

echo ""
echo "Usage:"
echo "  ./dlq-reinject --help"
echo "  ./dlq-reinject --source redis --redis localhost:6379 --nats nats://localhost:4222 --dry-run"
