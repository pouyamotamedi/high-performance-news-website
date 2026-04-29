package maintenance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// MaintenanceScheduler handles scheduled maintenance tasks
type MaintenanceScheduler struct {
	db       *sql.DB
	cron     *cron.Cron
	tasks    map[string]MaintenanceTask
	running  bool
}

// MaintenanceTask represents a scheduled maintenance task
type MaintenanceTask interface {
	Execute() error
	GetName() string
	GetDescription() string
}

// NewMaintenanceScheduler creates a new maintenance scheduler
func NewMaintenanceScheduler(db *sql.DB) *MaintenanceScheduler {
	return &MaintenanceScheduler{
		db:    db,
		cron:  cron.New(),
		tasks: make(map[string]MaintenanceTask),
	}
}

// Start starts the maintenance scheduler
func (ms *MaintenanceScheduler) Start() error {
	if ms.running {
		return fmt.Errorf("scheduler is already running")
	}

	// Load scheduled tasks from database
	err := ms.loadScheduledTasks()
	if err != nil {
		return fmt.Errorf("failed to load scheduled tasks: %w", err)
	}

	ms.cron.Start()
	ms.running = true

	log.Println("Maintenance scheduler started")
	return nil
}

// Stop stops the maintenance scheduler
func (ms *MaintenanceScheduler) Stop() {
	if ms.running {
		ms.cron.Stop()
		ms.running = false
		log.Println("Maintenance scheduler stopped")
	}
}

// ScheduleTask schedules a maintenance task
func (ms *MaintenanceScheduler) ScheduleTask(schedule MaintenanceSchedule, task MaintenanceTask) error {
	// Store schedule in database
	configJSON, _ := json.Marshal(schedule.Config)
	
	_, err := ms.db.Exec(`
		INSERT INTO maintenance_schedules (
			schedule_id, name, type, schedule_cron, enabled, config, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (schedule_id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			schedule_cron = EXCLUDED.schedule_cron,
			enabled = EXCLUDED.enabled,
			config = EXCLUDED.config,
			updated_at = EXCLUDED.updated_at
	`, schedule.ID, schedule.Name, string(schedule.Type), schedule.Schedule,
		schedule.Enabled, configJSON, time.Now(), time.Now())

	if err != nil {
		return fmt.Errorf("failed to store schedule: %w", err)
	}

	// Add to cron if enabled
	if schedule.Enabled {
		entryID, err := ms.cron.AddFunc(schedule.Schedule, func() {
			ms.executeTask(schedule.ID, task)
		})
		if err != nil {
			return fmt.Errorf("failed to add cron job: %w", err)
		}

		log.Printf("Scheduled task %s with cron entry %d", schedule.Name, entryID)
	}

	ms.tasks[schedule.ID] = task
	return nil
}

// executeTask executes a maintenance task
func (ms *MaintenanceScheduler) executeTask(scheduleID string, task MaintenanceTask) {
	log.Printf("Executing maintenance task: %s", task.GetName())

	start := time.Now()
	err := task.Execute()
	duration := time.Since(start)

	// Update last run time
	_, updateErr := ms.db.Exec(`
		UPDATE maintenance_schedules 
		SET last_run = $1 
		WHERE schedule_id = $2
	`, start, scheduleID)

	if updateErr != nil {
		log.Printf("Failed to update last run time: %v", updateErr)
	}

	if err != nil {
		log.Printf("Maintenance task %s failed: %v (duration: %v)", task.GetName(), err, duration)
	} else {
		log.Printf("Maintenance task %s completed successfully (duration: %v)", task.GetName(), duration)
	}
}

// loadScheduledTasks loads scheduled tasks from the database
func (ms *MaintenanceScheduler) loadScheduledTasks() error {
	rows, err := ms.db.Query(`
		SELECT schedule_id, name, type, schedule_cron, enabled, config
		FROM maintenance_schedules
		WHERE enabled = true
	`)
	if err != nil {
		return fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schedule MaintenanceSchedule
		var scheduleType string
		var configJSON []byte

		err := rows.Scan(&schedule.ID, &schedule.Name, &scheduleType, 
			&schedule.Schedule, &schedule.Enabled, &configJSON)
		if err != nil {
			log.Printf("Error scanning schedule: %v", err)
			continue
		}

		schedule.Type = MaintenanceType(scheduleType)
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &schedule.Config)
		}

		// Create appropriate task based on type
		task := ms.createTaskFromType(schedule.Type, schedule.Config)
		if task != nil {
			_, err = ms.cron.AddFunc(schedule.Schedule, func() {
				ms.executeTask(schedule.ID, task)
			})
			if err != nil {
				log.Printf("Failed to add cron job for %s: %v", schedule.Name, err)
				continue
			}

			ms.tasks[schedule.ID] = task
			log.Printf("Loaded scheduled task: %s", schedule.Name)
		}
	}

	return rows.Err()
}

