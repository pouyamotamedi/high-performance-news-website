package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"high-performance-news-website/internal/config"

	_ "github.com/lib/pq"
)

// TestDatabaseIntegration tests the complete database system integration
func TestDatabaseIntegration(t *testing.T) {
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
	testDBName := "integration_test_" + time.Now().Format("20060102_150405")
	_, err = testDB.Exec("CREATE DATABASE " + testDBName)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_, _ = testDB.Exec("DROP DATABASE IF EXISTS " + testDBName)
	}()

	// Test configuration
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   testDBName,
		SSLMode:  "disable",
		MaxConns: 150,
		MinConns: 40,
	}

	// Step 1: Test database connection
	t.Log("Step 1: Testing database connection...")
	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.Close()

	if err := db.Health(); err != nil {
		t.Fatalf("Database health check failed: %v", err)
	}

	// Step 2: Test migration system
	t.Log("Step 2: Testing migration system...")
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("Failed to get migrations path: %v", err)
	}

	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Skipf("Migrations directory not found: %s", migrationsPath)
	}

	migrator, err := NewMigrator(db.DB, migrationsPath)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	// Run migrations up
	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations up: %v", err)
	}

	// Verify tables were created
	tables := []string{"users", "categories", "tags", "articles", "article_tags", "article_views", "article_engagement"}
	for _, table := range tables {
		var exists bool
		err := db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)", table).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check if table %s exists: %v", table, err)
		}
		if !exists {
			t.Errorf("Table %s was not created", table)
		}
	}

	// Step 3: Test partition creation
	t.Log("Step 3: Testing partition creation...")
	if err := migrator.CreatePartitions(); err != nil {
		t.Fatalf("Failed to create partitions: %v", err)
	}

	// Step 4: Test partition manager
	t.Log("Step 4: Testing partition manager...")
	pm := NewPartitionManager(db.DB)

	// Test daily partition creation
	if err := pm.CreateDailyPartitions(); err != nil {
		t.Fatalf("Failed to create daily partitions: %v", err)
	}

	// Test partition maintenance setup
	if err := pm.SchedulePartitionMaintenance(); err != nil {
		t.Fatalf("Failed to schedule partition maintenance: %v", err)
	}

	// Test getting partition info
	partitions, err := pm.GetPartitionInfo()
	if err != nil {
		t.Fatalf("Failed to get partition info: %v", err)
	}
	if len(partitions) == 0 {
		t.Error("Expected partitions to exist")
	}
	t.Logf("Found %d partitions", len(partitions))

	// Step 5: Test prepared statements
	t.Log("Step 5: Testing prepared statements...")
	stmt, err := db.GetPreparedStatement(StmtGetArticle)
	if err != nil {
		t.Fatalf("Failed to get prepared statement: %v", err)
	}
	if stmt == nil {
		t.Error("Prepared statement is nil")
	}

	// Step 6: Test basic CRUD operations
	t.Log("Step 6: Testing basic CRUD operations...")

	// Insert a test user
	var userID int64
	err = db.QueryRow(`
		INSERT INTO users (username, email, password_hash, role, first_name, last_name)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "testuser", "test@example.com", "hashedpassword", "admin", "Test", "User").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	// Insert a test category
	var categoryID int64
	err = db.QueryRow(`
		INSERT INTO categories (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`, "Test Category", "test-category", "A test category").Scan(&categoryID)
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	// Insert a test article using prepared statement
	insertStmt, err := db.GetPreparedStatement(StmtInsertArticle)
	if err != nil {
		t.Fatalf("Failed to get insert article statement: %v", err)
	}

	var articleID int64
	var createdAt time.Time
	publishedAt := time.Now()
	err = insertStmt.QueryRow(
		"Test Article",
		"test-article",
		"This is a test article content.",
		"Test excerpt",
		userID,
		categoryID,
		"published",
		publishedAt,
		"Test Article - Meta Title",
		"Test article meta description",
		"",
		"NewsArticle",
	).Scan(&articleID, &createdAt)
	if err != nil {
		t.Fatalf("Failed to insert test article: %v", err)
	}

	// Test retrieving the article using prepared statement
	getStmt, err := db.GetPreparedStatement(StmtGetArticle)
	if err != nil {
		t.Fatalf("Failed to get article statement: %v", err)
	}

	var retrievedArticle struct {
		ID              int64
		Title           string
		Slug            string
		Content         string
		Excerpt         string
		AuthorID        int64
		CategoryID      int64
		PublishedAt     time.Time
		ViewCount       int64
		LikeCount       int64
		DislikeCount    int64
		MetaTitle       string
		MetaDescription string
		CanonicalURL    string
		SchemaType      string
	}

	err = getStmt.QueryRow("test-article").Scan(
		&retrievedArticle.ID,
		&retrievedArticle.Title,
		&retrievedArticle.Slug,
		&retrievedArticle.Content,
		&retrievedArticle.Excerpt,
		&retrievedArticle.AuthorID,
		&retrievedArticle.CategoryID,
		&retrievedArticle.PublishedAt,
		&retrievedArticle.ViewCount,
		&retrievedArticle.LikeCount,
		&retrievedArticle.DislikeCount,
		&retrievedArticle.MetaTitle,
		&retrievedArticle.MetaDescription,
		&retrievedArticle.CanonicalURL,
		&retrievedArticle.SchemaType,
	)
	if err != nil {
		t.Fatalf("Failed to retrieve test article: %v", err)
	}

	if retrievedArticle.ID != articleID {
		t.Errorf("Expected article ID %d, got %d", articleID, retrievedArticle.ID)
	}
	if retrievedArticle.Title != "Test Article" {
		t.Errorf("Expected title 'Test Article', got '%s'", retrievedArticle.Title)
	}

	// Step 7: Test analytics tables
	t.Log("Step 7: Testing analytics tables...")

	// Insert article view
	viewStmt, err := db.GetPreparedStatement(StmtInsertView)
	if err != nil {
		t.Fatalf("Failed to get insert view statement: %v", err)
	}

	_, err = viewStmt.Exec(articleID, "192.168.1.1", "Test User Agent", "https://example.com")
	if err != nil {
		t.Fatalf("Failed to insert article view: %v", err)
	}

	// Verify view was inserted
	var viewCount int
	err = db.QueryRow("SELECT COUNT(*) FROM article_views WHERE article_id = $1", articleID).Scan(&viewCount)
	if err != nil {
		t.Fatalf("Failed to count article views: %v", err)
	}
	if viewCount != 1 {
		t.Errorf("Expected 1 view, got %d", viewCount)
	}

	// Step 8: Test connection pool performance
	t.Log("Step 8: Testing connection pool performance...")
	stats := db.GetStats()
	t.Logf("Connection pool stats: Open=%d, InUse=%d, Idle=%d, MaxOpen=%d", 
		stats.OpenConnections, stats.InUse, stats.Idle, stats.MaxOpenConnections)

	if stats.MaxOpenConnections != cfg.MaxConns {
		t.Errorf("Expected max connections %d, got %d", cfg.MaxConns, stats.MaxOpenConnections)
	}

	// Step 9: Test migration rollback
	t.Log("Step 9: Testing migration rollback...")
	if err := migrator.Down(); err != nil {
		t.Fatalf("Failed to run migrations down: %v", err)
	}

	// Verify tables were dropped
	var tableExists bool
	err = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if users table exists after rollback: %v", err)
	}
	if tableExists {
		t.Error("Users table still exists after migration rollback")
	}

	t.Log("Database integration test completed successfully!")
}

// TestPgBouncerIntegration tests PgBouncer integration specifically
func TestPgBouncerIntegration(t *testing.T) {
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
		Host:     "localhost",
		Port:     6432, // PgBouncer port
		User:     "postgres",
		Password: "postgres",
		DBName:   "news_website",
		SSLMode:  "disable",
		MaxConns: 150,
		MinConns: 40,
	}

	// Test PgBouncer connection
	db, err := NewPgBouncerConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to create PgBouncer connection: %v", err)
	}
	defer db.Close()

	// Test health check through PgBouncer
	if err := db.Health(); err != nil {
		t.Fatalf("PgBouncer health check failed: %v", err)
	}

	// Test prepared statements work through PgBouncer
	stmt, err := db.GetPreparedStatement(StmtGetHomepage)
	if err != nil {
		t.Fatalf("Failed to get prepared statement through PgBouncer: %v", err)
	}
	if stmt == nil {
		t.Error("Prepared statement is nil through PgBouncer")
	}

	// Test connection pool settings for PgBouncer
	stats := db.GetStats()
	if stats.MaxOpenConnections != 200 {
		t.Errorf("Expected max connections 200 for PgBouncer, got %d", stats.MaxOpenConnections)
	}

	// Test concurrent operations through PgBouncer
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer func() { done <- true }()
			
			var result int
			err := db.QueryRow("SELECT 1").Scan(&result)
			if err != nil {
				t.Errorf("Concurrent query through PgBouncer failed: %v", err)
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}

	t.Log("PgBouncer integration test completed successfully!")
}

// BenchmarkDatabaseOperations benchmarks database operations for performance validation
func BenchmarkDatabaseOperations(b *testing.B) {
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

	db, err := NewConnection(cfg)
	if err != nil {
		b.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.Close()

	b.ResetTimer()

	b.Run("SimpleQuery", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result int
			err := db.QueryRow("SELECT 1").Scan(&result)
			if err != nil {
				b.Fatalf("Query failed: %v", err)
			}
		}
	})

	b.Run("PreparedStatement", func(b *testing.B) {
		stmt, err := db.Prepare("SELECT $1")
		if err != nil {
			b.Fatalf("Failed to prepare statement: %v", err)
		}
		defer stmt.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result int
			err := stmt.QueryRow(i).Scan(&result)
			if err != nil {
				b.Fatalf("Prepared statement query failed: %v", err)
			}
		}
	})

	b.Run("ConcurrentQueries", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var result int
				err := db.QueryRow("SELECT 1").Scan(&result)
				if err != nil {
					b.Fatalf("Concurrent query failed: %v", err)
				}
			}
		})
	})
}