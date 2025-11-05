package reporter

import (
	"context"

	"github.com/ebpf-microsegment/src/agent/pkg/flow"
)

// Reporter is an interface for reporting flow data
// Implementations can be local (SQLite) or remote (gRPC)
type Reporter interface {
	// Report sends a single flow to the reporter
	Report(ctx context.Context, flow *flow.Flow) error

	// ReportBatch sends multiple flows (for efficiency)
	ReportBatch(ctx context.Context, flows []*flow.Flow) error

	// Start initializes the reporter
	Start() error

	// Stop gracefully shuts down the reporter
	Stop() error
}
