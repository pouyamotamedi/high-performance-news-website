package testing

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// TestStabilityOptimizer provides automated test stability optimization
type TestStabilityOptimizer struct {
	db                    *sql.DB
	reliabilityTracker    *TestReliabilityTracker
	remediationEngine     *RemediationEngine
	environmentOptimizer  *EnvironmentOptimizer
	notificationManager   NotificationManagerInterface
	config                *StabilityOptimizerConfig
}

// StabilityOptimizerConfig holds configuration for the stability optimizer
type StabilityOptimizerConfig struct {
	AutoRemediationEnabled    bool          `json:"auto_remediation_enabled"`
	StabilityThreshold        float64       `json:"stability_threshold"`        // Minimum acceptable stability (default: 0.8)
	OptimizationInterval      time.Duration `json:"optimization_interval"`      // How often to run optimization (default: 1 hour)
	EnvironmentOptimization   bool          `json:"environment_optimization"`   // Enable environment optimization
	AutoQuarantineEnabled     bool          `json:"auto_quarantine_enabled"`    // Enable automatic quarantine
	NotificationEnabled       bool          `json:"notification_enabled"`       // Enable notifications
	MaxRemediationAttempts    int           `json:"max_remediation_attempts"`   // Max auto-remediation attempts (default: 3)
	RemediationCooldown       time.Duration `json:"remediation_cooldown"`       // Cooldown between attempts (default: 2 hours)
}

// DefaultStabilityOptimizerConfig returns default configuration
func DefaultStabilityOptimizerConfig() *StabilityOptimizerConfig {
	return &StabilityOptimizerConfig{
		AutoRemediationEnabled:  false, // Start with manual approval
		StabilityThreshold:      0.8,
		OptimizationInterval:    1 * time.Hour,
		EnvironmentOptimization: true,
		AutoQuarantineEnabled:   true,
		NotificationEnabled:     true,
		MaxRemediationAttempts:  3,
		RemediationCooldown:     2 * time.Hour,
	}
}

// NewTestStabilityOptimizer creates a new test stability optimizer
func NewTestStabilityOptimizer(db *sql.DB, reliabilityTracker *TestReliabilityTracker, config *StabilityOptimizerConfig) *TestStabilityOptimizer {
	if config == nil {
		config = DefaultStabilityOptimizerConfig()
	}

	optimizer := &TestStabilityOptimizer{
		db:                 db,
		reliabilityTracker: reliabilityTracker,
		config:             config,
	}

	optimizer.remediationEngine = NewRemediationEngine()
	optimizer.remediationEngine.SetDatabase(db)
	optimizer.environmentOptimizer = NewEnvironmentOptimizer(db)
	optimizer.notificationManager = &SimpleNotificationManager{}

	return optimizer
}

// OptimizeTestStability performs comprehensive test stability optimization
func (o *TestStabilityOptimizer) OptimizeTestStability() (*StabilityOptimizationReport, error) {
	report := &StabilityOptimizationReport{
		StartTime:     time.Now(),
		Optimizations: make([]OptimizationAction, 0),
		Metrics:       make(map[string]interface{}),
	}

	log.Printf("Starting test stability optimization...")

	// 1. Identify unstable tests
	unstableTests, err := o.identifyUnstableTests()
	if err != nil {
		return nil, fmt.Errorf("failed to identify unstable tests: %w", err)
	}

	report.UnstableTestsFound = len(unstableTests)
	log.Printf("Found %d unstable tests", len(unstableTests))

	// 2. Generate remediation suggestions for each unstable test
	for _, test := range unstableTests {
		suggestions, err := o.remediationEngine.GenerateRemediationSuggestions(test.TestName, test.TestSuite)
		if err != nil {
			log.Printf("Failed to generate suggestions for %s: %v", test.TestName, err)
			continue
		}

		// 3. Apply automatic remediations if enabled
		if o.config.AutoRemediationEnabled {
			applied := o.applyAutomaticRemediations(test, suggestions)
			report.Optimizations = append(report.Optimizations, applied...)
		}

		// 4. Quarantine severely flaky tests if enabled
		if o.config.AutoQuarantineEnabled && test.FlakinessScore > 0.7 {
			err := o.quarantineTest(test, "Automatic quarantine due to high flakiness score")
			if err != nil {
				log.Printf("Failed to quarantine test %s: %v", test.TestName, err)
			} else {
				report.Optimizations = append(report.Optimizations, OptimizationAction{
					Type:        "quarantine",
					TestName:    test.TestName,
					TestSuite:   test.TestSuite,
					Description: "Automatically quarantined due to high flakiness",
					Applied:     true,
					Timestamp:   time.Now(),
				})
				report.TestsQuarantined++
			}
		}
	}

	// 5. Optimize test environments
	if o.config.EnvironmentOptimization {
		envOptimizations, err := o.optimizeTestEnvironments()
		if err != nil {
			log.Printf("Failed to optimize environments: %v", err)
		} else {
			report.Optimizations = append(report.Optimizations, envOptimizations...)
			report.EnvironmentOptimizations = len(envOptimizations)
		}
	}

	// 6. Generate stability improvement recommendations
	recommendations := o.generateStabilityRecommendations(unstableTests)
	report.Recommendations = recommendations

	// 7. Update optimization metrics
	o.updateOptimizationMetrics(report)

	// 8. Send notifications if enabled
	if o.config.NotificationEnabled {
		o.sendOptimizationNotifications(report)
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)

	log.Printf("Completed test stability optimization in %v", report.Duration)
	return report, nil
}

