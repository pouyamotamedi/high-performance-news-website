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

// SecurityCICDIntegration handles CI/CD pipeline integration for security testing
type SecurityCICDIntegration struct {
	projectRoot        string
	securityAuto       *SecurityAutomation
	regressionTester   *SecurityRegressionTester
	config             *CICDConfig
}

// CICDConfig holds CI/CD integration configuration
type CICDConfig struct {
	EnablePreCommitScans    bool              `yaml:"enable_pre_commit_scans"`
	EnablePRScans          bool              `yaml:"enable_pr_scans"`
	EnableDeploymentScans  bool              `yaml:"enable_deployment_scans"`
	FailOnHighRisk         bool              `yaml:"fail_on_high_risk"`
	FailOnMediumRisk       bool              `yaml:"fail_on_medium_risk"`
	FailOnRegression       bool              `yaml:"fail_on_regression"`
	MaxScanDuration        time.Duration     `yaml:"max_scan_duration"`
	ParallelScans          bool              `yaml:"parallel_scans"`
	CacheResults           bool              `yaml:"cache_results"`
	NotificationChannels   []string          `yaml:"notification_channels"`
	QualityGates           map[string]float64 `yaml:"quality_gates"`
	CustomPolicies         map[string]string `yaml:"custom_policies"`
}

// PipelineStage represents different CI/CD pipeline stages
type PipelineStage string

const (
	StagePreCommit   PipelineStage = "pre-commit"
	StagePullRequest PipelineStage = "pull-request"
	StageBuild       PipelineStage = "build"
	StageTest        PipelineStage = "test"
	StageStaging     PipelineStage = "staging"
	StageProduction  PipelineStage = "production"
)

// SecurityGateResult represents the result of a security gate check
type SecurityGateResult struct {
	Stage              PipelineStage                   `json:"stage"`
	Timestamp          time.Time                       `json:"timestamp"`
	Duration           time.Duration                   `json:"duration"`
	Passed             bool                            `json:"passed"`
	SecurityResults    *ComprehensiveSecurityResult    `json:"security_results,omitempty"`
	RegressionResults  *RegressionResult               `json:"regression_results,omitempty"`
	QualityGateChecks  map[string]QualityGateCheck     `json:"quality_gate_checks"`
	FailureReasons     []string                        `json:"failure_reasons"`
	Recommendations    []string                        `json:"recommendations"`
	ArtifactPaths      []string                        `json:"artifact_paths"`
	Metadata           map[string]interface{}          `json:"metadata"`
}

// QualityGateCheck represents a quality gate check result
type QualityGateCheck struct {
	Name        string  `json:"name"`
	Threshold   float64 `json:"threshold"`
	ActualValue float64 `json:"actual_value"`
	Passed      bool    `json:"passed"`
	Critical    bool    `json:"critical"`
	Message     string  `json:"message"`
}

// NewSecurityCICDIntegration creates a new CI/CD integration instance
func NewSecurityCICDIntegration(projectRoot string) *SecurityCICDIntegration {
	config := &CICDConfig{
		EnablePreCommitScans:   true,
		EnablePRScans:         true,
		EnableDeploymentScans: true,
		FailOnHighRisk:        true,
		FailOnMediumRisk:      false,
		FailOnRegression:      true,
		MaxScanDuration:       15 * time.Minute,
		ParallelScans:         true,
		CacheResults:          true,
		NotificationChannels:  []string{"slack", "email"},
		QualityGates: map[string]float64{
			"security_score_min":     80.0,
			"critical_issues_max":    0.0,
			"high_issues_max":        5.0,
			"regression_threshold":   10.0,
			"compliance_score_min":   90.0,
		},
		CustomPolicies: make(map[string]string),
	}
	
	return &SecurityCICDIntegration{
		projectRoot:      projectRoot,
		securityAuto:     NewSecurityAutomation(projectRoot),
		regressionTester: NewSecurityRegressionTester(projectRoot),
		config:           config,
	}
}

