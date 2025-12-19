// Package bpftest provides testing utilities for eBPF programs.
//
// This file implements PacketBuilder for constructing test packets with
// proper headers and checksums for BPF_PROG_TEST_RUN testing.
package bpftest

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"net"
)

// TCP flags
const (
	TCP_FLAG_FIN = 0x01
	TCP_FLAG_SYN = 0x02
	TCP_FLAG_RST = 0x04
	TCP_FLAG_PSH = 0x08
	TCP_FLAG_ACK = 0x10
	TCP_FLAG_URG = 0x20
	TCP_FLAG_ECE = 0x40
	TCP_FLAG_CWR = 0x80
)

// ICMP types
const (
	ICMP_ECHO_REPLY   = 0
	ICMP_ECHO_REQUEST = 8
	ICMP_DEST_UNREACH = 3
	ICMP_TIME_EXCEED  = 11
)

// PacketBuilder provides a fluent API for building test packets.
type PacketBuilder struct {
	buf *bytes.Buffer

	// Ethernet header fields
	srcMAC   net.HardwareAddr
	dstMAC   net.HardwareAddr
	etherType uint16

	// IP header fields
	srcIP    net.IP
	dstIP    net.IP
	protocol uint8
	ttl      uint8
	tos      uint8
	ipID     uint16

	// TCP/UDP header fields
	srcPort  uint16
	dstPort  uint16
	tcpFlags uint8
	seqNum   uint32
	ackNum   uint32
	window   uint16

	// ICMP fields
	icmpType uint8
	icmpCode uint8
	icmpID   uint16
	icmpSeq  uint16

	// Payload
	payload []byte
}

// NewPacketBuilder creates a new packet builder with default values.
func NewPacketBuilder() *PacketBuilder {
	return &PacketBuilder{
		buf:       new(bytes.Buffer),
		srcMAC:    net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		dstMAC:    net.HardwareAddr{0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb},
		etherType: 0x0800, // IPv4
		srcIP:     net.ParseIP("10.0.0.1"),
		dstIP:     net.ParseIP("10.0.0.2"),
		protocol:  IPPROTO_TCP,
		ttl:       64,
		srcPort:   12345,
		dstPort:   80,
		window:    65535,
		seqNum:    1000,
		ackNum:    0,
	}
}

// WithSrcMAC sets the source MAC address.
func (p *PacketBuilder) WithSrcMAC(mac string) *PacketBuilder {
	if hw, err := net.ParseMAC(mac); err == nil {
		p.srcMAC = hw
	}
	return p
}

// WithDstMAC sets the destination MAC address.
func (p *PacketBuilder) WithDstMAC(mac string) *PacketBuilder {
	if hw, err := net.ParseMAC(mac); err == nil {
		p.dstMAC = hw
	}
	return p
}

// WithSrcIP sets the source IP address.
func (p *PacketBuilder) WithSrcIP(ip string) *PacketBuilder {
	if parsed := net.ParseIP(ip); parsed != nil {
		p.srcIP = parsed.To4()
	}
	return p
}

// WithDstIP sets the destination IP address.
func (p *PacketBuilder) WithDstIP(ip string) *PacketBuilder {
	if parsed := net.ParseIP(ip); parsed != nil {
		p.dstIP = parsed.To4()
	}
	return p
}

// WithSrcPort sets the source port.
func (p *PacketBuilder) WithSrcPort(port uint16) *PacketBuilder {
	p.srcPort = port
	return p
}

// WithDstPort sets the destination port.
func (p *PacketBuilder) WithDstPort(port uint16) *PacketBuilder {
	p.dstPort = port
	return p
}

// TCP sets the protocol to TCP.
func (p *PacketBuilder) TCP() *PacketBuilder {
	p.protocol = IPPROTO_TCP
	return p
}

// UDP sets the protocol to UDP.
func (p *PacketBuilder) UDP() *PacketBuilder {
	p.protocol = IPPROTO_UDP
	return p
}

