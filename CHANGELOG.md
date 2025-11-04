# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added - Flow Collection API (Phase 1-3) - 2025-11-04

#### eBPF Data Plane
- **New**: 48-byte packed `struct flow_event` in `common_types.h`
- **New**: `push_flow_event()` helper function for event pushing
- **New**: Flow event types: `FLOW_EVENT_NEW`, `FLOW_EVENT_UPDATE`, `FLOW_EVENT_CLOSED`, `FLOW_EVENT_TIMEOUT`
- **New**: Flow direction enum: `FLOW_DIRECTION_INGRESS`, `FLOW_DIRECTION_EGRESS`
- **New**: Flow state enum: `FLOW_STATE_ACTIVE`, `FLOW_STATE_CLOSED`, `FLOW_STATE_TIMEOUT`
- **Enhanced**: Ring Buffer (256KB) for flow events to user-space
- **Enhanced**: `create_session()` now pushes flow events with policy ID

#### Go Control Plane
- **New**: `pkg/flow/` package with complete flow management
  - `types.go` - Core data structures (350 lines)
  - `collector.go` - Flow event collector (380 lines)
  - `storage.go` - SQLite persistence layer (575 lines)
  - `aggregator.go` - Flow aggregation and analysis (150 lines)
  - `types_test.go` - Unit tests with 100% coverage (380 lines)
- **New**: Flow Collector features:
  - Ring Buffer event reading
  - Active flow tracking with mutex protection
  - Workload label enrichment interface
  - Automatic cleanup of inactive flows (configurable timeout)
  - Graceful shutdown with flow flushing
- **New**: SQLite Storage features:
  - Optimized configuration (WAL mode, 64MB cache)
  - 8 indexes for fast queries
  - Complex filtering support (time, IP, protocol, labels)
  - Pagination and sorting
  - Aggregation queries (summary, top talkers)
  - Automatic old flow deletion

#### API Layer
- **New**: 7 Flow API endpoints in `pkg/api/handlers/flow.go` (413 lines):
  - `GET /api/v1/flows` - Query flows with filtering and pagination
  - `GET /api/v1/flows/:id` - Get single flow by ID
  - `GET /api/v1/flows/summary` - Flow statistics summary
  - `GET /api/v1/flows/active` - Get active flows from collector
  - `GET /api/v1/flows/metrics` - Collector performance metrics
  - `GET /api/v1/flows/dependencies` - Application dependency mapping
  - `GET /api/v1/flows/top-talkers` - Top N source IPs by traffic
- **New**: API models in `pkg/api/models/flow.go`
- **Enhanced**: `pkg/api/server.go` with `SetFlowComponents()` method
- **Enhanced**: `pkg/api/router.go` with conditional Flow route registration

#### Documentation
- **New**: [Flow Quick Start Guide](docs/flow-quick-start.md) - 10-minute tutorial
- **New**: [Flow Implementation Summary](docs/flow-collection-implementation-summary.md) - 32,000-word technical documentation
- **New**: [Flow Progress Report](docs/flow-implementation-progress.md) - Status and roadmap
- **New**: OpenSpec proposal and design documents in `openspec/changes/add-flow-collection-api/`
- **Updated**: README.md with Flow Collection API section

#### Testing
- **New**: Comprehensive unit tests for `types.go` with 100% coverage
- **New**: Performance benchmarks for flow parsing and conversion
- **Validated**: All eBPF code passes verifier
- **Validated**: All Go code compiles without errors

#### Metrics & Performance
- Implemented collector metrics: events processed, dropped, active flows, drop rate
- Target performance: 10,000 flows/s (not yet tested)
- Target API latency: <100ms for 1000 records (not yet tested)

### Pending (Phase 4-5)
- [ ] WebSocket Hub for real-time flow streaming
- [ ] DataPlane Ring Buffer integration
- [ ] WorkloadManager integration for label enrichment
- [ ] Main program integration
- [ ] Integration tests (target 80% coverage)
- [ ] Performance tests (10K flows/s)
- [ ] Prometheus metrics export
- [ ] Configuration file support

---

## [0.2.0] - 2024-11-XX (Previous Release)

### Added
- Session tracking with LRU hash maps
- Policy management API
- Wildcard policy support
- Statistics collection

### Changed
- Optimized eBPF program performance
- Improved policy lookup algorithm

---

## [0.1.0] - 2024-10-XX (Initial Release)

### Added
- Basic eBPF TC program
- 5-tuple flow matching
- Simple ALLOW/DENY policies
- Command-line agent
- REST API server

---

**Note**: This project is under active development. The Flow Collection API is currently at Phase 1-3 (60% complete). WebSocket real-time streaming and production optimizations are planned for Phase 4-5.
