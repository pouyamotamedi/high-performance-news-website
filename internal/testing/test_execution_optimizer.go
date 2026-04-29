package testing

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// TestExecutionOptimizer optimizes test execution performance
type TestExecutionOptimizer struct {
	bottleneckDetector   *BottleneckDetector
	resourceMonitor      *ResourceMonitor
	executionScheduler   *ExecutionScheduler
	costAnalyzer         *CostAnalyzer
	optimizationEngine   *OptimizationEngine
}

// BottleneckDetector identifies execution bottlenecks
type BottleneckDetector struct {
	executionMetrics map[string]*ExecutionMetrics
	thresholds       *BottleneckThresholds
	mutex            sync.RWMutex
}

// ResourceMonitor monitors test resource usage
type ResourceMonitor struct {
	cpuUsage    map[string][]float64
	memoryUsage map[string][]int64
	ioUsage     map[string][]float64
	networkUsage map[string][]float64
	mutex       sync.RWMutex
}

// ExecutionScheduler manages test execution scheduling
type ExecutionScheduler struct {
	queues          map[Priority]*TestQueue
	resourceLimits  *ResourceLimits
	schedulingRules []SchedulingRule
}

// CostAnalyzer analyzes test execution costs
type CostAnalyzer struct {
	costMetrics    map[string]*CostMetrics
	pricingModel   *PricingModel
}

// OptimizationEngine provides optimization recommendations
type OptimizationEngine struct {
	rules           []OptimizationRule
	recommendations map[string][]OptimizationRecommendation
}

// ExecutionMetrics contains execution performance metrics
type ExecutionMetrics struct {
	TestID           string        `json:"test_id"`
	AverageTime      time.Duration `json:"average_time"`
	MinTime          time.Duration `json:"min_time"`
	MaxTime          time.Duration `json:"max_time"`
	Variance         float64       `json:"variance"`
	ThroughputPerMin float64       `json:"throughput_per_min"`
	QueueTime        time.Duration `json:"queue_time"`
	SetupTime        time.Duration `json:"setup_time"`
	ExecutionTime    time.Duration `json:"execution_time"`
	TeardownTime     time.Duration `json:"teardown_time"`
	ResourceWaitTime time.Duration `json:"resource_wait_time"`
}

// BottleneckThresholds defines thresholds for bottleneck detection
type BottleneckThresholds struct {
	MaxQueueTime        time.Duration `json:"max_queue_time"`
	MaxResourceWaitTime time.Duration `json:"max_resource_wait_time"`
	MaxVariance         float64       `json:"max_variance"`
	MinThroughput       float64       `json:"min_throughput"`
}

// ResourceLimits defines resource limits for scheduling
type ResourceLimits struct {
	MaxCPU         float64 `json:"max_cpu"`
	MaxMemory      int64   `json:"max_memory"`
	MaxIO          float64 `json:"max_io"`
	MaxNetwork     float64 `json:"max_network"`
	MaxConcurrency int     `json:"max_concurrency"`
}

// SchedulingRule defines a scheduling rule
type SchedulingRule struct {
	Name        string                 `json:"name"`
	Condition   func(TestCase) bool    `json:"-"`
	Action      func(*TestCase) error  `json:"-"`
	Priority    int                    `json:"priority"`
	Description string                 `json:"description"`
}

// CostMetrics contains cost analysis metrics
type CostMetrics struct {
	TestID              string  `json:"test_id"`
	ComputeCost         float64 `json:"compute_cost"`
	StorageCost         float64 `json:"storage_cost"`
	NetworkCost         float64 `json:"network_cost"`
	TotalCost           float64 `json:"total_cost"`
	CostPerExecution    float64 `json:"cost_per_execution"`
	CostPerMinute       float64 `json:"cost_per_minute"`
	ResourceEfficiency  float64 `json:"resource_efficiency"`
}

// PricingModel defines cost calculation model
type PricingModel struct {
	CPUCostPerHour     float64 `json:"cpu_cost_per_hour"`
	MemoryCostPerGBHour float64 `json:"memory_cost_per_gb_hour"`
	StorageCostPerGBHour float64 `json:"storage_cost_per_gb_hour"`
	NetworkCostPerGB   float64 `json:"network_cost_per_gb"`
}

