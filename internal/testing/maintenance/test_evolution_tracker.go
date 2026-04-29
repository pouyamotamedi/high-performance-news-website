package maintenance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// TestEvolutionTracker tracks the evolution and changes of tests over time
type TestEvolutionTracker struct {
	db *sql.DB
}

// NewTestEvolutionTracker creates a new test evolution tracker
func NewTestEvolutionTracker(db *sql.DB) *TestEvolutionTracker {
	return &TestEvolutionTracker{
		db: db,
	}
}

// TrackTestChange records a change to a test
func (tet *TestEvolutionTracker) TrackTestChange(testID string, changeType ChangeType, description, author, reason string, impact Impact) error {
	change := TestChange{
		ID:          fmt.Sprintf("change_%s_%d", testID, time.Now().Unix()),
		Type:        changeType,
		Description: description,
		Author:      author,
		Timestamp:   time.Now(),
		Impact:      impact,
		Reason:      reason,
	}

	// Store change in database
	impactJSON, _ := json.Marshal(impact)
	
	_, err := tet.db.Exec(`
		INSERT INTO test_changes (
			change_id, test_id, change_type, description, author, 
			timestamp, impact, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, change.ID, testID, string(change.Type), change.Description, 
		change.Author, change.Timestamp, impactJSON, change.Reason)

	if err != nil {
		return fmt.Errorf("failed to track test change: %w", err)
	}

	// Update test evolution record
	return tet.updateTestEvolution(testID, change)
}

// updateTestEvolution updates the evolution record for a test
func (tet *TestEvolutionTracker) updateTestEvolution(testID string, change TestChange) error {
	// Check if evolution record exists
	var exists bool
	err := tet.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM test_evolution WHERE test_id = $1)
	`, testID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check evolution record: %w", err)
	}

	if !exists {
		// Create new evolution record
		evolution := TestEvolution{
			TestID:    testID,
			Changes:   []TestChange{change},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		changesJSON, _ := json.Marshal(evolution.Changes)
		_, err = tet.db.Exec(`
			INSERT INTO test_evolution (test_id, changes, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
		`, evolution.TestID, changesJSON, evolution.CreatedAt, evolution.UpdatedAt)

		return err
	} else {
		// Update existing evolution record
		var changesJSON []byte
		err = tet.db.QueryRow(`
			SELECT changes FROM test_evolution WHERE test_id = $1
		`, testID).Scan(&changesJSON)
		if err != nil {
			return fmt.Errorf("failed to get existing changes: %w", err)
		}

		var changes []TestChange
		json.Unmarshal(changesJSON, &changes)
		changes = append(changes, change)

		updatedChangesJSON, _ := json.Marshal(changes)
		_, err = tet.db.Exec(`
			UPDATE test_evolution 
			SET changes = $1, updated_at = $2
			WHERE test_id = $3
		`, updatedChangesJSON, time.Now(), testID)

		return err
	}
}

// RecordMetricSnapshot records a metric snapshot for a test
func (tet *TestEvolutionTracker) RecordMetricSnapshot(testID string, metrics TestMetricSnapshot) error {
	metricsJSON, _ := json.Marshal(metrics)
	
	_, err := tet.db.Exec(`
		INSERT INTO test_metric_snapshots (
			test_id, timestamp, coverage, runtime_ms, failure_rate, 
			execution_count, complexity, metrics_json
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, testID, metrics.Timestamp, metrics.Coverage, 
		metrics.Runtime.Milliseconds(), metrics.FailureRate, 
		metrics.ExecutionCount, metrics.Complexity, metricsJSON)

	if err != nil {
		return fmt.Errorf("failed to record metric snapshot: %w", err)
	}

	// Update evolution record with new metrics
	return tet.updateEvolutionMetrics(testID, metrics)
}

// updateEvolutionMetrics updates the metrics in the evolution record
func (tet *TestEvolutionTracker) updateEvolutionMetrics(testID string, metrics TestMetricSnapshot) error {
	// Get existing evolution record
	var metricsJSON []byte
	err := tet.db.QueryRow(`
		SELECT metrics FROM test_evolution WHERE test_id = $1
	`, testID).Scan(&metricsJSON)
	
	var existingMetrics []TestMetricSnapshot
	if err == nil && len(metricsJSON) > 0 {
		json.Unmarshal(metricsJSON, &existingMetrics)
	}

	// Add new metrics
	existingMetrics = append(existingMetrics, metrics)

	// Keep only last 100 snapshots to avoid excessive storage
	if len(existingMetrics) > 100 {
		existingMetrics = existingMetrics[len(existingMetrics)-100:]
	}

	updatedMetricsJSON, _ := json.Marshal(existingMetrics)
	
	_, err = tet.db.Exec(`
		UPDATE test_evolution 
		SET metrics = $1, updated_at = $2
		WHERE test_id = $3
	`, updatedMetricsJSON, time.Now(), testID)

	return err
}

// GetTestEvolution retrieves the complete evolution history of a test
func (tet *TestEvolutionTracker) GetTestEvolution(testID string) (*TestEvolution, error) {
	var evolution TestEvolution
	var changesJSON, metricsJSON []byte

	err := tet.db.QueryRow(`
		SELECT test_id, changes, metrics, created_at, updated_at
		FROM test_evolution
		WHERE test_id = $1
	`, testID).Scan(&evolution.TestID, &changesJSON, &metricsJSON, 
		&evolution.CreatedAt, &evolution.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("evolution record not found for test: %s", testID)
		}
		return nil, fmt.Errorf("failed to get test evolution: %w", err)
	}

	// Unmarshal changes and metrics
	if len(changesJSON) > 0 {
		json.Unmarshal(changesJSON, &evolution.Changes)
	}
	if len(metricsJSON) > 0 {
		json.Unmarshal(metricsJSON, &evolution.Metrics)
	}

	return &evolution, nil
}

// AnalyzeTestEvolution analyzes the evolution patterns of a test
func (tet *TestEvolutionTracker) AnalyzeTestEvolution(testID string) (*EvolutionAnalysis, error) {
	evolution, err := tet.GetTestEvolution(testID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test evolution: %w", err)
	}

	analysis := &EvolutionAnalysis{
		TestID:    testID,
		Timestamp: time.Now(),
	}

	// Analyze change patterns
	analysis.ChangeFrequency = tet.calculateChangeFrequency(evolution.Changes)
	analysis.ChangeTypes = tet.analyzeChangeTypes(evolution.Changes)
	analysis.ImpactTrends = tet.analyzeImpactTrends(evolution.Changes)

	// Analyze metric trends
	if len(evolution.Metrics) > 0 {
		analysis.MetricTrends = tet.analyzeMetricTrends(evolution.Metrics)
		analysis.QualityTrend = tet.calculateQualityTrend(evolution.Metrics)
		analysis.StabilityTrend = tet.calculateStabilityTrend(evolution.Metrics)
	}

	// Generate insights
	analysis.Insights = tet.generateEvolutionInsights(analysis)

	return analysis, nil
}

// EvolutionAnalysis represents analysis of test evolution
type EvolutionAnalysis struct {
	TestID          string                    `json:"test_id"`
	Timestamp       time.Time                 `json:"timestamp"`
	ChangeFrequency float64                   `json:"change_frequency"` // Changes per month
	ChangeTypes     map[string]int            `json:"change_types"`
	ImpactTrends    map[string][]float64      `json:"impact_trends"`
	MetricTrends    map[string][]float64      `json:"metric_trends"`
	QualityTrend    string                    `json:"quality_trend"` // "improving", "degrading", "stable"
	StabilityTrend  string                    `json:"stability_trend"`
	Insights        []EvolutionInsight        `json:"insights"`
}

// EvolutionInsight represents an insight from evolution analysis
type EvolutionInsight struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Suggestion  string    `json:"suggestion"`
	Confidence  float64   `json:"confidence"`
}

// calculateChangeFrequency calculates the frequency of changes
func (tet *TestEvolutionTracker) calculateChangeFrequency(changes []TestChange) float64 {
	if len(changes) < 2 {
		return 0
	}

	// Calculate time span
	firstChange := changes[0].Timestamp
	lastChange := changes[len(changes)-1].Timestamp
	timeSpan := lastChange.Sub(firstChange)

	if timeSpan.Hours() < 24 {
		return 0 // Not enough data
	}

	// Calculate changes per month
	monthsSpan := timeSpan.Hours() / (24 * 30)
	return float64(len(changes)) / monthsSpan
}

// analyzeChangeTypes analyzes the distribution of change types
func (tet *TestEvolutionTracker) analyzeChangeTypes(changes []TestChange) map[string]int {
	types := make(map[string]int)
	
	for _, change := range changes {
		types[string(change.Type)]++
	}

	return types
}

// analyzeImpactTrends analyzes trends in change impacts
func (tet *TestEvolutionTracker) analyzeImpactTrends(changes []TestChange) map[string][]float64 {
	trends := map[string][]float64{
		"coverage":   []float64{},
		"runtime":    []float64{},
		"stability":  []float64{},
		"complexity": []float64{},
	}

	for _, change := range changes {
		trends["coverage"] = append(trends["coverage"], change.Impact.CoverageChange)
		trends["runtime"] = append(trends["runtime"], change.Impact.RuntimeChange.Seconds())
		trends["stability"] = append(trends["stability"], change.Impact.StabilityChange)
		trends["complexity"] = append(trends["complexity"], float64(change.Impact.ComplexityChange))
	}

	return trends
}

// analyzeMetricTrends analyzes trends in test metrics
func (tet *TestEvolutionTracker) analyzeMetricTrends(metrics []TestMetricSnapshot) map[string][]float64 {
	trends := map[string][]float64{
		"coverage":       []float64{},
		"runtime":        []float64{},
		"failure_rate":   []float64{},
		"execution_count": []float64{},
		"complexity":     []float64{},
	}

	for _, metric := range metrics {
		trends["coverage"] = append(trends["coverage"], metric.Coverage)
		trends["runtime"] = append(trends["runtime"], metric.Runtime.Seconds())
		trends["failure_rate"] = append(trends["failure_rate"], metric.FailureRate)
		trends["execution_count"] = append(trends["execution_count"], float64(metric.ExecutionCount))
		trends["complexity"] = append(trends["complexity"], float64(metric.Complexity))
	}

	return trends
}

// calculateQualityTrend calculates the overall quality trend
func (tet *TestEvolutionTracker) calculateQualityTrend(metrics []TestMetricSnapshot) string {
	if len(metrics) < 3 {
		return "insufficient_data"
	}

	// Calculate quality score for each snapshot
	scores := make([]float64, len(metrics))
	for i, metric := range metrics {
		// Quality score based on coverage, low failure rate, and reasonable complexity
		score := metric.Coverage * 0.4 + // Coverage weight: 40%
			(1.0-metric.FailureRate)*0.4 + // Stability weight: 40%
			(1.0/float64(metric.Complexity+1))*0.2 // Simplicity weight: 20%
		scores[i] = score
	}

	// Analyze trend
	recentScores := scores[len(scores)-3:] // Last 3 snapshots
	earlyScores := scores[:3]              // First 3 snapshots

	recentAvg := tet.average(recentScores)
	earlyAvg := tet.average(earlyScores)

	diff := recentAvg - earlyAvg
	if diff > 0.05 {
		return "improving"
	} else if diff < -0.05 {
		return "degrading"
	}
	return "stable"
}

// calculateStabilityTrend calculates the stability trend
func (tet *TestEvolutionTracker) calculateStabilityTrend(metrics []TestMetricSnapshot) string {
	if len(metrics) < 3 {
		return "insufficient_data"
	}

	// Extract failure rates
	failureRates := make([]float64, len(metrics))
	for i, metric := range metrics {
		failureRates[i] = metric.FailureRate
	}

	// Analyze trend in failure rates
	recentRates := failureRates[len(failureRates)-3:]
	earlyRates := failureRates[:3]

	recentAvg := tet.average(recentRates)
	earlyAvg := tet.average(earlyRates)

	diff := recentAvg - earlyAvg
	if diff < -0.02 { // Failure rate decreased
		return "improving"
	} else if diff > 0.02 { // Failure rate increased
		return "degrading"
	}
	return "stable"
}

// average calculates the average of a slice of float64
func (tet *TestEvolutionTracker) average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// generateEvolutionInsights generates insights from evolution analysis
func (tet *TestEvolutionTracker) generateEvolutionInsights(analysis *EvolutionAnalysis) []EvolutionInsight {
	var insights []EvolutionInsight

	// High change frequency insight
	if analysis.ChangeFrequency > 5 { // More than 5 changes per month
		insights = append(insights, EvolutionInsight{
			Type:        "high_change_frequency",
			Description: fmt.Sprintf("Test has high change frequency: %.1f changes per month", analysis.ChangeFrequency),
			Severity:    "medium",
			Suggestion:  "Consider stabilizing the test or reviewing if frequent changes indicate underlying issues",
			Confidence:  0.8,
		})
	}

	// Quality degradation insight
	if analysis.QualityTrend == "degrading" {
		insights = append(insights, EvolutionInsight{
			Type:        "quality_degradation",
			Description: "Test quality is degrading over time",
			Severity:    "high",
			Suggestion:  "Review recent changes and consider refactoring to improve test quality",
			Confidence:  0.9,
		})
	}

	// Stability issues insight
	if analysis.StabilityTrend == "degrading" {
		insights = append(insights, EvolutionInsight{
			Type:        "stability_issues",
			Description: "Test stability is decreasing over time",
			Severity:    "high",
			Suggestion:  "Investigate causes of increased failure rate and improve test reliability",
			Confidence:  0.85,
		})
	}

	// Frequent modifications insight
	if modCount, exists := analysis.ChangeTypes["modified"]; exists && modCount > 10 {
		insights = append(insights, EvolutionInsight{
			Type:        "frequent_modifications",
			Description: fmt.Sprintf("Test has been modified %d times", modCount),
			Severity:    "medium",
			Suggestion:  "Consider if the test is testing too many things or if requirements are unstable",
			Confidence:  0.7,
		})
	}

	// Positive trends insight
	if analysis.QualityTrend == "improving" && analysis.StabilityTrend == "improving" {
		insights = append(insights, EvolutionInsight{
			Type:        "positive_evolution",
			Description: "Test is showing positive evolution in quality and stability",
			Severity:    "info",
			Suggestion:  "Continue current practices that are improving test quality",
			Confidence:  0.8,
		})
	}

	return insights
}

// GetEvolutionSummary gets a summary of evolution for all tests
func (tet *TestEvolutionTracker) GetEvolutionSummary() (*EvolutionSummary, error) {
	summary := &EvolutionSummary{
		Timestamp: time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	// Get total number of tests with evolution data
	var totalTests int
	err := tet.db.QueryRow(`
		SELECT COUNT(*) FROM test_evolution
	`).Scan(&totalTests)
	if err != nil {
		return nil, fmt.Errorf("failed to get total tests: %w", err)
	}
	summary.Metrics["total_tests_tracked"] = totalTests

	// Get average change frequency
	var avgChangeFreq sql.NullFloat64
	err = tet.db.QueryRow(`
		SELECT AVG(
			CASE 
				WHEN json_array_length(changes) > 1 THEN
					json_array_length(changes) / 
					GREATEST(1, EXTRACT(EPOCH FROM (updated_at - created_at)) / (30 * 24 * 3600))
				ELSE 0
			END
		) as avg_change_frequency
		FROM test_evolution
	`).Scan(&avgChangeFreq)
	if err == nil && avgChangeFreq.Valid {
		summary.Metrics["average_change_frequency"] = avgChangeFreq.Float64
	}

	// Get most active tests
	activeTests, err := tet.getMostActiveTests(10)
	if err != nil {
		log.Printf("Failed to get most active tests: %v", err)
	} else {
		summary.MostActiveTests = activeTests
	}

	// Get quality trends distribution
	qualityTrends, err := tet.getQualityTrendsDistribution()
	if err != nil {
		log.Printf("Failed to get quality trends: %v", err)
	} else {
		summary.QualityTrends = qualityTrends
	}

	return summary, nil
}

// EvolutionSummary represents a summary of test evolution
type EvolutionSummary struct {
	Timestamp       time.Time              `json:"timestamp"`
	Metrics         map[string]interface{} `json:"metrics"`
	MostActiveTests []ActiveTestInfo       `json:"most_active_tests"`
	QualityTrends   map[string]int         `json:"quality_trends"`
}

// ActiveTestInfo represents information about an active test
type ActiveTestInfo struct {
	TestID          string    `json:"test_id"`
	ChangeCount     int       `json:"change_count"`
	LastChange      time.Time `json:"last_change"`
	ChangeFrequency float64   `json:"change_frequency"`
}

// getMostActiveTests gets the most actively changing tests
func (tet *TestEvolutionTracker) getMostActiveTests(limit int) ([]ActiveTestInfo, error) {
	rows, err := tet.db.Query(`
		SELECT test_id, json_array_length(changes) as change_count, updated_at,
			   CASE 
				   WHEN json_array_length(changes) > 1 THEN
					   json_array_length(changes) / 
					   GREATEST(1, EXTRACT(EPOCH FROM (updated_at - created_at)) / (30 * 24 * 3600))
				   ELSE 0
			   END as change_frequency
		FROM test_evolution
		ORDER BY change_count DESC, change_frequency DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activeTests []ActiveTestInfo
	for rows.Next() {
		var test ActiveTestInfo
		err := rows.Scan(&test.TestID, &test.ChangeCount, &test.LastChange, &test.ChangeFrequency)
		if err != nil {
			continue
		}
		activeTests = append(activeTests, test)
	}

	return activeTests, rows.Err()
}

// getQualityTrendsDistribution gets the distribution of quality trends
func (tet *TestEvolutionTracker) getQualityTrendsDistribution() (map[string]int, error) {
	// This would require analyzing all tests, which is computationally expensive
	// For now, return a placeholder implementation
	trends := map[string]int{
		"improving":        0,
		"stable":          0,
		"degrading":       0,
		"insufficient_data": 0,
	}

	// In a real implementation, you would:
	// 1. Get all test evolutions
	// 2. Analyze each one for quality trends
	// 3. Count the distribution
	// This is simplified for demonstration

	return trends, nil
}

// AnalyzeChangeImpact analyzes the impact of changes across all tests
func (tet *TestEvolutionTracker) AnalyzeChangeImpact() (*ChangeImpactAnalysis, error) {
	analysis := &ChangeImpactAnalysis{
		Timestamp: time.Now(),
		Impacts:   make(map[string]ImpactStatistics),
	}

	// Analyze coverage impact
	coverageStats, err := tet.getImpactStatistics("coverage")
	if err != nil {
		return nil, fmt.Errorf("failed to get coverage impact stats: %w", err)
	}
	analysis.Impacts["coverage"] = coverageStats

	// Analyze runtime impact
	runtimeStats, err := tet.getImpactStatistics("runtime")
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime impact stats: %w", err)
	}
	analysis.Impacts["runtime"] = runtimeStats

	// Analyze stability impact
	stabilityStats, err := tet.getImpactStatistics("stability")
	if err != nil {
		return nil, fmt.Errorf("failed to get stability impact stats: %w", err)
	}
	analysis.Impacts["stability"] = stabilityStats

	return analysis, nil
}

// ChangeImpactAnalysis represents analysis of change impacts
type ChangeImpactAnalysis struct {
	Timestamp time.Time                    `json:"timestamp"`
	Impacts   map[string]ImpactStatistics  `json:"impacts"`
}

// ImpactStatistics represents statistics about change impacts
type ImpactStatistics struct {
	PositiveChanges int     `json:"positive_changes"`
	NegativeChanges int     `json:"negative_changes"`
	NeutralChanges  int     `json:"neutral_changes"`
	AverageImpact   float64 `json:"average_impact"`
	MaxImpact       float64 `json:"max_impact"`
	MinImpact       float64 `json:"min_impact"`
}

// getImpactStatistics gets impact statistics for a specific metric
func (tet *TestEvolutionTracker) getImpactStatistics(metric string) (ImpactStatistics, error) {
	var stats ImpactStatistics

	// This is a simplified implementation
	// In practice, you would query the database for impact data
	// and calculate statistics based on the specific metric

	return stats, nil
}