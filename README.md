# eBPF Microsegmentation

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![Linux Kernel](https://img.shields.io/badge/Linux-6.x+-FCC624?logo=linux&logoColor=black)](https://kernel.org/)
[![eBPF](https://img.shields.io/badge/eBPF-TC%20Hook-orange)](https://ebpf.io/)

**[English](README.md) | [中文](README_CN.md)**

A high-performance, kernel-native microsegmentation solution using eBPF for fine-grained network traffic control in cloud-native environments.

## Overview

eBPF Microsegmentation provides network isolation and access control at the kernel level, delivering sub-microsecond latency for packet processing. The system consists of:

- **Data Plane**: eBPF programs attached to TC (Traffic Control) hooks for line-rate packet filtering
- **Control Plane**: Go-based agent and server components for policy management and monitoring
- **Web UI**: React-based dashboard for visualization and management

## Features

- **High Performance**: Hot path latency < 1μs, cold path < 20μs
- **Session Tracking**: LRU-based connection tracking with 100K concurrent sessions
- **Multi-tier Policy Matching**: Exact match + Wildcard (CIDR/port ranges) + Default policy
- **Per-CPU Statistics**: Lock-free counters with zero CPU contention
- **Real-time Events**: Ring buffer for flow events (new connections, denials)
- **RESTful API**: Full CRUD operations for policy management
- **gRPC Communication**: Agent-Server communication with Protocol Buffers
- **TCP State Machine**: Connection state tracking for stateful filtering

## Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                     User / External Systems                     │
│                  (Web UI / API / Orchestrators)                 │
└─────────────────────────────┬──────────────────────────────────┘
                              │ HTTP/gRPC
┌─────────────────────────────▼──────────────────────────────────┐
│                    Control Plane (User Space)                   │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────────┐ │
│  │    Server    │  │    Agent     │  │   Policy Manager      │ │
│  │  (gRPC API)  │◄─┤  (eBPF Mgr)  │──┤   + DataPlane Mgr     │ │
│  └──────────────┘  └──────────────┘  └───────────────────────┘ │
└─────────────────────────────┬──────────────────────────────────┘
                              │ Cilium eBPF Library
┌─────────────────────────────▼──────────────────────────────────┐
│                     Data Plane (Kernel Space)                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  TC eBPF Program                                         │  │
│  │  • Packet parsing (5-tuple)  • Session tracking (LRU)    │  │
│  │  • Policy matching (Hash)    • Statistics (Per-CPU)      │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  eBPF Maps: session_map | policy_map | stats_map | ...   │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Linux kernel 6.x+ (with eBPF support)
- Go 1.21+
- Clang/LLVM (for eBPF compilation)
- PostgreSQL 14+ (for server)
- Node.js 18+ (for web UI)

### Installation

```bash
# Clone the repository
git clone https://github.com/your-org/ebpf-based-microsegment.git
cd ebpf-based-microsegment

# Install dependencies
make deps

# Build all components
make all

# Or build specific components
make agent    # Build agent only
make server   # Build server only
```

### Running

**Start the Server:**
```bash
# Initialize database
./src/server/scripts/migrate.sh

# Start server
./bin/microsegment-server --config config/server.yaml
```

**Start the Agent:**
```bash
# Requires root privileges for eBPF
sudo ./bin/microsegment-agent --interface eth0 --server localhost:50051
```

**Start Web UI:**
```bash
cd web
npm install
npm run dev
```

### Quick Demo

```bash
# Start all components (server + agent + web)
./start-all.sh

# Access Web UI at http://localhost:5173
# API available at http://localhost:8080
```

## Configuration

### Agent Configuration

| Flag | Description | Default |
|------|-------------|---------|
| `--interface` | Network interface to attach eBPF | `eth0` |
| `--server` | Server gRPC address | `localhost:50051` |
| `--api-addr` | Local API listen address | `127.0.0.1:8080` |
| `--log-level` | Log level (debug/info/warn/error) | `info` |

### Server Configuration

Configuration via `config/server.yaml`:

```yaml
server:
  grpc_port: 50051
  http_port: 8081

database:
  host: localhost
  port: 5432
  user: microsegment_user
  password: secret
  name: microsegment
```

## API Reference

### Policy Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/policies` | Create policy |
| GET | `/api/v1/policies` | List all policies |
| GET | `/api/v1/policies/:id` | Get policy by ID |
| PUT | `/api/v1/policies/:id` | Update policy |
| DELETE | `/api/v1/policies/:id` | Delete policy |

### Statistics

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/stats` | Get all statistics |
| GET | `/api/v1/stats/packets` | Get packet statistics |
| GET | `/api/v1/stats/policies` | Get policy hit statistics |

### Health Check

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Simple health check |
| GET | `/api/v1/status` | Detailed system status |

### Example: Create Policy

```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": 1001,
    "src_ip": "10.0.0.0/24",
    "dst_ip": "192.168.1.100",
    "dst_port": 443,
    "protocol": "tcp",
    "action": "allow"
  }'
```

## Build Options

```bash
# Production build (optimized, all features)
make build-production

# Debug build (with debug logging)
make build-debug

# Minimal build (no NAT/fragment handling)
make build-minimal

# Show current configuration
make show-config
```

### eBPF Feature Flags

| Flag | Description | Default |
|------|-------------|---------|
| `DEBUG_MODE` | Enable eBPF debug logging | 0 |
| `ENABLE_IP_FRAGMENT_HANDLING` | Handle IP fragments | 1 |
| `ENABLE_NAT_SUPPORT` | NAT detection support | 1 |

## Performance

| Metric | Value | Notes |
|--------|-------|-------|
| Hot path latency | < 1μs | 99%+ packets (existing sessions) |
| Cold path latency | 5-20μs | New sessions with policy lookup |
| Exact policy match | ~0.1μs | O(1) hash lookup |
| Wildcard policy match | 2-20μs | Index scan with CIDR matching |
| Max concurrent sessions | 100K | LRU auto-eviction |
| Max policies | 10K exact + 1K wildcard | Configurable |

## Project Structure

```
.
├── src/
│   ├── agent/          # Agent component (eBPF management)
│   │   ├── cmd/        # Entry point
│   │   └── pkg/        # Packages (api, dataplane, policy)
│   ├── bpf/            # eBPF C programs
│   └── server/         # Server component (policy server)
├── api/
│   └── proto/          # Protocol Buffer definitions
├── web/                # React Web UI
├── config/             # Configuration files
├── deploy/             # Deployment scripts (systemd, docker)
├── docs/               # Documentation
└── tests/              # Integration tests
```

## Testing

```bash
# Run unit tests
make test

# Run integration tests (requires root)
sudo make test-integration

# Run specific test
cd src/agent && go test -v ./pkg/dataplane/...
```

## Deployment

### Systemd

```bash
# Install systemd services
sudo ./deploy/scripts/install-systemd.sh

# Start services
sudo systemctl start microsegment-server
sudo systemctl start microsegment-agent
```

### Docker

```bash
# Build and run with docker-compose
docker-compose up -d
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Cilium eBPF Library](https://github.com/cilium/ebpf) - Go library for eBPF
- [libbpf](https://github.com/libbpf/libbpf) - eBPF library
- [NeuVector](https://github.com/neuvector/neuvector) - Reference for network security concepts
