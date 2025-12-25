# Identity-Based Access Control Architecture Design

## Overview

This document presents the architecture design for implementing Cilium-style identity-based access control in the eBPF microsegmentation project.

## Design Decisions (User Confirmed)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Feature Scope | Complete System | Full identity infrastructure needed for production |
| Compatibility | Coexistence Mode | IP-based and Identity-based policies coexist for gradual migration |
| Identity Allocation | Server Centralized | Server allocates IDs, Agent caches, cluster-wide uniqueness |
| Priority | Fast Verification | 2-3 weeks MVP, then iterate |

---

## Architecture Options Analysis

### Option A: Minimal Changes Approach

**Core Idea**: Extend existing structures minimally, reuse as much as possible.

#### Changes Required

1. **eBPF Layer** (`src/bpf/`)
   - Add `ipcache_map` (LPM_TRIE): IP → Identity mapping
   - Add `identity_policy_map` (HASH): SrcIdentity × DstIdentity × Port × Protocol → Action
   - Modify `policy_match.h` to query identity first, fall back to IP

2. **Agent Layer** (`src/agent/`)
   - Add `pkg/identity/cache.go`: Local identity cache synced from Server
   - Extend `pkg/workload/types.go`: Add `IdentityID uint32` field
   - Extend `pkg/policy/policy.go`: Add identity-based policy methods

3. **Server Layer** (`src/server/`)
   - Add `pkg/identity/allocator.go`: Central identity allocator
   - Add `pkg/storage/identity_storage.go`: Identity persistence
   - Extend gRPC APIs for identity sync

**Pros**:
- Minimal code changes (~1500 LOC)
- Fast to implement (~2 weeks)
- Low risk to existing functionality

**Cons**:
- Identity logic scattered across modules
- Hard to unit test identity allocation
- Technical debt accumulates

**Estimated Effort**: 2 weeks

---

### Option B: Clean Architecture Approach

**Core Idea**: Design a proper identity subsystem with clear boundaries.

#### Module Structure

```
src/agent/pkg/
├── identity/           # Identity core module
│   ├── types.go       # Identity, NumericIdentity, IPIdentityPair
│   ├── cache.go       # Local identity cache with TTL
│   ├── syncer.go      # gRPC sync with Server
│   └── resolver.go    # IP → Identity resolution
├── ipcache/           # IPCache module
│   ├── ipcache.go     # LPM trie wrapper
│   ├── listener.go    # Change notifications
│   └── bpf_sync.go    # Sync to eBPF map
├── selector/          # Label selector module
│   ├── selector.go    # Selector parsing and matching
│   └── cache.go       # Selector → Identities cache
└── policy/
    └── identity_policy.go  # Identity-based policy

src/server/pkg/
├── identity/
│   ├── allocator.go   # Central ID allocator
│   ├── manager.go     # Identity lifecycle
│   └── gc.go          # Garbage collection
└── storage/
    └── identity_storage.go  # PostgreSQL persistence
```

#### Interface Definitions

```go
// Identity Manager interface
type IdentityManager interface {
    AllocateIdentity(labels map[string]string) (NumericIdentity, error)
    ReleaseIdentity(id NumericIdentity) error
    GetIdentity(id NumericIdentity) (*Identity, error)
    GetIdentityByLabels(labels map[string]string) (*Identity, error)
}

// IPCache interface
type IPCache interface {
    Upsert(prefix netip.Prefix, identity NumericIdentity) error
    Delete(prefix netip.Prefix) error
    Lookup(ip netip.Addr) (NumericIdentity, bool)
    Subscribe(listener IPCacheListener)
}

// Identity Policy interface
type IdentityPolicy interface {
    AddRule(src, dst SelectorExpression, ports []PortRule, action Action) error
    DeleteRule(id uint32) error
    Evaluate(srcID, dstID NumericIdentity, port uint16, proto uint8) Action
}
```

