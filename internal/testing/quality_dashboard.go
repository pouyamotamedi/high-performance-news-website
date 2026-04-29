package testing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// QualityDashboard provides a web interface for test quality metrics
type QualityDashboard struct {
	monitor         *TestExecutionMonitor
	securityTracker *SecurityVulnerabilityTracker
	perfTracker     *PerformanceRegressionTracker
	qualityGates    *QualityGateManager
	db              *sql.DB
}

// DashboardData represents the complete dashboard data
type DashboardData struct {
	Overview            *QualityOverview            `json:"overview"`
	TestMetrics         *ExecutionMetrics           `json:"test_metrics"`
	FlakyTests          []FlakyTestInfo             `json:"flaky_tests"`
	FailurePatterns     []TestFailurePattern        `json:"failure_patterns"`
	CoverageTrends      []CoverageTrend             `json:"coverage_trends"`
	SecurityVulns       []SecurityVulnerability     `json:"security_vulnerabilities"`
	PerformanceRegress  []PerformanceRegression     `json:"performance_regressions"`
	QualityGateStatus   *QualityGateStatus          `json:"quality_gate_status"`
	RecentAlerts        []TestAlert                 `json:"recent_alerts"`
	LastUpdated         time.Time                   `json:"last_updated"`
}

// QualityOverview provides high-level quality metrics
type QualityOverview struct {
	OverallHealth       string  `json:"overall_health"`       // "excellent", "good", "warning", "critical"
	HealthScore         float64 `json:"health_score"`         // 0-100
	TestSuccessRate     float64 `json:"test_success_rate"`
	CoveragePercent     float64 `json:"coverage_percent"`
	FlakyTestCount      int64   `json:"flaky_test_count"`
	SecurityIssues      int64   `json:"security_issues"`
	PerformanceIssues   int64   `json:"performance_issues"`
	QualityGatesPassed  int     `json:"quality_gates_passed"`
	QualityGatesTotal   int     `json:"quality_gates_total"`
}

