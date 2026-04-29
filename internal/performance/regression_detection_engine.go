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

// IntelligentRegressionDetector provides advanced regression detection with confidence scoring
type IntelligentRegressionDetector struct {
	db                    *sql.DB
	baselineManager       *EnhancedBaselineManager
	confidenceThreshold   float64
	adaptiveThresholds    map[string]AdaptiveThreshold
	alertSuppressionRules map[string]time.Time
	optimizationEngine    *OptimizationEngine
}

// AdaptiveThreshold represents a threshold that adapts based on historical data
type AdaptiveThreshold struct {
	BaseThreshold    float64   `json:"base_threshold"`
	CurrentThreshold float64   `json:"current_threshold"`
	Confidence       float64   `json:"confidence"`
	LastUpdated      time.Time `json:"last_updated"`
	HistoricalData   []float64 `json:"historical_data"`
}

// OptimizationEngine provides performance optimization suggestions
type OptimizationEngine struct {
	knowledgeBase map[string][]OptimizationRule
	patterns      map[string]PerformancePattern
}

// OptimizationRule defines rules for optimization suggestions
type OptimizationRule struct {
	Condition   string  `json:"condition"`
	Suggestion  string  `json:"suggestion"`
	Category    string  `json:"category"`
	Priority    string  `json:"priority"`
	Effort      string  `json:"effort"`
	Impact      string  `json:"impact"`
	Confidence  float64 `json:"confidence"`
}

// PerformancePattern represents a known performance pattern
type PerformancePattern struct {
	Name        string            `json:"name"`
	Indicators  map[string]string `json:"indicators"`
	RootCause   string            `json:"root_cause"`
	Solutions   []string          `json:"solutions"`
	Confidence  float64           `json:"confidence"`
}

// RegressionAnalysisRequest contains parameters for regression analysis
type RegressionAnalysisRequest struct {
	TestName        string                 `json:"test_name"`
	CurrentVersion  string                 `json:"current_version"`
	Environment     string                 `json:"environment"`
	CurrentMetrics  map[string]MetricData  `json:"current_metrics"`
	AnalysisOptions AnalysisOptions        `json:"analysis_options"`
}

// AnalysisOptions configures the regression analysis
type AnalysisOptions struct {
	UseAdaptiveThresholds bool    `json:"use_adaptive_thresholds"`
	ConfidenceLevel       float64 `json:"confidence_level"`
	IncludeOptimizations  bool    `json:"include_optimizations"`
	SuppressAlerts        bool    `json:"suppress_alerts"`
	DetailedAnalysis      bool    `json:"detailed_analysis"`
}

// NewIntelligentRegressionDetector creates a new intelligent regression detector
func NewIntelligentRegressionDetector(db *sql.DB, baselineManager *EnhancedBaselineManager) *IntelligentRegressionDetector {
	detector := &IntelligentRegressionDetector{
		db:                    db,
		baselineManager:       baselineManager,
		confidenceThreshold:   0.80, // 80% confidence threshold
		adaptiveThresholds:    make(map[string]AdaptiveThreshold),
		alertSuppressionRules: make(map[string]time.Time),
		optimizationEngine:    NewOptimizationEngine(),
	}

	// Initialize adaptive thresholds
	detector.initializeAdaptiveThresholds()

	return detector
}

