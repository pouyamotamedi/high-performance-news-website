package testing

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestReliabilityTracker(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Close()

	// Create test tables
	setupReliabilityTables(t, testDB.DB)
	defer testDB.Cleanup(t)

	// Create tracker with test configuration
	config := &TestReliabilityConfig{
		FlakinessThreshold:       0.3,
		ReliabilityThreshold:     0.8,
		MinExecutionsForAnalysis: 5,
		AnalysisWindow:          24 * time.Hour,
		QuarantineCooldown:      1 * time.Hour,
		PatternDetectionWindow:  3 * time.Hour,
	}

	tracker := NewTestReliabilityTracker(testDB.DB, config)

	t.Run("TrackTestExecution", func(t *testing.T) {
		execution := &TestExecutionRecord{
			TestName:     "TestExample",
			TestSuite:    "ExampleSuite",
			Status:       "passed",
			Duration:     100 * time.Millisecond,
			StartTime:    time.Now(),
			EndTime:      time.Now().Add(100 * time.Millisecond),
			Environment:  "test",
			BuildID:      "build-123",
			CommitHash:   "abc123",
			Branch:       "main",
		}

		err := tracker.TrackTestExecution(execution)
		assert.NoError(t, err)
	})

	t.Run("GetTestReliabilityMetrics", func(t *testing.T) {
		// Create multiple test executions
		testName := "TestReliabilityMetrics"
		testSuite := "MetricsSuite"

		executions := []*TestExecutionRecord{
			{TestName: testName, TestSuite: testSuite, Status: "passed", Duration: 100 * time.Millisecond, StartTime: time.Now().Add(-10 * time.Minute), EndTime: time.Now().Add(-10*time.Minute + 100*time.Millisecond), Environment: "test"},
			{TestName: testName, TestSuite: testSuite, Status: "passed", Duration: 110 * time.Millisecond, StartTime: time.Now().Add(-9 * time.Minute), EndTime: time.Now().Add(-9*time.Minute + 110*time.Millisecond), Environment: "test"},
			{TestName: testName, TestSuite: testSuite, Status: "failed", Duration: 150 * time.Millisecond, StartTime: time.Now().Add(-8 * time.Minute), EndTime: time.Now().Add(-8*time.Minute + 150*time.Millisecond), Environment: "test", ErrorMessage: "Connection timeout"},
			{TestName: testName, TestSuite: testSuite, Status: "passed", Duration: 105 * time.Millisecond, StartTime: time.Now().Add(-7 * time.Minute), EndTime: time.Now().Add(-7*time.Minute + 105*time.Millisecond), Environment: "test"},
			{TestName: testName, TestSuite: testSuite, Status: "failed", Duration: 200 * time.Millisecond, StartTime: time.Now().Add(-6 * time.Minute), EndTime: time.Now().Add(-6*time.Minute + 200*time.Millisecond), Environment: "prod", ErrorMessage: "Network error"},
			{TestName: testName, TestSuite: testSuite, Status: "passed", Duration: 95 * time.Millisecond, StartTime: time.Now().Add(-5 * time.Minute), EndTime: time.Now().Add(-5*time.Minute + 95*time.Millisecond), Environment: "test"},
		}

		for _, exec := range executions {
			err := tracker.TrackTestExecution(exec)
			require.NoError(t, err)
		}

		// Get metrics
		metrics, err := tracker.GetTestReliabilityMetrics(testName, testSuite)
		require.NoError(t, err)
		assert.NotNil(t, metrics)

		// Verify basic metrics
		assert.Equal(t, testName, metrics.TestName)
		assert.Equal(t, testSuite, metrics.TestSuite)
		assert.Equal(t, int64(6), metrics.TotalExecutions)
		assert.Equal(t, int64(4), metrics.SuccessfulExecutions)
		assert.Equal(t, int64(2), metrics.FailedExecutions)

		// Verify reliability score (4/6 = 0.667)
		assert.InDelta(t, 0.667, metrics.ReliabilityScore, 0.01)

		// Verify flakiness score is calculated
		assert.Greater(t, metrics.FlakinessScore, 0.0)

		// Verify environment impact is analyzed
		assert.Contains(t, metrics.EnvironmentImpact, "test")
		assert.Contains(t, metrics.EnvironmentImpact, "prod")
	})

	t.Run("GetFlakyTests", func(t *testing.T) {
		// Create a flaky test
		flakyTestName := "FlakyTest"
		flakyTestSuite := "FlakySuite"

		// Create alternating pass/fail pattern (high flakiness)
		statuses := []string{"passed", "failed", "passed", "failed", "passed", "failed", "passed"}
		for i, status := range statuses {
			execution := &TestExecutionRecord{
				TestName:    flakyTestName,
				TestSuite:   flakyTestSuite,
				Status:      status,
				Duration:    100 * time.Millisecond,
				StartTime:   time.Now().Add(-time.Duration(len(statuses)-i) * time.Minute),
				EndTime:     time.Now().Add(-time.Duration(len(statuses)-i)*time.Minute + 100*time.Millisecond),
				Environment: "test",
			}
			if status == "failed" {
				execution.ErrorMessage = "Intermittent failure"
			}

			err := tracker.TrackTestExecution(execution)
			require.NoError(t, err)
		}

		// Get flaky tests
		flakyTests, err := tracker.GetFlakyTests()
		require.NoError(t, err)

		// Should find our flaky test
		found := false
		for _, test := range flakyTests {
			if test.TestName == flakyTestName && test.TestSuite == flakyTestSuite {
				found = true
				assert.GreaterOrEqual(t, test.FlakinessScore, config.FlakinessThreshold)
				break
			}
		}
		assert.True(t, found, "Flaky test should be detected")
	})

	t.Run("QuarantineAndReintegrate", func(t *testing.T) {
		testName := "QuarantineTest"
		testSuite := "QuarantineSuite"

		// Quarantine test
		err := tracker.QuarantineTest(testName, testSuite, "High flakiness detected")
		assert.NoError(t, err)

		// Try to reintegrate immediately (should fail due to cooldown)
		err = tracker.ReintegrateTest(testName, testSuite)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cooldown period")

		// Update config to have no cooldown for testing
		tracker.config.QuarantineCooldown = 0

		// Now reintegration should work
		err = tracker.ReintegrateTest(testName, testSuite)
		assert.NoError(t, err)
	})

	t.Run("GenerateStabilityReport", func(t *testing.T) {
		report, err := tracker.GenerateStabilityReport()
		require.NoError(t, err)
		assert.NotNil(t, report)

		// Verify report structure
		assert.NotZero(t, report.GeneratedAt)
		assert.GreaterOrEqual(t, report.TotalTests, 0)
		assert.NotNil(t, report.TestsByReliability)
		assert.NotNil(t, report.EnvironmentAnalysis)
	})
}

