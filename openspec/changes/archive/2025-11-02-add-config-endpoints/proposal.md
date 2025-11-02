# Proposal: Add Configuration Endpoints

## Summary
Add runtime configuration management endpoints to the control plane API, allowing administrators to dynamically adjust log levels and statistics intervals without restarting the agent.

## Why
Operators need the ability to change logging verbosity and monitoring intervals at runtime for effective troubleshooting and operational flexibility. Requiring an agent restart for these changes causes service disruption and increases mean time to resolution (MTTR) during incidents. This change enables zero-downtime configuration adjustments for non-structural settings.

## Motivation
Currently, the microsegmentation agent requires a restart to change configuration settings like log level or statistics reporting interval. This creates operational friction during troubleshooting and monitoring. Runtime configuration management enables:

1. **Dynamic Debugging**: Enable debug logging on-demand without service interruption
2. **Adaptive Monitoring**: Adjust statistics intervals based on operational needs
3. **Operational Flexibility**: Fine-tune agent behavior in production environments

## Scope
This change adds two new API endpoints to the control plane API:

- `GET /api/v1/config` - Retrieve current runtime configuration
- `PUT /api/v1/config` - Update mutable configuration fields

### In Scope
- Runtime log level changes (debug, info, warn, error)
- Statistics interval updates (1-300 seconds)
- Configuration validation and error handling
- Comprehensive test coverage

### Out of Scope
- Network interface changes (requires restart)
- API server host/port changes (requires restart)
- Configuration persistence to disk (future enhancement)
- Configuration history/audit logging (future enhancement)

## Proposed Solution
Implement a new `ConfigHandler` in the control plane API that:

1. Maintains runtime configuration state
2. Validates configuration updates against defined constraints
3. Applies changes immediately (log level) or signals changes (stats interval)
4. Returns appropriate HTTP status codes for success/failure scenarios

## Implementation Status
**✅ COMPLETED** - This proposal documents already-implemented functionality.

The configuration endpoints were implemented in commit `50d5c63`:
- Handler implementation: `src/agent/pkg/api/handlers/config.go`
- Test suite: `src/agent/pkg/api/handlers/config_test.go` (12 tests)
- Router integration: `src/agent/pkg/api/router.go`

## Acceptance Criteria
- [x] GET endpoint returns current configuration
- [x] PUT endpoint validates input (log level, stats interval)
- [x] Log level changes apply immediately via logrus
- [x] Invalid inputs return 400 Bad Request with error details
- [x] Read-only fields (interface, API host/port) cannot be modified
- [x] Test coverage ≥ 90% for config handler
- [x] All tests pass