// OptimizationRule defines an optimization rule
type OptimizationRule struct {
	Name        string                                    `json:"name"`
	Condition   func(*ExecutionMetrics) bool              `json:"-"`
	Analyze     func(*ExecutionMetrics) []string          `json:"-"`
	Recommend   func(*ExecutionMetrics) []OptimizationRecommendation `json:"-"`
	Priority    int                                       `json:"priority"`
	Description string                                    `json:"description"`
}

// OptimizationRecommendation represents an optimization recommendation
type OptimizationRecommendation struct {
	Type            OptimizationType `json:"type"`
	Priority        Priority         `json:"priority"`
	Description     string           `json:"description"`
	ExpectedImpact  string           `json:"expected_impact"`
	Implementation  string           `json:"implementation"`
	EstimatedSavings EstimatedSavings `json:"estimated_savings"`
	Automated       bool             `json:"automated"`
}

// OptimizationType represents the type of optimization
type OptimizationType string

const (
	OptimizationParallel     OptimizationType = "parallel"
	OptimizationResource     OptimizationType = "resource"
	OptimizationScheduling   OptimizationType = "scheduling"
	OptimizationCaching      OptimizationType = "caching"
	OptimizationBatching     OptimizationType = "batching"
	OptimizationElimination  OptimizationType = "elimination"
)

// NewTestExecutionOptimizer creates a new test execution optimizer
func NewTestExecutionOptimizer() *TestExecutionOptimizer {
	return &TestExecutionOptimizer{
		bottleneckDetector: NewBottleneckDetector(),
		resourceMonitor:    NewResourceMonitor(),
		executionScheduler: NewExecutionScheduler(),
		costAnalyzer:       NewCostAnalyzer(),
		optimizationEngine: NewOptimizationEngine(),
	}
}

// NewBottleneckDetector creates a new bottleneck detector
func NewBottleneckDetector() *BottleneckDetector {
	return &BottleneckDetector{
		executionMetrics: make(map[string]*ExecutionMetrics),
		thresholds: &BottleneckThresholds{
			MaxQueueTime:        5 * time.Minute,
			MaxResourceWaitTime: 2 * time.Minute,
			MaxVariance:         0.5, // 50% variance
			MinThroughput:       1.0, // 1 test per minute minimum
		},
	}
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor() *ResourceMonitor {
	return &ResourceMonitor{
		cpuUsage:     make(map[string][]float64),
		memoryUsage:  make(map[string][]int64),
		ioUsage:      make(map[string][]float64),
		networkUsage: make(map[string][]float64),
	}
}

// NewExecutionScheduler creates a new execution scheduler
func NewExecutionScheduler() *ExecutionScheduler {
	return &ExecutionScheduler{
		queues: make(map[Priority]*TestQueue),
		resourceLimits: &ResourceLimits{
			MaxCPU:         800, // 8 cores * 100%
			MaxMemory:      16 * 1024 * 1024 * 1024, // 16GB
			MaxIO:          100,
			MaxNetwork:     100,
			MaxConcurrency: 10,
		},
		schedulingRules: []SchedulingRule{
			{
				Name:        "prioritize_fast_tests",
				Condition:   func(test TestCase) bool { return test.EstimatedTime < 30*time.Second },
				Action:      func(test *TestCase) error { return nil },
				Priority:    1,
				Description: "Prioritize fast tests for quick feedback",
			},
			{
				Name:        "batch_similar_tests",
				Condition:   func(test TestCase) bool { return len(test.Tags) > 0 },
				Action:      func(test *TestCase) error { return nil },
				Priority:    2,
				Description: "Batch tests with similar characteristics",
			},
		},
	}
}

// NewCostAnalyzer creates a new cost analyzer
func NewCostAnalyzer() *CostAnalyzer {
	return &CostAnalyzer{
		costMetrics: make(map[string]*CostMetrics),
		pricingModel: &PricingModel{
			CPUCostPerHour:       0.10,  // $0.10 per CPU hour
			MemoryCostPerGBHour:  0.02,  // $0.02 per GB hour
			StorageCostPerGBHour: 0.001, // $0.001 per GB hour
			NetworkCostPerGB:     0.05,  // $0.05 per GB
		},
	}
}

// NewOptimizationEngine creates a new optimization engine
func NewOptimizationEngine() *OptimizationEngine {
	engine := &OptimizationEngine{
		recommendations: make(map[string][]OptimizationRecommendation),
	}
	
	engine.rules = []OptimizationRule{
		{
			Name:        "detect_parallel_opportunities",
			Condition:   func(metrics *ExecutionMetrics) bool { return metrics.QueueTime > 1*time.Minute },
			Analyze:     engine.analyzeParallelOpportunities,
			Recommend:   engine.recommendParallelOptimization,
			Priority:    1,
			Description: "Detect opportunities for parallel execution",
		},
		{
			Name:        "detect_resource_bottlenecks",
			Condition:   func(metrics *ExecutionMetrics) bool { return metrics.ResourceWaitTime > 30*time.Second },
			Analyze:     engine.analyzeResourceBottlenecks,
			Recommend:   engine.recommendResourceOptimization,
			Priority:    2,
			Description: "Detect resource allocation bottlenecks",
		},
		{
			Name:        "detect_scheduling_inefficiencies",
			Condition:   func(metrics *ExecutionMetrics) bool { return metrics.Variance > 0.3 },
			Analyze:     engine.analyzeSchedulingInefficiencies,
			Recommend:   engine.recommendSchedulingOptimization,
			Priority:    3,
			Description: "Detect scheduling inefficiencies",
		},
	}
	
	return engine
}

// OptimizeTestExecution optimizes test execution performance
func (o *TestExecutionOptimizer) OptimizeTestExecution(ctx context.Context, tests []TestCase) (*OptimizationResult, error) {
	log.Printf("Optimizing execution for %d tests", len(tests))
	
	// Step 1: Detect bottlenecks
	bottlenecks, err := o.bottleneckDetector.DetectBottlenecks(ctx, tests)
	if err != nil {
		return nil, fmt.Errorf("failed to detect bottlenecks: %w", err)
	}
	
	// Step 2: Monitor resource usage
	resourceUsage, err := o.resourceMonitor.AnalyzeResourceUsage(ctx, tests)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze resource usage: %w", err)
	}
	
	// Step 3: Optimize scheduling
	schedulingPlan, err := o.executionScheduler.OptimizeScheduling(ctx, tests)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize scheduling: %w", err)
	}
	
	// Step 4: Analyze costs
	costAnalysis, err := o.costAnalyzer.AnalyzeCosts(ctx, tests)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze costs: %w", err)
	}
	
	// Step 5: Generate optimization recommendations
	recommendations, err := o.optimizationEngine.GenerateRecommendations(ctx, bottlenecks, resourceUsage)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recommendations: %w", err)
	}
	
	result := &OptimizationResult{
		Bottlenecks:       bottlenecks,
		ResourceUsage:     resourceUsage,
		SchedulingPlan:    schedulingPlan,
		CostAnalysis:      costAnalysis,
		Recommendations:   recommendations,
		EstimatedSavings:  o.calculateTotalSavings(recommendations),
	}
	
	log.Printf("Generated %d optimization recommendations", len(recommendations))
	return result, nil
}

