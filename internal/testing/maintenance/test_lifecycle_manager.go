package maintenance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// TestLifecycleManager manages the lifecycle of tests
type TestLifecycleManager struct {
	db *sql.DB
}

// NewTestLifecycleManager creates a new test lifecycle manager
func NewTestLifecycleManager(db *sql.DB) *TestLifecycleManager {
	return &TestLifecycleManager{
		db: db,
	}
}

// CreateTest creates a new test and records its lifecycle event
func (tlm *TestLifecycleManager) CreateTest(testID, reason string, metadata map[string]interface{}) error {
	return tlm.recordLifecycleEvent(testID, EventCreated, reason, metadata)
}

// ActivateTest activates a test
func (tlm *TestLifecycleManager) ActivateTest(testID, reason string) error {
	// Update test status
	err := tlm.updateTestStatus(testID, StatusActive)
	if err != nil {
		return fmt.Errorf("failed to update test status: %w", err)
	}

	// Record lifecycle event
	return tlm.recordLifecycleEvent(testID, EventActivated, reason, nil)
}

// DeprecateTest marks a test as deprecated
func (tlm *TestLifecycleManager) DeprecateTest(testID, reason string, metadata map[string]interface{}) error {
	// Update test status
	err := tlm.updateTestStatus(testID, StatusDeprecated)
	if err != nil {
		return fmt.Errorf("failed to update test status: %w", err)
	}

	// Record lifecycle event
	return tlm.recordLifecycleEvent(testID, EventDeprecated, reason, metadata)
}

// ObsoleteTest marks a test as obsolete
func (tlm *TestLifecycleManager) ObsoleteTest(testID, reason string) error {
	// Update test status
	err := tlm.updateTestStatus(testID, StatusObsolete)
	if err != nil {
		return fmt.Errorf("failed to update test status: %w", err)
	}

	// Record lifecycle event
	return tlm.recordLifecycleEvent(testID, EventObsoleted, reason, nil)
}

// QuarantineTest quarantines a test due to issues
func (tlm *TestLifecycleManager) QuarantineTest(testID, reason string, metadata map[string]interface{}) error {
	// Update test status
	err := tlm.updateTestStatus(testID, StatusQuarantined)
	if err != nil {
		return fmt.Errorf("failed to update test status: %w", err)
	}

	// Record lifecycle event
	return tlm.recordLifecycleEvent(testID, EventQuarantined, reason, metadata)
}

// RemoveTest removes a test from the system
func (tlm *TestLifecycleManager) RemoveTest(testID, reason string) error {
	// Record lifecycle event before removal
	err := tlm.recordLifecycleEvent(testID, EventRemoved, reason, nil)
	if err != nil {
		return fmt.Errorf("failed to record removal event: %w", err)
	}

	// Remove test metadata (soft delete by updating status)
	_, err = tlm.db.Exec(`
		UPDATE test_metadata 
		SET status = 'removed', updated_at = $1
		WHERE test_id = $2
	`, time.Now(), testID)

	return err
}

// MigrateTest migrates a test to a new version or framework
func (tlm *TestLifecycleManager) MigrateTest(testID, reason string, metadata map[string]interface{}) error {
	return tlm.recordLifecycleEvent(testID, EventMigrated, reason, metadata)
}

