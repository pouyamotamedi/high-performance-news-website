package testing

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// AITestOptimizer handles AI-powered test execution optimization
type AITestOptimizer struct {
	llmClient         LLMClient
	executionAnalyzer *ExecutionAnalyzer
	failureAnalyzer   *FailureAnalyzer
	coverageAnalyzer  *CoverageAnalyzer
	debugAssistant    *DebugAssistant
	config           *OptimizerConfig
}

// OptimizerConfig configuration for test optimization
type OptimizerConfig struct {
	EnableIntelligentSelection bool          `json:"enable_intelligent_selection"`
	EnableParallelOptimization bool          `json:"enable_parallel_optimization"`
	EnableFailureAnalysis      bool          `json:"enable_failure_analysis"`
	EnableCoverageOptimization bool          `json:"enable_coverage_optimization"`
	MaxExecutionTime          time.Duration `json:"max_execution_time"`
	OptimizationInterval      time.Duration `json:"optimization_interval"`
	ConfidenceThreshold       float64       `json:"confidence_threshold"`
}

// ExecutionAnalyzer analyzes test execution patterns for optimization
type ExecutionAnalyzer struct {
	executionHistory []ExecutionRecord
	performanceMetrics map[string]PerformanceMetric
}

// FailureAnalyzer analyzes test failures using AI
type FailureAnalyzer struct {
	failurePatterns []FailurePattern
	rootCauseEngine *RootCauseEngine
}

// CoverageAnalyzer optimizes test coverag
// CoverageAnalyzer analyzes and optimizes test coverage
type CoverageAnalyzer struct {
	coverageData map[string]CoverageInfo
	optimizer    *CoverageOptimizer
}

// DebugAssistant provides AI-assisted debugging and root cause analysis
type DebugAssistant struct {
	llmClient LLMClient
	logAnalyzer *LogAnalyzer
}

// TestExecution represents a test execution record
type TestExecution struct {
	TestName      string        `json:"test_name"`
	TestFile      string        `json:"test_file"`
	Duration      time.Duration `json:"duration"`
	MemoryUsage   int64         `json:"memory_usage"`
	CPUUsage      float64       `json:"cpu_usage"`
	Success       bool          `json:"success"`
	ExecutedAt    time.Time     `json:"executed_at"`
	Dependencies  []string      `json:"dependencies"`
	CodeCoverage  float64       `json:"code_coverage"`
}

// TestFailure represents a test failure record
type TestFailure struct {
	TestName     string    `json:"test_name"`
	TestFile     string    `json:"test_file"`
	ErrorMessage string    `json:"error_message"`
	StackTrace   string    `json:"stack_trace"`
	FailedAt     time.Time `json:"failed_at"`
	FailureType  string    `json:"failure_type"`
	Context      map[string]interface{} `json:"context"`
}

// PerformanceMetrics represents performance metrics for a test
type PerformanceMetrics struct {
	AverageDuration   time.Duration `json:"average_duration"`
	MinDuration       time.Duration `json:"min_duration"`
	MaxDuration       time.Duration `json:"max_duration"`
	AverageMemory     int64         `json:"average_memory"`
	AverageCPU        float64       `json:"average_cpu"`
	ExecutionCount    int           `json:"execution_count"`
	SuccessRate       float64       `json:"success_rate"`
	LastOptimized     time.Time     `json:"last_optimized"`
}

// CoverageInfo represents coverage information for code
type CoverageInfo struct {
	FilePath        string            `json:"file_path"`
	TotalLines      int               `json:"total_lines"`
	CoveredLines    int               `json:"covered_lines"`
	CoveragePercent float64           `json:"coverage_percent"`
	UncoveredLines  []int             `json:"uncovered_lines"`
	TestsContributing []string        `json:"tests_contributing"`
}

// OptimizationRecommendation represents an AI recommendation for test optimization
type OptimizationRecommendation struct {
	ID              string                 `json:"id"`
	Type            OptimizationType       `json:"type"`
	Priority        Priority               `json:"priority"`
	TestName        string                 `json:"test_name"`
	TestFile        string                 `json:"test_file"`
	Description     string                 `json:"description"`
	Rationale       string                 `json:"rationale"`
	ExpectedBenefit string                 `json:"expected_benefit"`
	Implementation  string                 `json:"implementation"`
	Confidence      float64                `json:"confidence"`
	EstimatedSavings time.Duration         `json:"estimated_savings"`
	CreatedAt       time.Time              `json:"created_at"`
}