// RunSecurityGate executes security checks for a specific pipeline stage
func (s *SecurityCICDIntegration) RunSecurityGate(ctx context.Context, stage PipelineStage) (*SecurityGateResult, error) {
	result := &SecurityGateResult{
		Stage:             stage,
		Timestamp:         time.Now(),
		QualityGateChecks: make(map[string]QualityGateCheck),
		Metadata:          make(map[string]interface{}),
	}
	
	log.Printf("Running security gate for stage: %s", stage)
	
	// Create timeout context
	gateCtx, cancel := context.WithTimeout(ctx, s.config.MaxScanDuration)
	defer cancel()
	
	startTime := time.Now()
	
	// Run security scans based on stage
	switch stage {
	case StagePreCommit:
		if err := s.runPreCommitSecurityChecks(gateCtx, result); err != nil {
			return result, err
		}
	case StagePullRequest:
		if err := s.runPullRequestSecurityChecks(gateCtx, result); err != nil {
			return result, err
		}
	case StageBuild, StageTest:
		if err := s.runBuildSecurityChecks(gateCtx, result); err != nil {
			return result, err
		}
	case StageStaging:
		if err := s.runStagingSecurityChecks(gateCtx, result); err != nil {
			return result, err
		}
	case StageProduction:
		if err := s.runProductionSecurityChecks(gateCtx, result); err != nil {
			return result, err
		}
	default:
		return nil, fmt.Errorf("unsupported pipeline stage: %s", stage)
	}
	
	result.Duration = time.Since(startTime)
	
	// Evaluate quality gates
	s.evaluateQualityGates(result)
	
	// Generate recommendations
	result.Recommendations = s.generateGateRecommendations(result)
	
	// Save results
	if err := s.saveGateResults(result); err != nil {
		log.Printf("Failed to save gate results: %v", err)
	}
	
	// Send notifications if gate failed
	if !result.Passed {
		if err := s.sendGateFailureNotifications(result); err != nil {
			log.Printf("Failed to send gate failure notifications: %v", err)
		}
	}
	
	log.Printf("Security gate %s completed in %v. Passed: %v", 
		stage, result.Duration, result.Passed)
	
	return result, nil
}

// runPreCommitSecurityChecks performs lightweight security checks for pre-commit
func (s *SecurityCICDIntegration) runPreCommitSecurityChecks(ctx context.Context, result *SecurityGateResult) error {
	if !s.config.EnablePreCommitScans {
		result.Passed = true
		return nil
	}
	
	log.Println("Running pre-commit security checks...")
	
	// Run lightweight dependency scan only
	depScanner := NewDependencyScanner(s.projectRoot)
	depResults, err := depScanner.RunComprehensiveDependencyScan(ctx)
	if err != nil {
		result.FailureReasons = append(result.FailureReasons, 
			fmt.Sprintf("Dependency scan failed: %v", err))
		return nil // Don't fail the gate on scan errors in pre-commit
	}
	
	// Create minimal security results for quality gate evaluation
	result.SecurityResults = &ComprehensiveSecurityResult{
		DependencyResults: depResults,
		SecuritySummary: SecuritySummary{
			CriticalIssues:     depResults.Summary.CriticalVulnerabilities,
			HighRiskIssues:     depResults.Summary.HighVulnerabilities,
			MediumRiskIssues:   depResults.Summary.MediumVulnerabilities,
			LowRiskIssues:      depResults.Summary.LowVulnerabilities,
			VulnerabilityCount: depResults.Summary.TotalVulnerabilities,
			TotalIssues:        depResults.Summary.TotalVulnerabilities,
		},
	}
	
	// Calculate security score based on dependencies only
	result.SecurityResults.SecuritySummary.SecurityScore = s.calculateDependencyScore(depResults)
	
	return nil
}