// identifyUnstableTests identifies tests that need stability optimization
func (o *TestStabilityOptimizer) identifyUnstableTests() ([]TestReliabilityMetrics, error) {
	flakyTests, err := o.reliabilityTracker.GetFlakyTests()
	if err != nil {
		return nil, err
	}

	var unstableTests []TestReliabilityMetrics
	for _, test := range flakyTests {
		// Include tests below stability threshold or with high flakiness
		if test.ReliabilityScore < o.config.StabilityThreshold || 
		   test.FlakinessScore > 0.3 {
			unstableTests = append(unstableTests, test)
		}
	}

	// Sort by priority (most problematic first)
	sort.Slice(unstableTests, func(i, j int) bool {
		// Prioritize by flakiness score, then by reliability score
		if unstableTests[i].FlakinessScore != unstableTests[j].FlakinessScore {
			return unstableTests[i].FlakinessScore > unstableTests[j].FlakinessScore
		}
		return unstableTests[i].ReliabilityScore < unstableTests[j].ReliabilityScore
	})

	return unstableTests, nil
}

// applyAutomaticRemediations applies safe automatic remediations
func (o *TestStabilityOptimizer) applyAutomaticRemediations(test TestReliabilityMetrics, suggestions []RemediationSuggestion) []OptimizationAction {
	var actions []OptimizationAction

	// Check if we've already attempted remediation recently
	if o.hasRecentRemediationAttempt(test.TestName, test.TestSuite) {
		return actions
	}

	// Only apply safe, non-destructive remediations automatically
	for _, suggestion := range suggestions {
		if o.isSafeForAutoRemediation(suggestion) {
			action := OptimizationAction{
				Type:        "auto_remediation",
				TestName:    test.TestName,
				TestSuite:   test.TestSuite,
				Description: fmt.Sprintf("Applied: %s", suggestion.Action),
				Applied:     false,
				Timestamp:   time.Now(),
				Details:     suggestion,
			}

			// Apply the remediation
			success := o.applyRemediation(test, suggestion)
			action.Applied = success

			if success {
				o.recordRemediationAttempt(test.TestName, test.TestSuite, suggestion.Type)
				log.Printf("Applied automatic remediation for %s: %s", test.TestName, suggestion.Action)
			}

			actions = append(actions, action)
		}
	}

	return actions
}

// isSafeForAutoRemediation determines if a suggestion is safe for automatic application
func (o *TestStabilityOptimizer) isSafeForAutoRemediation(suggestion RemediationSuggestion) bool {
	// Only apply environment and configuration changes automatically
	safeTypes := map[string]bool{
		"environment": true,
		"timing":      false, // Requires code changes
		"dependency":  false, // Requires code changes
		"resource":    true,  // Can adjust resource limits
	}

	if !safeTypes[suggestion.Type] {
		return false
	}

	// Only apply high-confidence suggestions automatically
	if suggestion.Confidence < 0.8 {
		return false
	}

	// Additional safety checks based on suggestion content
	actionLower := strings.ToLower(suggestion.Action)
	
	// Avoid destructive actions
	destructiveKeywords := []string{"delete", "remove", "drop", "truncate", "reset"}
	for _, keyword := range destructiveKeywords {
		if strings.Contains(actionLower, keyword) {
			return false
		}
	}

	return true
}

