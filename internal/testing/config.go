package testing

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// TestEnvironment represents a complete test environment
type TestEnvironment struct {
	DB          *sql.DB
	Cache       *MockCacheService
	Config      *TestConfig
	DataGen     *TestDataGenerator
	Validator   *TestValidator
	Cleanup     []func() error
	TempDirs    []string
}

// NewTestEnvironment creates a new test environment
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	env := &TestEnvironment{
		Cache:     NewMockCacheService(),
		Config:    GetTestConfig(),
		DataGen:   NewTestDataGenerator(),
		Validator: NewTestValidator(),
		Cleanup:   make([]func() error, 0),
		TempDirs:  make([]string, 0),
	}

	// Setup database if available
	if db := env.setupDatabase(t); db != nil {
		env.DB = db
		env.AddCleanup(func() error {
			return db.Close()
		})
	}

	// Setup cleanup on test completion
	t.Cleanup(func() {
		env.TearDown()
	})

	return env
}

// setupDatabase sets up a test database connection
func (env *TestEnvironment) setupDatabase(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", env.Config.DatabaseURL)
	if err != nil {
		t.Logf("Skipping database tests: %v", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Logf("Skipping database tests: database not available: %v", err)
		db.Close()
		return nil
	}

	// Run migrations if available
	env.runMigrations(t, db)

	return db
}

// runMigrations runs database migrations for testing
func (env *TestEnvironment) runMigrations(t *testing.T, db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		t.Logf("Warning: could not create migration driver: %v", err)
		return
	}

	// Look for migrations directory
	migrationsPath := "file://migrations"
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		// Try relative paths
		paths := []string{
			"file://../../migrations",
			"file://../../../migrations",
			"file://../../../../migrations",
		}
		
		found := false
		for _, path := range paths {
			if _, err := os.Stat(path[7:]); err == nil {
				migrationsPath = path
				found = true
				break
			}
		}
		
		if !found {
			t.Logf("Warning: migrations directory not found, skipping migrations")
			return
		}
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		t.Logf("Warning: could not create migrator: %v", err)
		return
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Logf("Warning: could not run migrations: %v", err)
	}
}

// CreateTempDir creates a temporary directory for testing
func (env *TestEnvironment) CreateTempDir(prefix string) (string, error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", err
	}
	
	env.TempDirs = append(env.TempDirs, dir)
	return dir, nil
}

// AddCleanup adds a cleanup function to be called during teardown
func (env *TestEnvironment) AddCleanup(fn func() error) {
	env.Cleanup = append(env.Cleanup, fn)
}

// TearDown cleans up the test environment
func (env *TestEnvironment) TearDown() {
	// Clean up temporary directories
	for _, dir := range env.TempDirs {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("Warning: failed to remove temp dir %s: %v", dir, err)
		}
	}

	// Run cleanup functions in reverse order
	for i := len(env.Cleanup) - 1; i >= 0; i-- {
		if err := env.Cleanup[i](); err != nil {
			log.Printf("Warning: cleanup function failed: %v", err)
		}
	}
}

// HasDatabase returns true if database is available for testing
func (env *TestEnvironment) HasDatabase() bool {
	return env.DB != nil
}

// RequireDatabase skips the test if database is not available
func (env *TestEnvironment) RequireDatabase(t *testing.T) {
	if !env.HasDatabase() {
		t.Skip("Database not available for testing")
	}
}

// CleanDatabase cleans all test data from the database
func (env *TestEnvironment) CleanDatabase(t *testing.T) {
	if !env.HasDatabase() {
		return
	}

	tables := []string{
		"article_views", "article_tags", "articles",
		"categories", "tags", "users", "comments",
		"content_sources", "push_notifications",
		"advertisements", "analytics_events",
		"email_subscribers", "monitoring_metrics",
		"widget_themes", "backup_jobs",
		"configuration_settings", "consistency_checks",
		"performance_baselines",
	}

	for _, table := range tables {
		_, err := env.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id > 0", table))
		if err != nil {
			// Some tables might not exist, that's okay
			t.Logf("Note: could not clean table %s: %v", table, err)
		}
	}
}

// TestSuite provides a structured way to organize tests
type TestSuite struct {
	Name        string
	Environment *TestEnvironment
	Tests       []TestCase
	Benchmarks  []BenchmarkCase
}

// TestCase represents a single test case
type TestCase struct {
	Name        string
	Description string
	Setup       func(*TestEnvironment) error
	Test        func(*testing.T, *TestEnvironment)
	Teardown    func(*TestEnvironment) error
	Skip        bool
	SkipReason  string
}

// BenchmarkCase represents a single benchmark case
type BenchmarkCase struct {
	Name        string
	Description string
	Setup       func(*TestEnvironment) error
	Benchmark   func(*testing.B, *TestEnvironment)
	Teardown    func(*TestEnvironment) error
}