// OptimizationType defines types of optimization recommendations
type OptimizationType string

const (
	OptimizationParallelize    OptimizationType = "parallelize"
	OptimizationMemoize        OptimizationType = "memoize"
	OptimizationSkipRedundant  OptimizationType = "skip_redundant"
	OptimizationOptimizeSetup  OptimizationType = "optimize_setup"
	OptimizationCacheResults   OptimizationType = "cache_results"
	OptimizationReduceScope    OptimizationType = "reduce_scope"
	OptimizationMockExternal   OptimizationType = "mock_external"
)

// TestSelectionStrategy represents intelligent test selection
type TestSelectionStrategy struct {
	ChangedFiles     []string  `json:"changed_files"`
	ImpactedTests    []string  `json:"impacted_tests"`
	PriorityTests    []string  `json:"priority_tests"`
	SkippedTests     []string  `json:"skipped_tests"`
	SelectionReason  string    `json:"selection_reason"`
	EstimatedTime    time.Duration `json:"estimated_time"`
	CoverageImpact   float64   `json:"coverage_impact"`
}

// FailurePattern represents a detected failure pattern
type FailurePattern struct {
	ID          string    `json:"id"`
	Pattern     string    `json:"pattern"`
	Frequency   int       `json:"frequency"`
	Tests       []string  `json:"tests"`
	RootCause   string    `json:"root_cause"`
	Solution    string    `json:"solution"`
	Confidence  float64   `json:"confidence"`
	DetectedAt  time.Time `json:"detected_at"`
}

// CoverageOptimizer optimizes test coverage
type CoverageOptimizer struct {
	targetCoverage float64
	redundancyDetector *RedundancyDetector
}

// PatternDetector detects patterns in test failures
type PatternDetector struct {
	patterns []FailurePattern
}

// LogAnalyzer analyzes logs for debugging
type LogAnalyzer struct {
	logPatterns []LogPattern
}

