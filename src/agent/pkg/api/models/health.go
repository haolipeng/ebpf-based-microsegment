package models

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string `json:"status"` // "ok", "degraded", "down"
	Message string `json:"message"`
}

// StatusResponse represents detailed system status
type StatusResponse struct {
	Status      string                 `json:"status"` // "ok", "degraded", "down"
	Version     string                 `json:"version"`
	Interface   string                 `json:"interface"`
	DataPlane   DataPlaneStatus        `json:"data_plane"`
	API         APIStatus              `json:"api"`
	Statistics  *StatisticsResponse    `json:"statistics,omitempty"`
	PolicyCount int                    `json:"policy_count"`
	Uptime      int64                  `json:"uptime_seconds"`
}

// DataPlaneStatus represents data plane status
type DataPlaneStatus struct {
	Status  string `json:"status"` // "running", "stopped", "error"
	Message string `json:"message"`
}

// APIStatus represents API server status
type APIStatus struct {
	Status  string `json:"status"` // "running", "stopped", "error"
	Message string `json:"message"`
}

