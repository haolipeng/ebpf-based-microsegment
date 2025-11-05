package reporter

import (
	"context"

	"github.com/ebpf-microsegment/src/agent/pkg/flow"
	"github.com/sirupsen/logrus"
)

// LocalReporter reports flows to local SQLite storage
type LocalReporter struct {
	storage flow.Storage
}

// NewLocalReporter creates a new LocalReporter
func NewLocalReporter(storage flow.Storage) *LocalReporter {
	return &LocalReporter{
		storage: storage,
	}
}

// Report saves a single flow to local storage
func (r *LocalReporter) Report(ctx context.Context, f *flow.Flow) error {
	return r.storage.SaveFlow(f)
}

// ReportBatch saves multiple flows to local storage
func (r *LocalReporter) ReportBatch(ctx context.Context, flows []*flow.Flow) error {
	for _, f := range flows {
		if err := r.storage.SaveFlow(f); err != nil {
			logrus.Errorf("Failed to save flow: %v", err)
			return err
		}
	}
	return nil
}

// Start initializes the local reporter (no-op for SQLite)
func (r *LocalReporter) Start() error {
	logrus.Info("Local reporter started (standalone mode)")
	return nil
}

// Stop gracefully shuts down the local reporter (no-op)
func (r *LocalReporter) Stop() error {
	logrus.Info("Local reporter stopped")
	return nil
}
