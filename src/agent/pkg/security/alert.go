package security

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AlertHandler is an interface for handling security alerts
type AlertHandler interface {
	// HandleAlert processes a security alert
	HandleAlert(alert *SecurityAlert) error
}

// AlertManager manages security alert generation and distribution
type AlertManager struct {
	validator *PathValidator

	// Alert handlers
	handlers []AlertHandler

	// Rate limiting
	alertCounts  map[string]int // Key: alert type + process path
	alertMutex   sync.RWMutex
	rateLimitMax int           // Maximum alerts per window
	rateWindow   time.Duration // Rate limit window

	// Context for cleanup
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	alertsGenerated uint64
	alertsSuppressed uint64
	metricsMutex    sync.RWMutex
}

// AlertManagerConfig holds configuration for the alert manager
type AlertManagerConfig struct {
	// RateLimitMax is the maximum number of alerts per window
	RateLimitMax int

	// RateWindow is the rate limiting window duration
	RateWindow time.Duration

	// EnableRateLimiting enables alert rate limiting
	EnableRateLimiting bool
}

// DefaultAlertManagerConfig returns default alert manager configuration
func DefaultAlertManagerConfig() AlertManagerConfig {
	return AlertManagerConfig{
		RateLimitMax:       10,
		RateWindow:         1 * time.Minute,
		EnableRateLimiting: true,
	}
}

// NewAlertManager creates a new alert manager
func NewAlertManager(validator *PathValidator, config AlertManagerConfig) *AlertManager {
	ctx, cancel := context.WithCancel(context.Background())

	am := &AlertManager{
		validator:    validator,
		handlers:     []AlertHandler{},
		alertCounts:  make(map[string]int),
		rateLimitMax: config.RateLimitMax,
		rateWindow:   config.RateWindow,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Start rate limit cleanup goroutine
	if config.EnableRateLimiting {
		am.wg.Add(1)
		go am.rateLimitCleanup()
	}

	return am
}

// AddHandler adds an alert handler
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.handlers = append(am.handlers, handler)
}

// Stop gracefully stops the alert manager
func (am *AlertManager) Stop() error {
	am.cancel()
	am.wg.Wait()
	return nil
}

// CheckProcessSuspicion checks if a process is suspicious and generates alerts
func (am *AlertManager) CheckProcessSuspicion(procInfo ProcessInfo, flowInfo *FlowInfo) []*SecurityAlert {
	alerts := []*SecurityAlert{}

	// Validate process path
	validationResult := am.validator.ValidatePath(procInfo.Path, procInfo.Comm)

	// Generate alerts based on validation result
	if validationResult.IsSuspicious {
		for _, reason := range validationResult.Reasons {
			alertType := am.mapReasonToAlertType(reason)
			level := am.determineAlertLevel(validationResult.Confidence)

			alert := am.createAlert(alertType, level, procInfo, flowInfo, reason)
			if alert != nil && !am.isRateLimited(alert) {
				alerts = append(alerts, alert)
			}
		}
	}

	// Check privilege escalation
	if am.validator.IsPrivilegeEscalation(procInfo.Path, procInfo.UID) {
		alert := am.createAlert(
			AlertTypePrivilegeEscalation,
			AlertLevelCritical,
			procInfo,
			flowInfo,
			fmt.Sprintf("Privileged process (UID %d) running from untrusted location: %s", procInfo.UID, procInfo.Path),
		)
		if alert != nil && !am.isRateLimited(alert) {
			alerts = append(alerts, alert)
		}
	}

	// Check anomalous connection
	if flowInfo != nil && am.validator.IsAnomalousConnection(procInfo, flowInfo) {
		alert := am.createAlert(
			AlertTypeAnomalousConnection,
			AlertLevelWarning,
			procInfo,
			flowInfo,
			fmt.Sprintf("Anomalous network connection: %s:%d -> %s:%d",
				flowInfo.SourceIP, flowInfo.SourcePort, flowInfo.DestIP, flowInfo.DestPort),
		)
		if alert != nil && !am.isRateLimited(alert) {
			alerts = append(alerts, alert)
		}
	}

	// Dispatch alerts to handlers
	for _, alert := range alerts {
		am.dispatchAlert(alert)
	}

	return alerts
}

