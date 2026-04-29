package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DependencyScanner handles comprehensive dependency vulnerability scanning
type DependencyScanner struct {
	projectRoot string
	reportDir   string
	config      *DependencyScanConfig
	scanners    map[string]DependencyTool
}

// DependencyScanConfig holds dependency scanning configuration
type DependencyScanConfig struct {
	EnableSnyk       bool          `yaml:"enable_snyk"`
	EnableGovulncheck bool         `yaml:"enable_govulncheck"`
	EnableNancy      bool          `yaml:"enable_nancy"`
	ScanTimeout      time.Duration `yaml:"scan_timeout"`
	FailOnHigh       bool          `yaml:"fail_on_high"`
	FailOnCritical   bool          `yaml:"fail_on_critical"`
	ExcludePackages  []string      `yaml:"exclude_packages"`
	IncludeDevDeps   bool          `yaml:"include_dev_deps"`
}

// DependencyTool interface for different scanning tools
type DependencyTool interface {
	Name() string
	IsAvailable() bool
	Scan(ctx context.Context, projectPath string) (*DependencyScanResult, error)
}

// ComprehensiveDependencyScanResult combines results from all scanners
type ComprehensiveDependencyScanResult struct {
	ProjectPath    string                           `json:"project_path"`
	StartTime      time.Time                        `json:"start_time"`
	EndTime        time.Time                        `json:"end_time"`
	Duration       time.Duration                    `json:"duration"`
	Status         string                           `json:"status"`
	ScannerResults map[string]*DependencyScanResult `json:"scanner_results"`
	Summary        DependencyScanSummary            `json:"summary"`
	ReportPaths    []string                         `json:"report_paths"`
	Metadata       map[string]interface{}           `json:"metadata"`
}

// DependencyScanResult represents results from a single scanner
type DependencyScanResult struct {
	ScannerName        string                 `json:"scanner_name"`
	Status             string                 `json:"status"`
	StartTime          time.Time              `json:"start_time"`
	EndTime            time.Time              `json:"end_time"`
	Duration           time.Duration          `json:"duration"`
	Vulnerabilities    []Vulnerability        `json:"vulnerabilities"`
	PackagesScanned    int                    `json:"packages_scanned"`
	VulnerabilitiesFound int                  `json:"vulnerabilities_found"`
	ErrorMessage       string                 `json:"error_message,omitempty"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// Vulnerability represents a dependency vulnerability
type Vulnerability struct {
	ID              string            `json:"id"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Severity        string            `json:"severity"`
	CVSS            float64           `json:"cvss"`
	CWE             []string          `json:"cwe"`
	Package         string            `json:"package"`
	Version         string            `json:"version"`
	FixedVersion    string            `json:"fixed_version,omitempty"`
	References      []string          `json:"references"`
	PublishedDate   time.Time         `json:"published_date"`
	ModifiedDate    time.Time         `json:"modified_date"`
	Scanner         string            `json:"scanner"`
	Upgradable      bool              `json:"upgradable"`
	Patchable       bool              `json:"patchable"`
	DependencyPath  []string          `json:"dependency_path"`
	Metadata        map[string]string `json:"metadata"`
}

// DependencyScanSummary provides overall scan summary
type DependencyScanSummary struct {
	TotalVulnerabilities    int                              `json:"total_vulnerabilities"`
	CriticalVulnerabilities int                              `json:"critical_vulnerabilities"`
	HighVulnerabilities     int                              `json:"high_vulnerabilities"`
	MediumVulnerabilities   int                              `json:"medium_vulnerabilities"`
	LowVulnerabilities      int                              `json:"low_vulnerabilities"`
	UpgradableVulns         int                              `json:"upgradable_vulns"`
	PatchableVulns          int                              `json:"patchable_vulns"`
	UniquePackages          int                              `json:"unique_packages"`
	TopVulnerabilities      []Vulnerability                  `json:"top_vulnerabilities"`
	ScannerResults          map[string]DependencyScannerInfo `json:"scanner_results"`
	RiskScore               float64                          `json:"risk_score"`
}

// DependencyScannerInfo provides scanner-specific information
type DependencyScannerInfo struct {
	ScannerName         string        `json:"scanner_name"`
	Status              string        `json:"status"`
	Duration            time.Duration `json:"duration"`
	VulnerabilitiesFound int          `json:"vulnerabilities_found"`
	PackagesScanned     int           `json:"packages_scanned"`
	ErrorMessage        string        `json:"error_message,omitempty"`
}

