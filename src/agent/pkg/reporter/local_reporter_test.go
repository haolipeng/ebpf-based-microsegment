package reporter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/flow"
)

// MockStorage is a mock implementation of flow.Storage for testing
type MockStorage struct {
	SaveFlowFunc   func(*flow.Flow) error
	UpdateFlowFunc func(*flow.Flow) error
	GetFlowFunc    func(string) (*flow.Flow, error)
	savedFlows     []*flow.Flow
	saveErr        error
}

func (m *MockStorage) SaveFlow(f *flow.Flow) error {
	if m.SaveFlowFunc != nil {
		return m.SaveFlowFunc(f)
	}
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedFlows = append(m.savedFlows, f)
	return nil
}

func (m *MockStorage) UpdateFlow(f *flow.Flow) error {
	if m.UpdateFlowFunc != nil {
		return m.UpdateFlowFunc(f)
	}
	return nil
}

func (m *MockStorage) GetFlow(id string) (*flow.Flow, error) {
	if m.GetFlowFunc != nil {
		return m.GetFlowFunc(id)
	}
	return nil, nil
}

func (m *MockStorage) QueryFlows(query *flow.FlowQuery) ([]*flow.Flow, error) {
	return nil, nil
}

func (m *MockStorage) GetFlowSummary(startTime, endTime time.Time) (*flow.FlowSummary, error) {
	return nil, nil
}

func (m *MockStorage) DeleteOldFlows(olderThan time.Duration) (int64, error) {
	return 0, nil
}

func (m *MockStorage) Close() error {
	return nil
}

// Test_NewLocalReporter tests the constructor
func Test_NewLocalReporter(t *testing.T) {
	storage := &MockStorage{}
	reporter := NewLocalReporter(storage)

	if reporter == nil {
		t.Fatal("Expected non-nil reporter")
	}
	// Note: Can't directly compare storage field as it's an interface
	// but we verify it works in other tests
}

// Test_LocalReporter_Report tests single flow reporting
func Test_LocalReporter_Report(t *testing.T) {
	tests := []struct {
		name      string
		flow      *flow.Flow
		saveErr   error
		expectErr bool
	}{
		{
			name: "successful report",
			flow: &flow.Flow{
				SourceIP:   "10.0.0.1",
				DestIP:     "10.0.0.2",
				SourcePort: 8080,
				DestPort:   443,
				Protocol:   "tcp",
			},
			saveErr:   nil,
			expectErr: false,
		},
		{
			name: "storage error",
			flow: &flow.Flow{
				SourceIP: "10.0.0.1",
			},
			saveErr:   errors.New("storage failure"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MockStorage{saveErr: tt.saveErr}
			reporter := NewLocalReporter(storage)

			ctx := context.Background()
			err := reporter.Report(ctx, tt.flow)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectErr && len(storage.savedFlows) != 1 {
				t.Errorf("Expected 1 saved flow, got %d", len(storage.savedFlows))
			}
		})
	}
}

// Test_LocalReporter_ReportBatch tests batch flow reporting
func Test_LocalReporter_ReportBatch(t *testing.T) {
	tests := []struct {
		name       string
		flows      []*flow.Flow
		saveErr    error
		expectErr  bool
		savedCount int
	}{
		{
			name: "successful batch report",
			flows: []*flow.Flow{
				{SourceIP: "10.0.0.1", DestIP: "10.0.0.2"},
				{SourceIP: "10.0.0.3", DestIP: "10.0.0.4"},
				{SourceIP: "10.0.0.5", DestIP: "10.0.0.6"},
			},
			saveErr:    nil,
			expectErr:  false,
			savedCount: 3,
		},
		{
			name: "empty batch",
			flows: []*flow.Flow{},
			saveErr:    nil,
			expectErr:  false,
			savedCount: 0,
		},
		{
			name: "storage error on second flow",
			flows: []*flow.Flow{
				{SourceIP: "10.0.0.1", DestIP: "10.0.0.2"},
				{SourceIP: "10.0.0.3", DestIP: "10.0.0.4"},
			},
			saveErr:    errors.New("storage failure"),
			expectErr:  true,
			savedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MockStorage{}
			callCount := 0

			if tt.saveErr != nil {
				// Fail on second call
				storage.SaveFlowFunc = func(f *flow.Flow) error {
					callCount++
					if callCount == 2 {
						return tt.saveErr
					}
					storage.savedFlows = append(storage.savedFlows, f)
					return nil
				}
			}

			reporter := NewLocalReporter(storage)

			ctx := context.Background()
			err := reporter.ReportBatch(ctx, tt.flows)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectErr && len(storage.savedFlows) != tt.savedCount {
				t.Errorf("Expected %d saved flows, got %d", tt.savedCount, len(storage.savedFlows))
			}
		})
	}
}

// Test_LocalReporter_Start tests reporter startup
func Test_LocalReporter_Start(t *testing.T) {
	storage := &MockStorage{}
	reporter := NewLocalReporter(storage)

	err := reporter.Start()
	if err != nil {
		t.Errorf("Expected no error from Start(), got: %v", err)
	}
}

// Test_LocalReporter_Stop tests reporter shutdown
func Test_LocalReporter_Stop(t *testing.T) {
	storage := &MockStorage{}
	reporter := NewLocalReporter(storage)

	err := reporter.Stop()
	if err != nil {
		t.Errorf("Expected no error from Stop(), got: %v", err)
	}
}

// Test_LocalReporter_ZeroOverhead verifies that LocalReporter adds minimal overhead
func Test_LocalReporter_ZeroOverhead(t *testing.T) {
	storage := &MockStorage{}
	reporter := NewLocalReporter(storage)

	// This test verifies that Report() is a thin wrapper around storage.SaveFlow()
	testFlow := &flow.Flow{
		SourceIP: "10.0.0.1",
		DestIP:   "10.0.0.2",
	}

	ctx := context.Background()
	err := reporter.Report(ctx, testFlow)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify the flow was saved
	if len(storage.savedFlows) != 1 {
		t.Errorf("Expected 1 flow to be saved, got %d", len(storage.savedFlows))
	}

	// Verify it's the exact same flow (no copying/transformation)
	if storage.savedFlows[0] != testFlow {
		t.Error("Expected the same flow instance to be passed to storage")
	}
}

// Test_LocalReporter_ContextCancellation tests behavior when context is cancelled
func Test_LocalReporter_ContextCancellation(t *testing.T) {
	storage := &MockStorage{
		SaveFlowFunc: func(f *flow.Flow) error {
			// Storage doesn't actually respect context in this simple implementation
			// but we're testing that the reporter handles it gracefully
			return nil
		},
	}
	reporter := NewLocalReporter(storage)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	testFlow := &flow.Flow{
		SourceIP: "10.0.0.1",
		DestIP:   "10.0.0.2",
	}

	// LocalReporter doesn't check context cancellation (by design for standalone mode)
	// This test documents that behavior
	err := reporter.Report(ctx, testFlow)
	if err != nil {
		t.Errorf("LocalReporter should not fail on cancelled context: %v", err)
	}
}

// Test_LocalReporter_NilFlow tests handling of nil flow
func Test_LocalReporter_NilFlow(t *testing.T) {
	storage := &MockStorage{}
	reporter := NewLocalReporter(storage)

	ctx := context.Background()

	// This will likely panic or be handled by storage layer
	// Test documents the behavior
	err := reporter.Report(ctx, nil)

	// We expect this to pass through to storage without crashing in reporter
	_ = err // Storage may or may not handle nil gracefully
}