func TestFailurePatternAnalyzer(t *testing.T) {
	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Close()

	setupReliabilityTables(t, testDB.DB)
	defer testDB.Cleanup(t)

	analyzer := NewFailurePatternAnalyzer(testDB.DB)

	t.Run("AnalyzeIntermittentPattern", func(t *testing.T) {
		// Create executions with intermittent pattern
		executions := []TestExecutionRecord{
			{Status: "passed", StartTime: time.Now().Add(-10 * time.Minute)},
			{Status: "failed", StartTime: time.Now().Add(-9 * time.Minute), ErrorMessage: "Timeout"},
			{Status: "passed", StartTime: time.Now().Add(-8 * time.Minute)},
			{Status: "failed", StartTime: time.Now().Add(-7 * time.Minute), ErrorMessage: "Connection error"},
			{Status: "passed", StartTime: time.Now().Add(-6 * time.Minute)},
			{Status: "failed", StartTime: time.Now().Add(-5 * time.Minute), ErrorMessage: "Timeout"},
			{Status: "passed", StartTime: time.Now().Add(-4 * time.Minute)},
			{Status: "passed", StartTime: time.Now().Add(-3 * time.Minute)},
			{Status: "failed", StartTime: time.Now().Add(-2 * time.Minute), ErrorMessage: "Network error"},
			{Status: "passed", StartTime: time.Now().Add(-1 * time.Minute)},
		}

		patterns, err := analyzer.AnalyzeFailurePatterns("IntermittentTest", "TestSuite", executions)
		require.NoError(t, err)

		// Should detect intermittent pattern
		found := false
		for _, pattern := range patterns {
			if pattern.Type == "intermittent" {
				found = true
				assert.Greater(t, pattern.Frequency, 0.0)
				assert.Greater(t, pattern.Confidence, 0.0)
				break
			}
		}
		assert.True(t, found, "Should detect intermittent pattern")
	})

	t.Run("AnalyzeEnvironmentPattern", func(t *testing.T) {
		// Create executions with environment-specific failures
		executions := []TestExecutionRecord{
			{Status: "passed", Environment: "test", StartTime: time.Now().Add(-10 * time.Minute)},
			{Status: "passed", Environment: "test", StartTime: time.Now().Add(-9 * time.Minute)},
			{Status: "passed", Environment: "test", StartTime: time.Now().Add(-8 * time.Minute)},
			{Status: "failed", Environment: "prod", StartTime: time.Now().Add(-7 * time.Minute)},
			{Status: "failed", Environment: "prod", StartTime: time.Now().Add(-6 * time.Minute)},
			{Status: "failed", Environment: "prod", StartTime: time.Now().Add(-5 * time.Minute)},
			{Status: "passed", Environment: "test", StartTime: time.Now().Add(-4 * time.Minute)},
		}

		patterns, err := analyzer.AnalyzeFailurePatterns("EnvTest", "TestSuite", executions)
		require.NoError(t, err)

		// Should detect environment pattern
		found := false
		for _, pattern := range patterns {
			if pattern.Type == "environment" {
				found = true
				assert.Contains(t, pattern.Description, "prod")
				break
			}
		}
		assert.True(t, found, "Should detect environment pattern")
	})
}

