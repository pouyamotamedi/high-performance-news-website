package performance

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
	// This would typically use a test database
	// For now, we'll skip if no test DB is available
	db, err := sql.Open("postgres", "postgres://test:test@localhost/test_db?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available")
	}
	
	// Create tables for testing
	createTestTables(t, db)
	
	return db
}

func createTestTables(t *testing.T, db *sql.DB) {
	// Create test tables
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS performance_baselines (
			id BIGSERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			version VARCHAR(100) NOT NULL,
			metrics JSONB NOT NULL,
			environment VARCHAR(50) NOT NULL DEFAULT 'development',
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

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
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)
	require.NoError(t, err)
}

func cleanupTestDB(t *testing.T, db *sql.DB) {
	_, err := db.Exec("DROP TABLE IF EXISTS performance_regression_results")
	require.NoError(t, err)
	_, err = db.Exec("DROP TABLE IF EXISTS performance_baselines")
	require.NoError(t, err)
	db.Close()
}

func TestBaselineManager_StoreBaseline(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	bm := NewBaselineManager(db)

	baseline := &PerformanceBaseline{
		TestName:    "load_test",
		Version:     "v1.0.0",
		Environment: "test",
		Metrics: map[string]MetricData{
			"http_req_duration": {
				Mean:      100.5,
				P95:       200.0,
				P99:       300.0,
				Min:       50.0,
				Max:       500.0,
				Count:     1000,
				StdDev:    25.5,
				Unit:      "ms",
				Threshold: 10.0,
			},
			"article_creation_duration": {
				Mean:      500.0,
				P95:       800.0,
				P99:       1200.0,
				Min:       200.0,
				Max:       2000.0,
				Count:     100,
				StdDev:    150.0,
				Unit:      "ms",
				Threshold: 20.0,
			},
		},
	}

	err := bm.StoreBaseline(baseline)
	assert.NoError(t, err)
	assert.NotZero(t, baseline.ID)
}

func TestBaselineManager_GetActiveBaseline(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	bm := NewBaselineManager(db)

	// Store a baseline first
	baseline := &PerformanceBaseline{
		TestName:    "load_test",
		Version:     "v1.0.0",
		Environment: "test",
		Metrics: map[string]MetricData{
			"http_req_duration": {
				Mean:      100.5,
				P95:       200.0,
				P99:       300.0,
				Min:       50.0,
				Max:       500.0,
				Count:     1000,
				StdDev:    25.5,
				Unit:      "ms",
				Threshold: 10.0,
			},
		},
	}

	err := bm.StoreBaseline(baseline)
	require.NoError(t, err)

	// Retrieve the baseline
	retrieved, err := bm.GetActiveBaseline("load_test", "test")
	assert.NoError(t, err)
	assert.Equal(t, baseline.TestName, retrieved.TestName)
	assert.Equal(t, baseline.Version, retrieved.Version)
	assert.Equal(t, baseline.Environment, retrieved.Environment)
	assert.Len(t, retrieved.Metrics, 1)
	
	metric := retrieved.Metrics["http_req_duration"]
	assert.Equal(t, 100.5, metric.Mean)
	assert.Equal(t, 200.0, metric.P95)
	assert.Equal(t, "ms", metric.Unit)
}

func TestBaselineManager_CompareWithBaseline(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	bm := NewBaselineManager(db)

	// Store a baseline
	baseline := &PerformanceBaseline{
		TestName:    "load_test",
		Version:     "v1.0.0",
		Environment: "test",
		Metrics: map[string]MetricData{
			"http_req_duration": {
				Mean:      100.0,
				P95:       200.0,
				P99:       300.0,
				Min:       50.0,
				Max:       500.0,
				Count:     1000,
				StdDev:    25.0,
				Unit:      "ms",
				Threshold: 10.0,
			},
		},
	}

	err := bm.StoreBaseline(baseline)
	require.NoError(t, err)

	// Test with regression (25% increase)
	currentMetrics := map[string]MetricData{
		"http_req_duration": {
			Mean:   125.0,
			P95:    250.0, // 25% increase from baseline
			P99:    375.0,
			Min:    60.0,
			Max:    600.0,
			Count:  1000,
			StdDev: 30.0,
			Unit:   "ms",
		},
	}

	result, err := bm.CompareWithBaseline("load_test", "v1.1.0", "test", currentMetrics)
	assert.NoError(t, err)
	assert.Equal(t, "load_test", result.TestName)
	assert.Equal(t, "v1.1.0", result.CurrentVersion)
	assert.Equal(t, "v1.0.0", result.BaselineVersion)
	assert.Len(t, result.Regressions, 1)
	assert.Len(t, result.Improvements, 0)
	
	regression := result.Regressions[0]
	assert.Equal(t, "http_req_duration", regression.MetricName)
	assert.Equal(t, 200.0, regression.BaselineValue)
	assert.Equal(t, 250.0, regression.CurrentValue)
	assert.Equal(t, 25.0, regression.PercentChange)
	assert.Equal(t, "medium", regression.Severity) // 25% > 15% threshold
	
	assert.Equal(t, 1, result.Summary.TotalMetrics)
	assert.Equal(t, 1, result.Summary.RegressedMetrics)
	assert.Equal(t, 0, result.Summary.ImprovedMetrics)
	assert.Equal(t, StatusWarning, result.OverallStatus)
}