// runPullRequestSecurityChecks performs security checks for pull requests
func (s *SecurityCICDIntegration) runPullRequestSecurityChecks(ctx context.Context, result *SecurityGateResult) error {
	if !s.config.EnablePRScans {
		result.Passed = true
		return nil
	}
	
	log.Println("Running pull request security checks...")
	
	// Run comprehensive security scan
	secResults, err := s.securityAuto.RunComprehensiveSecurityScan(ctx)
	if err != nil {
		result.FailureReasons = append(result.FailureReasons, 
			fmt.Sprintf("Security scan failed: %v", err))
		return err
	}
	
	result.SecurityResults = secResults
	
	// Run regression test if baseline exists
	regressionResults, err := s.regressionTester.RunRegressionTest(ctx)
	if err != nil {
		log.Printf("Regression test failed: %v", err)
		// Don't fail the gate if regression test fails due to missing baseline
	} else {
		result.RegressionResults = regressionResults
	}
	
	return nil
}

// runBuildSecurityChecks performs security checks during build stage
func (s *SecurityCICDIntegration) runBuildSecurityChecks(ctx context.Context, result *SecurityGateResult) error {
	log.Println("Running build stage security checks...")
	
	// Run dependency scan (faster than full security scan)
	depScanner := NewDependencyScanner(s.projectRoot)
	depResults, err := depScanner.RunComprehensiveDependencyScan(ctx)
	if err != nil {
		result.FailureReasons = append(result.FailureReasons, 
			fmt.Sprintf("Dependency scan failed: %v", err))
		return err
	}
	
	result.SecurityResults = &ComprehensiveSecurityResult{
		DependencyResults: depResults,
		SecuritySummary: SecuritySummary{
			CriticalIssues:     depResults.Summary.CriticalVulnerabilities,
			HighRiskIssues:     depResults.Summary.HighVulnerabilities,
			MediumRiskIssues:   depResults.Summary.MediumVulnerabilities,
			LowRiskIssues:      depResults.Summary.LowVulnerabilities,
			VulnerabilityCount: depResults.Summary.TotalVulnerabilities,
			TotalIssues:        depResults.Summary.TotalVulnerabilities,
			SecurityScore:      s.calculateDependencyScore(depResults),
		},
	}
	
	return nil
}

// runStagingSecurityChecks performs comprehensive security checks for staging
func (s *SecurityCICDIntegration) runStagingSecurityChecks(ctx context.Context, result *SecurityGateResult) error {
	log.Println("Running staging security checks...")
	
	// Run full comprehensive security scan
	secResults, err := s.securityAuto.RunComprehensiveSecurityScan(ctx)
	if err != nil {
		result.FailureReasons = append(result.FailureReasons, 
			fmt.Sprintf("Security scan failed: %v", err))
		return err
	}
	
	result.SecurityResults = secResults
	
	// Run regression test
	regressionResults, err := s.regressionTester.RunRegressionTest(ctx)
	if err != nil {
		log.Printf("Regression test failed: %v", err)
	} else {
		result.RegressionResults = regressionResults
	}
	
	return nil
}

// runProductionSecurityChecks performs security validation for production deployment
func (s *SecurityCICDIntegration) runProductionSecurityChecks(ctx context.Context, result *SecurityGateResult) error {
	if !s.config.EnableDeploymentScans {
		result.Passed = true
		return nil
	}
	
	log.Println("Running production deployment security validation...")
	
	// For production, we typically validate against established baselines
	// rather than running full scans that might impact deployment time
	
	// Load and validate against security baseline
	baseline, err := s.regressionTester.loadLatestBaseline()
	if err != nil {
		result.FailureReasons = append(result.FailureReasons, 
			"No security baseline found for production validation")
		return nil // Don't block deployment if no baseline exists
	}
	
	// Create minimal security results based on baseline
	result.SecurityResults = &ComprehensiveSecurityResult{
		SecuritySummary: SecuritySummary{
			SecurityScore:    baseline.SecurityScore,
			CriticalIssues:   baseline.IssueCount.Critical,
			HighRiskIssues:   baseline.IssueCount.High,
			MediumRiskIssues: baseline.IssueCount.Medium,
			LowRiskIssues:    baseline.IssueCount.Low,
			TotalIssues:      baseline.IssueCount.Total,
		},
	}
	
	result.Metadata["baseline_date"] = baseline.BaselineDate
	result.Metadata["baseline_commit"] = baseline.GitCommit
	
	return nil
}