// DetectRegressions performs intelligent regression detection with confidence scoring
func (ird *IntelligentRegressionDetector) DetectRegressions(request RegressionAnalysisRequest) (*EnhancedRegressionResult, error) {
	// Get baseline for comparison
	baseline, err := ird.baselineManager.GetActiveBaseline(request.TestName, request.Environment)
	if err != nil {
		return nil, fmt.Errorf("failed to get baseline: %w", err)
	}

	// Perform basic regression analysis
	basicResult, err := ird.baselineManager.CompareWithBaseline(
		request.TestName, 
		request.CurrentVersion, 
		request.Environment, 
		request.CurrentMetrics,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to perform basic regression analysis: %w", err)
	}

	// Enhance the result with intelligent analysis
	enhancedResult := &EnhancedRegressionResult{
		RegressionResult: *basicResult,
	}

	// Calculate confidence score
	enhancedResult.ConfidenceScore = ird.calculateConfidenceScore(basicResult, request.CurrentMetrics, baseline.Metrics)

	// Determine statistical significance
	enhancedResult.StatisticalSignificance = ird.determineStatisticalSignificance(basicResult, request.AnalysisOptions.ConfidenceLevel)

	// Perform root cause analysis
	if request.AnalysisOptions.DetailedAnalysis {
		enhancedResult.RootCauseAnalysis = ird.performRootCauseAnalysis(basicResult, request.CurrentMetrics, baseline.Metrics)
	}

	// Generate optimization suggestions
	if request.AnalysisOptions.IncludeOptimizations {
		enhancedResult.OptimizationSuggestions = ird.generateOptimizationSuggestions(basicResult, request.CurrentMetrics)
	}

	// Make alerting decision
	enhancedResult.AlertingDecision = ird.makeAlertingDecision(enhancedResult, request.AnalysisOptions)

	// Update adaptive thresholds if enabled
	if request.AnalysisOptions.UseAdaptiveThresholds {
		ird.updateAdaptiveThresholds(request.CurrentMetrics, basicResult)
	}

	// Store the enhanced result
	err = ird.storeEnhancedRegressionResult(enhancedResult)
	if err != nil {
		log.Printf("Warning: failed to store enhanced regression result: %v", err)
	}

	return enhancedResult, nil
}

// calculateConfidenceScore calculates confidence in the regression detection
func (ird *IntelligentRegressionDetector) calculateConfidenceScore(result *RegressionResult, currentMetrics, baselineMetrics map[string]MetricData) float64 {
	if len(result.Regressions) == 0 {
		return 1.0 // High confidence in no regressions
	}

	totalConfidence := 0.0
	weightSum := 0.0

	for _, regression := range result.Regressions {
		// Base confidence on magnitude of change
		magnitudeConfidence := ird.calculateMagnitudeConfidence(regression.PercentChange, regression.Threshold)

		// Adjust for metric reliability
		reliabilityFactor := ird.getMetricReliability(regression.MetricName)

		// Adjust for sample size (if available)
		sampleSizeFactor := ird.calculateSampleSizeFactor(currentMetrics[regression.MetricName], baselineMetrics[regression.MetricName])

		// Adjust for historical consistency
		consistencyFactor := ird.calculateConsistencyFactor(regression.MetricName)

		// Calculate weighted confidence
		metricWeight := ird.baselineManager.regressionEngine.metricWeights[regression.MetricName]
		if metricWeight == 0 {
			metricWeight = 1.0
		}

		confidence := magnitudeConfidence * reliabilityFactor * sampleSizeFactor * consistencyFactor
		totalConfidence += confidence * metricWeight
		weightSum += metricWeight
	}

	if weightSum == 0 {
		return 0.5 // Neutral confidence
	}

	return math.Min(1.0, totalConfidence/weightSum)
}

// calculateMagnitudeConfidence calculates confidence based on the magnitude of change
func (ird *IntelligentRegressionDetector) calculateMagnitudeConfidence(percentChange, threshold float64) float64 {
	ratio := math.Abs(percentChange) / threshold
	
	if ratio >= 3.0 {
		return 0.95 // Very high confidence for large changes
	} else if ratio >= 2.0 {
		return 0.85 // High confidence
	} else if ratio >= 1.5 {
		return 0.75 // Medium confidence
	} else if ratio >= 1.0 {
		return 0.60 // Low confidence
	}
	
	return 0.40 // Very low confidence for changes below threshold
}

// getMetricReliability returns the reliability factor for a metric
func (ird *IntelligentRegressionDetector) getMetricReliability(metricName string) float64 {
	reliabilityMap := map[string]float64{
		"http_req_duration":         0.90, // High reliability
		"article_creation_duration": 0.85, // Good reliability
		"database_query_duration":   0.95, // Very high reliability
		"cache_hit_rate":           0.80, // Good reliability
		"error_rate":               0.95, // Very high reliability
		"memory_usage_mb":          0.75, // Medium reliability (can be noisy)
		"cpu_usage_percent":        0.70, // Medium reliability (can be noisy)
	}

	if reliability, exists := reliabilityMap[metricName]; exists {
		return reliability
	}
	return 0.75 // Default reliability
}