func TestRemediationEngine(t *testing.T) {
	engine := NewRemediationEngine()

	t.Run("GenerateRemediationSuggestions", func(t *testing.T) {
		// Create mock executions with timeout errors
		executions := []TestExecutionRecord{
			{Status: "failed", ErrorMessage: "Connection timeout after 30 seconds", Duration: 35 * time.Second},
			{Status: "failed", ErrorMessage: "Request timed out", Duration: 32 * time.Second},
			{Status: "passed", Duration: 1 * time.Second},
			{Status: "failed", ErrorMessage: "Timeout waiting for response", Duration: 30 * time.Second},
		}

		// Mock failure patterns
		patterns := []FailurePattern{
			{
				Type:        "error_message",
				Description: "Common error pattern: timeout",
				Frequency:   0.75,
				Confidence:  0.9,
			},
		}

		// Generate suggestions based on patterns
		suggestions := engine.generateSuggestionsForPattern(patterns[0], executions)
		
		assert.NotEmpty(t, suggestions)
		
		// Should suggest timeout-related remediation
		found := false
		for _, suggestion := range suggestions {
			if suggestion.Type == "timing" && 
			   strings.Contains(strings.ToLower(suggestion.Description), "timeout") {
				found = true
				assert.Equal(t, "high", suggestion.Priority)
				assert.Greater(t, suggestion.Confidence, 0.8)
				break
			}
		}
		assert.True(t, found, "Should generate timeout remediation suggestion")
	})
}

func TestTestStabilityOptimizer(t *testing.T) {
	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Close()

	setupReliabilityTables(t, testDB.DB)
	defer testDB.Cleanup(t)

	config := DefaultTestReliabilityConfig()
	tracker := NewTestReliabilityTracker(testDB.DB, config)
	
	optimizerConfig := DefaultStabilityOptimizerConfig()
	optimizerConfig.AutoRemediationEnabled = false // Disable for testing
	optimizer := NewTestStabilityOptimizer(testDB.DB, tracker, optimizerConfig)

	t.Run("OptimizeTestStability", func(t *testing.T) {
		// Create some unstable test data
		testName := "UnstableTest"
		testSuite := "OptimizationSuite"

		// Create flaky test executions
		statuses := []string{"passed", "failed", "passed", "failed", "failed", "passed"}
		for i, status := range statuses {
			execution := &TestExecutionRecord{
				TestName:    testName,
				TestSuite:   testSuite,
				Status:      status,
				Duration:    100 * time.Millisecond,
				StartTime:   time.Now().Add(-time.Duration(len(statuses)-i) * time.Minute),
				EndTime:     time.Now().Add(-time.Duration(len(statuses)-i)*time.Minute + 100*time.Millisecond),
				Environment: "test",
			}
			if status == "failed" {
				execution.ErrorMessage = "Intermittent failure"
			}

			err := tracker.TrackTestExecution(execution)
			require.NoError(t, err)
		}

		// Run optimization
		report, err := optimizer.OptimizeTestStability()
		require.NoError(t, err)
		assert.NotNil(t, report)

		// Verify report structure
		assert.NotZero(t, report.StartTime)
		assert.NotZero(t, report.EndTime)
		assert.Greater(t, report.Duration, time.Duration(0))
		assert.GreaterOrEqual(t, report.UnstableTestsFound, 0)
	})
}