// OptimizationResult contains the results of optimization analysis
type OptimizationResult struct {
	Bottlenecks       []Bottleneck                    `json:"bottlenecks"`
	ResourceUsage     *ResourceUsageAnalysis          `json:"resource_usage"`
	SchedulingPlan    *SchedulingPlan                 `json:"scheduling_plan"`
	CostAnalysis      *CostAnalysis                   `json:"cost_analysis"`
	Recommendations   []OptimizationRecommendation    `json:"recommendations"`
	EstimatedSavings  EstimatedSavings                `json:"estimated_savings"`
}

// Bottleneck represents a detected bottleneck
type Bottleneck struct {
	Type        BottleneckType `json:"type"`
	Severity    Severity       `json:"severity"`
	Description string         `json:"description"`
	Impact      string         `json:"impact"`
	TestIDs     []string       `json:"test_ids"`
	Metrics     interface{}    `json:"metrics"`
}

// BottleneckType represents the type of bottleneck
type BottleneckType string

const (
	BottleneckQueue    BottleneckType = "queue"
	BottleneckResource BottleneckType = "resource"
	BottleneckIO       BottleneckType = "io"
	BottleneckNetwork  BottleneckType = "network"
	BottleneckMemory   BottleneckType = "memory"
	BottleneckCPU      BottleneckType = "cpu"
)

// Severity represents the severity of an issue
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// ResourceUsageAnalysis contains resource usage analysis
type ResourceUsageAnalysis struct {
	CPUUtilization     float64                    `json:"cpu_utilization"`
	MemoryUtilization  float64                    `json:"memory_utilization"`
	IOUtilization      float64                    `json:"io_utilization"`
	NetworkUtilization float64                    `json:"network_utilization"`
	Efficiency         float64                    `json:"efficiency"`
	Recommendations    []ResourceRecommendation   `json:"recommendations"`
}

