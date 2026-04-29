package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SecurityScanner handles automated security testing
type SecurityScanner struct {
	zapAPIURL    string
	zapAPIKey    string
	snykToken    string
	projectRoot  string
	httpClient   *http.Client
	reportDir    string
}

// SecurityScanResult represents the result of a security scan
type SecurityScanResult struct {
	Timestamp    time.Time              `json:"timestamp"`
	ScanType     string                 `json:"scan_type"`
	Status       string                 `json:"status"`
	Issues       []SecurityIssue        `json:"issues"`
	Summary      SecuritySummary        `json:"summary"`
	Duration     time.Duration          `json:"duration"`
	ReportPath   string                 `json:"report_path"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// SecurityIssue represents a security vulnerability or issue
type SecurityIssue struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	Confidence  string            `json:"confidence"`
	Category    string            `json:"category"`
	File        string            `json:"file,omitempty"`
	Line        int               `json:"line,omitempty"`
	CWE         string            `json:"cwe,omitempty"`
	CVSS        float64           `json:"cvss,omitempty"`
	Solution    string            `json:"solution,omitempty"`
	References  []string          `json:"references,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// SecuritySummary provides an overview of security scan results
type SecuritySummary struct {
	TotalIssues    int `json:"total_issues"`
	CriticalIssues int `json:"critical_issues"`
	HighIssues     int `json:"high_issues"`
	MediumIssues   int `json:"medium_issues"`
	LowIssues      int `json:"low_issues"`
	InfoIssues     int `json:"info_issues"`
}

// NewSecurityScanner creates a new security scanner instance
func NewSecurityScanner(projectRoot string) *SecurityScanner {
	return &SecurityScanner{
		zapAPIURL:   getEnvOrDefault("ZAP_API_URL", "http://localhost:8080"),
		zapAPIKey:   os.Getenv("ZAP_API_KEY"),
		snykToken:   os.Getenv("SNYK_TOKEN"),
		projectRoot: projectRoot,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		reportDir: filepath.Join(projectRoot, "reports", "security"),
	}
}

// RunFullSecurityScan performs a comprehensive security scan
func (s *SecurityScanner) RunFullSecurityScan(ctx context.Context) (*SecurityScanResult, error) {
	start := time.Now()
	
	// Ensure report directory exists
	if err := os.MkdirAll(s.reportDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create report directory: %w", err)
	}

	result := &SecurityScanResult{
		Timestamp: start,
		ScanType:  "full_security_scan",
		Status:    "running",
		Issues:    []SecurityIssue{},
		Metadata:  make(map[string]interface{}),
	}

	// Run static security analysis (gosec is already configured in golangci-lint)
	staticResults, err := s.runStaticSecurityAnalysis(ctx)
	if err != nil {
		log.Printf("Static security analysis failed: %v", err)
		result.Metadata["static_analysis_error"] = err.Error()
	} else {
		result.Issues = append(result.Issues, staticResults...)
	}

	// Run dependency vulnerability scan
	depResults, err := s.runDependencyVulnerabilityScan(ctx)
	if err != nil {
		log.Printf("Dependency vulnerability scan failed: %v", err)
		result.Metadata["dependency_scan_error"] = err.Error()
	} else {
		result.Issues = append(result.Issues, depResults...)
	}

	// Run OWASP ZAP scan if available
	zapResults, err := s.runOWASPZAPScan(ctx)
	if err != nil {
		log.Printf("OWASP ZAP scan failed: %v", err)
		result.Metadata["zap_scan_error"] = err.Error()
	} else {
		result.Issues = append(result.Issues, zapResults...)
	}

	// Calculate summary
	result.Summary = s.calculateSummary(result.Issues)
	result.Duration = time.Since(start)
	result.Status = "completed"

	// Generate report
	reportPath, err := s.generateSecurityReport(result)
	if err != nil {
		log.Printf("Failed to generate security report: %v", err)
	} else {
		result.ReportPath = reportPath
	}

	return result, nil
}

// runStaticSecurityAnalysis runs gosec and other static analysis tools
func (s *SecurityScanner) runStaticSecurityAnalysis(ctx context.Context) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	// Run gosec directly for detailed security analysis
	cmd := exec.CommandContext(ctx, "gosec", "-fmt", "json", "-out", "gosec-report.json", "./...")
	cmd.Dir = s.projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// gosec returns non-zero exit code when issues are found
		log.Printf("gosec output: %s", string(output))
	}

	// Parse gosec results
	gosecReportPath := filepath.Join(s.projectRoot, "gosec-report.json")
	if _, err := os.Stat(gosecReportPath); err == nil {
		gosecIssues, err := s.parseGosecReport(gosecReportPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gosec report: %w", err)
		}
		issues = append(issues, gosecIssues...)
		
		// Clean up temporary file
		os.Remove(gosecReportPath)
	}

	return issues, nil
}

