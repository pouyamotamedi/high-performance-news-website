package testing

import (
	"context"
	"fmt"
	"log"
	"time"
)

// AITestingExample demonstrates how to use the AI testing system
type AITestingExample struct {
	generator    *AITestGenerator
	optimizer    *AITestOptimizer
	dataGen      *AITestDataGenerator
	scenarioCreator *AITestScenarioCreator
	maintenance  *AITestMaintenance
}

// NewAITestingExample creates a new AI testing example
func NewAITestingExample(llmClient LLMClient) *AITestingExample {
	return &AITestingExample{
		generator: NewAITestGenerator(llmClient, &AITestConfig{
			LLMProvider: "openai",
			Model:       "gpt-4",
			MaxTokens:   2000,
			Temperature: 0.7,
			Timeout:     30 * time.Second,
		}),
		optimizer: NewAITestOptimizer(llmClient, &OptimizerConfig{
			EnableIntelligentSelection: true,
			EnableParallelOptimization: true,
			EnableFailureAnalysis:      true,
			EnableCoverageOptimization: true,
			ConfidenceThreshold:        0.7,
		}),
		dataGen: NewAITestDataGenerator(llmClient, &AIDataConfig{
			DefaultLanguage: "en",
			MaxArticles:     1000,
			ContentLength: ContentLengthRange{
				MinWords: 100,
				MaxWords: 500,
			},
			Relationships: RelationshipConfig{
				TranslationRate:    0.3,
				CategoryRate:       0.8,
				TagsPerArticle:     3,
				AuthorsCount:       20,
				CategoriesCount:    10,
			},
		}),
		scenarioCreator: NewAITestScenarioCreator(llmClient, &ScenarioConfig{
			MaxScenariosPerRequirement: 5,
			IncludeNegativeTests:       true,
			IncludePerformanceTests:    true,
			IncludeSecurityTests:       true,
		}),
		maintenance: NewAITestMaintenance(llmClient, &MaintenanceConfig{
			AutoUpdateEnabled:    false,
			ConfidenceThreshold:  0.8,
			MaxUpdatesPerRun:     10,
			BackupBeforeUpdate:   true,
			RequireHumanApproval: true,
		}),
	}
}

