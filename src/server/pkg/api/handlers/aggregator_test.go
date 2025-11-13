package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/aggregator"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAggregatorRouter creates a test Gin router with the aggregator handler
func setupAggregatorRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	agg := aggregator.NewFlowAggregator(db)
	handler := NewAggregatorHandler(agg)
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	return router
}

// TestNewAggregatorHandler tests the constructor
func TestNewAggregatorHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	agg := aggregator.NewFlowAggregator(db)
	handler := NewAggregatorHandler(agg)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.aggregator)
}

// TestGetDependencies_Success tests successful dependency retrieval
func TestGetDependencies_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	// Mock the dependencies query
	rows := sqlmock.NewRows([]string{
		"source_label", "dest_label", "flow_count", "total_bytes", "total_packets", "protocols", "avg_duration_ms",
	}).
		AddRow("web", "db", 100, 102400, 1000, []byte(`[6]`), 150.5).
		AddRow("web", "cache", 50, 51200, 500, []byte(`[6]`), 100.0)

	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnRows(rows)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/dependencies?group_by=app", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "app", response["group_by"])
	assert.NotNil(t, response["dependencies"])
	assert.NotNil(t, response["time_range"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetDependencies_Error tests error handling
func TestGetDependencies_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	// Mock database error
	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnError(sql.ErrConnDone)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/dependencies", nil)
	router.ServeHTTP(w, req)

	// Assert error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get dependencies")
}

// TestGetDependencies_WithTimeRange tests time range parsing
func TestGetDependencies_WithTimeRange(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)

	// Mock the dependencies query (empty result is fine for this test)
	rows := sqlmock.NewRows([]string{
		"source_label", "dest_label", "flow_count", "total_bytes", "total_packets", "protocols", "avg_duration_ms",
	})

	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnRows(rows)

	// Make request with time range
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/dependencies?start_time="+
		startTime.Format(time.RFC3339)+"&end_time="+endTime.Format(time.RFC3339), nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetTopTalkers_Success tests successful top talkers retrieval
func TestGetTopTalkers_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	// Mock top talkers by bytes query
	byBytesRows := sqlmock.NewRows([]string{
		"ip_address", "direction", "total_metric", "total_bytes", "total_packets", "flow_count", "labels",
	}).
		AddRow("192.168.1.100", "source", 1024000, 1024000, 1000, 150, []byte(`{"app":"web"}`)).
		AddRow("192.168.1.101", "destination", 512000, 512000, 500, 75, []byte(`{"app":"db"}`))

	// Mock top talkers by packets query
	byPacketsRows := sqlmock.NewRows([]string{
		"ip_address", "direction", "total_metric", "total_bytes", "total_packets", "flow_count", "labels",
	}).
		AddRow("192.168.1.100", "source", 1000, 1024000, 1000, 150, []byte(`{"app":"web"}`))

	// Mock top talkers by flow count query
	byFlowCountRows := sqlmock.NewRows([]string{
		"ip_address", "direction", "flow_count", "total_bytes", "total_packets", "labels",
	}).
		AddRow("192.168.1.100", "source", 150, 1024000, 1000, []byte(`{"app":"web"}`))

	mock.ExpectQuery("WITH source_stats AS").WillReturnRows(byBytesRows)
	mock.ExpectQuery("WITH source_stats AS").WillReturnRows(byPacketsRows)
	mock.ExpectQuery("WITH source_stats AS").WillReturnRows(byFlowCountRows)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/top-talkers", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["top_talkers"])
	assert.NotNil(t, response["time_range"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetTopTalkers_WithTopN tests top_n parameter
func TestGetTopTalkers_WithTopN(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	// Mock empty results for all three queries
	emptyRows := sqlmock.NewRows([]string{
		"ip_address", "direction", "total_metric", "total_bytes", "total_packets", "flow_count", "labels",
	})
	emptyFlowCountRows := sqlmock.NewRows([]string{
		"ip_address", "direction", "flow_count", "total_bytes", "total_packets", "labels",
	})

	mock.ExpectQuery("WITH source_stats AS").WillReturnRows(emptyRows)
	mock.ExpectQuery("WITH source_stats AS").WillReturnRows(emptyRows)
	mock.ExpectQuery("WITH source_stats AS").WillReturnRows(emptyFlowCountRows)

	// Make request with top_n parameter
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/top-talkers?top_n=5", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetTopTalkers_Error tests error handling
func TestGetTopTalkers_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	// Mock database error
	mock.ExpectQuery("WITH source_stats AS").
		WillReturnError(sql.ErrConnDone)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/top-talkers", nil)
	router.ServeHTTP(w, req)

	// Assert error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get top talkers")
}

// TestGetAggregatedStats_Success tests successful stats retrieval
func TestGetAggregatedStats_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	// Mock dependencies query
	depsRows := sqlmock.NewRows([]string{
		"source_label", "dest_label", "flow_count", "total_bytes", "total_packets", "protocols", "avg_duration_ms",
	}).AddRow("web", "db", 100, 102400, 1000, []byte(`[6]`), 150.5)

	// Mock summary query
	summaryRows := sqlmock.NewRows([]string{
		"total_flows", "total_bytes", "total_packets", "unique_endpoints", "avg_duration_ms",
	}).AddRow(1000, 10240000, 100000, 50, 150.5)

	mock.ExpectQuery("SELECT (.+) FROM flows").WillReturnRows(depsRows)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(summaryRows)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/stats", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.NotNil(t, result["dependencies"])
	assert.NotNil(t, result["summary"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetAggregatedStats_WithTopTalkers tests include_top_talkers parameter
func TestGetAggregatedStats_WithTopTalkers(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	// Mock dependencies query
	depsRows := sqlmock.NewRows([]string{
		"source_label", "dest_label", "flow_count", "total_bytes", "total_packets", "protocols", "avg_duration_ms",
	})

	// Mock top talkers queries (3 queries)
	emptyTalkerRows := sqlmock.NewRows([]string{
		"ip_address", "direction", "total_metric", "total_bytes", "total_packets", "flow_count", "labels",
	})
	emptyFlowCountRows := sqlmock.NewRows([]string{
		"ip_address", "direction", "flow_count", "total_bytes", "total_packets", "labels",
	})

	// Mock summary query
	summaryRows := sqlmock.NewRows([]string{
		"total_flows", "total_bytes", "total_packets", "unique_endpoints", "avg_duration_ms",
	}).AddRow(0, 0, 0, 0, 0.0)

	mock.ExpectQuery("SELECT (.+) FROM flows").WillReturnRows(depsRows)
	mock.ExpectQuery("WITH source_stats AS").WillReturnRows(emptyTalkerRows)
	mock.ExpectQuery("WITH source_stats AS").WillReturnRows(emptyTalkerRows)
	mock.ExpectQuery("WITH source_stats AS").WillReturnRows(emptyFlowCountRows)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(summaryRows)

	// Make request with include_top_talkers parameter
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/stats?include_top_talkers=true", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetAggregatedStats_Error tests error handling
func TestGetAggregatedStats_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	// Mock database error
	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnError(sql.ErrConnDone)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/stats", nil)
	router.ServeHTTP(w, req)

	// Assert error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get aggregated stats")
}

// TestParseAggregationQuery_DefaultValues tests default parameter values
func TestParseAggregationQuery_DefaultValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	// Mock query (empty result is fine)
	rows := sqlmock.NewRows([]string{
		"source_label", "dest_label", "flow_count", "total_bytes", "total_packets", "protocols", "avg_duration_ms",
	})

	mock.ExpectQuery("SELECT (.+) FROM flows").WillReturnRows(rows)

	// Make request without parameters (should use defaults)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/dependencies", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Default group_by is "app"
	assert.Equal(t, "app", response["group_by"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestParseAggregationQuery_CustomGroupBy tests custom group_by parameter
func TestParseAggregationQuery_CustomGroupBy(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := setupAggregatorRouter(db)

	// Mock query (empty result is fine)
	rows := sqlmock.NewRows([]string{
		"source_label", "dest_label", "flow_count", "total_bytes", "total_packets", "protocols", "avg_duration_ms",
	})

	mock.ExpectQuery("SELECT (.+) FROM flows").WillReturnRows(rows)

	// Make request with custom group_by
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/aggregator/dependencies?group_by=env", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Custom group_by is "env"
	assert.Equal(t, "env", response["group_by"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