// runDependencyVulnerabilityScan scans for known vulnerabilities in dependencies
func (s *SecurityScanner) runDependencyVulnerabilityScan(ctx context.Context) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	if s.snykToken == "" {
		log.Println("SNYK_TOKEN not set, skipping Snyk vulnerability scan")
		return issues, nil
	}

	// Run Snyk test
	cmd := exec.CommandContext(ctx, "snyk", "test", "--json")
	cmd.Dir = s.projectRoot
	cmd.Env = append(os.Environ(), "SNYK_TOKEN="+s.snykToken)

	output, err := cmd.Output()
	if err != nil {
		// Snyk returns non-zero exit code when vulnerabilities are found
		log.Printf("Snyk scan completed with issues")
	}

	// Parse Snyk results
	if len(output) > 0 {
		snykIssues, err := s.parseSnykReport(output)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Snyk report: %w", err)
		}
		issues = append(issues, snykIssues...)
	}

	return issues, nil
}

// runOWASPZAPScan performs dynamic security testing using OWASP ZAP
func (s *SecurityScanner) runOWASPZAPScan(ctx context.Context) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	if s.zapAPIKey == "" {
		log.Println("ZAP_API_KEY not set, skipping OWASP ZAP scan")
		return issues, nil
	}

	// Check if ZAP is running
	if !s.isZAPRunning() {
		log.Println("OWASP ZAP is not running, skipping dynamic security scan")
		return issues, nil
	}

	// Start a new session
	sessionName := fmt.Sprintf("security-scan-%d", time.Now().Unix())
	if err := s.zapNewSession(sessionName); err != nil {
		return nil, fmt.Errorf("failed to create ZAP session: %w", err)
	}

	// Define target URL (assuming local development server)
	targetURL := "http://localhost:8080"

	// Spider the application
	if err := s.zapSpider(targetURL); err != nil {
		log.Printf("ZAP spider failed: %v", err)
	}

	// Run active scan
	if err := s.zapActiveScan(targetURL); err != nil {
		log.Printf("ZAP active scan failed: %v", err)
	}

	// Get scan results
	zapIssues, err := s.zapGetAlerts()
	if err != nil {
		return nil, fmt.Errorf("failed to get ZAP alerts: %w", err)
	}

	issues = append(issues, zapIssues...)

	return issues, nil
}

// parseGosecReport parses gosec JSON output
func (s *SecurityScanner) parseGosecReport(reportPath string) ([]SecurityIssue, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, err
	}

	var gosecReport struct {
		Issues []struct {
			Severity   string `json:"severity"`
			Confidence string `json:"confidence"`
			CWE        struct {
				ID string `json:"id"`
			} `json:"cwe"`
			RuleID      string `json:"rule_id"`
			Details     string `json:"details"`
			File        string `json:"file"`
			Code        string `json:"code"`
			Line        string `json:"line"`
			Column      string `json:"column"`
		} `json:"Issues"`
	}

	if err := json.Unmarshal(data, &gosecReport); err != nil {
		return nil, err
	}

	var issues []SecurityIssue
	for _, issue := range gosecReport.Issues {
		securityIssue := SecurityIssue{
			ID:          issue.RuleID,
			Title:       fmt.Sprintf("Security Issue: %s", issue.RuleID),
			Description: issue.Details,
			Severity:    strings.ToLower(issue.Severity),
			Confidence:  strings.ToLower(issue.Confidence),
			Category:    "static_analysis",
			File:        issue.File,
			CWE:         issue.CWE.ID,
			Solution:    "Review the code and apply appropriate security measures",
		}

		// Convert line number
		if issue.Line != "" {
			fmt.Sscanf(issue.Line, "%d", &securityIssue.Line)
		}

		issues = append(issues, securityIssue)
	}

	return issues, nil
}

// parseSnykReport parses Snyk JSON output
func (s *SecurityScanner) parseSnykReport(data []byte) ([]SecurityIssue, error) {
	var snykReport struct {
		Vulnerabilities []struct {
			ID          string  `json:"id"`
			Title       string  `json:"title"`
			Description string  `json:"description"`
			Severity    string  `json:"severity"`
			CVSS        float64 `json:"cvssScore"`
			CWE         []string `json:"cwe"`
			References  []struct {
				URL string `json:"url"`
			} `json:"references"`
			From []string `json:"from"`
		} `json:"vulnerabilities"`
	}

	if err := json.Unmarshal(data, &snykReport); err != nil {
		return nil, err
	}

	var issues []SecurityIssue
	for _, vuln := range snykReport.Vulnerabilities {
		var references []string
		for _, ref := range vuln.References {
			references = append(references, ref.URL)
		}

		var cwe string
		if len(vuln.CWE) > 0 {
			cwe = vuln.CWE[0]
		}

		securityIssue := SecurityIssue{
			ID:          vuln.ID,
			Title:       vuln.Title,
			Description: vuln.Description,
			Severity:    strings.ToLower(vuln.Severity),
			Category:    "dependency_vulnerability",
			CVSS:        vuln.CVSS,
			CWE:         cwe,
			References:  references,
			Solution:    "Update the vulnerable dependency to a secure version",
			Metadata: map[string]string{
				"dependency_path": strings.Join(vuln.From, " > "),
			},
		}

		issues = append(issues, securityIssue)
	}

	return issues, nil
}