func TestBaselineManager_CompareWithBaseline_Improvement(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	bm := NewBaselineManager(db)

	// Store a baseline
	baseline := &PerformanceBaseline{
		TestName:    "load_test",
		Version:     "v1.0.0",
		Environment: "test",
		Metrics: map[string]MetricData{
			"http_req_duration": {
				Mean:      100.0,
				P95:       200.0,
				P99:       300.0,
				Min:       50.0,
				Max:       500.0,
				Count:     1000,
				StdDev:    25.0,
				Unit:      "ms",
				Threshold: 10.0,
			},
		},
	}

	err := bm.StoreBaseline(baseline)
	require.NoError(t, err)

	// Test with improvement (10% decrease)
	currentMetrics := map[string]MetricData{
		"http_req_duration": {
			Mean:   90.0,
			P95:    180.0, // 10% decrease from baseline
			P99:    270.0,
			Min:    45.0,
			Max:    450.0,
			Count:  1000,
			StdDev: 22.0,
			Unit:   "ms",
		},
	}

	result, err := bm.CompareWithBaseline("load_test", "v1.1.0", "test", currentMetrics)
	assert.NoError(t, err)
	assert.Len(t, result.Regressions, 0)
	assert.Len(t, result.Improvements, 1)
	
	improvement := result.Improvements[0]
	assert.Equal(t, "http_req_duration", improvement.MetricName)
	assert.Equal(t, 200.0, improvement.BaselineValue)
	assert.Equal(t, 180.0, improvement.CurrentValue)
	assert.Equal(t, -10.0, improvement.PercentChange)
	
	assert.Equal(t, 1, result.Summary.ImprovedMetrics)
	assert.Equal(t, StatusPass, result.OverallStatus)
}

func TestBaselineManager_CalculateSeverity(t *testing.T) {
	bm := &BaselineManager{}

	tests := []struct {
		percentChange float64
		threshold     float64
		expected      string
	}{
		{5.0, 10.0, "low"},      // Below threshold
		{15.0, 10.0, "medium"},  // 1.5x threshold
		{20.0, 10.0, "high"},    // 2x threshold
		{35.0, 10.0, "critical"}, // 3x threshold
	}

	for _, test := range tests {
		result := bm.calculateSeverity(test.percentChange, test.threshold)
		assert.Equal(t, test.expected, result, 
			"percentChange: %f, threshold: %f", test.percentChange, test.threshold)
	}
}

func TestBaselineManager_DetermineOverallStatus(t *testing.T) {
	bm := &BaselineManager{}

	tests := []struct {
		critical int
		regressed int
		total    int
		score    float64
		expected RegressionStatus
	}{
		{1, 1, 10, 50.0, StatusCritical}, // Has critical regressions
		{0, 5, 10, 60.0, StatusFail},     // Score < 70
		{0, 2, 10, 80.0, StatusWarning},  // Has regressions but score >= 70
		{0, 0, 10, 90.0, StatusPass},     // No regressions, good score
	}

	for _, test := range tests {
		result := bm.determineOverallStatus(test.critical, test.regressed, test.total, test.score)
		assert.Equal(t, test.expected, result,
			"critical: %d, regressed: %d, total: %d, score: %f", 
			test.critical, test.regressed, test.total, test.score)
	}
}

