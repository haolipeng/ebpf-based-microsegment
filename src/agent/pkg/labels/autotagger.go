// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: workload metadata (runtime, K8s), auto-tag rules
// output: auto-generated labels based on rules
// pos: label auto-tagger - if file updated, must sync with this header comment and pkg/labels/CLAUDE.md
package labels

import (
	"strings"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/workload"
)

// AutoTagger provides automatic label inference based on workload characteristics
type AutoTagger struct {
	// Configuration options
	enabled bool
}

// NewAutoTagger creates a new auto-tagger instance
func NewAutoTagger() *AutoTagger {
	return &AutoTagger{
		enabled: true,
	}
}

// SetEnabled enables or disables auto-tagging
func (a *AutoTagger) SetEnabled(enabled bool) {
	a.enabled = enabled
}

// IsEnabled returns whether auto-tagging is enabled
func (a *AutoTagger) IsEnabled() bool {
	return a.enabled
}

// InferRoleFromImage infers the role label from a container image name
// Returns empty string if role cannot be inferred
func InferRoleFromImage(image string) string {
	if image == "" {
		return ""
	}

	// Trim whitespace
	image = strings.TrimSpace(image)
	if image == "" {
		return ""
	}

	// Convert to lowercase for case-insensitive matching
	imageLower := strings.ToLower(image)

	// Remove tag/version/sha256 (e.g., "nginx:1.19" -> "nginx", "nginx@sha256:..." -> "nginx")
	imageName := imageLower

	// Handle @ for sha256
	if idx := strings.Index(imageLower, "@"); idx != -1 {
		imageName = imageLower[:idx]
	}

	// Handle : for tags (but be careful with registry ports like registry.io:5000)
	// Split by colon and take the last part that doesn't look like a port
	parts := strings.Split(imageName, ":")
	if len(parts) > 1 {
		// If last part is numeric (likely a port), keep everything; otherwise remove tag
		lastPart := parts[len(parts)-1]
		// Check if it's all digits (port number)
		isPort := true
		for _, c := range lastPart {
			if c < '0' || c > '9' {
				isPort = false
				break
			}
		}
		if !isPort {
			// Remove the tag part
			imageName = strings.Join(parts[:len(parts)-1], ":")
		}
	}

	// Remove registry prefix (e.g., "docker.io/nginx" -> "nginx", "registry.io:5000/nginx" -> "nginx")
	if idx := strings.LastIndex(imageName, "/"); idx != -1 {
		imageName = imageName[idx+1:]
	}

	// Web servers
	if strings.Contains(imageName, "nginx") ||
		strings.Contains(imageName, "apache") ||
		strings.Contains(imageName, "httpd") ||
		strings.Contains(imageName, "caddy") ||
		strings.Contains(imageName, "lighttpd") {
		return "web"
	}

	// API/Application servers
	if strings.Contains(imageName, "tomcat") ||
		strings.Contains(imageName, "jetty") ||
		strings.Contains(imageName, "wildfly") ||
		strings.Contains(imageName, "jboss") {
		return "api"
	}

	// Databases
	if strings.Contains(imageName, "mysql") ||
		strings.Contains(imageName, "mariadb") ||
		strings.Contains(imageName, "postgres") ||
		strings.Contains(imageName, "postgresql") ||
		strings.Contains(imageName, "mongo") ||
		strings.Contains(imageName, "cassandra") ||
		strings.Contains(imageName, "couchdb") ||
		strings.Contains(imageName, "cockroach") ||
		strings.Contains(imageName, "timescale") {
		return "db"
	}

	// Cache systems
	if strings.Contains(imageName, "redis") ||
		strings.Contains(imageName, "memcached") ||
		strings.Contains(imageName, "varnish") ||
		strings.Contains(imageName, "hazelcast") {
		return "cache"
	}

	// Message queues
	if strings.Contains(imageName, "rabbitmq") ||
		strings.Contains(imageName, "kafka") ||
		strings.Contains(imageName, "activemq") ||
		strings.Contains(imageName, "nats") ||
		strings.Contains(imageName, "pulsar") {
		return "mq"
	}

	// Load balancers / Proxies
	if strings.Contains(imageName, "haproxy") ||
		strings.Contains(imageName, "envoy") ||
		strings.Contains(imageName, "traefik") ||
		strings.Contains(imageName, "kong") {
		return "lb"
	}

	// Search engines
	if strings.Contains(imageName, "elasticsearch") ||
		strings.Contains(imageName, "solr") ||
		strings.Contains(imageName, "opensearch") {
		return "search"
	}

	// Monitoring/Observability
	if strings.Contains(imageName, "prometheus") ||
		strings.Contains(imageName, "grafana") ||
		strings.Contains(imageName, "jaeger") ||
		strings.Contains(imageName, "zipkin") {
		return "monitoring"
	}

	// Cannot infer role
	return ""
}

