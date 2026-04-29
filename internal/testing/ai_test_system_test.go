package testing

import (
	"context"
	"strings"
	"testing"
	"time"
)

// MockLLMClient implements LLMClient for testing
type MockLLMClient struct {
	responses map[string]string
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: make(map[string]string),
	}
}

func (m *MockLLMClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	// Return predefined responses based on prompt content
	if contains(prompt, "edge case") {
		return `[
			{
				"name": "Empty Input Edge Case",
				"description": "Test behavior with empty input values",
				"type": "edge_case",
				"priority": "high",
				"edge_cases": [
					{
						"name": "Empty String Input",
						"input": "",
						"description": "Test with empty string input",
						"risk": "medium"
					}
				],
				"test_data": {
					"input": "",
					"expected_error": "validation_error"
				},
				"expected": {
					"status": "error",
					"response": null,
					"errors": ["Input cannot be empty"]
				},
				"confidence": 0.9
			}
		]`, nil
	}
	
	if contains(prompt, "fuzzing") {
		return `[
			{
				"name": "SQL Injection Fuzzing",
				"description": "Test API endpoint against SQL injection attacks",
				"type": "fuzzing",
				"priority": "critical",
				"test_data": {
					"malicious_payload": "'; DROP TABLE articles; --",
					"endpoint": "/api/articles"
				},
				"expected": {
					"status": "error",
					"response": "Invalid input detected"
				},
				"confidence": 0.95
			}
		]`, nil
	}
	
	if contains(prompt, "performance") {
		return `[
			{
				"name": "High Load Performance Test",
				"description": "Test system under high concurrent load",
				"type": "performance",
				"priority": "high",
				"test_data": {
					"concurrent_users": 1000,
					"duration": "5m",
					"ramp_up": "1m"
				},
				"expected": {
					"performance": {
						"max_duration": "1s",
						"max_memory": 1073741824,
						"max_cpu": 80.0
					}
				},
				"confidence": 0.85
			}
		]`, nil
	}
	
	if contains(prompt, "test selection") {
		return `{
			"impacted_tests": ["TestArticleRepository", "TestArticleService"],
			"priority_tests": ["TestAuthSecurity", "TestUserAuth"],
			"skipped_tests": ["TestLongRunningIntegration"],
			"selection_reason": "AI-based selection considering code changes and performance data",
			"estimated_time_minutes": 15,
			"coverage_impact_percent": 85.0
		}`, nil
	}
	
	if contains(prompt, "failure analysis") {
		return `{
			"root_cause": "Race condition in concurrent database access",
			"impact_assessment": "High - affects system reliability",
			"immediate_solutions": ["Add proper database locking", "Implement retry mechanism"],
			"long_term_solutions": ["Redesign concurrent access patterns", "Implement connection pooling"],
			"prevention_strategies": ["Add concurrency tests", "Implement monitoring"],
			"confidence_score": 0.88
		}`, nil
	}
	
	if contains(prompt, "debug") {
		return `{
			"root_cause": "Nil pointer dereference in article processing",
			"error_explanation": "The code attempts to access a field on a nil pointer",
			"fix_suggestions": [
				{
					"description": "Add nil check before accessing pointer",
					"code": "if article != nil { return article.Title }",
					"priority": "high",
					"confidence": 0.9
				}
			],
			"related_issues": ["Similar nil pointer issues in other handlers"],
			"confidence": 0.92,
			"debugging_steps": [
				{
					"step": 1,
					"description": "Check if article is nil before processing",
					"command": "go test -v -run TestArticleProcessing",
					"expected": "Should show nil pointer location"
				}
			]
		}`, nil
	}
	
	// Default response
	return "AI response not configured for this prompt type", nil
}

func (m *MockLLMClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	// For structured generation, return the same as GenerateText but parsed
	response, err := m.GenerateText(ctx, prompt)
	return response, err
}

func contains(text, substr string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substr))
}

