package testing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
)

// TestDataInvariantProperties runs all data invariant property tests
func TestDataInvariantProperties(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Close()
	defer testDB.Cleanup(t)

	// Setup mock cache
	mockCache := NewMockCacheService()
	defer mockCache.Close()

	// Create property test config
	config := &PropertyTestConfig{
		Iterations:     50, // Reduced for faster testing
		MaxDataSize:    100,
		Timeout:        10 * time.Second,
		ShrinkAttempts: 5,
		RandomSeed:     12345, // Fixed seed for reproducible tests
	}

	// Create data invariant tester
	tester := NewDataInvariantTester(testDB.DB, mockCache, config)

	t.Run("PartitionDataConsistency", func(t *testing.T) {
		result := tester.TestPartitionDataConsistency(t)
		
		assert.True(t, result.Passed, "Partition data consistency property should pass")
		assert.Equal(t, "partition_data_consistency", result.Property)
		assert.Equal(t, config.Iterations, result.Iterations)
		
		if !result.Passed {
			t.Logf("Failed at iteration %d: %s", result.FailedIteration, result.FailureReason)
			t.Logf("Counter example: %+v", result.CounterExample)
		}
		
		t.Logf("Property test completed in %v", result.Duration)
	})

	t.Run("CacheInvalidationCorrectness", func(t *testing.T) {
		result := tester.TestCacheInvalidationCorrectness(t)
		
		assert.True(t, result.Passed, "Cache invalidation correctness property should pass")
		assert.Equal(t, "cache_invalidation_correctness", result.Property)
		
		if !result.Passed {
			t.Logf("Failed at iteration %d: %s", result.FailedIteration, result.FailureReason)
			t.Logf("Counter example: %+v", result.CounterExample)
		}
		
		t.Logf("Property test completed in %v", result.Duration)
	})

	t.Run("SEOMetadataConsistency", func(t *testing.T) {
		result := tester.TestSEOMetadataConsistency(t)
		
		assert.True(t, result.Passed, "SEO metadata consistency property should pass")
		assert.Equal(t, "seo_metadata_consistency", result.Property)
		
		if !result.Passed {
			t.Logf("Failed at iteration %d: %s", result.FailedIteration, result.FailureReason)
			t.Logf("Counter example: %+v", result.CounterExample)
		}
		
		t.Logf("Property test completed in %v", result.Duration)
	})

	t.Run("UserPermissionInvariants", func(t *testing.T) {
		result := tester.TestUserPermissionInvariants(t)
		
		assert.True(t, result.Passed, "User permission invariants property should pass")
		assert.Equal(t, "user_permission_invariants", result.Property)
		
		if !result.Passed {
			t.Logf("Failed at iteration %d: %s", result.FailedIteration, result.FailureReason)
			t.Logf("Counter example: %+v", result.CounterExample)
		}
		
		t.Logf("Property test completed in %v", result.Duration)
	})
}

// TestPropertyTestConfiguration tests the property testing configuration
func TestPropertyTestConfiguration(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultPropertyTestConfig()
		
		assert.Equal(t, 100, config.Iterations)
		assert.Equal(t, 1000, config.MaxDataSize)
		assert.Equal(t, 30*time.Second, config.Timeout)
		assert.Equal(t, 10, config.ShrinkAttempts)
		assert.NotZero(t, config.RandomSeed)
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := &PropertyTestConfig{
			Iterations:     25,
			MaxDataSize:    500,
			Timeout:        15 * time.Second,
			ShrinkAttempts: 5,
			RandomSeed:     54321,
		}

		testDB := SetupTestDatabase(t)
		if testDB == nil {
			t.Skip("Database not available for testing")
		}
		defer testDB.Close()

		mockCache := NewMockCacheService()
		defer mockCache.Close()

		tester := NewDataInvariantTester(testDB.DB, mockCache, config)
		
		assert.Equal(t, config, tester.config)
		assert.NotNil(t, tester.generator)
		assert.NotNil(t, tester.rand)
	})
}