// DemonstrateCompleteWorkflow shows a complete AI testing workflow
func (e *AITestingExample) DemonstrateCompleteWorkflow(ctx context.Context) error {
	fmt.Println("=== AI Testing System Demonstration ===")
	
	// Step 1: Generate test scenarios from requirements
	fmt.Println("\n1. Generating Test Scenarios from Requirements...")
	
	requirements := []string{
		"WHEN article is published THEN system SHALL validate content and generate SEO metadata",
		"WHEN user authentication fails THEN system SHALL log attempt and return error",
		"WHEN system receives 1000+ concurrent requests THEN response time SHALL remain under 1 second",
	}
	
	request := ScenarioCreationRequest{
		Requirements:     requirements,
		CodeContext:      "High-performance news website with 50K articles/day",
		TestTypes:        []TestScenarioType{EdgeCaseScenario, SecurityScenario, PerformanceScenario},
		Priority:         PriorityHigh,
		MaxScenarios:     15,
		IncludeEdgeCases: true,
	}
	
	testSuite, err := e.scenarioCreator.CreateTestSuite(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to create test suite: %w", err)
	}
	
	fmt.Printf("✓ Generated %d test scenarios\n", len(testSuite.Scenarios))
	fmt.Printf("✓ Coverage: %.1f%% of requirements\n", testSuite.Coverage.CoveragePercentage)
	
	// Step 2: Generate realistic test data
	fmt.Println("\n2. Generating Realistic Test Data...")
	
	testData, err := e.dataGen.GenerateRealisticTestData(ctx, 100)
	if err != nil {
		return fmt.Errorf("failed to generate test data: %w", err)
	}
	
	fmt.Printf("✓ Generated %d articles\n", len(testData.Articles))
	fmt.Printf("✓ Generated %d authors\n", len(testData.Authors))
	fmt.Printf("✓ Generated %d categories\n", len(testData.Categories))
	fmt.Printf("✓ Language distribution: %v\n", testData.Metadata.LanguageDistribution)
	
	// Step 3: Generate edge case scenarios
	fmt.Println("\n3. Generating Edge Case Scenarios...")
	
	edgeCases, err := e.generator.GenerateEdgeCaseScenarios(ctx, 
		"Article publishing with multilingual content", 
		"func PublishArticle(article *Article) error")
	if err != nil {
		return fmt.Errorf("failed to generate edge cases: %w", err)
	}
	
	fmt.Printf("✓ Generated %d edge case scenarios\n", len(edgeCases))
	for _, scenario := range edgeCases {
		fmt.Printf("  - %s (confidence: %.2f)\n", scenario.Name, scenario.Confidence)
	}
	
	// Step 4: Generate API fuzzing scenarios
	fmt.Println("\n4. Generating API Fuzzing Scenarios...")
	
	apiEndpoint := APIEndpoint{
		Path:   "/api/v1/articles",
		Method: "POST",
		Parameters: []Parameter{
			{Name: "title", Type: "string"},
			{Name: "content", Type: "string"},
			{Name: "author_id", Type: "int"},
		},
	}
	
	fuzzingScenarios, err := e.generator.GenerateAPIFuzzingScenarios(ctx, apiEndpoint)
	if err != nil {
		return fmt.Errorf("failed to generate fuzzing scenarios: %w", err)
	}
	
	fmt.Printf("✓ Generated %d fuzzing scenarios\n", len(fuzzingScenarios))
	for _, scenario := range fuzzingScenarios {
		fmt.Printf("  - %s (priority: %s)\n", scenario.Name, scenario.Priority)
	}
	
	// Step 5: Optimize test execution
	fmt.Println("\n5. Optimizing Test Execution...")
	
	testSuiteNames := []string{
		"TestArticleRepository",
		"TestArticleService",
		"TestUserAuthentication",
		"TestSEOGeneration",
		"TestMultilingualContent",
		"TestPerformanceLoad",
	}
	
	changedFiles := []string{
		"internal/services/article_service.go",
		"internal/repositories/article_repository.go",
	}
	
	strategy, err := e.optimizer.OptimizeTestExecution(ctx, testSuiteNames, changedFiles)
	if err != nil {
		return fmt.Errorf("failed to optimize test execution: %w", err)
	}
	
	fmt.Printf("✓ Identified %d impacted tests\n", len(strategy.ImpactedTests))
	fmt.Printf("✓ Identified %d priority tests\n", len(strategy.PriorityTests))
	fmt.Printf("✓ Can skip %d tests\n", len(strategy.SkippedTests))
	fmt.Printf("✓ Estimated execution time: %v\n", strategy.EstimatedTime)
	fmt.Printf("✓ Coverage impact: %.1f%%\n", strategy.CoverageImpact)
	
	// Step 6: Analyze test failures
	fmt.Println("\n6. Analyzing Test Failures...")
	
	sampleFailures := []TestFailure{
		{
			TestName:     "TestArticlePublishing",
			TestFile:     "article_service_test.go",
			ErrorMessage: "timeout waiting for database response",
			StackTrace:   "goroutine 1 [running]: database.Query(...)",
			FailedAt:     time.Now(),
			FailureType:  "timeout",
		},
		{
			TestName:     "TestUserAuth",
			TestFile:     "auth_test.go",
			ErrorMessage: "connection refused",
			StackTrace:   "goroutine 2 [running]: net.Dial(...)",
			FailedAt:     time.Now(),
			FailureType:  "connection",
		},
	}
	
	patterns, err := e.optimizer.AnalyzeTestFailures(ctx, sampleFailures)
	if err != nil {
		return fmt.Errorf("failed to analyze failures: %w", err)
	}
	
	fmt.Printf("✓ Detected %d failure patterns\n", len(patterns))
	for _, pattern := range patterns {
		fmt.Printf("  - %s (frequency: %d, confidence: %.2f)\n", 
			pattern.Pattern, pattern.Frequency, pattern.Confidence)
	}
	
	// Step 7: Provide debug assistance
	fmt.Println("\n7. Providing Debug Assistance...")
	
	debugFailure := TestFailure{
		TestName:     "TestArticleProcessing",
		TestFile:     "article_test.go",
		ErrorMessage: "runtime error: invalid memory address or nil pointer dereference",
		StackTrace:   "panic: runtime error: invalid memory address or nil pointer dereference",
		FailedAt:     time.Now(),
		FailureType:  "panic",
	}
	
	logs := []string{
		"INFO: Processing article ID 123",
		"ERROR: Article not found in database",
		"PANIC: Attempted to access nil article pointer",
	}
	
	debugAnalysis, err := e.optimizer.ProvideDebugAssistance(ctx, debugFailure, logs)
	if err != nil {
		return fmt.Errorf("failed to provide debug assistance: %w", err)
	}
	
	fmt.Printf("✓ Root cause: %s\n", debugAnalysis.RootCause)
	fmt.Printf("✓ Generated %d fix suggestions\n", len(debugAnalysis.FixSuggestions))
	fmt.Printf("✓ Provided %d debugging steps\n", len(debugAnalysis.DebuggingSteps))
	fmt.Printf("✓ Analysis confidence: %.2f\n", debugAnalysis.Confidence)
	
	// Step 8: Optimize test coverage
	fmt.Println("\n8. Optimizing Test Coverage...")
	
	coverageData := map[string]CoverageInfo{
		"internal/services/article_service.go": {
			FilePath:        "internal/services/article_service.go",
			TotalLines:      200,
			CoveredLines:    170,
			CoveragePercent: 85.0,
			UncoveredLines:  []int{10, 15, 20, 25, 30},
			TestsContributing: []string{"TestArticleService", "TestArticleCreation"},
		},
		"internal/repositories/article_repository.go": {
			FilePath:        "internal/repositories/article_repository.go",
			TotalLines:      150,
			CoveredLines:    120,
			CoveragePercent: 80.0,
			UncoveredLines:  []int{5, 10, 15, 20, 25},
			TestsContributing: []string{"TestArticleRepository", "TestArticleDB"},
		},
	}
	
	coverageRecommendations, err := e.optimizer.OptimizeTestCoverage(ctx, coverageData)
	if err != nil {
		return fmt.Errorf("failed to optimize coverage: %w", err)
	}
	
	fmt.Printf("✓ Generated %d coverage optimization recommendations\n", len(coverageRecommendations))
	for _, rec := range coverageRecommendations {
		fmt.Printf("  - %s: %s (confidence: %.2f)\n", 
			rec.Type, rec.Description, rec.Confidence)
	}
	
	// Step 9: Analyze test suite for maintenance
	fmt.Println("\n9. Analyzing Test Suite for Maintenance...")
	
	testPaths := []string{
		"internal/services/article_service_test.go",
		"internal/repositories/article_repository_test.go",
		"internal/auth/auth_test.go",
	}
	
	maintenanceReport, err := e.maintenance.AnalyzeTestSuite(ctx, testPaths)
	if err != nil {
		return fmt.Errorf("failed to analyze test suite: %w", err)
	}
	
	fmt.Printf("✓ Analyzed %d test files\n", maintenanceReport.TestsAnalyzed)
	fmt.Printf("✓ Found %d issues\n", len(maintenanceReport.IssuesFound))
	fmt.Printf("✓ Generated %d recommendations\n", len(maintenanceReport.Recommendations))
	fmt.Printf("✓ Overall health score: %.2f\n", maintenanceReport.OverallHealth.OverallScore)
	
	// Step 10: Summary
	fmt.Println("\n=== AI Testing Workflow Summary ===")
	fmt.Printf("✓ Test Scenarios: %d generated\n", len(testSuite.Scenarios))
	fmt.Printf("✓ Test Data: %d articles, %d authors, %d categories\n", 
		len(testData.Articles), len(testData.Authors), len(testData.Categories))
	fmt.Printf("✓ Edge Cases: %d scenarios\n", len(edgeCases))
	fmt.Printf("✓ Security Tests: %d fuzzing scenarios\n", len(fuzzingScenarios))
	fmt.Printf("✓ Test Optimization: %d impacted, %d priority, %d skippable\n",
		len(strategy.ImpactedTests), len(strategy.PriorityTests), len(strategy.SkippedTests))
	fmt.Printf("✓ Failure Analysis: %d patterns detected\n", len(patterns))
	fmt.Printf("✓ Coverage Optimization: %d recommendations\n", len(coverageRecommendations))
	fmt.Printf("✓ Maintenance Analysis: %.2f health score\n", maintenanceReport.OverallHealth.OverallScore)
	
	fmt.Println("\n🎉 AI Testing System demonstration completed successfully!")
	
	return nil
}

