package testing

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"
)

// FlakyTestDetector detects and manages flaky tests
type FlakyTestDetector struct {
	db *sql.DB
}

// NewFlakyTestDetector creates a new flaky test detector
func NewFlakyTestDetector(db *sql.DB) *FlakyTestDetector {
	return &FlakyTestDetector{db: db}
}

// UpdateTestFlakiness updates flakiness information for a test
func (f *FlakyTestDetector) UpdateTestFlakiness(execution *TestExecutionRecord) error {
	// Get recent executions for this test (last 50 runs)
	recentExecutions, err := f.getRecentExecutions(execution.TestName, execution.TestSuite, 50)
	if err != nil {
		return fmt.Errorf("failed to get recent executions: %w", err)
	}

	if len(recentExecutions) < 5 {
		// Not enough data to calculate flakiness
		return nil
	}

	// Calculate flakiness score
	flakinessScore := f.calculateFlakinessScore(recentExecutions)
	
	// Update or insert flakiness record
	query := `
		INSERT INTO test_flakiness (test_name, test_suite, flakiness_score, total_runs, failure_count, last_failure, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (test_name) DO UPDATE SET
			flakiness_score = EXCLUDED.flakiness_score,
			total_runs = EXCLUDED.total_runs,
			failure_count = EXCLUDED.failure_count,
			last_failure = EXCLUDED.last_failure,
			updated_at = NOW()
	`

	totalRuns := int64(len(recentExecutions))
	failureCount := f.countFailures(recentExecutions)
	var lastFailure *time.Time
	
	for _, exec := range recentExecutions {
		if exec.Status == "failed" || exec.Status == "error" {
			if lastFailure == nil || exec.StartTime.After(*lastFailure) {
				lastFailure = &exec.StartTime
			}
		}
	}

	_, err = f.db.Exec(query, execution.TestName, execution.TestSuite, 
		flakinessScore, totalRuns, failureCount, lastFailure)
	
	if err != nil {
		return fmt.Errorf("failed to update test flakiness: %w", err)
	}

	// Check if test should be quarantined
	if flakinessScore > 0.3 {
		f.quarantineTest(execution.TestName, execution.TestSuite, flakinessScore)
	}

	return nil
}

// UpdateFlakinessScores updates flakiness scores for all tests
func (f *FlakyTestDetector) UpdateFlakinessScores() error {
	// Get all unique test names that have recent executions
	query := `
		SELECT DISTINCT test_name, test_suite
		FROM test_executions
		WHERE start_time >= NOW() - INTERVAL '7 days'
	`

	rows, err := f.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to get test names: %w", err)
	}
	defer rows.Close()

	var tests []struct {
		name  string
		suite string
	}

	for rows.Next() {
		var test struct {
			name  string
			suite string
		}
		if err := rows.Scan(&test.name, &test.suite); err != nil {
			continue
		}
		tests = append(tests, test)
	}

	// Update flakiness for each test
	for _, test := range tests {
		execution := &TestExecutionRecord{
			TestName:  test.name,
			TestSuite: test.suite,
		}
		if err := f.UpdateTestFlakiness(execution); err != nil {
			log.Printf("Failed to update flakiness for test %s: %v", test.name, err)
		}
	}

	return nil
}

