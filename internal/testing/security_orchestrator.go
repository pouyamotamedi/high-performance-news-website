package testing

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SecurityOrchestrator coordinates all security testing activities
type SecurityOrchestrator struct {
	scanner   *SecurityScanner
	alerting  *SecurityAlerting
	config    *SecurityConfig
	projectRoot string
	reportDir   string
}

// SecurityConfig holds configuration for security testing
type SecurityConfig struct {
	// OWASP ZAP Configuration
	ZAPEnabled    bool   `yaml:"zap_enabled"`
	ZAPConfigPath string `yaml:"zap_config_path"`
	
	// Snyk Configuration
	SnykEnabled    bool   `yaml:"snyk_enabled"`
	SnykConfigPath string `yaml:"snyk_config_path"`
	
	// Static Analysis Configuration
	StaticAnalysisEnabled bool `yaml:"static_analysis_enabled"`
	GosecEnabled         bool `yaml:"gosec_enabled"`
	
	// Reporting Configuration
	GenerateReports bool   `yaml:"generate_reports"`
	ReportFormats   []string `yaml:"report_formats"`
	
	// Alerting Configuration
	AlertingEnabled bool `yaml:"alerting_enabled"`
	
	// CI/CD Integration
	FailOnCritical bool `yaml:"fail_on_critical"`
	FailOnHigh     bool `yaml:"fail_on_high"`
	MaxIssues      int  `yaml:"max_issues"`
	
	// Parallel Execution
	MaxConcurrentScans int `yaml:"max_concurrent_scans"`
	
	// Timeout Settings
	ScanTimeout time.Duration `yaml:"scan_timeout"`
}

