package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// APIContractTester provides property-based testing for API contracts
type APIContractTester struct {
	*PropertyTester
	server *httptest.Server
	client *http.Client
}

// NewAPIContractTester creates a new API contract tester
func NewAPIContractTester(server *httptest.Server, config *PropertyTestConfig) *APIContractTester {
	return &APIContractTester{
		PropertyTester: &PropertyTester{
			config:    config,
			generator: NewTestDataGenerator(),
		},
		server: server,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// APIResponse represents a generic API response for testing
type APIResponse struct {
	StatusCode int                    `json:"status_code"`
	Headers    map[string][]string    `json:"headers"`
	Body       map[string]interface{} `json:"body"`
	RawBody    []byte                 `json:"raw_body"`
}

// APITestCase represents a test case for API contract testing
type APITestCase struct {
	Method   string                 `json:"method"`
	Path     string                 `json:"path"`
	Headers  map[string]string      `json:"headers"`
	Body     interface{}            `json:"body"`
	Expected APIResponseExpectation `json:"expected"`
}

// APIResponseExpectation defines what to expect from an API response
type APIResponseExpectation struct {
	StatusCode    int                    `json:"status_code"`
	ContentType   string                 `json:"content_type"`
	Schema        map[string]interface{} `json:"schema"`
	RequiredFields []string              `json:"required_fields"`
	MaxResponseTime time.Duration        `json:"max_response_time"`
}

// TestAPIResponseSchema tests that API responses always conform to expected schemas
func (t *APIContractTester) TestAPIResponseSchema(test *testing.T) PropertyTestResult {
	start := time.Now()
	result := PropertyTestResult{
		Property:   "api_response_schema",
		Iterations: t.config.Iterations,
	}

	// Define API endpoints to test
	endpoints := []APITestCase{
		{
			Method: "GET",
			Path:   "/api/v1/articles",
			Expected: APIResponseExpectation{
				StatusCode:  200,
				ContentType: "application/json",
				RequiredFields: []string{"data", "pagination"},
				MaxResponseTime: 2 * time.Second,
			},
		},
		{
			Method: "GET",
			Path:   "/api/v1/articles/1",
			Expected: APIResponseExpectation{
				StatusCode:  200,
				ContentType: "application/json",
				RequiredFields: []string{"id", "title", "content", "author_id"},
				MaxResponseTime: 1 * time.Second,
			},
		},
		{
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
			Expected: APIResponseExpectation{
				StatusCode:  201,
				ContentType: "application/json",
				RequiredFields: []string{"id", "title", "content"},
				MaxResponseTime: 3 * time.Second,
			},
		},
	}

	for i := 0; i < t.config.Iterations; i++ {
		for _, endpoint := range endpoints {
			// Make API request
			response, duration, err := t.makeAPIRequest(endpoint)
			if err != nil {
				result.Passed = false
				result.FailedIteration = i + 1
				result.FailureReason = fmt.Sprintf("API request failed: %v", err)
				result.CounterExample = endpoint
				result.Duration = time.Since(start)
				return result
			}

			// Validate response schema
			if err := t.validateResponseSchema(response, endpoint.Expected); err != nil {
				result.Passed = false
				result.FailedIteration = i + 1
				result.FailureReason = fmt.Sprintf("Schema validation failed: %v", err)
				result.CounterExample = map[string]interface{}{
					"endpoint": endpoint,
					"response": response,
					"duration": duration,
				}
				result.Duration = time.Since(start)
				return result
			}

			// Validate response time
			if duration > endpoint.Expected.MaxResponseTime {
				result.Passed = false
				result.FailedIteration = i + 1
				result.FailureReason = fmt.Sprintf("Response time %v exceeds maximum %v", duration, endpoint.Expected.MaxResponseTime)
				result.CounterExample = map[string]interface{}{
					"endpoint": endpoint,
					"duration": duration,
				}
				result.Duration = time.Since(start)
				return result
			}
		}
	}

	result.Passed = true
	result.Duration = time.Since(start)
	return result
}

// TestAPIBehaviorConsistency tests that API behavior is consistent across similar requests
func (t *APIContractTester) TestAPIBehaviorConsistency(test *testing.T) PropertyTestResult {
	start := time.Now()
	result := PropertyTestResult{
		Property:   "api_behavior_consistency",
		Iterations: t.config.Iterations,
	}

	for i := 0; i < t.config.Iterations; i++ {
		// Test idempotency for GET requests
		getEndpoint := APITestCase{
			Method: "GET",
			Path:   "/api/v1/articles/1",
		}

		// Make the same request multiple times
		responses := make([]APIResponse, 3)
		for j := 0; j < 3; j++ {
			response, _, err := t.makeAPIRequest(getEndpoint)
			if err != nil {
				result.Passed = false
				result.FailedIteration = i + 1
				result.FailureReason = fmt.Sprintf("GET request failed: %v", err)
				result.CounterExample = getEndpoint
				result.Duration = time.Since(start)
				return result
			}
			responses[j] = *response
		}

		// Verify responses are identical (idempotency)
		if !t.areResponsesIdentical(responses) {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = "GET requests are not idempotent - responses differ"
			result.CounterExample = responses
			result.Duration = time.Since(start)
			return result
		}

		// Test POST behavior consistency
		article := t.generator.GenerateTestArticle()
		postEndpoint := APITestCase{
			Method: "POST",
			Path:   "/api/v1/articles",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]interface{}{
				"title":       article.Title,
				"content":     article.Content,
				"author_id":   article.AuthorID,
				"category_id": article.CategoryID,
				"status":      article.Status,
			},
		}

		// Create article
		createResponse, _, err := t.makeAPIRequest(postEndpoint)
		if err != nil {
			result.Passed = false
			result.FailedIteration = i + 1
			result.FailureReason = fmt.Sprintf("POST request failed: %v", err)
			result.CounterExample = postEndpoint
			result.Duration = time.Since(start)
			return result
		}

		// Verify created article can be retrieved
		if createResponse.StatusCode == 201 {
			if id, ok := createResponse.Body["id"].(float64); ok {
				getCreatedEndpoint := APITestCase{
					Method: "GET",
					Path:   fmt.Sprintf("/api/v1/articles/%.0f", id),
				}

				getResponse, _, err := t.makeAPIRequest(getCreatedEndpoint)
				if err != nil || getResponse.StatusCode != 200 {
					result.Passed = false
					result.FailedIteration = i + 1
					result.FailureReason = "Created article cannot be retrieved"
					result.CounterExample = map[string]interface{}{
						"created_id": id,
						"get_error":  err,
						"get_status": getResponse.StatusCode,
					}
					result.Duration = time.Since(start)
					return result
				}

				// Verify data consistency between create and get responses
				if err := t.verifyDataConsistency(createResponse.Body, getResponse.Body); err != nil {
					result.Passed = false
					result.FailedIteration = i + 1
					result.FailureReason = fmt.Sprintf("Data inconsistency: %v", err)
					result.CounterExample = map[string]interface{}{
						"create_response": createResponse.Body,
						"get_response":    getResponse.Body,
					}
					result.Duration = time.Since(start)
					return result
				}
			}
		}
	}

	result.Passed = true
	result.Duration = time.Since(start)
	return result
}

// TestAPIErrorHandling tests that API error handling is consistent and informative
func (t *APIContractTester) TestAPIErrorHandling(test *testing.T) PropertyTestResult {
	start := time.Now()
	result := PropertyTestResult{
		Property:   "api_error_handling",
		Iterations: t.config.Iterations,
	}

	// Define error test cases
	errorCases := []APITestCase{
		{
			Method: "GET",
			Path:   "/api/v1/articles/999999", // Non-existent article
			Expected: APIResponseExpectation{
				StatusCode: 404,
				ContentType: "application/json",
				RequiredFields: []string{"error", "message"},
			},
		},
		{
			Method: "POST",
			Path:   "/api/v1/articles",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]interface{}{
				"title": "", // Invalid empty title
			},
			Expected: APIResponseExpectation{
				StatusCode: 400,
				ContentType: "application/json",
				RequiredFields: []string{"error", "message"},
			},
		},
		{
			Method: "POST",
			Path:   "/api/v1/articles",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: "invalid json", // Invalid JSON
			Expected: APIResponseExpectation{
				StatusCode: 400,
				ContentType: "application/json",
				RequiredFields: []string{"error", "message"},
			},
		},
	}

	for i := 0; i < t.config.Iterations; i++ {
		for _, errorCase := range errorCases {
			response, _, err := t.makeAPIRequest(errorCase)
			if err != nil {
				result.Passed = false
				result.FailedIteration = i + 1
				result.FailureReason = fmt.Sprintf("Error case request failed: %v", err)
				result.CounterExample = errorCase
				result.Duration = time.Since(start)
				return result
			}

			// Verify error response format
			if err := t.validateErrorResponse(response, errorCase.Expected); err != nil {
				result.Passed = false
				result.FailedIteration = i + 1
				result.FailureReason = fmt.Sprintf("Error response validation failed: %v", err)
				result.CounterExample = map[string]interface{}{
					"error_case": errorCase,
					"response":   response,
				}
				result.Duration = time.Since(start)
				return result
			}
		}
	}

	result.Passed = true
	result.Duration = time.Since(start)
	return result
}

