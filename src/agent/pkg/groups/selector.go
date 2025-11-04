// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package groups

import (
	"github.com/ebpf-microsegment/src/agent/pkg/workload"
)

// EvaluateSelector evaluates whether a workload matches a label selector
// Returns true if the workload matches the selector, false otherwise
func EvaluateSelector(wl *workload.Workload, sel LabelSelector) bool {
	// Validate selector first
	if err := sel.Validate(); err != nil {
		return false
	}

	switch sel.Operator {
	case OpEqual:
		return evaluateEqual(wl, sel)
	case OpNotEqual:
		return evaluateNotEqual(wl, sel)
	case OpIn:
		return evaluateIn(wl, sel)
	case OpNotIn:
		return evaluateNotIn(wl, sel)
	case OpExists:
		return evaluateExists(wl, sel)
	case OpNotExists:
		return evaluateNotExists(wl, sel)
	default:
		// Unknown operator - should not happen if Validate() passed
		return false
	}
}

// evaluateEqual checks if label key equals the specified value
// Operator: key = "value"
func evaluateEqual(wl *workload.Workload, sel LabelSelector) bool {
	value, exists := wl.Labels[sel.Key]
	if !exists {
		return false
	}

	// OpEqual requires exactly one value
	if len(sel.Values) == 0 {
		return false
	}

	return value == sel.Values[0]
}

// evaluateNotEqual checks if label key does not equal the specified value
// Operator: key != "value"
func evaluateNotEqual(wl *workload.Workload, sel LabelSelector) bool {
	value, exists := wl.Labels[sel.Key]
	if !exists {
		// If key doesn't exist, it's not equal to any value
		return true
	}

	// OpNotEqual requires exactly one value
	if len(sel.Values) == 0 {
		return false
	}

	return value != sel.Values[0]
}

// evaluateIn checks if label value is in the specified list
// Operator: key in ["value1", "value2"]
func evaluateIn(wl *workload.Workload, sel LabelSelector) bool {
	value, exists := wl.Labels[sel.Key]
	if !exists {
		return false
	}

	// Check if value is in the list
	for _, v := range sel.Values {
		if value == v {
			return true
		}
	}

	return false
}

// evaluateNotIn checks if label value is not in the specified list
// Operator: key not-in ["value1", "value2"]
func evaluateNotIn(wl *workload.Workload, sel LabelSelector) bool {
	value, exists := wl.Labels[sel.Key]
	if !exists {
		// If key doesn't exist, it's not in any list
		return true
	}

	// Check if value is NOT in the list
	for _, v := range sel.Values {
		if value == v {
			return false
		}
	}

	return true
}

// evaluateExists checks if a label key exists (any value)
// Operator: key exists
func evaluateExists(wl *workload.Workload, sel LabelSelector) bool {
	_, exists := wl.Labels[sel.Key]
	return exists
}

// evaluateNotExists checks if a label key does not exist
// Operator: key not-exists
func evaluateNotExists(wl *workload.Workload, sel LabelSelector) bool {
	_, exists := wl.Labels[sel.Key]
	return !exists
}

// EvaluateSelectors evaluates whether a workload matches ALL selectors (AND logic)
// Returns true only if the workload matches all selectors
func EvaluateSelectors(wl *workload.Workload, selectors []LabelSelector) bool {
	// Empty selector list matches nothing
	if len(selectors) == 0 {
		return false
	}

	// All selectors must match (AND logic)
	for _, sel := range selectors {
		if !EvaluateSelector(wl, sel) {
			return false
		}
	}

	return true
}

// MatchesGroup checks if a workload matches a group's selectors
// This is a convenience function that combines group retrieval and evaluation
func MatchesGroup(wl *workload.Workload, group *Group) bool {
	if group == nil {
		return false
	}

	return EvaluateSelectors(wl, group.Selectors)
}
