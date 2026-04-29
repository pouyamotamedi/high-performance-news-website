package performance

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

// TestBaselineManagementIntegration tests the complete baseline management workflow
func TestBaselineManagementIntegration(t *testing.T) {
	// Skip if no test database available
	db := setupIntegrationTestDB(t)
	if db == nil {
		t.Skip("Integration test database not available")
		return
	}
	defer cleanupIntegrationTestDB(t, db)

	// Create managers
	enhancedManager := NewEnhancedBaselineManager(db)
	automatedManager := NewAutomatedBaselineManager(db, enhancedManager)
	regressionDetector := NewIntelligentRegressionDetector(db, enhancedManager)

	testName := "integration_test_load"
	version := "v1.0.0"
	environment := "integration_test"

	t.Run("EstablishAutomatedBaseline", func(t *testing.T) {
		result, err := automatedManager.EstablishAutomatedBaseline(testName, version, environment)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify baseline was established
		assert.Equal(t, testName, result.TestName)
		assert.Equal(t, version, result.Version)
		assert.Equal(t, environment, result.Environment)
		assert.Greater(t, result.QualityScore, 0.0)
		assert.Greater(t, result.DataPoints, 0)
		assert.NotEmpty(t, result.ValidationResults)

		// Verify statistical analysis
		assert.Greater(t, result.StatisticalAnalysis.SampleSize, 0)
		assert.GreaterOrEqual(t, result.StatisticalAnalysis.DataCompleteness, 0.0)
		assert.LessOrEqual(t, result.StatisticalAnalysis.DataCompleteness, 1.0)

		// Verify trend analysis
		assert.Contains(t, []string{"increasing", "decreasing", "stable"}, result.TrendAnalysis.TrendDirection)
		assert.GreaterOrEqual(t, result.TrendAnalysis.TrendStrength, 0.0)
		assert.LessOrEqual(t, result.TrendAnalysis.TrendStrength, 1.0)

		// Verify capacity analysis
		assert.Contains(t, []string{"low", "medium", "high"}, result.CapacityAnalysis.BottleneckRisk)
		assert.NotNil(t, result.CapacityAnalysis.CurrentUtilization)

		t.Logf("Baseline established with quality score: %.2f", result.QualityScore)
		t.Logf("Validation results: %d", len(result.ValidationResults))
		t.Logf("Recommendations: %d", len(result.Recommendations))
	})

	t.Run("RetrieveBaseline", func(t *testing.T) {
		// Retrieve the baseline using the base manager
		baseline, err := enhancedManager.GetActiveBaseline(testName, environment)
		require.NoError(t, err)
		require.NotNil(t, baseline)

		assert.Equal(t, testName, baseline.TestName)
		assert.Equal(t, version, baseline.Version)
		assert.Equal(t, environment, baseline.Environment)
		assert.True(t, baseline.IsActive)
		assert.NotEmpty(t, baseline.Metrics)

		t.Logf("Retrieved baseline with %d metrics", len(baseline.Metrics))
	})

	t.Run("PerformRegressionDetection", func(t *testing.T) {
		// Generate current metrics that show some regression
		currentMetrics := map[string]MetricData{
			"http_req_duration": {
				Mean:   130.0, // 30% increase from baseline ~100ms
				P95:    195.0, // 30% increase from baseline ~150ms
				P99:    260.0, // 30% increase from baseline ~200ms
				Min:    60.0,
				Max:    400.0,
				Count:  1000,
				StdDev: 35.0,
				Unit:   "ms",
			},
			"article_creation_duration": {
				Mean:   650.0, // 30% increase from baseline ~500ms
				P95:    1040.0, // 30% increase from baseline ~800ms
				P99:    1560.0, // 30% increase from baseline ~1200ms
				Min:    250.0,
				Max:    2500.0,
				Count:  100,
				StdDev: 200.0,
				Unit:   "ms",
			},
		}

		// Perform regression detection
		request := RegressionAnalysisRequest{
			TestName:       testName,
			CurrentVersion: "v1.1.0",
			Environment:    environment,
			CurrentMetrics: currentMetrics,
			AnalysisOptions: AnalysisOptions{
				UseAdaptiveThresholds: true,
				ConfidenceLevel:       0.95,
				IncludeOptimizations:  true,
				SuppressAlerts:        false,
				DetailedAnalysis:      true,
			},
		}

		result, err := regressionDetector.DetectRegressions(request)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should detect regressions due to 30% increase
		assert.Greater(t, len(result.Regressions), 0)
		assert.Greater(t, result.ConfidenceScore, 0.0)
		assert.LessOrEqual(t, result.ConfidenceScore, 1.0)

		// Verify regression details
		for _, regression := range result.Regressions {
			assert.Greater(t, regression.PercentChange, 0.0) // Should be positive (degradation)
			assert.Contains(t, []string{"low", "medium", "high", "critical"}, regression.Severity)
			t.Logf("Regression detected in %s: %.1f%% increase (%s severity)",
				regression.MetricName, regression.PercentChange, regression.Severity)
		}

		// Should have root cause analysis
		if len(result.RootCauseAnalysis) > 0 {
			for _, hypothesis := range result.RootCauseAnalysis {
				assert.NotEmpty(t, hypothesis.Hypothesis)
				assert.Greater(t, hypothesis.Confidence, 0.0)
				assert.NotEmpty(t, hypothesis.Evidence)
				t.Logf("Root cause hypothesis: %s (confidence: %.2f)", hypothesis.Hypothesis, hypothesis.Confidence)
			}
		}

		// Should have optimization suggestions
		if len(result.OptimizationSuggestions) > 0 {
			for _, suggestion := range result.OptimizationSuggestions {
				assert.NotEmpty(t, suggestion.Category)
				assert.NotEmpty(t, suggestion.Suggestion)
				assert.Contains(t, []string{"low", "medium", "high", "critical"}, suggestion.Priority)
				t.Logf("Optimization suggestion: %s (%s priority)", suggestion.Suggestion, suggestion.Priority)
			}
		}

		// Should make alerting decision
		assert.NotEmpty(t, result.AlertingDecision.AlertLevel)
		assert.NotEmpty(t, result.AlertingDecision.Reason)
		t.Logf("Alerting decision: %s - %s", result.AlertingDecision.AlertLevel, result.AlertingDecision.Reason)
	})

	t.Run("UpdateBaseline", func(t *testing.T) {
		// Update baseline with improved metrics
		improvedMetrics := map[string]MetricData{
			"http_req_duration": {
				Mean:   90.0,  // 10% improvement
				P95:    135.0, // 10% improvement
				P99:    180.0, // 10% improvement
				Min:    45.0,
				Max:    270.0,
				Count:  1000,
				StdDev: 22.0,
				Unit:   "ms",
			},
			"article_creation_duration": {
				Mean:   450.0, // 10% improvement
				P95:    720.0, // 10% improvement
				P99:    1080.0, // 10% improvement
				Min:    180.0,
				Max:    1800.0,
				Count:  100,
				StdDev: 135.0,
				Unit:   "ms",
			},
		}

		// Use the base manager to update baseline
		baseManager := NewBaselineManager(db)
		err := baseManager.UpdateBaseline(testName, "v1.2.0", environment, improvedMetrics)
		require.NoError(t, err)

		// Verify updated baseline
		updatedBaseline, err := baseManager.GetActiveBaseline(testName, environment)
		require.NoError(t, err)
		assert.Equal(t, "v1.2.0", updatedBaseline.Version)

		t.Logf("Baseline updated to version %s", updatedBaseline.Version)
	})

	t.Run("ScheduledUpdates", func(t *testing.T) {
		// Test scheduling functionality
		schedulingEngine := automatedManager.schedulingEngine

		// Schedule an update
		nextUpdate := time.Now().Add(1 * time.Hour)
		err := schedulingEngine.ScheduleUpdate(testName, environment, nextUpdate)
		require.NoError(t, err)

		// Verify schedule was created
		schedules, err := schedulingEngine.GetDueUpdates()
		require.NoError(t, err)

		// Should not be due yet (scheduled for 1 hour from now)
		found := false
		for _, schedule := range schedules {
			if schedule.TestName == testName && schedule.Environment == environment {
				found = true
				break
			}
		}
		assert.False(t, found, "Schedule should not be due yet")

		// Schedule an immediate update
		immediateUpdate := time.Now().Add(-1 * time.Minute) // 1 minute ago
		err = schedulingEngine.ScheduleUpdate(testName+"_immediate", environment, immediateUpdate)
		require.NoError(t, err)

		// Should be due now
		schedules, err = schedulingEngine.GetDueUpdates()
		require.NoError(t, err)

		found = false
		for _, schedule := range schedules {
			if schedule.TestName == testName+"_immediate" && schedule.Environment == environment {
				found = true
				assert.Equal(t, "active", schedule.Status)
				t.Logf("Found due schedule: %s in %s", schedule.TestName, schedule.Environment)
				break
			}
		}
		assert.True(t, found, "Immediate schedule should be due")
	})

	t.Run("ValidationEngine", func(t *testing.T) {
		validationEngine := automatedManager.validationEngine

		// Create a baseline with known characteristics for validation
		testBaseline := &EnhancedPerformanceBaseline{
			StatisticalData: StatisticalMetrics{
				Outliers: make([]OutlierPoint, 40), // Good sample size
			},
		}

		results, err := validationEngine.ValidateBaseline(testBaseline)
		require.NoError(t, err)
		assert.Greater(t, len(results), 0)

		// Check validation results
		for _, result := range results {
			assert.NotEmpty(t, result.RuleName)
			assert.NotEmpty(t, result.Severity)
			assert.NotEmpty(t, result.Message)
			assert.Contains(t, []ValidationStatus{ValidationStatusPass, ValidationStatusWarning, ValidationStatusFail}, result.Status)
			t.Logf("Validation %s: %s (%s)", result.RuleName, result.Message, result.Status)
		}
	})
}