// ICMP sets the protocol to ICMP.
func (p *PacketBuilder) ICMP() *PacketBuilder {
	p.protocol = IPPROTO_ICMP
	return p
}

// WithTTL sets the IP TTL.
func (p *PacketBuilder) WithTTL(ttl uint8) *PacketBuilder {
	p.ttl = ttl
	return p
}

// WithTOS sets the IP Type of Service.
func (p *PacketBuilder) WithTOS(tos uint8) *PacketBuilder {
	p.tos = tos
	return p
}

// WithIPID sets the IP identification field.
func (p *PacketBuilder) WithIPID(id uint16) *PacketBuilder {
	p.ipID = id
	return p
}

// WithSeqNum sets the TCP sequence number.
func (p *PacketBuilder) WithSeqNum(seq uint32) *PacketBuilder {
	p.seqNum = seq
	return p
}

// WithAckNum sets the TCP acknowledgment number.
func (p *PacketBuilder) WithAckNum(ack uint32) *PacketBuilder {
	p.ackNum = ack
	return p
}

// WithWindow sets the TCP window size.
func (p *PacketBuilder) WithWindow(window uint16) *PacketBuilder {
	p.window = window
	return p
}

// WithTCPFlags sets the TCP flags directly.
func (p *PacketBuilder) WithTCPFlags(flags uint8) *PacketBuilder {
	p.tcpFlags = flags
	return p
}

// SYN sets the TCP SYN flag.
func (p *PacketBuilder) SYN() *PacketBuilder {
	p.tcpFlags = TCP_FLAG_SYN
	return p
}

// SYNACK sets the TCP SYN+ACK flags.
func (p *PacketBuilder) SYNACK() *PacketBuilder {
	p.tcpFlags = TCP_FLAG_SYN | TCP_FLAG_ACK
	return p
}

// ACK sets the TCP ACK flag.
func (p *PacketBuilder) ACK() *PacketBuilder {
	p.tcpFlags = TCP_FLAG_ACK
	return p
}

// FIN sets the TCP FIN flag.
func (p *PacketBuilder) FIN() *PacketBuilder {
	p.tcpFlags = TCP_FLAG_FIN
	return p
}

// FINACK sets the TCP FIN+ACK flags.
func (p *PacketBuilder) FINACK() *PacketBuilder {
	p.tcpFlags = TCP_FLAG_FIN | TCP_FLAG_ACK
	return p
}

// RST sets the TCP RST flag.
func (p *PacketBuilder) RST() *PacketBuilder {
	p.tcpFlags = TCP_FLAG_RST
	return p
}

// PSHACK sets the TCP PSH+ACK flags (data packet).
func (p *PacketBuilder) PSHACK() *PacketBuilder {
	p.tcpFlags = TCP_FLAG_PSH | TCP_FLAG_ACK
	return p
}

// WithICMPType sets the ICMP type.
func (p *PacketBuilder) WithICMPType(icmpType uint8) *PacketBuilder {
	p.icmpType = icmpType
	return p
}

// WithICMPCode sets the ICMP code.
func (p *PacketBuilder) WithICMPCode(icmpCode uint8) *PacketBuilder {
	p.icmpCode = icmpCode
	return p
}

// EchoRequest sets ICMP type to Echo Request.
func (p *PacketBuilder) EchoRequest() *PacketBuilder {
	p.icmpType = ICMP_ECHO_REQUEST
	p.icmpCode = 0
	return p
}

// EchoReply sets ICMP type to Echo Reply.
func (p *PacketBuilder) EchoReply() *PacketBuilder {
	p.icmpType = ICMP_ECHO_REPLY
	p.icmpCode = 0
	return p
}

// WithPayload sets the packet payload.
func (p *PacketBuilder) WithPayload(payload []byte) *PacketBuilder {
	p.payload = payload
	return p
}

