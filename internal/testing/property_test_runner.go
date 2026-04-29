package testing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"high-performance-news-website/pkg/cache"
)

// PropertyTestRunner orchestrates all property-based tests
type PropertyTestRunner struct {
	config         *PropertyTestConfig
	db             *sql.DB
	cache          cache.CacheService
	server         *httptest.Server
	resultsDir     string
	dataInvariant  *DataInvariantTester
	apiContract    *APIContractTester
}

// NewPropertyTestRunner creates a new property test runner
func NewPropertyTestRunner(db *sql.DB, cache cache.CacheService, server *httptest.Server, resultsDir string) *PropertyTestRunner {
	config := DefaultPropertyTestConfig()
	
	return &PropertyTestRunner{
		config:        config,
		db:            db,
		cache:         cache,
		server:        server,
		resultsDir:    resultsDir,
		dataInvariant: NewDataInvariantTester(db, cache, config),
		apiContract:   NewAPIContractTester(server, config),
	}
}

// PropertyTestSuite represents a complete suite of property tests
type PropertyTestSuite struct {
	Name        string                `json:"name"`
	StartTime   time.Time            `json:"start_time"`
	EndTime     time.Time            `json:"end_time"`
	Duration    time.Duration        `json:"duration"`
	TotalTests  int                  `json:"total_tests"`
	PassedTests int                  `json:"passed_tests"`
	FailedTests int                  `json:"failed_tests"`
	Results     []PropertyTestResult `json:"results"`
	Summary     TestSummary          `json:"summary"`
}

// TestSummary provides a summary of test results
type TestSummary struct {
	OverallStatus    string            `json:"overall_status"`
	CoverageAreas    []string          `json:"coverage_areas"`
	CriticalFailures []string          `json:"critical_failures"`
	Recommendations  []string          `json:"recommendations"`
	Metrics          map[string]float64 `json:"metrics"`
}

// RunAllPropertyTests executes all property-based tests and returns comprehensive results
func (r *PropertyTestRunner) RunAllPropertyTests(t *testing.T) *PropertyTestSuite {
	suite := &PropertyTestSuite{
		Name:      "Comprehensive Property Test Suite",
		StartTime: time.Now(),
		Results:   make([]PropertyTestResult, 0),
	}

	t.Log("Starting comprehensive property-based testing suite...")

	// Run data invariant tests
	t.Run("DataInvariantTests", func(t *testing.T) {
		suite.Results = append(suite.Results, r.runDataInvariantTests(t)...)
	})

	// Run API contract tests
	t.Run("APIContractTests", func(t *testing.T) {
		suite.Results = append(suite.Results, r.runAPIContractTests(t)...)
	})

	// Calculate final metrics
	suite.EndTime = time.Now()
	suite.Duration = suite.EndTime.Sub(suite.StartTime)
	suite.TotalTests = len(suite.Results)
	
	for _, result := range suite.Results {
		if result.Passed {
			suite.PassedTests++
		} else {
			suite.FailedTests++
		}
	}

	// Generate summary
	suite.Summary = r.generateTestSummary(suite)

	// Save results
	if err := r.saveTestResults(suite); err != nil {
		t.Logf("Warning: Failed to save test results: %v", err)
	}

	// Log summary
	r.logTestSummary(t, suite)

	return suite
}

// runDataInvariantTests executes all data invariant property tests
func (r *PropertyTestRunner) runDataInvariantTests(t *testing.T) []PropertyTestResult {
	var results []PropertyTestResult

	t.Log("Running data invariant property tests...")

	// Test partition data consistency
	t.Run("PartitionDataConsistency", func(t *testing.T) {
		result := r.dataInvariant.TestPartitionDataConsistency(t)
		results = append(results, result)
		r.logTestResult(t, result)
	})

	// Test cache invalidation correctness
	t.Run("CacheInvalidationCorrectness", func(t *testing.T) {
		result := r.dataInvariant.TestCacheInvalidationCorrectness(t)
		results = append(results, result)
		r.logTestResult(t, result)
	})

	// Test SEO metadata consistency
	t.Run("SEOMetadataConsistency", func(t *testing.T) {
		result := r.dataInvariant.TestSEOMetadataConsistency(t)
		results = append(results, result)
		r.logTestResult(t, result)
	})

	// Test user permission invariants
	t.Run("UserPermissionInvariants", func(t *testing.T) {
		result := r.dataInvariant.TestUserPermissionInvariants(t)
		results = append(results, result)
		r.logTestResult(t, result)
	})

	return results
}

