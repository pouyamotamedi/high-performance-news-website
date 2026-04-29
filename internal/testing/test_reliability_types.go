package testing

import (
	"time"
)

// TestExecutionRecord represents a single test execution
type TestExecutionRecord struct {
	ID           int64         `json:"id"`
	TestName     string        `json:"test_name"`
	TestSuite    string        `json:"test_suite"`
	Status       string        `json:"status"` // "passed", "failed", "error", "skipped"
	Duration     time.Duration `json:"duration"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Environment  string        `json:"environment"`
	BuildID      string        `json:"build_id"`
	CommitHash   string        `json:"commit_hash"`
	Branch       string        `json:"branch"`
}

// FlakyTestInfo represents information about a flaky test
type FlakyTestInfo struct {
	TestName        string    `json:"test_name"`
	TestSuite       string    `json:"test_suite"`
	FlakinessScore  float64   `json:"flakiness_score"`
	TotalRuns       int64     `json:"total_runs"`
	FailureCount    int64     `json:"failure_count"`
	FailureRate     float64   `json:"failure_rate"`
	LastFailure     time.Time `json:"last_failure"`
	Status          string    `json:"status"` // "active", "quarantined", "fixed"
	QuarantinedAt   time.Time `json:"quarantined_at,omitempty"`
	ReintegratedAt  time.Time `json:"reintegrated_at,omitempty"`
	FailurePatterns []FailurePattern `json:"failure_patterns"`
}

// FailurePattern represents a pattern in test failures
type FailurePattern struct {
	Type        string    `json:"type"`        // "intermittent", "consecutive", "environment", "timing"
	Description string    `json:"description"`
	Frequency   float64   `json:"frequency"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Confidence  float64   `json:"confidence"` // 0.0 to 1.0
}

// TestReliabilityMetrics represents comprehensive reliability metrics for a test
type TestReliabilityMetrics struct {
	TestName              string                 `json:"test_name"`
	TestSuite             string                 `json:"test_suite"`
	ReliabilityScore      float64                `json:"reliability_score"`      // 0.0 to 1.0 (1.0 = most reliable)
	FlakinessScore        float64                `json:"flakiness_score"`        // 0.0 to 1.0 (1.0 = most flaky)
	StabilityTrend        string                 `json:"stability_trend"`        // "improving", "stable", "degrading"
	TotalExecutions       int64                  `json:"total_executions"`
	SuccessfulExecutions  int64                  `json:"successful_executions"`
	FailedExecutions      int64                  `json:"failed_executions"`
	ErrorExecutions       int64                  `json:"error_executions"`
	SkippedExecutions     int64                  `json:"skipped_executions"`
	AverageDuration       time.Duration          `json:"average_duration"`
	DurationVariance      float64                `json:"duration_variance"`
	FailurePatterns       []FailurePattern       `json:"failure_patterns"`
	EnvironmentImpact     map[string]float64     `json:"environment_impact"`     // Environment -> failure rate
	TimeOfDayImpact       map[string]float64     `json:"time_of_day_impact"`     // Hour -> failure rate
	RecentPerformance     []ExecutionSummary     `json:"recent_performance"`     // Last 50 executions summary
	LastUpdated           time.Time              `json:"last_updated"`
}

// ExecutionSummary represents a summary of recent executions
type ExecutionSummary struct {
	Date         time.Time `json:"date"`
	Executions   int       `json:"executions"`
	Failures     int       `json:"failures"`
	FailureRate  float64   `json:"failure_rate"`
	AvgDuration  time.Duration `json:"avg_duration"`
}

// RemediationSuggestion represents a suggestion for fixing flaky tests
type RemediationSuggestion struct {
	Type        string    `json:"type"`        // "timing", "environment", "dependency", "resource"
	Priority    string    `json:"priority"`    // "high", "medium", "low"
	Description string    `json:"description"`
	Action      string    `json:"action"`
	Confidence  float64   `json:"confidence"`
	Examples    []string  `json:"examples,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// TestStabilityReport represents a comprehensive stability report
type TestStabilityReport struct {
	GeneratedAt          time.Time                        `json:"generated_at"`
	TotalTests           int                              `json:"total_tests"`
	StableTests          int                              `json:"stable_tests"`
	FlakyTests           int                              `json:"flaky_tests"`
	QuarantinedTests     int                              `json:"quarantined_tests"`
	OverallStability     float64                          `json:"overall_stability"`
	StabilityTrend       string                           `json:"stability_trend"`
	TestsByReliability   map[string][]TestReliabilityMetrics `json:"tests_by_reliability"` // "high", "medium", "low"
	EnvironmentAnalysis  EnvironmentStabilityAnalysis     `json:"environment_analysis"`
	RecommendedActions   []RemediationSuggestion          `json:"recommended_actions"`
}

// EnvironmentStabilityAnalysis represents analysis of test stability by environment
type EnvironmentStabilityAnalysis struct {
	Environments        []string           `json:"environments"`
	StabilityByEnv      map[string]float64 `json:"stability_by_env"`
	ProblematicEnvs     []string           `json:"problematic_envs"`
	RecommendedOptimizations []string      `json:"recommended_optimizations"`
}

// TestReliabilityConfig represents configuration for reliability tracking
type TestReliabilityConfig struct {
	FlakinessThreshold      float64       `json:"flakiness_threshold"`       // Threshold for quarantine (default: 0.3)
	ReliabilityThreshold    float64       `json:"reliability_threshold"`     // Minimum reliability score (default: 0.8)
	MinExecutionsForAnalysis int          `json:"min_executions_for_analysis"` // Minimum executions before analysis (default: 10)
	AnalysisWindow          time.Duration `json:"analysis_window"`           // Time window for analysis (default: 7 days)
	QuarantineCooldown      time.Duration `json:"quarantine_cooldown"`       // Cooldown before reintegration (default: 24 hours)
	PatternDetectionWindow  time.Duration `json:"pattern_detection_window"`  // Window for pattern detection (default: 3 days)
}

// DefaultTestReliabilityConfig returns default configuration
func DefaultTestReliabilityConfig() *TestReliabilityConfig {
	return &TestReliabilityConfig{
		FlakinessThreshold:       0.3,
		ReliabilityThreshold:     0.8,
		MinExecutionsForAnalysis: 10,
		AnalysisWindow:          7 * 24 * time.Hour,
		QuarantineCooldown:      24 * time.Hour,
		PatternDetectionWindow:  3 * 24 * time.Hour,
	}
}