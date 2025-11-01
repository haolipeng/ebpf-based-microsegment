// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ebpf-microsegment/src/agent/pkg/api/models"
	"github.com/ebpf-microsegment/src/agent/pkg/dataplane"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// MockDataPlaneForStats is a mock implementation of DataPlaneInterface for statistics testing
type MockDataPlaneForStats struct {
	statistics dataplane.Statistics
}

func NewMockDataPlaneForStats() *MockDataPlaneForStats {
	return &MockDataPlaneForStats{
		statistics: dataplane.Statistics{
			TotalPackets:   1000,
			AllowedPackets: 800,
			DeniedPackets:  200,
			NewSessions:    50,
			ClosedSessions: 30,
			ActiveSessions: 20,
			PolicyHits:     900,
			PolicyMisses:   100,
		},
	}
}

func (m *MockDataPlaneForStats) GetStatistics() dataplane.Statistics {
	return m.statistics
}

func (m *MockDataPlaneForStats) SetStatistics(stats dataplane.Statistics) {
	m.statistics = stats
}

// setupStatsTestRouter creates a test router with statistics handler
func setupStatsTestRouter(dp *MockDataPlaneForStats) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewStatisticsHandler(dp)

	router.GET("/api/v1/stats", handler.GetAllStats)
	router.GET("/api/v1/stats/packets", handler.GetPacketStats)
	router.GET("/api/v1/stats/sessions", handler.GetSessionStats)
	router.GET("/api/v1/stats/policies", handler.GetPolicyStats)

	return router
}

// TestGetAllStats_Success tests successful retrieval of all statistics
func TestGetAllStats_Success(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.StatisticsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify all statistics fields
	assert.Equal(t, uint64(1000), response.TotalPackets)
	assert.Equal(t, uint64(800), response.AllowedPackets)
	assert.Equal(t, uint64(200), response.DeniedPackets)
	assert.Equal(t, uint64(50), response.NewSessions)
	assert.Equal(t, uint64(30), response.ClosedSessions)
	assert.Equal(t, uint64(20), response.ActiveSessions)
	assert.Equal(t, uint64(900), response.PolicyHits)
	assert.Equal(t, uint64(100), response.PolicyMisses)
}

// TestGetAllStats_ZeroValues tests statistics with all zero values
func TestGetAllStats_ZeroValues(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	mockDP.SetStatistics(dataplane.Statistics{
		TotalPackets:   0,
		AllowedPackets: 0,
		DeniedPackets:  0,
		NewSessions:    0,
		ClosedSessions: 0,
		ActiveSessions: 0,
		PolicyHits:     0,
		PolicyMisses:   0,
	})
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.StatisticsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify all fields are zero
	assert.Equal(t, uint64(0), response.TotalPackets)
	assert.Equal(t, uint64(0), response.AllowedPackets)
	assert.Equal(t, uint64(0), response.DeniedPackets)
	assert.Equal(t, uint64(0), response.NewSessions)
	assert.Equal(t, uint64(0), response.ClosedSessions)
	assert.Equal(t, uint64(0), response.ActiveSessions)
	assert.Equal(t, uint64(0), response.PolicyHits)
	assert.Equal(t, uint64(0), response.PolicyMisses)
}

// TestGetAllStats_HighValues tests statistics with large values
func TestGetAllStats_HighValues(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	mockDP.SetStatistics(dataplane.Statistics{
		TotalPackets:   10000000,
		AllowedPackets: 9500000,
		DeniedPackets:  500000,
		NewSessions:    100000,
		ClosedSessions: 80000,
		ActiveSessions: 20000,
		PolicyHits:     9900000,
		PolicyMisses:   100000,
	})
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.StatisticsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify high values
	assert.Equal(t, uint64(10000000), response.TotalPackets)
	assert.Equal(t, uint64(9500000), response.AllowedPackets)
	assert.Equal(t, uint64(500000), response.DeniedPackets)
}

// TestGetPacketStats_Success tests successful packet statistics retrieval with rates
func TestGetPacketStats_Success(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/packets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PacketStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify packet statistics
	assert.Equal(t, uint64(1000), response.TotalPackets)
	assert.Equal(t, uint64(800), response.AllowedPackets)
	assert.Equal(t, uint64(200), response.DeniedPackets)

	// Verify calculated rates
	// AllowRate = 800/1000 * 100 = 80.0
	// DenyRate = 200/1000 * 100 = 20.0
	assert.InDelta(t, 80.0, response.AllowRate, 0.01)
	assert.InDelta(t, 20.0, response.DenyRate, 0.01)
}

// TestGetPacketStats_ZeroPackets tests packet statistics with zero packets
func TestGetPacketStats_ZeroPackets(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	mockDP.SetStatistics(dataplane.Statistics{
		TotalPackets:   0,
		AllowedPackets: 0,
		DeniedPackets:  0,
	})
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/packets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PacketStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify rates are zero when no packets
	assert.Equal(t, 0.0, response.AllowRate)
	assert.Equal(t, 0.0, response.DenyRate)
}

// TestGetPacketStats_AllAllowed tests packet statistics with all packets allowed
func TestGetPacketStats_AllAllowed(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	mockDP.SetStatistics(dataplane.Statistics{
		TotalPackets:   1000,
		AllowedPackets: 1000,
		DeniedPackets:  0,
	})
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/packets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PacketStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify 100% allow rate
	assert.InDelta(t, 100.0, response.AllowRate, 0.01)
	assert.InDelta(t, 0.0, response.DenyRate, 0.01)
}

