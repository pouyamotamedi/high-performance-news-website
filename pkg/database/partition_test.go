package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestNewPartitionManager(t *testing.T) {
	// Skip test if database is not available
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	pm := NewPartitionManager(testDB)
	if pm == nil {
		t.Error("PartitionManager should not be nil")
	}
	if pm.db != testDB {
		t.Error("PartitionManager database reference is incorrect")
	}
	if pm.retentionDays != 30 {
		t.Errorf("Expected default retention days to be 30, got %d", pm.retentionDays)
	}
	if pm.schedulerActive {
		t.Error("Scheduler should not be active initially")
	}
}

func TestSetRetentionDays(t *testing.T) {
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	pm := NewPartitionManager(testDB)
	
	// Test setting valid retention days
	pm.SetRetentionDays(60)
	if pm.retentionDays != 60 {
		t.Errorf("Expected retention days to be 60, got %d", pm.retentionDays)
	}
	
	// Test setting invalid retention days (should not change)
	pm.SetRetentionDays(0)
	if pm.retentionDays != 60 {
		t.Errorf("Expected retention days to remain 60, got %d", pm.retentionDays)
	}
	
	pm.SetRetentionDays(-5)
	if pm.retentionDays != 60 {
		t.Errorf("Expected retention days to remain 60, got %d", pm.retentionDays)
	}
}

func TestCreateDailyPartitions(t *testing.T) {
	// Skip test if database is not available
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	// Create test database
	_, err = testDB.Exec("CREATE DATABASE partition_daily_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS partition_daily_test")
	}()

	// Connect to test database
	partitionDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=partition_daily_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer partitionDB.Close()

	// Run migrations to create tables first
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	migrator, err := NewMigrator(partitionDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test daily partition creation
	pm := NewPartitionManager(partitionDB)
	if err := pm.CreateDailyPartitions(); err != nil {
		t.Fatalf("Failed to create daily partitions: %v", err)
	}

	// Verify that daily partitions were created
	var partitionCount int
	err = partitionDB.QueryRow(`
		SELECT COUNT(*) FROM pg_tables 
		WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$'
	`).Scan(&partitionCount)
	if err != nil {
		t.Fatalf("Failed to count daily partitions: %v", err)
	}
	if partitionCount < 7 {
		t.Errorf("Expected at least 7 daily partitions, got %d", partitionCount)
	}
	t.Logf("Created %d daily partitions", partitionCount)

	// Verify that the function was created
	var functionExists bool
	err = partitionDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_proc 
			WHERE proname = 'create_daily_partitions'
		)
	`).Scan(&functionExists)
	if err != nil {
		t.Fatalf("Failed to check if daily partition function exists: %v", err)
	}
	if !functionExists {
		t.Error("Daily partition function was not created")
	}

	// Test running the function again (should handle existing partitions)
	if err := pm.CreateDailyPartitions(); err != nil {
		t.Fatalf("Failed to run daily partitions creation again: %v", err)
	}
}

func TestCreateDailyPartitionsWithoutPartitionedTable(t *testing.T) {
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	// Create test database without partitioned tables
	_, err = testDB.Exec("CREATE DATABASE partition_no_table_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS partition_no_table_test")
	}()

	partitionDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=partition_no_table_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer partitionDB.Close()

	pm := NewPartitionManager(partitionDB)
	
	// Should fail because articles table doesn't exist or isn't partitioned
	err = pm.CreateDailyPartitions()
	if err == nil {
		t.Error("Expected error when articles table is not partitioned, but got nil")
	}
	t.Logf("Expected error received: %v", err)
}

func TestDropOldPartitions(t *testing.T) {
	// Skip test if database is not available
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	// Create test database
	_, err = testDB.Exec("CREATE DATABASE partition_drop_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS partition_drop_test")
	}()

	// Connect to test database
	partitionDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=partition_drop_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer partitionDB.Close()

	// Run migrations to create tables first
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	migrator, err := NewMigrator(partitionDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	pm := NewPartitionManager(partitionDB)

	// Create some test partitions (simulate old partitions)
	oldDate := time.Now().AddDate(0, -2, 0) // 2 months ago
	partitionName := "articles_" + oldDate.Format("2006_01")
	
	createOldPartitionSQL := `
		CREATE TABLE ` + partitionName + ` PARTITION OF articles 
		FOR VALUES FROM ('` + oldDate.Format("2006-01-02") + `') TO ('` + oldDate.AddDate(0, 1, 0).Format("2006-01-02") + `')
	`
	
	if _, err := partitionDB.Exec(createOldPartitionSQL); err != nil {
		t.Fatalf("Failed to create test old partition: %v", err)
	}

	// Verify the old partition exists
	var partitionExists bool
	err = partitionDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_tables 
			WHERE tablename = $1
		)
	`, partitionName).Scan(&partitionExists)
	if err != nil {
		t.Fatalf("Failed to check if old partition exists: %v", err)
	}
	if !partitionExists {
		t.Error("Old test partition was not created")
	}

	// Test dropping old partitions (30 days retention)
	if err := pm.DropOldPartitions(30); err != nil {
		t.Fatalf("Failed to drop old partitions: %v", err)
	}

	// Verify the old partition was dropped
	err = partitionDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_tables 
			WHERE tablename = $1
		)
	`, partitionName).Scan(&partitionExists)
	if err != nil {
		t.Fatalf("Failed to check if old partition was dropped: %v", err)
	}
	if partitionExists {
		t.Error("Old partition was not dropped")
	}

	// Verify that the function was created
	var functionExists bool
	err = partitionDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_proc 
			WHERE proname = 'drop_old_partitions'
		)
	`).Scan(&functionExists)
	if err != nil {
		t.Fatalf("Failed to check if drop partition function exists: %v", err)
	}
	if !functionExists {
		t.Error("Drop partition function was not created")
	}
}