// LogPattern represents a log pattern for analysis
type LogPattern struct {
	Pattern     string `json:"pattern"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// RedundancyDetector detects redundant test coverage
type RedundancyDetector struct {
	coverageMatrix map[string]map[string]bool
}

// NewAITestOptimizer creates a new AI test optimizer
func NewAITestOptimizer(llmClient LLMClient, config *OptimizerConfig) *AITestOptimizer {
	return &AITestOptimizer{
		llmClient:         llmClient,
		executionAnalyzer: NewExecutionAnalyzer(),
		failureAnalyzer:   NewFailureAnalyzer(),
		coverageAnalyzer:  NewCoverageAnalyzer(),
		debugAssistant:    NewDebugAssistant(llmClient),
		config:           config,
	}
}

// OptimizeTestExecution provides AI-powered test execution optimization
func (o *AITestOptimizer) OptimizeTestExecution(ctx context.Context, testSuite []string, changedFiles []string) (*TestSelectionStrategy, error) {
	// Analyze test execution history
	executionData := o.executionAnalyzer.AnalyzeExecutionHistory(testSuite)
	
	// Generate intelligent test selection
	selection, err := o.generateIntelligentSelection(ctx, testSuite, changedFiles, executionData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate intelligent selection: %w", err)
	}
	
	// Optimize test execution order
	optimizedOrder := o.optimizeExecutionOrder(selection.ImpactedTests, executionData)
	selection.ImpactedTests = optimizedOrder
	
	// Calculate parallel execution strategy
	if o.config.EnableParallelOptimization {
		parallelGroups := o.calculateParallelGroups(selection.ImpactedTests, executionData)
		selection.SelectionReason += fmt.Sprintf(" | Parallel groups: %d", len(parallelGroups))
	}
	
	return selection, nil
}

// generateIntelligentSelection generates intelligent test selection using AI
func (o *AITestOptimizer) generateIntelligentSelection(ctx context.Context, testSuite []string, changedFiles []string, executionData map[string]PerformanceMetrics) (*TestSelectionStrategy, error) {
	prompt := o.buildSelectionPrompt(testSuite, changedFiles, executionData)
	
	response, err := o.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate test selection: %w", err)
	}
	
	selection, err := o.parseSelectionResponse(response, testSuite, changedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to parse selection response: %w", err)
	}
	
	return selection, nil
}

// buildSelectionPrompt builds prompt for intelligent test selection
func (o *AITestOptimizer) buildSelectionPrompt(testSuite []string, changedFiles []string, executionData map[string]PerformanceMetrics) string {
	return fmt.Sprintf(`
Analyze the following test suite and changed files to recommend intelligent test selection:

Available Tests: %v
Changed Files: %v

Test Performance Data:
%s

Provide intelligent test selection recommendations:

1. IMPACTED TESTS: Tests that must run due to code changes
   - Direct impact: Tests that directly test changed code
   - Indirect impact: Tests that may be affected by changes
   - Integration impact: Tests that verify system integration

2. PRIORITY TESTS: Critical tests that should always run
   - Security-critical tests
   - Performance regression tests
   - Core functionality tests
   - Recently failing tests

3. SKIPPABLE TESTS: Tests that can be safely skipped
   - Tests unrelated to changes
   - Redundant coverage tests
   - Long-running tests with low impact
   - Flaky tests with low confidence

4. EXECUTION OPTIMIZATION:
   - Parallel execution opportunities
   - Test execution order optimization
   - Resource usage optimization
   - Time estimation

Consider:
- Test execution time and resource usage
- Code coverage impact
- Risk assessment
- Historical failure patterns
- Dependencies between tests

Provide rationale for each recommendation and estimated time savings.

Format as JSON with fields: impacted_tests, priority_tests, skipped_tests, selection_reason, estimated_time_minutes, coverage_impact_percent.
`, testSuite, changedFiles, o.formatPerformanceData(executionData))
}

// parseSelectionResponse parses AI response for test selection
func (o *AITestOptimizer) parseSelectionResponse(response string, testSuite []string, changedFiles []string) (*TestSelectionStrategy, error) {
	// For demonstration, return a sample selection strategy based on the response
	// In real implementation, would parse JSON response
	
	// Extract impacted tests based on changed files
	impactedTests := o.selectImpactedTests(testSuite, changedFiles)
	if len(impactedTests) == 0 && len(testSuite) > 0 {
		// Ensure we have at least some impacted tests for testing
		impactedTests = testSuite[:min(2, len(testSuite))]
	}
	
	strategy := &TestSelectionStrategy{
		ChangedFiles:    changedFiles,
		ImpactedTests:   impactedTests,
		PriorityTests:   o.selectPriorityTests(testSuite),
		SkippedTests:    o.selectSkippableTests(testSuite),
		SelectionReason: "AI-based selection considering code changes and test performance",
		EstimatedTime:   o.estimateExecutionTime(impactedTests),
		CoverageImpact:  85.0, // Estimated coverage percentage
	}
	
	return strategy, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AnalyzeTestFailures provides AI-powered failure analysis
func (o *AITestOptimizer) AnalyzeTestFailures(ctx context.Context, failures []TestFailure) ([]FailurePattern, error) {
	if !o.config.EnableFailureAnalysis {
		return nil, nil
	}
	
	// Detect patterns in failures
	patterns := o.failureAnalyzer.DetectPatterns(failures)
	
	// Enhance patterns with AI analysis
	enhancedPatterns, err := o.enhanceFailurePatternsWithAI(ctx, patterns, failures)
	if err != nil {
		return nil, fmt.Errorf("failed to enhance failure patterns: %w", err)
	}
	
	return enhancedPatterns, nil
}

// enhanceFailurePatternsWithAI enhances failure patterns using AI analysis
func (o *AITestOptimizer) enhanceFailurePatternsWithAI(ctx context.Context, patterns []FailurePattern, failures []TestFailure) ([]FailurePattern, error) {
	for i, pattern := range patterns {
		prompt := o.buildFailureAnalysisPrompt(pattern, failures)
		
		response, err := o.llmClient.GenerateText(ctx, prompt)
		if err != nil {
			log.Printf("Failed to analyze failure pattern %s: %v", pattern.ID, err)
			continue
		}
		
		enhancedPattern, err := o.parseFailureAnalysisResponse(response, pattern)
		if err != nil {
			log.Printf("Failed to parse failure analysis for pattern %s: %v", pattern.ID, err)
			continue
		}
		
		patterns[i] = enhancedPattern
	}
	
	return patterns, nil
}

// buildFailureAnalysisPrompt builds prompt for failure analysis
func (o *AITestOptimizer) buildFailureAnalysisPrompt(pattern FailurePattern, failures []TestFailure) string {
	return fmt.Sprintf(`
Analyze this test failure pattern to identify root causes and solutions:

Pattern: %s
Frequency: %d occurrences
Affected Tests: %v

Recent Failures:
%s

Provide detailed analysis:

1. ROOT CAUSE ANALYSIS:
   - Identify the underlying cause of failures
   - Consider timing issues, race conditions, dependencies
   - Analyze error messages and stack traces
   - Consider environmental factors

2. IMPACT ASSESSMENT:
   - Severity of the issue
   - Affected functionality
   - Risk to system stability
   - Development team impact

3. SOLUTION RECOMMENDATIONS:
   - Immediate fixes to resolve failures
   - Long-term improvements to prevent recurrence
   - Test improvements to catch issues earlier
   - Monitoring and alerting enhancements

4. PREVENTION STRATEGIES:
   - Code review guidelines
   - Testing best practices
   - CI/CD pipeline improvements
   - Monitoring and observability

Provide actionable recommendations with implementation details.

Format as JSON with fields: root_cause, impact_assessment, immediate_solutions, long_term_solutions, prevention_strategies, confidence_score.
`, pattern.Pattern, pattern.Frequency, pattern.Tests, o.formatFailureData(failures))
}

// parseFailureAnalysisResponse parses AI failure analysis response
func (o *AITestOptimizer) parseFailureAnalysisResponse(response string, pattern FailurePattern) (FailurePattern, error) {
	// For demonstration, enhance the pattern with sample analysis
	// In real implementation, would parse JSON response
	
	pattern.RootCause = "Timing-dependent test failure due to race condition"
	pattern.Solution = "Add proper synchronization mechanisms and deterministic timing"
	pattern.Confidence = 0.85
	
	return pattern, nil
}

// ProvideDebugAssistance provides AI-assisted debugging and root cause analysis
func (o *AITestOptimizer) ProvideDebugAssistance(ctx context.Context, failure TestFailure, logs []string) (*DebugAnalysis, error) {
	prompt := o.buildDebugPrompt(failure, logs)
	
	response, err := o.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate debug analysis: %w", err)
	}
	
	analysis, err := o.parseDebugResponse(response, failure)
	if err != nil {
		return nil, fmt.Errorf("failed to parse debug response: %w", err)
	}
	
	return analysis, nil
}

// DebugAnalysis represents AI-powered debug analysis
type DebugAnalysis struct {
	TestName        string            `json:"test_name"`
	RootCause       string            `json:"root_cause"`
	ErrorExplanation string           `json:"error_explanation"`
	FixSuggestions  []FixSuggestion   `json:"fix_suggestions"`
	RelatedIssues   []string          `json:"related_issues"`
	Confidence      float64           `json:"confidence"`
	DebuggingSteps  []DebuggingStep   `json:"debugging_steps"`
	GeneratedAt     time.Time         `json:"generated_at"`
}

// FixSuggestion represents a suggested fix
type FixSuggestion struct {
	Description string  `json:"description"`
	Code        string  `json:"code"`
	Priority    string  `json:"priority"`
	Confidence  float64 `json:"confidence"`
}

// DebuggingStep represents a debugging step
type DebuggingStep struct {
	Step        int    `json:"step"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Expected    string `json:"expected"`
}

