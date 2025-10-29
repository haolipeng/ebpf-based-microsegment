# Project Context

## Purpose
eBPF-based network microsegmentation system for container/VM environments. The project implements a TC (Traffic Control) eBPF program for packet filtering and policy enforcement at kernel level, providing high-performance network security with minimal overhead.

**Goals:**
- Build production-ready eBPF microsegmentation solution
- Learn eBPF programming through hands-on 6-week implementation
- Achieve <10μs latency overhead for packet processing
- Support 100,000+ concurrent sessions

## Tech Stack

### Core Technologies
- **eBPF (Extended Berkeley Packet Filter)** - Kernel-level packet processing
- **libbpf 1.x** - Modern eBPF loader and skeleton framework
- **C** - eBPF programs and user-space control plane
- **TC (Traffic Control)** - Linux traffic control subsystem (ingress/egress hooks)
- **Clang/LLVM** - eBPF program compilation
- **bpftool** - eBPF introspection and debugging

### Supporting Tools
- **Go** (optional) - Alternative for user-space control programs
- **Prometheus** - Metrics collection and monitoring
- **Grafana** - Visualization dashboards
- **Python** - Testing and automation scripts

### Development Environment
- Linux Kernel ≥ 5.10 (for BTF and CO-RE support)
- Ubuntu 22.04+ or equivalent
- VSCode with C/C++ extensions

## Project Conventions

### Code Style

**eBPF Programs (C):**
- File naming: `*.bpf.c` for eBPF code, `*.c` for user-space
- Use `SEC("tc")` for TC programs
- Always include boundary checks: `if ((void *)(hdr + 1) > data_end)`
- Prefer inline functions for code reusability (eBPF verifier friendly)
- Comment eBPF helper functions usage

**User-Space Programs (C):**
- Use libbpf skeleton: `xxx.skel.h` auto-generated from `xxx.bpf.o`
- Loader pattern: `xxx_loader.c` for program lifecycle management
- Signal handling for graceful shutdown (SIGINT/SIGTERM)
- Always check errors and provide meaningful messages

**Naming Conventions:**
- Maps: `policy_map`, `session_map`, `stats_map` (snake_case)
- Structs: `struct policy_key`, `struct tcp_state` (snake_case)
- Functions: `track_session()`, `match_policy()` (snake_case, verb-led)
- Constants: `MAX_ENTRIES`, `TC_ACT_OK` (UPPER_CASE)

### Architecture Patterns

**eBPF Program Structure:**
```
src/bpf/          # eBPF kernel programs
  ├── hello.bpf.c         # TC programs
  ├── parse_packet.bpf.c
  └── microsegment.bpf.c

src/user/         # User-space loaders
  ├── hello_loader.c
  └── microsegment_loader.c
```

**Key Patterns:**
- **Skeleton-based loading**: Use `bpf_object__open_and_load()` for automatic map/program loading
- **TC attachment**: Use `bpf_tc_hook_create()` + `bpf_tc_attach()` for libbpf TC API
- **Map pinning**: Pin maps to `/sys/fs/bpf/` for cross-process access
- **Graceful cleanup**: Use signal handlers + `bpf_tc_detach()` + `bpf_tc_hook_destroy()`

**Data Flow:**
```
Packet → TC Ingress Hook → eBPF Program → Policy Check → Session Tracking → Action (PASS/DROP)
```

### Testing Strategy

**Unit Tests:**
- Test individual eBPF helper functions with mock data
- Verify map operations (insert, lookup, delete)
- Test boundary conditions and edge cases

**Functional Tests:**
- End-to-end packet filtering scenarios
- Policy CRUD operations via CLI
- Session tracking accuracy (TCP/UDP)
- Multi-protocol support (IPv4/IPv6)

**Performance Tests:**
- Throughput measurement (iperf3)
- Latency measurement (ping RTT)
- Connection capacity (100k+ sessions)
- CPU overhead monitoring

**Stress Tests:**
- High packet rate (>1M pps)
- Connection churn (rapid open/close)
- Map pressure (near max_entries)

