// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
//
// input: workload label sets
// output: numeric identity IDs, label hashes, identity allocation results
// pos: identity - security identity management and allocation

package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

// NumericIdentity represents a security identity as a 32-bit unsigned integer
// (Duplicated from agent for now - should be moved to a shared package)
type NumericIdentity uint32

// Reserved identity constants
const (
	IdentityUnknown       NumericIdentity = 0
	IdentityHost          NumericIdentity = 1
	IdentityWorld         NumericIdentity = 2
	IdentityUnmanaged     NumericIdentity = 3
	IdentityHealth        NumericIdentity = 4
	IdentityInit          NumericIdentity = 5
	IdentityRemoteNode    NumericIdentity = 6
	IdentityKubeAPIServer NumericIdentity = 7
	ReservedIdentityMax   NumericIdentity = 255
	MinAllocatedIdentity  NumericIdentity = 256
	MaxAllocatedIdentity  NumericIdentity = 0x00FFFFFF
)

// Identity represents a security identity with associated labels
type Identity struct {
	ID             NumericIdentity   `json:"id" db:"id"`
	Labels         map[string]string `json:"labels" db:"labels"`
	LabelHash      string            `json:"label_hash" db:"label_hash"`
	ReferenceCount int               `json:"reference_count" db:"reference_count"`
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
func ComputeLabelHash(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for i, k := range keys {
		if i > 0 {
			builder.WriteByte(';')
		}
		builder.WriteString(k)
		builder.WriteByte('=')
		builder.WriteString(labels[k])
	}

	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

// IdentityStorage defines the interface for identity persistence
type IdentityStorage interface {
	// Create stores a new identity
	Create(identity *Identity) error

	// Get retrieves an identity by ID
	Get(id NumericIdentity) (*Identity, error)

	// GetByLabelHash retrieves an identity by label hash
	GetByLabelHash(hash string) (*Identity, error)

	// Update updates an existing identity
	Update(identity *Identity) error

	// Delete removes an identity
	Delete(id NumericIdentity) error

	// List returns all identities
	List() ([]*Identity, error)

	// GetNextID returns the next available identity ID
	GetNextID() (NumericIdentity, error)

	// SetNextID sets the next available identity ID
	SetNextID(id NumericIdentity) error
}

// Allocator manages the allocation and lifecycle of security identities
type Allocator struct {
	mu      sync.Mutex
	nextID  uint32 // Atomic counter for ID allocation
	storage IdentityStorage

	// In-memory cache for fast lookups
	cacheMu sync.RWMutex
	byID    map[NumericIdentity]*Identity
	byHash  map[string]*Identity

	// Event listeners
	listeners []func(event AllocatorEvent)
}

// AllocatorEvent represents an event from the allocator
type AllocatorEvent struct {
	Type     EventType
	Identity *Identity
}

// EventType represents the type of allocator event
type EventType string

const (
	EventTypeAllocate EventType = "allocate"
	EventTypeRelease  EventType = "release"
	EventTypeUpdate   EventType = "update"
)

// AllocatorConfig holds configuration for the identity allocator
type AllocatorConfig struct {
	// StartID is the first identity ID to allocate
	StartID NumericIdentity

	// Storage is the persistence backend
	Storage IdentityStorage
}

// NewAllocator creates a new identity allocator
func NewAllocator(config AllocatorConfig) (*Allocator, error) {
	startID := config.StartID
	if startID < MinAllocatedIdentity {
		startID = MinAllocatedIdentity
	}

	a := &Allocator{
		nextID:    uint32(startID),
		storage:   config.Storage,
		byID:      make(map[NumericIdentity]*Identity),
		byHash:    make(map[string]*Identity),
		listeners: make([]func(event AllocatorEvent), 0),
	}

	// Load existing identities from storage
	if config.Storage != nil {
		if err := a.loadFromStorage(); err != nil {
			return nil, fmt.Errorf("failed to load identities from storage: %w", err)
		}
	}

	return a, nil
}

// loadFromStorage loads all identities from storage into the cache
func (a *Allocator) loadFromStorage() error {
	if a.storage == nil {
		return nil
	}

	identities, err := a.storage.List()
	if err != nil {
		return err
	}

	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()

	maxID := uint32(MinAllocatedIdentity)
	for _, identity := range identities {
		a.byID[identity.ID] = identity
		if identity.LabelHash != "" {
			a.byHash[identity.LabelHash] = identity
		}
		if uint32(identity.ID) >= maxID {
			maxID = uint32(identity.ID) + 1
		}
	}

	// Update nextID to be after the highest allocated ID
	atomic.StoreUint32(&a.nextID, maxID)

	log.WithFields(log.Fields{
		"count":  len(identities),
		"nextID": maxID,
	}).Info("Loaded identities from storage")

	return nil
}

// AllocateIdentity allocates a new identity for the given label set
// If an identity with the same labels already exists, it returns the existing one
// Returns: (identity, isNew, error)
func (a *Allocator) AllocateIdentity(labels map[string]string) (*Identity, bool, error) {
	if len(labels) == 0 {
		return nil, false, fmt.Errorf("labels cannot be empty")
	}

	hash := ComputeLabelHash(labels)

	// Check cache first
	a.cacheMu.RLock()
	if existing, ok := a.byHash[hash]; ok {
		a.cacheMu.RUnlock()
		// Identity already exists - increment reference count
		a.mu.Lock()
		existing.ReferenceCount++
		if a.storage != nil {
			if err := a.storage.Update(existing); err != nil {
				log.WithError(err).Warn("Failed to update reference count in storage")
			}
		}
		a.mu.Unlock()
		return existing, false, nil
	}
	a.cacheMu.RUnlock()

	// Allocate new identity
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring lock
	a.cacheMu.RLock()
	if existing, ok := a.byHash[hash]; ok {
		a.cacheMu.RUnlock()
		existing.ReferenceCount++
		if a.storage != nil {
			a.storage.Update(existing)
		}
		return existing, false, nil
	}
	a.cacheMu.RUnlock()

	// Allocate new ID
	newID := NumericIdentity(atomic.AddUint32(&a.nextID, 1) - 1)
	if newID > MaxAllocatedIdentity {
		return nil, false, fmt.Errorf("identity ID space exhausted")
	}

	identity := &Identity{
		ID:             newID,
		Labels:         labels,
		LabelHash:      hash,
		ReferenceCount: 1,
	}

	// Store in persistence
	if a.storage != nil {
		if err := a.storage.Create(identity); err != nil {
			return nil, false, fmt.Errorf("failed to store identity: %w", err)
		}
	}

	// Update cache
	a.cacheMu.Lock()
	a.byID[newID] = identity
	a.byHash[hash] = identity
	a.cacheMu.Unlock()

	// Notify listeners
	a.notifyListeners(AllocatorEvent{
		Type:     EventTypeAllocate,
		Identity: identity,
	})

	log.WithFields(log.Fields{
		"id":     newID,
		"labels": labels,
	}).Info("Allocated new identity")

	return identity, true, nil
}

// ReleaseIdentity decrements the reference count and potentially removes the identity
func (a *Allocator) ReleaseIdentity(id NumericIdentity) error {
	if id.IsReserved() {
		return fmt.Errorf("cannot release reserved identity %d", id)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.cacheMu.RLock()
	identity, ok := a.byID[id]
	a.cacheMu.RUnlock()

	if !ok {
		return fmt.Errorf("identity %d not found", id)
	}

	identity.ReferenceCount--

	if identity.ReferenceCount <= 0 {
		// Remove identity
		a.cacheMu.Lock()
		delete(a.byID, id)
		delete(a.byHash, identity.LabelHash)
		a.cacheMu.Unlock()

		if a.storage != nil {
			if err := a.storage.Delete(id); err != nil {
				log.WithError(err).Warn("Failed to delete identity from storage")
			}
		}

		a.notifyListeners(AllocatorEvent{
			Type:     EventTypeRelease,
			Identity: identity,
		})

		log.WithField("id", id).Info("Released identity (reference count reached 0)")
	} else {
		// Update reference count
		if a.storage != nil {
			if err := a.storage.Update(identity); err != nil {
				log.WithError(err).Warn("Failed to update reference count in storage")
			}
		}
	}

	return nil
}

// GetIdentity retrieves an identity by ID
func (a *Allocator) GetIdentity(id NumericIdentity) (*Identity, error) {
	// Handle reserved identities
	if id.IsReserved() {
		return a.getReservedIdentity(id), nil
	}

	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()

	identity, ok := a.byID[id]
	if !ok {
		return nil, fmt.Errorf("identity %d not found", id)
	}

	return identity, nil
}

// GetIdentityByLabels retrieves an identity by its label set
func (a *Allocator) GetIdentityByLabels(labels map[string]string) (*Identity, error) {
	hash := ComputeLabelHash(labels)

	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()

	identity, ok := a.byHash[hash]
	if !ok {
		return nil, fmt.Errorf("identity with labels %v not found", labels)
	}

	return identity, nil
}

// ListIdentities returns all allocated identities
func (a *Allocator) ListIdentities() []*Identity {
	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()

	identities := make([]*Identity, 0, len(a.byID))
	for _, identity := range a.byID {
		identities = append(identities, identity)
	}

	return identities
}

// AddListener adds a listener for allocator events
func (a *Allocator) AddListener(listener func(event AllocatorEvent)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.listeners = append(a.listeners, listener)
}

// notifyListeners notifies all listeners of an event
func (a *Allocator) notifyListeners(event AllocatorEvent) {
	for _, listener := range a.listeners {
		go listener(event)
	}
}

// getReservedIdentity returns the identity for a reserved ID
func (a *Allocator) getReservedIdentity(id NumericIdentity) *Identity {
	var labels map[string]string

	switch id {
	case IdentityHost:
		labels = map[string]string{"reserved": "host"}
	case IdentityWorld:
		labels = map[string]string{"reserved": "world"}
	case IdentityUnmanaged:
		labels = map[string]string{"reserved": "unmanaged"}
	case IdentityHealth:
		labels = map[string]string{"reserved": "health"}
	case IdentityInit:
		labels = map[string]string{"reserved": "init"}
	case IdentityRemoteNode:
		labels = map[string]string{"reserved": "remote-node"}
	case IdentityKubeAPIServer:
		labels = map[string]string{"reserved": "kube-apiserver"}
	default:
		labels = map[string]string{"reserved": fmt.Sprintf("%d", id)}
	}

	return NewIdentity(id, labels)
}

// IsReserved checks if an identity ID is reserved
func (id NumericIdentity) IsReserved() bool {
	return id <= ReservedIdentityMax
}

// String returns a string representation of the identity
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
		return fmt.Sprintf("%d", id)
	}
}
