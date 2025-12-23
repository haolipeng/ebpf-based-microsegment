
// input: session timeout config, session_map
// output: session timeout operations, cleanup events
// pos: session timeout manager - if file updated, must sync with this header comment and pkg/session/CLAUDE.md
package session

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
)

// TimeoutManager manages session timeout and cleanup
type TimeoutManager struct {
	sessionMap *ebpf.Map
	config     SessionTimeoutConfig

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Statistics
	stats      SessionTimeoutStats
	statsMutex sync.RWMutex
}

// NewTimeoutManager creates a new session timeout manager
func NewTimeoutManager(sessionMap *ebpf.Map, config SessionTimeoutConfig) *TimeoutManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &TimeoutManager{
		sessionMap: sessionMap,
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
		stats:      SessionTimeoutStats{},
	}
}

// Start begins the timeout scanning loop
func (tm *TimeoutManager) Start() error {
	log.Info("[Session Timeout] Starting session timeout manager...")

	tm.wg.Add(1)
	go tm.scanLoop()

	log.Infof("[Session Timeout] Timeout manager started (scan interval: %v)", tm.config.ScanInterval)
	return nil
}

// Stop gracefully stops the timeout manager
func (tm *TimeoutManager) Stop() error {
	log.Info("[Session Timeout] Stopping timeout manager...")
	tm.cancel()
	tm.wg.Wait()
	log.Info("[Session Timeout] Timeout manager stopped")
	return nil
}

// scanLoop periodically scans the session map for timeouts
func (tm *TimeoutManager) scanLoop() {
	defer tm.wg.Done()

	// Initial delay before first scan
	time.Sleep(5 * time.Second)

	ticker := time.NewTicker(tm.config.ScanInterval)
	defer ticker.Stop()

	log.Infof("[Session Timeout] Scan loop started (TCP idle: %v, TCP close: %v, UDP idle: %v, max: %v)",
		tm.config.TCPIdleTimeout, tm.config.TCPCloseTimeout,
		tm.config.UDPIdleTimeout, tm.config.MaxSessionDuration)

	for {
		select {
		case <-tm.ctx.Done():
			log.Info("[Session Timeout] Scan loop stopped")
			return

		case <-ticker.C:
			//when trigger timer,do scan once.
			if err := tm.runScan(); err != nil {
				log.Errorf("[Session Timeout] Scan failed: %v", err)
				tm.statsMutex.Lock()
				tm.stats.ScanErrors++
				tm.statsMutex.Unlock()
			}
		}
	}
}

// timedOutSession stores information about a timed-out session
type timedOutSession struct {
	key    FlowKey
	value  SessionValue
	reason string
}