**Testing Commands:**
```bash
# Load program
sudo ./xxx_loader lo

# Run tests
./tests/integration_test.sh
./tests/performance_test.sh

# Verify with bpftool
sudo bpftool prog show
sudo bpftool map dump name policy_map
```

### Git Workflow

**Branching:**
- `master` - main development branch
- Feature branches: `feature/add-tcp-state-machine`
- No separate `main` branch (using `master`)

**Commit Conventions:**
- Use descriptive commit messages
- Generated commits include Claude Code attribution:
  ```
  Add TCP state machine tracking

  🤖 Generated with Claude Code
  Co-Authored-By: Claude <noreply@anthropic.com>
  ```

**Pull Request Process:**
- Not currently enforced (learning project)
- Future: require PR approval for production deployment

## Domain Context

### eBPF Fundamentals
- **Verifier**: eBPF programs must pass kernel verifier (safety checks)
- **BPF Maps**: Key-value stores for state sharing between kernel and user-space
- **TC Hooks**: Attach points for traffic control (ingress/egress)
- **CO-RE (Compile Once, Run Everywhere)**: BTF-based portability across kernel versions
- **Tail Calls**: Jump to another eBPF program (useful for program decomposition)

### Network Concepts
- **5-tuple**: (src_ip, dst_ip, src_port, dst_port, protocol) - flow identifier
- **Connection tracking**: Track TCP/UDP session state
- **Microsegmentation**: Fine-grained network policy enforcement per workload
- **TC return codes**: `TC_ACT_OK` (pass), `TC_ACT_SHOT` (drop), `TC_ACT_REDIRECT`

### Critical eBPF Requirements
- **Boundary checks**: MUST check `data_end` before accessing packet data
- **Verifier complexity limit**: Programs must be simple enough for verifier
- **Map size limits**: Plan for `max_entries` based on expected scale
- **Stack limit**: 512 bytes per eBPF program

## Important Constraints

### Technical Constraints
- **Kernel version**: Requires Linux ≥5.10 for BTF and modern libbpf features
- **eBPF complexity**: Programs must pass verifier (no unbounded loops, limited stack)
- **Performance**: Target <10μs latency overhead per packet
- **Memory**: Map entries consume kernel memory (plan capacity carefully)

### Development Constraints
- **Learning-focused**: Code should be educational, not production-hardened initially
- **Incremental**: Build complexity gradually over 6 weeks
- **Documentation**: All concepts explained for eBPF beginners

### Operational Constraints
- **Graceful degradation**: Program failures should not crash system
- **Observability**: Must provide visibility via bpftool, Prometheus, logs
- **Hot reload**: Policy updates without reloading eBPF programs

## External Dependencies

### Required System Libraries
- **libbpf** (≥1.0) - eBPF loading and skeleton generation
- **libelf** - ELF file parsing
- **zlib** - Compression support for BTF

### Development Tools
- **clang** (≥10) - Compile eBPF programs to BPF bytecode
- **llvm** - LLVM toolchain for eBPF
- **bpftool** - Inspect and manage eBPF objects
- **tc** - Linux traffic control utility

### Optional Dependencies
- **Prometheus** - Metrics collection (Week 6)
- **Grafana** - Monitoring dashboards (Week 6)
- **iperf3** - Performance testing
- **netcat/curl** - Functional testing

### Reference Projects
- **libbpf-bootstrap** - Official examples for modern libbpf development
- **ZFW (Zero Trust Firewall)** - Reference architecture analyzed in `docs/zfw-architecture-analysis.md`
- **Cilium** - Production eBPF networking (inspiration)

## Learning Resources

### Primary Documentation
- `docs/weekly-guide/` - 6-week structured learning plan
- `docs/zfw-architecture-analysis.md` - In-depth analysis of production eBPF firewall
- `specs/ebpf-tc-implementation.md` - Implementation specification

### Key eBPF References
- libbpf documentation: https://libbpf.readthedocs.io/
- Linux kernel BPF docs: https://www.kernel.org/doc/html/latest/bpf/
- Cilium BPF Reference Guide: https://docs.cilium.io/en/stable/bpf/
