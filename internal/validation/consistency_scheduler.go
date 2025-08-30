package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"high-performance-news-website/pkg/database"
)

// CheckScheduler manages scheduling and automation of consistency checks
type CheckScheduler struct {
	db               *database.DB
	checker          *ConsistencyChecker
	reporter         *ConsistencyReporter
	monitoringClient MonitoringClient
	isRunning        bool
	stopChan         chan struct{}
	mu               sync.RWMutex
	schedules        map[string]*ScheduleConfig
}

// ScheduleConfig defines when and how consistency checks should run
type ScheduleConfig struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Cron        string        `json:"cron"`        // Cron expression for scheduling
	Interval    time.Duration `json:"interval"`    // Alternative to cron for simple intervals
	Enabled     bool          `json:"enabled"`
	SampleSize  int           `json:"sample_size"`
	CheckTypes  []string      `json:"check_types"` // Types of checks to run
	LastRun     *time.Time    `json:"last_run,omitempty"`
	NextRun     *time.Time    `json:"next_run,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// CheckResult stores the result of a scheduled consistency check
type CheckResult struct {
	ID           string                 `json:"id"`
	ScheduleID   string                 `json:"schedule_id"`
	CheckID      string                 `json:"check_id"`
	Status       string                 `json:"status"`
	IssuesFound  int                    `json:"issues_found"`
	Duration     time.Duration          `json:"duration"`
	SampleSize   int                    `json:"sample_size"`
	Metadata     map[string]interface{} `json:"metadata"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// MonitoringClient interface for integration with monitoring systems
type MonitoringClient interface {
	SendMetric(name string, value float64, tags map[string]string) error
	SendAlert(level string, message string, details map[string]interface{}) error
}

// NewCheckScheduler creates a new consistency check scheduler
func NewCheckScheduler() *CheckScheduler {
	return &CheckScheduler{
		stopChan:  make(chan struct{}),
		schedules: make(map[string]*ScheduleConfig),
	}
}

// SetDependencies sets the required dependencies
func (s *CheckScheduler) SetDependencies(db *database.DB, checker *ConsistencyChecker, reporter *ConsistencyReporter, monitoring MonitoringClient) {
	s.db = db
	s.checker = checker
	s.reporter = reporter
	s.monitoringClient = monitoring
}

// Start begins the scheduler
func (s *CheckScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	// Load schedules from database
	if err := s.loadSchedules(ctx); err != nil {
		return fmt.Errorf("failed to load schedules: %w", err)
	}

	// Create default schedules if none exist
	if len(s.schedules) == 0 {
		if err := s.createDefaultSchedules(ctx); err != nil {
			return fmt.Errorf("failed to create default schedules: %w", err)
		}
	}

	s.isRunning = true
	go s.run(ctx)

	log.Printf("Consistency check scheduler started with %d schedules", len(s.schedules))
	return nil
}

// Stop stops the scheduler
func (s *CheckScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	close(s.stopChan)
	s.isRunning = false
	log.Printf("Consistency check scheduler stopped")
}

// run is the main scheduler loop
func (s *CheckScheduler) run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkSchedules(ctx)
		}
	}
}

// checkSchedules checks if any scheduled checks need to run
func (s *CheckScheduler) checkSchedules(ctx context.Context) {
	s.mu.RLock()
	schedules := make([]*ScheduleConfig, 0, len(s.schedules))
	for _, schedule := range s.schedules {
		schedules = append(schedules, schedule)
	}
	s.mu.RUnlock()

	now := time.Now()

	for _, schedule := range schedules {
		if !schedule.Enabled {
			continue
		}

		shouldRun := false

		// Check if it's time to run based on interval
		if schedule.Interval > 0 {
			if schedule.LastRun == nil || now.Sub(*schedule.LastRun) >= schedule.Interval {
				shouldRun = true
			}
		}

		// Check if it's time to run based on cron (simplified check)
		if schedule.Cron != "" && schedule.NextRun != nil && now.After(*schedule.NextRun) {
			shouldRun = true
		}

		if shouldRun {
			go s.executeScheduledCheck(ctx, schedule)
		}
	}
}