// Build constructs the complete packet with all headers.
func (p *PacketBuilder) Build() []byte {
	p.buf.Reset()

	// Build Ethernet header
	p.buf.Write(p.dstMAC)
	p.buf.Write(p.srcMAC)
	binary.Write(p.buf, binary.BigEndian, p.etherType)

	// Build IP header
	ipHeader := p.buildIPHeader()
	p.buf.Write(ipHeader)

	// Build transport header based on protocol
	switch p.protocol {
	case IPPROTO_TCP:
		tcpHeader := p.buildTCPHeader()
		p.buf.Write(tcpHeader)
	case IPPROTO_UDP:
		udpHeader := p.buildUDPHeader()
		p.buf.Write(udpHeader)
	case IPPROTO_ICMP:
		icmpHeader := p.buildICMPHeader()
		p.buf.Write(icmpHeader)
	}

	// Add payload
	if len(p.payload) > 0 {
		p.buf.Write(p.payload)
	}

	return p.buf.Bytes()
}

// buildIPHeader constructs the IPv4 header.
func (p *PacketBuilder) buildIPHeader() []byte {
	header := make([]byte, 20)

	// Version (4) + IHL (5) = 0x45
	header[0] = 0x45
	// TOS
	header[1] = p.tos
	// Total length (will be calculated)
	totalLen := 20 + p.transportHeaderLen() + len(p.payload)
	binary.BigEndian.PutUint16(header[2:4], uint16(totalLen))
	// Identification
	binary.BigEndian.PutUint16(header[4:6], p.ipID)
	// Flags + Fragment offset
	header[6] = 0x40 // Don't fragment
	header[7] = 0x00
	// TTL
	header[8] = p.ttl
	// Protocol
	header[9] = p.protocol
	// Header checksum (will be calculated)
	header[10] = 0
	header[11] = 0
	// Source IP
	copy(header[12:16], p.srcIP.To4())
	// Destination IP
	copy(header[16:20], p.dstIP.To4())

	// Calculate checksum
	checksum := ipChecksum(header)
	binary.BigEndian.PutUint16(header[10:12], checksum)

	return header
}

// transportHeaderLen returns the transport header length.
func (p *PacketBuilder) transportHeaderLen() int {
	switch p.protocol {
	case IPPROTO_TCP:
		return 20 // Minimum TCP header
	case IPPROTO_UDP:
		return 8
	case IPPROTO_ICMP:
		return 8
	default:
		return 0
	}
}

// buildTCPHeader constructs the TCP header.
func (p *PacketBuilder) buildTCPHeader() []byte {
	header := make([]byte, 20)

	// Source port
	binary.BigEndian.PutUint16(header[0:2], p.srcPort)
	// Destination port
	binary.BigEndian.PutUint16(header[2:4], p.dstPort)
	// Sequence number
	binary.BigEndian.PutUint32(header[4:8], p.seqNum)
	// Acknowledgment number
	binary.BigEndian.PutUint32(header[8:12], p.ackNum)
	// Data offset (5 * 4 = 20 bytes) << 4 + reserved
	header[12] = 0x50
	// Flags
	header[13] = p.tcpFlags
	// Window
	binary.BigEndian.PutUint16(header[14:16], p.window)
	// Checksum (will be calculated)
	header[16] = 0
	header[17] = 0
	// Urgent pointer
	header[18] = 0
	header[19] = 0

	// Calculate TCP checksum with pseudo-header
	checksum := tcpChecksum(p.srcIP.To4(), p.dstIP.To4(), header, p.payload)
	binary.BigEndian.PutUint16(header[16:18], checksum)

	return header
}

// buildUDPHeader constructs the UDP header.
func (p *PacketBuilder) buildUDPHeader() []byte {
	header := make([]byte, 8)

	// Source port
	binary.BigEndian.PutUint16(header[0:2], p.srcPort)
	// Destination port
	binary.BigEndian.PutUint16(header[2:4], p.dstPort)
	// Length
	length := uint16(8 + len(p.payload))
	binary.BigEndian.PutUint16(header[4:6], length)
	// Checksum (optional for UDP over IPv4, set to 0)
	header[6] = 0
	header[7] = 0

	return header
}

