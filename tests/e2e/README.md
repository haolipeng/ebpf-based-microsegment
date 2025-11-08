# E2E Integration Tests

This directory contains end-to-end integration tests for the Flow Collection API using Docker Compose.

## Architecture

```
┌─────────┐     gRPC      ┌────────┐    PostgreSQL    ┌──────────┐
│  Agent  │ ────────────> │ Server │ ───────────────> │ Postgres │
└─────────┘               └────────┘                   └──────────┘
                              │
                              │ HTTP/WebSocket
                              ▼
                          ┌────────────┐
                          │ Test Runner│
                          └────────────┘
```

## Components

1. **PostgreSQL**: TimescaleDB database for flow storage
2. **Server**: Microsegment server (gRPC + HTTP + WebSocket)
3. **Agent**: Microsegment agent (simulated flow generator)
4. **Test Runner**: Automated test suite

## Running Tests

### Prerequisites

- Docker
- Docker Compose
- 8GB RAM minimum

### Quick Start

```bash
# Run all tests
cd tests/e2e
docker-compose up --build --abort-on-container-exit

# Check results
cat test-results/results.txt
```

### Individual Services

```bash
# Start only database and server
docker-compose up -d postgres server

# Check server health
curl http://localhost:8080/health

# View server logs
docker-compose logs -f server

# Stop all services
docker-compose down -v
```

## Test Scenarios

### Test 1: Server Health Check
- Verifies server is running and responding
- Endpoint: `GET /health`

### Test 2: Database Connectivity
- Validates PostgreSQL connection
- Checks database schema initialization

### Test 3: Flows Table Schema
- Verifies flows table exists
- Checks indexes are created

### Test 4: Agent → Server Flow Reporting
- Agent sends 50 synthetic flows
- Validates gRPC streaming works
- Checks batch processing

### Test 5: Flows in Database
- Queries database for received flows
- Validates data persistence

### Test 6: Flow List API
- Tests `GET /api/v1/flows`
- Validates pagination
- Checks response format

### Test 7: Flow Summary API
- Tests `GET /api/v1/flows/summary`
- Validates aggregated statistics

### Test 8: Dependencies API
- Tests `GET /api/v1/aggregator/dependencies`
- Validates label-based aggregation

### Test 9: Top Talkers API
- Tests `GET /api/v1/aggregator/top-talkers`
- Validates endpoint ranking

### Test 10: Label Enrichment
- Verifies SourceLabels and DestLabels are stored
- Validates JSONB data integrity

### Test 11: WebSocket Stats
- Tests `GET /api/v1/flows/stream/stats`
- Validates WebSocket Hub metrics

### Test 12: Data Integrity
- Checks for NULL values in critical fields
- Validates data consistency

## Test Results

Results are written to: `test-results/results.txt`

Example output:
```
E2E Integration Test Results
============================
Date: 2025-11-06 14:30:00
Server: http://server:8080

Tests Run: 12
Tests Passed: 12
Tests Failed: 0

Status: SUCCESS

Database Statistics:
- Total Flows: 50
- Flows with Labels: 50
```

## Troubleshooting

### Server won't start
```bash
# Check logs
docker-compose logs server

# Check database
docker-compose ps postgres
```

### No flows in database
```bash
# Check agent logs
docker-compose logs agent

# Manually query database
docker-compose exec postgres psql -U microsegment_user -d microsegment -c "SELECT COUNT(*) FROM flows"
```

### Tests failing
```bash
# Run tests with verbose output
docker-compose run test-runner /bin/bash -x /app/run-tests.sh
```

## Cleanup

```bash
# Stop and remove all containers, volumes
docker-compose down -v

# Remove images
docker-compose down --rmi all
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Run E2E tests
        run: |
          cd tests/e2e
          docker-compose up --build --abort-on-container-exit
          cat test-results/results.txt
      - name: Cleanup
        if: always()
        run: docker-compose down -v
```

## Future Enhancements

- [ ] Add WebSocket client test (real-time streaming)
- [ ] Add multi-node agent test
- [ ] Add failure recovery test (server restart)
- [ ] Add performance benchmarks
- [ ] Add TimescaleDB Hypertable tests
