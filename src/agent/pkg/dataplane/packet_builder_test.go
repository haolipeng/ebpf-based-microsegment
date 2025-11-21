// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// TCP flags
const (
	TCP_FLAG_FIN = 0x01
	TCP_FLAG_SYN = 0x02
	TCP_FLAG_RST = 0x04
	TCP_FLAG_PSH = 0x08
	TCP_FLAG_ACK = 0x10
	TCP_FLAG_URG = 0x20
)

// PacketBuilder helps construct test packets for eBPF program testing
type PacketBuilder struct {
	buf *bytes.Buffer
}

// NewPacketBuilder creates a new packet builder
func NewPacketBuilder() *PacketBuilder {
	return &PacketBuilder{
		buf: new(bytes.Buffer),
	}
}

// AddEthernetHeader adds an Ethernet header
func (pb *PacketBuilder) AddEthernetHeader(dstMAC, srcMAC net.HardwareAddr, etherType uint16) *PacketBuilder {
	pb.buf.Write(dstMAC)
	pb.buf.Write(srcMAC)
	binary.Write(pb.buf, binary.BigEndian, etherType)
	return pb
}

// AddIPv4Header adds an IPv4 header
func (pb *PacketBuilder) AddIPv4Header(srcIP, dstIP net.IP, protocol uint8, totalLen uint16) *PacketBuilder {
	ipHdr := make([]byte, 20)
	ipHdr[0] = 0x45 // Version (4) + IHL (5)
	ipHdr[1] = 0    // DSCP + ECN
	binary.BigEndian.PutUint16(ipHdr[2:4], totalLen)
	ipHdr[8] = 64          // TTL
	ipHdr[9] = protocol    // Protocol
	copy(ipHdr[12:16], srcIP.To4())
	copy(ipHdr[16:20], dstIP.To4())

	// Calculate checksum
	checksum := calculateIPChecksum(ipHdr)
	binary.BigEndian.PutUint16(ipHdr[10:12], checksum)

	pb.buf.Write(ipHdr)
	return pb
}

// AddTCPHeader adds a TCP header
func (pb *PacketBuilder) AddTCPHeader(srcPort, dstPort uint16, seq, ack uint32, flags uint8, window uint16) *PacketBuilder {
	tcpHdr := make([]byte, 20)
	binary.BigEndian.PutUint16(tcpHdr[0:2], srcPort)
	binary.BigEndian.PutUint16(tcpHdr[2:4], dstPort)
	binary.BigEndian.PutUint32(tcpHdr[4:8], seq)
	binary.BigEndian.PutUint32(tcpHdr[8:12], ack)
	tcpHdr[12] = 0x50     // Data offset (5 words = 20 bytes) << 4
	tcpHdr[13] = flags    // Flags
	binary.BigEndian.PutUint16(tcpHdr[14:16], window)
	// Checksum will be calculated if needed

	pb.buf.Write(tcpHdr)
	return pb
}

// AddUDPHeader adds a UDP header
func (pb *PacketBuilder) AddUDPHeader(srcPort, dstPort uint16, length uint16) *PacketBuilder {
	udpHdr := make([]byte, 8)
	binary.BigEndian.PutUint16(udpHdr[0:2], srcPort)
	binary.BigEndian.PutUint16(udpHdr[2:4], dstPort)
	binary.BigEndian.PutUint16(udpHdr[4:6], length)
	// Checksum at offset 6-8 (optional for IPv4)

	pb.buf.Write(udpHdr)
	return pb
}

// AddPayload adds payload data
func (pb *PacketBuilder) AddPayload(data []byte) *PacketBuilder {
	pb.buf.Write(data)
	return pb
}

// Build returns the constructed packet
func (pb *PacketBuilder) Build() []byte {
	return pb.buf.Bytes()
}

// BuildTCPPacket is a convenience function to build a complete TCP packet
func BuildTCPPacket(srcIP, dstIP string, srcPort, dstPort uint16, flags uint8, payload []byte) []byte {
	src := net.ParseIP(srcIP)
	dst := net.ParseIP(dstIP)

	// Calculate total length: IP header (20) + TCP header (20) + payload
	totalLen := uint16(20 + 20 + len(payload))

	pb := NewPacketBuilder()
	pb.AddEthernetHeader(
		net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, // Dst MAC
		net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x02}, // Src MAC
		0x0800, // IPv4
	)
	pb.AddIPv4Header(src, dst, 6, totalLen) // Protocol 6 = TCP
	pb.AddTCPHeader(srcPort, dstPort, 0, 0, flags, 65535)

	if len(payload) > 0 {
		pb.AddPayload(payload)
	}

	return pb.Build()
}

