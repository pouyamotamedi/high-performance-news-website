package testing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// ReliabilityCLIDemo demonstrates the test reliability tracking system
type ReliabilityCLIDemo struct {
	db      *sql.DB
	tracker *TestReliabilityTracker
}

// NewReliabilityCLIDemo creates a new CLI demo
func NewReliabilityCLIDemo(databaseURL string) (*ReliabilityCLIDemo, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	config := DefaultTestReliabilityConfig()
	tracker := NewTestReliabilityTracker(db, config)

	return &ReliabilityCLIDemo{
		db:      db,
		tracker: tracker,
	}, nil
}

// RunDemo runs the complete demonstration
func (d *ReliabilityCLIDemo) RunDemo() error {
	fmt.Println("🔍 Test Reliability Tracking System Demo")
	fmt.Println("========================================")

	// 1. Generate sample test data
	fmt.Println("\n📊 Generating sample test execution data...")
	if err := d.generateSampleData(); err != nil {
		return fmt.Errorf("failed to generate sample data: %w", err)
	}

	// 2. Demonstrate reliability metrics
	fmt.Println("\n📈 Analyzing test reliability metrics...")
	if err := d.demonstrateReliabilityMetrics(); err != nil {
		return fmt.Errorf("failed to demonstrate reliability metrics: %w", err)
	}

	// 3. Demonstrate flaky test detection
	fmt.Println("\n🚨 Detecting flaky tests...")
	if err := d.demonstrateFlakyTestDetection(); err != nil {
		return fmt.Errorf("failed to demonstrate flaky test detection: %w", err)
	}

	// 4. Demonstrate remediation suggestions
	fmt.Println("\n💡 Generating remediation suggestions...")
	if err := d.demonstrateRemediationSuggestions(); err != nil {
		return fmt.Errorf("failed to demonstrate remediation suggestions: %w", err)
	}

	// 5. Demonstrate stability optimization
	fmt.Println("\n⚡ Running stability optimization...")
	if err := d.demonstrateStabilityOptimization(); err != nil {
		return fmt.Errorf("failed to demonstrate stability optimization: %w", err)
	}

	// 6. Generate comprehensive report
	fmt.Println("\n📋 Generating stability report...")
	if err := d.generateStabilityReport(); err != nil {
		return fmt.Errorf("failed to generate stability report: %w", err)
	}

	fmt.Println("\n✅ Demo completed successfully!")
	return nil
}

// generateSampleData creates realistic test execution data
func (d *ReliabilityCLIDemo) generateSampleData() error {
	testCases := []struct {
		name        string
		suite       string
		pattern     string // "stable", "flaky", "degrading", "environment_specific"
		executions  int
	}{
		{"UserLoginTest", "AuthSuite", "stable", 50},
		{"DatabaseConnectionTest", "IntegrationSuite", "flaky", 40},
		{"PaymentProcessingTest", "PaymentSuite", "degrading", 35},
		{"FileUploadTest", "FilesSuite", "environment_specific", 45},
		{"SearchFunctionalityTest", "SearchSuite", "stable", 30},
		{"CachePerformanceTest", "PerformanceSuite", "flaky", 25},
	}

	for _, tc := range testCases {
		fmt.Printf("  Creating %d executions for %s (%s pattern)...\n", tc.executions, tc.name, tc.pattern)
		
		for i := 0; i < tc.executions; i++ {
			execution := d.generateExecution(tc.name, tc.suite, tc.pattern, i, tc.executions)
			if err := d.tracker.TrackTestExecution(execution); err != nil {
				return fmt.Errorf("failed to track execution: %w", err)
			}
		}
	}

	fmt.Printf("  ✅ Generated %d test executions across %d test cases\n", 
		func() int { 
			total := 0
			for _, tc := range testCases { 
				total += tc.executions 
			}
			return total
		}(), len(testCases))

	return nil
}

