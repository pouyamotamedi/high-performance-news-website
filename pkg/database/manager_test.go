package database

import (
	"database/sql"
	"testing"
	"time"

	"high-performance-news-website/internal/config"

	_ "github.com/lib/pq"
)

func TestNewManager(t *testing.T) {
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

	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		DBName:       "postgres",
		SSLMode:      "disable",
		MaxConns:     150,
		MinConns:     40,
		UsePgBouncer: false,
	}

	manager, err := NewManager(cfg, "../../migrations")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer manager.Close()

	// Test that all components are initialized
	if manager.DB == nil {
		t.Error("Database connection is nil")
	}
	if manager.Migrator == nil {
		t.Error("Migrator is nil")
	}
	if manager.PartitionManager == nil {
		t.Error("PartitionManager is nil")
	}
}

func TestManagerWithPgBouncer(t *testing.T) {
	// Skip test if PgBouncer is not available
	testDB, err := sql.Open("postgres", 
		"host=localhost port=6432 user=postgres password=postgres dbname=news_website sslmode=disable")
	if err != nil {
		t.Skip("PgBouncer not available for testing")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("PgBouncer not available for testing")
	}

	cfg := &config.DatabaseConfig{
		Host:          "localhost",
		Port:          5432,
		User:          "postgres",
		Password:      "postgres",
		DBName:        "news_website",
		SSLMode:       "disable",
		MaxConns:      150,
		MinConns:      40,
		UsePgBouncer:  true,
		PgBouncerPort: 6432,
	}

	manager, err := NewManager(cfg, "../../migrations")
	if err != nil {
		t.Fatalf("Failed to create database manager with PgBouncer: %v", err)
	}
	defer manager.Close()

	// Test health check through PgBouncer
	if err := manager.Health(); err != nil {
		t.Fatalf("Manager health check failed with PgBouncer: %v", err)
	}

	// Test stats with PgBouncer
	stats := manager.GetStats()
	if !stats.UsePgBouncer {
		t.Error("Expected UsePgBouncer to be true in stats")
	}
}

func TestManagerInitialize(t *testing.T) {
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
	testDBName := "manager_test_" + time.Now().Format("20060102_150405")
	_, err = testDB.Exec("CREATE DATABASE " + testDBName)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS " + testDBName)
	}()

	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		DBName:       testDBName,
		SSLMode:      "disable",
		MaxConns:     150,
		MinConns:     40,
		UsePgBouncer: false,
	}

	manager, err := NewManager(cfg, "../../migrations")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer manager.Close()

	// Test initialization
	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize database manager: %v", err)
	}

	// Verify tables were created
	var tableExists bool
	err = manager.DB.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if users table exists: %v", err)
	}
	if !tableExists {
		t.Error("Users table was not created during initialization")
	}

	// Test health check after initialization
	if err := manager.Health(); err != nil {
		t.Fatalf("Manager health check failed after initialization: %v", err)
	}

	// Test stats after initialization
	stats := manager.GetStats()
	if stats.MigrationVersion == 0 {
		t.Error("Expected migration version > 0 after initialization")
	}
	if stats.PartitionCount == 0 {
		t.Error("Expected partitions to be created during initialization")
	}
}

func TestManagerMigrate(t *testing.T) {
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
	testDBName := "manager_migrate_test_" + time.Now().Format("20060102_150405")
	_, err = testDB.Exec("CREATE DATABASE " + testDBName)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS " + testDBName)
	}()

	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		DBName:       testDBName,
		SSLMode:      "disable",
		MaxConns:     150,
		MinConns:     40,
		UsePgBouncer: false,
	}

	manager, err := NewManager(cfg, "../../migrations")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer manager.Close()

	// Test migrate up
	if err := manager.Migrate("up", 0); err != nil {
		t.Fatalf("Failed to migrate up: %v", err)
	}

	// Verify tables were created
	var tableExists bool
	err = manager.DB.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if users table exists: %v", err)
	}
	if !tableExists {
		t.Error("Users table was not created by migration up")
	}

	// Test migrate down
	if err := manager.Migrate("down", 0); err != nil {
		t.Fatalf("Failed to migrate down: %v", err)
	}

	// Verify tables were dropped
	err = manager.DB.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if users table exists after down: %v", err)
	}
	if tableExists {
		t.Error("Users table still exists after migration down")
	}

	// Test invalid direction
	if err := manager.Migrate("invalid", 0); err == nil {
		t.Error("Expected error for invalid migration direction")
	}
}

func TestManagerMaintenance(t *testing.T) {
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
	testDBName := "manager_maintenance_test_" + time.Now().Format("20060102_150405")
	_, err = testDB.Exec("CREATE DATABASE " + testDBName)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS " + testDBName)
	}()

	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		DBName:       testDBName,
		SSLMode:      "disable",
		MaxConns:     150,
		MinConns:     40,
		UsePgBouncer: false,
	}

	manager, err := NewManager(cfg, "../../migrations")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer manager.Close()

	// Initialize first
	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize database manager: %v", err)
	}

	// Test manual maintenance
	if err := manager.RunMaintenance(); err != nil {
		t.Fatalf("Failed to run maintenance: %v", err)
	}

	// Test create daily partitions
	if err := manager.CreateDailyPartitions(); err != nil {
		t.Fatalf("Failed to create daily partitions: %v", err)
	}

	// Test drop old partitions
	if err := manager.DropOldPartitions(30); err != nil {
		t.Fatalf("Failed to drop old partitions: %v", err)
	}
}

func TestManagerStats(t *testing.T) {
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

	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		DBName:       "postgres",
		SSLMode:      "disable",
		MaxConns:     150,
		MinConns:     40,
		UsePgBouncer: false,
	}

	manager, err := NewManager(cfg, "../../migrations")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}
	defer manager.Close()

	// Test getting stats
	stats := manager.GetStats()
	
	if stats.ConnectionStats == nil {
		t.Error("Connection stats should not be nil")
	}
	
	if stats.UsePgBouncer != cfg.UsePgBouncer {
		t.Errorf("Expected UsePgBouncer %t, got %t", cfg.UsePgBouncer, stats.UsePgBouncer)
	}

	// Partitions might be empty if no migrations have been run
	if stats.Partitions == nil {
		t.Error("Partitions should not be nil (can be empty)")
	}
}

func TestManagerClose(t *testing.T) {
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

	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		DBName:       "postgres",
		SSLMode:      "disable",
		MaxConns:     150,
		MinConns:     40,
		UsePgBouncer: false,
	}

	manager, err := NewManager(cfg, "../../migrations")
	if err != nil {
		t.Fatalf("Failed to create database manager: %v", err)
	}

	// Test health before close
	if err := manager.Health(); err != nil {
		t.Fatalf("Manager health check failed before close: %v", err)
	}

	// Test close
	if err := manager.Close(); err != nil {
		t.Fatalf("Failed to close database manager: %v", err)
	}

	// Test that operations fail after close
	if err := manager.Health(); err == nil {
		t.Error("Expected health check to fail after close")
	}
}