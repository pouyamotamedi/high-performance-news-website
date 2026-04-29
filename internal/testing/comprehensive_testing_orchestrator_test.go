package testing

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestComprehensiveTestingOrchestrator tests the comprehensive testing orchestrator
func TestComprehensiveTestingOrchestrator(t *testing.T) {
	// Create mock database connection
	db := createMockDB(t)
	defer db.Close()

	// Create orchestrator with test configuration
	config := &OrchestratorConfig{
		MaxConcurrentEnvironments: 2,
		EnvironmentTimeout:        30 * time.Second,
		MaxParallelTests:          5,
		TestTimeout:               10 * time.Minute,
		RetryAttempts:             2,
		MinCoverageThreshold:      90.0,
		MaxFlakinessThreshold:     5.0,
		PerformanceRegressionLimit: 15.0,
		EnableAITestGeneration:    true,
		AIConfidenceThreshold:     0.6,
		HealthCheckInterval:       10 * time.Second,
		MetricsCollectionInterval: 5 * time.Second,
		ReportGenerationInterval:  1 * time.Hour,
		RetainReportsFor:          7 * 24 * time.Hour,
	}

	orchestrator, err := NewComprehensiveTestingOrchestrator(db, config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Test orchestrator startup
	t.Run("TestOrchestratorStartup", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := orchestrator.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start orchestrator: %v", err)
		}

		// Verify orchestrator is running
		if !orchestrator.isRunning {
			t.Error("Expected orchestrator to be running")
		}

		// Test system health
		health := orchestrator.GetSystemHealth()
		if health.OverallStatus == "" {
			t.Error("Expected non-empty overall status")
		}

		if health.Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp")
		}
	})

	// Test test plan creation
	t.Run("TestCreateOptimizedTestPlan", func(t *testing.T) {
		requirements := TestPlanRequirements{
			Name:                    "Comprehensive Test Plan",
			Description:             "Full system testing with all components",
			TestTypes:               []string{"unit", "integration", "performance", "security"},
			CoverageTarget:          95.0,
			PerformanceTargets:      map[string]float64{"response_time": 500.0, "throughput": 1000.0},
			SecurityRequirements:    []string{"owasp_top_10", "sql_injection", "xss"},
			EnvironmentRequirements: []string{"postgres", "dragonfly", "monitoring"},
			MaxExecutionTime:        45 * time.Minute,
			Priority:                PriorityHigh,
			Tags:                    []string{"comprehensive", "regression"},
		}

		plan, err := orchestrator.CreateOptimizedTestPlan(requirements)
		if err != nil {
			t.Fatalf("Failed to create test plan: %v", err)
		}

		// Verify plan structure
		if plan.ID == "" {
			t.Error("Expected non-empty plan ID")
		}

		if plan.Name != requirements.Name {
			t.Errorf("Expected plan name %s, got %s", requirements.Name, plan.Name)
		}

		if len(plan.QualityGates) == 0 {
			t.Error("Expected quality gates to be created")
		}

		if len(plan.EnvironmentSpecs) == 0 {
			t.Error("Expected environment specs to be created")
		}

		if len(plan.SuccessCriteria) == 0 {
			t.Error("Expected success criteria to be created")
		}

		if plan.ExpectedDuration == 0 {
			t.Error("Expected non-zero execution duration estimate")
		}

		// Verify test suites were created
		totalSuites := len(plan.UnitTests) + len(plan.IntegrationTests) + 
					   len(plan.PerformanceTests) + len(plan.SecurityTests)
		if totalSuites == 0 {
			t.Error("Expected test suites to be created")
		}

		t.Logf("Created test plan with %d test suites, estimated duration: %v", 
			totalSuites, plan.ExpectedDuration)
	})

	// Test comprehensive test execution
	t.Run("TestExecuteComprehensiveTestPlan", func(t *testing.T) {
		// Create a test plan
		requirements := TestPlanRequirements{
			Name:                    "Integration Test Plan",
			Description:             "Integration testing with core components",
			TestTypes:               []string{"unit", "integration"},
			CoverageTarget:          90.0,
			PerformanceTargets:      map[string]float64{"response_time": 1000.0},
			SecurityRequirements:    []string{"basic_security"},
			EnvironmentRequirements: []string{"postgres", "dragonfly"},
			MaxExecutionTime:        20 * time.Minute,
			Priority:                PriorityMedium,
			Tags:                    []string{"integration"},
		}

		plan, err := orchestrator.CreateOptimizedTestPlan(requirements)
		if err != nil {
			t.Fatalf("Failed to create test plan: %v", err)
		}

		// Execute the test plan
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		result, err := orchestrator.ExecuteComprehensiveTestPlan(ctx, plan)
		if err != nil {
			t.Fatalf("Failed to execute test plan: %v", err)
		}

		// Verify execution result
		if result.ExecutionID == "" {
			t.Error("Expected non-empty execution ID")
		}

		if result.PlanID != plan.ID {
			t.Errorf("Expected plan ID %s, got %s", plan.ID, result.PlanID)
		}

		if result.StartTime.IsZero() || result.EndTime.IsZero() {
			t.Error("Expected non-zero start and end times")
		}

		if result.Duration == 0 {
			t.Error("Expected non-zero execution duration")
		}

		if result.Status == "" {
			t.Error("Expected non-empty status")
		}

		// Verify test results
		if len(result.TestResults) == 0 {
			t.Error("Expected test results to be recorded")
		}

		// Verify quality gate results
		if len(result.QualityGateResults) == 0 {
			t.Error("Expected quality gate results")
		}

		// Verify analysis results
		if result.PerformanceAnalysis.OverallScore == 0 {
			t.Error("Expected performance analysis to be performed")
		}

		if result.SecurityAnalysis.OverallScore == 0 {
			t.Error("Expected security analysis to be performed")
		}

		// Verify recommendations
		if len(result.Recommendations) == 0 {
			t.Log("No recommendations generated (this may be expected for passing tests)")
		}

		t.Logf("Test execution completed: %s (Duration: %v, Status: %s, Success: %t)", 
			result.ExecutionID, result.Duration, result.Status, result.OverallSuccess)

		// Log detailed results
		for suiteName, suiteResult := range result.TestResults {
			t.Logf("Suite %s: %d/%d tests passed, coverage: %.1f%%", 
				suiteName, suiteResult.TestsPassed, suiteResult.TestsRun, suiteResult.Coverage)
		}

		for gateName, gateResult := range result.QualityGateResults {
			t.Logf("Quality Gate %s: %t (%.2f %s %.2f)", 
				gateName, gateResult.Passed, gateResult.ActualValue, 
				"vs", gateResult.Threshold)
		}
	})

	// Test AI test generation
	t.Run("TestAITestGeneration", func(t *testing.T) {
		if !orchestrator.config.EnableAITestGeneration {
			t.Skip("AI test generation disabled")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create a plan that will trigger AI test generation
		plan := &TestExecutionPlan{
			ID:          "ai-test-plan",
			Name:        "AI Test Generation Plan",
			Description: "Plan to test AI test generation",
			CreatedAt:   time.Now(),
			IntegrationTests: []TestSuite{
				{
					Name:        "Sample Integration Tests",
					Type:        "integration",
					Tests:       []string{"TestAPI"},
					Environment: "integration",
					Priority:    PriorityMedium,
				},
			},
			PerformanceTests: []TestSuite{
				{
					Name:        "Sample Performance Tests",
					Type:        "performance",
					Tests:       []string{"TestLoad"},
					Environment: "performance",
					Priority:    PriorityMedium,
				},
			},
		}

		aiTests, err := orchestrator.generateAITests(ctx, plan)
		if err != nil {
			t.Fatalf("Failed to generate AI tests: %v", err)
		}

		if len(aiTests) == 0 {
			t.Error("Expected AI tests to be generated")
		}

		// Verify AI test suites
		for _, suite := range aiTests {
			if suite.Name == "" {
				t.Error("Expected non-empty AI test suite name")
			}

			if len(suite.Tests) == 0 {
				t.Error("Expected AI test suite to contain tests")
			}

			if suite.EstimatedDuration == 0 {
				t.Error("Expected non-zero estimated duration for AI test suite")
			}

			t.Logf("Generated AI test suite: %s with %d tests", suite.Name, len(suite.Tests))
		}
	})

	// Test system health monitoring
	t.Run("TestSystemHealthMonitoring", func(t *testing.T) {
		// Allow some time for health monitoring to run
		time.Sleep(2 * time.Second)

		health := orchestrator.GetSystemHealth()

		// Verify health report structure
		if health.Timestamp.IsZero() {
			t.Error("Expected non-zero health check timestamp")
		}

		if health.OverallStatus == "" {
			t.Error("Expected non-empty overall status")
		}

		if len(health.ComponentHealth) == 0 {
			t.Error("Expected component health information")
		}

		// Verify specific components
		expectedComponents := []string{"environment_manager", "monitoring_integration", "database"}
		for _, component := range expectedComponents {
			if _, exists := health.ComponentHealth[component]; !exists {
				t.Errorf("Expected health information for component: %s", component)
			}
		}

		// Verify metrics
		if health.Metrics.LastUpdated.IsZero() {
			t.Error("Expected non-zero metrics timestamp")
		}

		t.Logf("System health: %s, Components: %d, Uptime: %v", 
			health.OverallStatus, len(health.ComponentHealth), health.SystemMetrics.Uptime)
	})

	// Test metrics collection
	t.Run("TestMetricsCollection", func(t *testing.T) {
		// Allow some time for metrics collection
		time.Sleep(3 * time.Second)

		metrics := orchestrator.collectCurrentMetrics()

		// Verify metrics structure
		if metrics.LastUpdated.IsZero() {
			t.Error("Expected non-zero metrics timestamp")
		}

		// Verify metrics are being tracked
		if metrics.TotalTestsExecuted < 0 {
			t.Error("Expected non-negative total tests executed")
		}

		if metrics.OverallCodeCoverage < 0 || metrics.OverallCodeCoverage > 100 {
			t.Error("Expected code coverage between 0 and 100")
		}

		if metrics.FlakyTestPercentage < 0 || metrics.FlakyTestPercentage > 100 {
			t.Error("Expected flaky test percentage between 0 and 100")
		}

		t.Logf("Metrics - Tests: %d, Coverage: %.1f%%, Flaky: %.1f%%, Active Envs: %d", 
			metrics.TotalTestsExecuted, metrics.OverallCodeCoverage, 
			metrics.FlakyTestPercentage, metrics.ActiveEnvironments)
	})

	// Test error handling and resilience
	t.Run("TestErrorHandlingAndResilience", func(t *testing.T) {
		// Test with invalid plan
		invalidPlan := &TestExecutionPlan{
			ID:          "invalid-plan",
			Name:        "Invalid Plan",
			Description: "Plan with invalid configuration",
			EnvironmentSpecs: []EnvironmentSpec{
				{
					Name: "nonexistent",
					Type: "invalid_type",
					Resources: ResourceAllocation{
						Memory:   -1, // Invalid memory
						CPUQuota: -1, // Invalid CPU
					},
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		result, err := orchestrator.ExecuteComprehensiveTestPlan(ctx, invalidPlan)
		
		// Should handle errors gracefully
		if err == nil {
			t.Error("Expected error for invalid plan")
		}

		if result != nil && result.Status != "failed" {
			t.Error("Expected failed status for invalid plan")
		}

		t.Logf("Error handling test completed - error: %v", err)
	})

	// Test orchestrator shutdown
	t.Run("TestOrchestratorShutdown", func(t *testing.T) {
		err := orchestrator.Stop()
		if err != nil {
			t.Errorf("Failed to stop orchestrator: %v", err)
		}

		// Verify orchestrator is stopped
		if orchestrator.isRunning {
			t.Error("Expected orchestrator to be stopped")
		}

		t.Log("Orchestrator shutdown completed successfully")
	})
}

// TestComprehensiveTestingOrchestratorPerformance tests performance aspects
func TestComprehensiveTestingOrchestratorPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db := createMockDB(t)
	defer db.Close()

	config := DefaultOrchestratorConfig()
	config.MaxConcurrentEnvironments = 3
	config.MaxParallelTests = 8

	orchestrator, err := NewComprehensiveTestingOrchestrator(db, config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = orchestrator.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start orchestrator: %v", err)
	}
	defer orchestrator.Stop()

	// Test high-volume test plan execution
	t.Run("HighVolumeTestExecution", func(t *testing.T) {
		requirements := TestPlanRequirements{
			Name:                    "High Volume Test Plan",
			Description:             "Large scale testing with multiple suites",
			TestTypes:               []string{"unit", "integration", "performance"},
			CoverageTarget:          85.0,
			PerformanceTargets:      map[string]float64{"response_time": 2000.0},
			EnvironmentRequirements: []string{"postgres", "dragonfly"},
			MaxExecutionTime:        30 * time.Minute,
			Priority:                PriorityMedium,
			Tags:                    []string{"high_volume", "performance"},
		}

		start := time.Now()
		plan, err := orchestrator.CreateOptimizedTestPlan(requirements)
		if err != nil {
			t.Fatalf("Failed to create high volume test plan: %v", err)
		}
		planCreationTime := time.Since(start)

		start = time.Now()
		result, err := orchestrator.ExecuteComprehensiveTestPlan(ctx, plan)
		if err != nil {
			t.Fatalf("Failed to execute high volume test plan: %v", err)
		}
		executionTime := time.Since(start)

		// Verify performance metrics
		if planCreationTime > 10*time.Second {
			t.Errorf("Plan creation took too long: %v", planCreationTime)
		}

		if executionTime > 5*time.Minute {
			t.Errorf("Test execution took too long: %v", executionTime)
		}

		if result.Duration > 3*time.Minute {
			t.Errorf("Reported execution duration too long: %v", result.Duration)
		}

		t.Logf("Performance test completed - Plan creation: %v, Execution: %v, Reported: %v", 
			planCreationTime, executionTime, result.Duration)
	})

	// Test concurrent test plan execution
	t.Run("ConcurrentTestExecution", func(t *testing.T) {
		numConcurrent := 3
		results := make(chan *ComprehensiveTestResult, numConcurrent)
		errors := make(chan error, numConcurrent)

		start := time.Now()

		// Start concurrent executions
		for i := 0; i < numConcurrent; i++ {
			go func(index int) {
				requirements := TestPlanRequirements{
					Name:                    fmt.Sprintf("Concurrent Test Plan %d", index),
					Description:             fmt.Sprintf("Concurrent testing plan %d", index),
					TestTypes:               []string{"unit", "integration"},
					CoverageTarget:          80.0,
					EnvironmentRequirements: []string{"postgres"},
					MaxExecutionTime:        10 * time.Minute,
					Priority:                PriorityLow,
					Tags:                    []string{"concurrent"},
				}

				plan, err := orchestrator.CreateOptimizedTestPlan(requirements)
				if err != nil {
					errors <- fmt.Errorf("failed to create plan %d: %w", index, err)
					return
				}

				result, err := orchestrator.ExecuteComprehensiveTestPlan(ctx, plan)
				if err != nil {
					errors <- fmt.Errorf("failed to execute plan %d: %w", index, err)
					return
				}

				results <- result
			}(i)
		}

		// Collect results
		successCount := 0
		for i := 0; i < numConcurrent; i++ {
			select {
			case result := <-results:
				successCount++
				t.Logf("Concurrent execution %d completed: %s", successCount, result.Status)
			case err := <-errors:
				t.Errorf("Concurrent execution failed: %v", err)
			case <-time.After(8 * time.Minute):
				t.Error("Concurrent execution timed out")
				return
			}
		}

		totalTime := time.Since(start)

		if successCount != numConcurrent {
			t.Errorf("Expected %d successful executions, got %d", numConcurrent, successCount)
		}

		if totalTime > 6*time.Minute {
			t.Errorf("Concurrent execution took too long: %v", totalTime)
		}

		t.Logf("Concurrent test completed - %d executions in %v", successCount, totalTime)
	})
}

// createMockDB creates a mock database connection for testing
func createMockDB(t *testing.T) *sql.DB {
	// For testing, we'll use an in-memory SQLite database
	// In a real implementation, this would connect to a test PostgreSQL instance
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	// Create necessary tables for testing
	createTestTables(t, db)

	return db
}

// createTestTables creates the necessary tables for testing
func createTestTables(t *testing.T, db *sql.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS test_executions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			test_name TEXT NOT NULL,
			test_suite TEXT NOT NULL,
			status TEXT NOT NULL,
			duration INTEGER NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME NOT NULL,
			error_message TEXT,
			environment TEXT,
			build_id TEXT,
			commit_hash TEXT,
			branch TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS test_reliability_metrics (
			test_name TEXT NOT NULL,
			test_suite TEXT NOT NULL,
			reliability_score REAL NOT NULL,
			flakiness_score REAL NOT NULL,
			stability_trend TEXT NOT NULL,
			total_executions INTEGER NOT NULL,
			successful_executions INTEGER NOT NULL,
			failed_executions INTEGER NOT NULL,
			error_executions INTEGER NOT NULL,
			skipped_executions INTEGER NOT NULL,
			average_duration INTEGER NOT NULL,
			duration_variance REAL NOT NULL,
			failure_patterns TEXT,
			environment_impact TEXT,
			time_of_day_impact TEXT,
			recent_performance TEXT,
			last_updated DATETIME NOT NULL,
			PRIMARY KEY (test_name, test_suite)
		)`,
		`CREATE TABLE IF NOT EXISTS test_quarantine (
			test_name TEXT NOT NULL,
			test_suite TEXT NOT NULL,
			quarantined_at DATETIME NOT NULL,
			reason TEXT NOT NULL,
			status TEXT NOT NULL,
			reintegrated_at DATETIME,
			PRIMARY KEY (test_name, test_suite)
		)`,
		`CREATE TABLE IF NOT EXISTS comprehensive_test_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			execution_id TEXT NOT NULL UNIQUE,
			plan_id TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME NOT NULL,
			duration INTEGER NOT NULL,
			status TEXT NOT NULL,
			overall_success BOOLEAN NOT NULL,
			quality_gates_passed BOOLEAN NOT NULL,
			result_data TEXT NOT NULL,
			created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS test_reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			report_type TEXT NOT NULL,
			report_data TEXT NOT NULL,
			generated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS test_remediation_suggestions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			test_name TEXT NOT NULL,
			test_suite TEXT NOT NULL,
			suggestion TEXT NOT NULL,
			created_at DATETIME NOT NULL
		)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}
	}
}