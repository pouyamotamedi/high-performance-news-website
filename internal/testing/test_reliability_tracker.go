package testing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"time"
)

// NotificationManagerInterface defines the interface for notification management
type NotificationManagerInterface interface {
	NotifyTestQuarantined(testName, testSuite, reason string)
	NotifyTestReintegrated(testName, testSuite string)
	NotifyTestsQuarantined(count int)
	NotifyHighPriorityRecommendations(count int)
}

// SimpleNotificationManager provides a simple implementation
type SimpleNotificationManager struct{}

func (s *SimpleNotificationManager) NotifyTestQuarantined(testName, testSuite, reason string) {
	log.Printf("Test quarantined: %s.%s - %s", testSuite, testName, reason)
}

func (s *SimpleNotificationManager) NotifyTestReintegrated(testName, testSuite string) {
	log.Printf("Test reintegrated: %s.%s", testSuite, testName)
}

func (s *SimpleNotificationManager) NotifyTestsQuarantined(count int) {
	log.Printf("Multiple tests quarantined: %d tests", count)
}

func (s *SimpleNotificationManager) NotifyHighPriorityRecommendations(count int) {
	log.Printf("High priority recommendations available: %d recommendations", count)
}

// TestReliabilityTracker provides intelligent test reliability tracking
type TestReliabilityTracker struct {
	db                *sql.DB
	config            *TestReliabilityConfig
	patternAnalyzer   *FailurePatternAnalyzer
	remediationEngine *RemediationEngine
	notificationMgr   NotificationManagerInterface
}

// NewTestReliabilityTracker creates a new test reliability tracker
func NewTestReliabilityTracker(db *sql.DB, config *TestReliabilityConfig) *TestReliabilityTracker {
	if config == nil {
		config = DefaultTestReliabilityConfig()
	}

	return &TestReliabilityTracker{
		db:                db,
		config:            config,
		patternAnalyzer:   NewFailurePatternAnalyzer(db),
		remediationEngine: NewRemediationEngine(),
		notificationMgr:   &SimpleNotificationManager{},
	}
}

// TrackTestExecution records a test execution and updates reliability metrics
func (t *TestReliabilityTracker) TrackTestExecution(execution *TestExecutionRecord) error {
	// Store the execution record
	if err := t.storeExecutionRecord(execution); err != nil {
		return fmt.Errorf("failed to store execution record: %w", err)
	}

	// Update reliability metrics
	if err := t.updateReliabilityMetrics(execution.TestName, execution.TestSuite); err != nil {
		log.Printf("Failed to update reliability metrics for %s: %v", execution.TestName, err)
	}

	// Analyze failure patterns if this was a failure
	if execution.Status == "failed" || execution.Status == "error" {
		if err := t.analyzeFailurePatterns(execution); err != nil {
			log.Printf("Failed to analyze failure patterns for %s: %v", execution.TestName, err)
		}
	}

	return nil
}

// GetTestReliabilityMetrics returns comprehensive reliability metrics for a test
func (t *TestReliabilityTracker) GetTestReliabilityMetrics(testName, testSuite string) (*TestReliabilityMetrics, error) {
	// Get recent executions
	executions, err := t.getRecentExecutions(testName, testSuite, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent executions: %w", err)
	}

	if len(executions) < t.config.MinExecutionsForAnalysis {
		return &TestReliabilityMetrics{
			TestName:  testName,
			TestSuite: testSuite,
			ReliabilityScore: 1.0, // Default to reliable until proven otherwise
			FlakinessScore:   0.0,
			StabilityTrend:   "stable",
			LastUpdated:      time.Now(),
		}, nil
	}

	metrics := &TestReliabilityMetrics{
		TestName:    testName,
		TestSuite:   testSuite,
		LastUpdated: time.Now(),
	}

	// Calculate basic metrics
	t.calculateBasicMetrics(metrics, executions)

	// Calculate reliability and flakiness scores
	t.calculateReliabilityScores(metrics, executions)

	// Analyze stability trend
	t.analyzeStabilityTrend(metrics, executions)

	// Analyze failure patterns
	patterns, err := t.patternAnalyzer.AnalyzeFailurePatterns(testName, testSuite, executions)
	if err != nil {
		log.Printf("Failed to analyze failure patterns: %v", err)
	} else {
		metrics.FailurePatterns = patterns
	}

	// Analyze environment impact
	t.analyzeEnvironmentImpact(metrics, executions)

	// Analyze time-of-day impact
	t.analyzeTimeOfDayImpact(metrics, executions)

	// Generate recent performance summary
	t.generateRecentPerformanceSummary(metrics, executions)

	return metrics, nil
}