// DemonstrateSpecificFeatures shows specific AI testing features
func (e *AITestingExample) DemonstrateSpecificFeatures(ctx context.Context) error {
	fmt.Println("=== AI Testing Specific Features ===")
	
	// Feature 1: Multilingual test data generation
	fmt.Println("\n1. Multilingual Test Data Generation...")
	
	multilingualData, err := e.dataGen.GenerateRealisticTestData(ctx, 30)
	if err != nil {
		return fmt.Errorf("failed to generate multilingual data: %w", err)
	}
	
	languageCount := make(map[string]int)
	for _, article := range multilingualData.Articles {
		languageCount[article.Language]++
	}
	
	fmt.Printf("✓ Generated articles in %d languages\n", len(languageCount))
	for lang, count := range languageCount {
		fmt.Printf("  - %s: %d articles\n", lang, count)
	}
	
	// Feature 2: Performance scenario generation
	fmt.Println("\n2. Performance Scenario Generation...")
	
	baseline := PerformanceBaseline{
		AvgResponseTime: 100 * time.Millisecond,
		PeakThroughput:  1000,
		MemoryUsage:     512 * 1024 * 1024,
		CPUUsage:        50.0,
	}
	
	perfScenarios, err := e.generator.GeneratePerformanceScenarios(ctx, 
		"Test news website handling 50K articles/day", baseline)
	if err != nil {
		return fmt.Errorf("failed to generate performance scenarios: %w", err)
	}
	
	fmt.Printf("✓ Generated %d performance scenarios\n", len(perfScenarios))
	for _, scenario := range perfScenarios {
		fmt.Printf("  - %s: max duration %v\n", 
			scenario.Name, scenario.Expected.Performance.MaxDuration)
	}
	
	// Feature 3: Intelligent test selection
	fmt.Println("\n3. Intelligent Test Selection...")
	
	allTests := []string{
		"TestArticleCreation", "TestArticleUpdate", "TestArticleDelete",
		"TestUserRegistration", "TestUserLogin", "TestUserLogout",
		"TestSEOGeneration", "TestCanonicalURLs", "TestSchemaMarkup",
		"TestMultilingualContent", "TestTranslationLinks",
		"TestPerformanceLoad", "TestDatabaseConnections", "TestCacheOperations",
	}
	
	recentChanges := []string{
		"internal/services/article_service.go",
		"internal/seo/schema_generator.go",
	}
	
	selection, err := e.optimizer.OptimizeTestExecution(ctx, allTests, recentChanges)
	if err != nil {
		return fmt.Errorf("failed to optimize test selection: %w", err)
	}
	
	fmt.Printf("✓ Selected %d/%d tests to run\n", len(selection.ImpactedTests), len(allTests))
	fmt.Printf("✓ Estimated time savings: %v\n", 
		e.calculateTimeSavings(len(allTests), len(selection.ImpactedTests)))
	fmt.Printf("✓ Maintained %.1f%% coverage\n", selection.CoverageImpact)
	
	// Feature 4: AI-powered debugging
	fmt.Println("\n4. AI-Powered Debugging...")
	
	complexFailure := TestFailure{
		TestName:     "TestComplexArticleWorkflow",
		TestFile:     "integration_test.go",
		ErrorMessage: "assertion failed: expected 3 articles, got 2",
		StackTrace:   "TestComplexArticleWorkflow:45\n  workflow.ProcessArticles:123\n  repository.SaveArticle:67",
		FailedAt:     time.Now(),
		FailureType:  "assertion",
		Context: map[string]interface{}{
			"articles_processed": 3,
			"articles_saved":     2,
			"database_errors":    1,
		},
	}
	
	complexLogs := []string{
		"INFO: Starting article workflow with 3 articles",
		"INFO: Processing article 1: 'Breaking News'",
		"INFO: Processing article 2: 'Tech Update'", 
		"INFO: Processing article 3: 'Sports Report'",
		"ERROR: Database constraint violation for article 3",
		"INFO: Workflow completed with 2 successful saves",
	}
	
	debugging, err := e.optimizer.ProvideDebugAssistance(ctx, complexFailure, complexLogs)
	if err != nil {
		return fmt.Errorf("failed to provide debugging assistance: %w", err)
	}
	
	fmt.Printf("✓ Identified root cause: %s\n", debugging.RootCause)
	fmt.Printf("✓ Provided detailed explanation: %s\n", debugging.ErrorExplanation)
	fmt.Printf("✓ Generated %d actionable fix suggestions\n", len(debugging.FixSuggestions))
	
	if len(debugging.FixSuggestions) > 0 {
		fmt.Printf("  Top suggestion: %s\n", debugging.FixSuggestions[0].Description)
	}
	
	fmt.Println("\n✅ AI Testing Features demonstration completed!")
	
	return nil
}