// createAlert creates a new security alert
func (am *AlertManager) createAlert(alertType AlertType, level AlertLevel, procInfo ProcessInfo, flowInfo *FlowInfo, reason string) *SecurityAlert {
	alert := &SecurityAlert{
		AlertID:     uuid.New().String(),
		Level:       level,
		Type:        alertType,
		ProcessInfo: procInfo,
		FlowInfo:    flowInfo,
		Reason:      reason,
		Timestamp:   time.Now(),
		Metadata:    make(map[string]string),
	}

	// Add metadata
	if procInfo.ContainerID != "" {
		alert.Metadata["container_id"] = procInfo.ContainerID
	}

	return alert
}

// mapReasonToAlertType maps a validation reason to an alert type
func (am *AlertManager) mapReasonToAlertType(reason string) AlertType {
	switch reason {
	case "deleted executable":
		return AlertTypeDeletedExecutable
	case "hidden executable":
		return AlertTypeHiddenExecutable
	case "suspicious directory":
		return AlertTypeSuspiciousPath
	case "process name mismatch":
		return AlertTypeNameMismatch
	default:
		return AlertTypeSuspiciousPath
	}
}

// determineAlertLevel determines alert level based on confidence score
func (am *AlertManager) determineAlertLevel(confidence int) AlertLevel {
	if confidence >= 80 {
		return AlertLevelCritical
	} else if confidence >= 50 {
		return AlertLevelWarning
	}
	return AlertLevelInfo
}

// isRateLimited checks if an alert should be rate limited
func (am *AlertManager) isRateLimited(alert *SecurityAlert) bool {
	// Generate rate limit key
	key := fmt.Sprintf("%s:%s", alert.Type, alert.ProcessInfo.Path)

	am.alertMutex.Lock()
	defer am.alertMutex.Unlock()

	count := am.alertCounts[key]
	if count >= am.rateLimitMax {
		am.incrementSuppressed()
		return true
	}

	am.alertCounts[key]++
	return false
}

// rateLimitCleanup periodically cleans up rate limit counters
func (am *AlertManager) rateLimitCleanup() {
	defer am.wg.Done()

	ticker := time.NewTicker(am.rateWindow)
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.alertMutex.Lock()
			am.alertCounts = make(map[string]int)
			am.alertMutex.Unlock()
		}
	}
}

// dispatchAlert sends an alert to all registered handlers
func (am *AlertManager) dispatchAlert(alert *SecurityAlert) {
	am.incrementGenerated()

	for _, handler := range am.handlers {
		if err := handler.HandleAlert(alert); err != nil {
			log.Printf("[Security Alert] Error handling alert: %v", err)
		}
	}
}

// incrementGenerated increments the generated alerts counter
func (am *AlertManager) incrementGenerated() {
	am.metricsMutex.Lock()
	defer am.metricsMutex.Unlock()
	am.alertsGenerated++
}

// incrementSuppressed increments the suppressed alerts counter
func (am *AlertManager) incrementSuppressed() {
	am.metricsMutex.Lock()
	defer am.metricsMutex.Unlock()
	am.alertsSuppressed++
}

// GetMetrics returns alert manager metrics
func (am *AlertManager) GetMetrics() (generated, suppressed uint64) {
	am.metricsMutex.RLock()
	defer am.metricsMutex.RUnlock()
	return am.alertsGenerated, am.alertsSuppressed
}

// LogAlertHandler is a simple alert handler that logs alerts
type LogAlertHandler struct{}

// HandleAlert logs a security alert
func (h *LogAlertHandler) HandleAlert(alert *SecurityAlert) error {
	log.Printf("[Security Alert] [%s] [%s] PID=%d Path=%s Container=%s Reason=%s",
		alert.Level, alert.Type, alert.ProcessInfo.PID, alert.ProcessInfo.Path,
		alert.ProcessInfo.ContainerID, alert.Reason)
	return nil
}

// NewLogAlertHandler creates a new log alert handler
func NewLogAlertHandler() *LogAlertHandler {
	return &LogAlertHandler{}
}
