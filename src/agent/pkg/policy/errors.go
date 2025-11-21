// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import "fmt"

// PolicyError wraps errors that occur during policy operations.
// It distinguishes between critical eBPF errors and non-critical storage errors.
type PolicyError struct {
	RuleID       uint32 // Policy rule ID for identification
	Operation    string // Operation being performed (e.g., "add", "delete", "update")
	EBPFError    error  // Critical: eBPF map operation error (policy not active)
	StorageError error  // Non-critical: Storage/persistence error (policy active but not persisted)
}

// Error implements the error interface
func (e *PolicyError) Error() string {
	if e.EBPFError != nil {
		return fmt.Sprintf("policy %d %s failed: eBPF error: %v", e.RuleID, e.Operation, e.EBPFError)
	}
	if e.StorageError != nil {
		return fmt.Sprintf("policy %d %s partial: storage error: %v", e.RuleID, e.Operation, e.StorageError)
	}
	return fmt.Sprintf("policy %d %s: unknown error", e.RuleID, e.Operation)
}

// Unwrap implements error unwrapping for Go 1.13+ errors.Is/As
func (e *PolicyError) Unwrap() error {
	if e.EBPFError != nil {
		return e.EBPFError
	}
	return e.StorageError
}

// IsPartial returns true if the policy operation partially succeeded.
// This means the policy is active in eBPF but failed to persist to storage.
func (e *PolicyError) IsPartial() bool {
	return e.EBPFError == nil && e.StorageError != nil
}

// IsCritical returns true if the policy operation failed critically.
// This means the policy is not active in eBPF.
func (e *PolicyError) IsCritical() bool {
	return e.EBPFError != nil
}

// HasStorageError returns true if there was a storage error.
func (e *PolicyError) HasStorageError() bool {
	return e.StorageError != nil
}

// HasEBPFError returns true if there was an eBPF error.
func (e *PolicyError) HasEBPFError() bool {
	return e.EBPFError != nil
}

// NewPolicyError creates a new PolicyError
func NewPolicyError(ruleID uint32, operation string, ebpfErr, storageErr error) *PolicyError {
	return &PolicyError{
		RuleID:       ruleID,
		Operation:    operation,
		EBPFError:    ebpfErr,
		StorageError: storageErr,
	}
}