// SecurityVulnerability represents a security vulnerability
type SecurityVulnerability struct {
	ID          int64     `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Component   string    `json:"component"`
	CVSS        float64   `json:"cvss_score"`
	FirstSeen   time.Time `json:"first_seen"`
	Status      string    `json:"status"`
	Remediation string    `json:"remediation"`
}

// PerformanceRegression represents a performance regression
type PerformanceRegression struct {
	ID              int64     `json:"id"`
	TestName        string    `json:"test_name"`
	Metric          string    `json:"metric"`
	BaselineValue   float64   `json:"baseline_value"`
	CurrentValue    float64   `json:"current_value"`
	RegressionPct   float64   `json:"regression_percent"`
	DetectedAt      time.Time `json:"detected_at"`
	Severity        string    `json:"severity"`
	Status          string    `json:"status"`
}

// QualityGateStatus represents the status of quality gates
type QualityGateStatus struct {
	OverallStatus   string              `json:"overall_status"`
	Gates          []QualityGate       `json:"gates"`
	LastEvaluation time.Time           `json:"last_evaluation"`
}

// QualityGate represents a single quality gate
type QualityGate struct {
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	Threshold   float64 `json:"threshold"`
	CurrentValue float64 `json:"current_value"`
	Description string  `json:"description"`
}

// NewQualityDashboard creates a new quality dashboard
func NewQualityDashboard(db *sql.DB) *QualityDashboard {
	return &QualityDashboard{
		monitor:         NewTestExecutionMonitor(db),
		securityTracker: NewSecurityVulnerabilityTracker(db),
		perfTracker:     NewPerformanceRegressionTracker(db),
		qualityGates:    NewQualityGateManager(db),
		db:              db,
	}
}

// SetupRoutes sets up the dashboard HTTP routes
func (d *QualityDashboard) SetupRoutes(router *gin.Engine) {
	dashboard := router.Group("/dashboard")
	{
		dashboard.GET("/", d.handleDashboardHome)
		dashboard.GET("/api/data", d.handleDashboardData)
		dashboard.GET("/api/metrics/:timeRange", d.handleMetrics)
		dashboard.GET("/api/flaky-tests", d.handleFlakyTests)
		dashboard.GET("/api/failure-patterns", d.handleFailurePatterns)
		dashboard.GET("/api/coverage-trends/:days", d.handleCoverageTrends)
		dashboard.GET("/api/security-vulnerabilities", d.handleSecurityVulnerabilities)
		dashboard.GET("/api/performance-regressions", d.handlePerformanceRegressions)
		dashboard.GET("/api/quality-gates", d.handleQualityGates)
		dashboard.POST("/api/quality-gates/evaluate", d.handleEvaluateQualityGates)
		dashboard.GET("/health", d.handleHealthCheck)
	}

	// Serve static files for dashboard UI
	router.Static("/dashboard/static", "./web/dashboard")
}

// handleDashboardHome serves the main dashboard page
func (d *QualityDashboard) handleDashboardHome(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Test Quality Dashboard",
	})
}

// handleDashboardData returns complete dashboard data
func (d *QualityDashboard) handleDashboardData(c *gin.Context) {
	ctx := context.Background()
	
	data, err := d.GetDashboardData(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// handleMetrics returns test metrics for a time range
func (d *QualityDashboard) handleMetrics(c *gin.Context) {
	timeRange := c.Param("timeRange")
	
	metrics, err := d.monitor.GetTestMetrics(timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// handleFlakyTests returns flaky test information
func (d *QualityDashboard) handleFlakyTests(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	flakyTests, err := d.monitor.GetFlakyTests(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flakyTests)
}

// handleFailurePatterns returns failure pattern analysis
func (d *QualityDashboard) handleFailurePatterns(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	patterns, err := d.monitor.GetFailurePatterns(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, patterns)
}

// handleCoverageTrends returns coverage trend data
func (d *QualityDashboard) handleCoverageTrends(c *gin.Context) {
	days := 30
	if d := c.Param("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	trends, err := d.monitor.GetCoverageTrends(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trends)
}

// handleSecurityVulnerabilities returns security vulnerability data
func (d *QualityDashboard) handleSecurityVulnerabilities(c *gin.Context) {
	vulns, err := d.securityTracker.GetVulnerabilities()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, vulns)
}

// handlePerformanceRegressions returns performance regression data
func (d *QualityDashboard) handlePerformanceRegressions(c *gin.Context) {
	regressions, err := d.perfTracker.GetRegressions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, regressions)
}

// handleQualityGates returns quality gate status
func (d *QualityDashboard) handleQualityGates(c *gin.Context) {
	status, err := d.qualityGates.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// handleEvaluateQualityGates triggers quality gate evaluation
func (d *QualityDashboard) handleEvaluateQualityGates(c *gin.Context) {
	ctx := context.Background()
	
	status, err := d.qualityGates.EvaluateGates(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// handleHealthCheck returns dashboard health status
func (d *QualityDashboard) handleHealthCheck(c *gin.Context) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"services": map[string]string{
			"database":         "healthy",
			"test_monitor":     "healthy",
			"security_tracker": "healthy",
			"perf_tracker":     "healthy",
		},
	}

	// Check database connectivity
	if err := d.db.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["services"].(map[string]string)["database"] = "unhealthy"
	}

	c.JSON(http.StatusOK, health)
}

// GetDashboardData retrieves complete dashboard data
func (d *QualityDashboard) GetDashboardData(ctx context.Context) (*DashboardData, error) {
	data := &DashboardData{
		LastUpdated: time.Now(),
	}

	// Get test metrics
	metrics, err := d.monitor.GetTestMetrics("24h")
	if err != nil {
		return nil, fmt.Errorf("failed to get test metrics: %w", err)
	}
	data.TestMetrics = metrics

	// Get flaky tests
	flakyTests, err := d.monitor.GetFlakyTests(10)
	if err != nil {
		return nil, fmt.Errorf("failed to get flaky tests: %w", err)
	}
	data.FlakyTests = flakyTests

	// Get failure patterns
	patterns, err := d.monitor.GetFailurePatterns(10)
	if err != nil {
		return nil, fmt.Errorf("failed to get failure patterns: %w", err)
	}
	data.FailurePatterns = patterns

	// Get coverage trends
	trends, err := d.monitor.GetCoverageTrends(7)
	if err != nil {
		return nil, fmt.Errorf("failed to get coverage trends: %w", err)
	}
	data.CoverageTrends = trends

	// Get security vulnerabilities
	vulns, err := d.securityTracker.GetVulnerabilities()
	if err != nil {
		return nil, fmt.Errorf("failed to get security vulnerabilities: %w", err)
	}
	data.SecurityVulns = vulns

	// Get performance regressions
	regressions, err := d.perfTracker.GetRegressions()
	if err != nil {
		return nil, fmt.Errorf("failed to get performance regressions: %w", err)
	}
	data.PerformanceRegress = regressions

	// Get quality gate status
	gateStatus, err := d.qualityGates.GetStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get quality gate status: %w", err)
	}
	data.QualityGateStatus = gateStatus

	// Calculate overview
	data.Overview = d.calculateOverview(data)

	return data, nil
}

// calculateOverview calculates the quality overview metrics
func (d *QualityDashboard) calculateOverview(data *DashboardData) *QualityOverview {
	overview := &QualityOverview{}

	if data.TestMetrics != nil {
		overview.TestSuccessRate = data.TestMetrics.SuccessRate
		overview.CoveragePercent = data.TestMetrics.CoveragePercent
		overview.FlakyTestCount = data.TestMetrics.FlakyTests
	}

	overview.SecurityIssues = int64(len(data.SecurityVulns))
	overview.PerformanceIssues = int64(len(data.PerformanceRegress))

	if data.QualityGateStatus != nil {
		overview.QualityGatesTotal = len(data.QualityGateStatus.Gates)
		for _, gate := range data.QualityGateStatus.Gates {
			if gate.Status == "passed" {
				overview.QualityGatesPassed++
			}
		}
	}

	// Calculate health score (0-100)
	healthScore := 100.0

	// Deduct points for issues
	healthScore -= float64(overview.SecurityIssues) * 5    // -5 per security issue
	healthScore -= float64(overview.PerformanceIssues) * 3 // -3 per performance issue
	healthScore -= float64(overview.FlakyTestCount) * 2    // -2 per flaky test

	// Deduct for low success rate
	if overview.TestSuccessRate < 95 {
		healthScore -= (95 - overview.TestSuccessRate) * 2
	}

	// Deduct for low coverage
	if overview.CoveragePercent < 95 {
		healthScore -= (95 - overview.CoveragePercent) * 1
	}

	// Ensure score is between 0 and 100
	if healthScore < 0 {
		healthScore = 0
	}
	overview.HealthScore = healthScore

	// Determine overall health
	if healthScore >= 90 {
		overview.OverallHealth = "excellent"
	} else if healthScore >= 75 {
		overview.OverallHealth = "good"
	} else if healthScore >= 50 {
		overview.OverallHealth = "warning"
	} else {
		overview.OverallHealth = "critical"
	}

	return overview
}

// SecurityVulnerabilityTracker tracks security vulnerabilities
type SecurityVulnerabilityTracker struct {
	db *sql.DB
}

// NewSecurityVulnerabilityTracker creates a new security vulnerability tracker
func NewSecurityVulnerabilityTracker(db *sql.DB) *SecurityVulnerabilityTracker {
	tracker := &SecurityVulnerabilityTracker{db: db}
	tracker.initializeTables()
	return tracker
}

// initializeTables creates the security vulnerability tables
func (s *SecurityVulnerabilityTracker) initializeTables() error {
	query := `
		CREATE TABLE IF NOT EXISTS security_vulnerabilities (
			id BIGSERIAL PRIMARY KEY,
			type VARCHAR(100) NOT NULL,
			severity VARCHAR(50) NOT NULL,
			description TEXT NOT NULL,
			component VARCHAR(255),
			cvss_score DECIMAL(3,1),
			first_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			status VARCHAR(50) DEFAULT 'open',
			remediation TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`

	_, err := s.db.Exec(query)
	return err
}

// RecordVulnerability records a new security vulnerability
func (s *SecurityVulnerabilityTracker) RecordVulnerability(vuln *SecurityVulnerability) error {
	query := `
		INSERT INTO security_vulnerabilities (type, severity, description, component, cvss_score, remediation)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, first_seen
	`

	err := s.db.QueryRow(query,
		vuln.Type, vuln.Severity, vuln.Description,
		vuln.Component, vuln.CVSS, vuln.Remediation,
	).Scan(&vuln.ID, &vuln.FirstSeen)

	return err
}

// GetVulnerabilities returns current security vulnerabilities
func (s *SecurityVulnerabilityTracker) GetVulnerabilities() ([]SecurityVulnerability, error) {
	query := `
		SELECT id, type, severity, description, component, cvss_score, first_seen, status, remediation
		FROM security_vulnerabilities
		WHERE status != 'resolved'
		ORDER BY cvss_score DESC, first_seen DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get vulnerabilities: %w", err)
	}
	defer rows.Close()

	var vulns []SecurityVulnerability
	for rows.Next() {
		var vuln SecurityVulnerability
		var component sql.NullString
		var cvss sql.NullFloat64
		var remediation sql.NullString

		err := rows.Scan(
			&vuln.ID, &vuln.Type, &vuln.Severity, &vuln.Description,
			&component, &cvss, &vuln.FirstSeen, &vuln.Status, &remediation,
		)
		if err != nil {
			continue
		}

		if component.Valid {
			vuln.Component = component.String
		}
		if cvss.Valid {
			vuln.CVSS = cvss.Float64
		}
		if remediation.Valid {
			vuln.Remediation = remediation.String
		}

		vulns = append(vulns, vuln)
	}

	return vulns, nil
}

