// Package netutil provides network validation functions.
package netutil

import (
	"fmt"
	"net"
	"strings"
)

// ValidateCIDR validates CIDR notation with comprehensive checks.
//
// Checks performed:
//   - CIDR format is valid
//   - IP version matches mask length (IPv4: /0-32, IPv6: /0-128)
//   - Network address has no host bits set (e.g., 192.168.1.1/24 is invalid, use 192.168.1.0/24)
//
// Returns the normalized IP and network, or error if validation fails.
func ValidateCIDR(cidr string) (net.IP, *net.IPNet, error) {
	// Parse CIDR
	ip, ipnet, err := ParseCIDR(cidr)
	if err != nil {
		return nil, nil, err
	}

	// Validate mask length
	ones, bits := ipnet.Mask.Size()
	if IsIPv4(ip) {
		// IPv4: mask must be 0-32
		if bits != 32 {
			return nil, nil, fmt.Errorf("invalid IPv4 mask: /%d (bits=%d)", ones, bits)
		}
		if ones < 0 || ones > 32 {
			return nil, nil, fmt.Errorf("IPv4 mask out of range: /%d (must be 0-32)", ones)
		}
	} else {
		// IPv6: mask must be 0-128
		if bits != 128 {
			return nil, nil, fmt.Errorf("invalid IPv6 mask: /%d (bits=%d)", ones, bits)
		}
		if ones < 0 || ones > 128 {
			return nil, nil, fmt.Errorf("IPv6 mask out of range: /%d (must be 0-128)", ones)
		}
	}

	// Verify network address (no host bits set)
	// This prevents mistakes like 192.168.1.1/24 when user meant 192.168.1.0/24
	if !ip.Equal(ipnet.IP) {
		ones, _ := ipnet.Mask.Size()
		return nil, nil, fmt.Errorf("CIDR %s has host bits set, use %s/%d",
			cidr, ipnet.IP.String(), ones)
	}

	return ip, ipnet, nil
}

// ValidateIP validates an IP address string.
// Returns the parsed IP or error if invalid.
func ValidateIP(ipStr string) (net.IP, error) {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return nil, fmt.Errorf("empty IP address")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	return ip, nil
}

// ValidatePort validates a port number (0-65535).
// Port 0 is considered valid (means "any port" in policy context).
func ValidatePort(port uint16) error {
	// All uint16 values are valid ports (0-65535)
	return nil
}

// ValidatePortInt validates a port number as int.
func ValidatePortInt(port int) error {
	if port < 0 || port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 0-65535)", port)
	}
	return nil
}

// ValidateProtocol validates protocol string (tcp, udp, icmp, icmpv6, or empty for "any").
func ValidateProtocol(proto string) error {
	if proto == "" {
		return nil // Empty means "any protocol"
	}

	proto = strings.ToLower(strings.TrimSpace(proto))

	validProtos := map[string]bool{
		"tcp":     true,
		"udp":     true,
		"icmp":    true,
		"icmpv6":  true,
		"sctp":    true,
		"gre":     true,
		"esp":     true,
		"ah":      true,
		"ipip":    true,
		"ipv6":    true,
	}

	if !validProtos[proto] {
		return fmt.Errorf("invalid protocol: %s (supported: tcp, udp, icmp, icmpv6, sctp, gre, esp, ah, ipip, ipv6)", proto)
	}

	return nil
}

// ProtocolStringToNumber converts protocol string to protocol number.
// Returns 0 for unknown protocols or empty string (any).
func ProtocolStringToNumber(proto string) uint8 {
	proto = strings.ToLower(strings.TrimSpace(proto))

	protoMap := map[string]uint8{
		"tcp":     6,
		"udp":     17,
		"icmp":    1,
		"icmpv6":  58,
		"sctp":    132,
		"gre":     47,
		"esp":     50,
		"ah":      51,
		"ipip":    4,
		"ipv6":    41,
	}

	if num, ok := protoMap[proto]; ok {
		return num
	}

	return 0 // Unknown or "any"
}

// ProtocolNumberToString converts protocol number to string.
func ProtocolNumberToString(proto uint8) string {
	protoMap := map[uint8]string{
		6:   "tcp",
		17:  "udp",
		1:   "icmp",
		58:  "icmpv6",
		132: "sctp",
		47:  "gre",
		50:  "esp",
		51:  "ah",
		4:   "ipip",
		41:  "ipv6",
	}

	if name, ok := protoMap[proto]; ok {
		return name
	}

	return fmt.Sprintf("proto-%d", proto)
}

// ValidateMACAddress validates a MAC address string.
// Accepts formats: "00:11:22:33:44:55", "00-11-22-33-44-55", "001122334455"
func ValidateMACAddress(mac string) (net.HardwareAddr, error) {
	mac = strings.TrimSpace(mac)
	if mac == "" {
		return nil, fmt.Errorf("empty MAC address")
	}

	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC address %s: %w", mac, err)
	}

	return hwAddr, nil
}

// IsPrivateIP checks if an IP address is in private ranges (RFC 1918 for IPv4).
func IsPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		// IPv4 private ranges:
		// 10.0.0.0/8
		// 172.16.0.0/12
		// 192.168.0.0/16
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}

	// IPv6 private ranges: fc00::/7 (Unique Local Address)
	if ip16 := ip.To16(); ip16 != nil {
		return ip16[0] == 0xfc || ip16[0] == 0xfd
	}

	return false
}

// IsLoopback checks if an IP address is a loopback address.
func IsLoopback(ip net.IP) bool {
	return ip.IsLoopback()
}

// IsLinkLocal checks if an IP address is a link-local address.
func IsLinkLocal(ip net.IP) bool {
	return ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// IsMulticast checks if an IP address is a multicast address.
func IsMulticast(ip net.IP) bool {
	return ip.IsMulticast()
}

// ValidateDirection validates policy direction string.
func ValidateDirection(direction string) error {
	if direction == "" {
		return fmt.Errorf("empty direction")
	}

	direction = strings.ToLower(strings.TrimSpace(direction))

	validDirections := map[string]bool{
		"ingress": true,
		"egress":  true,
		"any":     true,
	}

	if !validDirections[direction] {
		return fmt.Errorf("invalid direction: %s (must be ingress/egress/any)", direction)
	}

	return nil
}

// ValidateAction validates policy action string.
func ValidateAction(action string) error {
	if action == "" {
		return fmt.Errorf("empty action")
	}

	action = strings.ToLower(strings.TrimSpace(action))

	validActions := map[string]bool{
		"allow": true,
		"deny":  true,
		"log":   true,
	}

	if !validActions[action] {
		return fmt.Errorf("invalid action: %s (must be allow/deny/log)", action)
	}

	return nil
}