// SecurityTestSuite represents a complete security test execution
type SecurityTestSuite struct {
	ID        string                 `json:"id"`
	StartTime time.Time             `json:"start_time"`
	EndTime   time.Time             `json:"end_time"`
	Duration  time.Duration         `json:"duration"`
	Status    string                `json:"status"`
	Results   []SecurityScanResult  `json:"results"`
	Summary   SecurityTestSummary   `json:"summary"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SecurityTestSummary provides an overview of all security tests
type SecurityTestSummary struct {
	TotalScans       int `json:"total_scans"`
	SuccessfulScans  int `json:"successful_scans"`
	FailedScans      int `json:"failed_scans"`
	TotalIssues      int `json:"total_issues"`
	CriticalIssues   int `json:"critical_issues"`
	HighIssues       int `json:"high_issues"`
	MediumIssues     int `json:"medium_issues"`
	LowIssues        int `json:"low_issues"`
	InfoIssues       int `json:"info_issues"`
	RecommendedActions []string `json:"recommended_actions"`
}

// NewSecurityOrchestrator creates a new security orchestrator
func NewSecurityOrchestrator(projectRoot string) *SecurityOrchestrator {
	config := &SecurityConfig{
		ZAPEnabled:            true,
		ZAPConfigPath:         filepath.Join(projectRoot, "configs", "owasp-zap-config.yaml"),
		SnykEnabled:           true,
		SnykConfigPath:        filepath.Join(projectRoot, ".snyk"),
		StaticAnalysisEnabled: true,
		GosecEnabled:          true,
		GenerateReports:       true,
		ReportFormats:         []string{"json", "html"},
		AlertingEnabled:       true,
		FailOnCritical:        true,
		FailOnHigh:           false,
		MaxIssues:            50,
		MaxConcurrentScans:   3,
		ScanTimeout:          30 * time.Minute,
	}

	reportDir := filepath.Join(projectRoot, "reports", "security")
	os.MkdirAll(reportDir, 0755)

	return &SecurityOrchestrator{
		scanner:     NewSecurityScanner(projectRoot),
		alerting:    NewSecurityAlerting(),
		config:      config,
		projectRoot: projectRoot,
		reportDir:   reportDir,
	}
}

// RunComprehensiveSecuritySuite executes all configured security tests
func (s *SecurityOrchestrator) RunComprehensiveSecuritySuite(ctx context.Context) (*SecurityTestSuite, error) {
	suite := &SecurityTestSuite{
		ID:        fmt.Sprintf("security-suite-%d", time.Now().Unix()),
		StartTime: time.Now(),
		Status:    "running",
		Results:   []SecurityScanResult{},
		Metadata:  make(map[string]interface{}),
	}

	log.Printf("Starting comprehensive security test suite: %s", suite.ID)

	// Create context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, s.config.ScanTimeout)
	defer cancel()

	// Run security scans in parallel
	var wg sync.WaitGroup
	resultsChan := make(chan SecurityScanResult, 10)
	errorsChan := make(chan error, 10)

	// Static Analysis Scan
	if s.config.StaticAnalysisEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("Running static security analysis...")
			result, err := s.runStaticAnalysisScan(scanCtx)
			if err != nil {
				errorsChan <- fmt.Errorf("static analysis failed: %w", err)
				return
			}
			resultsChan <- *result
		}()
	}

	// Dependency Vulnerability Scan
	if s.config.SnykEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("Running dependency vulnerability scan...")
			result, err := s.runDependencyVulnerabilityScan(scanCtx)
			if err != nil {
				errorsChan <- fmt.Errorf("dependency scan failed: %w", err)
				return
			}
			resultsChan <- *result
		}()
	}

	// OWASP ZAP Dynamic Scan
	if s.config.ZAPEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("Running OWASP ZAP dynamic security scan...")
			result, err := s.runDynamicSecurityScan(scanCtx)
			if err != nil {
				errorsChan <- fmt.Errorf("dynamic scan failed: %w", err)
				return
			}
			resultsChan <- *result
		}()
	}

	// Wait for all scans to complete
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	// Collect results
	var errors []error
	for {
		select {
		case result, ok := <-resultsChan:
			if !ok {
				resultsChan = nil
			} else {
				suite.Results = append(suite.Results, result)
				log.Printf("Completed %s scan with %d issues", result.ScanType, result.Summary.TotalIssues)
			}
		case err, ok := <-errorsChan:
			if !ok {
				errorsChan = nil
			} else {
				errors = append(errors, err)
				log.Printf("Scan error: %v", err)
			}
		}

		if resultsChan == nil && errorsChan == nil {
			break
		}
	}

	// Calculate suite summary
	suite.Summary = s.calculateSuiteSummary(suite.Results)
	suite.EndTime = time.Now()
	suite.Duration = suite.EndTime.Sub(suite.StartTime)

	// Determine overall status
	if len(errors) > 0 {
		suite.Status = "completed_with_errors"
		suite.Metadata["errors"] = errors
	} else {
		suite.Status = "completed"
	}

	// Generate comprehensive report
	if s.config.GenerateReports {
		reportPath, err := s.generateSuiteReport(suite)
		if err != nil {
			log.Printf("Failed to generate suite report: %v", err)
		} else {
			suite.Metadata["report_path"] = reportPath
			log.Printf("Security test suite report generated: %s", reportPath)
		}
	}

	// Send alerts if configured
	if s.config.AlertingEnabled {
		if err := s.processAlerting(ctx, suite); err != nil {
			log.Printf("Failed to send security alerts: %v", err)
		}
	}

	// Check if build should fail
	if s.shouldFailBuild(suite.Summary) {
		return suite, fmt.Errorf("security test suite failed quality gates: %d critical, %d high issues", 
			suite.Summary.CriticalIssues, suite.Summary.HighIssues)
	}

	log.Printf("Security test suite completed successfully in %v", suite.Duration)
	return suite, nil
}

// runStaticAnalysisScan runs static security analysis
func (s *SecurityOrchestrator) runStaticAnalysisScan(ctx context.Context) (*SecurityScanResult, error) {
	result := &SecurityScanResult{
		Timestamp: time.Now(),
		ScanType:  "static_analysis",
		Status:    "running",
		Issues:    []SecurityIssue{},
		Metadata:  make(map[string]interface{}),
	}

	start := time.Now()

	// Run the scanner's static analysis
	issues, err := s.scanner.runStaticSecurityAnalysis(ctx)
	if err != nil {
		result.Status = "failed"
		result.Metadata["error"] = err.Error()
		return result, err
	}

	result.Issues = issues
	result.Summary = s.scanner.calculateSummary(issues)
	result.Duration = time.Since(start)
	result.Status = "completed"

	return result, nil
}

// runDependencyVulnerabilityScan runs dependency vulnerability scanning
func (s *SecurityOrchestrator) runDependencyVulnerabilityScan(ctx context.Context) (*SecurityScanResult, error) {
	result := &SecurityScanResult{
		Timestamp: time.Now(),
		ScanType:  "dependency_vulnerability",
		Status:    "running",
		Issues:    []SecurityIssue{},
		Metadata:  make(map[string]interface{}),
	}

	start := time.Now()

	// Run the scanner's dependency analysis
	issues, err := s.scanner.runDependencyVulnerabilityScan(ctx)
	if err != nil {
		result.Status = "failed"
		result.Metadata["error"] = err.Error()
		return result, err
	}

	result.Issues = issues
	result.Summary = s.scanner.calculateSummary(issues)
	result.Duration = time.Since(start)
	result.Status = "completed"

	return result, nil
}

// runDynamicSecurityScan runs OWASP ZAP dynamic security scanning
func (s *SecurityOrchestrator) runDynamicSecurityScan(ctx context.Context) (*SecurityScanResult, error) {
	result := &SecurityScanResult{
		Timestamp: time.Now(),
		ScanType:  "dynamic_analysis",
		Status:    "running",
		Issues:    []SecurityIssue{},
		Metadata:  make(map[string]interface{}),
	}

	start := time.Now()

	// Run the scanner's OWASP ZAP analysis
	issues, err := s.scanner.runOWASPZAPScan(ctx)
	if err != nil {
		result.Status = "failed"
		result.Metadata["error"] = err.Error()
		return result, err
	}

	result.Issues = issues
	result.Summary = s.scanner.calculateSummary(issues)
	result.Duration = time.Since(start)
	result.Status = "completed"

	return result, nil
}

// calculateSuiteSummary calculates overall summary for the test suite
func (s *SecurityOrchestrator) calculateSuiteSummary(results []SecurityScanResult) SecurityTestSummary {
	summary := SecurityTestSummary{
		TotalScans: len(results),
	}

	for _, result := range results {
		if result.Status == "completed" {
			summary.SuccessfulScans++
		} else {
			summary.FailedScans++
		}

		summary.TotalIssues += result.Summary.TotalIssues
		summary.CriticalIssues += result.Summary.CriticalIssues
		summary.HighIssues += result.Summary.HighIssues
		summary.MediumIssues += result.Summary.MediumIssues
		summary.LowIssues += result.Summary.LowIssues
		summary.InfoIssues += result.Summary.InfoIssues
	}

	// Generate recommended actions
	summary.RecommendedActions = s.generateRecommendedActions(summary)

	return summary
}

// generateRecommendedActions generates actionable recommendations based on scan results
func (s *SecurityOrchestrator) generateRecommendedActions(summary SecurityTestSummary) []string {
	var actions []string

	if summary.CriticalIssues > 0 {
		actions = append(actions, fmt.Sprintf("URGENT: Address %d critical security issues immediately", summary.CriticalIssues))
	}

	if summary.HighIssues > 5 {
		actions = append(actions, fmt.Sprintf("High Priority: Review and fix %d high-severity issues", summary.HighIssues))
	}

	if summary.MediumIssues > 20 {
		actions = append(actions, "Medium Priority: Consider addressing medium-severity issues in next sprint")
	}

	if summary.FailedScans > 0 {
		actions = append(actions, "Fix security scan failures to ensure complete coverage")
	}

	if summary.TotalIssues == 0 {
		actions = append(actions, "Excellent! No security issues detected. Continue monitoring.")
	}

	return actions
}

// shouldFailBuild determines if the build should fail based on security results
func (s *SecurityOrchestrator) shouldFailBuild(summary SecurityTestSummary) bool {
	if s.config.FailOnCritical && summary.CriticalIssues > 0 {
		return true
	}

	if s.config.FailOnHigh && summary.HighIssues > 0 {
		return true
	}

	if s.config.MaxIssues > 0 && summary.TotalIssues > s.config.MaxIssues {
		return true
	}

	return false
}

// processAlerting sends alerts for security scan results
func (s *SecurityOrchestrator) processAlerting(ctx context.Context, suite *SecurityTestSuite) error {
	// Create a consolidated scan result for alerting
	consolidatedResult := &SecurityScanResult{
		Timestamp: suite.StartTime,
		ScanType:  "comprehensive_security_suite",
		Status:    suite.Status,
		Duration:  suite.Duration,
		Summary: SecuritySummary{
			TotalIssues:    suite.Summary.TotalIssues,
			CriticalIssues: suite.Summary.CriticalIssues,
			HighIssues:     suite.Summary.HighIssues,
			MediumIssues:   suite.Summary.MediumIssues,
			LowIssues:      suite.Summary.LowIssues,
			InfoIssues:     suite.Summary.InfoIssues,
		},
		Metadata: suite.Metadata,
	}

	// Collect all issues from all scans
	var allIssues []SecurityIssue
	for _, result := range suite.Results {
		allIssues = append(allIssues, result.Issues...)
	}
	consolidatedResult.Issues = allIssues

	// Add report path if available
	if reportPath, ok := suite.Metadata["report_path"].(string); ok {
		consolidatedResult.ReportPath = reportPath
	}

	return s.alerting.ProcessSecurityScanResult(ctx, consolidatedResult)
}

// generateSuiteReport generates a comprehensive report for the entire test suite
func (s *SecurityOrchestrator) generateSuiteReport(suite *SecurityTestSuite) (string, error) {
	timestamp := suite.StartTime.Format("20060102-150405")
	reportPath := filepath.Join(s.reportDir, fmt.Sprintf("security-suite-report-%s.json", timestamp))

	// Generate JSON report
	if err := writeJSONReport(suite, reportPath); err != nil {
		return "", fmt.Errorf("failed to write JSON report: %w", err)
	}

	// Generate HTML report if configured
	for _, format := range s.config.ReportFormats {
		if format == "html" {
			htmlReportPath := filepath.Join(s.reportDir, fmt.Sprintf("security-suite-report-%s.html", timestamp))
			if err := s.generateHTMLSuiteReport(suite, htmlReportPath); err != nil {
				log.Printf("Failed to generate HTML report: %v", err)
			}
		}
	}

	return reportPath, nil
}

// generateHTMLSuiteReport generates an HTML report for the security test suite
func (s *SecurityOrchestrator) generateHTMLSuiteReport(suite *SecurityTestSuite, reportPath string) error {
	// This would generate a comprehensive HTML report
	// For brevity, we'll create a basic implementation
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Security Test Suite Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .summary { margin: 20px 0; }
        .scan-result { border: 1px solid #ddd; margin: 10px 0; padding: 15px; border-radius: 5px; }
        .critical { border-left: 5px solid #d32f2f; }
        .high { border-left: 5px solid #f57c00; }
        .medium { border-left: 5px solid #fbc02d; }
        .low { border-left: 5px solid #388e3c; }
        .success { color: #4caf50; }
        .warning { color: #ff9800; }
        .error { color: #f44336; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Security Test Suite Report</h1>
        <p><strong>Suite ID:</strong> %s</p>
        <p><strong>Start Time:</strong> %s</p>
        <p><strong>Duration:</strong> %s</p>
        <p><strong>Status:</strong> <span class="%s">%s</span></p>
    </div>
    
    <div class="summary">
        <h2>Summary</h2>
        <p><strong>Total Scans:</strong> %d</p>
        <p><strong>Successful Scans:</strong> %d</p>
        <p><strong>Failed Scans:</strong> %d</p>
        <p><strong>Total Issues:</strong> %d</p>
        <p><strong>Critical:</strong> %d</p>
        <p><strong>High:</strong> %d</p>
        <p><strong>Medium:</strong> %d</p>
        <p><strong>Low:</strong> %d</p>
        <p><strong>Info:</strong> %d</p>
    </div>
</body>
</html>`,
		suite.ID,
		suite.StartTime.Format("2006-01-02 15:04:05"),
		suite.Duration.String(),
		getStatusClass(suite.Status),
		suite.Status,
		suite.Summary.TotalScans,
		suite.Summary.SuccessfulScans,
		suite.Summary.FailedScans,
		suite.Summary.TotalIssues,
		suite.Summary.CriticalIssues,
		suite.Summary.HighIssues,
		suite.Summary.MediumIssues,
		suite.Summary.LowIssues,
		suite.Summary.InfoIssues,
	)

	return os.WriteFile(reportPath, []byte(htmlContent), 0644)
}

// Helper functions
func getStatusClass(status string) string {
	switch status {
	case "completed":
		return "success"
	case "completed_with_errors":
		return "warning"
	default:
		return "error"
	}
}

// writeJSONReport writes a JSON report to the specified path
func writeJSONReport(data interface{}, path string) error {
	// This would use encoding/json to write the report
	// For now, we'll create a simple implementation
	return fmt.Errorf("JSON report generation not implemented")
}