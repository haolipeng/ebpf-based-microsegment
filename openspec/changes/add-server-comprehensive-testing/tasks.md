# Implementation Tasks: Server Comprehensive Testing

**Change ID**: `add-server-comprehensive-testing`
**Created**: 2025-11-06
**Estimated Effort**: 3-5 days

---

## Task Overview

| Phase | Tasks | Status | Est. Time |
|-------|-------|--------|-----------|
| Phase 1: Test Infrastructure | 3 tasks | ✅ COMPLETED | 0.5 day |
| Phase 2: Storage Unit Tests | 4 tasks | ✅ COMPLETED | 1 day |
| Phase 3: gRPC Service Tests | 3 tasks | ✅ COMPLETED | 1 day |
| Phase 4: HTTP API Tests | 3 tasks | ⏳ PENDING | 0.5 day |
| Phase 5: Integration Tests | 3 tasks | ⏳ PENDING | 1 day |
| Phase 6: E2E Tests | 2 tasks | ⏳ PENDING | 0.5 day |
| Phase 7: Performance Tests | 2 tasks | ⏳ PENDING | 0.5 day |
| **Total** | **20 tasks** | **50% Complete** | **5 days** |

---

## Phase 1: Test Infrastructure Setup (0.5 day) ✅ **COMPLETED**

### Task 1.1: Setup Test Dependencies ✅
- [x] Add testing dependencies to go.mod
  ```bash
  cd src/server
  go get github.com/stretchr/testify
  go get github.com/testcontainers/testcontainers-go
  ```
- [x] Create test utilities package
  ```bash
  mkdir -p pkg/testutil
  ```

### Task 1.2: Create Test Fixtures ✅
- [x] Create `pkg/testutil/fixtures.go`
  - [x] Mock flow event factory
  - [x] Mock policy factory
  - [x] Mock agent registration factory
  - [x] Test database helper functions
- [x] Create `pkg/testutil/database.go`
  - [x] TestDB container setup with testcontainers
  - [x] Database helper functions (TruncateAllTables, CountRows, etc.)
  - [x] Quick insert functions for test data
- [x] Create `pkg/testutil/assertions.go`
  - [x] Custom assertion functions
  - [x] Proto message equality checks

### Task 1.3: Setup CI Pipeline ✅
- [x] Create `.github/workflows/server-tests.yml`
- [x] Configure PostgreSQL service for CI
- [x] Add test coverage reporting
- [x] Add badge to README
- [x] Add comprehensive testing documentation to README

---

## Phase 2: Storage Layer Unit Tests (1 day) ✅ **COMPLETED**

### Task 2.1: Test PostgreSQL Connection ✅
- [x] Create `pkg/storage/postgres_test.go`
- [x] Test `NewPostgresDB()` with valid config
- [x] Test `NewPostgresDB()` with invalid config
- [x] Test connection pool behavior
- [x] Test `InitSchema()` creates all tables
- [x] Test `InitSchema()` is idempotent
- **Coverage**: 85% (NewPostgresDB: 70%, InitSchema: 100%)

### Task 2.2: Test FlowStorage ✅
- [x] Create `pkg/storage/flow_storage_test.go`
- [x] Test `BatchSaveFlowEvents()` single event
- [x] Test `BatchSaveFlowEvents()` batch of multiple events
- [x] Test `BatchSaveFlowEvents()` with transaction errors
- [x] Test `QueryFlows()` with time range filter
- [x] Test `QueryFlows()` with agent ID filter
- [x] Test `QueryFlows()` with protocol filter
- [x] Test `QueryFlows()` basic pagination
- [x] Test `GetFlowSummary()` aggregation
- [x] Test `GetFlowDependencies()` label grouping
- [x] Test `intToIP()` conversion utility
- **Coverage**: 94.5% (functions ranging from 87.5% to 100%)