// generateExecution creates a single test execution based on pattern
func (d *ReliabilityCLIDemo) generateExecution(testName, testSuite, pattern string, index, total int) *TestExecutionRecord {
	baseTime := time.Now().Add(-time.Duration(total-index) * 10 * time.Minute)
	
	execution := &TestExecutionRecord{
		TestName:    testName,
		TestSuite:   testSuite,
		StartTime:   baseTime,
		Environment: d.selectEnvironment(pattern, index),
		BuildID:     fmt.Sprintf("build-%d", 1000+index),
		CommitHash:  fmt.Sprintf("commit%d", index),
		Branch:      "main",
	}

	// Generate status and duration based on pattern
	switch pattern {
	case "stable":
		execution.Status = "passed"
		execution.Duration = time.Duration(80+rand.Intn(40)) * time.Millisecond
		if rand.Float32() < 0.05 { // 5% failure rate
			execution.Status = "failed"
			execution.ErrorMessage = "Rare infrastructure issue"
			execution.Duration = time.Duration(200+rand.Intn(100)) * time.Millisecond
		}

	case "flaky":
		// Intermittent pattern - alternating with some randomness
		if (index%3 == 0 && rand.Float32() < 0.7) || rand.Float32() < 0.3 {
			execution.Status = "failed"
			execution.ErrorMessage = d.selectFlakyError()
			execution.Duration = time.Duration(150+rand.Intn(200)) * time.Millisecond
		} else {
			execution.Status = "passed"
			execution.Duration = time.Duration(90+rand.Intn(60)) * time.Millisecond
		}

	case "degrading":
		// Increasing failure rate over time
		failureRate := float32(index) / float32(total) * 0.6 // Up to 60% failure rate
		if rand.Float32() < failureRate {
			execution.Status = "failed"
			execution.ErrorMessage = "Performance degradation detected"
			execution.Duration = time.Duration(200+rand.Intn(300)) * time.Millisecond
		} else {
			execution.Status = "passed"
			execution.Duration = time.Duration(100+rand.Intn(100)) * time.Millisecond
		}

	case "environment_specific":
		if execution.Environment == "prod" && rand.Float32() < 0.6 {
			execution.Status = "failed"
			execution.ErrorMessage = "Production environment configuration issue"
			execution.Duration = time.Duration(180+rand.Intn(150)) * time.Millisecond
		} else {
			execution.Status = "passed"
			execution.Duration = time.Duration(85+rand.Intn(50)) * time.Millisecond
		}
	}

	execution.EndTime = execution.StartTime.Add(execution.Duration)
	return execution
}

// selectEnvironment chooses environment based on pattern
func (d *ReliabilityCLIDemo) selectEnvironment(pattern string, index int) string {
	environments := []string{"test", "staging", "prod"}
	
	if pattern == "environment_specific" {
		// More production executions for environment-specific issues
		if index%3 == 0 {
			return "prod"
		}
	}
	
	return environments[index%len(environments)]
}

// selectFlakyError returns a random flaky test error message
func (d *ReliabilityCLIDemo) selectFlakyError() string {
	errors := []string{
		"Connection timeout after 30 seconds",
		"Element not found: retry limit exceeded",
		"Network unreachable",
		"Assertion failed: expected 'true' but was 'false'",
		"Database connection pool exhausted",
		"Race condition detected in concurrent access",
		"Memory allocation failed",
		"Service temporarily unavailable",
	}
	return errors[rand.Intn(len(errors))]
}

// demonstrateReliabilityMetrics shows reliability metrics for each test
func (d *ReliabilityCLIDemo) demonstrateReliabilityMetrics() error {
	tests := []struct{ name, suite string }{
		{"UserLoginTest", "AuthSuite"},
		{"DatabaseConnectionTest", "IntegrationSuite"},
		{"PaymentProcessingTest", "PaymentSuite"},
		{"FileUploadTest", "FilesSuite"},
	}

	for _, test := range tests {
		metrics, err := d.tracker.GetTestReliabilityMetrics(test.name, test.suite)
		if err != nil {
			return err
		}

		fmt.Printf("\n  📊 %s.%s:\n", test.suite, test.name)
		fmt.Printf("     Reliability Score: %.3f\n", metrics.ReliabilityScore)
		fmt.Printf("     Flakiness Score: %.3f\n", metrics.FlakinessScore)
		fmt.Printf("     Stability Trend: %s\n", metrics.StabilityTrend)
		fmt.Printf("     Total Executions: %d\n", metrics.TotalExecutions)
		fmt.Printf("     Success Rate: %.1f%%\n", 
			float64(metrics.SuccessfulExecutions)/float64(metrics.TotalExecutions)*100)
		fmt.Printf("     Average Duration: %v\n", metrics.AverageDuration)

		if len(metrics.FailurePatterns) > 0 {
			fmt.Printf("     Failure Patterns:\n")
			for _, pattern := range metrics.FailurePatterns {
				fmt.Printf("       - %s: %s (confidence: %.2f)\n", 
					pattern.Type, pattern.Description, pattern.Confidence)
			}
		}

		if len(metrics.EnvironmentImpact) > 0 {
			fmt.Printf("     Environment Impact:\n")
			for env, rate := range metrics.EnvironmentImpact {
				fmt.Printf("       - %s: %.1f%% failure rate\n", env, rate*100)
			}
		}
	}

	return nil
}