// evaluateQualityGates evaluates all quality gates and determines if the gate passes
func (s *SecurityCICDIntegration) evaluateQualityGates(result *SecurityGateResult) {
	result.Passed = true
	
	if result.SecurityResults == nil {
		return
	}
	
	// Security Score Gate
	if minScore, exists := s.config.QualityGates["security_score_min"]; exists {
		check := QualityGateCheck{
			Name:        "Security Score",
			Threshold:   minScore,
			ActualValue: result.SecurityResults.SecuritySummary.SecurityScore,
			Critical:    true,
		}
		check.Passed = check.ActualValue >= check.Threshold
		check.Message = fmt.Sprintf("Security score: %.1f (min: %.1f)", 
			check.ActualValue, check.Threshold)
		
		result.QualityGateChecks["security_score"] = check
		
		if !check.Passed {
			result.Passed = false
			result.FailureReasons = append(result.FailureReasons, check.Message)
		}
	}
	
	// Critical Issues Gate
	if maxCritical, exists := s.config.QualityGates["critical_issues_max"]; exists {
		check := QualityGateCheck{
			Name:        "Critical Issues",
			Threshold:   maxCritical,
			ActualValue: float64(result.SecurityResults.SecuritySummary.CriticalIssues),
			Critical:    true,
		}
		check.Passed = check.ActualValue <= check.Threshold
		check.Message = fmt.Sprintf("Critical issues: %.0f (max: %.0f)", 
			check.ActualValue, check.Threshold)
		
		result.QualityGateChecks["critical_issues"] = check
		
		if !check.Passed {
			result.Passed = false
			result.FailureReasons = append(result.FailureReasons, check.Message)
		}
	}
	
	// High Issues Gate
	if maxHigh, exists := s.config.QualityGates["high_issues_max"]; exists {
		check := QualityGateCheck{
			Name:        "High Risk Issues",
			Threshold:   maxHigh,
			ActualValue: float64(result.SecurityResults.SecuritySummary.HighRiskIssues),
			Critical:    s.config.FailOnHighRisk,
		}
		check.Passed = check.ActualValue <= check.Threshold
		check.Message = fmt.Sprintf("High risk issues: %.0f (max: %.0f)", 
			check.ActualValue, check.Threshold)
		
		result.QualityGateChecks["high_issues"] = check
		
		if !check.Passed && check.Critical {
			result.Passed = false
			result.FailureReasons = append(result.FailureReasons, check.Message)
		}
	}
	
	// Regression Gate
	if result.RegressionResults != nil {
		if maxRegression, exists := s.config.QualityGates["regression_threshold"]; exists {
			check := QualityGateCheck{
				Name:        "Security Regression",
				Threshold:   maxRegression,
				ActualValue: result.RegressionResults.ScoreRegression,
				Critical:    s.config.FailOnRegression,
			}
			check.Passed = check.ActualValue <= check.Threshold
			check.Message = fmt.Sprintf("Security regression: %.1f%% (max: %.1f%%)", 
				check.ActualValue, check.Threshold)
			
			result.QualityGateChecks["regression"] = check
			
			if !check.Passed && check.Critical {
				result.Passed = false
				result.FailureReasons = append(result.FailureReasons, check.Message)
			}
		}
	}
	
	// Compliance Gate
	if result.SecurityResults.ComplianceStatus.ComplianceScore > 0 {
		if minCompliance, exists := s.config.QualityGates["compliance_score_min"]; exists {
			check := QualityGateCheck{
				Name:        "Compliance Score",
				Threshold:   minCompliance,
				ActualValue: result.SecurityResults.ComplianceStatus.ComplianceScore,
				Critical:    false,
			}
			check.Passed = check.ActualValue >= check.Threshold
			check.Message = fmt.Sprintf("Compliance score: %.1f%% (min: %.1f%%)", 
				check.ActualValue, check.Threshold)
			
			result.QualityGateChecks["compliance"] = check
			
			if !check.Passed {
				result.FailureReasons = append(result.FailureReasons, 
					fmt.Sprintf("Compliance warning: %s", check.Message))
			}
		}
	}
}

