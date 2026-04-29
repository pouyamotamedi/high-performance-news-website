package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SecurityPatchManager handles security patch management workflow
type SecurityPatchManager struct {
	projectRoot   string
	reportDir     string
	config        *PatchManagementConfig
	advisoryDB    *SecurityAdvisoryDB
}

// PatchManagementConfig holds patch management configuration
type PatchManagementConfig struct {
	AutoApplyPatches      bool          `yaml:"auto_apply_patches"`
	CriticalPatchWindow   time.Duration `yaml:"critical_patch_window"`   // e.g., 24h
	HighPatchWindow       time.Duration `yaml:"high_patch_window"`       // e.g., 7 days
	MediumPatchWindow     time.Duration `yaml:"medium_patch_window"`     // e.g., 30 days
	NotificationChannels  []string      `yaml:"notification_channels"`
	ApprovalRequired      bool          `yaml:"approval_required"`
	TestingRequired       bool          `yaml:"testing_required"`
	RollbackEnabled       bool          `yaml:"rollback_enabled"`
}

// SecurityAdvisoryDB manages security advisory database
type SecurityAdvisoryDB struct {
	advisories map[string]SecurityAdvisory
	lastUpdate time.Time
}

// SecurityAdvisory represents a security advisory
type SecurityAdvisory struct {
	ID              string            `json:"id"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Severity        string            `json:"severity"`
	CVSS            float64           `json:"cvss"`
	CWE             []string          `json:"cwe"`
	AffectedPackages []AffectedPackage `json:"affected_packages"`
	References      []string          `json:"references"`
	PublishedDate   time.Time         `json:"published_date"`
	ModifiedDate    time.Time         `json:"modified_date"`
	PatchAvailable  bool              `json:"patch_available"`
	PatchVersion    string            `json:"patch_version"`
	Workarounds     []string          `json:"workarounds"`
	Metadata        map[string]string `json:"metadata"`
}

// AffectedPackage represents a package affected by a security advisory
type AffectedPackage struct {
	Name            string   `json:"name"`
	Ecosystem       string   `json:"ecosystem"`
	AffectedVersions []string `json:"affected_versions"`
	FixedVersions   []string `json:"fixed_versions"`
}

// PatchRecommendation represents a patch recommendation
type PatchRecommendation struct {
	AdvisoryID      string            `json:"advisory_id"`
	Package         string            `json:"package"`
	CurrentVersion  string            `json:"current_version"`
	RecommendedVersion string         `json:"recommended_version"`
	Severity        string            `json:"severity"`
	CVSS            float64           `json:"cvss"`
	Priority        PatchPriority     `json:"priority"`
	Deadline        time.Time         `json:"deadline"`
	AutoApplicable  bool              `json:"auto_applicable"`
	TestingRequired bool              `json:"testing_required"`
	RiskAssessment  RiskAssessment    `json:"risk_assessment"`
	Metadata        map[string]string `json:"metadata"`
}

// PatchPriority represents patch priority levels
type PatchPriority string

const (
	PatchPriorityImmediate PatchPriority = "immediate"
	PatchPriorityHigh      PatchPriority = "high"
	PatchPriorityMedium    PatchPriority = "medium"
	PatchPriorityLow       PatchPriority = "low"
)

// RiskAssessment represents risk assessment for a patch
type RiskAssessment struct {
	ExploitabilityScore float64 `json:"exploitability_score"`
	ImpactScore         float64 `json:"impact_score"`
	BusinessRisk        string  `json:"business_risk"`
	TechnicalRisk       string  `json:"technical_risk"`
	RecommendedAction   string  `json:"recommended_action"`
}

// PatchManagementReport represents a comprehensive patch management report
type PatchManagementReport struct {
	GeneratedAt       time.Time             `json:"generated_at"`
	ProjectPath       string                `json:"project_path"`
	TotalAdvisories   int                   `json:"total_advisories"`
	Recommendations   []PatchRecommendation `json:"recommendations"`
	Summary           PatchSummary          `json:"summary"`
	ActionPlan        []PatchAction         `json:"action_plan"`
	ComplianceStatus  ComplianceStatus      `json:"compliance_status"`
	ReportPath        string                `json:"report_path"`
}

// PatchSummary provides patch management summary
type PatchSummary struct {
	ImmediatePriority int     `json:"immediate_priority"`
	HighPriority      int     `json:"high_priority"`
	MediumPriority    int     `json:"medium_priority"`
	LowPriority       int     `json:"low_priority"`
	AutoApplicable    int     `json:"auto_applicable"`
	ManualReview      int     `json:"manual_review"`
	OverduePatches    int     `json:"overdue_patches"`
	RiskScore         float64 `json:"risk_score"`
}

// PatchAction represents a recommended patch action
type PatchAction struct {
	ID          string        `json:"id"`
	Type        string        `json:"type"` // update, workaround, monitor
	Description string        `json:"description"`
	Priority    PatchPriority `json:"priority"`
	Deadline    time.Time     `json:"deadline"`
	Commands    []string      `json:"commands"`
	Validation  []string      `json:"validation"`
	Rollback    []string      `json:"rollback"`
}

// NewSecurityPatchManager creates a new security patch manager
func NewSecurityPatchManager(projectRoot string) *SecurityPatchManager {
	config := &PatchManagementConfig{
		AutoApplyPatches:     false, // Conservative default
		CriticalPatchWindow:  24 * time.Hour,
		HighPatchWindow:      7 * 24 * time.Hour,
		MediumPatchWindow:    30 * 24 * time.Hour,
		NotificationChannels: []string{"email", "slack"},
		ApprovalRequired:     true,
		TestingRequired:      true,
		RollbackEnabled:      true,
	}

	return &SecurityPatchManager{
		projectRoot: projectRoot,
		reportDir:   filepath.Join(projectRoot, "reports", "security", "patches"),
		config:      config,
		advisoryDB:  &SecurityAdvisoryDB{
			advisories: make(map[string]SecurityAdvisory),
			lastUpdate: time.Now(),
		},
	}
}

// GeneratePatchManagementReport generates a comprehensive patch management report
func (s *SecurityPatchManager) GeneratePatchManagementReport(ctx context.Context, vulnerabilities []Vulnerability) (*PatchManagementReport, error) {
	// Ensure report directory exists
	if err := os.MkdirAll(s.reportDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create patch report directory: %w", err)
	}

	report := &PatchManagementReport{
		GeneratedAt: time.Now(),
		ProjectPath: s.projectRoot,
	}

	log.Printf("Generating patch management report for %d vulnerabilities", len(vulnerabilities))

	// Update advisory database
	if err := s.updateAdvisoryDatabase(ctx); err != nil {
		log.Printf("Warning: failed to update advisory database: %v", err)
	}

	// Generate patch recommendations
	recommendations := s.generatePatchRecommendations(vulnerabilities)
	report.Recommendations = recommendations
	report.TotalAdvisories = len(recommendations)

	// Calculate summary
	report.Summary = s.calculatePatchSummary(recommendations)

	// Generate action plan
	report.ActionPlan = s.generateActionPlan(recommendations)

	// Assess compliance status
	report.ComplianceStatus = s.assessComplianceStatus(recommendations)

	// Generate report file
	reportPath, err := s.generatePatchReport(report)
	if err != nil {
		log.Printf("Failed to generate patch report file: %v", err)
	} else {
		report.ReportPath = reportPath
	}

	log.Printf("Patch management report generated: %d recommendations, risk score: %.1f",
		len(recommendations), report.Summary.RiskScore)

	return report, nil
}

// updateAdvisoryDatabase updates the security advisory database
func (s *SecurityPatchManager) updateAdvisoryDatabase(ctx context.Context) error {
	// In a real implementation, this would fetch from:
	// - National Vulnerability Database (NVD)
	// - GitHub Security Advisories
	// - Go vulnerability database
	// - Vendor-specific advisories

	log.Println("Updating security advisory database...")

	// Simulate advisory database update
	s.advisoryDB.lastUpdate = time.Now()

	// Add some sample advisories for demonstration
	s.advisoryDB.advisories["CVE-2023-1234"] = SecurityAdvisory{
		ID:          "CVE-2023-1234",
		Title:       "Critical vulnerability in example package",
		Description: "A critical vulnerability that allows remote code execution",
		Severity:    "critical",
		CVSS:        9.8,
		CWE:         []string{"CWE-78"},
		AffectedPackages: []AffectedPackage{
			{
				Name:             "github.com/example/vulnerable",
				Ecosystem:        "go",
				AffectedVersions: []string{"< 1.2.3"},
				FixedVersions:    []string{"1.2.3", "1.3.0"},
			},
		},
		PublishedDate:  time.Now().Add(-7 * 24 * time.Hour),
		ModifiedDate:   time.Now().Add(-1 * 24 * time.Hour),
		PatchAvailable: true,
		PatchVersion:   "1.2.3",
		References:     []string{"https://nvd.nist.gov/vuln/detail/CVE-2023-1234"},
	}

	return nil
}

// generatePatchRecommendations generates patch recommendations based on vulnerabilities
func (s *SecurityPatchManager) generatePatchRecommendations(vulnerabilities []Vulnerability) []PatchRecommendation {
	var recommendations []PatchRecommendation

	for _, vuln := range vulnerabilities {
		recommendation := PatchRecommendation{
			AdvisoryID:         vuln.ID,
			Package:            vuln.Package,
			CurrentVersion:     vuln.Version,
			RecommendedVersion: vuln.FixedVersion,
			Severity:           vuln.Severity,
			CVSS:               vuln.CVSS,
			Priority:           s.calculatePatchPriority(vuln),
			Deadline:           s.calculatePatchDeadline(vuln),
			AutoApplicable:     s.isAutoApplicable(vuln),
			TestingRequired:    s.requiresTesting(vuln),
			RiskAssessment:     s.assessRisk(vuln),
			Metadata: map[string]string{
				"scanner":         vuln.Scanner,
				"upgradable":      fmt.Sprintf("%t", vuln.Upgradable),
				"patchable":       fmt.Sprintf("%t", vuln.Patchable),
			},
		}

		recommendations = append(recommendations, recommendation)
	}

	// Sort by priority and CVSS score
	sort.Slice(recommendations, func(i, j int) bool {
		if recommendations[i].Priority != recommendations[j].Priority {
			return s.priorityWeight(recommendations[i].Priority) > s.priorityWeight(recommendations[j].Priority)
		}
		return recommendations[i].CVSS > recommendations[j].CVSS
	})

	return recommendations
}

// calculatePatchPriority calculates patch priority based on vulnerability
func (s *SecurityPatchManager) calculatePatchPriority(vuln Vulnerability) PatchPriority {
	switch strings.ToLower(vuln.Severity) {
	case "critical":
		return PatchPriorityImmediate
	case "high":
		if vuln.CVSS >= 8.0 {
			return PatchPriorityImmediate
		}
		return PatchPriorityHigh
	case "medium":
		return PatchPriorityMedium
	default:
		return PatchPriorityLow
	}
}

// calculatePatchDeadline calculates patch deadline based on severity
func (s *SecurityPatchManager) calculatePatchDeadline(vuln Vulnerability) time.Time {
	now := time.Now()
	
	switch strings.ToLower(vuln.Severity) {
	case "critical":
		return now.Add(s.config.CriticalPatchWindow)
	case "high":
		return now.Add(s.config.HighPatchWindow)
	case "medium":
		return now.Add(s.config.MediumPatchWindow)
	default:
		return now.Add(90 * 24 * time.Hour) // 90 days for low severity
	}
}

// isAutoApplicable determines if a patch can be auto-applied
func (s *SecurityPatchManager) isAutoApplicable(vuln Vulnerability) bool {
	if !s.config.AutoApplyPatches {
		return false
	}
	
	// Only auto-apply if:
	// 1. Fixed version is available
	// 2. It's an upgradable dependency
	// 3. Severity is not critical (requires manual review)
	return vuln.FixedVersion != "" && 
		   vuln.Upgradable && 
		   strings.ToLower(vuln.Severity) != "critical"
}

// requiresTesting determines if a patch requires testing
func (s *SecurityPatchManager) requiresTesting(vuln Vulnerability) bool {
	if !s.config.TestingRequired {
		return false
	}
	
	// Require testing for:
	// 1. Critical and high severity vulnerabilities
	// 2. Major version updates
	// 3. Core dependencies
	severity := strings.ToLower(vuln.Severity)
	return severity == "critical" || severity == "high"
}

// assessRisk assesses the risk of a vulnerability
func (s *SecurityPatchManager) assessRisk(vuln Vulnerability) RiskAssessment {
	exploitabilityScore := vuln.CVSS / 10.0 // Normalize to 0-1
	impactScore := exploitabilityScore      // Simplified
	
	businessRisk := "medium"
	technicalRisk := "medium"
	
	if vuln.CVSS >= 9.0 {
		businessRisk = "critical"
		technicalRisk = "high"
	} else if vuln.CVSS >= 7.0 {
		businessRisk = "high"
		technicalRisk = "medium"
	}
	
	recommendedAction := "Update to fixed version"
	if vuln.FixedVersion == "" {
		recommendedAction = "Apply workarounds and monitor for patches"
	}
	
	return RiskAssessment{
		ExploitabilityScore: exploitabilityScore,
		ImpactScore:         impactScore,
		BusinessRisk:        businessRisk,
		TechnicalRisk:       technicalRisk,
		RecommendedAction:   recommendedAction,
	}
}

// calculatePatchSummary calculates patch management summary
func (s *SecurityPatchManager) calculatePatchSummary(recommendations []PatchRecommendation) PatchSummary {
	summary := PatchSummary{}
	
	now := time.Now()
	totalRisk := 0.0
	
	for _, rec := range recommendations {
		switch rec.Priority {
		case PatchPriorityImmediate:
			summary.ImmediatePriority++
		case PatchPriorityHigh:
			summary.HighPriority++
		case PatchPriorityMedium:
			summary.MediumPriority++
		case PatchPriorityLow:
			summary.LowPriority++
		}
		
		if rec.AutoApplicable {
			summary.AutoApplicable++
		} else {
			summary.ManualReview++
		}
		
		if rec.Deadline.Before(now) {
			summary.OverduePatches++
		}
		
		totalRisk += rec.CVSS
	}
	
	if len(recommendations) > 0 {
		summary.RiskScore = totalRisk / float64(len(recommendations))
	}
	
	return summary
}

// generateActionPlan generates a prioritized action plan
func (s *SecurityPatchManager) generateActionPlan(recommendations []PatchRecommendation) []PatchAction {
	var actions []PatchAction
	
	for i, rec := range recommendations {
		if rec.RecommendedVersion != "" {
			action := PatchAction{
				ID:          fmt.Sprintf("PATCH-%03d", i+1),
				Type:        "update",
				Description: fmt.Sprintf("Update %s from %s to %s", rec.Package, rec.CurrentVersion, rec.RecommendedVersion),
				Priority:    rec.Priority,
				Deadline:    rec.Deadline,
				Commands: []string{
					fmt.Sprintf("go get %s@%s", rec.Package, rec.RecommendedVersion),
					"go mod tidy",
				},
				Validation: []string{
					"go mod verify",
					"go test ./...",
				},
				Rollback: []string{
					fmt.Sprintf("go get %s@%s", rec.Package, rec.CurrentVersion),
					"go mod tidy",
				},
			}
			actions = append(actions, action)
		} else {
			action := PatchAction{
				ID:          fmt.Sprintf("MONITOR-%03d", i+1),
				Type:        "monitor",
				Description: fmt.Sprintf("Monitor %s for security patches", rec.Package),
				Priority:    rec.Priority,
				Deadline:    rec.Deadline,
				Commands: []string{
					"# No patch available - implement workarounds",
					"# Monitor security advisories",
				},
			}
			actions = append(actions, action)
		}
	}
	
	return actions
}

// assessComplianceStatus assesses patch management compliance
func (s *SecurityPatchManager) assessComplianceStatus(recommendations []PatchRecommendation) ComplianceStatus {
	status := ComplianceStatus{
		FailedChecks: []string{},
	}
	
	now := time.Now()
	overdueCount := 0
	criticalCount := 0
	
	for _, rec := range recommendations {
		if rec.Deadline.Before(now) {
			overdueCount++
		}
		if rec.Priority == PatchPriorityImmediate {
			criticalCount++
		}
	}
	
	// Check compliance criteria
	if overdueCount > 0 {
		status.FailedChecks = append(status.FailedChecks, 
			fmt.Sprintf("%d overdue patches", overdueCount))
	}
	
	if criticalCount > 0 {
		status.FailedChecks = append(status.FailedChecks, 
			fmt.Sprintf("%d immediate priority patches pending", criticalCount))
	}
	
	status.OverallCompliant = len(status.FailedChecks) == 0
	
	// Calculate compliance score
	totalRecommendations := len(recommendations)
	if totalRecommendations > 0 {
		compliantCount := totalRecommendations - overdueCount - criticalCount
		status.ComplianceScore = float64(compliantCount) / float64(totalRecommendations) * 100
	} else {
		status.ComplianceScore = 100
	}
	
	return status
}

// generatePatchReport generates a patch management report file
func (s *SecurityPatchManager) generatePatchReport(report *PatchManagementReport) (string, error) {
	timestamp := report.GeneratedAt.Format("20060102-150405")
	reportPath := filepath.Join(s.reportDir, fmt.Sprintf("patch-management-%s.json", timestamp))
	
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	
	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		return "", err
	}
	
	// Also generate markdown report
	markdownPath := filepath.Join(s.reportDir, fmt.Sprintf("patch-management-%s.md", timestamp))
	if err := s.generateMarkdownReport(report, markdownPath); err != nil {
		log.Printf("Failed to generate markdown report: %v", err)
	}
	
	return reportPath, nil
}

// generateMarkdownReport generates a markdown patch management report
func (s *SecurityPatchManager) generateMarkdownReport(report *PatchManagementReport, reportPath string) error {
	var content strings.Builder
	
	content.WriteString("# Security Patch Management Report\n\n")
	content.WriteString(fmt.Sprintf("**Generated:** %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("**Project:** %s\n\n", report.ProjectPath))
	
	// Executive Summary
	content.WriteString("## Executive Summary\n\n")
	content.WriteString(fmt.Sprintf("- **Total Advisories:** %d\n", report.TotalAdvisories))
	content.WriteString(fmt.Sprintf("- **Immediate Priority:** %d\n", report.Summary.ImmediatePriority))
	content.WriteString(fmt.Sprintf("- **High Priority:** %d\n", report.Summary.HighPriority))
	content.WriteString(fmt.Sprintf("- **Overdue Patches:** %d\n", report.Summary.OverduePatches))
	content.WriteString(fmt.Sprintf("- **Risk Score:** %.1f/10\n", report.Summary.RiskScore))
	content.WriteString(fmt.Sprintf("- **Compliance Score:** %.1f%%\n\n", report.ComplianceStatus.ComplianceScore))
	
	// Immediate Actions
	if report.Summary.ImmediatePriority > 0 {
		content.WriteString("## 🚨 Immediate Actions Required\n\n")
		for _, rec := range report.Recommendations {
			if rec.Priority == PatchPriorityImmediate {
				content.WriteString(fmt.Sprintf("### %s\n", rec.Package))
				content.WriteString(fmt.Sprintf("- **Current Version:** %s\n", rec.CurrentVersion))
				content.WriteString(fmt.Sprintf("- **Recommended Version:** %s\n", rec.RecommendedVersion))
				content.WriteString(fmt.Sprintf("- **CVSS Score:** %.1f\n", rec.CVSS))
				content.WriteString(fmt.Sprintf("- **Deadline:** %s\n", rec.Deadline.Format("2006-01-02")))
				content.WriteString("\n")
			}
		}
	}
	
	// Action Plan
	content.WriteString("## Action Plan\n\n")
	for _, action := range report.ActionPlan {
		content.WriteString(fmt.Sprintf("### %s - %s\n", action.ID, action.Type))
		content.WriteString(fmt.Sprintf("**Description:** %s\n", action.Description))
		content.WriteString(fmt.Sprintf("**Priority:** %s\n", action.Priority))
		content.WriteString(fmt.Sprintf("**Deadline:** %s\n\n", action.Deadline.Format("2006-01-02")))
		
		if len(action.Commands) > 0 {
			content.WriteString("**Commands:**\n```bash\n")
			for _, cmd := range action.Commands {
				content.WriteString(cmd + "\n")
			}
			content.WriteString("```\n\n")
		}
	}
	
	return os.WriteFile(reportPath, []byte(content.String()), 0644)
}

// Helper functions
func (s *SecurityPatchManager) priorityWeight(priority PatchPriority) int {
	switch priority {
	case PatchPriorityImmediate:
		return 4
	case PatchPriorityHigh:
		return 3
	case PatchPriorityMedium:
		return 2
	case PatchPriorityLow:
		return 1
	default:
		return 0
	}
}