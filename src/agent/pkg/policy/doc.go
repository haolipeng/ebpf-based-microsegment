// Package policy provides policy management functionality for the
// microsegmentation system.
//
// It handles:
//   - Policy creation, retrieval, update, and deletion
//   - Translation between Go structs and eBPF map entries
//   - Policy validation and conflict detection
//
// # Policy Model
//
// A policy is defined by a 5-tuple:
//   - Source IP (CIDR notation)
//   - Destination IP (CIDR notation)
//   - Source port (0 for any)
//   - Destination port (0 for any)
//   - Protocol (tcp, udp, icmp, any)
//
// And an action:
//   - allow: Permit the traffic
//   - deny: Drop the traffic
//   - log: Permit but generate audit logs
//
// # Example Usage
//
//	// Create policy manager
//	pm := policy.NewManager(dataPlane)
//
//	// Add a policy
//	p := &policy.Policy{
//	    RuleID:   1001,
//	    SrcIP:    "0.0.0.0/0",
//	    DstIP:    "10.0.0.5",
//	    DstPort:  443,
//	    Protocol: "tcp",
//	    Action:   "allow",
//	    Priority: 100,
//	}
//
//	if err := pm.AddPolicy(p); err != nil {
//	    log.Fatal(err)
//	}
//
//	// List all policies
//	policies, err := pm.ListPolicies()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Delete a policy
//	if err := pm.DeletePolicy(p); err != nil {
//	    log.Fatal(err)
//	}
//
// # Implementation Details
//
// Policies are stored in an eBPF HASH map in the kernel.
// The PolicyManager translates between Go structs and the
// binary format expected by eBPF programs.
//
// Currently supported:
//   - Exact 5-tuple matching
//   - IPv4 addresses
//
// Future enhancements:
//   - CIDR range matching (LPM trie)
//   - IPv6 support
//   - Priority-based rule ordering
//
// # Thread Safety
//
// The PolicyManager is NOT thread-safe. Concurrent access should
// be protected by the caller (e.g., using sync.RWMutex).
package policy

