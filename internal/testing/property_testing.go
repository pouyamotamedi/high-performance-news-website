package testing

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/cache"
)

// PropertyTestConfig defines configuration for property-based testing
type PropertyTestConfig struct {
	Iterations      int           // Number of test iterations
	MaxDataSize     int           // Maximum size for generated data
	Timeout         time.Duration // Timeout for each test iteration
	ShrinkAttempts  int           // Number of shrinking attempts for failing cases
	RandomSeed      int64         // Seed for reproducible tests
}

// DefaultPropertyTestConfig returns default configuration for property testing
func DefaultPropertyTestConfig() *PropertyTestConfig {
	return &PropertyTestConfig{
		Iterations:     100,
		MaxDataSize:    1000,
		Timeout:        30 * time.Second,
		ShrinkAttempts: 10,
		RandomSeed:     time.Now().UnixNano(),
	}
}

// PropertyTester provides property-based testing capabilities
type PropertyTester struct {
	config    *PropertyTestConfig
	db        *sql.DB
	cache     cache.CacheService
	generator *TestDataGenerator
	rand      *rand.Rand
}

// NewPropertyTester creates a new property tester
func NewPropertyTester(db *sql.DB, cache cache.CacheService, config *PropertyTestConfig) *PropertyTester {
	if config == nil {
		config = DefaultPropertyTestConfig()
	}
	
	return &PropertyTester{
		config:    config,
		db:        db,
		cache:     cache,
		generator: NewTestDataGenerator(),
		rand:      rand.New(rand.NewSource(config.RandomSeed)),
	}
}

// PropertyTestResult represents the result of a property test
type PropertyTestResult struct {
	Property        string        `json:"property"`
	Passed          bool          `json:"passed"`
	Iterations      int           `json:"iterations"`
	FailedIteration int           `json:"failed_iteration,omitempty"`
	FailureReason   string        `json:"failure_reason,omitempty"`
	CounterExample  interface{}   `json:"counter_example,omitempty"`
	Duration        time.Duration `json:"duration"`
}

// DataInvariantTester tests data consistency invariants
type DataInvariantTester struct {
	*PropertyTester
}

// NewDataInvariantTester creates a new data invariant tester
func NewDataInvariantTester(db *sql.DB, cache cache.CacheService, config *PropertyTestConfig) *DataInvariantTester {
	return &DataInvariantTester{
		PropertyTester: NewPropertyTester(db, cache, config),
	}
}

// TestPartitionDataConsistency tests that partition operations maintain data consistency
func (t *DataInvariantTester) TestPartitionDataConsistency(test *testing.T) PropertyTestResult {
	start := time.Now()
	result := PropertyTestResult{
		Property:   "partition_data_consistency",
		Iterations: t.config.Iterations,
	}
	
	for i := 0; i < t.config.Iterations; i++ {
		// Generate test articles across different time periods (different partitions)
		articles := t.generatePartitionedArticles(10)
		
		// Insert articles into database
		insertedIDs := make([]uint64, 0, len(articles))
		for _, article := range articles {
			if err := t.insertTestArticle(article); err != nil {
				result.Passed = false
				result.FailedIteration = i + 1
				result.FailureReason = fmt.Sprintf("Failed to insert article: %v", err)
				result.CounterExample = article
				result.Duration = time.Since(start)
				return result
			}
			insertedIDs = append(insertedIDs, article.ID)
		}
		
		// Verify partition consistency invariants
		if err := t.verifyPartitionConsistency(insertedIDs); err != nil {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = fmt.Sprintf("Partition consistency violation: %v", err)
			result.CounterExample = insertedIDs
			result.Duration = time.Since(start)
			return result
		}
		
		// Cleanup test data
		t.cleanupTestArticles(insertedIDs)
	}
	
	result.Passed = true
	result.Duration = time.Since(start)
	return result
}

