package testing

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// TestMaintenanceAdvisor provides recommendations for test suite maintenance
type TestMaintenanceAdvisor struct {
	testRegistry    *TestRegistry
	metricsCollector *TestMetricsCollector
	consolidator    *TestConsolidator
}

// MaintenanceRecommendation represents a maintenance recommendation
type MaintenanceRecommendation struct {
	Type        RecommendationType `json:"type"`
	Priority    Priority          `json:"priority"`
	Description string            `json:"description"`
	Impact      string            `json:"impact"`
	Effort      string            `json:"effort"`
	TestIDs     []string          `json:"test_ids"`
	Actions     []MaintenanceAction `json:"actions"`
	Savings     EstimatedSavings   `json:"savings"`
}

// RecommendationType represents the type of maintenance recommendation
type RecommendationType string

const (
	RecommendationConsolidate    RecommendationType = "consolidate"
	RecommendationRemoveDuplicate RecommendationType = "remove_duplicate"
	RecommendationOptimize       RecommendationType = "optimize"
	RecommendationRefactor       RecommendationType = "refactor"
	RecommendationCleanup        RecommendationType = "cleanup"
	RecommendationSplit          RecommendationType = "split"
)

// Priority represents the priority level of a recommendation
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// MaintenanceAction represents a specific action to take
type MaintenanceAction struct {
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters"`
	Automated   bool              `json:"automated"`
}

// EstimatedSavings represents estimated savings from maintenance
type EstimatedSavings struct {
	ExecutionTime time.Duration `json:"execution_time"`
	ResourceUsage float64       `json:"resource_usage"`
	Maintenance   float64       `json:"maintenance"`
}

// TestConsolidator handles test consolidation logic
type TestConsolidator struct {
	similarityThreshold float64
}

// TestMetricsCollector collects test metrics for analysis
type TestMetricsCollector struct {
	executionHistory map[string][]ExecutionRecord
}

