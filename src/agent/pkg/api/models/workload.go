package models

// WorkloadRequest represents a workload creation/update request
type WorkloadRequest struct {
	ID     string            `json:"id" binding:"required"`
	Name   string            `json:"name" binding:"required"`
	HostID string            `json:"host_id"`
	IPs    []string          `json:"ips" binding:"required,min=1,dive,required"`
	Labels map[string]string `json:"labels"`
}

// WorkloadResponse represents a workload in API responses
type WorkloadResponse struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	HostID string            `json:"host_id,omitempty"`
	IPs    []string          `json:"ips"`
	Labels map[string]string `json:"labels"`
}

// WorkloadListResponse represents a list of workloads
type WorkloadListResponse struct {
	Workloads []WorkloadResponse `json:"workloads"`
	Count     int                `json:"count"`
}
