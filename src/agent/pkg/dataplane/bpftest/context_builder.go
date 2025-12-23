// Package bpftest provides testing utilities for eBPF programs.
//
// This file implements the SKBuffContextBuilder for constructing __sk_buff
// context structures used in BPF_PROG_TEST_RUN.

// input: __sk_buff context fields (ifindex, ingress/egress, mark, protocol)
// output: serialized __sk_buff context for BPF_PROG_TEST_RUN
// pos: eBPF test context builder for SK_BUFF - if file updated, must sync with this header comment and pkg/dataplane/CLAUDE.md
package bpftest

import (
	"encoding/binary"
)

// SKBuffContextBuilder provides a fluent API for constructing __sk_buff contexts.
//
// Example:
//
//	ctx := NewSKBuffContextBuilder().
//	    Ingress().
//	    WithIfindex(1).
//	    WithMark(0x100).
//	    Build()
type SKBuffContextBuilder struct {
	ctx SKBuffContext
}

// NewSKBuffContextBuilder creates a new context builder with default values.
func NewSKBuffContextBuilder() *SKBuffContextBuilder {
	return &SKBuffContextBuilder{
		ctx: SKBuffContext{
			Ingress:  true,
			Protocol: ETH_P_IP,
		},
	}
}

// Ingress sets the context for ingress traffic.
func (b *SKBuffContextBuilder) Ingress() *SKBuffContextBuilder {
	b.ctx.Ingress = true
	return b
}

// Egress sets the context for egress traffic.
func (b *SKBuffContextBuilder) Egress() *SKBuffContextBuilder {
	b.ctx.Ingress = false
	return b
}

// WithIfindex sets the interface index.
func (b *SKBuffContextBuilder) WithIfindex(ifindex uint32) *SKBuffContextBuilder {
	b.ctx.Ifindex = ifindex
	return b
}

// WithMark sets the packet mark (used for policy routing).
func (b *SKBuffContextBuilder) WithMark(mark uint32) *SKBuffContextBuilder {
	b.ctx.Mark = mark
	return b
}

// WithPriority sets the packet priority.
func (b *SKBuffContextBuilder) WithPriority(priority uint32) *SKBuffContextBuilder {
	b.ctx.Priority = priority
	return b
}

// WithProtocol sets the L3 protocol.
func (b *SKBuffContextBuilder) WithProtocol(protocol uint16) *SKBuffContextBuilder {
	b.ctx.Protocol = protocol
	return b
}

// WithVLAN sets the VLAN ID.
func (b *SKBuffContextBuilder) WithVLAN(vlanID uint16) *SKBuffContextBuilder {
	b.ctx.VLANID = vlanID
	return b
}

// IPv4 sets the protocol to IPv4.
func (b *SKBuffContextBuilder) IPv4() *SKBuffContextBuilder {
	b.ctx.Protocol = ETH_P_IP
	return b
}

// IPv6 sets the protocol to IPv6.
func (b *SKBuffContextBuilder) IPv6() *SKBuffContextBuilder {
	b.ctx.Protocol = ETH_P_IPV6
	return b
}

// ARP sets the protocol to ARP.
func (b *SKBuffContextBuilder) ARP() *SKBuffContextBuilder {
	b.ctx.Protocol = ETH_P_ARP
	return b
}

// Build returns the constructed SKBuffContext.
func (b *SKBuffContextBuilder) Build() *SKBuffContext {
	result := b.ctx
	return &result
}

// Ethernet protocol constants (network byte order values)
const (
	ETH_P_IP   uint16 = 0x0800 // IPv4
	ETH_P_IPV6 uint16 = 0x86DD // IPv6
	ETH_P_ARP  uint16 = 0x0806 // ARP
	ETH_P_8021Q uint16 = 0x8100 // 802.1Q VLAN
)

// SKBuffData represents the full __sk_buff structure for advanced testing.
// This matches the kernel's __sk_buff definition for TC programs.
//
// Reference: include/uapi/linux/bpf.h
type SKBuffData struct {
	Len             uint32 // Packet length
	PktType         uint32 // Packet type
	Mark            uint32 // Packet mark
	QueueMapping    uint32 // Queue mapping
	Protocol        uint32 // L3 protocol
	VlanPresent     uint32 // VLAN tag present
	VlanTCI         uint32 // VLAN TCI
	VlanProto       uint32 // VLAN protocol
	Priority        uint32 // Packet priority
	IngressIfindex  uint32 // Ingress interface index
	Ifindex         uint32 // Interface index
	TCIndex         uint32 // TC class ID
	CB              [5]uint32 // Control buffer
	Hash            uint32 // Packet hash
	TCClassID       uint32 // TC class ID
	Data            uint32 // Packet data start
	DataEnd         uint32 // Packet data end
	NapiID          uint32 // NAPI ID
	Family          uint32 // Address family
	RemoteAddr4     uint32 // Remote IPv4 address
	LocalAddr4      uint32 // Local IPv4 address
	RemoteAddr6     [4]uint32 // Remote IPv6 address
	LocalAddr6      [4]uint32 // Local IPv6 address
	RemotePort      uint32 // Remote port
	LocalPort       uint32 // Local port
	DataMeta        uint32 // Metadata start
	FlowKeys        uint64 // Flow keys pointer
	TstampType      uint64 // Timestamp type
	Tstamp          uint64 // Timestamp
	WireLen         uint32 // Original wire length
	GSOSegs         uint32 // GSO segments
	SK              uint64 // Socket pointer
	GSOSize         uint32 // GSO size
	Pad             uint32 // Padding
	Hwtstamp        uint64 // Hardware timestamp
}

