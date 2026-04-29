package testing

import (
	"fmt"
	"log"
	"math"
)

// QualityGates manages quality gate evaluation for CI/CD pipelines
type QualityGates struct {
	config QualityGateConfig
}

// NewQualityGates creates a new quality gates instance
func NewQualityGates(config QualityGateConfig) *QualityGates {
	return &QualityGates{
		config: config,
	}
}

// EvaluatePreCommit evaluates quality gates for pre-commit pipeline
func (q *QualityGates) EvaluatePreCommit(result *PipelineResult) QualityGateResult {
	log.Printf("Evaluating pre-commit quality gates")
	
	gateResult := QualityGateResult{
		Passed:      true,
		Score:       100.0,
		Thresholds:  q.getThresholds(),
		Violations:  []QualityViolation{},
		Recommendations: []string{},
	}
	
	// Check for failed required stages
	for _, stage := range result.Stages {
		if stage.Status == StageStatusFailed {
			violation := QualityViolation{
				Rule:        "required_stage_passed",
				Threshold:   "all stages must pass",
				ActualValue: fmt.Sprintf("stage %s failed", stage.Name),
				Severity:    "critical",
				Message:     fmt.Sprintf("Required stage '%s' failed: %s", stage.Name, stage.Error),
			}
			gateResult.Violations = append(gateResult.Violations, violation)
			gateResult.Passed = false
			gateResult.Score -= 25.0
		}
	}
	
	// Check code coverage if unit tests ran
	for _, stage := range result.Stages {
		if stage.Name == "unit_tests" && stage.Metrics != nil {
			if coverage, ok := stage.Metrics["coverage"].(float64); ok {
				if coverage < q.config.MinCodeCoverage {
					violation := QualityViolation{
						Rule:        "min_code_coverage",
						Threshold:   q.config.MinCodeCoverage,
						ActualValue: coverage,
						Severity:    "high",
						Message:     fmt.Sprintf("Code coverage %.1f%% is below minimum %.1f%%", coverage, q.config.MinCodeCoverage),
					}
					gateResult.Violations = append(gateResult.Violations, violation)
					gateResult.Passed = false
					gateResult.Score -= 20.0
				}
			}
		}
	}
	
	// Check security issues
	for _, stage := range result.Stages {
		if stage.Type == StageTypeSecurity && stage.Metrics != nil {
			if criticalIssues, ok := stage.Metrics["critical_issues"].(int); ok {
				if criticalIssues > q.config.MaxSecurityIssues {
					violation := QualityViolation{
						Rule:        "max_security_issues",
						Threshold:   q.config.MaxSecurityIssues,
						ActualValue: criticalIssues,
						Severity:    "critical",
						Message:     fmt.Sprintf("Found %d critical security issues (max allowed: %d)", criticalIssues, q.config.MaxSecurityIssues),
					}
					gateResult.Violations = append(gateResult.Violations, violation)
					gateResult.Passed = false
					gateResult.Score -= 30.0
				}
			}
		}
	}
	
	// Add recommendations based on violations
	gateResult.Recommendations = q.generateRecommendations(gateResult.Violations, "pre_commit")
	
	// Ensure score doesn't go below 0
	if gateResult.Score < 0 {
		gateResult.Score = 0
	}
	
	log.Printf("Pre-commit quality gates evaluation completed: passed=%v, score=%.1f", gateResult.Passed, gateResult.Score)
	return gateResult
}

// EvaluatePullRequest evaluates quality gates for pull request pipeline
func (q *QualityGates) EvaluatePullRequest(result *PipelineResult) QualityGateResult {
	log.Printf("Evaluating pull request quality gates")
	
	gateResult := QualityGateResult{
		Passed:      true,
		Score:       100.0,
		Thresholds:  q.getThresholds(),
		Violations:  []QualityViolation{},
		Recommendations: []string{},
	}
	
	// Check for failed required stages
	for _, stage := range result.Stages {
		if stage.Status == StageStatusFailed {
			severity := "high"
			scoreDeduction := 15.0
			
			// Critical stages have higher impact
			if stage.Name == "unit_tests" || stage.Name == "integration_tests" || stage.Type == StageTypeSecurity {
				severity = "critical"
				scoreDeduction = 25.0
			}
			
			violation := QualityViolation{
				Rule:        "stage_passed",
				Threshold:   "stage must pass",
				ActualValue: fmt.Sprintf("stage %s failed", stage.Name),
				Severity:    severity,
				Message:     fmt.Sprintf("Stage '%s' failed: %s", stage.Name, stage.Error),
			}
			gateResult.Violations = append(gateResult.Violations, violation)
			
			// Only fail the gate for required stages
			if stage.Required {
				gateResult.Passed = false
			}
			gateResult.Score -= scoreDeduction
		}
	}
	
	// Comprehensive quality checks
	gateResult = q.evaluateCodeCoverage(result, gateResult)
	gateResult = q.evaluateSecurityIssues(result, gateResult)
	gateResult = q.evaluatePerformanceRegression(result, gateResult)
	gateResult = q.evaluateMutationScore(result, gateResult)
	gateResult = q.evaluateTestFailures(result, gateResult)
	
	// Add recommendations
	gateResult.Recommendations = q.generateRecommendations(gateResult.Violations, "pull_request")
	
	// Ensure score doesn't go below 0
	if gateResult.Score < 0 {
		gateResult.Score = 0
	}
	
	log.Printf("Pull request quality gates evaluation completed: passed=%v, score=%.1f", gateResult.Passed, gateResult.Score)
	return gateResult
}