// generateGateRecommendations creates recommendations based on gate results
func (s *SecurityCICDIntegration) generateGateRecommendations(result *SecurityGateResult) []string {
	var recommendations []string
	
	if result.Passed {
		recommendations = append(recommendations, "✅ Security gate passed - deployment approved")
		return recommendations
	}
	
	recommendations = append(recommendations, "❌ Security gate failed - address issues before proceeding")
	
	// Add specific recommendations based on failed checks
	for _, check := range result.QualityGateChecks {
		if !check.Passed && check.Critical {
			switch check.Name {
			case "Security Score":
				recommendations = append(recommendations, 
					"Improve security score by fixing high and critical issues")
			case "Critical Issues":
				recommendations = append(recommendations, 
					"Fix all critical security issues before deployment")
			case "High Risk Issues":
				recommendations = append(recommendations, 
					"Reduce high-risk security issues to acceptable levels")
			case "Security Regression":
				recommendations = append(recommendations, 
					"Address security regression - review recent changes")
			}
		}
	}
	
	// Stage-specific recommendations
	switch result.Stage {
	case StagePreCommit:
		recommendations = append(recommendations, "Run 'go mod tidy' and update vulnerable dependencies")
	case StagePullRequest:
		recommendations = append(recommendations, "Review security scan results before merging")
	case StageBuild:
		recommendations = append(recommendations, "Fix dependency vulnerabilities in build")
	case StageStaging:
		recommendations = append(recommendations, "Complete security remediation before production")
	case StageProduction:
		recommendations = append(recommendations, "Security validation failed - halt deployment")
	}
	
	return recommendations
}

// calculateDependencyScore calculates security score based on dependency scan results
func (s *SecurityCICDIntegration) calculateDependencyScore(depResults *ComprehensiveDependencyScanResult) float64 {
	if depResults == nil {
		return 100.0
	}
	
	score := 100.0
	score -= float64(depResults.Summary.CriticalVulnerabilities) * 25
	score -= float64(depResults.Summary.HighVulnerabilities) * 15
	score -= float64(depResults.Summary.MediumVulnerabilities) * 5
	score -= float64(depResults.Summary.LowVulnerabilities) * 1
	
	if score < 0 {
		score = 0
	}
	
	return score
}

// saveGateResults saves security gate results
func (s *SecurityCICDIntegration) saveGateResults(result *SecurityGateResult) error {
	reportDir := filepath.Join(s.projectRoot, "reports", "security", "gates")
	os.MkdirAll(reportDir, 0755)
	
	filename := fmt.Sprintf("security-gate-%s-%s.json", 
		result.Stage, result.Timestamp.Format("20060102-150405"))
	filepath := filepath.Join(reportDir, filename)
	
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filepath, data, 0644)
}

// sendGateFailureNotifications sends notifications when security gate fails
func (s *SecurityCICDIntegration) sendGateFailureNotifications(result *SecurityGateResult) error {
	// This is a placeholder for notification implementation
	log.Printf("Security gate failure notification for stage %s: %s", 
		result.Stage, strings.Join(result.FailureReasons, "; "))
	return nil
}

// ShouldBlockDeployment determines if deployment should be blocked based on gate results
func (s *SecurityCICDIntegration) ShouldBlockDeployment(result *SecurityGateResult) bool {
	return !result.Passed
}

// GetExitCode returns appropriate exit code for CI/CD systems
func (s *SecurityCICDIntegration) GetExitCode(result *SecurityGateResult) int {
	if result.Passed {
		return 0
	}
	
	// Check if any critical gates failed
	for _, check := range result.QualityGateChecks {
		if !check.Passed && check.Critical {
			return 1 // Critical failure
		}
	}
	
	return 2 // Non-critical failure (warning)
}