// buildDebugPrompt builds prompt for debug assistance
func (o *AITestOptimizer) buildDebugPrompt(failure TestFailure, logs []string) string {
	return fmt.Sprintf(`
Provide comprehensive debugging assistance for this test failure:

Test: %s
Error: %s
Stack Trace:
%s

Related Logs:
%s

Provide detailed debugging analysis:

1. ROOT CAUSE ANALYSIS:
   - Analyze the error message and stack trace
   - Identify the specific line and function causing the failure
   - Determine if it's a code issue, test issue, or environmental issue
   - Consider timing, dependencies, and state issues

2. ERROR EXPLANATION:
   - Explain what the error means in plain language
   - Describe why this error occurred
   - Explain the impact and consequences

3. FIX SUGGESTIONS:
   - Provide specific code fixes with examples
   - Suggest test improvements
   - Recommend configuration changes
   - Prioritize fixes by impact and effort

4. DEBUGGING STEPS:
   - Step-by-step debugging procedure
   - Commands to run for investigation
   - What to look for in outputs
   - How to verify the fix

5. PREVENTION:
   - How to prevent similar issues in the future
   - Code review guidelines
   - Testing improvements
   - Monitoring recommendations

Provide actionable, specific recommendations with code examples.

Format as JSON with detailed analysis and recommendations.
`, failure.TestName, failure.ErrorMessage, failure.StackTrace, strings.Join(logs, "\n"))
}

