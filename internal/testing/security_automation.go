package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SecurityAutomation orchestrates comprehensive security testing
type SecurityAutomation struct {
	projectRoot       string
	reportDir         string
	zapAutomation     *ZAPAutomation
	dependencyScanner *DependencyScanner
	testScenarios     *SecurityTestScenarios
	patchManager      *SecurityPatchManager
	config            *SecurityConfig
}

// SecurityConfig holds configuration for security automation
type SecurityConfig struct {
	EnableZAP              bool              `yaml:"enable_zap"`
	EnableDependencyScans  bool              `yaml:"enable_dependency_scans"`
	FailOnHighRisk         bool              `yaml:"fail_on_high_risk"`
	FailOnMediumRisk       bool              `yaml:"fail_on_medium_risk"`
	MaxHighRiskAlerts      int               `yaml:"max_high_risk_alerts"`
	MaxMediumRiskAlerts    int               `yaml:"max_medium_risk_alerts"`
	ScanTimeout            time.Duration     `yaml:"scan_timeout"`
	NotificationWebhook    string            `yaml:"notification_webhook"`
	SlackWebhook           string            `yaml:"slack_webhook"`
	CustomPolicies         map[string]string `yaml:"custom_policies"`
}

// ComprehensiveSecurityResult combines all security scan results
type ComprehensiveSecurityResult struct {
	ProjectName        string                             `json:"project_name"`
	ProjectPath        string                             `json:"project_path"`
	StartTime          time.Time                          `json:"start_time"`
	EndTime            time.Time                          `json:"end_time"`
	Duration           time.Duration                      `json:"duration"`
	Status             string                             `json:"status"`
	ZAPResults         *ZAPScanResult                     `json:"zap_results,omitempty"`
	DependencyResults  *ComprehensiveDependencyScanResult `json:"dependency_results,omitempty"`
	ScenarioResults    []SecurityTestResult               `json:"scenario_results,omitempty"`
	PatchManagement    *PatchManagementReport             `json:"patch_management,omitempty"`
	SecuritySummary    SecuritySummary                    `json:"security_summary"`
	ComplianceStatus   ComplianceStatus                   `json:"compliance_status"`
	Recommendations    []string                           `json:"recommendations"`
	ReportPaths        []string                           `json:"report_paths"`
	Metadata           map[string]interface{}             `json:"metadata"`
}

// SecuritySummary provides overall security assessment
type SecuritySummary struct {
	TotalIssues           int                    `json:"total_issues"`
	CriticalIssues        int                    `json:"critical_issues"`
	HighRiskIssues        int                    `json:"high_risk_issues"`
	MediumRiskIssues      int                    `json:"medium_risk_issues"`
	LowRiskIssues         int                    `json:"low_risk_issues"`
	VulnerabilityCount    int                    `json:"vulnerability_count"`
	SecurityScore         float64                `json:"security_score"`
	RiskLevel             string                 `json:"risk_level"`
	TopIssues             []TopSecurityIssue     `json:"top_issues"`
	ScannerResults        map[string]ScannerInfo `json:"scanner_results"`
}

// ComplianceStatus tracks security compliance
type ComplianceStatus struct {
	OWASPTop10Compliant   bool     `json:"owasp_top10_compliant"`
	DependencyCompliant   bool     `json:"dependency_compliant"`
	SecurityPolicyCompliant bool   `json:"security_policy_compliant"`
	OverallCompliant      bool     `json:"overall_compliant"`
	FailedChecks          []string `json:"failed_checks"`
	ComplianceScore       float64  `json:"compliance_score"`
}

