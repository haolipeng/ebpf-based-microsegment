// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package labels

import (
	"fmt"
	"regexp"
	"strings"
)

// Validator validates label keys and values according to configured constraints
type Validator struct {
	constraints LabelConstraints
}

// NewValidator creates a new label validator with default constraints
func NewValidator() *Validator {
	return &Validator{
		constraints: DefaultConstraints(),
	}
}

// NewValidatorWithConstraints creates a validator with custom constraints
func NewValidatorWithConstraints(constraints LabelConstraints) *Validator {
	return &Validator{
		constraints: constraints,
	}
}

// Regular expressions for label validation
var (
	// labelKeyRegex matches valid label keys
	// Valid characters: alphanumeric, '-', '_', '.'
	// Must start and end with alphanumeric
	// Format: [prefix/]name where prefix is optional
	labelKeyRegex = regexp.MustCompile(`^([a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?/)?[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$`)

	// labelValueRegex matches valid label values
	// Valid characters: alphanumeric, '-', '_', '.'
	// Must start and end with alphanumeric (unless empty)
	// Can be empty if allowed by constraints
	labelValueRegex = regexp.MustCompile(`^([a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?)?$`)

	// simpleKeyRegex for keys without prefix (used for single-segment keys)
	simpleKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$`)

	// singleCharKeyRegex allows single character keys
	singleCharKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9]$`)
)

// ValidateLabelKey validates a label key according to the configured constraints
func (v *Validator) ValidateLabelKey(key string) error {
	if key == "" {
		return fmt.Errorf("label key cannot be empty")
	}

	// Check length
	if len(key) > v.constraints.MaxKeyLength {
		return fmt.Errorf("label key exceeds maximum length: %d > %d", len(key), v.constraints.MaxKeyLength)
	}

	// Check for reserved prefix if enforcement is enabled
	if v.constraints.EnforceReservedPrefixes && IsReservedPrefix(key) {
		return fmt.Errorf("label key uses reserved prefix: %s", key)
	}

	// Validate format
	// Special case: single character keys are valid
	if len(key) == 1 {
		if !singleCharKeyRegex.MatchString(key) {
			return fmt.Errorf("invalid label key format: %s (must be alphanumeric)", key)
		}
		return nil
	}

	// Check if key contains a slash (prefix/name format)
	if strings.Contains(key, "/") {
		// Validate full key with optional prefix
		if !labelKeyRegex.MatchString(key) {
			return fmt.Errorf("invalid label key format: %s (must match [prefix/]name pattern)", key)
		}

		// Validate prefix length (before slash)
		parts := strings.SplitN(key, "/", 2)
		if len(parts[0]) > 253 {
			return fmt.Errorf("label key prefix exceeds maximum length: %d > 253", len(parts[0]))
		}
		if len(parts[1]) > 63 {
			return fmt.Errorf("label key name exceeds maximum length: %d > 63", len(parts[1]))
		}
	} else {
		// Simple key without prefix
		if !simpleKeyRegex.MatchString(key) {
			return fmt.Errorf("invalid label key format: %s (must be alphanumeric, '-', '_', '.' and start/end with alphanumeric)", key)
		}
	}

	return nil
}

// ValidateLabelValue validates a label value according to the configured constraints
func (v *Validator) ValidateLabelValue(value string) error {
	// Check if empty value is allowed
	if value == "" {
		if !v.constraints.AllowEmptyValue {
			return fmt.Errorf("label value cannot be empty")
		}
		return nil
	}

	// Check length
	if len(value) > v.constraints.MaxValueLength {
		return fmt.Errorf("label value exceeds maximum length: %d > %d", len(value), v.constraints.MaxValueLength)
	}

	// Validate format
	if !labelValueRegex.MatchString(value) {
		return fmt.Errorf("invalid label value format: %s (must be alphanumeric, '-', '_', '.' and start/end with alphanumeric)", value)
	}

	return nil
}

// ValidateLabel validates both key and value of a label
func (v *Validator) ValidateLabel(key, value string) error {
	if err := v.ValidateLabelKey(key); err != nil {
		return err
	}

	if err := v.ValidateLabelValue(value); err != nil {
		return err
	}

	return nil
}