// parseDebugResponse parses AI debug response
func (o *AITestOptimizer) parseDebugResponse(response string, failure TestFailure) (*DebugAnalysis, error) {
	// For demonstration, return sample debug analysis
	// In real implementation, would parse JSON response
	
	analysis := &DebugAnalysis{
		TestName:         failure.TestName,
		RootCause:        "Race condition in concurrent test execution",
		ErrorExplanation: "The test is failing due to timing-dependent behavior that creates non-deterministic results",
		FixSuggestions: []FixSuggestion{
			{
				Description: "Add proper synchronization using channels",
				Code:        "done := make(chan bool)\ngo func() { /* async operation */ done <- true }()\n<-done",
				Priority:    "high",
				Confidence:  0.9,
			},
		},
		RelatedIssues:  []string{"Similar timing issues in other tests"},
		Confidence:     0.85,
		DebuggingSteps: []DebuggingStep{
			{
				Step:        1,
				Description: "Run the test multiple times to confirm flakiness",
				Command:     "go test -run TestName -count=10",
				Expected:    "Should show intermittent failures",
			},
		},
		GeneratedAt: time.Now(),
	}
	
	return analysis, nil
}

// OptimizeTestCoverage provides AI-powered test coverage optimization
func (o *AITestOptimizer) OptimizeTestCoverage(ctx context.Context, coverageData map[string]CoverageInfo) ([]OptimizationRecommendation, error) {
	if !o.config.EnableCoverageOptimization {
		return nil, nil
	}
	
	// Analyze coverage gaps and redundancies
	gaps := o.coverageAnalyzer.IdentifyCoverageGaps(coverageData)
	redundancies := o.coverageAnalyzer.IdentifyRedundantCoverage(coverageData)
	
	// Generate AI-powered optimization recommendations
	recommendations, err := o.generateCoverageOptimizations(ctx, gaps, redundancies, coverageData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate coverage optimizations: %w", err)
	}
	
	return recommendations, nil
}

// generateCoverageOptimizations generates coverage optimization recommendations
func (o *AITestOptimizer) generateCoverageOptimizations(ctx context.Context, gaps []CoverageGap, redundancies []CoverageRedundancy, coverageData map[string]CoverageInfo) ([]OptimizationRecommendation, error) {
	prompt := o.buildCoverageOptimizationPrompt(gaps, redundancies, coverageData)
	
	response, err := o.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate coverage optimizations: %w", err)
	}
	
	recommendations, err := o.parseCoverageOptimizationResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse coverage optimizations: %w", err)
	}
	
	return recommendations, nil
}

// CoverageGap represents a gap in test coverage
type CoverageGap struct {
	FilePath      string `json:"file_path"`
	UncoveredLines []int `json:"uncovered_lines"`
	Severity      string `json:"severity"`
	Function      string `json:"function"`
}

// CoverageRedundancy represents redundant test coverage
type CoverageRedundancy struct {
	FilePath     string   `json:"file_path"`
	Lines        []int    `json:"lines"`
	Tests        []string `json:"tests"`
	Redundancy   float64  `json:"redundancy"`
}