// ResourceRecommendation represents a resource optimization recommendation
type ResourceRecommendation struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Savings     float64 `json:"savings"`
}

// SchedulingPlan contains optimized scheduling plan
type SchedulingPlan struct {
	Phases              []SchedulingPhase `json:"phases"`
	EstimatedTime       time.Duration     `json:"estimated_time"`
	ParallelismFactor   float64           `json:"parallelism_factor"`
	ResourceUtilization float64           `json:"resource_utilization"`
}

// SchedulingPhase represents a phase in the scheduling plan
type SchedulingPhase struct {
	Name         string        `json:"name"`
	Tests        []string      `json:"tests"`
	EstimatedTime time.Duration `json:"estimated_time"`
	Parallelism  int           `json:"parallelism"`
	Resources    ResourceNeeds `json:"resources"`
}

// CostAnalysis contains cost analysis results
type CostAnalysis struct {
	TotalCost           float64                    `json:"total_cost"`
	CostPerTest         float64                    `json:"cost_per_test"`
	CostPerMinute       float64                    `json:"cost_per_minute"`
	CostBreakdown       map[string]float64         `json:"cost_breakdown"`
	OptimizationSavings float64                    `json:"optimization_savings"`
	Recommendations     []CostOptimizationRecommendation `json:"recommendations"`
}

// CostOptimizationRecommendation represents a cost optimization recommendation
type CostOptimizationRecommendation struct {
	Type            string  `json:"type"`
	Description     string  `json:"description"`
	PotentialSavings float64 `json:"potential_savings"`
	Implementation  string  `json:"implementation"`
}

// DetectBottlenecks detects execution bottlenecks
func (d *BottleneckDetector) DetectBottlenecks(ctx context.Context, tests []TestCase) ([]Bottleneck, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	var bottlenecks []Bottleneck
	
	for _, test := range tests {
		metrics, exists := d.executionMetrics[test.ID]
		if !exists {
			continue
		}
		
		// Check for queue bottlenecks
		if metrics.QueueTime > d.thresholds.MaxQueueTime {
			bottlenecks = append(bottlenecks, Bottleneck{
				Type:        BottleneckQueue,
				Severity:    d.calculateSeverity(float64(metrics.QueueTime), float64(d.thresholds.MaxQueueTime)),
				Description: fmt.Sprintf("Test %s has excessive queue time: %v", test.Name, metrics.QueueTime),
				Impact:      fmt.Sprintf("Delays feedback by %v", metrics.QueueTime),
				TestIDs:     []string{test.ID},
				Metrics:     metrics,
			})
		}
		
		// Check for resource bottlenecks
		if metrics.ResourceWaitTime > d.thresholds.MaxResourceWaitTime {
			bottlenecks = append(bottlenecks, Bottleneck{
				Type:        BottleneckResource,
				Severity:    d.calculateSeverity(float64(metrics.ResourceWaitTime), float64(d.thresholds.MaxResourceWaitTime)),
				Description: fmt.Sprintf("Test %s waits too long for resources: %v", test.Name, metrics.ResourceWaitTime),
				Impact:      fmt.Sprintf("Resource contention causes %v delay", metrics.ResourceWaitTime),
				TestIDs:     []string{test.ID},
				Metrics:     metrics,
			})
		}
		
		// Check for variance bottlenecks
		if metrics.Variance > d.thresholds.MaxVariance {
			bottlenecks = append(bottlenecks, Bottleneck{
				Type:        BottleneckQueue, // High variance often indicates queueing issues
				Severity:    d.calculateSeverity(metrics.Variance, d.thresholds.MaxVariance),
				Description: fmt.Sprintf("Test %s has high execution variance: %.1f%%", test.Name, metrics.Variance*100),
				Impact:      "Unpredictable execution times affect scheduling",
				TestIDs:     []string{test.ID},
				Metrics:     metrics,
			})
		}
		
		// Check for throughput bottlenecks
		if metrics.ThroughputPerMin < d.thresholds.MinThroughput {
			bottlenecks = append(bottlenecks, Bottleneck{
				Type:        BottleneckResource,
				Severity:    d.calculateSeverity(d.thresholds.MinThroughput, metrics.ThroughputPerMin),
				Description: fmt.Sprintf("Test %s has low throughput: %.2f tests/min", test.Name, metrics.ThroughputPerMin),
				Impact:      "Low throughput increases total execution time",
				TestIDs:     []string{test.ID},
				Metrics:     metrics,
			})
		}
	}
	
	return bottlenecks, nil
}

