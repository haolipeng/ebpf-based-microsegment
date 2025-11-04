// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package labels

import (
	"strings"
	"testing"
)

func TestValidateLabelKey(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		// Valid keys
		{"simple key", "role", false},
		{"key with hyphen", "app-name", false},
		{"key with underscore", "app_name", false},
		{"key with dot", "app.name", false},
		{"key with prefix", "kubernetes.io/name", false},
		{"key with complex prefix", "app.kubernetes.io/name", false},
		{"single char", "a", false},
		{"dimension label role", "role", false},
		{"dimension label app", "app", false},
		{"dimension label env", "env", false},
		{"dimension label loc", "loc", false},
		{"kubernetes style", "app.kubernetes.io/name", false},
		{"version label", "version-1.0", false},

		// Invalid keys
		{"empty key", "", true},
		{"key with space", "my label", true},
		{"key with special char @", "foo@bar", true},
		{"key with special char !", "my-label!", true},
		{"key starting with hyphen", "-invalid", true},
		{"key ending with hyphen", "invalid-", true},
		{"key starting with dot", ".invalid", true},
		{"key ending with dot", "invalid.", true},
		{"key with double slash", "foo//bar", true},
		{"key starting with slash", "/invalid", true},
		{"key ending with slash", "invalid/", true},
		{"only special chars", "---", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLabelKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLabelKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLabelKeyLength(t *testing.T) {
	validator := NewValidator()

	// Test maximum length (253 characters)
	validKey := strings.Repeat("a", 252) + "b" // 253 chars
	err := validator.ValidateLabelKey(validKey)
	if err != nil {
		t.Errorf("ValidateLabelKey() with 253 chars should be valid, got error: %v", err)
	}

	// Test exceeding maximum length
	invalidKey := strings.Repeat("a", 254)
	err = validator.ValidateLabelKey(invalidKey)
	if err == nil {
		t.Error("ValidateLabelKey() with 254 chars should be invalid")
	}
}

func TestValidateLabelKeyWithPrefix(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid prefix", "example.com/key", false},
		{"valid complex prefix", "app.kubernetes.io/name", false},
		{"prefix too long", strings.Repeat("a", 254) + "/key", true},
		{"name too long", "prefix/" + strings.Repeat("a", 64), true},
		{"valid max prefix length", strings.Repeat("a", 189) + "/" + strings.Repeat("b", 63), false}, // Total 253 chars (189 + / + 63)
		{"valid max name after prefix", "prefix/" + strings.Repeat("a", 56), false}, // Total 63 chars (prefix/ + 56)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLabelKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLabelKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLabelValue(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Valid values
		{"simple value", "web", false},
		{"value with hyphen", "web-server", false},
		{"value with underscore", "web_server", false},
		{"value with dot", "v1.0", false},
		{"complex value", "frontend-v2.1", false},
		{"empty value", "", false}, // Allowed by default
		{"single char", "a", false},
		{"numeric value", "123", false},
		{"mixed alphanumeric", "app1-v2.0", false},

		// Invalid values
		{"value with space", "web server", true},
		{"value with special char @", "web@server", true},
		{"value with special char !", "web!", true},
		{"value starting with hyphen", "-invalid", true},
		{"value ending with hyphen", "invalid-", true},
		{"value starting with dot", ".invalid", true},
		{"value ending with dot", "invalid.", true},
		{"only special chars", "---", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLabelValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLabelValue(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLabelValueLength(t *testing.T) {
	validator := NewValidator()

	// Test maximum length (63 characters)
	validValue := strings.Repeat("a", 62) + "b" // 63 chars
	err := validator.ValidateLabelValue(validValue)
	if err != nil {
		t.Errorf("ValidateLabelValue() with 63 chars should be valid, got error: %v", err)
	}

	// Test exceeding maximum length
	invalidValue := strings.Repeat("a", 64)
	err = validator.ValidateLabelValue(invalidValue)
	if err == nil {
		t.Error("ValidateLabelValue() with 64 chars should be invalid")
	}
}

func TestValidateLabelValueEmptyWithStrictConstraints(t *testing.T) {
	// Strict validator doesn't allow empty values
	strictValidator := NewValidatorWithConstraints(StrictConstraints())

	err := strictValidator.ValidateLabelValue("")
	if err == nil {
		t.Error("StrictValidator should reject empty values")
	}

	// Default validator allows empty values
	defaultValidator := NewValidator()
	err = defaultValidator.ValidateLabelValue("")
	if err != nil {
		t.Errorf("DefaultValidator should allow empty values, got error: %v", err)
	}
}

func TestValidateLabel(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{"valid label", "role", "web", false},
		{"valid with prefix", "app.kubernetes.io/name", "frontend", false},
		{"valid complex", "env", "prod-v2.1", false},
		{"invalid key", "my label", "value", true},
		{"invalid value", "key", "invalid value", true},
		{"both invalid", "my label", "invalid value", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLabel(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLabel(%q, %q) error = %v, wantErr %v", tt.key, tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLabels(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		labels  map[string]string
		wantErr bool
	}{
		{
			name: "all valid",
			labels: map[string]string{
				"role": "web",
				"app":  "frontend",
				"env":  "prod",
			},
			wantErr: false,
		},
		{
			name: "one invalid key",
			labels: map[string]string{
				"role":      "web",
				"my label":  "value",
				"env":       "prod",
			},
			wantErr: true,
		},
		{
			name: "one invalid value",
			labels: map[string]string{
				"role": "web",
				"app":  "invalid value",
				"env":  "prod",
			},
			wantErr: true,
		},
		{
			name:    "empty map",
			labels:  map[string]string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLabels(tt.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLabels() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLabelsAll(t *testing.T) {
	validator := NewValidator()

	labels := map[string]string{
		"role":      "web",
		"my label":  "value",       // Invalid key
		"app":       "invalid app", // Invalid value (space)
		"env":       "prod",
	}

	errors := validator.ValidateLabelsAll(labels)
	if len(errors) != 2 {
		t.Errorf("ValidateLabelsAll() should return 2 errors, got %d", len(errors))
	}
}

func TestReservedPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"system prefix", "system.internal", true},
		{"k8s prefix", "k8s.namespace", true},
		{"internal prefix", "internal.debug", true},
		{"ebpf prefix", "ebpf.map", true},
		{"no prefix", "role", false},
		{"user prefix", "myapp.custom", false},
		{"partial match", "systemd", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsReservedPrefix(tt.key)
			if result != tt.expected {
				t.Errorf("IsReservedPrefix(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestReservedPrefixEnforcement(t *testing.T) {
	// Default validator doesn't enforce reserved prefixes
	defaultValidator := NewValidator()
	err := defaultValidator.ValidateLabelKey("system.internal")
	if err != nil {
		t.Errorf("Default validator should allow reserved prefixes, got error: %v", err)
	}

	// Strict validator enforces reserved prefixes
	strictValidator := NewValidatorWithConstraints(StrictConstraints())
	err = strictValidator.ValidateLabelKey("system.internal")
	if err == nil {
		t.Error("Strict validator should reject reserved prefixes")
	}
}

func TestIsDimensionLabel(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"role dimension", "role", true},
		{"app dimension", "app", true},
		{"env dimension", "env", true},
		{"loc dimension", "loc", true},
		{"custom label", "version", false},
		{"custom label", "team", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDimensionLabel(tt.key)
			if result != tt.expected {
				t.Errorf("IsDimensionLabel(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestAllDimensions(t *testing.T) {
	dimensions := AllDimensions()

	if len(dimensions) != 4 {
		t.Errorf("AllDimensions() should return 4 dimensions, got %d", len(dimensions))
	}

	expectedDimensions := []LabelDimension{LabelRole, LabelApp, LabelEnv, LabelLocation}
	for i, expected := range expectedDimensions {
		if dimensions[i] != expected {
			t.Errorf("AllDimensions()[%d] = %v, want %v", i, dimensions[i], expected)
		}
	}
}

func TestSanitizeLabelKey(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name           string
		key            string
		expectedKey    string
		expectedChange bool
	}{
		{"valid key", "role", "role", false},
		{"key with space", "my label", "my-label", true},
		{"key with special chars", "foo@bar!", "foo-bar", true},
		{"leading hyphen", "-invalid", "invalid", true},
		{"trailing hyphen", "invalid-", "invalid", true},
		{"multiple consecutive hyphens", "my---label", "my---label", false},
		{"empty key", "", "", false},
		{"too long", strings.Repeat("a", 300), strings.Repeat("a", 253), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, changed := validator.SanitizeLabelKey(tt.key)
			if sanitized != tt.expectedKey {
				t.Errorf("SanitizeLabelKey(%q) = %q, want %q", tt.key, sanitized, tt.expectedKey)
			}
			if changed != tt.expectedChange {
				t.Errorf("SanitizeLabelKey(%q) changed = %v, want %v", tt.key, changed, tt.expectedChange)
			}
		})
	}
}

func TestSanitizeLabelValue(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name           string
		value          string
		expectedValue  string
		expectedChange bool
	}{
		{"valid value", "web", "web", false},
		{"value with space", "web server", "web-server", true},
		{"value with special chars", "v1.0@prod!", "v1.0-prod", true},
		{"leading hyphen", "-invalid", "invalid", true},
		{"trailing hyphen", "invalid-", "invalid", true},
		{"empty value", "", "", false},
		{"too long", strings.Repeat("a", 100), strings.Repeat("a", 63), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, changed := validator.SanitizeLabelValue(tt.value)
			if sanitized != tt.expectedValue {
				t.Errorf("SanitizeLabelValue(%q) = %q, want %q", tt.value, sanitized, tt.expectedValue)
			}
			if changed != tt.expectedChange {
				t.Errorf("SanitizeLabelValue(%q) changed = %v, want %v", tt.value, changed, tt.expectedChange)
			}
		})
	}
}

func TestSanitizeLabels(t *testing.T) {
	validator := NewValidator()

	labels := map[string]string{
		"role":       "web",
		"my label":   "value",
		"app@name":   "frontend server",
		"valid-key":  "valid-value",
	}

	sanitized, changed := validator.SanitizeLabels(labels)

	if !changed {
		t.Error("SanitizeLabels() should report changes")
	}

	// Check that valid labels are preserved
	if sanitized["role"] != "web" {
		t.Errorf("Valid label 'role' should be preserved")
	}

	if sanitized["valid-key"] != "valid-value" {
		t.Errorf("Valid label 'valid-key' should be preserved")
	}

	// Check that invalid labels are sanitized
	if _, exists := sanitized["my label"]; exists {
		t.Error("Invalid key 'my label' should be sanitized")
	}

	if sanitized["my-label"] != "value" {
		t.Errorf("Sanitized key should be 'my-label', got %q", sanitized["my-label"])
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Test package-level convenience functions
	err := ValidateLabelKey("role")
	if err != nil {
		t.Errorf("ValidateLabelKey() error = %v", err)
	}

	err = ValidateLabelValue("web")
	if err != nil {
		t.Errorf("ValidateLabelValue() error = %v", err)
	}

	err = ValidateLabel("role", "web")
	if err != nil {
		t.Errorf("ValidateLabel() error = %v", err)
	}

	labels := map[string]string{"role": "web", "env": "prod"}
	err = ValidateLabels(labels)
	if err != nil {
		t.Errorf("ValidateLabels() error = %v", err)
	}
}

func TestConstraints(t *testing.T) {
	t.Run("default constraints", func(t *testing.T) {
		c := DefaultConstraints()
		if c.MaxKeyLength != 253 {
			t.Errorf("DefaultConstraints().MaxKeyLength = %d, want 253", c.MaxKeyLength)
		}
		if c.MaxValueLength != 63 {
			t.Errorf("DefaultConstraints().MaxValueLength = %d, want 63", c.MaxValueLength)
		}
		if !c.AllowEmptyValue {
			t.Error("DefaultConstraints().AllowEmptyValue should be true")
		}
		if c.EnforceReservedPrefixes {
			t.Error("DefaultConstraints().EnforceReservedPrefixes should be false")
		}
	})

	t.Run("strict constraints", func(t *testing.T) {
		c := StrictConstraints()
		if c.MaxKeyLength != 253 {
			t.Errorf("StrictConstraints().MaxKeyLength = %d, want 253", c.MaxKeyLength)
		}
		if c.MaxValueLength != 63 {
			t.Errorf("StrictConstraints().MaxValueLength = %d, want 63", c.MaxValueLength)
		}
		if c.AllowEmptyValue {
			t.Error("StrictConstraints().AllowEmptyValue should be false")
		}
		if !c.EnforceReservedPrefixes {
			t.Error("StrictConstraints().EnforceReservedPrefixes should be true")
		}
	})
}

func TestLabelDimensionString(t *testing.T) {
	tests := []struct {
		dimension LabelDimension
		expected  string
	}{
		{LabelRole, "role"},
		{LabelApp, "app"},
		{LabelEnv, "env"},
		{LabelLocation, "loc"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dimension), func(t *testing.T) {
			if tt.dimension.String() != tt.expected {
				t.Errorf("%v.String() = %q, want %q", tt.dimension, tt.dimension.String(), tt.expected)
			}
		})
	}
}

func TestCommonValues(t *testing.T) {
	t.Run("common role values", func(t *testing.T) {
		if len(CommonRoleValues) == 0 {
			t.Error("CommonRoleValues should not be empty")
		}

		expectedRoles := []string{"web", "api", "db", "cache", "mq"}
		for _, role := range expectedRoles {
			found := false
			for _, common := range CommonRoleValues {
				if common == role {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("CommonRoleValues should contain %q", role)
			}
		}
	})

	t.Run("common env values", func(t *testing.T) {
		if len(CommonEnvValues) == 0 {
			t.Error("CommonEnvValues should not be empty")
		}

		expectedEnvs := []string{"prod", "staging", "dev", "test", "qa"}
		for _, env := range expectedEnvs {
			found := false
			for _, common := range CommonEnvValues {
				if common == env {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("CommonEnvValues should contain %q", env)
			}
		}
	})
}

// Benchmark tests
func BenchmarkValidateLabelKey(b *testing.B) {
	validator := NewValidator()
	key := "app.kubernetes.io/name"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateLabelKey(key)
	}
}

func BenchmarkValidateLabelValue(b *testing.B) {
	validator := NewValidator()
	value := "frontend-v2.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateLabelValue(value)
	}
}

func BenchmarkValidateLabel(b *testing.B) {
	validator := NewValidator()
	key := "role"
	value := "web"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateLabel(key, value)
	}
}

func BenchmarkValidateLabels(b *testing.B) {
	validator := NewValidator()
	labels := map[string]string{
		"role": "web",
		"app":  "frontend",
		"env":  "prod",
		"loc":  "us-west-2",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateLabels(labels)
	}
}