// GetFlakyTests returns tests that are currently considered flaky
func (t *TestReliabilityTracker) GetFlakyTests() ([]TestReliabilityMetrics, error) {
	query := `
		SELECT DISTINCT test_name, test_suite
		FROM test_executions
		WHERE start_time >= $1
		GROUP BY test_name, test_suite
		HAVING COUNT(*) >= $2
	`

	cutoff := time.Now().Add(-t.config.AnalysisWindow)
	rows, err := t.db.Query(query, cutoff, t.config.MinExecutionsForAnalysis)
	if err != nil {
		return nil, fmt.Errorf("failed to get test list: %w", err)
	}
	defer rows.Close()

	var flakyTests []TestReliabilityMetrics
	for rows.Next() {
		var testName, testSuite string
		if err := rows.Scan(&testName, &testSuite); err != nil {
			continue
		}

		metrics, err := t.GetTestReliabilityMetrics(testName, testSuite)
		if err != nil {
			log.Printf("Failed to get metrics for %s: %v", testName, err)
			continue
		}

		if metrics.FlakinessScore >= t.config.FlakinessThreshold {
			flakyTests = append(flakyTests, *metrics)
		}
	}

	// Sort by flakiness score (highest first)
	sort.Slice(flakyTests, func(i, j int) bool {
		return flakyTests[i].FlakinessScore > flakyTests[j].FlakinessScore
	})

	return flakyTests, nil
}

// QuarantineTest quarantines a flaky test
func (t *TestReliabilityTracker) QuarantineTest(testName, testSuite string, reason string) error {
	query := `
		INSERT INTO test_quarantine (test_name, test_suite, quarantined_at, reason, status)
		VALUES ($1, $2, NOW(), $3, 'quarantined')
		ON CONFLICT (test_name, test_suite) DO UPDATE SET
			quarantined_at = NOW(),
			reason = EXCLUDED.reason,
			status = 'quarantined'
	`

	_, err := t.db.Exec(query, testName, testSuite, reason)
	if err != nil {
		return fmt.Errorf("failed to quarantine test: %w", err)
	}

	// Generate remediation suggestions
	suggestions, err := t.remediationEngine.GenerateRemediationSuggestions(testName, testSuite)
	if err != nil {
		log.Printf("Failed to generate remediation suggestions for %s: %v", testName, err)
	} else {
		t.storeRemediationSuggestions(testName, testSuite, suggestions)
	}

	// Send notification
	t.notificationMgr.NotifyTestQuarantined(testName, testSuite, reason)

	log.Printf("Quarantined flaky test: %s.%s - %s", testSuite, testName, reason)
	return nil
}

// ReintegrateTest reintegrates a quarantined test
func (t *TestReliabilityTracker) ReintegrateTest(testName, testSuite string) error {
	// Check if test has been quarantined long enough
	query := `
		SELECT quarantined_at FROM test_quarantine
		WHERE test_name = $1 AND test_suite = $2 AND status = 'quarantined'
	`

	var quarantinedAt time.Time
	err := t.db.QueryRow(query, testName, testSuite).Scan(&quarantinedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("test %s.%s is not quarantined", testSuite, testName)
		}
		return fmt.Errorf("failed to check quarantine status: %w", err)
	}

	if time.Since(quarantinedAt) < t.config.QuarantineCooldown {
		return fmt.Errorf("test %s.%s is still in cooldown period", testSuite, testName)
	}

	// Update quarantine status
	updateQuery := `
		UPDATE test_quarantine 
		SET status = 'reintegrated', reintegrated_at = NOW()
		WHERE test_name = $1 AND test_suite = $2 AND status = 'quarantined'
	`

	_, err = t.db.Exec(updateQuery, testName, testSuite)
	if err != nil {
		return fmt.Errorf("failed to reintegrate test: %w", err)
	}

	// Send notification
	t.notificationMgr.NotifyTestReintegrated(testName, testSuite)

	log.Printf("Reintegrated test: %s.%s", testSuite, testName)
	return nil
}

