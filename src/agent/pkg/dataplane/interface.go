// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

// DataPlaneInterface defines the operations for data plane management.
// This interface is useful for testing and dependency injection.
type DataPlaneInterface interface {
	GetStatistics() Statistics
}

// Ensure DataPlane implements DataPlaneInterface
var _ DataPlaneInterface = (*DataPlane)(nil)
