package testing

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// TestReporter handles comprehensive test result aggregation and reporting
type TestReporter struct {
	config         *ReporterConfig
	storage        *ResultStorage
	trendAnalyzer  *TrendAnalyzer
	dashboardGen   *DashboardGenerator
	notifier       *NotificationManager
}

// ReporterConfig holds configuration for test reporting
type ReporterConfig struct {
	OutputDir           string                 `json:"output_dir"`
	ReportFormats       []ReportFormat         `json:"report_formats"`
	TrendAnalysisPeriod time.Duration          `json:"trend_analysis_period"`
	KPIThresholds       map[string]float64     `json:"kpi_thresholds"`
	NotificationRules   []NotificationRule     `json:"notification_rules"`
	DashboardConfig     DashboardConfig        `json:"dashboard_config"`
}

// ReportFormat defines supported report formats
type ReportFormat string

const (
	FormatHTML     ReportFormat = "html"
	FormatJSON     ReportFormat = "json"
	FormatCSV      ReportFormat = "csv"
	FormatMarkdown ReportFormat = "markdown"
	FormatJUnit    ReportFormat = "junit"
)

// TestReport represents a comprehensive test report
type TestReport struct {
	ID              string                 `json:"id"`
	GeneratedAt     time.Time              `json:"generated_at"`
	Period          ReportPeriod           `json:"period"`
	Summary         TestSummary            `json:"summary"`
	PipelineResults []PipelineResult       `json:"pipeline_results"`
	QualityMetrics  QualityMetrics         `json:"quality_metrics"`
	TrendAnalysis   TrendAnalysis          `json:"trend_analysis"`
	KPIStatus       map[string]KPIStatus   `json:"kpi_status"`
	Recommendations []Recommendation       `json:"recommendations"`
	Artifacts       []ReportArtifact       `json:"artifacts"`
}

// ReportPeriod defines the time period for the report
type ReportPeriod struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Type      string    `json:"type"` // "daily", "weekly", "monthly", "custom"
}

// TestSummary provides high-level test execution summary
type TestSummary struct {
	TotalPipelines      int                    `json:"total_pipelines"`
	SuccessfulPipelines int                    `json:"successful_pipelines"`
	FailedPipelines     int                    `json:"failed_pipelines"`
	SuccessRate         float64                `json:"success_rate"`
	AverageExecutionTime time.Duration         `json:"average_execution_time"`
	TotalTestsRun       int                    `json:"total_tests_run"`
	TestsPassedTotal    int                    `json:"tests_passed_total"`
	TestsFailedTotal    int                    `json:"tests_failed_total"`
	CodeCoverageAvg     float64                `json:"code_coverage_avg"`
	SecurityIssuesFound int                    `json:"security_issues_found"`
	PerformanceRegression float64              `json:"performance_regression"`
	StageBreakdown      map[StageType]StageStats `json:"stage_breakdown"`
}

// StageStats provides statistics for a specific stage type
type StageStats struct {
	TotalExecutions  int           `json:"total_executions"`
	SuccessfulRuns   int           `json:"successful_runs"`
	FailedRuns       int           `json:"failed_runs"`
	SuccessRate      float64       `json:"success_rate"`
	AverageTime      time.Duration `json:"average_time"`
	CommonFailures   []string      `json:"common_failures"`
}

// QualityMetrics tracks quality-related metrics
type QualityMetrics struct {
	CodeCoverage        CoverageMetrics        `json:"code_coverage"`
	TestEffectiveness   TestEffectivenessMetrics `json:"test_effectiveness"`
	SecurityMetrics     SecurityMetrics        `json:"security_metrics"`
	PerformanceMetrics  PerformanceMetrics     `json:"performance_metrics"`
	MutationTestMetrics MutationTestMetrics    `json:"mutation_test_metrics"`
}

// CoverageMetrics tracks code coverage statistics
type CoverageMetrics struct {
	Overall     float64            `json:"overall"`
	ByPackage   map[string]float64 `json:"by_package"`
	Trend       []CoveragePoint    `json:"trend"`
	Threshold   float64            `json:"threshold"`
	Status      MetricStatus       `json:"status"`
}

// CoveragePoint represents a coverage measurement at a point in time
type CoveragePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Coverage  float64   `json:"coverage"`
	Commit    string    `json:"commit"`
}

// TestEffectivenessMetrics tracks how effective tests are
type TestEffectivenessMetrics struct {
	FlakyTestRate       float64          `json:"flaky_test_rate"`
	TestExecutionTime   time.Duration    `json:"test_execution_time"`
	TestMaintenance     MaintenanceStats `json:"test_maintenance"`
	DefectDetectionRate float64          `json:"defect_detection_rate"`
}

// MaintenanceStats tracks test maintenance metrics
type MaintenanceStats struct {
	TestsAdded    int `json:"tests_added"`
	TestsModified int `json:"tests_modified"`
	TestsRemoved  int `json:"tests_removed"`
	TestsFixed    int `json:"tests_fixed"`
}

