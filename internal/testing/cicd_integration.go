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

// CICDIntegration manages comprehensive CI/CD pipeline integration
type CICDIntegration struct {
	config           *CICDConfig
	testRunner       *TestRunner
	aiValidator      *AIValidator
	securityScanner  *SecurityScanner
	performanceTester *PerformanceTester
	qualityGates     *QualityGates
	reporter         *TestReporter
}

// CICDConfig holds configuration for CI/CD integration
type CICDConfig struct {
	PreCommitEnabled     bool                   `json:"pre_commit_enabled"`
	PullRequestEnabled   bool                   `json:"pull_request_enabled"`
	DeploymentEnabled    bool                   `json:"deployment_enabled"`
	QualityGates         QualityGateConfig      `json:"quality_gates"`
	NotificationChannels []NotificationChannel  `json:"notification_channels"`
	Environments         []Environment          `json:"environments"`
	TestStages           []TestStage            `json:"test_stages"`
}

// QualityGateConfig defines quality gate thresholds
type QualityGateConfig struct {
	MinCodeCoverage      float64 `json:"min_code_coverage"`
	MaxSecurityIssues    int     `json:"max_security_issues"`
	MaxPerformanceRegression float64 `json:"max_performance_regression"`
	MinMutationScore     float64 `json:"min_mutation_score"`
	MaxTestFailures      int     `json:"max_test_failures"`
}

// TestStage represents a stage in the testing pipeline
type TestStage struct {
	Name         string        `json:"name"`
	Type         StageType     `json:"type"`
	Parallel     bool          `json:"parallel"`
	Timeout      time.Duration `json:"timeout"`
	Required     bool          `json:"required"`
	Dependencies []string      `json:"dependencies"`
	Commands     []string      `json:"commands"`
}

// StageType defines the type of test stage
type StageType string

const (
	StageTypeUnit        StageType = "unit"
	StageTypeIntegration StageType = "integration"
	StageTypeE2E         StageType = "e2e"
	StageTypeSecurity    StageType = "security"
	StageTypePerformance StageType = "performance"
	StageTypeAI          StageType = "ai_validation"
	StageTypeMutation    StageType = "mutation"
)

// Environment represents a deployment environment
type Environment struct {
	Name        string            `json:"name"`
	Type        EnvironmentType   `json:"type"`
	URL         string            `json:"url"`
	Config      map[string]string `json:"config"`
	HealthCheck string            `json:"health_check"`
}

// EnvironmentType defines environment types
type EnvironmentType string

const (
	EnvTypeDev     EnvironmentType = "development"
	EnvTypeStaging EnvironmentType = "staging"
	EnvTypeProd    EnvironmentType = "production"
)

// PipelineResult represents the result of a CI/CD pipeline execution
type PipelineResult struct {
	ID           string                    `json:"id"`
	Trigger      PipelineTrigger          `json:"trigger"`
	StartTime    time.Time                `json:"start_time"`
	EndTime      time.Time                `json:"end_time"`
	Duration     time.Duration            `json:"duration"`
	Status       PipelineStatus           `json:"status"`
	Stages       []StageResult            `json:"stages"`
	QualityGates QualityGateResult        `json:"quality_gates"`
	Artifacts    []Artifact               `json:"artifacts"`
	Notifications []NotificationResult     `json:"notifications"`
}

// PipelineTrigger defines what triggered the pipeline
type PipelineTrigger struct {
	Type      TriggerType `json:"type"`
	Source    string      `json:"source"`
	Branch    string      `json:"branch"`
	Commit    string      `json:"commit"`
	Author    string      `json:"author"`
	Message   string      `json:"message"`
	ChangedFiles []string `json:"changed_files"`
}

// TriggerType defines pipeline trigger types
type TriggerType string

const (
	TriggerPreCommit    TriggerType = "pre_commit"
	TriggerPullRequest  TriggerType = "pull_request"
	TriggerDeployment   TriggerType = "deployment"
	TriggerScheduled    TriggerType = "scheduled"
	TriggerManual       TriggerType = "manual"
)

// NewCICDIntegration creates a new CI/CD integration instance
func NewCICDIntegration(config *CICDConfig) *CICDIntegration {
	return &CICDIntegration{
		config:           config,
		testRunner:       NewTestRunner(),
		aiValidator:      NewAIValidator(),
		securityScanner:  NewSecurityScanner(),
		performanceTester: NewPerformanceTester(),
		qualityGates:     NewQualityGates(config.QualityGates),
		reporter:         NewTestReporter(),
	}
}

