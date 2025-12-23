// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: multiple label sources (user, auto-tagged)
// output: merged label map with priority resolution
// pos: label merger with conflict resolution - if file updated, must sync with this header comment and pkg/labels/CLAUDE.md
package labels

import (
	"github.com/haolipeng/ebpf-based-microsegment/pkg/runtime"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/workload"
	log "github.com/sirupsen/logrus"
)

// LabelMerger merges labels from multiple sources with priority
type LabelMerger struct {
	runtimeDetector runtime.RuntimeDetector
	autoTagger      *AutoTagger
}

// NewLabelMerger creates a new label merger
func NewLabelMerger(detector runtime.RuntimeDetector, tagger *AutoTagger) *LabelMerger {
	return &LabelMerger{
		runtimeDetector: detector,
		autoTagger:      tagger,
	}
}

// GetEffectiveLabels merges labels from all sources with proper priority
// Priority (low to high):
//  1. System metadata (host, namespace)
//  2. Auto-inferred labels (from AutoTagger)
//  3. Container runtime labels (Kubernetes Pod labels)
//  4. User-defined labels (highest priority)
func (m *LabelMerger) GetEffectiveLabels(wl *workload.Workload) map[string]string {
	if wl == nil {
		return make(map[string]string)
	}

	effectiveLabels := make(map[string]string)

	// Layer 1: System metadata (lowest priority)
	if wl.HostID != "" {
		effectiveLabels["host"] = wl.HostID
	}
	if wl.Namespace != "" {
		effectiveLabels["namespace"] = wl.Namespace
	}

	// Layer 2: AutoTagger inferred labels (medium-low priority)
	// Only add if key doesn't exist
	if m.autoTagger != nil {
		inferredLabels := m.autoTagger.AutoTagWorkload(wl)
		for k, v := range inferredLabels {
			if _, exists := effectiveLabels[k]; !exists {
				effectiveLabels[k] = v
			}
		}
	}

	// Layer 3: Container runtime labels - Kubernetes Pod Labels (high priority)
	if m.runtimeDetector != nil && wl.ContainerID != "" {
		runtimeLabels, err := m.runtimeDetector.GetContainerLabels(wl.ContainerID)
		if err != nil {
			log.Debugf("Could not get runtime labels for container %s: %v", wl.ContainerID, err)
		} else {
			for k, v := range runtimeLabels {
				// Map Kubernetes standard labels to four-dimension model
				mappedKey := m.mapLabelKey(k)
				effectiveLabels[mappedKey] = v
			}
		}
	}

	// Layer 4: User-defined labels (highest priority)
	// Directly overwrite all previous labels
	for k, v := range wl.Labels {
		effectiveLabels[k] = v
	}

	log.Debugf("Merged labels for workload %s: %d labels total", wl.ID, len(effectiveLabels))
	return effectiveLabels
}

// mapLabelKey maps Kubernetes standard labels to the four-dimension model
// (role, app, env, loc)
func (m *LabelMerger) mapLabelKey(k8sKey string) string {
	// Kubernetes standard label mappings
	mappings := map[string]string{
		// Kubernetes recommended labels
		"app.kubernetes.io/name":      "app",
		"app.kubernetes.io/component": "role",
		"app.kubernetes.io/version":   "version",
		"app.kubernetes.io/instance":  "instance",
		"app.kubernetes.io/part-of":   "app",

		// Common labels
		"app":         "app",
		"application": "app",
		"role":        "role",
		"tier":        "role",
		"component":   "role",
		"environment": "env",
		"env":         "env",

		// Location labels
		"topology.kubernetes.io/zone":            "loc",
		"topology.kubernetes.io/region":          "region",
		"failure-domain.beta.kubernetes.io/zone": "loc", // deprecated but still used

		// Team/ownership
		"team": "team",
		"owner": "owner",
	}

	if mapped, exists := mappings[k8sKey]; exists {
		return mapped
	}

	// Return original key if no mapping exists
	return k8sKey
}

// MergeLabelsInPlace updates the workload's labels with effective labels
// This modifies the workload object in place
func (m *LabelMerger) MergeLabelsInPlace(wl *workload.Workload) {
	effectiveLabels := m.GetEffectiveLabels(wl)
	wl.Labels = effectiveLabels
}

// GetLabelSource returns the source of a label (for debugging/auditing)
func (m *LabelMerger) GetLabelSource(wl *workload.Workload, key string) string {
	// Check user-defined labels first (highest priority)
	if _, exists := wl.Labels[key]; exists {
		return "user"
	}

	// Check runtime labels
	if m.runtimeDetector != nil && wl.ContainerID != "" {
		runtimeLabels, err := m.runtimeDetector.GetContainerLabels(wl.ContainerID)
		if err == nil {
			for k := range runtimeLabels {
				if m.mapLabelKey(k) == key {
					return "k8s"
				}
			}
		}
	}

	// Check inferred labels
	if m.autoTagger != nil {
		inferredLabels := m.autoTagger.AutoTagWorkload(wl)
		if _, exists := inferredLabels[key]; exists {
			return "inferred"
		}
	}

	// Check system metadata
	if (key == "host" && wl.HostID != "") || (key == "namespace" && wl.Namespace != "") {
		return "system"
	}

	return "unknown"
}

// GetAllLabelSources returns a map of label keys to their sources
// This is useful for debugging and auditing label origins
func (m *LabelMerger) GetAllLabelSources(wl *workload.Workload) map[string]string {
	effectiveLabels := m.GetEffectiveLabels(wl)
	sources := make(map[string]string)

	for key := range effectiveLabels {
		sources[key] = m.GetLabelSource(wl, key)
	}

	return sources
}
