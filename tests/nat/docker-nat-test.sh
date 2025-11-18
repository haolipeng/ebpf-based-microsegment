#!/bin/bash
# Docker NAT Testing Script
# Tests microsegmentation NAT support in Docker bridge network environment
#
# This script:
# 1. Creates a Docker container with a bridge network (SNAT scenario)
# 2. Configures microsegmentation policy using original container IP
# 3. Tests connectivity to verify NAT address restoration
# 4. Validates NAT statistics and policy matching

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CONTAINER_NAME="nat-test-container"
TEST_IMAGE="nginx:alpine"
AGENT_API="http://localhost:8080"

echo "========================================="
echo "  Docker NAT Testing for Microsegmentation"
echo "========================================="
echo ""

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Cleanup function
cleanup() {
    print_info "Cleaning up test environment..."
    docker rm -f $CONTAINER_NAME 2>/dev/null || true
    print_success "Cleanup complete"
}

# Register cleanup on exit
trap cleanup EXIT

# Check prerequisites
print_info "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    exit 1
fi

if ! command -v curl &> /dev/null; then
    print_error "curl is not installed"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    print_error "jq is not installed (required for JSON parsing)"
    exit 1
fi

print_success "Prerequisites OK"

# Check if agent is running
print_info "Checking if microsegmentation agent is running..."
if ! curl -s -f "$AGENT_API/health" > /dev/null 2>&1; then
    print_error "Microsegmentation agent is not running at $AGENT_API"
    print_info "Please start the agent first: sudo ./bin/microsegment-agent"
    exit 1
fi
print_success "Agent is running"

# Step 1: Start test container
print_info "Starting Docker test container..."
docker run -d --name $CONTAINER_NAME $TEST_IMAGE
sleep 2

# Get container IP
CONTAINER_IP=$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $CONTAINER_NAME)
if [ -z "$CONTAINER_IP" ]; then
    print_error "Failed to get container IP address"
    exit 1
fi

print_success "Container started with IP: $CONTAINER_IP"

# Get container network details
NETWORK=$(docker inspect -f '{{range $key, $value := .NetworkSettings.Networks}}{{$key}}{{end}}' $CONTAINER_NAME)
print_info "Container network: $NETWORK"

# Step 2: Get NAT configuration
print_info "Checking current NAT configuration..."
curl -s "$AGENT_API/api/v1/config/nat" | jq '.' || print_error "Failed to get NAT config"

# Step 3: Add policy for container (using original IP)
print_info "Adding policy for container (original IP: $CONTAINER_IP)..."

POLICY_JSON=$(cat <<EOF
{
  "rule_id": 100,
  "src_ip": "$CONTAINER_IP/32",
  "dst_ip": "0.0.0.0/0",
  "dst_port": 0,
  "protocol": "any",
  "action": "allow",
  "direction": "egress"
}
EOF
)

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$AGENT_API/api/v1/policies" \
    -H "Content-Type: application/json" \
    -d "$POLICY_JSON")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
    print_success "Policy added successfully"
else
    print_error "Failed to add policy (HTTP $HTTP_CODE)"
    echo "$BODY"
    exit 1
fi

# Step 4: Test connectivity from container
print_info "Testing connectivity from container..."
if docker exec $CONTAINER_NAME wget -q -O- http://example.com > /dev/null 2>&1; then
    print_success "Container can access external network (NAT working)"
else
    print_error "Container cannot access external network"
    exit 1
fi

# Step 5: Check NAT statistics
print_info "Checking NAT statistics..."
NAT_STATS=$(curl -s "$AGENT_API/api/v1/stats/nat")

if [ $? -eq 0 ]; then
    echo "$NAT_STATS" | jq '.'

    # Parse statistics
    TOTAL_LOOKUPS=$(echo "$NAT_STATS" | jq -r '.total_lookups // 0')
    CACHE_HITS=$(echo "$NAT_STATS" | jq -r '.cache_hits // 0')
    SNAT_DETECTED=$(echo "$NAT_STATS" | jq -r '.snat_detected // 0')

    print_info "NAT Lookups: $TOTAL_LOOKUPS"
    print_info "Cache Hits: $CACHE_HITS"
    print_info "SNAT Detected: $SNAT_DETECTED"

    if [ "$SNAT_DETECTED" -gt 0 ]; then
        print_success "SNAT detection working!"
    else
        print_error "No SNAT detected (expected for Docker bridge network)"
    fi
else
    print_error "Failed to get NAT statistics"
fi

# Step 6: Check flow events
print_info "Checking flow events..."
FLOWS=$(curl -s "$AGENT_API/api/v1/flows?src_ip=$CONTAINER_IP&limit=10")

if [ $? -eq 0 ]; then
    FLOW_COUNT=$(echo "$FLOWS" | jq '.total // 0')
    print_info "Flows from container IP ($CONTAINER_IP): $FLOW_COUNT"

    if [ "$FLOW_COUNT" -gt 0 ]; then
        print_success "Flow events captured with original container IP"
        echo "$FLOWS" | jq '.flows[0]' 2>/dev/null || true
    else
        print_error "No flows found for container IP"
        print_info "This might indicate NAT address restoration is not working correctly"
    fi
else
    print_error "Failed to query flow events"
fi

# Step 7: Check conntrack sync statistics
print_info "Checking conntrack synchronization statistics..."
CONNTRACK_STATS=$(curl -s "$AGENT_API/api/v1/stats/conntrack" 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$CONNTRACK_STATS" ]; then
    echo "$CONNTRACK_STATS" | jq '.' || echo "$CONNTRACK_STATS"
    print_success "Conntrack sync statistics available"
else
    print_info "Conntrack statistics endpoint not available (optional)"
fi

# Step 8: Summary
echo ""
echo "========================================="
echo "  Test Summary"
echo "========================================="
print_success "Docker container created: $CONTAINER_NAME ($CONTAINER_IP)"
print_success "Policy configured for original container IP"
print_success "Connectivity test passed"

if [ "$SNAT_DETECTED" -gt 0 ]; then
    print_success "NAT detection: WORKING ✓"
else
    print_error "NAT detection: NOT DETECTED ✗"
fi

if [ "$FLOW_COUNT" -gt 0 ]; then
    print_success "Policy matching: WORKING ✓"
else
    print_error "Policy matching: FAILED ✗"
fi

echo ""
print_info "Test environment will be cleaned up automatically"
print_info "To keep the container for manual inspection, press Ctrl+C before cleanup"
sleep 3

exit 0