// GenerateStabilityReport generates a comprehensive stability report
func (t *TestReliabilityTracker) GenerateStabilityReport() (*TestStabilityReport, error) {
	report := &TestStabilityReport{
		GeneratedAt:        time.Now(),
		TestsByReliability: make(map[string][]TestReliabilityMetrics),
	}

	// Get all tests with sufficient executions
	allTests, err := t.getAllTestsWithMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get test metrics: %w", err)
	}

	report.TotalTests = len(allTests)

	// Categorize tests by reliability
	for _, test := range allTests {
		if test.ReliabilityScore >= 0.9 {
			report.TestsByReliability["high"] = append(report.TestsByReliability["high"], test)
			report.StableTests++
		} else if test.ReliabilityScore >= 0.7 {
			report.TestsByReliability["medium"] = append(report.TestsByReliability["medium"], test)
		} else {
			report.TestsByReliability["low"] = append(report.TestsByReliability["low"], test)
			report.FlakyTests++
		}

		if test.FlakinessScore >= t.config.FlakinessThreshold {
			report.QuarantinedTests++
		}
	}

	// Calculate overall stability
	if report.TotalTests > 0 {
		report.OverallStability = float64(report.StableTests) / float64(report.TotalTests)
	}

	// Analyze stability trend
	report.StabilityTrend = t.analyzeOverallStabilityTrend(allTests)

	// Generate environment analysis
	report.EnvironmentAnalysis = t.analyzeEnvironmentStability(allTests)

	// Generate recommended actions
	report.RecommendedActions = t.generateRecommendedActions(allTests)

	return report, nil
}

// Private helper methods