// NewTestSuite creates a new test suite
func NewTestSuite(name string) *TestSuite {
	return &TestSuite{
		Name:       name,
		Tests:      make([]TestCase, 0),
		Benchmarks: make([]BenchmarkCase, 0),
	}
}

// AddTest adds a test case to the suite
func (ts *TestSuite) AddTest(testCase TestCase) {
	ts.Tests = append(ts.Tests, testCase)
}

// AddBenchmark adds a benchmark case to the suite
func (ts *TestSuite) AddBenchmark(benchmarkCase BenchmarkCase) {
	ts.Benchmarks = append(ts.Benchmarks, benchmarkCase)
}

// Run runs all tests in the suite
func (ts *TestSuite) Run(t *testing.T) {
	ts.Environment = NewTestEnvironment(t)

	for _, testCase := range ts.Tests {
		t.Run(testCase.Name, func(t *testing.T) {
			if testCase.Skip {
				t.Skip(testCase.SkipReason)
				return
			}

			// Setup
			if testCase.Setup != nil {
				if err := testCase.Setup(ts.Environment); err != nil {
					t.Fatalf("Test setup failed: %v", err)
				}
			}

			// Run test
			testCase.Test(t, ts.Environment)

			// Teardown
			if testCase.Teardown != nil {
				if err := testCase.Teardown(ts.Environment); err != nil {
					t.Errorf("Test teardown failed: %v", err)
				}
			}

			// Clean database after each test
			ts.Environment.CleanDatabase(t)
		})
	}
}

// RunBenchmarks runs all benchmarks in the suite
func (ts *TestSuite) RunBenchmarks(b *testing.B) {
	// Create a dummy test for environment setup
	t := &testing.T{}
	ts.Environment = NewTestEnvironment(t)

	for _, benchmarkCase := range ts.Benchmarks {
		b.Run(benchmarkCase.Name, func(b *testing.B) {
			// Setup
			if benchmarkCase.Setup != nil {
				if err := benchmarkCase.Setup(ts.Environment); err != nil {
					b.Fatalf("Benchmark setup failed: %v", err)
				}
			}

			// Run benchmark
			benchmarkCase.Benchmark(b, ts.Environment)

			// Teardown
			if benchmarkCase.Teardown != nil {
				if err := benchmarkCase.Teardown(ts.Environment); err != nil {
					b.Errorf("Benchmark teardown failed: %v", err)
				}
			}
		})
	}
}

// TestMetrics tracks test execution metrics
type TestMetrics struct {
	TotalTests     int
	PassedTests    int
	FailedTests    int
	SkippedTests   int
	TotalDuration  time.Duration
	CoveragePercent float64
}

// NewTestMetrics creates a new test metrics tracker
func NewTestMetrics() *TestMetrics {
	return &TestMetrics{}
}

// RecordTestResult records the result of a test
func (tm *TestMetrics) RecordTestResult(passed bool, skipped bool, duration time.Duration) {
	tm.TotalTests++
	tm.TotalDuration += duration

	if skipped {
		tm.SkippedTests++
	} else if passed {
		tm.PassedTests++
	} else {
		tm.FailedTests++
	}
}

// GetSuccessRate returns the test success rate as a percentage
func (tm *TestMetrics) GetSuccessRate() float64 {
	if tm.TotalTests == 0 {
		return 0
	}
	return float64(tm.PassedTests) / float64(tm.TotalTests) * 100
}

// Report generates a test metrics report
func (tm *TestMetrics) Report() string {
	return fmt.Sprintf(
		"Test Results: %d total, %d passed, %d failed, %d skipped\n"+
			"Success Rate: %.2f%%\n"+
			"Total Duration: %v\n"+
			"Coverage: %.2f%%",
		tm.TotalTests, tm.PassedTests, tm.FailedTests, tm.SkippedTests,
		tm.GetSuccessRate(),
		tm.TotalDuration,
		tm.CoveragePercent,
	)
}

// FileTestHelper provides utilities for file-based testing
type FileTestHelper struct {
	TempDir string
}

// NewFileTestHelper creates a new file test helper
func NewFileTestHelper(t *testing.T) *FileTestHelper {
	tempDir, err := os.MkdirTemp("", "filetest_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return &FileTestHelper{
		TempDir: tempDir,
	}
}

// CreateTestFile creates a test file with the given content
func (fth *FileTestHelper) CreateTestFile(filename, content string) (string, error) {
	filePath := filepath.Join(fth.TempDir, filename)
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

// ReadTestFile reads the content of a test file
func (fth *FileTestHelper) ReadTestFile(filename string) (string, error) {
	filePath := filepath.Join(fth.TempDir, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// FileExists checks if a file exists in the temp directory
func (fth *FileTestHelper) FileExists(filename string) bool {
	filePath := filepath.Join(fth.TempDir, filename)
	_, err := os.Stat(filePath)
	return err == nil
}