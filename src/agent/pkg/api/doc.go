// Package api provides a RESTful HTTP API server for managing the eBPF-based
// microsegmentation system.
//
// The API server exposes endpoints for:
//   - Policy management (create, read, update, delete)
//   - Real-time statistics queries (packets, sessions, policies)
//   - Health checks and system status monitoring
//   - Configuration management
//
// # Architecture
//
// The API server is built on the Gin web framework and integrates with:
//   - eBPF data plane for packet processing
//   - Policy manager for policy CRUD operations
//   - Statistics collector for performance metrics
//
// # Example Usage
//
// Basic server setup:
//
//	cfg := api.DefaultConfig()
//	cfg.Port = 8080
//	cfg.EnableCORS = true
//
//	server, err := api.NewAPIServer(cfg, dataPlane, policyManager)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if err := server.Start(); err != nil {
//	    log.Fatal(err)
//	}
//	defer server.Stop()
//
// # Endpoints
//
// Health check:
//   - GET /api/v1/health  - Simple health check
//   - GET /api/v1/status  - Detailed system status
//
// Policy management:
//   - POST   /api/v1/policies     - Create policy
//   - GET    /api/v1/policies     - List all policies
//   - GET    /api/v1/policies/:id - Get specific policy
//   - PUT    /api/v1/policies/:id - Update policy
//   - DELETE /api/v1/policies/:id - Delete policy
//
// Statistics:
//   - GET /api/v1/stats          - All statistics
//   - GET /api/v1/stats/packets  - Packet statistics
//   - GET /api/v1/stats/sessions - Session statistics
//   - GET /api/v1/stats/policies - Policy statistics
//
// # Configuration
//
// Server configuration can be customized:
//
//	cfg := &api.Config{
//	    Host:         "127.0.0.1",
//	    Port:         8080,
//	    ReadTimeout:  10 * time.Second,
//	    WriteTimeout: 10 * time.Second,
//	    EnableCORS:   true,
//	    LogLevel:     "info",
//	}
//
// # Middleware
//
// The server includes the following middleware:
//   - Recovery: Catches panics and prevents server crashes
//   - Logger: Logs all HTTP requests with timing information
//   - CORS: Enables cross-origin resource sharing for web UIs
//
// # Thread Safety
//
// The API server is designed to handle concurrent requests safely.
// All operations on the eBPF data plane and policy manager are thread-safe.
package api

