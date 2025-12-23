// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: Docker API requests
// output: container metadata (labels, IPs, names)
// pos: Docker runtime adapter - if file updated, must sync with this header comment and pkg/runtime/CLAUDE.md
package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

const (
	// Default Docker socket path
	DefaultDockerSocket = "unix:///var/run/docker.sock"

	// Docker label prefixes to filter
	DockerLabelPrefix = "com.docker."
	K8sDockerPrefix   = "io.kubernetes."
)

// DockerDetector detects containers and extracts labels from Docker runtime
type DockerDetector struct {
	client     *client.Client
	ctx        context.Context
	cancel     context.CancelFunc
	socketPath string
}

// NewDockerDetector creates a new Docker detector
func NewDockerDetector(socketPath string) (*DockerDetector, error) {
	if socketPath == "" {
		socketPath = DefaultDockerSocket
	}

	// Create Docker client
	cli, err := client.NewClientWithOpts(
		client.WithHost(socketPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker at %s: %w", socketPath, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Verify connection by pinging Docker daemon
	_, err = cli.Ping(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to ping Docker daemon: %w", err)
	}

	detector := &DockerDetector{
		client:     cli,
		ctx:        ctx,
		cancel:     cancel,
		socketPath: socketPath,
	}

	log.Infof("Connected to Docker at %s", socketPath)
	return detector, nil
}

// GetContainerLabels retrieves labels for a specific container
func (d *DockerDetector) GetContainerLabels(containerID string) (map[string]string, error) {
	// Inspect container
	containerJSON, err := d.client.ContainerInspect(d.ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	labels := make(map[string]string)

	// Extract user-defined labels (filter out Docker and Kubernetes system labels)
	for k, v := range containerJSON.Config.Labels {
		// Skip Docker system labels
		if strings.HasPrefix(k, DockerLabelPrefix) {
			continue
		}

		// Handle Kubernetes system labels
		if strings.HasPrefix(k, K8sDockerPrefix) {
			// Extract useful system information
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
func (d *DockerDetector) ListContainersWithLabels() (map[string]map[string]string, error) {
	// List all containers (running and stopped)
	containers, err := d.client.ContainerList(d.ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make(map[string]map[string]string)

	for _, container := range containers {
		labels, err := d.GetContainerLabels(container.ID)
		if err != nil {
			log.Warnf("Failed to get labels for container %s: %v", container.ID, err)
			continue
		}
		result[container.ID] = labels
	}

	log.Debugf("Listed %d containers with labels", len(result))
	return result, nil
}

// WatchContainerEvents watches for container lifecycle events
func (d *DockerDetector) WatchContainerEvents(callback func(ContainerEvent)) error {
	// Subscribe to Docker events
	eventCh, errCh := d.client.Events(d.ctx, events.ListOptions{})

	go func() {
		for {
			select {
			case event := <-eventCh:
				// Parse event
				parsedEvent := d.parseEvent(event)
				if parsedEvent != nil {
					callback(*parsedEvent)
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

	log.Info("Started watching Docker container events")
	return nil
}

// parseEvent converts a Docker event to a ContainerEvent
func (d *DockerDetector) parseEvent(event events.Message) *ContainerEvent {
	// Filter to container events only
	if event.Type != events.ContainerEventType {
		return nil
	}

	// Determine event type based on action
	var eventType EventType
	switch event.Action {
	case "create":
		eventType = EventCreated
	case "start":
		eventType = EventStarted
	case "die", "stop":
		eventType = EventStopped
	case "destroy":
		eventType = EventRemoved
	default:
		// Ignore other event types
		return nil
	}

	// Extract container ID
	containerID := event.Actor.ID

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
		Timestamp:   event.Time,
	}
}

// GetContainerInfo retrieves detailed information about a container
func (d *DockerDetector) GetContainerInfo(containerID string) (*ContainerInfo, error) {
	// Inspect container
	containerJSON, err := d.client.ContainerInspect(d.ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	// Extract labels
	labels, _ := d.GetContainerLabels(containerID)

	// Extract Kubernetes info
	podName := containerJSON.Config.Labels["io.kubernetes.pod.name"]
	podNS := containerJSON.Config.Labels["io.kubernetes.pod.namespace"]

	// Determine state
	state := "unknown"
	if containerJSON.State != nil {
		if containerJSON.State.Running {
			state = "running"
		} else if containerJSON.State.Paused {
			state = "paused"
		} else {
			state = "stopped"
		}
	}

	// Get created timestamp
	var createdAt int64
	if containerJSON.Created != "" {
		if t, err := time.Parse(time.RFC3339Nano, containerJSON.Created); err == nil {
			createdAt = t.Unix()
		}
	}

	containerInfo := &ContainerInfo{
		ID:        containerID,
		Name:      strings.TrimPrefix(containerJSON.Name, "/"), // Docker adds "/" prefix
		Image:     containerJSON.Config.Image,
		Labels:    labels,
		State:     state,
		CreatedAt: createdAt,
		PodName:   podName,
		PodNS:     podNS,
	}

	return containerInfo, nil
}

// Close closes the detector and releases resources
func (d *DockerDetector) Close() error {
	if d.cancel != nil {
		d.cancel()
	}

	if d.client != nil {
		err := d.client.Close()
		if err != nil {
			return fmt.Errorf("failed to close Docker client: %w", err)
		}
	}

	log.Info("Docker detector closed")
	return nil
}
