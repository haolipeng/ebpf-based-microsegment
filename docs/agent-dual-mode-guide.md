# Agent Dual-Mode Operation Guide

This guide explains how to run the microsegmentation agent in both **standalone** and **agent-server** modes.

## Table of Contents

1. [Overview](#overview)
2. [Standalone Mode](#standalone-mode)
3. [Agent-Server Mode](#agent-server-mode)
4. [Configuration Reference](#configuration-reference)
5. [Migration Guide](#migration-guide)
6. [Troubleshooting](#troubleshooting)

---

## Overview

The microsegmentation agent supports two operation modes:

### Standalone Mode

- **Purpose**: Simple, single-node deployments
- **Storage**: Local SQLite database
- **Use Case**: Testing, development, small deployments
- **Pros**: Simple setup, no external dependencies
- **Cons**: No centralized management, limited scalability

```
┌─────────────────────┐
│  eBPF Flow Collector │
└──────────┬───────────┘
           │
           ▼
┌─────────────────────┐
│   LocalReporter     │
└──────────┬───────────┘
           │
           ▼
┌─────────────────────┐
│  SQLite Storage     │
└─────────────────────┘
```

### Agent-Server Mode

- **Purpose**: Multi-node, enterprise deployments
- **Storage**: Central PostgreSQL database (via server)
- **Use Case**: Production, multi-cluster environments
- **Pros**: Centralized management, scalability, policy sync
- **Cons**: Requires server infrastructure

```
┌─────────────────────┐
│  eBPF Flow Collector │
└──────────┬───────────┘
           │
           ▼
┌─────────────────────┐
│   GRPCReporter      │ ────┬──► Batch Queue (100 flows)
└──────────┬───────────┘     │
           │                 │
           ▼                 ▼
┌─────────────────────┬─────────────┐
│   AgentClient       │ gRPC Stream │
│   - Registration    │             │
│   - Heartbeat (30s) │             │
│   - Policy Sync     │             │
└─────────────────────┴──────┬──────┘
                             │
                             ▼
                  ┌──────────────────┐
                  │ Microsegment     │
                  │ Server           │
                  │ (PostgreSQL)     │
                  └──────────────────┘
```

---

## Standalone Mode

### Quick Start

1. **Create configuration file**:

   ```bash
   mkdir -p /etc/microsegment
   cp config/agent-standalone.yaml /etc/microsegment/agent.yaml
   ```

2. **Edit configuration** (if needed):

   ```yaml
   mode: standalone
   interface: lo  # or your network interface
   log_level: info

   storage:
     path: /var/lib/microsegment/flows.db

   api:
     enabled: true
     host: 127.0.0.1
     port: 8080
   ```

3. **Run the agent**:

   ```bash
   sudo ./bin/microsegment-agent --config /etc/microsegment/agent.yaml
   ```

4. **Verify operation**:

   ```bash
   # Check API health
   curl http://localhost:8080/health

   # View flows
   curl http://localhost:8080/api/v1/flows

   # Check logs
   journalctl -u microsegment-agent -f
   ```

### Data Storage

Flows are stored in SQLite at the configured path (default: `/var/lib/microsegment/flows.db`).

**View flows directly**:

```bash
sqlite3 /var/lib/microsegment/flows.db "SELECT * FROM flows LIMIT 10;"
```

**Backup flows**:

```bash
cp /var/lib/microsegment/flows.db /backup/flows-$(date +%Y%m%d).db
```

---

## Agent-Server Mode

### Prerequisites

- Microsegmentation server deployed and accessible
- Network connectivity to server's gRPC port (default: 9090)
- Agent version 1.1.0 or higher

### Quick Start

1. **Create configuration file**:

   ```bash
   mkdir -p /etc/microsegment
   cp config/agent-server.yaml /etc/microsegment/agent.yaml
   ```

2. **Edit configuration**:

   ```yaml
   mode: agent-server
   interface: eth0  # your network interface
   log_level: info

   agent_server:
     enabled: true
     server_addr: "server.example.com:9090"  # Update with your server
     agent_id: "agent-node1"  # Optional, auto-generated if not set
     batch_size: 100
     batch_timeout: 5s
     reconnect_interval: 30s

   api:
     enabled: true
     host: 127.0.0.1
     port: 8080
   ```

3. **Run the agent**:

   ```bash
   sudo ./bin/microsegment-agent --config /etc/microsegment/agent.yaml
   ```

4. **Verify connection**:

   Check agent logs for successful connection:

   ```bash
   journalctl -u microsegment-agent -n 50
   ```

   Look for:
   - `✓ Connected to server at server.example.com:9090`
   - `Agent registered successfully`
   - `✓ Synced N policies (version X)`

5. **Verify on server**:

   ```bash
   # List registered agents
   curl http://server.example.com:8080/api/v1/agents

   # Check flows are being received
   curl http://server.example.com:8080/api/v1/flows?agent_id=agent-node1
   ```

### Agent Lifecycle

**Registration**:
- Agent connects to server on startup
- Sends registration request with hostname, IPs, version
- Receives server configuration (heartbeat interval, batch settings)

**Heartbeat**:
- Sent every 30 seconds (configurable)
- Includes agent metrics (CPU, memory, flow count)
- Server marks agent as inactive if heartbeat stops

**Policy Sync**:
- Initial sync on startup
- Periodic re-sync (future enhancement)
- Push updates from server (future enhancement)

**Flow Reporting**:
- Flows batched (default: 100 events or 5 seconds)
- Sent via gRPC streaming
- Asynchronous to avoid blocking eBPF collection

**Graceful Shutdown**:
- Agent sends unregister request
- Flushes pending flow batches
- Closes gRPC connections cleanly

---

## Configuration Reference

### Complete Configuration Options

```yaml
# Operation mode (required)
mode: standalone  # or agent-server

# Network interface for eBPF attachment (required)
interface: eth0

# Log level (default: info)
log_level: info  # debug, info, warn, error

# Statistics print interval in seconds (default: 30)
stats_interval: 30

# Storage configuration (standalone mode only)
storage:
  path: /var/lib/microsegment/flows.db

# API server configuration
api:
  enabled: true
  host: 127.0.0.1
  port: 8080
  enable_cors: true

# Agent-Server mode configuration (required if mode=agent-server)
agent_server:
  enabled: true

  # Server gRPC address (required)
  server_addr: "server.example.com:9090"

  # Unique agent identifier (optional, auto-generated from hostname if not set)
  agent_id: "agent-node1"

  # Batch size: number of flows to accumulate before sending (default: 100)
  batch_size: 100

  # Batch timeout: max time to wait before sending partial batch (default: 5s)
  batch_timeout: 5s

  # Reconnect interval: time to wait before reconnecting on failure (default: 30s)
  reconnect_interval: 30s
```

### Environment Variable Overrides

All configuration can be overridden via environment variables with prefix `MICROSEGMENT_`:

```bash
export MICROSEGMENT_MODE=agent-server
export MICROSEGMENT_AGENT_SERVER_SERVER_ADDR=server.example.com:9090
export MICROSEGMENT_LOG_LEVEL=debug

sudo ./bin/microsegment-agent
```

---

## Migration Guide

### From Standalone to Agent-Server

1. **Backup existing data**:

   ```bash
   cp /var/lib/microsegment/flows.db /backup/flows-backup.db
   ```

2. **Update configuration**:

   ```bash
   # Edit /etc/microsegment/agent.yaml
   vim /etc/microsegment/agent.yaml
   ```

   Change:
   ```yaml
   mode: standalone
   ```

   To:
   ```yaml
   mode: agent-server
   agent_server:
     server_addr: "server.example.com:9090"
   ```

3. **Restart agent**:

   ```bash
   sudo systemctl restart microsegment-agent
   ```

4. **Verify migration**:

   ```bash
   # Check agent logs
   journalctl -u microsegment-agent -f

   # Verify agent registered on server
   curl http://server.example.com:8080/api/v1/agents
   ```

5. **Optional: Import historical data**:

   ```bash
   # TODO: Add script to import SQLite data to server
   ./scripts/import-flows-to-server.sh /backup/flows-backup.db
   ```

### Rollback to Standalone

1. **Stop agent**:

   ```bash
   sudo systemctl stop microsegment-agent
   ```

2. **Update configuration**:

   ```yaml
   mode: standalone
   storage:
     path: /var/lib/microsegment/flows.db
   ```

3. **Restore data (if needed)**:

   ```bash
   cp /backup/flows-backup.db /var/lib/microsegment/flows.db
   ```

4. **Start agent**:

   ```bash
   sudo systemctl start microsegment-agent
   ```

---

## Troubleshooting

### Common Issues

#### Agent won't connect to server

**Symptoms**:
```
ERROR: failed to connect to server: connection refused
```

**Solutions**:

1. **Verify server is running**:
   ```bash
   curl http://server.example.com:8080/health
   ```

2. **Check network connectivity**:
   ```bash
   telnet server.example.com 9090
   # or
   nc -zv server.example.com 9090
   ```

3. **Verify firewall allows port 9090**:
   ```bash
   sudo iptables -L | grep 9090
   ```

4. **Check DNS resolution**:
   ```bash
   dig server.example.com
   ```

5. **Verify server_addr in config**:
   ```bash
   grep server_addr /etc/microsegment/agent.yaml
   ```

#### Flows not appearing in server

**Symptoms**:
- Agent connected successfully
- No flows visible in server database

**Solutions**:

1. **Check agent logs for errors**:
   ```bash
   journalctl -u microsegment-agent -n 100 | grep ERROR
   ```

2. **Verify eBPF programs loaded**:
   ```bash
   sudo bpftool prog list | grep microsegment
   ```

3. **Check agent statistics**:
   ```bash
   journalctl -u microsegment-agent | grep Statistics
   ```

4. **Verify server receives streams**:
   ```bash
   # On server
   journalctl -u microsegment-server | grep "flow events"
   ```

5. **Check batch queue status** (add metrics):
   ```bash
   curl http://localhost:8080/api/v1/metrics
   ```

#### Registration fails

**Symptoms**:
```
ERROR: registration rejected: invalid agent ID
```

**Solutions**:

1. **Check agent_id format**:
   - Must be unique across all agents
   - No special characters except `-` and `_`

2. **Verify server policy**:
   - Server may require pre-registration
   - Check server logs for rejection reason

3. **Check server capacity**:
   - Server may have reached max agent limit

#### High memory usage

**Symptoms**:
- Agent memory usage grows over time
- OOM kills on low-memory systems

**Solutions**:

1. **Reduce batch size**:
   ```yaml
   agent_server:
     batch_size: 50  # Reduce from default 100
   ```

2. **Reduce batch timeout**:
   ```yaml
   agent_server:
     batch_timeout: 2s  # Send more frequently
   ```

3. **Monitor queue depth**:
   - Check for dropped flows in logs
   - Increase server capacity if bottleneck is server-side

### Debug Mode

Enable debug logging for detailed troubleshooting:

```yaml
log_level: debug
```

Or via environment:

```bash
MICROSEGMENT_LOG_LEVEL=debug sudo ./bin/microsegment-agent --config /etc/microsegment/agent.yaml
```

Debug logs include:
- gRPC connection details
- Batch queue status
- Flow conversion details
- Heartbeat responses
- Policy sync details

### Health Checks

**Agent health** (both modes):
```bash
curl http://localhost:8080/health
```

**Server health** (agent-server mode):
```bash
curl http://server.example.com:8080/health
```

**Agent statistics**:
```bash
curl http://localhost:8080/api/v1/stats
```

---

## Performance Tuning

### Batch Size Optimization

**Small deployments** (< 1000 flows/sec):
```yaml
agent_server:
  batch_size: 50
  batch_timeout: 10s
```

**Medium deployments** (1000-10000 flows/sec):
```yaml
agent_server:
  batch_size: 100  # Default
  batch_timeout: 5s  # Default
```

**Large deployments** (> 10000 flows/sec):
```yaml
agent_server:
  batch_size: 500
  batch_timeout: 2s
```

### Network Optimization

**Low latency networks** (< 1ms RTT):
- Smaller batch sizes
- More frequent sends
- Lower batch timeout

**High latency networks** (> 50ms RTT):
- Larger batch sizes
- Less frequent sends
- Higher batch timeout

**Unreliable networks**:
- Shorter reconnect intervals
- Local caching (future enhancement)
- Retry logic (future enhancement)

---

## Next Steps

- [Server Deployment Guide](../src/server/README.md)
- [gRPC Protocol Reference](../proto/README.md)
- [Architecture Comparison](architecture-comparison.md)
- [Performance Benchmarks](performance-benchmarks.md)

---

**Version**: 1.1.0
**Last Updated**: 2025-11-05
**Status**: Implementation Complete