// calculateSampleSizeFactor adjusts confidence based on sample size
func (ird *IntelligentRegressionDetector) calculateSampleSizeFactor(currentMetric, baselineMetric MetricData) float64 {
	minSampleSize := int64(30) // Minimum for statistical significance
	
	currentSample := currentMetric.Count
	baselineSample := baselineMetric.Count
	
	minSample := currentSample
	if baselineSample < minSample {
		minSample = baselineSample
	}
	
	if minSample >= minSampleSize*2 {
		return 1.0 // Full confidence
	} else if minSample >= minSampleSize {
		return 0.85 // Good confidence
	} else if minSample >= minSampleSize/2 {
		return 0.70 // Medium confidence
	}
	
	return 0.50 // Low confidence for small samples
}

// calculateConsistencyFactor adjusts confidence based on historical consistency
func (ird *IntelligentRegressionDetector) calculateConsistencyFactor(metricName string) float64 {
	// Get recent regression history for this metric
	recentResults, err := ird.getRecentRegressionHistory(metricName, 10)
	if err != nil || len(recentResults) < 3 {
		return 0.80 // Default factor when no history
	}

	// Calculate consistency of regression detection
	regressionCount := 0
	for _, result := range recentResults {
		for _, regression := range result.Regressions {
			if regression.MetricName == metricName {
				regressionCount++
				break
			}
		}
	}

	consistencyRatio := float64(regressionCount) / float64(len(recentResults))
	
	if consistencyRatio > 0.7 {
		return 1.0 // High consistency - metric frequently regresses
	} else if consistencyRatio > 0.4 {
		return 0.85 // Medium consistency
	} else if consistencyRatio > 0.2 {
		return 0.70 // Low consistency
	}
	
	return 0.90 // Very low regression rate - high confidence when it does regress
}

// determineStatisticalSignificance determines if regressions are statistically significant
func (ird *IntelligentRegressionDetector) determineStatisticalSignificance(result *RegressionResult, confidenceLevel float64) bool {
	if len(result.Regressions) == 0 {
		return false
	}

	significantRegressions := 0
	for _, regression := range result.Regressions {
		// Simple statistical significance test based on magnitude and threshold
		if math.Abs(regression.PercentChange) > regression.Threshold*1.5 {
			significantRegressions++
		}
	}

	// Consider significant if more than 50% of regressions are statistically significant
	return float64(significantRegressions)/float64(len(result.Regressions)) > 0.5
}

// performRootCauseAnalysis analyzes potential root causes of regressions
func (ird *IntelligentRegressionDetector) performRootCauseAnalysis(result *RegressionResult, currentMetrics, baselineMetrics map[string]MetricData) []RootCauseHypothesis {
	hypotheses := []RootCauseHypothesis{}

	// Analyze patterns in the regressions
	for _, regression := range result.Regressions {
		hypotheses = append(hypotheses, ird.analyzeMetricRegression(regression, currentMetrics, baselineMetrics)...)
	}

	// Look for cross-metric patterns
	crossMetricHypotheses := ird.analyzeCrossMetricPatterns(result.Regressions, currentMetrics)
	hypotheses = append(hypotheses, crossMetricHypotheses...)

	// Sort by confidence
	sort.Slice(hypotheses, func(i, j int) bool {
		return hypotheses[i].Confidence > hypotheses[j].Confidence
	})

	// Return top 5 hypotheses
	if len(hypotheses) > 5 {
		hypotheses = hypotheses[:5]
	}

	return hypotheses
}

