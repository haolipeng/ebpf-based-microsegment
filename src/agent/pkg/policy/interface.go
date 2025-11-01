// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

// Manager interface defines the operations for policy management.
// This interface is useful for testing and dependency injection.
type Manager interface {
	AddPolicy(p *Policy) error
	DeletePolicy(p *Policy) error
	ListPolicies() ([]Policy, error)
}

// Ensure PolicyManager implements Manager interface
var _ Manager = (*PolicyManager)(nil)