// NewDependencyScanner creates a new dependency scanner instance
func NewDependencyScanner(projectRoot string) *DependencyScanner {
	config := &DependencyScanConfig{
		EnableSnyk:       true,
		EnableGovulncheck: true,
		EnableNancy:      true,
		ScanTimeout:      10 * time.Minute,
		FailOnHigh:       true,
		FailOnCritical:   true,
		IncludeDevDeps:   false,
		ExcludePackages:  []string{},
	}

	scanner := &DependencyScanner{
		projectRoot: projectRoot,
		reportDir:   filepath.Join(projectRoot, "reports", "security", "dependencies"),
		config:      config,
		scanners:    make(map[string]DependencyTool),
	}

	// Initialize available scanners
	scanner.initializeScanners()

	return scanner
}

// initializeScanners initializes available dependency scanning tools
func (d *DependencyScanner) initializeScanners() {
	// Initialize Snyk scanner
	snykScanner := &SnykScanner{
		token:       os.Getenv("SNYK_TOKEN"),
		projectRoot: d.projectRoot,
	}
	if snykScanner.IsAvailable() {
		d.scanners["snyk"] = snykScanner
	}

	// Initialize govulncheck scanner
	govulnScanner := &GovulncheckScanner{
		projectRoot: d.projectRoot,
	}
	if govulnScanner.IsAvailable() {
		d.scanners["govulncheck"] = govulnScanner
	}

	// Initialize Nancy scanner
	nancyScanner := &NancyScanner{
		projectRoot: d.projectRoot,
	}
	if nancyScanner.IsAvailable() {
		d.scanners["nancy"] = nancyScanner
	}

	log.Printf("Initialized %d dependency scanners", len(d.scanners))
}

// RunComprehensiveDependencyScan runs all available dependency scanners
func (d *DependencyScanner) RunComprehensiveDependencyScan(ctx context.Context) (*ComprehensiveDependencyScanResult, error) {
	// Ensure report directory exists
	if err := os.MkdirAll(d.reportDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create dependency report directory: %w", err)
	}

	result := &ComprehensiveDependencyScanResult{
		ProjectPath:    d.projectRoot,
		StartTime:      time.Now(),
		Status:         "running",
		ScannerResults: make(map[string]*DependencyScanResult),
		Metadata:       make(map[string]interface{}),
	}

	log.Printf("Starting comprehensive dependency scan with %d scanners", len(d.scanners))

	// Create timeout context
	scanCtx, cancel := context.WithTimeout(ctx, d.config.ScanTimeout)
	defer cancel()

	// Run each scanner
	for name, scanner := range d.scanners {
		log.Printf("Running %s dependency scan...", name)
		
		scanResult, err := scanner.Scan(scanCtx, d.projectRoot)
		if err != nil {
			log.Printf("%s scan failed: %v", name, err)
			scanResult = &DependencyScanResult{
				ScannerName:  name,
				Status:       "failed",
				StartTime:    time.Now(),
				EndTime:      time.Now(),
				ErrorMessage: err.Error(),
				Metadata:     make(map[string]interface{}),
			}
		}
		
		result.ScannerResults[name] = scanResult
	}

	// Calculate comprehensive summary
	result.Summary = d.calculateComprehensiveSummary(result)
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Status = "completed"

	// Generate reports
	reportPaths, err := d.generateComprehensiveReports(result)
	if err != nil {
		log.Printf("Failed to generate some reports: %v", err)
	}
	result.ReportPaths = reportPaths

	log.Printf("Dependency scan completed: %d vulnerabilities found across %d scanners",
		result.Summary.TotalVulnerabilities, len(d.scanners))

	return result, nil
}