// NewSKBuffData creates a new SKBuffData with default values.
func NewSKBuffData() *SKBuffData {
	return &SKBuffData{
		Protocol: uint32(ETH_P_IP),
	}
}

// SetIngress configures the context for ingress processing.
func (s *SKBuffData) SetIngress(ifindex uint32) *SKBuffData {
	s.IngressIfindex = ifindex
	s.Ifindex = ifindex
	return s
}

// SetEgress configures the context for egress processing.
func (s *SKBuffData) SetEgress(ifindex uint32) *SKBuffData {
	s.IngressIfindex = 0
	s.Ifindex = ifindex
	return s
}

// SetMark sets the packet mark.
func (s *SKBuffData) SetMark(mark uint32) *SKBuffData {
	s.Mark = mark
	return s
}

// SetVLAN sets VLAN information.
func (s *SKBuffData) SetVLAN(vlanID uint16) *SKBuffData {
	s.VlanPresent = 1
	s.VlanTCI = uint32(vlanID)
	s.VlanProto = uint32(ETH_P_8021Q)
	return s
}

// SetProtocol sets the L3 protocol.
func (s *SKBuffData) SetProtocol(proto uint16) *SKBuffData {
	s.Protocol = uint32(proto)
	return s
}

// SetPacketLength sets the packet length.
func (s *SKBuffData) SetPacketLength(length uint32) *SKBuffData {
	s.Len = length
	s.WireLen = length
	return s
}

// ToBytes serializes the SKBuffData to bytes for passing to BPF_PROG_TEST_RUN.
// Note: The kernel only uses a subset of these fields for TC programs.
func (s *SKBuffData) ToBytes() []byte {
	// Allocate buffer for the full structure
	// The actual size needed depends on kernel version
	buf := make([]byte, 256)

	// Pack the structure fields
	// These offsets are based on the kernel's __sk_buff layout
	binary.LittleEndian.PutUint32(buf[0:4], s.Len)
	binary.LittleEndian.PutUint32(buf[4:8], s.PktType)
	binary.LittleEndian.PutUint32(buf[8:12], s.Mark)
	binary.LittleEndian.PutUint32(buf[12:16], s.QueueMapping)
	binary.LittleEndian.PutUint32(buf[16:20], s.Protocol)
	binary.LittleEndian.PutUint32(buf[20:24], s.VlanPresent)
	binary.LittleEndian.PutUint32(buf[24:28], s.VlanTCI)
	binary.LittleEndian.PutUint32(buf[28:32], s.VlanProto)
	binary.LittleEndian.PutUint32(buf[32:36], s.Priority)
	binary.LittleEndian.PutUint32(buf[36:40], s.IngressIfindex)
	binary.LittleEndian.PutUint32(buf[40:44], s.Ifindex)
	binary.LittleEndian.PutUint32(buf[44:48], s.TCIndex)

	// CB array (5 * 4 bytes)
	for i, v := range s.CB {
		binary.LittleEndian.PutUint32(buf[48+i*4:52+i*4], v)
	}

	binary.LittleEndian.PutUint32(buf[68:72], s.Hash)
	binary.LittleEndian.PutUint32(buf[72:76], s.TCClassID)

	return buf
}

// IngressContext creates a simple ingress context for common test cases.
func IngressContext() *SKBuffContext {
	return NewSKBuffContextBuilder().
		Ingress().
		WithIfindex(1).
		IPv4().
		Build()
}

// EgressContext creates a simple egress context for common test cases.
func EgressContext() *SKBuffContext {
	return NewSKBuffContextBuilder().
		Egress().
		WithIfindex(1).
		IPv4().
		Build()
}

// IngressWithVLAN creates an ingress context with VLAN tagging.
func IngressWithVLAN(vlanID uint16) *SKBuffContext {
	return NewSKBuffContextBuilder().
		Ingress().
		WithIfindex(1).
		WithVLAN(vlanID).
		IPv4().
		Build()
}

// EgressWithVLAN creates an egress context with VLAN tagging.
func EgressWithVLAN(vlanID uint16) *SKBuffContext {
	return NewSKBuffContextBuilder().
		Egress().
		WithIfindex(1).
		WithVLAN(vlanID).
		IPv4().
		Build()
}

// ContextWithMark creates a context with a specific packet mark.
func ContextWithMark(ingress bool, mark uint32) *SKBuffContext {
	builder := NewSKBuffContextBuilder().
		WithIfindex(1).
		WithMark(mark).
		IPv4()

	if ingress {
		builder.Ingress()
	} else {
		builder.Egress()
	}

	return builder.Build()
}