// TestPropertyTestDataGeneration tests the data generation for property tests
func TestPropertyTestDataGeneration(t *testing.T) {
	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Close()
	defer testDB.Cleanup(t)

	mockCache := NewMockCacheService()
	defer mockCache.Close()

	config := &PropertyTestConfig{
		Iterations:  10,
		RandomSeed:  12345,
	}

	tester := NewDataInvariantTester(testDB.DB, mockCache, config)

	t.Run("PartitionedArticleGeneration", func(t *testing.T) {
		articles := tester.generatePartitionedArticles(5)
		
		require.Len(t, articles, 5)
		
		for _, article := range articles {
			assert.NotEmpty(t, article.Title)
			assert.NotEmpty(t, article.Slug)
			assert.NotEmpty(t, article.Content)
			assert.NotZero(t, article.AuthorID)
			assert.NotZero(t, article.CategoryID)
			assert.NotNil(t, article.PublishedAt)
			assert.Equal(t, "fa", article.LanguageCode) // Default language
		}
		
		// Verify articles are distributed across different time periods
		times := make(map[string]bool)
		for _, article := range articles {
			dateKey := article.PublishedAt.Format("2006-01-02")
			times[dateKey] = true
		}
		
		// Should have some variety in dates (not all the same day)
		t.Logf("Generated articles across %d different dates", len(times))
	})

	t.Run("UserRoleGeneration", func(t *testing.T) {
		roles := []string{"admin", "editor", "reporter", "contributor"}
		
		for _, role := range roles {
			user := tester.generateUserWithRole(models.UserRole(role))
			
			assert.NotEmpty(t, user.Username)
			assert.NotEmpty(t, user.Email)
			assert.Equal(t, models.UserRole(role), user.Role)
			assert.True(t, user.IsActive)
		}
	})
}

// TestPropertyTestHelpers tests the helper methods used in property testing
func TestPropertyTestHelpers(t *testing.T) {
	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Close()
	defer testDB.Cleanup(t)

	mockCache := NewMockCacheService()
	defer mockCache.Close()

	config := DefaultPropertyTestConfig()
	tester := NewDataInvariantTester(testDB.DB, mockCache, config)

	t.Run("ArticleInsertionAndCleanup", func(t *testing.T) {
		article := tester.generator.GenerateTestArticle()
		
		// Test insertion
		err := tester.insertTestArticle(article)
		require.NoError(t, err)
		assert.NotZero(t, article.ID)
		assert.False(t, article.CreatedAt.IsZero())
		
		// Verify article exists in database
		var count int
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM articles WHERE id = $1", article.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		
		// Test cleanup
		tester.cleanupTestArticles([]uint64{article.ID})
		
		// Verify article is removed
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM articles WHERE id = $1", article.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("UserInsertionAndCleanup", func(t *testing.T) {
		user := tester.generateUserWithRole(models.RoleEditor)
		
		// Test insertion
		err := tester.insertTestUser(user)
		require.NoError(t, err)
		assert.NotZero(t, user.ID)
		assert.False(t, user.CreatedAt.IsZero())
		
		// Verify user exists in database
		var count int
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", user.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		
		// Test cleanup
		tester.cleanupTestUsers([]uint64{user.ID})
		
		// Verify user is removed
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", user.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("CacheOperations", func(t *testing.T) {
		article := tester.generator.GenerateTestArticle()
		article.ID = 123
		
		cacheKey := "test:article:123"
		
		// Test caching
		err := tester.cacheArticle(cacheKey, article)
		require.NoError(t, err)
		
		// Verify cached data exists
		exists, err := mockCache.Exists(nil, cacheKey)
		require.NoError(t, err)
		assert.True(t, exists)
		
		// Test cache retrieval
		cached, err := mockCache.Get(nil, cacheKey)
		require.NoError(t, err)
		assert.NotNil(t, cached)
		assert.Contains(t, string(cached), "123")
		assert.Contains(t, string(cached), article.Title)
	})
}

// BenchmarkPropertyTesting benchmarks the property testing performance
func BenchmarkPropertyTesting(b *testing.B) {
	testDB := SetupTestDatabase(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available for benchmarking")
	}
	defer testDB.Close()

	mockCache := NewMockCacheService()
	defer mockCache.Close()

	config := &PropertyTestConfig{
		Iterations:  10, // Reduced for benchmarking
		RandomSeed:  12345,
	}

	tester := NewDataInvariantTester(testDB.DB, mockCache, config)

	b.Run("PartitionConsistency", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := tester.TestPartitionDataConsistency(&testing.T{})
			if !result.Passed {
				b.Fatalf("Property test failed: %s", result.FailureReason)
			}
		}
	})

	b.Run("CacheInvalidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := tester.TestCacheInvalidationCorrectness(&testing.T{})
			if !result.Passed {
				b.Fatalf("Property test failed: %s", result.FailureReason)
			}
		}
	})
}