// applyRemediation applies a specific remediation suggestion
func (o *TestStabilityOptimizer) applyRemediation(test TestReliabilityMetrics, suggestion RemediationSuggestion) bool {
	switch suggestion.Type {
	case "environment":
		return o.applyEnvironmentRemediation(test, suggestion)
	case "resource":
		return o.applyResourceRemediation(test, suggestion)
	default:
		// Other types require manual intervention
		return false
	}
}

// applyEnvironmentRemediation applies environment-related remediations
func (o *TestStabilityOptimizer) applyEnvironmentRemediation(test TestReliabilityMetrics, suggestion RemediationSuggestion) bool {
	// Identify problematic environments
	var problematicEnvs []string
	for env, failureRate := range test.EnvironmentImpact {
		if failureRate > 0.5 {
			problematicEnvs = append(problematicEnvs, env)
		}
	}

	if len(problematicEnvs) == 0 {
		return false
	}

	// Apply environment optimizations
	for _, env := range problematicEnvs {
		success := o.environmentOptimizer.OptimizeEnvironment(env, test.TestName, test.TestSuite)
		if !success {
			return false
		}
	}

	return true
}

// applyResourceRemediation applies resource-related remediations
func (o *TestStabilityOptimizer) applyResourceRemediation(test TestReliabilityMetrics, suggestion RemediationSuggestion) bool {
	// Adjust resource limits based on test performance patterns
	actionLower := strings.ToLower(suggestion.Action)

	if strings.Contains(actionLower, "memory") {
		return o.adjustMemoryLimits(test)
	}

	if strings.Contains(actionLower, "timeout") {
		return o.adjustTimeoutSettings(test)
	}

	if strings.Contains(actionLower, "connection") {
		return o.adjustConnectionSettings(test)
	}

	return false
}

// adjustMemoryLimits adjusts memory limits for test execution
func (o *TestStabilityOptimizer) adjustMemoryLimits(test TestReliabilityMetrics) bool {
	// Calculate recommended memory limit based on test patterns
	baseMemory := 512 // MB
	
	// Increase memory for tests with high duration variance
	if test.DurationVariance > 1000000000 { // High variance in nanoseconds
		baseMemory *= 2
	}

	// Store the adjustment
	query := `
		INSERT INTO test_environment_adjustments (test_name, test_suite, adjustment_type, adjustment_value, applied_at)
		VALUES ($1, $2, 'memory_limit', $3, NOW())
	`

	_, err := o.db.Exec(query, test.TestName, test.TestSuite, baseMemory)
	return err == nil
}

// adjustTimeoutSettings adjusts timeout settings for test execution
func (o *TestStabilityOptimizer) adjustTimeoutSettings(test TestReliabilityMetrics) bool {
	// Calculate recommended timeout based on average duration
	recommendedTimeout := test.AverageDuration * 3 // 3x average duration

	// Minimum timeout of 30 seconds
	if recommendedTimeout < 30*time.Second {
		recommendedTimeout = 30 * time.Second
	}

	// Maximum timeout of 10 minutes
	if recommendedTimeout > 10*time.Minute {
		recommendedTimeout = 10 * time.Minute
	}

	query := `
		INSERT INTO test_environment_adjustments (test_name, test_suite, adjustment_type, adjustment_value, applied_at)
		VALUES ($1, $2, 'timeout', $3, NOW())
	`

	_, err := o.db.Exec(query, test.TestName, test.TestSuite, int64(recommendedTimeout.Seconds()))
	return err == nil
}

// adjustConnectionSettings adjusts connection pool settings
func (o *TestStabilityOptimizer) adjustConnectionSettings(test TestReliabilityMetrics) bool {
	// Increase connection pool size for tests with connection issues
	query := `
		INSERT INTO test_environment_adjustments (test_name, test_suite, adjustment_type, adjustment_value, applied_at)
		VALUES ($1, $2, 'connection_pool_size', $3, NOW())
	`

	_, err := o.db.Exec(query, test.TestName, test.TestSuite, 20) // Increase to 20 connections
	return err == nil
}

// optimizeTestEnvironments optimizes test execution environments
func (o *TestStabilityOptimizer) optimizeTestEnvironments() ([]OptimizationAction, error) {
	return o.environmentOptimizer.OptimizeAllEnvironments()
}

