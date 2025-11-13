// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package labels

import (
	"testing"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/workload"
)

// TestInferRoleFromImage tests role inference from container image names
func TestInferRoleFromImage(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		wantRole string
	}{
		// Web servers
		{"nginx latest", "nginx:latest", "web"},
		{"nginx with version", "nginx:1.19", "web"},
		{"nginx with registry", "docker.io/nginx:latest", "web"},
		{"apache", "apache:2.4", "web"},
		{"httpd", "httpd:latest", "web"},
		{"caddy", "caddy:2.0", "web"},
		{"lighttpd", "lighttpd", "web"},

		// Case insensitivity
		{"NGINX uppercase", "NGINX:latest", "web"},
		{"Nginx mixed case", "Nginx:Latest", "web"},

		// Databases
		{"mysql", "mysql:8.0", "db"},
		{"mariadb", "mariadb:10.5", "db"},
		{"postgres", "postgres:13", "db"},
		{"postgresql", "postgresql:latest", "db"},
		{"mongo", "mongo:4.4", "db"},
		{"mongodb", "mongodb:latest", "db"},
		{"cassandra", "cassandra:3.11", "db"},
		{"couchdb", "couchdb:3.0", "db"},
		{"cockroachdb", "cockroachdb/cockroach:latest", "db"},
		{"timescaledb", "timescale/timescaledb:latest", "db"},

		// Cache systems
		{"redis", "redis:6.2", "cache"},
		{"redis alpine", "redis:alpine", "cache"},
		{"memcached", "memcached:latest", "cache"},
		{"varnish", "varnish:6.0", "cache"},
		{"hazelcast", "hazelcast/hazelcast:latest", "cache"},

		// Message queues
		{"rabbitmq", "rabbitmq:3.8", "mq"},
		{"kafka", "kafka:2.8", "mq"},
		{"confluentinc kafka", "confluentinc/cp-kafka:latest", "mq"},
		{"activemq", "activemq:latest", "mq"},
		{"nats", "nats:latest", "mq"},
		{"pulsar", "apachepulsar/pulsar:latest", "mq"},

		// Load balancers
		{"haproxy", "haproxy:2.4", "lb"},
		{"envoy", "envoyproxy/envoy:latest", "lb"},
		{"traefik", "traefik:v2.5", "lb"},
		{"kong", "kong:latest", "lb"},

		// Application servers
		{"tomcat", "tomcat:9.0", "api"},
		{"jetty", "jetty:latest", "api"},
		{"wildfly", "wildfly:latest", "api"},
		{"jboss", "jboss/wildfly:latest", "api"},

		// Search engines
		{"elasticsearch", "elasticsearch:7.10", "search"},
		{"solr", "solr:8.0", "search"},
		{"opensearch", "opensearchproject/opensearch:latest", "search"},

		// Monitoring
		{"prometheus", "prom/prometheus:latest", "monitoring"},
		{"grafana", "grafana/grafana:latest", "monitoring"},
		{"jaeger", "jaeger:latest", "monitoring"},  // Changed to simpler image name
		{"zipkin", "openzipkin/zipkin:latest", "monitoring"},

		// Unknown/custom images
		{"custom app", "mycompany/custom-app:v1.0", ""},
		{"python base", "python:3.9", ""},
		{"node base", "node:16", ""},
		{"empty string", "", ""},
		{"just a name", "myapp", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferRoleFromImage(tt.image)
			if got != tt.wantRole {
				t.Errorf("InferRoleFromImage(%q) = %q, want %q", tt.image, got, tt.wantRole)
			}
		})
	}
}

