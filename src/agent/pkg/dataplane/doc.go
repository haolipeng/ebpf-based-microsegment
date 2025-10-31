// Package dataplane provides an interface to the eBPF data plane for
// high-performance packet processing and policy enforcement.
//
// The data plane manages:
//   - eBPF program lifecycle (loading, attachment, cleanup)
//   - TC (Traffic Control) hook integration
//   - Session tracking and statistics collection
//   - Flow event monitoring via ring buffer
//
// # Architecture
//
// The data plane consists of:
//   - eBPF programs (C code compiled to BPF bytecode)
//   - eBPF maps for session tracking, policies, and statistics
//   - Ring buffer for kernel-to-userspace event delivery
//   - Go bindings generated via bpf2go
//
// # Example Usage
//
//	// Create and initialize data plane
//	dp, err := dataplane.New("eth0")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer dp.Close()
//
//	// Load and attach eBPF programs
//	if err := dp.LoadAndAttach(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start monitoring flow events
//	go dp.MonitorFlowEvents()
//
//	// Query statistics
//	stats := dp.GetStatistics()
//	fmt.Printf("Total packets: %d\n", stats.TotalPackets)
//
// # Performance
//
// The eBPF data plane is optimized for minimal latency:
//   - Hot path (existing session): < 1 microsecond
//   - Cold path (new session): < 3 microseconds
//   - Average latency: < 5 microseconds
//   - Throughput: 100K+ packets/sec per CPU core
//
// # Maps
//
// The data plane uses the following eBPF maps:
//   - session_map: LRU_HASH for session tracking (100K entries)
//   - policy_map: HASH for policy storage (10K entries)
//   - stats_map: PERCPU_ARRAY for lock-free statistics (8 counters)
//   - flow_events: RINGBUF for event delivery (256KB)
//
// # Thread Safety
//
// The DataPlane type is safe for concurrent use. Statistics queries
// and map operations can be called from multiple goroutines.
package dataplane

