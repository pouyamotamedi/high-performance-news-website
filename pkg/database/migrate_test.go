package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lib/pq"
)

func TestNewMigrator(t *testing.T) {
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
	_, err = testDB.Exec("CREATE DATABASE migrate_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS migrate_test")
	}()

	// Connect to test database
	migrateDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=migrate_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer migrateDB.Close()

	// Get migrations path
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	// Test migrator creation
	migrator, err := NewMigrator(migrateDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	// Test getting version (should be no version initially)
	version, dirty, err := migrator.Version()
	if err != nil {
		t.Logf("No migration version found (expected for new database): %v", err)
	}
	t.Logf("Initial migration version: %d, dirty: %t", version, dirty)
}

func TestMigrationUpDown(t *testing.T) {
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
	_, err = testDB.Exec("CREATE DATABASE migrate_updown_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS migrate_updown_test")
	}()

	// Connect to test database
	migrateDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=migrate_updown_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer migrateDB.Close()

	// Get migrations path
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	// Create migrator
	migrator, err := NewMigrator(migrateDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	// Test migration up
	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations up: %v", err)
	}

	// Check that tables were created
	var tableExists bool
	err = migrateDB.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if users table exists: %v", err)
	}
	if !tableExists {
		t.Error("Users table was not created by migration")
	}

	// Check version after up migration
	version, dirty, err := migrator.Version()
	if err != nil {
		t.Fatalf("Failed to get migration version: %v", err)
	}
	if dirty {
		t.Error("Migration is in dirty state after up")
	}
	t.Logf("Migration version after up: %d", version)

	// Test migration down
	if err := migrator.Down(); err != nil {
		t.Fatalf("Failed to run migrations down: %v", err)
	}

	// Check that tables were dropped
	err = migrateDB.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if users table exists after down: %v", err)
	}
	if tableExists {
		t.Error("Users table still exists after migration down")
	}
}

func TestMigrationSteps(t *testing.T) {
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
	_, err = testDB.Exec("CREATE DATABASE migrate_steps_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS migrate_steps_test")
	}()

	// Connect to test database
	migrateDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=migrate_steps_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer migrateDB.Close()

	// Get migrations path
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	// Create migrator
	migrator, err := NewMigrator(migrateDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	// Test stepping up 1 migration
	if err := migrator.Steps(1); err != nil {
		t.Fatalf("Failed to run 1 step up: %v", err)
	}

	// Check version
	version, dirty, err := migrator.Version()
	if err != nil {
		t.Fatalf("Failed to get migration version: %v", err)
	}
	if dirty {
		t.Error("Migration is in dirty state after step up")
	}
	if version == 0 {
		t.Error("Expected version > 0 after stepping up")
	}

	// Test stepping down 1 migration
	if err := migrator.Steps(-1); err != nil {
		t.Fatalf("Failed to run 1 step down: %v", err)
	}

	// Check version after step down
	versionAfterDown, dirty, err := migrator.Version()
	if err != nil {
		// Version might not exist after stepping down to 0
		t.Logf("No version after stepping down (expected): %v", err)
	} else {
		if dirty {
			t.Error("Migration is in dirty state after step down")
		}
		if versionAfterDown >= version {
			t.Errorf("Expected version to decrease after step down, got %d (was %d)", versionAfterDown, version)
		}
	}
}

func TestCreatePartitions(t *testing.T) {
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
	_, err = testDB.Exec("CREATE DATABASE partition_test")
	if err != nil {
		t.Logf("Test database creation failed (might already exist): %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS partition_test")
	}()

	// Connect to test database
	partitionDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=partition_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer partitionDB.Close()

	// Get migrations path
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	// Create migrator and run migrations first
	migrator, err := NewMigrator(partitionDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	// Run migrations to create tables
	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test partition creation
	if err := migrator.CreatePartitions(); err != nil {
		t.Fatalf("Failed to create partitions: %v", err)
	}

	// Verify that partition functions were created
	var functionExists bool
	err = partitionDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_proc 
			WHERE proname = 'create_monthly_partitions'
		)
	`).Scan(&functionExists)
	if err != nil {
		t.Fatalf("Failed to check if partition function exists: %v", err)
	}
	if !functionExists {
		t.Error("Partition function was not created")
	}

	// Verify that some partitions were created
	var partitionCount int
	err = partitionDB.QueryRow(`
		SELECT COUNT(*) FROM pg_tables 
		WHERE tablename ~ '^articles_\d{4}_\d{2}$'
	`).Scan(&partitionCount)
	if err != nil {
		t.Fatalf("Failed to count partitions: %v", err)
	}
	if partitionCount == 0 {
		t.Error("No article partitions were created")
	}
	t.Logf("Created %d article partitions", partitionCount)
}

func TestMigratorClose(t *testing.T) {
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

	// Get migrations path
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	// Create migrator
	migrator, err := NewMigrator(testDB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	// Test close
	if err := migrator.Close(); err != nil {
		t.Fatalf("Failed to close migrator: %v", err)
	}

	// Test that operations fail after close
	if err := migrator.Up(); err == nil {
		t.Error("Expected Up() to fail after Close()")
	}
}