// ExecutePreCommitPipeline runs comprehensive pre-commit validation
func (c *CICDIntegration) ExecutePreCommitPipeline(ctx context.Context, changes []string) (*PipelineResult, error) {
	log.Printf("Starting pre-commit pipeline for %d changed files", len(changes))
	
	result := &PipelineResult{
		ID:        generatePipelineID(),
		Trigger:   c.buildTrigger(TriggerPreCommit, changes),
		StartTime: time.Now(),
		Status:    PipelineStatusRunning,
	}
	
	// Stage 1: Fast validation (parallel)
	fastStages := []TestStage{
		{Name: "lint", Type: StageTypeUnit, Parallel: true, Timeout: 2 * time.Minute, Required: true},
		{Name: "format_check", Type: StageTypeUnit, Parallel: true, Timeout: 1 * time.Minute, Required: true},
		{Name: "ai_validation", Type: StageTypeAI, Parallel: true, Timeout: 3 * time.Minute, Required: true},
	}
	
	for _, stage := range fastStages {
		stageResult := c.executeStage(ctx, stage, changes)
		result.Stages = append(result.Stages, stageResult)
		
		if stageResult.Status == StageStatusFailed && stage.Required {
			result.Status = PipelineStatusFailed
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, fmt.Errorf("required stage %s failed", stage.Name)
		}
	}
	
	// Stage 2: Unit tests for affected code
	affectedTests := c.findAffectedTests(changes)
	unitStage := TestStage{
		Name:     "unit_tests",
		Type:     StageTypeUnit,
		Timeout:  10 * time.Minute,
		Required: true,
	}
	
	unitResult := c.executeUnitTests(ctx, unitStage, affectedTests)
	result.Stages = append(result.Stages, unitResult)
	
	if unitResult.Status == StageStatusFailed {
		result.Status = PipelineStatusFailed
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, fmt.Errorf("unit tests failed")
	}
	
	// Stage 3: Security scan for changed files
	securityStage := TestStage{
		Name:     "security_scan",
		Type:     StageTypeSecurity,
		Timeout:  5 * time.Minute,
		Required: true,
	}
	
	securityResult := c.executeSecurityScan(ctx, securityStage, changes)
	result.Stages = append(result.Stages, securityResult)
	
	// Evaluate quality gates
	result.QualityGates = c.qualityGates.EvaluatePreCommit(result)
	
	if result.QualityGates.Passed {
		result.Status = PipelineStatusPassed
	} else {
		result.Status = PipelineStatusFailed
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	// Send notifications
	c.sendNotifications(result)
	
	log.Printf("Pre-commit pipeline completed in %v with status: %s", result.Duration, result.Status)
	return result, nil
}

// ExecutePullRequestPipeline runs comprehensive validation for pull requests
func (c *CICDIntegration) ExecutePullRequestPipeline(ctx context.Context, prInfo *PullRequestInfo) (*PipelineResult, error) {
	log.Printf("Starting pull request pipeline for PR #%d", prInfo.Number)
	
	result := &PipelineResult{
		ID:        generatePipelineID(),
		Trigger:   c.buildPRTrigger(prInfo),
		StartTime: time.Now(),
		Status:    PipelineStatusRunning,
	}
	
	// Stage 1: Full test suite (staged execution)
	stages := []TestStage{
		{Name: "unit_tests", Type: StageTypeUnit, Timeout: 15 * time.Minute, Required: true},
		{Name: "integration_tests", Type: StageTypeIntegration, Timeout: 20 * time.Minute, Required: true, Dependencies: []string{"unit_tests"}},
		{Name: "security_full_scan", Type: StageTypeSecurity, Timeout: 10 * time.Minute, Required: true, Parallel: true},
		{Name: "performance_regression", Type: StageTypePerformance, Timeout: 15 * time.Minute, Required: false, Dependencies: []string{"integration_tests"}},
		{Name: "mutation_testing", Type: StageTypeMutation, Timeout: 25 * time.Minute, Required: false, Parallel: true},
	}
	
	// Execute stages with dependency management
	for _, stage := range stages {
		if c.shouldExecuteStage(stage, result.Stages) {
			stageResult := c.executeStage(ctx, stage, prInfo.ChangedFiles)
			result.Stages = append(result.Stages, stageResult)
			
			if stageResult.Status == StageStatusFailed && stage.Required {
				result.Status = PipelineStatusFailed
				break
			}
		}
	}
	
	// Evaluate quality gates
	result.QualityGates = c.qualityGates.EvaluatePullRequest(result)
	
	if result.Status != PipelineStatusFailed {
		if result.QualityGates.Passed {
			result.Status = PipelineStatusPassed
		} else {
			result.Status = PipelineStatusFailed
		}
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	// Generate comprehensive report
	c.generatePRReport(result, prInfo)
	
	// Send notifications
	c.sendNotifications(result)
	
	log.Printf("Pull request pipeline completed in %v with status: %s", result.Duration, result.Status)
	return result, nil
}

// ExecuteDeploymentPipeline runs deployment validation and smoke tests
func (c *CICDIntegration) ExecuteDeploymentPipeline(ctx context.Context, env *Environment, deployInfo *DeploymentInfo) (*PipelineResult, error) {
	log.Printf("Starting deployment pipeline for environment: %s", env.Name)
	
	result := &PipelineResult{
		ID:        generatePipelineID(),
		Trigger:   c.buildDeploymentTrigger(deployInfo),
		StartTime: time.Now(),
		Status:    PipelineStatusRunning,
	}
	
	// Pre-deployment validation
	preDeployStages := []TestStage{
		{Name: "smoke_tests", Type: StageTypeE2E, Timeout: 5 * time.Minute, Required: true},
		{Name: "health_check", Type: StageTypeE2E, Timeout: 2 * time.Minute, Required: true},
		{Name: "security_validation", Type: StageTypeSecurity, Timeout: 3 * time.Minute, Required: true},
	}
	
	for _, stage := range preDeployStages {
		stageResult := c.executeDeploymentStage(ctx, stage, env)
		result.Stages = append(result.Stages, stageResult)
		
		if stageResult.Status == StageStatusFailed && stage.Required {
			result.Status = PipelineStatusFailed
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, fmt.Errorf("deployment validation failed at stage: %s", stage.Name)
		}
	}
	
	// Post-deployment validation (if deployment succeeded)
	if deployInfo.DeploymentStatus == "success" {
		postDeployStages := []TestStage{
			{Name: "post_deploy_health", Type: StageTypeE2E, Timeout: 3 * time.Minute, Required: true},
			{Name: "performance_baseline", Type: StageTypePerformance, Timeout: 10 * time.Minute, Required: false},
		}
		
		for _, stage := range postDeployStages {
			stageResult := c.executeDeploymentStage(ctx, stage, env)
			result.Stages = append(result.Stages, stageResult)
		}
	}
	
	// Evaluate deployment quality gates
	result.QualityGates = c.qualityGates.EvaluateDeployment(result, env)
	
	if result.QualityGates.Passed {
		result.Status = PipelineStatusPassed
	} else {
		result.Status = PipelineStatusFailed
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	// Send notifications
	c.sendNotifications(result)
	
	log.Printf("Deployment pipeline completed in %v with status: %s", result.Duration, result.Status)
	return result, nil
}

// executeStage executes a single test stage
func (c *CICDIntegration) executeStage(ctx context.Context, stage TestStage, files []string) StageResult {
	log.Printf("Executing stage: %s", stage.Name)
	
	stageCtx, cancel := context.WithTimeout(ctx, stage.Timeout)
	defer cancel()
	
	result := StageResult{
		Name:      stage.Name,
		Type:      stage.Type,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	switch stage.Type {
	case StageTypeUnit:
		result = c.executeUnitStage(stageCtx, stage, files)
	case StageTypeIntegration:
		result = c.executeIntegrationStage(stageCtx, stage)
	case StageTypeSecurity:
		result = c.executeSecurityStage(stageCtx, stage, files)
	case StageTypePerformance:
		result = c.executePerformanceStage(stageCtx, stage)
	case StageTypeAI:
		result = c.executeAIStage(stageCtx, stage, files)
	case StageTypeMutation:
		result = c.executeMutationStage(stageCtx, stage)
	default:
		result.Status = StageStatusFailed
		result.Error = fmt.Sprintf("unknown stage type: %s", stage.Type)
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	log.Printf("Stage %s completed in %v with status: %s", stage.Name, result.Duration, result.Status)
	return result
}

// executeUnitStage executes unit testing stage
func (c *CICDIntegration) executeUnitStage(ctx context.Context, stage TestStage, files []string) StageResult {
	result := StageResult{
		Name:      stage.Name,
		Type:      stage.Type,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	switch stage.Name {
	case "lint":
		output, err := c.runCommand(ctx, "golangci-lint", "run", "--timeout=2m")
		if err != nil {
			result.Status = StageStatusFailed
			result.Error = err.Error()
			result.Output = output
		} else {
			result.Status = StageStatusPassed
			result.Output = "Linting passed"
		}
		
	case "format_check":
		output, err := c.runCommand(ctx, "gofmt", "-l", ".")
		if err != nil || strings.TrimSpace(output) != "" {
			result.Status = StageStatusFailed
			result.Error = "Code formatting issues found"
			result.Output = output
		} else {
			result.Status = StageStatusPassed
			result.Output = "Code formatting is correct"
		}
		
	case "unit_tests":
		output, err := c.runCommand(ctx, "go", "test", "-v", "-race", "-coverprofile=coverage.out", "./...")
		if err != nil {
			result.Status = StageStatusFailed
			result.Error = err.Error()
		} else {
			result.Status = StageStatusPassed
			// Extract coverage information
			coverage := c.extractCoverage(output)
			result.Metrics = map[string]interface{}{
				"coverage": coverage,
			}
		}
		result.Output = output
	}
	
	return result
}

// Helper functions and additional methods would continue here...
// This is the foundation for the CI/CD integration system

// executeUnitTests executes unit tests for specific test files
func (c *CICDIntegration) executeUnitTests(ctx context.Context, stage TestStage, testFiles []string) StageResult {
	result := StageResult{
		Name:      stage.Name,
		Type:      stage.Type,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	if len(testFiles) == 0 {
		result.Status = StageStatusSkipped
		result.Output = "No affected tests found"
		return result
	}
	
	// Run unit tests with coverage
	args := []string{"test", "-v", "-race", "-coverprofile=test-results/coverage.out", "-covermode=atomic"}
	args = append(args, testFiles...)
	
	output, err := c.runCommand(ctx, "go", args...)
	if err != nil {
		result.Status = StageStatusFailed
		result.Error = err.Error()
	} else {
		result.Status = StageStatusPassed
		// Extract test metrics
		metrics := c.extractTestMetrics(output)
		result.Metrics = metrics
	}
	result.Output = output
	
	return result
}

// executeSecurityScan executes security scanning for changed files
func (c *CICDIntegration) executeSecurityScan(ctx context.Context, stage TestStage, files []string) StageResult {
	result := StageResult{
		Name:      stage.Name,
		Type:      stage.Type,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	// Run gosec security scanner
	output, err := c.runCommand(ctx, "gosec", "-fmt=json", "-out=security-report.json", "./...")
	if err != nil {
		result.Status = StageStatusFailed
		result.Error = err.Error()
	} else {
		// Parse security report and check for critical issues
		issues := c.parseSecurityReport("security-report.json")
		criticalIssues := c.filterCriticalSecurityIssues(issues)
		
		if len(criticalIssues) > c.config.QualityGates.MaxSecurityIssues {
			result.Status = StageStatusFailed
			result.Error = fmt.Sprintf("Found %d critical security issues (max allowed: %d)", 
				len(criticalIssues), c.config.QualityGates.MaxSecurityIssues)
		} else {
			result.Status = StageStatusPassed
		}
		
		result.Metrics = map[string]interface{}{
			"total_issues":    len(issues),
			"critical_issues": len(criticalIssues),
		}
	}
	result.Output = output
	
	return result
}

// executeIntegrationStage executes integration tests
func (c *CICDIntegration) executeIntegrationStage(ctx context.Context, stage TestStage) StageResult {
	result := StageResult{
		Name:      stage.Name,
		Type:      stage.Type,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	// Run integration tests
	output, err := c.runCommand(ctx, "go", "test", "-v", "-tags=integration", "-timeout=20m", "./internal/integration/...")
	if err != nil {
		result.Status = StageStatusFailed
		result.Error = err.Error()
	} else {
		result.Status = StageStatusPassed
		metrics := c.extractTestMetrics(output)
		result.Metrics = metrics
	}
	result.Output = output
	
	return result
}

// executeSecurityStage executes comprehensive security testing
func (c *CICDIntegration) executeSecurityStage(ctx context.Context, stage TestStage, files []string) StageResult {
	result := StageResult{
		Name:      stage.Name,
		Type:      stage.Type,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	// Run comprehensive security scan
	securityResult := c.securityScanner.RunComprehensiveScan(ctx, files)
	
	if securityResult.HasCriticalIssues() {
		result.Status = StageStatusFailed
		result.Error = "Critical security vulnerabilities found"
	} else {
		result.Status = StageStatusPassed
	}
	
	result.Metrics = map[string]interface{}{
		"vulnerabilities": securityResult.VulnerabilityCount,
		"severity_high":   securityResult.HighSeverityCount,
		"severity_medium": securityResult.MediumSeverityCount,
	}
	result.Output = securityResult.Summary
	
	return result
}

// executePerformanceStage executes performance regression tests
func (c *CICDIntegration) executePerformanceStage(ctx context.Context, stage TestStage) StageResult {
	result := StageResult{
		Name:      stage.Name,
		Type:      stage.Type,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	// Run performance regression tests
	perfResult := c.performanceTester.RunRegressionTests(ctx)
	
	if perfResult.HasRegression(c.config.QualityGates.MaxPerformanceRegression) {
		result.Status = StageStatusFailed
		result.Error = fmt.Sprintf("Performance regression detected: %.2f%%", perfResult.RegressionPercentage)
	} else {
		result.Status = StageStatusPassed
	}
	
	result.Metrics = map[string]interface{}{
		"regression_percentage": perfResult.RegressionPercentage,
		"baseline_performance":  perfResult.BaselineMetrics,
		"current_performance":   perfResult.CurrentMetrics,
	}
	result.Output = perfResult.Summary
	
	return result
}

// executeAIStage executes AI code validation
func (c *CICDIntegration) executeAIStage(ctx context.Context, stage TestStage, files []string) StageResult {
	result := StageResult{
		Name:      stage.Name,
		Type:      stage.Type,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	// Filter Go files
	goFiles := c.filterGoFiles(files)
	if len(goFiles) == 0 {
		result.Status = StageStatusSkipped
		result.Output = "No Go files to validate"
		return result
	}
	
	// Run AI validation
	validationResult := c.aiValidator.ValidateFiles(ctx, goFiles)
	
	if validationResult.HasCriticalIssues() {
		result.Status = StageStatusFailed
		result.Error = "Critical AI code validation issues found"
	} else {
		result.Status = StageStatusPassed
	}
	
	result.Metrics = map[string]interface{}{
		"total_issues":    validationResult.TotalIssues,
		"critical_issues": validationResult.CriticalIssues,
		"files_validated": len(goFiles),
	}
	result.Output = validationResult.Summary
	
	return result
}

// executeMutationStage executes mutation testing
func (c *CICDIntegration) executeMutationStage(ctx context.Context, stage TestStage) StageResult {
	result := StageResult{
		Name:      stage.Name,
		Type:      stage.Type,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	// Run mutation testing on critical packages
	output, err := c.runCommand(ctx, "./bin/mutation-tester", "--min-score", fmt.Sprintf("%.1f", c.config.QualityGates.MinMutationScore))
	if err != nil {
		result.Status = StageStatusFailed
		result.Error = err.Error()
	} else {
		result.Status = StageStatusPassed
	}
	result.Output = output
	
	return result
}

// executeDeploymentStage executes deployment-specific test stages
func (c *CICDIntegration) executeDeploymentStage(ctx context.Context, stage TestStage, env *Environment) StageResult {
	result := StageResult{
		Name:      stage.Name,
		Type:      stage.Type,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	switch stage.Name {
	case "smoke_tests":
		result = c.executeSmokeTests(ctx, env)
	case "health_check":
		result = c.executeHealthCheck(ctx, env)
	case "security_validation":
		result = c.executeSecurityValidation(ctx, env)
	case "post_deploy_health":
		result = c.executePostDeployHealth(ctx, env)
	case "performance_baseline":
		result = c.executePerformanceBaseline(ctx, env)
	default:
		result.Status = StageStatusFailed
		result.Error = fmt.Sprintf("unknown deployment stage: %s", stage.Name)
	}
	
	return result
}

// executeSmokeTests runs smoke tests against the environment
func (c *CICDIntegration) executeSmokeTests(ctx context.Context, env *Environment) StageResult {
	result := StageResult{
		Name:      "smoke_tests",
		Type:      StageTypeE2E,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	// Run basic smoke tests
	tests := []SmokeTest{
		{Name: "health_endpoint", URL: env.URL + "/health"},
		{Name: "api_status", URL: env.URL + "/api/v1/status"},
		{Name: "database_connection", URL: env.URL + "/api/v1/health/db"},
	}
	
	var failedTests []string
	for _, test := range tests {
		if !c.runSmokeTest(ctx, test) {
			failedTests = append(failedTests, test.Name)
		}
	}
	
	if len(failedTests) > 0 {
		result.Status = StageStatusFailed
		result.Error = fmt.Sprintf("Smoke tests failed: %v", failedTests)
	} else {
		result.Status = StageStatusPassed
	}
	
	result.Metrics = map[string]interface{}{
		"total_tests":  len(tests),
		"failed_tests": len(failedTests),
	}
	
	return result
}

// executeHealthCheck performs comprehensive health check
func (c *CICDIntegration) executeHealthCheck(ctx context.Context, env *Environment) StageResult {
	result := StageResult{
		Name:      "health_check",
		Type:      StageTypeE2E,
		StartTime: time.Now(),
		Status:    StageStatusRunning,
	}
	
	// Run health check command if specified
	if env.HealthCheck != "" {
		output, err := c.runCommand(ctx, "sh", "-c", env.HealthCheck)
		if err != nil {
			result.Status = StageStatusFailed
			result.Error = err.Error()
		} else {
			result.Status = StageStatusPassed
		}
		result.Output = output
	} else {
		result.Status = StageStatusSkipped
		result.Output = "No health check configured"
	}
	
	return result
}

// Helper functions

// findAffectedTests finds test files affected by code changes
func (c *CICDIntegration) findAffectedTests(changedFiles []string) []string {
	var testFiles []string
	
	for _, file := range changedFiles {
		if strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go") {
			// Find corresponding test file
			testFile := strings.Replace(file, ".go", "_test.go", 1)
			if c.fileExists(testFile) {
				testFiles = append(testFiles, testFile)
			}
			
			// Also include package-level tests
			dir := filepath.Dir(file)
			packageTests := filepath.Join(dir, "*_test.go")
			testFiles = append(testFiles, packageTests)
		}
	}
	
	return c.deduplicateStrings(testFiles)
}

// filterGoFiles filters for Go source files
func (c *CICDIntegration) filterGoFiles(files []string) []string {
	var goFiles []string
	for _, file := range files {
		if strings.HasSuffix(file, ".go") {
			goFiles = append(goFiles, file)
		}
	}
	return goFiles
}

// runCommand executes a command and returns output
func (c *CICDIntegration) runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// extractCoverage extracts coverage percentage from test output
func (c *CICDIntegration) extractCoverage(output string) float64 {
	// Parse coverage from go test output
	// This is a simplified implementation
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "coverage:") {
			// Extract percentage
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "coverage:" && i+1 < len(parts) {
					coverageStr := strings.TrimSuffix(parts[i+1], "%")
					if coverage, err := parseFloat(coverageStr); err == nil {
						return coverage
					}
				}
			}
		}
	}
	return 0.0
}

// extractTestMetrics extracts test metrics from output
func (c *CICDIntegration) extractTestMetrics(output string) map[string]interface{} {
	metrics := make(map[string]interface{})
	
	lines := strings.Split(output, "\n")
	var passed, failed, skipped int
	
	for _, line := range lines {
		if strings.Contains(line, "PASS") {
			passed++
		} else if strings.Contains(line, "FAIL") {
			failed++
		} else if strings.Contains(line, "SKIP") {
			skipped++
		}
	}
	
	metrics["tests_passed"] = passed
	metrics["tests_failed"] = failed
	metrics["tests_skipped"] = skipped
	metrics["total_tests"] = passed + failed + skipped
	
	return metrics
}

// shouldExecuteStage determines if a stage should be executed based on dependencies
func (c *CICDIntegration) shouldExecuteStage(stage TestStage, completedStages []StageResult) bool {
	if len(stage.Dependencies) == 0 {
		return true
	}
	
	for _, dep := range stage.Dependencies {
		found := false
		for _, completed := range completedStages {
			if completed.Name == dep && completed.Status == StageStatusPassed {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}

// buildTrigger creates a pipeline trigger from changes
func (c *CICDIntegration) buildTrigger(triggerType TriggerType, changes []string) PipelineTrigger {
	return PipelineTrigger{
		Type:         triggerType,
		ChangedFiles: changes,
	}
}

// buildPRTrigger creates a pipeline trigger from PR info
func (c *CICDIntegration) buildPRTrigger(prInfo *PullRequestInfo) PipelineTrigger {
	return PipelineTrigger{
		Type:         TriggerPullRequest,
		Source:       prInfo.Source,
		Branch:       prInfo.Branch,
		Commit:       prInfo.Commit,
		Author:       prInfo.Author,
		ChangedFiles: prInfo.ChangedFiles,
	}
}

// buildDeploymentTrigger creates a pipeline trigger from deployment info
func (c *CICDIntegration) buildDeploymentTrigger(deployInfo *DeploymentInfo) PipelineTrigger {
	return PipelineTrigger{
		Type:   TriggerDeployment,
		Source: deployInfo.Source,
		Branch: deployInfo.Branch,
		Commit: deployInfo.Commit,
	}
}

// Utility functions
func (c *CICDIntegration) fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func (c *CICDIntegration) deduplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

func generatePipelineID() string {
	return fmt.Sprintf("pipeline-%d", time.Now().Unix())
}

func parseFloat(s string) (float64, error) {
	// Simplified float parsing
	return 0.0, nil
}

// Additional types and structures needed for the CI/CD integration

// StageResult represents the result of a test stage execution
type StageResult struct {
	Name      string                 `json:"name"`
	Type      StageType              `json:"type"`
	Status    StageStatus            `json:"status"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
	Output    string                 `json:"output"`
	Error     string                 `json:"error,omitempty"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// StageStatus defines the status of a test stage
type StageStatus string

const (
	StageStatusPending StageStatus = "pending"
	StageStatusRunning StageStatus = "running"
	StageStatusPassed  StageStatus = "passed"
	StageStatusFailed  StageStatus = "failed"
	StageStatusSkipped StageStatus = "skipped"
)

// PipelineStatus defines the overall status of a pipeline
type PipelineStatus string

const (
	PipelineStatusPending PipelineStatus = "pending"
	PipelineStatusRunning PipelineStatus = "running"
	PipelineStatusPassed  PipelineStatus = "passed"
	PipelineStatusFailed  PipelineStatus = "failed"
)

// QualityGateResult represents the result of quality gate evaluation
type QualityGateResult struct {
	Passed      bool                   `json:"passed"`
	Score       float64                `json:"score"`
	Thresholds  map[string]interface{} `json:"thresholds"`
	Violations  []QualityViolation     `json:"violations"`
	Recommendations []string           `json:"recommendations"`
}

// QualityViolation represents a quality gate violation
type QualityViolation struct {
	Rule        string      `json:"rule"`
	Threshold   interface{} `json:"threshold"`
	ActualValue interface{} `json:"actual_value"`
	Severity    string      `json:"severity"`
	Message     string      `json:"message"`
}

// Artifact represents a build or test artifact
type Artifact struct {
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Created  time.Time `json:"created"`
	Checksum string    `json:"checksum"`
}

// NotificationResult represents the result of sending a notification
type NotificationResult struct {
	Channel string    `json:"channel"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
	SentAt  time.Time `json:"sent_at"`
}

// NotificationChannel defines a notification channel
type NotificationChannel struct {
	Type   string            `json:"type"`
	Config map[string]string `json:"config"`
}

// PullRequestInfo contains information about a pull request
type PullRequestInfo struct {
	Number       int      `json:"number"`
	Source       string   `json:"source"`
	Branch       string   `json:"branch"`
	Commit       string   `json:"commit"`
	Author       string   `json:"author"`
	Title        string   `json:"title"`
	ChangedFiles []string `json:"changed_files"`
}

// DeploymentInfo contains information about a deployment
type DeploymentInfo struct {
	Source           string `json:"source"`
	Branch           string `json:"branch"`
	Commit           string `json:"commit"`
	Environment      string `json:"environment"`
	DeploymentStatus string `json:"deployment_status"`
}

// SmokeTest represents a smoke test
type SmokeTest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}