// PerformanceRegressionTracker tracks performance regressions
type PerformanceRegressionTracker struct {
	db *sql.DB
}

// NewPerformanceRegressionTracker creates a new performance regression tracker
func NewPerformanceRegressionTracker(db *sql.DB) *PerformanceRegressionTracker {
	tracker := &PerformanceRegressionTracker{db: db}
	tracker.initializeTables()
	return tracker
}

// initializeTables creates the performance regression tables
func (p *PerformanceRegressionTracker) initializeTables() error {
	query := `
		CREATE TABLE IF NOT EXISTS performance_regressions (
			id BIGSERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			metric VARCHAR(100) NOT NULL,
			baseline_value DECIMAL(15,6) NOT NULL,
			current_value DECIMAL(15,6) NOT NULL,
			regression_percent DECIMAL(5,2) NOT NULL,
			detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			severity VARCHAR(50) DEFAULT 'medium',
			status VARCHAR(50) DEFAULT 'open'
		)
	`

	_, err := p.db.Exec(query)
	return err
}

// RecordRegression records a performance regression
func (p *PerformanceRegressionTracker) RecordRegression(regression *PerformanceRegression) error {
	query := `
		INSERT INTO performance_regressions (test_name, metric, baseline_value, current_value, regression_percent, severity)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, detected_at
	`

	err := p.db.QueryRow(query,
		regression.TestName, regression.Metric, regression.BaselineValue,
		regression.CurrentValue, regression.RegressionPct, regression.Severity,
	).Scan(&regression.ID, &regression.DetectedAt)

	return err
}