// buildICMPHeader constructs the ICMP header.
func (p *PacketBuilder) buildICMPHeader() []byte {
	header := make([]byte, 8)

	// Type
	header[0] = p.icmpType
	// Code
	header[1] = p.icmpCode
	// Checksum (will be calculated)
	header[2] = 0
	header[3] = 0
	// Identifier
	binary.BigEndian.PutUint16(header[4:6], p.icmpID)
	// Sequence number
	binary.BigEndian.PutUint16(header[6:8], p.icmpSeq)

	// Calculate ICMP checksum
	checksum := icmpChecksum(header, p.payload)
	binary.BigEndian.PutUint16(header[2:4], checksum)

	return header
}

// ipChecksum calculates the IP header checksum.
func ipChecksum(header []byte) uint16 {
	var sum uint32
	for i := 0; i < len(header); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(header[i : i+2]))
	}
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}

// tcpChecksum calculates the TCP checksum including pseudo-header.
func tcpChecksum(srcIP, dstIP net.IP, tcpHeader, payload []byte) uint16 {
	// Build pseudo-header
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], srcIP)
	copy(pseudoHeader[4:8], dstIP)
	pseudoHeader[8] = 0
	pseudoHeader[9] = IPPROTO_TCP
	tcpLen := uint16(len(tcpHeader) + len(payload))
	binary.BigEndian.PutUint16(pseudoHeader[10:12], tcpLen)

	// Combine all data for checksum
	var data []byte
	data = append(data, pseudoHeader...)
	data = append(data, tcpHeader...)
	data = append(data, payload...)

	// Pad if odd length
	if len(data)%2 != 0 {
		data = append(data, 0)
	}

	return checksumRFC1071(data)
}

// icmpChecksum calculates the ICMP checksum.
func icmpChecksum(header, payload []byte) uint16 {
	var data []byte
	data = append(data, header...)
	data = append(data, payload...)

	// Pad if odd length
	if len(data)%2 != 0 {
		data = append(data, 0)
	}

	return checksumRFC1071(data)
}

// checksumRFC1071 calculates the Internet checksum per RFC 1071.
func checksumRFC1071(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}

// Convenience functions for common packet types

// BuildTCPSYN creates a TCP SYN packet.
func BuildTCPSYN(srcIP, dstIP string, srcPort, dstPort uint16) []byte {
	return NewPacketBuilder().
		WithSrcIP(srcIP).
		WithDstIP(dstIP).
		WithSrcPort(srcPort).
		WithDstPort(dstPort).
		TCP().
		SYN().
		Build()
}

// BuildTCPSYNACK creates a TCP SYN+ACK packet.
func BuildTCPSYNACK(srcIP, dstIP string, srcPort, dstPort uint16, ackNum uint32) []byte {
	return NewPacketBuilder().
		WithSrcIP(srcIP).
		WithDstIP(dstIP).
		WithSrcPort(srcPort).
		WithDstPort(dstPort).
		TCP().
		SYNACK().
		WithAckNum(ackNum).
		Build()
}

// BuildTCPACK creates a TCP ACK packet.
func BuildTCPACK(srcIP, dstIP string, srcPort, dstPort uint16, seqNum, ackNum uint32) []byte {
	return NewPacketBuilder().
		WithSrcIP(srcIP).
		WithDstIP(dstIP).
		WithSrcPort(srcPort).
		WithDstPort(dstPort).
		TCP().
		ACK().
		WithSeqNum(seqNum).
		WithAckNum(ackNum).
		Build()
}

// BuildTCPData creates a TCP data packet (PSH+ACK).
func BuildTCPData(srcIP, dstIP string, srcPort, dstPort uint16, seqNum, ackNum uint32, payload []byte) []byte {
	return NewPacketBuilder().
		WithSrcIP(srcIP).
		WithDstIP(dstIP).
		WithSrcPort(srcPort).
		WithDstPort(dstPort).
		TCP().
		PSHACK().
		WithSeqNum(seqNum).
		WithAckNum(ackNum).
		WithPayload(payload).
		Build()
}