// analyzeMetricRegression analyzes a specific metric regression
func (ird *IntelligentRegressionDetector) analyzeMetricRegression(regression MetricRegression, currentMetrics, baselineMetrics map[string]MetricData) []RootCauseHypothesis {
	hypotheses := []RootCauseHypothesis{}

	switch regression.MetricName {
	case "http_req_duration":
		if regression.PercentChange > 50 {
			hypotheses = append(hypotheses, RootCauseHypothesis{
				Hypothesis:    "Database performance degradation",
				Confidence:    0.80,
				Evidence:      []string{"Significant API response time increase"},
				Investigation: "Check database query performance and connection pool status",
			})
		}
		if regression.PercentChange > 30 {
			hypotheses = append(hypotheses, RootCauseHypothesis{
				Hypothesis:    "Cache miss rate increase",
				Confidence:    0.70,
				Evidence:      []string{"API response time increase"},
				Investigation: "Check cache hit rates and cache configuration",
			})
		}

	case "database_query_duration":
		if regression.PercentChange > 100 {
			hypotheses = append(hypotheses, RootCauseHypothesis{
				Hypothesis:    "Missing or inefficient database indexes",
				Confidence:    0.85,
				Evidence:      []string{"Significant database query time increase"},
				Investigation: "Analyze query execution plans and index usage",
			})
		}
		if regression.PercentChange > 50 {
			hypotheses = append(hypotheses, RootCauseHypothesis{
				Hypothesis:    "Database connection pool exhaustion",
				Confidence:    0.75,
				Evidence:      []string{"Database query time increase"},
				Investigation: "Check database connection pool metrics and configuration",
			})
		}

	case "cache_hit_rate":
		if regression.PercentChange < -20 {
			hypotheses = append(hypotheses, RootCauseHypothesis{
				Hypothesis:    "Cache configuration change or cache invalidation issue",
				Confidence:    0.90,
				Evidence:      []string{"Significant cache hit rate decrease"},
				Investigation: "Review recent cache configuration changes and invalidation patterns",
			})
		}

	case "error_rate":
		if regression.PercentChange > 100 {
			hypotheses = append(hypotheses, RootCauseHypothesis{
				Hypothesis:    "Recent code deployment introduced bugs",
				Confidence:    0.85,
				Evidence:      []string{"Error rate doubled or more"},
				Investigation: "Review recent deployments and error logs",
			})
		}
	}

	return hypotheses
}

// analyzeCrossMetricPatterns looks for patterns across multiple metrics
func (ird *IntelligentRegressionDetector) analyzeCrossMetricPatterns(regressions []MetricRegression, currentMetrics map[string]MetricData) []RootCauseHypothesis {
	hypotheses := []RootCauseHypothesis{}

	// Check for database-related pattern
	dbMetricsRegressed := 0
	apiMetricsRegressed := 0
	
	for _, regression := range regressions {
		if regression.MetricName == "database_query_duration" || regression.MetricName == "article_creation_duration" {
			dbMetricsRegressed++
		}
		if regression.MetricName == "http_req_duration" || regression.MetricName == "api_response_time" {
			apiMetricsRegressed++
		}
	}

	if dbMetricsRegressed >= 2 && apiMetricsRegressed >= 1 {
		hypotheses = append(hypotheses, RootCauseHypothesis{
			Hypothesis:    "Database performance issue affecting entire application",
			Confidence:    0.90,
			Evidence:      []string{"Multiple database and API metrics regressed"},
			Investigation: "Focus on database performance analysis and optimization",
		})
	}

	// Check for resource exhaustion pattern
	resourceMetrics := []string{"memory_usage_mb", "cpu_usage_percent"}
	resourceRegressed := 0
	
	for _, regression := range regressions {
		for _, resourceMetric := range resourceMetrics {
			if regression.MetricName == resourceMetric {
				resourceRegressed++
				break
			}
		}
	}

	if resourceRegressed >= 1 && len(regressions) >= 3 {
		hypotheses = append(hypotheses, RootCauseHypothesis{
			Hypothesis:    "System resource exhaustion causing widespread performance issues",
			Confidence:    0.85,
			Evidence:      []string{"Resource utilization increased along with multiple performance metrics"},
			Investigation: "Check system resource usage and consider scaling",
		})
	}

	return hypotheses
}

// generateOptimizationSuggestions generates performance optimization suggestions
func (ird *IntelligentRegressionDetector) generateOptimizationSuggestions(result *RegressionResult, currentMetrics map[string]MetricData) []OptimizationSuggestion {
	suggestions := []OptimizationSuggestion{}

	for _, regression := range result.Regressions {
		metricSuggestions := ird.optimizationEngine.getSuggestionsForMetric(regression.MetricName, regression.PercentChange)
		suggestions = append(suggestions, metricSuggestions...)
	}

	// Add general suggestions based on overall performance
	generalSuggestions := ird.optimizationEngine.getGeneralSuggestions(result, currentMetrics)
	suggestions = append(suggestions, generalSuggestions...)

	// Sort by priority and confidence
	sort.Slice(suggestions, func(i, j int) bool {
		priorityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
		iPriority := priorityOrder[suggestions[i].Priority]
		jPriority := priorityOrder[suggestions[j].Priority]
		
		if iPriority != jPriority {
			return iPriority > jPriority
		}
		return suggestions[i].Confidence > suggestions[j].Confidence
	})

	// Return top 10 suggestions
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}

	return suggestions
}

