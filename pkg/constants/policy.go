// Package constants provides shared constants for the microsegmentation project.
package constants

// Policy actions
const (
	PolicyActionAllow = "allow"
	PolicyActionDeny  = "deny"
	PolicyActionLog   = "log"
)

// Policy directions
const (
	PolicyDirectionIngress = "ingress"
	PolicyDirectionEgress  = "egress"
	PolicyDirectionAny     = "any"
)

// Protocol names
const (
	ProtocolTCP    = "tcp"
	ProtocolUDP    = "udp"
	ProtocolICMP   = "icmp"
	ProtocolICMPv6 = "icmpv6"
	ProtocolSCTP   = "sctp"
	ProtocolGRE    = "gre"
	ProtocolESP    = "esp"
	ProtocolAH     = "ah"
	ProtocolIPIP   = "ipip"
	ProtocolIPv6   = "ipv6"
)

// Protocol numbers (IANA assigned)
const (
	ProtocolNumberICMP   = 1
	ProtocolNumberIPIP   = 4
	ProtocolNumberTCP    = 6
	ProtocolNumberUDP    = 17
	ProtocolNumberIPv6   = 41
	ProtocolNumberGRE    = 47
	ProtocolNumberESP    = 50
	ProtocolNumberAH     = 51
	ProtocolNumberICMPv6 = 58
	ProtocolNumberSCTP   = 132
)

// TCP State Machine States (sync with eBPF)
const (
	TCPStateClosed      = 0
	TCPStateListen      = 1
	TCPStateSynSent     = 2
	TCPStateSynRecv     = 3
	TCPStateEstablished = 4
	TCPStateFinWait1    = 5
	TCPStateFinWait2    = 6
	TCPStateCloseWait   = 7
	TCPStateClosing     = 8
	TCPStateLastAck     = 9
	TCPStateTimeWait    = 10
)

// Session states
const (
	SessionStateNew     = 0
	SessionStateActive  = 1
	SessionStateClosing = 2
	SessionStateClosed  = 3
)

// Flow event types
const (
	FlowEventNew    = 0
	FlowEventUpdate = 1
	FlowEventClosed = 2
)

// Default values
const (
	// DefaultPolicyPriority is the default priority for policies
	DefaultPolicyPriority = 1000

	// WildcardPort represents "any port" in policies
	WildcardPort = 0

	// WildcardProtocol represents "any protocol" in policies
	WildcardProtocol = 0
)