// getRecentExecutions gets recent executions for a test
func (f *FlakyTestDetector) getRecentExecutions(testName, testSuite string, limit int) ([]TestExecutionRecord, error) {
	query := `
		SELECT test_name, test_suite, status, duration, start_time, end_time, error_message
		FROM test_executions
		WHERE test_name = $1 AND test_suite = $2
		ORDER BY start_time DESC
		LIMIT $3
	`

	rows, err := f.db.Query(query, testName, testSuite, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []TestExecutionRecord
	for rows.Next() {
		var exec TestExecutionRecord
		var duration int64
		
		err := rows.Scan(&exec.TestName, &exec.TestSuite, &exec.Status, 
			&duration, &exec.StartTime, &exec.EndTime, &exec.ErrorMessage)
		if err != nil {
			continue
		}
		
		exec.Duration = time.Duration(duration)
		executions = append(executions, exec)
	}

	return executions, nil
}

// calculateFlakinessScore calculates flakiness score based on execution history
func (f *FlakyTestDetector) calculateFlakinessScore(executions []TestExecutionRecord) float64 {
	if len(executions) < 5 {
		return 0.0
	}

	// Count failures and calculate patterns
	failures := 0
	consecutiveFailures := 0
	maxConsecutiveFailures := 0
	currentStreak := 0
	
	for i, exec := range executions {
		if exec.Status == "failed" || exec.Status == "error" {
			failures++
			currentStreak++
			if currentStreak > maxConsecutiveFailures {
				maxConsecutiveFailures = currentStreak
			}
		} else {
			if currentStreak > 0 {
				consecutiveFailures++
			}
			currentStreak = 0
		}
		
		// Weight recent failures more heavily
		if i < 10 && (exec.Status == "failed" || exec.Status == "error") {
			failures += 1 // Double weight for recent failures
		}
	}

	total := float64(len(executions))
	failureRate := float64(failures) / total
	
	// Calculate flakiness score (0.0 to 1.0)
	// Base score from failure rate
	score := failureRate
	
	// Increase score for intermittent failures (flaky pattern)
	if consecutiveFailures > 0 && failures < len(executions) {
		intermittencyBonus := float64(consecutiveFailures) / total * 0.5
		score += intermittencyBonus
	}
	
	// Increase score for consecutive failure streaks
	if maxConsecutiveFailures > 1 && maxConsecutiveFailures < len(executions) {
		streakPenalty := float64(maxConsecutiveFailures) / total * 0.3
		score += streakPenalty
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return math.Round(score*10000) / 10000 // Round to 4 decimal places
}

// countFailures counts the number of failures in executions
func (f *FlakyTestDetector) countFailures(executions []TestExecutionRecord) int64 {
	count := int64(0)
	for _, exec := range executions {
		if exec.Status == "failed" || exec.Status == "error" {
			count++
		}
	}
	return count
}

// quarantineTest quarantines a flaky test
func (f *FlakyTestDetector) quarantineTest(testName, testSuite string, flakinessScore float64) error {
	log.Printf("Quarantining flaky test: %s (score: %.4f)", testName, flakinessScore)
	
	query := `
		UPDATE test_flakiness 
		SET status = 'quarantined', updated_at = NOW()
		WHERE test_name = $1 AND test_suite = $2
	`
	
	_, err := f.db.Exec(query, testName, testSuite)
	if err != nil {
		return fmt.Errorf("failed to quarantine test: %w", err)
	}

	// TODO: Integrate with CI/CD to actually skip quarantined tests
	// This would require updating the test runner configuration

	return nil
}

// GetQuarantinedTests returns currently quarantined tests
func (f *FlakyTestDetector) GetQuarantinedTests() ([]FlakyTestInfo, error) {
	query := `
		SELECT test_name, test_suite, flakiness_score, total_runs, failure_count, last_failure, status
		FROM test_flakiness
		WHERE status = 'quarantined'
		ORDER BY flakiness_score DESC
	`

	rows, err := f.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get quarantined tests: %w", err)
	}
	defer rows.Close()

	var tests []FlakyTestInfo
	for rows.Next() {
		var test FlakyTestInfo
		var lastFailure sql.NullTime

		err := rows.Scan(&test.TestName, &test.TestSuite, &test.FlakinessScore,
			&test.TotalRuns, &test.FailureCount, &lastFailure, &test.Status)
		if err != nil {
			continue
		}

		if lastFailure.Valid {
			test.LastFailure = lastFailure.Time
		}

		if test.TotalRuns > 0 {
			test.FailureRate = float64(test.FailureCount) / float64(test.TotalRuns) * 100
		}

		tests = append(tests, test)
	}

	return tests, nil
}

// ReintegrateTest reintegrates a quarantined test back into the suite
func (f *FlakyTestDetector) ReintegrateTest(testName, testSuite string) error {
	query := `
		UPDATE test_flakiness 
		SET status = 'active', updated_at = NOW()
		WHERE test_name = $1 AND test_suite = $2 AND status = 'quarantined'
	`
	
	result, err := f.db.Exec(query, testName, testSuite)
	if err != nil {
		return fmt.Errorf("failed to reintegrate test: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("test %s not found in quarantine", testName)
	}

	log.Printf("Reintegrated test: %s", testName)
	return nil
}