// TestRegressionDetectionAccuracy tests the accuracy of regression detection
func TestRegressionDetectionAccuracy(t *testing.T) {
	db := setupIntegrationTestDB(t)
	if db == nil {
		t.Skip("Integration test database not available")
		return
	}
	defer cleanupIntegrationTestDB(t, db)

	enhancedManager := NewEnhancedBaselineManager(db)
	automatedManager := NewAutomatedBaselineManager(db, enhancedManager)
	regressionDetector := NewIntelligentRegressionDetector(db, enhancedManager)

	testName := "accuracy_test"
	version := "v1.0.0"
	environment := "test"

	// Establish baseline
	_, err := automatedManager.EstablishAutomatedBaseline(testName, version, environment)
	require.NoError(t, err)

	testCases := []struct {
		name           string
		multiplier     float64
		expectRegression bool
		expectedSeverity string
	}{
		{"No Change", 1.0, false, ""},
		{"Minor Improvement", 0.95, false, ""},
		{"Minor Degradation", 1.05, false, ""}, // Below threshold
		{"Moderate Degradation", 1.20, true, "medium"}, // Above 15% threshold
		{"Significant Degradation", 1.50, true, "high"}, // Above 30% threshold
		{"Critical Degradation", 2.0, true, "critical"}, // 100% increase
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate metrics with the specified multiplier
			currentMetrics := map[string]MetricData{
				"http_req_duration": {
					Mean:   100.0 * tc.multiplier,
					P95:    150.0 * tc.multiplier,
					P99:    200.0 * tc.multiplier,
					Min:    50.0,
					Max:    300.0 * tc.multiplier,
					Count:  1000,
					StdDev: 25.0 * tc.multiplier,
					Unit:   "ms",
				},
			}

			request := RegressionAnalysisRequest{
				TestName:       testName,
				CurrentVersion: "v1.1.0",
				Environment:    environment,
				CurrentMetrics: currentMetrics,
				AnalysisOptions: AnalysisOptions{
					UseAdaptiveThresholds: false, // Use fixed thresholds for accuracy testing
					ConfidenceLevel:       0.95,
					IncludeOptimizations:  false,
					SuppressAlerts:        true,
					DetailedAnalysis:      false,
				},
			}

			result, err := regressionDetector.DetectRegressions(request)
			require.NoError(t, err)

			if tc.expectRegression {
				assert.Greater(t, len(result.Regressions), 0, "Should detect regression for %s", tc.name)
				if len(result.Regressions) > 0 {
					regression := result.Regressions[0]
					if tc.expectedSeverity != "" {
						assert.Equal(t, tc.expectedSeverity, regression.Severity, "Severity mismatch for %s", tc.name)
					}
					t.Logf("%s: Detected %.1f%% change (%s severity)", tc.name, regression.PercentChange, regression.Severity)
				}
			} else {
				assert.Equal(t, 0, len(result.Regressions), "Should not detect regression for %s", tc.name)
				t.Logf("%s: No regression detected (as expected)", tc.name)
			}
		})
	}
}