// TestInferRoleFromPorts tests role inference from listening ports
func TestInferRoleFromPorts(t *testing.T) {
	tests := []struct {
		name     string
		ports    []uint16
		wantRole string
	}{
		// Web servers
		{"HTTP only", []uint16{80}, "web"},
		{"HTTPS only", []uint16{443}, "web"},
		{"HTTP and HTTPS", []uint16{80, 443}, "web"},
		{"alternative HTTP", []uint16{8080}, "web"},
		{"alternative HTTPS", []uint16{8443}, "web"},
		{"HTTP with custom", []uint16{80, 9000}, "web"},

		// Databases
		{"MySQL", []uint16{3306}, "db"},
		{"PostgreSQL", []uint16{5432}, "db"},
		{"MongoDB primary", []uint16{27017}, "db"},
		{"MongoDB shard", []uint16{27018}, "db"},
		{"MongoDB config", []uint16{27019}, "db"},
		{"MySQL and custom", []uint16{3306, 33060}, "db"},

		// Cache
		{"Redis", []uint16{6379}, "cache"},
		{"Memcached", []uint16{11211}, "cache"},
		{"Redis with custom", []uint16{6379, 16379}, "cache"},

		// Message queues
		{"RabbitMQ AMQP", []uint16{5672}, "mq"},
		{"RabbitMQ management", []uint16{15672}, "mq"},
		{"RabbitMQ both", []uint16{5672, 15672}, "mq"},
		{"Kafka", []uint16{9092}, "mq"},
		{"Kafka SSL", []uint16{9093}, "mq"},

		// Search
		{"Elasticsearch HTTP", []uint16{9200}, "search"},
		{"Elasticsearch transport", []uint16{9300}, "search"},
		{"Elasticsearch both", []uint16{9200, 9300}, "search"},

		// Monitoring
		{"Prometheus", []uint16{9090}, "monitoring"},
		{"Grafana", []uint16{3000}, "monitoring"},

		// Edge cases
		{"empty ports", []uint16{}, ""},
		{"nil ports", nil, ""},
		{"unknown port", []uint16{12345}, ""},
		{"multiple unknown", []uint16{12345, 54321}, ""},
		{"SSH only", []uint16{22}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferRoleFromPorts(tt.ports)
			if got != tt.wantRole {
				t.Errorf("InferRoleFromPorts(%v) = %q, want %q", tt.ports, got, tt.wantRole)
			}
		})
	}
}

