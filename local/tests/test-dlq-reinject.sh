#!/bin/bash
# DLQ Reinjection Test
# Tests the dead letter queue and message reinjection functionality

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080"
TIMESTAMP=$(date +%s)
TEST_DOMAIN="dlq-test-${TIMESTAMP}.com"
TEST_API_KEY="dlq-test-key-${TIMESTAMP}"
TEST_API_KEY_HASH=$(echo -n "$TEST_API_KEY" | sha256sum | awk '{print $1}')

echo -e "${BLUE}=== DLQ Reinjection Test ===${NC}"
echo ""
echo -e "${CYAN}Test Domain: ${TEST_DOMAIN}${NC}"
echo -e "${CYAN}API Key: ${TEST_API_KEY}${NC}"
echo ""

# Counter for tests
PASSED=0
FAILED=0
SITE_ID=""
JWT_TOKEN=""

# Test function
test_step() {
    local name=$1
    echo -e "\n${CYAN}>>> $name${NC}"
}

pass_test() {
    local msg=$1
    echo -e "${GREEN}✓ $msg${NC}"
    PASSED=$((PASSED + 1))
}

fail_test() {
    local msg=$1
    echo -e "${RED}✗ $msg${NC}"
    FAILED=$((FAILED + 1))
}

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}=== Cleaning Up ===${NC}"

    if [ -n "$SITE_ID" ]; then
        echo "Removing DLQ messages for site ${SITE_ID}..."
        docker exec redis-primary redis-cli DEL "dlq:site:${SITE_ID}" 2>/dev/null || true
        docker exec redis-primary redis-cli DEL "dlq:counter:site:${SITE_ID}" 2>/dev/null || true
        docker exec redis-primary redis-cli ZREM "dlq:stats" "${SITE_ID}" 2>/dev/null || true
    fi

    echo "Removing test site from database..."
    docker exec mysql mysql -uroot -proot thinkpixel -e "DELETE FROM wp_thinkpixel_sites WHERE domain = '${TEST_DOMAIN}';" 2>/dev/null || true

    echo "Cleanup complete"
}

trap cleanup EXIT

# ============================================================
# Step 0: Clear old DLQ messages from Redis
# ============================================================
test_step "Step 0: Clearing old DLQ messages from Redis"

# Clear all DLQ keys from previous test runs
CLEARED_KEYS=$(docker exec redis-primary redis-cli --scan --pattern "dlq:*" | wc -l)
docker exec redis-primary redis-cli --scan --pattern "dlq:*" | xargs -r docker exec -i redis-primary redis-cli DEL > /dev/null 2>&1 || true

if [ "$CLEARED_KEYS" -gt 0 ]; then
    echo "Cleared $CLEARED_KEYS old DLQ keys from Redis"
else
    echo "No old DLQ keys found"
fi

pass_test "Redis DLQ cleaned"

# ============================================================
# Step 1: Setup test site
# ============================================================
test_step "Step 1: Creating test site in database"

# Insert test site with Redis backend and intentionally broken IndexingNode to trigger DLQ
INSERT_OUTPUT=$(docker exec mysql mysql -uroot -proot thinkpixel -e "
INSERT INTO wp_thinkpixel_sites (
    api_key, protocol, domain, path, status,
    jwt_ttl, model, chunk_size, chunk_overlap,
    indexing_node_type, indexing_node, max_search_results,
    validation_status, verified_at, expires_at
) VALUES (
    '${TEST_API_KEY_HASH}',
    'http',
    '${TEST_DOMAIN}',
    '/',
    'active',
    900,
    'mpnetv2',
    512,
    128,
    'redis',
    'invalid-host:26379/mymaster',
    20,
    'verified',
    NOW(),
    DATE_ADD(NOW(), INTERVAL 30 DAY)
);" 2>&1)

INSERT_EXIT=$?

if [ $INSERT_EXIT -eq 0 ]; then
    pass_test "Test site created in database"
else
    fail_test "Failed to create test site"
    echo "MySQL output: $INSERT_OUTPUT"
    exit 1
fi

# Give MySQL a moment to commit
sleep 1

# Get the site ID
SITE_ID=$(docker exec mysql mysql -uroot -proot thinkpixel -sN -e "SELECT id FROM wp_thinkpixel_sites WHERE domain = '${TEST_DOMAIN}';" 2>/dev/null)

if [ -z "$SITE_ID" ]; then
    fail_test "Failed to retrieve site ID"
    echo "Checking if site exists in database..."
    docker exec mysql mysql -uroot -proot thinkpixel -e "SELECT id, domain, api_key, status FROM wp_thinkpixel_sites WHERE domain LIKE '%dlq-test%';"
    exit 1
fi

echo "Site ID: ${SITE_ID}"

# ============================================================
# Step 2: Authenticate and get JWT token
# ============================================================
test_step "Step 2: Authenticating to get JWT token"

AUTH_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/token" \
    -H "X-API-Key: ${TEST_API_KEY}" \
    2>/dev/null)

AUTH_STATUS=$(echo "$AUTH_RESPONSE" | tail -n1)
AUTH_BODY=$(echo "$AUTH_RESPONSE" | sed '$d')

if [ "$AUTH_STATUS" = "200" ]; then
    JWT_TOKEN=$(echo "$AUTH_BODY" | jq -r '.token')
    if [ -n "$JWT_TOKEN" ] && [ "$JWT_TOKEN" != "null" ]; then
        pass_test "Authentication successful, JWT token obtained"
    else
        fail_test "Authentication succeeded but no token in response"
        exit 1
    fi
else
    fail_test "Authentication failed (HTTP $AUTH_STATUS)"
    echo "Response: $AUTH_BODY"
    exit 1
fi

# ============================================================
# Step 3: Send documents that will fail and go to DLQ
# ============================================================
test_step "Step 3: Sending documents that will fail (broken Redis connection)"

# Send multiple documents to ensure they go to DLQ
STORE_DATA='[
    {
        "url": "http://'${TEST_DOMAIN}'/doc1",
        "title": "DLQ Test Document 1",
        "text": "This is test document 1 that should fail and go to DLQ because the Redis connection is broken.",
        "meta": {"test": "dlq-1"}
    },
    {
        "url": "http://'${TEST_DOMAIN}'/doc2",
        "title": "DLQ Test Document 2",
        "text": "This is test document 2 that should fail and go to DLQ because the Redis connection is broken.",
        "meta": {"test": "dlq-2"}
    },
    {
        "url": "http://'${TEST_DOMAIN}'/doc3",
        "title": "DLQ Test Document 3",
        "text": "This is test document 3 that should fail and go to DLQ because the Redis connection is broken.",
        "meta": {"test": "dlq-3"}
    }
]'

