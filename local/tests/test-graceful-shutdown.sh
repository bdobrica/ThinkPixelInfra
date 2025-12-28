#!/bin/bash
# Test script for graceful shutdown (Docker environment)

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Use docker compose (not docker-compose)
COMPOSE="docker compose -f ../docker-compose.yaml"
SERVICE_NAME="api-gateway"

echo -e "${BLUE}=== API Gateway Graceful Shutdown Test ===${NC}"
echo ""

# Get the API Gateway container name
CONTAINER_NAME=$($COMPOSE ps -q $SERVICE_NAME 2>/dev/null)

if [ -z "$CONTAINER_NAME" ]; then
    echo -e "${RED}❌ FAIL: API Gateway container not running${NC}"
    echo "Run 'make up' first to start the services"
    exit 1
fi

# Get the container PID
CONTAINER_ID=$($COMPOSE ps -q $SERVICE_NAME)
echo -e "Container ID: ${YELLOW}$CONTAINER_ID${NC}"
echo ""

# Test the ping endpoint
echo "Testing /ping endpoint..."
if curl -s http://localhost:8080/ping | grep -q "ok"; then
    echo -e "${GREEN}✅ Server is responding${NC}"
else
    echo -e "${RED}❌ Server not responding${NC}"
    exit 1
fi
echo ""

# Check logs before shutdown
echo -e "${BLUE}Last 5 lines before shutdown:${NC}"
$COMPOSE logs --tail=5 $SERVICE_NAME
echo ""

# Send graceful shutdown signal to the container
echo "Sending SIGTERM for graceful shutdown..."
$COMPOSE kill -s SIGTERM $SERVICE_NAME

# Wait for container to stop (max 35 seconds)
echo "Waiting for graceful shutdown to complete..."
TIMEOUT=35
STOPPED=0

for i in $(seq 1 $TIMEOUT); do
    if ! docker ps -q --filter "id=$CONTAINER_ID" | grep -q .; then
        echo -e "${GREEN}✅ Container stopped gracefully in ${i} seconds${NC}"
        STOPPED=1
        break
    fi
    sleep 1
done

echo ""

# Check logs after shutdown
echo -e "${BLUE}Last 20 lines after shutdown:${NC}"
$COMPOSE logs --tail=20 $SERVICE_NAME
echo ""

if [ $STOPPED -eq 1 ]; then
    echo -e "${GREEN}=== Test PASSED ===${NC}"
    exit 0
else
    echo -e "${RED}❌ FAIL: Container did not stop within $TIMEOUT seconds${NC}"
    echo "Force stopping container..."
    $COMPOSE kill $SERVICE_NAME
    exit 1
fi