// Test AI Test Generator
func TestAITestGenerator_GenerateEdgeCaseScenarios(t *testing.T) {
	mockClient := NewMockLLMClient()
	config := &AITestConfig{
		LLMProvider: "mock",
		Model:       "test-model",
		MaxTokens:   1000,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
	}
	
	generator := NewAITestGenerator(mockClient, config)
	
	ctx := context.Background()
	requirements := "Test article creation with various input conditions"
	codeContext := "func CreateArticle(title, content string) (*Article, error)"
	
	scenarios, err := generator.GenerateEdgeCaseScenarios(ctx, requirements, codeContext)
	if err != nil {
		t.Fatalf("Failed to generate edge case scenarios: %v", err)
	}
	
	if len(scenarios) == 0 {
		t.Fatal("No scenarios generated")
	}
	
	scenario := scenarios[0]
	if scenario.Name == "" {
		t.Error("Scenario name is empty")
	}
	
	if scenario.Type != EdgeCaseScenario {
		t.Errorf("Expected scenario type %s, got %s", EdgeCaseScenario, scenario.Type)
	}
	
	if scenario.Confidence < 0.5 {
		t.Errorf("Scenario confidence too low: %f", scenario.Confidence)
	}
}

func TestAITestGenerator_GenerateAPIFuzzingScenarios(t *testing.T) {
	mockClient := NewMockLLMClient()
	config := &AITestConfig{
		LLMProvider: "mock",
		Model:       "test-model",
	}
	
	generator := NewAITestGenerator(mockClient, config)
	
	endpoint := APIEndpoint{
		Path:   "/api/articles",
		Method: "POST",
		Parameters: []Parameter{
			{Name: "title", Type: "string"},
			{Name: "content", Type: "string"},
		},
	}
	
	ctx := context.Background()
	scenarios, err := generator.GenerateAPIFuzzingScenarios(ctx, endpoint)
	if err != nil {
		t.Fatalf("Failed to generate fuzzing scenarios: %v", err)
	}
	
	if len(scenarios) == 0 {
		t.Fatal("No fuzzing scenarios generated")
	}
	
	scenario := scenarios[0]
	if scenario.Type != FuzzingScenario {
		t.Errorf("Expected scenario type %s, got %s", FuzzingScenario, scenario.Type)
	}
	
	if scenario.Priority != PriorityCritical {
		t.Errorf("Expected critical priority for security scenario, got %s", scenario.Priority)
	}
}

func TestAITestGenerator_GeneratePerformanceScenarios(t *testing.T) {
	mockClient := NewMockLLMClient()
	config := &AITestConfig{
		LLMProvider: "mock",
		Model:       "test-model",
	}
	
	generator := NewAITestGenerator(mockClient, config)
	
	baseline := PerformanceBaseline{
		AvgResponseTime: 100 * time.Millisecond,
		PeakThroughput:  1000,
		MemoryUsage:     512 * 1024 * 1024,
		CPUUsage:        50.0,
	}
	
	ctx := context.Background()
	requirements := "Test article publishing performance under load"
	
	scenarios, err := generator.GeneratePerformanceScenarios(ctx, requirements, baseline)
	if err != nil {
		t.Fatalf("Failed to generate performance scenarios: %v", err)
	}
	
	if len(scenarios) == 0 {
		t.Fatal("No performance scenarios generated")
	}
	
	scenario := scenarios[0]
	if scenario.Type != PerformanceScenario {
		t.Errorf("Expected scenario type %s, got %s", PerformanceScenario, scenario.Type)
	}
	
	if scenario.Expected.Performance.MaxDuration == 0 {
		t.Error("Performance scenario should have duration expectations")
	}
}

