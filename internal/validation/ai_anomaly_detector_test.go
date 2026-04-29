package validation

import (
	"testing"
	"time"
)

func TestAIAnomalyDetector_DetectQueryAnomalies(t *testing.T) {
	metrics := &AICodeMetrics{
		DatabaseQueries: &QueryMetrics{
			AvgExecutionTime: 150.0, // 150ms - above baseline of 50ms
			SlowQueries:      10,
			TotalQueries:     100,
		},
	}
	
	detector := NewAIAnomalyDetector(metrics)
	
	anomalies := detector.detectQueryAnomalies()
	
	if len(anomalies) == 0 {
		t.Error("Expected query performance anomaly to be detected")
	}
	
	anomaly := anomalies[0]
	if anomaly.Type != AnomalyTypeQueryPerformance {
		t.Errorf("Expected anomaly type %s, got %s", AnomalyTypeQueryPerformance, anomaly.Type)
	}
	
	if anomaly.Severity != SeverityWarning {
		t.Errorf("Expected severity %s, got %s", SeverityWarning, anomaly.Severity)
	}
	
	// Test slow query rate anomaly
	metrics.DatabaseQueries.SlowQueries = 10
	metrics.DatabaseQueries.TotalQueries = 100 // 10% slow queries
	
	anomalies = detector.detectQueryAnomalies()
	
	// Should detect both execution time and slow query rate anomalies
	if len(anomalies) < 2 {
		t.Errorf("Expected at least 2 anomalies, got %d", len(anomalies))
	}
}

func TestAIAnomalyDetector_DetectErrorAnomalies(t *testing.T) {
	metrics := &AICodeMetrics{
		ErrorHandling: &ErrorMetrics{
			TotalErrors:             100,
			UnhandledErrors:         15, // 15% error rate - above baseline of 2%
			HandledErrors:           85,
			HandlingEffectiveness:   75.0, // Below 80% threshold
		},
	}
	
	detector := NewAIAnomalyDetector(metrics)
	
	anomalies := detector.detectErrorAnomalies()
	
	if len(anomalies) == 0 {
		t.Error("Expected error rate anomaly to be detected")
	}
	
	// Should detect both high error rate and low handling effectiveness
	if len(anomalies) < 2 {
		t.Errorf("Expected at least 2 anomalies, got %d", len(anomalies))
	}
	
	errorRateAnomaly := anomalies[0]
	if errorRateAnomaly.Type != AnomalyTypeErrorRate {
		t.Errorf("Expected anomaly type %s, got %s", AnomalyTypeErrorRate, errorRateAnomaly.Type)
	}
}

func TestAIAnomalyDetector_DetectConsistencyAnomalies(t *testing.T) {
	metrics := &AICodeMetrics{
		BusinessLogic: &LogicMetrics{
			ConsistencyScore: 75.0, // Below baseline of 95%
		},
	}
	
	detector := NewAIAnomalyDetector(metrics)
	
	anomalies := detector.detectConsistencyAnomalies()
	
	if len(anomalies) == 0 {
		t.Error("Expected consistency anomaly to be detected")
	}
	
	anomaly := anomalies[0]
	if anomaly.Type != AnomalyTypeConsistency {
		t.Errorf("Expected anomaly type %s, got %s", AnomalyTypeConsistency, anomaly.Type)
	}
	
	if anomaly.Severity != SeverityWarning {
		t.Errorf("Expected severity %s, got %s", SeverityWarning, anomaly.Severity)
	}
}