// InferRoleFromPorts infers the role label from listening ports
// Returns empty string if role cannot be inferred
// Ports should be a list of listening ports (e.g., [80, 443])
func InferRoleFromPorts(ports []uint16) string {
	if len(ports) == 0 {
		return ""
	}

	// Create a map for quick lookup
	portSet := make(map[uint16]bool)
	for _, port := range ports {
		portSet[port] = true
	}

	// Web servers (HTTP/HTTPS)
	if portSet[80] || portSet[443] || portSet[8080] || portSet[8443] {
		return "web"
	}

	// MySQL/MariaDB
	if portSet[3306] {
		return "db"
	}

	// PostgreSQL
	if portSet[5432] {
		return "db"
	}

	// MongoDB
	if portSet[27017] || portSet[27018] || portSet[27019] {
		return "db"
	}

	// Redis
	if portSet[6379] {
		return "cache"
	}

	// Memcached
	if portSet[11211] {
		return "cache"
	}

	// RabbitMQ
	if portSet[5672] || portSet[15672] {
		return "mq"
	}

	// Kafka
	if portSet[9092] || portSet[9093] {
		return "mq"
	}

	// Elasticsearch
	if portSet[9200] || portSet[9300] {
		return "search"
	}

	// Prometheus
	if portSet[9090] {
		return "monitoring"
	}

	// Grafana
	if portSet[3000] {
		return "monitoring"
	}

	// Cannot infer role from ports
	return ""
}

// AutoTagWorkload automatically generates suggested labels for a workload
// Returns a map of suggested labels
// These labels should be reviewed and approved before being applied
func (a *AutoTagger) AutoTagWorkload(wl *workload.Workload) map[string]string {
	if !a.enabled || wl == nil {
		return nil
	}

	suggestedLabels := make(map[string]string)

	// Try to infer role from image
	if wl.Image != "" {
		if role := InferRoleFromImage(wl.Image); role != "" {
			suggestedLabels["role"] = role
			suggestedLabels["role_inference_source"] = "image"
		}
	}

	// If role not inferred from image, try ports
	if _, hasRole := suggestedLabels["role"]; !hasRole && len(wl.Ports) > 0 {
		if role := InferRoleFromPorts(wl.Ports); role != "" {
			suggestedLabels["role"] = role
			suggestedLabels["role_inference_source"] = "ports"
		}
	}

	// Mark all labels as inferred
	if len(suggestedLabels) > 0 {
		suggestedLabels["inferred"] = "true"
		return suggestedLabels
	}

	// Return nil if no labels were inferred
	return nil
}

// AutoTagWorkloadWithMerge automatically generates and merges labels with existing labels
// User-defined labels always take precedence over inferred labels
// Returns the merged label set
func (a *AutoTagger) AutoTagWorkloadWithMerge(wl *workload.Workload) map[string]string {
	if wl == nil {
		return nil
	}

	// Start with existing labels
	mergedLabels := make(map[string]string)
	for k, v := range wl.Labels {
		mergedLabels[k] = v
	}

	// Get suggested labels
	suggestedLabels := a.AutoTagWorkload(wl)

	// Merge suggested labels (only add if key doesn't exist)
	for k, v := range suggestedLabels {
		if _, exists := mergedLabels[k]; !exists {
			mergedLabels[k] = v
		}
	}

	return mergedLabels
}

// GetInferenceConfidence returns a confidence score (0.0-1.0) for the inference
// This is a simple heuristic based on the inference source
func GetInferenceConfidence(inferenceSource string) float64 {
	switch inferenceSource {
	case "image":
		return 0.8 // Image-based inference is usually reliable
	case "ports":
		return 0.6 // Port-based inference is less reliable (many services use common ports)
	default:
		return 0.0
	}
}

// ImageRoleMapping returns the complete mapping of image patterns to roles
// This is useful for documentation and testing
func ImageRoleMapping() map[string]string {
	return map[string]string{
		// Web servers
		"nginx":     "web",
		"apache":    "web",
		"httpd":     "web",
		"caddy":     "web",
		"lighttpd":  "web",

		// API/Application servers
		"tomcat":   "api",
		"jetty":    "api",
		"wildfly":  "api",
		"jboss":    "api",

		// Databases
		"mysql":      "db",
		"mariadb":    "db",
		"postgres":   "db",
		"postgresql": "db",
		"mongo":      "db",
		"cassandra":  "db",
		"couchdb":    "db",
		"cockroach":  "db",
		"timescale":  "db",

		// Cache
		"redis":      "cache",
		"memcached":  "cache",
		"varnish":    "cache",
		"hazelcast":  "cache",

		// Message queues
		"rabbitmq":  "mq",
		"kafka":     "mq",
		"activemq":  "mq",
		"nats":      "mq",
		"pulsar":    "mq",

		// Load balancers
		"haproxy": "lb",
		"envoy":   "lb",
		"traefik": "lb",
		"kong":    "lb",

		// Search
		"elasticsearch": "search",
		"solr":          "search",
		"opensearch":    "search",

		// Monitoring
		"prometheus": "monitoring",
		"grafana":    "monitoring",
		"jaeger":     "monitoring",
		"zipkin":     "monitoring",
	}
}

// PortRoleMapping returns the complete mapping of ports to roles
// This is useful for documentation and testing
func PortRoleMapping() map[uint16]string {
	return map[uint16]string{
		// Web
		80:   "web",
		443:  "web",
		8080: "web",
		8443: "web",

		// Databases
		3306:  "db",      // MySQL
		5432:  "db",      // PostgreSQL
		27017: "db",      // MongoDB
		27018: "db",      // MongoDB
		27019: "db",      // MongoDB

		// Cache
		6379:  "cache",   // Redis
		11211: "cache",   // Memcached

		// Message queues
		5672:  "mq",      // RabbitMQ
		15672: "mq",      // RabbitMQ Management
		9092:  "mq",      // Kafka
		9093:  "mq",      // Kafka

		// Search
		9200: "search",   // Elasticsearch
		9300: "search",   // Elasticsearch

		// Monitoring
		9090: "monitoring", // Prometheus
		3000: "monitoring", // Grafana
	}
}
