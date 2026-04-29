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

// SecurityRegressionTester handles security regression testing
type SecurityRegressionTester struct {
	projectRoot     string
	baselineDir     string
	securityAuto    *SecurityAutomation
	config          *RegressionConfig
}

// RegressionConfig holds configuration for regression testing
type RegressionConfig struct {
	BaselineThreshold       float64           `yaml:"baseline_threshold"`
	MaxRegressionPercent    float64           `yaml:"max_regression_percent"`
	CriticalIssueThreshold  int               `yaml:"critical_issue_threshold"`
	HighIssueThreshold      int               `yaml:"high_issue_threshold"`
	AutoUpdateBaseline      bool              `yaml:"auto_update_baseline"`
	RegressionAlertWebhook  string            `yaml:"regression_alert_webhook"`
	IgnorePatterns          []string          `yaml:"ignore_patterns"`
	CustomRules             map[string]string `yaml:"custom_rules"`
}

// SecurityBaseline represents a security baseline
type SecurityBaseline struct {
	ProjectName       string                 `json:"project_name"`
	BaselineDate      time.Time              `json:"baseline_date"`
	GitCommit         string                 `json:"git_commit"`
	SecurityScore     float64                `json:"security_score"`
	IssueCount        SecurityIssueCount     `json:"issue_count"`
	KnownIssues       []BaselineIssue        `json:"known_issues"`
	ScannerVersions   map[string]string      `json:"scanner_versions"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// SecurityIssueCount tracks issue counts by severity
type SecurityIssueCount struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Total    int `json:"total"`
}

// BaselineIssue represents a known issue in the baseline
type BaselineIssue struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	Source      string    `json:"source"`
	Hash        string    `json:"hash"`
	FirstSeen   time.Time `json:"first_seen"`
	Suppressed  bool      `json:"suppressed"`
	Reason      string    `json:"reason"`
}

// RegressionResult represents the result of a regression test
type RegressionResult struct {
	TestDate          time.Time              `json:"test_date"`
	BaselineUsed      string                 `json:"baseline_used"`
	CurrentResults    *ComprehensiveSecurityResult `json:"current_results"`
	Baseline          *SecurityBaseline      `json:"baseline"`
	RegressionFound   bool                   `json:"regression_found"`
	ScoreRegression   float64                `json:"score_regression"`
	NewIssues         []SecurityIssue        `json:"new_issues"`
	ResolvedIssues    []SecurityIssue        `json:"resolved_issues"`
	RegressionSummary RegressionSummary      `json:"regression_summary"`
	Recommendations   []string               `json:"recommendations"`
}

// SecurityIssue represents a security issue for regression tracking
type SecurityIssue struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	Source      string    `json:"source"`
	Hash        string    `json:"hash"`
	Description string    `json:"description"`
	FirstSeen   time.Time `json:"first_seen"`
}

// RegressionSummary provides summary of regression analysis
type RegressionSummary struct {
	NewCriticalIssues int     `json:"new_critical_issues"`
	NewHighIssues     int     `json:"new_high_issues"`
	NewMediumIssues   int     `json:"new_medium_issues"`
	NewLowIssues      int     `json:"new_low_issues"`
	ResolvedIssues    int     `json:"resolved_issues"`
	ScoreChange       float64 `json:"score_change"`
	RegressionLevel   string  `json:"regression_level"`
}

// NewSecurityRegressionTester creates a new regression tester
func NewSecurityRegressionTester(projectRoot string) *SecurityRegressionTester {
	baselineDir := filepath.Join(projectRoot, "reports", "security", "baselines")
	os.MkdirAll(baselineDir, 0755)
	
	config := &RegressionConfig{
		BaselineThreshold:      80.0,
		MaxRegressionPercent:   10.0,
		CriticalIssueThreshold: 0,
		HighIssueThreshold:     5,
		AutoUpdateBaseline:     false,
		IgnorePatterns:         []string{},
		CustomRules:            make(map[string]string),
	}
	
	return &SecurityRegressionTester{
		projectRoot:  projectRoot,
		baselineDir:  baselineDir,
		securityAuto: NewSecurityAutomation(projectRoot),
		config:       config,
	}
}

// EstablishBaseline creates a new security baseline
func (s *SecurityRegressionTester) EstablishBaseline(ctx context.Context) (*SecurityBaseline, error) {
	log.Println("Establishing new security baseline...")
	
	// Run comprehensive security scan
	scanResult, err := s.securityAuto.RunComprehensiveSecurityScan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run security scan for baseline: %w", err)
	}
	
	// Get git commit hash
	gitCommit := s.getCurrentGitCommit()
	
	// Create baseline
	baseline := &SecurityBaseline{
		ProjectName:  scanResult.ProjectName,
		BaselineDate: time.Now(),
		GitCommit:    gitCommit,
		SecurityScore: scanResult.SecuritySummary.SecurityScore,
		IssueCount: SecurityIssueCount{
			Critical: scanResult.SecuritySummary.CriticalIssues,
			High:     scanResult.SecuritySummary.HighRiskIssues,
			Medium:   scanResult.SecuritySummary.MediumRiskIssues,
			Low:      scanResult.SecuritySummary.LowRiskIssues,
			Total:    scanResult.SecuritySummary.TotalIssues,
		},
		KnownIssues:     s.convertToBaselineIssues(scanResult.SecuritySummary.TopIssues),
		ScannerVersions: s.getScannerVersions(),
		Metadata:        make(map[string]interface{}),
	}
	
	// Save baseline
	if err := s.saveBaseline(baseline); err != nil {
		return nil, fmt.Errorf("failed to save baseline: %w", err)
	}
	
	log.Printf("Security baseline established with score %.1f and %d total issues", 
		baseline.SecurityScore, baseline.IssueCount.Total)
	
	return baseline, nil
}

// RunRegressionTest performs security regression testing
func (s *SecurityRegressionTester) RunRegressionTest(ctx context.Context) (*RegressionResult, error) {
	log.Println("Running security regression test...")
	
	// Load latest baseline
	baseline, err := s.loadLatestBaseline()
	if err != nil {
		return nil, fmt.Errorf("failed to load baseline: %w", err)
	}
	
	// Run current security scan
	currentResults, err := s.securityAuto.RunComprehensiveSecurityScan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run current security scan: %w", err)
	}
	
	// Perform regression analysis
	result := &RegressionResult{
		TestDate:       time.Now(),
		BaselineUsed:   baseline.BaselineDate.Format("2006-01-02"),
		CurrentResults: currentResults,
		Baseline:       baseline,
	}
	
	// Calculate score regression
	result.ScoreRegression = baseline.SecurityScore - currentResults.SecuritySummary.SecurityScore
	
	// Identify new and resolved issues
	result.NewIssues, result.ResolvedIssues = s.compareIssues(baseline, currentResults)
	
	// Calculate regression summary
	result.RegressionSummary = s.calculateRegressionSummary(result)
	
	// Determine if regression occurred
	result.RegressionFound = s.isRegressionDetected(result)
	
	// Generate recommendations
	result.Recommendations = s.generateRegressionRecommendations(result)
	
	// Save regression test result
	if err := s.saveRegressionResult(result); err != nil {
		log.Printf("Failed to save regression result: %v", err)
	}
	
	// Send alerts if regression detected
	if result.RegressionFound {
		if err := s.sendRegressionAlert(result); err != nil {
			log.Printf("Failed to send regression alert: %v", err)
		}
	}
	
	log.Printf("Regression test completed. Regression detected: %v", result.RegressionFound)
	
	return result, nil
}

// compareIssues compares current issues with baseline to find new and resolved issues
func (s *SecurityRegressionTester) compareIssues(baseline *SecurityBaseline, current *ComprehensiveSecurityResult) ([]SecurityIssue, []SecurityIssue) {
	var newIssues, resolvedIssues []SecurityIssue
	
	// Convert current issues to SecurityIssue format
	currentIssueMap := make(map[string]SecurityIssue)
	for _, issue := range current.SecuritySummary.TopIssues {
		secIssue := SecurityIssue{
			ID:          issue.ID,
			Title:       issue.Title,
			Severity:    issue.Severity,
			Category:    issue.Category,
			Source:      issue.Source,
			Hash:        s.calculateIssueHash(issue),
			Description: issue.Description,
			FirstSeen:   time.Now(),
		}
		currentIssueMap[secIssue.Hash] = secIssue
	}
	
	// Convert baseline issues to map
	baselineIssueMap := make(map[string]BaselineIssue)
	for _, issue := range baseline.KnownIssues {
		baselineIssueMap[issue.Hash] = issue
	}
	
	// Find new issues (in current but not in baseline)
	for hash, issue := range currentIssueMap {
		if _, exists := baselineIssueMap[hash]; !exists {
			newIssues = append(newIssues, issue)
		}
	}
	
	// Find resolved issues (in baseline but not in current)
	for hash, baselineIssue := range baselineIssueMap {
		if _, exists := currentIssueMap[hash]; !exists && !baselineIssue.Suppressed {
			resolvedIssue := SecurityIssue{
				ID:          baselineIssue.ID,
				Title:       baselineIssue.Title,
				Severity:    baselineIssue.Severity,
				Category:    baselineIssue.Category,
				Source:      baselineIssue.Source,
				Hash:        baselineIssue.Hash,
				FirstSeen:   baselineIssue.FirstSeen,
			}
			resolvedIssues = append(resolvedIssues, resolvedIssue)
		}
	}
	
	return newIssues, resolvedIssues
}

// calculateRegressionSummary calculates regression summary statistics
func (s *SecurityRegressionTester) calculateRegressionSummary(result *RegressionResult) RegressionSummary {
	summary := RegressionSummary{
		ResolvedIssues: len(result.ResolvedIssues),
		ScoreChange:    result.ScoreRegression,
	}
	
	// Count new issues by severity
	for _, issue := range result.NewIssues {
		switch issue.Severity {
		case "critical":
			summary.NewCriticalIssues++
		case "high":
			summary.NewHighIssues++
		case "medium":
			summary.NewMediumIssues++
		case "low":
			summary.NewLowIssues++
		}
	}
	
	// Determine regression level
	if summary.NewCriticalIssues > 0 {
		summary.RegressionLevel = "critical"
	} else if summary.NewHighIssues > s.config.HighIssueThreshold {
		summary.RegressionLevel = "high"
	} else if result.ScoreRegression > s.config.MaxRegressionPercent {
		summary.RegressionLevel = "medium"
	} else if summary.NewHighIssues > 0 || summary.NewMediumIssues > 10 {
		summary.RegressionLevel = "low"
	} else {
		summary.RegressionLevel = "none"
	}
	
	return summary
}

// isRegressionDetected determines if a security regression occurred
func (s *SecurityRegressionTester) isRegressionDetected(result *RegressionResult) bool {
	// Critical issues always indicate regression
	if result.RegressionSummary.NewCriticalIssues > s.config.CriticalIssueThreshold {
		return true
	}
	
	// High issues above threshold
	if result.RegressionSummary.NewHighIssues > s.config.HighIssueThreshold {
		return true
	}
	
	// Security score regression above threshold
	if result.ScoreRegression > s.config.MaxRegressionPercent {
		return true
	}
	
	return false
}

// generateRegressionRecommendations creates recommendations based on regression results
func (s *SecurityRegressionTester) generateRegressionRecommendations(result *RegressionResult) []string {
	var recommendations []string
	
	if result.RegressionFound {
		recommendations = append(recommendations, "REGRESSION DETECTED: Immediate action required")
		
		if result.RegressionSummary.NewCriticalIssues > 0 {
			recommendations = append(recommendations, 
				fmt.Sprintf("Fix %d new critical security issues immediately", result.RegressionSummary.NewCriticalIssues))
		}
		
		if result.RegressionSummary.NewHighIssues > 0 {
			recommendations = append(recommendations, 
				fmt.Sprintf("Address %d new high-severity security issues", result.RegressionSummary.NewHighIssues))
		}
		
		if result.ScoreRegression > 0 {
			recommendations = append(recommendations, 
				fmt.Sprintf("Security score decreased by %.1f points - investigate recent changes", result.ScoreRegression))
		}
	} else {
		recommendations = append(recommendations, "No security regression detected")
		
		if len(result.ResolvedIssues) > 0 {
			recommendations = append(recommendations, 
				fmt.Sprintf("Good progress: %d security issues resolved", len(result.ResolvedIssues)))
		}
	}
	
	// General recommendations
	recommendations = append(recommendations, "Continue regular security scanning")
	recommendations = append(recommendations, "Review and update security baseline periodically")
	
	if s.config.AutoUpdateBaseline && !result.RegressionFound && result.ScoreRegression < 0 {
		recommendations = append(recommendations, "Consider updating security baseline with improved results")
	}
	
	return recommendations
}

// Helper methods
func (s *SecurityRegressionTester) convertToBaselineIssues(topIssues []TopSecurityIssue) []BaselineIssue {
	var baselineIssues []BaselineIssue
	
	for _, issue := range topIssues {
		baselineIssue := BaselineIssue{
			ID:        issue.ID,
			Title:     issue.Title,
			Severity:  issue.Severity,
			Category:  issue.Category,
			Source:    issue.Source,
			Hash:      s.calculateIssueHash(issue),
			FirstSeen: time.Now(),
			Suppressed: false,
		}
		baselineIssues = append(baselineIssues, baselineIssue)
	}
	
	return baselineIssues
}

func (s *SecurityRegressionTester) calculateIssueHash(issue TopSecurityIssue) string {
	// Create a hash based on issue characteristics
	// This is a simplified implementation - in practice, you'd use a proper hash function
	hashInput := fmt.Sprintf("%s|%s|%s|%s", issue.Title, issue.Severity, issue.Category, issue.Source)
	return fmt.Sprintf("%x", []byte(hashInput))[:16]
}

func (s *SecurityRegressionTester) getCurrentGitCommit() string {
	// This is a placeholder - implement actual git commit retrieval
	return "unknown"
}

func (s *SecurityRegressionTester) getScannerVersions() map[string]string {
	// This is a placeholder - implement actual scanner version retrieval
	return map[string]string{
		"zap":          "2.14.0",
		"snyk":         "1.1200.0",
		"govulncheck":  "latest",
		"nancy":        "1.0.42",
	}
}

func (s *SecurityRegressionTester) saveBaseline(baseline *SecurityBaseline) error {
	filename := fmt.Sprintf("baseline-%s.json", baseline.BaselineDate.Format("20060102-150405"))
	filepath := filepath.Join(s.baselineDir, filename)
	
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filepath, data, 0644)
}

func (s *SecurityRegressionTester) loadLatestBaseline() (*SecurityBaseline, error) {
	files, err := filepath.Glob(filepath.Join(s.baselineDir, "baseline-*.json"))
	if err != nil {
		return nil, err
	}
	
	if len(files) == 0 {
		return nil, fmt.Errorf("no baseline found - run EstablishBaseline first")
	}
	
	// Get the latest baseline file (assuming filename sorting works)
	latestFile := files[len(files)-1]
	
	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}
	
	var baseline SecurityBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, err
	}
	
	return &baseline, nil
}

func (s *SecurityRegressionTester) saveRegressionResult(result *RegressionResult) error {
	filename := fmt.Sprintf("regression-test-%s.json", result.TestDate.Format("20060102-150405"))
	filepath := filepath.Join(s.baselineDir, filename)
	
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filepath, data, 0644)
}

func (s *SecurityRegressionTester) sendRegressionAlert(result *RegressionResult) error {
	if s.config.RegressionAlertWebhook == "" {
		return nil
	}
	
	// This is a placeholder for regression alert implementation
	log.Printf("Regression alert would be sent to: %s", s.config.RegressionAlertWebhook)
	return nil
}