STORE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/store" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${JWT_TOKEN}" \
    -d "${STORE_DATA}" \
    2>/dev/null)

STORE_STATUS=$(echo "$STORE_RESPONSE" | tail -n1)

if [ "$STORE_STATUS" = "202" ]; then
    pass_test "Documents queued (will fail due to broken connection)"
else
    STORE_BODY=$(echo "$STORE_RESPONSE" | sed '$d')
    fail_test "Failed to queue documents (HTTP $STORE_STATUS)"
    echo "Response: $STORE_BODY"
fi

# ============================================================
# Step 4: Wait for messages to fail and go to DLQ
# ============================================================
test_step "Step 4: Waiting for messages to fail and appear in DLQ"

echo "Waiting 60 seconds for retry exhaustion and DLQ storage..."
sleep 60

# Check DLQ stats via API
DLQ_STATS=$(curl -s -X GET "$API_URL/dlq/stats" \
    -H "Authorization: Bearer ${JWT_TOKEN}" \
    2>/dev/null)

echo "DLQ Stats: $DLQ_STATS"

# Check DLQ messages for our site
DLQ_MESSAGES=$(curl -s -X GET "$API_URL/dlq/messages?site_id=${SITE_ID}" \
    -H "Authorization: Bearer ${JWT_TOKEN}" \
    2>/dev/null)

MESSAGE_COUNT=$(echo "$DLQ_MESSAGES" | jq '. | length' 2>/dev/null || echo "0")

if [ "$MESSAGE_COUNT" -gt 0 ]; then
    pass_test "Found $MESSAGE_COUNT messages in DLQ"
    echo "DLQ Messages: $DLQ_MESSAGES" | jq '.' 2>/dev/null || echo "$DLQ_MESSAGES"
else
    fail_test "No messages found in DLQ (expected at least 1)"
    echo "Response: $DLQ_MESSAGES"
fi

# ============================================================
# Step 5: Fix the IndexingNode to enable successful reinjection
# ============================================================
test_step "Step 5: Fixing IndexingNode to enable successful processing"

# Update the site to use correct Redis Sentinel connection
docker exec mysql mysql -uroot -proot thinkpixel -e "
UPDATE wp_thinkpixel_sites
SET indexing_node = 'redis-sentinel:26379/mymaster'
WHERE id = ${SITE_ID};
"

if [ $? -eq 0 ]; then
    # Restart api-gateway to clear the in-memory site cache
    echo "Restarting api-gateway to clear cache..."
    docker restart api-gateway > /dev/null 2>&1

    # Wait for api-gateway to be healthy
    echo "Waiting for api-gateway to be ready..."
    MAX_RETRIES=30
    RETRY_COUNT=0
    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if curl -sf http://localhost:8080/ping > /dev/null 2>&1; then
            echo "API Gateway is healthy"
            break
        fi
        RETRY_COUNT=$((RETRY_COUNT + 1))
        sleep 1
    done

    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        fail_test "API Gateway did not become healthy after restart"
    else
        # Wait extra time for NATS subscriber to initialize
        echo "Waiting for NATS subscriber to initialize..."
        sleep 10
        pass_test "IndexingNode updated and cache cleared"
    fi
else
    fail_test "Failed to update IndexingNode"
