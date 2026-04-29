package testing

import (
	"log"
	"math"
	"sort"
	"time"
)

// TrendAnalyzer analyzes trends in test metrics over time
type TrendAnalyzer struct {
	historicalData *HistoricalDataStore
}

// HistoricalDataStore stores historical test data for trend analysis
type HistoricalDataStore struct {
	reports []TestReport
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer() *TrendAnalyzer {
	return &TrendAnalyzer{
		historicalData: &HistoricalDataStore{
			reports: []TestReport{},
		},
	}
}

// AnalyzeTrends analyzes trends for the given pipeline results
func (t *TrendAnalyzer) AnalyzeTrends(results []PipelineResult, period ReportPeriod) TrendAnalysis {
	log.Printf("Analyzing trends for period: %s to %s", period.StartTime.Format(time.RFC3339), period.EndTime.Format(time.RFC3339))
	
	analysis := TrendAnalysis{
		Predictions: []TrendPrediction{},
	}
	
	// Analyze quality trend
	analysis.QualityTrend = t.analyzeQualityTrend(results)
	
	// Analyze coverage trend
	analysis.CoverageTrend = t.analyzeCoverageTrend(results)
	
	// Analyze performance trend
	analysis.PerformanceTrend = t.analyzePerformanceTrend(results)
	
	// Analyze security trend
	analysis.SecurityTrend = t.analyzeSecurityTrend(results)
	
	// Generate predictions
	analysis.Predictions = t.generatePredictions(results)
	
	return analysis
}

// analyzeQualityTrend analyzes overall quality trend
func (t *TrendAnalyzer) analyzeQualityTrend(results []PipelineResult) TrendDirection {
	if len(results) < 2 {
		return TrendStable
	}
	
	// Calculate quality scores over time
	var scores []float64
	for _, result := range results {
		score := t.calculateQualityScore(result)
		scores = append(scores, score)
	}
	
	return t.determineTrendDirection(scores)
}

// analyzeCoverageTrend analyzes code coverage trend
func (t *TrendAnalyzer) analyzeCoverageTrend(results []PipelineResult) TrendDirection {
	var coverageValues []float64
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Metrics != nil {
				if coverage, ok := stage.Metrics["coverage"].(float64); ok {
					coverageValues = append(coverageValues, coverage)
				}
			}
		}
	}
	
	if len(coverageValues) < 2 {
		return TrendStable
	}
	
	return t.determineTrendDirection(coverageValues)
}

// analyzePerformanceTrend analyzes performance trend
func (t *TrendAnalyzer) analyzePerformanceTrend(results []PipelineResult) TrendDirection {
	var regressionValues []float64
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Type == StageTypePerformance && stage.Metrics != nil {
				if regression, ok := stage.Metrics["regression_percentage"].(float64); ok {
					// Invert regression values (lower regression = better performance)
					regressionValues = append(regressionValues, -regression)
				}
			}
		}
	}
	
	if len(regressionValues) < 2 {
		return TrendStable
	}
	
	return t.determineTrendDirection(regressionValues)
}

// analyzeSecurityTrend analyzes security trend
func (t *TrendAnalyzer) analyzeSecurityTrend(results []PipelineResult) TrendDirection {
	var securityScores []float64
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Type == StageTypeSecurity && stage.Metrics != nil {
				if criticalIssues, ok := stage.Metrics["critical_issues"].(int); ok {
					// Convert to score (fewer issues = higher score)
					score := math.Max(0, 100-float64(criticalIssues)*10)
					securityScores = append(securityScores, score)
				}
			}
		}
	}
	
	if len(securityScores) < 2 {
		return TrendStable
	}
	
	return t.determineTrendDirection(securityScores)
}

// determineTrendDirection determines trend direction from a series of values
func (t *TrendAnalyzer) determineTrendDirection(values []float64) TrendDirection {
	if len(values) < 2 {
		return TrendStable
	}
	
	// Calculate linear regression slope
	slope := t.calculateSlope(values)
	
	// Determine trend based on slope
	if slope > 0.5 {
		return TrendImproving
	} else if slope < -0.5 {
		return TrendDeclining
	} else {
		return TrendStable
	}
}