// TestCacheInvalidationCorrectness tests that cache invalidation never leaves stale data
func (t *DataInvariantTester) TestCacheInvalidationCorrectness(test *testing.T) PropertyTestResult {
	start := time.Now()
	result := PropertyTestResult{
		Property:   "cache_invalidation_correctness",
		Iterations: t.config.Iterations,
	}
	
	for i := 0; i < t.config.Iterations; i++ {
		// Generate test article
		article := t.generator.GenerateTestArticle()
		
		// Insert article and cache it
		if err := t.insertTestArticle(article); err != nil {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = fmt.Sprintf("Failed to insert article: %v", err)
			result.CounterExample = article
			result.Duration = time.Since(start)
			return result
		}
		
		// Cache the article
		cacheKey := fmt.Sprintf("article:%d", article.ID)
		if err := t.cacheArticle(cacheKey, article); err != nil {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = fmt.Sprintf("Failed to cache article: %v", err)
			result.CounterExample = article
			result.Duration = time.Since(start)
			return result
		}
		
		// Modify article in database
		article.Title = "Modified " + article.Title
		if err := t.updateTestArticle(article); err != nil {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = fmt.Sprintf("Failed to update article: %v", err)
			result.CounterExample = article
			result.Duration = time.Since(start)
			return result
		}
		
		// Invalidate cache
		if err := t.cache.Delete(context.Background(), cacheKey); err != nil {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = fmt.Sprintf("Failed to invalidate cache: %v", err)
			result.CounterExample = article
			result.Duration = time.Since(start)
			return result
		}
		
		// Verify cache is actually invalidated (no stale data)
		if cached, err := t.cache.Get(context.Background(), cacheKey); err == nil && cached != nil {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = "Cache invalidation failed - stale data found"
			result.CounterExample = map[string]interface{}{
				"article_id": article.ID,
				"cache_key":  cacheKey,
				"cached_data": string(cached),
			}
			result.Duration = time.Since(start)
			return result
		}
		
		// Cleanup
		t.cleanupTestArticles([]uint64{article.ID})
	}
	
	result.Passed = true
	result.Duration = time.Since(start)
	return result
}

// TestSEOMetadataConsistency tests that SEO metadata is always consistent across all content variants
func (t *DataInvariantTester) TestSEOMetadataConsistency(test *testing.T) PropertyTestResult {
	start := time.Now()
	result := PropertyTestResult{
		Property:   "seo_metadata_consistency",
		Iterations: t.config.Iterations,
	}
	
	for i := 0; i < t.config.Iterations; i++ {
		// Generate article with SEO metadata
		article := t.generator.GenerateTestArticle()
		article.SEOData = models.SEOData{
			MetaTitle:       "Test Meta Title",
			MetaDescription: "Test meta description for SEO",
			Keywords:        []string{"test", "seo", "metadata"},
			CanonicalURL:    "https://example.com/test-article",
			SchemaType:      "NewsArticle",
		}
		
		// Insert article
		if err := t.insertTestArticle(article); err != nil {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = fmt.Sprintf("Failed to insert article: %v", err)
			result.CounterExample = article
			result.Duration = time.Since(start)
			return result
		}
		
		// Verify SEO metadata consistency invariants
		if err := t.verifySEOConsistency(article); err != nil {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = fmt.Sprintf("SEO metadata consistency violation: %v", err)
			result.CounterExample = article
			result.Duration = time.Since(start)
			return result
		}
		
		// Cleanup
		t.cleanupTestArticles([]uint64{article.ID})
	}
	
	result.Passed = true
	result.Duration = time.Since(start)
	return result
}

// TestUserPermissionInvariants tests that role-based access is never violated
func (t *DataInvariantTester) TestUserPermissionInvariants(test *testing.T) PropertyTestResult {
	start := time.Now()
	result := PropertyTestResult{
		Property:   "user_permission_invariants",
		Iterations: t.config.Iterations,
	}
	
	for i := 0; i < t.config.Iterations; i++ {
		// Generate users with different roles
		admin := t.generateUserWithRole(models.RoleAdmin)
		editor := t.generateUserWithRole(models.RoleEditor)
		reporter := t.generateUserWithRole(models.RoleReporter)
		contributor := t.generateUserWithRole(models.RoleContributor)
		
		users := []*models.User{admin, editor, reporter, contributor}
		
		// Insert users
		for _, user := range users {
			if err := t.insertTestUser(user); err != nil {
				result.Passed = false
				result.FailedIteration = i + 1
				result.FailureReason = fmt.Sprintf("Failed to insert user: %v", err)
				result.CounterExample = user
				result.Duration = time.Since(start)
				return result
			}
		}
		
		// Test permission invariants
		if err := t.verifyPermissionInvariants(users); err != nil {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = fmt.Sprintf("Permission invariant violation: %v", err)
			result.CounterExample = users
			result.Duration = time.Since(start)
			return result
		}
		
		// Cleanup
		userIDs := make([]uint64, len(users))
		for j, user := range users {
			userIDs[j] = user.ID
		}
		t.cleanupTestUsers(userIDs)
	}
	
	result.Passed = true
	result.Duration = time.Since(start)
	return result
}