// BuildUDPPacket is a convenience function to build a complete UDP packet
func BuildUDPPacket(srcIP, dstIP string, srcPort, dstPort uint16, payload []byte) []byte {
	src := net.ParseIP(srcIP)
	dst := net.ParseIP(dstIP)

	// Calculate lengths
	udpLen := uint16(8 + len(payload))
	totalLen := uint16(20 + udpLen)

	pb := NewPacketBuilder()
	pb.AddEthernetHeader(
		net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
		net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		0x0800,
	)
	pb.AddIPv4Header(src, dst, 17, totalLen) // Protocol 17 = UDP
	pb.AddUDPHeader(srcPort, dstPort, udpLen)

	if len(payload) > 0 {
		pb.AddPayload(payload)
	}

	return pb.Build()
}

// calculateIPChecksum calculates IP header checksum
func calculateIPChecksum(header []byte) uint16 {
	sum := uint32(0)

	// Sum all 16-bit words
	for i := 0; i < len(header); i += 2 {
		if i == 10 { // Skip checksum field
			continue
		}
		sum += uint32(binary.BigEndian.Uint16(header[i : i+2]))
	}

	// Add carry
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}

// Test packet builder functions
func TestPacketBuilder(t *testing.T) {
	t.Run("build TCP SYN packet", func(t *testing.T) {
		pkt := BuildTCPPacket("192.168.1.100", "192.168.1.1", 12345, 80, TCP_FLAG_SYN, nil)

		// Verify packet structure
		require.True(t, len(pkt) >= 54, "packet too short") // 14 (eth) + 20 (ip) + 20 (tcp)

		// Check Ethernet header
		require.Equal(t, uint16(0x0800), binary.BigEndian.Uint16(pkt[12:14]), "wrong EtherType")

		// Check IP header
		require.Equal(t, byte(0x45), pkt[14], "wrong IP version/IHL")
		require.Equal(t, byte(6), pkt[23], "wrong IP protocol")

		// Check source/dest IPs
		srcIP := net.IP(pkt[26:30])
		dstIP := net.IP(pkt[30:34])
		require.Equal(t, "192.168.1.100", srcIP.String())
		require.Equal(t, "192.168.1.1", dstIP.String())

		// Check TCP header
		tcpSrcPort := binary.BigEndian.Uint16(pkt[34:36])
		tcpDstPort := binary.BigEndian.Uint16(pkt[36:38])
		tcpFlags := pkt[47]

		require.Equal(t, uint16(12345), tcpSrcPort)
		require.Equal(t, uint16(80), tcpDstPort)
		require.Equal(t, uint8(TCP_FLAG_SYN), tcpFlags)
	})

	t.Run("build UDP packet", func(t *testing.T) {
		payload := []byte("test data")
		pkt := BuildUDPPacket("10.0.0.1", "10.0.0.2", 5000, 53, payload)

		require.True(t, len(pkt) >= 42+len(payload)) // 14 (eth) + 20 (ip) + 8 (udp) + payload

		// Check IP protocol
		require.Equal(t, byte(17), pkt[23], "wrong IP protocol for UDP")

		// Check UDP ports
		udpSrcPort := binary.BigEndian.Uint16(pkt[34:36])
		udpDstPort := binary.BigEndian.Uint16(pkt[36:38])

		require.Equal(t, uint16(5000), udpSrcPort)
		require.Equal(t, uint16(53), udpDstPort)

		// Check payload
		pktPayload := pkt[42:]
		require.Equal(t, payload, pktPayload)
	})

	t.Run("build TCP packet with payload", func(t *testing.T) {
		payload := []byte("HTTP GET /index.html")
		pkt := BuildTCPPacket("172.16.0.1", "172.16.0.2", 8080, 443, TCP_FLAG_PSH|TCP_FLAG_ACK, payload)

		require.True(t, len(pkt) >= 54+len(payload))

		// Verify payload is present
		pktPayload := pkt[54:]
		require.Equal(t, payload, pktPayload)
	})
}

// TestIPChecksum tests the IP checksum calculation
func TestIPChecksum(t *testing.T) {
	// Create a simple IP header
	header := make([]byte, 20)
	header[0] = 0x45
	header[9] = 6 // TCP
	copy(header[12:16], net.ParseIP("192.168.1.1").To4())
	copy(header[16:20], net.ParseIP("192.168.1.2").To4())

	checksum := calculateIPChecksum(header)
	require.NotZero(t, checksum, "checksum should not be zero")

	// Set checksum in header
	binary.BigEndian.PutUint16(header[10:12], checksum)

	// Verify checksum by calculating again (should get same result since we skip checksum field)
	verifySum := calculateIPChecksum(header)
	require.Equal(t, checksum, verifySum, "checksum verification failed")

	// Also verify by summing all fields including checksum
	sum := uint32(0)
	for i := 0; i < len(header); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(header[i : i+2]))
	}
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	// Checksum is correct if the sum of all words (including checksum) equals 0xFFFF
	require.Equal(t, uint16(0xFFFF), uint16(sum), "checksum verification by sum failed")
}