// generateStabilityRecommendations generates high-level stability recommendations
func (o *TestStabilityOptimizer) generateStabilityRecommendations(unstableTests []TestReliabilityMetrics) []StabilityRecommendation {
	var recommendations []StabilityRecommendation

	if len(unstableTests) == 0 {
		return recommendations
	}

	// Analyze common patterns across unstable tests
	commonPatterns := o.analyzeCommonPatterns(unstableTests)

	// Generate recommendations based on patterns
	for pattern, count := range commonPatterns {
		if count >= 3 { // Pattern appears in at least 3 tests
			recommendation := o.generatePatternRecommendation(pattern, count, len(unstableTests))
			if recommendation != nil {
				recommendations = append(recommendations, *recommendation)
			}
		}
	}

	// Add general recommendations
	generalRecommendations := o.generateGeneralRecommendations(unstableTests)
	recommendations = append(recommendations, generalRecommendations...)

	// Sort by priority
	sort.Slice(recommendations, func(i, j int) bool {
		priorityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
		return priorityOrder[recommendations[i].Priority] > priorityOrder[recommendations[j].Priority]
	})

	return recommendations
}

// analyzeCommonPatterns analyzes common failure patterns across tests
func (o *TestStabilityOptimizer) analyzeCommonPatterns(tests []TestReliabilityMetrics) map[string]int {
	patterns := make(map[string]int)

	for _, test := range tests {
		for _, pattern := range test.FailurePatterns {
			patterns[pattern.Type]++
		}

		// Analyze environment patterns
		for env, failureRate := range test.EnvironmentImpact {
			if failureRate > 0.5 {
				patterns["environment_"+env]++
			}
		}

		// Analyze stability trends
		if test.StabilityTrend == "degrading" {
			patterns["degrading_trend"]++
		}
	}

	return patterns
}

// generatePatternRecommendation generates a recommendation based on a common pattern
func (o *TestStabilityOptimizer) generatePatternRecommendation(pattern string, count, totalTests int) *StabilityRecommendation {
	percentage := float64(count) / float64(totalTests) * 100

	switch {
	case strings.HasPrefix(pattern, "environment_"):
		env := strings.TrimPrefix(pattern, "environment_")
		return &StabilityRecommendation{
			Type:        "infrastructure",
			Priority:    "high",
			Title:       fmt.Sprintf("Environment Issues in %s", env),
			Description: fmt.Sprintf("%.1f%% of unstable tests have issues in %s environment", percentage, env),
			Action:      fmt.Sprintf("Review and optimize %s environment configuration", env),
			Impact:      "high",
			Effort:      "medium",
		}

	case pattern == "intermittent":
		return &StabilityRecommendation{
			Type:        "code_quality",
			Priority:    "high",
			Title:       "Widespread Intermittent Failures",
			Description: fmt.Sprintf("%.1f%% of unstable tests show intermittent failure patterns", percentage),
			Action:      "Implement comprehensive race condition detection and fix timing issues",
			Impact:      "high",
			Effort:      "high",
		}

	case pattern == "timing":
		return &StabilityRecommendation{
			Type:        "performance",
			Priority:    "medium",
			Title:       "Timing-Related Issues",
			Description: fmt.Sprintf("%.1f%% of unstable tests have timing-related failures", percentage),
			Action:      "Review and optimize test execution scheduling and resource allocation",
			Impact:      "medium",
			Effort:      "medium",
		}

	case pattern == "degrading_trend":
		return &StabilityRecommendation{
			Type:        "monitoring",
			Priority:    "critical",
			Title:       "Degrading Test Stability Trend",
			Description: fmt.Sprintf("%.1f%% of unstable tests show degrading stability trends", percentage),
			Action:      "Immediate investigation required - system stability is declining",
			Impact:      "critical",
			Effort:      "high",
		}
	}

	return nil
}

// generateGeneralRecommendations generates general stability recommendations
func (o *TestStabilityOptimizer) generateGeneralRecommendations(tests []TestReliabilityMetrics) []StabilityRecommendation {
	var recommendations []StabilityRecommendation

	// Calculate overall metrics
	totalTests := len(tests)
	highFlakinessTests := 0
	lowReliabilityTests := 0

	for _, test := range tests {
		if test.FlakinessScore > 0.5 {
			highFlakinessTests++
		}
		if test.ReliabilityScore < 0.7 {
			lowReliabilityTests++
		}
	}

	// High flakiness recommendation
	if float64(highFlakinessTests)/float64(totalTests) > 0.3 {
		recommendations = append(recommendations, StabilityRecommendation{
			Type:        "process",
			Priority:    "high",
			Title:       "High Overall Test Flakiness",
			Description: fmt.Sprintf("%d out of %d tests have high flakiness scores", highFlakinessTests, totalTests),
			Action:      "Implement systematic flaky test remediation process",
			Impact:      "high",
			Effort:      "high",
		})
	}

	// Low reliability recommendation
	if float64(lowReliabilityTests)/float64(totalTests) > 0.4 {
		recommendations = append(recommendations, StabilityRecommendation{
			Type:        "quality",
			Priority:    "critical",
			Title:       "Low Overall Test Reliability",
			Description: fmt.Sprintf("%d out of %d tests have low reliability scores", lowReliabilityTests, totalTests),
			Action:      "Comprehensive test suite review and quality improvement initiative required",
			Impact:      "critical",
			Effort:      "high",
		})
	}

	return recommendations
}