### Task 2.3: Test PolicyStorage ✅
- [x] Create `pkg/storage/policy_storage_test.go`
- [x] Test `GetAllPolicies()` empty database
- [x] Test `GetAllPolicies()` with policies
- [x] Test `CreatePolicy()` success
- [x] Test `CreatePolicy()` error handling
- [x] Test `UpdatePolicy()` success
- [x] Test `UpdatePolicy()` not found
- [x] Test `DeletePolicy()` success
- [x] Test `DeletePolicy()` not found
- [x] Test version increment functionality
- **Coverage**: 99.2% (functions ranging from 95.2% to 100%)

### Task 2.4: Test AgentStorage ✅
- [x] Create `pkg/storage/agent_storage_test.go`
- [x] Test `RegisterAgent()` new agent
- [x] Test `RegisterAgent()` updates existing agent
- [x] Test `UpdateHeartbeat()` without metrics
- [x] Test `UpdateHeartbeat()` with metrics
- [x] Test `GetAllAgents()` returns all agents
- [x] Test `GetAgentByID()` success
- [x] Test `GetAgentByID()` not found
- **Coverage**: 98.6% (functions ranging from 92.9% to 100%)

**Phase 2 Summary**:
- **Total Test Files Created**: 4 (postgres_test.go, flow_storage_test.go, policy_storage_test.go, agent_storage_test.go)
- **Total Tests Written**: 48 unit tests
- **Overall Storage Package Coverage**: 94.1%
- **Dependencies Added**: github.com/DATA-DOG/go-sqlmock v1.5.2

---

## Phase 3: gRPC Service Tests (1 day) ✅ **COMPLETED**

### Task 3.1: Test FlowService ✅
- [x] Create `pkg/grpc/flow_service_test.go`
- [x] Test `ReportFlowEvents()` receives single event
- [x] Test `ReportFlowEvents()` receives batch
- [x] Test `ReportFlowEvents()` handles stream errors
- [x] Test `ReportFlowEvents()` returns correct statistics (rejection/validation)
- [x] Test `QueryFlows()` calls storage correctly
- [x] Test `GetFlowSummary()` calls storage correctly
- [x] Test `eventToFlow()` conversion utility
- [x] Test `intToIP()` conversion utility
- **Coverage**: flow_service.go: 91.1% (ReportFlowEvents: 88.9%, others: 100%)

### Task 3.2: Test PolicyService ✅
- [x] Create `pkg/grpc/policy_service_test.go`
- [x] Test `SyncPolicies()` returns all policies
- [x] Test `SyncPolicies()` returns current version
- [x] Test `SubscribePolicies()` sends initial policies
- [x] Test `SubscribePolicies()` handles storage errors
- [x] Test `ReportPolicyStats()` acknowledges receipt (MVP)
- **Coverage**: policy_service.go: 97.5% (SubscribePolicies: 90.0%, others: 100%)

### Task 3.3: Test AgentService ✅
- [x] Create `pkg/grpc/agent_service_test.go`
- [x] Test `RegisterAgent()` saves agent info
- [x] Test `RegisterAgent()` returns server config
- [x] Test `Heartbeat()` updates timestamp
- [x] Test `Heartbeat()` with/without metrics
- [x] Test `ReportStatus()` acknowledges receipt (MVP)
- [x] Test `UnregisterAgent()` graceful shutdown
- **Coverage**: agent_service.go: 100% (all functions)

**Phase 3 Summary**:
- **Total Test Files Created**: 3 (flow_service_test.go, policy_service_test.go, agent_service_test.go)
- **Total Tests Written**: 34 unit tests
- **Overall gRPC Package Coverage**: 94.6%
- **Testing Approach**: Mock-based testing using sqlmock for database operations, custom stream mocks for gRPC streaming

---

## Phase 4: HTTP API Tests (0.5 day)

### Task 4.1: Test Flow Handlers
- [ ] Create `pkg/api/handlers/flow_test.go`
- [ ] Test `ListFlows()` returns paginated results
- [ ] Test `ListFlows()` handles query parameters
- [ ] Test `GetFlow()` returns single flow
- [ ] Test `GetFlow()` returns 404 for missing flow
- [ ] Test `GetFlowSummary()` returns aggregated stats
- [ ] Test `GetFlowDependencies()` returns dependencies

