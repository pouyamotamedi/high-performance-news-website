package performance

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAutomatedBaselineManager(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	enhancedManager := NewEnhancedBaselineManager(db)
	abm := NewAutomatedBaselineManager(db, enhancedManager)

	assert.NotNil(t, abm)
	assert.NotNil(t, abm.enhancedManager)
	assert.NotNil(t, abm.regressionDetector)
	assert.NotNil(t, abm.validationEngine)
	assert.NotNil(t, abm.schedulingEngine)
}

func TestGenerateSampleMetrics(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	enhancedManager := NewEnhancedBaselineManager(db)
	abm := NewAutomatedBaselineManager(db, enhancedManager)

	metrics, err := abm.generateSampleMetrics("test_load", "development")
	assert.NoError(t, err)
	assert.Len(t, metrics, 50) // Should generate 50 sample points

	// Check that all expected metrics are present
	for _, metricSet := range metrics {
		assert.Contains(t, metricSet, "http_req_duration")
		assert.Contains(t, metricSet, "article_creation_duration")
		assert.Contains(t, metricSet, "database_query_duration")
		assert.Contains(t, metricSet, "cache_hit_rate")

		// Verify metric structure
		httpMetric := metricSet["http_req_duration"]
		assert.Greater(t, httpMetric.Mean, 0.0)
		assert.Greater(t, httpMetric.P95, httpMetric.Mean)
		assert.Greater(t, httpMetric.P99, httpMetric.P95)
		assert.Equal(t, "ms", httpMetric.Unit)
	}
}

func TestCalculateQualityScore(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	enhancedManager := NewEnhancedBaselineManager(db)
	abm := NewAutomatedBaselineManager(db, enhancedManager)

	// Test with all passing validations
	passingResults := []ValidationResult{
		{Status: ValidationStatusPass, Severity: "critical"},
		{Status: ValidationStatusPass, Severity: "high"},
		{Status: ValidationStatusPass, Severity: "medium"},
	}

	score := abm.calculateQualityScore(passingResults, nil)
	assert.Equal(t, 100.0, score)

	// Test with mixed results
	mixedResults := []ValidationResult{
		{Status: ValidationStatusPass, Severity: "critical"},
		{Status: ValidationStatusWarning, Severity: "high"},
		{Status: ValidationStatusFail, Severity: "medium"},
	}

	score = abm.calculateQualityScore(mixedResults, nil)
	assert.Greater(t, score, 0.0)
	assert.Less(t, score, 100.0)

	// Test with no validations
	score = abm.calculateQualityScore([]ValidationResult{}, nil)
	assert.Equal(t, 50.0, score)
}

func TestGenerateRecommendations(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	enhancedManager := NewEnhancedBaselineManager(db)
	abm := NewAutomatedBaselineManager(db, enhancedManager)

	// Create test baseline with high resource utilization
	baseline := &EnhancedPerformanceBaseline{
		CapacityData: CapacityMetrics{
			ResourceUtilization: map[string]float64{
				"cpu":    0.85, // High CPU utilization
				"memory": 0.75, // Moderate memory utilization
			},
		},
		TrendData: TrendMetrics{
			TrendDirection: "increasing",
			TrendStrength:  0.8, // Strong degradation trend
		},
	}

	// Test with failed validation
	validationResults := []ValidationResult{
		{
			Status:   ValidationStatusFail,
			RuleName: "minimum_sample_size",
			Severity: "critical",
			Message:  "Insufficient sample size",
		},
	}

	recommendations := abm.generateRecommendations(baseline, validationResults)

	// Should have recommendations for validation failure, capacity, and trend
	assert.GreaterOrEqual(t, len(recommendations), 3)

	// Check for validation failure recommendation
	hasValidationRec := false
	for _, rec := range recommendations {
		if rec.Type == "validation_failure" {
			hasValidationRec = true
			assert.Equal(t, "critical", rec.Priority)
			break
		}
	}
	assert.True(t, hasValidationRec)

	// Check for capacity recommendation
	hasCapacityRec := false
	for _, rec := range recommendations {
		if rec.Type == "capacity_planning" {
			hasCapacityRec = true
			break
		}
	}
	assert.True(t, hasCapacityRec)

	// Check for trend recommendation
	hasTrendRec := false
	for _, rec := range recommendations {
		if rec.Type == "performance_trend" {
			hasTrendRec = true
			assert.Equal(t, "high", rec.Priority)
			break
		}
	}
	assert.True(t, hasTrendRec)
}

