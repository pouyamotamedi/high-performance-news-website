package testing

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAPIServer creates a mock API server for testing
func createMockAPIServer() *httptest.Server {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock articles endpoints
	router.GET("/api/v1/articles", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"data": []gin.H{
				{
					"id":          1,
					"title":       "Test Article 1",
					"content":     "Test content 1",
					"author_id":   1,
					"category_id": 1,
					"status":      "published",
				},
				{
					"id":          2,
					"title":       "Test Article 2",
					"content":     "Test content 2",
					"author_id":   2,
					"category_id": 1,
					"status":      "published",
				},
			},
			"pagination": gin.H{
				"page":       1,
				"per_page":   10,
				"total":      2,
				"total_pages": 1,
			},
		})
	})

	router.GET("/api/v1/articles/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "999999" {
			c.JSON(404, gin.H{
				"error":   "not_found",
				"message": "Article not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"id":          1,
			"title":       "Test Article",
			"content":     "Test content",
			"author_id":   1,
			"category_id": 1,
			"status":      "published",
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
		})
	})

	router.POST("/api/v1/articles", func(c *gin.Context) {
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{
				"error":   "invalid_json",
				"message": "Invalid JSON format",
			})
			return
		}

		title, titleExists := body["title"].(string)
		if !titleExists || title == "" {
			c.JSON(400, gin.H{
				"error":   "validation_error",
				"message": "Title is required",
			})
			return
		}

		c.JSON(201, gin.H{
			"id":          123,
			"title":       title,
			"content":     body["content"],
			"author_id":   body["author_id"],
			"category_id": body["category_id"],
			"status":      body["status"],
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
		})
	})

	return httptest.NewServer(router)
}

// TestAPIContractProperties runs all API contract property tests
func TestAPIContractProperties(t *testing.T) {
	// Create mock server
	server := createMockAPIServer()
	defer server.Close()

	// Create property test config
	config := &PropertyTestConfig{
		Iterations:  20, // Reduced for faster testing
		Timeout:     5 * time.Second,
		RandomSeed:  12345,
	}

	// Create API contract tester
	tester := NewAPIContractTester(server, config)

	t.Run("APIResponseSchema", func(t *testing.T) {
		result := tester.TestAPIResponseSchema(t)
		
		assert.True(t, result.Passed, "API response schema property should pass")
		assert.Equal(t, "api_response_schema", result.Property)
		assert.Equal(t, config.Iterations, result.Iterations)
		
		if !result.Passed {
			t.Logf("Failed at iteration %d: %s", result.FailedIteration, result.FailureReason)
			t.Logf("Counter example: %+v", result.CounterExample)
		}
		
		t.Logf("Property test completed in %v", result.Duration)
	})

	t.Run("APIBehaviorConsistency", func(t *testing.T) {
		result := tester.TestAPIBehaviorConsistency(t)
		
		assert.True(t, result.Passed, "API behavior consistency property should pass")
		assert.Equal(t, "api_behavior_consistency", result.Property)
		
		if !result.Passed {
			t.Logf("Failed at iteration %d: %s", result.FailedIteration, result.FailureReason)
			t.Logf("Counter example: %+v", result.CounterExample)
		}
		
		t.Logf("Property test completed in %v", result.Duration)
	})

	t.Run("APIErrorHandling", func(t *testing.T) {
		result := tester.TestAPIErrorHandling(t)
		
		assert.True(t, result.Passed, "API error handling property should pass")
		assert.Equal(t, "api_error_handling", result.Property)
		
		if !result.Passed {
			t.Logf("Failed at iteration %d: %s", result.FailedIteration, result.FailureReason)
			t.Logf("Counter example: %+v", result.CounterExample)
		}
		
		t.Logf("Property test completed in %v", result.Duration)
	})

	t.Run("APIPerformance", func(t *testing.T) {
		result := tester.TestAPIPerformance(t)
		
		assert.True(t, result.Passed, "API performance property should pass")
		assert.Equal(t, "api_performance", result.Property)
		
		if !result.Passed {
			t.Logf("Failed at iteration %d: %s", result.FailedIteration, result.FailureReason)
			t.Logf("Counter example: %+v", result.CounterExample)
		}
		
		t.Logf("Property test completed in %v", result.Duration)
	})
}

