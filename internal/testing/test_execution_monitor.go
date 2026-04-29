package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"high-performance-news-website/pkg/database"
)

// TestExecutionMonitor provides real-time monitoring of test execution
type TestExecutionMonitor struct {
	db                *database.DB
	activeExecutions  map[string]*TestExecution
	executionHistory  []TestExecution
	resourceTracker   *ResourceTracker
	alertManager      *TestAlertManager
	dashboardData     *DashboardData
	mu                sync.RWMutex
	isRunning         bool
	stopChan          chan struct{}
	metricsCollector  *MetricsCollector
}

// TestExecution represents a running test execution
type TestExecution struct {
	ID              string                 `json:"id"`
	TestSuite       string                 `json:"test_suite"`
	TestName        string                 `json:"test_name"`
	Status          ExecutionStatus        `json:"status"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	Duration        time.Duration          `json:"duration"`
	Progress        float64                `json:"progress"`
	ResourceUsage   ResourceUsage          `json:"resource_usage"`
	Metrics         map[string]interface{} `json:"metrics"`
	Environment     string                 `json:"environment"`
	Tags            []string               `json:"tags"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	Logs            []LogEntry             `json:"logs"`
	Dependencies    []string               `json:"dependencies"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// ExecutionStatus represents the status of a test execution
type ExecutionStatus string

const (
	StatusPending    ExecutionStatus = "pending"
	StatusRunning    ExecutionStatus = "running"
	StatusCompleted  ExecutionStatus = "completed"
	StatusFailed     ExecutionStatus = "failed"
	StatusCancelled  ExecutionStatus = "cancelled"
	StatusTimeout    ExecutionStatus = "timeout"
)

// ResourceUsage tracks resource consumption during test execution
type ResourceUsage struct {
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryMB      float64   `json:"memory_mb"`
	DiskIOKB      float64   `json:"disk_io_kb"`
	NetworkIOKB   float64   `json:"network_io_kb"`
	DatabaseConns int       `json:"database_connections"`
	CacheHits     int64     `json:"cache_hits"`
	CacheMisses   int64     `json:"cache_misses"`
	Timestamp     time.Time `json:"timestamp"`
}

// LogEntry represents a log entry during test execution
type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// DashboardData contains aggregated data for monitoring dashboards
type DashboardData struct {
	ActiveTests       int                    `json:"active_tests"`
	CompletedTests    int                    `json:"completed_tests"`
	FailedTests       int                    `json:"failed_tests"`
	TotalExecutions   int                    `json:"total_executions"`
	AverageExecution  time.Duration          `json:"average_execution_time"`
	ResourceUtilization ResourceUsage        `json:"resource_utilization"`
	TestSuiteMetrics  map[string]SuiteMetrics `json:"test_suite_metrics"`
	AlertSummary      AlertSummary           `json:"alert_summary"`
	LastUpdated       time.Time              `json:"last_updated"`
}

// SuiteMetrics contains metrics for a specific test suite
type SuiteMetrics struct {
	Name            string        `json:"name"`
	ActiveTests     int           `json:"active_tests"`
	CompletedTests  int           `json:"completed_tests"`
	FailedTests     int           `json:"failed_tests"`
	AverageRuntime  time.Duration `json:"average_runtime"`
	SuccessRate     float64       `json:"success_rate"`
	LastExecution   *time.Time    `json:"last_execution,omitempty"`
}

// AlertSummary contains alert information
type AlertSummary struct {
	ActiveAlerts    int `json:"active_alerts"`
	CriticalAlerts  int `json:"critical_alerts"`
	WarningAlerts   int `json:"warning_alerts"`
	ResolvedToday   int `json:"resolved_today"`
}

// NewTestExecutionMonitor creates a new test execution monitor
func NewTestExecutionMonitor(db *database.DB) *TestExecutionMonitor {
	return &TestExecutionMonitor{
		db:               db,
		activeExecutions: make(map[string]*TestExecution),
		executionHistory: make([]TestExecution, 0),
		resourceTracker:  NewResourceTracker(),
		alertManager:     NewTestAlertManager(),
		dashboardData:    &DashboardData{},
		stopChan:         make(chan struct{}),
		metricsCollector: NewMetricsCollector(),
	}
}

// Start begins monitoring test executions
func (m *TestExecutionMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("test execution monitor is already running")
	}

	// Initialize database tables
	if err := m.initializeTables(ctx); err != nil {
		return fmt.Errorf("failed to initialize monitoring tables: %w", err)
	}

	// Start resource tracking
	if err := m.resourceTracker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start resource tracker: %w", err)
	}

	// Start alert manager
	if err := m.alertManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start alert manager: %w", err)
	}

	// Start metrics collector
	if err := m.metricsCollector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start metrics collector: %w", err)
	}

	m.isRunning = true

	// Start monitoring goroutines
	go m.monitorExecutions(ctx)
	go m.updateDashboard(ctx)
	go m.cleanupHistory(ctx)

	log.Printf("Test execution monitor started")
	return nil
}

// Stop stops the test execution monitor
func (m *TestExecutionMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return
	}

	close(m.stopChan)
	m.resourceTracker.Stop()
	m.alertManager.Stop()
	m.metricsCollector.Stop()
	m.isRunning = false

	log.Printf("Test execution monitor stopped")
}

// StartTestExecution begins monitoring a new test execution
func (m *TestExecutionMonitor) StartTestExecution(testSuite, testName, environment string, tags []string, dependencies []string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	execution := &TestExecution{
		ID:           generateExecutionID(),
		TestSuite:    testSuite,
		TestName:     testName,
		Status:       StatusPending,
		StartTime:    time.Now(),
		Progress:     0.0,
		Environment:  environment,
		Tags:         tags,
		Dependencies: dependencies,
		Metrics:      make(map[string]interface{}),
		Logs:         make([]LogEntry, 0),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	m.activeExecutions[execution.ID] = execution

	// Store in database
	if err := m.storeExecution(context.Background(), execution); err != nil {
		log.Printf("Failed to store test execution: %v", err)
	}

	// Send start notification
	m.alertManager.NotifyTestStart(execution)

	log.Printf("Started monitoring test execution: %s/%s (ID: %s)", testSuite, testName, execution.ID)
	return execution.ID
}

// UpdateTestExecution updates the status and metrics of a running test
func (m *TestExecutionMonitor) UpdateTestExecution(executionID string, status ExecutionStatus, progress float64, metrics map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	execution, exists := m.activeExecutions[executionID]
	if !exists {
		return fmt.Errorf("test execution %s not found", executionID)
	}

	execution.Status = status
	execution.Progress = progress
	execution.UpdatedAt = time.Now()

	// Update metrics
	for key, value := range metrics {
		execution.Metrics[key] = value
	}

	// Update resource usage
	execution.ResourceUsage = m.resourceTracker.GetCurrentUsage()

	// Handle completion
	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled || status == StatusTimeout {
		now := time.Now()
		execution.EndTime = &now
		execution.Duration = now.Sub(execution.StartTime)

		// Move to history
		m.executionHistory = append(m.executionHistory, *execution)
		delete(m.activeExecutions, executionID)

		// Send completion notification
		m.alertManager.NotifyTestCompletion(execution)
	}

	// Update in database
	if err := m.updateExecution(context.Background(), execution); err != nil {
		log.Printf("Failed to update test execution: %v", err)
	}

	return nil
}

// AddLogEntry adds a log entry to a test execution
func (m *TestExecutionMonitor) AddLogEntry(executionID, level, message, source string, context map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	execution, exists := m.activeExecutions[executionID]
	if !exists {
		return fmt.Errorf("test execution %s not found", executionID)
	}

	logEntry := LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
		Source:    source,
		Context:   context,
	}

	execution.Logs = append(execution.Logs, logEntry)
	execution.UpdatedAt = time.Now()

	// Limit log entries to prevent memory issues
	if len(execution.Logs) > 1000 {
		execution.Logs = execution.Logs[len(execution.Logs)-1000:]
	}

	return nil
}

// GetActiveExecutions returns all currently active test executions
func (m *TestExecutionMonitor) GetActiveExecutions() map[string]*TestExecution {
	m.mu.RLock()
	defer m.mu.RUnlock()

	executions := make(map[string]*TestExecution)
	for id, execution := range m.activeExecutions {
		executions[id] = execution
	}
	return executions
}

// GetExecutionHistory returns recent test execution history
func (m *TestExecutionMonitor) GetExecutionHistory(limit int) []TestExecution {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.executionHistory) {
		limit = len(m.executionHistory)
	}

	// Return most recent executions
	start := len(m.executionHistory) - limit
	if start < 0 {
		start = 0
	}

	history := make([]TestExecution, limit)
	copy(history, m.executionHistory[start:])
	return history
}

// GetDashboardData returns current dashboard data
func (m *TestExecutionMonitor) GetDashboardData() *DashboardData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to avoid race conditions
	data := *m.dashboardData
	return &data
}

// GetExecutionById returns a specific test execution by ID
func (m *TestExecutionMonitor) GetExecutionById(executionID string) (*TestExecution, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check active executions first
	if execution, exists := m.activeExecutions[executionID]; exists {
		return execution, nil
	}

	// Check history
	for _, execution := range m.executionHistory {
		if execution.ID == executionID {
			return &execution, nil
		}
	}

	return nil, fmt.Errorf("test execution %s not found", executionID)
}

// monitorExecutions continuously monitors active test executions
func (m *TestExecutionMonitor) monitorExecutions(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkExecutionHealth()
		}
	}
}

// checkExecutionHealth checks for stuck or problematic test executions
func (m *TestExecutionMonitor) checkExecutionHealth() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, execution := range m.activeExecutions {
		// Check for timeout (tests running longer than 30 minutes)
		if now.Sub(execution.StartTime) > 30*time.Minute {
			execution.Status = StatusTimeout
			execution.ErrorMessage = "Test execution timed out"
			endTime := now
			execution.EndTime = &endTime
			execution.Duration = now.Sub(execution.StartTime)

			// Move to history
			m.executionHistory = append(m.executionHistory, *execution)
			delete(m.activeExecutions, id)

			// Send timeout alert
			m.alertManager.NotifyTestTimeout(execution)

			log.Printf("Test execution timed out: %s/%s (ID: %s)", execution.TestSuite, execution.TestName, execution.ID)
		}

		// Update resource usage
		execution.ResourceUsage = m.resourceTracker.GetCurrentUsage()
		execution.UpdatedAt = now
	}
}

// updateDashboard periodically updates dashboard data
func (m *TestExecutionMonitor) updateDashboard(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.refreshDashboardData()
		}
	}
}

// refreshDashboardData updates the dashboard data
func (m *TestExecutionMonitor) refreshDashboardData() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Count active tests
	m.dashboardData.ActiveTests = len(m.activeExecutions)

	// Count completed and failed tests from history (last 24 hours)
	now := time.Now()
	completed := 0
	failed := 0
	totalDuration := time.Duration(0)
	suiteMetrics := make(map[string]*SuiteMetrics)

	for _, execution := range m.executionHistory {
		if execution.EndTime != nil && now.Sub(*execution.EndTime) <= 24*time.Hour {
			if execution.Status == StatusCompleted {
				completed++
			} else if execution.Status == StatusFailed {
				failed++
			}
			totalDuration += execution.Duration

			// Update suite metrics
			if _, exists := suiteMetrics[execution.TestSuite]; !exists {
				suiteMetrics[execution.TestSuite] = &SuiteMetrics{
					Name: execution.TestSuite,
				}
			}
			suite := suiteMetrics[execution.TestSuite]
			if execution.Status == StatusCompleted {
				suite.CompletedTests++
			} else if execution.Status == StatusFailed {
				suite.FailedTests++
			}
		}
	}

	m.dashboardData.CompletedTests = completed
	m.dashboardData.FailedTests = failed
	m.dashboardData.TotalExecutions = completed + failed

	if m.dashboardData.TotalExecutions > 0 {
		m.dashboardData.AverageExecution = totalDuration / time.Duration(m.dashboardData.TotalExecutions)
	}

	// Calculate success rates for suites
	finalSuiteMetrics := make(map[string]SuiteMetrics)
	for name, metrics := range suiteMetrics {
		total := metrics.CompletedTests + metrics.FailedTests
		if total > 0 {
			metrics.SuccessRate = float64(metrics.CompletedTests) / float64(total)
		}
		finalSuiteMetrics[name] = *metrics
	}
	m.dashboardData.TestSuiteMetrics = finalSuiteMetrics

	// Update resource utilization
	m.dashboardData.ResourceUtilization = m.resourceTracker.GetCurrentUsage()

	// Update alert summary
	m.dashboardData.AlertSummary = m.alertManager.GetAlertSummary()

	m.dashboardData.LastUpdated = now
}

// cleanupHistory periodically cleans up old execution history
func (m *TestExecutionMonitor) cleanupHistory(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.performHistoryCleanup()
		}
	}
}

// performHistoryCleanup removes old execution history
func (m *TestExecutionMonitor) performHistoryCleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Keep only last 7 days of history in memory
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	newHistory := make([]TestExecution, 0)

	for _, execution := range m.executionHistory {
		if execution.EndTime != nil && execution.EndTime.After(cutoff) {
			newHistory = append(newHistory, execution)
		}
	}

	oldCount := len(m.executionHistory)
	m.executionHistory = newHistory
	newCount := len(m.executionHistory)

	if oldCount != newCount {
		log.Printf("Cleaned up execution history: removed %d old entries, kept %d", oldCount-newCount, newCount)
	}
}

// Database operations

// initializeTables creates necessary database tables
func (m *TestExecutionMonitor) initializeTables(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS test_executions (
			id VARCHAR(255) PRIMARY KEY,
			test_suite VARCHAR(255) NOT NULL,
			test_name VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP,
			duration_ms BIGINT,
			progress FLOAT DEFAULT 0.0,
			environment VARCHAR(100),
			tags JSONB,
			dependencies JSONB,
			metrics JSONB,
			resource_usage JSONB,
			error_message TEXT,
			logs JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_test_executions_status ON test_executions(status);
		CREATE INDEX IF NOT EXISTS idx_test_executions_suite ON test_executions(test_suite);
		CREATE INDEX IF NOT EXISTS idx_test_executions_start_time ON test_executions(start_time);
		CREATE INDEX IF NOT EXISTS idx_test_executions_environment ON test_executions(environment);
	`

	_, err := m.db.ExecContext(ctx, query)
	return err
}

