// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: N/A (type definition)
// output: identity data structures (SecurityIdentity, NumericIdentity)
// pos: identity type definitions - if file updated, must sync with this header comment and pkg/identity/CLAUDE.md
package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// NumericIdentity represents a security identity as a 32-bit unsigned integer.
// The identity encodes both scope and value:
//   - Bits 24-31: Scope (0x00=Global, 0x01=Local, 0x02=RemoteNode)
//   - Bits 0-23: Identity value
type NumericIdentity uint32

// Identity scope constants (high 8 bits)
const (
	// IdentityScopeMask masks the scope bits
	IdentityScopeMask NumericIdentity = 0xFF000000

	// IdentityScopeGlobal indicates cluster-wide scope
	IdentityScopeGlobal NumericIdentity = 0x00000000

	// IdentityScopeLocal indicates node-local scope (e.g., CIDR policies)
	IdentityScopeLocal NumericIdentity = 0x01000000

	// IdentityScopeRemoteNode indicates remote node identity
	IdentityScopeRemoteNode NumericIdentity = 0x02000000
)

// Reserved identity constants (IDs 0-255 are reserved)
const (
	// IdentityUnknown represents an unknown or uninitialized identity
	IdentityUnknown NumericIdentity = 0

	// IdentityHost represents the local host
	IdentityHost NumericIdentity = 1

	// IdentityWorld represents external/internet traffic
	IdentityWorld NumericIdentity = 2

	// IdentityUnmanaged represents unmanaged endpoints
	IdentityUnmanaged NumericIdentity = 3

	// IdentityHealth represents health check endpoints
	IdentityHealth NumericIdentity = 4

	// IdentityInit represents initializing endpoints
	IdentityInit NumericIdentity = 5

	// IdentityRemoteNode represents remote cluster nodes
	IdentityRemoteNode NumericIdentity = 6

	// IdentityKubeAPIServer represents Kubernetes API server
	IdentityKubeAPIServer NumericIdentity = 7

	// ReservedIdentityMax is the maximum reserved identity ID
	ReservedIdentityMax NumericIdentity = 255

	// MinAllocatedIdentity is the first allocatable identity ID
	MinAllocatedIdentity NumericIdentity = 256

	// MaxAllocatedIdentity is the maximum allocatable identity ID
	MaxAllocatedIdentity NumericIdentity = 0x00FFFFFF // 16M identities
)

// IsReserved returns true if the identity is a reserved identity
func (id NumericIdentity) IsReserved() bool {
	return id <= ReservedIdentityMax
}

// Scope returns the scope of the identity
func (id NumericIdentity) Scope() NumericIdentity {
	return id & IdentityScopeMask
}

// Value returns the identity value without scope bits
func (id NumericIdentity) Value() uint32 {
	return uint32(id & ^IdentityScopeMask)
}

// IsGlobal returns true if the identity has global scope
func (id NumericIdentity) IsGlobal() bool {
	return id.Scope() == IdentityScopeGlobal
}

// IsLocal returns true if the identity has local scope
func (id NumericIdentity) IsLocal() bool {
	return id.Scope() == IdentityScopeLocal
}

// IsRemoteNode returns true if the identity represents a remote node
func (id NumericIdentity) IsRemoteNode() bool {
	return id.Scope() == IdentityScopeRemoteNode
}

// String returns a human-readable representation of the identity
func (id NumericIdentity) String() string {
	switch id {
	case IdentityUnknown:
		return "unknown"
	case IdentityHost:
		return "host"
	case IdentityWorld:
		return "world"
	case IdentityUnmanaged:
		return "unmanaged"
	case IdentityHealth:
		return "health"
	case IdentityInit:
		return "init"
	case IdentityRemoteNode:
		return "remote-node"
	case IdentityKubeAPIServer:
		return "kube-apiserver"
	default:
		if id.IsReserved() {
			return fmt.Sprintf("reserved:%d", id)
		}
		scope := ""
		switch id.Scope() {
		case IdentityScopeLocal:
			scope = "local:"
		case IdentityScopeRemoteNode:
			scope = "remote:"
		}
		return fmt.Sprintf("%s%d", scope, id.Value())
	}
}