**Pros**:
- Clean separation of concerns
- Easy to test and maintain
- Future-proof extensibility

**Cons**:
- More code (~3000+ LOC)
- Longer implementation time (~4-5 weeks)
- Higher initial complexity

**Estimated Effort**: 4-5 weeks

---

### Option C: Pragmatic Balance Approach (RECOMMENDED)

**Core Idea**: Clean interfaces but minimal MVP implementation, designed for incremental enhancement.

#### Phase 1: MVP (2 weeks)

**Week 1: Core Infrastructure**

Day 1-2: Identity Data Model
```go
// src/agent/pkg/identity/types.go
type NumericIdentity uint32

const (
    IdentityUnknown NumericIdentity = 0
    IdentityHost    NumericIdentity = 1
    IdentityWorld   NumericIdentity = 2
    IdentityLocal   NumericIdentity = 256  // Start of dynamic IDs
)

type Identity struct {
    ID        NumericIdentity
    Labels    map[string]string
    LabelHash string  // SHA256 of sorted labels
}

// src/agent/pkg/identity/cache.go
type IdentityCache struct {
    mu         sync.RWMutex
    byID       map[NumericIdentity]*Identity
    byHash     map[string]*Identity  // LabelHash → Identity
}
```

Day 3-4: IPCache and eBPF Map
```c
// src/bpf/headers/ipcache.h
struct ipcache_key {
    struct bpf_lpm_trie_key lpm;
    __u32 ip[4];  // IPv4/IPv6
} __attribute__((packed));

struct ipcache_value {
    __u32 identity;
    __u8  pad[4];
} __attribute__((packed));

// New map
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __uint(max_entries, 65536);
    __type(key, struct ipcache_key);
    __type(value, struct ipcache_value);
    __uint(pinning, LIBBPF_PIN_BY_NAME);
    __uint(map_flags, BPF_F_NO_PREALLOC);
} ipcache_map SEC(".maps");
```

Day 5: Identity Policy Map
```c
// src/bpf/headers/identity_policy.h
struct identity_policy_key {
    __u32 src_identity;
    __u32 dst_identity;
    __u16 dst_port;
    __u8  protocol;
    __u8  pad;
} __attribute__((packed));

struct identity_policy_value {
    __u8  action;
    __u8  log_enabled;
    __u16 priority;
    __u32 rule_id;
} __attribute__((packed));

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 50000);
    __type(key, struct identity_policy_key);
    __type(value, struct identity_policy_value);
    __uint(pinning, LIBBPF_PIN_BY_NAME);
} identity_policy_map SEC(".maps");
```

**Week 2: Integration**

Day 1-2: Server Identity Allocator
```go
// src/server/pkg/identity/allocator.go
type Allocator struct {
    nextID    uint32  // Atomic counter
    storage   IdentityStorage
    mu        sync.Mutex
}

func (a *Allocator) AllocateOrGet(labels map[string]string) (*Identity, bool, error) {
    hash := computeLabelHash(labels)

    // Check if identity exists
    if id, err := a.storage.GetByLabelHash(hash); err == nil {
        return id, false, nil  // Existing identity
    }

    // Allocate new
    newID := atomic.AddUint32(&a.nextID, 1)
    identity := &Identity{
        ID:        NumericIdentity(newID),
        Labels:    labels,
        LabelHash: hash,
    }

    if err := a.storage.Create(identity); err != nil {
        return nil, false, err
    }

    return identity, true, nil  // New identity
}
```

Day 3-4: gRPC Sync Protocol
```protobuf
// api/proto/identity/identity.proto
message Identity {
    uint32 id = 1;
    map<string, string> labels = 2;
}

message IPIdentityPair {
    string prefix = 1;  // CIDR notation
    uint32 identity = 2;
}

service IdentityService {
    rpc StreamIdentities(stream IdentityRequest) returns (stream IdentityResponse);
    rpc GetIPCache(GetIPCacheRequest) returns (GetIPCacheResponse);
}
```

