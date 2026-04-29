package testing

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	_ "github.com/lib/pq"
	"high-performance-news-website/internal/models"
)

// TestConfig holds configuration for test infrastructure
type TestConfig struct {
	DatabaseURL string
	CacheURL    string
	TestDataDir string
}

// GetTestConfig returns test configuration from environment or defaults
func GetTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL: getEnvOrDefault("TEST_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/news_website_test?sslmode=disable"),
		CacheURL:    getEnvOrDefault("TEST_CACHE_URL", "redis://localhost:6379/1"),
		TestDataDir: getEnvOrDefault("TEST_DATA_DIR", "./testdata"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// MockCacheService provides a unified mock cache service for all tests
type MockCacheService struct {
	mock.Mock
	data map[string][]byte
	ttls map[string]time.Time
	mu   sync.RWMutex
}

// NewMockCacheService creates a new mock cache service
func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		data: make(map[string][]byte),
		ttls: make(map[string]time.Time),
	}
}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	
	// If mock expectations are set, use them
	if len(args) > 0 {
		return args.Get(0).([]byte), args.Error(1)
	}
	
	// Otherwise use internal data store
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if ttl, exists := m.ttls[key]; exists && time.Now().After(ttl) {
		delete(m.data, key)
		delete(m.ttls, key)
		return nil, fmt.Errorf("key not found")
	}
	
	if data, exists := m.data[key]; exists {
		return data, nil
	}
	
	return nil, fmt.Errorf("key not found")
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	
	// If mock expectations are set, use them
	if len(args) > 0 {
		return args.Error(0)
	}
	
	// Otherwise use internal data store
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.data[key] = value
	if ttl > 0 {
		m.ttls[key] = time.Now().Add(ttl)
	}
	
	return nil
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	
	// If mock expectations are set, use them
	if len(args) > 0 {
		return args.Error(0)
	}
	
	// Otherwise use internal data store
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.data, key)
	delete(m.ttls, key)
	
	return nil
}

func (m *MockCacheService) DeletePattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	
	// If mock expectations are set, use them
	if len(args) > 0 {
		return args.Error(0)
	}
	
	// Simple pattern matching for testing (just prefix matching)
	m.mu.Lock()
	defer m.mu.Unlock()
	
	prefix := pattern
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix = pattern[:len(pattern)-1]
	}
	
	for key := range m.data {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(m.data, key)
			delete(m.ttls, key)
		}
	}
	
	return nil
}

func (m *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	
	// If mock expectations are set, use them
	if len(args) > 0 {
		return args.Bool(0), args.Error(1)
	}
	
	// Otherwise use internal data store
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if ttl, exists := m.ttls[key]; exists && time.Now().After(ttl) {
		delete(m.data, key)
		delete(m.ttls, key)
		return false, nil
	}
	
	_, exists := m.data[key]
	return exists, nil
}

func (m *MockCacheService) Close() error {
	args := m.Called()
	if len(args) > 0 {
		return args.Error(0)
	}
	return nil
}

func (m *MockCacheService) Health(ctx context.Context) error {
	args := m.Called(ctx)
	if len(args) > 0 {
		return args.Error(0)
	}
	return nil
}

// Clear clears all data from the mock cache
func (m *MockCacheService) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.data = make(map[string][]byte)
	m.ttls = make(map[string]time.Time)
}

// TestDatabase provides database setup and teardown for tests
type TestDatabase struct {
	DB     *sql.DB
	config *TestConfig
}

// SetupTestDatabase creates a test database connection
func SetupTestDatabase(t *testing.T) *TestDatabase {
	config := GetTestConfig()
	
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		t.Skipf("Skipping database tests: %v", err)
		return nil
	}
	
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping database tests: database not available: %v", err)
		return nil
	}
	
	return &TestDatabase{
		DB:     db,
		config: config,
	}
}

// Close closes the test database connection
func (td *TestDatabase) Close() error {
	if td.DB != nil {
		return td.DB.Close()
	}
	return nil
}