// TestAPIPerformance tests that API performance meets specified thresholds
func (t *APIContractTester) TestAPIPerformance(test *testing.T) PropertyTestResult {
	start := time.Now()
	result := PropertyTestResult{
		Property:   "api_performance",
		Iterations: t.config.Iterations,
	}

	// Performance test cases
	performanceCases := []APITestCase{
		{
			Method: "GET",
			Path:   "/api/v1/articles",
			Expected: APIResponseExpectation{
				MaxResponseTime: 2 * time.Second,
			},
		},
		{
			Method: "GET",
			Path:   "/api/v1/articles/1",
			Expected: APIResponseExpectation{
				MaxResponseTime: 1 * time.Second,
			},
		},
	}

	for i := 0; i < t.config.Iterations; i++ {
		for _, perfCase := range performanceCases {
			_, duration, err := t.makeAPIRequest(perfCase)
			if err != nil {
				result.Passed = false
				result.FailedIteration = i + 1
				result.FailureReason = fmt.Sprintf("Performance test request failed: %v", err)
				result.CounterExample = perfCase
				result.Duration = time.Since(start)
				return result
			}

			if duration > perfCase.Expected.MaxResponseTime {
				result.Passed = false
				result.FailedIteration = i + 1
				result.FailureReason = fmt.Sprintf("Response time %v exceeds threshold %v", duration, perfCase.Expected.MaxResponseTime)
				result.CounterExample = map[string]interface{}{
					"endpoint": perfCase,
					"duration": duration,
				}
				result.Duration = time.Since(start)
				return result
			}
		}
	}

	result.Passed = true
	result.Duration = time.Since(start)
	return result
}