// calculateSlope calculates the slope of a linear regression line
func (t *TrendAnalyzer) calculateSlope(values []float64) float64 {
	n := float64(len(values))
	if n < 2 {
		return 0
	}
	
	var sumX, sumY, sumXY, sumX2 float64
	
	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	// Calculate slope using least squares method
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0
	}
	
	slope := (n*sumXY - sumX*sumY) / denominator
	return slope
}

// calculateQualityScore calculates an overall quality score for a pipeline result
func (t *TrendAnalyzer) calculateQualityScore(result PipelineResult) float64 {
	score := 100.0
	
	// Deduct points for failed stages
	for _, stage := range result.Stages {
		if stage.Status == StageStatusFailed {
			if stage.Required {
				score -= 25.0
			} else {
				score -= 10.0
			}
		}
	}
	
	// Adjust based on metrics
	for _, stage := range result.Stages {
		if stage.Metrics != nil {
			// Coverage impact
			if coverage, ok := stage.Metrics["coverage"].(float64); ok {
				if coverage < 80 {
					score -= (80 - coverage) * 0.5
				}
			}
			
			// Security impact
			if criticalIssues, ok := stage.Metrics["critical_issues"].(int); ok {
				score -= float64(criticalIssues) * 10
			}
			
			// Performance impact
			if regression, ok := stage.Metrics["regression_percentage"].(float64); ok {
				if regression > 5 {
					score -= (regression - 5) * 2
				}
			}
		}
	}
	
	return math.Max(0, math.Min(100, score))
}

// generatePredictions generates trend predictions
func (t *TrendAnalyzer) generatePredictions(results []PipelineResult) []TrendPrediction {
	var predictions []TrendPrediction
	
	// Coverage prediction
	coveragePrediction := t.predictCoverage(results)
	if coveragePrediction != nil {
		predictions = append(predictions, *coveragePrediction)
	}
	
	// Security prediction
	securityPrediction := t.predictSecurity(results)
	if securityPrediction != nil {
		predictions = append(predictions, *securityPrediction)
	}
	
	// Performance prediction
	performancePrediction := t.predictPerformance(results)
	if performancePrediction != nil {
		predictions = append(predictions, *performancePrediction)
	}
	
	return predictions
}

// predictCoverage predicts future coverage trends
func (t *TrendAnalyzer) predictCoverage(results []PipelineResult) *TrendPrediction {
	var coverageValues []float64
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Metrics != nil {
				if coverage, ok := stage.Metrics["coverage"].(float64); ok {
					coverageValues = append(coverageValues, coverage)
				}
			}
		}
	}
	
	if len(coverageValues) < 3 {
		return nil
	}
	
	slope := t.calculateSlope(coverageValues)
	currentCoverage := coverageValues[len(coverageValues)-1]
	predictedCoverage := currentCoverage + slope*7 // Predict 7 days ahead
	
	confidence := t.calculateConfidence(coverageValues)
	
	return &TrendPrediction{
		Metric:     "code_coverage",
		Prediction: predictedCoverage,
		Confidence: confidence,
		TimeFrame:  "1 week",
		Reasoning:  t.generateCoverageReasoning(slope, confidence),
	}
}

// predictSecurity predicts future security trends
func (t *TrendAnalyzer) predictSecurity(results []PipelineResult) *TrendPrediction {
	var securityScores []float64
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Type == StageTypeSecurity && stage.Metrics != nil {
				if criticalIssues, ok := stage.Metrics["critical_issues"].(int); ok {
					score := math.Max(0, 100-float64(criticalIssues)*10)
					securityScores = append(securityScores, score)
				}
			}
		}
	}
	
	if len(securityScores) < 3 {
		return nil
	}
	
	slope := t.calculateSlope(securityScores)
	currentScore := securityScores[len(securityScores)-1]
	predictedScore := currentScore + slope*7
	
	confidence := t.calculateConfidence(securityScores)
	
	return &TrendPrediction{
		Metric:     "security_score",
		Prediction: predictedScore,
		Confidence: confidence,
		TimeFrame:  "1 week",
		Reasoning:  t.generateSecurityReasoning(slope, confidence),
	}
}

