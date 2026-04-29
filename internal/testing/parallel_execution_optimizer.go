package testing

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"
)

// ParallelExecutionOptimizer optimizes test execution for parallel processing
type ParallelExecutionOptimizer struct {
	resourceManager *ResourceManager
	scheduler       *TestScheduler
}

// ResourceManager manages test execution resources
type ResourceManager struct {
	maxCPU    int
	maxMemory int64
	maxIO     int
}

// TestScheduler schedules test execution
type TestScheduler struct {
	queues map[string]*TestQueue
}

// TestQueue represents a queue of tests
type TestQueue struct {
	Name     string
	Tests    []TestCase
	Priority int
	MaxSize  int
}

// ResourceUsage represents resource usage of a test
type ResourceUsage struct {
	CPU    float64 `json:"cpu"`    // CPU usage percentage
	Memory int64   `json:"memory"` // Memory usage in bytes
	IO     float64 `json:"io"`     // IO usage percentage
	Network bool   `json:"network"` // Whether test uses network
}

// ResourceNeeds represents resource requirements
type ResourceNeeds struct {
	TotalCPU    float64 `json:"total_cpu"`
	TotalMemory int64   `json:"total_memory"`
	TotalIO     float64 `json:"total_io"`
	Workers     int     `json:"workers"`
}

// Dependency represents a test dependency
type Dependency struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// NewParallelExecutionOptimizer creates a new parallel execution optimizer
func NewParallelExecutionOptimizer() *ParallelExecutionOptimizer {
	return &ParallelExecutionOptimizer{
		resourceManager: &ResourceManager{
			maxCPU:    8,  // 8 CPU cores
			maxMemory: 16 * 1024 * 1024 * 1024, // 16GB
			maxIO:     100, // 100% IO capacity
		},
		scheduler: &TestScheduler{
			queues: make(map[string]*TestQueue),
		},
	}
}

// OptimizeExecution optimizes test execution for parallel processing
func (o *ParallelExecutionOptimizer) OptimizeExecution(ctx context.Context, tests []TestCase, maxWorkers int) (*ExecutionPlan, error) {
	log.Printf("Optimizing execution for %d tests with %d max workers", len(tests), maxWorkers)
	
	// Step 1: Analyze test dependencies
	dependencies := o.analyzeDependencies(tests)
	
	// Step 2: Group tests by resource requirements and dependencies
	groups := o.groupTestsForParallelExecution(tests, dependencies)
	
	// Step 3: Create execution phases based on dependencies
	phases := o.createExecutionPhases(groups, dependencies)
	
	// Step 4: Optimize resource allocation
	optimizedPhases := o.optimizeResourceAllocation(phases, maxWorkers)
	
	// Step 5: Create parallel groups within phases
	parallelGroups := o.createParallelGroups(optimizedPhases, maxWorkers)
	
	// Step 6: Calculate total estimated time
	totalEstimated := o.calculateTotalEstimatedTime(optimizedPhases)
	
	// Step 7: Calculate resource needs
	resourceNeeds := o.calculateResourceNeeds(tests, maxWorkers)
	
	plan := &ExecutionPlan{
		Phases:         optimizedPhases,
		ParallelGroups: parallelGroups,
		TotalEstimated: totalEstimated,
		ResourceNeeds:  resourceNeeds,
		Dependencies:   dependencies,
	}
	
	log.Printf("Created execution plan with %d phases and %d parallel groups, estimated time: %v", 
		len(plan.Phases), len(plan.ParallelGroups), plan.TotalEstimated)
	
	return plan, nil
}

// analyzeDependencies analyzes dependencies between tests
func (o *ParallelExecutionOptimizer) analyzeDependencies(tests []TestCase) []Dependency {
	var dependencies []Dependency
	
	// Create a map for quick lookup
	testMap := make(map[string]TestCase)
	for _, test := range tests {
		testMap[test.ID] = test
	}
	
	// Analyze dependencies
	for _, test := range tests {
		for _, dep := range test.Dependencies {
			if _, exists := testMap[dep]; exists {
				dependencies = append(dependencies, Dependency{
					From: test.ID,
					To:   dep,
					Type: "code_dependency",
				})
			}
		}
		
		// Check for database dependencies
		if o.requiresDatabase(test) {
			for _, other := range tests {
				if other.ID != test.ID && o.requiresDatabase(other) && o.conflictsWithDatabase(test, other) {
					dependencies = append(dependencies, Dependency{
						From: test.ID,
						To:   other.ID,
						Type: "database_conflict",
					})
				}
			}
		}
		
		// Check for resource conflicts
		if test.ResourceUsage.CPU > 50 { // High CPU tests
			for _, other := range tests {
				if other.ID != test.ID && other.ResourceUsage.CPU > 50 {
					dependencies = append(dependencies, Dependency{
						From: test.ID,
						To:   other.ID,
						Type: "resource_conflict",
					})
				}
			}
		}
	}
	
	return dependencies
}

