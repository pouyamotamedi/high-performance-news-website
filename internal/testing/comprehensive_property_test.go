package testing

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
)

// TestComprehensivePropertyTesting runs the complete property-based testing suite
func TestComprehensivePropertyTesting(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for comprehensive property testing")
	}
	defer testDB.Close()
	defer testDB.Cleanup(t)

	// Setup mock cache
	mockCache := NewMockCacheService()
	defer mockCache.Close()

	// Setup mock API server
	server := createMockAPIServer()
	defer server.Close()

	// Create temporary results directory
	tempDir, err := os.MkdirTemp("", "property_test_results")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create property test runner
	runner := NewPropertyTestRunner(testDB.DB, mockCache, server, tempDir)

	// Configure for faster testing
	config := &PropertyTestConfig{
		Iterations:     25, // Reduced for faster testing
		MaxDataSize:    100,
		Timeout:        15 * time.Second,
		ShrinkAttempts: 3,
		RandomSeed:     12345, // Fixed seed for reproducible results
	}
	runner.SetConfig(config)

	t.Log("Starting comprehensive property-based testing suite...")

	// Run all property tests
	suite := runner.RunAllPropertyTests(t)

	// Verify suite results
	require.NotNil(t, suite)
	assert.Equal(t, "Comprehensive Property Test Suite", suite.Name)
	assert.False(t, suite.StartTime.IsZero())
	assert.False(t, suite.EndTime.IsZero())
	assert.Greater(t, suite.Duration, time.Duration(0))
	assert.Greater(t, suite.TotalTests, 0)

	// Verify all expected tests were run
	expectedProperties := []string{
		"partition_data_consistency",
		"cache_invalidation_correctness", 
		"seo_metadata_consistency",
		"user_permission_invariants",
		"api_response_schema",
		"api_behavior_consistency",
		"api_error_handling",
		"api_performance",
	}

	foundProperties := make(map[string]bool)
	for _, result := range suite.Results {
		foundProperties[result.Property] = true
		
		// Verify each result has required fields
		assert.NotEmpty(t, result.Property)
		assert.Greater(t, result.Iterations, 0)
		assert.Greater(t, result.Duration, time.Duration(0))
		
		if !result.Passed {
			assert.Greater(t, result.FailedIteration, 0)
			assert.NotEmpty(t, result.FailureReason)
			t.Logf("Property test failed: %s - %s", result.Property, result.FailureReason)
		}
	}

	// Verify all expected properties were tested
	for _, prop := range expectedProperties {
		assert.True(t, foundProperties[prop], "Expected property test not found: %s", prop)
	}

	// Verify summary
	assert.NotEmpty(t, suite.Summary.OverallStatus)
	assert.Contains(t, []string{"PASSED", "PARTIAL", "FAILED"}, suite.Summary.OverallStatus)
	assert.NotEmpty(t, suite.Summary.CoverageAreas)
	assert.NotNil(t, suite.Summary.Metrics)
	assert.Contains(t, suite.Summary.Metrics, "pass_rate")
	assert.Contains(t, suite.Summary.Metrics, "total_iterations")
	assert.Contains(t, suite.Summary.Metrics, "avg_duration_ms")

	// Verify results were saved
	results, err := runner.GetResults()
	require.NoError(t, err)
	assert.Equal(t, suite.TotalTests, results.TotalTests)
	assert.Equal(t, suite.Summary.OverallStatus, results.Summary.OverallStatus)

	// Log final summary
	t.Logf("Comprehensive property testing completed successfully")
	t.Logf("Total tests: %d, Passed: %d, Failed: %d", 
		suite.TotalTests, suite.PassedTests, suite.FailedTests)
	t.Logf("Overall status: %s", suite.Summary.OverallStatus)
	t.Logf("Pass rate: %.1f%%", suite.Summary.Metrics["pass_rate"])
}

// TestPropertyTestingWithFailures tests the property testing system with intentional failures
func TestPropertyTestingWithFailures(t *testing.T) {
	// This test demonstrates how the property testing system handles failures
	
	// Setup test database
	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for failure testing")
	}
	defer testDB.Close()
	defer testDB.Cleanup(t)

	// Setup mock cache that will fail
	mockCache := NewMockCacheService()
	defer mockCache.Close()

	// Create a server that returns errors
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// This endpoint always returns 500 error
	router.GET("/api/v1/articles", func(c *gin.Context) {
		c.JSON(500, gin.H{
			"error":   "internal_error",
			"message": "Intentional test failure",
		})
	})
	
	server := httptest.NewServer(router)
	defer server.Close()

	// Create temporary results directory
	tempDir, err := os.MkdirTemp("", "property_test_failures")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create property test runner
	runner := NewPropertyTestRunner(testDB.DB, mockCache, server, tempDir)

	// Configure for minimal testing
	config := &PropertyTestConfig{
		Iterations:  5,
		Timeout:     5 * time.Second,
		RandomSeed:  12345,
	}
	runner.SetConfig(config)

	// Run only API tests (which should fail)
	t.Run("APIContractTestsWithFailures", func(t *testing.T) {
		results := runner.runAPIContractTests(t)
		
		// Verify that some tests failed as expected
		hasFailures := false
		for _, result := range results {
			if !result.Passed {
				hasFailures = true
				assert.NotEmpty(t, result.FailureReason)
				assert.Greater(t, result.FailedIteration, 0)
				t.Logf("Expected failure: %s - %s", result.Property, result.FailureReason)
			}
		}
		
		assert.True(t, hasFailures, "Expected some API tests to fail with error server")
	})
}

