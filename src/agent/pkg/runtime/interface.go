// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package runtime

// RuntimeDetector defines the interface for detecting and extracting labels from container runtimes
type RuntimeDetector interface {
	// GetContainerLabels retrieves labels for a specific container
	GetContainerLabels(containerID string) (map[string]string, error)

	// ListContainersWithLabels lists all containers and their labels
	ListContainersWithLabels() (map[string]map[string]string, error)

	// WatchContainerEvents watches for container lifecycle events
	// The callback is invoked for each event (create, start, stop, remove)
	WatchContainerEvents(callback func(ContainerEvent)) error

	// Close closes the detector and releases resources
	Close() error
}

// ContainerEvent represents a container lifecycle event
type ContainerEvent struct {
	Type        EventType         // Event type (created, started, stopped, removed)
	ContainerID string            // Container ID
	Labels      map[string]string // Container labels
	Timestamp   int64             // Event timestamp (Unix timestamp)
}

// EventType represents the type of container event
type EventType string

const (
	// EventCreated is emitted when a container is created
	EventCreated EventType = "created"

	// EventStarted is emitted when a container starts
	EventStarted EventType = "started"

	// EventStopped is emitted when a container stops
	EventStopped EventType = "stopped"

	// EventRemoved is emitted when a container is removed
	EventRemoved EventType = "removed"
)

// ContainerInfo holds basic information about a container
type ContainerInfo struct {
	ID          string            // Container ID
	Name        string            // Container name
	Image       string            // Container image
	Labels      map[string]string // Container labels
	State       string            // Container state (running, stopped, etc.)
	CreatedAt   int64             // Creation timestamp
	PodName     string            // Kubernetes Pod name (if applicable)
	PodNS       string            // Kubernetes Pod namespace (if applicable)
}