// Identity represents a security identity with associated labels
type Identity struct {
	// ID is the numeric identity
	ID NumericIdentity `json:"id" db:"id"`

	// Labels are the security-relevant labels that define this identity
	Labels map[string]string `json:"labels" db:"labels"`

	// LabelHash is the SHA256 hash of the sorted label key=value pairs
	// Used for fast lookup and deduplication
	LabelHash string `json:"label_hash" db:"label_hash"`

	// ReferenceCount tracks how many endpoints use this identity
	ReferenceCount int `json:"reference_count" db:"reference_count"`
}

// NewIdentity creates a new Identity with the given ID and labels
func NewIdentity(id NumericIdentity, labels map[string]string) *Identity {
	identity := &Identity{
		ID:             id,
		Labels:         labels,
		ReferenceCount: 0,
	}
	identity.LabelHash = ComputeLabelHash(labels)
	return identity
}

// ComputeLabelHash computes a deterministic hash of the label set
// Labels are sorted by key to ensure consistent hashing
func ComputeLabelHash(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	// Sort keys for deterministic ordering
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build canonical string representation
	var builder strings.Builder
	for i, k := range keys {
		if i > 0 {
			builder.WriteByte(';')
		}
		builder.WriteString(k)
		builder.WriteByte('=')
		builder.WriteString(labels[k])
	}

	// Compute SHA256 hash
	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

// HasLabel checks if the identity has a specific label with value
func (i *Identity) HasLabel(key, value string) bool {
	if i.Labels == nil {
		return false
	}
	v, ok := i.Labels[key]
	return ok && v == value
}

// MatchesLabels checks if all provided labels match the identity's labels
func (i *Identity) MatchesLabels(labels map[string]string) bool {
	if i.Labels == nil && len(labels) > 0 {
		return false
	}
	for k, v := range labels {
		if i.Labels[k] != v {
			return false
		}
	}
	return true
}

// Clone creates a deep copy of the identity
func (i *Identity) Clone() *Identity {
	labels := make(map[string]string, len(i.Labels))
	for k, v := range i.Labels {
		labels[k] = v
	}
	return &Identity{
		ID:             i.ID,
		Labels:         labels,
		LabelHash:      i.LabelHash,
		ReferenceCount: i.ReferenceCount,
	}
}

// String returns a human-readable representation of the identity
func (i *Identity) String() string {
	return fmt.Sprintf("Identity{ID: %s, Labels: %v}", i.ID.String(), i.Labels)
}

// IPIdentityPair represents a mapping from an IP prefix to a security identity
type IPIdentityPair struct {
	// Prefix is the IP prefix in CIDR notation (e.g., "10.0.0.1/32")
	Prefix string `json:"prefix" db:"prefix"`

	// Identity is the security identity for this prefix
	Identity NumericIdentity `json:"identity" db:"identity"`

	// Metadata contains additional information about the mapping
	Metadata IPIdentityMetadata `json:"metadata,omitempty"`
}

// IPIdentityMetadata contains additional context for an IP-identity mapping
type IPIdentityMetadata struct {
	// Source indicates where this mapping came from
	Source string `json:"source,omitempty"`

	// Namespace is the Kubernetes namespace (if applicable)
	Namespace string `json:"namespace,omitempty"`

	// PodName is the Kubernetes pod name (if applicable)
	PodName string `json:"pod_name,omitempty"`

	// HostIP is the host IP if this is a pod IP
	HostIP string `json:"host_ip,omitempty"`
}

// IdentityAllocatorEvent represents an event from the identity allocator
type IdentityAllocatorEvent struct {
	// Type is the event type
	Type IdentityEventType `json:"type"`

	// Identity is the affected identity
	Identity *Identity `json:"identity"`

	// OldIdentity is the previous identity (for updates)
	OldIdentity *Identity `json:"old_identity,omitempty"`
}

// IdentityEventType represents the type of identity event
type IdentityEventType string

const (
	// IdentityEventAdd indicates a new identity was allocated
	IdentityEventAdd IdentityEventType = "add"

	// IdentityEventUpdate indicates an identity was updated
	IdentityEventUpdate IdentityEventType = "update"

	// IdentityEventDelete indicates an identity was released
	IdentityEventDelete IdentityEventType = "delete"
)
