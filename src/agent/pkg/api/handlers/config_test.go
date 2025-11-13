// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/api/models"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// setupConfigTestRouter creates a test router with config handler
func setupConfigTestRouter() (*gin.Engine, *ConfigHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewConfigHandler("lo", "info", 5, "127.0.0.1", 8080)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/config", handler.GetConfig)
		v1.PUT("/config", handler.UpdateConfig)
	}

	return router, handler
}

func TestGetConfig_Success(t *testing.T) {
	router, _ := setupConfigTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConfigResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "lo", response.Interface)
	assert.Equal(t, "info", response.LogLevel)
	assert.Equal(t, 5, response.StatsInterval)
	assert.Equal(t, "127.0.0.1", response.APIHost)
	assert.Equal(t, 8080, response.APIPort)
}

func TestUpdateConfig_LogLevel_Success(t *testing.T) {
	router, handler := setupConfigTestRouter()

	// Set log level to info initially
	log.SetLevel(log.InfoLevel)

	newLogLevel := "debug"
	reqBody := models.ConfigUpdateRequest{
		LogLevel: &newLogLevel,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConfigResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "debug", response.LogLevel)
	assert.Equal(t, "debug", handler.currentConfig.LogLevel)
	assert.Equal(t, log.DebugLevel, log.GetLevel())
}

func TestUpdateConfig_StatsInterval_Success(t *testing.T) {
	router, handler := setupConfigTestRouter()

	newInterval := 10
	reqBody := models.ConfigUpdateRequest{
		StatsInterval: &newInterval,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConfigResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, 10, response.StatsInterval)
	assert.Equal(t, 10, handler.currentConfig.StatsInterval)
}

func TestUpdateConfig_BothFields_Success(t *testing.T) {
	router, handler := setupConfigTestRouter()

	newLogLevel := "warn"
	newInterval := 15
	reqBody := models.ConfigUpdateRequest{
		LogLevel:      &newLogLevel,
		StatsInterval: &newInterval,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConfigResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "warn", response.LogLevel)
	assert.Equal(t, 15, response.StatsInterval)
	assert.Equal(t, "warn", handler.currentConfig.LogLevel)
	assert.Equal(t, 15, handler.currentConfig.StatsInterval)
	assert.Equal(t, log.WarnLevel, log.GetLevel())
}

func TestUpdateConfig_InvalidLogLevel(t *testing.T) {
	router, handler := setupConfigTestRouter()

	originalLogLevel := handler.currentConfig.LogLevel

	invalidLogLevel := "invalid"
	reqBody := models.ConfigUpdateRequest{
		LogLevel: &invalidLogLevel,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Contains(t, errorResp.Error, "Invalid request")

	// Verify config was not changed
	assert.Equal(t, originalLogLevel, handler.currentConfig.LogLevel)
}

func TestUpdateConfig_InvalidStatsInterval_TooLow(t *testing.T) {
	router, _ := setupConfigTestRouter()

	invalidInterval := 0
	reqBody := models.ConfigUpdateRequest{
		StatsInterval: &invalidInterval,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Contains(t, errorResp.Error, "Invalid request")
}

func TestUpdateConfig_InvalidStatsInterval_TooHigh(t *testing.T) {
	router, _ := setupConfigTestRouter()

	invalidInterval := 301
	reqBody := models.ConfigUpdateRequest{
		StatsInterval: &invalidInterval,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Contains(t, errorResp.Error, "Invalid request")
}

func TestUpdateConfig_NoFieldsProvided(t *testing.T) {
	router, _ := setupConfigTestRouter()

	reqBody := models.ConfigUpdateRequest{}

	bodyBytes, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Contains(t, errorResp.Error, "No configuration fields provided")
}

func TestUpdateConfig_InvalidJSON(t *testing.T) {
	router, _ := setupConfigTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid request format", errorResp.Error)
}

func TestUpdateConfig_InvalidLogLevelValue(t *testing.T) {
	router, _ := setupConfigTestRouter()

	// Test log level not in the allowed list (debug, info, warn, error)
	invalidLogLevel := "trace"
	reqBody := models.ConfigUpdateRequest{
		LogLevel: &invalidLogLevel,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Contains(t, errorResp.Error, "Invalid request")
}

func TestUpdateConfig_ReadOnlyFields(t *testing.T) {
	router, handler := setupConfigTestRouter()

	// Verify that interface, API host, and API port remain unchanged
	originalInterface := handler.currentConfig.Interface
	originalAPIHost := handler.currentConfig.APIHost
	originalAPIPort := handler.currentConfig.APIPort

	newLogLevel := "error"
	reqBody := models.ConfigUpdateRequest{
		LogLevel: &newLogLevel,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/config", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConfigResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify read-only fields did not change
	assert.Equal(t, originalInterface, response.Interface)
	assert.Equal(t, originalAPIHost, response.APIHost)
	assert.Equal(t, originalAPIPort, response.APIPort)
}

func TestNewConfigHandler(t *testing.T) {
	handler := NewConfigHandler("eth0", "debug", 10, "0.0.0.0", 9090)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.currentConfig)
	assert.Equal(t, "eth0", handler.currentConfig.Interface)
	assert.Equal(t, "debug", handler.currentConfig.LogLevel)
	assert.Equal(t, 10, handler.currentConfig.StatsInterval)
	assert.Equal(t, "0.0.0.0", handler.currentConfig.APIHost)
	assert.Equal(t, 9090, handler.currentConfig.APIPort)
}
