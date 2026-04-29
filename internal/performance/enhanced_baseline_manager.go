package performance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"time"
)

// EnhancedBaselineManager provides advanced performance baseline management with statistical analysis
type EnhancedBaselineManager struct {
	db                *sql.DB
	statisticalEngine *StatisticalAnalysisEngine
	trendAnalyzer     *TrendAnalyzer
	capacityPlanner   *CapacityPlanner
	regressionEngine  *RegressionDetectionEngine
}

// StatisticalAnalysisEngine handles statistical analysis of performance metrics
type StatisticalAnalysisEngine struct {
	confidenceLevel float64 // Default 95%
	sampleSize      int     // Minimum sample size for statistical significance
}

// TrendAnalyzer provides performance trend analysis and forecasting
type TrendAnalyzer struct {
	forecastPeriods int     // Number of periods to forecast
	seasonality     bool    // Whether to account for seasonal patterns
	smoothingFactor float64 // Exponential smoothing factor
}

// CapacityPlanner provides capacity planning recommendations
type CapacityPlanner struct {
	resourceThresholds map[string]float64 // Resource utilization thresholds
	growthFactors      map[string]float64 // Expected growth factors
}

// RegressionDetectionEngine provides intelligent regression detection
type RegressionDetectionEngine struct {
	confidenceScoring bool                   // Whether to use confidence scoring
	adaptiveThresholds bool                  // Whether to use adaptive thresholds
	metricWeights     map[string]float64     // Weights for different metrics
	alertingRules     []AlertingRule         // Custom alerting rules
}

// EnhancedPerformanceBaseline extends the basic baseline with statistical data
type EnhancedPerformanceBaseline struct {
	PerformanceBaseline
	StatisticalData StatisticalMetrics `json:"statistical_data"`
	TrendData       TrendMetrics       `json:"trend_data"`
	CapacityData    CapacityMetrics    `json:"capacity_data"`
}

// StatisticalMetrics contains statistical analysis of performance data
type StatisticalMetrics struct {
	ConfidenceIntervals map[string]ConfidenceInterval `json:"confidence_intervals"`
	Outliers           []OutlierPoint                `json:"outliers"`
	Distribution       map[string]DistributionStats  `json:"distribution"`
	Correlation        map[string]float64            `json:"correlation"`
	Seasonality        map[string]SeasonalPattern    `json:"seasonality"`
}

// ConfidenceInterval represents statistical confidence interval
type ConfidenceInterval struct {
	Lower      float64 `json:"lower"`
	Upper      float64 `json:"upper"`
	Confidence float64 `json:"confidence"`
}

// OutlierPoint represents an outlier in the data
type OutlierPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	ZScore    float64   `json:"z_score"`
	Metric    string    `json:"metric"`
}

// DistributionStats contains distribution statistics
type DistributionStats struct {
	Skewness float64 `json:"skewness"`
	Kurtosis float64 `json:"kurtosis"`
	Mode     float64 `json:"mode"`
	Variance float64 `json:"variance"`
}

// SeasonalPattern represents seasonal patterns in metrics
type SeasonalPattern struct {
	Period    string             `json:"period"` // daily, weekly, monthly
	Amplitude float64            `json:"amplitude"`
	Phase     float64            `json:"phase"`
	Patterns  map[string]float64 `json:"patterns"` // hour/day/month -> factor
}

// TrendMetrics contains trend analysis and forecasting data
type TrendMetrics struct {
	TrendDirection string                 `json:"trend_direction"` // increasing, decreasing, stable
	TrendStrength  float64                `json:"trend_strength"`  // 0-1
	Forecast       map[string]ForecastData `json:"forecast"`
	ChangePoints   []ChangePoint          `json:"change_points"`
}

// ForecastData contains forecasted values
type ForecastData struct {
	PredictedValue    float64           `json:"predicted_value"`
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
	Timestamp         time.Time         `json:"timestamp"`
}

