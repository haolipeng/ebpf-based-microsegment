package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRouter creates a test Gin router with the flow handler
func setupTestRouter(flowStorage *storage.FlowStorage) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewFlowHandler(flowStorage)
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	return router
}

// TestNewFlowHandler tests the constructor
func TestNewFlowHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	handler := NewFlowHandler(flowStorage)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.flowStorage)
}

// TestListFlows_Success tests successful flow listing
func TestListFlows_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	router := setupTestRouter(flowStorage)

	// Mock COUNT query
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Mock SELECT query
	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count",
		"policy_id", "policy_action", "state", "agent_id", "source_labels", "dest_labels",
	}).
		AddRow(uint64(1), uint64(time.Now().UnixNano()), "192.168.1.1", "192.168.1.2",
			uint32(12345), uint32(80), uint32(6), uint32(1),
			uint64(10), uint64(1024), uint32(1), uint32(1), uint32(1),
			"agent-1", []byte(`{}`), []byte(`{}`)).
		AddRow(uint64(2), uint64(time.Now().UnixNano()), "192.168.1.3", "192.168.1.4",
			uint32(54321), uint32(443), uint32(6), uint32(1),
			uint64(20), uint64(2048), uint32(2), uint32(1), uint32(1),
			"agent-1", []byte(`{}`), []byte(`{}`))
	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnRows(rows)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/flows?limit=10&offset=0", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(2), response["total_count"])
	assert.Equal(t, float64(10), response["limit"])
	assert.Equal(t, float64(0), response["offset"])
	assert.False(t, response["has_more"].(bool))

	flows := response["flows"].([]interface{})
	assert.Len(t, flows, 2)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestListFlows_WithPagination tests pagination
func TestListFlows_WithPagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	router := setupTestRouter(flowStorage)

	// Mock COUNT query - 100 total flows
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	// Mock SELECT query - return empty for simplicity
	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count",
		"policy_id", "policy_action", "state", "agent_id", "source_labels", "dest_labels",
	})
	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnRows(rows)

	// Make request with offset=10, limit=20
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/flows?limit=20&offset=10", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(100), response["total_count"])
	assert.Equal(t, float64(20), response["limit"])
	assert.Equal(t, float64(10), response["offset"])
	assert.True(t, response["has_more"].(bool)) // offset(10) + limit(20) < total(100)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestListFlows_WithFilters tests query parameter filtering
func TestListFlows_WithFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	router := setupTestRouter(flowStorage)

	// Mock COUNT query
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Mock SELECT query
	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count",
		"policy_id", "policy_action", "state", "agent_id", "source_labels", "dest_labels",
	}).AddRow(uint64(1), uint64(time.Now().UnixNano()), "192.168.1.1", "192.168.1.2",
		uint32(12345), uint32(80), uint32(6), uint32(1),
		uint64(10), uint64(1024), uint32(1), uint32(1), uint32(1),
		"agent-1", []byte(`{}`), []byte(`{}`))
	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnRows(rows)

	// Make request with filters
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/flows?agent_id=agent-1&source_ip=192.168.1.1&protocol=tcp", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(1), response["total_count"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestListFlows_StorageError tests error handling
func TestListFlows_StorageError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	router := setupTestRouter(flowStorage)

	// Mock query error
	mock.ExpectQuery("SELECT COUNT").
		WillReturnError(sql.ErrConnDone)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/flows", nil)
	router.ServeHTTP(w, req)

	// Assert error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to query flows")
}

// TestGetFlow_Success tests getting a single flow
func TestGetFlow_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	router := setupTestRouter(flowStorage)

	// Mock COUNT query
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Mock SELECT query
	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count",
		"policy_id", "policy_action", "state", "agent_id", "source_labels", "dest_labels",
	}).AddRow(uint64(123), uint64(time.Now().UnixNano()), "192.168.1.1", "192.168.1.2",
		uint32(12345), uint32(80), uint32(6), uint32(1),
		uint64(10), uint64(1024), uint32(1), uint32(1), uint32(1),
		"agent-1", []byte(`{"app":"web"}`), []byte(`{"app":"db"}`))
	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnRows(rows)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/flows/123", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var flow map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &flow)
	require.NoError(t, err)

	assert.Equal(t, float64(123), flow["id"])
	assert.Equal(t, "192.168.1.1", flow["src_ip"])
	assert.Equal(t, "192.168.1.2", flow["dst_ip"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetFlow_NotFound tests 404 handling
func TestGetFlow_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	router := setupTestRouter(flowStorage)

	// Mock COUNT query
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// Mock SELECT query - empty result
	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count",
		"policy_id", "policy_action", "state", "agent_id", "source_labels", "dest_labels",
	})
	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnRows(rows)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/flows/999", nil)
	router.ServeHTTP(w, req)

	// Assert 404 response
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Flow not found", response["error"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetFlowSummary_Success tests getting flow statistics
func TestGetFlowSummary_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	router := setupTestRouter(flowStorage)

	// Mock summary query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) as total_flows").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_flows", "active_flows", "closed_flows", "total_packets", "total_bytes",
			"allowed_flows", "denied_flows", "unique_source_ips", "unique_dest_ips", "avg_duration_ms",
		}).AddRow(100, 60, 40, 10000, 1024000, 80, 20, 10, 20, 150.5))

	// Mock protocol stats query
	mock.ExpectQuery("SELECT(.+)protocol(.+)FROM flows").
		WillReturnRows(sqlmock.NewRows([]string{
			"protocol", "count", "bytes",
		}).
			AddRow("6", 80, 80000).  // TCP
			AddRow("17", 15, 15000). // UDP
			AddRow("1", 5, 5000))    // ICMP

	// Make request
	w := httptest.NewRecorder()
	startTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	endTime := time.Now().Format(time.RFC3339)
	req, _ := http.NewRequest("GET", "/api/v1/flows/summary?start_time="+startTime+"&end_time="+endTime, nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var summary map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &summary)
	require.NoError(t, err)

	assert.Equal(t, float64(100), summary["total_flows"])
	assert.Equal(t, float64(10000), summary["total_packets"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetFlowSummary_StorageError tests error handling
func TestGetFlowSummary_StorageError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	router := setupTestRouter(flowStorage)

	// Mock query error
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) as total_flows").
		WillReturnError(sql.ErrConnDone)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/flows/summary", nil)
	router.ServeHTTP(w, req)

	// Assert error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get flow summary")
}

// TestGetFlowDependencies_Success tests getting application dependencies
func TestGetFlowDependencies_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	router := setupTestRouter(flowStorage)

	// Mock dependencies query
	rows := sqlmock.NewRows([]string{
		"source_label", "dest_label", "flow_count", "total_bytes", "protocols",
	}).
		AddRow("web", "db", 100, 102400, []byte(`["6","17"]`)).
		AddRow("web", "cache", 50, 51200, []byte(`["6"]`))

	mock.ExpectQuery("SELECT.*source_labels").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "app").
		WillReturnRows(rows)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/flows/dependencies?group_by=app", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "app", response["group_by"])
	assert.NotNil(t, response["dependencies"])
	assert.NotEmpty(t, response["start_time"])
	assert.NotEmpty(t, response["end_time"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetFlowDependencies_StorageError tests error handling
func TestGetFlowDependencies_StorageError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	router := setupTestRouter(flowStorage)

	// Mock query error
	mock.ExpectQuery("SELECT.*source_labels").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "app").
		WillReturnError(sql.ErrConnDone)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/flows/dependencies", nil)
	router.ServeHTTP(w, req)

	// Assert error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get flow dependencies")
}
