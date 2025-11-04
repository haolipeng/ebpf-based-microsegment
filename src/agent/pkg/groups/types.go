// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package groups

import (
	"encoding/json"
	"fmt"
	"time"
)

// SelectorOperator defines the operator for label matching
type SelectorOperator string

const (
	// OpEqual matches when label value equals the specified value
	// Example: key = "web"
	OpEqual SelectorOperator = "="

	// OpNotEqual matches when label value does not equal the specified value
	// Example: key != "dev"
	OpNotEqual SelectorOperator = "!="

	// OpIn matches when label value is in the specified list
	// Example: key in ["web", "api"]
	OpIn SelectorOperator = "in"

	// OpNotIn matches when label value is not in the specified list
	// Example: key not-in ["dev", "test"]
	OpNotIn SelectorOperator = "not-in"

	// OpExists matches when the label key exists (any value)
	// Example: key exists
	OpExists SelectorOperator = "exists"

	// OpNotExists matches when the label key does not exist
	// Example: key not-exists
	OpNotExists SelectorOperator = "not-exists"
)

// AllOperators returns all defined selector operators
func AllOperators() []SelectorOperator {
	return []SelectorOperator{
		OpEqual,
		OpNotEqual,
		OpIn,
		OpNotIn,
		OpExists,
		OpNotExists,
	}
}

// IsValidOperator checks if an operator is valid
func IsValidOperator(op SelectorOperator) bool {
	switch op {
	case OpEqual, OpNotEqual, OpIn, OpNotIn, OpExists, OpNotExists:
		return true
	default:
		return false
	}
}

// String returns the string representation of the operator
func (op SelectorOperator) String() string {
	return string(op)
}

// RequiresValues returns true if the operator requires values to be specified
func (op SelectorOperator) RequiresValues() bool {
	switch op {
	case OpEqual, OpNotEqual, OpIn, OpNotIn:
		return true
	case OpExists, OpNotExists:
		return false
	default:
		return false
	}
}

// LabelSelector defines how to select workloads by labels
type LabelSelector struct {
	// Key is the label key to match against
	Key string `json:"key"`

	// Operator is the comparison operator
	Operator SelectorOperator `json:"operator"`

	// Values is the list of values to match (required for =, !=, in, not-in)
	// Not used for exists and not-exists operators
	Values []string `json:"values,omitempty"`
}

// Validate validates the label selector
func (ls *LabelSelector) Validate() error {
	if ls.Key == "" {
		return fmt.Errorf("selector key cannot be empty")
	}

	if !IsValidOperator(ls.Operator) {
		return fmt.Errorf("invalid operator: %s", ls.Operator)
	}

	if ls.Operator.RequiresValues() {
		if len(ls.Values) == 0 {
			return fmt.Errorf("operator %s requires at least one value", ls.Operator)
		}
	}

	return nil
}

// String returns a human-readable representation of the selector
func (ls *LabelSelector) String() string {
	switch ls.Operator {
	case OpEqual:
		if len(ls.Values) > 0 {
			return fmt.Sprintf("%s=%s", ls.Key, ls.Values[0])
		}
		return fmt.Sprintf("%s=?", ls.Key)
	case OpNotEqual:
		if len(ls.Values) > 0 {
			return fmt.Sprintf("%s!=%s", ls.Key, ls.Values[0])
		}
		return fmt.Sprintf("%s!=?", ls.Key)
	case OpIn:
		return fmt.Sprintf("%s in %v", ls.Key, ls.Values)
	case OpNotIn:
		return fmt.Sprintf("%s not-in %v", ls.Key, ls.Values)
	case OpExists:
		return fmt.Sprintf("%s exists", ls.Key)
	case OpNotExists:
		return fmt.Sprintf("%s not-exists", ls.Key)
	default:
		return fmt.Sprintf("%s %s %v", ls.Key, ls.Operator, ls.Values)
	}
}