// SecurityMetrics tracks security-related metrics
type SecurityMetrics struct {
	VulnerabilitiesFound    int                    `json:"vulnerabilities_found"`
	CriticalVulnerabilities int                    `json:"critical_vulnerabilities"`
	VulnerabilityTrend      []VulnerabilityPoint   `json:"vulnerability_trend"`
	SecurityScoreAvg        float64                `json:"security_score_avg"`
	ComplianceStatus        map[string]bool        `json:"compliance_status"`
}

// VulnerabilityPoint represents vulnerability count at a point in time
type VulnerabilityPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	Total           int       `json:"total"`
	Critical        int       `json:"critical"`
	High            int       `json:"high"`
	Medium          int       `json:"medium"`
	Low             int       `json:"low"`
}

// PerformanceMetrics tracks performance-related metrics
type PerformanceMetrics struct {
	RegressionPercentage float64              `json:"regression_percentage"`
	BaselineComparison   BaselineComparison   `json:"baseline_comparison"`
	PerformanceTrend     []PerformancePoint   `json:"performance_trend"`
	Bottlenecks          []PerformanceBottleneck `json:"bottlenecks"`
}

// BaselineComparison compares current performance to baseline
type BaselineComparison struct {
	ResponseTime    ComparisonResult `json:"response_time"`
	Throughput      ComparisonResult `json:"throughput"`
	ResourceUsage   ComparisonResult `json:"resource_usage"`
	ErrorRate       ComparisonResult `json:"error_rate"`
}

// ComparisonResult represents a comparison between current and baseline
type ComparisonResult struct {
	Current    float64 `json:"current"`
	Baseline   float64 `json:"baseline"`
	Change     float64 `json:"change"`
	Status     MetricStatus `json:"status"`
}

// PerformancePoint represents performance measurement at a point in time
type PerformancePoint struct {
	Timestamp    time.Time `json:"timestamp"`
	ResponseTime float64   `json:"response_time"`
	Throughput   float64   `json:"throughput"`
	ErrorRate    float64   `json:"error_rate"`
}

// PerformanceBottleneck identifies performance bottlenecks
type PerformanceBottleneck struct {
	Component   string  `json:"component"`
	Metric      string  `json:"metric"`
	Impact      float64 `json:"impact"`
	Severity    string  `json:"severity"`
	Suggestion  string  `json:"suggestion"`
}

// MutationTestMetrics tracks mutation testing effectiveness
type MutationTestMetrics struct {
	MutationScore     float64            `json:"mutation_score"`
	MutantsKilled     int                `json:"mutants_killed"`
	MutantsGenerated  int                `json:"mutants_generated"`
	WeakTests         []WeakTestInfo     `json:"weak_tests"`
	ScoreTrend        []MutationPoint    `json:"score_trend"`
}

// WeakTestInfo identifies tests that need improvement
type WeakTestInfo struct {
	TestName        string   `json:"test_name"`
	Package         string   `json:"package"`
	MissedMutations int      `json:"missed_mutations"`
	Suggestions     []string `json:"suggestions"`
}

// MutationPoint represents mutation score at a point in time
type MutationPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
	Package   string    `json:"package"`
}

// MetricStatus represents the status of a metric
type MetricStatus string

const (
	StatusGood    MetricStatus = "good"
	StatusWarning MetricStatus = "warning"
	StatusCritical MetricStatus = "critical"
)

// TrendAnalysis provides trend analysis for various metrics
type TrendAnalysis struct {
	QualityTrend      TrendDirection `json:"quality_trend"`
	CoverageTrend     TrendDirection `json:"coverage_trend"`
	PerformanceTrend  TrendDirection `json:"performance_trend"`
	SecurityTrend     TrendDirection `json:"security_trend"`
	Predictions       []TrendPrediction `json:"predictions"`
}

// TrendDirection indicates the direction of a trend
type TrendDirection string

const (
	TrendImproving  TrendDirection = "improving"
	TrendStable     TrendDirection = "stable"
	TrendDeclining  TrendDirection = "declining"
)

// TrendPrediction provides predictions based on trend analysis
type TrendPrediction struct {
	Metric      string    `json:"metric"`
	Prediction  float64   `json:"prediction"`
	Confidence  float64   `json:"confidence"`
	TimeFrame   string    `json:"time_frame"`
	Reasoning   string    `json:"reasoning"`
}

// KPIStatus tracks Key Performance Indicator status
type KPIStatus struct {
	Name        string       `json:"name"`
	Current     float64      `json:"current"`
	Target      float64      `json:"target"`
	Status      MetricStatus `json:"status"`
	Trend       TrendDirection `json:"trend"`
	LastUpdated time.Time    `json:"last_updated"`
}