// ExecutionRecord represents a test execution record
type ExecutionRecord struct {
	TestID      string        `json:"test_id"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
	Timestamp   time.Time     `json:"timestamp"`
	Resources   ResourceUsage `json:"resources"`
}

// CoverageData represents test coverage information
type CoverageData struct {
	Files       []string  `json:"files"`
	Lines       []int     `json:"lines"`
	Branches    []int     `json:"branches"`
	Coverage    float64   `json:"coverage"`
}

// TestRegistry manages test registration and metadata
type TestRegistry struct {
	tests map[string]TestCase
}

// NewTestMaintenanceAdvisor creates a new test maintenance advisor
func NewTestMaintenanceAdvisor() *TestMaintenanceAdvisor {
	return &TestMaintenanceAdvisor{
		testRegistry:     NewTestRegistry(),
		metricsCollector: NewTestMetricsCollector(),
		consolidator:     NewTestConsolidator(),
	}
}

// NewTestRegistry creates a new test registry
func NewTestRegistry() *TestRegistry {
	return &TestRegistry{
		tests: make(map[string]TestCase),
	}
}

// NewTestMetricsCollector creates a new test metrics collector
func NewTestMetricsCollector() *TestMetricsCollector {
	return &TestMetricsCollector{
		executionHistory: make(map[string][]ExecutionRecord),
	}
}

// NewTestConsolidator creates a new test consolidator
func NewTestConsolidator() *TestConsolidator {
	return &TestConsolidator{
		similarityThreshold: 0.8,
	}
}

// AnalyzeTestSuite analyzes the test suite and provides maintenance recommendations
func (a *TestMaintenanceAdvisor) AnalyzeTestSuite(ctx context.Context) ([]MaintenanceRecommendation, error) {
	log.Println("Analyzing test suite for maintenance recommendations")
	
	var recommendations []MaintenanceRecommendation
	
	// Get all tests
	tests, err := a.testRegistry.GetAllTests(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tests: %w", err)
	}
	
	// Analyze for different types of maintenance opportunities
	consolidationRecs := a.analyzeConsolidationOpportunities(tests)
	recommendations = append(recommendations, consolidationRecs...)
	
	duplicateRecs := a.analyzeDuplicateTests(tests)
	recommendations = append(recommendations, duplicateRecs...)
	
	optimizationRecs := a.analyzeOptimizationOpportunities(tests)
	recommendations = append(recommendations, optimizationRecs...)
	
	cleanupRecs := a.analyzeCleanupOpportunities(tests)
	recommendations = append(recommendations, cleanupRecs...)
	
	// Sort by priority and impact
	sort.Slice(recommendations, func(i, j int) bool {
		return a.getPriorityScore(recommendations[i].Priority) > a.getPriorityScore(recommendations[j].Priority)
	})
	
	log.Printf("Generated %d maintenance recommendations", len(recommendations))
	return recommendations, nil
}

// analyzeConsolidationOpportunities identifies tests that can be consolidated
func (a *TestMaintenanceAdvisor) analyzeConsolidationOpportunities(tests []TestCase) []MaintenanceRecommendation {
	var recommendations []MaintenanceRecommendation
	
	// Group tests by package
	packageGroups := make(map[string][]TestCase)
	for _, test := range tests {
		packageGroups[test.Package] = append(packageGroups[test.Package], test)
	}
	
	// Analyze each package for consolidation opportunities
	for pkg, pkgTests := range packageGroups {
		if len(pkgTests) < 3 {
			continue // Need at least 3 tests to consider consolidation
		}
		
		// Look for tests with similar setup/teardown
		similarGroups := a.findSimilarTests(pkgTests)
		
		for _, group := range similarGroups {
			if len(group) >= 3 {
				var testIDs []string
				var totalTime time.Duration
				
				for _, test := range group {
					testIDs = append(testIDs, test.ID)
					totalTime += test.EstimatedTime
				}
				
				// Estimate savings from consolidation
				estimatedSavings := totalTime / 3 // Assume 66% time savings
				
				recommendations = append(recommendations, MaintenanceRecommendation{
					Type:        RecommendationConsolidate,
					Priority:    a.calculateConsolidationPriority(group),
					Description: fmt.Sprintf("Consolidate %d similar tests in package %s", len(group), pkg),
					Impact:      fmt.Sprintf("Reduce execution time by ~%v", estimatedSavings),
					Effort:      "Medium",
					TestIDs:     testIDs,
					Actions: []MaintenanceAction{
						{
							Type:        "consolidate_tests",
							Description: "Merge similar test cases into a single parameterized test",
							Parameters: map[string]string{
								"package": pkg,
								"method":  "table_driven_test",
							},
							Automated: false,
						},
					},
					Savings: EstimatedSavings{
						ExecutionTime: estimatedSavings,
						ResourceUsage: 0.3, // 30% resource savings
						Maintenance:   0.5, // 50% maintenance savings
					},
				})
			}
		}
	}
	
	return recommendations
}

// analyzeDuplicateTests identifies duplicate or near-duplicate tests
func (a *TestMaintenanceAdvisor) analyzeDuplicateTests(tests []TestCase) []MaintenanceRecommendation {
	var recommendations []MaintenanceRecommendation
	
	// Compare each test with every other test
	for i, test1 := range tests {
		for j, test2 := range tests[i+1:] {
			similarity := a.calculateTestSimilarity(test1, test2)
			
			if similarity > 0.9 { // Very high similarity indicates potential duplicate
				recommendations = append(recommendations, MaintenanceRecommendation{
					Type:        RecommendationRemoveDuplicate,
					Priority:    PriorityMedium,
					Description: fmt.Sprintf("Tests %s and %s appear to be duplicates (%.1f%% similar)", test1.Name, test2.Name, similarity*100),
					Impact:      fmt.Sprintf("Remove redundant test execution (~%v saved)", test2.EstimatedTime),
					Effort:      "Low",
					TestIDs:     []string{test1.ID, test2.ID},
					Actions: []MaintenanceAction{
						{
							Type:        "remove_duplicate",
							Description: "Remove or merge duplicate test",
							Parameters: map[string]string{
								"keep":   test1.ID,
								"remove": test2.ID,
							},
							Automated: false,
						},
					},
					Savings: EstimatedSavings{
						ExecutionTime: test2.EstimatedTime,
						ResourceUsage: 0.5,
						Maintenance:   0.5,
					},
				})
			}
		}
	}
	
	return recommendations
}

// analyzeOptimizationOpportunities identifies tests that can be optimized
func (a *TestMaintenanceAdvisor) analyzeOptimizationOpportunities(tests []TestCase) []MaintenanceRecommendation {
	var recommendations []MaintenanceRecommendation
	
	for _, test := range tests {
		// Identify slow tests that could be optimized
		if test.EstimatedTime > 60*time.Second {
			priority := PriorityMedium
			if test.EstimatedTime > 300*time.Second { // 5 minutes
				priority = PriorityHigh
			}
			
			recommendations = append(recommendations, MaintenanceRecommendation{
				Type:        RecommendationOptimize,
				Priority:    priority,
				Description: fmt.Sprintf("Test %s is slow (%v) and could be optimized", test.Name, test.EstimatedTime),
				Impact:      fmt.Sprintf("Potential 30-50%% execution time reduction"),
				Effort:      "Medium",
				TestIDs:     []string{test.ID},
				Actions: []MaintenanceAction{
					{
						Type:        "optimize_test",
						Description: "Analyze and optimize slow test execution",
						Parameters: map[string]string{
							"test_id": test.ID,
							"target":  "reduce_by_30_percent",
						},
						Automated: false,
					},
				},
				Savings: EstimatedSavings{
					ExecutionTime: test.EstimatedTime / 3, // Assume 33% improvement
					ResourceUsage: 0.2,
					Maintenance:   0.1,
				},
			})
		}
		
		// Identify tests with high resource usage
		if test.ResourceUsage.Memory > 1024*1024*1024 { // 1GB
			recommendations = append(recommendations, MaintenanceRecommendation{
				Type:        RecommendationOptimize,
				Priority:    PriorityMedium,
				Description: fmt.Sprintf("Test %s uses excessive memory (%d MB)", test.Name, test.ResourceUsage.Memory/(1024*1024)),
				Impact:      "Reduce memory pressure and improve parallel execution",
				Effort:      "Medium",
				TestIDs:     []string{test.ID},
				Actions: []MaintenanceAction{
					{
						Type:        "optimize_memory",
						Description: "Optimize memory usage in test",
						Parameters: map[string]string{
							"test_id": test.ID,
							"target":  "reduce_memory_50_percent",
						},
						Automated: false,
					},
				},
				Savings: EstimatedSavings{
					ExecutionTime: 0,
					ResourceUsage: 0.5,
					Maintenance:   0.1,
				},
			})
		}
	}
	
	return recommendations
}

// analyzeCleanupOpportunities identifies tests that need cleanup
func (a *TestMaintenanceAdvisor) analyzeCleanupOpportunities(tests []TestCase) []MaintenanceRecommendation {
	var recommendations []MaintenanceRecommendation
	
	for _, test := range tests {
		// Identify flaky tests that need attention
		if test.Flakiness > 0.2 { // 20% flakiness
			priority := PriorityMedium
			if test.Flakiness > 0.5 {
				priority = PriorityHigh
			}
			
			recommendations = append(recommendations, MaintenanceRecommendation{
				Type:        RecommendationCleanup,
				Priority:    priority,
				Description: fmt.Sprintf("Test %s is flaky (%.1f%% failure rate) and needs stabilization", test.Name, test.Flakiness*100),
				Impact:      "Improve CI/CD reliability and reduce false failures",
				Effort:      "Medium",
				TestIDs:     []string{test.ID},
				Actions: []MaintenanceAction{
					{
						Type:        "stabilize_test",
						Description: "Investigate and fix flaky test behavior",
						Parameters: map[string]string{
							"test_id":   test.ID,
							"flakiness": fmt.Sprintf("%.2f", test.Flakiness),
						},
						Automated: false,
					},
				},
				Savings: EstimatedSavings{
					ExecutionTime: 0,
					ResourceUsage: 0,
					Maintenance:   0.8, // High maintenance savings from fixing flaky tests
				},
			})
		}
		
		// Identify tests with low success rate
		if test.SuccessRate < 0.8 { // Less than 80% success rate
			recommendations = append(recommendations, MaintenanceRecommendation{
				Type:        RecommendationCleanup,
				Priority:    PriorityHigh,
				Description: fmt.Sprintf("Test %s has low success rate (%.1f%%) and needs investigation", test.Name, test.SuccessRate*100),
				Impact:      "Improve test reliability and reduce debugging time",
				Effort:      "High",
				TestIDs:     []string{test.ID},
				Actions: []MaintenanceAction{
					{
						Type:        "investigate_failures",
						Description: "Investigate and fix test failures",
						Parameters: map[string]string{
							"test_id":      test.ID,
							"success_rate": fmt.Sprintf("%.2f", test.SuccessRate),
						},
						Automated: false,
					},
				},
				Savings: EstimatedSavings{
					ExecutionTime: 0,
					ResourceUsage: 0,
					Maintenance:   1.0, // Very high maintenance savings
				},
			})
		}
	}
	
	return recommendations
}

// findSimilarTests finds groups of similar tests
func (a *TestMaintenanceAdvisor) findSimilarTests(tests []TestCase) [][]TestCase {
	var groups [][]TestCase
	used := make(map[string]bool)
	
	for _, test1 := range tests {
		if used[test1.ID] {
			continue
		}
		
		var group []TestCase
		group = append(group, test1)
		used[test1.ID] = true
		
		for _, test2 := range tests {
			if used[test2.ID] {
				continue
			}
			
			similarity := a.calculateTestSimilarity(test1, test2)
			if similarity > a.consolidator.similarityThreshold {
				group = append(group, test2)
				used[test2.ID] = true
			}
		}
		
		if len(group) > 1 {
			groups = append(groups, group)
		}
	}
	
	return groups
}

// calculateTestSimilarity calculates similarity between two tests
func (a *TestMaintenanceAdvisor) calculateTestSimilarity(test1, test2 TestCase) float64 {
	similarity := 0.0
	
	// Package similarity (40% weight)
	if test1.Package == test2.Package {
		similarity += 0.4
	}
	
	// Name similarity (30% weight)
	nameSimilarity := a.calculateStringSimilarity(test1.Name, test2.Name)
	similarity += nameSimilarity * 0.3
	
	// Tag similarity (20% weight)
	tagSimilarity := a.calculateTagSimilarity(test1.Tags, test2.Tags)
	similarity += tagSimilarity * 0.2
	
	// Execution time similarity (10% weight)
	timeDiff := float64(abs(int64(test1.EstimatedTime - test2.EstimatedTime)))
	maxTime := float64(max(int64(test1.EstimatedTime), int64(test2.EstimatedTime)))
	if maxTime > 0 {
		timeSimilarity := 1.0 - (timeDiff / maxTime)
		similarity += timeSimilarity * 0.1
	}
	
	return similarity
}

// calculateStringSimilarity calculates similarity between two strings
func (a *TestMaintenanceAdvisor) calculateStringSimilarity(s1, s2 string) float64 {
	// Simple Jaccard similarity based on words
	words1 := strings.Fields(strings.ToLower(s1))
	words2 := strings.Fields(strings.ToLower(s2))
	
	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	// Create sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	
	for _, word := range words1 {
		set1[word] = true
	}
	
	for _, word := range words2 {
		set2[word] = true
	}
	
	// Calculate intersection and union
	intersection := 0
	union := len(set1)
	
	for word := range set2 {
		if set1[word] {
			intersection++
		} else {
			union++
		}
	}
	
	if union == 0 {
		return 1.0
	}
	
	return float64(intersection) / float64(union)
}

// calculateTagSimilarity calculates similarity between two tag sets
func (a *TestMaintenanceAdvisor) calculateTagSimilarity(tags1, tags2 []string) float64 {
	if len(tags1) == 0 && len(tags2) == 0 {
		return 1.0
	}
	
	if len(tags1) == 0 || len(tags2) == 0 {
		return 0.0
	}
	
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	
	for _, tag := range tags1 {
		set1[tag] = true
	}
	
	for _, tag := range tags2 {
		set2[tag] = true
	}
	
	intersection := 0
	union := len(set1)
	
	for tag := range set2 {
		if set1[tag] {
			intersection++
		} else {
			union++
		}
	}
	
	if union == 0 {
		return 1.0
	}
	
	return float64(intersection) / float64(union)
}

// calculateConsolidationPriority calculates priority for consolidation recommendation
func (a *TestMaintenanceAdvisor) calculateConsolidationPriority(tests []TestCase) Priority {
	var totalTime time.Duration
	for _, test := range tests {
		totalTime += test.EstimatedTime
	}
	
	// Higher priority for larger time savings
	if totalTime > 10*time.Minute {
		return PriorityHigh
	} else if totalTime > 5*time.Minute {
		return PriorityMedium
	}
	
	return PriorityLow
}

// getPriorityScore returns numeric score for priority
func (a *TestMaintenanceAdvisor) getPriorityScore(priority Priority) int {
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

// GetAllTests returns all registered tests
func (r *TestRegistry) GetAllTests(ctx context.Context) ([]TestCase, error) {
	var tests []TestCase
	for _, test := range r.tests {
		tests = append(tests, test)
	}
	return tests, nil
}

// RegisterTest registers a test in the registry
func (r *TestRegistry) RegisterTest(test TestCase) {
	r.tests[test.ID] = test
}

// Helper functions
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}