// ChangePoint represents a significant change in trend
type ChangePoint struct {
	Timestamp   time.Time `json:"timestamp"`
	Metric      string    `json:"metric"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
}

// CapacityMetrics contains capacity planning data
type CapacityMetrics struct {
	ResourceUtilization map[string]float64            `json:"resource_utilization"`
	ScalingRecommendations []ScalingRecommendation    `json:"scaling_recommendations"`
	CapacityForecast    map[string]CapacityForecast   `json:"capacity_forecast"`
	BottleneckAnalysis  []BottleneckAnalysis          `json:"bottleneck_analysis"`
}

// ScalingRecommendation provides scaling recommendations
type ScalingRecommendation struct {
	Resource    string    `json:"resource"`
	Action      string    `json:"action"` // scale_up, scale_down, optimize
	Urgency     string    `json:"urgency"` // low, medium, high, critical
	Reason      string    `json:"reason"`
	Timeline    string    `json:"timeline"`
	Impact      string    `json:"impact"`
	Confidence  float64   `json:"confidence"`
}

// CapacityForecast provides capacity forecasting
type CapacityForecast struct {
	Resource           string    `json:"resource"`
	CurrentUtilization float64   `json:"current_utilization"`
	ForecastedPeak     float64   `json:"forecasted_peak"`
	TimeToCapacity     time.Time `json:"time_to_capacity"`
	RecommendedAction  string    `json:"recommended_action"`
}

// BottleneckAnalysis identifies performance bottlenecks
type BottleneckAnalysis struct {
	Component   string  `json:"component"`
	Severity    string  `json:"severity"`
	Impact      float64 `json:"impact"` // Performance impact percentage
	Description string  `json:"description"`
	Resolution  string  `json:"resolution"`
}

// EnhancedRegressionResult extends basic regression result with confidence scoring
type EnhancedRegressionResult struct {
	RegressionResult
	ConfidenceScore    float64                    `json:"confidence_score"`
	StatisticalSignificance bool                  `json:"statistical_significance"`
	RootCauseAnalysis  []RootCauseHypothesis     `json:"root_cause_analysis"`
	OptimizationSuggestions []OptimizationSuggestion `json:"optimization_suggestions"`
	AlertingDecision   AlertingDecision          `json:"alerting_decision"`
}

// RootCauseHypothesis represents a potential root cause
type RootCauseHypothesis struct {
	Hypothesis  string  `json:"hypothesis"`
	Confidence  float64 `json:"confidence"`
	Evidence    []string `json:"evidence"`
	Investigation string `json:"investigation"`
}

// OptimizationSuggestion provides performance optimization suggestions
type OptimizationSuggestion struct {
	Category    string  `json:"category"` // database, cache, application, infrastructure
	Suggestion  string  `json:"suggestion"`
	Priority    string  `json:"priority"` // low, medium, high, critical
	Effort      string  `json:"effort"`   // low, medium, high
	Impact      string  `json:"impact"`   // low, medium, high
	Confidence  float64 `json:"confidence"`
}

// AlertingDecision determines whether to send alerts
type AlertingDecision struct {
	ShouldAlert     bool     `json:"should_alert"`
	AlertLevel      string   `json:"alert_level"` // info, warning, critical
	Reason          string   `json:"reason"`
	SuppressUntil   *time.Time `json:"suppress_until,omitempty"`
	EscalationPath  []string `json:"escalation_path"`
}

// AlertingRule defines custom alerting rules
type AlertingRule struct {
	Name        string                 `json:"name"`
	Condition   string                 `json:"condition"`
	Threshold   float64                `json:"threshold"`
	Duration    time.Duration          `json:"duration"`
	Severity    string                 `json:"severity"`
	Suppression time.Duration          `json:"suppression"`
	Actions     []AlertAction          `json:"actions"`
}

// AlertAction defines actions to take when alert is triggered
type AlertAction struct {
	Type       string            `json:"type"` // email, slack, webhook, auto_scale
	Target     string            `json:"target"`
	Parameters map[string]string `json:"parameters"`
}

// NewEnhancedBaselineManager creates a new enhanced baseline manager
func NewEnhancedBaselineManager(db *sql.DB) *EnhancedBaselineManager {
	return &EnhancedBaselineManager{
		db: db,
		statisticalEngine: &StatisticalAnalysisEngine{
			confidenceLevel: 0.95,
			sampleSize:      30,
		},
		trendAnalyzer: &TrendAnalyzer{
			forecastPeriods: 7, // 7 days forecast
			seasonality:     true,
			smoothingFactor: 0.3,
		},
		capacityPlanner: &CapacityPlanner{
			resourceThresholds: map[string]float64{
				"cpu":    0.80, // 80% CPU threshold
				"memory": 0.85, // 85% memory threshold
				"disk":   0.90, // 90% disk threshold
				"db_connections": 0.75, // 75% DB connection threshold
			},
			growthFactors: map[string]float64{
				"traffic": 1.15, // 15% monthly growth
				"data":    1.10, // 10% monthly growth
			},
		},
		regressionEngine: &RegressionDetectionEngine{
			confidenceScoring:  true,
			adaptiveThresholds: true,
			metricWeights: map[string]float64{
				"http_req_duration":         1.0,
				"article_creation_duration": 0.8,
				"database_query_duration":   0.9,
				"cache_hit_rate":           0.7,
				"error_rate":               1.2, // Higher weight for errors
			},
			alertingRules: []AlertingRule{
				{
					Name:        "Critical Response Time Regression",
					Condition:   "response_time_increase",
					Threshold:   50.0, // 50% increase
					Duration:    5 * time.Minute,
					Severity:    "critical",
					Suppression: 30 * time.Minute,
				},
				{
					Name:        "Error Rate Spike",
					Condition:   "error_rate_increase",
					Threshold:   100.0, // 100% increase (doubling)
					Duration:    2 * time.Minute,
					Severity:    "critical",
					Suppression: 15 * time.Minute,
				},
			},
		},
	}
}

// EstablishEnhancedBaseline creates a comprehensive baseline with statistical analysis
func (ebm *EnhancedBaselineManager) EstablishEnhancedBaseline(testName, version, environment string, rawMetrics []map[string]MetricData) (*EnhancedPerformanceBaseline, error) {
	if len(rawMetrics) < ebm.statisticalEngine.sampleSize {
		return nil, fmt.Errorf("insufficient data points: need at least %d, got %d", ebm.statisticalEngine.sampleSize, len(rawMetrics))
	}

	// Calculate basic statistics
	aggregatedMetrics := ebm.aggregateMetrics(rawMetrics)

	// Perform statistical analysis
	statisticalData := ebm.performStatisticalAnalysis(rawMetrics)

	// Perform trend analysis
	trendData := ebm.performTrendAnalysis(rawMetrics)

	// Perform capacity analysis
	capacityData := ebm.performCapacityAnalysis(rawMetrics)

	baseline := &EnhancedPerformanceBaseline{
		PerformanceBaseline: PerformanceBaseline{
			TestName:    testName,
			Version:     version,
			Environment: environment,
			Metrics:     aggregatedMetrics,
			CreatedAt:   time.Now(),
			IsActive:    true,
		},
		StatisticalData: statisticalData,
		TrendData:       trendData,
		CapacityData:    capacityData,
	}

	// Store the enhanced baseline
	err := ebm.storeEnhancedBaseline(baseline)
	if err != nil {
		return nil, fmt.Errorf("failed to store enhanced baseline: %w", err)
	}

	log.Printf("Established enhanced baseline for %s version %s with %d data points", testName, version, len(rawMetrics))
	return baseline, nil
}

// aggregateMetrics calculates aggregate statistics from raw metrics
func (ebm *EnhancedBaselineManager) aggregateMetrics(rawMetrics []map[string]MetricData) map[string]MetricData {
	aggregated := make(map[string]MetricData)
	
	// Collect all metric names
	metricNames := make(map[string]bool)
	for _, metrics := range rawMetrics {
		for name := range metrics {
			metricNames[name] = true
		}
	}

	// Calculate statistics for each metric
	for metricName := range metricNames {
		var values []float64
		var unit string
		var threshold float64

		for _, metrics := range rawMetrics {
			if metric, exists := metrics[metricName]; exists {
				values = append(values, metric.P95) // Use P95 as primary value
				unit = metric.Unit
				threshold = metric.Threshold
			}
		}

		if len(values) > 0 {
			sort.Float64s(values)
			
			aggregated[metricName] = MetricData{
				Mean:      ebm.calculateMean(values),
				P95:       ebm.calculatePercentile(values, 95),
				P99:       ebm.calculatePercentile(values, 99),
				Min:       values[0],
				Max:       values[len(values)-1],
				Count:     int64(len(values)),
				StdDev:    ebm.calculateStdDev(values),
				Unit:      unit,
				Threshold: threshold,
			}
		}
	}

	return aggregated
}

// performStatisticalAnalysis performs comprehensive statistical analysis
func (ebm *EnhancedBaselineManager) performStatisticalAnalysis(rawMetrics []map[string]MetricData) StatisticalMetrics {
	confidenceIntervals := make(map[string]ConfidenceInterval)
	outliers := []OutlierPoint{}
	distribution := make(map[string]DistributionStats)
	correlation := make(map[string]float64)
	seasonality := make(map[string]SeasonalPattern)

	// Collect all metric names
	metricNames := make(map[string]bool)
	for _, metrics := range rawMetrics {
		for name := range metrics {
			metricNames[name] = true
		}
	}

	// Calculate statistics for each metric
	for metricName := range metricNames {
		var values []float64
		var timestamps []time.Time

		for i, metrics := range rawMetrics {
			if metric, exists := metrics[metricName]; exists {
				values = append(values, metric.P95)
				timestamps = append(timestamps, time.Now().Add(-time.Duration(len(rawMetrics)-i)*time.Minute))
			}
		}

		if len(values) >= ebm.statisticalEngine.sampleSize {
			// Calculate confidence interval
			confidenceIntervals[metricName] = ebm.calculateConfidenceInterval(values, ebm.statisticalEngine.confidenceLevel)

			// Detect outliers
			outliers = append(outliers, ebm.detectOutliers(values, timestamps, metricName)...)

			// Calculate distribution statistics
			distribution[metricName] = ebm.calculateDistributionStats(values)

			// Calculate correlation with other metrics (simplified)
			correlation[metricName] = ebm.calculateAutoCorrelation(values)

			// Detect seasonal patterns
			seasonality[metricName] = ebm.detectSeasonalPatterns(values, timestamps)
		}
	}

	return StatisticalMetrics{
		ConfidenceIntervals: confidenceIntervals,
		Outliers:           outliers,
		Distribution:       distribution,
		Correlation:        correlation,
		Seasonality:        seasonality,
	}
}

// performTrendAnalysis analyzes trends and creates forecasts
func (ebm *EnhancedBaselineManager) performTrendAnalysis(rawMetrics []map[string]MetricData) TrendMetrics {
	forecast := make(map[string]ForecastData)
	changePoints := []ChangePoint{}

	// Collect all metric names
	metricNames := make(map[string]bool)
	for _, metrics := range rawMetrics {
		for name := range metrics {
			metricNames[name] = true
		}
	}

	// Analyze trends for each metric
	for metricName := range metricNames {
		var values []float64
		var timestamps []time.Time

		for i, metrics := range rawMetrics {
			if metric, exists := metrics[metricName]; exists {
				values = append(values, metric.P95)
				timestamps = append(timestamps, time.Now().Add(-time.Duration(len(rawMetrics)-i)*time.Minute))
			}
		}

		if len(values) >= 10 { // Need at least 10 points for trend analysis
			// Detect change points
			changePoints = append(changePoints, ebm.detectChangePoints(values, timestamps, metricName)...)

			// Create forecast
			forecast[metricName] = ebm.createForecast(values, timestamps)
		}
	}

	// Determine overall trend direction and strength
	trendDirection, trendStrength := ebm.calculateOverallTrend(rawMetrics)

	return TrendMetrics{
		TrendDirection: trendDirection,
		TrendStrength:  trendStrength,
		Forecast:       forecast,
		ChangePoints:   changePoints,
	}
}

// performCapacityAnalysis analyzes capacity and provides recommendations
func (ebm *EnhancedBaselineManager) performCapacityAnalysis(rawMetrics []map[string]MetricData) CapacityMetrics {
	resourceUtilization := make(map[string]float64)
	scalingRecommendations := []ScalingRecommendation{}
	capacityForecast := make(map[string]CapacityForecast)
	bottleneckAnalysis := []BottleneckAnalysis{}

	// Calculate current resource utilization
	if len(rawMetrics) > 0 {
		latestMetrics := rawMetrics[len(rawMetrics)-1]
		
		// Extract resource utilization metrics
		if cpuMetric, exists := latestMetrics["cpu_usage_percent"]; exists {
			resourceUtilization["cpu"] = cpuMetric.Mean / 100.0
		}
		if memMetric, exists := latestMetrics["memory_usage_mb"]; exists {
			// Assume 4GB total memory for calculation
			resourceUtilization["memory"] = memMetric.Mean / 4096.0
		}
		if dbMetric, exists := latestMetrics["db_connections_active"]; exists {
			// Assume 100 max connections
			resourceUtilization["db_connections"] = dbMetric.Mean / 100.0
		}
	}

	// Generate scaling recommendations
	for resource, utilization := range resourceUtilization {
		threshold := ebm.capacityPlanner.resourceThresholds[resource]
		
		if utilization > threshold {
			urgency := "medium"
			if utilization > threshold*1.2 {
				urgency = "high"
			}
			if utilization > threshold*1.5 {
				urgency = "critical"
			}

			recommendation := ScalingRecommendation{
				Resource:   resource,
				Action:     "scale_up",
				Urgency:    urgency,
				Reason:     fmt.Sprintf("Current utilization %.1f%% exceeds threshold %.1f%%", utilization*100, threshold*100),
				Timeline:   ebm.calculateScalingTimeline(urgency),
				Impact:     ebm.calculateScalingImpact(resource, utilization),
				Confidence: ebm.calculateRecommendationConfidence(resource, utilization),
			}
			scalingRecommendations = append(scalingRecommendations, recommendation)
		}
	}

	// Generate capacity forecasts
	for resource, utilization := range resourceUtilization {
		growthFactor := ebm.capacityPlanner.growthFactors["traffic"] // Default growth factor
		
		forecast := CapacityForecast{
			Resource:           resource,
			CurrentUtilization: utilization,
			ForecastedPeak:     utilization * growthFactor,
			TimeToCapacity:     ebm.calculateTimeToCapacity(utilization, growthFactor),
			RecommendedAction:  ebm.getRecommendedAction(utilization, growthFactor),
		}
		capacityForecast[resource] = forecast
	}

	// Identify bottlenecks
	bottleneckAnalysis = ebm.identifyBottlenecks(rawMetrics)

	return CapacityMetrics{
		ResourceUtilization:    resourceUtilization,
		ScalingRecommendations: scalingRecommendations,
		CapacityForecast:       capacityForecast,
		BottleneckAnalysis:     bottleneckAnalysis,
	}
}

// Helper functions for statistical calculations

func (ebm *EnhancedBaselineManager) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (ebm *EnhancedBaselineManager) calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	index := (percentile / 100.0) * float64(len(values)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	
	if lower == upper {
		return values[lower]
	}
	
	weight := index - float64(lower)
	return values[lower]*(1-weight) + values[upper]*weight
}

func (ebm *EnhancedBaselineManager) calculateStdDev(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	
	mean := ebm.calculateMean(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	
	variance := sumSquares / float64(len(values)-1)
	return math.Sqrt(variance)
}

func (ebm *EnhancedBaselineManager) calculateConfidenceInterval(values []float64, confidence float64) ConfidenceInterval {
	if len(values) < 2 {
		return ConfidenceInterval{Lower: 0, Upper: 0, Confidence: confidence}
	}
	
	mean := ebm.calculateMean(values)
	stdDev := ebm.calculateStdDev(values)
	
	// Use t-distribution for small samples
	var tValue float64
	if len(values) >= 30 {
		// Use normal distribution approximation for large samples
		if confidence == 0.95 {
			tValue = 1.96
		} else if confidence == 0.99 {
			tValue = 2.58
		} else {
			tValue = 1.96 // Default to 95%
		}
	} else {
		// Simplified t-value for small samples (should use proper t-table)
		if confidence == 0.95 {
			tValue = 2.0
		} else if confidence == 0.99 {
			tValue = 2.6
		} else {
			tValue = 2.0
		}
	}
	
	margin := tValue * (stdDev / math.Sqrt(float64(len(values))))
	
	return ConfidenceInterval{
		Lower:      mean - margin,
		Upper:      mean + margin,
		Confidence: confidence,
	}
}

func (ebm *EnhancedBaselineManager) detectOutliers(values []float64, timestamps []time.Time, metricName string) []OutlierPoint {
	if len(values) < 3 {
		return []OutlierPoint{}
	}
	
	mean := ebm.calculateMean(values)
	stdDev := ebm.calculateStdDev(values)
	outliers := []OutlierPoint{}
	
	for i, value := range values {
		zScore := math.Abs((value - mean) / stdDev)
		if zScore > 2.5 { // Values more than 2.5 standard deviations away
			outlier := OutlierPoint{
				Timestamp: timestamps[i],
				Value:     value,
				ZScore:    zScore,
				Metric:    metricName,
			}
			outliers = append(outliers, outlier)
		}
	}
	
	return outliers
}

func (ebm *EnhancedBaselineManager) calculateDistributionStats(values []float64) DistributionStats {
	if len(values) < 3 {
		return DistributionStats{}
	}
	
	mean := ebm.calculateMean(values)
	variance := 0.0
	skewness := 0.0
	kurtosis := 0.0
	
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
		skewness += diff * diff * diff
		kurtosis += diff * diff * diff * diff
	}
	
	n := float64(len(values))
	variance /= (n - 1)
	stdDev := math.Sqrt(variance)
	
	if stdDev > 0 {
		skewness = (skewness / n) / math.Pow(stdDev, 3)
		kurtosis = (kurtosis / n) / math.Pow(stdDev, 4) - 3 // Excess kurtosis
	}
	
	// Calculate mode (simplified - most frequent value in binned data)
	mode := ebm.calculateMode(values)
	
	return DistributionStats{
		Skewness: skewness,
		Kurtosis: kurtosis,
		Mode:     mode,
		Variance: variance,
	}
}

func (ebm *EnhancedBaselineManager) calculateMode(values []float64) float64 {
	// Simplified mode calculation - return median for continuous data
	sort.Float64s(values)
	n := len(values)
	if n%2 == 0 {
		return (values[n/2-1] + values[n/2]) / 2
	}
	return values[n/2]
}

func (ebm *EnhancedBaselineManager) calculateAutoCorrelation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	
	// Calculate lag-1 autocorrelation
	mean := ebm.calculateMean(values)
	numerator := 0.0
	denominator := 0.0
	
	for i := 0; i < len(values)-1; i++ {
		numerator += (values[i] - mean) * (values[i+1] - mean)
	}
	
	for _, v := range values {
		denominator += (v - mean) * (v - mean)
	}
	
	if denominator == 0 {
		return 0
	}
	
	return numerator / denominator
}

func (ebm *EnhancedBaselineManager) detectSeasonalPatterns(values []float64, timestamps []time.Time) SeasonalPattern {
	// Simplified seasonal pattern detection
	// In a real implementation, this would use more sophisticated algorithms like FFT
	
	if len(values) < 24 { // Need at least 24 data points
		return SeasonalPattern{Period: "none", Amplitude: 0, Phase: 0, Patterns: make(map[string]float64)}
	}
	
	// Detect hourly patterns (simplified)
	hourlyPatterns := make(map[string]float64)
	hourlyValues := make(map[int][]float64)
	
	for i, timestamp := range timestamps {
		hour := timestamp.Hour()
		hourlyValues[hour] = append(hourlyValues[hour], values[i])
	}
	
	overallMean := ebm.calculateMean(values)
	maxAmplitude := 0.0
	
	for hour, hourValues := range hourlyValues {
		if len(hourValues) > 0 {
			hourMean := ebm.calculateMean(hourValues)
			factor := hourMean / overallMean
			hourlyPatterns[fmt.Sprintf("%02d", hour)] = factor
			
			amplitude := math.Abs(factor - 1.0)
			if amplitude > maxAmplitude {
				maxAmplitude = amplitude
			}
		}
	}
	
	return SeasonalPattern{
		Period:    "daily",
		Amplitude: maxAmplitude,
		Phase:     0, // Simplified
		Patterns:  hourlyPatterns,
	}
}

func (ebm *EnhancedBaselineManager) detectChangePoints(values []float64, timestamps []time.Time, metricName string) []ChangePoint {
	changePoints := []ChangePoint{}
	
	if len(values) < 10 {
		return changePoints
	}
	
	// Simple change point detection using moving averages
	windowSize := 5
	threshold := 0.2 // 20% change threshold
	
	for i := windowSize; i < len(values)-windowSize; i++ {
		beforeWindow := values[i-windowSize : i]
		afterWindow := values[i : i+windowSize]
		
		beforeMean := ebm.calculateMean(beforeWindow)
		afterMean := ebm.calculateMean(afterWindow)
		
		if beforeMean > 0 {
			changePercent := math.Abs((afterMean - beforeMean) / beforeMean)
			
			if changePercent > threshold {
				severity := "low"
				if changePercent > 0.5 {
					severity = "high"
				} else if changePercent > 0.3 {
					severity = "medium"
				}
				
				changePoint := ChangePoint{
					Timestamp:   timestamps[i],
					Metric:      metricName,
					Severity:    severity,
					Description: fmt.Sprintf("%.1f%% change detected", changePercent*100),
				}
				changePoints = append(changePoints, changePoint)
			}
		}
	}
	
	return changePoints
}

func (ebm *EnhancedBaselineManager) createForecast(values []float64, timestamps []time.Time) ForecastData {
	if len(values) < 3 {
		return ForecastData{}
	}
	
	// Simple exponential smoothing forecast
	alpha := ebm.trendAnalyzer.smoothingFactor
	
	// Calculate smoothed values
	smoothed := make([]float64, len(values))
	smoothed[0] = values[0]
	
	for i := 1; i < len(values); i++ {
		smoothed[i] = alpha*values[i] + (1-alpha)*smoothed[i-1]
	}
	
	// Forecast next value
	predictedValue := smoothed[len(smoothed)-1]
	
	// Calculate prediction interval (simplified)
	recentValues := values[len(values)-5:] // Last 5 values
	stdDev := ebm.calculateStdDev(recentValues)
	
	confidenceInterval := ConfidenceInterval{
		Lower:      predictedValue - 1.96*stdDev,
		Upper:      predictedValue + 1.96*stdDev,
		Confidence: 0.95,
	}
	
	forecastTime := timestamps[len(timestamps)-1].Add(time.Hour) // Forecast 1 hour ahead
	
	return ForecastData{
		PredictedValue:     predictedValue,
		ConfidenceInterval: confidenceInterval,
		Timestamp:          forecastTime,
	}
}

func (ebm *EnhancedBaselineManager) calculateOverallTrend(rawMetrics []map[string]MetricData) (string, float64) {
	if len(rawMetrics) < 3 {
		return "stable", 0.0
	}
	
	// Calculate trend for key metrics
	keyMetrics := []string{"http_req_duration", "article_creation_duration", "database_query_duration"}
	trendScores := []float64{}
	
	for _, metricName := range keyMetrics {
		var values []float64
		for _, metrics := range rawMetrics {
			if metric, exists := metrics[metricName]; exists {
				values = append(values, metric.P95)
			}
		}
		
		if len(values) >= 3 {
			// Simple linear trend calculation
			n := float64(len(values))
			sumX := n * (n - 1) / 2
			sumY := ebm.calculateMean(values) * n
			sumXY := 0.0
			sumX2 := 0.0
			
			for i, y := range values {
				x := float64(i)
				sumXY += x * y
				sumX2 += x * x
			}
			
			slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
			trendScores = append(trendScores, slope)
		}
	}
	
	if len(trendScores) == 0 {
		return "stable", 0.0
	}
	
	avgTrend := ebm.calculateMean(trendScores)
	trendStrength := math.Abs(avgTrend)
	
	direction := "stable"
	if avgTrend > 0.1 {
		direction = "increasing"
	} else if avgTrend < -0.1 {
		direction = "decreasing"
	}
	
	return direction, trendStrength
}

// Capacity planning helper functions

func (ebm *EnhancedBaselineManager) calculateScalingTimeline(urgency string) string {
	switch urgency {
	case "critical":
		return "immediate"
	case "high":
		return "within 24 hours"
	case "medium":
		return "within 1 week"
	default:
		return "within 1 month"
	}
}

func (ebm *EnhancedBaselineManager) calculateScalingImpact(resource string, utilization float64) string {
	if utilization > 0.9 {
		return "high - performance degradation likely"
	} else if utilization > 0.8 {
		return "medium - performance may be affected"
	}
	return "low - preventive scaling"
}

func (ebm *EnhancedBaselineManager) calculateRecommendationConfidence(resource string, utilization float64) float64 {
	// Higher confidence for higher utilization
	if utilization > 0.9 {
		return 0.95
	} else if utilization > 0.8 {
		return 0.85
	}
	return 0.75
}

func (ebm *EnhancedBaselineManager) calculateTimeToCapacity(utilization, growthFactor float64) time.Time {
	if utilization >= 1.0 {
		return time.Now() // Already at capacity
	}
	
	// Simple calculation: time = log(1/utilization) / log(growthFactor)
	// Assuming monthly growth
	monthsToCapacity := math.Log(1.0/utilization) / math.Log(growthFactor)
	
	return time.Now().Add(time.Duration(monthsToCapacity * 30 * 24) * time.Hour)
}

func (ebm *EnhancedBaselineManager) getRecommendedAction(utilization, growthFactor float64) string {
	timeToCapacity := ebm.calculateTimeToCapacity(utilization, growthFactor)
	daysToCapacity := time.Until(timeToCapacity).Hours() / 24
	
	if daysToCapacity < 30 {
		return "scale_up_immediately"
	} else if daysToCapacity < 90 {
		return "plan_scaling"
	}
	return "monitor"
}

func (ebm *EnhancedBaselineManager) identifyBottlenecks(rawMetrics []map[string]MetricData) []BottleneckAnalysis {
	bottlenecks := []BottleneckAnalysis{}
	
	if len(rawMetrics) == 0 {
		return bottlenecks
	}
	
	latestMetrics := rawMetrics[len(rawMetrics)-1]
	
	// Check database bottlenecks
	if dbMetric, exists := latestMetrics["database_query_duration"]; exists {
		if dbMetric.P95 > 50 { // 50ms threshold
			bottleneck := BottleneckAnalysis{
				Component:   "database",
				Severity:    ebm.calculateBottleneckSeverity(dbMetric.P95, 50),
				Impact:      (dbMetric.P95 - 50) / 50 * 100, // Percentage impact
				Description: fmt.Sprintf("Database queries averaging %.1fms (threshold: 50ms)", dbMetric.P95),
				Resolution:  "Optimize queries, add indexes, or scale database",
			}
			bottlenecks = append(bottlenecks, bottleneck)
		}
	}
	
	// Check API response time bottlenecks
	if apiMetric, exists := latestMetrics["http_req_duration"]; exists {
		if apiMetric.P95 > 200 { // 200ms threshold
			bottleneck := BottleneckAnalysis{
				Component:   "api",
				Severity:    ebm.calculateBottleneckSeverity(apiMetric.P95, 200),
				Impact:      (apiMetric.P95 - 200) / 200 * 100,
				Description: fmt.Sprintf("API responses averaging %.1fms (threshold: 200ms)", apiMetric.P95),
				Resolution:  "Optimize application code, improve caching, or scale horizontally",
			}
			bottlenecks = append(bottlenecks, bottleneck)
		}
	}
	
	// Check cache performance bottlenecks
	if cacheMetric, exists := latestMetrics["cache_hit_rate"]; exists {
		if cacheMetric.Mean < 0.8 { // 80% threshold
			bottleneck := BottleneckAnalysis{
				Component:   "cache",
				Severity:    ebm.calculateBottleneckSeverity(0.8-cacheMetric.Mean, 0.2),
				Impact:      (0.8 - cacheMetric.Mean) / 0.8 * 100,
				Description: fmt.Sprintf("Cache hit rate %.1f%% (threshold: 80%%)", cacheMetric.Mean*100),
				Resolution:  "Optimize cache configuration, increase cache size, or improve cache keys",
			}
			bottlenecks = append(bottlenecks, bottleneck)
		}
	}
	
	return bottlenecks
}

func (ebm *EnhancedBaselineManager) calculateBottleneckSeverity(value, threshold float64) string {
	ratio := value / threshold
	if ratio > 3.0 {
		return "critical"
	} else if ratio > 2.0 {
		return "high"
	} else if ratio > 1.5 {
		return "medium"
	}
	return "low"
}

// storeEnhancedBaseline stores the enhanced baseline in the database
func (ebm *EnhancedBaselineManager) storeEnhancedBaseline(baseline *EnhancedPerformanceBaseline) error {
	// Deactivate previous baselines
	_, err := ebm.db.Exec(`
		UPDATE performance_baselines 
		SET is_active = false 
		WHERE test_name = $1 AND environment = $2 AND is_active = true`,
		baseline.TestName, baseline.Environment)
	if err != nil {
		return fmt.Errorf("failed to deactivate previous baselines: %w", err)
	}

	// Prepare enhanced data for storage
	enhancedData := map[string]interface{}{
		"statistical_data": baseline.StatisticalData,
		"trend_data":       baseline.TrendData,
		"capacity_data":    baseline.CapacityData,
	}
	
	enhancedJSON, err := json.Marshal(enhancedData)
	if err != nil {
		return fmt.Errorf("failed to marshal enhanced data: %w", err)
	}

	metricsJSON, err := json.Marshal(baseline.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// Store enhanced baseline
	err = ebm.db.QueryRow(`
		INSERT INTO performance_baselines (test_name, version, metrics, environment, is_active, created_at, enhanced_data)
		VALUES ($1, $2, $3, $4, true, $5, $6)
		RETURNING id`,
		baseline.TestName, baseline.Version, metricsJSON, baseline.Environment, baseline.CreatedAt, enhancedJSON).
		Scan(&baseline.ID)

	if err != nil {
		return fmt.Errorf("failed to store enhanced baseline: %w", err)
	}

	return nil
}

// GetActiveBaseline retrieves the active baseline for a test (delegates to base manager)
func (ebm *EnhancedBaselineManager) GetActiveBaseline(testName, environment string) (*PerformanceBaseline, error) {
	baseManager := NewBaselineManager(ebm.db)
	return baseManager.GetActiveBaseline(testName, environment)
}

// CompareWithBaseline compares current metrics with baseline and detects regressions (delegates to base manager)
func (ebm *EnhancedBaselineManager) CompareWithBaseline(testName, currentVersion, environment string, currentMetrics map[string]MetricData) (*RegressionResult, error) {
	baseManager := NewBaselineManager(ebm.db)
	return baseManager.CompareWithBaseline(testName, currentVersion, environment, currentMetrics)
}