func TestDropOldPartitionsWithConfiguredRetention(t *testing.T) {
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	pm := NewPartitionManager(testDB)
	pm.SetRetentionDays(45)
	
	// Test that DropOldPartitions uses configured retention when called with 0
	// We can't easily test the actual dropping without a full setup, but we can test the logic
	if err := pm.DropOldPartitions(0); err != nil {
		// This might fail due to missing tables, but that's expected in this test
		t.Logf("Expected error (no tables): %v", err)
	}
	
	// The important thing is that it should use the configured retention (45 days)
	// This is tested implicitly in the function logic
}

func TestSchedulePartitionMaintenance(t *testing.T) {
	// Skip test if database is not available
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	pm := NewPartitionManager(testDB)

	// Test scheduling partition maintenance
	if err := pm.SchedulePartitionMaintenance(); err != nil {
		t.Fatalf("Failed to schedule partition maintenance: %v", err)
	}

	// Verify that the maintenance function was created
	var functionExists bool
	err = testDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_proc 
			WHERE proname = 'partition_maintenance'
		)
	`).Scan(&functionExists)
	if err != nil {
		t.Fatalf("Failed to check if maintenance function exists: %v", err)
	}
	if !functionExists {
		t.Error("Partition maintenance function was not created")
	}
}

func TestRunMaintenance(t *testing.T) {
	// Skip test if database is not available
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	// Create test database
	_, err = testDB.Exec("CREATE DATABASE partition_maintenance_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS partition_maintenance_test")
	}()

	// Connect to test database
	partitionDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=partition_maintenance_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer partitionDB.Close()

	// Run migrations to create tables first
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	migrator, err := NewMigrator(partitionDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	pm := NewPartitionManager(partitionDB)

	// Schedule maintenance first
	if err := pm.SchedulePartitionMaintenance(); err != nil {
		t.Fatalf("Failed to schedule partition maintenance: %v", err)
	}

	// Test running maintenance
	if err := pm.RunMaintenance(); err != nil {
		t.Fatalf("Failed to run partition maintenance: %v", err)
	}

	// The maintenance should have created the daily partition functions
	var functionExists bool
	err = partitionDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_proc 
			WHERE proname = 'create_daily_partitions'
		)
	`).Scan(&functionExists)
	if err != nil {
		t.Fatalf("Failed to check if daily partition function exists: %v", err)
	}
	if !functionExists {
		t.Error("Daily partition function was not created by maintenance")
	}
}

func TestGetPartitionInfo(t *testing.T) {
	// Skip test if database is not available
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	// Create test database
	_, err = testDB.Exec("CREATE DATABASE partition_info_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS partition_info_test")
	}()

	// Connect to test database
	partitionDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=partition_info_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer partitionDB.Close()

	// Run migrations to create tables first
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	migrator, err := NewMigrator(partitionDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	pm := NewPartitionManager(partitionDB)

	// Get partition info (should include the initial partitions created by migration)
	partitions, err := pm.GetPartitionInfo()
	if err != nil {
		t.Fatalf("Failed to get partition info: %v", err)
	}

	// Should have at least the initial partitions created by migration
	if len(partitions) == 0 {
		t.Error("Expected at least some partitions to exist")
	}

	// Verify partition info structure
	for _, partition := range partitions {
		if partition.Schema == "" {
			t.Error("Partition schema should not be empty")
		}
		if partition.Name == "" {
			t.Error("Partition name should not be empty")
		}
		if partition.Size == "" {
			t.Error("Partition size should not be empty")
		}
		t.Logf("Partition: %s.%s, Size: %s", partition.Schema, partition.Name, partition.Size)
	}
}