func TestAIAnomalyDetector_DetectPerformanceAnomalies(t *testing.T) {
	metrics := &AICodeMetrics{
		Performance: &PerformanceMetrics{
			MemoryUsage:    250 * 1024 * 1024, // 250MB - above baseline of 100MB
			ResponseTimes:  []float64{400, 450, 500, 380, 420}, // Above baseline of 200ms
			ThroughputRPS:  300.0, // Below baseline of 500 RPS
		},
	}
	
	detector := NewAIAnomalyDetector(metrics)
	
	anomalies := detector.detectPerformanceAnomalies()
	
	if len(anomalies) == 0 {
		t.Error("Expected performance anomalies to be detected")
	}
	
	// Should detect memory, response time, and throughput anomalies
	if len(anomalies) < 3 {
		t.Errorf("Expected at least 3 anomalies, got %d", len(anomalies))
	}
	
	// Check for memory usage anomaly
	memoryAnomalyFound := false
	responseTimeAnomalyFound := false
	throughputAnomalyFound := false
	
	for _, anomaly := range anomalies {
		switch anomaly.Type {
		case AnomalyTypeMemoryUsage:
			memoryAnomalyFound = true
		case AnomalyTypeResponseTime:
			responseTimeAnomalyFound = true
		case AnomalyTypeThroughput:
			throughputAnomalyFound = true
		}
	}
	
	if !memoryAnomalyFound {
		t.Error("Expected memory usage anomaly to be detected")
	}
	if !responseTimeAnomalyFound {
		t.Error("Expected response time anomaly to be detected")
	}
	if !throughputAnomalyFound {
		t.Error("Expected throughput anomaly to be detected")
	}
}

func TestAIAnomalyDetector_DetectRegressionAnomalies(t *testing.T) {
	detector := NewAIAnomalyDetector(&AICodeMetrics{
		Performance: &PerformanceMetrics{
			PerformanceScore: 70.0, // Current score
		},
	})
	
	// Populate historical data with declining performance scores
	detector.historicalData.PerformanceScores = []float64{
		90.0, 88.0, 85.0, 82.0, 80.0, 78.0, 75.0, 73.0, 71.0, 70.0,
	}
	
	anomalies := detector.detectRegressionAnomalies()
	
	if len(anomalies) == 0 {
		t.Error("Expected regression anomaly to be detected")
	}
	
	anomaly := anomalies[0]
	if anomaly.Type != AnomalyTypeRegression {
		t.Errorf("Expected anomaly type %s, got %s", AnomalyTypeRegression, anomaly.Type)
	}
	
	if anomaly.TrendAnalysis == nil {
		t.Error("Expected trend analysis to be included")
	}
	
	if anomaly.TrendAnalysis.Direction != TrendDirectionDown {
		t.Errorf("Expected trend direction %s, got %s", TrendDirectionDown, anomaly.TrendAnalysis.Direction)
	}
}

func TestAIAnomalyDetector_UpdateHistoricalData(t *testing.T) {
	metrics := &AICodeMetrics{
		DatabaseQueries: &QueryMetrics{
			AvgExecutionTime: 75.0,
		},
		ErrorHandling: &ErrorMetrics{
			TotalErrors:     10,
			UnhandledErrors: 2,
		},
		BusinessLogic: &LogicMetrics{
			ConsistencyScore: 92.0,
		},
		Performance: &PerformanceMetrics{
			PerformanceScore: 85.0,
			MemoryUsage:      120 * 1024 * 1024,
			ResponseTimes:    []float64{180, 200, 190},
			ThroughputRPS:    450.0,
		},
	}
	
	detector := NewAIAnomalyDetector(metrics)
	
	// Update historical data
	detector.updateHistoricalData()
	
	// Check that data was added
	if len(detector.historicalData.QueryTimes) != 1 {
		t.Errorf("Expected 1 query time entry, got %d", len(detector.historicalData.QueryTimes))
	}
	
	if detector.historicalData.QueryTimes[0] != 75.0 {
		t.Errorf("Expected query time 75.0, got %f", detector.historicalData.QueryTimes[0])
	}
	
	if len(detector.historicalData.ErrorRates) != 1 {
		t.Errorf("Expected 1 error rate entry, got %d", len(detector.historicalData.ErrorRates))
	}
	
	expectedErrorRate := 20.0 // 2/10 * 100
	if detector.historicalData.ErrorRates[0] != expectedErrorRate {
		t.Errorf("Expected error rate %f, got %f", expectedErrorRate, detector.historicalData.ErrorRates[0])
	}
	
	// Test size limit
	for i := 0; i < 1005; i++ {
		detector.updateHistoricalData()
	}
	
	if len(detector.historicalData.QueryTimes) > detector.historicalData.MaxHistorySize {
		t.Errorf("Historical data exceeded max size: %d > %d", 
			len(detector.historicalData.QueryTimes), detector.historicalData.MaxHistorySize)
	}
}

