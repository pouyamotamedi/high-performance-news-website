package testing

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestMultilingualDataGenerationIntegration tests the complete multilingual data generation system
func TestMultilingualDataGenerationIntegration(t *testing.T) {
	// Setup test database
	db, err := setupTestDatabase()
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}
	defer db.Close()

	// Create test data generator
	generator := NewTestDataGenerator(db)

	// Test small-scale generation
	t.Run("SmallScaleGeneration", func(t *testing.T) {
		articles, err := generator.GenerateMultilingualTestData(30) // 10 per language
		if err != nil {
			t.Fatalf("Failed to generate test data: %v", err)
		}

		if len(articles) != 30 {
			t.Errorf("Expected 30 articles, got %d", len(articles))
		}

		// Verify language distribution
		langCounts := make(map[string]int)
		for _, article := range articles {
			langCounts[article.LanguageCode]++
		}

		expectedLangs := []string{"en", "fa", "ar"}
		for _, lang := range expectedLangs {
			if langCounts[lang] != 10 {
				t.Errorf("Expected 10 articles for language %s, got %d", lang, langCounts[lang])
			}
		}

		// Verify article quality
		for _, article := range articles {
			if article.Title == "" {
				t.Error("Article has empty title")
			}
			if article.Content == "" {
				t.Error("Article has empty content")
			}
			if article.LanguageCode == "" {
				t.Error("Article has empty language code")
			}
		}
	})

	// Test large-scale generation
	t.Run("LargeScaleGeneration", func(t *testing.T) {
		dataset, err := generator.GenerateLargeScaleTestData(1000)
		if err != nil {
			t.Fatalf("Failed to generate large-scale test data: %v", err)
		}

		if len(dataset.Articles) != 1000 {
			t.Errorf("Expected 1000 articles, got %d", len(dataset.Articles))
		}

		// Verify metrics
		if dataset.Metrics.QualityScore < 0.95 {
			t.Errorf("Quality score too low: %f", dataset.Metrics.QualityScore)
		}

		// Verify language distribution
		totalByLang := 0
		for _, count := range dataset.Metrics.LanguageDistribution {
			totalByLang += count
		}
		if totalByLang != len(dataset.Articles) {
			t.Errorf("Language distribution doesn't match total articles")
		}
	})

	// Test bulk insert
	t.Run("BulkInsert", func(t *testing.T) {
		articles, err := generator.GenerateMultilingualTestData(100)
		if err != nil {
			t.Fatalf("Failed to generate test data: %v", err)
		}

		err = generator.BulkInsertArticles(articles)
		if err != nil {
			t.Fatalf("Failed to bulk insert articles: %v", err)
		}

		// Verify insertion
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count articles: %v", err)
		}

		if count != len(articles) {
			t.Errorf("Expected %d articles in database, got %d", len(articles), count)
		}
	})
}

