package testing

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"
)

// Task24Implementation implements intelligent test selection and execution optimization
type Task24Implementation struct {
	impactAnalyzer    *Task24ImpactAnalyzer
	timePredictor     *Task24TimePredictor
	executionOptimizer *Task24ExecutionOptimizer
	maintenanceAdvisor *Task24MaintenanceAdvisor
}

// Task24TestInfo represents test information for task 24
type Task24TestInfo struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Package         string            `json:"package"`
	FilePath        string            `json:"file_path"`
	Dependencies    []string          `json:"dependencies"`
	Tags            []string          `json:"tags"`
	EstimatedTime   time.Duration     `json:"estimated_time"`
	SuccessRate     float64          `json:"success_rate"`
	ImpactScore     float64          `json:"impact_score"`
	Flakiness       float64          `json:"flakiness"`
	ResourceUsage   Task24ResourceUsage `json:"resource_usage"`
}

// Task24ResourceUsage represents resource usage for a test
type Task24ResourceUsage struct {
	CPU    float64 `json:"cpu"`    // CPU usage percentage
	Memory int64   `json:"memory"` // Memory usage in bytes
	IO     float64 `json:"io"`     // IO usage percentage
	Network bool   `json:"network"` // Whether test uses network
}

// Task24SelectionRequest represents a test selection request
type Task24SelectionRequest struct {
	ChangedFiles     []string      `json:"changed_files"`
	TimeLimit        time.Duration `json:"time_limit"`
	Priority         string        `json:"priority"` // "fast", "balanced", "complete", "critical"
	ParallelWorkers  int          `json:"parallel_workers"`
	TargetCoverage   float64      `json:"target_coverage"`
}

// Task24SelectionResult contains the test selection results
type Task24SelectionResult struct {
	SelectedTests    []Task24TestInfo     `json:"selected_tests"`
	ExecutionPlan    *Task24ExecutionPlan `json:"execution_plan"`
	EstimatedTime    time.Duration        `json:"estimated_time"`
	ExpectedCoverage float64             `json:"expected_coverage"`
	Reasoning        []string            `json:"reasoning"`
	Optimizations    []Task24Optimization `json:"optimizations"`
}

// Task24ExecutionPlan defines how tests should be executed
type Task24ExecutionPlan struct {
	Phases          []Task24ExecutionPhase `json:"phases"`
	ParallelGroups  []Task24ParallelGroup  `json:"parallel_groups"`
	TotalEstimated  time.Duration          `json:"total_estimated"`
	ResourceNeeds   Task24ResourceNeeds    `json:"resource_needs"`
}

// Task24ExecutionPhase represents a phase of test execution
type Task24ExecutionPhase struct {
	Name        string             `json:"name"`
	Tests       []Task24TestInfo   `json:"tests"`
	Parallel    bool              `json:"parallel"`
	MaxWorkers  int               `json:"max_workers"`
	Timeout     time.Duration     `json:"timeout"`
	Priority    int               `json:"priority"`
}

// Task24ParallelGroup represents tests that can run in parallel
type Task24ParallelGroup struct {
	ID          string              `json:"id"`
	Tests       []Task24TestInfo    `json:"tests"`
	MaxWorkers  int                `json:"max_workers"`
	Resources   Task24ResourceNeeds `json:"resources"`
	Estimated   time.Duration      `json:"estimated"`
}

// Task24ResourceNeeds represents resource requirements
type Task24ResourceNeeds struct {
	TotalCPU    float64 `json:"total_cpu"`
	TotalMemory int64   `json:"total_memory"`
	TotalIO     float64 `json:"total_io"`
	Workers     int     `json:"workers"`
}

// Task24Optimization represents an optimization recommendation
type Task24Optimization struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

// Task24ImpactAnalyzer analyzes code change impact
type Task24ImpactAnalyzer struct {
	dependencyMap map[string][]string
}

