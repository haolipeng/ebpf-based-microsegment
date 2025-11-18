// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"strings"
	"testing"
)

// TestDetectBPFCTSupport tests BPF conntrack helper detection
func TestDetectBPFCTSupport(t *testing.T) {
	// Note: This test may fail on kernels < 5.18
	// The test should gracefully handle both supported and unsupported kernels
	supported, err := DetectBPFCTSupport()

	// Either we get a clear result (true/false) or an error
	if err != nil {
		t.Logf("BPF CT support detection failed (expected on old kernels): %v", err)
		if supported {
			t.Error("If error is returned, supported should be false")
		}
		return
	}

	t.Logf("BPF CT support: %v", supported)
}

// TestGetNATCapabilities tests NAT capabilities detection
func TestGetNATCapabilities(t *testing.T) {
	caps := GetNATCapabilities()

	if caps == nil {
		t.Fatal("GetNATCapabilities returned nil")
	}

	// User-space sync should always be supported
	if !caps.SupportUserSync {
		t.Error("User-space sync should always be supported")
	}

	// Kernel version should be detected
	if caps.KernelVersion == "" || caps.KernelVersion == "unknown" {
		t.Error("Kernel version should be detected")
	}

	// Kernel version should be parseable
	if caps.KernelMajor == 0 && caps.KernelMinor == 0 {
		t.Error("Kernel version should be parsed (major and minor > 0)")
	}

	t.Logf("NAT Capabilities: %s", caps.String())
	t.Logf("  Kernel: %s (v%d.%d)", caps.KernelVersion, caps.KernelMajor, caps.KernelMinor)
	t.Logf("  BPF Helper: %v", caps.SupportBPFHelper)
	t.Logf("  User Sync: %v", caps.SupportUserSync)
}

// TestNATCapabilitiesString tests the String() method
func TestNATCapabilitiesString(t *testing.T) {
	caps := &NATCapabilities{
		SupportBPFHelper: true,
		SupportUserSync:  true,
		KernelVersion:    "5.18.0",
		KernelMajor:      5,
		KernelMinor:      18,
	}

	str := caps.String()
	if !strings.Contains(str, "5.18.0") {
		t.Errorf("String() should contain kernel version, got: %s", str)
	}
	if !strings.Contains(str, "BPF_CT_Helper") {
		t.Errorf("String() should contain BPF_CT_Helper when supported, got: %s", str)
	}
	if !strings.Contains(str, "UserSync") {
		t.Errorf("String() should contain UserSync when supported, got: %s", str)
	}

	t.Logf("Capabilities string: %s", str)
}

// TestNATCapabilitiesSupportsNAT tests the SupportsNAT() method
func TestNATCapabilitiesSupportsNAT(t *testing.T) {
	tests := []struct {
		name             string
		caps             *NATCapabilities
		expectedSupports bool
	}{
		{
			name: "BPF helper only",
			caps: &NATCapabilities{
				SupportBPFHelper: true,
				SupportUserSync:  false,
			},
			expectedSupports: true,
		},
		{
			name: "User sync only",
			caps: &NATCapabilities{
				SupportBPFHelper: false,
				SupportUserSync:  true,
			},
			expectedSupports: true,
		},
		{
			name: "Both supported",
			caps: &NATCapabilities{
				SupportBPFHelper: true,
				SupportUserSync:  true,
			},
			expectedSupports: true,
		},
		{
			name: "Neither supported (should not happen in reality)",
			caps: &NATCapabilities{
				SupportBPFHelper: false,
				SupportUserSync:  false,
			},
			expectedSupports: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.caps.SupportsNAT()
			if result != tt.expectedSupports {
				t.Errorf("SupportsNAT() = %v, want %v", result, tt.expectedSupports)
			}
		})
	}
}

// TestNATCapabilitiesGetPreferredMode tests the GetPreferredMode() method
func TestNATCapabilitiesGetPreferredMode(t *testing.T) {
	tests := []struct {
		name         string
		caps         *NATCapabilities
		expectedMode string
	}{
		{
			name: "BPF helper preferred",
			caps: &NATCapabilities{
				SupportBPFHelper: true,
				SupportUserSync:  true,
			},
			expectedMode: "bpf_helper",
		},
		{
			name: "User sync fallback",
			caps: &NATCapabilities{
				SupportBPFHelper: false,
				SupportUserSync:  true,
			},
			expectedMode: "user_sync",
		},
		{
			name: "Disabled (neither supported)",
			caps: &NATCapabilities{
				SupportBPFHelper: false,
				SupportUserSync:  false,
			},
			expectedMode: "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.caps.GetPreferredMode()
			if result != tt.expectedMode {
				t.Errorf("GetPreferredMode() = %v, want %v", result, tt.expectedMode)
			}
		})
	}
}

// TestGetKernelVersion tests kernel version detection
func TestGetKernelVersion(t *testing.T) {
	version, err := getKernelVersion()
	if err != nil {
		t.Fatalf("Failed to get kernel version: %v", err)
	}

	if version == "" {
		t.Error("Kernel version should not be empty")
	}

	// Version should contain at least one dot
	if !strings.Contains(version, ".") {
		t.Errorf("Invalid kernel version format: %s", version)
	}

	t.Logf("Detected kernel version: %s", version)
}

// TestParseKernelVersion tests kernel version parsing
func TestParseKernelVersion(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		expectedMajor int
		expectedMinor int
	}{
		{
			name:          "standard version",
			version:       "5.15.0-91-generic",
			expectedMajor: 5,
			expectedMinor: 15,
		},
		{
			name:          "simple version",
			version:       "6.4.0",
			expectedMajor: 6,
			expectedMinor: 4,
		},
		{
			name:          "invalid version (no dots)",
			version:       "invalid",
			expectedMajor: 0,
			expectedMinor: 0,
		},
		{
			name:          "invalid version (empty)",
			version:       "",
			expectedMajor: 0,
			expectedMinor: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor := parseKernelVersion(tt.version)
			if major != tt.expectedMajor || minor != tt.expectedMinor {
				t.Errorf("parseKernelVersion(%s) = (%d, %d), want (%d, %d)",
					tt.version, major, minor, tt.expectedMajor, tt.expectedMinor)
			}
		})
	}
}

// BenchmarkGetNATCapabilities benchmarks NAT capabilities detection
func BenchmarkGetNATCapabilities(b *testing.B) {
	for i := 0; i < b.N; i++ {
		caps := GetNATCapabilities()
		if caps == nil {
			b.Fatal("GetNATCapabilities returned nil")
		}
	}
}

// BenchmarkDetectBPFCTSupport benchmarks BPF CT support detection
func BenchmarkDetectBPFCTSupport(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = DetectBPFCTSupport()
	}
}