// buildCoverageOptimizationPrompt builds prompt for coverage optimization
func (o *AITestOptimizer) buildCoverageOptimizationPrompt(gaps []CoverageGap, redundancies []CoverageRedundancy, coverageData map[string]CoverageInfo) string {
	return fmt.Sprintf(`
Analyze test coverage data and provide optimization recommendations:

Coverage Gaps:
%v

Coverage Redundancies:
%v

Overall Coverage Data:
%s

Provide coverage optimization recommendations:

1. COVERAGE GAPS:
   - Identify critical uncovered code paths
   - Prioritize gaps by risk and importance
   - Suggest specific tests to add
   - Recommend test strategies for complex code

2. REDUNDANT COVERAGE:
   - Identify tests that cover the same code
   - Suggest test consolidation opportunities
   - Recommend test removal or refactoring
   - Maintain adequate coverage while reducing redundancy

3. OPTIMIZATION STRATEGIES:
   - Improve test efficiency
   - Reduce test execution time
   - Optimize test data and setup
   - Enhance test maintainability

4. QUALITY IMPROVEMENTS:
   - Improve test assertions
   - Add edge case coverage
   - Enhance error condition testing
   - Improve integration test coverage

Provide specific, actionable recommendations with implementation details.

Format as JSON array of optimization recommendations.
`, gaps, redundancies, o.formatCoverageData(coverageData))
}

// parseCoverageOptimizationResponse parses coverage optimization response
func (o *AITestOptimizer) parseCoverageOptimizationResponse(response string) ([]OptimizationRecommendation, error) {
	// For demonstration, return sample recommendations
	// In real implementation, would parse JSON response
	
	recommendations := []OptimizationRecommendation{
		{
			ID:              generateOptimizationID(),
			Type:            OptimizationReduceScope,
			Priority:        PriorityMedium,
			TestName:        "TestArticleRepository",
			Description:     "Reduce redundant coverage in article repository tests",
			Rationale:       "Multiple tests cover the same code paths with minimal variation",
			ExpectedBenefit: "Reduce test execution time by 30%",
			Implementation:  "Consolidate similar test cases and use parameterized tests",
			Confidence:      0.8,
			EstimatedSavings: 2 * time.Minute,
			CreatedAt:       time.Now(),
		},
	}
	
	return recommendations, nil
}

// Helper methods and implementations

func (o *AITestOptimizer) selectImpactedTests(testSuite []string, changedFiles []string) []string {
	// Simplified implementation - in real version would analyze dependencies
	var impacted []string
	for _, test := range testSuite {
		if o.isTestImpacted(test, changedFiles) {
			impacted = append(impacted, test)
		}
	}
	
	// Ensure we return at least some tests if there are changed files
	if len(impacted) == 0 && len(changedFiles) > 0 && len(testSuite) > 0 {
		// Return first few tests as potentially impacted
		for i := 0; i < min(2, len(testSuite)); i++ {
			impacted = append(impacted, testSuite[i])
		}
	}
	
	return impacted
}

func (o *AITestOptimizer) selectPriorityTests(testSuite []string) []string {
	// Return tests that should always run (security, critical functionality)
	var priority []string
	for _, test := range testSuite {
		if strings.Contains(test, "Security") || strings.Contains(test, "Auth") {
			priority = append(priority, test)
		}
	}
	return priority
}

func (o *AITestOptimizer) selectSkippableTests(testSuite []string) []string {
	// Return tests that can be skipped (long-running, low impact)
	var skippable []string
	for _, test := range testSuite {
		if strings.Contains(test, "Integration") && len(testSuite) > 10 {
			skippable = append(skippable, test)
		}
	}
	return skippable
}

func (o *AITestOptimizer) isTestImpacted(test string, changedFiles []string) bool {
	// Simplified logic - in real implementation would analyze dependencies
	for _, file := range changedFiles {
		// Extract component name from file path
		fileName := strings.TrimSuffix(file, ".go")
		if strings.Contains(fileName, "/") {
			parts := strings.Split(fileName, "/")
			fileName = parts[len(parts)-1]
		}
		
		// Check if test name contains the component name
		if strings.Contains(strings.ToLower(test), strings.ToLower(fileName)) {
			return true
		}
		
		// Check for common patterns
		if strings.Contains(file, "article") && strings.Contains(strings.ToLower(test), "article") {
			return true
		}
		if strings.Contains(file, "service") && strings.Contains(strings.ToLower(test), "service") {
			return true
		}
		if strings.Contains(file, "repository") && strings.Contains(strings.ToLower(test), "repository") {
			return true
		}
	}
	return false
}

func (o *AITestOptimizer) estimateExecutionTime(tests []string) time.Duration {
	// Simplified estimation - in real implementation would use historical data
	return time.Duration(len(tests)) * 30 * time.Second
}

