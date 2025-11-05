# Microsegmentation Server (MVP)

The central server component for the eBPF-based microsegmentation system. This is a **Minimum Viable Product (MVP)** implementation that provides core functionality for Agent-Server architecture.

## Overview

The server acts as a central control plane that:
- Receives flow events from multiple agents via gRPC streaming
- Stores flow data in PostgreSQL
- Distributes policies to agents
- Manages agent registration and heartbeats
- Provides HTTP API for querying flows and managing policies

## Architecture

```
┌─────────────────────────────────────────────┐
│          Microsegment Server                │
│                                             │
│  ┌────────────────────────────────────┐    │
│  │  HTTP API (:8080)                  │    │
│  │  - GET /health                     │    │
│  │  - GET /api/v1/agents              │    │
│  │  - GET /api/v1/flows               │    │
│  │  - GET /api/v1/policies            │    │
│  └────────────────────────────────────┘    │
│                                             │
│  ┌────────────────────────────────────┐    │
│  │  gRPC Server (:9090)               │    │
│  │  - FlowService                     │    │
│  │  - PolicyService                   │    │
│  │  - AgentService                    │    │
│  └────────────────────────────────────┘    │
│                                             │
│  ┌────────────────────────────────────┐    │
│  │  PostgreSQL Storage                │    │
│  │  - flows table                     │    │
│  │  - policies table                  │    │
│  │  - agents table                    │    │
│  └────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

## Prerequisites

### Required Software
- **Go 1.22+** - For building the server
- **PostgreSQL 14+** - For data storage
- **protoc 3.12+** - For Protocol Buffers (if rebuilding)

### Install PostgreSQL

**Ubuntu/Debian**:
```bash
sudo apt-get update
sudo apt-get install -y postgresql-14
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

**macOS**:
```bash
brew install postgresql@14
brew services start postgresql@14
```

## Quick Start

### 1. Setup Database

Create database and user:
```bash
sudo -u postgres psql <<EOF
CREATE DATABASE microsegment;
CREATE USER microsegment_user WITH PASSWORD 'secret';
GRANT ALL PRIVILEGES ON DATABASE microsegment TO microsegment_user;
\q
EOF
```

The server will automatically create tables on first run.

### 2. Configure Server

Edit `config/server.yaml`:
```yaml
server:
  host: "0.0.0.0"
  port: 8080

grpc:
  host: "0.0.0.0"
  port: 9090

database:
  host: "localhost"
  port: 5432
  user: "microsegment_user"
  password: "secret"
  dbname: "microsegment"
  sslmode: "disable"

log:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
```

### 3. Build Server

From project root:
```bash
make server
```

This creates `bin/microsegment-server`.

### 4. Run Server

```bash
./bin/microsegment-server --config src/server/config/server.yaml
```

Or with default config:
```bash
cd src/server
../../bin/microsegment-server
```

### 5. Verify Server

Check health:
```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "service": "microsegment-server",
  "version": "1.0.0-mvp"
}
```

## API Endpoints

### HTTP API (Port 8080)

**Health Check**:
```bash
GET /health
```

**List Agents**:
```bash
GET /api/v1/agents
```

**List Policies**:
```bash
GET /api/v1/policies
```

**Query Flows** (placeholder):
```bash
GET /api/v1/flows
```

### gRPC API (Port 9090)

See [proto definitions](../../proto/README.md) for details:
- `FlowService.ReportFlowEvents` - Accept streaming flow events
- `FlowService.QueryFlows` - Query historical flows
- `PolicyService.SyncPolicies` - Full policy synchronization
- `PolicyService.SubscribePolicies` - Subscribe to policy updates
- `AgentService.RegisterAgent` - Register new agent
- `AgentService.Heartbeat` - Periodic health check

## Database Schema

The server automatically creates the following tables:

### flows
Stores flow events from agents:
- `id` - Auto-increment primary key
- `timestamp_ns` - Flow timestamp in nanoseconds
- `src_ip`, `dst_ip` - Source/destination IPs
- `src_port`, `dst_port` - Ports
- `protocol` - TCP/UDP/ICMP
- `packet_count`, `byte_count` - Traffic statistics
- `agent_id` - Reporting agent
- `source_labels`, `dest_labels` - JSONB labels