// TestAutoTagWorkload tests automatic workload tagging
func TestAutoTagWorkload(t *testing.T) {
	tagger := NewAutoTagger()

	tests := []struct {
		name          string
		workload      *workload.Workload
		wantLabels    map[string]string
		checkRole     bool
		checkInferred bool
		checkSource   bool
	}{
		{
			name: "nginx with image inference",
			workload: &workload.Workload{
				ID:    "wl-1",
				Name:  "nginx-server",
				Image: "nginx:latest",
			},
			wantLabels: map[string]string{
				"role":                   "web",
				"role_inference_source":  "image",
				"inferred":               "true",
			},
			checkRole:     true,
			checkInferred: true,
			checkSource:   true,
		},
		{
			name: "redis with port inference",
			workload: &workload.Workload{
				ID:    "wl-2",
				Name:  "cache-server",
				Image: "mycompany/custom-cache:v1",  // Changed from custom-redis to avoid image match
				Ports: []uint16{6379},
			},
			wantLabels: map[string]string{
				"role":                   "cache",
				"role_inference_source":  "ports",
				"inferred":               "true",
			},
			checkRole:     true,
			checkInferred: true,
			checkSource:   true,
		},
		{
			name: "mysql with both image and ports",
			workload: &workload.Workload{
				ID:    "wl-3",
				Name:  "mysql-db",
				Image: "mysql:8.0",
				Ports: []uint16{3306},
			},
			wantLabels: map[string]string{
				"role":                   "db",
				"role_inference_source":  "image", // Image takes precedence
				"inferred":               "true",
			},
			checkRole:     true,
			checkInferred: true,
			checkSource:   true,
		},
		{
			name: "custom app - no inference",
			workload: &workload.Workload{
				ID:    "wl-4",
				Name:  "custom-app",
				Image: "mycompany/custom-app:v1.0",
				Ports: []uint16{8000},
			},
			wantLabels:    nil,
			checkRole:     false,
			checkInferred: false,
			checkSource:   false,
		},
		{
			name:          "nil workload",
			workload:      nil,
			wantLabels:    nil,
			checkRole:     false,
			checkInferred: false,
			checkSource:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tagger.AutoTagWorkload(tt.workload)

			if tt.wantLabels == nil {
				if got != nil {
					t.Errorf("AutoTagWorkload() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("AutoTagWorkload() = nil, want labels")
			}

			if tt.checkRole {
				if role, ok := got["role"]; !ok {
					t.Errorf("missing 'role' label")
				} else if role != tt.wantLabels["role"] {
					t.Errorf("role = %q, want %q", role, tt.wantLabels["role"])
				}
			}

			if tt.checkInferred {
				if inferred, ok := got["inferred"]; !ok {
					t.Errorf("missing 'inferred' label")
				} else if inferred != "true" {
					t.Errorf("inferred = %q, want 'true'", inferred)
				}
			}

			if tt.checkSource {
				if source, ok := got["role_inference_source"]; !ok {
					t.Errorf("missing 'role_inference_source' label")
				} else if source != tt.wantLabels["role_inference_source"] {
					t.Errorf("role_inference_source = %q, want %q", source, tt.wantLabels["role_inference_source"])
				}
			}
		})
	}
}

// TestAutoTagWorkloadWithMerge tests label merging with existing labels
func TestAutoTagWorkloadWithMerge(t *testing.T) {
	tagger := NewAutoTagger()

	tests := []struct {
		name           string
		workload       *workload.Workload
		existingLabels map[string]string
		wantRole       string
		userDefinedWins bool
	}{
		{
			name: "merge with empty labels",
			workload: &workload.Workload{
				ID:     "wl-1",
				Name:   "nginx",
				Image:  "nginx:latest",
				Labels: map[string]string{},
			},
			wantRole: "web",
			userDefinedWins: false,
		},
		{
			name: "user-defined role takes precedence",
			workload: &workload.Workload{
				ID:    "wl-2",
				Name:  "custom-nginx",
				Image: "nginx:latest",
				Labels: map[string]string{
					"role": "custom-web", // User-defined
					"app":  "frontend",
				},
			},
			wantRole: "custom-web", // User's value preserved
			userDefinedWins: true,
		},
		{
			name: "merge adds inferred labels",
			workload: &workload.Workload{
				ID:    "wl-3",
				Name:  "mysql",
				Image: "mysql:8.0",
				Labels: map[string]string{
					"app": "database",
					"env": "production",
				},
			},
			wantRole: "db",
			userDefinedWins: false,
		},
		{
			name: "nil workload",
			workload: nil,
			wantRole: "",
			userDefinedWins: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tagger.AutoTagWorkloadWithMerge(tt.workload)

			if tt.workload == nil {
				if got != nil {
					t.Errorf("AutoTagWorkloadWithMerge(nil) = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("AutoTagWorkloadWithMerge() = nil, want labels")
			}

			// Check role
			if role, ok := got["role"]; ok {
				if role != tt.wantRole {
					t.Errorf("role = %q, want %q", role, tt.wantRole)
				}
			} else if tt.wantRole != "" {
				t.Errorf("missing 'role' label, want %q", tt.wantRole)
			}

			// Verify user-defined labels are preserved
			if tt.workload != nil && tt.workload.Labels != nil {
				for k, v := range tt.workload.Labels {
					if gotV, ok := got[k]; !ok {
						t.Errorf("user-defined label %q missing", k)
					} else if gotV != v {
						t.Errorf("user-defined label %q = %q, want %q", k, gotV, v)
					}
				}
			}
		})
	}
}

// TestAutoTaggerEnableDisable tests enabling/disabling the tagger
func TestAutoTaggerEnableDisable(t *testing.T) {
	tagger := NewAutoTagger()

	// Should be enabled by default
	if !tagger.IsEnabled() {
		t.Error("tagger should be enabled by default")
	}

	// Disable
	tagger.SetEnabled(false)
	if tagger.IsEnabled() {
		t.Error("tagger should be disabled")
	}

	// Test that disabled tagger returns nil
	wl := &workload.Workload{
		ID:    "wl-1",
		Name:  "nginx",
		Image: "nginx:latest",
	}

	labels := tagger.AutoTagWorkload(wl)
	if labels != nil {
		t.Errorf("disabled tagger should return nil, got %v", labels)
	}

	// Re-enable
	tagger.SetEnabled(true)
	if !tagger.IsEnabled() {
		t.Error("tagger should be enabled")
	}

	labels = tagger.AutoTagWorkload(wl)
	if labels == nil {
		t.Error("enabled tagger should return labels")
	}
}

// TestGetInferenceConfidence tests confidence scoring
func TestGetInferenceConfidence(t *testing.T) {
	tests := []struct {
		source         string
		wantConfidence float64
	}{
		{"image", 0.8},
		{"ports", 0.6},
		{"unknown", 0.0},
		{"", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := GetInferenceConfidence(tt.source)
			if got != tt.wantConfidence {
				t.Errorf("GetInferenceConfidence(%q) = %f, want %f", tt.source, got, tt.wantConfidence)
			}
		})
	}
}

// TestImageRoleMapping tests the image role mapping documentation
func TestImageRoleMapping(t *testing.T) {
	mapping := ImageRoleMapping()

	// Verify some key mappings exist
	expectedMappings := map[string]string{
		"nginx":      "web",
		"mysql":      "db",
		"redis":      "cache",
		"rabbitmq":   "mq",
		"haproxy":    "lb",
		"prometheus": "monitoring",
	}

	for image, expectedRole := range expectedMappings {
		if role, ok := mapping[image]; !ok {
			t.Errorf("ImageRoleMapping missing %q", image)
		} else if role != expectedRole {
			t.Errorf("ImageRoleMapping[%q] = %q, want %q", image, role, expectedRole)
		}
	}

	// Verify mapping is not empty
	if len(mapping) == 0 {
		t.Error("ImageRoleMapping should not be empty")
	}
}

// TestPortRoleMapping tests the port role mapping documentation
func TestPortRoleMapping(t *testing.T) {
	mapping := PortRoleMapping()

	// Verify some key mappings exist
	expectedMappings := map[uint16]string{
		80:    "web",
		443:   "web",
		3306:  "db",
		5432:  "db",
		6379:  "cache",
		5672:  "mq",
		9200:  "search",
		9090:  "monitoring",
	}

	for port, expectedRole := range expectedMappings {
		if role, ok := mapping[port]; !ok {
			t.Errorf("PortRoleMapping missing port %d", port)
		} else if role != expectedRole {
			t.Errorf("PortRoleMapping[%d] = %q, want %q", port, role, expectedRole)
		}
	}

	// Verify mapping is not empty
	if len(mapping) == 0 {
		t.Error("PortRoleMapping should not be empty")
	}
}

// TestInferRoleFromImageEdgeCases tests edge cases for image inference
func TestInferRoleFromImageEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		wantRole string
	}{
		{"very long image name", "registry.example.com:5000/my-org/my-team/nginx-custom-build:v1.2.3-alpine", "web"},
		{"image with sha256", "nginx@sha256:abcd1234", "web"},
		{"image with multiple colons", "registry.io:5000/nginx:latest", "web"},
		{"image with underscores", "my_nginx_image:latest", "web"},
		{"image with dashes", "my-nginx-image:latest", "web"},
		{"whitespace around image", "  nginx:latest  ", "web"},  // Whitespace is trimmed
		{"partial match postgres", "my-postgres-app", "db"},
		{"partial match redis", "redis-cluster-node", "cache"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferRoleFromImage(tt.image)
			if got != tt.wantRole {
				t.Errorf("InferRoleFromImage(%q) = %q, want %q", tt.image, got, tt.wantRole)
			}
		})
	}
}