// TopSecurityIssue represents a high-priority security issue
type TopSecurityIssue struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Severity    string  `json:"severity"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Solution    string  `json:"solution"`
	CVSS        float64 `json:"cvss,omitempty"`
	CWE         string  `json:"cwe,omitempty"`
	Source      string  `json:"source"`
	URL         string  `json:"url,omitempty"`
}

// ScannerInfo provides information about individual scanners
type ScannerInfo struct {
	Name             string        `json:"name"`
	Status           string        `json:"status"`
	Duration         time.Duration `json:"duration"`
	IssuesFound      int           `json:"issues_found"`
	ErrorMessage     string        `json:"error_message,omitempty"`
}

// NewSecurityAutomation creates a new security automation instance
func NewSecurityAutomation(projectRoot string) *SecurityAutomation {
	reportDir := filepath.Join(projectRoot, "reports", "security")
	os.MkdirAll(reportDir, 0755)
	
	config := &SecurityConfig{
		EnableZAP:             true,
		EnableDependencyScans: true,
		FailOnHighRisk:        true,
		FailOnMediumRisk:      false,
		MaxHighRiskAlerts:     0,
		MaxMediumRiskAlerts:   10,
		ScanTimeout:           30 * time.Minute,
		CustomPolicies:        make(map[string]string),
	}
	
	return &SecurityAutomation{
		projectRoot:       projectRoot,
		reportDir:         reportDir,
		zapAutomation:     NewZAPAutomation(projectRoot),
		dependencyScanner: NewDependencyScanner(projectRoot),
		testScenarios:     NewSecurityTestScenarios("http://localhost:8080"),
		patchManager:      NewSecurityPatchManager(projectRoot),
		config:            config,
	}
}

// RunComprehensiveSecurityScan performs all security scans
func (s *SecurityAutomation) RunComprehensiveSecurityScan(ctx context.Context) (*ComprehensiveSecurityResult, error) {
	result := &ComprehensiveSecurityResult{
		ProjectName: filepath.Base(s.projectRoot),
		ProjectPath: s.projectRoot,
		StartTime:   time.Now(),
		Status:      "running",
		Metadata:    make(map[string]interface{}),
	}
	
	log.Printf("Starting comprehensive security scan for: %s", result.ProjectName)
	
	// Create timeout context
	scanCtx, cancel := context.WithTimeout(ctx, s.config.ScanTimeout)
	defer cancel()
	
	// Run dependency vulnerability scanning
	if s.config.EnableDependencyScans {
		log.Println("Running dependency vulnerability scans...")
		depResult, err := s.dependencyScanner.RunComprehensiveDependencyScan(scanCtx)
		if err != nil {
			log.Printf("Dependency scan failed: %v", err)
			result.Metadata["dependency_scan_error"] = err.Error()
		} else {
			result.DependencyResults = depResult
		}
	}

	// Run ZAP dynamic security testing
	if s.config.EnableZAP {
		log.Println("Running OWASP ZAP dynamic security scan...")
		zapResult, err := s.zapAutomation.RunComprehensiveZAPScan(scanCtx)
		if err != nil {
			log.Printf("ZAP scan failed: %v", err)
			result.Metadata["zap_scan_error"] = err.Error()
		} else {
			result.ZAPResults = zapResult
		}
	}

	// Run security test scenarios
	log.Println("Running security test scenarios...")
	scenarioResults := s.testScenarios.RunAllScenarios(scanCtx)
	result.ScenarioResults = scenarioResults

	// Generate patch management report
	if result.DependencyResults != nil {
		log.Println("Generating patch management report...")
		var allVulns []Vulnerability
		for _, scanResult := range result.DependencyResults.ScannerResults {
			allVulns = append(allVulns, scanResult.Vulnerabilities...)
		}
		
		patchReport, err := s.patchManager.GeneratePatchManagementReport(scanCtx, allVulns)
		if err != nil {
			log.Printf("Patch management report failed: %v", err)
			result.Metadata["patch_management_error"] = err.Error()
		} else {
			result.PatchManagement = patchReport
		}
	}
	
	// Calculate comprehensive security summary
	result.SecuritySummary = s.calculateSecuritySummary(result)
	result.ComplianceStatus = s.assessComplianceStatus(result)
	result.Recommendations = s.generateSecurityRecommendations(result)
	
	// Generate comprehensive reports
	reportPaths, err := s.generateComprehensiveReports(result)
	if err != nil {
		log.Printf("Failed to generate some reports: %v", err)
	}
	result.ReportPaths = reportPaths
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Status = "completed"
	
	log.Printf("Comprehensive security scan completed in %v", result.Duration)
	log.Printf("Security Summary: %d total issues (%d critical, %d high, %d medium, %d low)",
		result.SecuritySummary.TotalIssues,
		result.SecuritySummary.CriticalIssues,
		result.SecuritySummary.HighRiskIssues,
		result.SecuritySummary.MediumRiskIssues,
		result.SecuritySummary.LowRiskIssues)
	
	return result, nil
}

// calculateSecuritySummary aggregates security findings from all scanners
func (s *SecurityAutomation) calculateSecuritySummary(result *ComprehensiveSecurityResult) SecuritySummary {
	summary := SecuritySummary{
		ScannerResults: make(map[string]ScannerInfo),
		TopIssues:      []TopSecurityIssue{},
	}
	
	// Aggregate dependency scan results
	if result.DependencyResults != nil {
		summary.VulnerabilityCount += result.DependencyResults.Summary.TotalVulnerabilities
		summary.CriticalIssues += result.DependencyResults.Summary.CriticalVulnerabilities
		summary.HighRiskIssues += result.DependencyResults.Summary.HighVulnerabilities
		summary.MediumRiskIssues += result.DependencyResults.Summary.MediumVulnerabilities
		summary.LowRiskIssues += result.DependencyResults.Summary.LowVulnerabilities
		summary.TotalIssues += result.DependencyResults.Summary.TotalVulnerabilities
		
		// Add scanner results for each dependency scanner
		for scannerName, scannerSummary := range result.DependencyResults.Summary.ScannerResults {
			summary.ScannerResults[scannerName] = ScannerInfo{
				Name:        scannerSummary.ScannerName,
				Status:      scannerSummary.Status,
				Duration:    scannerSummary.Duration,
				IssuesFound: scannerSummary.VulnerabilitiesFound,
			}
		}
		
		// Add top vulnerabilities
		for _, vuln := range result.DependencyResults.Summary.TopVulnerabilities {
			if len(summary.TopIssues) >= 10 { // Limit total top issues
				break
			}
			
			summary.TopIssues = append(summary.TopIssues, TopSecurityIssue{
				ID:          vuln.ID,
				Title:       vuln.Title,
				Severity:    vuln.Severity,
				Category:    "dependency",
				Description: fmt.Sprintf("Vulnerability in %s", vuln.Package),
				CVSS:        vuln.CVSS,
				Source:      vuln.Scanner,
			})
		}
	}

	// Aggregate ZAP scan results
	if result.ZAPResults != nil {
		summary.TotalIssues += result.ZAPResults.Summary.TotalAlerts
		summary.HighRiskIssues += result.ZAPResults.Summary.HighRiskAlerts
		summary.MediumRiskIssues += result.ZAPResults.Summary.MediumRiskAlerts
		summary.LowRiskIssues += result.ZAPResults.Summary.LowRiskAlerts

		summary.ScannerResults["zap"] = ScannerInfo{
			Name:        "OWASP ZAP",
			Status:      result.ZAPResults.Status,
			Duration:    result.ZAPResults.Duration,
			IssuesFound: result.ZAPResults.Summary.TotalAlerts,
		}

		// Add top ZAP alerts
		for _, alert := range result.ZAPResults.Alerts {
			if len(summary.TopIssues) >= 10 {
				break
			}
			if alert.Risk == "high" || alert.Risk == "medium" {
				summary.TopIssues = append(summary.TopIssues, TopSecurityIssue{
					ID:          alert.ID,
					Title:       alert.Alert,
					Severity:    alert.Risk,
					Category:    "web_application",
					Description: alert.Description,
					Solution:    alert.Solution,
					CWE:         alert.CWEId,
					Source:      "zap",
					URL:         alert.URL,
				})
			}
		}
	}

	// Aggregate security test scenario results
	for _, scenarioResult := range result.ScenarioResults {
		if scenarioResult.Status == "failed" {
			summary.TotalIssues++
			// Categorize by severity (assuming medium for failed scenarios)
			summary.MediumRiskIssues++
		}
	}
	
	// Calculate security score (0-100)
	summary.SecurityScore = s.calculateSecurityScore(summary)
	summary.RiskLevel = s.determineRiskLevel(summary)
	
	return summary
}

// assessComplianceStatus evaluates security compliance
func (s *SecurityAutomation) assessComplianceStatus(result *ComprehensiveSecurityResult) ComplianceStatus {
	status := ComplianceStatus{
		FailedChecks: []string{},
	}
	
	// Check dependency compliance
	if result.DependencyResults != nil {
		criticalVulns := result.DependencyResults.Summary.CriticalVulnerabilities
		highVulns := result.DependencyResults.Summary.HighVulnerabilities
		
		status.DependencyCompliant = criticalVulns == 0 && highVulns <= s.config.MaxHighRiskAlerts
		if !status.DependencyCompliant {
			status.FailedChecks = append(status.FailedChecks, 
				fmt.Sprintf("High-risk dependencies found: %d critical, %d high", criticalVulns, highVulns))
		}
	} else {
		status.DependencyCompliant = true // No dependency scan performed
	}
	
	// Check security policy compliance
	status.SecurityPolicyCompliant = s.checkSecurityPolicyCompliance(result)
	if !status.SecurityPolicyCompliant {
		status.FailedChecks = append(status.FailedChecks, "Security policy violations detected")
	}
	
	// Overall compliance
	status.OverallCompliant = status.DependencyCompliant && status.SecurityPolicyCompliant
	
	// Calculate compliance score
	complianceChecks := 2
	passedChecks := 0
	if status.DependencyCompliant {
		passedChecks++
	}
	if status.SecurityPolicyCompliant {
		passedChecks++
	}
	
	status.ComplianceScore = float64(passedChecks) / float64(complianceChecks) * 100
	
	return status
}

// generateSecurityRecommendations creates actionable security recommendations
func (s *SecurityAutomation) generateSecurityRecommendations(result *ComprehensiveSecurityResult) []string {
	var recommendations []string
	
	// Dependency-based recommendations
	if result.DependencyResults != nil {
		if result.DependencyResults.Summary.CriticalVulnerabilities > 0 {
			recommendations = append(recommendations, 
				"CRITICAL: Update dependencies with critical vulnerabilities immediately")
		}
		
		if result.DependencyResults.Summary.HighVulnerabilities > 0 {
			recommendations = append(recommendations, 
				"Update dependencies with high-severity vulnerabilities")
		}
		
		if result.DependencyResults.Summary.UpgradableVulns > 0 {
			recommendations = append(recommendations, 
				fmt.Sprintf("Update %d dependencies to secure versions", 
					result.DependencyResults.Summary.UpgradableVulns))
		}
	}
	
	// General security recommendations
	recommendations = append(recommendations, "Implement automated security scanning in CI/CD pipeline")
	recommendations = append(recommendations, "Set up continuous security monitoring")
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Security posture is good - maintain current practices")
	}
	
	return recommendations
}

// Helper methods
func (s *SecurityAutomation) calculateSecurityScore(summary SecuritySummary) float64 {
	// Base score starts at 100
	score := 100.0
	
	// Deduct points for issues
	score -= float64(summary.CriticalIssues) * 20  // -20 per critical
	score -= float64(summary.HighRiskIssues) * 10  // -10 per high
	score -= float64(summary.MediumRiskIssues) * 5 // -5 per medium
	score -= float64(summary.LowRiskIssues) * 1    // -1 per low
	
	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}
	
	return score
}

func (s *SecurityAutomation) determineRiskLevel(summary SecuritySummary) string {
	if summary.CriticalIssues > 0 {
		return "critical"
	}
	if summary.HighRiskIssues > 5 {
		return "high"
	}
	if summary.HighRiskIssues > 0 || summary.MediumRiskIssues > 10 {
		return "medium"
	}
	if summary.MediumRiskIssues > 0 || summary.LowRiskIssues > 20 {
		return "low"
	}
	return "minimal"
}

func (s *SecurityAutomation) checkSecurityPolicyCompliance(result *ComprehensiveSecurityResult) bool {
	// Check against security policy requirements
	if result.SecuritySummary.CriticalIssues > 0 {
		return false
	}
	
	if result.SecuritySummary.HighRiskIssues > s.config.MaxHighRiskAlerts {
		return false
	}
	
	if result.SecuritySummary.MediumRiskIssues > s.config.MaxMediumRiskAlerts {
		return false
	}
	
	return true
}

// generateComprehensiveReports creates comprehensive security reports
func (s *SecurityAutomation) generateComprehensiveReports(result *ComprehensiveSecurityResult) ([]string, error) {
	timestamp := result.StartTime.Format("20060102-150405")
	var reportPaths []string
	
	// Generate JSON report
	jsonPath := filepath.Join(s.reportDir, fmt.Sprintf("comprehensive-security-report-%s.json", timestamp))
	if err := s.generateJSONSecurityReport(result, jsonPath); err != nil {
		log.Printf("Failed to generate JSON security report: %v", err)
	} else {
		reportPaths = append(reportPaths, jsonPath)
	}
	
	return reportPaths, nil
}

// generateJSONSecurityReport creates a JSON report
func (s *SecurityAutomation) generateJSONSecurityReport(result *ComprehensiveSecurityResult, reportPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(reportPath, data, 0644)
}

// ShouldFailBuild determines if the build should fail based on security results
func (s *SecurityAutomation) ShouldFailBuild(result *ComprehensiveSecurityResult) bool {
	if s.config.FailOnHighRisk && result.SecuritySummary.HighRiskIssues > s.config.MaxHighRiskAlerts {
		return true
	}
	
	if s.config.FailOnMediumRisk && result.SecuritySummary.MediumRiskIssues > s.config.MaxMediumRiskAlerts {
		return true
	}
	
	if result.SecuritySummary.CriticalIssues > 0 {
		return true
	}
	
	return false
}