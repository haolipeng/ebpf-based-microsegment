#!/bin/bash
# Test script for Demo 2: 5-Tuple Extractor

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

IFACE=${1:-lo}

echo "========================================="
echo "Demo 2: 5-Tuple Extractor Test"
echo "========================================="
echo "Interface: $IFACE"
echo ""

# Check if eBPF program is loaded
if ! sudo tc filter show dev $IFACE ingress | grep -q "direct-action"; then
    echo -e "${YELLOW}Warning: eBPF program not loaded${NC}"
    echo "Run 'sudo make load' first"
    exit 1
fi

echo -e "${BLUE}Step 1: Generate ICMP traffic${NC}"
echo "Sending 3 ping packets to 127.0.0.1..."
ping -c 3 127.0.0.1 > /dev/null 2>&1
echo -e "${GREEN}✓ ICMP traffic generated${NC}"
echo ""

echo -e "${BLUE}Step 2: Generate TCP traffic (if possible)${NC}"
if command -v nc &> /dev/null; then
    echo "Starting TCP listener on port 8888..."
    nc -l 8888 &
    NC_PID=$!
    sleep 1

    echo "Connecting to localhost:8888..."
    echo "Hello from test" | nc 127.0.0.1 8888 &
    sleep 1

    # Cleanup
    kill $NC_PID 2>/dev/null || true
    echo -e "${GREEN}✓ TCP traffic generated${NC}"
else
    echo -e "${YELLOW}! nc (netcat) not found, skipping TCP test${NC}"
fi
echo ""

echo -e "${BLUE}Step 3: Generate UDP traffic (if possible)${NC}"
if command -v nc &> /dev/null; then
    echo "Starting UDP listener on port 9999..."
    nc -u -l 9999 &
    NC_PID=$!
    sleep 1

    echo "Sending UDP packet to localhost:9999..."
    echo "Hello UDP" | nc -u 127.0.0.1 9999 &
    sleep 1

    # Cleanup
    kill $NC_PID 2>/dev/null || true
    echo -e "${GREEN}✓ UDP traffic generated${NC}"
else
    echo -e "${YELLOW}! nc (netcat) not found, skipping UDP test${NC}"
fi
echo ""

echo -e "${BLUE}Step 4: Check protocol statistics${NC}"
echo "Protocol counters (non-zero):"
sudo bpftool map dump name protocol_counter | grep -v ': 00 00' | while read line; do
    # Parse protocol number from key
    if [[ $line =~ key:\ ([0-9a-f]+) ]]; then
        proto_hex="${BASH_REMATCH[1]}"
        proto_dec=$((16#$proto_hex))

        # Map protocol numbers to names
        case $proto_dec in
            1) proto_name="ICMP" ;;
            6) proto_name="TCP" ;;
            17) proto_name="UDP" ;;
            *) proto_name="Protocol $proto_dec" ;;
        esac

        echo "  $proto_name: $line"
    fi
done
echo ""

echo "========================================="
echo -e "${GREEN}✓ Test completed successfully!${NC}"
echo "========================================="
echo ""
echo "To view detailed flow logs, run in another terminal:"
echo "  sudo cat /sys/kernel/debug/tracing/trace_pipe"
echo ""
echo "You should see output like:"
echo "  [FLOW] Proto=1  127.0.0.1:0 → 127.0.0.1:0 (ICMP)"
echo "  [FLOW] Proto=6  127.0.0.1:12345 → 127.0.0.1:8888 (TCP)"
echo "  [FLOW] Proto=17  127.0.0.1:54321 → 127.0.0.1:9999 (UDP)"