// executeScheduledCheck executes a scheduled consistency check
func (s *CheckScheduler) executeScheduledCheck(ctx context.Context, schedule *ScheduleConfig) {
	log.Printf("Executing scheduled consistency check: %s", schedule.Name)

	result := &CheckResult{
		ID:         generateResultID(),
		ScheduleID: schedule.ID,
		CreatedAt:  time.Now(),
		SampleSize: schedule.SampleSize,
		Metadata:   make(map[string]interface{}),
	}

	start := time.Now()

	// Execute the consistency check
	check, err := s.checker.ValidateDataConsistency(ctx)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = err.Error()
		log.Printf("Scheduled consistency check failed: %v", err)
	} else {
		result.CheckID = check.ID
		result.Status = string(check.Status)
		result.IssuesFound = len(check.Issues)
		result.Metadata = check.Metadata

		// Process issues through reporter
		if len(check.Issues) > 0 {
			if err := s.reporter.ProcessIssues(ctx, check.Issues); err != nil {
				log.Printf("Failed to process issues from scheduled check: %v", err)
			}
		}
	}

	result.Duration = time.Since(start)

	// Store the result
	if err := s.storeCheckResult(ctx, result); err != nil {
		log.Printf("Failed to store check result: %v", err)
	}

	// Update schedule last run time
	s.updateScheduleLastRun(ctx, schedule.ID, time.Now())

	// Send metrics to monitoring system
	if s.monitoringClient != nil {
		s.sendMetrics(result)
	}

	// Send alerts if necessary
	if result.IssuesFound > 0 && s.monitoringClient != nil {
		s.sendAlertsIfNeeded(result)
	}

	log.Printf("Scheduled consistency check completed: %s (found %d issues in %v)", 
		schedule.Name, result.IssuesFound, result.Duration)
}