// Test AI Test Optimizer
func TestAITestOptimizer_OptimizeTestExecution(t *testing.T) {
	mockClient := NewMockLLMClient()
	config := &OptimizerConfig{
		EnableIntelligentSelection: true,
		EnableParallelOptimization: true,
		EnableFailureAnalysis:      true,
		EnableCoverageOptimization: true,
		MaxExecutionTime:          30 * time.Minute,
		ConfidenceThreshold:       0.7,
	}
	
	optimizer := NewAITestOptimizer(mockClient, config)
	
	testSuite := []string{
		"TestArticleRepository",
		"TestArticleService", 
		"TestUserAuth",
		"TestAuthSecurity",
		"TestLongRunningIntegration",
	}
	
	changedFiles := []string{
		"internal/repositories/article_repository.go",
		"internal/services/article_service.go",
	}
	
	ctx := context.Background()
	strategy, err := optimizer.OptimizeTestExecution(ctx, testSuite, changedFiles)
	if err != nil {
		t.Fatalf("Failed to optimize test execution: %v", err)
	}
	
	if len(strategy.ImpactedTests) == 0 {
		t.Error("Should have identified impacted tests")
	}
	
	if len(strategy.PriorityTests) == 0 {
		t.Error("Should have identified priority tests")
	}
	
	if strategy.EstimatedTime == 0 {
		t.Error("Should have estimated execution time")
	}
	
	if strategy.CoverageImpact < 0 || strategy.CoverageImpact > 100 {
		t.Errorf("Coverage impact should be between 0-100, got %f", strategy.CoverageImpact)
	}
}

func TestAITestOptimizer_AnalyzeTestFailures(t *testing.T) {
	mockClient := NewMockLLMClient()
	config := &OptimizerConfig{
		EnableFailureAnalysis: true,
		ConfidenceThreshold:   0.7,
	}
	
	optimizer := NewAITestOptimizer(mockClient, config)
	
	failures := []TestFailure{
		{
			TestName:     "TestArticleCreation",
			TestFile:     "article_test.go",
			ErrorMessage: "timeout waiting for database response",
			StackTrace:   "goroutine 1 [running]: ...",
			FailedAt:     time.Now(),
			FailureType:  "timeout",
		},
		{
			TestName:     "TestArticleUpdate",
			TestFile:     "article_test.go", 
			ErrorMessage: "timeout waiting for database response",
			StackTrace:   "goroutine 2 [running]: ...",
			FailedAt:     time.Now(),
			FailureType:  "timeout",
		},
	}
	
	ctx := context.Background()
	patterns, err := optimizer.AnalyzeTestFailures(ctx, failures)
	if err != nil {
		t.Fatalf("Failed to analyze test failures: %v", err)
	}
	
	if len(patterns) == 0 {
		t.Error("Should have detected failure patterns")
	}
	
	pattern := patterns[0]
	if pattern.Frequency < 2 {
		t.Errorf("Expected pattern frequency >= 2, got %d", pattern.Frequency)
	}
	
	if pattern.RootCause == "" {
		t.Error("Pattern should have root cause analysis")
	}
	
	if pattern.Solution == "" {
		t.Error("Pattern should have solution recommendation")
	}
}

func TestAITestOptimizer_ProvideDebugAssistance(t *testing.T) {
	mockClient := NewMockLLMClient()
	config := &OptimizerConfig{
		ConfidenceThreshold: 0.7,
	}
	
	optimizer := NewAITestOptimizer(mockClient, config)
	
	failure := TestFailure{
		TestName:     "TestArticleProcessing",
		TestFile:     "article_test.go",
		ErrorMessage: "runtime error: invalid memory address or nil pointer dereference",
		StackTrace:   "panic: runtime error: invalid memory address or nil pointer dereference\n[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x...]",
		FailedAt:     time.Now(),
		FailureType:  "panic",
	}
	
	logs := []string{
		"INFO: Processing article ID 123",
		"ERROR: Article not found in database",
		"PANIC: Attempted to access nil article pointer",
	}
	
	ctx := context.Background()
	analysis, err := optimizer.ProvideDebugAssistance(ctx, failure, logs)
	if err != nil {
		t.Fatalf("Failed to provide debug assistance: %v", err)
	}
	
	if analysis.RootCause == "" {
		t.Error("Debug analysis should provide root cause")
	}
	
	if analysis.ErrorExplanation == "" {
		t.Error("Debug analysis should provide error explanation")
	}
	
	if len(analysis.FixSuggestions) == 0 {
		t.Error("Debug analysis should provide fix suggestions")
	}
	
	if len(analysis.DebuggingSteps) == 0 {
		t.Error("Debug analysis should provide debugging steps")
	}
	
	if analysis.Confidence < 0.7 {
		t.Errorf("Debug analysis confidence too low: %f", analysis.Confidence)
	}
}