// recordLifecycleEvent records a lifecycle event for a test
func (tlm *TestLifecycleManager) recordLifecycleEvent(testID string, eventType LifecycleEvent, reason string, metadata map[string]interface{}) error {
	eventID := fmt.Sprintf("event_%s_%d", testID, time.Now().Unix())
	
	metadataJSON, _ := json.Marshal(metadata)
	
	_, err := tlm.db.Exec(`
		INSERT INTO test_lifecycle_events (
			event_id, test_id, event_type, timestamp, metadata, reason
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, eventID, testID, string(eventType), time.Now(), metadataJSON, reason)

	if err != nil {
		return fmt.Errorf("failed to record lifecycle event: %w", err)
	}

	return nil
}

// updateTestStatus updates the status of a test
func (tlm *TestLifecycleManager) updateTestStatus(testID string, status TestStatus) error {
	_, err := tlm.db.Exec(`
		UPDATE test_metadata 
		SET status = $1, updated_at = $2
		WHERE test_id = $3
	`, string(status), time.Now(), testID)

	return err
}

// GetTestLifecycle retrieves the complete lifecycle history of a test
func (tlm *TestLifecycleManager) GetTestLifecycle(testID string) ([]TestLifecycleEvent, error) {
	rows, err := tlm.db.Query(`
		SELECT event_id, test_id, event_type, timestamp, metadata, reason
		FROM test_lifecycle_events
		WHERE test_id = $1
		ORDER BY timestamp ASC
	`, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lifecycle events: %w", err)
	}
	defer rows.Close()

	var events []TestLifecycleEvent
	for rows.Next() {
		var event TestLifecycleEvent
		var eventType string
		var metadataJSON []byte

		err := rows.Scan(&event.ID, &event.TestID, &eventType, 
			&event.Timestamp, &metadataJSON, &event.Reason)
		if err != nil {
			log.Printf("Error scanning lifecycle event: %v", err)
			continue
		}

		event.EventType = LifecycleEvent(eventType)
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &event.Metadata)
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

// AnalyzeTestLifecycle analyzes the lifecycle patterns of tests
func (tlm *TestLifecycleManager) AnalyzeTestLifecycle() (*LifecycleAnalysis, error) {
	analysis := &LifecycleAnalysis{
		Timestamp: time.Now(),
		Metrics:   make(map[string]int),
		Trends:    make(map[string][]TimeSeriesPoint),
	}

	// Get test status distribution
	statusRows, err := tlm.db.Query(`
		SELECT status, COUNT(*) as count
		FROM test_metadata
		GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query test status distribution: %w", err)
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var status string
		var count int
		err := statusRows.Scan(&status, &count)
		if err != nil {
			continue
		}
		analysis.Metrics[fmt.Sprintf("status_%s", status)] = count
	}

	// Get lifecycle event trends (last 30 days)
	trendRows, err := tlm.db.Query(`
		SELECT event_type, DATE(timestamp) as event_date, COUNT(*) as count
		FROM test_lifecycle_events
		WHERE timestamp >= $1
		GROUP BY event_type, DATE(timestamp)
		ORDER BY event_date ASC
	`, time.Now().AddDate(0, 0, -30))
	if err != nil {
		return nil, fmt.Errorf("failed to query lifecycle trends: %w", err)
	}
	defer trendRows.Close()

	for trendRows.Next() {
		var eventType, eventDate string
		var count int
		err := trendRows.Scan(&eventType, &eventDate, &count)
		if err != nil {
			continue
		}

		date, _ := time.Parse("2006-01-02", eventDate)
		point := TimeSeriesPoint{
			Timestamp: date,
			Value:     float64(count),
		}

		analysis.Trends[eventType] = append(analysis.Trends[eventType], point)
	}

	// Calculate lifecycle metrics
	analysis.AverageTestAge = tlm.calculateAverageTestAge()
	analysis.DeprecationRate = tlm.calculateDeprecationRate()
	analysis.RemovalRate = tlm.calculateRemovalRate()

	return analysis, nil
}

// LifecycleAnalysis represents analysis of test lifecycle patterns
type LifecycleAnalysis struct {
	Timestamp       time.Time                      `json:"timestamp"`
	Metrics         map[string]int                 `json:"metrics"`
	Trends          map[string][]TimeSeriesPoint   `json:"trends"`
	AverageTestAge  time.Duration                  `json:"average_test_age"`
	DeprecationRate float64                        `json:"deprecation_rate"`
	RemovalRate     float64                        `json:"removal_rate"`
}

// TimeSeriesPoint represents a point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// calculateAverageTestAge calculates the average age of active tests
func (tlm *TestLifecycleManager) calculateAverageTestAge() time.Duration {
	var avgAge sql.NullFloat64
	
	err := tlm.db.QueryRow(`
		SELECT AVG(EXTRACT(EPOCH FROM (NOW() - created_at))) as avg_age_seconds
		FROM test_metadata
		WHERE status = 'active'
	`).Scan(&avgAge)

	if err != nil || !avgAge.Valid {
		return 0
	}

	return time.Duration(avgAge.Float64) * time.Second
}

// calculateDeprecationRate calculates the rate of test deprecation
func (tlm *TestLifecycleManager) calculateDeprecationRate() float64 {
	var deprecatedCount, totalCount int

	tlm.db.QueryRow(`
		SELECT COUNT(*) FROM test_metadata WHERE status = 'deprecated'
	`).Scan(&deprecatedCount)

	tlm.db.QueryRow(`
		SELECT COUNT(*) FROM test_metadata WHERE status != 'removed'
	`).Scan(&totalCount)

	if totalCount == 0 {
		return 0
	}

	return float64(deprecatedCount) / float64(totalCount)
}

