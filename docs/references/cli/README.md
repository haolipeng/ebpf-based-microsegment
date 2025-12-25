# CLI Reference

Command-line interface reference for Agent and Server components.

## Overview

| Component | Framework | Binary |
|-----------|-----------|--------|
| Agent | Cobra | `microsegment-agent` |
| Server | Go flag | `microsegment-server` |

---

# Agent CLI

## Synopsis

```bash
microsegment-agent [flags]
```

## Description

eBPF-based microsegmentation agent that runs on each node to enforce network policies.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--config` | `-c` | string | - | Path to YAML configuration file |
| `--help` | `-h` | - | - | Show help message |

## Usage Examples

```bash
# Start with configuration file
./bin/microsegment-agent --config config/agent.yaml

# Short form
./bin/microsegment-agent -c config/agent.yaml

# Start with defaults (uses built-in defaults)
./bin/microsegment-agent

# Override with environment variables
MICROSEGMENT_INTERFACE=eth1 \
MICROSEGMENT_LOG_LEVEL=debug \
./bin/microsegment-agent -c config/agent.yaml

# Standalone mode for debugging
MICROSEGMENT_MODE=standalone ./bin/microsegment-agent
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Configuration error |
| 2 | eBPF load failure |
| 3 | Network interface error |

---

# Server CLI

## Synopsis

```bash
microsegment-server [flags]
```

## Description

Central control plane server that manages policies, collects flow data, and provides REST API.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `config/server.yaml` | Path to YAML configuration file |
| `--help` | - | - | Show help message |

## Usage Examples

```bash
# Start with configuration file
./bin/microsegment-server --config config/server.yaml

# Start with default config path
./bin/microsegment-server

# Override database settings via environment
MICROSEGMENT_DATABASE_HOST=postgres.local \
MICROSEGMENT_DATABASE_PASSWORD=secret \
./bin/microsegment-server --config config/server.yaml
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Configuration error |
| 2 | Database connection failure |
| 3 | Port binding failure |

---

# Environment Variables

Both components support environment variable overrides with `MICROSEGMENT_` prefix.

## Naming Convention

```
MICROSEGMENT_<SECTION>_<PARAMETER>
```

Example: `server.server_addr` → `MICROSEGMENT_SERVER_SERVER_ADDR`

## Common Variables

### Agent

```bash
# Basic configuration
MICROSEGMENT_INTERFACE=eth0          # Network interface
MICROSEGMENT_LOG_LEVEL=info          # debug/info/warn/error
MICROSEGMENT_MODE=agent-server       # agent-server/standalone

# API server
MICROSEGMENT_API_ENABLED=true
MICROSEGMENT_API_HOST=127.0.0.1
MICROSEGMENT_API_PORT=8081

# Server connection
MICROSEGMENT_SERVER_SERVER_ADDR=localhost:9090
MICROSEGMENT_SERVER_AGENT_ID=my-agent
MICROSEGMENT_SERVER_BATCH_SIZE=100

# Data plane
MICROSEGMENT_DATAPLANE_MODE=auto
MICROSEGMENT_DATAPLANE_ENABLE_NAT=true

# Kubernetes
MICROSEGMENT_KUBERNETES_ENABLED=false
MICROSEGMENT_KUBERNETES_CONFIG_MODE=auto
```

### Server

```bash
# HTTP server
MICROSEGMENT_SERVER_HOST=0.0.0.0
MICROSEGMENT_SERVER_PORT=8080

# gRPC server
MICROSEGMENT_GRPC_HOST=0.0.0.0
MICROSEGMENT_GRPC_PORT=9090

# Database
MICROSEGMENT_DATABASE_HOST=localhost
MICROSEGMENT_DATABASE_PORT=5432
MICROSEGMENT_DATABASE_USER=microsegment_user
MICROSEGMENT_DATABASE_PASSWORD=secret
MICROSEGMENT_DATABASE_DBNAME=microsegment

# Logging
MICROSEGMENT_LOG_LEVEL=info
MICROSEGMENT_LOG_FORMAT=json
```

---

# Quick Start Commands

## Development Setup

```bash
# 1. Start PostgreSQL (required for Server)
docker run -d --name postgres \
  -e POSTGRES_USER=microsegment_user \
  -e POSTGRES_PASSWORD=secret \
  -e POSTGRES_DB=microsegment \
  -p 5432:5432 postgres:14

# 2. Start Server
./bin/microsegment-server --config config/server.yaml

# 3. Start Agent (requires root for eBPF)
sudo ./bin/microsegment-agent --config config/agent.yaml
```

## Quick Test

```bash
# Check server health
curl http://localhost:8080/health

# List policies
curl http://localhost:8080/api/v1/policies

# Check agent status (if API enabled)
curl http://localhost:8081/api/v1/status
```

## Systemd Service

```bash
# Install services
sudo cp deploy/systemd/*.service /etc/systemd/system/
sudo cp deploy/config/systemd/*.yaml /etc/microsegment/

# Enable and start
sudo systemctl enable microsegment-server microsegment-agent
sudo systemctl start microsegment-server microsegment-agent

# Check status
sudo systemctl status microsegment-agent
sudo journalctl -u microsegment-agent -f
```

---

# Troubleshooting

## Common Issues

### Agent: "Failed to load eBPF program"

```bash
# Check kernel version (requires 5.4+)
uname -r

# Check eBPF support
ls /sys/fs/bpf/

# Run with debug logging
MICROSEGMENT_LOG_LEVEL=debug ./bin/microsegment-agent -c config/agent.yaml
```

### Agent: "Interface not found"

```bash
# List available interfaces
ip link show

# Set correct interface
MICROSEGMENT_INTERFACE=ens192 ./bin/microsegment-agent -c config/agent.yaml
```

### Server: "Database connection failed"

```bash
# Test database connectivity
psql -h localhost -U microsegment_user -d microsegment

# Check environment variables
echo $MICROSEGMENT_DATABASE_HOST
echo $MICROSEGMENT_DATABASE_PASSWORD
```

### Server: "Port already in use"

```bash
# Find process using the port
lsof -i :8080
lsof -i :9090

# Use different port
MICROSEGMENT_SERVER_PORT=8090 ./bin/microsegment-server
```

---

# Build from Source

```bash
# Build all components
make all

# Build agent only
make agent

# Build server only
make server

# Build with debug symbols
make build-debug

# Output binaries
ls -la bin/
# bin/microsegment-agent
# bin/microsegment-server
```

---

*Last updated: 2024-12-24*