// calculateSeverity calculates severity based on threshold violation
func (d *BottleneckDetector) calculateSeverity(actual, threshold float64) Severity {
	ratio := actual / threshold
	
	if ratio >= 3.0 {
		return SeverityCritical
	} else if ratio >= 2.0 {
		return SeverityHigh
	} else if ratio >= 1.5 {
		return SeverityMedium
	}
	
	return SeverityLow
}

// AnalyzeResourceUsage analyzes resource usage patterns
func (m *ResourceMonitor) AnalyzeResourceUsage(ctx context.Context, tests []TestCase) (*ResourceUsageAnalysis, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Calculate average utilization
	totalCPU := 0.0
	totalMemory := int64(0)
	totalIO := 0.0
	totalNetwork := 0.0
	count := 0
	
	for _, test := range tests {
		if cpuUsage, exists := m.cpuUsage[test.ID]; exists && len(cpuUsage) > 0 {
			for _, usage := range cpuUsage {
				totalCPU += usage
				count++
			}
		}
		
		if memUsage, exists := m.memoryUsage[test.ID]; exists && len(memUsage) > 0 {
			for _, usage := range memUsage {
				totalMemory += usage
			}
		}
		
		if ioUsage, exists := m.ioUsage[test.ID]; exists && len(ioUsage) > 0 {
			for _, usage := range ioUsage {
				totalIO += usage
			}
		}
		
		if netUsage, exists := m.networkUsage[test.ID]; exists && len(netUsage) > 0 {
			for _, usage := range netUsage {
				totalNetwork += usage
			}
		}
	}
	
	var recommendations []ResourceRecommendation
	
	avgCPU := totalCPU / float64(count)
	avgMemory := float64(totalMemory) / float64(count)
	avgIO := totalIO / float64(count)
	avgNetwork := totalNetwork / float64(count)
	
	// Generate recommendations based on usage patterns
	if avgCPU < 30 {
		recommendations = append(recommendations, ResourceRecommendation{
			Type:        "cpu_underutilization",
			Description: "CPU utilization is low, consider increasing parallelism",
			Impact:      "Could reduce total execution time by 30-50%",
			Savings:     0.4,
		})
	}
	
	if avgMemory > 8*1024*1024*1024 { // 8GB
		recommendations = append(recommendations, ResourceRecommendation{
			Type:        "memory_optimization",
			Description: "High memory usage detected, consider memory optimization",
			Impact:      "Could reduce memory pressure and improve stability",
			Savings:     0.2,
		})
	}
	
	efficiency := (avgCPU + avgIO + avgNetwork) / 300 // Normalize to 0-1
	if efficiency > 1.0 {
		efficiency = 1.0
	}
	
	return &ResourceUsageAnalysis{
		CPUUtilization:     avgCPU,
		MemoryUtilization:  avgMemory / (16 * 1024 * 1024 * 1024), // Normalize to 16GB
		IOUtilization:      avgIO,
		NetworkUtilization: avgNetwork,
		Efficiency:         efficiency,
		Recommendations:    recommendations,
	}, nil
}

