package security

import (
	"path/filepath"
	"strings"
)

// PathValidator validates process paths and classifies them by security category
type PathValidator struct {
	// SystemDirs are trusted system binary directories
	SystemDirs []string

	// SuspiciousDirs are directories commonly used by malware
	SuspiciousDirs []string

	// UserDirs are user-owned directories
	UserDirs []string

	// MaliciousNames is a blacklist of known malicious process names
	MaliciousNames map[string]bool
}

// DefaultPathValidator returns a validator with default configuration
func DefaultPathValidator() *PathValidator {
	return &PathValidator{
		SystemDirs: []string{
			"/usr/bin",
			"/usr/sbin",
			"/bin",
			"/sbin",
			"/usr/local/bin",
			"/usr/local/sbin",
			"/lib/systemd",
			"/usr/lib/systemd",
		},
		SuspiciousDirs: []string{
			"/tmp",
			"/var/tmp",
			"/dev/shm",
			"/var/run",
			"/run/shm",
		},
		UserDirs: []string{
			"/home",
			"/root",
		},
		MaliciousNames: map[string]bool{
			"nc":         true, // Netcat in suspicious location
			"ncat":       true,
			"bash":       true, // Shell in suspicious location
			"sh":         true,
			"dash":       true,
			"python":     true, // Interpreter in suspicious location
			"perl":       true,
			"ruby":       true,
			"wget":       true, // Download tools in suspicious location
			"curl":       true,
			"cryptominer": true,
			"xmrig":      true, // Known cryptominers
		},
	}
}

// ValidatePath validates a process path and returns classification result
func (v *PathValidator) ValidatePath(path string, comm string) *PathValidationResult {
	result := &PathValidationResult{
		Category:     PathCategoryUnknown,
		IsSuspicious: false,
		Reasons:      []string{},
		Confidence:   0,
	}

	// Check if path is empty
	if path == "" {
		result.Reasons = append(result.Reasons, "empty path")
		result.IsSuspicious = true
		result.Confidence = 50
		return result
	}

	// Check for deleted executable
	if strings.HasSuffix(path, " (deleted)") {
		result.Reasons = append(result.Reasons, "deleted executable")
		result.IsSuspicious = true
		result.Category = PathCategorySuspicious
		result.Confidence = 90
		return result
	}

	// Check for hidden executable
	basename := filepath.Base(path)
	if strings.HasPrefix(basename, ".") {
		result.Reasons = append(result.Reasons, "hidden executable")
		result.IsSuspicious = true
		result.Confidence += 30
	}

	// Classify by directory
	category := v.classifyByDirectory(path)
	result.Category = category

	// Check if in suspicious directory
	if category == PathCategorySuspicious {
		result.IsSuspicious = true
		result.Reasons = append(result.Reasons, "suspicious directory")
		result.Confidence += 70

		// Extra suspicious if known malicious name in suspicious location
		if v.MaliciousNames[comm] {
			result.Reasons = append(result.Reasons, "known malicious process name")
			result.Confidence = 100
		}
	}

	// Check process name mismatch
	if comm != "" && basename != "" && comm != basename {
		// Allow common patterns (e.g., "python3" vs "python3.10")
		if !strings.HasPrefix(basename, comm) && !strings.HasPrefix(comm, basename) {
			result.Reasons = append(result.Reasons, "process name mismatch")
			result.IsSuspicious = true
			result.Confidence += 40
		}
	}

	// Cap confidence at 100
	if result.Confidence > 100 {
		result.Confidence = 100
	}

	return result
}

// classifyByDirectory classifies a path by its directory location
func (v *PathValidator) classifyByDirectory(path string) PathCategory {
	// Check system directories
	for _, dir := range v.SystemDirs {
		if strings.HasPrefix(path, dir+"/") || path == dir {
			return PathCategorySystem
		}
	}

	// Check suspicious directories
	for _, dir := range v.SuspiciousDirs {
		if strings.HasPrefix(path, dir+"/") || path == dir {
			return PathCategorySuspicious
		}
	}

	// Check user directories
	for _, dir := range v.UserDirs {
		if strings.HasPrefix(path, dir+"/") || path == dir {
			return PathCategoryUser
		}
	}

	return PathCategoryUnknown
}

// IsPrivilegeEscalation checks if a process represents privilege escalation
// (UID 0 process running from user or suspicious directory)
func (v *PathValidator) IsPrivilegeEscalation(path string, uid uint32) bool {
	if uid != 0 {
		return false
	}

	category := v.classifyByDirectory(path)
	return category == PathCategoryUser || category == PathCategorySuspicious
}

// IsAnomalousConnection checks if a network connection is anomalous for a process
func (v *PathValidator) IsAnomalousConnection(procInfo ProcessInfo, flowInfo *FlowInfo) bool {
	if flowInfo == nil {
		return false
	}

	// System processes should not connect to unusual ports
	category := v.classifyByDirectory(procInfo.Path)
	if category == PathCategorySystem {
		// Check for connections to uncommon ports
		if flowInfo.DestPort > 10000 && flowInfo.DestPort != 443 && flowInfo.DestPort != 8080 {
			return true
		}
	}

	// Suspicious directory processes making any connection is anomalous
	if category == PathCategorySuspicious {
		return true
	}

	return false
}

// GetSuspiciousScore returns a suspiciousness score (0-100) for a process
func (v *PathValidator) GetSuspiciousScore(procInfo ProcessInfo, flowInfo *FlowInfo) int {
	score := 0

	// Validate path
	validationResult := v.ValidatePath(procInfo.Path, procInfo.Comm)
	score += validationResult.Confidence

	// Check privilege escalation
	if v.IsPrivilegeEscalation(procInfo.Path, procInfo.UID) {
		score += 50
	}

	// Check anomalous connection
	if v.IsAnomalousConnection(procInfo, flowInfo) {
		score += 30
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}
