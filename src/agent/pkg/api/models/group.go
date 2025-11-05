package models

// GroupRequest represents a group creation/update request
type GroupRequest struct {
	Name        string            `json:"name" binding:"required"`
	MatchLabels map[string]string `json:"match_labels"`
	MatchAny    []string          `json:"match_any"`
	MatchAll    []string          `json:"match_all"`
	MemberIDs   []string          `json:"member_ids"`
	IsStatic    bool              `json:"is_static"`
}

// GroupResponse represents a group in API responses
type GroupResponse struct {
	Name        string            `json:"name"`
	MatchLabels map[string]string `json:"match_labels,omitempty"`
	MatchAny    []string          `json:"match_any,omitempty"`
	MatchAll    []string          `json:"match_all,omitempty"`
	MemberIDs   []string          `json:"member_ids"`
	MemberCount int               `json:"member_count"`
	IsStatic    bool              `json:"is_static"`
}

// GroupListResponse represents a list of groups
type GroupListResponse struct {
	Groups []GroupResponse `json:"groups"`
	Count  int             `json:"count"`
}

// GroupMembersResponse represents the members of a group
type GroupMembersResponse struct {
	GroupName string             `json:"group_name"`
	Members   []WorkloadResponse `json:"members"`
	Count     int                `json:"count"`
}
