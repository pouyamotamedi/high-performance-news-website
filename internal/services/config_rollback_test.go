package services

import (
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

func TestConfigService_SnapshotAndRollback(t *testing.T) {
	service := setupConfigServiceTest()
	
	// Get initial configuration values
	initialSiteName, err := service.Get("site_name")
	if err != nil {
		t.Fatalf("Failed to get initial site_name: %v", err)
	}
	
	initialCacheTTL, err := service.Get("cache_ttl")
	if err != nil {
		t.Fatalf("Failed to get initial cache_ttl: %v", err)
	}
	
	// Create a snapshot of current configuration
	snapshot, err := service.CreateSnapshot("test_rollback", "Snapshot for rollback testing", 1)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}
	
	// Verify snapshot contains current configuration
	if len(snapshot.Config) == 0 {
		t.Error("Snapshot should contain configurations")
	}
	
	// Check that snapshot contains our test configurations
	if siteNameConfig, exists := snapshot.Config["site_name"]; !exists {
		t.Error("Snapshot should contain site_name configuration")
	} else if siteNameConfig.Value != initialSiteName {
		t.Errorf("Snapshot site_name should be %s, got %s", initialSiteName, siteNameConfig.Value)
	}
	
	// Modify some configurations
	newSiteName := "Modified Site Name"
	newCacheTTL := 7200
	
	err = service.Set("site_name", newSiteName)
	if err != nil {
		t.Fatalf("Failed to update site_name: %v", err)
	}
	
	err = service.Set("cache_ttl", newCacheTTL)
	if err != nil {
		t.Fatalf("Failed to update cache_ttl: %v", err)
	}
	
	// Verify configurations were changed
	currentSiteName, err := service.Get("site_name")
	if err != nil {
		t.Fatalf("Failed to get current site_name: %v", err)
	}
	if currentSiteName != newSiteName {
		t.Errorf("Expected site_name to be %s, got %s", newSiteName, currentSiteName)
	}
	
	currentCacheTTL, err := service.GetTyped("cache_ttl")
	if err != nil {
		t.Fatalf("Failed to get current cache_ttl: %v", err)
	}
	if currentCacheTTL != newCacheTTL {
		t.Errorf("Expected cache_ttl to be %d, got %v", newCacheTTL, currentCacheTTL)
	}
	
	// Restore from snapshot
	err = service.RestoreSnapshot(snapshot.ID, 1)
	if err != nil {
		t.Fatalf("Failed to restore snapshot: %v", err)
	}
	
	// Verify configurations were restored
	restoredSiteName, err := service.Get("site_name")
	if err != nil {
		t.Fatalf("Failed to get restored site_name: %v", err)
	}
	if restoredSiteName != initialSiteName {
		t.Errorf("Expected restored site_name to be %s, got %s", initialSiteName, restoredSiteName)
	}
	
	restoredCacheTTL, err := service.Get("cache_ttl")
	if err != nil {
		t.Fatalf("Failed to get restored cache_ttl: %v", err)
	}
	if restoredCacheTTL != initialCacheTTL {
		t.Errorf("Expected restored cache_ttl to be %s, got %s", initialCacheTTL, restoredCacheTTL)
	}
}

func TestConfigService_MultipleSnapshots(t *testing.T) {
	service := setupConfigServiceTest()
	
	// Create first snapshot
	snapshot1, err := service.CreateSnapshot("snapshot_1", "First snapshot", 1)
	if err != nil {
		t.Fatalf("Failed to create first snapshot: %v", err)
	}
	
	// Modify configuration
	err = service.Set("site_name", "First Modification")
	if err != nil {
		t.Fatalf("Failed to update site_name: %v", err)
	}
	
	// Create second snapshot
	snapshot2, err := service.CreateSnapshot("snapshot_2", "Second snapshot", 1)
	if err != nil {
		t.Fatalf("Failed to create second snapshot: %v", err)
	}
	
	// Modify configuration again
	err = service.Set("site_name", "Second Modification")
	if err != nil {
		t.Fatalf("Failed to update site_name again: %v", err)
	}
	
	// Restore to second snapshot
	err = service.RestoreSnapshot(snapshot2.ID, 1)
	if err != nil {
		t.Fatalf("Failed to restore to second snapshot: %v", err)
	}
	
	// Verify we're at the second snapshot state
	siteName, err := service.Get("site_name")
	if err != nil {
		t.Fatalf("Failed to get site_name: %v", err)
	}
	if siteName != "First Modification" {
		t.Errorf("Expected site_name to be 'First Modification', got %s", siteName)
	}
	
	// Restore to first snapshot
	err = service.RestoreSnapshot(snapshot1.ID, 1)
	if err != nil {
		t.Fatalf("Failed to restore to first snapshot: %v", err)
	}
	
	// Verify we're at the first snapshot state
	siteName, err = service.Get("site_name")
	if err != nil {
		t.Fatalf("Failed to get site_name: %v", err)
	}
	if siteName != "High Performance News Website" {
		t.Errorf("Expected site_name to be 'High Performance News Website', got %s", siteName)
	}
}