// Task24TimePredictor predicts test execution times
type Task24TimePredictor struct {
	historicalData map[string][]Task24ExecutionRecord
}

// Task24ExecutionRecord represents a test execution record
type Task24ExecutionRecord struct {
	TestID    string        `json:"test_id"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	Timestamp time.Time     `json:"timestamp"`
}

// Task24ExecutionOptimizer optimizes test execution
type Task24ExecutionOptimizer struct {
	bottleneckDetector *Task24BottleneckDetector
	resourceMonitor    *Task24ResourceMonitor
}

// Task24BottleneckDetector detects execution bottlenecks
type Task24BottleneckDetector struct {
	executionMetrics map[string]*Task24ExecutionMetrics
}

// Task24ExecutionMetrics contains execution performance metrics
type Task24ExecutionMetrics struct {
	TestID           string        `json:"test_id"`
	AverageTime      time.Duration `json:"average_time"`
	QueueTime        time.Duration `json:"queue_time"`
	ResourceWaitTime time.Duration `json:"resource_wait_time"`
	Variance         float64       `json:"variance"`
}

// Task24ResourceMonitor monitors resource usage
type Task24ResourceMonitor struct {
	cpuUsage    map[string][]float64
	memoryUsage map[string][]int64
}

// Task24MaintenanceAdvisor provides maintenance recommendations
type Task24MaintenanceAdvisor struct {
	consolidationRules []Task24ConsolidationRule
}

// Task24ConsolidationRule defines test consolidation rules
type Task24ConsolidationRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Condition   string `json:"condition"`
}

// Task24OptimizationResult contains optimization analysis results
type Task24OptimizationResult struct {
	Bottlenecks       []Task24Bottleneck           `json:"bottlenecks"`
	ResourceAnalysis  *Task24ResourceAnalysis      `json:"resource_analysis"`
	Recommendations   []Task24OptimizationRecommendation `json:"recommendations"`
	EstimatedSavings  Task24EstimatedSavings       `json:"estimated_savings"`
}

// Task24Bottleneck represents a detected bottleneck
type Task24Bottleneck struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	TestIDs     []string `json:"test_ids"`
}

// Task24ResourceAnalysis contains resource usage analysis
type Task24ResourceAnalysis struct {
	CPUUtilization     float64 `json:"cpu_utilization"`
	MemoryUtilization  float64 `json:"memory_utilization"`
	Efficiency         float64 `json:"efficiency"`
}

// Task24OptimizationRecommendation represents an optimization recommendation
type Task24OptimizationRecommendation struct {
	Type            string                 `json:"type"`
	Priority        string                 `json:"priority"`
	Description     string                 `json:"description"`
	ExpectedImpact  string                 `json:"expected_impact"`
	EstimatedSavings Task24EstimatedSavings `json:"estimated_savings"`
}

// Task24EstimatedSavings represents estimated savings
type Task24EstimatedSavings struct {
	ExecutionTime time.Duration `json:"execution_time"`
	ResourceUsage float64       `json:"resource_usage"`
	Maintenance   float64       `json:"maintenance"`
}

// NewTask24Implementation creates a new Task 24 implementation
func NewTask24Implementation() *Task24Implementation {
	return &Task24Implementation{
		impactAnalyzer:     NewTask24ImpactAnalyzer(),
		timePredictor:      NewTask24TimePredictor(),
		executionOptimizer: NewTask24ExecutionOptimizer(),
		maintenanceAdvisor: NewTask24MaintenanceAdvisor(),
	}
}

// NewTask24ImpactAnalyzer creates a new impact analyzer
func NewTask24ImpactAnalyzer() *Task24ImpactAnalyzer {
	return &Task24ImpactAnalyzer{
		dependencyMap: make(map[string][]string),
	}
}

// NewTask24TimePredictor creates a new time predictor
func NewTask24TimePredictor() *Task24TimePredictor {
	return &Task24TimePredictor{
		historicalData: make(map[string][]Task24ExecutionRecord),
	}
}

// NewTask24ExecutionOptimizer creates a new execution optimizer
func NewTask24ExecutionOptimizer() *Task24ExecutionOptimizer {
	return &Task24ExecutionOptimizer{
		bottleneckDetector: &Task24BottleneckDetector{
			executionMetrics: make(map[string]*Task24ExecutionMetrics),
		},
		resourceMonitor: &Task24ResourceMonitor{
			cpuUsage:    make(map[string][]float64),
			memoryUsage: make(map[string][]int64),
		},
	}
}

// NewTask24MaintenanceAdvisor creates a new maintenance advisor
func NewTask24MaintenanceAdvisor() *Task24MaintenanceAdvisor {
	return &Task24MaintenanceAdvisor{
		consolidationRules: []Task24ConsolidationRule{
			{
				Name:        "similar_package_tests",
				Description: "Consolidate tests in the same package with similar setup",
				Condition:   "same_package && similar_setup",
			},
			{
				Name:        "duplicate_functionality",
				Description: "Merge tests that test the same functionality",
				Condition:   "same_functionality && high_similarity",
			},
		},
	}
}

// SelectIntelligentTests implements intelligent test selection (Task 24.1)
func (t *Task24Implementation) SelectIntelligentTests(ctx context.Context, request Task24SelectionRequest) (*Task24SelectionResult, error) {
	log.Printf("Starting intelligent test selection for %d changed files", len(request.ChangedFiles))
	
	// Step 1: Analyze impact of code changes
	affectedTests, err := t.impactAnalyzer.AnalyzeCodeImpact(ctx, request.ChangedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze code impact: %w", err)
	}
	
	// Step 2: Predict execution times
	for i := range affectedTests {
		prediction, err := t.timePredictor.PredictTime(ctx, &affectedTests[i])
		if err != nil {
			log.Printf("Warning: failed to predict time for test %s: %v", affectedTests[i].ID, err)
			continue
		}
		affectedTests[i].EstimatedTime = prediction
	}
	
	// Step 3: Select optimal test set based on priority and constraints
	selectedTests := t.selectOptimalTests(affectedTests, request)
	
	// Step 4: Create execution plan with parallel optimization
	executionPlan := t.createExecutionPlan(selectedTests, request.ParallelWorkers)
	
	// Step 5: Generate reasoning and optimizations
	reasoning := t.generateSelectionReasoning(selectedTests, request)
	optimizations := t.identifyOptimizations(selectedTests, executionPlan)
	
	result := &Task24SelectionResult{
		SelectedTests:    selectedTests,
		ExecutionPlan:    executionPlan,
		EstimatedTime:    executionPlan.TotalEstimated,
		ExpectedCoverage: t.calculateExpectedCoverage(selectedTests),
		Reasoning:        reasoning,
		Optimizations:    optimizations,
	}
	
	log.Printf("Selected %d tests with estimated time %v", len(selectedTests), result.EstimatedTime)
	return result, nil
}

// OptimizeTestExecution implements test execution performance optimization (Task 24.2)
func (t *Task24Implementation) OptimizeTestExecution(ctx context.Context, tests []Task24TestInfo) (*Task24OptimizationResult, error) {
	log.Printf("Optimizing execution for %d tests", len(tests))
	
	// Step 1: Detect bottlenecks
	bottlenecks, err := t.executionOptimizer.DetectBottlenecks(ctx, tests)
	if err != nil {
		return nil, fmt.Errorf("failed to detect bottlenecks: %w", err)
	}
	
	// Step 2: Analyze resource usage
	resourceAnalysis, err := t.executionOptimizer.AnalyzeResourceUsage(ctx, tests)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze resource usage: %w", err)
	}
	
	// Step 3: Generate optimization recommendations
	recommendations := t.generateOptimizationRecommendations(bottlenecks, resourceAnalysis)
	
	// Step 4: Calculate estimated savings
	estimatedSavings := t.calculateEstimatedSavings(recommendations)
	
	result := &Task24OptimizationResult{
		Bottlenecks:       bottlenecks,
		ResourceAnalysis:  resourceAnalysis,
		Recommendations:   recommendations,
		EstimatedSavings:  estimatedSavings,
	}
	
	log.Printf("Generated %d optimization recommendations", len(recommendations))
	return result, nil
}

// AnalyzeCodeImpact analyzes the impact of code changes on test selection
func (a *Task24ImpactAnalyzer) AnalyzeCodeImpact(ctx context.Context, changedFiles []string) ([]Task24TestInfo, error) {
	var affectedTests []Task24TestInfo
	
	// Simulate impact analysis - in real implementation, this would:
	// 1. Parse code dependencies
	// 2. Identify affected test files
	// 3. Calculate impact scores
	
	for i, file := range changedFiles {
		// Create mock affected tests based on changed files
		test := Task24TestInfo{
			ID:            fmt.Sprintf("test_%d", i+1),
			Name:          fmt.Sprintf("Test for %s", file),
			Package:       "internal/testing",
			FilePath:      file,
			Dependencies:  []string{file},
			Tags:          []string{"unit"},
			EstimatedTime: time.Duration(30+i*10) * time.Second,
			SuccessRate:   0.95,
			ImpactScore:   0.8 - float64(i)*0.1, // Decreasing impact
			Flakiness:     0.05,
			ResourceUsage: Task24ResourceUsage{
				CPU:     float64(20 + i*10),
				Memory:  int64((100 + i*50) * 1024 * 1024),
				IO:      float64(10 + i*5),
				Network: i%2 == 0,
			},
		}
		affectedTests = append(affectedTests, test)
	}
	
	return affectedTests, nil
}

// PredictTime predicts execution time for a test
func (p *Task24TimePredictor) PredictTime(ctx context.Context, test *Task24TestInfo) (time.Duration, error) {
	// Check historical data
	if history, exists := p.historicalData[test.ID]; exists && len(history) > 0 {
		// Calculate average from recent executions
		var total time.Duration
		count := 0
		for _, record := range history {
			if record.Success {
				total += record.Duration
				count++
			}
		}
		if count > 0 {
			avg := total / time.Duration(count)
			// Apply factors based on resource usage
			factor := 1.0
			if test.ResourceUsage.CPU > 70 {
				factor *= 1.2 // 20% slower for high CPU
			}
			if test.ResourceUsage.Network {
				factor *= 1.3 // 30% slower for network tests
			}
			return time.Duration(float64(avg) * factor), nil
		}
	}
	
	// Use estimated time with adjustments
	basetime := test.EstimatedTime
	if basetime == 0 {
		basetime = 30 * time.Second // Default
	}
	
	// Adjust based on resource usage and flakiness
	factor := 1.0 + test.Flakiness // Add time for potential retries
	if test.ResourceUsage.CPU > 50 {
		factor *= 1.1
	}
	
	return time.Duration(float64(basetime) * factor), nil
}

// selectOptimalTests selects the optimal set of tests based on priority and constraints
func (t *Task24Implementation) selectOptimalTests(candidates []Task24TestInfo, request Task24SelectionRequest) []Task24TestInfo {
	// Sort by priority score (impact/time ratio)
	sort.Slice(candidates, func(i, j int) bool {
		scoreI := t.calculatePriorityScore(candidates[i])
		scoreJ := t.calculatePriorityScore(candidates[j])
		return scoreI > scoreJ
	})
	
	var selected []Task24TestInfo
	var totalTime time.Duration
	
	for _, test := range candidates {
		// Check time constraint
		if request.TimeLimit > 0 && totalTime+test.EstimatedTime > request.TimeLimit {
			continue
		}
		
		// Apply priority-based filtering
		if t.shouldIncludeTest(test, request.Priority) {
			selected = append(selected, test)
			totalTime += test.EstimatedTime
		}
		
		// Stop if we have enough tests for target coverage
		if request.TargetCoverage > 0 && len(selected) >= int(request.TargetCoverage*float64(len(candidates))) {
			break
		}
	}
	
	return selected
}

// calculatePriorityScore calculates a priority score for test selection
func (t *Task24Implementation) calculatePriorityScore(test Task24TestInfo) float64 {
	// Base score from impact
	score := test.ImpactScore
	
	// Adjust for execution time (prefer faster tests)
	timeSeconds := test.EstimatedTime.Seconds()
	if timeSeconds > 0 {
		score = score / (1 + timeSeconds/60) // Normalize by minutes
	}
	
	// Penalize flaky tests
	score *= (1.0 - test.Flakiness)
	
	// Boost tests with high success rate
	score *= test.SuccessRate
	
	return score
}

// shouldIncludeTest determines if a test should be included based on priority
func (t *Task24Implementation) shouldIncludeTest(test Task24TestInfo, priority string) bool {
	switch priority {
	case "fast":
		return test.EstimatedTime < 60*time.Second && test.ImpactScore > 0.5 // More lenient for fast
	case "critical":
		return test.ImpactScore > 0.8 || t.isCriticalTest(test)
	case "balanced":
		return test.EstimatedTime < 5*time.Minute && test.ImpactScore > 0.4
	case "complete":
		return true
	default:
		return test.ImpactScore > 0.5
	}
}

// isCriticalTest determines if a test is critical
func (t *Task24Implementation) isCriticalTest(test Task24TestInfo) bool {
	criticalTags := []string{"critical", "smoke", "security", "performance"}
	
	for _, tag := range test.Tags {
		for _, critical := range criticalTags {
			if tag == critical {
				return true
			}
		}
	}
	
	return test.ImpactScore > 0.9
}

// createExecutionPlan creates an optimized execution plan
func (t *Task24Implementation) createExecutionPlan(tests []Task24TestInfo, maxWorkers int) *Task24ExecutionPlan {
	// Group tests by execution characteristics
	var fastTests, mediumTests, slowTests []Task24TestInfo
	
	for _, test := range tests {
		if test.EstimatedTime < 30*time.Second {
			fastTests = append(fastTests, test)
		} else if test.EstimatedTime < 5*time.Minute {
			mediumTests = append(mediumTests, test)
		} else {
			slowTests = append(slowTests, test)
		}
	}
	
	var phases []Task24ExecutionPhase
	var parallelGroups []Task24ParallelGroup
	var totalTime time.Duration
	
	// Phase 1: Fast tests in parallel
	if len(fastTests) > 0 {
		workers := min(len(fastTests), maxWorkers, 4)
		phaseTime := t.calculatePhaseTime(fastTests, workers)
		
		phases = append(phases, Task24ExecutionPhase{
			Name:       "fast_parallel",
			Tests:      fastTests,
			Parallel:   true,
			MaxWorkers: workers,
			Timeout:    5 * time.Minute,
			Priority:   1,
		})
		
		parallelGroups = append(parallelGroups, Task24ParallelGroup{
			ID:         "fast_group",
			Tests:      fastTests,
			MaxWorkers: workers,
			Estimated:  phaseTime,
		})
		
		totalTime += phaseTime
	}
	
	// Phase 2: Medium tests with limited parallelism
	if len(mediumTests) > 0 {
		workers := min(len(mediumTests), maxWorkers/2, 2)
		phaseTime := t.calculatePhaseTime(mediumTests, workers)
		
		phases = append(phases, Task24ExecutionPhase{
			Name:       "medium_parallel",
			Tests:      mediumTests,
			Parallel:   true,
			MaxWorkers: workers,
			Timeout:    15 * time.Minute,
			Priority:   2,
		})
		
		parallelGroups = append(parallelGroups, Task24ParallelGroup{
			ID:         "medium_group",
			Tests:      mediumTests,
			MaxWorkers: workers,
			Estimated:  phaseTime,
		})
		
		totalTime += phaseTime
	}
	
	// Phase 3: Slow tests sequentially
	if len(slowTests) > 0 {
		phaseTime := t.calculatePhaseTime(slowTests, 1)
		
		phases = append(phases, Task24ExecutionPhase{
			Name:       "slow_sequential",
			Tests:      slowTests,
			Parallel:   false,
			MaxWorkers: 1,
			Timeout:    30 * time.Minute,
			Priority:   3,
		})
		
		parallelGroups = append(parallelGroups, Task24ParallelGroup{
			ID:         "slow_group",
			Tests:      slowTests,
			MaxWorkers: 1,
			Estimated:  phaseTime,
		})
		
		totalTime += phaseTime
	}
	
	return &Task24ExecutionPlan{
		Phases:         phases,
		ParallelGroups: parallelGroups,
		TotalEstimated: totalTime,
		ResourceNeeds: Task24ResourceNeeds{
			TotalCPU:    t.calculateTotalCPU(tests),
			TotalMemory: t.calculateTotalMemory(tests),
			Workers:     maxWorkers,
		},
	}
}

// calculatePhaseTime calculates execution time for a phase
func (t *Task24Implementation) calculatePhaseTime(tests []Task24TestInfo, workers int) time.Duration {
	if len(tests) == 0 {
		return 0
	}
	
	if workers <= 1 {
		// Sequential execution
		var total time.Duration
		for _, test := range tests {
			total += test.EstimatedTime
		}
		return total
	}
	
	// Parallel execution - simulate load balancing
	sort.Slice(tests, func(i, j int) bool {
		return tests[i].EstimatedTime > tests[j].EstimatedTime
	})
	
	workerQueues := make([]time.Duration, workers)
	
	for _, test := range tests {
		// Find worker with least work
		minWorker := 0
		for i := 1; i < workers; i++ {
			if workerQueues[i] < workerQueues[minWorker] {
				minWorker = i
			}
		}
		workerQueues[minWorker] += test.EstimatedTime
	}
	
	// Return maximum time (bottleneck worker)
	maxTime := workerQueues[0]
	for _, time := range workerQueues {
		if time > maxTime {
			maxTime = time
		}
	}
	
	return maxTime
}

// DetectBottlenecks detects execution bottlenecks
func (o *Task24ExecutionOptimizer) DetectBottlenecks(ctx context.Context, tests []Task24TestInfo) ([]Task24Bottleneck, error) {
	var bottlenecks []Task24Bottleneck
	
	// Analyze resource usage patterns
	highCPUTests := []string{}
	highMemoryTests := []string{}
	networkTests := []string{}
	
	for _, test := range tests {
		if test.ResourceUsage.CPU > 70 {
			highCPUTests = append(highCPUTests, test.ID)
		}
		if test.ResourceUsage.Memory > 2*1024*1024*1024 { // 2GB
			highMemoryTests = append(highMemoryTests, test.ID)
		}
		if test.ResourceUsage.Network {
			networkTests = append(networkTests, test.ID)
		}
	}
	
	// Create bottleneck reports
	if len(highCPUTests) > 0 {
		bottlenecks = append(bottlenecks, Task24Bottleneck{
			Type:        "cpu_contention",
			Severity:    "medium",
			Description: fmt.Sprintf("%d tests have high CPU usage", len(highCPUTests)),
			TestIDs:     highCPUTests,
		})
	}
	
	if len(highMemoryTests) > 0 {
		bottlenecks = append(bottlenecks, Task24Bottleneck{
			Type:        "memory_pressure",
			Severity:    "high",
			Description: fmt.Sprintf("%d tests have high memory usage", len(highMemoryTests)),
			TestIDs:     highMemoryTests,
		})
	}
	
	if len(networkTests) > 3 {
		bottlenecks = append(bottlenecks, Task24Bottleneck{
			Type:        "network_dependency",
			Severity:    "medium",
			Description: fmt.Sprintf("%d tests depend on network", len(networkTests)),
			TestIDs:     networkTests,
		})
	}
	
	return bottlenecks, nil
}

// AnalyzeResourceUsage analyzes resource usage patterns
func (o *Task24ExecutionOptimizer) AnalyzeResourceUsage(ctx context.Context, tests []Task24TestInfo) (*Task24ResourceAnalysis, error) {
	if len(tests) == 0 {
		return &Task24ResourceAnalysis{}, nil
	}
	
	totalCPU := 0.0
	totalMemory := int64(0)
	
	for _, test := range tests {
		totalCPU += test.ResourceUsage.CPU
		totalMemory += test.ResourceUsage.Memory
	}
	
	avgCPU := totalCPU / float64(len(tests))
	avgMemory := float64(totalMemory) / float64(len(tests))
	
	// Calculate efficiency (simplified)
	efficiency := avgCPU / 100.0 // Normalize CPU usage to 0-1
	if efficiency > 1.0 {
		efficiency = 1.0
	}
	
	return &Task24ResourceAnalysis{
		CPUUtilization:    avgCPU,
		MemoryUtilization: avgMemory / (16 * 1024 * 1024 * 1024), // Normalize to 16GB
		Efficiency:        efficiency,
	}, nil
}

// generateOptimizationRecommendations generates optimization recommendations
func (t *Task24Implementation) generateOptimizationRecommendations(bottlenecks []Task24Bottleneck, resourceAnalysis *Task24ResourceAnalysis) []Task24OptimizationRecommendation {
	var recommendations []Task24OptimizationRecommendation
	
	// Analyze bottlenecks
	for _, bottleneck := range bottlenecks {
		switch bottleneck.Type {
		case "cpu_contention":
			recommendations = append(recommendations, Task24OptimizationRecommendation{
				Type:        "parallel_optimization",
				Priority:    "high",
				Description: "Reduce CPU contention by optimizing parallel execution",
				ExpectedImpact: "30-50% reduction in execution time",
				EstimatedSavings: Task24EstimatedSavings{
					ExecutionTime: 5 * time.Minute,
					ResourceUsage: 0.3,
				},
			})
			
		case "memory_pressure":
			recommendations = append(recommendations, Task24OptimizationRecommendation{
				Type:        "resource_optimization",
				Priority:    "high",
				Description: "Optimize memory usage to reduce pressure",
				ExpectedImpact: "Improved stability and reduced failures",
				EstimatedSavings: Task24EstimatedSavings{
					ResourceUsage: 0.4,
					Maintenance:   0.2,
				},
			})
			
		case "network_dependency":
			recommendations = append(recommendations, Task24OptimizationRecommendation{
				Type:        "scheduling_optimization",
				Priority:    "medium",
				Description: "Schedule network tests to avoid contention",
				ExpectedImpact: "20-30% reduction in network-related delays",
				EstimatedSavings: Task24EstimatedSavings{
					ExecutionTime: 2 * time.Minute,
					ResourceUsage: 0.1,
				},
			})
		}
	}
	
	// Analyze resource usage
	if resourceAnalysis.CPUUtilization < 30 {
		recommendations = append(recommendations, Task24OptimizationRecommendation{
			Type:        "parallel_optimization",
			Priority:    "medium",
			Description: "Increase parallelism to better utilize CPU",
			ExpectedImpact: "40-60% improvement in CPU utilization",
			EstimatedSavings: Task24EstimatedSavings{
				ExecutionTime: 8 * time.Minute,
				ResourceUsage: 0.4,
			},
		})
	}
	
	if resourceAnalysis.Efficiency < 0.6 {
		recommendations = append(recommendations, Task24OptimizationRecommendation{
			Type:        "scheduling_optimization",
			Priority:    "medium",
			Description: "Optimize test scheduling for better resource efficiency",
			ExpectedImpact: "20-30% improvement in overall efficiency",
			EstimatedSavings: Task24EstimatedSavings{
				ExecutionTime: 6 * time.Minute,
				ResourceUsage: 0.25,
			},
		})
	}
	
	return recommendations
}

// Helper functions
func (t *Task24Implementation) calculateExpectedCoverage(tests []Task24TestInfo) float64 {
	// Simplified coverage calculation
	return float64(len(tests)) / 100.0 // Assume 100 total tests for simplicity
}

func (t *Task24Implementation) generateSelectionReasoning(tests []Task24TestInfo, request Task24SelectionRequest) []string {
	var reasoning []string
	
	reasoning = append(reasoning, fmt.Sprintf("Analyzed %d changed files for impact", len(request.ChangedFiles)))
	reasoning = append(reasoning, fmt.Sprintf("Selected %d tests based on %s priority", len(tests), request.Priority))
	
	if request.TimeLimit > 0 {
		reasoning = append(reasoning, fmt.Sprintf("Constrained by time limit of %v", request.TimeLimit))
	}
	
	criticalCount := 0
	for _, test := range tests {
		if t.isCriticalTest(test) {
			criticalCount++
		}
	}
	
	if criticalCount > 0 {
		reasoning = append(reasoning, fmt.Sprintf("Included %d critical tests", criticalCount))
	}
	
	return reasoning
}

func (t *Task24Implementation) identifyOptimizations(tests []Task24TestInfo, plan *Task24ExecutionPlan) []Task24Optimization {
	var optimizations []Task24Optimization
	
	if len(plan.ParallelGroups) > 1 {
		optimizations = append(optimizations, Task24Optimization{
			Type:        "parallel_execution",
			Description: fmt.Sprintf("Executing %d test groups in parallel", len(plan.ParallelGroups)),
			Impact:      "Reduces total execution time by up to 60%",
		})
	}
	
	// Check for consolidation opportunities
	packageGroups := make(map[string][]Task24TestInfo)
	for _, test := range tests {
		packageGroups[test.Package] = append(packageGroups[test.Package], test)
	}
	
	for pkg, pkgTests := range packageGroups {
		if len(pkgTests) > 5 {
			optimizations = append(optimizations, Task24Optimization{
				Type:        "test_consolidation",
				Description: fmt.Sprintf("Package %s has %d tests that could be consolidated", pkg, len(pkgTests)),
				Impact:      "Could reduce setup/teardown overhead",
			})
		}
	}
	
	return optimizations
}

func (t *Task24Implementation) calculateTotalCPU(tests []Task24TestInfo) float64 {
	total := 0.0
	for _, test := range tests {
		total += test.ResourceUsage.CPU
	}
	return total
}

func (t *Task24Implementation) calculateTotalMemory(tests []Task24TestInfo) int64 {
	total := int64(0)
	for _, test := range tests {
		total += test.ResourceUsage.Memory
	}
	return total
}

func (t *Task24Implementation) calculateEstimatedSavings(recommendations []Task24OptimizationRecommendation) Task24EstimatedSavings {
	var total Task24EstimatedSavings
	
	for _, rec := range recommendations {
		total.ExecutionTime += rec.EstimatedSavings.ExecutionTime
		total.ResourceUsage += rec.EstimatedSavings.ResourceUsage
		total.Maintenance += rec.EstimatedSavings.Maintenance
	}
	
	return total
}

// min returns the minimum of multiple integers
func min(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	
	minVal := values[0]
	for _, val := range values[1:] {
		if val < minVal {
			minVal = val
		}
	}
	return minVal
}