func (o *AITestOptimizer) optimizeExecutionOrder(tests []string, executionData map[string]PerformanceMetrics) []string {
	// Sort tests by execution time (fastest first)
	sort.Slice(tests, func(i, j int) bool {
		dataI, okI := executionData[tests[i]]
		dataJ, okJ := executionData[tests[j]]
		
		if !okI && !okJ {
			return false
		}
		if !okI {
			return false
		}
		if !okJ {
			return true
		}
		
		return dataI.AverageDuration < dataJ.AverageDuration
	})
	
	return tests
}

func (o *AITestOptimizer) calculateParallelGroups(tests []string, executionData map[string]PerformanceMetrics) [][]string {
	// Group tests that can run in parallel
	var groups [][]string
	currentGroup := []string{}
	
	for _, test := range tests {
		// Simple grouping logic - in real implementation would analyze dependencies
		if len(currentGroup) < 3 {
			currentGroup = append(currentGroup, test)
		} else {
			groups = append(groups, currentGroup)
			currentGroup = []string{test}
		}
	}
	
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}
	
	return groups
}

func (o *AITestOptimizer) formatPerformanceData(data map[string]PerformanceMetrics) string {
	var result strings.Builder
	
	for testName, metrics := range data {
		result.WriteString(fmt.Sprintf("Test: %s\n", testName))
		result.WriteString(fmt.Sprintf("  Average Duration: %v\n", metrics.AverageDuration))
		result.WriteString(fmt.Sprintf("  Success Rate: %.2f%%\n", metrics.SuccessRate*100))
		result.WriteString(fmt.Sprintf("  Execution Count: %d\n", metrics.ExecutionCount))
		result.WriteString("\n")
	}
	
	return result.String()
}

func (o *AITestOptimizer) formatFailureData(failures []TestFailure) string {
	var result strings.Builder
	
	for _, failure := range failures {
		result.WriteString(fmt.Sprintf("Test: %s\n", failure.TestName))
		result.WriteString(fmt.Sprintf("Error: %s\n", failure.ErrorMessage))
		result.WriteString(fmt.Sprintf("Failed At: %v\n", failure.FailedAt))
		result.WriteString("\n")
	}
	
	return result.String()
}

func (o *AITestOptimizer) formatCoverageData(data map[string]CoverageInfo) string {
	var result strings.Builder
	
	for filePath, coverage := range data {
		result.WriteString(fmt.Sprintf("File: %s\n", filePath))
		result.WriteString(fmt.Sprintf("  Coverage: %.2f%%\n", coverage.CoveragePercent))
		result.WriteString(fmt.Sprintf("  Lines: %d/%d\n", coverage.CoveredLines, coverage.TotalLines))
		result.WriteString("\n")
	}
	
	return result.String()
}

func generateOptimizationID() string {
	return fmt.Sprintf("opt-%d", time.Now().UnixNano())
}

// Additional helper types and functions

