#!/bin/bash
# Performance benchmark script for eBPF microsegmentation

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}eBPF Microsegmentation Benchmark${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Error: This script must be run as root${NC}"
    exit 1
fi

# Configuration
IFACE="${1:-lo}"
DURATION="${2:-30}"
AGENT_BIN="../../bin/microsegment-agent"
PERF_TEST_BIN="../../bin/perf-test"

echo -e "${YELLOW}Configuration:${NC}"
echo "  Interface: $IFACE"
echo "  Duration: ${DURATION}s"
echo ""

# Build if needed
if [ ! -f "$AGENT_BIN" ]; then
    echo -e "${YELLOW}Building agent...${NC}"
    cd ../.. && make agent && cd -
fi

# Compile performance test tool
echo -e "${YELLOW}Building performance test tool...${NC}"
cd ../../src/agent
go build -o ../../bin/perf-test cmd/perf_test.go
cd -
echo -e "${GREEN}✓ Built performance test tool${NC}"
echo ""

# Start performance test
echo -e "${GREEN}Starting performance test...${NC}"
echo -e "${YELLOW}Tip: In another terminal, generate traffic:${NC}"
echo "  ping 127.0.0.1"
echo "  curl http://127.0.0.1"
echo "  hping3 -S 127.0.0.1 -p 80 --flood"
echo ""

$PERF_TEST_BIN -iface "$IFACE" -duration "$DURATION" -interval 5

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Benchmark Complete${NC}"
echo -e "${GREEN}========================================${NC}"

# Additional performance analysis
echo ""
echo -e "${YELLOW}Additional Performance Tips:${NC}"
echo "1. Use 'bpftool prog profile' for detailed latency measurement"
echo "2. Use 'perf stat' to measure CPU cycles and cache misses"
echo "3. Test with traffic generators like 'pktgen' or 'T-Rex'"
echo "4. Monitor with 'bpftool prog show' and 'bpftool map show'"
echo ""
echo -e "${YELLOW}Example commands:${NC}"
echo "  # Show loaded eBPF programs:"
echo "  bpftool prog show"
echo ""
echo "  # Profile eBPF program:"
echo "  bpftool prog profile id <prog_id> duration 10"
echo ""
echo "  # Show map statistics:"
echo "  bpftool map show"
echo "  bpftool map dump name session_map | head -20"

