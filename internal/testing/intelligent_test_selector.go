package testing

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// IntelligentTestSelector provides intelligent test selection and prioritization
type IntelligentTestSelector struct {
	impactAnalyzer    *ImpactAnalyzer
	timePredictor     *ExecutionTimePredictor
	parallelOptimizer *ParallelExecutionOptimizer
	maintenanceAdvisor *TestMaintenanceAdvisor
	testRegistry      *TestRegistry
}

// TestSelectionRequest represents a request for test selection
type TestSelectionRequest struct {
	ChangedFiles     []string          `json:"changed_files"`
	TimeLimit        time.Duration     `json:"time_limit"`
	Priority         SelectionPriority `json:"priority"`
	ParallelWorkers  int              `json:"parallel_workers"`
	TargetCoverage   float64          `json:"target_coverage"`
	ExcludePatterns  []string         `json:"exclude_patterns"`
}

// TestSelectionResult contains the selected tests and execution plan
type TestSelectionResult struct {
	SelectedTests    []TestCase        `json:"selected_tests"`
	ExecutionPlan    *ExecutionPlan    `json:"execution_plan"`
	EstimatedTime    time.Duration     `json:"estimated_time"`
	ExpectedCoverage float64          `json:"expected_coverage"`
	Reasoning        []string         `json:"reasoning"`
	Optimizations    []Optimization   `json:"optimizations"`
}

// SelectionPriority defines test selection priority levels
type SelectionPriority string

const (
	SelectionPriorityFast     SelectionPriority = "fast"     // Quick feedback, essential tests only
	SelectionPriorityBalanced SelectionPriority = "balanced" // Balance between speed and coverage
	SelectionPriorityComplete SelectionPriority = "complete" // Comprehensive testing
	SelectionPriorityCritical SelectionPriority = "critical" // Critical path tests only
)

// TestCaseMetadata represents extended metadata for test cases
type TestCaseMetadata struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Package         string            `json:"package"`
	FilePath        string            `json:"file_path"`
	Dependencies    []string          `json:"dependencies"`
	Tags            []string          `json:"tags"`
	EstimatedTime   time.Duration     `json:"estimated_time"`
	HistoricalTime  time.Duration     `json:"historical_time"`
	SuccessRate     float64          `json:"success_rate"`
	Impact          ImpactScore      `json:"impact"`
	LastRun         time.Time        `json:"last_run"`
	Flakiness       float64          `json:"flakiness"`
	ResourceUsage   ResourceUsage    `json:"resource_usage"`
	CoverageData    CoverageData     `json:"coverage_data"`
}

// ExecutionPlan defines how tests should be executed
type ExecutionPlan struct {
	Phases          []ExecutionPhase  `json:"phases"`
	ParallelGroups  []ParallelGroup   `json:"parallel_groups"`
	TotalEstimated  time.Duration     `json:"total_estimated"`
	ResourceNeeds   ResourceNeeds     `json:"resource_needs"`
	Dependencies    []Dependency      `json:"dependencies"`
}

// ExecutionPhase represents a phase of test execution
type ExecutionPhase struct {
	Name        string        `json:"name"`
	Tests       []TestCase    `json:"tests"`
	Parallel    bool          `json:"parallel"`
	MaxWorkers  int           `json:"max_workers"`
	Timeout     time.Duration `json:"timeout"`
	Priority    int           `json:"priority"`
}

// ParallelGroup represents tests that can run in parallel
type ParallelGroup struct {
	ID          string        `json:"id"`
	Tests       []TestCase    `json:"tests"`
	MaxWorkers  int           `json:"max_workers"`
	Resources   ResourceNeeds `json:"resources"`
	Estimated   time.Duration `json:"estimated"`
}

// NewIntelligentTestSelector creates a new intelligent test selector
func NewIntelligentTestSelector() *IntelligentTestSelector {
	return &IntelligentTestSelector{
		impactAnalyzer:     NewImpactAnalyzer(),
		timePredictor:      NewExecutionTimePredictor(),
		parallelOptimizer:  NewParallelExecutionOptimizer(),
		maintenanceAdvisor: NewTestMaintenanceAdvisor(),
		testRegistry:       NewTestRegistry(),
	}
}

