# Configuration Reference

Complete configuration reference for Agent and Server components.

## Overview

Both components support three configuration layers (priority: low → high):
1. **Default values** - Built into code
2. **YAML config file** - Specified via `--config` / `-c`
3. **Environment variables** - Prefix: `MICROSEGMENT_`

---

## Configuration Files Location

| File | Purpose | Environment |
|------|---------|-------------|
| `/config/agent.yaml` | Agent production config | Production |
| `/config/server.yaml` | Server production config | Production |
| `/config/agent-k8s-example.yaml` | K8s deployment example | Kubernetes |
| `/deploy/config/systemd/*.yaml` | Systemd deployment | Bare metal |

---

# Agent Configuration

## Complete Example

```yaml
# Basic settings
interface: eth0
log_level: info
stats_interval: 30
mode: agent-server

# REST API server
api:
  enabled: true
  host: "127.0.0.1"
  port: 8081
  enable_cors: true

# Server connection (required for agent-server mode)
server:
  server_addr: "localhost:9090"
  agent_id: ""
  batch_size: 100
  batch_timeout: 5s
  reconnect_interval: 30s
  max_retries: 3
  retry_base_delay: 1s
  retry_max_delay: 30s

# Flow collection
flow:
  storage_path: "./data/flows.db"
  flow_timeout: 5m
  cleanup_interval: 1m
  retention_days: 7

# Data plane settings
dataplane:
  mode: auto
  prefer_xdp: true
  allow_generic_xdp: true
  enable_nat: false
  enable_fragment: false

# Kubernetes integration
kubernetes:
  enabled: false
  config_mode: auto
  kubeconfig_path: ""
  api_server: ""
  qps: 5.0
  burst: 10
  timeout: 30
  health_check:
    enabled: true
    interval: 30
    timeout: 5
  namespaces:
    include: []
    exclude:
      - kube-system
      - kube-public
      - kube-node-lease
```

## Parameter Reference

### Basic Settings

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `interface` | string | `eth0` | Network interface to attach eBPF programs (required) |
| `log_level` | string | `info` | Log level: `debug`/`info`/`warn`/`error` |
| `stats_interval` | int | `30` | Statistics print interval (seconds) |
| `mode` | string | `agent-server` | Run mode: `agent-server` / `standalone` |

### API Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `api.enabled` | bool | `true` | Enable REST API server |
| `api.host` | string | `127.0.0.1` | API server bind address |
| `api.port` | int | `8080` | API server port |
| `api.enable_cors` | bool | `true` | Enable CORS support |

### Server Connection

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `server.server_addr` | string | - | gRPC server address (required in agent-server mode) |
| `server.agent_id` | string | auto | Agent unique ID (auto-generated if empty) |
| `server.batch_size` | int | `100` | Flow events batch size |
| `server.batch_timeout` | duration | `5s` | Max batch wait time |
| `server.reconnect_interval` | duration | `30s` | Reconnection interval on failure |
| `server.max_retries` | int | `3` | Max retry attempts |
| `server.retry_base_delay` | duration | `1s` | Exponential backoff base delay |
| `server.retry_max_delay` | duration | `30s` | Max retry delay |

### Flow Collection

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `flow.storage_path` | string | `./data/flows.db` | SQLite DB path (standalone mode) |
| `flow.flow_timeout` | duration | `5m` | Inactive flow timeout |
| `flow.cleanup_interval` | duration | `1m` | Cleanup interval |
| `flow.retention_days` | int | `7` | Data retention period |

### Data Plane

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `dataplane.mode` | string | `auto` | Mode: `auto`/`xdp-native`/`xdp-generic`/`tc` |
| `dataplane.prefer_xdp` | bool | `true` | Prefer XDP in auto mode |
| `dataplane.allow_generic_xdp` | bool | `true` | Allow generic XDP fallback |
| `dataplane.enable_nat` | bool | `true` | Enable NAT support |
| `dataplane.enable_fragment` | bool | `true` | Enable IP fragment handling |

**Data Plane Mode Comparison**:

| Mode | Performance | Compatibility | Notes |
|------|-------------|---------------|-------|
| `xdp-native` | Best | Low | Requires driver support |
| `xdp-generic` | Medium | High | Kernel fallback |
| `tc` | Medium | Best | Works on all interfaces |
| `auto` | Adaptive | Best | Auto-selects best available |

### Kubernetes Integration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `kubernetes.enabled` | bool | `false` | Enable K8s integration |
| `kubernetes.config_mode` | string | `auto` | Mode: `auto`/`in-cluster`/`kubeconfig` |
| `kubernetes.kubeconfig_path` | string | `~/.kube/config` | Kubeconfig file path |
| `kubernetes.api_server` | string | - | K8s API server URL (optional) |
| `kubernetes.qps` | float32 | `5.0` | API client QPS limit |
| `kubernetes.burst` | int | `10` | API client burst size |
| `kubernetes.timeout` | int | `30` | Request timeout (seconds) |
| `kubernetes.namespaces.include` | []string | `[]` | Include namespaces (empty = all) |
| `kubernetes.namespaces.exclude` | []string | `[kube-system,...]` | Exclude namespaces |