// calculateRemovalRate calculates the rate of test removal
func (tlm *TestLifecycleManager) calculateRemovalRate() float64 {
	var removedCount, totalCount int

	// Count tests removed in the last 30 days
	tlm.db.QueryRow(`
		SELECT COUNT(*) FROM test_lifecycle_events 
		WHERE event_type = 'removed' AND timestamp >= $1
	`, time.Now().AddDate(0, 0, -30)).Scan(&removedCount)

	tlm.db.QueryRow(`
		SELECT COUNT(*) FROM test_metadata
	`).Scan(&totalCount)

	if totalCount == 0 {
		return 0
	}

	return float64(removedCount) / float64(totalCount)
}

// ScheduleDeprecation schedules tests for deprecation based on criteria
func (tlm *TestLifecycleManager) ScheduleDeprecation(criteria DeprecationCriteria) ([]string, error) {
	var scheduledTests []string

	// Build query based on criteria
	query := `
		SELECT test_id FROM test_metadata 
		WHERE status = 'active'
	`
	args := []interface{}{}
	argIndex := 1

	if criteria.MaxAge > 0 {
		query += fmt.Sprintf(" AND created_at < $%d", argIndex)
		args = append(args, time.Now().Add(-criteria.MaxAge))
		argIndex++
	}

	if criteria.MinFailureRate > 0 {
		query += fmt.Sprintf(" AND failure_rate > $%d", argIndex)
		args = append(args, criteria.MinFailureRate)
		argIndex++
	}

	if criteria.MaxExecutionCount > 0 {
		query += fmt.Sprintf(" AND execution_count < $%d", argIndex)
		args = append(args, criteria.MaxExecutionCount)
		argIndex++
	}

	rows, err := tlm.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tests for deprecation: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var testID string
		err := rows.Scan(&testID)
		if err != nil {
			continue
		}

		// Schedule for deprecation
		err = tlm.DeprecateTest(testID, "Scheduled deprecation based on criteria", 
			map[string]interface{}{
				"criteria": criteria,
				"scheduled_at": time.Now(),
			})
		if err != nil {
			log.Printf("Failed to deprecate test %s: %v", testID, err)
			continue
		}

		scheduledTests = append(scheduledTests, testID)
	}

	return scheduledTests, rows.Err()
}

// DeprecationCriteria defines criteria for automatic test deprecation
type DeprecationCriteria struct {
	MaxAge            time.Duration `json:"max_age"`
	MinFailureRate    float64       `json:"min_failure_rate"`
	MaxExecutionCount int           `json:"max_execution_count"`
	RequiredTags      []string      `json:"required_tags"`
	ExcludedTags      []string      `json:"excluded_tags"`
}