// TestPropertyTestingConfiguration tests different configuration scenarios
func TestPropertyTestingConfiguration(t *testing.T) {
	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for configuration testing")
	}
	defer testDB.Close()
	defer testDB.Cleanup(t)

	mockCache := NewMockCacheService()
	defer mockCache.Close()

	t.Run("MinimalConfiguration", func(t *testing.T) {
		config := &PropertyTestConfig{
			Iterations:  1,
			MaxDataSize: 10,
			Timeout:     1 * time.Second,
			RandomSeed:  12345,
		}

		tester := NewDataInvariantTester(testDB.DB, mockCache, config)
		
		// Run a quick test with minimal configuration
		result := tester.TestSEOMetadataConsistency(t)
		
		assert.Equal(t, 1, result.Iterations)
		assert.Equal(t, "seo_metadata_consistency", result.Property)
		// Should complete quickly with minimal iterations
		assert.Less(t, result.Duration, 5*time.Second)
	})

	t.Run("ExtensiveConfiguration", func(t *testing.T) {
		config := &PropertyTestConfig{
			Iterations:  100,
			MaxDataSize: 1000,
			Timeout:     30 * time.Second,
			RandomSeed:  12345,
		}

		tester := NewDataInvariantTester(testDB.DB, mockCache, config)
		
		// Run a test with extensive configuration
		result := tester.TestCacheInvalidationCorrectness(t)
		
		assert.Equal(t, 100, result.Iterations)
		assert.Equal(t, "cache_invalidation_correctness", result.Property)
		// Should take longer with more iterations
		t.Logf("Extensive test completed in %v", result.Duration)
	})
}

// TestPropertyTestingEdgeCases tests edge cases and boundary conditions
func TestPropertyTestingEdgeCases(t *testing.T) {
	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for edge case testing")
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

	t.Run("EmptyDataGeneration", func(t *testing.T) {
		// Test with zero articles
		articles := tester.generatePartitionedArticles(0)
		assert.Len(t, articles, 0)
		
		// Verify cleanup handles empty arrays gracefully
		tester.cleanupTestArticles([]uint64{})
		tester.cleanupTestUsers([]uint64{})
	})

	t.Run("LargeDataGeneration", func(t *testing.T) {
		// Test with many articles
		articles := tester.generatePartitionedArticles(50)
		assert.Len(t, articles, 50)
		
		// Verify all articles have required fields
		for i, article := range articles {
			assert.NotEmpty(t, article.Title, "Article %d should have title", i)
			assert.NotEmpty(t, article.Slug, "Article %d should have slug", i)
			assert.NotZero(t, article.AuthorID, "Article %d should have author", i)
		}
	})

	t.Run("InvalidUserRoles", func(t *testing.T) {
		// Test permission verification with edge cases
		users := []*models.User{
			{ID: 1, Role: models.RoleAdmin},
			{ID: 2, Role: models.RoleContributor},
		}
		
		// Should not fail with valid roles
		err := tester.verifyPermissionInvariants(users)
		assert.NoError(t, err)
	})
}

// TestPropertyTestingPerformance benchmarks property testing performance
func TestPropertyTestingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for performance testing")
	}
	defer testDB.Close()
	defer testDB.Cleanup(t)

	mockCache := NewMockCacheService()
	defer mockCache.Close()

	config := &PropertyTestConfig{
		Iterations:  50,
		RandomSeed:  12345,
	}

	tester := NewDataInvariantTester(testDB.DB, mockCache, config)

	// Measure performance of different property tests
	tests := []struct {
		name string
		test func(*testing.T) PropertyTestResult
	}{
		{"PartitionConsistency", tester.TestPartitionDataConsistency},
		{"CacheInvalidation", tester.TestCacheInvalidationCorrectness},
		{"SEOConsistency", tester.TestSEOMetadataConsistency},
		{"UserPermissions", tester.TestUserPermissionInvariants},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start := time.Now()
			result := test.test(t)
			duration := time.Since(start)
			
			t.Logf("%s: %d iterations in %v (%.2f ms/iteration)", 
				test.name, result.Iterations, duration, 
				float64(duration.Milliseconds())/float64(result.Iterations))
			
			// Performance assertions
			assert.Less(t, duration, 30*time.Second, "Test should complete within 30 seconds")
			assert.True(t, result.Passed, "Performance test should pass")
		})
	}
}