// EvaluateDeployment evaluates quality gates for deployment pipeline
func (q *QualityGates) EvaluateDeployment(result *PipelineResult, env *Environment) QualityGateResult {
	log.Printf("Evaluating deployment quality gates for environment: %s", env.Name)
	
	gateResult := QualityGateResult{
		Passed:      true,
		Score:       100.0,
		Thresholds:  q.getDeploymentThresholds(env),
		Violations:  []QualityViolation{},
		Recommendations: []string{},
	}
	
	// Check smoke tests
	for _, stage := range result.Stages {
		if stage.Name == "smoke_tests" && stage.Status == StageStatusFailed {
			violation := QualityViolation{
				Rule:        "smoke_tests_passed",
				Threshold:   "all smoke tests must pass",
				ActualValue: "smoke tests failed",
				Severity:    "critical",
				Message:     fmt.Sprintf("Smoke tests failed: %s", stage.Error),
			}
			gateResult.Violations = append(gateResult.Violations, violation)
			gateResult.Passed = false
			gateResult.Score -= 40.0
		}
	}
	
	// Check health checks
	for _, stage := range result.Stages {
		if stage.Name == "health_check" && stage.Status == StageStatusFailed {
			violation := QualityViolation{
				Rule:        "health_check_passed",
				Threshold:   "health check must pass",
				ActualValue: "health check failed",
				Severity:    "critical",
				Message:     fmt.Sprintf("Health check failed: %s", stage.Error),
			}
			gateResult.Violations = append(gateResult.Violations, violation)
			gateResult.Passed = false
			gateResult.Score -= 30.0
		}
	}
	
	// Check security validation
	for _, stage := range result.Stages {
		if stage.Name == "security_validation" && stage.Status == StageStatusFailed {
			violation := QualityViolation{
				Rule:        "security_validation_passed",
				Threshold:   "security validation must pass",
				ActualValue: "security validation failed",
				Severity:    "critical",
				Message:     fmt.Sprintf("Security validation failed: %s", stage.Error),
			}
			gateResult.Violations = append(gateResult.Violations, violation)
			gateResult.Passed = false
			gateResult.Score -= 35.0
		}
	}
	
	// Environment-specific checks
	if env.Type == EnvTypeProd {
		// Production has stricter requirements
		gateResult = q.evaluateProductionReadiness(result, gateResult)
	}
	
	// Add recommendations
	gateResult.Recommendations = q.generateRecommendations(gateResult.Violations, "deployment")
	
	// Ensure score doesn't go below 0
	if gateResult.Score < 0 {
		gateResult.Score = 0
	}
	
	log.Printf("Deployment quality gates evaluation completed: passed=%v, score=%.1f", gateResult.Passed, gateResult.Score)
	return gateResult
}

// evaluateCodeCoverage checks code coverage requirements
func (q *QualityGates) evaluateCodeCoverage(result *PipelineResult, gateResult QualityGateResult) QualityGateResult {
	for _, stage := range result.Stages {
		if (stage.Name == "unit_tests" || stage.Type == StageTypeUnit) && stage.Metrics != nil {
			if coverage, ok := stage.Metrics["coverage"].(float64); ok {
				if coverage < q.config.MinCodeCoverage {
					violation := QualityViolation{
						Rule:        "min_code_coverage",
						Threshold:   q.config.MinCodeCoverage,
						ActualValue: coverage,
						Severity:    "high",
						Message:     fmt.Sprintf("Code coverage %.1f%% is below minimum %.1f%%", coverage, q.config.MinCodeCoverage),
					}
					gateResult.Violations = append(gateResult.Violations, violation)
					gateResult.Passed = false
					gateResult.Score -= 20.0
				} else if coverage < q.config.MinCodeCoverage+5.0 {
					// Warning for coverage close to threshold
					gateResult.Recommendations = append(gateResult.Recommendations, 
						fmt.Sprintf("Code coverage %.1f%% is close to minimum threshold. Consider adding more tests.", coverage))
				}
			}
		}
	}
	return gateResult
}