Day 5: Policy Match Integration
```c
// src/bpf/headers/policy_match.h - Add to match_policy()
static __always_inline int match_policy_with_identity(
    struct flow_key *key,
    __u8 direction
) {
    // Step 1: Lookup source identity from IPCache
    struct ipcache_key src_key = {};
    src_key.lpm.prefixlen = 128;  // Full match
    __builtin_memcpy(src_key.ip, key->src_ip, 16);

    struct ipcache_value *src_id = bpf_map_lookup_elem(&ipcache_map, &src_key);
    if (!src_id)
        return -1;  // Fall back to IP policy

    // Step 2: Lookup destination identity
    struct ipcache_key dst_key = {};
    dst_key.lpm.prefixlen = 128;
    __builtin_memcpy(dst_key.ip, key->dst_ip, 16);

    struct ipcache_value *dst_id = bpf_map_lookup_elem(&ipcache_map, &dst_key);
    if (!dst_id)
        return -1;  // Fall back to IP policy

    // Step 3: Match identity policy
    struct identity_policy_key policy_key = {
        .src_identity = src_id->identity,
        .dst_identity = dst_id->identity,
        .dst_port = key->dst_port,
        .protocol = key->protocol,
    };

    struct identity_policy_value *val = bpf_map_lookup_elem(&identity_policy_map, &policy_key);
    if (val) {
        return val->action;
    }

    // Step 4: Try wildcard dst_port (0)
    policy_key.dst_port = 0;
    val = bpf_map_lookup_elem(&identity_policy_map, &policy_key);
    if (val) {
        return val->action;
    }

    return -1;  // No identity policy, fall back to IP
}
```

#### Phase 2: Enhancements (Post-MVP)

1. **Label Selector Cache**: Pre-compute selector → identity set mappings
2. **Incremental Sync**: Only sync changed identities
3. **Reference Counting**: GC unused identities
4. **Web UI**: Identity management dashboard
5. **Metrics**: Identity allocation/hit rate monitoring

---

## Recommended Approach: Option C

Based on the user's requirement for **fast verification in 2-3 weeks**, Option C is recommended.

### Key Trade-offs

| Aspect | Option C Choice | Future Enhancement |
|--------|-----------------|-------------------|
| Identity Allocation | Simple atomic counter | Add reference counting |
| IPCache Sync | Full sync on startup | Add incremental sync |
| Selector Matching | Runtime evaluation | Add pre-computed cache |
| Garbage Collection | Manual cleanup | Add automatic GC |

### Technical Debt Accepted

1. No reference counting in MVP (manual identity cleanup)
2. Full IPCache sync (not incremental)
3. No selector pre-computation (evaluate at policy add time)
4. Basic error handling (no retry logic)

### Files to Create/Modify

#### New Files

| File | Purpose | LOC Est |
|------|---------|---------|
| `src/agent/pkg/identity/types.go` | Core identity types | 80 |
| `src/agent/pkg/identity/cache.go` | Local identity cache | 150 |
| `src/agent/pkg/identity/syncer.go` | gRPC sync client | 200 |
| `src/agent/pkg/ipcache/ipcache.go` | IPCache manager | 180 |
| `src/agent/pkg/ipcache/bpf_sync.go` | BPF map sync | 120 |
| `src/server/pkg/identity/allocator.go` | ID allocator | 150 |
| `src/server/pkg/identity/manager.go` | Identity CRUD | 200 |
| `src/server/pkg/storage/identity_storage.go` | DB persistence | 180 |
| `src/bpf/headers/ipcache.h` | IPCache BPF structures | 50 |
| `src/bpf/headers/identity_policy.h` | Identity policy structures | 60 |
| `api/proto/identity/identity.proto` | gRPC definitions | 80 |

**Total New Code**: ~1450 LOC

#### Modified Files

