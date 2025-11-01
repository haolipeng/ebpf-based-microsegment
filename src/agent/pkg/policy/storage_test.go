// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSQLiteStorage_NewAndClose tests creating and closing storage
func TestSQLiteStorage_NewAndClose(t *testing.T) {
	// Create temporary database
	dbPath := "/tmp/test_policy_storage.db"
	defer os.Remove(dbPath)

	// Create storage
	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	assert.NotNil(t, storage)

	// Close storage
	err = storage.Close()
	assert.NoError(t, err)
}

// TestSQLiteStorage_SaveAndLoad tests saving and loading policies
func TestSQLiteStorage_SaveAndLoad(t *testing.T) {
	dbPath := "/tmp/test_policy_save_load.db"
	defer os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Create test policy
	policy := &Policy{
		RuleID:   1,
		SrcIP:    "192.168.1.100",
		DstIP:    "10.0.0.1",
		SrcPort:  0,
		DstPort:  80,
		Protocol: "tcp",
		Action:   "allow",
		Priority: 100,
	}

	// Save policy
	err = storage.SavePolicy(policy)
	assert.NoError(t, err)

	// Load policies
	policies, err := storage.LoadPolicies()
	assert.NoError(t, err)
	assert.Len(t, policies, 1)

	// Verify policy data
	loaded := policies[0]
	assert.Equal(t, policy.RuleID, loaded.RuleID)
	assert.Equal(t, policy.SrcIP, loaded.SrcIP)
	assert.Equal(t, policy.DstIP, loaded.DstIP)
	assert.Equal(t, policy.SrcPort, loaded.SrcPort)
	assert.Equal(t, policy.DstPort, loaded.DstPort)
	assert.Equal(t, policy.Protocol, loaded.Protocol)
	assert.Equal(t, policy.Action, loaded.Action)
	assert.Equal(t, policy.Priority, loaded.Priority)
}

// TestSQLiteStorage_SaveMultiplePolicies tests saving multiple policies
func TestSQLiteStorage_SaveMultiplePolicies(t *testing.T) {
	dbPath := "/tmp/test_policy_multiple.db"
	defer os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Create multiple policies
	policies := []*Policy{
		{
			RuleID:   1,
			SrcIP:    "192.168.1.100",
			DstIP:    "10.0.0.1",
			SrcPort:  0,
			DstPort:  80,
			Protocol: "tcp",
			Action:   "allow",
			Priority: 100,
		},
		{
			RuleID:   2,
			SrcIP:    "192.168.1.0/24",
			DstIP:    "10.0.0.2",
			SrcPort:  0,
			DstPort:  443,
			Protocol: "tcp",
			Action:   "deny",
			Priority: 200,
		},
		{
			RuleID:   3,
			SrcIP:    "0.0.0.0",
			DstIP:    "0.0.0.0",
			SrcPort:  0,
			DstPort:  0,
			Protocol: "any",
			Action:   "log",
			Priority: 50,
		},
	}

	// Save all policies
	for _, p := range policies {
		err = storage.SavePolicy(p)
		assert.NoError(t, err)
	}

	// Load and verify
	loaded, err := storage.LoadPolicies()
	assert.NoError(t, err)
	assert.Len(t, loaded, 3)

	// Verify they are sorted by priority DESC, rule_id ASC
	assert.Equal(t, uint32(2), loaded[0].RuleID) // Priority 200
	assert.Equal(t, uint32(1), loaded[1].RuleID) // Priority 100
	assert.Equal(t, uint32(3), loaded[2].RuleID) // Priority 50
}

// TestSQLiteStorage_UpdatePolicy tests updating an existing policy
func TestSQLiteStorage_UpdatePolicy(t *testing.T) {
	dbPath := "/tmp/test_policy_update.db"
	defer os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Create and save initial policy
	policy := &Policy{
		RuleID:   1,
		SrcIP:    "192.168.1.100",
		DstIP:    "10.0.0.1",
		SrcPort:  0,
		DstPort:  80,
		Protocol: "tcp",
		Action:   "allow",
		Priority: 100,
	}

	err = storage.SavePolicy(policy)
	require.NoError(t, err)

	// Update policy
	policy.Action = "deny"
	policy.Priority = 200
	err = storage.SavePolicy(policy)
	assert.NoError(t, err)

	// Load and verify update
	policies, err := storage.LoadPolicies()
	assert.NoError(t, err)
	assert.Len(t, policies, 1)
	assert.Equal(t, "deny", policies[0].Action)
	assert.Equal(t, uint16(200), policies[0].Priority)
}