// SelectTests intelligently selects and prioritizes tests based on the request
func (s *IntelligentTestSelector) SelectTests(ctx context.Context, request TestSelectionRequest) (*TestSelectionResult, error) {
	log.Printf("Starting intelligent test selection for %d changed files", len(request.ChangedFiles))
	
	// Step 1: Analyze impact of code changes
	impactAnalysis, err := s.impactAnalyzer.AnalyzeImpact(ctx, request.ChangedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze impact: %w", err)
	}
	
	// Step 2: Get all available tests
	allTests, err := s.testRegistry.GetAllTests(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get test registry: %w", err)
	}
	
	// Step 3: Filter tests based on impact and priority
	candidateTests := s.filterTestsByImpact(allTests, impactAnalysis, request.Priority)
	
	// Step 4: Predict execution times
	for i := range candidateTests {
		prediction, err := s.timePredictor.PredictExecutionTime(ctx, &candidateTests[i])
		if err != nil {
			log.Printf("Warning: failed to predict time for test %s: %v", candidateTests[i].ID, err)
			continue
		}
		candidateTests[i].EstimatedTime = prediction.EstimatedTime
	}
	
	// Step 5: Select optimal test set within time constraints
	selectedTests := s.selectOptimalTestSet(candidateTests, request)
	
	// Step 6: Create execution plan with parallel optimization
	executionPlan, err := s.parallelOptimizer.OptimizeExecution(ctx, selectedTests, request.ParallelWorkers)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize execution: %w", err)
	}
	
	// Step 7: Calculate metrics and reasoning
	result := &TestSelectionResult{
		SelectedTests:    selectedTests,
		ExecutionPlan:    executionPlan,
		EstimatedTime:    executionPlan.TotalEstimated,
		ExpectedCoverage: s.calculateExpectedCoverage(selectedTests),
		Reasoning:        s.generateReasoning(impactAnalysis, selectedTests, request),
		Optimizations:    s.identifyOptimizations(selectedTests, executionPlan),
	}
	
	log.Printf("Selected %d tests with estimated time %v", len(selectedTests), result.EstimatedTime)
	return result, nil
}

// filterTestsByImpact filters tests based on impact analysis and priority
func (s *IntelligentTestSelector) filterTestsByImpact(allTests []TestCase, impact *ImpactAnalysis, priority SelectionPriority) []TestCase {
	var filtered []TestCase
	
	for _, test := range allTests {
		// Check if test is affected by changes
		if !s.isTestAffected(test, impact) && priority != PriorityComplete {
			continue
		}
		
		// Apply priority-based filtering
		switch priority {
		case PriorityFast:
			// Only include fast, high-impact tests
			if test.EstimatedTime > 30*time.Second || test.Impact.Overall < 0.7 {
				continue
			}
		case PriorityCritical:
			// Only include critical path tests
			if !s.isCriticalTest(test) {
				continue
			}
		case PriorityBalanced:
			// Include tests with reasonable time and good impact
			if test.EstimatedTime > 5*time.Minute && test.Impact.Overall < 0.5 {
				continue
			}
		case PriorityComplete:
			// Include all tests (already handled above)
		}
		
		// Exclude flaky tests unless they're critical
		if test.Flakiness > 0.3 && !s.isCriticalTest(test) {
			continue
		}
		
		filtered = append(filtered, test)
	}
	
	return filtered
}

// selectOptimalTestSet selects the optimal set of tests within constraints
func (s *IntelligentTestSelector) selectOptimalTestSet(candidates []TestCase, request TestSelectionRequest) []TestCase {
	// Sort by priority score (impact/time ratio)
	sort.Slice(candidates, func(i, j int) bool {
		scoreI := s.calculatePriorityScore(candidates[i])
		scoreJ := s.calculatePriorityScore(candidates[j])
		return scoreI > scoreJ
	})
	
	var selected []TestCase
	var totalTime time.Duration
	coverageMap := make(map[string]bool)
	
	for _, test := range candidates {
		// Check time constraint
		if request.TimeLimit > 0 && totalTime+test.EstimatedTime > request.TimeLimit {
			continue
		}
		
		// Check if test adds significant coverage
		if s.addsCoverage(test, coverageMap) || s.isCriticalTest(test) {
			selected = append(selected, test)
			totalTime += test.EstimatedTime
			
			// Update coverage tracking
			for _, file := range test.CoverageData.Files {
				coverageMap[file] = true
			}
		}
		
		// Stop if we've reached target coverage
		if request.TargetCoverage > 0 && s.calculateCoverage(coverageMap) >= request.TargetCoverage {
			break
		}
	}
	
	return selected
}

// calculatePriorityScore calculates a priority score for test selection
func (s *IntelligentTestSelector) calculatePriorityScore(test TestCase) float64 {
	// Base score from impact
	score := test.Impact.Overall
	
	// Adjust for execution time (prefer faster tests)
	timeSeconds := test.EstimatedTime.Seconds()
	if timeSeconds > 0 {
		score = score / (1 + timeSeconds/60) // Normalize by minutes
	}
	
	// Boost critical tests
	if s.isCriticalTest(test) {
		score *= 2.0
	}
	
	// Penalize flaky tests
	score *= (1.0 - test.Flakiness)
	
	// Boost tests with high success rate
	score *= test.SuccessRate
	
	return score
}