// TestInferRoleFromPortsPrecedence tests port precedence (web takes priority)
func TestInferRoleFromPortsPrecedence(t *testing.T) {
	// When multiple roles could match, the first match wins
	// (current implementation: web ports are checked first)

	tests := []struct {
		name     string
		ports    []uint16
		wantRole string
	}{
		{
			name:     "HTTP and MySQL - web wins",
			ports:    []uint16{80, 3306},
			wantRole: "web",
		},
		{
			name:     "HTTPS and Redis - web wins",
			ports:    []uint16{443, 6379},
			wantRole: "web",
		},
		{
			name:     "MySQL and MongoDB - db wins (first db match)",
			ports:    []uint16{3306, 27017},
			wantRole: "db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferRoleFromPorts(tt.ports)
			if got != tt.wantRole {
				t.Errorf("InferRoleFromPorts(%v) = %q, want %q", tt.ports, got, tt.wantRole)
			}
		})
	}
}

// BenchmarkInferRoleFromImage benchmarks image-based role inference
func BenchmarkInferRoleFromImage(b *testing.B) {
	images := []string{
		"nginx:latest",
		"mysql:8.0",
		"redis:alpine",
		"custom-app:v1.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InferRoleFromImage(images[i%len(images)])
	}
}

// BenchmarkInferRoleFromPorts benchmarks port-based role inference
func BenchmarkInferRoleFromPorts(b *testing.B) {
	portSets := [][]uint16{
		{80, 443},
		{3306},
		{6379},
		{12345, 54321},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InferRoleFromPorts(portSets[i%len(portSets)])
	}
}

// BenchmarkAutoTagWorkload benchmarks full auto-tagging
func BenchmarkAutoTagWorkload(b *testing.B) {
	tagger := NewAutoTagger()
	wl := &workload.Workload{
		ID:    "wl-bench",
		Name:  "nginx",
		Image: "nginx:latest",
		Ports: []uint16{80, 443},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tagger.AutoTagWorkload(wl)
	}
}