func TestAITestOptimizer_OptimizeTestCoverage(t *testing.T) {
	mockClient := NewMockLLMClient()
	config := &OptimizerConfig{
		EnableCoverageOptimization: true,
		ConfidenceThreshold:        0.7,
	}
	
	optimizer := NewAITestOptimizer(mockClient, config)
	
	coverageData := map[string]CoverageInfo{
		"internal/services/article_service.go": {
			FilePath:        "internal/services/article_service.go",
			TotalLines:      100,
			CoveredLines:    85,
			CoveragePercent: 85.0,
			UncoveredLines:  []int{10, 15, 20, 25, 30},
			TestsContributing: []string{"TestArticleService", "TestArticleCreation"},
		},
		"internal/repositories/article_repository.go": {
			FilePath:        "internal/repositories/article_repository.go",
			TotalLines:      150,
			CoveredLines:    120,
			CoveragePercent: 80.0,
			UncoveredLines:  []int{5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
			TestsContributing: []string{"TestArticleRepository", "TestArticleDB"},
		},
	}
	
	ctx := context.Background()
	recommendations, err := optimizer.OptimizeTestCoverage(ctx, coverageData)
	if err != nil {
		t.Fatalf("Failed to optimize test coverage: %v", err)
	}
	
	if len(recommendations) == 0 {
		t.Error("Should have generated coverage optimization recommendations")
	}
	
	rec := recommendations[0]
	if rec.Type == "" {
		t.Error("Recommendation should have a type")
	}
	
	if rec.Description == "" {
		t.Error("Recommendation should have a description")
	}
	
	if rec.Confidence < 0.5 {
		t.Errorf("Recommendation confidence too low: %f", rec.Confidence)
	}
}

// Test AI Test Data Generator
func TestAITestDataGenerator_GenerateRealisticTestData(t *testing.T) {
	mockClient := NewMockLLMClient()
	config := &AIDataConfig{
		DefaultLanguage: "en",
		MaxArticles:     100,
		ContentLength: ContentLengthRange{
			MinWords: 100,
			MaxWords: 500,
		},
		Relationships: RelationshipConfig{
			TranslationRate: 0.3,
			CategoryRate:    0.8,
			TagsPerArticle:  3,
			AuthorsCount:    10,
			CategoriesCount: 5,
		},
	}
	
	generator := NewAITestDataGenerator(mockClient, config)
	
	ctx := context.Background()
	data, err := generator.GenerateRealisticTestData(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to generate test data: %v", err)
	}
	
	if len(data.Articles) == 0 {
		t.Error("Should have generated articles")
	}
	
	if len(data.Authors) == 0 {
		t.Error("Should have generated authors")
	}
	
	if len(data.Categories) == 0 {
		t.Error("Should have generated categories")
	}
	
	if data.Metadata.TotalArticles != len(data.Articles) {
		t.Errorf("Metadata total articles mismatch: expected %d, got %d", 
			len(data.Articles), data.Metadata.TotalArticles)
	}
	
	// Check multilingual support
	hasMultipleLanguages := false
	languages := make(map[string]bool)
	for _, article := range data.Articles {
		languages[article.Language] = true
	}
	if len(languages) > 1 {
		hasMultipleLanguages = true
	}
	
	if !hasMultipleLanguages {
		t.Log("Note: Generated data should ideally include multiple languages")
	}
	
	// Check SEO metadata
	for _, article := range data.Articles {
		if article.SEOMetadata.MetaTitle == "" {
			t.Error("Article should have SEO meta title")
		}
		if article.SEOMetadata.CanonicalURL == "" {
			t.Error("Article should have canonical URL")
		}
	}
}

// Test AI Test Scenario Creator
func TestAITestScenarioCreator_CreateTestSuite(t *testing.T) {
	mockClient := NewMockLLMClient()
	config := &ScenarioConfig{
		MaxScenariosPerRequirement: 5,
		ScenarioComplexity:         "medium",
		IncludeNegativeTests:       true,
		IncludePerformanceTests:    true,
		IncludeSecurityTests:       true,
		TestTimeout:                30 * time.Second,
	}
	
	creator := NewAITestScenarioCreator(mockClient, config)
	
	request := ScenarioCreationRequest{
		Requirements: []string{
			"WHEN article is published THEN system SHALL validate content and generate SEO metadata",
			"WHEN user authentication fails THEN system SHALL log attempt and return error",
		},
		CodeContext:      "Article publishing and user authentication system",
		TestTypes:        []TestScenarioType{EdgeCaseScenario, SecurityScenario, PerformanceScenario},
		Priority:         PriorityHigh,
		MaxScenarios:     10,
		IncludeEdgeCases: true,
	}
	
	ctx := context.Background()
	suite, err := creator.CreateTestSuite(ctx, request)
	if err != nil {
		t.Fatalf("Failed to create test suite: %v", err)
	}
	
	if suite.Name == "" {
		t.Error("Test suite should have a name")
	}
	
	if len(suite.Scenarios) == 0 {
		t.Error("Test suite should contain scenarios")
	}
	
	if suite.Coverage.TotalRequirements != len(request.Requirements) {
		t.Errorf("Coverage should track total requirements: expected %d, got %d",
			len(request.Requirements), suite.Coverage.TotalRequirements)
	}
	
	// Check scenario types
	hasEdgeCase := false
	hasSecurity := false
	hasPerformance := false
	
	for _, scenario := range suite.Scenarios {
		switch scenario.Type {
		case EdgeCaseScenario:
			hasEdgeCase = true
		case SecurityScenario:
			hasSecurity = true
		case PerformanceScenario:
			hasPerformance = true
		}
	}
	
	if !hasEdgeCase {
		t.Log("Note: Test suite should ideally include edge case scenarios")
	}
	if !hasSecurity {
		t.Log("Note: Test suite should ideally include security scenarios")
	}
	if !hasPerformance {
		t.Log("Note: Test suite should ideally include performance scenarios")
	}
}

// Test AI Test Maintenance
func TestAITestMaintenance_AnalyzeTestSuite(t *testing.T) {
	mockClient := NewMockLLMClient()
	config := &MaintenanceConfig{
		AutoUpdateEnabled:    false,
		UpdateFrequency:      24 * time.Hour,
		ConfidenceThreshold:  0.7,
		MaxUpdatesPerRun:     10,
		BackupBeforeUpdate:   true,
		RequireHumanApproval: true,
	}
	
	maintenance := NewAITestMaintenance(mockClient, config)
	
	testPaths := []string{
		"internal/services/article_service_test.go",
		"internal/repositories/article_repository_test.go",
		"internal/auth/auth_test.go",
	}
	
	ctx := context.Background()
	report, err := maintenance.AnalyzeTestSuite(ctx, testPaths)
	if err != nil {
		t.Fatalf("Failed to analyze test suite: %v", err)
	}
	
	if report.ID == "" {
		t.Error("Report should have an ID")
	}
	
	if report.TestsAnalyzed != len(testPaths) {
		t.Errorf("Should have analyzed %d tests, got %d", len(testPaths), report.TestsAnalyzed)
	}
	
	if report.OverallHealth.OverallScore < 0 || report.OverallHealth.OverallScore > 1 {
		t.Errorf("Overall health score should be between 0-1, got %f", report.OverallHealth.OverallScore)
	}
	
	// Check that we have some analysis results
	if len(report.IssuesFound) == 0 && len(report.Recommendations) == 0 {
		t.Log("Note: Analysis should typically find some issues or recommendations")
	}
}

// Integration test for the complete AI testing workflow
func TestAITestingWorkflow_Integration(t *testing.T) {
	mockClient := NewMockLLMClient()
	
	// Step 1: Generate test scenarios
	scenarioConfig := &ScenarioConfig{
		MaxScenariosPerRequirement: 3,
		IncludeNegativeTests:       true,
		IncludePerformanceTests:    true,
		IncludeSecurityTests:       true,
	}
	
	creator := NewAITestScenarioCreator(mockClient, scenarioConfig)
	
	request := ScenarioCreationRequest{
		Requirements: []string{
			"WHEN article is created THEN system SHALL validate input and store in database",
		},
		MaxScenarios:     5,
		IncludeEdgeCases: true,
	}
	
	ctx := context.Background()
	suite, err := creator.CreateTestSuite(ctx, request)
	if err != nil {
		t.Fatalf("Failed to create test suite: %v", err)
	}
	
	// Step 2: Generate test data
	dataConfig := &AIDataConfig{
		MaxArticles: 50,
		ContentLength: ContentLengthRange{MinWords: 100, MaxWords: 300},
		Relationships: RelationshipConfig{
			AuthorsCount:    5,
			CategoriesCount: 3,
			TagsPerArticle:  2,
		},
	}
	
	dataGenerator := NewAITestDataGenerator(mockClient, dataConfig)
	testData, err := dataGenerator.GenerateRealisticTestData(ctx, 20)
	if err != nil {
		t.Fatalf("Failed to generate test data: %v", err)
	}
	
	// Step 3: Optimize test execution
	optimizerConfig := &OptimizerConfig{
		EnableIntelligentSelection: true,
		EnableFailureAnalysis:      true,
		ConfidenceThreshold:        0.7,
	}
	
	optimizer := NewAITestOptimizer(mockClient, optimizerConfig)
	
	testSuite := []string{"TestArticleCreation", "TestArticleValidation", "TestArticleStorage"}
	changedFiles := []string{"internal/services/article_service.go"}
	
	strategy, err := optimizer.OptimizeTestExecution(ctx, testSuite, changedFiles)
	if err != nil {
		t.Fatalf("Failed to optimize test execution: %v", err)
	}
	
	// Step 4: Analyze for maintenance
	maintenanceConfig := &MaintenanceConfig{
		ConfidenceThreshold: 0.7,
		MaxUpdatesPerRun:    5,
	}
	
	maintenance := NewAITestMaintenance(mockClient, maintenanceConfig)
	
	testPaths := []string{"internal/services/article_service_test.go"}
	report, err := maintenance.AnalyzeTestSuite(ctx, testPaths)
	if err != nil {
		t.Fatalf("Failed to analyze test suite for maintenance: %v", err)
	}
	
	// Verify the complete workflow
	if len(suite.Scenarios) == 0 {
		t.Error("Workflow should generate test scenarios")
	}
	
	if len(testData.Articles) == 0 {
		t.Error("Workflow should generate test data")
	}
	
	if len(strategy.ImpactedTests) == 0 {
		t.Error("Workflow should identify impacted tests")
	}
	
	if report.TestsAnalyzed == 0 {
		t.Error("Workflow should analyze tests for maintenance")
	}
	
	t.Logf("AI Testing Workflow completed successfully:")
	t.Logf("- Generated %d test scenarios", len(suite.Scenarios))
	t.Logf("- Generated %d test articles", len(testData.Articles))
	t.Logf("- Identified %d impacted tests", len(strategy.ImpactedTests))
	t.Logf("- Analyzed %d test files", report.TestsAnalyzed)
	t.Logf("- Overall test health score: %.2f", report.OverallHealth.OverallScore)
}