// runAPIContractTests executes all API contract property tests
func (r *PropertyTestRunner) runAPIContractTests(t *testing.T) []PropertyTestResult {
	var results []PropertyTestResult

	if r.server == nil {
		t.Log("Skipping API contract tests - no server provided")
		return results
	}

	t.Log("Running API contract property tests...")

	// Test API response schema
	t.Run("APIResponseSchema", func(t *testing.T) {
		result := r.apiContract.TestAPIResponseSchema(t)
		results = append(results, result)
		r.logTestResult(t, result)
	})

	// Test API behavior consistency
	t.Run("APIBehaviorConsistency", func(t *testing.T) {
		result := r.apiContract.TestAPIBehaviorConsistency(t)
		results = append(results, result)
		r.logTestResult(t, result)
	})

	// Test API error handling
	t.Run("APIErrorHandling", func(t *testing.T) {
		result := r.apiContract.TestAPIErrorHandling(t)
		results = append(results, result)
		r.logTestResult(t, result)
	})

	// Test API performance
	t.Run("APIPerformance", func(t *testing.T) {
		result := r.apiContract.TestAPIPerformance(t)
		results = append(results, result)
		r.logTestResult(t, result)
	})

	return results
}

// generateTestSummary creates a comprehensive summary of test results
func (r *PropertyTestRunner) generateTestSummary(suite *PropertyTestSuite) TestSummary {
	summary := TestSummary{
		CoverageAreas:    make([]string, 0),
		CriticalFailures: make([]string, 0),
		Recommendations:  make([]string, 0),
		Metrics:          make(map[string]float64),
	}

	// Determine overall status
	if suite.FailedTests == 0 {
		summary.OverallStatus = "PASSED"
	} else if suite.FailedTests < suite.TotalTests/2 {
		summary.OverallStatus = "PARTIAL"
	} else {
		summary.OverallStatus = "FAILED"
	}

	// Calculate metrics
	if suite.TotalTests > 0 {
		summary.Metrics["pass_rate"] = float64(suite.PassedTests) / float64(suite.TotalTests) * 100
		summary.Metrics["total_iterations"] = 0
		summary.Metrics["avg_duration_ms"] = 0

		totalIterations := 0
		totalDuration := time.Duration(0)

		for _, result := range suite.Results {
			totalIterations += result.Iterations
			totalDuration += result.Duration
		}

		summary.Metrics["total_iterations"] = float64(totalIterations)
		if suite.TotalTests > 0 {
			summary.Metrics["avg_duration_ms"] = float64(totalDuration.Milliseconds()) / float64(suite.TotalTests)
		}
	}

	// Identify coverage areas
	coverageMap := make(map[string]bool)
	for _, result := range suite.Results {
		switch result.Property {
		case "partition_data_consistency":
			coverageMap["Database Partitioning"] = true
		case "cache_invalidation_correctness":
			coverageMap["Cache Management"] = true
		case "seo_metadata_consistency":
			coverageMap["SEO Compliance"] = true
		case "user_permission_invariants":
			coverageMap["Security & Permissions"] = true
		case "api_response_schema":
			coverageMap["API Schema Validation"] = true
		case "api_behavior_consistency":
			coverageMap["API Behavior"] = true
		case "api_error_handling":
			coverageMap["Error Handling"] = true
		case "api_performance":
			coverageMap["Performance"] = true
		}
	}

	for area := range coverageMap {
		summary.CoverageAreas = append(summary.CoverageAreas, area)
	}

	// Identify critical failures and recommendations
	for _, result := range suite.Results {
		if !result.Passed {
			summary.CriticalFailures = append(summary.CriticalFailures, 
				fmt.Sprintf("%s: %s", result.Property, result.FailureReason))

			// Generate recommendations based on failure type
			switch result.Property {
			case "partition_data_consistency":
				summary.Recommendations = append(summary.Recommendations,
					"Review database partition management and referential integrity constraints")
			case "cache_invalidation_correctness":
				summary.Recommendations = append(summary.Recommendations,
					"Implement more robust cache invalidation strategies and monitoring")
			case "seo_metadata_consistency":
				summary.Recommendations = append(summary.Recommendations,
					"Strengthen SEO metadata validation and consistency checks")
			case "user_permission_invariants":
				summary.Recommendations = append(summary.Recommendations,
					"Review role-based access control implementation and permission hierarchies")
			case "api_response_schema", "api_behavior_consistency":
				summary.Recommendations = append(summary.Recommendations,
					"Implement stricter API contract validation and schema enforcement")
			case "api_error_handling":
				summary.Recommendations = append(summary.Recommendations,
					"Improve API error handling consistency and error message quality")
			case "api_performance":
				summary.Recommendations = append(summary.Recommendations,
					"Optimize API performance and implement response time monitoring")
			}
		}
	}

	// Add general recommendations based on metrics
	if summary.Metrics["pass_rate"] < 100 {
		summary.Recommendations = append(summary.Recommendations,
			"Consider increasing property test iterations for better coverage")
	}

	if summary.Metrics["avg_duration_ms"] > 5000 {
		summary.Recommendations = append(summary.Recommendations,
			"Optimize property test performance to reduce execution time")
	}

	return summary
}

