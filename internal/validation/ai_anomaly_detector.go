package validation

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// AIAnomalyDetector detects anomalies in AI-generated code behavior
type AIAnomalyDetector struct {
	metrics         *AICodeMetrics
	baselines       *AICodeBaselines
	thresholds      *AnomalyThresholds
	historicalData  *HistoricalMetrics
}

// AICodeBaselines stores baseline performance metrics for AI-generated code
type AICodeBaselines struct {
	QueryExecutionTime    float64   `json:"query_execution_time_ms"`
	ErrorRate            float64   `json:"error_rate_percent"`
	ConsistencyScore     float64   `json:"consistency_score"`
	PerformanceScore     float64   `json:"performance_score"`
	MemoryUsage          int64     `json:"memory_usage_bytes"`
	ResponseTime         float64   `json:"response_time_ms"`
	ThroughputRPS        float64   `json:"throughput_rps"`
	LastUpdated          time.Time `json:"last_updated"`
	SampleSize           int       `json:"sample_size"`
	ConfidenceInterval   float64   `json:"confidence_interval"`
}

// AnomalyThresholds defines thresholds for anomaly detection
type AnomalyThresholds struct {
	QueryTimeMultiplier     float64 `json:"query_time_multiplier"`
	ErrorRateIncrease       float64 `json:"error_rate_increase_percent"`
	ConsistencyScoreDecrease float64 `json:"consistency_score_decrease_percent"`
	PerformanceScoreDecrease float64 `json:"performance_score_decrease_percent"`
	MemoryUsageMultiplier   float64 `json:"memory_usage_multiplier"`
	ResponseTimeMultiplier  float64 `json:"response_time_multiplier"`
	ThroughputDecrease      float64 `json:"throughput_decrease_percent"`
}

// HistoricalMetrics stores historical data for trend analysis
type HistoricalMetrics struct {
	QueryTimes      []float64   `json:"query_times"`
	ErrorRates      []float64   `json:"error_rates"`
	ConsistencyScores []float64 `json:"consistency_scores"`
	PerformanceScores []float64 `json:"performance_scores"`
	MemoryUsages    []int64     `json:"memory_usages"`
	ResponseTimes   []float64   `json:"response_times"`
	Throughputs     []float64   `json:"throughputs"`
	Timestamps      []time.Time `json:"timestamps"`
	MaxHistorySize  int         `json:"max_history_size"`
}

// Anomaly represents a detected anomaly in AI code behavior
type Anomaly struct {
	Type            AnomalyType   `json:"type"`
	Severity        AnomalySeverity `json:"severity"`
	Description     string        `json:"description"`
	CurrentValue    interface{}   `json:"current_value"`
	BaselineValue   interface{}   `json:"baseline_value"`
	Threshold       interface{}   `json:"threshold"`
	DetectedAt      time.Time     `json:"detected_at"`
	Confidence      float64       `json:"confidence"`
	Recommendation  string        `json:"recommendation"`
	TrendAnalysis   *TrendAnalysis `json:"trend_analysis,omitempty"`
}

// AnomalyType represents the type of anomaly
type AnomalyType string

const (
	AnomalyTypeQueryPerformance  AnomalyType = "query_performance"
	AnomalyTypeErrorRate         AnomalyType = "error_rate"
	AnomalyTypeConsistency       AnomalyType = "consistency"
	AnomalyTypePerformance       AnomalyType = "performance"
	AnomalyTypeMemoryUsage       AnomalyType = "memory_usage"
	AnomalyTypeResponseTime      AnomalyType = "response_time"
	AnomalyTypeThroughput        AnomalyType = "throughput"
	AnomalyTypeRegression        AnomalyType = "regression"
)

// AnomalySeverity represents the severity of an anomaly
type AnomalySeverity string

const (
	SeverityInfo     AnomalySeverity = "info"
	SeverityWarning  AnomalySeverity = "warning"
	SeverityCritical AnomalySeverity = "critical"
)

