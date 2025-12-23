// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: IP addresses, identity IDs
// output: IP-to-identity mapping cache
// pos: IP cache (IP to identity mapping) - if file updated, must sync with this header comment and pkg/ipcache/CLAUDE.md
package ipcache

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/identity"
	log "github.com/sirupsen/logrus"
)

// IPCacheKey represents the key structure for the IPCache BPF map
// Uses LPM (Longest Prefix Match) trie for efficient CIDR matching
type IPCacheKey struct {
	PrefixLen uint32    // Number of bits in the prefix
	IP        [16]byte  // IPv4/IPv6 address (IPv4 stored as IPv4-mapped IPv6)
}

// IPCacheValue represents the value structure for the IPCache BPF map
type IPCacheValue struct {
	Identity uint32    // Security identity
	Pad      [4]byte   // Padding for alignment
}

// IPCacheEntry represents a cached IP-to-identity mapping
type IPCacheEntry struct {
	Prefix   netip.Prefix
	Identity identity.NumericIdentity
	Metadata identity.IPIdentityMetadata
}

// IPCacheListener is called when IPCache entries change
type IPCacheListener interface {
	OnIPIdentityChange(prefix netip.Prefix, oldID, newID identity.NumericIdentity)
}

// IPCache manages IP-to-identity mappings and syncs them to BPF maps
type IPCache struct {
	mu        sync.RWMutex
	entries   map[string]*IPCacheEntry // prefix string -> entry
	bpfMap    *ebpf.Map
	listeners []IPCacheListener
}

// NewIPCache creates a new IPCache instance
func NewIPCache() *IPCache {
	return &IPCache{
		entries:   make(map[string]*IPCacheEntry),
		listeners: make([]IPCacheListener, 0),
	}
}

// SetBPFMap sets the BPF map for syncing IPCache entries
func (c *IPCache) SetBPFMap(m *ebpf.Map) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bpfMap = m
}

// AddListener adds a listener for IPCache changes
func (c *IPCache) AddListener(listener IPCacheListener) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, listener)
}

// Upsert adds or updates an IP-to-identity mapping
func (c *IPCache) Upsert(prefix netip.Prefix, id identity.NumericIdentity, metadata identity.IPIdentityMetadata) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := prefix.String()
	oldEntry := c.entries[key]
	oldID := identity.IdentityUnknown
	if oldEntry != nil {
		oldID = oldEntry.Identity
	}

	// Update local cache
	entry := &IPCacheEntry{
		Prefix:   prefix,
		Identity: id,
		Metadata: metadata,
	}
	c.entries[key] = entry

	// Sync to BPF map
	if c.bpfMap != nil {
		if err := c.syncToBPF(prefix, id); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"prefix":   prefix,
				"identity": id,
			}).Error("Failed to sync IPCache entry to BPF map")
			return err
		}
	}

	// Notify listeners
	if oldID != id {
		c.notifyListeners(prefix, oldID, id)
	}

	log.WithFields(log.Fields{
		"prefix":   prefix,
		"identity": id,
		"source":   metadata.Source,
	}).Debug("Upserted IPCache entry")

	return nil
}

// Delete removes an IP-to-identity mapping
func (c *IPCache) Delete(prefix netip.Prefix) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := prefix.String()
	oldEntry := c.entries[key]
	if oldEntry == nil {
		return nil // Nothing to delete
	}

	oldID := oldEntry.Identity
	delete(c.entries, key)

	// Delete from BPF map
	if c.bpfMap != nil {
		if err := c.deleteFromBPF(prefix); err != nil {
			log.WithError(err).WithField("prefix", prefix).Error("Failed to delete IPCache entry from BPF map")
			return err
		}
	}

	// Notify listeners
	c.notifyListeners(prefix, oldID, identity.IdentityUnknown)

	log.WithField("prefix", prefix).Debug("Deleted IPCache entry")

	return nil
}

