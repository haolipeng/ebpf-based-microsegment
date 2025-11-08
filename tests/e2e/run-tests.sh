#!/bin/bash

# E2E Integration Test Suite
# Tests the complete flow: Agent → Server → Database → API

set -e

SERVER_URL=${SERVER_URL:-http://server:8080}
POSTGRES_URL=${POSTGRES_URL:-postgres://microsegment_user:secret@postgres:5432/microsegment}

RESULTS_FILE=${RESULTS_FILE:-/test-results/results.txt}
mkdir -p $(dirname $RESULTS_FILE)

echo "==================================="
echo "E2E Integration Test Suite"
echo "==================================="
echo "Server: $SERVER_URL"
echo "Started: $(date)"
echo ""

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper functions
test_start() {
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "Test $TESTS_RUN: $1 ... "
}

test_pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo "✓ PASS"
}

test_fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo "✗ FAIL: $1"
}

# Wait for server to be ready
echo "Waiting for server to be ready..."
for i in {1..30}; do
    if curl -sf "$SERVER_URL/health" > /dev/null 2>&1; then
        echo "Server is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "ERROR: Server did not become ready"
        exit 1
    fi
    sleep 1
done

echo ""
echo "=== Running Tests ==="
echo ""

# Test 1: Server Health Check
test_start "Server health endpoint"
RESPONSE=$(curl -sf "$SERVER_URL/health" || echo "FAILED")
if echo "$RESPONSE" | grep -q "ok\|healthy\|UP"; then
    test_pass
else
    test_fail "Health check failed"
fi

# Test 2: Database Connection
test_start "Database connectivity"
if psql "$POSTGRES_URL" -c "SELECT 1" > /dev/null 2>&1; then
    test_pass
else
    test_fail "Cannot connect to database"
fi

# Test 3: Flows Table Exists
test_start "Flows table schema"
TABLE_EXISTS=$(psql "$POSTGRES_URL" -t -c "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'flows')")
if echo "$TABLE_EXISTS" | grep -q "t"; then
    test_pass
else
    test_fail "Flows table does not exist"
fi

# Test 4: Wait for Agent to send flows
echo ""
echo "Waiting for Agent to send flows (15 seconds)..."
sleep 15

# Test 5: Check if flows exist in database
test_start "Flows in database"
FLOW_COUNT=$(psql "$POSTGRES_URL" -t -c "SELECT COUNT(*) FROM flows" | tr -d ' ')
if [ "$FLOW_COUNT" -gt 0 ]; then
    test_pass
    echo "   Found $FLOW_COUNT flows in database"
else
    test_fail "No flows found in database"
fi

# Test 6: Flow API - List Flows
test_start "GET /api/v1/flows"
RESPONSE=$(curl -sf "$SERVER_URL/api/v1/flows?limit=10" || echo "FAILED")
if echo "$RESPONSE" | grep -q "flows\|data"; then
    test_pass
    FLOW_COUNT=$(echo "$RESPONSE" | jq -r '.flows | length' 2>/dev/null || echo "0")
    echo "   API returned $FLOW_COUNT flows"
else
    test_fail "API request failed"
fi

# Test 7: Flow API - Summary
test_start "GET /api/v1/flows/summary"
RESPONSE=$(curl -sf "$SERVER_URL/api/v1/flows/summary" || echo "FAILED")
if echo "$RESPONSE" | grep -q "total_flows\|total_bytes"; then
    test_pass
else
    test_fail "Summary API failed"
fi

# Test 8: Aggregator API - Dependencies
test_start "GET /api/v1/aggregator/dependencies"
RESPONSE=$(curl -sf "$SERVER_URL/api/v1/aggregator/dependencies?group_by=app" || echo "FAILED")
if echo "$RESPONSE" | grep -q "dependencies\|group_by"; then
    test_pass
else
    test_fail "Dependencies API failed"
fi

# Test 9: Aggregator API - Top Talkers
test_start "GET /api/v1/aggregator/top-talkers"
RESPONSE=$(curl -sf "$SERVER_URL/api/v1/aggregator/top-talkers?top_n=5" || echo "FAILED")
if echo "$RESPONSE" | grep -q "top_talkers\|by_bytes"; then
    test_pass
else
    test_fail "Top talkers API failed"
fi

# Test 10: Verify Label Enrichment
test_start "Label enrichment in database"
LABELED_COUNT=$(psql "$POSTGRES_URL" -t -c "SELECT COUNT(*) FROM flows WHERE source_labels IS NOT NULL AND source_labels::text != '{}'" | tr -d ' ')
if [ "$LABELED_COUNT" -gt 0 ]; then
    test_pass
    echo "   Found $LABELED_COUNT flows with labels"
else
    test_fail "No flows with labels found"
fi

# Test 11: WebSocket Stream Stats
test_start "GET /api/v1/flows/stream/stats"
RESPONSE=$(curl -sf "$SERVER_URL/api/v1/flows/stream/stats" || echo "FAILED")
if echo "$RESPONSE" | grep -q "connected_clients\|total_messages"; then
    test_pass
else
    test_fail "WebSocket stats API failed"
fi

# Test 12: Data Integrity Check
test_start "Flow data integrity"
INVALID_COUNT=$(psql "$POSTGRES_URL" -t -c "SELECT COUNT(*) FROM flows WHERE src_ip IS NULL OR dst_ip IS NULL" | tr -d ' ')
if [ "$INVALID_COUNT" -eq 0 ]; then
    test_pass
else
    test_fail "Found $INVALID_COUNT flows with invalid data"
fi

# Summary
echo ""
echo "==================================="
echo "Test Summary"
echo "==================================="
echo "Total Tests: $TESTS_RUN"
echo "Passed:      $TESTS_PASSED ✓"
echo "Failed:      $TESTS_FAILED ✗"
echo "Completed:   $(date)"
echo "==================================="

# Write results to file
cat > "$RESULTS_FILE" << EOF
E2E Integration Test Results
============================
Date: $(date)
Server: $SERVER_URL

Tests Run: $TESTS_RUN
Tests Passed: $TESTS_PASSED
Tests Failed: $TESTS_FAILED

Status: $([ $TESTS_FAILED -eq 0 ] && echo "SUCCESS" || echo "FAILURE")

Database Statistics:
- Total Flows: $FLOW_COUNT
- Flows with Labels: $LABELED_COUNT
EOF

# Exit with appropriate code
if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    echo "✓ All tests passed!"
    exit 0
else
    echo ""
    echo "✗ Some tests failed"
    exit 1
fi