// Cleanup performs test cleanup operations
func (td *TestDatabase) Cleanup(t *testing.T) {
	if td.DB == nil {
		return
	}
	
	// Clean up test data
	tables := []string{
		"article_views", "article_tags", "articles", 
		"categories", "tags", "users", "comments",
		"content_sources", "push_notifications",
	}
	
	for _, table := range tables {
		_, err := td.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id > 0", table))
		if err != nil {
			t.Logf("Warning: failed to clean table %s: %v", table, err)
		}
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct {
	counter int64
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{counter: 1}
}

// GenerateTestArticle creates a test article with realistic data
func (g *TestDataGenerator) GenerateTestArticle() *models.Article {
	g.counter++
	now := time.Now()
	
	return &models.Article{
		Title:      fmt.Sprintf("Test Article %d", g.counter),
		Slug:       fmt.Sprintf("test-article-%d", g.counter),
		Content:    fmt.Sprintf("This is test content for article %d with sufficient length to test various functionality.", g.counter),
		Excerpt:    fmt.Sprintf("Test excerpt for article %d", g.counter),
		AuthorID:   1,
		CategoryID: 1,
		Status:     "published",
		PublishedAt: &now,
		SEOData: models.SEOData{
			MetaTitle:       fmt.Sprintf("Test Article %d - Meta Title", g.counter),
			MetaDescription: fmt.Sprintf("Meta description for test article %d", g.counter),
			SchemaType:      "NewsArticle",
		},
	}
}

// GenerateTestUser creates a test user
func (g *TestDataGenerator) GenerateTestUser() *models.User {
	g.counter++
	
	return &models.User{
		Username:  fmt.Sprintf("testuser%d", g.counter),
		Email:     fmt.Sprintf("test%d@example.com", g.counter),
		FirstName: fmt.Sprintf("Test%d", g.counter),
		LastName:  "User",
		Role:      "author",
		IsActive:  true,
	}
}

// GenerateTestCategory creates a test category
func (g *TestDataGenerator) GenerateTestCategory() *models.Category {
	g.counter++
	
	return &models.Category{
		Name:        fmt.Sprintf("Test Category %d", g.counter),
		Slug:        fmt.Sprintf("test-category-%d", g.counter),
		Description: fmt.Sprintf("Description for test category %d", g.counter),
		SortOrder:   int(g.counter),
	}
}

// GenerateTestTag creates a test tag
func (g *TestDataGenerator) GenerateTestTag() *models.Tag {
	g.counter++
	
	return &models.Tag{
		Name:        fmt.Sprintf("test-tag-%d", g.counter),
		Slug:        fmt.Sprintf("test-tag-%d", g.counter),
		Description: fmt.Sprintf("Description for test tag %d", g.counter),
	}
}

// TestCoverageTracker tracks test coverage metrics
type TestCoverageTracker struct {
	TestedFunctions map[string]bool
	TotalFunctions  int
	mu              sync.RWMutex
}

// NewTestCoverageTracker creates a new coverage tracker
func NewTestCoverageTracker() *TestCoverageTracker {
	return &TestCoverageTracker{
		TestedFunctions: make(map[string]bool),
	}
}

// MarkFunctionTested marks a function as tested
func (t *TestCoverageTracker) MarkFunctionTested(functionName string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.TestedFunctions[functionName] = true
}

// GetCoveragePercentage returns the current coverage percentage
func (t *TestCoverageTracker) GetCoveragePercentage() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.TotalFunctions == 0 {
		return 0
	}
	
	return float64(len(t.TestedFunctions)) / float64(t.TotalFunctions) * 100
}

// AssertMinimumCoverage fails the test if coverage is below the minimum
func (t *TestCoverageTracker) AssertMinimumCoverage(test *testing.T, minimum float64) {
	coverage := t.GetCoveragePercentage()
	if coverage < minimum {
		test.Errorf("Test coverage %.2f%% is below minimum required %.2f%%", coverage, minimum)
	}
}

// BenchmarkHelper provides utilities for benchmark tests
type BenchmarkHelper struct {
	DataGenerator *TestDataGenerator
	Cache         *MockCacheService
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper() *BenchmarkHelper {
	return &BenchmarkHelper{
		DataGenerator: NewTestDataGenerator(),
		Cache:         NewMockCacheService(),
	}
}

// SetupBenchmarkData creates test data for benchmarks
func (b *BenchmarkHelper) SetupBenchmarkData(count int) []*models.Article {
	articles := make([]*models.Article, count)
	for i := 0; i < count; i++ {
		articles[i] = b.DataGenerator.GenerateTestArticle()
	}
	return articles
}

// TestValidator provides validation utilities for tests
type TestValidator struct{}

// NewTestValidator creates a new test validator
func NewTestValidator() *TestValidator {
	return &TestValidator{}
}

// ValidateArticleFields validates that all required article fields are set
func (v *TestValidator) ValidateArticleFields(t *testing.T, article *models.Article) {
	if article.Title == "" {
		t.Error("Article title should not be empty")
	}
	if article.Slug == "" {
		t.Error("Article slug should not be empty")
	}
	if article.Content == "" {
		t.Error("Article content should not be empty")
	}
	if article.AuthorID == 0 {
		t.Error("Article author ID should be set")
	}
	if article.CategoryID == 0 {
		t.Error("Article category ID should be set")
	}
	if article.Status == "" {
		t.Error("Article status should be set")
	}
}

// ValidateUserFields validates that all required user fields are set
func (v *TestValidator) ValidateUserFields(t *testing.T, user *models.User) {
	if user.Username == "" {
		t.Error("User username should not be empty")
	}
	if user.Email == "" {
		t.Error("User email should not be empty")
	}
	if user.FirstName == "" {
		t.Error("User first name should not be empty")
	}
	if user.Role == "" {
		t.Error("User role should be set")
	}
}

// ValidateCategoryFields validates that all required category fields are set
func (v *TestValidator) ValidateCategoryFields(t *testing.T, category *models.Category) {
	if category.Name == "" {
		t.Error("Category name should not be empty")
	}
	if category.Slug == "" {
		t.Error("Category slug should not be empty")
	}
}

// ValidateTagFields validates that all required tag fields are set
func (v *TestValidator) ValidateTagFields(t *testing.T, tag *models.Tag) {
	if tag.Name == "" {
		t.Error("Tag name should not be empty")
	}
	if tag.Slug == "" {
		t.Error("Tag slug should not be empty")
	}
}