# REST API Reference

Complete REST API documentation for the eBPF Microsegmentation Server.

## Overview

- **Base URL**: `http://<server>:8080/api/v1`
- **Framework**: Gin Web Framework (Go)
- **Content-Type**: `application/json`
- **Authentication**: None (MVP phase)

---

## Quick Reference

| Category | Endpoints | Description |
|----------|-----------|-------------|
| [Health](#health-check) | `/health` | Service health status |
| [Agents](#agent-api) | `/api/v1/agents` | Agent management |
| [Policies](#policy-api) | `/api/v1/policies` | Policy CRUD operations |
| [Flows](#flow-api) | `/api/v1/flows` | Traffic flow queries |
| [Alerts](#alert-api) | `/api/v1/alerts` | Security alerts |
| [Topology](#topology-api) | `/api/v1/topology` | Network topology |
| [Aggregator](#aggregator-api) | `/api/v1/aggregator` | Data aggregation |

---

## Health Check

### GET /health

Check server health status.

**Response**:
```json
{
  "status": "healthy",
  "service": "microsegment-server",
  "version": "1.0.0-mvp"
}
```

---

## Agent API

### GET /api/v1/agents

List all connected agents.

**Response**:
```json
{
  "agents": [
    {
      "id": "agent-node1",
      "hostname": "worker-1",
      "ip": "10.0.1.5",
      "status": "online",
      "last_seen": "2025-01-01T10:30:00Z",
      "version": "1.0.0"
    }
  ]
}
```

### GET /api/v1/agents/:id

Get agent details by ID.

**Parameters**:
- `id` (path): Agent ID

---

## Policy API

### GET /api/v1/policies

List all policies.

**Response**:
```json
{
  "policies": [...],
  "version": "v1.2.3"
}
```

### POST /api/v1/policies

Create a new policy.

**Request Body**:
```json
{
  "src_ip": "10.0.1.0/24",
  "dst_ip": "10.0.2.0/24",
  "src_port": 0,
  "dst_port": 80,
  "protocol": 6,
  "action": 0,
  "priority": 50,
  "source_labels": {"app": "nginx"},
  "dest_labels": {"env": "prod"},
  "description": "Allow nginx to prod",
  "process_name": "nginx",
  "process_path": "/usr/sbin/nginx",
  "match_mode": 0
}
```

**Field Reference**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `src_ip` | string | No | Source IP/CIDR |
| `dst_ip` | string | No | Destination IP/CIDR |
| `src_port` | int | No | Source port (0 = any) |
| `dst_port` | int | No | Destination port (0 = any) |
| `protocol` | int | No | Protocol (6=TCP, 17=UDP, 0=any) |
| `action` | int | Yes | 0=ALLOW, 1=DENY, 2=LOG |
| `priority` | int | No | 0-100 (higher = more priority) |
| `source_labels` | object | No | Source label selector |
| `dest_labels` | object | No | Destination label selector |
| `description` | string | No | Human-readable description |
| `process_name` | string | No | Process name filter |
| `process_path` | string | No | Process path filter |
| `match_mode` | int | No | 0=EXACT, 1=PREFIX, 2=WILDCARD |

### PUT /api/v1/policies/:id

Update an existing policy.

### DELETE /api/v1/policies/:id

Delete a policy by ID.

---

## Flow API

### GET /api/v1/flows

Query traffic flow events.

**Query Parameters**:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 100 | Max results |
| `offset` | int | 0 | Pagination offset |
| `start_time` | ISO8601 | -1h | Start time |
| `end_time` | ISO8601 | now | End time |
| `agent_id` | string | - | Filter by agent |
| `source_ip` | string | - | Filter by source IP |
| `dest_ip` | string | - | Filter by dest IP |
| `protocol` | string | - | Filter by protocol (tcp/udp) |
| `source_labels` | JSON | - | Filter by source labels |
| `dest_labels` | JSON | - | Filter by dest labels |

**Response**:
```json
{
  "flows": [
    {
      "src_ip": "10.0.1.5",
      "dst_ip": "10.0.2.10",
      "src_port": 54321,
      "dst_port": 80,
      "protocol": 6,
      "action": 0,
      "timestamp": 1672531200000000000
    }
  ],
  "total_count": 1250,
  "limit": 100,
  "offset": 0,
  "has_more": true
}
```

### GET /api/v1/flows/:id

Get single flow event details.

### GET /api/v1/flows/summary

Get 7-day flow statistics summary.

### GET /api/v1/flows/dependencies

Get application dependency graph based on labels.

---

## Alert API

### GET /api/v1/alerts

Query security alerts.

**Query Parameters**:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `page_size` | int | 50 | Results per page (max 100) |
| `level` | string | - | Filter: info/warning/critical |
| `type` | int | - | Alert type code |
| `process_path` | string | - | Filter by process |
| `start_time` | ISO8601 | - | Start time |
| `end_time` | ISO8601 | - | End time |

**Alert Levels**:
- `0` / `info`: Informational
- `1` / `warning`: Warning
- `2` / `critical`: Critical

**Response**:
```json
{
  "alerts": [
    {
      "id": "alert-001",
      "level": 2,
      "type": 101,
      "message": "Unauthorized process attempted connection",
      "process_path": "/tmp/malware",
      "src_ip": "10.0.1.5",
      "dst_ip": "8.8.8.8",
      "timestamp": 1672531200000000000
    }
  ],
  "total_count": 42,
  "page": 1,
  "page_size": 50
}
```

### GET /api/v1/alerts/:id

Get alert details.

### GET /api/v1/alerts/stats

Get alert statistics.

**Query Parameters**:
- `time_window`: `24h` / `7d` / `30d`

**Response**:
```json
{
  "by_level": {"info": 10, "warning": 20, "critical": 12},
  "by_type": {"101": 5, "102": 37},
  "top_processes": ["/tmp/malware", "/bin/bash"],
  "time_window": "24h"
}
```

---

## Topology API

### GET /api/v1/topology

Get complete network topology (nodes + edges).

**Query Parameters**:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `namespace` | string | - | K8s namespace filter |
| `include_external` | bool | true | Include external nodes |
| `min_flow_count` | int | 0 | Min flow threshold |
| `policy_action` | string | - | ALLOW/DENY filter |
| `group_by` | string | app | Grouping label key |
| `time_range` | string | 1h | Time window |
| `node_type` | string[] | - | Filter: pod/service/external |
| `label` | string[] | - | Label filter (key:value) |

**Response**:
```json
{
  "nodes": [
    {
      "id": "pod-001",
      "name": "nginx-deployment-xyz",
      "type": "pod",
      "namespace": "default",
      "labels": {"app": "nginx", "env": "prod"},
      "ip": "10.0.1.5",
      "flow_count": 1520,
      "status": "healthy"
    }
  ],
  "edges": [
    {
      "source": "pod-001",
      "target": "pod-002",
      "protocol": "tcp",
      "port": 5432,
      "flow_count": 850,
      "bytes_sent": 10485760,
      "policy_action": "ALLOW"
    }
  ],
  "total_count": {"nodes": 15, "edges": 24}
}
```

### GET /api/v1/topology/nodes

List all topology nodes.

### GET /api/v1/topology/nodes/:id

Get node details.

### GET /api/v1/topology/edges

List all topology edges.

### GET /api/v1/topology/edges/:src/:dst

Get edge details between two nodes.

### GET /api/v1/topology/stats

Get topology statistics.

---

## Aggregator API

### GET /api/v1/aggregator/dependencies

Get application dependencies aggregated by label.

**Query Parameters**:
- `group_by`: Label key for grouping (default: `app`)
- `start_time` / `end_time`: Time range

### GET /api/v1/aggregator/top-talkers

Get top traffic consumers.

**Query Parameters**:
- `top_n`: Number of results (default: 10)

### GET /api/v1/aggregator/stats

Get aggregated statistics.

**Query Parameters**:
- `include_top_talkers`: Include top talkers in response

---

## Error Response Format

All errors follow a consistent format:

```json
{
  "success": false,
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid request: policy ID is required",
    "details": null
  },
  "timestamp": "2025-01-01T10:30:00Z"
}
```

**Error Codes**:

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `BAD_REQUEST` | 400 | Invalid request parameters |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Access denied |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource conflict |
| `INTERNAL_SERVER_ERROR` | 500 | Server error |
| `INVALID_PARAMETER` | 400 | Parameter validation failed |
| `DATABASE_ERROR` | 500 | Database operation failed |

---

## CORS Headers

```http
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS, PATCH
Access-Control-Allow-Headers: *
```

---

## Examples

### cURL Examples

```bash
# List all policies
curl http://localhost:8080/api/v1/policies

# Create a policy
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{"src_ip": "10.0.1.0/24", "dst_ip": "10.0.2.0/24", "dst_port": 80, "protocol": 6, "action": 0}'

# Query flows with filters
curl "http://localhost:8080/api/v1/flows?limit=50&protocol=tcp&start_time=2025-01-01T00:00:00Z"

# Get topology
curl "http://localhost:8080/api/v1/topology?namespace=default&group_by=app"

# Get alerts
curl "http://localhost:8080/api/v1/alerts?level=critical&page_size=20"
```

---

*Last updated: 2024-12-24*
