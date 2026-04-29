package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Test loading default configuration
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify default values
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected server host '0.0.0.0', got '%s'", cfg.Server.Host)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected server port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected database host 'localhost', got '%s'", cfg.Database.Host)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("Expected database port 5432, got %d", cfg.Database.Port)
	}
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("NEWS_SERVER_PORT", "9090")
	os.Setenv("NEWS_DATABASE_HOST", "testhost")
	defer func() {
		os.Unsetenv("NEWS_SERVER_PORT")
		os.Unsetenv("NEWS_DATABASE_HOST")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variables override defaults
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected server port 9090 from env var, got %d", cfg.Server.Port)
	}

	if cfg.Database.Host != "testhost" {
		t.Errorf("Expected database host 'testhost' from env var, got '%s'", cfg.Database.Host)
	}
}

func TestGetDatabaseDSN(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "testuser",
			Password: "testpass",
			DBName:   "testdb",
			SSLMode:  "disable",
		},
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	actual := cfg.GetDatabaseDSN()

	if actual != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, actual)
	}
}