// Helper methods

func (o *TestStabilityOptimizer) quarantineTest(test TestReliabilityMetrics, reason string) error {
	return o.reliabilityTracker.QuarantineTest(test.TestName, test.TestSuite, reason)
}

func (o *TestStabilityOptimizer) hasRecentRemediationAttempt(testName, testSuite string) bool {
	query := `
		SELECT COUNT(*) FROM test_remediation_attempts
		WHERE test_name = $1 AND test_suite = $2
		  AND attempted_at >= $3
	`

	cutoff := time.Now().Add(-o.config.RemediationCooldown)
	var count int
	err := o.db.QueryRow(query, testName, testSuite, cutoff).Scan(&count)
	
	return err == nil && count > 0
}

func (o *TestStabilityOptimizer) recordRemediationAttempt(testName, testSuite, remediationType string) {
	query := `
		INSERT INTO test_remediation_attempts (test_name, test_suite, remediation_type, attempted_at)
		VALUES ($1, $2, $3, NOW())
	`

	_, err := o.db.Exec(query, testName, testSuite, remediationType)
	if err != nil {
		log.Printf("Failed to record remediation attempt: %v", err)
	}
}

func (o *TestStabilityOptimizer) updateOptimizationMetrics(report *StabilityOptimizationReport) {
	report.Metrics["optimization_duration"] = report.Duration.Seconds()
	report.Metrics["tests_analyzed"] = report.UnstableTestsFound
	report.Metrics["optimizations_applied"] = len(report.Optimizations)
	report.Metrics["success_rate"] = o.calculateSuccessRate(report.Optimizations)
}

func (o *TestStabilityOptimizer) calculateSuccessRate(optimizations []OptimizationAction) float64 {
	if len(optimizations) == 0 {
		return 0.0
	}

	successful := 0
	for _, opt := range optimizations {
		if opt.Applied {
			successful++
		}
	}

	return float64(successful) / float64(len(optimizations))
}

func (o *TestStabilityOptimizer) sendOptimizationNotifications(report *StabilityOptimizationReport) {
	if report.TestsQuarantined > 0 {
		o.notificationManager.NotifyTestsQuarantined(report.TestsQuarantined)
	}

	if len(report.Recommendations) > 0 {
		highPriorityRecs := 0
		for _, rec := range report.Recommendations {
			if rec.Priority == "critical" || rec.Priority == "high" {
				highPriorityRecs++
			}
		}

		if highPriorityRecs > 0 {
			o.notificationManager.NotifyHighPriorityRecommendations(highPriorityRecs)
		}
	}
}

// Data structures for optimization reporting

type StabilityOptimizationReport struct {
	StartTime                 time.Time                    `json:"start_time"`
	EndTime                   time.Time                    `json:"end_time"`
	Duration                  time.Duration                `json:"duration"`
	UnstableTestsFound        int                          `json:"unstable_tests_found"`
	TestsQuarantined          int                          `json:"tests_quarantined"`
	EnvironmentOptimizations  int                          `json:"environment_optimizations"`
	Optimizations             []OptimizationAction         `json:"optimizations"`
	Recommendations           []StabilityRecommendation    `json:"recommendations"`
	Metrics                   map[string]interface{}       `json:"metrics"`
}

type OptimizationAction struct {
	Type        string                 `json:"type"`
	TestName    string                 `json:"test_name"`
	TestSuite   string                 `json:"test_suite"`
	Description string                 `json:"description"`
	Applied     bool                   `json:"applied"`
	Timestamp   time.Time              `json:"timestamp"`
	Details     interface{}            `json:"details,omitempty"`
}

type StabilityRecommendation struct {
	Type        string `json:"type"`        // "infrastructure", "code_quality", "process", etc.
	Priority    string `json:"priority"`    // "critical", "high", "medium", "low"
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Impact      string `json:"impact"`      // "critical", "high", "medium", "low"
	Effort      string `json:"effort"`      // "high", "medium", "low"
}