// Group represents a named collection of workloads selected by label selectors
type Group struct {
	// Name is the unique identifier for the group
	Name string `json:"name" db:"name"`

	// Description provides human-readable information about the group
	Description string `json:"description,omitempty" db:"description"`

	// Selectors define the label matching rules
	// Workloads must match ALL selectors (AND logic)
	Selectors []LabelSelector `json:"selectors" db:"selectors"`

	// CreatedAt is the timestamp when the group was created
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt is the timestamp when the group was last updated
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// NewGroup creates a new group with the given name
func NewGroup(name string) *Group {
	now := time.Now()
	return &Group{
		Name:      name,
		Selectors: make([]LabelSelector, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddSelector adds a label selector to the group
func (g *Group) AddSelector(selector LabelSelector) error {
	if err := selector.Validate(); err != nil {
		return fmt.Errorf("invalid selector: %w", err)
	}

	g.Selectors = append(g.Selectors, selector)
	g.UpdatedAt = time.Now()
	return nil
}

// SetSelectors replaces all selectors with the provided list
func (g *Group) SetSelectors(selectors []LabelSelector) error {
	// Validate all selectors first
	for i, selector := range selectors {
		if err := selector.Validate(); err != nil {
			return fmt.Errorf("invalid selector at index %d: %w", i, err)
		}
	}

	g.Selectors = selectors
	g.UpdatedAt = time.Now()
	return nil
}

// Validate validates the group
func (g *Group) Validate() error {
	if g.Name == "" {
		return fmt.Errorf("group name cannot be empty")
	}

	if len(g.Selectors) == 0 {
		return fmt.Errorf("group must have at least one selector")
	}

	for i, selector := range g.Selectors {
		if err := selector.Validate(); err != nil {
			return fmt.Errorf("invalid selector at index %d: %w", i, err)
		}
	}

	return nil
}

// String returns a human-readable representation of the group
func (g *Group) String() string {
	return fmt.Sprintf("Group[%s]: %d selectors", g.Name, len(g.Selectors))
}

// MarshalJSON customizes JSON serialization for Group
// This ensures selectors are properly serialized as JSON
func (g *Group) MarshalJSON() ([]byte, error) {
	type Alias Group
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(g),
	})
}

// UnmarshalJSON customizes JSON deserialization for Group
func (g *Group) UnmarshalJSON(data []byte) error {
	type Alias Group
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(g),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	return nil
}

// GroupSummary provides a lightweight representation of a group
type GroupSummary struct {
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	SelectorCount int       `json:"selector_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ToSummary converts a Group to a GroupSummary
func (g *Group) ToSummary() *GroupSummary {
	return &GroupSummary{
		Name:          g.Name,
		Description:   g.Description,
		SelectorCount: len(g.Selectors),
		CreatedAt:     g.CreatedAt,
		UpdatedAt:     g.UpdatedAt,
	}
}

// Helper functions for creating common selector patterns

// NewEqualSelector creates a selector with the equals operator
func NewEqualSelector(key, value string) LabelSelector {
	return LabelSelector{
		Key:      key,
		Operator: OpEqual,
		Values:   []string{value},
	}
}

// NewNotEqualSelector creates a selector with the not-equals operator
func NewNotEqualSelector(key, value string) LabelSelector {
	return LabelSelector{
		Key:      key,
		Operator: OpNotEqual,
		Values:   []string{value},
	}
}

// NewInSelector creates a selector with the in operator
func NewInSelector(key string, values []string) LabelSelector {
	return LabelSelector{
		Key:      key,
		Operator: OpIn,
		Values:   values,
	}
}

// NewNotInSelector creates a selector with the not-in operator
func NewNotInSelector(key string, values []string) LabelSelector {
	return LabelSelector{
		Key:      key,
		Operator: OpNotIn,
		Values:   values,
	}
}

// NewExistsSelector creates a selector with the exists operator
func NewExistsSelector(key string) LabelSelector {
	return LabelSelector{
		Key:      key,
		Operator: OpExists,
		Values:   nil,
	}
}

// NewNotExistsSelector creates a selector with the not-exists operator
func NewNotExistsSelector(key string) LabelSelector {
	return LabelSelector{
		Key:      key,
		Operator: OpNotExists,
		Values:   nil,
	}
}
