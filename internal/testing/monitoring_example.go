package testing

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// MonitoringService demonstrates how to integrate the test execution monitoring
type MonitoringService struct {
	monitor   *TestExecutionMonitor
	dashboard *QualityDashboard
	db        *sql.DB
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(db *sql.DB) *MonitoringService {
	return &MonitoringService{
		monitor:   NewTestExecutionMonitor(db),
		dashboard: NewQualityDashboard(db),
		db:        db,
	}
}

// Start starts the monitoring service
func (m *MonitoringService) Start(ctx context.Context) error {
	log.Println("Starting test execution monitoring service...")

	// Start the test execution monitor
	if err := m.monitor.StartMonitoring(ctx); err != nil {
		return err
	}

	// Start periodic tasks
	go m.runPeriodicTasks(ctx)

	log.Println("Test execution monitoring service started successfully")
	return nil
}

// Stop stops the monitoring service
func (m *MonitoringService) Stop() {
	log.Println("Stopping test execution monitoring service...")
	m.monitor.StopMonitoring()
	log.Println("Test execution monitoring service stopped")
}

// SetupDashboardRoutes sets up the dashboard routes
func (m *MonitoringService) SetupDashboardRoutes(router *gin.Engine) {
	m.dashboard.SetupRoutes(router)
}

// RecordTestResult records a test execution result
func (m *MonitoringService) RecordTestResult(testSuite, testName, status string, 
	duration time.Duration, errorMsg string, coverage float64) error {
	
	execution := &TestExecutionRecord{
		TestSuite:    testSuite,
		TestName:     testName,
		Status:       status,
		Duration:     duration,
		StartTime:    time.Now().Add(-duration),
		EndTime:      time.Now(),
		ErrorMessage: errorMsg,
		Coverage:     coverage,
		Environment:  "test",
		Branch:       getCurrentBranch(),
		CommitHash:   getCurrentCommitHash(),
	}

	return m.monitor.RecordTestExecution(execution)
}

// GetHealthStatus returns the current health status
func (m *MonitoringService) GetHealthStatus() (*MonitoringHealthStatus, error) {
	metrics, err := m.monitor.GetTestMetrics("24h")
	if err != nil {
		return nil, err
	}

	flakyTests, err := m.monitor.GetFlakyTests(10)
	if err != nil {
		return nil, err
	}

	patterns, err := m.monitor.GetFailurePatterns(10)
	if err != nil {
		return nil, err
	}

	status := &MonitoringHealthStatus{
		OverallHealth:    calculateOverallHealth(metrics, flakyTests, patterns),
		TestMetrics:      metrics,
		FlakyTestCount:   int64(len(flakyTests)),
		FailurePatterns:  int64(len(patterns)),
		LastUpdated:      time.Now(),
	}

	return status, nil
}

// runPeriodicTasks runs periodic maintenance tasks
func (m *MonitoringService) runPeriodicTasks(ctx context.Context) {
	// Update coverage daily
	coverageTicker := time.NewTicker(24 * time.Hour)
	defer coverageTicker.Stop()

	// Cleanup old data weekly
	cleanupTicker := time.NewTicker(7 * 24 * time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-coverageTicker.C:
			m.updateDailyCoverage()
		case <-cleanupTicker.C:
			m.cleanupOldData()
		}
	}
}

// updateDailyCoverage updates the daily coverage metrics
func (m *MonitoringService) updateDailyCoverage() {
	log.Println("Updating daily coverage metrics...")
	
	tracker := NewCoverageTracker(m.db)
	if err := tracker.UpdateDailyCoverage(); err != nil {
		log.Printf("Failed to update daily coverage: %v", err)
	} else {
		log.Println("Daily coverage metrics updated successfully")
	}
}

// cleanupOldData cleans up old monitoring data
func (m *MonitoringService) cleanupOldData() {
	log.Println("Cleaning up old monitoring data...")

	// Cleanup old alerts
	alertManager := NewTestAlertManager(m.db)
	if err := alertManager.CleanupOldAlerts(); err != nil {
		log.Printf("Failed to cleanup old alerts: %v", err)
	}

	// Cleanup old failure patterns
	patternAnalyzer := NewFailurePatternAnalyzer(m.db)
	if err := patternAnalyzer.CleanupOldPatterns(); err != nil {
		log.Printf("Failed to cleanup old patterns: %v", err)
	}

	log.Println("Old monitoring data cleanup completed")
}

// Helper functions
func getCurrentBranch() string {
	return "main"
}

func getCurrentCommitHash() string {
	return "abc123def456"
}

func calculateOverallHealth(metrics *ExecutionMetrics, flakyTests []FlakyTestInfo, patterns []TestFailurePattern) string {
	score := 100.0

	if metrics.SuccessRate < 95 {
		score -= (95 - metrics.SuccessRate) * 2
	}

	if metrics.CoveragePercent < 95 {
		score -= (95 - metrics.CoveragePercent)
	}

	score -= float64(len(flakyTests)) * 2
	score -= float64(len(patterns)) * 1

	if score >= 90 {
		return "excellent"
	} else if score >= 75 {
		return "good"
	} else if score >= 50 {
		return "warning"
	}
	return "critical"
}

// MonitoringHealthStatus represents the overall health status
type MonitoringHealthStatus struct {
	OverallHealth   string        `json:"overall_health"`
	TestMetrics     *ExecutionMetrics  `json:"test_metrics"`
	FlakyTestCount  int64         `json:"flaky_test_count"`
	FailurePatterns int64         `json:"failure_patterns"`
	LastUpdated     time.Time     `json:"last_updated"`
}