// GetRegressions returns current performance regressions
func (p *PerformanceRegressionTracker) GetRegressions() ([]PerformanceRegression, error) {
	query := `
		SELECT id, test_name, metric, baseline_value, current_value, regression_percent, detected_at, severity, status
		FROM performance_regressions
		WHERE status != 'resolved'
		ORDER BY regression_percent DESC, detected_at DESC
	`

	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get regressions: %w", err)
	}
	defer rows.Close()

	var regressions []PerformanceRegression
	for rows.Next() {
		var regression PerformanceRegression

		err := rows.Scan(
			&regression.ID, &regression.TestName, &regression.Metric,
			&regression.BaselineValue, &regression.CurrentValue, &regression.RegressionPct,
			&regression.DetectedAt, &regression.Severity, &regression.Status,
		)
		if err != nil {
			continue
		}

		regressions = append(regressions, regression)
	}

	return regressions, nil
}

// QualityGateManager manages quality gates
type QualityGateManager struct {
	db    *sql.DB
	gates []QualityGate
}

// NewQualityGateManager creates a new quality gate manager
func NewQualityGateManager(db *sql.DB) *QualityGateManager {
	manager := &QualityGateManager{db: db}
	manager.initializeGates()
	return manager
}

// initializeGates initializes default quality gates
func (q *QualityGateManager) initializeGates() {
	q.gates = []QualityGate{
		{
			Name:        "Test Success Rate",
			Threshold:   95.0,
			Description: "Minimum test success rate required",
		},
		{
			Name:        "Code Coverage",
			Threshold:   95.0,
			Description: "Minimum code coverage required",
		},
		{
			Name:        "Flaky Test Count",
			Threshold:   5.0,
			Description: "Maximum number of flaky tests allowed",
		},
		{
			Name:        "Security Vulnerabilities",
			Threshold:   0.0,
			Description: "Maximum number of high/critical security vulnerabilities",
		},
		{
			Name:        "Performance Regressions",
			Threshold:   2.0,
			Description: "Maximum number of performance regressions allowed",
		},
	}
}