// TestGetPacketStats_AllDenied tests packet statistics with all packets denied
func TestGetPacketStats_AllDenied(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	mockDP.SetStatistics(dataplane.Statistics{
		TotalPackets:   1000,
		AllowedPackets: 0,
		DeniedPackets:  1000,
	})
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/packets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PacketStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify 100% deny rate
	assert.InDelta(t, 0.0, response.AllowRate, 0.01)
	assert.InDelta(t, 100.0, response.DenyRate, 0.01)
}

// TestGetSessionStats_Success tests successful session statistics retrieval
func TestGetSessionStats_Success(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.SessionStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify session statistics
	assert.Equal(t, uint64(50), response.NewSessions)
	assert.Equal(t, uint64(30), response.ClosedSessions)
	assert.Equal(t, uint64(20), response.ActiveSessions)
}

// TestGetSessionStats_NoSessions tests session statistics with no sessions
func TestGetSessionStats_NoSessions(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	mockDP.SetStatistics(dataplane.Statistics{
		NewSessions:    0,
		ClosedSessions: 0,
		ActiveSessions: 0,
	})
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.SessionStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify all session counts are zero
	assert.Equal(t, uint64(0), response.NewSessions)
	assert.Equal(t, uint64(0), response.ClosedSessions)
	assert.Equal(t, uint64(0), response.ActiveSessions)
}

// TestGetSessionStats_ManyActiveSessions tests session statistics with many active sessions
func TestGetSessionStats_ManyActiveSessions(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	mockDP.SetStatistics(dataplane.Statistics{
		NewSessions:    10000,
		ClosedSessions: 8000,
		ActiveSessions: 2000,
	})
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.SessionStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify session counts
	assert.Equal(t, uint64(10000), response.NewSessions)
	assert.Equal(t, uint64(8000), response.ClosedSessions)
	assert.Equal(t, uint64(2000), response.ActiveSessions)
}

// TestGetPolicyStats_Success tests successful policy statistics retrieval with hit rate
func TestGetPolicyStats_Success(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/policies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PolicyStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify policy statistics
	assert.Equal(t, uint64(900), response.PolicyHits)
	assert.Equal(t, uint64(100), response.PolicyMisses)

	// Verify calculated hit rate
	// HitRate = 900/(900+100) * 100 = 90.0
	assert.InDelta(t, 90.0, response.HitRate, 0.01)
}

// TestGetPolicyStats_ZeroLookups tests policy statistics with no lookups
func TestGetPolicyStats_ZeroLookups(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	mockDP.SetStatistics(dataplane.Statistics{
		PolicyHits:   0,
		PolicyMisses: 0,
	})
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/policies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PolicyStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify hit rate is zero when no lookups
	assert.Equal(t, 0.0, response.HitRate)
}

// TestGetPolicyStats_PerfectHitRate tests policy statistics with 100% hit rate
func TestGetPolicyStats_PerfectHitRate(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	mockDP.SetStatistics(dataplane.Statistics{
		PolicyHits:   1000,
		PolicyMisses: 0,
	})
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/policies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PolicyStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify 100% hit rate
	assert.InDelta(t, 100.0, response.HitRate, 0.01)
}

// TestGetPolicyStats_ZeroHitRate tests policy statistics with 0% hit rate
func TestGetPolicyStats_ZeroHitRate(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlaneForStats()
	mockDP.SetStatistics(dataplane.Statistics{
		PolicyHits:   0,
		PolicyMisses: 1000,
	})
	router := setupStatsTestRouter(mockDP)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/stats/policies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PolicyStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify 0% hit rate
	assert.InDelta(t, 0.0, response.HitRate, 0.01)
}

// TestStatistics_ResponseStructure tests the JSON response structure for all endpoints
func TestStatistics_ResponseStructure(t *testing.T) {
	mockDP := NewMockDataPlaneForStats()
	router := setupStatsTestRouter(mockDP)

	testCases := []struct {
		name           string
		endpoint       string
		expectedFields []string
	}{
		{
			name:     "all stats response",
			endpoint: "/api/v1/stats",
			expectedFields: []string{
				"total_packets", "allowed_packets", "denied_packets",
				"new_sessions", "closed_sessions", "active_sessions",
				"policy_hits", "policy_misses",
			},
		},
		{
			name:     "packet stats response",
			endpoint: "/api/v1/stats/packets",
			expectedFields: []string{
				"total_packets", "allowed_packets", "denied_packets",
				"allow_rate", "deny_rate",
			},
		},
		{
			name:     "session stats response",
			endpoint: "/api/v1/stats/sessions",
			expectedFields: []string{
				"new_sessions", "closed_sessions", "active_sessions",
			},
		},
		{
			name:     "policy stats response",
			endpoint: "/api/v1/stats/policies",
			expectedFields: []string{
				"policy_hits", "policy_misses", "hit_rate",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute
			req, _ := http.NewRequest(http.MethodGet, tc.endpoint, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusOK, w.Code)

			// Verify JSON structure
			var jsonMap map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &jsonMap)
			assert.NoError(t, err)

			// Verify all expected fields are present
			for _, field := range tc.expectedFields {
				assert.Contains(t, jsonMap, field, "Missing field: %s", field)
			}
		})
	}
}

// TestStatistics_ContentType tests that all endpoints return JSON
func TestStatistics_ContentType(t *testing.T) {
	mockDP := NewMockDataPlaneForStats()
	router := setupStatsTestRouter(mockDP)

	endpoints := []string{
		"/api/v1/stats",
		"/api/v1/stats/packets",
		"/api/v1/stats/sessions",
		"/api/v1/stats/policies",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		})
	}
}