### Task 4.2: Test Aggregator Handlers
- [ ] Create `pkg/api/handlers/aggregator_test.go`
- [ ] Test `GetDependencies()` returns dependency graph
- [ ] Test `GetTopTalkers()` returns ranked endpoints
- [ ] Test `GetAggregatedStats()` returns statistics

### Task 4.3: Test WebSocket Handlers
- [ ] Create `pkg/api/handlers/flow_stream_test.go`
- [ ] Test WebSocket connection upgrade
- [ ] Test real-time flow broadcasting
- [ ] Test client disconnection handling

---

## Phase 5: Integration Tests (1 day)

### Task 5.1: Database Integration Tests
- [ ] Create `pkg/integration/database_test.go`
- [ ] Setup testcontainers for PostgreSQL
- [ ] Test complete flow: save → query → aggregate
- [ ] Test concurrent writes
- [ ] Test transaction rollback
- [ ] Cleanup test containers

### Task 5.2: gRPC Integration Tests
- [ ] Create `pkg/integration/grpc_test.go`
- [ ] Start in-memory gRPC server
- [ ] Test Agent registration flow
- [ ] Test Flow reporting with real client
- [ ] Test Policy subscription with real client
- [ ] Cleanup server

### Task 5.3: HTTP Integration Tests
- [ ] Create `pkg/integration/http_test.go`
- [ ] Start test HTTP server
- [ ] Test complete API workflows
- [ ] Test error responses (400, 404, 500)
- [ ] Test concurrent API requests

---

## Phase 6: End-to-End Tests (0.5 day)

### Task 6.1: Complete Flow Lifecycle Test
- [ ] Create `tests/e2e/flow_lifecycle_test.go`
- [ ] Scenario: Agent connects → Reports flows → Query returns data
- [ ] Verify flows in database
- [ ] Verify WebSocket clients receive updates
- [ ] Verify API returns correct data

### Task 6.2: Policy Distribution Test
- [ ] Create `tests/e2e/policy_distribution_test.go`
- [ ] Scenario: Create policy → Agents subscribe → Receive update
- [ ] Verify policy version increments
- [ ] Verify all subscribers receive update
- [ ] Verify update content is correct

---

## Phase 7: Performance Tests (0.5 day)

### Task 7.1: Load Testing
- [ ] Create `tests/benchmark/load_test.go`
- [ ] Benchmark `BatchSaveFlowEvents()` throughput
  - Target: 10,000 flows/second
- [ ] Benchmark `QueryFlows()` latency
  - Target: <100ms for 1000 records
- [ ] Benchmark concurrent gRPC connections
  - Target: 1000+ concurrent agents
- [ ] Generate performance report

### Task 7.2: Memory and CPU Profiling
- [ ] Create `tests/benchmark/profile_test.go`
- [ ] Profile CPU usage under load
- [ ] Profile memory allocations
- [ ] Identify bottlenecks
- [ ] Document optimization opportunities

---

## Validation Criteria

### Coverage Targets
- [ ] Storage package: ≥70% coverage
- [ ] gRPC package: ≥70% coverage
- [ ] API handlers: ≥70% coverage
- [ ] Overall server: ≥60% coverage

### Test Pass Rate
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] All E2E tests pass
- [ ] Performance benchmarks meet targets

### CI/CD
- [ ] Tests run on every PR
- [ ] Coverage report generated
- [ ] Failed tests block merge

---

## Testing Best Practices

1. **Use table-driven tests** for multiple scenarios
2. **Mock external dependencies** in unit tests
3. **Use testcontainers** for integration tests with real services
4. **Clean up resources** in test teardown
5. **Test error cases** not just happy paths
6. **Keep tests fast**: unit <1s, integration <5s
7. **Make tests deterministic**: no flaky tests

---

## Documentation Updates

- [ ] Update `src/server/README.md` with testing instructions
- [ ] Add testing section to contributing guide
- [ ] Document how to run tests locally
- [ ] Document CI/CD pipeline

---

**Estimated Total Effort**: 5 days
**Dependencies**: Completed server implementation (archived)
**Next Steps**: Execute Phase 1 to setup test infrastructure