// EvaluateGates evaluates all quality gates
func (q *QualityGateManager) EvaluateGates(ctx context.Context) (*QualityGateStatus, error) {
	status := &QualityGateStatus{
		Gates:          make([]QualityGate, len(q.gates)),
		LastEvaluation: time.Now(),
		OverallStatus:  "passed",
	}

	// Copy gates and evaluate each one
	copy(status.Gates, q.gates)

	for i := range status.Gates {
		gate := &status.Gates[i]
		
		switch gate.Name {
		case "Test Success Rate":
			metrics, err := q.getTestMetrics()
			if err != nil {
				gate.Status = "error"
				continue
			}
			gate.CurrentValue = metrics.SuccessRate
			
		case "Code Coverage":
			metrics, err := q.getTestMetrics()
			if err != nil {
				gate.Status = "error"
				continue
			}
			gate.CurrentValue = metrics.CoveragePercent
			
		case "Flaky Test Count":
			count, err := q.getFlakyTestCount()
			if err != nil {
				gate.Status = "error"
				continue
			}
			gate.CurrentValue = float64(count)
			
		case "Security Vulnerabilities":
			count, err := q.getSecurityVulnCount()
			if err != nil {
				gate.Status = "error"
				continue
			}
			gate.CurrentValue = float64(count)
			
		case "Performance Regressions":
			count, err := q.getPerformanceRegressionCount()
			if err != nil {
				gate.Status = "error"
				continue
			}
			gate.CurrentValue = float64(count)
		}

		// Determine gate status
		if gate.Status != "error" {
			if (gate.Name == "Security Vulnerabilities" || gate.Name == "Performance Regressions" || gate.Name == "Flaky Test Count") {
				// For these gates, lower is better
				if gate.CurrentValue <= gate.Threshold {
					gate.Status = "passed"
				} else {
					gate.Status = "failed"
					status.OverallStatus = "failed"
				}
			} else {
				// For these gates, higher is better
				if gate.CurrentValue >= gate.Threshold {
					gate.Status = "passed"
				} else {
					gate.Status = "failed"
					status.OverallStatus = "failed"
				}
			}
		} else {
			status.OverallStatus = "error"
		}
	}

	return status, nil
}

// GetStatus returns the current quality gate status
func (q *QualityGateManager) GetStatus() (*QualityGateStatus, error) {
	ctx := context.Background()
	return q.EvaluateGates(ctx)
}

// Helper methods for quality gate evaluation
func (q *QualityGateManager) getTestMetrics() (*ExecutionMetrics, error) {
	query := `
		SELECT 
			COUNT(*) as total_executions,
			COUNT(CASE WHEN status = 'passed' THEN 1 END) as passed_executions,
			AVG(CASE WHEN coverage > 0 THEN coverage END) as avg_coverage
		FROM test_executions 
		WHERE start_time >= NOW() - INTERVAL '24 hours'
	`

	var total, passed int64
	var avgCoverage sql.NullFloat64

	err := q.db.QueryRow(query).Scan(&total, &passed, &avgCoverage)
	if err != nil {
		return nil, err
	}

	metrics := &ExecutionMetrics{
		TotalExecutions:  total,
		PassedExecutions: passed,
	}

	if total > 0 {
		metrics.SuccessRate = float64(passed) / float64(total) * 100
	}

	if avgCoverage.Valid {
		metrics.CoveragePercent = avgCoverage.Float64
	}

	return metrics, nil
}

func (q *QualityGateManager) getFlakyTestCount() (int64, error) {
	var count int64
	err := q.db.QueryRow("SELECT COUNT(*) FROM test_flakiness WHERE flakiness_score > 0.1").Scan(&count)
	return count, err
}

func (q *QualityGateManager) getSecurityVulnCount() (int64, error) {
	var count int64
	err := q.db.QueryRow("SELECT COUNT(*) FROM security_vulnerabilities WHERE status != 'resolved' AND severity IN ('high', 'critical')").Scan(&count)
	return count, err
}

func (q *QualityGateManager) getPerformanceRegressionCount() (int64, error) {
	var count int64
	err := q.db.QueryRow("SELECT COUNT(*) FROM performance_regressions WHERE status != 'resolved'").Scan(&count)
	return count, err
}