// predictPerformance predicts future performance trends
func (t *TrendAnalyzer) predictPerformance(results []PipelineResult) *TrendPrediction {
	var regressionValues []float64
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Type == StageTypePerformance && stage.Metrics != nil {
				if regression, ok := stage.Metrics["regression_percentage"].(float64); ok {
					regressionValues = append(regressionValues, regression)
				}
			}
		}
	}
	
	if len(regressionValues) < 3 {
		return nil
	}
	
	slope := t.calculateSlope(regressionValues)
	currentRegression := regressionValues[len(regressionValues)-1]
	predictedRegression := currentRegression + slope*7
	
	confidence := t.calculateConfidence(regressionValues)
	
	return &TrendPrediction{
		Metric:     "performance_regression",
		Prediction: predictedRegression,
		Confidence: confidence,
		TimeFrame:  "1 week",
		Reasoning:  t.generatePerformanceReasoning(slope, confidence),
	}
}

// calculateConfidence calculates confidence level for predictions
func (t *TrendAnalyzer) calculateConfidence(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}
	
	// Calculate variance
	mean := t.calculateMean(values)
	var variance float64
	for _, value := range values {
		variance += math.Pow(value-mean, 2)
	}
	variance /= float64(len(values))
	
	// Lower variance = higher confidence
	confidence := math.Max(0, math.Min(100, 100-variance))
	return confidence
}

// calculateMean calculates the mean of a slice of values
func (t *TrendAnalyzer) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

// generateCoverageReasoning generates reasoning for coverage predictions
func (t *TrendAnalyzer) generateCoverageReasoning(slope, confidence float64) string {
	if slope > 0.5 {
		return "Coverage is improving steadily based on recent test additions"
	} else if slope < -0.5 {
		return "Coverage is declining, possibly due to new code without corresponding tests"
	} else {
		return "Coverage remains stable with minor fluctuations"
	}
}

// generateSecurityReasoning generates reasoning for security predictions
func (t *TrendAnalyzer) generateSecurityReasoning(slope, confidence float64) string {
	if slope > 0.5 {
		return "Security posture is improving through vulnerability remediation efforts"
	} else if slope < -0.5 {
		return "Security issues are increasing, requiring immediate attention"
	} else {
		return "Security posture remains stable with consistent monitoring"
	}
}

// generatePerformanceReasoning generates reasoning for performance predictions
func (t *TrendAnalyzer) generatePerformanceReasoning(slope, confidence float64) string {
	if slope > 0.5 {
		return "Performance regression is increasing, indicating potential bottlenecks"
	} else if slope < -0.5 {
		return "Performance is improving through optimization efforts"
	} else {
		return "Performance remains stable within acceptable thresholds"
	}
}

// StoreHistoricalData stores a report for future trend analysis
func (t *TrendAnalyzer) StoreHistoricalData(report TestReport) {
	t.historicalData.reports = append(t.historicalData.reports, report)
	
	// Keep only last 30 reports to manage memory
	if len(t.historicalData.reports) > 30 {
		t.historicalData.reports = t.historicalData.reports[len(t.historicalData.reports)-30:]
	}
}

// GetHistoricalTrends returns historical trend data
func (t *TrendAnalyzer) GetHistoricalTrends(metric string, days int) []TrendDataPoint {
	var dataPoints []TrendDataPoint
	
	cutoff := time.Now().AddDate(0, 0, -days)
	
	for _, report := range t.historicalData.reports {
		if report.GeneratedAt.After(cutoff) {
			point := TrendDataPoint{
				Timestamp: report.GeneratedAt,
			}
			
			switch metric {
			case "coverage":
				point.Value = report.QualityMetrics.CodeCoverage.Overall
			case "security":
				point.Value = report.QualityMetrics.SecurityMetrics.SecurityScoreAvg
			case "performance":
				point.Value = report.QualityMetrics.PerformanceMetrics.RegressionPercentage
			}
			
			dataPoints = append(dataPoints, point)
		}
	}
	
	// Sort by timestamp
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})
	
	return dataPoints
}

// TrendDataPoint represents a data point in a trend
type TrendDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}