// isTestAffected checks if a test is affected by the code changes
func (s *IntelligentTestSelector) isTestAffected(test TestCase, impact *ImpactAnalysis) bool {
	// Check direct file dependencies
	for _, dep := range test.Dependencies {
		if impact.AffectedFiles[dep] {
			return true
		}
	}
	
	// Check package-level impact
	testPackage := filepath.Dir(test.FilePath)
	for affectedFile := range impact.AffectedFiles {
		if strings.HasPrefix(affectedFile, testPackage) {
			return true
		}
	}
	
	return false
}

// isCriticalTest determines if a test is critical
func (s *IntelligentTestSelector) isCriticalTest(test TestCase) bool {
	criticalTags := []string{"critical", "smoke", "security", "performance"}
	
	for _, tag := range test.Tags {
		for _, critical := range criticalTags {
			if strings.Contains(strings.ToLower(tag), critical) {
				return true
			}
		}
	}
	
	// High impact tests are also considered critical
	return test.Impact.Overall > 0.8
}

// addsCoverage checks if a test adds significant coverage
func (s *IntelligentTestSelector) addsCoverage(test TestCase, existing map[string]bool) bool {
	newFiles := 0
	for _, file := range test.CoverageData.Files {
		if !existing[file] {
			newFiles++
		}
	}
	
	// Consider it adds coverage if it covers at least 1 new file
	// or if it's a critical test
	return newFiles > 0 || s.isCriticalTest(test)
}

// calculateCoverage calculates current coverage percentage
func (s *IntelligentTestSelector) calculateCoverage(coverageMap map[string]bool) float64 {
	// This is a simplified calculation
	// In practice, you'd want to calculate actual line/branch coverage
	return float64(len(coverageMap)) / 100.0 // Assume 100 total files for simplicity
}

// calculateExpectedCoverage calculates expected coverage from selected tests
func (s *IntelligentTestSelector) calculateExpectedCoverage(tests []TestCase) float64 {
	coverageMap := make(map[string]bool)
	
	for _, test := range tests {
		for _, file := range test.CoverageData.Files {
			coverageMap[file] = true
		}
	}
	
	return s.calculateCoverage(coverageMap)
}

// generateReasoning generates human-readable reasoning for test selection
func (s *IntelligentTestSelector) generateReasoning(impact *ImpactAnalysis, selected []TestCase, request TestSelectionRequest) []string {
	var reasoning []string
	
	reasoning = append(reasoning, fmt.Sprintf("Analyzed %d changed files for impact", len(request.ChangedFiles)))
	reasoning = append(reasoning, fmt.Sprintf("Selected %d tests based on %s priority", len(selected), request.Priority))
	
	if request.TimeLimit > 0 {
		reasoning = append(reasoning, fmt.Sprintf("Constrained by time limit of %v", request.TimeLimit))
	}
	
	criticalCount := 0
	for _, test := range selected {
		if s.isCriticalTest(test) {
			criticalCount++
		}
	}
	
	if criticalCount > 0 {
		reasoning = append(reasoning, fmt.Sprintf("Included %d critical tests", criticalCount))
	}
	
	reasoning = append(reasoning, fmt.Sprintf("Expected coverage: %.1f%%", s.calculateExpectedCoverage(selected)*100))
	
	return reasoning
}

// identifyOptimizations identifies potential optimizations
func (s *IntelligentTestSelector) identifyOptimizations(tests []TestCase, plan *ExecutionPlan) []Optimization {
	var optimizations []Optimization
	
	// Check for parallel execution opportunities
	if len(plan.ParallelGroups) > 1 {
		optimizations = append(optimizations, Optimization{
			Type:        "parallel_execution",
			Description: fmt.Sprintf("Executing %d test groups in parallel", len(plan.ParallelGroups)),
			Impact:      "Reduces total execution time by up to 60%",
		})
	}
	
	// Check for test consolidation opportunities
	packageGroups := make(map[string][]TestCase)
	for _, test := range tests {
		packageGroups[test.Package] = append(packageGroups[test.Package], test)
	}
	
	for pkg, pkgTests := range packageGroups {
		if len(pkgTests) > 5 {
			optimizations = append(optimizations, Optimization{
				Type:        "test_consolidation",
				Description: fmt.Sprintf("Package %s has %d tests that could be consolidated", pkg, len(pkgTests)),
				Impact:      "Could reduce setup/teardown overhead",
			})
		}
	}
	
	return optimizations
}

// Optimization represents a potential optimization
type Optimization struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}