// ZAP API helper methods
func (s *SecurityScanner) isZAPRunning() bool {
	resp, err := s.httpClient.Get(fmt.Sprintf("%s/JSON/core/view/version/?apikey=%s", s.zapAPIURL, s.zapAPIKey))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (s *SecurityScanner) zapNewSession(sessionName string) error {
	url := fmt.Sprintf("%s/JSON/core/action/newSession/?apikey=%s&name=%s", s.zapAPIURL, s.zapAPIKey, sessionName)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *SecurityScanner) zapSpider(targetURL string) error {
	url := fmt.Sprintf("%s/JSON/spider/action/scan/?apikey=%s&url=%s", s.zapAPIURL, s.zapAPIKey, targetURL)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Wait for spider to complete (simplified)
	time.Sleep(30 * time.Second)
	return nil
}

func (s *SecurityScanner) zapActiveScan(targetURL string) error {
	url := fmt.Sprintf("%s/JSON/ascan/action/scan/?apikey=%s&url=%s", s.zapAPIURL, s.zapAPIKey, targetURL)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Wait for active scan to complete (simplified)
	time.Sleep(60 * time.Second)
	return nil
}

func (s *SecurityScanner) zapGetAlerts() ([]SecurityIssue, error) {
	url := fmt.Sprintf("%s/JSON/core/view/alerts/?apikey=%s", s.zapAPIURL, s.zapAPIKey)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var zapResponse struct {
		Alerts []struct {
			Alert       string `json:"alert"`
			Risk        string `json:"risk"`
			Confidence  string `json:"confidence"`
			Description string `json:"description"`
			Solution    string `json:"solution"`
			Reference   string `json:"reference"`
			CWEId       string `json:"cweid"`
			WASCId      string `json:"wascid"`
			URL         string `json:"url"`
		} `json:"alerts"`
	}

	if err := json.Unmarshal(body, &zapResponse); err != nil {
		return nil, err
	}

	var issues []SecurityIssue
	for _, alert := range zapResponse.Alerts {
		issue := SecurityIssue{
			ID:          fmt.Sprintf("ZAP-%s", alert.CWEId),
			Title:       alert.Alert,
			Description: alert.Description,
			Severity:    strings.ToLower(alert.Risk),
			Confidence:  strings.ToLower(alert.Confidence),
			Category:    "dynamic_analysis",
			CWE:         alert.CWEId,
			Solution:    alert.Solution,
			References:  []string{alert.Reference},
			Metadata: map[string]string{
				"url":     alert.URL,
				"wasc_id": alert.WASCId,
			},
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// calculateSummary calculates security issue summary
func (s *SecurityScanner) calculateSummary(issues []SecurityIssue) SecuritySummary {
	summary := SecuritySummary{}
	
	for _, issue := range issues {
		summary.TotalIssues++
		switch issue.Severity {
		case "critical":
			summary.CriticalIssues++
		case "high":
			summary.HighIssues++
		case "medium":
			summary.MediumIssues++
		case "low":
			summary.LowIssues++
		case "info", "informational":
			summary.InfoIssues++
		}
	}
	
	return summary
}

// generateSecurityReport generates a comprehensive security report
func (s *SecurityScanner) generateSecurityReport(result *SecurityScanResult) (string, error) {
	timestamp := result.Timestamp.Format("20060102-150405")
	reportPath := filepath.Join(s.reportDir, fmt.Sprintf("security-report-%s.json", timestamp))
	
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	
	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		return "", err
	}
	
	// Also generate HTML report
	htmlReportPath := filepath.Join(s.reportDir, fmt.Sprintf("security-report-%s.html", timestamp))
	if err := s.generateHTMLReport(result, htmlReportPath); err != nil {
		log.Printf("Failed to generate HTML report: %v", err)
	}
	
	return reportPath, nil
}

// generateHTMLReport generates an HTML security report
func (s *SecurityScanner) generateHTMLReport(result *SecurityScanResult, reportPath string) error {
	// For simplicity, we'll create a basic HTML report
	// In a real implementation, you'd use Go's template package
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Security Scan Report</title>
</head>
<body>
    <h1>Security Scan Report</h1>
    <p>Scan Type: %s</p>
    <p>Status: %s</p>
    <p>Total Issues: %d</p>
</body>
</html>`, result.ScanType, result.Status, result.Summary.TotalIssues)

	return os.WriteFile(reportPath, []byte(htmlContent), 0644)
}