| File | Changes | LOC Est |
|------|---------|---------|
| `src/bpf/tc_microsegment.bpf.c` | Add IPCache/identity maps | +50 |
| `src/bpf/headers/policy_match.h` | Add identity matching | +80 |
| `src/agent/pkg/workload/types.go` | Add IdentityID field | +10 |
| `src/agent/cmd/main.go` | Initialize identity syncer | +30 |
| `src/server/cmd/main.go` | Initialize identity allocator | +30 |
| `src/server/pkg/grpc/agent_service.go` | Add identity sync RPC | +100 |

**Total Modified Code**: ~300 LOC

### Implementation Roadmap

```
Week 1:
├── Day 1: Identity types and cache (Agent)
├── Day 2: IPCache BPF structures and map
├── Day 3: Identity policy BPF map
├── Day 4: Identity allocator (Server)
└── Day 5: Identity storage (PostgreSQL)

Week 2:
├── Day 1: gRPC protocol definition
├── Day 2: Identity sync service (Server)
├── Day 3: Identity syncer (Agent)
├── Day 4: Policy match integration
└── Day 5: Integration testing

Week 3 (Buffer/Polish):
├── Day 1-2: Bug fixes and edge cases
├── Day 3: Basic Web UI for identities
└── Day 4-5: Documentation and demo
```

---

## Data Flow Diagram

```
                                    Server
                    ┌─────────────────────────────────────────┐
                    │                                         │
    Web UI          │   ┌─────────────┐   ┌───────────────┐  │
   ┌──────┐         │   │  Identity   │   │   Identity    │  │
   │Policy│────────►│   │  Allocator  │◄──│   Storage     │  │
   │ API  │         │   └──────┬──────┘   │  (PostgreSQL) │  │
   └──────┘         │          │          └───────────────┘  │
                    │          │ gRPC                        │
                    │          ▼                             │
                    │   ┌─────────────┐                      │
                    │   │  Identity   │                      │
                    │   │  Sync Svc   │                      │
                    │   └──────┬──────┘                      │
                    └──────────┼──────────────────────────────┘
                               │ Stream
                    ┌──────────▼──────────────────────────────┐
                    │                Agent                    │
                    │   ┌─────────────┐   ┌───────────────┐  │
                    │   │  Identity   │   │   Workload    │  │
                    │   │  Syncer     │◄──│   Manager     │  │
                    │   └──────┬──────┘   └───────┬───────┘  │
                    │          │                  │          │
                    │          ▼                  ▼          │
                    │   ┌─────────────┐   ┌───────────────┐  │
                    │   │  Identity   │   │   IPCache     │  │
                    │   │  Cache      │──►│   Manager     │  │
                    │   └─────────────┘   └───────┬───────┘  │
                    │                             │          │
                    └─────────────────────────────┼──────────┘
                                                  │ BPF Map Sync
                    ┌─────────────────────────────▼──────────┐
                    │              eBPF (Kernel)             │
                    │   ┌─────────────┐   ┌───────────────┐  │
                    │   │ ipcache_map │   │identity_policy│  │
                    │   │ (LPM_TRIE)  │   │    _map       │  │
                    │   └──────┬──────┘   └───────┬───────┘  │
                    │          │                  │          │
                    │          └──────┬───────────┘          │
                    │                 ▼                      │
                    │   ┌─────────────────────────────────┐  │
                    │   │      Policy Match Logic         │  │
                    │   │  1. Lookup src/dst identity     │  │
                    │   │  2. Match identity policy       │  │
                    │   │  3. Fallback to IP policy       │  │
                    │   └─────────────────────────────────┘  │
                    └────────────────────────────────────────┘
```

---

## Next Steps

1. **Create implementation tasks** in todo list
2. **Start with identity types** (`src/agent/pkg/identity/types.go`)
3. **Implement eBPF maps** in parallel
4. **Integrate and test** incrementally

---

**Document Version**: 1.0
**Created**: 2025-12-18
**Author**: AI Assistant