// calculateComprehensiveSummary calculates overall scan summary
func (d *DependencyScanner) calculateComprehensiveSummary(result *ComprehensiveDependencyScanResult) DependencyScanSummary {
	summary := DependencyScanSummary{
		ScannerResults: make(map[string]DependencyScannerInfo),
	}

	allVulns := make(map[string]Vulnerability) // Deduplicate by ID
	packageSet := make(map[string]bool)

	// Aggregate results from all scanners
	for scannerName, scanResult := range result.ScannerResults {
		// Add scanner info
		summary.ScannerResults[scannerName] = DependencyScannerInfo{
			ScannerName:         scanResult.ScannerName,
			Status:              scanResult.Status,
			Duration:            scanResult.Duration,
			VulnerabilitiesFound: scanResult.VulnerabilitiesFound,
			PackagesScanned:     scanResult.PackagesScanned,
			ErrorMessage:        scanResult.ErrorMessage,
		}

		// Aggregate vulnerabilities (deduplicate by ID)
		for _, vuln := range scanResult.Vulnerabilities {
			allVulns[vuln.ID] = vuln
			packageSet[vuln.Package] = true
		}
	}

	// Count vulnerabilities by severity
	var topVulns []Vulnerability
	for _, vuln := range allVulns {
		summary.TotalVulnerabilities++
		
		switch strings.ToLower(vuln.Severity) {
		case "critical":
			summary.CriticalVulnerabilities++
		case "high":
			summary.HighVulnerabilities++
		case "medium":
			summary.MediumVulnerabilities++
		case "low":
			summary.LowVulnerabilities++
		}

		if vuln.Upgradable {
			summary.UpgradableVulns++
		}
		if vuln.Patchable {
			summary.PatchableVulns++
		}

		// Collect top vulnerabilities (critical and high)
		if strings.ToLower(vuln.Severity) == "critical" || strings.ToLower(vuln.Severity) == "high" {
			topVulns = append(topVulns, vuln)
		}
	}

	summary.UniquePackages = len(packageSet)
	summary.TopVulnerabilities = topVulns
	if len(summary.TopVulnerabilities) > 10 {
		summary.TopVulnerabilities = summary.TopVulnerabilities[:10] // Limit to top 10
	}

	// Calculate risk score (0-100)
	summary.RiskScore = d.calculateRiskScore(summary)

	return summary
}

// calculateRiskScore calculates overall risk score
func (d *DependencyScanner) calculateRiskScore(summary DependencyScanSummary) float64 {
	if summary.TotalVulnerabilities == 0 {
		return 0
	}

	// Weight vulnerabilities by severity
	score := float64(summary.CriticalVulnerabilities)*10 +
		float64(summary.HighVulnerabilities)*7 +
		float64(summary.MediumVulnerabilities)*4 +
		float64(summary.LowVulnerabilities)*1

	// Normalize to 0-100 scale (assuming max reasonable score of 100)
	normalizedScore := (score / 100) * 100
	if normalizedScore > 100 {
		normalizedScore = 100
	}

	return normalizedScore
}

// generateComprehensiveReports generates comprehensive dependency reports
func (d *DependencyScanner) generateComprehensiveReports(result *ComprehensiveDependencyScanResult) ([]string, error) {
	timestamp := result.StartTime.Format("20060102-150405")
	var reportPaths []string

	// Generate JSON report
	jsonPath := filepath.Join(d.reportDir, fmt.Sprintf("dependency-scan-%s.json", timestamp))
	if err := d.generateJSONReport(result, jsonPath); err != nil {
		log.Printf("Failed to generate JSON report: %v", err)
	} else {
		reportPaths = append(reportPaths, jsonPath)
	}

	// Generate CSV report for vulnerabilities
	csvPath := filepath.Join(d.reportDir, fmt.Sprintf("vulnerabilities-%s.csv", timestamp))
	if err := d.generateCSVReport(result, csvPath); err != nil {
		log.Printf("Failed to generate CSV report: %v", err)
	} else {
		reportPaths = append(reportPaths, csvPath)
	}

	// Generate security advisory report
	advisoryPath := filepath.Join(d.reportDir, fmt.Sprintf("security-advisory-%s.md", timestamp))
	if err := d.generateSecurityAdvisoryReport(result, advisoryPath); err != nil {
		log.Printf("Failed to generate security advisory report: %v", err)
	} else {
		reportPaths = append(reportPaths, advisoryPath)
	}

	return reportPaths, nil
}

// generateJSONReport generates a JSON report
func (d *DependencyScanner) generateJSONReport(result *ComprehensiveDependencyScanResult, reportPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(reportPath, data, 0644)
}

// generateCSVReport generates a CSV report of vulnerabilities
func (d *DependencyScanner) generateCSVReport(result *ComprehensiveDependencyScanResult, reportPath string) error {
	var csvContent strings.Builder
	csvContent.WriteString("ID,Package,Version,Severity,CVSS,Title,Fixed Version,Upgradable,Scanner\n")

	// Collect all unique vulnerabilities
	allVulns := make(map[string]Vulnerability)
	for _, scanResult := range result.ScannerResults {
		for _, vuln := range scanResult.Vulnerabilities {
			allVulns[vuln.ID] = vuln
		}
	}

	for _, vuln := range allVulns {
		csvContent.WriteString(fmt.Sprintf("%s,%s,%s,%s,%.1f,%s,%s,%t,%s\n",
			vuln.ID,
			vuln.Package,
			vuln.Version,
			vuln.Severity,
			vuln.CVSS,
			strings.ReplaceAll(vuln.Title, ",", ";"), // Escape commas
			vuln.FixedVersion,
			vuln.Upgradable,
			vuln.Scanner,
		))
	}

	return os.WriteFile(reportPath, []byte(csvContent.String()), 0644)
}

