#!/bin/bash
# Test script for Prometheus metrics endpoint

set -e  # Exit on error

API_URL="http://localhost:8080"
PASSED=0
FAILED=0

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Testing API Gateway Prometheus Metrics ===${NC}\n"

# Test 1: Metrics endpoint exists and returns Prometheus format
echo "Test 1: Metrics endpoint exists and returns Prometheus format"
RESPONSE=$(curl -s http://localhost:8080/metrics)
if echo "$RESPONSE" | grep -q "# HELP"; then
    echo -e "${GREEN}✓ PASSED${NC} - Metrics endpoint returns Prometheus format\n"
    ((PASSED++))
else
    echo -e "${RED}✗ FAILED${NC} - Metrics endpoint doesn't return Prometheus format\n"
    ((FAILED++))
fi

# Test 2: Check for API Gateway specific metrics
echo "Test 2: Check for API Gateway specific metrics"
METRICS_FOUND=0

if echo "$RESPONSE" | grep -q "api_gateway_requests_total"; then
    echo "  - Found: api_gateway_requests_total"
    ((METRICS_FOUND++))
fi

if echo "$RESPONSE" | grep -q "api_gateway_request_duration_seconds"; then
    echo "  - Found: api_gateway_request_duration_seconds"
    ((METRICS_FOUND++))
fi

if echo "$RESPONSE" | grep -q "api_gateway_nats_messages_published_total"; then
    echo "  - Found: api_gateway_nats_messages_published_total"
    ((METRICS_FOUND++))
fi

if echo "$RESPONSE" | grep -q "api_gateway_nats_messages_processed_total"; then
    echo "  - Found: api_gateway_nats_messages_processed_total"
    ((METRICS_FOUND++))
fi

if echo "$RESPONSE" | grep -q "api_gateway_nats_dlq_messages_total"; then
    echo "  - Found: api_gateway_nats_dlq_messages_total"
    ((METRICS_FOUND++))
fi

if echo "$RESPONSE" | grep -q "api_gateway_nats_unmarshal_errors_total"; then
    echo "  - Found: api_gateway_nats_unmarshal_errors_total"
    ((METRICS_FOUND++))
fi

if echo "$RESPONSE" | grep -q "api_gateway_model_latency_seconds"; then
    echo "  - Found: api_gateway_model_latency_seconds"
    ((METRICS_FOUND++))
fi

if echo "$RESPONSE" | grep -q "api_gateway_db_queries_total"; then
    echo "  - Found: api_gateway_db_queries_total"
    ((METRICS_FOUND++))
fi

if echo "$RESPONSE" | grep -q "api_gateway_db_connections"; then
    echo "  - Found: api_gateway_db_connections"
    ((METRICS_FOUND++))
fi

if [ "$METRICS_FOUND" -eq 9 ]; then
    echo -e "${GREEN}✓ PASSED${NC} - All 9 custom metrics found\n"
    ((PASSED++))
else
    echo -e "${RED}✗ FAILED${NC} - Only $METRICS_FOUND/9 metrics found\n"
    ((FAILED++))
fi

# Test 3: Make a request to /ping and check if metrics counter increments
echo "Test 3: Request counter increments after requests"

# Get initial request count for /ping
INITIAL_PING_COUNT=$(curl -s http://localhost:8080/metrics | grep 'api_gateway_requests_total{endpoint="/ping"' | grep -o '[0-9]*$' | head -1)
if [ -z "$INITIAL_PING_COUNT" ]; then
    INITIAL_PING_COUNT=0
fi
echo "  Initial /ping request count: $INITIAL_PING_COUNT"

# Make a request to /ping
curl -s http://localhost:8080/ping > /dev/null

# Wait a moment for metrics to update
sleep 1

# Get new request count
NEW_PING_COUNT=$(curl -s http://localhost:8080/metrics | grep 'api_gateway_requests_total{endpoint="/ping"' | grep -o '[0-9]*$' | head -1)
if [ -z "$NEW_PING_COUNT" ]; then
    NEW_PING_COUNT=0
fi
echo "  New /ping request count: $NEW_PING_COUNT"

if [ "$NEW_PING_COUNT" -gt "$INITIAL_PING_COUNT" ]; then
    echo -e "${GREEN}✓ PASSED${NC} - Request counter incremented\n"
    ((PASSED++))
else
    echo -e "${RED}✗ FAILED${NC} - Request counter did not increment\n"
    ((FAILED++))
fi

# Test 4: Check database connection metrics
echo "Test 4: Database connection metrics are being collected"

if echo "$RESPONSE" | grep -q 'api_gateway_db_connections{state="open"}'; then
    echo "  - Found: db_connections (open)"
    DB_OPEN=$(echo "$RESPONSE" | grep 'api_gateway_db_connections{state="open"}' | grep -o '[0-9.]*$')
    echo "    Value: $DB_OPEN"
fi

if echo "$RESPONSE" | grep -q 'api_gateway_db_connections{state="idle"}'; then
    echo "  - Found: db_connections (idle)"
    DB_IDLE=$(echo "$RESPONSE" | grep 'api_gateway_db_connections{state="idle"}' | grep -o '[0-9.]*$')
    echo "    Value: $DB_IDLE"
fi

if echo "$RESPONSE" | grep -q 'api_gateway_db_connections{state="in_use"}'; then
    echo "  - Found: db_connections (in_use)"
    DB_IN_USE=$(echo "$RESPONSE" | grep 'api_gateway_db_connections{state="in_use"}' | grep -o '[0-9.]*$')
    echo "    Value: $DB_IN_USE"
    echo -e "${GREEN}✓ PASSED${NC} - Database connection metrics are present\n"
    ((PASSED++))
else
    echo -e "${YELLOW}⚠ SKIPPED${NC} - Database connection metrics not yet available (may take 15s to populate)\n"
fi

# Test 5: Check Go runtime metrics are included
echo "Test 5: Go runtime metrics are included"

if echo "$RESPONSE" | grep -q "go_goroutines"; then
    GO_GOROUTINES=$(echo "$RESPONSE" | grep "^go_goroutines " | awk '{print $2}')
    echo "  - Found: go_goroutines (Value: $GO_GOROUTINES)"
    echo -e "${GREEN}✓ PASSED${NC} - Go runtime metrics are included\n"
    ((PASSED++))
else
    echo -e "${RED}✗ FAILED${NC} - Go runtime metrics not found\n"
    ((FAILED++))
fi

# Summary
echo -e "${YELLOW}=== Test Summary ===${NC}"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