// Recommendation provides actionable recommendations
type Recommendation struct {
	ID          string             `json:"id"`
	Category    RecommendationCategory `json:"category"`
	Priority    Priority           `json:"priority"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Actions     []ActionItem       `json:"actions"`
	Impact      ImpactAssessment   `json:"impact"`
	CreatedAt   time.Time          `json:"created_at"`
}

// RecommendationCategory categorizes recommendations
type RecommendationCategory string

const (
	CategoryCoverage    RecommendationCategory = "coverage"
	CategorySecurity    RecommendationCategory = "security"
	CategoryPerformance RecommendationCategory = "performance"
	CategoryMaintenance RecommendationCategory = "maintenance"
	CategoryProcess     RecommendationCategory = "process"
)

// Priority defines recommendation priority
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// ActionItem represents a specific action to take
type ActionItem struct {
	Description string        `json:"description"`
	Effort      string        `json:"effort"`
	Timeline    time.Duration `json:"timeline"`
	Owner       string        `json:"owner"`
}

// ImpactAssessment assesses the impact of implementing a recommendation
type ImpactAssessment struct {
	QualityImprovement    float64 `json:"quality_improvement"`
	PerformanceImprovement float64 `json:"performance_improvement"`
	MaintenanceReduction  float64 `json:"maintenance_reduction"`
	RiskReduction         float64 `json:"risk_reduction"`
}

// ReportArtifact represents a generated report artifact
type ReportArtifact struct {
	Name        string       `json:"name"`
	Type        ReportFormat `json:"type"`
	Path        string       `json:"path"`
	Size        int64        `json:"size"`
	GeneratedAt time.Time    `json:"generated_at"`
	Checksum    string       `json:"checksum"`
}

// NewTestReporter creates a new test reporter instance
func NewTestReporter(config *ReporterConfig) *TestReporter {
	return &TestReporter{
		config:        config,
		storage:       NewResultStorage(),
		trendAnalyzer: NewTrendAnalyzer(),
		dashboardGen:  NewDashboardGenerator(config.DashboardConfig),
		notifier:      NewNotificationManager(config.NotificationRules),
	}
}

// GenerateReport generates a comprehensive test report
func (r *TestReporter) GenerateReport(period ReportPeriod, pipelineResults []PipelineResult) (*TestReport, error) {
	log.Printf("Generating test report for period: %s to %s", period.StartTime.Format(time.RFC3339), period.EndTime.Format(time.RFC3339))
	
	report := &TestReport{
		ID:          generateReportID(),
		GeneratedAt: time.Now(),
		Period:      period,
	}
	
	// Generate summary
	report.Summary = r.generateSummary(pipelineResults)
	
	// Aggregate pipeline results
	report.PipelineResults = pipelineResults
	
	// Calculate quality metrics
	report.QualityMetrics = r.calculateQualityMetrics(pipelineResults)
	
	// Perform trend analysis
	report.TrendAnalysis = r.trendAnalyzer.AnalyzeTrends(pipelineResults, period)
	
	// Evaluate KPIs
	report.KPIStatus = r.evaluateKPIs(report.QualityMetrics)
	
	// Generate recommendations
	report.Recommendations = r.generateRecommendations(report)
	
	// Generate report artifacts in multiple formats
	artifacts, err := r.generateArtifacts(report)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report artifacts: %w", err)
	}
	report.Artifacts = artifacts
	
	// Store report for future trend analysis
	if err := r.storage.StoreReport(report); err != nil {
		log.Printf("Warning: failed to store report: %v", err)
	}
	
	// Send notifications if configured
	r.sendNotifications(report)
	
	log.Printf("Test report generated successfully: %s", report.ID)
	return report, nil
}

// generateSummary creates a high-level summary of test results
func (r *TestReporter) generateSummary(results []PipelineResult) TestSummary {
	summary := TestSummary{
		TotalPipelines: len(results),
		StageBreakdown: make(map[StageType]StageStats),
	}
	
	var totalExecutionTime time.Duration
	var totalTests, passedTests, failedTests int
	var coverageSum float64
	var coverageCount int
	var securityIssues int
	var performanceRegressions []float64
	
	// Track stage statistics
	stageStats := make(map[StageType]*StageStats)
	
	for _, result := range results {
		// Pipeline-level stats
		if result.Status == PipelineStatusPassed {
			summary.SuccessfulPipelines++
		} else if result.Status == PipelineStatusFailed {
			summary.FailedPipelines++
		}
		
		totalExecutionTime += result.Duration
		
		// Stage-level stats
		for _, stage := range result.Stages {
			if stageStats[stage.Type] == nil {
				stageStats[stage.Type] = &StageStats{
					CommonFailures: []string{},
				}
			}
			
			stats := stageStats[stage.Type]
			stats.TotalExecutions++
			
			if stage.Status == StageStatusPassed {
				stats.SuccessfulRuns++
			} else if stage.Status == StageStatusFailed {
				stats.FailedRuns++
				if stage.Error != "" {
					stats.CommonFailures = append(stats.CommonFailures, stage.Error)
				}
			}
			
			// Extract metrics
			if stage.Metrics != nil {
				if tests, ok := stage.Metrics["total_tests"].(int); ok {
					totalTests += tests
				}
				if passed, ok := stage.Metrics["tests_passed"].(int); ok {
					passedTests += passed
				}
				if failed, ok := stage.Metrics["tests_failed"].(int); ok {
					failedTests += failed
				}
				if coverage, ok := stage.Metrics["coverage"].(float64); ok {
					coverageSum += coverage
					coverageCount++
				}
				if criticalIssues, ok := stage.Metrics["critical_issues"].(int); ok {
					securityIssues += criticalIssues
				}
				if regression, ok := stage.Metrics["regression_percentage"].(float64); ok {
					performanceRegressions = append(performanceRegressions, regression)
				}
			}
		}
	}
	
	// Calculate derived metrics
	if summary.TotalPipelines > 0 {
		summary.SuccessRate = float64(summary.SuccessfulPipelines) / float64(summary.TotalPipelines) * 100
		summary.AverageExecutionTime = totalExecutionTime / time.Duration(summary.TotalPipelines)
	}
	
	summary.TotalTestsRun = totalTests
	summary.TestsPassedTotal = passedTests
	summary.TestsFailedTotal = failedTests
	
	if coverageCount > 0 {
		summary.CodeCoverageAvg = coverageSum / float64(coverageCount)
	}
	
	summary.SecurityIssuesFound = securityIssues
	
	if len(performanceRegressions) > 0 {
		var regressionSum float64
		for _, regression := range performanceRegressions {
			regressionSum += regression
		}
		summary.PerformanceRegression = regressionSum / float64(len(performanceRegressions))
	}
	
	// Finalize stage statistics
	for stageType, stats := range stageStats {
		if stats.TotalExecutions > 0 {
			stats.SuccessRate = float64(stats.SuccessfulRuns) / float64(stats.TotalExecutions) * 100
		}
		
		// Deduplicate common failures
		stats.CommonFailures = r.deduplicateAndLimit(stats.CommonFailures, 5)
		
		summary.StageBreakdown[stageType] = *stats
	}
	
	return summary
}

// calculateQualityMetrics calculates comprehensive quality metrics
func (r *TestReporter) calculateQualityMetrics(results []PipelineResult) QualityMetrics {
	metrics := QualityMetrics{}
	
	// Calculate coverage metrics
	metrics.CodeCoverage = r.calculateCoverageMetrics(results)
	
	// Calculate test effectiveness metrics
	metrics.TestEffectiveness = r.calculateTestEffectivenessMetrics(results)
	
	// Calculate security metrics
	metrics.SecurityMetrics = r.calculateSecurityMetrics(results)
	
	// Calculate performance metrics
	metrics.PerformanceMetrics = r.calculatePerformanceMetrics(results)
	
	// Calculate mutation test metrics
	metrics.MutationTestMetrics = r.calculateMutationTestMetrics(results)
	
	return metrics
}

// calculateCoverageMetrics calculates code coverage metrics
func (r *TestReporter) calculateCoverageMetrics(results []PipelineResult) CoverageMetrics {
	metrics := CoverageMetrics{
		ByPackage: make(map[string]float64),
		Trend:     []CoveragePoint{},
		Threshold: r.config.KPIThresholds["min_coverage"],
	}
	
	var coverageSum float64
	var coverageCount int
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Metrics != nil {
				if coverage, ok := stage.Metrics["coverage"].(float64); ok {
					coverageSum += coverage
					coverageCount++
					
					// Add to trend
					metrics.Trend = append(metrics.Trend, CoveragePoint{
						Timestamp: stage.StartTime,
						Coverage:  coverage,
						Commit:    result.Trigger.Commit,
					})
				}
				
				// Package-level coverage (if available)
				if packageCoverage, ok := stage.Metrics["package_coverage"].(map[string]float64); ok {
					for pkg, cov := range packageCoverage {
						metrics.ByPackage[pkg] = cov
					}
				}
			}
		}
	}
	
	if coverageCount > 0 {
		metrics.Overall = coverageSum / float64(coverageCount)
	}
	
	// Determine status
	if metrics.Overall >= metrics.Threshold {
		metrics.Status = StatusGood
	} else if metrics.Overall >= metrics.Threshold-5 {
		metrics.Status = StatusWarning
	} else {
		metrics.Status = StatusCritical
	}
	
	return metrics
}

// Helper functions continue...
func (r *TestReporter) deduplicateAndLimit(items []string, limit int) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, item := range items {
		if !seen[item] && len(result) < limit {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

func generateReportID() string {
	return fmt.Sprintf("report-%d", time.Now().Unix())
}

// calculateTestEffectivenessMetrics calculates test effectiveness metrics
func (r *TestReporter) calculateTestEffectivenessMetrics(results []PipelineResult) TestEffectivenessMetrics {
	metrics := TestEffectivenessMetrics{}
	
	var totalExecutionTime time.Duration
	var executionCount int
	var flakyTests int
	var totalTests int
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Type == StageTypeUnit || stage.Type == StageTypeIntegration {
				totalExecutionTime += stage.Duration
				executionCount++
				
				if stage.Metrics != nil {
					if tests, ok := stage.Metrics["total_tests"].(int); ok {
						totalTests += tests
					}
					if flaky, ok := stage.Metrics["flaky_tests"].(int); ok {
						flakyTests += flaky
					}
				}
			}
		}
	}
	
	if executionCount > 0 {
		metrics.TestExecutionTime = totalExecutionTime / time.Duration(executionCount)
	}
	
	if totalTests > 0 {
		metrics.FlakyTestRate = float64(flakyTests) / float64(totalTests) * 100
	}
	
	// Calculate defect detection rate (simplified)
	metrics.DefectDetectionRate = 85.0 // This would be calculated from actual defect data
	
	return metrics
}

// calculateSecurityMetrics calculates security-related metrics
func (r *TestReporter) calculateSecurityMetrics(results []PipelineResult) SecurityMetrics {
	metrics := SecurityMetrics{
		VulnerabilityTrend: []VulnerabilityPoint{},
		ComplianceStatus:   make(map[string]bool),
	}
	
	var totalVulns, criticalVulns int
	var securityScores []float64
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Type == StageTypeSecurity && stage.Metrics != nil {
				if total, ok := stage.Metrics["total_issues"].(int); ok {
					totalVulns += total
				}
				if critical, ok := stage.Metrics["critical_issues"].(int); ok {
					criticalVulns += critical
				}
				if score, ok := stage.Metrics["security_score"].(float64); ok {
					securityScores = append(securityScores, score)
				}
				
				// Add to trend
				point := VulnerabilityPoint{
					Timestamp: stage.StartTime,
					Total:     totalVulns,
					Critical:  criticalVulns,
				}
				metrics.VulnerabilityTrend = append(metrics.VulnerabilityTrend, point)
			}
		}
	}
	
	metrics.VulnerabilitiesFound = totalVulns
	metrics.CriticalVulnerabilities = criticalVulns
	
	if len(securityScores) > 0 {
		var sum float64
		for _, score := range securityScores {
			sum += score
		}
		metrics.SecurityScoreAvg = sum / float64(len(securityScores))
	}
	
	// Set compliance status (simplified)
	metrics.ComplianceStatus["OWASP"] = criticalVulns == 0
	metrics.ComplianceStatus["Security_Standards"] = metrics.SecurityScoreAvg > 80
	
	return metrics
}

// calculatePerformanceMetrics calculates performance-related metrics
func (r *TestReporter) calculatePerformanceMetrics(results []PipelineResult) PerformanceMetrics {
	metrics := PerformanceMetrics{
		PerformanceTrend: []PerformancePoint{},
		Bottlenecks:      []PerformanceBottleneck{},
	}
	
	var regressions []float64
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Type == StageTypePerformance && stage.Metrics != nil {
				if regression, ok := stage.Metrics["regression_percentage"].(float64); ok {
					regressions = append(regressions, regression)
				}
				
				// Extract performance data for trend
				if responseTime, ok := stage.Metrics["response_time"].(float64); ok {
					point := PerformancePoint{
						Timestamp:    stage.StartTime,
						ResponseTime: responseTime,
					}
					if throughput, ok := stage.Metrics["throughput"].(float64); ok {
						point.Throughput = throughput
					}
					if errorRate, ok := stage.Metrics["error_rate"].(float64); ok {
						point.ErrorRate = errorRate
					}
					metrics.PerformanceTrend = append(metrics.PerformanceTrend, point)
				}
			}
		}
	}
	
	if len(regressions) > 0 {
		var sum float64
		for _, regression := range regressions {
			sum += regression
		}
		metrics.RegressionPercentage = sum / float64(len(regressions))
	}
	
	// Identify bottlenecks (simplified)
	if metrics.RegressionPercentage > 10 {
		bottleneck := PerformanceBottleneck{
			Component:  "Database",
			Metric:     "Query Response Time",
			Impact:     metrics.RegressionPercentage,
			Severity:   "High",
			Suggestion: "Optimize database queries and add indexes",
		}
		metrics.Bottlenecks = append(metrics.Bottlenecks, bottleneck)
	}
	
	return metrics
}

// calculateMutationTestMetrics calculates mutation testing metrics
func (r *TestReporter) calculateMutationTestMetrics(results []PipelineResult) MutationTestMetrics {
	metrics := MutationTestMetrics{
		WeakTests:  []WeakTestInfo{},
		ScoreTrend: []MutationPoint{},
	}
	
	var scores []float64
	var totalKilled, totalGenerated int
	
	for _, result := range results {
		for _, stage := range result.Stages {
			if stage.Type == StageTypeMutation && stage.Metrics != nil {
				if score, ok := stage.Metrics["mutation_score"].(float64); ok {
					scores = append(scores, score)
					
					// Add to trend
					point := MutationPoint{
						Timestamp: stage.StartTime,
						Score:     score,
						Package:   "overall",
					}
					metrics.ScoreTrend = append(metrics.ScoreTrend, point)
				}
				
				if killed, ok := stage.Metrics["mutants_killed"].(int); ok {
					totalKilled += killed
				}
				if generated, ok := stage.Metrics["mutants_generated"].(int); ok {
					totalGenerated += generated
				}
			}
		}
	}
	
	if len(scores) > 0 {
		var sum float64
		for _, score := range scores {
			sum += score
		}
		metrics.MutationScore = sum / float64(len(scores))
	}
	
	metrics.MutantsKilled = totalKilled
	metrics.MutantsGenerated = totalGenerated
	
	return metrics
}

// evaluateKPIs evaluates Key Performance Indicators
func (r *TestReporter) evaluateKPIs(metrics QualityMetrics) map[string]KPIStatus {
	kpis := make(map[string]KPIStatus)
	
	// Code Coverage KPI
	coverageKPI := KPIStatus{
		Name:        "Code Coverage",
		Current:     metrics.CodeCoverage.Overall,
		Target:      r.config.KPIThresholds["min_coverage"],
		LastUpdated: time.Now(),
	}
	
	if coverageKPI.Current >= coverageKPI.Target {
		coverageKPI.Status = StatusGood
		coverageKPI.Trend = TrendStable
	} else {
		coverageKPI.Status = StatusWarning
		coverageKPI.Trend = TrendDeclining
	}
	
	kpis["code_coverage"] = coverageKPI
	
	// Security KPI
	securityKPI := KPIStatus{
		Name:        "Security Score",
		Current:     metrics.SecurityMetrics.SecurityScoreAvg,
		Target:      r.config.KPIThresholds["min_security_score"],
		LastUpdated: time.Now(),
	}
	
	if securityKPI.Current >= securityKPI.Target {
		securityKPI.Status = StatusGood
	} else {
		securityKPI.Status = StatusCritical
	}
	
	kpis["security_score"] = securityKPI
	
	// Performance KPI
	performanceKPI := KPIStatus{
		Name:        "Performance Regression",
		Current:     metrics.PerformanceMetrics.RegressionPercentage,
		Target:      r.config.KPIThresholds["max_performance_regression"],
		LastUpdated: time.Now(),
	}
	
	if performanceKPI.Current <= performanceKPI.Target {
		performanceKPI.Status = StatusGood
	} else {
		performanceKPI.Status = StatusWarning
	}
	
	kpis["performance_regression"] = performanceKPI
	
	// Test Effectiveness KPI
	effectivenessKPI := KPIStatus{
		Name:        "Test Effectiveness",
		Current:     metrics.TestEffectiveness.DefectDetectionRate,
		Target:      r.config.KPIThresholds["min_test_effectiveness"],
		LastUpdated: time.Now(),
	}
	
	if effectivenessKPI.Current >= effectivenessKPI.Target {
		effectivenessKPI.Status = StatusGood
	} else {
		effectivenessKPI.Status = StatusWarning
	}
	
	kpis["test_effectiveness"] = effectivenessKPI
	
	return kpis
}

// generateRecommendations generates actionable recommendations
func (r *TestReporter) generateRecommendations(report *TestReport) []Recommendation {
	var recommendations []Recommendation
	
	// Coverage recommendations
	if report.QualityMetrics.CodeCoverage.Overall < r.config.KPIThresholds["min_coverage"] {
		rec := Recommendation{
			ID:          "coverage-improvement",
			Category:    CategoryCoverage,
			Priority:    PriorityHigh,
			Title:       "Improve Code Coverage",
			Description: fmt.Sprintf("Code coverage is %.1f%%, below target of %.1f%%", 
				report.QualityMetrics.CodeCoverage.Overall, r.config.KPIThresholds["min_coverage"]),
			Actions: []ActionItem{
				{
					Description: "Add unit tests for uncovered code paths",
					Effort:      "Medium",
					Timeline:    2 * 24 * time.Hour,
					Owner:       "Development Team",
				},
				{
					Description: "Review and improve existing test quality",
					Effort:      "Low",
					Timeline:    1 * 24 * time.Hour,
					Owner:       "QA Team",
				},
			},
			Impact: ImpactAssessment{
				QualityImprovement: 15.0,
				RiskReduction:      20.0,
			},
			CreatedAt: time.Now(),
		}
		recommendations = append(recommendations, rec)
	}
	
	// Security recommendations
	if report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities > 0 {
		rec := Recommendation{
			ID:          "security-critical",
			Category:    CategorySecurity,
			Priority:    PriorityCritical,
			Title:       "Address Critical Security Vulnerabilities",
			Description: fmt.Sprintf("Found %d critical security vulnerabilities that must be addressed immediately", 
				report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities),
			Actions: []ActionItem{
				{
					Description: "Review and fix all critical security issues",
					Effort:      "High",
					Timeline:    4 * time.Hour,
					Owner:       "Security Team",
				},
				{
					Description: "Implement additional security controls",
					Effort:      "Medium",
					Timeline:    1 * 24 * time.Hour,
					Owner:       "Development Team",
				},
			},
			Impact: ImpactAssessment{
				RiskReduction: 80.0,
			},
			CreatedAt: time.Now(),
		}
		recommendations = append(recommendations, rec)
	}
	
	// Performance recommendations
	if report.QualityMetrics.PerformanceMetrics.RegressionPercentage > r.config.KPIThresholds["max_performance_regression"] {
		rec := Recommendation{
			ID:          "performance-regression",
			Category:    CategoryPerformance,
			Priority:    PriorityHigh,
			Title:       "Address Performance Regression",
			Description: fmt.Sprintf("Performance regression of %.1f%% detected, exceeding threshold of %.1f%%", 
				report.QualityMetrics.PerformanceMetrics.RegressionPercentage, 
				r.config.KPIThresholds["max_performance_regression"]),
			Actions: []ActionItem{
				{
					Description: "Profile and optimize performance bottlenecks",
					Effort:      "High",
					Timeline:    3 * 24 * time.Hour,
					Owner:       "Performance Team",
				},
			},
			Impact: ImpactAssessment{
				PerformanceImprovement: 25.0,
			},
			CreatedAt: time.Now(),
		}
		recommendations = append(recommendations, rec)
	}
	
	// Test maintenance recommendations
	if report.QualityMetrics.TestEffectiveness.FlakyTestRate > 5.0 {
		rec := Recommendation{
			ID:          "flaky-tests",
			Category:    CategoryMaintenance,
			Priority:    PriorityMedium,
			Title:       "Reduce Flaky Test Rate",
			Description: fmt.Sprintf("Flaky test rate is %.1f%%, which impacts CI/CD reliability", 
				report.QualityMetrics.TestEffectiveness.FlakyTestRate),
			Actions: []ActionItem{
				{
					Description: "Identify and fix flaky tests",
					Effort:      "Medium",
					Timeline:    2 * 24 * time.Hour,
					Owner:       "QA Team",
				},
				{
					Description: "Implement test stability monitoring",
					Effort:      "Low",
					Timeline:    4 * time.Hour,
					Owner:       "DevOps Team",
				},
			},
			Impact: ImpactAssessment{
				MaintenanceReduction: 30.0,
			},
			CreatedAt: time.Now(),
		}
		recommendations = append(recommendations, rec)
	}
	
	return recommendations
}

// generateArtifacts generates report artifacts in multiple formats
func (r *TestReporter) generateArtifacts(report *TestReport) ([]ReportArtifact, error) {
	var artifacts []ReportArtifact
	
	// Ensure output directory exists
	if err := os.MkdirAll(r.config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	
	for _, format := range r.config.ReportFormats {
		artifact, err := r.generateArtifact(report, format)
		if err != nil {
			log.Printf("Warning: failed to generate %s report: %v", format, err)
			continue
		}
		artifacts = append(artifacts, artifact)
	}
	
	return artifacts, nil
}

// generateArtifact generates a single report artifact
func (r *TestReporter) generateArtifact(report *TestReport, format ReportFormat) (ReportArtifact, error) {
	filename := fmt.Sprintf("test-report-%s.%s", report.ID, string(format))
	filepath := filepath.Join(r.config.OutputDir, filename)
	
	var err error
	switch format {
	case FormatJSON:
		err = r.generateJSONReport(report, filepath)
	case FormatHTML:
		err = r.generateHTMLReport(report, filepath)
	case FormatCSV:
		err = r.generateCSVReport(report, filepath)
	case FormatMarkdown:
		err = r.generateMarkdownReport(report, filepath)
	case FormatJUnit:
		err = r.generateJUnitReport(report, filepath)
	default:
		return ReportArtifact{}, fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return ReportArtifact{}, err
	}
	
	// Get file info
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return ReportArtifact{}, err
	}
	
	artifact := ReportArtifact{
		Name:        filename,
		Type:        format,
		Path:        filepath,
		Size:        fileInfo.Size(),
		GeneratedAt: time.Now(),
		Checksum:    r.calculateChecksum(filepath),
	}
	
	return artifact, nil
}

// generateJSONReport generates a JSON format report
func (r *TestReporter) generateJSONReport(report *TestReport, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// generateHTMLReport generates an HTML format report
func (r *TestReporter) generateHTMLReport(report *TestReport, filepath string) error {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Test Report - {{.ID}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .summary { display: flex; justify-content: space-between; margin: 20px 0; }
        .metric { text-align: center; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
        .good { background-color: #d4edda; }
        .warning { background-color: #fff3cd; }
        .critical { background-color: #f8d7da; }
        .recommendations { margin: 20px 0; }
        .recommendation { margin: 10px 0; padding: 10px; border-left: 4px solid #007bff; }
        .high-priority { border-left-color: #dc3545; }
        .critical-priority { border-left-color: #6f42c1; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Test Report</h1>
        <p><strong>Report ID:</strong> {{.ID}}</p>
        <p><strong>Generated:</strong> {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        <p><strong>Period:</strong> {{.Period.StartTime.Format "2006-01-02"}} to {{.Period.EndTime.Format "2006-01-02"}}</p>
    </div>
    
    <div class="summary">
        <div class="metric {{if ge .Summary.SuccessRate 90.0}}good{{else if ge .Summary.SuccessRate 70.0}}warning{{else}}critical{{end}}">
            <h3>Success Rate</h3>
            <p>{{printf "%.1f" .Summary.SuccessRate}}%</p>
        </div>
        <div class="metric {{if ge .QualityMetrics.CodeCoverage.Overall 95.0}}good{{else if ge .QualityMetrics.CodeCoverage.Overall 80.0}}warning{{else}}critical{{end}}">
            <h3>Code Coverage</h3>
            <p>{{printf "%.1f" .QualityMetrics.CodeCoverage.Overall}}%</p>
        </div>
        <div class="metric {{if eq .QualityMetrics.SecurityMetrics.CriticalVulnerabilities 0}}good{{else}}critical{{end}}">
            <h3>Security Issues</h3>
            <p>{{.QualityMetrics.SecurityMetrics.CriticalVulnerabilities}} Critical</p>
        </div>
        <div class="metric {{if le .QualityMetrics.PerformanceMetrics.RegressionPercentage 5.0}}good{{else if le .QualityMetrics.PerformanceMetrics.RegressionPercentage 15.0}}warning{{else}}critical{{end}}">
            <h3>Performance</h3>
            <p>{{printf "%.1f" .QualityMetrics.PerformanceMetrics.RegressionPercentage}}% Regression</p>
        </div>
    </div>
    
    <h2>Recommendations</h2>
    <div class="recommendations">
        {{range .Recommendations}}
        <div class="recommendation {{if eq .Priority "high"}}high-priority{{else if eq .Priority "critical"}}critical-priority{{end}}">
            <h4>{{.Title}} ({{.Priority}})</h4>
            <p>{{.Description}}</p>
            <ul>
                {{range .Actions}}
                <li>{{.Description}} ({{.Effort}} effort, {{.Timeline}})</li>
                {{end}}
            </ul>
        </div>
        {{end}}
    </div>
</body>
</html>`
	
	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return err
	}
	
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return t.Execute(file, report)
}

