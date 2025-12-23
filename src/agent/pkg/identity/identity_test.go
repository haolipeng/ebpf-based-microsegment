// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package identity

import (
	"testing"
)

func TestComputeLabelHash(t *testing.T) {
	tests := []struct {
		name     string
		labels1  map[string]string
		labels2  map[string]string
		wantSame bool
	}{
		{
			name:     "empty labels",
			labels1:  map[string]string{},
			labels2:  map[string]string{},
			wantSame: true,
		},
		{
			name:     "same labels different order",
			labels1:  map[string]string{"app": "nginx", "env": "prod"},
			labels2:  map[string]string{"env": "prod", "app": "nginx"},
			wantSame: true,
		},
		{
			name:     "different labels",
			labels1:  map[string]string{"app": "nginx"},
			labels2:  map[string]string{"app": "apache"},
			wantSame: false,
		},
		{
			name:     "different keys",
			labels1:  map[string]string{"app": "nginx"},
			labels2:  map[string]string{"service": "nginx"},
			wantSame: false,
		},
		{
			name:     "subset labels",
			labels1:  map[string]string{"app": "nginx", "env": "prod"},
			labels2:  map[string]string{"app": "nginx"},
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := ComputeLabelHash(tt.labels1)
			hash2 := ComputeLabelHash(tt.labels2)

			if tt.wantSame && hash1 != hash2 {
				t.Errorf("Expected same hash, got hash1=%s, hash2=%s", hash1, hash2)
			}
			if !tt.wantSame && hash1 == hash2 {
				t.Errorf("Expected different hash, but both are %s", hash1)
			}
		})
	}
}

func TestNumericIdentity(t *testing.T) {
	tests := []struct {
		id         NumericIdentity
		wantStr    string
		isReserved bool
		isGlobal   bool
		isLocal    bool
	}{
		{IdentityUnknown, "unknown", true, true, false},
		{IdentityHost, "host", true, true, false},
		{IdentityWorld, "world", true, true, false},
		{IdentityHealth, "health", true, true, false},
		{NumericIdentity(256), "256", false, true, false},
		{NumericIdentity(1000), "1000", false, true, false},
		{IdentityScopeLocal | NumericIdentity(100), "local:100", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.wantStr, func(t *testing.T) {
			if got := tt.id.String(); got != tt.wantStr {
				t.Errorf("String() = %s, want %s", got, tt.wantStr)
			}
			if got := tt.id.IsReserved(); got != tt.isReserved {
				t.Errorf("IsReserved() = %v, want %v", got, tt.isReserved)
			}
			if got := tt.id.IsGlobal(); got != tt.isGlobal {
				t.Errorf("IsGlobal() = %v, want %v", got, tt.isGlobal)
			}
			if got := tt.id.IsLocal(); got != tt.isLocal {
				t.Errorf("IsLocal() = %v, want %v", got, tt.isLocal)
			}
		})
	}
}

func TestIdentity(t *testing.T) {
	labels := map[string]string{
		"app":  "nginx",
		"env":  "production",
		"tier": "frontend",
	}

	identity := NewIdentity(NumericIdentity(256), labels)

	// Check basic fields
	if identity.ID != 256 {
		t.Errorf("ID = %d, want 256", identity.ID)
	}
	if identity.LabelHash == "" {
		t.Error("LabelHash should not be empty")
	}

	// Check HasLabel
	if !identity.HasLabel("app", "nginx") {
		t.Error("HasLabel(app, nginx) should be true")
	}
	if identity.HasLabel("app", "apache") {
		t.Error("HasLabel(app, apache) should be false")
	}
	if identity.HasLabel("nonexistent", "value") {
		t.Error("HasLabel(nonexistent, value) should be false")
	}

	// Check MatchesLabels
	if !identity.MatchesLabels(map[string]string{"app": "nginx"}) {
		t.Error("MatchesLabels should match subset")
	}
	if !identity.MatchesLabels(map[string]string{"app": "nginx", "env": "production"}) {
		t.Error("MatchesLabels should match multiple labels")
	}
	if identity.MatchesLabels(map[string]string{"app": "apache"}) {
		t.Error("MatchesLabels should not match different value")
	}

	// Check Clone
	clone := identity.Clone()
	if clone.ID != identity.ID {
		t.Error("Clone should have same ID")
	}
	if clone.LabelHash != identity.LabelHash {
		t.Error("Clone should have same LabelHash")
	}
	// Modify clone labels should not affect original
	clone.Labels["new"] = "label"
	if _, ok := identity.Labels["new"]; ok {
		t.Error("Modifying clone should not affect original")
	}
}

