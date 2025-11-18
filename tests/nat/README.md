# NAT Support Testing

This directory contains test scripts for validating NAT (Network Address Translation) support in the microsegmentation system.

## Overview

The NAT support enables correct policy matching in Docker and Kubernetes environments where SNAT/DNAT is used. The system detects NAT transformations and restores original IP addresses before policy matching.

## Test Scripts

### `docker-nat-test.sh`

Tests NAT support in Docker bridge network environment (SNAT scenario).

**What it tests:**
1. Docker container creation with bridge network
2. Policy configuration using original container IP
3. NAT detection and address restoration
4. Connectivity verification
5. NAT statistics validation

**Prerequisites:**
- Docker installed and running
- Microsegmentation agent running (`./bin/microsegment-agent`)
- `curl` and `jq` installed

**Usage:**
```bash
# Start the agent first (in another terminal)
sudo ./bin/microsegment-agent -c config.yaml

# Run the NAT test
cd tests/nat
sudo ./docker-nat-test.sh
```

**Expected Output:**
```
=========================================
  Docker NAT Testing for Microsegmentation
=========================================

✓ Prerequisites OK
✓ Agent is running
✓ Container started with IP: 172.17.0.2
✓ Policy added successfully
✓ Container can access external network (NAT working)
✓ SNAT detection working!
✓ Flow events captured with original container IP

=========================================
  Test Summary
=========================================
✓ Docker container created: nat-test-container (172.17.0.2)
✓ Policy configured for original container IP
✓ Connectivity test passed
✓ NAT detection: WORKING ✓
✓ Policy matching: WORKING ✓
```

## How NAT Detection Works

### Architecture

```
Container (172.17.0.2) → Docker Bridge → Host (NAT) → Internet
                          ↓ SNAT
           Src IP: 172.17.0.2 → 192.168.1.100

eBPF Program:
1. Sees packet with src=192.168.1.100 (post-NAT)
2. Queries conntrack cache
3. Finds original src=172.17.0.2 (pre-NAT)
4. Matches policy using original IP
5. Creates session using post-NAT IP (for subsequent lookups)
```

### Flow Diagram

```
┌─────────────┐
│  Packet In  │ (Post-NAT: 192.168.1.100)
└──────┬──────┘
       │
       ▼
┌──────────────────────┐
│ Extract 5-tuple      │ (Post-NAT addresses)
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ NAT Detection        │
│ detect_nat_and_      │
│ restore_with_maps()  │
└──────┬───────────────┘
       │
       ├─→ Query BPF Helper (kernel >= 5.18) ✗ (not implemented)
       │
       └─→ Query Conntrack Cache Map ✓
           ↓
    ┌──────────────┐
    │ Found Entry  │ → Original: 172.17.0.2
    └──────┬───────┘
           │
           ▼
    ┌──────────────────┐
    │ Policy Matching  │ (Using pre-NAT IP: 172.17.0.2)
    └──────┬───────────┘
           │
           ▼
    ┌──────────────────┐
    │ Session Creation │ (Using post-NAT IP as key)
    └──────────────────┘
```

## Conntrack Synchronization

The conntrack synchronization runs in user-space and syncs kernel conntrack entries to eBPF maps:

**Sync Process:**
1. **Initial Full Sync**: On startup, dump all conntrack entries
2. **Periodic Sync**: Every 30 seconds (default), re-sync full table
3. **Event Sync**: *Currently disabled* (will be re-enabled with context support)

**Conntrack Entry Format:**
```c
struct conntrack_key {
    u32 src_ip[4];      // Post-NAT source IP
    u32 dst_ip[4];      // Post-NAT dest IP
    u16 src_port;       // Post-NAT source port
    u16 dst_port;       // Post-NAT dest port
    u8  protocol;
    u8  ip_version;
};

struct conntrack_entry {
    struct flow_key original_tuple;  // Pre-NAT addresses
    struct flow_key reply_tuple;     // Reply direction
    u64 timestamp;
    u32 status;        // Conntrack status flags
    u8  nat_type;      // SNAT/DNAT/BOTH/NONE
};
```

## NAT Statistics

View NAT detection statistics via API:

```bash
curl http://localhost:8080/api/v1/stats/nat | jq
```

**Example Output:**
```json
{
  "total_lookups": 1234,
  "cache_hits": 1200,
  "cache_misses": 34,
  "bpf_helper_success": 0,
  "bpf_helper_failed": 0,
  "snat_detected": 500,
  "dnat_detected": 0,
  "both_detected": 0,
  "no_nat_detected": 734,
  "restore_success": 500,
  "restore_failed": 0,
  "cache_hit_rate": 0.972
}
```

## Configuration

NAT configuration can be set via API:

```bash
# Enable NAT detection with cache lookup
curl -X PUT http://localhost:8080/api/v1/config/nat \
  -H "Content-Type: application/json" \
  -d '{
    "match_mode": 0,
    "enable_cache": true,
    "enable_bpf_helper": false,
    "log_events": false
  }'
```

**Match Modes:**
- `0` - NAT_MATCH_MODE_ORIGINAL: Match using pre-NAT addresses (recommended)
- `1` - NAT_MATCH_MODE_TRANSLATED: Match using post-NAT addresses
- `2` - NAT_MATCH_MODE_BOTH: Try both

## Troubleshooting

### No SNAT Detected

If `snat_detected` is 0:
1. Check if conntrack sync is running: `curl http://localhost:8080/api/v1/stats/conntrack`
2. Verify container traffic is going through NAT: `docker exec <container> ip route`
3. Check conntrack entries: `sudo conntrack -L | grep <container_ip>`

### Policy Not Matching

If flows are not captured with original IP:
1. Verify NAT mode is set to "original": `curl http://localhost:8080/api/v1/config/nat`
2. Check cache hit rate (should be >90% after initial sync)
3. Enable debug logging: Set `log_events: true` in NAT config

### Low Cache Hit Rate

If cache hit rate is low (<80%):
1. Increase sync interval in conntrack config
2. Check for connection churn (many short-lived connections)
3. Verify LRU map size is sufficient (default: 200K entries)

## References

- NAT Support Design: `/docs/NAT_SUPPORT_IMPLEMENTATION.md`
- Conntrack Package: `/src/agent/pkg/conntrack/`
- BPF NAT Support: `/src/bpf/headers/nat_support.h`
