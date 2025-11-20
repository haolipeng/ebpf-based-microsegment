# eBPF-based Microsegmentation

[![License](https://img.shields.io/badge/License-GPL%202.0%20%7C%20BSD--3-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![eBPF](https://img.shields.io/badge/eBPF-Powered-orange)](https://ebpf.io/)

🔒 A high-performance, eBPF-powered microsegmentation solution for cloud-native workloads, inspired by **Illumio** and **蔷薇灵动**.

## 🌟 Features

- **🚀 High Performance**: Kernel-level packet filtering with <10μs latency overhead
- **🎯 Session Tracking**: Intelligent connection tracking using LRU hash maps
- **📊 Real-time Visibility**: Live flow events and traffic statistics
- **📈 Flow Collection API**: Network flow data collection, storage, and analysis (Phase 1-3 ✅)
  - 7 REST API endpoints for flow queries
  - SQLite persistence with optimized indexes
  - Application dependency mapping
  - Top Talkers analysis
- **🏷️ Label-based Policies**: Cloud-native policy management (in progress)
- **👤 Process-Level Policies**: Application-aware microsegmentation
  - Process name-based policy matching in TC/XDP programs (Issue #47 ✅)
  - Integration with eBPF tracepoint process monitoring (Issue #46 ✅)
  - Flow events enriched with process context (PID, comm, container ID, path) (Issue #49 ✅)
  - ProcessMonitor daemon for process info enrichment (Issue #48 ✅)
  - Security monitoring and suspicious process detection (Issue #50 ✅)
- **🔴 Real-time Flow Streaming**: WebSocket push for live flow events (Phase 4, coming soon)
- **🤖 Auto Policy Generation**: ML-powered policy recommendations (planned)
- **🛡️ Zero Trust Ready**: Built for zero trust network architecture

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Web Console (前端)                    │
│        React + D3.js (流量拓扑可视化)                   │
└────────────────────┬────────────────────────────────────┘
                     │ REST API
┌────────────────────▼────────────────────────────────────┐
│              Control Plane (控制平面)                    │
│    Go: 策略管理 + 标签管理 + 流量分析                   │
└────────────────────┬────────────────────────────────────┘
                     │ gRPC/JSON
┌────────────────────▼────────────────────────────────────┐
│               Data Plane (数据平面)                      │
│    eBPF + TC: 策略执行引擎                               │
│    - 5-tuple Flow Matching                              │
│    - Session Tracking (LRU_HASH)                        │
│    - Policy Enforcement (ALLOW/DENY/LOG)                │
│    - Ring Buffer Events                                  │
└─────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Linux Kernel ≥ 5.10 (with BTF support)
- Go ≥ 1.21
- Clang ≥ 11
- libbpf development files

### Installation

```bash
# Install dependencies (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install -y clang llvm libbpf-dev linux-headers-$(uname -r) build-essential

# Clone repository
git clone https://github.com/yourusername/ebpf-based-microsegment.git
cd ebpf-based-microsegment

# Download Go dependencies
make deps

# Generate eBPF bindings and build
make bpf
make agent
```

### Running the Agent

```bash
# Run on loopback interface (for testing)
sudo ./bin/microsegment-agent --interface lo --log-level info

# Run on production interface
sudo ./bin/microsegment-agent --interface eth0 --log-level warn --stats-interval 10
```

### CLI Options

```
Flags:
  -i, --interface string       Network interface to attach eBPF program (default "lo")
  -l, --log-level string       Log level (debug, info, warn, error) (default "info")
  -s, --stats-interval int     Statistics print interval in seconds (default 5)
  -h, --help                   help for microsegment-agent
```

## 📖 Documentation

### Core Documentation
- [Project Structure](PROJECT_STRUCTURE.md) - Detailed directory layout and module descriptions
- [Implementation Plan](docs/microsegmentation-mvp-implementation-plan.md) - MVP roadmap and milestones
- [Architecture Design](design-docs/architecture/design.md) - Technical architecture details
- [Weekly Guide](docs/weekly-guide/) - 6-week learning and implementation guide

### eBPF Data Plane Development (NEW! 🔥)
- **[Quick Start Guide](docs/EBPF_QUICK_START_GUIDE.md)** - Priority roadmap and implementation guide
- **[Full Roadmap](docs/EBPF_MICROSEGMENTATION_ROADMAP.md)** - Complete feature development plan (15-20 days)
- **Current Status**: ~60-65% complete | **Next Priority**: NAT Support (P0)

### Flow Collection API (Phase 1-3 ✅)
- [Quick Start Guide](docs/flow-quick-start.md) - Get started with Flow Collection API in 10 minutes
- [Implementation Summary](docs/flow-collection-implementation-summary.md) - Complete technical documentation (32,000 words)
- [Progress Report](docs/flow-implementation-progress.md) - Current status and roadmap
- [OpenSpec Design](openspec/changes/add-flow-collection-api/design.md) - Detailed architecture design

## 🚀 Deployment

### Systemd Deployment (Linux Services)

Deploy on traditional Linux environments using Systemd services:

```bash
# Quick installation
sudo ./deploy/scripts/install-systemd.sh

# Check service status
systemctl status microsegment-server microsegment-agent

# View logs
journalctl -u microsegment-server -f
```

📚 [Full Systemd Deployment Guide](deploy/systemd/README.md)

### Coming Soon
- **Docker Deployment**: Container-based deployment with Docker Compose
- **Kubernetes Deployment**: Production-grade orchestration for K8s clusters

## 🛠️ Development

### Project Structure

```
ebpf-based-microsegment/
├── src/
│   ├── bpf/                    # eBPF kernel programs (C)
│   │   ├── headers/           # Shared header files
│   │   └── tc_microsegment.bpf.c
│   └── agent/                 # User-space agent (Go)
│       ├── cmd/               # CLI entrypoint
│       └── pkg/               # Packages
│           ├── dataplane/     # eBPF program management
│           ├── policy/        # Policy CRUD operations
│           └── stats/         # Statistics collection
├── docs/                      # Documentation
├── tests/                     # Test suites
└── scripts/                   # Build and deployment scripts
```

### Build Commands

```bash
make help              # Show all available targets
make bpf               # Generate eBPF Go bindings
make agent             # Build the agent binary
make test              # Run unit tests
make test-integration  # Run integration tests
make clean             # Clean build artifacts
make fmt               # Format Go code
make lint              # Run linters
make install           # Install to /usr/local/bin
```

### Testing Traffic

```bash
# Terminal 1: Start agent
sudo ./bin/microsegment-agent --interface lo --log-level debug

# Terminal 2: Generate traffic
ping 127.0.0.1
curl http://127.0.0.1:8080

# Terminal 3: Monitor eBPF logs
sudo cat /sys/kernel/debug/tracing/trace_pipe
```

## 🎯 Roadmap

### ✅ Phase 1: Data Plane (Weeks 1-2)
- [x] eBPF session tracking (LRU_HASH)
- [x] 5-tuple policy matching
- [x] Policy enforcement (ALLOW/DENY/LOG)
- [x] Flow events and statistics
- [ ] Performance optimization (<10μs)

### 🚧 Phase 2: Control Plane (Week 3)
- [ ] RESTful API service
- [ ] Policy management (CRUD)
- [ ] gRPC communication with data plane
- [ ] PostgreSQL persistence

### 📅 Phase 3: Label System (Week 4)
- [ ] Workload auto-discovery (containers/processes)
- [ ] Auto-tagging engine (Role/App/Env/Location)
- [ ] Label-driven policy matching
- [ ] Flow data collection

### 📅 Phase 4: Visualization (Week 5)
- [ ] Application dependency mapping
- [ ] React + D3.js web UI
- [ ] Interactive topology graph
- [ ] Real-time flow analytics

### 📅 Phase 5: Intelligence (Week 6)
- [ ] Learning mode (traffic pattern observation)
- [ ] Auto policy generation
- [ ] Anomaly detection
- [ ] Policy recommendations

### 📅 Phase 6: Production Ready (Weeks 7-8)
- [ ] Comprehensive testing
- [ ] Performance benchmarks
- [ ] Documentation
- [ ] Docker/K8s deployment

## 🔬 Technical Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Data Plane | eBPF + TC | Kernel-level packet filtering |
| User Space | Go + Cilium eBPF | eBPF program management |
| Control Plane | Go + gRPC | Policy and label management |
| Database | PostgreSQL | Policy persistence |
| Time Series | InfluxDB | Flow data storage |
| Frontend | React + D3.js | Visualization dashboard |
| Container | Docker + K8s | Deployment platform |

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under GPL 2.0 OR BSD-3-Clause - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [Illumio](https://www.illumio.com/) and 蔷薇灵动
- Built with [Cilium eBPF](https://github.com/cilium/ebpf)
- Architecture influenced by [NeuVector](https://github.com/neuvector/neuvector) and [ZFW](https://github.com/netfoundry/zfw)

## 📧 Contact

- Project Link: [https://github.com/yourusername/ebpf-based-microsegment](https://github.com/yourusername/ebpf-based-microsegment)
- Documentation: [https://ebpf-microsegment.readthedocs.io](https://ebpf-microsegment.readthedocs.io)

---

Made with ❤️ and eBPF

## Process Monitoring (New Feature)

### Overview

The system now includes eBPF tracepoint-based process monitoring that captures process execution events in real-time. This feature solves the short-lived process problem and provides process context for network traffic analysis.

### Key Features

- **Real-time Process Capture**: Hooks into `sched_process_exec` tracepoint
- **Low Overhead**: ~1-2μs per exec event, negligible CPU impact
- **Container-Aware**: Extracts container ID for Docker/Kubernetes workloads
- **Dual-Layer Cache**: Kernel-side LRU map + userspace cache
- **Short-Lived Process Support**: Captures curl, wget, scripts before they exit

### Architecture

```
Process Exec → Tracepoint → eBPF Handler
                               ├─→ Cache to Map (TC/XDP fast lookup)
                               └─→ Ring Buffer (Userspace processing)
```

### Usage

**Build**:
```bash
make bpf
```

**Test**:
```bash
sudo ./tests/test_process_monitor.sh
```

**Documentation**:
- [Process Monitoring Guide](docs/process-monitoring.md)
- Source: `src/bpf/process_monitor.bpf.c`
- Headers: `src/bpf/headers/process_monitor.h`

### Technical Details

- **Maps**:
  - `process_info_map`: LRU Hash (10000 entries, ~820KB)
  - `process_events`: Ring Buffer (256KB, ~2700 events)

- **Data Captured**:
  - Process ID (PID)
  - Command name (comm, 16 bytes)
  - Execution timestamp
  - Container ID (extracted in userspace)

- **Performance**:
  - Memory: ~1.1MB kernel + cache in userspace
  - CPU: < 0.1% at 100 execs/sec
  - Latency: 1-2μs per event

### Integration

- **Issue #47**: TC/XDP programs query `process_info_map` for process context ✅ **COMPLETED**
  - Wildcard policies now support `process_name` field
  - Process-aware policy matching with priority (process policies > network policies)
  - Flow events enriched with process information (PID, comm, container ID)
- **Issue #48**: ProcessMonitor daemon consumes ring buffer events ✅ **COMPLETED**
  - LRU+TTL cache (20000 entries, 5min TTL)
  - Process path resolution via /proc/<pid>/exe
  - GetProcessInfo(pid) API for FlowCollector
  - Background cleanup of expired cache entries
- **Issue #49**: FlowCollector and PolicyManager extensions ✅ **COMPLETED**
  - FlowEvent parsing of process fields (PID, process_name, container_id, exec_time)
  - Flow structure enriched with process context
  - ProcessMonitor integration for full process path enrichment
  - Process info available in flow events for analysis and policy decisions
- **Issue #50**: Security hardening implementation ✅ **COMPLETED**
  - Process path legitimacy validation (system vs suspicious directories)
  - Suspicious process detection (deleted executables, hidden files, privilege escalation)
  - Structured security alert generation with severity levels (INFO, WARNING, CRITICAL)
  - Alert rate limiting to prevent flooding (10 alerts per minute)
  - Real-time security monitoring integrated with FlowCollector
  - LogAlertHandler for immediate security alert logging
- **Future**: Advanced security features (K8s Pod metadata, process hash verification)

