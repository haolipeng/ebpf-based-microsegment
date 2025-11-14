package api

import (
	"github.com/haolipeng/ebpf-based-microsegment/pkg/api/handlers"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/flow"
	log "github.com/sirupsen/logrus"
)

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Create handlers
	healthHandler := handlers.NewHealthHandler(s.dataPlane, s.policyManager, s.config.Version, s.config.Interface)
	policyHandler := handlers.NewPolicyHandler(s.policyManager)
	statsHandler := handlers.NewStatisticsHandler(s.dataPlane)
	configHandler := handlers.NewConfigHandler(
		s.config.Interface,
		s.config.LogLevel,
		s.config.StatsInterval,
		s.config.Host,
		s.config.Port,
	)

	// API v1 group
	v1 := s.router.Group("/api/v1")
	{
		// Health and status endpoints
		v1.GET("/health", healthHandler.GetHealth)
		v1.GET("/status", healthHandler.GetStatus)

		// Policy management endpoints (low-level IP-based policies)
		policies := v1.Group("/policies")
		{
			policies.POST("", policyHandler.CreatePolicy)
			policies.GET("", policyHandler.ListPolicies)
			policies.GET("/:id", policyHandler.GetPolicy)
			policies.PUT("/:id", policyHandler.UpdatePolicy)
			policies.DELETE("/:id", policyHandler.DeletePolicy)
		}

		// Workload management endpoints
		if s.workloadMgr != nil {
			workloadHandler := handlers.NewWorkloadHandler(s.workloadMgr)
			workloads := v1.Group("/workloads")
			{
				workloads.POST("", workloadHandler.CreateWorkload)
				workloads.GET("", workloadHandler.ListWorkloads)
				workloads.GET("/:id", workloadHandler.GetWorkload)
				workloads.PUT("/:id", workloadHandler.UpdateWorkload)
				workloads.DELETE("/:id", workloadHandler.DeleteWorkload)
			}
		} else {
			log.Debug("Workload manager not configured, workload endpoints disabled")
		}

		// Group management endpoints
		if s.groupMgr != nil {
			groupHandler := handlers.NewGroupHandler(s.groupMgr)
			groups := v1.Group("/groups")
			{
				groups.POST("", groupHandler.CreateGroup)
				groups.GET("", groupHandler.ListGroups)
				groups.GET("/:name", groupHandler.GetGroup)
				groups.GET("/:name/members", groupHandler.GetGroupMembers)
				groups.PUT("/:name", groupHandler.UpdateGroup)
				groups.DELETE("/:name", groupHandler.DeleteGroup)
			}
		} else {
			log.Debug("Group manager not configured, group endpoints disabled")
		}

		// Policy rule management endpoints (high-level label-based policies)
		if s.policyManager != nil {
			policyRuleHandler := handlers.NewPolicyRuleHandler(s.policyManager)
			policyRules := v1.Group("/policy-rules")
			{
				policyRules.POST("", policyRuleHandler.CreatePolicyRule)
				policyRules.GET("", policyRuleHandler.ListPolicyRules)
				policyRules.GET("/:id", policyRuleHandler.GetPolicyRule)
				policyRules.GET("/:id/compiled", policyRuleHandler.GetCompiledPolicies)
				policyRules.PUT("/:id", policyRuleHandler.UpdatePolicyRule)
				policyRules.DELETE("/:id", policyRuleHandler.DeletePolicyRule)
			}
		}

		// Statistics endpoints
		stats := v1.Group("/stats")
		{
			stats.GET("", statsHandler.GetAllStats)
			stats.GET("/packets", statsHandler.GetPacketStats)
			stats.GET("/sessions", statsHandler.GetSessionStats)
			stats.GET("/policies", statsHandler.GetPolicyStats)
		}

		// Configuration endpoints
		config := v1.Group("/config")
		{
			config.GET("", configHandler.GetConfig)
			config.PUT("", configHandler.UpdateConfig)
		}

		// Flow collection endpoints (conditionally enabled)
		// Note: Flow components are set via SetFlowComponents() if flow collection is enabled
		if s.flowCollector != nil && s.flowStorage != nil {
			// Type assertions to convert interfaces to concrete types
			collector, ok1 := s.flowCollector.(*flow.Collector)
			storage, ok2 := s.flowStorage.(flow.Storage)

			if ok1 && ok2 {
				flowHandler := handlers.NewFlowHandler(collector, storage)
				flows := v1.Group("/flows")
				{
					flows.GET("", flowHandler.ListFlows)                    // List/query flows
					flows.GET("/:id", flowHandler.GetFlow)                  // Get single flow
					flows.GET("/summary", flowHandler.GetFlowSummary)       // Flow statistics summary
					flows.GET("/active", flowHandler.GetActiveFlows)        // Active flows from collector
					flows.GET("/metrics", flowHandler.GetCollectorMetrics)  // Collector metrics
					flows.GET("/dependencies", flowHandler.GetDependencies) // Workload dependencies
					flows.GET("/top-talkers", flowHandler.GetTopTalkers)    // Top talkers analysis

					// WebSocket endpoints for real-time streaming
					if hub := collector.GetWebSocketHub(); hub != nil {
						streamHandler := handlers.NewFlowStreamHandler(hub)
						flows.GET("/stream", streamHandler.HandleWebSocket)      // WebSocket upgrade
						flows.GET("/stream/stats", streamHandler.GetHubStats)    // WebSocket hub statistics
						log.Debug("WebSocket flow streaming endpoints registered")
					}

					// Lifecycle management endpoints
					if lifecycleManager := collector.GetLifecycleManager(); lifecycleManager != nil {
						lifecycleHandler := handlers.NewLifecycleHandler(lifecycleManager)
						flows.GET("/lifecycle/stats", lifecycleHandler.GetLifecycleStats) // Lifecycle statistics
						log.Debug("Lifecycle management endpoints registered")
					}
				}
			}
		}
	}
}