### policies
Stores policy rules:
- `rule_id` - Policy ID
- `src_ip`, `dst_ip` - IP ranges (CIDR)
- `protocol` - Network protocol
- `action` - ALLOW/DENY/LOG
- `priority` - Rule priority
- `source_labels`, `dest_labels` - JSONB labels

### agents
Stores registered agents:
- `agent_id` - Unique agent identifier
- `hostname` - Agent hostname
- `version` - Agent version
- `ip_addresses` - Agent IP addresses
- `last_heartbeat` - Last heartbeat timestamp
- `status` - Agent status (active/inactive)

## Configuration

### Environment Variables

Override config with environment variables:
```bash
export MICROSEGMENT_DATABASE_HOST=postgres-server
export MICROSEGMENT_DATABASE_PASSWORD=production-secret
export MICROSEGMENT_LOG_LEVEL=debug

./bin/microsegment-server
```

### Connection Pool Settings

Adjust for your workload:
```yaml
database:
  max_open_conns: 25     # Max concurrent connections
  max_idle_conns: 5      # Idle connections in pool
  conn_max_lifetime: "5m" # Connection lifetime
```

## Development

### Build from Source

```bash
# Build server
make server

# Build with debug info
cd src/server
go build -gcflags="all=-N -l" -o ../../bin/microsegment-server ./cmd
```

### Run Tests

```bash
cd src/server
go test ./...
```

### Code Structure

```
src/server/
├── cmd/
│   └── main.go              # Server entry point
├── pkg/
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── storage/
│   │   ├── postgres.go      # Database connection
│   │   ├── flow_storage.go  # Flow data persistence
│   │   ├── policy_storage.go # Policy data persistence
│   │   └── agent_storage.go # Agent data persistence
│   ├── grpc/
│   │   ├── flow_service.go  # FlowService implementation
│   │   ├── policy_service.go # PolicyService implementation
│   │   └── agent_service.go # AgentService implementation
│   └── api/
│       └── (HTTP handlers in main.go for MVP)
├── config/
│   └── server.yaml          # Example configuration
└── migrations/              # Database migrations (future)
```

## MVP Limitations

This is a simplified MVP implementation. The following features are **NOT** implemented:

- ❌ TimescaleDB hypertables (using standard PostgreSQL)
- ❌ Advanced aggregation queries
- ❌ Dependency graph analysis
- ❌ Real-time policy update streaming (only initial sync)
- ❌ TLS/authentication
- ❌ Multi-tenancy
- ❌ Metrics and monitoring endpoints
- ❌ Database migrations framework

## Production Readiness

For production deployment, consider:

1. **Enable TLS** for gRPC and HTTPS
2. **Add authentication** (API keys, mTLS)
3. **Use TimescaleDB** for better time-series performance
4. **Implement connection pooling** tuning
5. **Add monitoring** (Prometheus metrics)
6. **Setup backups** for PostgreSQL
7. **Use managed PostgreSQL** (AWS RDS, GCP Cloud SQL)
8. **Implement rate limiting**
9. **Add request validation**
10. **Setup log aggregation** (ELK, Loki)

## Troubleshooting

### Server won't start

**Check database connection**:
```bash
psql -h localhost -U microsegment_user -d microsegment
```

**Check logs**:
```bash
./bin/microsegment-server --config config/server.yaml 2>&1 | grep ERROR
```

### Port already in use

Change ports in `config/server.yaml`:
```yaml
server:
  port: 8081  # HTTP API

grpc:
  port: 9091  # gRPC server
```

### Agent can't connect

**Check gRPC port**:
```bash
netstat -tulpn | grep 9090
```

**Test gRPC connection**:
```bash
grpcurl -plaintext localhost:9090 list
```

## Related Documentation

- [gRPC Protocol Definitions](../../proto/README.md)
- [Agent-Server Migration Plan](../../docs/agent-server-migration-plan.md)
- [Architecture Comparison](../../docs/architecture-comparison.md)

## Support

For issues or questions:
1. Check logs at INFO or DEBUG level
2. Review database schema with `\dt` in psql
3. Test gRPC with `grpcurl`
4. Open an issue in the project repository

---

**Version**: 1.0.0-mvp
**Status**: Minimum Viable Product
**Next Steps**: Implement Agent gRPC client for remote reporting