// saveTestResults saves test results to JSON file
func (r *PropertyTestRunner) saveTestResults(suite *PropertyTestSuite) error {
	if r.resultsDir == "" {
		return nil // No results directory specified
	}

	// Ensure results directory exists
	if err := os.MkdirAll(r.resultsDir, 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("property_test_results_%s.json", 
		suite.StartTime.Format("20060102_150405"))
	filepath := filepath.Join(r.resultsDir, filename)

	// Marshal results to JSON
	data, err := json.MarshalIndent(suite, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}

	return nil
}

// logTestResult logs individual test result
func (r *PropertyTestRunner) logTestResult(t *testing.T, result PropertyTestResult) {
	status := "PASS"
	if !result.Passed {
		status = "FAIL"
	}

	t.Logf("[%s] %s - %d iterations in %v", 
		status, result.Property, result.Iterations, result.Duration)

	if !result.Passed {
		t.Logf("  Failed at iteration %d: %s", result.FailedIteration, result.FailureReason)
		if result.CounterExample != nil {
			t.Logf("  Counter example: %+v", result.CounterExample)
		}
	}
}

// logTestSummary logs comprehensive test summary
func (r *PropertyTestRunner) logTestSummary(t *testing.T, suite *PropertyTestSuite) {
	t.Logf("\n" + strings.Repeat("=", 80))
	t.Logf("PROPERTY TEST SUITE SUMMARY")
	t.Logf(strings.Repeat("=", 80))
	t.Logf("Overall Status: %s", suite.Summary.OverallStatus)
	t.Logf("Total Tests: %d", suite.TotalTests)
	t.Logf("Passed: %d", suite.PassedTests)
	t.Logf("Failed: %d", suite.FailedTests)
	t.Logf("Pass Rate: %.1f%%", suite.Summary.Metrics["pass_rate"])
	t.Logf("Total Duration: %v", suite.Duration)
	t.Logf("Average Test Duration: %.1fms", suite.Summary.Metrics["avg_duration_ms"])
	t.Logf("Total Iterations: %.0f", suite.Summary.Metrics["total_iterations"])

	if len(suite.Summary.CoverageAreas) > 0 {
		t.Logf("\nCoverage Areas:")
		for _, area := range suite.Summary.CoverageAreas {
			t.Logf("  - %s", area)
		}
	}

	if len(suite.Summary.CriticalFailures) > 0 {
		t.Logf("\nCritical Failures:")
		for _, failure := range suite.Summary.CriticalFailures {
			t.Logf("  - %s", failure)
		}
	}

	if len(suite.Summary.Recommendations) > 0 {
		t.Logf("\nRecommendations:")
		for _, rec := range suite.Summary.Recommendations {
			t.Logf("  - %s", rec)
		}
	}

	t.Logf(strings.Repeat("=", 80))
}

// SetConfig updates the property test configuration
func (r *PropertyTestRunner) SetConfig(config *PropertyTestConfig) {
	r.config = config
	r.dataInvariant.config = config
	r.apiContract.config = config
}

// GetResults returns the latest test results if available
func (r *PropertyTestRunner) GetResults() (*PropertyTestSuite, error) {
	if r.resultsDir == "" {
		return nil, fmt.Errorf("no results directory configured")
	}

	// Find the most recent results file
	files, err := filepath.Glob(filepath.Join(r.resultsDir, "property_test_results_*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to find results files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no results files found")
	}

	// Get the most recent file (files are sorted by name which includes timestamp)
	latestFile := files[len(files)-1]

	// Read and parse the results
	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read results file: %w", err)
	}

	var suite PropertyTestSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}

	return &suite, nil
}