func TestAIAnomalyDetector_AnalyzeTrend(t *testing.T) {
	detector := NewAIAnomalyDetector(&AICodeMetrics{})
	
	// Test upward trend
	upwardData := []float64{10, 12, 14, 16, 18, 20, 22, 24, 26, 28}
	trend := detector.analyzeTrend(upwardData)
	
	if trend == nil {
		t.Fatal("Expected trend analysis result")
	}
	
	if trend.Direction != TrendDirectionUp {
		t.Errorf("Expected upward trend, got %s", trend.Direction)
	}
	
	if trend.Slope <= 0 {
		t.Errorf("Expected positive slope for upward trend, got %f", trend.Slope)
	}
	
	if trend.RSquared < 0.9 {
		t.Errorf("Expected high R-squared for linear data, got %f", trend.RSquared)
	}
	
	// Test downward trend
	downwardData := []float64{100, 95, 90, 85, 80, 75, 70, 65, 60, 55}
	trend = detector.analyzeTrend(downwardData)
	
	if trend.Direction != TrendDirectionDown {
		t.Errorf("Expected downward trend, got %s", trend.Direction)
	}
	
	if trend.Slope >= 0 {
		t.Errorf("Expected negative slope for downward trend, got %f", trend.Slope)
	}
	
	// Test stable trend
	stableData := []float64{50, 51, 49, 50, 52, 48, 50, 51, 49, 50}
	trend = detector.analyzeTrend(stableData)
	
	if trend.Direction != TrendDirectionStable {
		t.Errorf("Expected stable trend, got %s", trend.Direction)
	}
	
	// Test volatile trend
	volatileData := []float64{10, 50, 20, 80, 15, 90, 25, 70, 30, 60}
	trend = detector.analyzeTrend(volatileData)
	
	if trend.Direction != TrendDirectionVolatile {
		t.Errorf("Expected volatile trend, got %s", trend.Direction)
	}
	
	if trend.RSquared > 0.3 {
		t.Errorf("Expected low R-squared for volatile data, got %f", trend.RSquared)
	}
}

func TestAIAnomalyDetector_CalculateSeverity(t *testing.T) {
	detector := NewAIAnomalyDetector(&AICodeMetrics{})
	
	testCases := []struct {
		current   float64
		baseline  float64
		threshold float64
		expected  AnomalySeverity
	}{
		{100, 50, 75, SeverityCritical}, // 100% deviation
		{75, 50, 60, SeverityWarning},   // 50% deviation
		{60, 50, 55, SeverityInfo},      // 20% deviation
	}
	
	for _, tc := range testCases {
		severity := detector.calculateSeverity(tc.current, tc.baseline, tc.threshold)
		if severity != tc.expected {
			t.Errorf("Expected severity %s for current=%f, baseline=%f, threshold=%f, got %s",
				tc.expected, tc.current, tc.baseline, tc.threshold, severity)
		}
	}
}

func TestAIAnomalyDetector_CalculateConfidence(t *testing.T) {
	detector := NewAIAnomalyDetector(&AICodeMetrics{})
	
	testCases := []struct {
		current  float64
		baseline float64
		stdDev   float64
		minConf  float64
		maxConf  float64
	}{
		{100, 50, 10, 0.95, 1.0},  // High z-score
		{60, 50, 10, 0.6, 0.9},    // Medium z-score
		{55, 50, 10, 0.5, 0.7},    // Low z-score
	}
	
	for _, tc := range testCases {
		confidence := detector.calculateConfidence(tc.current, tc.baseline, tc.stdDev)
		if confidence < tc.minConf || confidence > tc.maxConf {
			t.Errorf("Expected confidence between %f and %f for current=%f, baseline=%f, stdDev=%f, got %f",
				tc.minConf, tc.maxConf, tc.current, tc.baseline, tc.stdDev, confidence)
		}
	}
}

