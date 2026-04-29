package testing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
)

// TestExecutionMonitor monitors test execution and provides analytics
type TestExecutionMonitor struct {
	db                *sql.DB
	alertThresholds   *AlertThresholds
	executionCache    map[string]*Tlds
	flakyDetector     *FlakyTestDetector
	patternAnalyzer   *FailurePatternAnalyzer
	coverageTracker   *CoverageTracker
	alertManager      *TestAlertManager
	mu                sync.RWMutex
	isMonitoring      bool
	stopChan          chan struct{}
}

// TestExecutionRecord represents a single test execution record for monitoring
type TestExecutionRecord struct {
	ID              int64                  `json:"id" db:"id"`
	TestSuite       string                 `json:"test_suite" db:"test_suite"`
	TestName        string                 `json:"test_name" db:"test_name"`
	Status          string                 `json:"status" db:"status"` // passed, failed, skipped, error
	Duration        time.Duration          `json:"duration" db:"duration"`
	StartTime       time.Time              `json:"start_time" db:"start_time"`
	EndTime         time.Time              `json:"end_time" db:"end_time"`
	ErrorMessage    string                 `json:"error_message" db:"error_message"`
	StackTrace      string                 `json:"stack_trace" db:"stack_trace"`
	Coverage        float64                `json:"coverage" db:"coverage"`
	Environment     string                 `json:"environment" db:"environment"`
	Branch          string                 `json:"branch" db:"branch"`
	CommitHash      string                 `json:"commit_hash" db:"commit_hash"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
}

// ExecutionMetrics represents aggregated test execution metrics
type ExecutionMetrics struct {
	TotalExecutions     int64     `json:"total_executions"`
	PassedExecutions    int64     `json:"passed_executions"`
	FailedExecutions    int64     `json:"failed_executions"`
	SkippedExecutions   int64     `json:"skipped_executions"`
	ErrorExecutions     int64     `json:"error_executions"`
	SuccessRate         float64   `json:"success_rate"`
	AverageDuration     float64   `json:"average_duration_ms"`
	CoveragePercent     float64   `json:"coverage_percent"`
	FlakyTests          int64     `json:"flaky_tests"`
	SlowTests           int64     `json:"slow_tests"`
	TimeRange           string    `json:"time_range"`
	LastUpdated         time.Time `json:"last_updated"`
}

// FlakyTestInfo represents information about a flaky test
type FlakyTestInfo struct {
	TestName        string    `json:"test_name"`
	TestSuite       string    `json:"test_suite"`
	FlakinessScore  float64   `json:"flakiness_score"`
	TotalRuns       int64     `json:"total_runs"`
	FailureCount    int64     `json:"failure_count"`
	LastFailure     time.Time `json:"last_failure"`
	FailureRate     float64   `json:"failure_rate"`
	Status          string    `json:"status"` // active, quarantined, fixed
}

// TestFailurePattern represents a pattern in test failures
type TestFailurePattern struct {
	Pattern         string    `json:"pattern"`
	Count           int64     `json:"count"`
	AffectedTests   []string  `json:"affected_tests"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	Severity        string    `json:"severity"`
	Category        string    `json:"category"` // infrastructure, code, environment
}

// CoverageTrend represents coverage trend over time
type CoverageTrend struct {
	Date            time.Time `json:"date"`
	CoveragePercent float64   `json:"coverage_percent"`
	TestCount       int64     `json:"test_count"`
	LinesTotal      int64     `json:"lines_total"`
	LinesCovered    int64     `json:"lines_covered"`
}