// Helper function to setup test tables
func setupReliabilityTables(t *testing.T, db *sql.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS test_executions (
			id SERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL,
			duration BIGINT NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP NOT NULL,
			error_message TEXT,
			environment VARCHAR(100),
			build_id VARCHAR(100),
			commit_hash VARCHAR(100),
			branch VARCHAR(100)
		)`,
		`CREATE TABLE IF NOT EXISTS test_reliability_metrics (
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			reliability_score FLOAT NOT NULL,
			flakiness_score FLOAT NOT NULL,
			stability_trend VARCHAR(50),
			total_executions BIGINT,
			successful_executions BIGINT,
			failed_executions BIGINT,
			error_executions BIGINT,
			skipped_executions BIGINT,
			average_duration BIGINT,
			duration_variance FLOAT,
			failure_patterns JSONB,
			environment_impact JSONB,
			time_of_day_impact JSONB,
			recent_performance JSONB,
			last_updated TIMESTAMP,
			PRIMARY KEY (test_name, test_suite)
		)`,
		`CREATE TABLE IF NOT EXISTS test_quarantine (
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			quarantined_at TIMESTAMP NOT NULL,
			reintegrated_at TIMESTAMP,
			reason TEXT,
			status VARCHAR(50) DEFAULT 'quarantined',
			PRIMARY KEY (test_name, test_suite)
		)`,
		`CREATE TABLE IF NOT EXISTS test_failure_patterns (
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			pattern_type VARCHAR(100) NOT NULL,
			description TEXT,
			frequency FLOAT,
			first_seen TIMESTAMP,
			last_seen TIMESTAMP,
			confidence FLOAT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (test_name, test_suite, pattern_type)
		)`,
		`CREATE TABLE IF NOT EXISTS test_remediation_suggestions (
			id SERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			suggestion JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS test_remediation_attempts (
			id SERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			remediation_type VARCHAR(100),
			attempted_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS test_environment_adjustments (
			id SERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			adjustment_type VARCHAR(100) NOT NULL,
			adjustment_value VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS environment_resource_limits (
			environment VARCHAR(100) NOT NULL,
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			cpu_limit INTEGER,
			memory_limit INTEGER,
			applied_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (environment, test_name, test_suite)
		)`,
		`CREATE TABLE IF NOT EXISTS environment_network_config (
			environment VARCHAR(100) NOT NULL,
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			config JSONB,
			applied_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (environment, test_name, test_suite)
		)`,
		`CREATE TABLE IF NOT EXISTS environment_storage_config (
			environment VARCHAR(100) NOT NULL,
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			config JSONB,
			applied_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (environment, test_name, test_suite)
		)`,
		`CREATE TABLE IF NOT EXISTS environment_process_config (
			environment VARCHAR(100) NOT NULL,
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			config JSONB,
			applied_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (environment, test_name, test_suite)
		)`,
		`CREATE TABLE IF NOT EXISTS environment_global_config (
			environment VARCHAR(100) NOT NULL,
			config_type VARCHAR(100) NOT NULL,
			config_value TEXT,
			applied_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (environment, config_type)
		)`,
		`CREATE TABLE IF NOT EXISTS environment_optimization_attempts (
			id SERIAL PRIMARY KEY,
			environment VARCHAR(100) NOT NULL,
			test_name VARCHAR(255) NOT NULL,
			test_suite VARCHAR(255) NOT NULL,
			success BOOLEAN,
			attempted_at TIMESTAMP DEFAULT NOW()
		)`,
	}

	for _, table := range tables {
		_, err := db.Exec(table)
		require.NoError(t, err, "Failed to create table")
	}
}