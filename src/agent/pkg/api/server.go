// Package api provides the RESTful HTTP API server for managing
// the eBPF microsegmentation system. It exposes endpoints for policy
// management, statistics queries, health checks, and system configuration.
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/dataplane"
	"github.com/ebpf-microsegment/src/agent/pkg/policy"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Server represents the HTTP API server that provides RESTful endpoints
// for managing policies, querying statistics, and monitoring system health.
// It uses the Gin framework and integrates with the eBPF data plane.
type Server struct {
	config        *Config
	dataPlane     *dataplane.DataPlane
	policyManager *policy.PolicyManager
	httpServer    *http.Server
	router        *gin.Engine
}

// NewAPIServer creates and initializes a new API server instance.
// It sets up the Gin router, configures middleware, and registers all routes.
//
// Parameters:
//   - cfg: API server configuration (nil uses defaults)
//   - dp: Data plane instance for eBPF operations
//   - pm: Policy manager for policy CRUD
//
// Returns:
//   - *Server: Initialized server instance
//   - error: Error if initialization fails
func NewAPIServer(cfg *Config, dp *dataplane.DataPlane, pm *policy.PolicyManager) (*Server, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Set Gin mode based on log level
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()

	server := &Server{
		config:        cfg,
		dataPlane:     dp,
		policyManager: pm,
		router:        router,
	}

	// Setup routes and middleware
	server.setupMiddleware()
	server.setupRoutes()

	return server, nil
}

// Start starts the HTTP server in a background goroutine.
// The server will listen on the configured host and port.
// This method returns immediately; the server runs asynchronously.
//
// Returns:
//   - error: Error if server fails to start
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	log.Infof("Starting API server on %s", addr)

	// Start server in goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP server.
// It waits for in-flight requests to complete (up to 30 seconds).
// After the timeout, the server will forcefully shutdown.
//
// Returns:
//   - error: Error if shutdown fails or times out
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}

	log.Info("Shutting down API server...")

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Errorf("API server forced to shutdown: %v", err)
		return err
	}

	log.Info("API server stopped gracefully")
	return nil
}

// GetRouter returns the underlying Gin router instance.
// This is primarily useful for testing purposes to inject
// test HTTP requests without starting the full HTTP server.
//
// Returns:
//   - *gin.Engine: The Gin router instance
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}