// generateMarkdownReport generates a Markdown format report
func (r *TestReporter) generateMarkdownReport(report *TestReport, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	fmt.Fprintf(file, "# Test Report - %s\n\n", report.ID)
	fmt.Fprintf(file, "**Generated:** %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "**Period:** %s to %s\n\n", 
		report.Period.StartTime.Format("2006-01-02"), 
		report.Period.EndTime.Format("2006-01-02"))
	
	fmt.Fprintf(file, "## Summary\n\n")
	fmt.Fprintf(file, "- **Total Pipelines:** %d\n", report.Summary.TotalPipelines)
	fmt.Fprintf(file, "- **Success Rate:** %.1f%%\n", report.Summary.SuccessRate)
	fmt.Fprintf(file, "- **Code Coverage:** %.1f%%\n", report.QualityMetrics.CodeCoverage.Overall)
	fmt.Fprintf(file, "- **Security Issues:** %d critical\n", report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities)
	fmt.Fprintf(file, "- **Performance Regression:** %.1f%%\n\n", report.QualityMetrics.PerformanceMetrics.RegressionPercentage)
	
	fmt.Fprintf(file, "## Recommendations\n\n")
	for _, rec := range report.Recommendations {
		fmt.Fprintf(file, "### %s (%s priority)\n\n", rec.Title, rec.Priority)
		fmt.Fprintf(file, "%s\n\n", rec.Description)
		fmt.Fprintf(file, "**Actions:**\n")
		for _, action := range rec.Actions {
			fmt.Fprintf(file, "- %s (%s effort)\n", action.Description, action.Effort)
		}
		fmt.Fprintf(file, "\n")
	}
	
	return nil
}

