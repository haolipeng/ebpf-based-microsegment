# Label-Based Policy API Documentation

This document describes the RESTful API endpoints for managing label-based policies in the eBPF microsegmentation agent.

## Table of Contents

1. [Overview](#overview)
2. [Workload Management API](#workload-management-api)
3. [Group Management API](#group-management-api)
4. [Policy Rule Management API](#policy-rule-management-api)
5. [Error Handling](#error-handling)
6. [Complete Workflow Examples](#complete-workflow-examples)

## Overview

The label-based policy system provides a cloud-native approach to network security policies:

- **Workloads**: Network entities (containers, VMs, pods) identified by labels
- **Groups**: Dynamic collections of workloads selected by label selectors
- **Policy Rules**: High-level rules defining allowed traffic between groups
- **Compiled Policies**: Low-level IP-based rules generated from policy rules (N×M expansion)

### API Base URL

All endpoints are prefixed with: `/api/v1`

### Authentication

Currently, no authentication is required (to be implemented in future versions).

---

## Workload Management API

Workloads represent network entities with IP addresses and labels.

### Create Workload

Creates a new workload with the specified labels and IP addresses.

**Endpoint:** `POST /api/v1/workloads`

**Request Body:**
```json
{
  "id": "web-frontend-1",
  "name": "Web Frontend Instance 1",
  "ips": ["10.0.1.10", "fd00::1"],
  "labels": {
    "app": "web",
    "tier": "frontend",
    "env": "production",
    "version": "v2.1.0"
  }
}
```

**Response:** `201 Created`
```json
{
  "id": "web-frontend-1",
  "name": "Web Frontend Instance 1",
  "ips": ["10.0.1.10", "fd00::1"],
  "labels": {
    "app": "web",
    "tier": "frontend",
    "env": "production",
    "version": "v2.1.0"
  }
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "id": "web-frontend-1",
    "name": "Web Frontend Instance 1",
    "ips": ["10.0.1.10"],
    "labels": {
      "app": "web",
      "tier": "frontend",
      "env": "production"
    }
  }'
```

---

### List Workloads

Retrieves all workloads in the system.

**Endpoint:** `GET /api/v1/workloads`

**Response:** `200 OK`
```json
{
  "workloads": [
    {
      "id": "web-frontend-1",
      "name": "Web Frontend Instance 1",
      "ips": ["10.0.1.10"],
      "labels": {
        "app": "web",
        "tier": "frontend",
        "env": "production"
      }
    },
    {
      "id": "api-backend-1",
      "name": "API Backend Instance 1",
      "ips": ["10.0.2.20"],
      "labels": {
        "app": "api",
        "tier": "backend",
        "env": "production"
      }
    }
  ],
  "count": 2
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/workloads
```

---

### Get Workload by ID

Retrieves a specific workload by its ID.

**Endpoint:** `GET /api/v1/workloads/:id`

**Response:** `200 OK`
```json
{
  "id": "web-frontend-1",
  "name": "Web Frontend Instance 1",
  "ips": ["10.0.1.10"],
  "labels": {
    "app": "web",
    "tier": "frontend",
    "env": "production"
  }
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/workloads/web-frontend-1
```

**Error Response:** `404 Not Found`
```json
{
  "code": 404,
  "error": "not_found",
  "message": "Workload with ID web-frontend-999 not found",
  "details": "workload not found"
}
```

---

### Update Workload

Updates an existing workload's properties.

**Endpoint:** `PUT /api/v1/workloads/:id`

**Request Body:**
```json
{
  "id": "web-frontend-1",
  "name": "Web Frontend Instance 1 (Updated)",
  "ips": ["10.0.1.10", "10.0.1.11"],
  "labels": {
    "app": "web",
    "tier": "frontend",
    "env": "production",
    "version": "v2.2.0"
  }
}
```

**Response:** `200 OK`
```json
{
  "id": "web-frontend-1",
  "name": "Web Frontend Instance 1 (Updated)",
  "ips": ["10.0.1.10", "10.0.1.11"],
  "labels": {
    "app": "web",
    "tier": "frontend",
    "env": "production",
    "version": "v2.2.0"
  }
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/v1/workloads/web-frontend-1 \
  -H "Content-Type: application/json" \
  -d '{
    "id": "web-frontend-1",
    "name": "Web Frontend Instance 1 (Updated)",
    "ips": ["10.0.1.10", "10.0.1.11"],
    "labels": {
      "app": "web",
      "tier": "frontend",
      "env": "production",
      "version": "v2.2.0"
    }
  }'
```

---

### Delete Workload

Deletes a workload from the system.

**Endpoint:** `DELETE /api/v1/workloads/:id`

**Response:** `200 OK`
```json
{
  "message": "Workload with ID web-frontend-1 deleted successfully"
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/v1/workloads/web-frontend-1
```

---

## Group Management API

Groups are dynamic collections of workloads selected by label selectors.

### Create Group

Creates a new group with label-based selectors.

**Endpoint:** `POST /api/v1/groups`

**Request Body:**
```json
{
  "name": "frontend-servers",
  "match_labels": {
    "tier": "frontend",
    "env": "production"
  }
}
```

**Response:** `201 Created`
```json
{
  "name": "frontend-servers",
  "match_labels": {
    "tier": "frontend",
    "env": "production"
  },
  "member_ids": ["web-frontend-1", "web-frontend-2"],
  "member_count": 2,
  "is_static": false
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/groups \
  -H "Content-Type: application/json" \
  -d '{
    "name": "frontend-servers",
    "match_labels": {
      "tier": "frontend",
      "env": "production"
    }
  }'
```

---

### List Groups

Retrieves all groups in the system.

**Endpoint:** `GET /api/v1/groups`

**Response:** `200 OK`
```json
{
  "groups": [
    {
      "name": "frontend-servers",
      "match_labels": {
        "tier": "frontend",
        "env": "production"
      },
      "member_ids": ["web-frontend-1", "web-frontend-2"],
      "member_count": 2,
      "is_static": false
    },
    {
      "name": "backend-servers",
      "match_labels": {
        "tier": "backend",
        "env": "production"
      },
      "member_ids": ["api-backend-1", "db-backend-1"],
      "member_count": 2,
      "is_static": false
    }
  ],
  "count": 2
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/groups
```

---

### Get Group by Name

Retrieves a specific group by name.

**Endpoint:** `GET /api/v1/groups/:name`

**Response:** `200 OK`
```json
{
  "name": "frontend-servers",
  "match_labels": {
    "tier": "frontend",
    "env": "production"
  },
  "member_ids": ["web-frontend-1", "web-frontend-2"],
  "member_count": 2,
  "is_static": false
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/groups/frontend-servers
```

---

### Get Group Members

Retrieves detailed information about all members of a group.

**Endpoint:** `GET /api/v1/groups/:name/members`

**Response:** `200 OK`
```json
{
  "group_name": "frontend-servers",
  "members": [
    {
      "id": "web-frontend-1",
      "name": "Web Frontend Instance 1",
      "ips": ["10.0.1.10"],
      "labels": {
        "app": "web",
        "tier": "frontend",
        "env": "production"
      }
    },
    {
      "id": "web-frontend-2",
      "name": "Web Frontend Instance 2",
      "ips": ["10.0.1.11"],
      "labels": {
        "app": "web",
        "tier": "frontend",
        "env": "production"
      }
    }
  ],
  "count": 2
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/groups/frontend-servers/members
```

---

### Update Group

Updates a group's label selectors.

**Endpoint:** `PUT /api/v1/groups/:name`

**Request Body:**
```json
{
  "name": "frontend-servers",
  "match_labels": {
    "tier": "frontend",
    "env": "production",
    "version": "v2"
  }
}
```

**Response:** `200 OK`
```json
{
  "name": "frontend-servers",
  "match_labels": {
    "tier": "frontend",
    "env": "production",
    "version": "v2"
  },
  "member_ids": ["web-frontend-1"],
  "member_count": 1,
  "is_static": false
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/v1/groups/frontend-servers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "frontend-servers",
    "match_labels": {
      "tier": "frontend",
      "env": "production",
      "version": "v2"
    }
  }'
```

---

### Delete Group

Deletes a group from the system.

**Endpoint:** `DELETE /api/v1/groups/:name`

**Response:** `200 OK`
```json
{
  "message": "Group frontend-servers deleted successfully"
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/v1/groups/frontend-servers
```

---

## Policy Rule Management API

Policy rules define high-level traffic policies between groups. These are automatically compiled into low-level IP-based eBPF rules.

### Create Policy Rule

Creates a new policy rule and compiles it into eBPF rules.

**Endpoint:** `POST /api/v1/policy-rules`

**Request Body:**
```json
{
  "name": "frontend-to-backend-http",
  "description": "Allow frontend servers to access backend API over HTTP/HTTPS",
  "from_group": "frontend-servers",
  "to_group": "backend-servers",
  "ports": [
    {
      "start": 80,
      "end": 80,
      "protocol": "tcp"
    },
    {
      "start": 443,
      "end": 443,
      "protocol": "tcp"
    }
  ],
  "action": "allow",
  "priority": 100,
  "enabled": true
}
```

**Response:** `201 Created`
```json
{
  "id": 1,
  "name": "frontend-to-backend-http",
  "description": "Allow frontend servers to access backend API over HTTP/HTTPS",
  "from_group": "frontend-servers",
  "to_group": "backend-servers",
  "ports": [
    {
      "start": 80,
      "end": 80,
      "protocol": "tcp"
    },
    {
      "start": 443,
      "end": 443,
      "protocol": "tcp"
    }
  ],
  "action": "allow",
  "priority": 100,
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/policy-rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "frontend-to-backend-http",
    "description": "Allow frontend to access backend over HTTP/HTTPS",
    "from_group": "frontend-servers",
    "to_group": "backend-servers",
    "ports": [
      {"start": 80, "end": 80, "protocol": "tcp"},
      {"start": 443, "end": 443, "protocol": "tcp"}
    ],
    "action": "allow",
    "priority": 100,
    "enabled": true
  }'
```

**Notes:**
- The rule is automatically compiled into N×M IP-based policies
- If `from_group` has 2 members and `to_group` has 3 members, this creates 2×3×2 = 12 compiled policies (2 ports)
- Compiled policies are immediately loaded into the eBPF dataplane

---

### List Policy Rules

Retrieves all policy rules in the system.

**Endpoint:** `GET /api/v1/policy-rules`

**Response:** `200 OK`
```json
{
  "rules": [
    {
      "id": 1,
      "name": "frontend-to-backend-http",
      "description": "Allow frontend to access backend over HTTP/HTTPS",
      "from_group": "frontend-servers",
      "to_group": "backend-servers",
      "ports": [
        {"start": 80, "end": 80, "protocol": "tcp"},
        {"start": 443, "end": 443, "protocol": "tcp"}
      ],
      "action": "allow",
      "priority": 100,
      "enabled": true,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    },
    {
      "id": 2,
      "name": "backend-to-database-mysql",
      "description": "Allow backend to access database",
      "from_group": "backend-servers",
      "to_group": "database-servers",
      "ports": [
        {"start": 3306, "end": 3306, "protocol": "tcp"}
      ],
      "action": "allow",
      "priority": 100,
      "enabled": true,
      "created_at": "2025-01-15T10:35:00Z",
      "updated_at": "2025-01-15T10:35:00Z"
    }
  ],
  "count": 2
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/policy-rules
```

---

### Get Policy Rule by ID

Retrieves a specific policy rule by its ID.

**Endpoint:** `GET /api/v1/policy-rules/:id`

**Response:** `200 OK`
```json
{
  "id": 1,
  "name": "frontend-to-backend-http",
  "description": "Allow frontend to access backend over HTTP/HTTPS",
  "from_group": "frontend-servers",
  "to_group": "backend-servers",
  "ports": [
    {"start": 80, "end": 80, "protocol": "tcp"},
    {"start": 443, "end": 443, "protocol": "tcp"}
  ],
  "action": "allow",
  "priority": 100,
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/policy-rules/1
```

---

### Get Compiled Policies for Rule

Retrieves all compiled IP-based policies generated from a policy rule. This provides full traceability.

**Endpoint:** `GET /api/v1/policy-rules/:id/compiled`

**Response:** `200 OK`
```json
{
  "policies": [
    {
      "rule_id": 100001,
      "source_rule_id": 1,
      "src_ip": "10.0.1.10",
      "dst_ip": "10.0.2.20",
      "dst_port": 80,
      "protocol": "tcp",
      "action": "allow",
      "priority": 100,
      "from_group": "frontend-servers",
      "to_group": "backend-servers",
      "from_workload_id": "web-frontend-1",
      "to_workload_id": "api-backend-1",
      "compilation_time": "2025-01-15T10:30:00Z",
      "compiler_version": "v1.0.0"
    },
    {
      "rule_id": 100002,
      "source_rule_id": 1,
      "src_ip": "10.0.1.10",
      "dst_ip": "10.0.2.20",
      "dst_port": 443,
      "protocol": "tcp",
      "action": "allow",
      "priority": 100,
      "from_group": "frontend-servers",
      "to_group": "backend-servers",
      "from_workload_id": "web-frontend-1",
      "to_workload_id": "api-backend-1",
      "compilation_time": "2025-01-15T10:30:00Z",
      "compiler_version": "v1.0.0"
    },
    {
      "rule_id": 100003,
      "source_rule_id": 1,
      "src_ip": "10.0.1.11",
      "dst_ip": "10.0.2.20",
      "dst_port": 80,
      "protocol": "tcp",
      "action": "allow",
      "priority": 100,
      "from_group": "frontend-servers",
      "to_group": "backend-servers",
      "from_workload_id": "web-frontend-2",
      "to_workload_id": "api-backend-1",
      "compilation_time": "2025-01-15T10:30:00Z",
      "compiler_version": "v1.0.0"
    },
    {
      "rule_id": 100004,
      "source_rule_id": 1,
      "src_ip": "10.0.1.11",
      "dst_ip": "10.0.2.20",
      "dst_port": 443,
      "protocol": "tcp",
      "action": "allow",
      "priority": 100,
      "from_group": "frontend-servers",
      "to_group": "backend-servers",
      "from_workload_id": "web-frontend-2",
      "to_workload_id": "api-backend-1",
      "compilation_time": "2025-01-15T10:30:00Z",
      "compiler_version": "v1.0.0"
    }
  ],
  "count": 4
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/policy-rules/1/compiled
```

**Notes:**
- This endpoint shows the N×M expansion of the policy rule
- Each compiled policy includes traceability information linking back to:
  - Source policy rule ID
  - Source and destination groups
  - Specific workload IDs involved
  - Compilation timestamp and compiler version
- Use this for debugging, auditing, and understanding policy application

---

### Update Policy Rule

Updates an existing policy rule and recompiles it.

**Endpoint:** `PUT /api/v1/policy-rules/:id`

**Request Body:**
```json
{
  "name": "frontend-to-backend-http",
  "description": "Allow frontend to access backend over HTTP/HTTPS (updated)",
  "from_group": "frontend-servers",
  "to_group": "backend-servers",
  "ports": [
    {"start": 80, "end": 80, "protocol": "tcp"},
    {"start": 443, "end": 443, "protocol": "tcp"},
    {"start": 8080, "end": 8080, "protocol": "tcp"}
  ],
  "action": "allow",
  "priority": 150,
  "enabled": true
}
```

**Response:** `200 OK`
```json
{
  "id": 1,
  "name": "frontend-to-backend-http",
  "description": "Allow frontend to access backend over HTTP/HTTPS (updated)",
  "from_group": "frontend-servers",
  "to_group": "backend-servers",
  "ports": [
    {"start": 80, "end": 80, "protocol": "tcp"},
    {"start": 443, "end": 443, "protocol": "tcp"},
    {"start": 8080, "end": 8080, "protocol": "tcp"}
  ],
  "action": "allow",
  "priority": 150,
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:45:00Z"
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/v1/policy-rules/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "frontend-to-backend-http",
    "description": "Allow frontend to access backend (updated)",
    "from_group": "frontend-servers",
    "to_group": "backend-servers",
    "ports": [
      {"start": 80, "end": 80, "protocol": "tcp"},
      {"start": 443, "end": 443, "protocol": "tcp"},
      {"start": 8080, "end": 8080, "protocol": "tcp"}
    ],
    "action": "allow",
    "priority": 150,
    "enabled": true
  }'
```

**Notes:**
- Old compiled policies are deleted from eBPF
- New policies are compiled and loaded
- This is an atomic operation (delete + recompile)

---

### Delete Policy Rule

Deletes a policy rule and removes all its compiled policies from eBPF.

**Endpoint:** `DELETE /api/v1/policy-rules/:id`

**Response:** `200 OK`
```json
{
  "message": "Policy rule with ID 1 deleted successfully"
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/v1/policy-rules/1
```

**Notes:**
- All compiled policies are removed from the eBPF dataplane
- Compilation metadata is cleaned up from storage
- This operation is atomic

---

## Error Handling

All API endpoints return consistent error responses.

### Error Response Format

```json
{
  "code": 400,
  "error": "validation_error",
  "message": "Invalid request body",
  "details": "Key: 'WorkloadRequest.IPs' Error:Field validation for 'IPs' failed on the 'required' tag"
}
```

### Common Error Codes

| HTTP Status | Error Type | Description |
|-------------|------------|-------------|
| 400 | validation_error | Request validation failed (missing fields, invalid format) |
| 404 | not_found | Requested resource not found |
| 500 | workload_error | Workload operation failed |
| 500 | group_error | Group operation failed |
| 500 | policy_rule_error | Policy rule operation failed |

### Example Error Responses

**Invalid IP Address:**
```json
{
  "code": 400,
  "error": "validation_error",
  "message": "Invalid IP address: 999.999.999.999",
  "details": null
}
```

**Workload Not Found:**
```json
{
  "code": 404,
  "error": "not_found",
  "message": "Workload with ID unknown-workload not found",
  "details": "workload not found"
}
```

**Group Validation Error:**
```json
{
  "code": 500,
  "error": "group_error",
  "message": "Failed to create group",
  "details": "validation failed: group must have at least one selector"
}
```

---

## Complete Workflow Examples

### Example 1: Three-Tier Application Setup

This example sets up a complete three-tier application with frontend, backend, and database tiers.

#### Step 1: Create Workloads

```bash
# Create frontend workloads
curl -X POST http://localhost:8080/api/v1/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "id": "web-1",
    "name": "Web Server 1",
    "ips": ["10.0.1.10"],
    "labels": {"tier": "frontend", "app": "web", "env": "prod"}
  }'

curl -X POST http://localhost:8080/api/v1/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "id": "web-2",
    "name": "Web Server 2",
    "ips": ["10.0.1.11"],
    "labels": {"tier": "frontend", "app": "web", "env": "prod"}
  }'

# Create backend workloads
curl -X POST http://localhost:8080/api/v1/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "id": "api-1",
    "name": "API Server 1",
    "ips": ["10.0.2.20"],
    "labels": {"tier": "backend", "app": "api", "env": "prod"}
  }'

curl -X POST http://localhost:8080/api/v1/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "id": "api-2",
    "name": "API Server 2",
    "ips": ["10.0.2.21"],
    "labels": {"tier": "backend", "app": "api", "env": "prod"}
  }'

# Create database workload
curl -X POST http://localhost:8080/api/v1/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "id": "db-1",
    "name": "Database Server",
    "ips": ["10.0.3.30"],
    "labels": {"tier": "database", "app": "postgres", "env": "prod"}
  }'
```

#### Step 2: Create Groups

```bash
# Create frontend group
curl -X POST http://localhost:8080/api/v1/groups \
  -H "Content-Type: application/json" \
  -d '{
    "name": "frontend-tier",
    "match_labels": {"tier": "frontend", "env": "prod"}
  }'

# Create backend group
curl -X POST http://localhost:8080/api/v1/groups \
  -H "Content-Type: application/json" \
  -d '{
    "name": "backend-tier",
    "match_labels": {"tier": "backend", "env": "prod"}
  }'

# Create database group
curl -X POST http://localhost:8080/api/v1/groups \
  -H "Content-Type: application/json" \
  -d '{
    "name": "database-tier",
    "match_labels": {"tier": "database", "env": "prod"}
  }'
```

#### Step 3: Verify Group Membership

```bash
# Check frontend group members
curl http://localhost:8080/api/v1/groups/frontend-tier/members

# Expected: web-1, web-2
```

#### Step 4: Create Policy Rules

```bash
# Allow frontend → backend (HTTP/HTTPS)
curl -X POST http://localhost:8080/api/v1/policy-rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "frontend-to-backend",
    "description": "Allow web servers to access API servers",
    "from_group": "frontend-tier",
    "to_group": "backend-tier",
    "ports": [
      {"start": 80, "end": 80, "protocol": "tcp"},
      {"start": 443, "end": 443, "protocol": "tcp"}
    ],
    "action": "allow",
    "priority": 100,
    "enabled": true
  }'

# Allow backend → database (PostgreSQL)
curl -X POST http://localhost:8080/api/v1/policy-rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "backend-to-database",
    "description": "Allow API servers to access database",
    "from_group": "backend-tier",
    "to_group": "database-tier",
    "ports": [
      {"start": 5432, "end": 5432, "protocol": "tcp"}
    ],
    "action": "allow",
    "priority": 100,
    "enabled": true
  }'
```

#### Step 5: Verify Compilation

```bash
# Check compiled policies for frontend-to-backend rule
curl http://localhost:8080/api/v1/policy-rules/1/compiled

# Expected: 2 frontend × 2 backend × 2 ports = 8 compiled policies
# - web-1 (10.0.1.10) → api-1 (10.0.2.20) : 80
# - web-1 (10.0.1.10) → api-1 (10.0.2.20) : 443
# - web-1 (10.0.1.10) → api-2 (10.0.2.21) : 80
# - web-1 (10.0.1.10) → api-2 (10.0.2.21) : 443
# - web-2 (10.0.1.11) → api-1 (10.0.2.20) : 80
# - web-2 (10.0.1.11) → api-1 (10.0.2.20) : 443
# - web-2 (10.0.1.11) → api-2 (10.0.2.21) : 80
# - web-2 (10.0.1.11) → api-2 (10.0.2.21) : 443
```

---

### Example 2: Dynamic Workload Addition

This example shows how adding a new workload automatically updates group membership and triggers policy recompilation.

```bash
# Initial state: 2 web servers in frontend-tier group

# Add a new web server
curl -X POST http://localhost:8080/api/v1/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "id": "web-3",
    "name": "Web Server 3",
    "ips": ["10.0.1.12"],
    "labels": {"tier": "frontend", "app": "web", "env": "prod"}
  }'

# Check group membership (now includes web-3)
curl http://localhost:8080/api/v1/groups/frontend-tier/members

# To recompile policies with new membership:
# (This would be done automatically by a workload change webhook or manual trigger)
# For now, update the policy rule to trigger recompilation:
curl -X PUT http://localhost:8080/api/v1/policy-rules/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "frontend-to-backend",
    "description": "Allow web servers to access API servers",
    "from_group": "frontend-tier",
    "to_group": "backend-tier",
    "ports": [
      {"start": 80, "end": 80, "protocol": "tcp"},
      {"start": 443, "end": 443, "protocol": "tcp"}
    ],
    "action": "allow",
    "priority": 100,
    "enabled": true
  }'

# Verify: Now 3 frontend × 2 backend × 2 ports = 12 compiled policies
curl http://localhost:8080/api/v1/policy-rules/1/compiled
```

---

### Example 3: Troubleshooting Policy Issues

This example shows how to use the API to troubleshoot connectivity issues.

```bash
# Scenario: web-1 cannot connect to api-1

# Step 1: Verify workload labels
curl http://localhost:8080/api/v1/workloads/web-1
curl http://localhost:8080/api/v1/workloads/api-1

# Step 2: Check group membership
curl http://localhost:8080/api/v1/groups/frontend-tier/members
curl http://localhost:8080/api/v1/groups/backend-tier/members

# Step 3: List all policy rules
curl http://localhost:8080/api/v1/policy-rules

# Step 4: Check compiled policies for relevant rule
curl http://localhost:8080/api/v1/policy-rules/1/compiled | \
  jq '.policies[] | select(.from_workload_id == "web-1" and .to_workload_id == "api-1")'

# Expected output should show compiled policies for this connection
# If no policies found, the rule may not be compiled correctly
```

---

### Example 4: Port Range Policy

This example creates a policy allowing a range of ports.

```bash
# Create workloads
curl -X POST http://localhost:8080/api/v1/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "id": "client-1",
    "name": "Client 1",
    "ips": ["10.0.5.10"],
    "labels": {"role": "client"}
  }'

curl -X POST http://localhost:8080/api/v1/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "id": "server-1",
    "name": "Server 1",
    "ips": ["10.0.6.20"],
    "labels": {"role": "server"}
  }'

# Create groups
curl -X POST http://localhost:8080/api/v1/groups \
  -H "Content-Type: application/json" \
  -d '{"name": "clients", "match_labels": {"role": "client"}}'

curl -X POST http://localhost:8080/api/v1/groups \
  -H "Content-Type: application/json" \
  -d '{"name": "servers", "match_labels": {"role": "server"}}'

# Create policy with port range 8000-8010
curl -X POST http://localhost:8080/api/v1/policy-rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "client-to-server-range",
    "description": "Allow clients to access server port range",
    "from_group": "clients",
    "to_group": "servers",
    "ports": [
      {"start": 8000, "end": 8010, "protocol": "tcp"}
    ],
    "action": "allow",
    "priority": 100,
    "enabled": true
  }'

# Check compiled policies
# Expected: 11 policies (one for each port 8000-8010)
curl http://localhost:8080/api/v1/policy-rules/3/compiled | jq '.count'
```

---

## Best Practices

### 1. Label Design

- Use consistent naming conventions for labels
- Use hierarchical labels for better organization (e.g., `app`, `tier`, `env`, `version`)
- Avoid using too many labels per workload (recommended: 3-10 labels)

### 2. Group Design

- Create groups based on logical tiers or functions
- Use specific label selectors to avoid unintended membership
- Regularly review group membership to ensure correctness

### 3. Policy Rule Design

- Use descriptive names and descriptions for rules
- Keep port lists concise (avoid large port ranges when possible)
- Monitor compilation warnings for large N×M expansions
- Use priority values consistently across rules

### 4. Performance Considerations

- **Compilation Warnings:**
  - Warning threshold: 1,000 compiled policies
  - Critical threshold: 10,000 compiled policies
- Monitor group sizes to avoid excessive N×M expansion
- Consider splitting large groups into smaller, more specific groups

### 5. Monitoring and Observability

- Use the `/compiled` endpoint to verify policy expansion
- Check compiled policy counts regularly
- Review group membership after workload changes
- Use traceability fields for debugging

---

## API Versioning

Current API version: **v1**

The API version is included in the URL path: `/api/v1/...`

Future versions will use `/api/v2/...`, `/api/v3/...`, etc.

---

## Rate Limiting

Currently, no rate limiting is implemented. This may be added in future versions.

---

## WebSocket Support

WebSocket support for real-time updates is not currently implemented but is planned for future versions to support:
- Real-time group membership changes
- Policy compilation notifications
- Workload lifecycle events

---

## Additional Resources

- [Architecture Overview](code-architecture-guide.md)
- [Label Auto-Acquisition](label-auto-acquisition.md)
- [OpenSpec Proposal](../openspec/changes/add-label-based-policy/design.md)
