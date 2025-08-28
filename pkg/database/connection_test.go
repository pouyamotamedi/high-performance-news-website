package database

import (
	"database/sql"
	"testing"
	"time"

	"high-performance-news-website/internal/config"

	_ "github.com/lib/pq"
)

func TestNewConnection(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "news_website_test",
		SSLMode:  "disable",
		MaxConns: 150,
		MinConns: 40,
	}

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
	_, err = testDB.Exec("CREATE DATABASE news_website_test")
	if err != nil {
		// Database might already exist, continue
		t.Logf("Test database creation failed (might already exist): %v", err)
	}

	// Test connection creation
	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.Close()

	// Test connection health
	if err := db.Health(); err != nil {
		t.Fatalf("Database health check failed: %v", err)
	}

	// Test connection pool settings
	stats := db.GetStats()
	if stats.MaxOpenConnections != cfg.MaxConns {
		t.Errorf("Expected MaxOpenConnections %d, got %d", cfg.MaxConns, stats.MaxOpenConnections)
	}

	// Test prepared statements initialization
	stmt, err := db.GetPreparedStatement(StmtGetArticle)
	if err != nil {
		t.Fatalf("Failed to get prepared statement: %v", err)
	}
	if stmt == nil {
		t.Error("Prepared statement is nil")
	}

	// Cleanup
	_, _ = testDB.Exec("DROP DATABASE IF EXISTS news_website_test")
}

func TestNewPgBouncerConnection(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     6432, // PgBouncer port
		User:     "postgres",
		Password: "postgres",
		DBName:   "news_website",
		SSLMode:  "disable",
		MaxConns: 150,
		MinConns: 40,
	}

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

	// Test PgBouncer connection creation
	db, err := NewPgBouncerConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to create PgBouncer connection: %v", err)
	}
	defer db.Close()

	// Test connection health
	if err := db.Health(); err != nil {
		t.Fatalf("PgBouncer health check failed: %v", err)
	}

	// Test connection pool settings for PgBouncer
	stats := db.GetStats()
	if stats.MaxOpenConnections != 200 {
		t.Errorf("Expected MaxOpenConnections 200 for PgBouncer, got %d", stats.MaxOpenConnections)
	}
}

func TestPreparedStatements(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres", // Use default postgres database
		SSLMode:  "disable",
		MaxConns: 10,
		MinConns: 2,
	}

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

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.Close()

	// Test all prepared statements exist
	expectedStmts := []string{
		StmtGetArticle,
		StmtGetHomepage,
		StmtGetCategory,
		StmtInsertView,
		StmtInsertArticle,
		StmtUpdateArticle,
	}

	for _, stmtName := range expectedStmts {
		stmt, err := db.GetPreparedStatement(stmtName)
		if err != nil {
			t.Errorf("Failed to get prepared statement %s: %v", stmtName, err)
		}
		if stmt == nil {
			t.Errorf("Prepared statement %s is nil", stmtName)
		}
	}

	// Test non-existent prepared statement
	_, err = db.GetPreparedStatement("non_existent")
	if err == nil {
		t.Error("Expected error for non-existent prepared statement")
	}
}

func TestConnectionPoolPerformance(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres",
		SSLMode:  "disable",
		MaxConns: 50,
		MinConns: 10,
	}

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

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.Close()

	// Test concurrent connections
	concurrency := 20
	done := make(chan bool, concurrency)

	start := time.Now()
	for i := 0; i < concurrency; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Perform a simple query
			var result int
			err := db.QueryRow("SELECT 1").Scan(&result)
			if err != nil {
				t.Errorf("Concurrent query failed: %v", err)
			}
			if result != 1 {
				t.Errorf("Expected result 1, got %d", result)
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
	
	duration := time.Since(start)
	t.Logf("Concurrent queries completed in %v", duration)

	// Check connection pool stats
	stats := db.GetStats()
	t.Logf("Connection pool stats: Open=%d, InUse=%d, Idle=%d", 
		stats.OpenConnections, stats.InUse, stats.Idle)
}

func TestConnectionClose(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres",
		SSLMode:  "disable",
		MaxConns: 10,
		MinConns: 2,
	}

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

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}

	// Test that connection works before closing
	if err := db.Health(); err != nil {
		t.Fatalf("Database health check failed before close: %v", err)
	}

	// Close the connection
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database connection: %v", err)
	}

	// Test that connection fails after closing
	if err := db.Health(); err == nil {
		t.Error("Expected health check to fail after close")
	}
}

// Benchmark tests for performance validation
func BenchmarkConnectionCreation(b *testing.B) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres",
		SSLMode:  "disable",
		MaxConns: 150,
		MinConns: 40,
	}

	// Skip benchmark if database is not available
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		b.Skip("PostgreSQL not available for benchmarking")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		b.Skip("PostgreSQL not available for benchmarking")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err := NewConnection(cfg)
		if err != nil {
			b.Fatalf("Failed to create connection: %v", err)
		}
		db.Close()
	}
}

func BenchmarkPreparedStatementExecution(b *testing.B) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres",
		SSLMode:  "disable",
		MaxConns: 150,
		MinConns: 40,
	}

	// Skip benchmark if database is not available
	testDB, err := sql.Open("postgres", 
		"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		b.Skip("PostgreSQL not available for benchmarking")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		b.Skip("PostgreSQL not available for benchmarking")
	}

	db, err := NewConnection(cfg)
	if err != nil {
		b.Fatalf("Failed to create connection: %v", err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result int
		err := db.QueryRow("SELECT 1").Scan(&result)
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}
}