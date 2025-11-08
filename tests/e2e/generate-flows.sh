#!/bin/sh

# Generate synthetic flow data and send to server via Agent Reporter
# This simulates eBPF flow events for testing

set -e

echo "Starting flow generator..."
echo "Server: ${SERVER_ADDR}"
echo "Agent ID: ${AGENT_ID}"

# Create a simple Go program to generate and send flows
cat > /tmp/flow-generator.go << 'EOF'
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/flow"
	"github.com/ebpf-microsegment/src/agent/pkg/reporter"
)

func main() {
	serverAddr := "server:9090"
	agentID := "test-agent-001"

	// Create reporter
	rep := reporter.NewGRPCReporter(serverAddr, agentID, 10)
	if err := rep.Start(); err != nil {
		panic(fmt.Sprintf("Failed to start reporter: %v", err))
	}
	defer rep.Stop()

	fmt.Println("Flow generator started, sending flows...")

	ctx := context.Background()

	// Generate 50 test flows
	for i := 0; i < 50; i++ {
		f := &flow.Flow{
			ID:         fmt.Sprintf("test-flow-%03d", i),
			SourceIP:   fmt.Sprintf("192.168.1.%d", 10+i),
			SourcePort: uint16(10000 + rand.Intn(50000)),
			DestIP:     fmt.Sprintf("10.0.0.%d", 1+i%10),
			DestPort:   80,
			Protocol:   "TCP",
			EventType:  "NEW",
			Direction:  "EGRESS",
			State:      "ACTIVE",
			PacketCount: uint64(100 + rand.Intn(1000)),
			ByteCount:   uint64(10000 + rand.Intn(100000)),
			StartTime:   time.Now(),
			SourceLabels: map[string]string{
				"app": fmt.Sprintf("app-%d", i%5),
				"env": "test",
			},
			DestLabels: map[string]string{
				"app": fmt.Sprintf("backend-%d", i%3),
				"env": "test",
			},
		}

		if err := rep.Report(ctx, f); err != nil {
			fmt.Printf("Warning: Failed to report flow %d: %v\n", i, err)
		}

		if (i+1) % 10 == 0 {
			fmt.Printf("Sent %d flows\n", i+1)
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("All flows sent, waiting for batch flush...")
	time.Sleep(7 * time.Second)
	fmt.Println("Flow generator completed")
}
EOF

# Note: In a real Docker environment, we would build and run this
# For now, this script serves as documentation
echo "Flow generator script created"
echo "Sleeping to keep container alive for testing..."
sleep infinity
