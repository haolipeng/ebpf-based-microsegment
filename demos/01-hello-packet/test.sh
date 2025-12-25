#!/bin/bash
# Test script for Demo 1: Hello Packet Counter

set -e

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

IFACE=${1:-lo}

echo "========================================="
echo "Demo 1: Hello Packet Counter Test"
echo "========================================="
echo "Interface: $IFACE"
echo ""

# Check if eBPF program is loaded
if ! sudo tc filter show dev $IFACE ingress | grep -q "direct-action"; then
    echo -e "${YELLOW}Warning: eBPF program not loaded${NC}"
    echo "Run 'sudo make load' first"
    exit 1
fi

echo -e "${BLUE}Step 1: Check initial counter value${NC}"
sudo bpftool map dump name packet_counter
echo ""

echo -e "${BLUE}Step 2: Generate test traffic (5 ping packets)${NC}"
ping -c 5 127.0.0.1 > /dev/null 2>&1
echo -e "${GREEN}✓ Sent 5 packets${NC}"
echo ""

echo -e "${BLUE}Step 3: Check updated counter value${NC}"
sudo bpftool map dump name packet_counter
echo ""

echo -e "${BLUE}Step 4: Generate more traffic (10 ping packets)${NC}"
ping -c 10 127.0.0.1 > /dev/null 2>&1
echo -e "${GREEN}✓ Sent 10 more packets${NC}"
echo ""

echo -e "${BLUE}Step 5: Final counter value${NC}"
sudo bpftool map dump name packet_counter
echo ""

echo "========================================="
echo -e "${GREEN}✓ Test completed successfully!${NC}"
echo "========================================="
echo ""
echo "Tips:"
echo "  - View real-time debug logs:"
echo "    sudo cat /sys/kernel/debug/tracing/trace_pipe"
echo ""
echo "  - Check counter anytime:"
echo "    sudo bpftool map dump name packet_counter"
echo ""
echo "  - Unload program when done:"
echo "    sudo make unload"