// loadSchedules loads schedules from the database
func (s *CheckScheduler) loadSchedules(ctx context.Context) error {
	query := `
		SELECT id, name, cron_expression, interval_seconds, enabled, sample_size, 
			   check_types, last_run, next_run, created_at, updated_at
		FROM consistency_schedules
		WHERE enabled = true
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schedule ScheduleConfig
		var intervalSeconds int
		var checkTypesJSON []byte
		var lastRun, nextRun *time.Time

		err := rows.Scan(
			&schedule.ID, &schedule.Name, &schedule.Cron, &intervalSeconds,
			&schedule.Enabled, &schedule.SampleSize, &checkTypesJSON,
			&lastRun, &nextRun, &schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			continue
		}

		schedule.Interval = time.Duration(intervalSeconds) * time.Second
		schedule.LastRun = lastRun
		schedule.NextRun = nextRun
		json.Unmarshal(checkTypesJSON, &schedule.CheckTypes)

		s.schedules[schedule.ID] = &schedule
	}

	return nil
}

// createDefaultSchedules creates default consistency check schedules
func (s *CheckScheduler) createDefaultSchedules(ctx context.Context) error {
	defaultSchedules := []*ScheduleConfig{
		{
			ID:         "daily_full_check",
			Name:       "Daily Full Consistency Check",
			Interval:   24 * time.Hour,
			Enabled:    true,
			SampleSize: 1000,
			CheckTypes: []string{"referential_integrity", "multilingual", "seo_metadata"},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			ID:         "hourly_quick_check",
			Name:       "Hourly Quick Consistency Check",
			Interval:   1 * time.Hour,
			Enabled:    true,
			SampleSize: 100,
			CheckTypes: []string{"referential_integrity"},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			ID:         "weekly_comprehensive_check",
			Name:       "Weekly Comprehensive Check",
			Interval:   7 * 24 * time.Hour,
			Enabled:    true,
			SampleSize: 5000,
			CheckTypes: []string{"referential_integrity", "multilingual", "seo_metadata"},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	for _, schedule := range defaultSchedules {
		if err := s.createSchedule(ctx, schedule); err != nil {
			return fmt.Errorf("failed to create default schedule %s: %w", schedule.ID, err)
		}
		s.schedules[schedule.ID] = schedule
	}

	return nil
}

// createSchedule creates a new schedule in the database
func (s *CheckScheduler) createSchedule(ctx context.Context, schedule *ScheduleConfig) error {
	query := `
		INSERT INTO consistency_schedules (
			id, name, cron_expression, interval_seconds, enabled, sample_size,
			check_types, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING
	`

	checkTypesJSON, _ := json.Marshal(schedule.CheckTypes)
	intervalSeconds := int(schedule.Interval.Seconds())

	_, err := s.db.ExecContext(ctx, query,
		schedule.ID, schedule.Name, schedule.Cron, intervalSeconds,
		schedule.Enabled, schedule.SampleSize, checkTypesJSON,
		schedule.CreatedAt, schedule.UpdatedAt,
	)

	return err
}

// storeCheckResult stores a check result in the database
func (s *CheckScheduler) storeCheckResult(ctx context.Context, result *CheckResult) error {
	query := `
		INSERT INTO consistency_check_results (
			id, schedule_id, check_id, status, issues_found, duration_ms,
			sample_size, metadata, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	metadataJSON, _ := json.Marshal(result.Metadata)
	durationMs := int64(result.Duration / time.Millisecond)

	_, err := s.db.ExecContext(ctx, query,
		result.ID, result.ScheduleID, result.CheckID, result.Status,
		result.IssuesFound, durationMs, result.SampleSize,
		metadataJSON, result.ErrorMessage, result.CreatedAt,
	)

	return err
}

// updateScheduleLastRun updates the last run time for a schedule
func (s *CheckScheduler) updateScheduleLastRun(ctx context.Context, scheduleID string, lastRun time.Time) {
	query := `
		UPDATE consistency_schedules 
		SET last_run = $1, updated_at = $1
		WHERE id = $2
	`

	_, err := s.db.ExecContext(ctx, query, lastRun, scheduleID)
	if err != nil {
		log.Printf("Failed to update schedule last run time: %v", err)
	}

	// Update in-memory schedule
	s.mu.Lock()
	if schedule, exists := s.schedules[scheduleID]; exists {
		schedule.LastRun = &lastRun
		schedule.UpdatedAt = lastRun
	}
	s.mu.Unlock()
}

// sendMetrics sends performance metrics to monitoring system
func (s *CheckScheduler) sendMetrics(result *CheckResult) {
	tags := map[string]string{
		"schedule_id": result.ScheduleID,
		"status":      result.Status,
	}

	// Send duration metric
	s.monitoringClient.SendMetric("consistency_check.duration", 
		float64(result.Duration/time.Millisecond), tags)

	// Send issues found metric
	s.monitoringClient.SendMetric("consistency_check.issues_found", 
		float64(result.IssuesFound), tags)

	// Send sample size metric
	s.monitoringClient.SendMetric("consistency_check.sample_size", 
		float64(result.SampleSize), tags)
}

// sendAlertsIfNeeded sends alerts based on check results
func (s *CheckScheduler) sendAlertsIfNeeded(result *CheckResult) {
	// Alert on high number of issues
	if result.IssuesFound > 50 {
		s.monitoringClient.SendAlert("warning", 
			fmt.Sprintf("Consistency check found %d issues", result.IssuesFound),
			map[string]interface{}{
				"schedule_id":   result.ScheduleID,
				"check_id":      result.CheckID,
				"issues_found":  result.IssuesFound,
				"duration":      result.Duration.String(),
			})
	}

	// Alert on check failures
	if result.Status == "failed" {
		s.monitoringClient.SendAlert("error",
			fmt.Sprintf("Consistency check failed: %s", result.ErrorMessage),
			map[string]interface{}{
				"schedule_id":    result.ScheduleID,
				"error_message":  result.ErrorMessage,
			})
	}

	// Alert on performance issues
	if result.Duration > 10*time.Minute {
		s.monitoringClient.SendAlert("warning",
			fmt.Sprintf("Consistency check took %v to complete", result.Duration),
			map[string]interface{}{
				"schedule_id": result.ScheduleID,
				"duration":    result.Duration.String(),
				"sample_size": result.SampleSize,
			})
	}
}

// GetSchedules returns all schedules
func (s *CheckScheduler) GetSchedules() map[string]*ScheduleConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedules := make(map[string]*ScheduleConfig)
	for id, schedule := range s.schedules {
		schedules[id] = schedule
	}
	return schedules
}

// GetCheckResults returns recent check results
func (s *CheckScheduler) GetCheckResults(ctx context.Context, limit int) ([]CheckResult, error) {
	query := `
		SELECT id, schedule_id, check_id, status, issues_found, duration_ms,
			   sample_size, metadata, error_message, created_at
		FROM consistency_check_results
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CheckResult
	for rows.Next() {
		var result CheckResult
		var durationMs int64
		var metadataJSON []byte
		var checkID *string

		err := rows.Scan(
			&result.ID, &result.ScheduleID, &checkID, &result.Status,
			&result.IssuesFound, &durationMs, &result.SampleSize,
			&metadataJSON, &result.ErrorMessage, &result.CreatedAt,
		)
		if err != nil {
			continue
		}

		if checkID != nil {
			result.CheckID = *checkID
		}
		result.Duration = time.Duration(durationMs) * time.Millisecond
		json.Unmarshal(metadataJSON, &result.Metadata)

		results = append(results, result)
	}

	return results, nil
}

// EnableSchedule enables a schedule
func (s *CheckScheduler) EnableSchedule(ctx context.Context, scheduleID string) error {
	query := `UPDATE consistency_schedules SET enabled = true, updated_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, scheduleID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if schedule, exists := s.schedules[scheduleID]; exists {
		schedule.Enabled = true
		schedule.UpdatedAt = time.Now()
	}
	s.mu.Unlock()

	return nil
}

// DisableSchedule disables a schedule
func (s *CheckScheduler) DisableSchedule(ctx context.Context, scheduleID string) error {
	query := `UPDATE consistency_schedules SET enabled = false, updated_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, scheduleID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if schedule, exists := s.schedules[scheduleID]; exists {
		schedule.Enabled = false
		schedule.UpdatedAt = time.Now()
	}
	s.mu.Unlock()

	return nil
}

// Utility functions

func generateResultID() string {
	return fmt.Sprintf("result_%d", time.Now().UnixNano())
}