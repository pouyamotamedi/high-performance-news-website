package performance

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnhancedBaselineManager(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	assert.NotNil(t, ebm)
	assert.NotNil(t, ebm.statisticalEngine)
	assert.NotNil(t, ebm.trendAnalyzer)
	assert.NotNil(t, ebm.capacityPlanner)
	assert.NotNil(t, ebm.regressionEngine)

	// Test default configuration
	assert.Equal(t, 0.95, ebm.statisticalEngine.confidenceLevel)
	assert.Equal(t, 30, ebm.statisticalEngine.sampleSize)
	assert.Equal(t, 7, ebm.trendAnalyzer.forecastPeriods)
	assert.True(t, ebm.trendAnalyzer.seasonality)
	assert.Equal(t, 0.3, ebm.trendAnalyzer.smoothingFactor)
}

func TestCalculateMean(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	// Test with normal values
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	mean := ebm.calculateMean(values)
	assert.Equal(t, 3.0, mean)

	// Test with empty slice
	emptyValues := []float64{}
	mean = ebm.calculateMean(emptyValues)
	assert.Equal(t, 0.0, mean)

	// Test with single value
	singleValue := []float64{42.0}
	mean = ebm.calculateMean(singleValue)
	assert.Equal(t, 42.0, mean)
}

func TestCalculatePercentile(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	// Test with sorted values
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	
	p50 := ebm.calculatePercentile(values, 50.0)
	assert.InDelta(t, 5.5, p50, 0.1)

	p95 := ebm.calculatePercentile(values, 95.0)
	assert.InDelta(t, 9.55, p95, 0.1)

	p99 := ebm.calculatePercentile(values, 99.0)
	assert.InDelta(t, 9.91, p99, 0.1)

	// Test with empty slice
	emptyValues := []float64{}
	p50 = ebm.calculatePercentile(emptyValues, 50.0)
	assert.Equal(t, 0.0, p50)
}

func TestCalculateStdDev(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	// Test with known values
	values := []float64{2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0}
	stdDev := ebm.calculateStdDev(values)
	assert.InDelta(t, 2.0, stdDev, 0.1) // Expected standard deviation is approximately 2.0

	// Test with single value
	singleValue := []float64{5.0}
	stdDev = ebm.calculateStdDev(singleValue)
	assert.Equal(t, 0.0, stdDev)

	// Test with empty slice
	emptyValues := []float64{}
	stdDev = ebm.calculateStdDev(emptyValues)
	assert.Equal(t, 0.0, stdDev)
}

func TestCalculateConfidenceInterval(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	// Test with normal distribution
	values := []float64{98.0, 99.0, 100.0, 101.0, 102.0}
	ci := ebm.calculateConfidenceInterval(values, 0.95)

	assert.Equal(t, 0.95, ci.Confidence)
	assert.Less(t, ci.Lower, 100.0) // Should be below the mean
	assert.Greater(t, ci.Upper, 100.0) // Should be above the mean
	assert.Less(t, ci.Lower, ci.Upper) // Lower should be less than upper

	// Test with insufficient data
	insufficientData := []float64{100.0}
	ci = ebm.calculateConfidenceInterval(insufficientData, 0.95)
	assert.Equal(t, 0.0, ci.Lower)
	assert.Equal(t, 0.0, ci.Upper)
}

func TestAggregateMetrics(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	// Create test raw metrics
	rawMetrics := []map[string]MetricData{
		{
			"http_req_duration": {
				P95:    100.0,
				Unit:   "ms",
				Threshold: 10.0,
			},
		},
		{
			"http_req_duration": {
				P95:    110.0,
				Unit:   "ms", 
				Threshold: 10.0,
			},
		},
		{
			"http_req_duration": {
				P95:    90.0,
				Unit:   "ms",
				Threshold: 10.0,
			},
		},
	}

	aggregated := ebm.aggregateMetrics(rawMetrics)

	assert.Contains(t, aggregated, "http_req_duration")
	
	metric := aggregated["http_req_duration"]
	assert.Equal(t, "ms", metric.Unit)
	assert.Equal(t, 10.0, metric.Threshold)
	assert.Equal(t, int64(3), metric.Count)
	assert.InDelta(t, 100.0, metric.Mean, 10.0) // Should be around 100
	assert.Equal(t, 90.0, metric.Min)
	assert.Equal(t, 110.0, metric.Max)
}

func TestDetectOutliers(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	// Create test data with known outliers
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 100.0} // 100.0 is an outlier
	timestamps := make([]time.Time, len(values))
	for i := range timestamps {
		timestamps[i] = time.Now().Add(time.Duration(i) * time.Minute)
	}

	outliers := ebm.detectOutliers(values, timestamps, "test_metric")

	assert.NotEmpty(t, outliers)
	assert.Equal(t, "test_metric", outliers[0].Metric)
	assert.Greater(t, outliers[0].ZScore, 2.0) // Should have high z-score
}

func TestCalculateMode(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	// Test with odd number of values
	values := []float64{1.0, 3.0, 5.0, 7.0, 9.0}
	mode := ebm.calculateMode(values)
	assert.Equal(t, 5.0, mode) // Should return median for continuous data

	// Test with even number of values
	evenValues := []float64{2.0, 4.0, 6.0, 8.0}
	mode = ebm.calculateMode(evenValues)
	assert.Equal(t, 5.0, mode) // Should return average of middle two values
}

func TestCalculateAutoCorrelation(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	// Test with perfectly correlated data (increasing trend)
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	correlation := ebm.calculateAutoCorrelation(values)
	assert.Greater(t, correlation, 0.5) // Should show positive correlation

	// Test with anti-correlated data
	antiValues := []float64{5.0, 4.0, 3.0, 2.0, 1.0}
	correlation = ebm.calculateAutoCorrelation(antiValues)
	assert.Less(t, correlation, -0.5) // Should show negative correlation

	// Test with insufficient data
	shortValues := []float64{1.0}
	correlation = ebm.calculateAutoCorrelation(shortValues)
	assert.Equal(t, 0.0, correlation)
}

func TestCalculateScalingTimeline(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	assert.Equal(t, "immediate", ebm.calculateScalingTimeline("critical"))
	assert.Equal(t, "within 24 hours", ebm.calculateScalingTimeline("high"))
	assert.Equal(t, "within 1 week", ebm.calculateScalingTimeline("medium"))
	assert.Equal(t, "within 1 month", ebm.calculateScalingTimeline("low"))
}

func TestCalculateScalingImpact(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	highImpact := ebm.calculateScalingImpact("cpu", 0.95)
	assert.Contains(t, highImpact, "high")

	mediumImpact := ebm.calculateScalingImpact("memory", 0.85)
	assert.Contains(t, mediumImpact, "medium")

	lowImpact := ebm.calculateScalingImpact("disk", 0.70)
	assert.Contains(t, lowImpact, "low")
}

func TestCalculateRecommendationConfidence(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ebm := NewEnhancedBaselineManager(db)

	highConfidence := ebm.calculateRecommendationConfidence("cpu", 0.95)
	assert.Equal(t, 0.95, highConfidence)

	mediumConfidence := ebm.calculateRecommendationConfidence("memory", 0.85)
	assert.Equal(t, 0.85, mediumConfidence)

	lowConfidence := ebm.calculateRecommendationConfidence("disk", 0.70)
	assert.Equal(t, 0.75, lowConfidence)
}