// calculateTimeSavings estimates time savings from test optimization
func (e *AITestingExample) calculateTimeSavings(totalTests, selectedTests int) time.Duration {
	avgTestTime := 30 * time.Second
	totalTime := time.Duration(totalTests) * avgTestTime
	selectedTime := time.Duration(selectedTests) * avgTestTime
	return totalTime - selectedTime
}

// RunExample runs the complete AI testing example
func RunAITestingExample() {
	// Create a mock LLM client for demonstration
	mockClient := NewMockLLMClient()
	
	// Create the AI testing example
	example := NewAITestingExample(mockClient)
	
	ctx := context.Background()
	
	// Run the complete workflow demonstration
	if err := example.DemonstrateCompleteWorkflow(ctx); err != nil {
		log.Fatalf("Failed to run complete workflow: %v", err)
	}
	
	// Run specific features demonstration
	if err := example.DemonstrateSpecificFeatures(ctx); err != nil {
		log.Fatalf("Failed to run specific features: %v", err)
	}
	
	fmt.Println("\n🚀 AI Testing System is ready for production use!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Configure your LLM provider (OpenAI, Anthropic, etc.)")
	fmt.Println("2. Set up CI/CD integration for automated test optimization")
	fmt.Println("3. Configure test maintenance schedules")
	fmt.Println("4. Train the system on your specific codebase patterns")
	fmt.Println("5. Monitor and refine AI recommendations based on results")
}