// evaluateSecurityIssues checks security issue thresholds
func (q *QualityGates) evaluateSecurityIssues(result *PipelineResult, gateResult QualityGateResult) QualityGateResult {
	for _, stage := range result.Stages {
		if stage.Type == StageTypeSecurity && stage.Metrics != nil {
			if criticalIssues, ok := stage.Metrics["critical_issues"].(int); ok {
				if criticalIssues > q.config.MaxSecurityIssues {
					violation := QualityViolation{
						Rule:        "max_security_issues",
						Threshold:   q.config.MaxSecurityIssues,
						ActualValue: criticalIssues,
						Severity:    "critical",
						Message:     fmt.Sprintf("Found %d critical security issues (max allowed: %d)", criticalIssues, q.config.MaxSecurityIssues),
					}
					gateResult.Violations = append(gateResult.Violations, violation)
					gateResult.Passed = false
					gateResult.Score -= 30.0
				}
			}
			
			// Check total security issues
			if totalIssues, ok := stage.Metrics["total_issues"].(int); ok {
				maxTotal := q.config.MaxSecurityIssues * 5 // Allow more non-critical issues
				if totalIssues > maxTotal {
					violation := QualityViolation{
						Rule:        "max_total_security_issues",
						Threshold:   maxTotal,
						ActualValue: totalIssues,
						Severity:    "medium",
						Message:     fmt.Sprintf("Found %d total security issues (max recommended: %d)", totalIssues, maxTotal),
					}
					gateResult.Violations = append(gateResult.Violations, violation)
					gateResult.Score -= 10.0
				}
			}
		}
	}
	return gateResult
}

// evaluatePerformanceRegression checks performance regression thresholds
func (q *QualityGates) evaluatePerformanceRegression(result *PipelineResult, gateResult QualityGateResult) QualityGateResult {
	for _, stage := range result.Stages {
		if stage.Type == StageTypePerformance && stage.Metrics != nil {
			if regression, ok := stage.Metrics["regression_percentage"].(float64); ok {
				if regression > q.config.MaxPerformanceRegression {
					violation := QualityViolation{
						Rule:        "max_performance_regression",
						Threshold:   q.config.MaxPerformanceRegression,
						ActualValue: regression,
						Severity:    "high",
						Message:     fmt.Sprintf("Performance regression %.1f%% exceeds threshold %.1f%%", regression, q.config.MaxPerformanceRegression),
					}
					gateResult.Violations = append(gateResult.Violations, violation)
					gateResult.Passed = false
					gateResult.Score -= 25.0
				} else if regression > q.config.MaxPerformanceRegression*0.8 {
					// Warning for regression close to threshold
					gateResult.Recommendations = append(gateResult.Recommendations, 
						fmt.Sprintf("Performance regression %.1f%% is approaching threshold. Monitor closely.", regression))
				}
			}
		}
	}
	return gateResult
}

// evaluateMutationScore checks mutation testing score
func (q *QualityGates) evaluateMutationScore(result *PipelineResult, gateResult QualityGateResult) QualityGateResult {
	for _, stage := range result.Stages {
		if stage.Type == StageTypeMutation && stage.Metrics != nil {
			if score, ok := stage.Metrics["mutation_score"].(float64); ok {
				if score < q.config.MinMutationScore {
					violation := QualityViolation{
						Rule:        "min_mutation_score",
						Threshold:   q.config.MinMutationScore,
						ActualValue: score,
						Severity:    "medium",
						Message:     fmt.Sprintf("Mutation score %.1f%% is below minimum %.1f%%", score, q.config.MinMutationScore),
					}
					gateResult.Violations = append(gateResult.Violations, violation)
					// Don't fail the gate for mutation score, but reduce score
					gateResult.Score -= 15.0
				}
			}
		}
	}
	return gateResult
}

// evaluateTestFailures checks test failure thresholds
func (q *QualityGates) evaluateTestFailures(result *PipelineResult, gateResult QualityGateResult) QualityGateResult {
	totalFailures := 0
	
	for _, stage := range result.Stages {
		if stage.Metrics != nil {
			if failures, ok := stage.Metrics["tests_failed"].(int); ok {
				totalFailures += failures
			}
		}
	}
	
	if totalFailures > q.config.MaxTestFailures {
		violation := QualityViolation{
			Rule:        "max_test_failures",
			Threshold:   q.config.MaxTestFailures,
			ActualValue: totalFailures,
			Severity:    "high",
			Message:     fmt.Sprintf("Total test failures %d exceeds maximum %d", totalFailures, q.config.MaxTestFailures),
		}
		gateResult.Violations = append(gateResult.Violations, violation)
		gateResult.Passed = false
		gateResult.Score -= 20.0
	}
	
	return gateResult
}