type ExecutionRecord struct {
	TestName    string        `json:"test_name"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
	ExecutedAt  time.Time     `json:"executed_at"`
	MemoryUsage int64         `json:"memory_usage"`
	CPUUsage    float64       `json:"cpu_usage"`
}

type PerformanceMetric struct {
	Name      string      `json:"name"`
	Value     interface{} `json:"value"`
	Threshold interface{} `json:"threshold"`
	Status    string      `json:"status"`
}

type RootCauseEngine struct {
	patterns []FailurePattern
}

// Constructor functions

func NewExecutionAnalyzer() *ExecutionAnalyzer {
	return &ExecutionAnalyzer{
		executionHistory:   make([]ExecutionRecord, 0),
		performanceMetrics: make(map[string]PerformanceMetric),
	}
}

func NewFailureAnalyzer() *FailureAnalyzer {
	return &FailureAnalyzer{
		failurePatterns: make([]FailurePattern, 0),
		rootCauseEngine: &RootCauseEngine{},
	}
}

func NewCoverageAnalyzer() *CoverageAnalyzer {
	return &CoverageAnalyzer{
		coverageData: make(map[string]CoverageInfo),
		optimizer:    &CoverageOptimizer{targetCoverage: 0.95},
	}
}

func NewDebugAssistant(llmClient LLMClient) *DebugAssistant {
	return &DebugAssistant{
		llmClient:   llmClient,
		logAnalyzer: &LogAnalyzer{},
	}
}

// ExecutionAnalyzer methods

func (e *ExecutionAnalyzer) AnalyzeExecutionHistory(testSuite []string) map[string]PerformanceMetrics {
	metrics := make(map[string]PerformanceMetrics)
	
	for _, testName := range testSuite {
		// Calculate metrics from execution history
		testRecords := e.getTestRecords(testName)
		if len(testRecords) == 0 {
			continue
		}
		
		var totalDuration time.Duration
		var successCount int
		var totalMemory int64
		var totalCPU float64
		
		for _, record := range testRecords {
			totalDuration += record.Duration
			totalMemory += record.MemoryUsage
			totalCPU += record.CPUUsage
			if record.Success {
				successCount++
			}
		}
		
		count := len(testRecords)
		metrics[testName] = PerformanceMetrics{
			AverageDuration: totalDuration / time.Duration(count),
			AverageMemory:   totalMemory / int64(count),
			AverageCPU:      totalCPU / float64(count),
			ExecutionCount:  count,
			SuccessRate:     float64(successCount) / float64(count),
			LastOptimized:   time.Now(),
		}
	}
	
	return metrics
}

func (e *ExecutionAnalyzer) getTestRecords(testName string) []ExecutionRecord {
	var records []ExecutionRecord
	
	for _, record := range e.executionHistory {
		if record.TestName == testName {
			records = append(records, record)
		}
	}
	
	return records
}

// FailureAnalyzer methods

func (f *FailureAnalyzer) DetectPatterns(failures []TestFailure) []FailurePattern {
	patternMap := make(map[string]*FailurePattern)
	
	for _, failure := range failures {
		// Simple pattern detection based on error message
		pattern := f.extractPattern(failure.ErrorMessage)
		
		if existing, exists := patternMap[pattern]; exists {
			existing.Frequency++
			existing.Tests = append(existing.Tests, failure.TestName)
		} else {
			patternMap[pattern] = &FailurePattern{
				ID:         generatePatternID(),
				Pattern:    pattern,
				Frequency:  1,
				Tests:      []string{failure.TestName},
				DetectedAt: time.Now(),
			}
		}
	}
	
	var patterns []FailurePattern
	for _, pattern := range patternMap {
		patterns = append(patterns, *pattern)
	}
	
	return patterns
}

func (f *FailureAnalyzer) extractPattern(errorMessage string) string {
	// Simple pattern extraction - in real implementation would be more sophisticated
	if strings.Contains(errorMessage, "timeout") {
		return "timeout_error"
	}
	if strings.Contains(errorMessage, "connection") {
		return "connection_error"
	}
	if strings.Contains(errorMessage, "nil pointer") {
		return "nil_pointer_error"
	}
	return "generic_error"
}

// CoverageAnalyzer methods

func (c *CoverageAnalyzer) IdentifyCoverageGaps(coverageData map[string]CoverageInfo) []CoverageGap {
	var gaps []CoverageGap
	
	for filePath, coverage := range coverageData {
		if coverage.CoveragePercent < 90.0 { // Less than 90% coverage
			gap := CoverageGap{
				FilePath:       filePath,
				UncoveredLines: coverage.UncoveredLines,
				Severity:       "medium",
			}
			
			if coverage.CoveragePercent < 70.0 {
				gap.Severity = "high"
			}
			
			gaps = append(gaps, gap)
		}
	}
	
	return gaps
}

func (c *CoverageAnalyzer) IdentifyRedundantCoverage(coverageData map[string]CoverageInfo) []CoverageRedundancy {
	var redundancies []CoverageRedundancy
	
	// Simple redundancy detection - in real implementation would be more sophisticated
	for filePath, coverage := range coverageData {
		if len(coverage.TestsContributing) > 5 { // More than 5 tests covering same lines
			redundancy := CoverageRedundancy{
				FilePath:   filePath,
				Tests:      coverage.TestsContributing,
				Redundancy: float64(len(coverage.TestsContributing)) / float64(coverage.CoveredLines),
			}
			redundancies = append(redundancies, redundancy)
		}
	}
	
	return redundancies
}

func generatePatternID() string {
	return fmt.Sprintf("pattern-%d", time.Now().UnixNano())
}
