# Operations Manual

Day-to-day operations guide for managing the eBPF Microsegmentation system.

## Overview

This guide covers:
- [Service Management](#service-management)
- [Health Monitoring](#health-monitoring)
- [Log Management](#log-management)
- [Performance Monitoring](#performance-monitoring)
- [Backup & Recovery](#backup--recovery)
- [Common Operations](#common-operations)
- [Troubleshooting](#troubleshooting)

---

## Service Management

### Systemd Services

```bash
# Start services
sudo systemctl start microsegment-server
sudo systemctl start microsegment-agent

# Stop services
sudo systemctl stop microsegment-agent
sudo systemctl stop microsegment-server

# Restart services
sudo systemctl restart microsegment-server
sudo systemctl restart microsegment-agent

# Check status
sudo systemctl status microsegment-server
sudo systemctl status microsegment-agent

# Enable at boot
sudo systemctl enable microsegment-server microsegment-agent
```

### Docker Deployment

```bash
# Start with docker-compose
docker-compose up -d

# View logs
docker-compose logs -f server
docker-compose logs -f agent

# Restart single service
docker-compose restart agent

# Stop all
docker-compose down
```

### Kubernetes Deployment

```bash
# Apply manifests
kubectl apply -f deploy/kubernetes/

# Check status
kubectl get pods -l app=microsegment
kubectl get daemonset microsegment-agent

# View logs
kubectl logs -l app=microsegment-server -f
kubectl logs -l app=microsegment-agent -f --all-containers

# Restart deployment
kubectl rollout restart deployment/microsegment-server
kubectl rollout restart daemonset/microsegment-agent
```

---

## Health Monitoring

### Health Endpoints

```bash
# Server health
curl http://localhost:8080/health

# Expected response
{
  "status": "healthy",
  "service": "microsegment-server",
  "version": "1.0.0"
}
```

### Key Metrics to Monitor

| Metric | Normal Range | Action if Abnormal |
|--------|--------------|-------------------|
| Server HTTP response time | < 100ms | Check database, increase resources |
| Agent-Server connection | Connected | Check network, restart agent |
| Database connections | < 80% of max | Increase pool size |
| Memory usage | < 80% | Check for memory leaks |
| Flow events per second | Depends on traffic | Scale if needed |
| Policy sync latency | < 1s | Check gRPC connection |

### Monitoring Commands

```bash
# Check server connections
ss -tlnp | grep microsegment

# Check agent eBPF programs
sudo bpftool prog list | grep microsegment

# Check eBPF maps
sudo bpftool map list

# View session map size
sudo bpftool map dump name session_map | wc -l
```

---

## Log Management

### Log Locations

| Component | Systemd | Docker |
|-----------|---------|--------|
| Server | `journalctl -u microsegment-server` | `docker logs microsegment-server` |
| Agent | `journalctl -u microsegment-agent` | `docker logs microsegment-agent` |

### Log Levels

```bash
# Set log level via environment
MICROSEGMENT_LOG_LEVEL=debug ./bin/microsegment-server

# Available levels: debug, info, warn, error
```

### Useful Log Commands

```bash
# Follow server logs
journalctl -u microsegment-server -f

# Filter by time
journalctl -u microsegment-agent --since "1 hour ago"

# Filter errors only
journalctl -u microsegment-server -p err

# Export logs
journalctl -u microsegment-server --since today > server-logs.txt
```

### Log Rotation

```bash
# /etc/logrotate.d/microsegment
/var/log/microsegment/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 root root
}
```

---

## Performance Monitoring

### Server Metrics

```bash
# API response time test
time curl http://localhost:8080/api/v1/policies

# Database query performance
psql -U microsegment_user -d microsegment -c "
  SELECT query, calls, mean_time, total_time
  FROM pg_stat_statements
  ORDER BY total_time DESC
  LIMIT 10;
"

# Connection pool status
psql -U microsegment_user -d microsegment -c "
  SELECT count(*) FROM pg_stat_activity
  WHERE datname = 'microsegment';
"
```

### Agent Metrics

```bash
# eBPF program statistics
sudo bpftool prog show

# Map memory usage
sudo bpftool map show

# Session count
sudo bpftool map dump name session_map 2>/dev/null | grep -c "key:"

# Policy count
sudo bpftool map dump name policy_map 2>/dev/null | grep -c "key:"
```

### Performance Tuning

```yaml
# Server: Increase database connections
database:
  max_open_conns: 50
  max_idle_conns: 10

# Agent: Adjust batch settings
server:
  batch_size: 200
  batch_timeout: 10s
```

---

## Backup & Recovery

### Database Backup

```bash
# Full backup
pg_dump -U microsegment_user -d microsegment > backup_$(date +%Y%m%d).sql

# Compressed backup
pg_dump -U microsegment_user -d microsegment | gzip > backup_$(date +%Y%m%d).sql.gz

# Automated backup script
#!/bin/bash
BACKUP_DIR=/var/backups/microsegment
mkdir -p $BACKUP_DIR
pg_dump -U microsegment_user -d microsegment | gzip > $BACKUP_DIR/backup_$(date +%Y%m%d_%H%M%S).sql.gz
find $BACKUP_DIR -name "*.sql.gz" -mtime +7 -delete
```

### Database Restore

```bash
# Stop services first
sudo systemctl stop microsegment-server

# Restore
psql -U microsegment_user -d microsegment < backup.sql

# Or from compressed
gunzip -c backup.sql.gz | psql -U microsegment_user -d microsegment

# Restart services
sudo systemctl start microsegment-server
```

### Configuration Backup

```bash
# Backup configs
tar -czvf microsegment-config-backup.tar.gz \
  /etc/microsegment/ \
  /config/

# Restore configs
tar -xzvf microsegment-config-backup.tar.gz -C /
```

---

## Common Operations

### Policy Management

```bash
# List all policies
curl http://localhost:8080/api/v1/policies | jq

# Add a policy
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "src_ip": "10.0.1.0/24",
    "dst_ip": "10.0.2.0/24",
    "dst_port": 80,
    "protocol": 6,
    "action": 0,
    "description": "Allow web traffic"
  }'

# Delete a policy
curl -X DELETE http://localhost:8080/api/v1/policies/123

# Export policies
curl http://localhost:8080/api/v1/policies > policies-backup.json

# Import policies (script needed)
for policy in $(cat policies-backup.json | jq -c '.policies[]'); do
  curl -X POST http://localhost:8080/api/v1/policies \
    -H "Content-Type: application/json" \
    -d "$policy"
done
```

### Agent Management

```bash
# List connected agents
curl http://localhost:8080/api/v1/agents | jq

# Check specific agent
curl http://localhost:8080/api/v1/agents/agent-node1 | jq
```

### Flow Analysis

```bash
# Recent flows
curl "http://localhost:8080/api/v1/flows?limit=100" | jq

# Filter by source IP
curl "http://localhost:8080/api/v1/flows?source_ip=10.0.1.5" | jq

# Get flow summary
curl http://localhost:8080/api/v1/flows/summary | jq

# Top talkers
curl http://localhost:8080/api/v1/aggregator/top-talkers | jq
```

### Alert Management

```bash
# Recent alerts
curl "http://localhost:8080/api/v1/alerts?page_size=50" | jq

# Critical alerts only
curl "http://localhost:8080/api/v1/alerts?level=critical" | jq

# Alert statistics
curl http://localhost:8080/api/v1/alerts/stats | jq
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check configuration
./bin/microsegment-server --config config/server.yaml 2>&1 | head -20

# Check port availability
ss -tlnp | grep -E "(8080|9090)"

# Check database
psql -h localhost -U microsegment_user -d microsegment -c "SELECT 1;"
```

### Agent Can't Connect to Server

```bash
# Test gRPC connectivity
grpcurl -plaintext localhost:9090 list

# Check firewall
sudo iptables -L -n | grep 9090

# Test with telnet
telnet server-ip 9090
```

### eBPF Load Failure

```bash
# Check kernel version
uname -r  # Should be 5.4+

# Check eBPF filesystem
mount | grep bpf

# Check capabilities
capsh --print | grep bpf

# Run with debug
MICROSEGMENT_LOG_LEVEL=debug sudo ./bin/microsegment-agent -c config/agent.yaml
```

### High Memory Usage

```bash
# Check process memory
ps aux | grep microsegment

# Check eBPF map sizes
sudo bpftool map list

# Reduce session map size if needed (requires recompile)
# Or increase cleanup frequency in config
```

### Policy Not Working

```bash
# Verify policy exists
curl http://localhost:8080/api/v1/policies | jq

# Check policy sync to agent
curl http://localhost:8081/api/v1/status

# Check eBPF policy map
sudo bpftool map dump name policy_map

# Enable debug logging
MICROSEGMENT_LOG_LEVEL=debug
```

### Database Issues

```bash
# Check connections
psql -U microsegment_user -d microsegment -c "
  SELECT * FROM pg_stat_activity WHERE datname='microsegment';
"

# Check disk space
df -h /var/lib/postgresql/

# Vacuum database
psql -U microsegment_user -d microsegment -c "VACUUM ANALYZE;"

# Reset connections
sudo systemctl restart postgresql
```

---

## Emergency Procedures

### Service Recovery

```bash
# Quick restart all services
sudo systemctl restart microsegment-server microsegment-agent

# Force kill if stuck
sudo pkill -9 microsegment

# Clear eBPF state
sudo rm -rf /sys/fs/bpf/microsegment/

# Rebuild and restart
make clean && make all
sudo systemctl restart microsegment-server microsegment-agent
```

### Database Recovery

```bash
# If database corrupted
sudo systemctl stop microsegment-server
sudo -u postgres pg_resetwal /var/lib/postgresql/data/

# Or restore from backup
psql -U microsegment_user -d microsegment < latest_backup.sql
sudo systemctl start microsegment-server
```

---

## Health Check Script

```bash
#!/bin/bash
# /usr/local/bin/microsegment-healthcheck.sh

# Check server
if ! curl -sf http://localhost:8080/health > /dev/null; then
    echo "ERROR: Server unhealthy"
    exit 1
fi

# Check agent (if API enabled)
if ! curl -sf http://localhost:8081/health > /dev/null 2>&1; then
    echo "WARNING: Agent API not responding"
fi

# Check database connections
DB_CONNS=$(psql -U microsegment_user -d microsegment -t -c "SELECT count(*) FROM pg_stat_activity WHERE datname='microsegment';")
if [ "$DB_CONNS" -gt 20 ]; then
    echo "WARNING: High database connections: $DB_CONNS"
fi

echo "OK: All checks passed"
exit 0
```

---

*Last updated: 2024-12-24*
