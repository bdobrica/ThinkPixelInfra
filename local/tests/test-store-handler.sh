#!/bin/bash
# Test script for Store Handler with soft-fail behavior

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080"
PASSED=0
FAILED=0

# Use mock-site service name for Docker network resolution
TEST_DOMAIN="mock-site"

echo -e "${BLUE}=== Store Handler Tests ===${NC}"
echo ""

# Cleanup any previous test data
docker exec mysql mysql -uroot -proot -D thinkpixel -e "DELETE FROM wp_thinkpixel_sites WHERE domain='$TEST_DOMAIN'" 2>/dev/null || true
docker exec mock-site rm -f /tmp/api_key.txt 2>/dev/null || true

# Test 1: Register a site first to get API key
echo -n "1. Registering test site ($TEST_DOMAIN)... "
REGISTER_RESPONSE=$(curl -s -X POST "$API_URL/register" \
    -H "Content-Type: application/json" \
    -d '{
        "domain": "'$TEST_DOMAIN'",
        "path": "/",
        "estimated_pages": 10,
        "average_page_size": 1000,
        "st_dev_page_size": 100
    }')

VALIDATION_TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"validation_token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$VALIDATION_TOKEN" ]; then
    echo -e "${GREEN}✓ PASS${NC} (Validation Token: ${VALIDATION_TOKEN:0:20}...)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC} - Could not get validation token"
    echo "Response: $REGISTER_RESPONSE"
    FAILED=$((FAILED + 1))
    exit 1
fi

# Wait for validation to complete (insecure mode should be fast)
echo -n "2. Waiting for validation to complete... "
sleep 5

# Query mock-site for the API key that was sent to it
API_KEY=$(docker exec mock-site cat /tmp/api_key.txt 2>/dev/null || true)

if [ -n "$API_KEY" ]; then
    echo -e "${GREEN}✓ PASS${NC} (API Key obtained)"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}⚠ WARNING${NC} - Validation may still be in progress"
    # Try waiting a bit more
    sleep 5
    API_KEY=$(docker exec mock-site cat /tmp/api_key.txt 2>/dev/null || true)

    if [ -n "$API_KEY" ]; then
        echo -e "  ${GREEN}✓ PASS${NC} (API Key obtained after retry)"
        PASSED=$((PASSED + 1))
    else
        echo -e "  ${RED}✗ FAIL${NC} - Validation failed or timed out"
        # Check validation status
        STATUS=$(docker exec mysql mysql -uroot -proot -D thinkpixel -sN -e \
            "SELECT validation_status FROM wp_thinkpixel_sites WHERE domain='$TEST_DOMAIN' AND path='/' LIMIT 1" 2>/dev/null || echo "not_found")
        echo "  Validation status: $STATUS"
        FAILED=$((FAILED + 1))
        exit 1
    fi
fi

# Test 3: Get JWT token
echo -n "3. Getting JWT token... "
TOKEN_RESPONSE=$(curl -s -X POST "$API_URL/auth/token" \
    -H "X-API-Key: $API_KEY")

JWT_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$JWT_TOKEN" ]; then
    echo -e "${GREEN}✓ PASS${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    echo "Response: $TOKEN_RESPONSE"
    FAILED=$((FAILED + 1))
    exit 1
fi

# Test 4: Store valid documents (should all succeed)
echo -n "4. Storing valid documents... "
STORE_RESPONSE=$(curl -s -X POST "$API_URL/store" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d '[
        {"id": 1, "text": "Test document 1", "extra": {"url": "http://test.local/page1"}},
        {"id": 2, "text": "Test document 2", "extra": {"url": "http://test.local/page2"}},
        {"id": 3, "text": "Test document 3", "extra": {"url": "http://test.local/page3"}}
    ]')

PENDING=$(echo "$STORE_RESPONSE" | grep -o '"pending_documents":[0-9]*' | cut -d':' -f2)
FAILED_DOCS=$(echo "$STORE_RESPONSE" | grep -o '"failed_documents":[0-9]*' | cut -d':' -f2)

if [ "$PENDING" = "3" ] && [ "$FAILED_DOCS" = "0" ]; then
    echo -e "${GREEN}✓ PASS${NC} (3 queued, 0 failed)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC} (Expected 3 queued/0 failed, got $PENDING queued/$FAILED_DOCS failed)"
    echo "Response: $STORE_RESPONSE"
    FAILED=$((FAILED + 1))
fi

# Test 5: Verify response structure includes new fields
echo -n "5. Verifying response structure... "
HAS_FAILED_DOCS=$(echo "$STORE_RESPONSE" | grep -c '"failed_documents"' || true)
HAS_FAILED_ITEMS=$(echo "$STORE_RESPONSE" | grep -c '"failed_items"' || true)

if [ "$HAS_FAILED_DOCS" -ge "1" ]; then
    echo -e "${GREEN}✓ PASS${NC} (Response includes failed_documents field)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC} (Response missing failed_documents field)"
    echo "Response: $STORE_RESPONSE"
    FAILED=$((FAILED + 1))
fi

# Test 6: Store empty array (should fail with 400)
echo -n "6. Storing empty array... "
EMPTY_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/store" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d '[]')

HTTP_CODE=$(echo "$EMPTY_RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "400" ]; then
    echo -e "${GREEN}✓ PASS${NC} (HTTP 400)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC} (Expected 400, got $HTTP_CODE)"
    FAILED=$((FAILED + 1))
fi

# Test 7: Store without authentication (should fail with 401)
echo -n "7. Storing without authentication... "
UNAUTH_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/store" \
    -H "Content-Type: application/json" \
    -d '[{"id": 1, "text": "Test"}]')

HTTP_CODE=$(echo "$UNAUTH_RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✓ PASS${NC} (HTTP 401)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC} (Expected 401, got $HTTP_CODE)"
    FAILED=$((FAILED + 1))
fi

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