// runScan executes a single timeout scan
func (tm *TimeoutManager) runScan() error {
	startTime := time.Now()
	nowNS := uint64(startTime.UnixNano())

	log.Debug("[Session Timeout] Starting timeout scan...")

	// Collect timed-out sessions with their details
	var timedOutSessions []timedOutSession
	var tcpIdleCount, tcpCloseCount, udpIdleCount, maxDurationCount uint64
	sessionsScanned := uint64(0)

	var key FlowKey
	var value SessionValue

	// Iterate through all sessions
	iter := tm.sessionMap.Iterate()
	for iter.Next(&key, &value) {
		sessionsScanned++

		// Calculate elapsed times
		elapsedSinceLast := time.Duration(nowNS - value.LastSeenTS)
		elapsedSinceCreate := time.Duration(nowNS - value.CreatedTS)

		// Check timeout conditions
		timedOut, reason := tm.shouldTimeout(&key, &value, elapsedSinceLast, elapsedSinceCreate)
		if timedOut {
			timedOutSessions = append(timedOutSessions, timedOutSession{
				key:    key,
				value:  value,
				reason: reason,
			})

			// Update counters
			switch reason {
			case "tcp_idle":
				tcpIdleCount++
			case "tcp_close":
				tcpCloseCount++
			case "udp_idle":
				udpIdleCount++
			case "max_duration":
				maxDurationCount++
			}
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("iteration error: %w", err)
	}

	// Delete timed-out sessions and log detailed timeout events
	deletedCount := 0
	for _, session := range timedOutSessions {
		if err := tm.sessionMap.Delete(&session.key); err != nil {
			log.Debugf("[Session Timeout] Failed to delete session: %v", err)
		} else {
			deletedCount++

			// Log detailed timeout event for each session
			srcIP := ipv6ToNetIP(session.key.SrcIP, session.key.IPVersion)
			dstIP := ipv6ToNetIP(session.key.DstIP, session.key.IPVersion)
			protocol := protocolToString(session.key.Protocol)

			totalPackets := session.value.PacketsToServer + session.value.PacketsToClient
			totalBytes := session.value.BytesToServer + session.value.BytesToClient

			log.Infof("[FLOW TIMEOUT] %s:%d -> %s:%d proto=%s reason=%s packets=%d bytes=%d",
				srcIP, session.key.SrcPort,
				dstIP, session.key.DstPort,
				protocol, session.reason,
				totalPackets, totalBytes)
		}
	}

	duration := time.Since(startTime)

	// Update statistics
	tm.statsMutex.Lock()
	tm.stats.TotalScans++
	tm.stats.TotalSessionsScanned += sessionsScanned
	tm.stats.TotalTimedOut += uint64(deletedCount)
	tm.stats.TCPIdleTimeouts += tcpIdleCount
	tm.stats.TCPCloseTimeouts += tcpCloseCount
	tm.stats.UDPIdleTimeouts += udpIdleCount
	tm.stats.MaxDurationTimeouts += maxDurationCount
	tm.stats.LastScanTime = time.Now()
	tm.stats.LastScanDuration = duration
	tm.statsMutex.Unlock()

	// add log of Scan Session Timeout result
	if deletedCount > 0 {
		log.Infof("[Session Timeout] Scan completed: scanned %d sessions, deleted %d (TCP idle: %d, TCP close: %d, UDP idle: %d, max: %d, duration: %v)",
			sessionsScanned, deletedCount, tcpIdleCount, tcpCloseCount, udpIdleCount, maxDurationCount, duration)
	} else {
		log.Debugf("[Session Timeout] Scan completed: scanned %d sessions, no timeouts (duration: %v)",
			sessionsScanned, duration)
	}

	return nil
}

// shouldTimeout checks if a session should be timed out
// Returns (should_timeout, reason)
func (tm *TimeoutManager) shouldTimeout(key *FlowKey, value *SessionValue, idleTime, createTime time.Duration) (bool, string) {
	// TCP sessions in closing/closed state: check close timeout
	if key.Protocol == ProtocolTCP && isTCPClosing(value.TCPState) {
		if createTime > tm.config.TCPCloseTimeout {
			return true, "tcp_close"
		}
	}

	// Check max session duration (applies to all protocols)
	if tm.config.MaxSessionDuration > 0 && createTime > tm.config.MaxSessionDuration {
		return true, "max_duration"
	}

	// TCP sessions: check idle timeout
	if key.Protocol == ProtocolTCP {
		if idleTime > tm.config.TCPIdleTimeout {
			return true, "tcp_idle"
		}
	}

	// UDP sessions: check idle timeout
	if key.Protocol == ProtocolUDP {
		if idleTime > tm.config.UDPIdleTimeout {
			return true, "udp_idle"
		}
	}

	return false, ""
}

// GetStats returns current timeout statistics
func (tm *TimeoutManager) GetStats() SessionTimeoutStats {
	tm.statsMutex.RLock()
	defer tm.statsMutex.RUnlock()
	return tm.stats
}

// ipv6ToNetIP converts [4]uint32 IPv6 address to net.IP
// Handles both native IPv6 and IPv4-mapped IPv6 addresses
func ipv6ToNetIP(ipv6 [4]uint32, ipVersion uint8) net.IP {
	// Check if this is IPv4-mapped IPv6 (::ffff:a.b.c.d)
	// IPv4-mapped: [0, 0, 0xffff0000, ipv4_addr] in network byte order
	if ipVersion == 4 || (ipv6[0] == 0 && ipv6[1] == 0 && ipv6[2] == 0x0000ffff) {
		// Extract IPv4 address from last 32 bits (little-endian)
		ip := ipv6[3]
		return net.IPv4(byte(ip), byte(ip>>8), byte(ip>>16), byte(ip>>24))
	}

	// Native IPv6 address (convert 4 x uint32 to 16 bytes)
	ipv6Bytes := make(net.IP, 16)
	for i := 0; i < 4; i++ {
		// Little-endian conversion
		ipv6Bytes[i*4] = byte(ipv6[i])
		ipv6Bytes[i*4+1] = byte(ipv6[i] >> 8)
		ipv6Bytes[i*4+2] = byte(ipv6[i] >> 16)
		ipv6Bytes[i*4+3] = byte(ipv6[i] >> 24)
	}
	return ipv6Bytes
}

// protocolToString converts protocol number to string
func protocolToString(protocol uint8) string {
	switch protocol {
	case ProtocolTCP:
		return "TCP"
	case ProtocolUDP:
		return "UDP"
	case 1: // ICMP
		return "ICMP"
	default:
		return fmt.Sprintf("PROTO_%d", protocol)
	}
}