// ValidateLabels validates a map of labels
// Returns the first validation error encountered
func (v *Validator) ValidateLabels(labels map[string]string) error {
	for key, value := range labels {
		if err := v.ValidateLabel(key, value); err != nil {
			return fmt.Errorf("invalid label %s=%s: %w", key, value, err)
		}
	}
	return nil
}

// ValidateLabelsAll validates all labels and returns all errors
func (v *Validator) ValidateLabelsAll(labels map[string]string) []error {
	var errors []error
	for key, value := range labels {
		if err := v.ValidateLabel(key, value); err != nil {
			errors = append(errors, fmt.Errorf("invalid label %s=%s: %w", key, value, err))
		}
	}
	return errors
}

// SanitizeLabelKey attempts to sanitize an invalid label key
// Returns the sanitized key and a boolean indicating if changes were made
func (v *Validator) SanitizeLabelKey(key string) (string, bool) {
	if key == "" {
		return key, false
	}

	original := key
	sanitized := key

	// Replace invalid characters with hyphens
	sanitized = regexp.MustCompile(`[^a-zA-Z0-9\-_./]`).ReplaceAllString(sanitized, "-")

	// Remove leading/trailing invalid characters
	sanitized = strings.Trim(sanitized, "-_.")

	// Truncate if too long
	if len(sanitized) > v.constraints.MaxKeyLength {
		sanitized = sanitized[:v.constraints.MaxKeyLength]
	}

	// Ensure it ends with alphanumeric
	sanitized = regexp.MustCompile(`[-_.]+$`).ReplaceAllString(sanitized, "")

	return sanitized, sanitized != original
}

// SanitizeLabelValue attempts to sanitize an invalid label value
// Returns the sanitized value and a boolean indicating if changes were made
func (v *Validator) SanitizeLabelValue(value string) (string, bool) {
	if value == "" {
		return value, false
	}

	original := value
	sanitized := value

	// Replace invalid characters with hyphens
	sanitized = regexp.MustCompile(`[^a-zA-Z0-9\-_.]`).ReplaceAllString(sanitized, "-")

	// Remove leading/trailing invalid characters
	sanitized = strings.Trim(sanitized, "-_.")

	// Truncate if too long
	if len(sanitized) > v.constraints.MaxValueLength {
		sanitized = sanitized[:v.constraints.MaxValueLength]
	}

	// Ensure it ends with alphanumeric
	sanitized = regexp.MustCompile(`[-_.]+$`).ReplaceAllString(sanitized, "")

	return sanitized, sanitized != original
}

// SanitizeLabels sanitizes all labels in a map
// Returns a new map with sanitized labels and a boolean indicating if any changes were made
func (v *Validator) SanitizeLabels(labels map[string]string) (map[string]string, bool) {
	sanitized := make(map[string]string)
	changed := false

	for key, value := range labels {
		sanitizedKey, keyChanged := v.SanitizeLabelKey(key)
		sanitizedValue, valueChanged := v.SanitizeLabelValue(value)

		if keyChanged || valueChanged {
			changed = true
		}

		// Skip if key becomes empty after sanitization
		if sanitizedKey != "" {
			sanitized[sanitizedKey] = sanitizedValue
		} else {
			changed = true // Skipping a key is a change
		}
	}

	return sanitized, changed
}

// Package-level convenience functions using default validator

var defaultValidator = NewValidator()

// ValidateLabelKey validates a label key using default constraints
func ValidateLabelKey(key string) error {
	return defaultValidator.ValidateLabelKey(key)
}

// ValidateLabelValue validates a label value using default constraints
func ValidateLabelValue(value string) error {
	return defaultValidator.ValidateLabelValue(value)
}

// ValidateLabel validates both key and value using default constraints
func ValidateLabel(key, value string) error {
	return defaultValidator.ValidateLabel(key, value)
}

// ValidateLabels validates a map of labels using default constraints
func ValidateLabels(labels map[string]string) error {
	return defaultValidator.ValidateLabels(labels)
}