func TestConfigService_SnapshotMetadata(t *testing.T) {
	service := setupConfigServiceTest()
	
	// Create snapshot with metadata
	name := "test_metadata_snapshot"
	description := "Testing snapshot metadata functionality"
	createdBy := uint64(123)
	
	snapshot, err := service.CreateSnapshot(name, description, createdBy)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}
	
	// Verify metadata
	if snapshot.Name != name {
		t.Errorf("Expected snapshot name to be %s, got %s", name, snapshot.Name)
	}
	
	if snapshot.Description != description {
		t.Errorf("Expected snapshot description to be %s, got %s", description, snapshot.Description)
	}
	
	if snapshot.CreatedBy != createdBy {
		t.Errorf("Expected snapshot created_by to be %d, got %d", createdBy, snapshot.CreatedBy)
	}
	
	// Verify timestamp is recent
	if time.Since(snapshot.CreatedAt) > time.Minute {
		t.Error("Snapshot creation time should be recent")
	}
}

func TestConfigService_PartialRestore(t *testing.T) {
	service := setupConfigServiceTest()
	
	// Get initial values
	initialSiteName, _ := service.Get("site_name")
	initialCacheTTL, _ := service.Get("cache_ttl")
	
	// Create snapshot
	snapshot, err := service.CreateSnapshot("partial_test", "Partial restore test", 1)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}
	
	// Modify multiple configurations
	err = service.Set("site_name", "Modified Name")
	if err != nil {
		t.Fatalf("Failed to update site_name: %v", err)
	}
	
	err = service.Set("cache_ttl", 9999)
	if err != nil {
		t.Fatalf("Failed to update cache_ttl: %v", err)
	}
	
	// Add a new configuration that wasn't in the snapshot
	newConfig := &models.Configuration{
		Key:         "new_config",
		Value:       "new_value",
		Type:        models.ConfigTypeString,
		Category:    "test",
		Description: "New configuration added after snapshot",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	service.config["new_config"] = newConfig
	
	// Restore from snapshot
	err = service.RestoreSnapshot(snapshot.ID, 1)
	if err != nil {
		t.Fatalf("Failed to restore snapshot: %v", err)
	}
	
	// Verify original configurations were restored
	restoredSiteName, _ := service.Get("site_name")
	if restoredSiteName != initialSiteName {
		t.Errorf("Expected site_name to be restored to %s, got %s", initialSiteName, restoredSiteName)
	}
	
	restoredCacheTTL, _ := service.Get("cache_ttl")
	if restoredCacheTTL != initialCacheTTL {
		t.Errorf("Expected cache_ttl to be restored to %s, got %s", initialCacheTTL, restoredCacheTTL)
	}
	
	// Verify new configuration still exists (wasn't in snapshot)
	_, err = service.Get("new_config")
	if err != nil {
		t.Error("New configuration should still exist after partial restore")
	}
}

func TestConfigService_EmptySnapshot(t *testing.T) {
	service := setupConfigServiceTest()
	
	// Clear all configurations for testing
	service.config = make(map[string]*models.Configuration)
	
	// Create snapshot of empty configuration
	snapshot, err := service.CreateSnapshot("empty_snapshot", "Empty configuration snapshot", 1)
	if err != nil {
		t.Fatalf("Failed to create empty snapshot: %v", err)
	}
	
	// Verify snapshot is empty
	if len(snapshot.Config) != 0 {
		t.Errorf("Expected empty snapshot to have 0 configurations, got %d", len(snapshot.Config))
	}
	
	// Add some configurations
	testConfig := &models.Configuration{
		Key:         "test_config",
		Value:       "test_value",
		Type:        models.ConfigTypeString,
		Category:    "test",
		Description: "Test configuration",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	service.config["test_config"] = testConfig
	
	// Restore empty snapshot
	err = service.RestoreSnapshot(snapshot.ID, 1)
	if err != nil {
		t.Fatalf("Failed to restore empty snapshot: %v", err)
	}
	
	// Verify configuration was removed (or at least not accessible)
	_, err = service.Get("test_config")
	if err == nil {
		t.Error("Expected configuration to be removed after restoring empty snapshot")
	}
}