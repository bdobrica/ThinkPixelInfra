#!/bin/bash
# API Gateway Integration Tests

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080"
# Use timestamp to make domain unique and avoid DB conflicts
TIMESTAMP=$(date +%s)
TEST_DOMAIN="example-${TIMESTAMP}.com"
TEST_PATH="/"

echo -e "${BLUE}=== API Gateway Integration Tests ===${NC}"
echo ""

# Counter for tests
PASSED=0
FAILED=0

# Test function
test_endpoint() {
    local name=$1
    local method=$2
    local endpoint=$3
    local data=$4
    local expected_status=$5
    local headers=$6

    echo -n "Testing $name... "

    if [ -z "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X $method "$API_URL$endpoint" $headers 2>/dev/null || echo "000")
    else
        response=$(curl -s -w "\n%{http_code}" -X $method "$API_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data" \
            $headers 2>/dev/null || echo "000")
    fi

    status=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)

    if [ "$status" = "$expected_status" ]; then
        echo -e "${GREEN}✓ PASS${NC} (HTTP $status)"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC} (Expected $expected_status, got $status)"
        echo "Response: $body"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

# Test 1: Ping endpoint
test_endpoint "Ping" "GET" "/ping" "" "200"

# Test 2: Register endpoint (should fail without proper data)
test_endpoint "Register without data" "POST" "/register" "" "400"

# Test 3: Register with valid data
REGISTER_DATA='{
    "domain": "'$TEST_DOMAIN'",
    "path": "'$TEST_PATH'",
    "estimated_pages": 100,
    "average_page_size": 5000,
    "st_dev_page_size": 1000
}'
test_endpoint "Register site" "POST" "/register" "$REGISTER_DATA" "200"

# Test 4: Auth without API key
test_endpoint "Auth without API key" "POST" "/auth/token" "" "400"

# Test 5: Auth with invalid API key
test_endpoint "Auth with invalid key" "POST" "/auth/token" "" "401" '-H "X-API-Key: invalid-key"'

# Test 6: Protected endpoint without auth
test_endpoint "Search without auth" "POST" "/search" '{"text":"test"}' "401"

# Test 7: Store endpoint without auth
test_endpoint "Store without auth" "POST" "/store" '[]' "401"

# Test 8: Max batch text size without auth
test_endpoint "Max batch without auth" "POST" "/store/max_batch_text_size" "" "401"

echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
