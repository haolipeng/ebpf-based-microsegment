// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// NATCapabilities represents NAT detection capabilities of the system
type NATCapabilities struct {
	SupportBPFHelper bool   // Supports BPF conntrack helper (bpf_ct_lookup_*)
	SupportUserSync  bool   // Supports user-space conntrack sync
	KernelVersion    string // Kernel version string
	KernelMajor      int    // Kernel major version
	KernelMinor      int    // Kernel minor version
}

// DetectBPFCTSupport detects if kernel supports bpf_ct_lookup_* helpers
// These helpers are available since Linux 5.18
func DetectBPFCTSupport() (bool, error) {
	kver, err := getKernelVersion()
	if err != nil {
		return false, fmt.Errorf("failed to get kernel version: %w", err)
	}

	// Parse kernel version
	parts := strings.Split(kver, ".")
	if len(parts) < 2 {
		return false, fmt.Errorf("invalid kernel version format: %s", kver)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, fmt.Errorf("failed to parse major version: %w", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("failed to parse minor version: %w", err)
	}

	// BPF CT helpers are available since 5.18
	// Ref: https://github.com/torvalds/linux/commit/b4c2b9593a1c
	if major > 5 || (major == 5 && minor >= 18) {
		log.Infof("Kernel %s supports BPF conntrack helpers", kver)
		return true, nil
	}

	log.Warnf("Kernel %s does not support BPF conntrack helpers (requires >= 5.18)", kver)
	return false, nil
}

// GetNATCapabilities returns the NAT detection capabilities of the system
func GetNATCapabilities() *NATCapabilities {
	kver := getKernelVersionSafe()
	major, minor := parseKernelVersion(kver)

	caps := &NATCapabilities{
		SupportBPFHelper: false,
		SupportUserSync:  true, // User-space sync is always supported
		KernelVersion:    kver,
		KernelMajor:      major,
		KernelMinor:      minor,
	}

	// Detect BPF CT helper support
	supported, err := DetectBPFCTSupport()
	if err == nil && supported {
		caps.SupportBPFHelper = true
	}

	return caps
}

// String returns a human-readable string representation of capabilities
func (c *NATCapabilities) String() string {
	var features []string

	if c.SupportBPFHelper {
		features = append(features, "BPF_CT_Helper")
	}
	if c.SupportUserSync {
		features = append(features, "UserSync")
	}

	return fmt.Sprintf("NAT Capabilities [Kernel: %s, Features: %s]",
		c.KernelVersion, strings.Join(features, ", "))
}

// SupportsNAT returns true if the system supports any form of NAT detection
func (c *NATCapabilities) SupportsNAT() bool {
	return c.SupportBPFHelper || c.SupportUserSync
}

// GetPreferredMode returns the preferred NAT detection mode
func (c *NATCapabilities) GetPreferredMode() string {
	if c.SupportBPFHelper {
		return "bpf_helper"
	}
	if c.SupportUserSync {
		return "user_sync"
	}
	return "disabled"
}

// getKernelVersion reads the kernel version from /proc/version
func getKernelVersion() (string, error) {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/version: %w", err)
	}

	// /proc/version format: "Linux version 5.15.0-91-generic ..."
	version := string(data)
	parts := strings.Fields(version)
	if len(parts) < 3 {
		return "", fmt.Errorf("unexpected /proc/version format: %s", version)
	}

	// Extract version number (e.g., "5.15.0-91-generic")
	return parts[2], nil
}

// getKernelVersionSafe returns kernel version, or "unknown" on error
func getKernelVersionSafe() string {
	kver, err := getKernelVersion()
	if err != nil {
		log.Warnf("Failed to get kernel version: %v", err)
		return "unknown"
	}
	return kver
}

// parseKernelVersion parses major and minor version numbers
func parseKernelVersion(kver string) (major, minor int) {
	parts := strings.Split(kver, ".")
	if len(parts) < 2 {
		return 0, 0
	}

	major, _ = strconv.Atoi(parts[0])
	minor, _ = strconv.Atoi(parts[1])
	return major, minor
}
