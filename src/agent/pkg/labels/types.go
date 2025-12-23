// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: N/A (type definition)
// output: label data structures (LabelDimension, LabelConstraints)
// pos: label type definitions - if file updated, must sync with this header comment and pkg/labels/CLAUDE.md
package labels

// LabelDimension represents a recommended label dimension in the Illumio-inspired model
// These dimensions are recommendations, not enforced requirements
type LabelDimension string

const (
	// LabelRole represents the technical role of a workload
	// Examples: "web", "api", "db", "cache", "mq", "worker", "gateway"
	LabelRole LabelDimension = "role"

	// LabelApp represents the business application
	// Examples: "frontend", "backend", "auth", "payment", "analytics"
	LabelApp LabelDimension = "app"

	// LabelEnv represents the deployment environment
	// Examples: "prod", "staging", "dev", "test", "qa"
	LabelEnv LabelDimension = "env"

	// LabelLocation represents the physical or logical location
	// Examples: "us-west-2", "eu-central-1", "dc-1", "az-a", "edge"
	LabelLocation LabelDimension = "loc"
)

// AllDimensions returns all defined label dimensions
func AllDimensions() []LabelDimension {
	return []LabelDimension{
		LabelRole,
		LabelApp,
		LabelEnv,
		LabelLocation,
	}
}

// IsDimensionLabel checks if a label key is one of the recommended dimensions
func IsDimensionLabel(key string) bool {
	return key == string(LabelRole) ||
		key == string(LabelApp) ||
		key == string(LabelEnv) ||
		key == string(LabelLocation)
}

// String returns the string representation of the dimension
func (d LabelDimension) String() string {
	return string(d)
}

// ReservedPrefixes are label key prefixes reserved for system use
var ReservedPrefixes = []string{
	"system.",      // System-generated labels
	"k8s.",         // Kubernetes integration labels
	"internal.",    // Internal system labels
	"ebpf.",        // eBPF-related labels
}

// IsReservedPrefix checks if a label key uses a reserved prefix
func IsReservedPrefix(key string) bool {
	for _, prefix := range ReservedPrefixes {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// CommonRoleValues are commonly used values for the "role" dimension
var CommonRoleValues = []string{
	"web",      // Web servers (nginx, Apache)
	"api",      // API servers
	"db",       // Databases (MySQL, PostgreSQL, MongoDB)
	"cache",    // Cache servers (Redis, Memcached)
	"mq",       // Message queues (RabbitMQ, Kafka)
	"worker",   // Background workers
	"gateway",  // API gateways
	"proxy",    // Proxy servers
	"lb",       // Load balancers
	"storage",  // Storage services
}

// CommonEnvValues are commonly used values for the "env" dimension
var CommonEnvValues = []string{
	"prod",     // Production
	"staging",  // Staging
	"dev",      // Development
	"test",     // Testing
	"qa",       // Quality Assurance
	"uat",      // User Acceptance Testing
}

// LabelConstraints defines validation constraints for labels
type LabelConstraints struct {
	// MaxKeyLength is the maximum length for a label key (Kubernetes: 253)
	MaxKeyLength int

	// MaxValueLength is the maximum length for a label value (Kubernetes: 63)
	MaxValueLength int

	// AllowEmptyValue indicates whether empty label values are allowed
	AllowEmptyValue bool

	// EnforceReservedPrefixes indicates whether to reject reserved prefixes
	EnforceReservedPrefixes bool
}

// DefaultConstraints returns the default label validation constraints
// These follow Kubernetes label conventions
func DefaultConstraints() LabelConstraints {
	return LabelConstraints{
		MaxKeyLength:            253,
		MaxValueLength:          63,
		AllowEmptyValue:         true,
		EnforceReservedPrefixes: false, // Allow reserved prefixes by default
	}
}

// StrictConstraints returns strict validation constraints
func StrictConstraints() LabelConstraints {
	return LabelConstraints{
		MaxKeyLength:            253,
		MaxValueLength:          63,
		AllowEmptyValue:         false,
		EnforceReservedPrefixes: true,
	}
}
