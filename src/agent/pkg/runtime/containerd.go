// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/events"
	"github.com/containerd/containerd/namespaces"
	log "github.com/sirupsen/logrus"
)

const (
	// Default containerd socket path
	DefaultContainerdSocket = "/run/containerd/containerd.sock"

	// Kubernetes namespace in containerd
	K8sNamespace = "k8s.io"

	// Kubernetes label prefixes to filter
	K8sLabelPrefix = "io.kubernetes."
)

// ContainerdDetector detects containers and extracts labels from containerd runtime
type ContainerdDetector struct {
	client      *containerd.Client
	namespace   string
	ctx         context.Context
	cancel      context.CancelFunc
	socketPath  string
}

// NewContainerdDetector creates a new containerd detector
func NewContainerdDetector(socketPath string) (*ContainerdDetector, error) {
	if socketPath == "" {
		socketPath = DefaultContainerdSocket
	}

	client, err := containerd.New(socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to containerd at %s: %w", socketPath, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	detector := &ContainerdDetector{
		client:     client,
		namespace:  K8sNamespace,
		ctx:        ctx,
		cancel:     cancel,
		socketPath: socketPath,
	}

	log.Infof("Connected to containerd at %s", socketPath)
	return detector, nil
}

// GetContainerLabels retrieves labels for a specific container
func (d *ContainerdDetector) GetContainerLabels(containerID string) (map[string]string, error) {
	ctx := namespaces.WithNamespace(d.ctx, d.namespace)

	// Load container
	container, err := d.client.LoadContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load container %s: %w", containerID, err)
	}

	// Get container info
	info, err := container.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info for %s: %w", containerID, err)
	}

	labels := make(map[string]string)

	// Extract user-defined labels (filter out Kubernetes system labels)
	for k, v := range info.Labels {
		// Skip Kubernetes system labels
		if strings.HasPrefix(k, K8sLabelPrefix) {
			// Optionally extract useful system information
			switch k {
			case "io.kubernetes.pod.namespace":
				labels["k8s.namespace"] = v
			case "io.kubernetes.pod.name":
				labels["k8s.pod"] = v
			case "io.kubernetes.pod.uid":
				labels["k8s.pod_uid"] = v
			case "io.kubernetes.container.name":
				labels["k8s.container"] = v
			}
			continue
		}

		// Preserve user-defined labels
		labels[k] = v
	}

	log.Debugf("Extracted %d labels for container %s", len(labels), containerID)
	return labels, nil
}

// ListContainersWithLabels lists all containers and their labels
func (d *ContainerdDetector) ListContainersWithLabels() (map[string]map[string]string, error) {
	ctx := namespaces.WithNamespace(d.ctx, d.namespace)

	// List all containers
	containers, err := d.client.Containers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make(map[string]map[string]string)

	for _, container := range containers {
		labels, err := d.GetContainerLabels(container.ID())
		if err != nil {
			log.Warnf("Failed to get labels for container %s: %v", container.ID(), err)
			continue
		}
		result[container.ID()] = labels
	}

	log.Debugf("Listed %d containers with labels", len(result))
	return result, nil
}

// WatchContainerEvents watches for container lifecycle events
// This is a simplified implementation - full event watching would require more complex subscription logic
func (d *ContainerdDetector) WatchContainerEvents(callback func(ContainerEvent)) error {
	ctx := namespaces.WithNamespace(d.ctx, d.namespace)

	// Subscribe to container events
	eventCh, errCh := d.client.EventService().Subscribe(ctx)

	go func() {
		for {
			select {
			case envelope := <-eventCh:
				// Parse event
				event := d.parseEvent(envelope)
				if event != nil {
					callback(*event)
				}

			case err := <-errCh:
				if err != nil {
					log.Errorf("Container event error: %v", err)
				}
				return

			case <-d.ctx.Done():
				log.Info("Container event watcher stopped")
				return
			}
		}
	}()

	log.Info("Started watching container events")
	return nil
}

// parseEvent converts a containerd event envelope to a ContainerEvent
func (d *ContainerdDetector) parseEvent(envelope *events.Envelope) *ContainerEvent {
	if envelope == nil {
		return nil
	}

	// Determine event type based on topic
	var eventType EventType
	switch envelope.Topic {
	case "/containers/create":
		eventType = EventCreated
	case "/containers/delete":
		eventType = EventRemoved
	case "/tasks/start":
		eventType = EventStarted
	case "/tasks/exit":
		eventType = EventStopped
	default:
		// Ignore other event types
		return nil
	}

	// Extract container ID from event
	// The event structure varies by type, so we need to handle this carefully
	// For simplicity, we'll extract from the namespace
	containerID := envelope.Namespace

	// Try to get labels for this container
	labels, err := d.GetContainerLabels(containerID)
	if err != nil {
		log.Debugf("Could not get labels for container %s in event: %v", containerID, err)
		labels = make(map[string]string)
	}

	return &ContainerEvent{
		Type:        eventType,
		ContainerID: containerID,
		Labels:      labels,
		Timestamp:   time.Now().Unix(),
	}
}

// GetContainerInfo retrieves detailed information about a container
func (d *ContainerdDetector) GetContainerInfo(containerID string) (*ContainerInfo, error) {
	ctx := namespaces.WithNamespace(d.ctx, d.namespace)

	// Load container
	container, err := d.client.LoadContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load container %s: %w", containerID, err)
	}

	// Get container info
	info, err := container.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info for %s: %w", containerID, err)
	}

	// Get image name
	image, err := container.Image(ctx)
	imageName := ""
	if err == nil && image != nil {
		imageName = image.Name()
	}

	// Get task (for state)
	task, err := container.Task(ctx, nil)
	state := "unknown"
	if err == nil && task != nil {
		status, _ := task.Status(ctx)
		state = string(status.Status)
	}

	// Extract labels
	labels, _ := d.GetContainerLabels(containerID)

	// Extract Kubernetes info
	podName := info.Labels["io.kubernetes.pod.name"]
	podNS := info.Labels["io.kubernetes.pod.namespace"]

	containerInfo := &ContainerInfo{
		ID:        containerID,
		Name:      info.ID, // containerd uses ID as name
		Image:     imageName,
		Labels:    labels,
		State:     state,
		CreatedAt: info.CreatedAt.Unix(),
		PodName:   podName,
		PodNS:     podNS,
	}

	return containerInfo, nil
}

// Close closes the detector and releases resources
func (d *ContainerdDetector) Close() error {
	if d.cancel != nil {
		d.cancel()
	}

	if d.client != nil {
		err := d.client.Close()
		if err != nil {
			return fmt.Errorf("failed to close containerd client: %w", err)
		}
	}

	log.Info("Containerd detector closed")
	return nil
}