func TestBaselineManager_IsResponseTimeMetric(t *testing.T) {
	tests := []struct {
		metricName string
		expected   bool
	}{
		{"http_req_duration", true},
		{"article_creation_duration", true},
		{"database_query_duration", true},
		{"api_response_time", true},
		{"db_connection_time", true},
		{"cache_hit_rate", false},
		{"throughput", false},
		{"articles_per_minute", false},
	}

	for _, test := range tests {
		result := isResponseTimeMetric(test.metricName)
		assert.Equal(t, test.expected, result, "metricName: %s", test.metricName)
	}
}

func TestBaselineManager_UpdateBaseline(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	bm := NewBaselineManager(db)

	// Store initial baseline
	baseline := &PerformanceBaseline{
		TestName:    "load_test",
		Version:     "v1.0.0",
		Environment: "test",
		Metrics: map[string]MetricData{
			"http_req_duration": {
				Mean:      100.0,
				P95:       200.0,
				P99:       300.0,
				Min:       50.0,
				Max:       500.0,
				Count:     1000,
				StdDev:    25.0,
				Unit:      "ms",
				Threshold: 10.0,
			},
		},
	}

	err := bm.StoreBaseline(baseline)
	require.NoError(t, err)

	// Update with new metrics
	newMetrics := map[string]MetricData{
		"http_req_duration": {
			Mean:   110.0,
			P95:    220.0,
			P99:    330.0,
			Min:    55.0,
			Max:    550.0,
			Count:  500,
			StdDev: 30.0,
			Unit:   "ms",
		},
		"new_metric": {
			Mean:   50.0,
			P95:    75.0,
			P99:    100.0,
			Min:    25.0,
			Max:    150.0,
			Count:  100,
			StdDev: 15.0,
			Unit:   "ms",
		},
	}

	err = bm.UpdateBaseline("load_test", "v1.1.0", "test", newMetrics)
	assert.NoError(t, err)

	// Verify the updated baseline
	updated, err := bm.GetActiveBaseline("load_test", "test")
	assert.NoError(t, err)
	assert.Equal(t, "v1.1.0", updated.Version)
	assert.Len(t, updated.Metrics, 2)

	// Check merged metric (70% old + 30% new)
	httpMetric := updated.Metrics["http_req_duration"]
	expectedP95 := 200.0*0.7 + 220.0*0.3 // 140 + 66 = 206
	assert.InDelta(t, expectedP95, httpMetric.P95, 0.1)

	// Check new metric
	newMetric := updated.Metrics["new_metric"]
	assert.Equal(t, 75.0, newMetric.P95)
}

// Benchmark tests
func BenchmarkBaselineManager_StoreBaseline(b *testing.B) {
	db := setupTestDB(&testing.T{})
	defer cleanupTestDB(&testing.T{}, db)

	bm := NewBaselineManager(db)

	baseline := &PerformanceBaseline{
		TestName:    "benchmark_test",
		Version:     "v1.0.0",
		Environment: "test",
		Metrics: map[string]MetricData{
			"http_req_duration": {
				Mean:      100.0,
				P95:       200.0,
				P99:       300.0,
				Min:       50.0,
				Max:       500.0,
				Count:     1000,
				StdDev:    25.0,
				Unit:      "ms",
				Threshold: 10.0,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		baseline.Version = "v1.0." + string(rune(i))
		err := bm.StoreBaseline(baseline)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBaselineManager_CompareWithBaseline(b *testing.B) {
	db := setupTestDB(&testing.T{})
	defer cleanupTestDB(&testing.T{}, db)

	bm := NewBaselineManager(db)

	// Store baseline
	baseline := &PerformanceBaseline{
		TestName:    "benchmark_test",
		Version:     "v1.0.0",
		Environment: "test",
		Metrics: map[string]MetricData{
			"http_req_duration": {
				Mean:      100.0,
				P95:       200.0,
				P99:       300.0,
				Min:       50.0,
				Max:       500.0,
				Count:     1000,
				StdDev:    25.0,
				Unit:      "ms",
				Threshold: 10.0,
			},
		},
	}

	err := bm.StoreBaseline(baseline)
	if err != nil {
		b.Fatal(err)
	}

	currentMetrics := map[string]MetricData{
		"http_req_duration": {
			Mean:   125.0,
			P95:    250.0,
			P99:    375.0,
			Min:    60.0,
			Max:    600.0,
			Count:  1000,
			StdDev: 30.0,
			Unit:   "ms",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.CompareWithBaseline("benchmark_test", "v1.1.0", "test", currentMetrics)
		if err != nil {
			b.Fatal(err)
		}
	}
}