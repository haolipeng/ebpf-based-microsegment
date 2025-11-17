# Compact Policy Storage

## Overview

The Compact Policy Storage module provides efficient slot management for wildcard policies in the Server component. It maintains a `first_free_index` pointer to minimize gaps in the policy array, reducing the overhead of eBPF linear scans on the Agent side.

## Features

- **Slot Reuse**: Automatically reuses empty slots when policies are deleted
- **First Free Index**: Maintains a pointer to the first available empty slot for O(1) insertion
- **Defragmentation**: Supports compaction to eliminate all gaps
- **Thread-Safe**: All operations are protected by mutex for concurrent access
- **Statistics**: Provides utilization metrics and slot statistics

## Architecture

```
CompactPolicyStorage
├── policies (map[uint32]*PolicySlot)  // RuleID -> Metadata
├── slots ([]*Policy)                  // Array of policies (may have nils)
├── firstFreeIndex (int)               // Index of first empty slot (-1 if none)
└── freeList ([]int)                   // List of all empty slots
```

### Slot Management

**Adding a Policy:**
1. Check if policy exists (update case)
2. If `firstFreeIndex >= 0`: reuse empty slot
3. Otherwise: append to end
4. Update `firstFreeIndex` to next empty slot

**Deleting a Policy:**
1. Mark slot as `nil`
2. Update `firstFreeIndex` if deleted slot is earlier
3. Add slot to `freeList`

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"

    commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
    policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
    "github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/policy"
)

func main() {
    // Create storage with max 1000 slots (eBPF limit)
    storage := policy.NewCompactPolicyStorage(1000)
    ctx := context.Background()

    // Add a policy
    p1 := &policypb.Policy{
        RuleId:   1001,
        SrcIp:    "10.0.0.0/24",
        DstIp:    "192.168.1.0/24",
        SrcPort:  0,
        DstPort:  80,
        Protocol: commonpb.Protocol_PROTOCOL_TCP,
        Action:   commonpb.PolicyAction_ACTION_ALLOW,
        Priority: 100,
    }
    storage.AddPolicy(ctx, p1)

    // Get active policies (for Agent sync)
    policies := storage.GetActivePolicies()
    fmt.Printf("Active policies: %d\n", len(policies))

    // Get storage statistics
    stats := storage.GetStats()
    fmt.Printf("Utilization: %.2f%%\n", stats.Utilization)
    fmt.Printf("Empty slots: %d\n", stats.EmptySlots)
}
```

### Slot Reuse Example

```go
// Add 3 policies
storage.AddPolicy(ctx, &policypb.Policy{RuleId: 1001, ...})
storage.AddPolicy(ctx, &policypb.Policy{RuleId: 1002, ...})
storage.AddPolicy(ctx, &policypb.Policy{RuleId: 1003, ...})

// State: [P1001, P1002, P1003]

// Delete middle policy
storage.DeletePolicy(ctx, 1002)

// State: [P1001, nil, P1003]
// firstFreeIndex = 1

// Add new policy (reuses slot 1)
storage.AddPolicy(ctx, &policypb.Policy{RuleId: 1004, ...})

// State: [P1001, P1004, P1003]
// firstFreeIndex = -1 (no empty slots)
```

### Defragmentation

When there are many gaps and you want to minimize eBPF scan overhead:

```go
// State: [P1, nil, P2, nil, nil, P3]
// 6 slots, 50% utilization

storage.Compact()

// State: [P1, P2, P3]
// 3 slots, 100% utilization
```

## Integration with eBPF Synchronization

The compact storage is designed to work seamlessly with Agent-side eBPF maps:

```go
// Server-side policy management
storage := policy.NewCompactPolicyStorage(1000)

// Add/update/delete policies...
storage.AddPolicy(ctx, newPolicy)
storage.DeletePolicy(ctx, oldRuleID)

// Get compact list for Agent sync
policies := storage.GetCompactPolicies()  // Includes nils for empty slots

// Agent receives and syncs to eBPF wildcard policy map
for i, p := range policies {
    if p != nil {
        wildcardMap.Put(uint32(i), convertToEBPFFormat(p))
    } else {
        wildcardMap.Put(uint32(i), emptyEntry)  // Clear slot
    }
}
```

## Performance Characteristics

| Operation | Time Complexity | Notes |
|-----------|----------------|-------|
| Add Policy | O(1) | Reuses first free slot |
| Delete Policy | O(1) | Marks slot as nil |
| Get Active Policies | O(n) | Filters out nils |
| Get Compact Policies | O(1) | Returns array copy |
| Compact | O(n) | Rebuilds without gaps |

## Statistics and Monitoring

```go
stats := storage.GetStats()

// Monitor utilization
if stats.Utilization < 70.0 {
    fmt.Println("Warning: High fragmentation, consider compacting")
}

// Monitor capacity
if stats.ActivePolicies >= stats.MaxSlots * 0.9 {
    fmt.Println("Warning: Approaching capacity limit")
}
```

## Thread Safety

All public methods are thread-safe and can be called concurrently from multiple goroutines.

## Limitations

- Maximum slots: Configurable (default: 1000, matching eBPF `MAX_ENTRIES_WILDCARD_POLICY`)
- Compaction temporarily blocks all operations

## Testing

Run unit tests:

```bash
cd src/server
go test -v ./pkg/policy -run TestCompact
```

Test coverage:
- Basic add/delete operations
- Slot reuse logic
- Update existing policies
- Defragmentation
- Concurrent operations
- Edge cases (max capacity, empty storage)

## Future Improvements

1. **Automatic Compaction**: Trigger compaction when fragmentation > threshold
2. **Persistent Storage**: Integration with PostgreSQL for durability
3. **Event Notification**: Notify agents when policies change
4. **Metrics Export**: Prometheus metrics for monitoring

## Related Components

- `src/agent/pkg/policy/policy.go`: Agent-side eBPF synchronization
- `src/bpf/headers/policy_match.h`: eBPF policy matching logic (Task #18 optimization)
- `src/agent/test/benchmark/`: Performance benchmarking framework (Task #21)

## References

- Task #18: eBPF 循环常量化与早停优化 (MAX_WILDCARD_LOOP=50)
- Task #19: Server 端紧凑存储实现 (This module)
- Task #21: 性能基准测试框架搭建
