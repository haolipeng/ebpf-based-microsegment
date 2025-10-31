package models

// StatisticsResponse represents all data plane statistics
type StatisticsResponse struct {
	TotalPackets   uint64 `json:"total_packets"`
	AllowedPackets uint64 `json:"allowed_packets"`
	DeniedPackets  uint64 `json:"denied_packets"`
	NewSessions    uint64 `json:"new_sessions"`
	ClosedSessions uint64 `json:"closed_sessions"`
	ActiveSessions uint64 `json:"active_sessions"`
	PolicyHits     uint64 `json:"policy_hits"`
	PolicyMisses   uint64 `json:"policy_misses"`
}

// PacketStatsResponse represents packet-specific statistics
type PacketStatsResponse struct {
	TotalPackets   uint64  `json:"total_packets"`
	AllowedPackets uint64  `json:"allowed_packets"`
	DeniedPackets  uint64  `json:"denied_packets"`
	AllowRate      float64 `json:"allow_rate"`
	DenyRate       float64 `json:"deny_rate"`
}

// SessionStatsResponse represents session-specific statistics
type SessionStatsResponse struct {
	NewSessions    uint64 `json:"new_sessions"`
	ClosedSessions uint64 `json:"closed_sessions"`
	ActiveSessions uint64 `json:"active_sessions"`
}

// PolicyStatsResponse represents policy-specific statistics
type PolicyStatsResponse struct {
	PolicyHits   uint64  `json:"policy_hits"`
	PolicyMisses uint64  `json:"policy_misses"`
	HitRate      float64 `json:"hit_rate"`
}