// TestAlert represents a test-related alert
type TestAlert struct {
	ID          int64     `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	TestName    string    `json:"test_name"`
	TestSuite   string    `json:"test_suite"`
	Threshold   float64   `json:"threshold"`
	CurrentValue float64  `json:"current_value"`
	TriggeredAt time.Time `json:"triggered_at"`
	Status      string    `json:"status"`
}

// AlertThresholds defines thresholds for test alerts
type AlertThresholds struct {
	SuccessRateWarning      float64 `json:"success_rate_warning"`      // 95%
	SuccessRateCritical     float64 `json:"success_rate_critical"`     // 90%
	FlakinessScoreWarning   float64 `json:"flakiness_score_warning"`   // 0.1
	FlakinessScoreCritical  float64 `json:"flakiness_score_critical"`  // 0.3
	CoverageDropWarning     float64 `json:"coverage_drop_warning"`     // 5%
	CoverageDropCritical    float64 `json:"coverage_drop_critical"`    // 10%
	DurationIncreaseWarning float64 `json:"duration_increase_warning"` // 50%
	DurationIncreaseCritical float64 `json:"duration_increase_critical"` // 100%
	FailurePatternThreshold int64   `json:"failure_pattern_threshold"` // 5 occurrences
}

// NewTestExecutionMonitor creates a new test execution monitor
func NewTestExecutionMonitor(db *sql.DB) *TestExecutionMonitor {
	monitor := &TestExecutionMonitor{
		db: db,
		alertThresholds: &AlertThresholds{
			SuccessRateWarning:       95.0,
			SuccessRateCritical:      90.0,
			FlakinessScoreWarning:    0.1,
			FlakinessScoreCritical:   0.3,
			CoverageDropWarning:      5.0,
			CoverageDropCritical:     10.0,
			DurationIncreaseWarning:  50.0,
			DurationIncreaseCritical: 100.0,
			FailurePatternThreshold:  5,
		},
		flakyDetector:   NewFlakyTestDetector(db),
		patternAnalyzer: NewFailurePatternAnalyzer(db),
		coverageTracker: NewCoverageTracker(db),
		alertManager:    NewTestAlertManager(db),
		stopChan:        make(chan struct{}),
	}

	if err := monitor.initializeTables(); err != nil {
		log.Printf("Failed to initialize test execution monitor tables: %v", err)
	}

	return monitor
}

// initializeTables creates the necessary database tables
func (m *TestExecutionMonitor) initializeTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS test_executions (
			id BIGSERIAL PRIMARY KEY,
			test_suite VARCHAR(255) NOT NULL,
			test_name VARCHAR(500) NOT NULL,
			status VARCHAR(50) NOT NULL,
			duration BIGINT NOT NULL, -- nanoseconds
			start_time TIMESTAMP WITH TIME ZONE NOT NULL,
			end_time TIMESTAMP WITH TIME ZONE NOT NULL,
			error_message TEXT,
			stack_trace TEXT,
			coverage DECIMAL(5,2) DEFAULT 0,
			environment VARCHAR(100) DEFAULT 'test',
			branch VARCHAR(255),
			commit_hash VARCHAR(64),
			metadata JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_test_executions_test_name ON test_executions(test_name)`,
		`CREATE INDEX IF NOT EXISTS idx_test_executions_start_time ON test_executions(start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_test_executions_status ON test_executions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_test_executions_suite_name ON test_executions(test_suite, test_name)`,
		
		`CREATE TABLE IF NOT EXISTS test_flakiness (
			id BIGSERIAL PRIMARY KEY,
			test_name VARCHAR(500) NOT NULL UNIQUE,
			test_suite VARCHAR(255) NOT NULL,
			flakiness_score DECIMAL(5,4) NOT NULL DEFAULT 0,
			total_runs BIGINT NOT NULL DEFAULT 0,
			failure_count BIGINT NOT NULL DEFAULT 0,
			last_failure TIMESTAMP WITH TIME ZONE,
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS test_alerts (
			id BIGSERIAL PRIMARY KEY,
			type VARCHAR(100) NOT NULL,
			severity VARCHAR(50) NOT NULL,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			test_name VARCHAR(500),
			test_suite VARCHAR(255),
			threshold DECIMAL(10,4),
			current_value DECIMAL(10,4),
			triggered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			resolved_at TIMESTAMP WITH TIME ZONE,
			status VARCHAR(50) DEFAULT 'active'
		)`,
	}

	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// StartMonitoring starts the test execution monitoring