// Lookup finds the identity for an IP address
func (c *IPCache) Lookup(addr netip.Addr) (identity.NumericIdentity, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Try exact match first (/32 for IPv4, /128 for IPv6)
	var bits int
	if addr.Is4() {
		bits = 32
	} else {
		bits = 128
	}
	prefix := netip.PrefixFrom(addr, bits)
	if entry, ok := c.entries[prefix.String()]; ok {
		return entry.Identity, true
	}

	// Try progressively shorter prefixes (LPM simulation)
	for bits > 0 {
		bits--
		prefix, _ = addr.Prefix(bits)
		if entry, ok := c.entries[prefix.String()]; ok {
			return entry.Identity, true
		}
	}

	return identity.IdentityUnknown, false
}

// LookupByIP finds the identity for an IP address string
func (c *IPCache) LookupByIP(ipStr string) (identity.NumericIdentity, bool) {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return identity.IdentityUnknown, false
	}
	return c.Lookup(addr)
}

// GetAll returns all entries in the IPCache
func (c *IPCache) GetAll() []*IPCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := make([]*IPCacheEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		entries = append(entries, &IPCacheEntry{
			Prefix:   entry.Prefix,
			Identity: entry.Identity,
			Metadata: entry.Metadata,
		})
	}
	return entries
}

