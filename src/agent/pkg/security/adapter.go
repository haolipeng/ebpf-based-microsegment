package security

// AlertManagerAdapter adapts AlertManager to flow.SecurityAlertManager interface
// This adapter is needed to avoid circular dependencies between security and flow packages
type AlertManagerAdapter struct {
	alertManager *AlertManager
}

// NewAlertManagerAdapter creates a new adapter for AlertManager
func NewAlertManagerAdapter(am *AlertManager) *AlertManagerAdapter {
	return &AlertManagerAdapter{
		alertManager: am,
	}
}

// CheckProcessSuspicion adapts the call to AlertManager.CheckProcessSuspicion
// This method uses interface{} types to avoid circular dependencies with flow package
func (a *AlertManagerAdapter) CheckProcessSuspicion(procInfo interface{}, flowInfo interface{}) interface{} {
	// Type assert procInfo to the expected structure
	pi, ok := procInfo.(struct {
		PID         uint32
		Comm        string
		Path        string
		ContainerID string
		UID         uint32
	})
	if !ok {
		return nil
	}

	// Convert to security.ProcessInfo
	secProcInfo := ProcessInfo{
		PID:         pi.PID,
		Comm:        pi.Comm,
		Path:        pi.Path,
		ContainerID: pi.ContainerID,
		UID:         pi.UID,
	}

	// Type assert flowInfo to the expected structure (can be nil)
	var secFlowInfo *FlowInfo
	if flowInfo != nil {
		if fi, ok := flowInfo.(*struct {
			SourceIP   string
			SourcePort uint16
			DestIP     string
			DestPort   uint16
			Protocol   string
		}); ok {
			secFlowInfo = &FlowInfo{
				SourceIP:   fi.SourceIP,
				SourcePort: fi.SourcePort,
				DestIP:     fi.DestIP,
				DestPort:   fi.DestPort,
				Protocol:   fi.Protocol,
			}
		}
	}

	// Call the actual AlertManager method
	alerts := a.alertManager.CheckProcessSuspicion(secProcInfo, secFlowInfo)

	// Return as interface{}
	return alerts
}