// GetDeprecatedTests retrieves all deprecated tests
func (tlm *TestLifecycleManager) GetDeprecatedTests() ([]*TestMetadata, error) {
	rows, err := tlm.db.Query(`
		SELECT test_id, file_path, test_name, test_type, 
			   code_coverage, last_modified, last_executed, 
			   execution_count, failure_rate, average_runtime_ms, 
			   complexity, status
		FROM test_metadata
		WHERE status = 'deprecated'
		ORDER BY last_modified DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query deprecated tests: %w", err)
	}
	defer rows.Close()

	var tests []*TestMetadata
	for rows.Next() {
		var test TestMetadata
		var status string
		var runtimeMs int64

		err := rows.Scan(&test.ID, &test.FilePath, &test.TestName, &test.TestType,
			&test.CodeCoverage, &test.LastModified, &test.LastExecuted,
			&test.ExecutionCount, &test.FailureRate, &runtimeMs,
			&test.Complexity, &status)
		if err != nil {
			log.Printf("Error scanning deprecated test: %v", err)
			continue
		}

		test.Status = TestStatus(status)
		test.AverageRuntime = time.Duration(runtimeMs) * time.Millisecond
		tests = append(tests, &test)
	}

	return tests, rows.Err()
}

// CleanupObsoleteTests removes obsolete tests that have been deprecated for a specified period
func (tlm *TestLifecycleManager) CleanupObsoleteTests(gracePeriod time.Duration) ([]string, error) {
	var cleanedTests []string

	// Find tests that have been deprecated for longer than the grace period
	rows, err := tlm.db.Query(`
		SELECT DISTINCT tm.test_id
		FROM test_metadata tm
		JOIN test_lifecycle_events tle ON tm.test_id = tle.test_id
		WHERE tm.status = 'deprecated'
		AND tle.event_type = 'deprecated'
		AND tle.timestamp < $1
	`, time.Now().Add(-gracePeriod))
	if err != nil {
		return nil, fmt.Errorf("failed to query obsolete tests: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var testID string
		err := rows.Scan(&testID)
		if err != nil {
			continue
		}

		// Mark as obsolete
		err = tlm.ObsoleteTest(testID, fmt.Sprintf("Automatic cleanup after %v grace period", gracePeriod))
		if err != nil {
			log.Printf("Failed to mark test %s as obsolete: %v", testID, err)
			continue
		}

		cleanedTests = append(cleanedTests, testID)
	}

	return cleanedTests, rows.Err()
}

// GetTestsByStatus retrieves tests by their lifecycle status
func (tlm *TestLifecycleManager) GetTestsByStatus(status TestStatus) ([]*TestMetadata, error) {
	rows, err := tlm.db.Query(`
		SELECT test_id, file_path, test_name, test_type, 
			   code_coverage, last_modified, last_executed, 
			   execution_count, failure_rate, average_runtime_ms, 
			   complexity, status
		FROM test_metadata
		WHERE status = $1
		ORDER BY last_modified DESC
	`, string(status))
	if err != nil {
		return nil, fmt.Errorf("failed to query tests by status: %w", err)
	}
	defer rows.Close()

	var tests []*TestMetadata
	for rows.Next() {
		var test TestMetadata
		var statusStr string
		var runtimeMs int64

		err := rows.Scan(&test.ID, &test.FilePath, &test.TestName, &test.TestType,
			&test.CodeCoverage, &test.LastModified, &test.LastExecuted,
			&test.ExecutionCount, &test.FailureRate, &runtimeMs,
			&test.Complexity, &statusStr)
		if err != nil {
			log.Printf("Error scanning test: %v", err)
			continue
		}

		test.Status = TestStatus(statusStr)
		test.AverageRuntime = time.Duration(runtimeMs) * time.Millisecond
		tests = append(tests, &test)
	}

	return tests, rows.Err()
}

// GenerateLifecycleReport generates a comprehensive lifecycle report
func (tlm *TestLifecycleManager) GenerateLifecycleReport() (*LifecycleReport, error) {
	report := &LifecycleReport{
		GeneratedAt: time.Now(),
		Summary:     make(map[string]interface{}),
	}

	// Get status distribution
	statusDist, err := tlm.getStatusDistribution()
	if err != nil {
		return nil, fmt.Errorf("failed to get status distribution: %w", err)
	}
	report.StatusDistribution = statusDist

	// Get recent events
	recentEvents, err := tlm.getRecentEvents(7) // Last 7 days
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}
	report.RecentEvents = recentEvents

	// Calculate summary metrics
	report.Summary["total_tests"] = statusDist["active"] + statusDist["deprecated"] + statusDist["obsolete"] + statusDist["quarantined"]
	report.Summary["active_percentage"] = float64(statusDist["active"]) / float64(report.Summary["total_tests"].(int)) * 100
	report.Summary["deprecated_percentage"] = float64(statusDist["deprecated"]) / float64(report.Summary["total_tests"].(int)) * 100

	return report, nil
}

// LifecycleReport represents a comprehensive lifecycle report
type LifecycleReport struct {
	GeneratedAt        time.Time                  `json:"generated_at"`
	StatusDistribution map[string]int             `json:"status_distribution"`
	RecentEvents       []TestLifecycleEvent       `json:"recent_events"`
	Summary            map[string]interface{}     `json:"summary"`
}

// getStatusDistribution gets the distribution of test statuses
func (tlm *TestLifecycleManager) getStatusDistribution() (map[string]int, error) {
	distribution := make(map[string]int)

	rows, err := tlm.db.Query(`
		SELECT status, COUNT(*) as count
		FROM test_metadata
		GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		err := rows.Scan(&status, &count)
		if err != nil {
			continue
		}
		distribution[status] = count
	}

	return distribution, rows.Err()
}

// getRecentEvents gets recent lifecycle events
func (tlm *TestLifecycleManager) getRecentEvents(days int) ([]TestLifecycleEvent, error) {
	rows, err := tlm.db.Query(`
		SELECT event_id, test_id, event_type, timestamp, reason
		FROM test_lifecycle_events
		WHERE timestamp >= $1
		ORDER BY timestamp DESC
		LIMIT 100
	`, time.Now().AddDate(0, 0, -days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TestLifecycleEvent
	for rows.Next() {
		var event TestLifecycleEvent
		var eventType string

		err := rows.Scan(&event.ID, &event.TestID, &eventType, 
			&event.Timestamp, &event.Reason)
		if err != nil {
			continue
		}

		event.EventType = LifecycleEvent(eventType)
		events = append(events, event)
	}

	return events, rows.Err()
}