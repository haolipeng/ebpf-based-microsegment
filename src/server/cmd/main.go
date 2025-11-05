package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	agentpb "github.com/ebpf-microsegment/src/proto/agent"
	flowpb "github.com/ebpf-microsegment/src/proto/flow"
	policypb "github.com/ebpf-microsegment/src/proto/policy"
	"github.com/ebpf-microsegment/src/server/pkg/config"
	grpcserv "github.com/ebpf-microsegment/src/server/pkg/grpc"
	"github.com/ebpf-microsegment/src/server/pkg/storage"
)

func main() {
	configPath := flag.String("config", "config/server.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Setup logging
	setupLogging(cfg.Log)

	logrus.Info("Starting microsegment-server...")
	logrus.Infof("HTTP API will listen on %s:%d", cfg.Server.Host, cfg.Server.Port)
	logrus.Infof("gRPC server will listen on %s:%d", cfg.GRPC.Host, cfg.GRPC.Port)

	// Initialize database
	db, err := storage.NewPostgresDB(
		cfg.Database.GetDSN(),
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
	)
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize schema
	if err := storage.InitSchema(db); err != nil {
		logrus.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create storage layers
	flowStorage := storage.NewFlowStorage(db)
	policyStorage := storage.NewPolicyStorage(db)
	agentStorage := storage.NewAgentStorage(db)

	// Start gRPC server
	grpcServer := startGRPCServer(cfg, flowStorage, policyStorage, agentStorage)
	defer grpcServer.GracefulStop()

	// Start HTTP API server
	httpServer := startHTTPServer(cfg, flowStorage, policyStorage, agentStorage)

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	logrus.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logrus.Errorf("HTTP server shutdown error: %v", err)
	}
	grpcServer.GracefulStop()

	logrus.Info("Server stopped")
}

func setupLogging(cfg config.LogConfig) {
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	if cfg.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
}

func startGRPCServer(cfg *config.Config, flowStorage *storage.FlowStorage, policyStorage *storage.PolicyStorage, agentStorage *storage.AgentStorage) *grpc.Server {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port))
	if err != nil {
		logrus.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register gRPC services
	flowpb.RegisterFlowServiceServer(grpcServer, grpcserv.NewFlowServiceServer(flowStorage))
	policypb.RegisterPolicyServiceServer(grpcServer, grpcserv.NewPolicyServiceServer(policyStorage))
	agentpb.RegisterAgentServiceServer(grpcServer, grpcserv.NewAgentServiceServer(agentStorage))

	go func() {
		logrus.Infof("gRPC server listening on %s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
		if err := grpcServer.Serve(lis); err != nil {
			logrus.Fatalf("gRPC server failed: %v", err)
		}
	}()

	return grpcServer
}

func startHTTPServer(cfg *config.Config, flowStorage *storage.FlowStorage, policyStorage *storage.PolicyStorage, agentStorage *storage.AgentStorage) *http.Server {
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "microsegment-server",
			"version": "1.0.0-mvp",
		})
	})

	// API routes (simplified for MVP)
	api := router.Group("/api/v1")
	{
		// Agent management
		api.GET("/agents", func(c *gin.Context) {
			agents, err := agentStorage.GetAllAgents(c.Request.Context())
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, agents)
		})

		// Flow queries (placeholder)
		api.GET("/flows", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Flow query API - coming soon"})
		})

		// Policy management (placeholder)
		api.GET("/policies", func(c *gin.Context) {
			policies, version, err := policyStorage.GetAllPolicies(c.Request.Context())
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{
				"policies": policies,
				"version":  version,
			})
		})
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		logrus.Infof("HTTP API server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("HTTP server failed: %v", err)
		}
	}()

	return srv
}