// groupTestsForParallelExecution groups tests for optimal parallel execution
func (o *ParallelExecutionOptimizer) groupTestsForParallelExecution(tests []TestCase, dependencies []Dependency) map[string][]TestCase {
	groups := make(map[string][]TestCase)
	
	// Group by test characteristics
	for _, test := range tests {
		groupKey := o.determineTestGroup(test)
		groups[groupKey] = append(groups[groupKey], test)
	}
	
	return groups
}

// determineTestGroup determines which group a test belongs to
func (o *ParallelExecutionOptimizer) determineTestGroup(test TestCase) string {
	// Group by execution characteristics
	if test.EstimatedTime < 10*time.Second {
		return "fast"
	} else if test.EstimatedTime < 60*time.Second {
		return "medium"
	} else {
		return "slow"
	}
}

// createExecutionPhases creates execution phases based on dependencies
func (o *ParallelExecutionOptimizer) createExecutionPhases(groups map[string][]TestCase, dependencies []Dependency) []ExecutionPhase {
	var phases []ExecutionPhase
	
	// Phase 1: Fast, independent tests
	if fastTests, exists := groups["fast"]; exists {
		independentFast := o.filterIndependentTests(fastTests, dependencies)
		if len(independentFast) > 0 {
			phases = append(phases, ExecutionPhase{
				Name:       "fast_parallel",
				Tests:      independentFast,
				Parallel:   true,
				MaxWorkers: 4,
				Timeout:    5 * time.Minute,
				Priority:   1,
			})
		}
	}
	
	// Phase 2: Medium tests with some parallelization
	if mediumTests, exists := groups["medium"]; exists {
		phases = append(phases, ExecutionPhase{
			Name:       "medium_parallel",
			Tests:      mediumTests,
			Parallel:   true,
			MaxWorkers: 2,
			Timeout:    15 * time.Minute,
			Priority:   2,
		})
	}
	
	// Phase 3: Slow tests, limited parallelization
	if slowTests, exists := groups["slow"]; exists {
		phases = append(phases, ExecutionPhase{
			Name:       "slow_sequential",
			Tests:      slowTests,
			Parallel:   false,
			MaxWorkers: 1,
			Timeout:    30 * time.Minute,
			Priority:   3,
		})
	}
	
	return phases
}

// filterIndependentTests filters tests that have no dependencies
func (o *ParallelExecutionOptimizer) filterIndependentTests(tests []TestCase, dependencies []Dependency) []TestCase {
	dependentTests := make(map[string]bool)
	
	// Mark tests that have dependencies
	for _, dep := range dependencies {
		dependentTests[dep.From] = true
	}
	
	var independent []TestCase
	for _, test := range tests {
		if !dependentTests[test.ID] {
			independent = append(independent, test)
		}
	}
	
	return independent
}

// optimizeResourceAllocation optimizes resource allocation for phases
func (o *ParallelExecutionOptimizer) optimizeResourceAllocation(phases []ExecutionPhase, maxWorkers int) []ExecutionPhase {
	optimized := make([]ExecutionPhase, len(phases))
	copy(optimized, phases)
	
	for i := range optimized {
		phase := &optimized[i]
		
		// Calculate optimal worker count based on resource usage
		totalCPU := 0.0
		totalMemory := int64(0)
		
		for _, test := range phase.Tests {
			totalCPU += test.ResourceUsage.CPU
			totalMemory += test.ResourceUsage.Memory
		}
		
		// Adjust worker count based on resource constraints
		cpuWorkers := int(float64(o.resourceManager.maxCPU) / (totalCPU / float64(len(phase.Tests))))
		memWorkers := int(o.resourceManager.maxMemory / (totalMemory / int64(len(phase.Tests))))
		
		optimalWorkers := min(cpuWorkers, memWorkers, maxWorkers, phase.MaxWorkers)
		if optimalWorkers < 1 {
			optimalWorkers = 1
		}
		
		phase.MaxWorkers = optimalWorkers
		
		log.Printf("Phase %s: optimized to %d workers (CPU limit: %d, Memory limit: %d)", 
			phase.Name, optimalWorkers, cpuWorkers, memWorkers)
	}
	
	return optimized
}