// OptimizeScheduling optimizes test execution scheduling
func (s *ExecutionScheduler) OptimizeScheduling(ctx context.Context, tests []TestCase) (*SchedulingPlan, error) {
	// Sort tests by priority and estimated time
	sortedTests := make([]TestCase, len(tests))
	copy(sortedTests, tests)
	
	sort.Slice(sortedTests, func(i, j int) bool {
		// Prioritize by impact score, then by execution time
		if sortedTests[i].Impact.Overall != sortedTests[j].Impact.Overall {
			return sortedTests[i].Impact.Overall > sortedTests[j].Impact.Overall
		}
		return sortedTests[i].EstimatedTime < sortedTests[j].EstimatedTime
	})
	
	// Create scheduling phases
	var phases []SchedulingPhase
	
	// Phase 1: Fast, high-impact tests
	var fastTests []string
	var fastTime time.Duration
	for _, test := range sortedTests {
		if test.EstimatedTime < 30*time.Second && test.Impact.Overall > 0.7 {
			fastTests = append(fastTests, test.ID)
			fastTime += test.EstimatedTime
		}
	}
	
	if len(fastTests) > 0 {
		phases = append(phases, SchedulingPhase{
			Name:          "fast_critical",
			Tests:         fastTests,
			EstimatedTime: fastTime / 4, // Assume 4-way parallelism
			Parallelism:   4,
			Resources: ResourceNeeds{
				TotalCPU:    200, // 2 cores
				TotalMemory: 4 * 1024 * 1024 * 1024, // 4GB
				Workers:     4,
			},
		})
	}
	
	// Phase 2: Medium tests
	var mediumTests []string
	var mediumTime time.Duration
	for _, test := range sortedTests {
		if test.EstimatedTime >= 30*time.Second && test.EstimatedTime < 5*time.Minute {
			mediumTests = append(mediumTests, test.ID)
			mediumTime += test.EstimatedTime
		}
	}
	
	if len(mediumTests) > 0 {
		phases = append(phases, SchedulingPhase{
			Name:          "medium_parallel",
			Tests:         mediumTests,
			EstimatedTime: mediumTime / 2, // Assume 2-way parallelism
			Parallelism:   2,
			Resources: ResourceNeeds{
				TotalCPU:    400, // 4 cores
				TotalMemory: 8 * 1024 * 1024 * 1024, // 8GB
				Workers:     2,
			},
		})
	}
	
	// Phase 3: Slow tests
	var slowTests []string
	var slowTime time.Duration
	for _, test := range sortedTests {
		if test.EstimatedTime >= 5*time.Minute {
			slowTests = append(slowTests, test.ID)
			slowTime += test.EstimatedTime
		}
	}
	
	if len(slowTests) > 0 {
		phases = append(phases, SchedulingPhase{
			Name:          "slow_sequential",
			Tests:         slowTests,
			EstimatedTime: slowTime, // Sequential execution
			Parallelism:   1,
			Resources: ResourceNeeds{
				TotalCPU:    800, // 8 cores
				TotalMemory: 16 * 1024 * 1024 * 1024, // 16GB
				Workers:     1,
			},
		})
	}
	
	// Calculate total estimated time
	totalTime := time.Duration(0)
	for _, phase := range phases {
		totalTime += phase.EstimatedTime
	}
	
	return &SchedulingPlan{
		Phases:              phases,
		EstimatedTime:       totalTime,
		ParallelismFactor:   s.calculateParallelismFactor(phases),
		ResourceUtilization: s.calculateResourceUtilization(phases),
	}, nil
}

// calculateParallelismFactor calculates the parallelism factor
func (s *ExecutionScheduler) calculateParallelismFactor(phases []SchedulingPhase) float64 {
	totalParallelism := 0
	for _, phase := range phases {
		totalParallelism += phase.Parallelism
	}
	
	if len(phases) == 0 {
		return 1.0
	}
	
	return float64(totalParallelism) / float64(len(phases))
}

// calculateResourceUtilization calculates resource utilization
func (s *ExecutionScheduler) calculateResourceUtilization(phases []SchedulingPhase) float64 {
	maxCPU := 0.0
	maxMemory := int64(0)
	
	for _, phase := range phases {
		if phase.Resources.TotalCPU > maxCPU {
			maxCPU = phase.Resources.TotalCPU
		}
		if phase.Resources.TotalMemory > maxMemory {
			maxMemory = phase.Resources.TotalMemory
		}
	}
	
	cpuUtil := maxCPU / s.resourceLimits.MaxCPU
	memUtil := float64(maxMemory) / float64(s.resourceLimits.MaxMemory)
	
	return (cpuUtil + memUtil) / 2.0
}

// AnalyzeCosts analyzes test execution costs
func (c *CostAnalyzer) AnalyzeCosts(ctx context.Context, tests []TestCase) (*CostAnalysis, error) {
	totalCost := 0.0
	costBreakdown := make(map[string]float64)
	
	for _, test := range tests {
		metrics, exists := c.costMetrics[test.ID]
		if !exists {
			// Calculate cost for new test
			metrics = c.calculateTestCost(test)
			c.costMetrics[test.ID] = metrics
		}
		
		totalCost += metrics.TotalCost
		costBreakdown["compute"] += metrics.ComputeCost
		costBreakdown["storage"] += metrics.StorageCost
		costBreakdown["network"] += metrics.NetworkCost
	}
	
	avgCostPerTest := totalCost / float64(len(tests))
	
	// Generate cost optimization recommendations
	var recommendations []CostOptimizationRecommendation
	
	if costBreakdown["compute"] > totalCost*0.7 {
		recommendations = append(recommendations, CostOptimizationRecommendation{
			Type:             "compute_optimization",
			Description:      "Compute costs are high, consider optimizing resource usage",
			PotentialSavings: costBreakdown["compute"] * 0.3,
			Implementation:   "Optimize parallel execution and resource allocation",
		})
	}
	
	return &CostAnalysis{
		TotalCost:           totalCost,
		CostPerTest:         avgCostPerTest,
		CostPerMinute:       totalCost / 60, // Assume 1 hour execution
		CostBreakdown:       costBreakdown,
		OptimizationSavings: c.calculateOptimizationSavings(recommendations),
		Recommendations:     recommendations,
	}, nil
}