// TrendAnalysis provides trend analysis for anomalies
type TrendAnalysis struct {
	Direction       TrendDirection `json:"direction"`
	Slope           float64        `json:"slope"`
	RSquared        float64        `json:"r_squared"`
	PredictedValue  float64        `json:"predicted_value"`
	TrendStrength   string         `json:"trend_strength"`
}

// TrendDirection represents the direction of a trend
type TrendDirection string

const (
	TrendDirectionUp       TrendDirection = "up"
	TrendDirectionDown     TrendDirection = "down"
	TrendDirectionStable   TrendDirection = "stable"
	TrendDirectionVolatile TrendDirection = "volatile"
)

// NewAIAnomalyDetector creates a new AI anomaly detector
func NewAIAnomalyDetector(metrics *AICodeMetrics) *AIAnomalyDetector {
	return &AIAnomalyDetector{
		metrics: metrics,
		baselines: &AICodeBaselines{
			QueryExecutionTime: 50.0,  // 50ms baseline
			ErrorRate:         2.0,    // 2% error rate baseline
			ConsistencyScore:  95.0,   // 95% consistency baseline
			PerformanceScore:  85.0,   // 85% performance baseline
			MemoryUsage:       100 * 1024 * 1024, // 100MB baseline
			ResponseTime:      200.0,  // 200ms response time baseline
			ThroughputRPS:     500.0,  // 500 RPS baseline
			ConfidenceInterval: 0.95,
		},
		thresholds: &AnomalyThresholds{
			QueryTimeMultiplier:      2.0,  // 2x slower than baseline
			ErrorRateIncrease:        50.0, // 50% increase in error rate
			ConsistencyScoreDecrease: 10.0, // 10% decrease in consistency
			PerformanceScoreDecrease: 15.0, // 15% decrease in performance
			MemoryUsageMultiplier:    2.0,  // 2x memory usage
			ResponseTimeMultiplier:   2.5,  // 2.5x response time
			ThroughputDecrease:       30.0, // 30% decrease in throughput
		},
		historicalData: &HistoricalMetrics{
			QueryTimes:      make([]float64, 0),
			ErrorRates:      make([]float64, 0),
			ConsistencyScores: make([]float64, 0),
			PerformanceScores: make([]float64, 0),
			MemoryUsages:    make([]int64, 0),
			ResponseTimes:   make([]float64, 0),
			Throughputs:     make([]float64, 0),
			Timestamps:      make([]time.Time, 0),
			MaxHistorySize:  1000, // Keep last 1000 data points
		},
	}
}

// DetectAnomalies detects anomalies in current metrics compared to baselines
func (d *AIAnomalyDetector) DetectAnomalies() []Anomaly {
	var anomalies []Anomaly
	
	// Update historical data
	d.updateHistoricalData()
	
	// Detect query performance anomalies
	if queryAnomalies := d.detectQueryAnomalies(); len(queryAnomalies) > 0 {
		anomalies = append(anomalies, queryAnomalies...)
	}
	
	// Detect error rate anomalies
	if errorAnomalies := d.detectErrorAnomalies(); len(errorAnomalies) > 0 {
		anomalies = append(anomalies, errorAnomalies...)
	}
	
	// Detect consistency anomalies
	if consistencyAnomalies := d.detectConsistencyAnomalies(); len(consistencyAnomalies) > 0 {
		anomalies = append(anomalies, consistencyAnomalies...)
	}
	
	// Detect performance anomalies
	if performanceAnomalies := d.detectPerformanceAnomalies(); len(performanceAnomalies) > 0 {
		anomalies = append(anomalies, performanceAnomalies...)
	}
	
	// Detect regression anomalies
	if regressionAnomalies := d.detectRegressionAnomalies(); len(regressionAnomalies) > 0 {
		anomalies = append(anomalies, regressionAnomalies...)
	}
	
	return anomalies
}