// TestDataIsolationIntegration tests the data isolation system
func TestDataIsolationIntegration(t *testing.T) {
	config := IsolationConfig{
		DatabaseHost:    "localhost",
		DatabasePort:    5432,
		CacheHost:      "localhost",
		CachePort:      6379,
		FileSystemRoot: "/tmp/test_isolation",
		MaxEnvironments: 10,
		DefaultTTL:     time.Hour,
		CleanupInterval: time.Minute,
		ResourceLimits: ResourceAllocation{
			DatabaseConnections: 10,
			CacheMemoryMB:      64,
			FileSystemSizeMB:   100,
			CPULimitPercent:    50,
			MemoryLimitMB:      256,
		},
	}

	manager := NewTestDataIsolationManager(config)

	t.Run("CreateEnvironment", func(t *testing.T) {
		env, err := manager.CreateIsolatedEnvironment("test-suite-1")
		if err != nil {
			t.Skipf("Skipping isolation test: %v", err)
		}

		if env.TestSuite != "test-suite-1" {
			t.Errorf("Expected test suite 'test-suite-1', got '%s'", env.TestSuite)
		}

		if env.Status != EnvironmentStatusReady {
			t.Errorf("Expected status 'ready', got '%s'", env.Status)
		}

		// Test database connection
		db, err := manager.GetDatabaseConnection(env.ID)
		if err != nil {
			t.Errorf("Failed to get database connection: %v", err)
		} else {
			err = db.Ping()
			if err != nil {
				t.Errorf("Database ping failed: %v", err)
			}
		}

		// Test cache connection
		cache, err := manager.GetCacheClient(env.ID)
		if err != nil {
			t.Errorf("Failed to get cache client: %v", err)
		} else {
			err = cache.Ping(cache.Context()).Err()
			if err != nil {
				t.Errorf("Cache ping failed: %v", err)
			}
		}

		// Test filesystem
		fsPath, err := manager.GetFileSystemPath(env.ID)
		if err != nil {
			t.Errorf("Failed to get filesystem path: %v", err)
		} else {
			if _, err := os.Stat(fsPath); os.IsNotExist(err) {
				t.Errorf("Filesystem path does not exist: %s", fsPath)
			}
		}

		// Cleanup
		err = manager.CleanupEnvironment(env.ID)
		if err != nil {
			t.Errorf("Failed to cleanup environment: %v", err)
		}
	})

	t.Run("MultipleEnvironments", func(t *testing.T) {
		var environments []*TestEnvironment

		// Create multiple environments
		for i := 0; i < 3; i++ {
			env, err := manager.CreateIsolatedEnvironment(fmt.Sprintf("test-suite-%d", i))
			if err != nil {
				t.Skipf("Skipping multiple environments test: %v", err)
			}
			environments = append(environments, env)
		}

		// Verify isolation
		for i, env := range environments {
			// Each environment should have unique database
			db, err := manager.GetDatabaseConnection(env.ID)
			if err != nil {
				t.Errorf("Failed to get database connection for env %d: %v", i, err)
				continue
			}

			// Create a test table in this environment
			_, err = db.Exec(fmt.Sprintf("CREATE TABLE test_table_%d (id INT)", i))
			if err != nil {
				t.Errorf("Failed to create test table in env %d: %v", i, err)
			}
		}

		// Verify that tables don't exist in other environments
		for i, env := range environments {
			db, err := manager.GetDatabaseConnection(env.ID)
			if err != nil {
				continue
			}

			for j := 0; j < 3; j++ {
				if i == j {
					continue // Skip own table
				}

				var exists bool
				err = db.QueryRow(fmt.Sprintf(`
					SELECT EXISTS (
						SELECT FROM information_schema.tables 
						WHERE table_name = 'test_table_%d'
					)
				`, j)).Scan(&exists)

				if err == nil && exists {
					t.Errorf("Table from environment %d found in environment %d", j, i)
				}
			}
		}

		// Cleanup all environments
		for _, env := range environments {
			err := manager.CleanupEnvironment(env.ID)
			if err != nil {
				t.Errorf("Failed to cleanup environment %s: %v", env.ID, err)
			}
		}
	})
}

// TestDataCleanupIntegration tests the data cleanup and archival system
func TestDataCleanupIntegration(t *testing.T) {
	// Setup isolation manager
	isolationConfig := IsolationConfig{
		DatabaseHost:    "localhost",
		DatabasePort:    5432,
		CacheHost:      "localhost",
		CachePort:      6379,
		FileSystemRoot: "/tmp/test_cleanup",
		MaxEnvironments: 10,
		DefaultTTL:     time.Hour,
		CleanupInterval: time.Minute,
	}

	isolationManager := NewTestDataIsolationManager(isolationConfig)

	// Setup cleanup manager
	cleanupConfig := CleanupConfig{
		ArchiveDirectory:   "/tmp/test_archives",
		MaxArchiveSize:     1024 * 1024 * 100, // 100MB
		CompressionEnabled: true,
		RetentionPeriod:    24 * time.Hour,
		CleanupBatchSize:   100,
		ParallelWorkers:    2,
		VerifyBeforeDelete: true,
	}

	cleanupManager := NewTestDataCleanupManager(isolationManager, cleanupConfig)

	t.Run("ArchiveEnvironment", func(t *testing.T) {
		// Create test environment
		env, err := isolationManager.CreateIsolatedEnvironment("archive-test")
		if err != nil {
			t.Skipf("Skipping archive test: %v", err)
		}

		// Add some test data to the environment
		db, err := isolationManager.GetDatabaseConnection(env.ID)
		if err == nil {
			db.Exec("CREATE TABLE test_data (id INT, name TEXT)")
			db.Exec("INSERT INTO test_data VALUES (1, 'test')")
		}

		// Archive the environment
		err = cleanupManager.archiveEnvironment(env)
		if err != nil {
			t.Errorf("Failed to archive environment: %v", err)
		}

		// Verify archive was created
		stats, err := cleanupManager.GetArchiveStats()
		if err != nil {
			t.Errorf("Failed to get archive stats: %v", err)
		}

		totalArchives := stats["total_archives"].(int)
		if totalArchives == 0 {
			t.Error("No archives found after archiving environment")
		}

		// Cleanup
		isolationManager.CleanupEnvironment(env.ID)
	})

	t.Run("CleanupProcess", func(t *testing.T) {
		// Create test environments with different ages
		oldEnv, err := isolationManager.CreateIsolatedEnvironment("old-test")
		if err != nil {
			t.Skipf("Skipping cleanup test: %v", err)
		}

		// Simulate old environment by modifying timestamps
		oldEnv.CreatedAt = time.Now().Add(-25 * time.Hour)
		oldEnv.LastAccessedAt = time.Now().Add(-25 * time.Hour)

		newEnv, err := isolationManager.CreateIsolatedEnvironment("new-test")
		if err != nil {
			t.Skipf("Skipping cleanup test: %v", err)
		}

		// Run cleanup
		result, err := cleanupManager.RunCleanup()
		if err != nil {
			t.Errorf("Cleanup failed: %v", err)
		}

		if result.EnvironmentsProcessed == 0 {
			t.Error("No environments were processed during cleanup")
		}

		log.Printf("Cleanup result: %+v", result)

		// Cleanup remaining environment
		isolationManager.CleanupEnvironment(newEnv.ID)
	})
}