// demonstrateFlakyTestDetection shows flaky test detection
func (d *ReliabilityCLIDemo) demonstrateFlakyTestDetection() error {
	flakyTests, err := d.tracker.GetFlakyTests()
	if err != nil {
		return err
	}

	if len(flakyTests) == 0 {
		fmt.Println("  ✅ No flaky tests detected")
		return nil
	}

	fmt.Printf("  🚨 Found %d flaky tests:\n", len(flakyTests))
	for i, test := range flakyTests {
		if i >= 5 { // Limit output
			fmt.Printf("  ... and %d more\n", len(flakyTests)-i)
			break
		}

		fmt.Printf("\n  %d. %s.%s\n", i+1, test.TestSuite, test.TestName)
		fmt.Printf("     Flakiness Score: %.3f\n", test.FlakinessScore)
		fmt.Printf("     Reliability Score: %.3f\n", test.ReliabilityScore)
		fmt.Printf("     Failure Rate: %.1f%%\n", 
			float64(test.FailedExecutions)/float64(test.TotalExecutions)*100)

		// Show top failure patterns
		if len(test.FailurePatterns) > 0 {
			fmt.Printf("     Top Patterns:\n")
			for j, pattern := range test.FailurePatterns {
				if j >= 2 { // Show top 2 patterns
					break
				}
				fmt.Printf("       - %s (%.1f%% frequency)\n", 
					pattern.Type, pattern.Frequency*100)
			}
		}
	}

	return nil
}

// demonstrateRemediationSuggestions shows remediation suggestions
func (d *ReliabilityCLIDemo) demonstrateRemediationSuggestions() error {
	// Get a flaky test for demonstration
	flakyTests, err := d.tracker.GetFlakyTests()
	if err != nil {
		return err
	}

	if len(flakyTests) == 0 {
		fmt.Println("  ℹ️  No flaky tests available for remediation suggestions")
		return nil
	}

	test := flakyTests[0] // Use the most flaky test
	fmt.Printf("  🔧 Remediation suggestions for %s.%s:\n", test.TestSuite, test.TestName)

	engine := NewRemediationEngine()
	engine.SetDatabase(d.db)
	
	suggestions, err := engine.GenerateRemediationSuggestions(test.TestName, test.TestSuite)
	if err != nil {
		return err
	}

	if len(suggestions) == 0 {
		fmt.Println("     No specific suggestions available")
		return nil
	}

	for i, suggestion := range suggestions {
		if i >= 3 { // Limit to top 3 suggestions
			break
		}

		fmt.Printf("\n     %d. %s (%s priority)\n", i+1, suggestion.Description, suggestion.Priority)
		fmt.Printf("        Action: %s\n", suggestion.Action)
		fmt.Printf("        Confidence: %.1f%%\n", suggestion.Confidence*100)
		
		if len(suggestion.Examples) > 0 {
			fmt.Printf("        Examples:\n")
			for j, example := range suggestion.Examples {
				if j >= 2 { // Show top 2 examples
					break
				}
				fmt.Printf("          - %s\n", example)
			}
		}
	}

	return nil
}