func TestValidationEngine(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock the validation rules query to return empty result (will use defaults)
	mock, ok := db.(*sqlmock.Sqlmock)
	require.True(t, ok)

	mock.ExpectQuery("SELECT rule_name, condition_type, threshold_value, severity, action_type, description FROM baseline_validation_rules").
		WillReturnError(sqlmock.ErrCancelled) // Force fallback to defaults

	ve := NewValidationEngine(db)

	// Should have default rules loaded
	assert.Greater(t, len(ve.rules), 0)
	assert.Contains(t, ve.rules, "minimum_sample_size")
	assert.Contains(t, ve.rules, "data_completeness")
	assert.Contains(t, ve.rules, "variance_threshold")

	// Test validation
	baseline := &EnhancedPerformanceBaseline{
		StatisticalData: StatisticalMetrics{
			Outliers: make([]OutlierPoint, 35), // Sufficient sample size
		},
	}

	results, err := ve.ValidateBaseline(baseline)
	assert.NoError(t, err)
	assert.Equal(t, len(ve.rules), len(results))

	// Check that minimum sample size passes
	for _, result := range results {
		if result.RuleName == "minimum_sample_size" {
			assert.Equal(t, ValidationStatusPass, result.Status)
			break
		}
	}
}

func TestValidateRule(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ve := NewValidationEngine(db)

	// Test minimum sample size rule
	rule := ValidationRule{
		Name:      "minimum_sample_size",
		Threshold: 30,
		Severity:  "critical",
	}

	baseline := &EnhancedPerformanceBaseline{
		StatisticalData: StatisticalMetrics{
			Outliers: make([]OutlierPoint, 35), // 35 + 30 = 65 > 30 threshold
		},
	}

	result := ve.validateRule(rule, baseline)
	assert.Equal(t, "minimum_sample_size", result.RuleName)
	assert.Equal(t, ValidationStatusPass, result.Status)
	assert.Equal(t, "critical", result.Severity)
	assert.Greater(t, result.Value, rule.Threshold)

	// Test with insufficient sample size
	baseline.StatisticalData.Outliers = make([]OutlierPoint, 5) // 5 + 30 = 35 > 30, still passes
	result = ve.validateRule(rule, baseline)
	assert.Equal(t, ValidationStatusPass, result.Status)

	// Test data completeness rule
	rule = ValidationRule{
		Name:      "data_completeness",
		Threshold: 0.90,
		Severity:  "high",
	}

	result = ve.validateRule(rule, baseline)
	assert.Equal(t, "data_completeness", result.RuleName)
	assert.Equal(t, ValidationStatusPass, result.Status) // Should pass with 0.95 completeness
	assert.Equal(t, 0.95, result.Value)

	// Test variance threshold rule
	rule = ValidationRule{
		Name:      "variance_threshold",
		Threshold: 0.30,
		Severity:  "medium",
	}

	result = ve.validateRule(rule, baseline)
	assert.Equal(t, "variance_threshold", result.RuleName)
	assert.Equal(t, ValidationStatusPass, result.Status) // Should pass with 0.25 variance
	assert.Equal(t, 0.25, result.Value)
}