// calculateTestCost calculates cost for a single test
func (c *CostAnalyzer) calculateTestCost(test TestCase) *CostMetrics {
	executionHours := test.EstimatedTime.Hours()
	
	computeCost := (test.ResourceUsage.CPU / 100) * c.pricingModel.CPUCostPerHour * executionHours
	memoryCost := float64(test.ResourceUsage.Memory) / (1024 * 1024 * 1024) * c.pricingModel.MemoryCostPerGBHour * executionHours
	networkCost := 0.0 // Simplified - would need actual network usage data
	
	totalCost := computeCost + memoryCost + networkCost
	
	return &CostMetrics{
		TestID:             test.ID,
		ComputeCost:        computeCost,
		StorageCost:        memoryCost,
		NetworkCost:        networkCost,
		TotalCost:          totalCost,
		CostPerExecution:   totalCost,
		CostPerMinute:      totalCost / test.EstimatedTime.Minutes(),
		ResourceEfficiency: c.calculateResourceEfficiency(test),
	}
}

// calculateResourceEfficiency calculates resource efficiency
func (c *CostAnalyzer) calculateResourceEfficiency(test TestCase) float64 {
	// Simple efficiency calculation based on resource utilization
	cpuEfficiency := test.ResourceUsage.CPU / 100
	memoryEfficiency := 1.0 // Simplified
	
	return (cpuEfficiency + memoryEfficiency) / 2.0
}

// calculateOptimizationSavings calculates potential savings from optimizations
func (c *CostAnalyzer) calculateOptimizationSavings(recommendations []CostOptimizationRecommendation) float64 {
	totalSavings := 0.0
	for _, rec := range recommendations {
		totalSavings += rec.PotentialSavings
	}
	return totalSavings
}

// GenerateRecommendations generates optimization recommendations
func (e *OptimizationEngine) GenerateRecommendations(ctx context.Context, bottlenecks []Bottleneck, resourceUsage *ResourceUsageAnalysis) ([]OptimizationRecommendation, error) {
	var recommendations []OptimizationRecommendation
	
	// Analyze bottlenecks and generate recommendations
	for _, bottleneck := range bottlenecks {
		switch bottleneck.Type {
		case BottleneckQueue:
			recommendations = append(recommendations, OptimizationRecommendation{
				Type:        OptimizationParallel,
				Priority:    PriorityHigh,
				Description: "Increase parallel execution to reduce queue times",
				ExpectedImpact: "Reduce queue time by 50-70%",
				Implementation: "Configure additional worker processes",
				EstimatedSavings: EstimatedSavings{
					ExecutionTime: 5 * time.Minute,
					ResourceUsage: 0.1,
					Maintenance:   0.0,
				},
				Automated: true,
			})
			
		case BottleneckResource:
			recommendations = append(recommendations, OptimizationRecommendation{
				Type:        OptimizationResource,
				Priority:    PriorityMedium,
				Description: "Optimize resource allocation and scheduling",
				ExpectedImpact: "Reduce resource wait time by 40-60%",
				Implementation: "Implement resource pooling and better scheduling",
				EstimatedSavings: EstimatedSavings{
					ExecutionTime: 3 * time.Minute,
					ResourceUsage: 0.3,
					Maintenance:   0.1,
				},
				Automated: false,
			})
		}
	}
	
	// Analyze resource usage and generate recommendations
	if resourceUsage.CPUUtilization < 30 {
		recommendations = append(recommendations, OptimizationRecommendation{
			Type:        OptimizationParallel,
			Priority:    PriorityMedium,
			Description: "Increase parallelism to better utilize CPU resources",
			ExpectedImpact: "Improve CPU utilization by 40-60%",
			Implementation: "Increase parallel worker count",
			EstimatedSavings: EstimatedSavings{
				ExecutionTime: 10 * time.Minute,
				ResourceUsage: 0.4,
				Maintenance:   0.0,
			},
			Automated: true,
		})
	}
	
	if resourceUsage.Efficiency < 0.6 {
		recommendations = append(recommendations, OptimizationRecommendation{
			Type:        OptimizationScheduling,
			Priority:    PriorityMedium,
			Description: "Optimize test scheduling to improve resource efficiency",
			ExpectedImpact: "Improve overall efficiency by 20-30%",
			Implementation: "Implement intelligent test batching and scheduling",
			EstimatedSavings: EstimatedSavings{
				ExecutionTime: 8 * time.Minute,
				ResourceUsage: 0.25,
				Maintenance:   0.05,
			},
			Automated: false,
		})
	}
	
	// Sort recommendations by priority
	sort.Slice(recommendations, func(i, j int) bool {
		return getPriorityValue(recommendations[i].Priority) > getPriorityValue(recommendations[j].Priority)
	})
	
	return recommendations, nil
}