func (m *TestExecutionMonitor) StartMonitoring(ctx context.Context) error {
	m.mu.Lock()
	if m.isMonitoring {
		m.mu.Unlock()
		return fmt.Errorf("monitoring is already running")
	}
	m.isMonitoring = true
	m.mu.Unlock()

	log.Println("Starting test execution monitoring...")

	// Start monitoring goroutines
	go m.monitorTestExecutions(ctx)
	go m.analyzeFailurePatterns(ctx)
	go m.updateFlakinessScores(ctx)
	go m.checkAlertThresholds(ctx)

	return nil
}

// StopMonitoring stops the test execution monitoring
func (m *TestExecutionMonitor) StopMonitoring() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isMonitoring {
		return
	}

	log.Println("Stopping test execution monitoring...")
	close(m.stopChan)
	m.isMonitoring = false
}

// RecordTestExecution records a test execution
func (m *TestExecutionMonitor) RecordTestExecution(execution *TestExecutionRecord) error {
	query := `
		INSERT INTO test_executions (
			test_suite, test_name, status, duration, start_time, end_time,
			error_message, stack_trace, coverage, environment, branch, commit_hash, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at
	`

	metadataJSON, _ := json.Marshal(execution.Metadata)

	err := m.db.QueryRow(query,
		execution.TestSuite, execution.TestName, execution.Status,
		execution.Duration.Nanoseconds(), execution.StartTime, execution.EndTime,
		execution.ErrorMessage, execution.StackTrace, execution.Coverage,
		execution.Environment, execution.Branch, execution.CommitHash, metadataJSON,
	).Scan(&execution.ID, &execution.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to record test execution: %w", err)
	}

	// Update flakiness tracking
	go m.updateTestFlakiness(execution)

	return nil
}

// GetTestMetrics returns test metrics for a given time range
func (m *TestExecutionMonitor) GetTestMetrics(timeRange string) (*ExecutionMetrics, error) {
	var interval string
	switch timeRange {
	case "1h":
		interval = "1 hour"
	case "24h":
		interval = "24 hours"
	case "7d":
		interval = "7 days"
	case "30d":
		interval = "30 days"
	default:
		interval = "24 hours"
		timeRange = "24h"
	}

	query := `
		SELECT 
			COUNT(*) as total_executions,
			COUNT(CASE WHEN status = 'passed' THEN 1 END) as passed_executions,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_executions,
			COUNT(CASE WHEN status = 'skipped' THEN 1 END) as skipped_executions,
			COUNT(CASE WHEN status = 'error' THEN 1 END) as error_executions,
			AVG(duration / 1000000.0) as avg_duration_ms,
			AVG(CASE WHEN coverage > 0 THEN coverage END) as avg_coverage
		FROM test_executions 
		WHERE start_time >= NOW() - INTERVAL '%s'
	`

	var metrics ExecutionMetrics
	var avgDuration, avgCoverage sql.NullFloat64

	err := m.db.QueryRow(fmt.Sprintf(query, interval)).Scan(
		&metrics.TotalExecutions, &metrics.PassedExecutions,
		&metrics.FailedExecutions, &metrics.SkippedExecutions,
		&metrics.ErrorExecutions, &avgDuration, &avgCoverage,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get test metrics: %w", err)
	}

	if metrics.TotalExecutions > 0 {
		metrics.SuccessRate = float64(metrics.PassedExecutions) / float64(metrics.TotalExecutions) * 100
	}

	if avgDuration.Valid {
		metrics.AverageDuration = avgDuration.Float64
	}

	if avgCoverage.Valid {
		metrics.CoveragePercent = avgCoverage.Float64
	}

	// Get flaky test count
	flakyCount, _ := m.getFlakyTestCount()
	metrics.FlakyTests = flakyCount

	// Get slow test count (tests taking > 5 seconds)
	slowCount, _ := m.getSlowTestCount(timeRange)
	metrics.SlowTests = slowCount

	metrics.TimeRange = timeRange
	metrics.LastUpdated = time.Now()

	return &metrics, nil
}