// TestAPIContractHelpers tests the helper methods used in API contract testing
func TestAPIContractHelpers(t *testing.T) {
	server := createMockAPIServer()
	defer server.Close()

	config := DefaultPropertyTestConfig()
	tester := NewAPIContractTester(server, config)

	t.Run("MakeAPIRequest", func(t *testing.T) {
		testCase := APITestCase{
			Method: "GET",
			Path:   "/api/v1/articles",
		}

		response, duration, err := tester.makeAPIRequest(testCase)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 200, response.StatusCode)
		assert.Greater(t, duration, time.Duration(0))
		assert.NotNil(t, response.Body)
		
		// Verify response structure
		assert.Contains(t, response.Body, "data")
		assert.Contains(t, response.Body, "pagination")
	})

	t.Run("MakeAPIRequestWithBody", func(t *testing.T) {
		testCase := APITestCase{
			Method: "POST",
			Path:   "/api/v1/articles",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]interface{}{
				"title":       "Test Article",
				"content":     "Test content",
				"author_id":   1,
				"category_id": 1,
				"status":      "draft",
			},
		}

		response, duration, err := tester.makeAPIRequest(testCase)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 201, response.StatusCode)
		assert.Greater(t, duration, time.Duration(0))
		
		// Verify created article data
		assert.Equal(t, float64(123), response.Body["id"])
		assert.Equal(t, "Test Article", response.Body["title"])
	})

	t.Run("ValidateResponseSchema", func(t *testing.T) {
		response := &APIResponse{
			StatusCode: 200,
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: map[string]interface{}{
				"data":       []interface{}{},
				"pagination": map[string]interface{}{},
			},
		}

		expected := APIResponseExpectation{
			StatusCode:     200,
			ContentType:    "application/json",
			RequiredFields: []string{"data", "pagination"},
		}

		err := tester.validateResponseSchema(response, expected)
		assert.NoError(t, err)
	})

	t.Run("ValidateResponseSchemaFailure", func(t *testing.T) {
		response := &APIResponse{
			StatusCode: 200,
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: map[string]interface{}{
				"data": []interface{}{},
				// Missing pagination field
			},
		}

		expected := APIResponseExpectation{
			StatusCode:     200,
			ContentType:    "application/json",
			RequiredFields: []string{"data", "pagination"},
		}

		err := tester.validateResponseSchema(response, expected)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pagination")
	})

	t.Run("ValidateErrorResponse", func(t *testing.T) {
		response := &APIResponse{
			StatusCode: 404,
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: map[string]interface{}{
				"error":   "not_found",
				"message": "Article not found",
			},
		}

		expected := APIResponseExpectation{
			StatusCode:     404,
			ContentType:    "application/json",
			RequiredFields: []string{"error", "message"},
		}

		err := tester.validateErrorResponse(response, expected)
		assert.NoError(t, err)
	})

	t.Run("AreResponsesIdentical", func(t *testing.T) {
		response1 := APIResponse{
			StatusCode: 200,
			Body: map[string]interface{}{
				"id":    1,
				"title": "Test",
				"updated_at": "2023-01-01T00:00:00Z", // Dynamic field
			},
		}

		response2 := APIResponse{
			StatusCode: 200,
			Body: map[string]interface{}{
				"id":    1,
				"title": "Test",
				"updated_at": "2023-01-02T00:00:00Z", // Different dynamic field
			},
		}

		response3 := APIResponse{
			StatusCode: 200,
			Body: map[string]interface{}{
				"id":    2, // Different static field
				"title": "Test",
				"updated_at": "2023-01-01T00:00:00Z",
			},
		}

		responses := []APIResponse{response1, response2}
		assert.True(t, tester.areResponsesIdentical(responses), "Responses should be identical ignoring dynamic fields")

		responses = []APIResponse{response1, response3}
		assert.False(t, tester.areResponsesIdentical(responses), "Responses should not be identical with different static fields")
	})

	t.Run("VerifyDataConsistency", func(t *testing.T) {
		createData := map[string]interface{}{
			"id":          123,
			"title":       "Test Article",
			"content":     "Test content",
			"author_id":   1,
			"category_id": 1,
		}

		getData := map[string]interface{}{
			"id":          123,
			"title":       "Test Article",
			"content":     "Test content",
			"author_id":   1,
			"category_id": 1,
			"created_at":  "2023-01-01T00:00:00Z", // Additional field in get response
		}

		err := tester.verifyDataConsistency(createData, getData)
		assert.NoError(t, err)

		// Test with inconsistent data
		inconsistentGetData := map[string]interface{}{
			"id":          123,
			"title":       "Different Title", // Inconsistent
			"content":     "Test content",
			"author_id":   1,
			"category_id": 1,
		}

		err = tester.verifyDataConsistency(createData, inconsistentGetData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title")
	})
}

// TestAPIContractErrorCases tests error handling in API contract testing
func TestAPIContractErrorCases(t *testing.T) {
	server := createMockAPIServer()
	defer server.Close()

	config := &PropertyTestConfig{
		Iterations: 5,
		Timeout:    5 * time.Second,
	}

	tester := NewAPIContractTester(server, config)

	t.Run("NotFoundError", func(t *testing.T) {
		testCase := APITestCase{
			Method: "GET",
			Path:   "/api/v1/articles/999999",
		}

		response, _, err := tester.makeAPIRequest(testCase)
		require.NoError(t, err)
		assert.Equal(t, 404, response.StatusCode)
		assert.Equal(t, "not_found", response.Body["error"])
		assert.Equal(t, "Article not found", response.Body["message"])
	})

	t.Run("ValidationError", func(t *testing.T) {
		testCase := APITestCase{
			Method: "POST",
			Path:   "/api/v1/articles",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]interface{}{
				"title": "", // Empty title should cause validation error
			},
		}

		response, _, err := tester.makeAPIRequest(testCase)
		require.NoError(t, err)
		assert.Equal(t, 400, response.StatusCode)
		assert.Equal(t, "validation_error", response.Body["error"])
		assert.Contains(t, response.Body["message"], "required")
	})

	t.Run("InvalidJSONError", func(t *testing.T) {
		testCase := APITestCase{
			Method: "POST",
			Path:   "/api/v1/articles",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: "invalid json string",
		}

		response, _, err := tester.makeAPIRequest(testCase)
		require.NoError(t, err)
		assert.Equal(t, 400, response.StatusCode)
		assert.Equal(t, "invalid_json", response.Body["error"])
	})
}

// BenchmarkAPIContractTesting benchmarks the API contract testing performance
func BenchmarkAPIContractTesting(b *testing.B) {
	server := createMockAPIServer()
	defer server.Close()

	config := &PropertyTestConfig{
		Iterations: 5, // Reduced for benchmarking
	}

	tester := NewAPIContractTester(server, config)

	b.Run("ResponseSchema", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := tester.TestAPIResponseSchema(&testing.T{})
			if !result.Passed {
				b.Fatalf("Property test failed: %s", result.FailureReason)
			}
		}
	})

	b.Run("BehaviorConsistency", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := tester.TestAPIBehaviorConsistency(&testing.T{})
			if !result.Passed {
				b.Fatalf("Property test failed: %s", result.FailureReason)
			}
		}
	})
}