// calculateTotalSavings calculates total estimated savings
func (o *TestExecutionOptimizer) calculateTotalSavings(recommendations []OptimizationRecommendation) EstimatedSavings {
	var total EstimatedSavings
	
	for _, rec := range recommendations {
		total.ExecutionTime += rec.EstimatedSavings.ExecutionTime
		total.ResourceUsage += rec.EstimatedSavings.ResourceUsage
		total.Maintenance += rec.EstimatedSavings.Maintenance
	}
	
	return total
}

// Helper functions for optimization engine
func (e *OptimizationEngine) analyzeParallelOpportunities(metrics *ExecutionMetrics) []string {
	var issues []string
	
	if metrics.QueueTime > 1*time.Minute {
		issues = append(issues, "High queue time indicates insufficient parallelism")
	}
	
	if metrics.ThroughputPerMin < 2.0 {
		issues = append(issues, "Low throughput suggests parallel execution opportunities")
	}
	
	return issues
}

func (e *OptimizationEngine) recommendParallelOptimization(metrics *ExecutionMetrics) []OptimizationRecommendation {
	return []OptimizationRecommendation{
		{
			Type:        OptimizationParallel,
			Priority:    PriorityHigh,
			Description: "Increase parallel worker count to reduce queue time",
			ExpectedImpact: fmt.Sprintf("Reduce queue time from %v to ~%v", metrics.QueueTime, metrics.QueueTime/2),
			Implementation: "Configure additional parallel workers",
			Automated: true,
		},
	}
}

func (e *OptimizationEngine) analyzeResourceBottlenecks(metrics *ExecutionMetrics) []string {
	var issues []string
	
	if metrics.ResourceWaitTime > 30*time.Second {
		issues = append(issues, "High resource wait time indicates resource contention")
	}
	
	return issues
}

func (e *OptimizationEngine) recommendResourceOptimization(metrics *ExecutionMetrics) []OptimizationRecommendation {
	return []OptimizationRecommendation{
		{
			Type:        OptimizationResource,
			Priority:    PriorityMedium,
			Description: "Optimize resource allocation to reduce wait times",
			ExpectedImpact: fmt.Sprintf("Reduce resource wait time from %v", metrics.ResourceWaitTime),
			Implementation: "Implement resource pooling and better allocation",
			Automated: false,
		},
	}
}

func (e *OptimizationEngine) analyzeSchedulingInefficiencies(metrics *ExecutionMetrics) []string {
	var issues []string
	
	if metrics.Variance > 0.3 {
		issues = append(issues, "High execution variance indicates scheduling inefficiencies")
	}
	
	return issues
}

func (e *OptimizationEngine) recommendSchedulingOptimization(metrics *ExecutionMetrics) []OptimizationRecommendation {
	return []OptimizationRecommendation{
		{
			Type:        OptimizationScheduling,
			Priority:    PriorityMedium,
			Description: "Improve test scheduling to reduce execution variance",
			ExpectedImpact: fmt.Sprintf("Reduce variance from %.1f%% to <20%%", metrics.Variance*100),
			Implementation: "Implement predictive scheduling algorithms",
			Automated: false,
		},
	}
}

// getPriorityValue returns numeric value for priority comparison
func getPriorityValue(priority Priority) int {
	switch priority {
	case PriorityCritical:
		return 4
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 0
	}
}