// GetFlakyTests returns information about flaky tests
func (m *TestExecutionMonitor) GetFlakyTests(limit int) ([]FlakyTestInfo, error) {
	query := `
		SELECT test_name, test_suite, flakiness_score, total_runs, failure_count, last_failure, status
		FROM test_flakiness
		WHERE flakiness_score > 0
		ORDER BY flakiness_score DESC
		LIMIT $1
	`

	rows, err := m.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get flaky tests: %w", err)
	}
	defer rows.Close()

	var flakyTests []FlakyTestInfo
	for rows.Next() {
		var test FlakyTestInfo
		var lastFailure sql.NullTime

		err := rows.Scan(
			&test.TestName, &test.TestSuite, &test.FlakinessScore,
			&test.TotalRuns, &test.FailureCount, &lastFailure, &test.Status,
		)
		if err != nil {
			continue
		}

		if lastFailure.Valid {
			test.LastFailure = lastFailure.Time
		}

		if test.TotalRuns > 0 {
			test.FailureRate = float64(test.FailureCount) / float64(test.TotalRuns) * 100
		}

		flakyTests = append(flakyTests, test)
	}

	return flakyTests, nil
}

// GetFailurePatterns returns failure pattern analysis
func (m *TestExecutionMonitor) GetFailurePatterns(limit int) ([]TestFailurePattern, error) {
	return m.patternAnalyzer.GetFailurePatterns(limit)
}

// GetCoverageTrends returns coverage trends over time
func (m *TestExecutionMonitor) GetCoverageTrends(days int) ([]CoverageTrend, error) {
	return m.coverageTracker.GetCoverageTrends(days)
}

// monitorTestExecutions continuously monitors test executions
func (m *TestExecutionMonitor) monitorTestExecutions(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.performHealthChecks()
		}
	}
}

// analyzeFailurePatterns analyzes failure patterns periodically
func (m *TestExecutionMonitor) analyzeFailurePatterns(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.patternAnalyzer.AnalyzePatterns()
		}
	}
}

// updateFlakinessScores updates flakiness scores periodically
func (m *TestExecutionMonitor) updateFlakinessScores(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.flakyDetector.UpdateFlakinessScores()
		}
	}
}

// checkAlertThresholds checks alert thresholds periodically
func (m *TestExecutionMonitor) checkAlertThresholds(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.evaluateAlertThresholds()
		}
	}
}

// performHealthChecks performs various health checks
func (m *TestExecutionMonitor) performHealthChecks() {
	// Check for test execution anomalies
	metrics, err := m.GetTestMetrics("1h")
	if err != nil {
		log.Printf("Failed to get test metrics for health check: %v", err)
		return
	}

	// Check success rate
	if metrics.SuccessRate < m.alertThresholds.SuccessRateCritical {
		m.alertManager.TriggerAlert(&TestAlert{
			Type:         "success_rate",
			Severity:     "critical",
			Title:        "Critical Test Success Rate",
			Description:  fmt.Sprintf("Test success rate dropped to %.2f%%", metrics.SuccessRate),
			Threshold:    m.alertThresholds.SuccessRateCritical,
			CurrentValue: metrics.SuccessRate,
		})
	} else if metrics.SuccessRate < m.alertThresholds.SuccessRateWarning {
		m.alertManager.TriggerAlert(&TestAlert{
			Type:         "success_rate",
			Severity:     "warning",
			Title:        "Low Test Success Rate",
			Description:  fmt.Sprintf("Test success rate dropped to %.2f%%", metrics.SuccessRate),
			Threshold:    m.alertThresholds.SuccessRateWarning,
			CurrentValue: metrics.SuccessRate,
		})
	}

	// Check coverage drop
	if metrics.CoveragePercent > 0 {
		previousCoverage := m.getPreviousCoverage()
		if previousCoverage > 0 {
			coverageDrop := previousCoverage - metrics.CoveragePercent
			if coverageDrop > m.alertThresholds.CoverageDropCritical {
				m.alertManager.TriggerAlert(&TestAlert{
					Type:         "coverage_drop",
					Severity:     "critical",
					Title:        "Critical Coverage Drop",
					Description:  fmt.Sprintf("Code coverage dropped by %.2f%%", coverageDrop),
					Threshold:    m.alertThresholds.CoverageDropCritical,
					CurrentValue: coverageDrop,
				})
			}
		}
	}
}