// generateCSVReport generates a CSV format report (simplified)
func (r *TestReporter) generateCSVReport(report *TestReport, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write CSV header
	fmt.Fprintf(file, "Metric,Value,Status\n")
	
	// Write metrics
	fmt.Fprintf(file, "Success Rate,%.1f%%,%s\n", report.Summary.SuccessRate, 
		r.getStatusString(report.Summary.SuccessRate >= 90))
	fmt.Fprintf(file, "Code Coverage,%.1f%%,%s\n", report.QualityMetrics.CodeCoverage.Overall,
		string(report.QualityMetrics.CodeCoverage.Status))
	fmt.Fprintf(file, "Critical Security Issues,%d,%s\n", report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities,
		r.getStatusString(report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities == 0))
	
	return nil
}

// generateJUnitReport generates a JUnit XML format report
func (r *TestReporter) generateJUnitReport(report *TestReport, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	fmt.Fprintf(file, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintf(file, `<testsuites name="CI/CD Pipeline Tests" tests="%d" failures="%d" time="%.2f">`,
		report.Summary.TotalTestsRun, report.Summary.TestsFailedTotal, report.Summary.AverageExecutionTime.Seconds())
	
	for _, result := range report.PipelineResults {
		fmt.Fprintf(file, `<testsuite name="Pipeline-%s" tests="%d" failures="%d" time="%.2f">`,
			result.ID, len(result.Stages), r.countFailedStages(result.Stages), result.Duration.Seconds())
		
		for _, stage := range result.Stages {
			if stage.Status == StageStatusFailed {
				fmt.Fprintf(file, `<testcase name="%s" classname="Pipeline" time="%.2f">`, stage.Name, stage.Duration.Seconds())
				fmt.Fprintf(file, `<failure message="%s">%s</failure>`, stage.Error, stage.Output)
				fmt.Fprintf(file, `</testcase>`)
			} else {
				fmt.Fprintf(file, `<testcase name="%s" classname="Pipeline" time="%.2f"/>`, stage.Name, stage.Duration.Seconds())
			}
		}
		
		fmt.Fprintf(file, `</testsuite>`)
	}
	
	fmt.Fprintf(file, `</testsuites>`)
	return nil
}

// Helper functions
func (r *TestReporter) getStatusString(good bool) string {
	if good {
		return "PASS"
	}
	return "FAIL"
}

func (r *TestReporter) countFailedStages(stages []StageResult) int {
	count := 0
	for _, stage := range stages {
		if stage.Status == StageStatusFailed {
			count++
		}
	}
	return count
}

func (r *TestReporter) calculateChecksum(filepath string) string {
	// Simplified checksum calculation
	return fmt.Sprintf("checksum-%d", time.Now().Unix())
}

// sendNotifications sends notifications based on report results
func (r *TestReporter) sendNotifications(report *TestReport) {
	if r.notifier != nil {
		r.notifier.SendReportNotifications(report)
	}
}