// Helper methods

func (t *APIContractTester) makeAPIRequest(testCase APITestCase) (*APIResponse, time.Duration, error) {
	var body io.Reader
	if testCase.Body != nil {
		if bodyStr, ok := testCase.Body.(string); ok {
			body = strings.NewReader(bodyStr)
		} else {
			jsonBody, err := json.Marshal(testCase.Body)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
			}
			body = bytes.NewReader(jsonBody)
		}
	}

	url := t.server.URL + testCase.Path
	req, err := http.NewRequest(testCase.Method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range testCase.Headers {
		req.Header.Set(key, value)
	}

	// Make request and measure duration
	start := time.Now()
	resp, err := t.client.Do(req)
	duration := time.Since(start)
	
	if err != nil {
		return nil, duration, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, duration, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON body if content type is JSON
	var parsedBody map[string]interface{}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") && len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &parsedBody); err != nil {
			// If JSON parsing fails, create error body
			parsedBody = map[string]interface{}{
				"parse_error": err.Error(),
				"raw_content": string(rawBody),
			}
		}
	}

	response := &APIResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       parsedBody,
		RawBody:    rawBody,
	}

	return response, duration, nil
}

func (t *APIContractTester) validateResponseSchema(response *APIResponse, expected APIResponseExpectation) error {
	// Check status code
	if response.StatusCode != expected.StatusCode {
		return fmt.Errorf("expected status %d, got %d", expected.StatusCode, response.StatusCode)
	}

	// Check content type
	if expected.ContentType != "" {
		contentType := ""
		if ct, ok := response.Headers["Content-Type"]; ok && len(ct) > 0 {
			contentType = ct[0]
		}
		if !strings.Contains(contentType, expected.ContentType) {
			return fmt.Errorf("expected content type %s, got %s", expected.ContentType, contentType)
		}
	}

	// Check required fields
	for _, field := range expected.RequiredFields {
		if _, exists := response.Body[field]; !exists {
			return fmt.Errorf("required field '%s' missing from response", field)
		}
	}

	return nil
}