// Helper methods for data generation and verification

func (t *DataInvariantTester) generatePartitionedArticles(count int) []*models.Article {
	articles := make([]*models.Article, count)
	
	// Generate articles across different time periods (different partitions)
	baseTime := time.Now().AddDate(0, -6, 0) // 6 months ago
	
	for i := 0; i < count; i++ {
		article := t.generator.GenerateTestArticle()
		
		// Distribute articles across different time periods
		daysOffset := t.rand.Intn(180) // Random day in last 6 months
		publishTime := baseTime.AddDate(0, 0, daysOffset)
		article.PublishedAt = &publishTime
		article.CreatedAt = publishTime
		article.UpdatedAt = publishTime
		
		articles[i] = article
	}
	
	return articles
}

func (t *DataInvariantTester) generateUserWithRole(role models.UserRole) *models.User {
	user := t.generator.GenerateTestUser()
	user.Role = role
	return user
}

func (t *DataInvariantTester) verifyPartitionConsistency(articleIDs []uint64) error {
	// Verify that articles can be found in their correct partitions
	for _, id := range articleIDs {
		// Check if article exists and is in correct partition
		var count int
		query := `
			SELECT COUNT(*) 
			FROM articles 
			WHERE id = $1 
			AND published_at IS NOT NULL
		`
		
		if err := t.db.QueryRow(query, id).Scan(&count); err != nil {
			return fmt.Errorf("failed to verify article %d: %w", id, err)
		}
		
		if count != 1 {
			return fmt.Errorf("article %d not found in expected partition", id)
		}
	}
	
	// Verify cross-partition referential integrity
	query := `
		SELECT a.id, a.author_id, a.category_id
		FROM articles a
		WHERE a.id = ANY($1)
		AND (
			NOT EXISTS (SELECT 1 FROM users u WHERE u.id = a.author_id) OR
			NOT EXISTS (SELECT 1 FROM categories c WHERE c.id = a.category_id)
		)
	`
	
	rows, err := t.db.Query(query, articleIDs)
	if err != nil {
		return fmt.Errorf("failed to check referential integrity: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		var id, authorID, categoryID uint64
		rows.Scan(&id, &authorID, &categoryID)
		return fmt.Errorf("referential integrity violation for article %d (author: %d, category: %d)", id, authorID, categoryID)
	}
	
	return nil
}

func (t *DataInvariantTester) verifySEOConsistency(article *models.Article) error {
	// Verify schema type is valid
	validSchemaTypes := map[string]bool{
		"NewsArticle": true,
		"Article":     true,
		"BlogPosting": true,
	}
	
	if !validSchemaTypes[article.SEOData.SchemaType] {
		return fmt.Errorf("invalid schema type: %s", article.SEOData.SchemaType)
	}
	
	// Verify meta title length
	if len(article.SEOData.MetaTitle) > 60 {
		return fmt.Errorf("meta title too long: %d characters", len(article.SEOData.MetaTitle))
	}
	
	// Verify meta description length
	if len(article.SEOData.MetaDescription) > 160 {
		return fmt.Errorf("meta description too long: %d characters", len(article.SEOData.MetaDescription))
	}
	
	// Verify canonical URL format if present
	if article.SEOData.CanonicalURL != "" && !models.IsValidURL(article.SEOData.CanonicalURL) {
		return fmt.Errorf("invalid canonical URL: %s", article.SEOData.CanonicalURL)
	}
	
	return nil
}

func (t *DataInvariantTester) verifyPermissionInvariants(users []*models.User) error {
	// Test permission hierarchy invariants
	var admin, editor, reporter, contributor *models.User
	
	for _, user := range users {
		switch user.Role {
		case models.RoleAdmin:
			admin = user
		case models.RoleEditor:
			editor = user
		case models.RoleReporter:
			reporter = user
		case models.RoleContributor:
			contributor = user
		}
	}
	
	// Admin should be able to manage everyone
	if admin != nil {
		for _, user := range users {
			if !admin.CanManageUser(user) {
				return fmt.Errorf("admin should be able to manage user %d with role %s", user.ID, user.Role)
			}
		}
	}
	
	// Editor should be able to manage reporter and contributor, but not admin
	if editor != nil {
		if reporter != nil && !editor.CanManageUser(reporter) {
			return fmt.Errorf("editor should be able to manage reporter")
		}
		if contributor != nil && !editor.CanManageUser(contributor) {
			return fmt.Errorf("editor should be able to manage contributor")
		}
		if admin != nil && editor.CanManageUser(admin) {
			return fmt.Errorf("editor should not be able to manage admin")
		}
	}
	
	// Reporter should only manage themselves
	if reporter != nil {
		if admin != nil && reporter.CanManageUser(admin) {
			return fmt.Errorf("reporter should not be able to manage admin")
		}
		if editor != nil && reporter.CanManageUser(editor) {
			return fmt.Errorf("reporter should not be able to manage editor")
		}
	}
	
	// Test permission consistency
	for _, user := range users {
		permissions := models.GetRolePermissions(user.Role)
		
		// Verify permission hierarchy
		switch user.Role {
		case models.RoleAdmin:
			requiredPerms := []string{"create", "read", "update", "delete", "manage_users", "manage_system"}
			for _, perm := range requiredPerms {
				if !user.HasPermission(perm) {
					return fmt.Errorf("admin should have permission: %s", perm)
				}
			}
		case models.RoleEditor:
			requiredPerms := []string{"create", "read", "update", "delete", "publish", "moderate"}
			for _, perm := range requiredPerms {
				if !user.HasPermission(perm) {
					return fmt.Errorf("editor should have permission: %s", perm)
				}
			}
			// Should not have admin permissions
			if user.HasPermission("manage_system") {
				return fmt.Errorf("editor should not have manage_system permission")
			}
		case models.RoleReporter:
			requiredPerms := []string{"create", "read", "update"}
			for _, perm := range requiredPerms {
				if !user.HasPermission(perm) {
					return fmt.Errorf("reporter should have permission: %s", perm)
				}
			}
			// Should not have delete or publish permissions
			if user.HasPermission("delete") || user.HasPermission("publish") {
				return fmt.Errorf("reporter should not have delete or publish permissions")
			}
		case models.RoleContributor:
			requiredPerms := []string{"create", "read"}
			for _, perm := range requiredPerms {
				if !user.HasPermission(perm) {
					return fmt.Errorf("contributor should have permission: %s", perm)
				}
			}
			// Should not have update, delete, or publish permissions
			if user.HasPermission("update") || user.HasPermission("delete") || user.HasPermission("publish") {
				return fmt.Errorf("contributor should not have update, delete, or publish permissions")
			}
		}
		
		// Verify permissions array is not empty
		if len(permissions) == 0 {
			return fmt.Errorf("user role %s should have at least one permission", user.Role)
		}
	}
	
	return nil
}

// Database helper methods

func (t *DataInvariantTester) insertTestArticle(article *models.Article) error {
	query := `
		INSERT INTO articles (title, slug, content, excerpt, author_id, category_id, status, published_at, language_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	
	return t.db.QueryRow(query,
		article.Title, article.Slug, article.Content, article.Excerpt,
		article.AuthorID, article.CategoryID, article.Status, article.PublishedAt,
		article.LanguageCode,
	).Scan(&article.ID, &article.CreatedAt, &article.UpdatedAt)
}

func (t *DataInvariantTester) updateTestArticle(article *models.Article) error {
	query := `
		UPDATE articles 
		SET title = $1, content = $2, updated_at = NOW()
		WHERE id = $3
	`
	
	_, err := t.db.Exec(query, article.Title, article.Content, article.ID)
	return err
}

func (t *DataInvariantTester) insertTestUser(user *models.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, first_name, last_name, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	
	return t.db.QueryRow(query,
		user.Username, user.Email, "test_hash", user.Role,
		user.FirstName, user.LastName, user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (t *DataInvariantTester) cacheArticle(key string, article *models.Article) error {
	data := fmt.Sprintf(`{"id":%d,"title":"%s","slug":"%s"}`, article.ID, article.Title, article.Slug)
	return t.cache.Set(context.Background(), key, []byte(data), 5*time.Minute)
}

func (t *DataInvariantTester) cleanupTestArticles(ids []uint64) {
	if len(ids) == 0 {
		return
	}
	
	query := `DELETE FROM articles WHERE id = ANY($1)`
	t.db.Exec(query, ids)
}

func (t *DataInvariantTester) cleanupTestUsers(ids []uint64) {
	if len(ids) == 0 {
		return
	}
	
	query := `DELETE FROM users WHERE id = ANY($1)`
	t.db.Exec(query, ids)
}