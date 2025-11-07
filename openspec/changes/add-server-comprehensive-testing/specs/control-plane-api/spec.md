# Spec Delta: Control Plane API Testing

This delta adds comprehensive testing requirements for the server control plane API.

---

## ADDED Requirements

### Requirement: Server Testing Infrastructure

The microsegment-server MUST have comprehensive automated testing coverage.

#### Scenario: Unit Tests for Storage Layer

**Given** the storage layer is implemented
**When** unit tests are executed
**Then** they MUST cover ≥70% of storage package code
**And** test PostgreSQL operations with mock/test databases
**And** verify data integrity constraints
**And** test error handling for invalid inputs

#### Scenario: Unit Tests for gRPC Services

**Given** gRPC services are implemented
**When** unit tests are executed
**Then** they MUST cover ≥70% of gRPC service code
**And** test client streaming behavior
**And** test server streaming behavior
**And** verify request/response transformations
**And** test error handling and status codes

#### Scenario: Unit Tests for HTTP API

**Given** HTTP API handlers are implemented
**When** unit tests are executed
**Then** they MUST cover ≥70% of HTTP handler code
**And** test all API endpoints
**And** verify JSON response formatting
**And** test query parameter parsing
**And** test HTTP status codes (200, 400, 404, 500)

### Requirement: Integration Testing

The microsegment-server MUST have integration tests validating component interactions.

#### Scenario: Database Integration

**Given** a test PostgreSQL instance is running
**When** integration tests execute
**Then** they MUST test complete database workflows
**And** verify data persistence across operations
**And** test concurrent write operations
**And** verify transaction isolation
**And** clean up test data after execution

#### Scenario: gRPC Integration

**Given** a test gRPC server is running
**When** integration tests execute with real clients
**Then** they MUST verify agent registration flow
**And** test flow event reporting end-to-end
**And** test policy subscription streaming
**And** verify network communication
**And** test connection handling (reconnect, timeout)

#### Scenario: HTTP API Integration

**Given** a test HTTP server is running
**When** integration tests execute API requests
**Then** they MUST verify complete request/response cycles
**And** test middleware execution (logging, CORS)
**And** verify database state changes
**And** test concurrent API requests
**And** verify error responses

### Requirement: End-to-End Testing

The microsegment-server MUST have E2E tests validating complete user workflows.

#### Scenario: Agent Flow Lifecycle

**Given** the server is running with database
**When** an agent connects and reports flows
**Then** flows MUST be persisted to database
**And** flows MUST be queryable via HTTP API
**And** WebSocket clients MUST receive real-time updates
**And** aggregated statistics MUST reflect new flows
**And** the workflow completes in <5 seconds

#### Scenario: Policy Distribution

**Given** the server is running with agents subscribed
**When** a new policy is created
**Then** the policy version MUST increment
**And** all subscribed agents MUST receive the update
**And** the update MUST contain correct policy data
**And** agents receive updates within 1 second

### Requirement: Performance Testing

The microsegment-server MUST have performance benchmarks validating scalability targets.

#### Scenario: Flow Ingestion Throughput

**Given** the server is running
**When** flows are reported at high rate
**Then** the server MUST handle ≥10,000 flows/second
**And** maintain <100ms average processing latency
**And** not exceed 500MB memory usage
**And** maintain CPU usage <80%

#### Scenario: Query Performance

**Given** the database contains 1,000,000 flows
**When** querying flows with filters
**Then** queries MUST return in <100ms for 1000 records
**And** pagination MUST not degrade with offset
**And** JSONB label queries MUST use GIN indexes
**And** aggregation queries MUST complete in <500ms

#### Scenario: Concurrent Connections

**Given** the server is running
**When** 1000+ agents connect concurrently
**Then** the server MUST accept all connections
**And** maintain <10ms gRPC latency per connection
**And** not exceed 1GB memory usage
**And** gracefully handle connection churn

### Requirement: Test Infrastructure

The project MUST provide testing utilities and CI/CD integration.

#### Scenario: Test Utilities

**Given** developers are writing tests
**When** they need test fixtures
**Then** a `pkg/testutil` package MUST provide:
- Mock flow event factories
- Mock policy factories
- Mock agent registration factories
- Test database helpers
- Test assertion utilities

#### Scenario: CI Pipeline

**Given** code changes are pushed to repository
**When** CI pipeline executes
**Then** it MUST run all unit tests
**And** run all integration tests with PostgreSQL service
**And** generate coverage report
**And** fail if coverage drops below 60%
**And** block merge if tests fail
**And** complete in <10 minutes

---

## Success Metrics

- Unit test coverage: ≥70% for storage/grpc/api packages
- Integration tests: All passing with real services
- E2E tests: Complete workflows validated
- Performance: Benchmarks meet targets (10K flows/s, <100ms queries)
- CI: Tests run on every commit, coverage tracked