// createTaskFromType creates a task instance based on the maintenance type
func (ms *MaintenanceScheduler) createTaskFromType(taskType MaintenanceType, config map[string]interface{}) MaintenanceTask {
	switch taskType {
	case MaintenanceAnalysis:
		return &AnalysisTask{db: ms.db, config: config}
	case MaintenanceCleanup:
		return &CleanupTask{db: ms.db, config: config}
	case MaintenanceOptimization:
		return &OptimizationTask{db: ms.db, config: config}
	case MaintenanceDeprecation:
		return &DeprecationTask{db: ms.db, config: config}
	case MaintenanceValidation:
		return &ValidationTask{db: ms.db, config: config}
	default:
		log.Printf("Unknown maintenance task type: %s", taskType)
		return nil
	}
}

// GetScheduledTasks returns all scheduled tasks
func (ms *MaintenanceScheduler) GetScheduledTasks() ([]MaintenanceSchedule, error) {
	rows, err := ms.db.Query(`
		SELECT schedule_id, name, type, schedule_cron, enabled, 
			   last_run, next_run, config, created_at, updated_at
		FROM maintenance_schedules
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	var schedules []MaintenanceSchedule
	for rows.Next() {
		var schedule MaintenanceSchedule
		var scheduleType string
		var configJSON []byte
		var lastRun, nextRun sql.NullTime

		err := rows.Scan(&schedule.ID, &schedule.Name, &scheduleType, 
			&schedule.Schedule, &schedule.Enabled, &lastRun, &nextRun,
			&configJSON, &schedule.CreatedAt, &schedule.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning schedule: %v", err)
			continue
		}

		schedule.Type = MaintenanceType(scheduleType)
		if lastRun.Valid {
			schedule.LastRun = &lastRun.Time
		}
		if nextRun.Valid {
			schedule.NextRun = &nextRun.Time
		}
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &schedule.Config)
		}

		schedules = append(schedules, schedule)
	}

	return schedules, rows.Err()
}

// UpdateSchedule updates a maintenance schedule
func (ms *MaintenanceScheduler) UpdateSchedule(scheduleID string, updates map[string]interface{}) error {
	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		switch field {
		case "name", "schedule_cron", "enabled":
			setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
			args = append(args, value)
			argIndex++
		case "config":
			configJSON, _ := json.Marshal(value)
			setParts = append(setParts, fmt.Sprintf("config = $%d", argIndex))
			args = append(args, configJSON)
			argIndex++
		}
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, scheduleID)

	query := fmt.Sprintf(`
		UPDATE maintenance_schedules 
		SET %s
		WHERE schedule_id = $%d
	`, strings.Join(setParts, ", "), argIndex)

	_, err := ms.db.Exec(query, args...)
	return err
}

// DeleteSchedule deletes a maintenance schedule
func (ms *MaintenanceScheduler) DeleteSchedule(scheduleID string) error {
	_, err := ms.db.Exec(`
		DELETE FROM maintenance_schedules 
		WHERE schedule_id = $1
	`, scheduleID)

	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	// Remove from tasks map
	delete(ms.tasks, scheduleID)

	return nil
}

// Maintenance Task Implementations

// AnalysisTask performs test suite analysis
type AnalysisTask struct {
	db     *sql.DB
	config map[string]interface{}
}

func (at *AnalysisTask) Execute() error {
	log.Println("Executing test suite analysis...")
	
	// Create test maintenance manager
	tmm := NewTestMaintenanceManager(at.db)
	
	// Get test path from config
	testPath := "."
	if path, ok := at.config["test_path"].(string); ok {
		testPath = path
	}

	// Analyze test suite
	analysis, err := tmm.AnalyzeTestSuite(testPath)
	if err != nil {
		return fmt.Errorf("failed to analyze test suite: %w", err)
	}

	// Update relationships
	err = tmm.UpdateTestRelationships(analysis)
	if err != nil {
		log.Printf("Warning: Failed to update relationships: %v", err)
	}

	log.Printf("Analysis completed: %d tests analyzed, %d issues found, %d suggestions generated",
		len(analysis.Tests), len(analysis.Issues), len(analysis.Suggestions))

	return nil
}

func (at *AnalysisTask) GetName() string {
	return "Test Suite Analysis"
}

func (at *AnalysisTask) GetDescription() string {
	return "Analyzes the test suite for issues and maintenance opportunities"
}

// CleanupTask performs test cleanup operations
type CleanupTask struct {
	db     *sql.DB
	config map[string]interface{}
}

func (ct *CleanupTask) Execute() error {
	log.Println("Executing test cleanup...")

	tmm := NewTestMaintenanceManager(ct.db)

	// Get grace period from config
	gracePeriod := 30 * 24 * time.Hour // Default 30 days
	if period, ok := ct.config["grace_period_days"].(float64); ok {
		gracePeriod = time.Duration(period) * 24 * time.Hour
	}

	// Cleanup obsolete tests
	cleanedTests, err := tmm.LifecycleManager().CleanupObsoleteTests(gracePeriod)
	if err != nil {
		return fmt.Errorf("failed to cleanup obsolete tests: %w", err)
	}

	log.Printf("Cleanup completed: %d obsolete tests cleaned up", len(cleanedTests))
	return nil
}

func (ct *CleanupTask) GetName() string {
	return "Test Cleanup"
}

func (ct *CleanupTask) GetDescription() string {
	return "Cleans up obsolete and deprecated tests"
}

// OptimizationTask performs test optimization
type OptimizationTask struct {
	db     *sql.DB
	config map[string]interface{}
}

func (ot *OptimizationTask) Execute() error {
	log.Println("Executing test optimization...")

	tqo := NewTestQualityOptimizer(ot.db)

	// Get test path from config
	testPath := "."
	if path, ok := ot.config["test_path"].(string); ok {
		testPath = path
	}

	// Analyze test quality
	report, err := tqo.AnalyzeTestQuality(testPath)
	if err != nil {
		return fmt.Errorf("failed to analyze test quality: %w", err)
	}

	log.Printf("Optimization analysis completed: %d quality issues found, %d refactoring opportunities identified",
		len(report.Issues), len(report.Opportunities))

	return nil
}

func (ot *OptimizationTask) GetName() string {
	return "Test Optimization"
}

func (ot *OptimizationTask) GetDescription() string {
	return "Analyzes and optimizes test quality and performance"
}

// DeprecationTask handles test deprecation
type DeprecationTask struct {
	db     *sql.DB
	config map[string]interface{}
}

func (dt *DeprecationTask) Execute() error {
	log.Println("Executing test deprecation...")

	tmm := NewTestMaintenanceManager(dt.db)

	// Build deprecation criteria from config
	criteria := DeprecationCriteria{
		MaxAge:            90 * 24 * time.Hour, // Default 90 days
		MinFailureRate:    0.2,                 // Default 20%
		MaxExecutionCount: 5,                   // Default 5 executions
	}

	if maxAgeDays, ok := dt.config["max_age_days"].(float64); ok {
		criteria.MaxAge = time.Duration(maxAgeDays) * 24 * time.Hour
	}
	if minFailureRate, ok := dt.config["min_failure_rate"].(float64); ok {
		criteria.MinFailureRate = minFailureRate
	}
	if maxExecCount, ok := dt.config["max_execution_count"].(float64); ok {
		criteria.MaxExecutionCount = int(maxExecCount)
	}

	// Schedule deprecation
	scheduledTests, err := tmm.LifecycleManager().ScheduleDeprecation(criteria)
	if err != nil {
		return fmt.Errorf("failed to schedule deprecation: %w", err)
	}

	log.Printf("Deprecation completed: %d tests scheduled for deprecation", len(scheduledTests))
	return nil
}

func (dt *DeprecationTask) GetName() string {
	return "Test Deprecation"
}

func (dt *DeprecationTask) GetDescription() string {
	return "Identifies and schedules tests for deprecation based on criteria"
}

// ValidationTask performs test validation
type ValidationTask struct {
	db     *sql.DB
	config map[string]interface{}
}

func (vt *ValidationTask) Execute() error {
	log.Println("Executing test validation...")

	// Perform various validation checks
	err := vt.validateTestIntegrity()
	if err != nil {
		return fmt.Errorf("test integrity validation failed: %w", err)
	}

	err = vt.validateTestRelationships()
	if err != nil {
		return fmt.Errorf("test relationships validation failed: %w", err)
	}

	log.Println("Test validation completed successfully")
	return nil
}

func (vt *ValidationTask) validateTestIntegrity() error {
	// Check for orphaned test metadata
	var orphanedCount int
	err := vt.db.QueryRow(`
		SELECT COUNT(*) FROM test_metadata tm
		LEFT JOIN test_lifecycle_events tle ON tm.test_id = tle.test_id
		WHERE tle.test_id IS NULL
	`).Scan(&orphanedCount)

	if err != nil {
		return fmt.Errorf("failed to check orphaned metadata: %w", err)
	}

	if orphanedCount > 0 {
		log.Printf("Warning: Found %d orphaned test metadata records", orphanedCount)
	}

	return nil
}

func (vt *ValidationTask) validateTestRelationships() error {
	// Check for broken relationships
	var brokenCount int
	err := vt.db.QueryRow(`
		SELECT COUNT(*) FROM test_relationships tr
		LEFT JOIN test_metadata tm1 ON tr.source_test_id = tm1.test_id
		LEFT JOIN test_metadata tm2 ON tr.target_test_id = tm2.test_id
		WHERE tm1.test_id IS NULL OR tm2.test_id IS NULL
	`).Scan(&brokenCount)

	if err != nil {
		return fmt.Errorf("failed to check broken relationships: %w", err)
	}

	if brokenCount > 0 {
		log.Printf("Warning: Found %d broken test relationships", brokenCount)
	}

	return nil
}

func (vt *ValidationTask) GetName() string {
	return "Test Validation"
}

func (vt *ValidationTask) GetDescription() string {
	return "Validates test data integrity and relationships"
}