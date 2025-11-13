// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package benchmark

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/api/handlers"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/api/models"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/dataplane"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/policy"
	"github.com/gin-gonic/gin"
)

// MockDataPlane for benchmarking
type BenchDataPlane struct {
	stats dataplane.Statistics
}

func NewBenchDataPlane() *BenchDataPlane {
	return &BenchDataPlane{
		stats: dataplane.Statistics{
			TotalPackets:   1000000,
			AllowedPackets: 800000,
			DeniedPackets:  200000,
			NewSessions:    5000,
			ClosedSessions: 3000,
			ActiveSessions: 2000,
			PolicyHits:     900000,
			PolicyMisses:   100000,
		},
	}
}

func (m *BenchDataPlane) GetStatistics() dataplane.Statistics {
	return m.stats
}

// MockPolicyManager for benchmarking
type BenchPolicyManager struct {
	policies map[uint32]*policy.Policy
}

func NewBenchPolicyManager() *BenchPolicyManager {
	pm := &BenchPolicyManager{
		policies: make(map[uint32]*policy.Policy),
	}

	// Pre-populate with some policies
	for i := uint32(1); i <= 100; i++ {
		pm.policies[i] = &policy.Policy{
			RuleID:   i,
			SrcIP:    "192.168.1.0/24",
			DstIP:    "10.0.0.0/8",
			DstPort:  80,
			Protocol: "tcp",
			Action:   "allow",
		}
	}

	return pm
}

func (m *BenchPolicyManager) AddPolicy(p *policy.Policy) error {
	m.policies[p.RuleID] = p
	return nil
}

func (m *BenchPolicyManager) DeletePolicy(p *policy.Policy) error {
	delete(m.policies, p.RuleID)
	return nil
}

func (m *BenchPolicyManager) ListPolicies() ([]policy.Policy, error) {
	policies := make([]policy.Policy, 0, len(m.policies))
	for _, p := range m.policies {
		policies = append(policies, *p)
	}
	return policies, nil
}

// setupRouter creates a test router with handlers for benchmarking
func setupRouter(b *testing.B) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	dp := NewBenchDataPlane()
	pm := NewBenchPolicyManager()

	healthHandler := handlers.NewHealthHandler(dp, pm)
	policyHandler := handlers.NewPolicyHandler(pm)
	statsHandler := handlers.NewStatisticsHandler(dp)
	configHandler := handlers.NewConfigHandler("lo", "error", 5, "127.0.0.1", 8080)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.GetHealth)
		v1.GET("/status", healthHandler.GetStatus)

		policies := v1.Group("/policies")
		{
			policies.POST("", policyHandler.CreatePolicy)
			policies.GET("", policyHandler.ListPolicies)
			policies.GET("/:id", policyHandler.GetPolicy)
			policies.PUT("/:id", policyHandler.UpdatePolicy)
			policies.DELETE("/:id", policyHandler.DeletePolicy)
		}

		stats := v1.Group("/stats")
		{
			stats.GET("", statsHandler.GetAllStats)
			stats.GET("/packets", statsHandler.GetPacketStats)
			stats.GET("/sessions", statsHandler.GetSessionStats)
			stats.GET("/policies", statsHandler.GetPolicyStats)
		}

		config := v1.Group("/config")
		{
			config.GET("", configHandler.GetConfig)
			config.PUT("", configHandler.UpdateConfig)
		}
	}

	return router
}

// BenchmarkHealthCheck measures health endpoint performance
func BenchmarkHealthCheck(b *testing.B) {
	router := setupRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/health", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkStatusCheck measures detailed status endpoint performance
func BenchmarkStatusCheck(b *testing.B) {
	router := setupRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/status", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkCreatePolicy measures policy creation performance
func BenchmarkCreatePolicy(b *testing.B) {
	router := setupRouter(b)

	policyReq := models.PolicyRequest{
		RuleID:   999,
		SrcIP:    "192.168.1.100",
		DstIP:    "10.0.0.50",
		DstPort:  443,
		Protocol: "tcp",
		Action:   "allow",
	}

	bodyBytes, _ := json.Marshal(policyReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/policies", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated && w.Code != http.StatusConflict {
			b.Fatalf("Expected 201 or 409, got %d", w.Code)
		}
	}
}

// BenchmarkListPolicies measures policy listing performance
func BenchmarkListPolicies(b *testing.B) {
	router := setupRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkGetPolicy measures single policy retrieval performance
func BenchmarkGetPolicy(b *testing.B) {
	router := setupRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/1", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkUpdatePolicy measures policy update performance
func BenchmarkUpdatePolicy(b *testing.B) {
	router := setupRouter(b)

	policyReq := models.PolicyRequest{
		RuleID:   1,
		SrcIP:    "192.168.2.0/24",
		DstIP:    "10.0.1.0/24",
		DstPort:  8080,
		Protocol: "tcp",
		Action:   "deny",
	}

	bodyBytes, _ := json.Marshal(policyReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/v1/policies/1", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkDeletePolicy measures policy deletion performance
func BenchmarkDeletePolicy(b *testing.B) {
	router := setupRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Re-add the policy before each delete
		policyReq := models.PolicyRequest{
			RuleID:   500,
			SrcIP:    "192.168.1.100",
			DstIP:    "10.0.0.50",
			DstPort:  443,
			Protocol: "tcp",
			Action:   "allow",
		}
		bodyBytes, _ := json.Marshal(policyReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/policies", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		b.StartTimer()

		w = httptest.NewRecorder()
		req, _ = http.NewRequest("DELETE", "/api/v1/policies/500", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			b.Fatalf("Expected 204, got %d", w.Code)
		}
	}
}

// BenchmarkGetAllStats measures statistics retrieval performance
func BenchmarkGetAllStats(b *testing.B) {
	router := setupRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/stats", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkGetPacketStats measures packet statistics retrieval performance
func BenchmarkGetPacketStats(b *testing.B) {
	router := setupRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/stats/packets", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkGetConfig measures configuration retrieval performance
func BenchmarkGetConfig(b *testing.B) {
	router := setupRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/config", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkUpdateConfig measures configuration update performance
func BenchmarkUpdateConfig(b *testing.B) {
	router := setupRouter(b)

	configReq := models.ConfigUpdateRequest{
		LogLevel: func() *string { s := "debug"; return &s }(),
	}

	bodyBytes, _ := json.Marshal(configReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkConcurrentHealthCheck measures health check performance under concurrent load
func BenchmarkConcurrentHealthCheck(b *testing.B) {
	router := setupRouter(b)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/health", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Errorf("Expected 200, got %d", w.Code)
			}
		}
	})
}

// BenchmarkConcurrentListPolicies measures policy listing under concurrent load
func BenchmarkConcurrentListPolicies(b *testing.B) {
	router := setupRouter(b)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/policies", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Errorf("Expected 200, got %d", w.Code)
			}
		}
	})
}

// BenchmarkConcurrentGetStats measures statistics query under concurrent load
func BenchmarkConcurrentGetStats(b *testing.B) {
	router := setupRouter(b)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/stats", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Errorf("Expected 200, got %d", w.Code)
			}
		}
	})
}