// TestSQLiteStorage_DeletePolicy tests deleting a policy
func TestSQLiteStorage_DeletePolicy(t *testing.T) {
	dbPath := "/tmp/test_policy_delete.db"
	defer os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Save policies
	policy1 := &Policy{RuleID: 1, SrcIP: "192.168.1.1", DstIP: "10.0.0.1", Protocol: "tcp", Action: "allow", Priority: 100}
	policy2 := &Policy{RuleID: 2, SrcIP: "192.168.1.2", DstIP: "10.0.0.2", Protocol: "udp", Action: "deny", Priority: 200}

	storage.SavePolicy(policy1)
	storage.SavePolicy(policy2)

	// Verify 2 policies exist
	policies, err := storage.LoadPolicies()
	require.NoError(t, err)
	assert.Len(t, policies, 2)

	// Delete policy 1
	err = storage.DeletePolicy(1)
	assert.NoError(t, err)

	// Verify only 1 policy remains
	policies, err = storage.LoadPolicies()
	assert.NoError(t, err)
	assert.Len(t, policies, 1)
	assert.Equal(t, uint32(2), policies[0].RuleID)
}

// TestSQLiteStorage_DeleteNonExistent tests deleting non-existent policy
func TestSQLiteStorage_DeleteNonExistent(t *testing.T) {
	dbPath := "/tmp/test_policy_delete_nonexist.db"
	defer os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Try to delete non-existent policy
	err = storage.DeletePolicy(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy not found")
}

// TestSQLiteStorage_GetPolicyCount tests counting policies
func TestSQLiteStorage_GetPolicyCount(t *testing.T) {
	dbPath := "/tmp/test_policy_count.db"
	defer os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Initially should be 0
	count, err := storage.GetPolicyCount()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add 3 policies
	for i := 1; i <= 3; i++ {
		policy := &Policy{
			RuleID:   uint32(i),
			SrcIP:    "192.168.1.1",
			DstIP:    "10.0.0.1",
			Protocol: "tcp",
			Action:   "allow",
			Priority: 100,
		}
		storage.SavePolicy(policy)
	}

	// Should be 3
	count, err = storage.GetPolicyCount()
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

// TestSQLiteStorage_ClearAll tests clearing all policies
func TestSQLiteStorage_ClearAll(t *testing.T) {
	dbPath := "/tmp/test_policy_clear.db"
	defer os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Add policies
	for i := 1; i <= 5; i++ {
		policy := &Policy{
			RuleID:   uint32(i),
			SrcIP:    "192.168.1.1",
			DstIP:    "10.0.0.1",
			Protocol: "tcp",
			Action:   "allow",
			Priority: 100,
		}
		storage.SavePolicy(policy)
	}

	// Verify policies exist
	count, _ := storage.GetPolicyCount()
	assert.Equal(t, 5, count)

	// Clear all
	err = storage.ClearAll()
	assert.NoError(t, err)

	// Verify all cleared
	count, _ = storage.GetPolicyCount()
	assert.Equal(t, 0, count)
}

// TestSQLiteStorage_LoadEmpty tests loading from empty database
func TestSQLiteStorage_LoadEmpty(t *testing.T) {
	dbPath := "/tmp/test_policy_empty.db"
	defer os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Load from empty database
	policies, err := storage.LoadPolicies()
	assert.NoError(t, err)
	assert.Len(t, policies, 0)
}

// TestSQLiteStorage_InvalidPath tests creating storage with invalid path
func TestSQLiteStorage_InvalidPath(t *testing.T) {
	// Try to create storage in non-existent directory
	_, err := NewSQLiteStorage("/nonexistent/path/test.db")
	assert.Error(t, err)
}

// TestSQLiteStorage_ConcurrentOperations tests concurrent save operations
func TestSQLiteStorage_ConcurrentOperations(t *testing.T) {
	dbPath := "/tmp/test_policy_concurrent.db"
	defer os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Save multiple policies concurrently
	done := make(chan bool, 3)
	for i := 1; i <= 3; i++ {
		go func(id int) {
			policy := &Policy{
				RuleID:   uint32(id),
				SrcIP:    "192.168.1.1",
				DstIP:    "10.0.0.1",
				Protocol: "tcp",
				Action:   "allow",
				Priority: uint16(id * 100),
			}
			storage.SavePolicy(policy)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify all policies were saved
	count, err := storage.GetPolicyCount()
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}
