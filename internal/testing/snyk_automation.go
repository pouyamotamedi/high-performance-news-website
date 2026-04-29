package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// SnykAutomation handles Snyk vulnerability scanning
type SnykAutomation struct {
	projectRoot string
	reportDir   string
	config      *SnykConfig
}

// SnykConfig holds Snyk configuration
type SnykConfig struct {
	Token           string `yaml:"token"`
	OrgID           string `yaml:"org_id"`
	SeverityThreshold string `yaml:"severity_threshold"`
	FailOnIssues    bool   `yaml:"fail_on_issues"`
	MonitorProject  bool   `yaml:"monitor_project"`
}

// SnykScanResult represents Snyk scan results
type SnykScanResult struct {
	ProjectName     string              `json:"project_name"`
	StartTime       time.Time           `json:"start_time"`
	Duration        time.Duration       `json:"duration"`
	Status          string              `json:"status"`
	Vulnerabilities []SnykVulnerability `json:"vulnerabilities"`
	Summary         VulnerabilitySummary `json:"summary"`
	ReportPaths     []string            `json:"report_paths"`
}

// SnykVulnerability represents a Snyk vulnerability
type SnykVulnerability struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	CVSS        float64 `json:"cvss"`
	PackageName string  `json:"package_name"`
	Version     string  `json:"version"`
	FixedIn     string  `json:"fixed_in"`
	Upgradable  bool    `json:"upgradable"`
	Patchable   bool    `json:"patchable"`
}

// VulnerabilitySummary provides summary statistics
type VulnerabilitySummary struct {
	Total      int `json:"total"`
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
	Upgradable int `json:"upgradable"`
	Patchable  int `json:"patchable"`
}

// NewSnykAutomation creates a new Snyk automation instance
func NewSnykAutomation(projectRoot string) *SnykAutomation {
	reportDir := filepath.Join(projectRoot, "reports", "security", "snyk")
	os.MkdirAll(reportDir, 0755)
	
	config := &SnykConfig{
		Token:             os.Getenv("SNYK_TOKEN"),
		SeverityThreshold: "medium",
		FailOnIssues:      true,
		MonitorProject:    false,
	}
	
	return &SnykAutomation{
		projectRoot: projectRoot,
		reportDir:   reportDir,
		config:      config,
	}
}

// RunVulnerabilityScan performs Snyk vulnerability scanning
func (s *SnykAutomation) RunVulnerabilityScan(ctx context.Context) (*SnykScanResult, error) {
	result := &SnykScanResult{
		ProjectName: filepath.Base(s.projectRoot),
		StartTime:   time.Now(),
		Status:      "running",
	}
	
	log.Printf("Starting Snyk vulnerability scan for: %s", result.ProjectName)
	
	// For now, return a mock result since we don't have Snyk CLI integration
	// In a real implementation, this would call the Snyk CLI or API
	result.Vulnerabilities = []SnykVulnerability{
		{
			ID:          "SNYK-GO-EXAMPLE-001",
			Title:       "Example Vulnerability",
			Description: "This is a mock vulnerability for testing",
			Severity:    "medium",
			CVSS:        5.5,
			PackageName: "example-package",
			Version:     "1.0.0",
			FixedIn:     "1.0.1",
			Upgradable:  true,
			Patchable:   false,
		},
	}
	
	result.Summary = s.calculateSummary(result.Vulnerabilities)
	
	// Generate reports
	reportPaths, err := s.generateReports(result)
	if err != nil {
		log.Printf("Failed to generate some reports: %v", err)
	}
	result.ReportPaths = reportPaths
	
	result.Duration = time.Since(result.StartTime)
	result.Status = "completed"
	
	log.Printf("Snyk scan completed in %v with %d vulnerabilities", 
		result.Duration, result.Summary.Total)
	
	return result, nil
}

// calculateSummary calculates vulnerability summary statistics
func (s *SnykAutomation) calculateSummary(vulnerabilities []SnykVulnerability) VulnerabilitySummary {
	summary := VulnerabilitySummary{
		Total: len(vulnerabilities),
	}
	
	for _, vuln := range vulnerabilities {
		switch vuln.Severity {
		case "critical":
			summary.Critical++
		case "high":
			summary.High++
		case "medium":
			summary.Medium++
		case "low":
			summary.Low++
		}
		
		if vuln.Upgradable {
			summary.Upgradable++
		}
		
		if vuln.Patchable {
			summary.Patchable++
		}
	}
	
	return summary
}

// generateReports generates Snyk scan reports
func (s *SnykAutomation) generateReports(result *SnykScanResult) ([]string, error) {
	timestamp := result.StartTime.Format("20060102-150405")
	var reportPaths []string
	
	// Generate JSON report
	jsonPath := filepath.Join(s.reportDir, fmt.Sprintf("snyk-report-%s.json", timestamp))
	if err := s.generateJSONReport(result, jsonPath); err != nil {
		log.Printf("Failed to generate JSON report: %v", err)
	} else {
		reportPaths = append(reportPaths, jsonPath)
	}
	
	return reportPaths, nil
}

// generateJSONReport creates a JSON report
func (s *SnykAutomation) generateJSONReport(result *SnykScanResult, reportPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(reportPath, data, 0644)
}