func TestAIAnomalyDetector_UpdateBaselines(t *testing.T) {
	detector := NewAIAnomalyDetector(&AICodeMetrics{})
	
	// Populate historical data
	for i := 0; i < 60; i++ {
		detector.historicalData.QueryTimes = append(detector.historicalData.QueryTimes, 45.0+float64(i%10))
		detector.historicalData.ErrorRates = append(detector.historicalData.ErrorRates, 1.5+float64(i%5)*0.1)
		detector.historicalData.ConsistencyScores = append(detector.historicalData.ConsistencyScores, 93.0+float64(i%8))
		detector.historicalData.PerformanceScores = append(detector.historicalData.PerformanceScores, 82.0+float64(i%12))
	}
	
	originalBaselines := *detector.baselines
	
	detector.UpdateBaselines()
	
	// Check that baselines were updated
	if detector.baselines.QueryExecutionTime == originalBaselines.QueryExecutionTime {
		t.Error("Query execution time baseline should have been updated")
	}
	
	if detector.baselines.ErrorRate == originalBaselines.ErrorRate {
		t.Error("Error rate baseline should have been updated")
	}
	
	if detector.baselines.ConsistencyScore == originalBaselines.ConsistencyScore {
		t.Error("Consistency score baseline should have been updated")
	}
	
	if detector.baselines.PerformanceScore == originalBaselines.PerformanceScore {
		t.Error("Performance score baseline should have been updated")
	}
	
	if detector.baselines.LastUpdated.Before(originalBaselines.LastUpdated) {
		t.Error("LastUpdated should have been updated")
	}
	
	if detector.baselines.SampleSize != 50 {
		t.Errorf("Expected sample size 50, got %d", detector.baselines.SampleSize)
	}
}

func TestAIAnomalyDetector_GetRecommendations(t *testing.T) {
	detector := NewAIAnomalyDetector(&AICodeMetrics{})
	
	testCases := []struct {
		current    float64
		baseline   float64
		recFunc    func(float64, float64) string
		shouldContain string
	}{
		{250, 50, detector.getQueryPerformanceRecommendation, "Critical"},
		{150, 50, detector.getQueryPerformanceRecommendation, "High"},
		{75, 50, detector.getQueryPerformanceRecommendation, "Medium"},
		{4, 2, detector.getErrorHandlingRecommendation, "Critical"},
		{3, 2, detector.getErrorHandlingRecommendation, "High"},
		{2.5, 2, detector.getErrorHandlingRecommendation, "Medium"},
		{75, 95, detector.getConsistencyRecommendation, "Critical"},
		{85, 95, detector.getConsistencyRecommendation, "High"},
		{90, 95, detector.getConsistencyRecommendation, "Medium"},
	}
	
	for _, tc := range testCases {
		recommendation := tc.recFunc(tc.current, tc.baseline)
		if recommendation == "" {
			t.Errorf("Expected non-empty recommendation for current=%f, baseline=%f", tc.current, tc.baseline)
		}
		
		if len(recommendation) < 10 {
			t.Errorf("Expected detailed recommendation, got: %s", recommendation)
		}
	}
}