// evaluateProductionReadiness checks production-specific requirements
func (q *QualityGates) evaluateProductionReadiness(result *PipelineResult, gateResult QualityGateResult) QualityGateResult {
	// Production requires all stages to pass
	for _, stage := range result.Stages {
		if stage.Status == StageStatusFailed {
			violation := QualityViolation{
				Rule:        "production_all_stages_pass",
				Threshold:   "all stages must pass for production",
				ActualValue: fmt.Sprintf("stage %s failed", stage.Name),
				Severity:    "critical",
				Message:     fmt.Sprintf("Production deployment requires all stages to pass. Stage '%s' failed.", stage.Name),
			}
			gateResult.Violations = append(gateResult.Violations, violation)
			gateResult.Passed = false
			gateResult.Score -= 50.0
		}
	}
	
	return gateResult
}

// generateRecommendations generates recommendations based on violations
func (q *QualityGates) generateRecommendations(violations []QualityViolation, pipelineType string) []string {
	var recommendations []string
	
	for _, violation := range violations {
		switch violation.Rule {
		case "min_code_coverage":
			recommendations = append(recommendations, 
				"Increase test coverage by adding unit tests for uncovered code paths")
		case "max_security_issues":
			recommendations = append(recommendations, 
				"Review and fix security vulnerabilities before proceeding")
		case "max_performance_regression":
			recommendations = append(recommendations, 
				"Investigate performance regression and optimize critical paths")
		case "min_mutation_score":
			recommendations = append(recommendations, 
				"Improve test quality by adding tests that catch more mutations")
		case "smoke_tests_passed":
			recommendations = append(recommendations, 
				"Fix smoke test failures before deploying to ensure basic functionality")
		case "health_check_passed":
			recommendations = append(recommendations, 
				"Resolve health check issues to ensure system stability")
		}
	}
	
	// Add general recommendations based on pipeline type
	switch pipelineType {
	case "pre_commit":
		if len(violations) > 0 {
			recommendations = append(recommendations, 
				"Fix all issues before committing to maintain code quality")
		}
	case "pull_request":
		if len(violations) > 0 {
			recommendations = append(recommendations, 
				"Address quality issues before merging to maintain main branch stability")
		}
	case "deployment":
		if len(violations) > 0 {
			recommendations = append(recommendations, 
				"Resolve all deployment issues before proceeding to ensure system reliability")
		}
	}
	
	return recommendations
}

// getThresholds returns the configured thresholds
func (q *QualityGates) getThresholds() map[string]interface{} {
	return map[string]interface{}{
		"min_code_coverage":         q.config.MinCodeCoverage,
		"max_security_issues":       q.config.MaxSecurityIssues,
		"max_performance_regression": q.config.MaxPerformanceRegression,
		"min_mutation_score":        q.config.MinMutationScore,
		"max_test_failures":         q.config.MaxTestFailures,
	}
}

// getDeploymentThresholds returns environment-specific thresholds
func (q *QualityGates) getDeploymentThresholds(env *Environment) map[string]interface{} {
	thresholds := q.getThresholds()
	
	// Stricter thresholds for production
	if env.Type == EnvTypeProd {
		thresholds["max_security_issues"] = 0 // No security issues allowed in production
		thresholds["max_test_failures"] = 0   // No test failures allowed in production
	}
	
	return thresholds
}

// CalculateQualityScore calculates an overall quality score
func (q *QualityGates) CalculateQualityScore(result *PipelineResult) float64 {
	score := 100.0
	
	// Deduct points for failed stages
	for _, stage := range result.Stages {
		if stage.Status == StageStatusFailed {
			if stage.Required {
				score -= 25.0
			} else {
				score -= 10.0
			}
		}
	}
	
	// Bonus points for good metrics
	for _, stage := range result.Stages {
		if stage.Metrics != nil {
			// Coverage bonus
			if coverage, ok := stage.Metrics["coverage"].(float64); ok {
				if coverage > q.config.MinCodeCoverage+10 {
					score += 5.0
				}
			}
			
			// Mutation score bonus
			if mutationScore, ok := stage.Metrics["mutation_score"].(float64); ok {
				if mutationScore > q.config.MinMutationScore+10 {
					score += 5.0
				}
			}
		}
	}
	
	// Ensure score is within bounds
	return math.Max(0, math.Min(100, score))
}