// demonstrateStabilityOptimization shows the stability optimization process
func (d *ReliabilityCLIDemo) demonstrateStabilityOptimization() error {
	config := DefaultStabilityOptimizerConfig()
	config.AutoRemediationEnabled = false // Disable for demo
	config.AutoQuarantineEnabled = true

	optimizer := NewTestStabilityOptimizer(d.db, d.tracker, config)
	
	report, err := optimizer.OptimizeTestStability()
	if err != nil {
		return err
	}

	fmt.Printf("  ⚡ Optimization completed in %v\n", report.Duration)
	fmt.Printf("     Unstable tests found: %d\n", report.UnstableTestsFound)
	fmt.Printf("     Tests quarantined: %d\n", report.TestsQuarantined)
	fmt.Printf("     Environment optimizations: %d\n", report.EnvironmentOptimizations)
	fmt.Printf("     Total optimizations applied: %d\n", len(report.Optimizations))

	if len(report.Optimizations) > 0 {
		fmt.Printf("\n     Recent optimizations:\n")
		for i, opt := range report.Optimizations {
			if i >= 3 { // Show top 3
				break
			}
			status := "❌ Failed"
			if opt.Applied {
				status = "✅ Applied"
			}
			fmt.Printf("       - %s: %s %s\n", opt.Type, opt.Description, status)
		}
	}

	if len(report.Recommendations) > 0 {
		fmt.Printf("\n     High-priority recommendations:\n")
		for i, rec := range report.Recommendations {
			if i >= 3 || (rec.Priority != "critical" && rec.Priority != "high") {
				continue
			}
			fmt.Printf("       - %s (%s): %s\n", rec.Title, rec.Priority, rec.Action)
		}
	}

	return nil
}

// generateStabilityReport creates and displays a comprehensive stability report
func (d *ReliabilityCLIDemo) generateStabilityReport() error {
	report, err := d.tracker.GenerateStabilityReport()
	if err != nil {
		return err
	}

	fmt.Printf("  📋 Test Suite Stability Report\n")
	fmt.Printf("     Generated: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("     Total Tests: %d\n", report.TotalTests)
	fmt.Printf("     Stable Tests: %d (%.1f%%)\n", report.StableTests, 
		float64(report.StableTests)/float64(report.TotalTests)*100)
	fmt.Printf("     Flaky Tests: %d (%.1f%%)\n", report.FlakyTests,
		float64(report.FlakyTests)/float64(report.TotalTests)*100)
	fmt.Printf("     Quarantined Tests: %d\n", report.QuarantinedTests)
	fmt.Printf("     Overall Stability: %.1f%%\n", report.OverallStability*100)
	fmt.Printf("     Stability Trend: %s\n", report.StabilityTrend)

	// Show reliability distribution
	fmt.Printf("\n     Reliability Distribution:\n")
	for level, tests := range report.TestsByReliability {
		fmt.Printf("       %s reliability: %d tests\n", level, len(tests))
	}

	// Show environment analysis
	fmt.Printf("\n     Environment Analysis:\n")
	for env, stability := range report.EnvironmentAnalysis.StabilityByEnv {
		fmt.Printf("       %s: %.1f%% stability\n", env, stability*100)
	}

	if len(report.EnvironmentAnalysis.ProblematicEnvs) > 0 {
		fmt.Printf("     Problematic Environments: %v\n", report.EnvironmentAnalysis.ProblematicEnvs)
	}

	// Show top recommendations
	if len(report.RecommendedActions) > 0 {
		fmt.Printf("\n     Top Recommendations:\n")
		for i, action := range report.RecommendedActions {
			if i >= 3 {
				break
			}
			fmt.Printf("       %d. %s (%s priority)\n", i+1, action.Description, action.Priority)
			fmt.Printf("          %s\n", action.Action)
		}
	}

	return nil
}

// Close closes the database connection
func (d *ReliabilityCLIDemo) Close() error {
	return d.db.Close()
}

// RunReliabilityDemo is the main entry point for the demo
func RunReliabilityDemo() {
	// Get database URL from environment or use default
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/news_website_test?sslmode=disable"
	}

	demo, err := NewReliabilityCLIDemo(databaseURL)
	if err != nil {
		log.Fatalf("Failed to create demo: %v", err)
	}
	defer demo.Close()

	if err := demo.RunDemo(); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}
}