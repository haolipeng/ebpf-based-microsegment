package models

// PolicyRequest represents a policy creation/update request
type PolicyRequest struct {
	RuleID   uint32 `json:"rule_id" binding:"required"`
	SrcIP    string `json:"src_ip" binding:"required"`
	DstIP    string `json:"dst_ip" binding:"required"`
	SrcPort  uint16 `json:"src_port"`
	DstPort  uint16 `json:"dst_port"`
	Protocol string `json:"protocol" binding:"required,oneof=tcp udp icmp any"`
	Action   string `json:"action" binding:"required,oneof=allow deny log"`
	Priority uint16 `json:"priority"`
}

// PolicyResponse represents a policy in API responses
type PolicyResponse struct {
	RuleID   uint32 `json:"rule_id"`
	SrcIP    string `json:"src_ip"`
	DstIP    string `json:"dst_ip"`
	SrcPort  uint16 `json:"src_port"`
	DstPort  uint16 `json:"dst_port"`
	Protocol string `json:"protocol"`
	Action   string `json:"action"`
	Priority uint16 `json:"priority"`
}

// PolicyListResponse represents a list of policies
type PolicyListResponse struct {
	Policies []PolicyResponse `json:"policies"`
	Count    int              `json:"count"`
}