// evaluateAlertThresholds evaluates all alert thresholds
func (m *TestExecutionMonitor) evaluateAlertThresholds() {
	// Check for flaky tests
	flakyTests, err := m.GetFlakyTests(100)
	if err != nil {
		log.Printf("Failed to get flaky tests for alert evaluation: %v", err)
		return
	}

	for _, test := range flakyTests {
		if test.FlakinessScore > m.alertThresholds.FlakinessScoreCritical {
			m.alertManager.TriggerAlert(&TestAlert{
				Type:         "flaky_test",
				Severity:     "critical",
				Title:        "Critical Flaky Test",
				Description:  fmt.Sprintf("Test %s has high flakiness score: %.4f", test.TestName, test.FlakinessScore),
				TestName:     test.TestName,
				TestSuite:    test.TestSuite,
				Threshold:    m.alertThresholds.FlakinessScoreCritical,
				CurrentValue: test.FlakinessScore,
			})
		}
	}

	// Check failure patterns
	patterns, err := m.GetFailurePatterns(20)
	if err != nil {
		log.Printf("Failed to get failure patterns for alert evaluation: %v", err)
		return
	}

	for _, pattern := range patterns {
		if pattern.Count >= m.alertThresholds.FailurePatternThreshold {
			m.alertManager.TriggerAlert(&TestAlert{
				Type:        "failure_pattern",
				Severity:    "warning",
				Title:       "Recurring Failure Pattern",
				Description: fmt.Sprintf("Pattern '%s' occurred %d times", pattern.Pattern, pattern.Count),
				Threshold:   float64(m.alertThresholds.FailurePatternThreshold),
				CurrentValue: float64(pattern.Count),
			})
		}
	}
}

// Helper methods
func (m *TestExecutionMonitor) updateTestFlakiness(execution *TestExecutionRecord) {
	m.flakyDetector.UpdateTestFlakiness(execution)
}

// getFlakyTestCount returns the count of flaky tests
func (m *TestExecutionMonitor) getFlakyTestCount() (int64, error) {
	var count int64
	err := m.db.QueryRow("SELECT COUNT(*) FROM test_flakiness WHERE flakiness_score > 0.1").Scan(&count)
	return count, err
}

// getSlowTestCount returns the count of slow tests
func (m *TestExecutionMonitor) getSlowTestCount(timeRange string) (int64, error) {
	var interval string
	switch timeRange {
	case "1h":
		interval = "1 hour"
	case "24h":
		interval = "24 hours"
	case "7d":
		interval = "7 days"
	case "30d":
		interval = "30 days"
	default:
		interval = "24 hours"
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM test_executions
		WHERE duration > 5000000000 -- 5 seconds in nanoseconds
		AND start_time >= NOW() - INTERVAL '%s'
	`, interval)

	var count int64
	err := m.db.QueryRow(query).Scan(&count)
	return count, err
}

// getPreviousCoverage returns the previous coverage percentage
func (m *TestExecutionMonitor) getPreviousCoverage() float64 {
	query := `
		SELECT coverage_percent
		FROM coverage_trends
		WHERE date < CURRENT_DATE
		ORDER BY date DESC
		LIMIT 1
	`

	var coverage float64
	err := m.db.QueryRow(query).Scan(&coverage)
	if err != nil {
		return 0
	}
	return coverage
}

// Tlds represents test execution data (placeholder for executionCache)
type Tlds struct {
	TestName     string
	LastExecution time.Time
	SuccessRate  float64
}