// TestPerformanceOptimization tests the performance optimization system
func TestPerformanceOptimization(t *testing.T) {
	// Setup test database
	db, err := setupTestDatabase()
	if err != nil {
		t.Skipf("Skipping performance test: %v", err)
	}
	defer db.Close()

	config := PerformanceConfig{
		MaxBatchSize:       1000,
		MaxConcurrency:     4,
		MemoryLimitMB:      512,
		CacheSize:          1000,
		ConnectionPoolSize: 10,
		QueryTimeout:       30 * time.Second,
		EnableProfiling:    true,
		OptimizationLevel:  2,
	}

	optimizer := NewTestDataPerformanceOptimizer(db, config)
	generator := NewTestDataGenerator(db)

	t.Run("OptimizedGeneration", func(t *testing.T) {
		startTime := time.Now()

		articles, err := optimizer.OptimizeDataGeneration(generator, 1000)
		if err != nil {
			t.Fatalf("Optimized generation failed: %v", err)
		}

		duration := time.Since(startTime)
		throughput := float64(len(articles)) / duration.Seconds()

		if len(articles) != 1000 {
			t.Errorf("Expected 1000 articles, got %d", len(articles))
		}

		log.Printf("Generated %d articles in %v (%.2f articles/sec)", len(articles), duration, throughput)

		// Verify performance is reasonable (should be > 10 articles/sec)
		if throughput < 10 {
			t.Errorf("Performance too low: %.2f articles/sec", throughput)
		}
	})

	t.Run("OptimizedBulkInsert", func(t *testing.T) {
		// Generate test data
		articles, err := generator.GenerateMultilingualTestData(500)
		if err != nil {
			t.Fatalf("Failed to generate test data: %v", err)
		}

		startTime := time.Now()

		err = optimizer.OptimizeBulkInsert(articles)
		if err != nil {
			t.Fatalf("Optimized bulk insert failed: %v", err)
		}

		duration := time.Since(startTime)
		throughput := float64(len(articles)) / duration.Seconds()

		log.Printf("Inserted %d articles in %v (%.2f articles/sec)", len(articles), duration, throughput)

		// Verify insertion
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count articles: %v", err)
		}

		if count < len(articles) {
			t.Errorf("Expected at least %d articles in database, got %d", len(articles), count)
		}
	})

	t.Run("PerformanceReport", func(t *testing.T) {
		report := optimizer.GetPerformanceReport()

		// Verify report structure
		expectedSections := []string{"memory_stats", "cache_stats", "batch_stats", "connection_stats", "overall_metrics", "recommendations"}
		for _, section := range expectedSections {
			if _, exists := report[section]; !exists {
				t.Errorf("Performance report missing section: %s", section)
			}
		}

		// Verify recommendations exist
		recommendations := report["recommendations"].([]string)
		if len(recommendations) == 0 {
			t.Error("No performance recommendations provided")
		}

		log.Printf("Performance recommendations: %v", recommendations)
	})
}