// makeAlertingDecision determines whether to send alerts
func (ird *IntelligentRegressionDetector) makeAlertingDecision(result *EnhancedRegressionResult, options AnalysisOptions) AlertingDecision {
	decision := AlertingDecision{
		ShouldAlert:    false,
		AlertLevel:     "info",
		Reason:         "No significant regressions detected",
		EscalationPath: []string{},
	}

	if options.SuppressAlerts {
		decision.Reason = "Alerts suppressed by configuration"
		return decision
	}

	// Check if we should suppress alerts based on recent alerts
	suppressionKey := fmt.Sprintf("%s_%s", result.TestName, result.CurrentVersion)
	if suppressUntil, exists := ird.alertSuppressionRules[suppressionKey]; exists {
		if time.Now().Before(suppressUntil) {
			decision.Reason = "Alert suppressed due to recent similar alert"
			decision.SuppressUntil = &suppressUntil
			return decision
		}
	}

	// Determine alert level based on regressions and confidence
	if len(result.Regressions) == 0 {
		return decision
	}

	criticalRegressions := 0
	highRegressions := 0
	
	for _, regression := range result.Regressions {
		switch regression.Severity {
		case "critical":
			criticalRegressions++
		case "high":
			highRegressions++
		}
	}

	// Make alerting decision
	if criticalRegressions > 0 && result.ConfidenceScore > 0.8 {
		decision.ShouldAlert = true
		decision.AlertLevel = "critical"
		decision.Reason = fmt.Sprintf("%d critical regressions detected with %.1f%% confidence", criticalRegressions, result.ConfidenceScore*100)
		decision.EscalationPath = []string{"oncall-engineer", "team-lead", "engineering-manager"}
		
		// Set suppression for 30 minutes
		ird.alertSuppressionRules[suppressionKey] = time.Now().Add(30 * time.Minute)
		
	} else if (criticalRegressions > 0 || highRegressions > 1) && result.ConfidenceScore > 0.7 {
		decision.ShouldAlert = true
		decision.AlertLevel = "warning"
		decision.Reason = fmt.Sprintf("Performance regressions detected with %.1f%% confidence", result.ConfidenceScore*100)
		decision.EscalationPath = []string{"team-channel"}
		
		// Set suppression for 15 minutes
		ird.alertSuppressionRules[suppressionKey] = time.Now().Add(15 * time.Minute)
		
	} else if len(result.Regressions) > 0 && result.ConfidenceScore > 0.6 {
		decision.ShouldAlert = true
		decision.AlertLevel = "info"
		decision.Reason = fmt.Sprintf("Minor performance regressions detected with %.1f%% confidence", result.ConfidenceScore*100)
		decision.EscalationPath = []string{"team-channel"}
	}

	return decision
}

// updateAdaptiveThresholds updates thresholds based on recent performance data
func (ird *IntelligentRegressionDetector) updateAdaptiveThresholds(currentMetrics map[string]MetricData, result *RegressionResult) {
	for metricName, metric := range currentMetrics {
		threshold, exists := ird.adaptiveThresholds[metricName]
		if !exists {
			threshold = AdaptiveThreshold{
				BaseThreshold:    10.0, // Default 10% threshold
				CurrentThreshold: 10.0,
				Confidence:       0.5,
				LastUpdated:      time.Now(),
				HistoricalData:   []float64{},
			}
		}

		// Add current metric to historical data
		threshold.HistoricalData = append(threshold.HistoricalData, metric.P95)
		
		// Keep only last 100 data points
		if len(threshold.HistoricalData) > 100 {
			threshold.HistoricalData = threshold.HistoricalData[1:]
		}

		// Update threshold based on historical variance
		if len(threshold.HistoricalData) >= 10 {
			variance := ird.calculateVariance(threshold.HistoricalData)
			coefficientOfVariation := math.Sqrt(variance) / ird.calculateMean(threshold.HistoricalData)
			
			// Adjust threshold based on metric stability
			if coefficientOfVariation < 0.1 {
				// Very stable metric - can use lower threshold
				threshold.CurrentThreshold = threshold.BaseThreshold * 0.8
				threshold.Confidence = 0.9
			} else if coefficientOfVariation < 0.2 {
				// Stable metric - use base threshold
				threshold.CurrentThreshold = threshold.BaseThreshold
				threshold.Confidence = 0.8
			} else {
				// Unstable metric - use higher threshold
				threshold.CurrentThreshold = threshold.BaseThreshold * 1.5
				threshold.Confidence = 0.6
			}
		}

		threshold.LastUpdated = time.Now()
		ird.adaptiveThresholds[metricName] = threshold
	}
}