// storeExecution stores a test execution in the database
func (m *TestExecutionMonitor) storeExecution(ctx context.Context, execution *TestExecution) error {
	query := `
		INSERT INTO test_executions (
			id, test_suite, test_name, status, start_time, end_time, duration_ms,
			progress, environment, tags, dependencies, metrics, resource_usage,
			error_message, logs, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			end_time = EXCLUDED.end_time,
			duration_ms = EXCLUDED.duration_ms,
			progress = EXCLUDED.progress,
			metrics = EXCLUDED.metrics,
			resource_usage = EXCLUDED.resource_usage,
			error_message = EXCLUDED.error_message,
			logs = EXCLUDED.logs,
			updated_at = EXCLUDED.updated_at
	`

	var durationMs *int64
	if execution.EndTime != nil {
		ms := int64(execution.Duration / time.Millisecond)
		durationMs = &ms
	}

	tagsJSON, _ := json.Marshal(execution.Tags)
	dependenciesJSON, _ := json.Marshal(execution.Dependencies)
	metricsJSON, _ := json.Marshal(execution.Metrics)
	resourceUsageJSON, _ := json.Marshal(execution.ResourceUsage)
	logsJSON, _ := json.Marshal(execution.Logs)

	_, err := m.db.ExecContext(ctx, query,
		execution.ID, execution.TestSuite, execution.TestName, execution.Status,
		execution.StartTime, execution.EndTime, durationMs, execution.Progress,
		execution.Environment, tagsJSON, dependenciesJSON, metricsJSON,
		resourceUsageJSON, execution.ErrorMessage, logsJSON,
		execution.CreatedAt, execution.UpdatedAt,
	)

	return err
}

// updateExecution updates an existing test execution in the database
func (m *TestExecutionMonitor) updateExecution(ctx context.Context, execution *TestExecution) error {
	return m.storeExecution(ctx, execution) // Uses ON CONFLICT DO UPDATE
}

// Utility functions

func generateExecutionID() string {
	return fmt.Sprintf("exec_%d_%d", time.Now().UnixNano(), time.Now().Unix()%1000)
}