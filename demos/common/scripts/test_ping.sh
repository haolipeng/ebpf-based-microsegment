#!/bin/bash
# Common test script for generating test traffic

set -e

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default target
TARGET=${1:-127.0.0.1}
COUNT=${2:-5}

echo "========================================="
echo "Generating Test Traffic"
echo "========================================="
echo "Target: $TARGET"
echo "Count: $COUNT packets"
echo ""

echo -e "${YELLOW}Sending ICMP ping...${NC}"
ping -c $COUNT $TARGET
echo ""

echo -e "${GREEN}✓ Test traffic generated${NC}"
echo "Check eBPF program output with:"
echo "  sudo cat /sys/kernel/debug/tracing/trace_pipe"
echo "  or"
echo "  sudo bpftool map dump name <map_name>"