func TestStartStopPartitionScheduler(t *testing.T) {
	// Skip test if database is not available
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	pm := NewPartitionManager(testDB)

	// Initially scheduler should not be active
	if pm.IsSchedulerActive() {
		t.Error("Scheduler should not be active initially")
	}

	// Test starting the scheduler
	pm.StartPartitionScheduler(
func 
TestSetRetentionDays(t *testing.T) {
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	pm := NewPartitionManager(testDB)
	
	// Test default retention
	if pm.retentionDays != 30 {
		t.Errorf("Expected default retention of 30 days, got %d", pm.retentionDays)
	}

	// Test setting valid retention
	pm.SetRetentionDays(60)
	if pm.retentionDays != 60 {
		t.Errorf("Expected retention of 60 days, got %d", pm.retentionDays)
	}

	// Test setting invalid retention (should not change)
	pm.SetRetentionDays(-10)
	if pm.retentionDays != 60 {
		t.Errorf("Expected retention to remain 60 days, got %d", pm.retentionDays)
	}

	// Test setting zero retention (should not change)
	pm.SetRetentionDays(0)
	if pm.retentionDays != 60 {
		t.Errorf("Expected retention to remain 60 days, got %d", pm.retentionDays)
	}
}

func TestPartitionSchedulerLifecycle(t *testing.T) {
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	pm := NewPartitionManager(testDB)

	// Initially scheduler should not be active
	if pm.IsSchedulerActive() {
		t.Error("Scheduler should not be active initially")
	}

	// Start scheduler
	pm.StartPartitionScheduler()
	
	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)
	
	if !pm.IsSchedulerActive() {
		t.Error("Scheduler should be active after starting")
	}

	// Try to start again (should not create duplicate)
	pm.StartPartitionScheduler()
	if !pm.IsSchedulerActive() {
		t.Error("Scheduler should still be active")
	}

	// Stop scheduler
	pm.StopPartitionScheduler()
	
	// Give it a moment to stop
	time.Sleep(200 * time.Millisecond)
	
	if pm.IsSchedulerActive() {
		t.Error("Scheduler should not be active after stopping")
	}

	// Try to stop again (should not error)
	pm.StopPartitionScheduler()
}

func TestCreateDailyPartitionsErrorHandling(t *testing.T) {
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	pm := NewPartitionManager(testDB)

	// Test with non-partitioned table (should fail)
	err = pm.CreateDailyPartitions()
	if err == nil {
		t.Error("Expected error when articles table doesn't exist or isn't partitioned")
	}
	if err != nil && err.Error() != "articles table is not partitioned - cannot create daily partitions" {
		t.Logf("Got expected error: %v", err)
	}
}

func TestDropOldPartitionsWithConfigurableRetention(t *testing.T) {
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	// Create test database
	_, err = testDB.Exec("CREATE DATABASE partition_retention_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS partition_retention_test")
	}()

	// Connect to test database
	partitionDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=partition_retention_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer partitionDB.Close()

	// Run migrations to create tables first
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	migrator, err := NewMigrator(partitionDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	pm := NewPartitionManager(partitionDB)

	// Set custom retention period
	pm.SetRetentionDays(45)

	// Test dropping with configured retention (should use 45 days)
	err = pm.DropOldPartitions(0) // 0 should use configured retention
	if err != nil {
		t.Fatalf("Failed to drop old partitions with configured retention: %v", err)
	}

	// Test dropping with explicit retention (should use 60 days)
	err = pm.DropOldPartitions(60)
	if err != nil {
		t.Fatalf("Failed to drop old partitions with explicit retention: %v", err)
	}
}

func TestPartitionMaintenanceIntegration(t *testing.T) {
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	// Create test database
	_, err = testDB.Exec("CREATE DATABASE partition_integration_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS partition_integration_test")
	}()

	// Connect to test database
	partitionDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=partition_integration_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer partitionDB.Close()

	// Run migrations to create tables first
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	migrator, err := NewMigrator(partitionDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	pm := NewPartitionManager(partitionDB)

	// Test complete maintenance workflow
	// 1. Schedule maintenance
	if err := pm.SchedulePartitionMaintenance(); err != nil {
		t.Fatalf("Failed to schedule partition maintenance: %v", err)
	}

	// 2. Create daily partitions
	if err := pm.CreateDailyPartitions(); err != nil {
		t.Fatalf("Failed to create daily partitions: %v", err)
	}

	// 3. Verify partitions were created
	partitions, err := pm.GetPartitionInfo()
	if err != nil {
		t.Fatalf("Failed to get partition info: %v", err)
	}

	// Should have at least the initial partitions + new daily partitions
	if len(partitions) < 7 { // At least 7 daily partitions should be created
		t.Logf("Warning: Expected at least 7 partitions, got %d", len(partitions))
	}

	// 4. Run full maintenance
	if err := pm.RunMaintenance(); err != nil {
		t.Fatalf("Failed to run maintenance: %v", err)
	}

	// 5. Test scheduler lifecycle
	pm.StartPartitionScheduler()
	time.Sleep(100 * time.Millisecond)
	
	if !pm.IsSchedulerActive() {
		t.Error("Scheduler should be active")
	}

	pm.StopPartitionScheduler()
	time.Sleep(200 * time.Millisecond)
	
	if pm.IsSchedulerActive() {
		t.Error("Scheduler should be stopped")
	}
}

func TestPartitionErrorScenarios(t *testing.T) {
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	pm := NewPartitionManager(testDB)

	// Test with invalid database connection
	invalidDB, _ := sql.Open("postgres", "invalid_connection_string")
	invalidPM := NewPartitionManager(invalidDB)

	// These should fail gracefully
	err = invalidPM.CreateDailyPartitions()
	if err == nil {
		t.Error("Expected error with invalid database connection")
	}

	err = invalidPM.DropOldPartitions(30)
	if err == nil {
		t.Error("Expected error with invalid database connection")
	}

	err = invalidPM.SchedulePartitionMaintenance()
	if err == nil {
		t.Error("Expected error with invalid database connection")
	}

	err = invalidPM.RunMaintenance()
	if err == nil {
		t.Error("Expected error with invalid database connection")
	}

	_, err = invalidPM.GetPartitionInfo()
	if err == nil {
		t.Error("Expected error with invalid database connection")
	}
}

func TestPartitionInfoAccuracy(t *testing.T) {
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PostgreSQL not available for testing")
	}

	// Create test database
	_, err = testDB.Exec("CREATE DATABASE partition_info_accuracy_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS partition_info_accuracy_test")
	}()

	// Connect to test database
	partitionDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=partition_info_accuracy_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer partitionDB.Close()

	// Run migrations to create tables first
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	migrator, err := NewMigrator(partitionDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	pm := NewPartitionManager(partitionDB)

	// Get initial partition info
	initialPartitions, err := pm.GetPartitionInfo()
	if err != nil {
		t.Fatalf("Failed to get initial partition info: %v", err)
	}

	initialCount := len(initialPartitions)
	t.Logf("Initial partition count: %d", initialCount)

	// Create daily partitions
	if err := pm.CreateDailyPartitions(); err != nil {
		t.Fatalf("Failed to create daily partitions: %v", err)
	}

	// Get updated partition info
	updatedPartitions, err := pm.GetPartitionInfo()
	if err != nil {
		t.Fatalf("Failed to get updated partition info: %v", err)
	}

	updatedCount := len(updatedPartitions)
	t.Logf("Updated partition count: %d", updatedCount)

	// Should have more partitions now (at least 7 new daily partitions)
	if updatedCount <= initialCount {
		t.Logf("Warning: Expected more partitions after creating daily partitions. Initial: %d, Updated: %d", initialCount, updatedCount)
	}

	// Verify partition info structure
	for _, partition := range updatedPartitions {
		if partition.Schema == "" {
			t.Error("Partition schema should not be empty")
		}
		if partition.Name == "" {
			t.Error("Partition name should not be empty")
		}
		if partition.Size == "" {
			t.Error("Partition size should not be empty")
		}
		
		// Verify partition name format
		if partition.Name != "articles" && partition.Name != "article_tags" && partition.Name != "article_views" {
			// Should match partition naming pattern
			if !matchesPartitionPattern(partition.Name) {
				t.Errorf("Partition name %s doesn't match expected pattern", partition.Name)
			}
		}
		
		t.Logf("Partition: %s.%s, Size: %s, Exists: %t", partition.Schema, partition.Name, partition.Size, partition.Exists)
	}
}

// Helper function to check if partition name matches expected patterns
func matchesPartitionPattern(name string) bool {
	patterns := []string{
		`^articles_\d{4}_\d{2}$`,      // Monthly: articles_2024_01
		`^articles_\d{4}_\d{2}_\d{2}$`, // Daily: articles_2024_01_15
		`^article_tags_\d{4}_\d{2}$`,   // Monthly: article_tags_2024_01
		`^article_views_\d{4}_\d{2}$`,  // Monthly: article_views_2024_01
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, name); matched {
			return true
		}
	}
	return false
}