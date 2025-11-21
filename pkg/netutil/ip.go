// Package netutil provides network utility functions for IP address conversion and validation.
package netutil

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

// IPToUint32 converts an IPv4 address (net.IP or string) to uint32 in network byte order (big-endian).
// Returns 0 for invalid IPv4 addresses or IPv6 addresses.
//
// Example:
//   IPToUint32(net.ParseIP("192.168.1.1")) => 0xc0a80101 (big-endian)
//   IPToUint32LE(net.ParseIP("192.168.1.1")) => 0x0101a8c0 (little-endian, for eBPF)
func IPToUint32(ip net.IP) uint32 {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip4)
}

// IPToUint32LE converts an IPv4 address to uint32 in little-endian byte order.
// This is used for eBPF programs which typically use little-endian.
//
// Example:
//   IPToUint32LE(net.ParseIP("192.168.1.1")) => 0x0101a8c0
func IPToUint32LE(ip net.IP) uint32 {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(ip4)
}

// StringToUint32 converts an IP string to uint32 (big-endian).
// Returns 0 for invalid IP addresses.
func StringToUint32(ipStr string) uint32 {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0
	}
	return IPToUint32(ip)
}

// StringToUint32LE converts an IP string to uint32 (little-endian).
// Returns 0 for invalid IP addresses.
func StringToUint32LE(ipStr string) uint32 {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0
	}
	return IPToUint32LE(ip)
}

// Uint32ToIP converts a uint32 to IPv4 address (big-endian).
//
// Example:
//   Uint32ToIP(0xc0a80101) => net.IPv4(192, 168, 1, 1)
func Uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}

// Uint32LEToIP converts a uint32 to IPv4 address (little-endian).
// This is used for eBPF programs.
//
// Example:
//   Uint32LEToIP(0x0101a8c0) => net.IPv4(192, 168, 1, 1)
func Uint32LEToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.LittleEndian.PutUint32(ip, n)
	return ip
}

// Uint32ToString converts a uint32 to IP string (big-endian).
func Uint32ToString(n uint32) string {
	return Uint32ToIP(n).String()
}

// Uint32LEToString converts a uint32 to IP string (little-endian).
func Uint32LEToString(n uint32) string {
	return Uint32LEToIP(n).String()
}

// IPv6ToUint32Array converts an IPv6 address to [4]uint32 array (big-endian).
// Returns zero array for invalid IPv6 addresses.
//
// This format is compatible with eBPF programs that represent IPv6 as:
//   __u32 ip[4];  // 4 x 32-bit words
func IPv6ToUint32Array(ip net.IP) [4]uint32 {
	ip16 := ip.To16()
	if ip16 == nil {
		return [4]uint32{}
	}

	var arr [4]uint32
	for i := 0; i < 4; i++ {
		arr[i] = binary.BigEndian.Uint32(ip16[i*4 : (i+1)*4])
	}
	return arr
}

// Uint32ArrayToIPv6 converts [4]uint32 array to IPv6 address (big-endian).
func Uint32ArrayToIPv6(arr [4]uint32) net.IP {
	ip := make(net.IP, 16)
	for i := 0; i < 4; i++ {
		binary.BigEndian.PutUint32(ip[i*4:(i+1)*4], arr[i])
	}
	return ip
}

// ParseCIDR parses CIDR notation with auto-detection of /32 or /128 for bare IPs.
//
// Examples:
//   ParseCIDR("192.168.1.0/24") => 192.168.1.0/24
//   ParseCIDR("192.168.1.1")    => 192.168.1.1/32 (auto-detected IPv4)
//   ParseCIDR("2001:db8::1")    => 2001:db8::1/128 (auto-detected IPv6)
func ParseCIDR(cidr string) (net.IP, *net.IPNet, error) {
	// Normalize input
	cidr = strings.TrimSpace(cidr)

	if cidr == "" {
		return nil, nil, fmt.Errorf("empty CIDR")
	}

	// Auto-detect and add default mask
	if !strings.Contains(cidr, "/") {
		// Parse IP to detect version
		ip := net.ParseIP(cidr)
		if ip == nil {
			return nil, nil, fmt.Errorf("invalid IP address: %s", cidr)
		}

		if ip.To4() != nil {
			cidr += "/32" // IPv4
		} else {
			cidr += "/128" // IPv6
		}
	}

	// Parse CIDR
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid CIDR %s: %w", cidr, err)
	}

	return ip, ipnet, nil
}

// MaskToUint32 converts a net.IPMask to uint32 (big-endian).
// Returns 0 for invalid masks.
func MaskToUint32(mask net.IPMask) uint32 {
	if len(mask) != 4 {
		return 0
	}
	return binary.BigEndian.Uint32(mask)
}

// Uint32ToMask converts a uint32 to net.IPMask (big-endian).
func Uint32ToMask(n uint32) net.IPMask {
	mask := make(net.IPMask, 4)
	binary.BigEndian.PutUint32(mask, n)
	return mask
}

// IsIPv4 checks if an IP address is IPv4.
func IsIPv4(ip net.IP) bool {
	return ip.To4() != nil
}

// IsIPv6 checks if an IP address is IPv6 (excluding IPv4-mapped IPv6).
func IsIPv6(ip net.IP) bool {
	return ip.To16() != nil && ip.To4() == nil
}

// NormalizeIP normalizes an IP address to its canonical form.
// IPv4 addresses are returned as 4-byte representation.
// IPv6 addresses are returned as 16-byte representation.
func NormalizeIP(ip net.IP) net.IP {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4
	}
	return ip.To16()
}