func (t *APIContractTester) validateErrorResponse(response *APIResponse, expected APIResponseExpectation) error {
	// Check status code
	if response.StatusCode != expected.StatusCode {
		return fmt.Errorf("expected error status %d, got %d", expected.StatusCode, response.StatusCode)
	}

	// Check that error response has proper structure
	if response.Body == nil {
		return fmt.Errorf("error response should have JSON body")
	}

	// Check required error fields
	for _, field := range expected.RequiredFields {
		if _, exists := response.Body[field]; !exists {
			return fmt.Errorf("required error field '%s' missing from response", field)
		}
	}

	// Verify error message is not empty
	if message, ok := response.Body["message"].(string); ok {
		if strings.TrimSpace(message) == "" {
			return fmt.Errorf("error message should not be empty")
		}
	}

	return nil
}

func (t *APIContractTester) areResponsesIdentical(responses []APIResponse) bool {
	if len(responses) < 2 {
		return true
	}

	first := responses[0]
	for i := 1; i < len(responses); i++ {
		if responses[i].StatusCode != first.StatusCode {
			return false
		}
		
		// Compare response bodies (excluding timestamps and other dynamic fields)
		if !t.areResponseBodiesEquivalent(first.Body, responses[i].Body) {
			return false
		}
	}

	return true
}

func (t *APIContractTester) areResponseBodiesEquivalent(body1, body2 map[string]interface{}) bool {
	// Skip dynamic fields that are expected to change
	dynamicFields := map[string]bool{
		"updated_at": true,
		"timestamp":  true,
		"request_id": true,
	}

	// Create copies without dynamic fields
	filtered1 := make(map[string]interface{})
	filtered2 := make(map[string]interface{})

	for k, v := range body1 {
		if !dynamicFields[k] {
			filtered1[k] = v
		}
	}

	for k, v := range body2 {
		if !dynamicFields[k] {
			filtered2[k] = v
		}
	}

	return reflect.DeepEqual(filtered1, filtered2)
}

func (t *APIContractTester) verifyDataConsistency(createData, getData map[string]interface{}) error {
	// Check that key fields match between create and get responses
	keyFields := []string{"title", "content", "author_id", "category_id", "status"}

	for _, field := range keyFields {
		createValue, createExists := createData[field]
		getValue, getExists := getData[field]

		if createExists && getExists {
			if !reflect.DeepEqual(createValue, getValue) {
				return fmt.Errorf("field '%s' inconsistent: create=%v, get=%v", field, createValue, getValue)
			}
		} else if createExists != getExists {
			return fmt.Errorf("field '%s' existence mismatch: create=%v, get=%v", field, createExists, getExists)
		}
	}

	return nil
}