// createParallelGroups creates parallel groups within phases
func (o *ParallelExecutionOptimizer) createParallelGroups(phases []ExecutionPhase, maxWorkers int) []ParallelGroup {
	var groups []ParallelGroup
	
	for _, phase := range phases {
		if !phase.Parallel {
			// Sequential phase - create single group
			groups = append(groups, ParallelGroup{
				ID:         fmt.Sprintf("%s_sequential", phase.Name),
				Tests:      phase.Tests,
				MaxWorkers: 1,
				Resources:  o.calculatePhaseResourceNeeds(phase.Tests),
				Estimated:  o.calculatePhaseEstimatedTime(phase.Tests, 1),
			})
			continue
		}
		
		// Parallel phase - create multiple groups
		groupSize := len(phase.Tests) / phase.MaxWorkers
		if groupSize < 1 {
			groupSize = 1
		}
		
		for i := 0; i < len(phase.Tests); i += groupSize {
			end := i + groupSize
			if end > len(phase.Tests) {
				end = len(phase.Tests)
			}
			
			groupTests := phase.Tests[i:end]
			groups = append(groups, ParallelGroup{
				ID:         fmt.Sprintf("%s_group_%d", phase.Name, i/groupSize),
				Tests:      groupTests,
				MaxWorkers: min(len(groupTests), phase.MaxWorkers),
				Resources:  o.calculatePhaseResourceNeeds(groupTests),
				Estimated:  o.calculatePhaseEstimatedTime(groupTests, min(len(groupTests), phase.MaxWorkers)),
			})
		}
	}
	
	return groups
}

// calculateTotalEstimatedTime calculates total estimated execution time
func (o *ParallelExecutionOptimizer) calculateTotalEstimatedTime(phases []ExecutionPhase) time.Duration {
	var total time.Duration
	
	for _, phase := range phases {
		phaseTime := o.calculatePhaseEstimatedTime(phase.Tests, phase.MaxWorkers)
		total += phaseTime
	}
	
	return total
}

// calculatePhaseEstimatedTime calculates estimated time for a phase
func (o *ParallelExecutionOptimizer) calculatePhaseEstimatedTime(tests []TestCase, workers int) time.Duration {
	if len(tests) == 0 {
		return 0
	}
	
	// Sort tests by estimated time (longest first)
	sortedTests := make([]TestCase, len(tests))
	copy(sortedTests, tests)
	sort.Slice(sortedTests, func(i, j int) bool {
		return sortedTests[i].EstimatedTime > sortedTests[j].EstimatedTime
	})
	
	// Simulate parallel execution
	workerQueues := make([]time.Duration, workers)
	
	for _, test := range sortedTests {
		// Find worker with least work
		minWorker := 0
		for i := 1; i < workers; i++ {
			if workerQueues[i] < workerQueues[minWorker] {
				minWorker = i
			}
		}
		
		// Add test to that worker
		workerQueues[minWorker] += test.EstimatedTime
	}
	
	// Return the maximum time (bottleneck worker)
	maxTime := workerQueues[0]
	for _, time := range workerQueues {
		if time > maxTime {
			maxTime = time
		}
	}
	
	return maxTime
}

// calculateResourceNeeds calculates resource needs for all tests
func (o *ParallelExecutionOptimizer) calculateResourceNeeds(tests []TestCase, workers int) ResourceNeeds {
	var totalCPU float64
	var totalMemory int64
	var totalIO float64
	
	for _, test := range tests {
		totalCPU += test.ResourceUsage.CPU
		totalMemory += test.ResourceUsage.Memory
		totalIO += test.ResourceUsage.IO
	}
	
	return ResourceNeeds{
		TotalCPU:    totalCPU,
		TotalMemory: totalMemory,
		TotalIO:     totalIO,
		Workers:     workers,
	}
}

// calculatePhaseResourceNeeds calculates resource needs for a phase
func (o *ParallelExecutionOptimizer) calculatePhaseResourceNeeds(tests []TestCase) ResourceNeeds {
	var maxCPU float64
	var maxMemory int64
	var maxIO float64
	
	for _, test := range tests {
		if test.ResourceUsage.CPU > maxCPU {
			maxCPU = test.ResourceUsage.CPU
		}
		if test.ResourceUsage.Memory > maxMemory {
			maxMemory = test.ResourceUsage.Memory
		}
		if test.ResourceUsage.IO > maxIO {
			maxIO = test.ResourceUsage.IO
		}
	}
	
	return ResourceNeeds{
		TotalCPU:    maxCPU,
		TotalMemory: maxMemory,
		TotalIO:     maxIO,
		Workers:     len(tests),
	}
}

// requiresDatabase checks if a test requires database access
func (o *ParallelExecutionOptimizer) requiresDatabase(test TestCase) bool {
	for _, tag := range test.Tags {
		if tag == "database" || tag == "integration" {
			return true
		}
	}
	return false
}

// conflictsWithDatabase checks if two tests conflict on database usage
func (o *ParallelExecutionOptimizer) conflictsWithDatabase(test1, test2 TestCase) bool {
	// Tests conflict if they both modify the same tables or use transactions
	for _, tag1 := range test1.Tags {
		for _, tag2 := range test2.Tags {
			if tag1 == tag2 && (tag1 == "transaction" || tag1 == "migration") {
				return true
			}
		}
	}
	return false
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