// TestPerformanceBaselineLifecycle tests the complete lifecycle of a performance baseline
func TestPerformanceBaselineLifecycle(t *testing.T) {
	db := setupIntegrationTestDB(t)
	if db == nil {
		t.Skip("Integration test database not available")
		return
	}
	defer cleanupIntegrationTestDB(t, db)

	enhancedManager := NewEnhancedBaselineManager(db)
	automatedManager := NewAutomatedBaselineManager(db, enhancedManager)

	testName := "lifecycle_test"
	environment := "test"

	// Phase 1: Initial baseline establishment
	t.Run("Phase1_InitialBaseline", func(t *testing.T) {
		result, err := automatedManager.EstablishAutomatedBaseline(testName, "v1.0.0", environment)
		require.NoError(t, err)
		assert.Greater(t, result.QualityScore, 70.0) // Should have good quality
		t.Logf("Phase 1: Initial baseline established with quality %.2f", result.QualityScore)
	})

	// Phase 2: Performance degradation
	t.Run("Phase2_PerformanceDegradation", func(t *testing.T) {
		// Simulate performance degradation by updating with worse metrics
		degradedMetrics := map[string]MetricData{
			"http_req_duration": {
				Mean:   150.0, // 50% worse
				P95:    225.0, // 50% worse
				P99:    300.0, // 50% worse
				Min:    75.0,
				Max:    450.0,
				Count:  1000,
				StdDev: 40.0,
				Unit:   "ms",
			},
		}

		baseManager := NewBaselineManager(db)
		err := baseManager.UpdateBaseline(testName, "v1.1.0", environment, degradedMetrics)
		require.NoError(t, err)
		t.Logf("Phase 2: Performance degradation simulated")
	})

	// Phase 3: Performance recovery
	t.Run("Phase3_PerformanceRecovery", func(t *testing.T) {
		// Simulate performance recovery
		recoveredMetrics := map[string]MetricData{
			"http_req_duration": {
				Mean:   95.0,  // Better than original
				P95:    140.0, // Better than original
				P99:    190.0, // Better than original
				Min:    45.0,
				Max:    285.0,
				Count:  1000,
				StdDev: 20.0,
				Unit:   "ms",
			},
		}

		baseManager := NewBaselineManager(db)
		err := baseManager.UpdateBaseline(testName, "v1.2.0", environment, recoveredMetrics)
		require.NoError(t, err)
		t.Logf("Phase 3: Performance recovery simulated")
	})

	// Phase 4: Final validation
	t.Run("Phase4_FinalValidation", func(t *testing.T) {
		finalBaseline, err := enhancedManager.GetActiveBaseline(testName, environment)
		require.NoError(t, err)
		assert.Equal(t, "v1.2.0", finalBaseline.Version)

		// Verify the final baseline reflects the recovered performance
		httpMetric := finalBaseline.Metrics["http_req_duration"]
		assert.Less(t, httpMetric.P95, 150.0) // Should be better than degraded state
		t.Logf("Phase 4: Final baseline P95: %.2fms", httpMetric.P95)
	})
}

