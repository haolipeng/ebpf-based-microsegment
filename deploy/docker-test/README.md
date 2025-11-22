# Docker Test Environment for Network Topology

This directory contains Docker Compose configurations for testing the network topology visualization feature.

## Quick Start (Recommended)

```bash
cd deploy/docker-test

# Make script executable
chmod +x start.sh

# Start simple environment (no build required)
./start.sh

# Or manually:
docker-compose -f docker-compose-simple.yml up -d
```

## Environment Options

### 1. Simple Environment (`docker-compose-simple.yml`)

**Best for quick local testing.** Uses only official Docker images, no build required.

**Services:**
| Service | IP | Port | Description |
|---------|-----|------|-------------|
| nginx-proxy | 172.31.0.10 | 8880 | Nginx reverse proxy |
| httpbin | 172.31.0.20 | 80 | HTTP request/response testing |
| redis | 172.31.0.30 | 6379 | Redis cache |
| postgres | 172.31.0.40 | 5432 | PostgreSQL database |
| client-1 | 172.31.0.100 | - | Traffic generator (HTTP + Redis) |
| client-2 | 172.31.0.101 | - | Traffic generator (HTTP + DB + Redis) |
| client-3 | 172.31.0.102 | - | Burst traffic generator |
| external-service | 172.31.0.200 | - | External API simulator |

**Network Topology:**

```
┌─────────────────────────────────────────────────────────────┐
│                        test-net (172.31.0.0/24)             │
│                                                             │
│   [client-1]──┐                                             │
│   [client-2]──┼──>[nginx-proxy:80]──>[httpbin:80]           │
│   [client-3]──┘         │                                   │
│       │                 │                                   │
│       └─────────────────┼───>[redis:6379]                   │
│       │                 │                                   │
│       └─────────────────┴───>[postgres:5432]                │
│                                                             │
│   [external-service] (simulates external API)               │
└─────────────────────────────────────────────────────────────┘
```

### 2. Full Environment (`docker-compose.yml`)

**For comprehensive testing.** Includes custom Python services, requires building.

**Additional Services:**
- Multi-tier architecture (frontend, backend, data)
- RabbitMQ message queue
- Elasticsearch search engine
- Custom API services with database/cache integration
- Background worker services

## Usage

### Start/Stop

```bash
# Start simple environment
./start.sh simple

# Start full environment (builds images)
./start.sh full

# Stop all containers
./start.sh stop

# Remove all containers and data
./start.sh clean

# Check status
./start.sh status
```

### View Logs

```bash
# Simple environment
docker-compose -f docker-compose-simple.yml logs -f

# Follow specific service
docker-compose -f docker-compose-simple.yml logs -f client-1

# Full environment
docker-compose -f docker-compose.yml logs -f
```

### Network Traffic Monitoring

```bash
# Real-time container stats
docker stats

# View network traffic
docker stats --format "table {{.Name}}\t{{.NetIO}}"

# Inspect network
docker network inspect docker-test_test-net
```

## Testing with Microsegmentation Agent

1. **Start the test environment:**
   ```bash
   ./start.sh
   ```

2. **Start the microsegment server:**
   ```bash
   cd /home/work/ebpf-based-microsegment
   ./bin/microsegment-server -c config/server.yaml
   ```

3. **Start the microsegment agent:**
   ```bash
   sudo ./bin/microsegment-agent -c config/agent.yaml
   ```

4. **Open the Web UI:**
   ```
   http://localhost:3000/topology
   ```

5. **Select view mode:**
   - **CONTAINER** - See container-level traffic
   - **IP** - See raw IP traffic

## Expected Traffic Patterns

The test environment generates the following traffic patterns:

| Source | Destination | Protocol | Pattern |
|--------|-------------|----------|---------|
| client-1 | nginx-proxy | HTTP | Every 3s |
| client-1 | redis | TCP/6379 | Every 3s |
| client-2 | nginx-proxy | HTTP | Every 4s |
| client-2 | postgres | TCP/5432 | Every 4s |
| client-2 | redis | TCP/6379 | Every 4s |
| client-3 | nginx-proxy | HTTP | Burst (5 req) every 5s |
| client-3 | redis | TCP/6379 | Pub/Sub |
| client-* | client-* | ICMP | Ping between clients |
| nginx-proxy | httpbin | HTTP | Proxied requests |

## Verifying Flow Collection

```bash
# Check if agent is collecting flows
curl -s http://localhost:8080/api/v1/flows | jq '.flows | length'

# Get flow summary
curl -s http://localhost:8080/api/v1/flows/summary | jq

# Filter by protocol
curl -s "http://localhost:8080/api/v1/flows?protocol=TCP" | jq

# Get topology data
curl -s http://localhost:8080/api/v1/topology?viewMode=CONTAINER | jq
```

## Troubleshooting

### Containers not starting

```bash
# Check Docker status
docker info

# Check for port conflicts
netstat -tlnp | grep -E '8880|6379|5432'

# View container logs
docker-compose -f docker-compose-simple.yml logs
```

### No traffic visible

1. Ensure agent is running with root privileges
2. Check agent logs for eBPF errors
3. Verify network interface is being monitored

```bash
# List network interfaces
ip link show

# Check agent config for monitored interfaces
cat config/agent.yaml | grep interface
```

### Agent not collecting Docker traffic

Ensure the agent monitors the Docker bridge interface:

```yaml
# config/agent.yaml
network:
  interfaces:
    - eth0
    - docker0
    - br-*  # Docker bridge networks
```

## Clean Up

```bash
# Stop and remove containers
./start.sh clean

# Remove Docker networks
docker network prune

# Remove unused images
docker image prune
```
