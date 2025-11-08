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

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	agentpb "github.com/ebpf-microsegment/src/proto/agent"
	flowpb "github.com/ebpf-microsegment/src/proto/flow"
	policypb "github.com/ebpf-microsegment/src/proto/policy"
	"github.com/ebpf-microsegment/src/server/pkg/config"
	grpcserv "github.com/ebpf-microsegment/src/server/pkg/grpc"
	"github.com/ebpf-microsegment/src/server/pkg/storage"
	"github.com/ebpf-microsegment/src/server/pkg/api/handlers"
	"github.com/ebpf-microsegment/src/server/pkg/api/middleware"
	"github.com/ebpf-microsegment/src/server/pkg/aggregator"
	ws "github.com/ebpf-microsegment/src/server/pkg/websocket"
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

	// Create and start WebSocket hub for real-time flow streaming
	wsHub := ws.NewHub()
	go wsHub.Run()
	logrus.Info("WebSocket hub started")

	// Create aggregator for flow analysis
	flowAggregator := aggregator.NewFlowAggregator(db)
	logrus.Info("Flow aggregator initialized")

	// Start gRPC server
	grpcServer := startGRPCServer(cfg, flowStorage, policyStorage, agentStorage, wsHub)
	defer grpcServer.GracefulStop()

	// Start HTTP API server
	httpServer := startHTTPServer(cfg, flowStorage, policyStorage, agentStorage, wsHub, flowAggregator)

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

func startGRPCServer(cfg *config.Config, flowStorage *storage.FlowStorage, policyStorage *storage.PolicyStorage, agentStorage *storage.AgentStorage, wsHub *ws.Hub) *grpc.Server {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port))
	if err != nil {
		logrus.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register gRPC services with WebSocket hub for real-time streaming
	flowpb.RegisterFlowServiceServer(grpcServer, grpcserv.NewFlowServiceServer(flowStorage, wsHub))
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

func startHTTPServer(cfg *config.Config, flowStorage *storage.FlowStorage, policyStorage *storage.PolicyStorage, agentStorage *storage.AgentStorage, wsHub *ws.Hub, flowAggregator *aggregator.FlowAggregator) *http.Server {
	router := gin.Default()

	// Global middleware
	router.Use(middleware.ErrorHandlerMiddleware())
	router.Use(middleware.RequestLoggerMiddleware())

	// Configure CORS to allow frontend access
	// For development, allow all origins and headers
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,  // Allow all origins for development
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"*"},  // Allow all headers
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,  // Must be false when AllowAllOrigins is true
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "microsegment-server",
			"version": "1.0.0-mvp",
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Agent management - use dedicated handler
		agentHandler := handlers.NewAgentHandler(agentStorage)
		agentHandler.RegisterRoutes(api)

		// Flow API - use dedicated handler
		flowHandler := handlers.NewFlowHandler(flowStorage)
		flowHandler.RegisterRoutes(api)

		// WebSocket real-time flow streaming
		flowStreamHandler := handlers.NewFlowStreamHandler(wsHub)
		flowStreamHandler.RegisterRoutes(api)

		// Aggregator for flow analysis and dependencies
		aggregatorHandler := handlers.NewAggregatorHandler(flowAggregator)
		aggregatorHandler.RegisterRoutes(api)

		// Policy management - use dedicated handler
		policyHandler := handlers.NewPolicyHandler(policyStorage)
		policyHandler.RegisterRoutes(api)
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