// BuildTCPFIN creates a TCP FIN packet.
func BuildTCPFIN(srcIP, dstIP string, srcPort, dstPort uint16, seqNum, ackNum uint32) []byte {
	return NewPacketBuilder().
		WithSrcIP(srcIP).
		WithDstIP(dstIP).
		WithSrcPort(srcPort).
		WithDstPort(dstPort).
		TCP().
		FINACK().
		WithSeqNum(seqNum).
		WithAckNum(ackNum).
		Build()
}

// BuildTCPRST creates a TCP RST packet.
func BuildTCPRST(srcIP, dstIP string, srcPort, dstPort uint16, seqNum uint32) []byte {
	return NewPacketBuilder().
		WithSrcIP(srcIP).
		WithDstIP(dstIP).
		WithSrcPort(srcPort).
		WithDstPort(dstPort).
		TCP().
		RST().
		WithSeqNum(seqNum).
		Build()
}

// BuildUDPPacket creates a UDP packet.
func BuildUDPPacket(srcIP, dstIP string, srcPort, dstPort uint16, payload []byte) []byte {
	return NewPacketBuilder().
		WithSrcIP(srcIP).
		WithDstIP(dstIP).
		WithSrcPort(srcPort).
		WithDstPort(dstPort).
		UDP().
		WithPayload(payload).
		Build()
}

// BuildICMPEchoRequest creates an ICMP Echo Request (ping) packet.
func BuildICMPEchoRequest(srcIP, dstIP string, id, seq uint16) []byte {
	p := NewPacketBuilder().
		WithSrcIP(srcIP).
		WithDstIP(dstIP).
		ICMP().
		EchoRequest()
	p.icmpID = id
	p.icmpSeq = seq
	return p.Build()
}

// BuildICMPEchoReply creates an ICMP Echo Reply packet.
func BuildICMPEchoReply(srcIP, dstIP string, id, seq uint16) []byte {
	p := NewPacketBuilder().
		WithSrcIP(srcIP).
		WithDstIP(dstIP).
		ICMP().
		EchoReply()
	p.icmpID = id
	p.icmpSeq = seq
	return p.Build()
}

// TCPHandshake represents a complete TCP three-way handshake.
type TCPHandshake struct {
	ClientIP   string
	ServerIP   string
	ClientPort uint16
	ServerPort uint16
	ClientISN  uint32 // Initial sequence number from client
	ServerISN  uint32 // Initial sequence number from server
}

// BuildSYN returns the SYN packet (client -> server).
func (h *TCPHandshake) BuildSYN() []byte {
	return NewPacketBuilder().
		WithSrcIP(h.ClientIP).
		WithDstIP(h.ServerIP).
		WithSrcPort(h.ClientPort).
		WithDstPort(h.ServerPort).
		TCP().
		SYN().
		WithSeqNum(h.ClientISN).
		Build()
}

// BuildSYNACK returns the SYN+ACK packet (server -> client).
func (h *TCPHandshake) BuildSYNACK() []byte {
	return NewPacketBuilder().
		WithSrcIP(h.ServerIP).
		WithDstIP(h.ClientIP).
		WithSrcPort(h.ServerPort).
		WithDstPort(h.ClientPort).
		TCP().
		SYNACK().
		WithSeqNum(h.ServerISN).
		WithAckNum(h.ClientISN + 1).
		Build()
}

// BuildACK returns the ACK packet (client -> server).
func (h *TCPHandshake) BuildACK() []byte {
	return NewPacketBuilder().
		WithSrcIP(h.ClientIP).
		WithDstIP(h.ServerIP).
		WithSrcPort(h.ClientPort).
		WithDstPort(h.ServerPort).
		TCP().
		ACK().
		WithSeqNum(h.ClientISN + 1).
		WithAckNum(h.ServerISN + 1).
		Build()
}

// GenerateFCS generates a fake Frame Check Sequence (for testing only).
// Real Ethernet FCS is typically handled by hardware.
func GenerateFCS(packet []byte) uint32 {
	return crc32.ChecksumIEEE(packet)
}
