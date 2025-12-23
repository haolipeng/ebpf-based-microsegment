
// input: N/A (model definition)
// output: policy rule API request/response models
// pos: policy rule API models - if file updated, must sync with this header comment and pkg/api/CLAUDE.md
package models

// PortRangeRequest represents a port range in API requests
type PortRangeRequest struct {
	Start    uint16 `json:"start" binding:"required,min=1,max=65535"`
	End      uint16 `json:"end" binding:"required,min=1,max=65535"`
	Protocol string `json:"protocol" binding:"required,oneof=tcp udp icmp"`
}

// PortRangeResponse represents a port range in API responses
type PortRangeResponse struct {
	Start    uint16 `json:"start"`
	End      uint16 `json:"end"`
	Protocol string `json:"protocol"`
}

// PolicyRuleRequest represents a policy rule creation/update request
type PolicyRuleRequest struct {
	ID          uint32             `json:"id,omitempty"`
	Name        string             `json:"name" binding:"required"`
	Description string             `json:"description"`
	FromGroup   string             `json:"from_group" binding:"required"`
	ToGroup     string             `json:"to_group" binding:"required"`
	Ports       []PortRangeRequest `json:"ports" binding:"required,min=1,dive,required"`
	Action      string             `json:"action" binding:"required,oneof=allow deny log"`
	Priority    uint16             `json:"priority" binding:"max=65535"`
	Enabled     bool               `json:"enabled"`
}

// PolicyRuleResponse represents a policy rule in API responses
type PolicyRuleResponse struct {
	ID          uint32              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	FromGroup   string              `json:"from_group"`
	ToGroup     string              `json:"to_group"`
	Ports       []PortRangeResponse `json:"ports"`
	Action      string              `json:"action"`
	Priority    uint16              `json:"priority"`
	Enabled     bool                `json:"enabled"`
	CreatedAt   string              `json:"created_at,omitempty"`
	UpdatedAt   string              `json:"updated_at,omitempty"`
}

// PolicyRuleListResponse represents a list of policy rules
type PolicyRuleListResponse struct {
	Rules []PolicyRuleResponse `json:"rules"`
	Count int                  `json:"count"`
}

// CompiledPolicyResponse represents a compiled policy in API responses
type CompiledPolicyResponse struct {
	RuleID          uint32 `json:"rule_id"`
	SourceRuleID    uint32 `json:"source_rule_id"`
	SrcIP           string `json:"src_ip"`
	DstIP           string `json:"dst_ip"`
	DstPort         uint16 `json:"dst_port"`
	Protocol        string `json:"protocol"`
	Action          string `json:"action"`
	Priority        uint16 `json:"priority"`
	FromGroup       string `json:"from_group"`
	ToGroup         string `json:"to_group"`
	FromWorkloadID  string `json:"from_workload_id"`
	ToWorkloadID    string `json:"to_workload_id"`
	CompilationTime string `json:"compilation_time"`
	CompilerVersion string `json:"compiler_version"`
}

// CompiledPoliciesResponse represents a list of compiled policies
type CompiledPoliciesResponse struct {
	Policies []CompiledPolicyResponse `json:"policies"`
	Count    int                      `json:"count"`
}