// Helper functions for integration tests

func setupIntegrationTestDB(t *testing.T) *sql.DB {
	// Try to connect to test database
	dbURL := "postgres://test:test@localhost/test_db?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Logf("Cannot connect to test database: %v", err)
		return nil
	}

	if err := db.Ping(); err != nil {
		t.Logf("Cannot ping test database: %v", err)
		db.Close()
		return nil
	}

	// Create required tables
	createIntegrationTestTables(t, db)
	return db
}

func createIntegrationTestTables(t *testing.T, db *sql.DB) {
	// Create performance_baselines table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS performance_baselines (
			id BIGSERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			version VARCHAR(100) NOT NULL,
			metrics JSONB NOT NULL,
			environment VARCHAR(50) NOT NULL DEFAULT 'development',
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			enhanced_data JSONB
		)`)
	require.NoError(t, err)

	// Create performance_regression_results table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS performance_regression_results (
			id BIGSERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			current_version VARCHAR(100) NOT NULL,
			baseline_version VARCHAR(100) NOT NULL,
			result_data JSONB NOT NULL,
			overall_status VARCHAR(20) NOT NULL,
			overall_score DECIMAL(5,2) NOT NULL DEFAULT 0.00,
			compared_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			enhanced_data JSONB
		)`)
	require.NoError(t, err)

	// Create automated baseline tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS automated_baseline_results (
			id BIGSERIAL PRIMARY KEY,
			baseline_id BIGINT NOT NULL,
			test_name VARCHAR(255) NOT NULL,
			environment VARCHAR(100) NOT NULL,
			version VARCHAR(100) NOT NULL,
			established_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			data_points INTEGER NOT NULL,
			quality_score DECIMAL(5,2) NOT NULL,
			next_update_scheduled TIMESTAMP WITH TIME ZONE NOT NULL,
			result_data JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)`)
	require.NoError(t, err)

	// Create baseline_schedules table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS baseline_schedules (
			id BIGSERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			environment VARCHAR(100) NOT NULL,
			update_frequency INTERVAL NOT NULL DEFAULT '24 hours',
			next_update TIMESTAMP WITH TIME ZONE NOT NULL,
			last_update TIMESTAMP WITH TIME ZONE,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			required_data_points INTEGER NOT NULL DEFAULT 50,
			collected_metrics JSONB,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			UNIQUE(test_name, environment)
		)`)
	require.NoError(t, err)

	// Create baseline_validation_rules table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS baseline_validation_rules (
			id BIGSERIAL PRIMARY KEY,
			rule_name VARCHAR(255) NOT NULL UNIQUE,
			condition_type VARCHAR(100) NOT NULL,
			threshold_value DECIMAL(10,4) NOT NULL,
			severity VARCHAR(50) NOT NULL,
			action_type VARCHAR(100) NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT true,
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)`)
	require.NoError(t, err)
}

func cleanupIntegrationTestDB(t *testing.T, db *sql.DB) {
	// Clean up test data
	tables := []string{
		"automated_baseline_results",
		"baseline_schedules", 
		"baseline_validation_rules",
		"performance_regression_results",
		"performance_baselines",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: failed to drop table %s: %v", table, err)
		}
	}

	db.Close()
}