func TestCache(t *testing.T) {
	cache := NewDefaultCache()

	// Test Upsert and GetByID
	identity := NewIdentity(NumericIdentity(256), map[string]string{"app": "nginx"})
	cache.Upsert(identity)

	got, ok := cache.GetByID(NumericIdentity(256))
	if !ok {
		t.Fatal("GetByID should find the identity")
	}
	if got.ID != 256 {
		t.Errorf("GetByID returned wrong ID: %d", got.ID)
	}

	// Test GetByLabels
	got, ok = cache.GetByLabels(map[string]string{"app": "nginx"})
	if !ok {
		t.Fatal("GetByLabels should find the identity")
	}
	if got.ID != 256 {
		t.Errorf("GetByLabels returned wrong ID: %d", got.ID)
	}

	// Test GetByLabelHash
	got, ok = cache.GetByLabelHash(identity.LabelHash)
	if !ok {
		t.Fatal("GetByLabelHash should find the identity")
	}
	if got.ID != 256 {
		t.Errorf("GetByLabelHash returned wrong ID: %d", got.ID)
	}

	// Test Size
	if cache.Size() != 1 {
		t.Errorf("Size() = %d, want 1", cache.Size())
	}

	// Test GetAll
	all := cache.GetAll()
	if len(all) != 1 {
		t.Errorf("GetAll() returned %d identities, want 1", len(all))
	}

	// Test Update
	updatedIdentity := NewIdentity(NumericIdentity(256), map[string]string{"app": "nginx", "env": "prod"})
	cache.Upsert(updatedIdentity)

	got, _ = cache.GetByID(NumericIdentity(256))
	if _, ok := got.Labels["env"]; !ok {
		t.Error("Update should add new label")
	}

	// Test Delete
	deleted := cache.Delete(NumericIdentity(256))
	if !deleted {
		t.Error("Delete should return true")
	}
	if cache.Size() != 0 {
		t.Errorf("Size after delete = %d, want 0", cache.Size())
	}

	_, ok = cache.GetByID(NumericIdentity(256))
	if ok {
		t.Error("GetByID should not find deleted identity")
	}

	// Test Clear
	cache.Upsert(NewIdentity(NumericIdentity(256), map[string]string{"app": "nginx"}))
	cache.Upsert(NewIdentity(NumericIdentity(257), map[string]string{"app": "apache"}))
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Size after Clear = %d, want 0", cache.Size())
	}
}

func TestCacheReservedIdentities(t *testing.T) {
	cache := NewDefaultCache()
	cache.UpsertReservedIdentities()

	// Check that reserved identities are present
	reservedIDs := []NumericIdentity{
		IdentityHost,
		IdentityWorld,
		IdentityUnmanaged,
		IdentityHealth,
		IdentityInit,
		IdentityRemoteNode,
		IdentityKubeAPIServer,
	}

	for _, id := range reservedIDs {
		got, ok := cache.GetByID(id)
		if !ok {
			t.Errorf("Reserved identity %d not found", id)
			continue
		}
		if !got.ID.IsReserved() {
			t.Errorf("Identity %d should be reserved", id)
		}
	}
}

func TestCacheEviction(t *testing.T) {
	// Create a small cache
	config := CacheConfig{
		MaxSize:         3,
		TTL:             0,
		CleanupInterval: 0,
	}
	cache := NewCache(config)

	// Add 4 identities (one more than max)
	for i := 256; i < 260; i++ {
		cache.Upsert(NewIdentity(NumericIdentity(i), map[string]string{"id": string(rune(i))}))
	}

	// Should have 3 identities (oldest evicted)
	if cache.Size() != 3 {
		t.Errorf("Size = %d, want 3 after eviction", cache.Size())
	}

	// The oldest (256) should be evicted
	_, ok := cache.GetByID(NumericIdentity(256))
	if ok {
		t.Error("Oldest identity should be evicted")
	}
}