// Helper functions

func (ird *IntelligentRegressionDetector) calculateVariance(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	
	mean := ird.calculateMean(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	
	return sumSquares / float64(len(values)-1)
}

func (ird *IntelligentRegressionDetector) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (ird *IntelligentRegressionDetector) getRecentRegressionHistory(metricName string, limit int) ([]RegressionResult, error) {
	// This would query the database for recent regression results
	// Simplified implementation for now
	return []RegressionResult{}, nil
}

func (ird *IntelligentRegressionDetector) initializeAdaptiveThresholds() {
	// Initialize with default thresholds for common metrics
	defaultThresholds := map[string]float64{
		"http_req_duration":         15.0, // 15% threshold
		"article_creation_duration": 20.0, // 20% threshold
		"database_query_duration":   25.0, // 25% threshold
		"cache_hit_rate":           10.0, // 10% threshold
		"error_rate":               50.0, // 50% threshold
		"memory_usage_mb":          30.0, // 30% threshold
		"cpu_usage_percent":        40.0, // 40% threshold
	}

	for metricName, threshold := range defaultThresholds {
		ird.adaptiveThresholds[metricName] = AdaptiveThreshold{
			BaseThreshold:    threshold,
			CurrentThreshold: threshold,
			Confidence:       0.7,
			LastUpdated:      time.Now(),
			HistoricalData:   []float64{},
		}
	}
}

func (ird *IntelligentRegressionDetector) storeEnhancedRegressionResult(result *EnhancedRegressionResult) error {
	// Store the enhanced regression result in the database
	enhancedData := map[string]interface{}{
		"confidence_score":           result.ConfidenceScore,
		"statistical_significance":   result.StatisticalSignificance,
		"root_cause_analysis":        result.RootCauseAnalysis,
		"optimization_suggestions":   result.OptimizationSuggestions,
		"alerting_decision":          result.AlertingDecision,
	}
	
	enhancedJSON, err := json.Marshal(enhancedData)
	if err != nil {
		return fmt.Errorf("failed to marshal enhanced data: %w", err)
	}

	resultJSON, err := json.Marshal(result.RegressionResult)
	if err != nil {
		return fmt.Errorf("failed to marshal regression result: %w", err)
	}

	_, err = ird.db.Exec(`
		INSERT INTO performance_regression_results 
		(test_name, current_version, baseline_version, result_data, overall_status, overall_score, compared_at, enhanced_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		result.TestName, result.CurrentVersion, result.BaselineVersion,
		resultJSON, string(result.OverallStatus), result.Summary.OverallScore, result.ComparedAt, enhancedJSON)

	return err
}

// NewOptimizationEngine creates a new optimization engine
func NewOptimizationEngine() *OptimizationEngine {
	engine := &OptimizationEngine{
		knowledgeBase: make(map[string][]OptimizationRule),
		patterns:      make(map[string]PerformancePattern),
	}

	engine.initializeKnowledgeBase()
	return engine
}

func (oe *OptimizationEngine) initializeKnowledgeBase() {
	// Database optimization rules
	oe.knowledgeBase["database_query_duration"] = []OptimizationRule{
		{
			Condition:  "increase > 50%",
			Suggestion: "Add missing database indexes for frequently queried columns",
			Category:   "database",
			Priority:   "high",
			Effort:     "medium",
			Impact:     "high",
			Confidence: 0.85,
		},
		{
			Condition:  "increase > 30%",
			Suggestion: "Optimize database connection pool configuration",
			Category:   "database",
			Priority:   "medium",
			Effort:     "low",
			Impact:     "medium",
			Confidence: 0.75,
		},
		{
			Condition:  "increase > 100%",
			Suggestion: "Consider database query optimization or partitioning",
			Category:   "database",
			Priority:   "critical",
			Effort:     "high",
			Impact:     "high",
			Confidence: 0.90,
		},
	}

	// API optimization rules
	oe.knowledgeBase["http_req_duration"] = []OptimizationRule{
		{
			Condition:  "increase > 40%",
			Suggestion: "Implement or optimize API response caching",
			Category:   "cache",
			Priority:   "high",
			Effort:     "medium",
			Impact:     "high",
			Confidence: 0.80,
		},
		{
			Condition:  "increase > 25%",
			Suggestion: "Review and optimize database queries in API endpoints",
			Category:   "application",
			Priority:   "medium",
			Effort:     "medium",
			Impact:     "medium",
			Confidence: 0.75,
		},
	}

	// Cache optimization rules
	oe.knowledgeBase["cache_hit_rate"] = []OptimizationRule{
		{
			Condition:  "decrease > 20%",
			Suggestion: "Review cache TTL settings and invalidation patterns",
			Category:   "cache",
			Priority:   "high",
			Effort:     "low",
			Impact:     "high",
			Confidence: 0.85,
		},
		{
			Condition:  "decrease > 10%",
			Suggestion: "Increase cache memory allocation if resources allow",
			Category:   "infrastructure",
			Priority:   "medium",
			Effort:     "low",
			Impact:     "medium",
			Confidence: 0.70,
		},
	}
}

func (oe *OptimizationEngine) getSuggestionsForMetric(metricName string, percentChange float64) []OptimizationSuggestion {
	suggestions := []OptimizationSuggestion{}
	
	rules, exists := oe.knowledgeBase[metricName]
	if !exists {
		return suggestions
	}

	for _, rule := range rules {
		if oe.evaluateCondition(rule.Condition, percentChange) {
			suggestion := OptimizationSuggestion{
				Category:   rule.Category,
				Suggestion: rule.Suggestion,
				Priority:   rule.Priority,
				Effort:     rule.Effort,
				Impact:     rule.Impact,
				Confidence: rule.Confidence,
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

func (oe *OptimizationEngine) getGeneralSuggestions(result *RegressionResult, currentMetrics map[string]MetricData) []OptimizationSuggestion {
	suggestions := []OptimizationSuggestion{}

	// If multiple metrics are regressed, suggest infrastructure scaling
	if len(result.Regressions) >= 3 {
		suggestions = append(suggestions, OptimizationSuggestion{
			Category:   "infrastructure",
			Suggestion: "Consider horizontal scaling due to multiple performance regressions",
			Priority:   "high",
			Effort:     "medium",
			Impact:     "high",
			Confidence: 0.80,
		})
	}

	// If error rate is high, suggest monitoring improvements
	for _, regression := range result.Regressions {
		if regression.MetricName == "error_rate" && regression.PercentChange > 50 {
			suggestions = append(suggestions, OptimizationSuggestion{
				Category:   "monitoring",
				Suggestion: "Implement enhanced error tracking and alerting",
				Priority:   "high",
				Effort:     "low",
				Impact:     "medium",
				Confidence: 0.75,
			})
		}
	}

	return suggestions
}

func (oe *OptimizationEngine) evaluateCondition(condition string, percentChange float64) bool {
	// Simple condition evaluation - in a real implementation, this would be more sophisticated
	absChange := math.Abs(percentChange)
	
	switch condition {
	case "increase > 100%":
		return percentChange > 100
	case "increase > 50%":
		return percentChange > 50
	case "increase > 40%":
		return percentChange > 40
	case "increase > 30%":
		return percentChange > 30
	case "increase > 25%":
		return percentChange > 25
	case "decrease > 20%":
		return percentChange < -20
	case "decrease > 10%":
		return percentChange < -10
	default:
		return absChange > 10 // Default threshold
	}
}