func TestSchedulingEngine(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	se := NewSchedulingEngine(db)

	// Test ScheduleUpdate
	testName := "load_test"
	environment := "staging"
	nextUpdate := time.Now().Add(24 * time.Hour)

	mock.ExpectExec("INSERT INTO baseline_schedules").
		WithArgs(testName, environment, nextUpdate).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = se.ScheduleUpdate(testName, environment, nextUpdate)
	assert.NoError(t, err)

	// Test GetDueUpdates
	rows := sqlmock.NewRows([]string{"id", "test_name", "environment", "update_frequency", "next_update", "last_update", "status", "required_data_points"}).
		AddRow(1, "load_test", "staging", "24h", time.Now().Add(-1*time.Hour), nil, "active", 50)

	mock.ExpectQuery("SELECT id, test_name, environment, update_frequency, next_update, last_update, status, required_data_points FROM baseline_schedules").
		WillReturnRows(rows)

	schedules, err := se.GetDueUpdates()
	assert.NoError(t, err)
	assert.Len(t, schedules, 1)

	schedule := schedules[0]
	assert.Equal(t, int64(1), schedule.ID)
	assert.Equal(t, "load_test", schedule.TestName)
	assert.Equal(t, "staging", schedule.Environment)
	assert.Equal(t, 24*time.Hour, schedule.UpdateFrequency)
	assert.Equal(t, "active", schedule.Status)
	assert.Equal(t, 50, schedule.RequiredDataPoints)

	// Test UpdateSchedule
	newNextUpdate := time.Now().Add(48 * time.Hour)
	mock.ExpectExec("UPDATE baseline_schedules SET next_update = \\$2, last_update = NOW\\(\\) WHERE id = \\$1").
		WithArgs(int64(1), newNextUpdate).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = se.UpdateSchedule(1, newNextUpdate)
	assert.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExtractAnalysis(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	enhancedManager := NewEnhancedBaselineManager(db)
	abm := NewAutomatedBaselineManager(db, enhancedManager)

	baseline := &EnhancedPerformanceBaseline{
		StatisticalData: StatisticalMetrics{
			Outliers: []OutlierPoint{{}, {}}, // 2 outliers
			Seasonality: map[string]SeasonalPattern{
				"daily": {Period: "daily"},
			},
		},
		TrendData: TrendMetrics{
			TrendDirection: "stable",
			TrendStrength:  0.3,
			ChangePoints:   []ChangePoint{{}, {}, {}}, // 3 change points
		},
		CapacityData: CapacityMetrics{
			ResourceUtilization: map[string]float64{
				"cpu":    0.75,
				"memory": 0.60,
			},
		},
	}

	// Test statistical analysis extraction
	statAnalysis := abm.extractStatisticalAnalysis(baseline)
	assert.Equal(t, 2, statAnalysis.OutliersDetected)
	assert.Equal(t, 3, statAnalysis.ChangePointsDetected)
	assert.True(t, statAnalysis.SeasonalityDetected)
	assert.Equal(t, "good", statAnalysis.DataQuality)
	assert.Equal(t, 0.95, statAnalysis.ConfidenceLevel)

	// Test trend analysis extraction
	trendAnalysis := abm.extractTrendAnalysis(baseline)
	assert.Equal(t, "stable", trendAnalysis.TrendDirection)
	assert.Equal(t, 0.3, trendAnalysis.TrendStrength)
	assert.Equal(t, 0.85, trendAnalysis.ForecastAccuracy)
	assert.Equal(t, 0.3, trendAnalysis.VolatilityIndex)

	// Test capacity analysis extraction
	capacityAnalysis := abm.extractCapacityAnalysis(baseline)
	assert.Equal(t, "low", capacityAnalysis.BottleneckRisk)
	assert.False(t, capacityAnalysis.ScalingNeeded)
	assert.Equal(t, 0.75, capacityAnalysis.CurrentUtilization["cpu"])
	assert.Equal(t, 0.60, capacityAnalysis.CurrentUtilization["memory"])

	// Test with high utilization
	baseline.CapacityData.ResourceUtilization["cpu"] = 0.95
	capacityAnalysis = abm.extractCapacityAnalysis(baseline)
	assert.Equal(t, "high", capacityAnalysis.BottleneckRisk)
	assert.True(t, capacityAnalysis.ScalingNeeded)
}

func TestGetValidationFailureAction(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	enhancedManager := NewEnhancedBaselineManager(db)
	abm := NewAutomatedBaselineManager(db, enhancedManager)

	// Test known rules
	action := abm.getValidationFailureAction("minimum_sample_size")
	assert.Equal(t, "Collect more performance data points", action)

	action = abm.getValidationFailureAction("data_completeness")
	assert.Equal(t, "Ensure all required metrics are collected", action)

	action = abm.getValidationFailureAction("variance_threshold")
	assert.Equal(t, "Investigate source of high variance", action)

	// Test unknown rule
	action = abm.getValidationFailureAction("unknown_rule")
	assert.Equal(t, "Review validation rule and data collection process", action)
}

// Integration test for the full automated baseline establishment process
func TestEstablishAutomatedBaselineIntegration(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock database operations for enhanced baseline storage
	mock.ExpectExec("UPDATE performance_baselines SET is_active = false").
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery("INSERT INTO performance_baselines").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	// Mock validation rules query
	mock.ExpectQuery("SELECT rule_name, condition_type, threshold_value, severity, action_type, description FROM baseline_validation_rules").
		WillReturnError(sqlmock.ErrCancelled) // Use defaults

	// Mock schedule insertion
	mock.ExpectExec("INSERT INTO baseline_schedules").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock automated result storage
	mock.ExpectExec("INSERT INTO automated_baseline_results").
		WillReturnResult(sqlmock.NewResult(1, 1))

	enhancedManager := NewEnhancedBaselineManager(db)
	abm := NewAutomatedBaselineManager(db, enhancedManager)

	result, err := abm.EstablishAutomatedBaseline("integration_test", "v1.0.0", "test")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify result structure
	assert.Equal(t, "integration_test", result.TestName)
	assert.Equal(t, "v1.0.0", result.Version)
	assert.Equal(t, "test", result.Environment)
	assert.Greater(t, result.QualityScore, 0.0)
	assert.Equal(t, 50, result.DataPoints) // Should match sample count
	assert.NotEmpty(t, result.ValidationResults)
	assert.NotEmpty(t, result.Recommendations)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}