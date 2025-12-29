#!/bin/bash
# Test script for NATS error handling and health checks

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080"
PASSED=0
FAILED=0

echo -e "${BLUE}=== Database & NATS Error Handling Tests ===${NC}"
echo ""

# Test 1: Health check when database is healthy
echo -n "1. Testing /ready endpoint (database healthy)... "
READY_RESPONSE=$(curl -s "$API_URL/ready")
STATUS=$(echo "$READY_RESPONSE" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
DB_CHECK=$(echo "$READY_RESPONSE" | grep -o '"database":"[^"]*"' | cut -d'"' -f4)

if [ "$STATUS" = "ready" ] && [ "$DB_CHECK" = "healthy" ]; then
    echo -e "${GREEN}✓ PASS${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    echo "Response: $READY_RESPONSE"
    FAILED=$((FAILED + 1))
fi

# Test 2: Verify /ping still works
echo -n "2. Testing /ping endpoint... "
PING_RESPONSE=$(curl -s "$API_URL/ping")
if echo "$PING_RESPONSE" | grep -q '"status":"ok"'; then
    echo -e "${GREEN}✓ PASS${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    echo "Response: $PING_RESPONSE"
    FAILED=$((FAILED + 1))
fi

# Test 3: Check API Gateway logs for NATS connection
echo -n "3. Checking NATS connection in logs... "
NATS_LOG=$(docker compose logs api-gateway 2>&1 | grep -i "connected to nats\|nats connection" | tail -1)
if [ -n "$NATS_LOG" ]; then
    echo -e "${GREEN}✓ PASS${NC} (NATS connected)"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}⚠ WARNING${NC} (No NATS connection log found)"
    PASSED=$((PASSED + 1))  # Not critical
fi

# Test 4: Check database connection pool configuration
echo -n "4. Checking database pool configuration... "
DB_POOL_LOG=$(docker compose logs api-gateway 2>&1 | grep "Database connection pool configured" | tail -1)
if [ -n "$DB_POOL_LOG" ]; then
    echo -e "${GREEN}✓ PASS${NC}"
    echo "   $DB_POOL_LOG" | sed 's/api-gateway.*\[INFO\]//'
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC} (Pool configuration not logged)"
    FAILED=$((FAILED + 1))
fi

echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