fi

# ============================================================
# Step 6: Test dry-run reinjection
# ============================================================
test_step "Step 6: Testing dry-run reinjection"

DRYRUN_OUTPUT=$(docker exec api-gateway /app/dlq-reinject \
    --dry-run \
    --site-id ${SITE_ID} \
    -v \
    2>&1)

echo "$DRYRUN_OUTPUT"

if echo "$DRYRUN_OUTPUT" | grep -q "DRY RUN MODE"; then
    pass_test "Dry-run mode executed successfully"
else
    fail_test "Dry-run mode failed"
fi

# ============================================================
# Step 7: Perform actual reinjection
# ============================================================
test_step "Step 7: Performing actual message reinjection"

REINJECT_OUTPUT=$(docker exec api-gateway /app/dlq-reinject \
    --site-id ${SITE_ID} \
    --delete \
    -v \
    2>&1)

echo "$REINJECT_OUTPUT"

if echo "$REINJECT_OUTPUT" | grep -q "Reinjection complete"; then
    REINJECTED=$(echo "$REINJECT_OUTPUT" | grep "Reinjection complete" | sed -n 's/.*\([0-9]\+\) succeeded.*/\1/p')
    if [ "$REINJECTED" -gt 0 ]; then
        pass_test "Successfully reinjected $REINJECTED messages"
    else
        fail_test "Reinjection completed but 0 messages reinjected"
    fi
else
    fail_test "Reinjection failed"
fi

# ============================================================
# Step 8: Wait and verify messages were processed
# ============================================================
test_step "Step 8: Waiting for reinjected messages to be processed"

echo "Waiting 20 seconds for message processing..."
sleep 20

# Check if DLQ is now empty
DLQ_MESSAGES_AFTER=$(curl -s -X GET "$API_URL/dlq/messages?site_id=${SITE_ID}" \
    -H "Authorization: Bearer ${JWT_TOKEN}" \
    2>/dev/null)

MESSAGE_COUNT_AFTER=$(echo "$DLQ_MESSAGES_AFTER" | jq '.count // 0' 2>/dev/null || echo "0")

if [ "$MESSAGE_COUNT_AFTER" = "0" ]; then
    pass_test "DLQ is now empty (messages were successfully processed)"
else
    fail_test "DLQ still contains $MESSAGE_COUNT_AFTER messages"
    echo "Remaining messages: $DLQ_MESSAGES_AFTER" | jq '.' 2>/dev/null || echo "$DLQ_MESSAGES_AFTER"
fi

# ============================================================
# Step 9: Verify embeddings were stored
# ============================================================
test_step "Step 9: Verifying embeddings were stored in Qdrant"

# First check if collection exists and has points
COLLECTION_NAME="site_${SITE_ID}"
COLLECTION_INFO=$(curl -s -X GET "http://localhost:6333/collections/${COLLECTION_NAME}" 2>/dev/null)
POINTS_COUNT=$(echo "$COLLECTION_INFO" | jq '.result.points_count // 0' 2>/dev/null || echo "0")

if [ "$POINTS_COUNT" -gt 0 ]; then
    echo "Qdrant collection has $POINTS_COUNT points"
else
    echo "Warning: Qdrant collection has 0 points. Checking api-gateway logs for errors..."
    # Check last 50 lines of logs for store-related errors
    STORE_ERRORS=$(docker logs --tail 50 api-gateway 2>&1 | grep -i "store\|error\|failed" | tail -10)
    if [ -n "$STORE_ERRORS" ]; then
        echo "Recent errors from api-gateway:"
        echo "$STORE_ERRORS"
    fi
fi

# Try to search for the reinjected documents
SEARCH_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/search" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${JWT_TOKEN}" \
    -d '{
        "text": "test document DLQ"
    }' \
    2>/dev/null)

SEARCH_STATUS=$(echo "$SEARCH_RESPONSE" | tail -n1)
SEARCH_BODY=$(echo "$SEARCH_RESPONSE" | sed '$d')

if [ "$SEARCH_STATUS" = "200" ]; then
    RESULT_COUNT=$(echo "$SEARCH_BODY" | jq '.results | length' 2>/dev/null || echo "0")
    if [ "$RESULT_COUNT" -gt 0 ]; then
        pass_test "Found $RESULT_COUNT results in search (embeddings successfully stored)"
        echo "Search results: $SEARCH_BODY" | jq '.' 2>/dev/null || echo "$SEARCH_BODY"
    else
        fail_test "Search succeeded but returned 0 results (Qdrant has $POINTS_COUNT points)"
    fi
else
    fail_test "Search failed (HTTP $SEARCH_STATUS, Qdrant has $POINTS_COUNT points)"
    echo "Response: $SEARCH_BODY"
fi

# ============================================================
# Summary
# ============================================================
echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
echo -e "Passed: ${GREEN}${PASSED}${NC}"
echo -e "Failed: ${RED}${FAILED}${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed ✗${NC}"
    exit 1
fi
