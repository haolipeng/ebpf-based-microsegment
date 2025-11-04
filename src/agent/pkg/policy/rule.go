// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PolicyRule represents a high-level label-based policy rule
// This is compiled into multiple low-level Policy objects based on group membership
type PolicyRule struct {
	ID          uint32      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	FromGroup   string      `json:"from_group"`   // Source group name
	ToGroup     string      `json:"to_group"`     // Destination group name
	Ports       []PortRange `json:"ports"`        // Port ranges allowed
	Action      string      `json:"action"`       // "allow" or "deny"
	Priority    uint16      `json:"priority"`     // Higher = more priority
	Enabled     bool        `json:"enabled"`      // Whether rule is active
	CreatedAt   string      `json:"created_at,omitempty"`
	UpdatedAt   string      `json:"updated_at,omitempty"`
}

// PortRange represents a port range with protocol
type PortRange struct {
	Protocol string `json:"protocol"`      // "tcp", "udp", "icmp", "any"
	Start    uint16 `json:"start"`         // Starting port (0 = any)
	End      uint16 `json:"end,omitempty"` // Ending port (0 or equal to start = single port)
}

// Validate validates a PolicyRule for correctness
func (r *PolicyRule) Validate() error {
	// Validate name
	if r.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if len(r.Name) > 255 {
		return fmt.Errorf("rule name too long (max 255 characters)")
	}

	// Validate groups
	if r.FromGroup == "" {
		return fmt.Errorf("from_group is required")
	}
	if r.ToGroup == "" {
		return fmt.Errorf("to_group is required")
	}

	// Validate action
	action := strings.ToLower(r.Action)
	if action != "allow" && action != "deny" {
		return fmt.Errorf("action must be 'allow' or 'deny', got: %s", r.Action)
	}

	// Validate ports
	if len(r.Ports) == 0 {
		return fmt.Errorf("at least one port range is required")
	}

	for i, port := range r.Ports {
		if err := port.Validate(); err != nil {
			return fmt.Errorf("port range %d invalid: %w", i, err)
		}
	}

	return nil
}

// Validate validates a PortRange for correctness
func (p *PortRange) Validate() error {
	// Validate protocol
	proto := strings.ToLower(p.Protocol)
	if proto != "tcp" && proto != "udp" && proto != "icmp" && proto != "any" {
		return fmt.Errorf("protocol must be 'tcp', 'udp', 'icmp', or 'any', got: %s", p.Protocol)
	}

	// Validate port range
	if p.Start > p.End && p.End != 0 {
		return fmt.Errorf("start port (%d) cannot be greater than end port (%d)", p.Start, p.End)
	}

	// Port 0 is allowed as wildcard
	if p.Start > 65535 {
		return fmt.Errorf("start port (%d) exceeds maximum (65535)", p.Start)
	}
	if p.End > 65535 {
		return fmt.Errorf("end port (%d) exceeds maximum (65535)", p.End)
	}

	return nil
}

// String returns a human-readable representation of PortRange
func (p *PortRange) String() string {
	if p.Start == 0 && p.End == 0 {
		return fmt.Sprintf("%s:any", p.Protocol)
	}
	if p.End == 0 || p.Start == p.End {
		return fmt.Sprintf("%s:%d", p.Protocol, p.Start)
	}
	return fmt.Sprintf("%s:%d-%d", p.Protocol, p.Start, p.End)
}

// String returns a human-readable representation of PolicyRule
func (r *PolicyRule) String() string {
	ports := make([]string, len(r.Ports))
	for i, p := range r.Ports {
		ports[i] = p.String()
	}
	return fmt.Sprintf("Rule[%d:%s] %s -> %s [%s] %s (priority=%d, enabled=%v)",
		r.ID, r.Name, r.FromGroup, r.ToGroup, strings.Join(ports, ","), r.Action, r.Priority, r.Enabled)
}

// PortsToJSON serializes port ranges to JSON string for database storage
func (r *PolicyRule) PortsToJSON() (string, error) {
	data, err := json.Marshal(r.Ports)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ports: %w", err)
	}
	return string(data), nil
}

// PortsFromJSON deserializes port ranges from JSON string
func (r *PolicyRule) PortsFromJSON(jsonStr string) error {
	if jsonStr == "" {
		return fmt.Errorf("ports JSON is empty")
	}

	var ports []PortRange
	if err := json.Unmarshal([]byte(jsonStr), &ports); err != nil {
		return fmt.Errorf("failed to unmarshal ports: %w", err)
	}

	r.Ports = ports
	return nil
}

// Clone creates a deep copy of the PolicyRule
func (r *PolicyRule) Clone() *PolicyRule {
	clone := &PolicyRule{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		FromGroup:   r.FromGroup,
		ToGroup:     r.ToGroup,
		Action:      r.Action,
		Priority:    r.Priority,
		Enabled:     r.Enabled,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	// Deep copy ports
	clone.Ports = make([]PortRange, len(r.Ports))
	copy(clone.Ports, r.Ports)

	return clone
}

// ExpandPorts expands port ranges into individual ports
// Returns a slice of (protocol, port) tuples
// If port range is 0-0 (any), returns (protocol, 0)
// This is useful for policy compilation
func (p *PortRange) ExpandPorts() []struct {
	Protocol string
	Port     uint16
} {
	var result []struct {
		Protocol string
		Port     uint16
	}

	// Wildcard port
	if p.Start == 0 && p.End == 0 {
		result = append(result, struct {
			Protocol string
			Port     uint16
		}{Protocol: p.Protocol, Port: 0})
		return result
	}

	// Single port
	if p.End == 0 || p.Start == p.End {
		result = append(result, struct {
			Protocol string
			Port     uint16
		}{Protocol: p.Protocol, Port: p.Start})
		return result
	}

	// Port range - expand all ports
	// Note: For large ranges, this could be expensive
	// Callers should be aware of this limitation
	for port := p.Start; port <= p.End; port++ {
		result = append(result, struct {
			Protocol string
			Port     uint16
		}{Protocol: p.Protocol, Port: port})
	}

	return result
}

// ContainsPort checks if a port is within this range
func (p *PortRange) ContainsPort(port uint16) bool {
	// Wildcard matches everything
	if p.Start == 0 && p.End == 0 {
		return true
	}

	// Single port
	if p.End == 0 || p.Start == p.End {
		return port == p.Start
	}

	// Range
	return port >= p.Start && port <= p.End
}