---

# Server Configuration

## Complete Example

```yaml
# HTTP server
server:
  host: "0.0.0.0"
  port: 8080

# gRPC server
grpc:
  host: "0.0.0.0"
  port: 9090

# PostgreSQL database
database:
  host: "localhost"
  port: 5432
  user: "microsegment_user"
  password: "secret"
  dbname: "microsegment"
  sslmode: "disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: "5m"

# Logging
log:
  level: "info"
  format: "json"
```

## Parameter Reference

### HTTP Server

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `server.host` | string | `0.0.0.0` | HTTP server bind address |
| `server.port` | int | `8080` | HTTP API port |

### gRPC Server

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `grpc.host` | string | `0.0.0.0` | gRPC server bind address |
| `grpc.port` | int | `9090` | gRPC server port |

### Database (PostgreSQL)

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `database.host` | string | `localhost` | PostgreSQL host |
| `database.port` | int | `5432` | PostgreSQL port |
| `database.user` | string | `microsegment_user` | Database username |
| `database.password` | string | `secret` | Database password |
| `database.dbname` | string | `microsegment` | Database name |
| `database.sslmode` | string | `disable` | SSL mode: `disable`/`require`/`verify-ca`/`verify-full` |
| `database.max_open_conns` | int | `25` | Max open connections |
| `database.max_idle_conns` | int | `5` | Max idle connections |
| `database.conn_max_lifetime` | duration | `5m` | Connection max lifetime |

### Logging

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `log.level` | string | `info` | Log level: `debug`/`info`/`warn`/`error` |
| `log.format` | string | `json` | Log format: `json`/`text` |

---

# Environment Variables

All configuration parameters can be overridden via environment variables.

## Naming Convention

```
MICROSEGMENT_<SECTION>_<PARAMETER>
```

Nested parameters use underscores: `server.server_addr` → `MICROSEGMENT_SERVER_SERVER_ADDR`

## Agent Environment Variables

```bash
# Basic
export MICROSEGMENT_INTERFACE=eth1
export MICROSEGMENT_LOG_LEVEL=debug
export MICROSEGMENT_MODE=standalone

# API
export MICROSEGMENT_API_ENABLED=true
export MICROSEGMENT_API_HOST=0.0.0.0
export MICROSEGMENT_API_PORT=8081

# Server connection
export MICROSEGMENT_SERVER_SERVER_ADDR=10.0.1.100:9090
export MICROSEGMENT_SERVER_AGENT_ID=agent-prod-01
export MICROSEGMENT_SERVER_BATCH_SIZE=200

# Data plane
export MICROSEGMENT_DATAPLANE_MODE=xdp-native
export MICROSEGMENT_DATAPLANE_ENABLE_NAT=true

# Kubernetes
export MICROSEGMENT_KUBERNETES_ENABLED=true
export MICROSEGMENT_KUBERNETES_CONFIG_MODE=in-cluster
```

## Server Environment Variables

```bash
# HTTP server
export MICROSEGMENT_SERVER_HOST=0.0.0.0
export MICROSEGMENT_SERVER_PORT=8080

# gRPC server
export MICROSEGMENT_GRPC_HOST=0.0.0.0
export MICROSEGMENT_GRPC_PORT=9090

# Database
export MICROSEGMENT_DATABASE_HOST=postgres.default.svc
export MICROSEGMENT_DATABASE_PORT=5432
export MICROSEGMENT_DATABASE_USER=admin
export MICROSEGMENT_DATABASE_PASSWORD=secure-password
export MICROSEGMENT_DATABASE_DBNAME=microsegment
export MICROSEGMENT_DATABASE_SSLMODE=require

# Logging
export MICROSEGMENT_LOG_LEVEL=info
export MICROSEGMENT_LOG_FORMAT=json
```

---

# Run Mode Comparison

| Feature | agent-server | standalone |
|---------|--------------|------------|
| Server connection | Required | Not needed |
| Flow storage | PostgreSQL (via Server) | Local SQLite |
| Policy source | Server (gRPC) | Local API |
| Use case | Production/Multi-node | Debug/Single-node |

---

# Validation Rules

## Agent

- `interface`: Required, non-empty
- `log_level`: Must be `debug`/`info`/`warn`/`error`
- `mode`: Must be `agent-server` or `standalone`
- `server.server_addr`: Required when `mode=agent-server`
- `dataplane.mode`: Must be `auto`/`xdp-native`/`xdp-generic`/`tc`
- `kubernetes.config_mode`: Must be `auto`/`in-cluster`/`kubeconfig`
- `kubernetes.health_check.timeout` < `kubernetes.health_check.interval`

## Server

- `server.port`: Valid port range (1-65535)
- `grpc.port`: Valid port range (1-65535)
- `database.*`: Valid PostgreSQL connection parameters

---

*Last updated: 2024-12-24*