// setupTestDatabase creates a test database connection
func setupTestDatabase() (*sql.DB, error) {
	// Try to connect to test database
	dsn := "host=localhost port=5432 user=postgres dbname=test_db sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create test tables
	err = createTestTables(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create test tables: %w", err)
	}

	return db, nil
}

// createTestTables creates the necessary test tables
func createTestTables(db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS articles (
			id BIGINT PRIMARY KEY,
			title TEXT NOT NULL,
			slug TEXT NOT NULL,
			content TEXT NOT NULL,
			excerpt TEXT,
			author_id BIGINT NOT NULL,
			category_id BIGINT NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			published_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			view_count BIGINT DEFAULT 0,
			like_count BIGINT DEFAULT 0,
			dislike_count BIGINT DEFAULT 0,
			language_code TEXT NOT NULL,
			translation_group_id BIGINT,
			meta_title TEXT,
			meta_description TEXT,
			canonical_url TEXT,
			schema_type TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			first_name TEXT,
			last_name TEXT,
			bio TEXT,
			avatar TEXT,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT UNIQUE NOT NULL,
			description TEXT,
			parent_id BIGINT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			id BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT UNIQUE NOT NULL,
			color TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// BenchmarkDataGeneration benchmarks data generation performance
func BenchmarkDataGeneration(b *testing.B) {
	generator := NewTestDataGenerator(nil)

	b.Run("SmallBatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := generator.GenerateMultilingualTestData(10)
			if err != nil {
				b.Fatalf("Generation failed: %v", err)
			}
		}
	})

	b.Run("MediumBatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := generator.GenerateMultilingualTestData(100)
			if err != nil {
				b.Fatalf("Generation failed: %v", err)
			}
		}
	})

	b.Run("LargeBatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := generator.GenerateMultilingualTestData(1000)
			if err != nil {
				b.Fatalf("Generation failed: %v", err)
			}
		}
	})
}

// BenchmarkDataIsolation benchmarks data isolation operations
func BenchmarkDataIsolation(b *testing.B) {
	config := IsolationConfig{
		DatabaseHost:    "localhost",
		DatabasePort:    5432,
		CacheHost:      "localhost",
		CachePort:      6379,
		FileSystemRoot: "/tmp/bench_isolation",
		MaxEnvironments: 100,
		DefaultTTL:     time.Hour,
		CleanupInterval: time.Minute,
	}

	manager := NewTestDataIsolationManager(config)

	b.Run("CreateEnvironment", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			env, err := manager.CreateIsolatedEnvironment(fmt.Sprintf("bench-test-%d", i))
			if err != nil {
				b.Skipf("Skipping benchmark: %v", err)
			}
			// Cleanup immediately to avoid resource exhaustion
			manager.CleanupEnvironment(env.ID)
		}
	})
}

// TestDataConsistencyValidation tests data consistency validation
func TestDataConsistencyValidation(t *testing.T) {
	config := IsolationConfig{
		DatabaseHost:    "localhost",
		DatabasePort:    5432,
		CacheHost:      "localhost",
		CachePort:      6379,
		FileSystemRoot: "/tmp/test_consistency",
		MaxEnvironments: 5,
		DefaultTTL:     time.Hour,
		CleanupInterval: time.Minute,
	}

	manager := NewTestDataIsolationManager(config)

	t.Run("ValidateConsistency", func(t *testing.T) {
		env, err := manager.CreateIsolatedEnvironment("consistency-test")
		if err != nil {
			t.Skipf("Skipping consistency test: %v", err)
		}

		// Validate initial consistency
		err = manager.ValidateDataConsistency(env.ID)
		if err != nil {
			t.Errorf("Initial consistency validation failed: %v", err)
		}

		// Add some test data
		db, err := manager.GetDatabaseConnection(env.ID)
		if err == nil {
			db.Exec("CREATE TABLE test_consistency (id INT PRIMARY KEY, data TEXT)")
			db.Exec("INSERT INTO test_consistency VALUES (1, 'test')")
		}

		// Validate consistency after adding data
		err = manager.ValidateDataConsistency(env.ID)
		if err != nil {
			t.Errorf("Consistency validation failed after adding data: %v", err)
		}

		// Test repair functionality
		err = manager.RepairDataConsistency(env.ID)
		if err != nil {
			t.Errorf("Data consistency repair failed: %v", err)
		}

		// Cleanup
		manager.CleanupEnvironment(env.ID)
	})
}