func (t *TestReliabilityTracker) storeExecutionRecord(execution *TestExecutionRecord) error {
	query := `
		INSERT INTO test_executions (
			test_name, test_suite, status, duration, start_time, end_time,
			error_message, environment, build_id, commit_hash, branch
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := t.db.Exec(query,
		execution.TestName, execution.TestSuite, execution.Status,
		execution.Duration.Nanoseconds(), execution.StartTime, execution.EndTime,
		execution.ErrorMessage, execution.Environment, execution.BuildID,
		execution.CommitHash, execution.Branch)

	return err
}

func (t *TestReliabilityTracker) getRecentExecutions(testName, testSuite string, limit int) ([]TestExecutionRecord, error) {
	query := `
		SELECT test_name, test_suite, status, duration, start_time, end_time,
			   error_message, environment, build_id, commit_hash, branch
		FROM test_executions
		WHERE test_name = $1 AND test_suite = $2
		  AND start_time >= $3
		ORDER BY start_time DESC
		LIMIT $4
	`

	cutoff := time.Now().Add(-t.config.AnalysisWindow)
	rows, err := t.db.Query(query, testName, testSuite, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []TestExecutionRecord
	for rows.Next() {
		var exec TestExecutionRecord
		var duration int64

		err := rows.Scan(&exec.TestName, &exec.TestSuite, &exec.Status,
			&duration, &exec.StartTime, &exec.EndTime, &exec.ErrorMessage,
			&exec.Environment, &exec.BuildID, &exec.CommitHash, &exec.Branch)
		if err != nil {
			continue
		}

		exec.Duration = time.Duration(duration)
		executions = append(executions, exec)
	}

	return executions, nil
}

func (t *TestReliabilityTracker) calculateBasicMetrics(metrics *TestReliabilityMetrics, executions []TestExecutionRecord) {
	metrics.TotalExecutions = int64(len(executions))

	var totalDuration time.Duration
	var durations []float64

	for _, exec := range executions {
		switch exec.Status {
		case "passed":
			metrics.SuccessfulExecutions++
		case "failed":
			metrics.FailedExecutions++
		case "error":
			metrics.ErrorExecutions++
		case "skipped":
			metrics.SkippedExecutions++
		}

		totalDuration += exec.Duration
		durations = append(durations, float64(exec.Duration.Nanoseconds()))
	}

	if metrics.TotalExecutions > 0 {
		metrics.AverageDuration = totalDuration / time.Duration(metrics.TotalExecutions)
		metrics.DurationVariance = calculateVariance(durations)
	}
}

func (t *TestReliabilityTracker) calculateReliabilityScores(metrics *TestReliabilityMetrics, executions []TestExecutionRecord) {
	if metrics.TotalExecutions == 0 {
		return
	}

	// Calculate basic reliability score
	successRate := float64(metrics.SuccessfulExecutions) / float64(metrics.TotalExecutions)
	metrics.ReliabilityScore = successRate

	// Calculate flakiness score using advanced pattern analysis
	metrics.FlakinessScore = t.calculateAdvancedFlakinessScore(executions)
}

func (t *TestReliabilityTracker) calculateAdvancedFlakinessScore(executions []TestExecutionRecord) float64 {
	if len(executions) < 5 {
		return 0.0
	}

	// Analyze different flakiness patterns
	intermittencyScore := t.calculateIntermittencyScore(executions)
	consecutiveFailureScore := t.calculateConsecutiveFailureScore(executions)
	environmentVariabilityScore := t.calculateEnvironmentVariabilityScore(executions)
	timingVariabilityScore := t.calculateTimingVariabilityScore(executions)

	// Weighted combination of different flakiness indicators
	flakinessScore := (intermittencyScore*0.4 + 
					  consecutiveFailureScore*0.3 + 
					  environmentVariabilityScore*0.2 + 
					  timingVariabilityScore*0.1)

	return math.Min(flakinessScore, 1.0)
}

func (t *TestReliabilityTracker) calculateIntermittencyScore(executions []TestExecutionRecord) float64 {
	if len(executions) < 5 {
		return 0.0
	}

	// Look for alternating pass/fail patterns
	transitions := 0
	for i := 1; i < len(executions); i++ {
		prev := executions[i-1].Status
		curr := executions[i].Status

		if (prev == "passed" && (curr == "failed" || curr == "error")) ||
		   ((prev == "failed" || prev == "error") && curr == "passed") {
			transitions++
		}
	}

	// Higher transition rate indicates more intermittent behavior
	transitionRate := float64(transitions) / float64(len(executions)-1)
	
	// Score increases with transition rate, but caps at reasonable level
	return math.Min(transitionRate * 2.0, 1.0)
}

func (t *TestReliabilityTracker) calculateConsecutiveFailureScore(executions []TestExecutionRecord) float64 {
	maxConsecutive := 0
	currentConsecutive := 0
	totalFailures := 0

	for _, exec := range executions {
		if exec.Status == "failed" || exec.Status == "error" {
			currentConsecutive++
			totalFailures++
			if currentConsecutive > maxConsecutive {
				maxConsecutive = currentConsecutive
			}
		} else {
			currentConsecutive = 0
		}
	}

	if totalFailures == 0 {
		return 0.0
	}

	// If all failures are consecutive, it's likely a persistent issue, not flakiness
	if maxConsecutive == totalFailures {
		return 0.1 // Low flakiness score for persistent failures
	}

	// Moderate consecutive failures indicate some flakiness
	consecutiveRatio := float64(maxConsecutive) / float64(len(executions))
	return math.Min(consecutiveRatio * 1.5, 0.8)
}

func (t *TestReliabilityTracker) calculateEnvironmentVariabilityScore(executions []TestExecutionRecord) float64 {
	envFailureRates := make(map[string][]bool)

	for _, exec := range executions {
		if exec.Environment == "" {
			continue
		}
		
		failed := exec.Status == "failed" || exec.Status == "error"
		envFailureRates[exec.Environment] = append(envFailureRates[exec.Environment], failed)
	}

	if len(envFailureRates) < 2 {
		return 0.0 // Can't compare environments
	}

	// Calculate variance in failure rates across environments
	var failureRates []float64
	for _, failures := range envFailureRates {
		if len(failures) < 3 {
			continue // Need sufficient data per environment
		}
		
		failureCount := 0
		for _, failed := range failures {
			if failed {
				failureCount++
			}
		}
		
		failureRate := float64(failureCount) / float64(len(failures))
		failureRates = append(failureRates, failureRate)
	}

	if len(failureRates) < 2 {
		return 0.0
	}

	variance := calculateVariance(failureRates)
	return math.Min(variance * 3.0, 1.0) // Scale variance to 0-1 range
}

func (t *TestReliabilityTracker) calculateTimingVariabilityScore(executions []TestExecutionRecord) float64 {
	if len(executions) < 10 {
		return 0.0
	}

	// Analyze if failures correlate with execution time
	var passDurations, failDurations []float64

	for _, exec := range executions {
		duration := float64(exec.Duration.Nanoseconds())
		if exec.Status == "passed" {
			passDurations = append(passDurations, duration)
		} else if exec.Status == "failed" || exec.Status == "error" {
			failDurations = append(failDurations, duration)
		}
	}

	if len(passDurations) < 3 || len(failDurations) < 3 {
		return 0.0
	}

	passAvg := calculateMean(passDurations)
	failAvg := calculateMean(failDurations)
	
	// If failed tests have significantly different duration, it might indicate timing issues
	if passAvg > 0 {
		durationDifference := math.Abs(failAvg - passAvg) / passAvg
		return math.Min(durationDifference, 0.5) // Cap at 0.5 for timing-related flakiness
	}

	return 0.0
}

func (t *TestReliabilityTracker) analyzeStabilityTrend(metrics *TestReliabilityMetrics, executions []TestExecutionRecord) {
	if len(executions) < 20 {
		metrics.StabilityTrend = "stable"
		return
	}

	// Split executions into recent and older halves
	mid := len(executions) / 2
	recentExecutions := executions[:mid]
	olderExecutions := executions[mid:]

	recentSuccessRate := t.calculateSuccessRate(recentExecutions)
	olderSuccessRate := t.calculateSuccessRate(olderExecutions)

	difference := recentSuccessRate - olderSuccessRate

	if difference > 0.1 {
		metrics.StabilityTrend = "improving"
	} else if difference < -0.1 {
		metrics.StabilityTrend = "degrading"
	} else {
		metrics.StabilityTrend = "stable"
	}
}

func (t *TestReliabilityTracker) calculateSuccessRate(executions []TestExecutionRecord) float64 {
	if len(executions) == 0 {
		return 0.0
	}

	successful := 0
	for _, exec := range executions {
		if exec.Status == "passed" {
			successful++
		}
	}

	return float64(successful) / float64(len(executions))
}

func (t *TestReliabilityTracker) analyzeEnvironmentImpact(metrics *TestReliabilityMetrics, executions []TestExecutionRecord) {
	metrics.EnvironmentImpact = make(map[string]float64)
	envStats := make(map[string]struct{ total, failures int })

	for _, exec := range executions {
		if exec.Environment == "" {
			continue
		}

		stats := envStats[exec.Environment]
		stats.total++
		if exec.Status == "failed" || exec.Status == "error" {
			stats.failures++
		}
		envStats[exec.Environment] = stats
	}

	for env, stats := range envStats {
		if stats.total >= 3 { // Need minimum executions for meaningful data
			failureRate := float64(stats.failures) / float64(stats.total)
			metrics.EnvironmentImpact[env] = failureRate
		}
	}
}

func (t *TestReliabilityTracker) analyzeTimeOfDayImpact(metrics *TestReliabilityMetrics, executions []TestExecutionRecord) {
	metrics.TimeOfDayImpact = make(map[string]float64)
	hourStats := make(map[int]struct{ total, failures int })

	for _, exec := range executions {
		hour := exec.StartTime.Hour()
		stats := hourStats[hour]
		stats.total++
		if exec.Status == "failed" || exec.Status == "error" {
			stats.failures++
		}
		hourStats[hour] = stats
	}

	for hour, stats := range hourStats {
		if stats.total >= 3 { // Need minimum executions for meaningful data
			failureRate := float64(stats.failures) / float64(stats.total)
			hourKey := fmt.Sprintf("%02d:00", hour)
			metrics.TimeOfDayImpact[hourKey] = failureRate
		}
	}
}

func (t *TestReliabilityTracker) generateRecentPerformanceSummary(metrics *TestReliabilityMetrics, executions []TestExecutionRecord) {
	// Group executions by date
	dailyStats := make(map[string]struct {
		executions int
		failures   int
		durations  []time.Duration
	})

	for _, exec := range executions {
		dateKey := exec.StartTime.Format("2006-01-02")
		stats := dailyStats[dateKey]
		stats.executions++
		if exec.Status == "failed" || exec.Status == "error" {
			stats.failures++
		}
		stats.durations = append(stats.durations, exec.Duration)
		dailyStats[dateKey] = stats
	}

	// Convert to summary format
	for dateStr, stats := range dailyStats {
		date, _ := time.Parse("2006-01-02", dateStr)
		
		var avgDuration time.Duration
		if len(stats.durations) > 0 {
			var total time.Duration
			for _, d := range stats.durations {
				total += d
			}
			avgDuration = total / time.Duration(len(stats.durations))
		}

		failureRate := 0.0
		if stats.executions > 0 {
			failureRate = float64(stats.failures) / float64(stats.executions)
		}

		summary := ExecutionSummary{
			Date:        date,
			Executions:  stats.executions,
			Failures:    stats.failures,
			FailureRate: failureRate,
			AvgDuration: avgDuration,
		}

		metrics.RecentPerformance = append(metrics.RecentPerformance, summary)
	}

	// Sort by date (most recent first)
	sort.Slice(metrics.RecentPerformance, func(i, j int) bool {
		return metrics.RecentPerformance[i].Date.After(metrics.RecentPerformance[j].Date)
	})

	// Keep only last 30 days
	if len(metrics.RecentPerformance) > 30 {
		metrics.RecentPerformance = metrics.RecentPerformance[:30]
	}
}

// Helper functions

func calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}

	mean := calculateMean(values)
	var sumSquaredDiffs float64

	for _, value := range values {
		diff := value - mean
		sumSquaredDiffs += diff * diff
	}

	return sumSquaredDiffs / float64(len(values)-1)
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	var sum float64
	for _, value := range values {
		sum += value
	}

	return sum / float64(len(values))
}

// Additional helper methods for comprehensive analysis

func (t *TestReliabilityTracker) updateReliabilityMetrics(testName, testSuite string) error {
	metrics, err := t.GetTestReliabilityMetrics(testName, testSuite)
	if err != nil {
		return err
	}

	// Store updated metrics in database
	return t.storeReliabilityMetrics(metrics)
}

func (t *TestReliabilityTracker) storeReliabilityMetrics(metrics *TestReliabilityMetrics) error {
	patternsJSON, _ := json.Marshal(metrics.FailurePatterns)
	envImpactJSON, _ := json.Marshal(metrics.EnvironmentImpact)
	timeImpactJSON, _ := json.Marshal(metrics.TimeOfDayImpact)
	recentPerfJSON, _ := json.Marshal(metrics.RecentPerformance)

	query := `
		INSERT INTO test_reliability_metrics (
			test_name, test_suite, reliability_score, flakiness_score, stability_trend,
			total_executions, successful_executions, failed_executions, error_executions,
			skipped_executions, average_duration, duration_variance, failure_patterns,
			environment_impact, time_of_day_impact, recent_performance, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (test_name, test_suite) DO UPDATE SET
			reliability_score = EXCLUDED.reliability_score,
			flakiness_score = EXCLUDED.flakiness_score,
			stability_trend = EXCLUDED.stability_trend,
			total_executions = EXCLUDED.total_executions,
			successful_executions = EXCLUDED.successful_executions,
			failed_executions = EXCLUDED.failed_executions,
			error_executions = EXCLUDED.error_executions,
			skipped_executions = EXCLUDED.skipped_executions,
			average_duration = EXCLUDED.average_duration,
			duration_variance = EXCLUDED.duration_variance,
			failure_patterns = EXCLUDED.failure_patterns,
			environment_impact = EXCLUDED.environment_impact,
			time_of_day_impact = EXCLUDED.time_of_day_impact,
			recent_performance = EXCLUDED.recent_performance,
			last_updated = EXCLUDED.last_updated
	`

	_, err := t.db.Exec(query,
		metrics.TestName, metrics.TestSuite, metrics.ReliabilityScore, metrics.FlakinessScore,
		metrics.StabilityTrend, metrics.TotalExecutions, metrics.SuccessfulExecutions,
		metrics.FailedExecutions, metrics.ErrorExecutions, metrics.SkippedExecutions,
		metrics.AverageDuration.Nanoseconds(), metrics.DurationVariance,
		string(patternsJSON), string(envImpactJSON), string(timeImpactJSON),
		string(recentPerfJSON), metrics.LastUpdated)

	return err
}

func (t *TestReliabilityTracker) analyzeFailurePatterns(execution *TestExecutionRecord) error {
	return t.patternAnalyzer.AnalyzeNewFailure(execution)
}

func (t *TestReliabilityTracker) getAllTestsWithMetrics() ([]TestReliabilityMetrics, error) {
	query := `
		SELECT DISTINCT test_name, test_suite
		FROM test_executions
		WHERE start_time >= $1
		GROUP BY test_name, test_suite
		HAVING COUNT(*) >= $2
	`

	cutoff := time.Now().Add(-t.config.AnalysisWindow)
	rows, err := t.db.Query(query, cutoff, t.config.MinExecutionsForAnalysis)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allTests []TestReliabilityMetrics
	for rows.Next() {
		var testName, testSuite string
		if err := rows.Scan(&testName, &testSuite); err != nil {
			continue
		}

		metrics, err := t.GetTestReliabilityMetrics(testName, testSuite)
		if err != nil {
			log.Printf("Failed to get metrics for %s: %v", testName, err)
			continue
		}

		allTests = append(allTests, *metrics)
	}

	return allTests, nil
}

func (t *TestReliabilityTracker) analyzeOverallStabilityTrend(tests []TestReliabilityMetrics) string {
	if len(tests) == 0 {
		return "stable"
	}

	improving := 0
	degrading := 0

	for _, test := range tests {
		switch test.StabilityTrend {
		case "improving":
			improving++
		case "degrading":
			degrading++
		}
	}

	if improving > degrading*2 {
		return "improving"
	} else if degrading > improving*2 {
		return "degrading"
	}

	return "stable"
}

func (t *TestReliabilityTracker) analyzeEnvironmentStability(tests []TestReliabilityMetrics) EnvironmentStabilityAnalysis {
	envStability := make(map[string][]float64)
	
	for _, test := range tests {
		for env, failureRate := range test.EnvironmentImpact {
			envStability[env] = append(envStability[env], failureRate)
		}
	}

	analysis := EnvironmentStabilityAnalysis{
		StabilityByEnv: make(map[string]float64),
	}

	for env, failureRates := range envStability {
		if len(failureRates) >= 3 {
			avgFailureRate := calculateMean(failureRates)
			stability := 1.0 - avgFailureRate // Convert failure rate to stability score
			
			analysis.Environments = append(analysis.Environments, env)
			analysis.StabilityByEnv[env] = stability

			if stability < 0.8 {
				analysis.ProblematicEnvs = append(analysis.ProblematicEnvs, env)
			}
		}
	}

	// Generate optimization recommendations
	if len(analysis.ProblematicEnvs) > 0 {
		analysis.RecommendedOptimizations = []string{
			"Review environment configuration for unstable environments",
			"Consider resource allocation adjustments",
			"Implement environment-specific test timeouts",
			"Add environment health monitoring",
		}
	}

	return analysis
}

func (t *TestReliabilityTracker) generateRecommendedActions(tests []TestReliabilityMetrics) []RemediationSuggestion {
	var actions []RemediationSuggestion

	flakyCount := 0
	for _, test := range tests {
		if test.FlakinessScore >= t.config.FlakinessThreshold {
			flakyCount++
		}
	}

	if flakyCount > len(tests)/10 { // More than 10% flaky tests
		actions = append(actions, RemediationSuggestion{
			Type:        "system",
			Priority:    "high",
			Description: "High number of flaky tests detected",
			Action:      "Review test infrastructure and environment stability",
			Confidence:  0.9,
			CreatedAt:   time.Now(),
		})
	}

	return actions
}

func (t *TestReliabilityTracker) storeRemediationSuggestions(testName, testSuite string, suggestions []RemediationSuggestion) error {
	for _, suggestion := range suggestions {
		suggestionJSON, _ := json.Marshal(suggestion)
		
		query := `
			INSERT INTO test_remediation_suggestions (test_name, test_suite, suggestion, created_at)
			VALUES ($1, $2, $3, $4)
		`
		
		_, err := t.db.Exec(query, testName, testSuite, string(suggestionJSON), suggestion.CreatedAt)
		if err != nil {
			log.Printf("Failed to store remediation suggestion: %v", err)
		}
	}
	
	return nil
}