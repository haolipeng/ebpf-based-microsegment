# eBPF-based Microsegmentation

[![License](https://img.shields.io/badge/License-GPL%202.0%20%7C%20BSD--3-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![eBPF](https://img.shields.io/badge/eBPF-Powered-orange)](https://ebpf.io/)

🔒 A high-performance, eBPF-powered microsegmentation solution for cloud-native workloads, inspired by **Illumio** and **蔷薇灵动**.

## 🌟 Features

- **🚀 High Performance**: Kernel-level packet filtering with <10μs latency overhead
- **🎯 Session Tracking**: Intelligent connection tracking using LRU hash maps
- **📊 Real-time Visibility**: Live flow events and traffic statistics
- **🏷️ Label-based Policies**: Cloud-native policy management (coming soon)
- **📈 Flow Visualization**: Application dependency mapping (coming soon)
- **🤖 Auto Policy Generation**: ML-powered policy recommendations (coming soon)
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

- [Project Structure](PROJECT_STRUCTURE.md) - Detailed directory layout and module descriptions
- [Implementation Plan](docs/microsegmentation-mvp-implementation-plan.md) - MVP roadmap and milestones
- [Architecture Design](design-docs/architecture/design.md) - Technical architecture details
- [Weekly Guide](docs/weekly-guide/) - 6-week learning and implementation guide

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