// Size returns the number of entries in the IPCache
func (c *IPCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Clear removes all entries from the IPCache
func (c *IPCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Delete all entries from BPF map
	if c.bpfMap != nil {
		for _, entry := range c.entries {
			if err := c.deleteFromBPF(entry.Prefix); err != nil {
				log.WithError(err).WithField("prefix", entry.Prefix).Warn("Failed to delete IPCache entry from BPF map during clear")
			}
		}
	}

	c.entries = make(map[string]*IPCacheEntry)
	log.Debug("Cleared IPCache")
	return nil
}

// syncToBPF syncs an entry to the BPF map
func (c *IPCache) syncToBPF(prefix netip.Prefix, id identity.NumericIdentity) error {
	if c.bpfMap == nil {
		return nil
	}

	key := prefixToBPFKey(prefix)
	value := IPCacheValue{
		Identity: uint32(id),
	}

	if err := c.bpfMap.Put(&key, &value); err != nil {
		return fmt.Errorf("failed to put IPCache entry: %w", err)
	}

	return nil
}

// deleteFromBPF removes an entry from the BPF map
func (c *IPCache) deleteFromBPF(prefix netip.Prefix) error {
	if c.bpfMap == nil {
		return nil
	}

	key := prefixToBPFKey(prefix)

	if err := c.bpfMap.Delete(&key); err != nil {
		// Ignore "not found" errors
		if err.Error() != "key does not exist" {
			return fmt.Errorf("failed to delete IPCache entry: %w", err)
		}
	}

	return nil
}

// notifyListeners notifies all listeners of an IPCache change
// Must be called with lock held (but listeners are called async)
func (c *IPCache) notifyListeners(prefix netip.Prefix, oldID, newID identity.NumericIdentity) {
	for _, listener := range c.listeners {
		go listener.OnIPIdentityChange(prefix, oldID, newID)
	}
}

// prefixToBPFKey converts a netip.Prefix to a BPF LPM trie key
func prefixToBPFKey(prefix netip.Prefix) IPCacheKey {
	key := IPCacheKey{}

	addr := prefix.Addr()
	if addr.Is4() {
		// IPv4: Store as IPv4-mapped IPv6 (::ffff:a.b.c.d)
		// Prefix length is adjusted: /32 IPv4 = /128 IPv6
		key.PrefixLen = uint32(prefix.Bits()) + 96
		ip4 := addr.As4()
		// Store in IPv4-mapped IPv6 format
		copy(key.IP[0:10], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		key.IP[10] = 0xff
		key.IP[11] = 0xff
		copy(key.IP[12:16], ip4[:])
	} else {
		// IPv6: Store directly
		key.PrefixLen = uint32(prefix.Bits())
		ip6 := addr.As16()
		copy(key.IP[:], ip6[:])
	}

	return key
}

// BPFKeyToPrefix converts a BPF LPM trie key to a netip.Prefix
func BPFKeyToPrefix(key *IPCacheKey) (netip.Prefix, error) {
	// Check if it's IPv4-mapped IPv6
	isIPv4Mapped := true
	for i := 0; i < 10; i++ {
		if key.IP[i] != 0 {
			isIPv4Mapped = false
			break
		}
	}
	if isIPv4Mapped && key.IP[10] == 0xff && key.IP[11] == 0xff {
		// IPv4
		var ip4 [4]byte
		copy(ip4[:], key.IP[12:16])
		addr := netip.AddrFrom4(ip4)
		bits := int(key.PrefixLen) - 96
		if bits < 0 || bits > 32 {
			return netip.Prefix{}, fmt.Errorf("invalid IPv4 prefix length: %d", key.PrefixLen)
		}
		return netip.PrefixFrom(addr, bits), nil
	}

	// IPv6
	var ip6 [16]byte
	copy(ip6[:], key.IP[:])
	addr := netip.AddrFrom16(ip6)
	bits := int(key.PrefixLen)
	if bits < 0 || bits > 128 {
		return netip.Prefix{}, fmt.Errorf("invalid IPv6 prefix length: %d", key.PrefixLen)
	}
	return netip.PrefixFrom(addr, bits), nil
}

// ParseIPOrCIDR parses an IP address or CIDR notation to a netip.Prefix
func ParseIPOrCIDR(s string) (netip.Prefix, error) {
	// Try parsing as CIDR first
	prefix, err := netip.ParsePrefix(s)
	if err == nil {
		return prefix, nil
	}

	// Try parsing as IP address
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("invalid IP or CIDR: %s", s)
	}

	// Create /32 or /128 prefix
	var bits int
	if addr.Is4() {
		bits = 32
	} else {
		bits = 128
	}
	return netip.PrefixFrom(addr, bits), nil
}

// SyncAllToBPF syncs all entries to the BPF map
// Useful for initialization or recovery
func (c *IPCache) SyncAllToBPF() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.bpfMap == nil {
		return fmt.Errorf("BPF map not set")
	}

	var lastErr error
	for _, entry := range c.entries {
		if err := c.syncToBPF(entry.Prefix, entry.Identity); err != nil {
			log.WithError(err).WithField("prefix", entry.Prefix).Error("Failed to sync IPCache entry")
			lastErr = err
		}
	}

	log.WithField("count", len(c.entries)).Info("Synced IPCache entries to BPF map")
	return lastErr
}

// ipToBytes converts a net.IP to a 16-byte array (IPv4-mapped if necessary)
func ipToBytes(ip net.IP) [16]byte {
	var result [16]byte

	ip4 := ip.To4()
	if ip4 != nil {
		// IPv4: Store as IPv4-mapped IPv6
		result[10] = 0xff
		result[11] = 0xff
		copy(result[12:16], ip4)
	} else {
		// IPv6
		ip16 := ip.To16()
		if ip16 != nil {
			copy(result[:], ip16)
		}
	}

	return result
}

// bytesToIP converts a 16-byte array to a net.IP
func bytesToIP(b [16]byte) net.IP {
	// Check if it's IPv4-mapped
	isIPv4Mapped := true
	for i := 0; i < 10; i++ {
		if b[i] != 0 {
			isIPv4Mapped = false
			break
		}
	}
	if isIPv4Mapped && b[10] == 0xff && b[11] == 0xff {
		return net.IPv4(b[12], b[13], b[14], b[15])
	}
	return net.IP(b[:])
}

// uint32ToIPv4 converts a uint32 in little-endian to net.IP
func uint32ToIPv4(ip uint32) net.IP {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, ip)
	return net.IPv4(buf[0], buf[1], buf[2], buf[3])
}
