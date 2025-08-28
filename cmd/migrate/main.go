package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
	"high-performance-news-website/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate <up|down>")
		os.Exit(1)
	}

	command := os.Args[1]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Create migrations table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	switch command {
	case "up":
		runMigrationsUp(db)
	case "down":
		runMigrationsDown(db)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Usage: migrate <up|down>")
		os.Exit(1)
	}
}

func runMigrationsUp(db *sql.DB) {
	// Get applied migrations
	appliedMigrations := getAppliedMigrations(db)

	// Get all migration files
	migrationFiles, err := getMigrationFiles("up")
	if err != nil {
		log.Fatalf("Failed to get migration files: %v", err)
	}

	// Apply pending migrations
	for _, file := range migrationFiles {
		version := extractVersion(file)
		if _, applied := appliedMigrations[version]; applied {
			fmt.Printf("Migration %s already applied, skipping\n", version)
			continue
		}

		fmt.Printf("Applying migration %s...\n", version)
		
		content, err := os.ReadFile(filepath.Join("migrations", file))
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", file, err)
		}

		// Execute migration
		_, err = db.Exec(string(content))
		if err != nil {
			log.Fatalf("Failed to apply migration %s: %v", version, err)
		}

		// Record migration as applied
		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			log.Fatalf("Failed to record migration %s: %v", version, err)
		}

		fmt.Printf("Migration %s applied successfully\n", version)
	}

	fmt.Println("All migrations applied successfully!")
}

func runMigrationsDown(db *sql.DB) {
	// Get applied migrations
	appliedMigrations := getAppliedMigrations(db)

	// Get all migration files
	migrationFiles, err := getMigrationFiles("down")
	if err != nil {
		log.Fatalf("Failed to get migration files: %v", err)
	}

	// Reverse the order for down migrations
	sort.Sort(sort.Reverse(sort.StringSlice(migrationFiles)))

	// Apply down migrations
	for _, file := range migrationFiles {
		version := extractVersion(file)
		if _, applied := appliedMigrations[version]; !applied {
			fmt.Printf("Migration %s not applied, skipping\n", version)
			continue
		}

		fmt.Printf("Rolling back migration %s...\n", version)
		
		content, err := os.ReadFile(filepath.Join("migrations", file))
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", file, err)
		}

		// Execute rollback
		_, err = db.Exec(string(content))
		if err != nil {
			log.Fatalf("Failed to rollback migration %s: %v", version, err)
		}

		// Remove migration record
		_, err = db.Exec("DELETE FROM schema_migrations WHERE version = $1", version)
		if err != nil {
			log.Fatalf("Failed to remove migration record %s: %v", version, err)
		}

		fmt.Printf("Migration %s rolled back successfully\n", version)
	}

	fmt.Println("All migrations rolled back successfully!")
}

func getAppliedMigrations(db *sql.DB) map[string]bool {
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		log.Fatalf("Failed to get applied migrations: %v", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			log.Fatalf("Failed to scan migration version: %v", err)
		}
		applied[version] = true
	}

	return applied
}

func getMigrationFiles(direction string) ([]string, error) {
	var files []string
	
	err := filepath.WalkDir("migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		if strings.HasSuffix(d.Name(), "."+direction+".sql") {
			files = append(files, d.Name())
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	sort.Strings(files)
	return files, nil
}

func extractVersion(filename string) string {
	// Extract version from filename like "001_initial_schema.up.sql"
	parts := strings.Split(filename, "_")
	if len(parts) > 0 {
		return parts[0]
	}
	return filename
}