// generateSecurityAdvisoryReport generates a markdown security advisory
func (d *DependencyScanner) generateSecurityAdvisoryReport(result *ComprehensiveDependencyScanResult, reportPath string) error {
	var content strings.Builder
	
	content.WriteString("# Security Advisory Report\n\n")
	content.WriteString(fmt.Sprintf("**Generated:** %s\n", result.StartTime.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("**Project:** %s\n", result.ProjectPath))
	content.WriteString(fmt.Sprintf("**Scan Duration:** %v\n\n", result.Duration))

	// Executive Summary
	content.WriteString("## Executive Summary\n\n")
	content.WriteString(fmt.Sprintf("- **Total Vulnerabilities:** %d\n", result.Summary.TotalVulnerabilities))
	content.WriteString(fmt.Sprintf("- **Critical:** %d\n", result.Summary.CriticalVulnerabilities))
	content.WriteString(fmt.Sprintf("- **High:** %d\n", result.Summary.HighVulnerabilities))
	content.WriteString(fmt.Sprintf("- **Medium:** %d\n", result.Summary.MediumVulnerabilities))
	content.WriteString(fmt.Sprintf("- **Low:** %d\n", result.Summary.LowVulnerabilities))
	content.WriteString(fmt.Sprintf("- **Risk Score:** %.1f/100\n\n", result.Summary.RiskScore))

	// Immediate Actions Required
	if result.Summary.CriticalVulnerabilities > 0 || result.Summary.HighVulnerabilities > 0 {
		content.WriteString("## 🚨 Immediate Actions Required\n\n")
		
		for _, vuln := range result.Summary.TopVulnerabilities {
			if strings.ToLower(vuln.Severity) == "critical" || strings.ToLower(vuln.Severity) == "high" {
				content.WriteString(fmt.Sprintf("### %s - %s\n", vuln.Severity, vuln.Package))
				content.WriteString(fmt.Sprintf("- **Vulnerability:** %s\n", vuln.Title))
				content.WriteString(fmt.Sprintf("- **Current Version:** %s\n", vuln.Version))
				if vuln.FixedVersion != "" {
					content.WriteString(fmt.Sprintf("- **Fixed Version:** %s\n", vuln.FixedVersion))
				}
				content.WriteString(fmt.Sprintf("- **CVSS Score:** %.1f\n", vuln.CVSS))
				if vuln.Upgradable {
					content.WriteString("- **Action:** Update package to fixed version\n")
				} else {
					content.WriteString("- **Action:** Review alternatives or apply workarounds\n")
				}
				content.WriteString("\n")
			}
		}
	}

	// Scanner Results
	content.WriteString("## Scanner Results\n\n")
	for scannerName, scannerInfo := range result.Summary.ScannerResults {
		content.WriteString(fmt.Sprintf("### %s\n", strings.Title(scannerName)))
		content.WriteString(fmt.Sprintf("- **Status:** %s\n", scannerInfo.Status))
		content.WriteString(fmt.Sprintf("- **Duration:** %v\n", scannerInfo.Duration))
		content.WriteString(fmt.Sprintf("- **Vulnerabilities Found:** %d\n", scannerInfo.VulnerabilitiesFound))
		content.WriteString(fmt.Sprintf("- **Packages Scanned:** %d\n", scannerInfo.PackagesScanned))
		if scannerInfo.ErrorMessage != "" {
			content.WriteString(fmt.Sprintf("- **Error:** %s\n", scannerInfo.ErrorMessage))
		}
		content.WriteString("\n")
	}

	// Recommendations
	content.WriteString("## Recommendations\n\n")
	content.WriteString("1. **Immediate:** Address all critical and high-severity vulnerabilities\n")
	content.WriteString("2. **Short-term:** Update packages with available fixes\n")
	content.WriteString("3. **Long-term:** Implement automated dependency scanning in CI/CD\n")
	content.WriteString("4. **Ongoing:** Monitor security advisories for used packages\n\n")

	return os.WriteFile(reportPath, []byte(content.String()), 0644)
}

// ShouldFailBuild determines if build should fail based on vulnerabilities
func (d *DependencyScanner) ShouldFailBuild(result *ComprehensiveDependencyScanResult) bool {
	if d.config.FailOnCritical && result.Summary.CriticalVulnerabilities > 0 {
		return true
	}
	if d.config.FailOnHigh && result.Summary.HighVulnerabilities > 0 {
		return true
	}
	return false
}