func TestAIAnomalyDetector_Integration(t *testing.T) {
	// Create metrics with various issues
	metrics := &AICodeMetrics{
		DatabaseQueries: &QueryMetrics{
			AvgExecutionTime: 120.0, // Above baseline
			SlowQueries:      8,
			TotalQueries:     100,
		},
		ErrorHandling: &ErrorMetrics{
			TotalErrors:             50,
			UnhandledErrors:         8, // 16% error rate
			HandledErrors:           42,
			HandlingEffectiveness:   75.0, // Below threshold
		},
		BusinessLogic: &LogicMetrics{
			ConsistencyScore: 78.0, // Below baseline
		},
		Performance: &PerformanceMetrics{
			MemoryUsage:    220 * 1024 * 1024, // Above baseline
			ResponseTimes:  []float64{350, 400, 380, 420, 390}, // Above baseline
			ThroughputRPS:  320.0, // Below baseline
			PerformanceScore: 72.0,
		},
	}
	
	detector := NewAIAnomalyDetector(metrics)
	
	// Populate some historical data for regression detection
	for i := 0; i < 15; i++ {
		score := 85.0 - float64(i)*1.0 // Declining performance
		detector.historicalData.PerformanceScores = append(detector.historicalData.PerformanceScores, score)
	}
	
	// Detect all anomalies
	anomalies := detector.DetectAnomalies()
	
	if len(anomalies) == 0 {
		t.Error("Expected multiple anomalies to be detected")
	}
	
	// Check that we detected various types of anomalies
	anomalyTypes := make(map[AnomalyType]bool)
	for _, anomaly := range anomalies {
		anomalyTypes[anomaly.Type] = true
		
		// Verify anomaly structure
		if anomaly.Description == "" {
			t.Error("Anomaly should have a description")
		}
		
		if anomaly.DetectedAt.IsZero() {
			t.Error("Anomaly should have a detection timestamp")
		}
		
		if anomaly.Confidence <= 0 || anomaly.Confidence > 1 {
			t.Errorf("Anomaly confidence should be between 0 and 1, got %f", anomaly.Confidence)
		}
		
		if anomaly.Recommendation == "" {
			t.Error("Anomaly should have a recommendation")
		}
	}
	
	// We should have detected multiple types of anomalies
	expectedTypes := []AnomalyType{
		AnomalyTypeQueryPerformance,
		AnomalyTypeErrorRate,
		AnomalyTypeConsistency,
		AnomalyTypeMemoryUsage,
		AnomalyTypeResponseTime,
		AnomalyTypeThroughput,
		AnomalyTypeRegression,
	}
	
	detectedCount := 0
	for _, expectedType := range expectedTypes {
		if anomalyTypes[expectedType] {
			detectedCount++
		}
	}
	
	if detectedCount < 5 {
		t.Errorf("Expected to detect at least 5 different anomaly types, got %d", detectedCount)
	}
	
	t.Logf("Detected %d anomalies of %d different types", len(anomalies), detectedCount)
	for _, anomaly := range anomalies {
		t.Logf("- %s: %s (Severity: %s, Confidence: %.2f)", 
			anomaly.Type, anomaly.Description, anomaly.Severity, anomaly.Confidence)
	}
}

// Benchmark tests
func BenchmarkAIAnomalyDetector_DetectAnomalies(b *testing.B) {
	metrics := &AICodeMetrics{
		DatabaseQueries: &QueryMetrics{
			AvgExecutionTime: 120.0,
			SlowQueries:      8,
			TotalQueries:     100,
		},
		ErrorHandling: &ErrorMetrics{
			TotalErrors:             50,
			UnhandledErrors:         8,
			HandledErrors:           42,
			HandlingEffectiveness:   75.0,
		},
		BusinessLogic: &LogicMetrics{
			ConsistencyScore: 78.0,
		},
		Performance: &PerformanceMetrics{
			MemoryUsage:    220 * 1024 * 1024,
			ResponseTimes:  []float64{350, 400, 380, 420, 390},
			ThroughputRPS:  320.0,
			PerformanceScore: 72.0,
		},
	}
	
	detector := NewAIAnomalyDetector(metrics)
	
	// Populate historical data
	for i := 0; i < 100; i++ {
		detector.historicalData.PerformanceScores = append(detector.historicalData.PerformanceScores, 85.0-float64(i)*0.1)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = detector.DetectAnomalies()
	}
}

func BenchmarkAIAnomalyDetector_AnalyzeTrend(b *testing.B) {
	detector := NewAIAnomalyDetector(&AICodeMetrics{})
	
	// Create test data
	data := make([]float64, 100)
	for i := 0; i < 100; i++ {
		data[i] = 50.0 + float64(i)*0.5 + float64(i%10)*2.0 // Trend with noise
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = detector.analyzeTrend(data)
	}
}