// detectQueryAnomalies detects database query performance anomalies
func (d *AIAnomalyDetector) detectQueryAnomalies() []Anomaly {
	var anomalies []Anomaly
	
	currentAvgTime := d.metrics.DatabaseQueries.AvgExecutionTime
	baselineTime := d.baselines.QueryExecutionTime
	threshold := baselineTime * d.thresholds.QueryTimeMultiplier
	
	if currentAvgTime > threshold {
		severity := d.calculateSeverity(currentAvgTime, baselineTime, threshold)
		confidence := d.calculateConfidence(currentAvgTime, baselineTime, d.getQueryTimeStdDev())
		
		anomaly := Anomaly{
			Type:           AnomalyTypeQueryPerformance,
			Severity:       severity,
			Description:    fmt.Sprintf("Database query execution time is %.2fx higher than baseline", currentAvgTime/baselineTime),
			CurrentValue:   currentAvgTime,
			BaselineValue:  baselineTime,
			Threshold:      threshold,
			DetectedAt:     time.Now(),
			Confidence:     confidence,
			Recommendation: d.getQueryPerformanceRecommendation(currentAvgTime, baselineTime),
			TrendAnalysis:  d.analyzeTrend(d.historicalData.QueryTimes),
		}
		
		anomalies = append(anomalies, anomaly)
	}
	
	// Check for slow query patterns
	if d.metrics.DatabaseQueries.SlowQueries > 0 {
		slowQueryRate := float64(d.metrics.DatabaseQueries.SlowQueries) / float64(d.metrics.DatabaseQueries.TotalQueries) * 100
		if slowQueryRate > 5.0 { // More than 5% slow queries
			anomaly := Anomaly{
				Type:           AnomalyTypeQueryPerformance,
				Severity:       SeverityWarning,
				Description:    fmt.Sprintf("High slow query rate: %.2f%% of queries are slow", slowQueryRate),
				CurrentValue:   slowQueryRate,
				BaselineValue:  5.0,
				Threshold:      5.0,
				DetectedAt:     time.Now(),
				Confidence:     0.9,
				Recommendation: "Review slow queries and optimize indexes or query patterns",
			}
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies
}

// detectErrorAnomalies detects error handling anomalies
func (d *AIAnomalyDetector) detectErrorAnomalies() []Anomaly {
	var anomalies []Anomaly
	
	if d.metrics.ErrorHandling.TotalErrors == 0 {
		return anomalies
	}
	
	currentErrorRate := float64(d.metrics.ErrorHandling.UnhandledErrors) / float64(d.metrics.ErrorHandling.TotalErrors) * 100
	baselineErrorRate := d.baselines.ErrorRate
	threshold := baselineErrorRate * (1 + d.thresholds.ErrorRateIncrease/100)
	
	if currentErrorRate > threshold {
		severity := d.calculateSeverity(currentErrorRate, baselineErrorRate, threshold)
		confidence := d.calculateConfidence(currentErrorRate, baselineErrorRate, d.getErrorRateStdDev())
		
		anomaly := Anomaly{
			Type:           AnomalyTypeErrorRate,
			Severity:       severity,
			Description:    fmt.Sprintf("Error rate increased to %.2f%% (baseline: %.2f%%)", currentErrorRate, baselineErrorRate),
			CurrentValue:   currentErrorRate,
			BaselineValue:  baselineErrorRate,
			Threshold:      threshold,
			DetectedAt:     time.Now(),
			Confidence:     confidence,
			Recommendation: d.getErrorHandlingRecommendation(currentErrorRate, baselineErrorRate),
			TrendAnalysis:  d.analyzeTrend(d.historicalData.ErrorRates),
		}
		
		anomalies = append(anomalies, anomaly)
	}
	
	// Check error handling effectiveness
	if d.metrics.ErrorHandling.HandlingEffectiveness < 80.0 {
		anomaly := Anomaly{
			Type:           AnomalyTypeErrorRate,
			Severity:       SeverityWarning,
			Description:    fmt.Sprintf("Low error handling effectiveness: %.2f%%", d.metrics.ErrorHandling.HandlingEffectiveness),
			CurrentValue:   d.metrics.ErrorHandling.HandlingEffectiveness,
			BaselineValue:  90.0,
			Threshold:      80.0,
			DetectedAt:     time.Now(),
			Confidence:     0.85,
			Recommendation: "Review error handling patterns in AI-generated code and add missing error checks",
		}
		anomalies = append(anomalies, anomaly)
	}
	
	return anomalies
}

// detectConsistencyAnomalies detects business logic consistency anomalies
func (d *AIAnomalyDetector) detectConsistencyAnomalies() []Anomaly {
	var anomalies []Anomaly
	
	currentScore := d.metrics.BusinessLogic.ConsistencyScore
	baselineScore := d.baselines.ConsistencyScore
	threshold := baselineScore * (1 - d.thresholds.ConsistencyScoreDecrease/100)
	
	if currentScore < threshold {
		severity := d.calculateSeverity(baselineScore-currentScore, 0, baselineScore-threshold)
		confidence := d.calculateConfidence(currentScore, baselineScore, d.getConsistencyStdDev())
		
		anomaly := Anomaly{
			Type:           AnomalyTypeConsistency,
			Severity:       severity,
			Description:    fmt.Sprintf("Business logic consistency score dropped to %.2f%% (baseline: %.2f%%)", currentScore, baselineScore),
			CurrentValue:   currentScore,
			BaselineValue:  baselineScore,
			Threshold:      threshold,
			DetectedAt:     time.Now(),
			Confidence:     confidence,
			Recommendation: d.getConsistencyRecommendation(currentScore, baselineScore),
			TrendAnalysis:  d.analyzeTrend(d.historicalData.ConsistencyScores),
		}
		
		anomalies = append(anomalies, anomaly)
	}
	
	return anomalies
}

// detectPerformanceAnomalies detects overall performance anomalies
func (d *AIAnomalyDetector) detectPerformanceAnomalies() []Anomaly {
	var anomalies []Anomaly
	
	// Memory usage anomaly
	currentMemory := d.metrics.Performance.MemoryUsage
	baselineMemory := d.baselines.MemoryUsage
	memoryThreshold := int64(float64(baselineMemory) * d.thresholds.MemoryUsageMultiplier)
	
	if currentMemory > memoryThreshold {
		severity := d.calculateSeverity(float64(currentMemory), float64(baselineMemory), float64(memoryThreshold))
		
		anomaly := Anomaly{
			Type:           AnomalyTypeMemoryUsage,
			Severity:       severity,
			Description:    fmt.Sprintf("Memory usage is %.2fx higher than baseline", float64(currentMemory)/float64(baselineMemory)),
			CurrentValue:   currentMemory,
			BaselineValue:  baselineMemory,
			Threshold:      memoryThreshold,
			DetectedAt:     time.Now(),
			Confidence:     0.9,
			Recommendation: "Review memory allocation patterns and check for memory leaks in AI-generated code",
			TrendAnalysis:  d.analyzeMemoryTrend(),
		}
		
		anomalies = append(anomalies, anomaly)
	}
	
	// Response time anomaly
	if len(d.metrics.Performance.ResponseTimes) > 0 {
		currentResponseTime := d.calculateAverage(d.metrics.Performance.ResponseTimes)
		baselineResponseTime := d.baselines.ResponseTime
		responseThreshold := baselineResponseTime * d.thresholds.ResponseTimeMultiplier
		
		if currentResponseTime > responseThreshold {
			severity := d.calculateSeverity(currentResponseTime, baselineResponseTime, responseThreshold)
			
			anomaly := Anomaly{
				Type:           AnomalyTypeResponseTime,
				Severity:       severity,
				Description:    fmt.Sprintf("Response time is %.2fx higher than baseline", currentResponseTime/baselineResponseTime),
				CurrentValue:   currentResponseTime,
				BaselineValue:  baselineResponseTime,
				Threshold:      responseThreshold,
				DetectedAt:     time.Now(),
				Confidence:     0.85,
				Recommendation: "Optimize AI-generated code performance and check for bottlenecks",
				TrendAnalysis:  d.analyzeTrend(d.historicalData.ResponseTimes),
			}
			
			anomalies = append(anomalies, anomaly)
		}
	}
	
	// Throughput anomaly
	currentThroughput := d.metrics.Performance.ThroughputRPS
	baselineThroughput := d.baselines.ThroughputRPS
	throughputThreshold := baselineThroughput * (1 - d.thresholds.ThroughputDecrease/100)
	
	if currentThroughput < throughputThreshold {
		severity := d.calculateSeverity(baselineThroughput-currentThroughput, 0, baselineThroughput-throughputThreshold)
		
		anomaly := Anomaly{
			Type:           AnomalyTypeThroughput,
			Severity:       severity,
			Description:    fmt.Sprintf("Throughput dropped to %.2f RPS (baseline: %.2f RPS)", currentThroughput, baselineThroughput),
			CurrentValue:   currentThroughput,
			BaselineValue:  baselineThroughput,
			Threshold:      throughputThreshold,
			DetectedAt:     time.Now(),
			Confidence:     0.8,
			Recommendation: "Review system capacity and optimize AI-generated code for better throughput",
			TrendAnalysis:  d.analyzeTrend(d.historicalData.Throughputs),
		}
		
		anomalies = append(anomalies, anomaly)
	}
	
	return anomalies
}

// detectRegressionAnomalies detects regression patterns in AI code quality
func (d *AIAnomalyDetector) detectRegressionAnomalies() []Anomaly {
	var anomalies []Anomaly
	
	// Analyze trends for regression detection
	if len(d.historicalData.PerformanceScores) >= 10 {
		trend := d.analyzeTrend(d.historicalData.PerformanceScores)
		if trend != nil && trend.Direction == TrendDirectionDown && trend.RSquared > 0.7 {
			anomaly := Anomaly{
				Type:           AnomalyTypeRegression,
				Severity:       SeverityWarning,
				Description:    "Detected performance regression trend in AI-generated code",
				CurrentValue:   d.metrics.Performance.PerformanceScore,
				BaselineValue:  d.baselines.PerformanceScore,
				DetectedAt:     time.Now(),
				Confidence:     trend.RSquared,
				Recommendation: "Review recent AI-generated code changes and identify performance degradation causes",
				TrendAnalysis:  trend,
			}
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies
}

// updateHistoricalData updates historical metrics for trend analysis
func (d *AIAnomalyDetector) updateHistoricalData() {
	now := time.Now()
	
	// Add current metrics to historical data
	d.addToHistory(&d.historicalData.QueryTimes, d.metrics.DatabaseQueries.AvgExecutionTime)
	d.addToHistory(&d.historicalData.ErrorRates, d.calculateCurrentErrorRate())
	d.addToHistory(&d.historicalData.ConsistencyScores, d.metrics.BusinessLogic.ConsistencyScore)
	d.addToHistory(&d.historicalData.PerformanceScores, d.metrics.Performance.PerformanceScore)
	d.addToHistoryInt64(&d.historicalData.MemoryUsages, d.metrics.Performance.MemoryUsage)
	
	if len(d.metrics.Performance.ResponseTimes) > 0 {
		avgResponseTime := d.calculateAverage(d.metrics.Performance.ResponseTimes)
		d.addToHistory(&d.historicalData.ResponseTimes, avgResponseTime)
	}
	
	d.addToHistory(&d.historicalData.Throughputs, d.metrics.Performance.ThroughputRPS)
	d.addToHistoryTime(&d.historicalData.Timestamps, now)
}

// addToHistory adds a value to historical data with size limit
func (d *AIAnomalyDetector) addToHistory(slice *[]float64, value float64) {
	if len(*slice) >= d.historicalData.MaxHistorySize {
		*slice = (*slice)[1:]
	}
	*slice = append(*slice, value)
}

// addToHistoryInt64 adds an int64 value to historical data with size limit
func (d *AIAnomalyDetector) addToHistoryInt64(slice *[]int64, value int64) {
	if len(*slice) >= d.historicalData.MaxHistorySize {
		*slice = (*slice)[1:]
	}
	*slice = append(*slice, value)
}

// addToHistoryTime adds a time value to historical data with size limit
func (d *AIAnomalyDetector) addToHistoryTime(slice *[]time.Time, value time.Time) {
	if len(*slice) >= d.historicalData.MaxHistorySize {
		*slice = (*slice)[1:]
	}
	*slice = append(*slice, value)
}

// calculateCurrentErrorRate calculates the current error rate
func (d *AIAnomalyDetector) calculateCurrentErrorRate() float64 {
	if d.metrics.ErrorHandling.TotalErrors == 0 {
		return 0.0
	}
	return float64(d.metrics.ErrorHandling.UnhandledErrors) / float64(d.metrics.ErrorHandling.TotalErrors) * 100
}

// calculateAverage calculates the average of a float64 slice
func (d *AIAnomalyDetector) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateSeverity calculates anomaly severity based on deviation from baseline
func (d *AIAnomalyDetector) calculateSeverity(current, baseline, threshold float64) AnomalySeverity {
	deviation := math.Abs(current - baseline) / baseline
	
	if deviation > 1.0 { // More than 100% deviation
		return SeverityCritical
	} else if deviation > 0.5 { // More than 50% deviation
		return SeverityWarning
	}
	return SeverityInfo
}

// calculateConfidence calculates confidence level for anomaly detection
func (d *AIAnomalyDetector) calculateConfidence(current, baseline, stdDev float64) float64 {
	if stdDev == 0 {
		return 0.9
	}
	
	zScore := math.Abs(current-baseline) / stdDev
	
	// Convert z-score to confidence (simplified)
	if zScore > 3.0 {
		return 0.99
	} else if zScore > 2.0 {
		return 0.95
	} else if zScore > 1.0 {
		return 0.8
	}
	return 0.6
}

// analyzeTrend analyzes trend in historical data
func (d *AIAnomalyDetector) analyzeTrend(data []float64) *TrendAnalysis {
	if len(data) < 5 {
		return nil
	}
	
	// Simple linear regression
	n := float64(len(data))
	var sumX, sumY, sumXY, sumX2 float64
	
	for i, y := range data {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n
	
	// Calculate R-squared
	var ssRes, ssTot float64
	meanY := sumY / n
	
	for i, y := range data {
		x := float64(i)
		predicted := slope*x + intercept
		ssRes += (y - predicted) * (y - predicted)
		ssTot += (y - meanY) * (y - meanY)
	}
	
	rSquared := 1 - (ssRes / ssTot)
	if math.IsNaN(rSquared) || math.IsInf(rSquared, 0) {
		rSquared = 0
	}
	
	// Determine trend direction
	var direction TrendDirection
	if math.Abs(slope) < 0.01 {
		direction = TrendDirectionStable
	} else if slope > 0 {
		direction = TrendDirectionUp
	} else {
		direction = TrendDirectionDown
	}
	
	// Check for volatility
	if rSquared < 0.3 {
		direction = TrendDirectionVolatile
	}
	
	// Predict next value
	nextX := float64(len(data))
	predictedValue := slope*nextX + intercept
	
	// Determine trend strength
	var strength string
	if rSquared > 0.8 {
		strength = "strong"
	} else if rSquared > 0.5 {
		strength = "moderate"
	} else {
		strength = "weak"
	}
	
	return &TrendAnalysis{
		Direction:      direction,
		Slope:          slope,
		RSquared:       rSquared,
		PredictedValue: predictedValue,
		TrendStrength:  strength,
	}
}

// analyzeMemoryTrend analyzes memory usage trend
func (d *AIAnomalyDetector) analyzeMemoryTrend() *TrendAnalysis {
	if len(d.historicalData.MemoryUsages) < 5 {
		return nil
	}
	
	// Convert int64 to float64 for trend analysis
	data := make([]float64, len(d.historicalData.MemoryUsages))
	for i, v := range d.historicalData.MemoryUsages {
		data[i] = float64(v)
	}
	
	return d.analyzeTrend(data)
}

// Standard deviation calculation helpers
func (d *AIAnomalyDetector) getQueryTimeStdDev() float64 {
	return d.calculateStdDev(d.historicalData.QueryTimes)
}

func (d *AIAnomalyDetector) getErrorRateStdDev() float64 {
	return d.calculateStdDev(d.historicalData.ErrorRates)
}

func (d *AIAnomalyDetector) getConsistencyStdDev() float64 {
	return d.calculateStdDev(d.historicalData.ConsistencyScores)
}

func (d *AIAnomalyDetector) calculateStdDev(data []float64) float64 {
	if len(data) < 2 {
		return 1.0 // Default standard deviation
	}
	
	mean := d.calculateAverage(data)
	var variance float64
	
	for _, v := range data {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(data) - 1)
	
	return math.Sqrt(variance)
}

// Recommendation generators
func (d *AIAnomalyDetector) getQueryPerformanceRecommendation(current, baseline float64) string {
	ratio := current / baseline
	if ratio > 5.0 {
		return "Critical: Query performance is severely degraded. Review AI-generated queries for missing indexes, inefficient joins, or N+1 query patterns."
	} else if ratio > 3.0 {
		return "High: Significant query performance degradation detected. Analyze slow query log and optimize AI-generated database access patterns."
	}
	return "Medium: Query performance is slower than baseline. Consider reviewing recent AI-generated database code for optimization opportunities."
}

func (d *AIAnomalyDetector) getErrorHandlingRecommendation(current, baseline float64) string {
	increase := (current - baseline) / baseline * 100
	if increase > 100 {
		return "Critical: Error rate has doubled. Review AI-generated error handling patterns and add missing error checks."
	} else if increase > 50 {
		return "High: Significant increase in error rate. Audit AI-generated code for proper error handling implementation."
	}
	return "Medium: Error rate increase detected. Review recent AI-generated code for error handling completeness."
}

func (d *AIAnomalyDetector) getConsistencyRecommendation(current, baseline float64) string {
	decrease := (baseline - current) / baseline * 100
	if decrease > 20 {
		return "Critical: Business logic consistency has significantly degraded. Review AI-generated business logic for rule violations."
	} else if decrease > 10 {
		return "High: Business logic consistency issues detected. Audit AI-generated code for business rule compliance."
	}
	return "Medium: Minor consistency issues detected. Review AI-generated business logic patterns."
}

// UpdateBaselines updates baseline metrics based on recent performance
func (d *AIAnomalyDetector) UpdateBaselines() {
	if len(d.historicalData.QueryTimes) >= 50 {
		// Use recent stable performance as new baseline
		recentData := d.historicalData.QueryTimes[len(d.historicalData.QueryTimes)-50:]
		d.baselines.QueryExecutionTime = d.calculateAverage(recentData)
	}
	
	if len(d.historicalData.ErrorRates) >= 50 {
		recentData := d.historicalData.ErrorRates[len(d.historicalData.ErrorRates)-50:]
		d.baselines.ErrorRate = d.calculateAverage(recentData)
	}
	
	if len(d.historicalData.ConsistencyScores) >= 50 {
		recentData := d.historicalData.ConsistencyScores[len(d.historicalData.ConsistencyScores)-50:]
		d.baselines.ConsistencyScore = d.calculateAverage(recentData)
	}
	
	if len(d.historicalData.PerformanceScores) >= 50 {
		recentData := d.historicalData.PerformanceScores[len(d.historicalData.PerformanceScores)-50:]
		d.baselines.PerformanceScore = d.calculateAverage(recentData)
	}
	
	d.baselines.LastUpdated = time.Now()
	d.baselines.SampleSize = 50
}

// GetBaselines returns current baseline metrics
func (d *AIAnomalyDetector) GetBaselines() *AICodeBaselines {
	return d.baselines
}

// GetHistoricalData returns historical metrics data
func (d *AIAnomalyDetector) GetHistoricalData() *HistoricalMetrics {
	return d.historicalData
}