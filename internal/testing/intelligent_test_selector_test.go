package testing

import (
	"context"
	"testing"
	"time"
)

func TestIntelligentTestSelector_SelectTests(t *testing.T) {
	selector := NewIntelligentTestSelector()
	
	// Create test cases
	tests := []TestCase{
		{
			ID:            "test1",
			Name:          "Fast Unit Test",
			Package:       "internal/models",
			EstimatedTime: 10 * time.Second,
			Tags:          []string{"unit", "fast"},
			Impact: ImpactScore{
				Overall: 0.8,
			},
			Flakiness:   0.1,
			SuccessRate: 0.95,
			CoverageData: CoverageData{
				Files: []string{"models/user.go", "models/article.go"},
			},
		},
		{
			ID:            "test2",
			Name:          "Integration Test",
			Package:       "internal/services",
			EstimatedTime: 60 * time.Second,
			Tags:          []string{"integration", "database"},
			Impact: ImpactScore{
				Overall: 0.9,
			},
			Flakiness:   0.05,
			SuccessRate: 0.98,
			CoverageData: CoverageData{
				Files: []string{"services/article_service.go"},
			},
		},
		{
			ID:            "test3",
			Name:          "Slow E2E Test",
			Package:       "internal/api",
			EstimatedTime: 300 * time.Second,
			Tags:          []string{"e2e", "slow"},
			Impact: ImpactScore{
				Overall: 0.6,
			},
			Flakiness:   0.3,
			SuccessRate: 0.85,
			CoverageData: CoverageData{
				Files: []string{"api/handlers.go"},
			},
		},
	}
	
	// Register tests
	for _, test := range tests {
		selector.testRegistry.RegisterTest(test)
	}
	
	ctx := context.Background()
	
	t.Run("Fast Priority Selection", func(t *testing.T) {
		request := TestSelectionRequest{
			ChangedFiles:    []string{"internal/models/user.go"},
			TimeLimit:       2 * time.Minute,
			Priority:        PriorityFast,
			ParallelWorkers: 2,
			TargetCoverage:  0.8,
		}
		
		result, err := selector.SelectTests(ctx, request)
		if err != nil {
			t.Fatalf("SelectTests failed: %v", err)
		}
		
		if len(result.SelectedTests) == 0 {
			t.Error("Expected at least one test to be selected")
		}
		
		// Should prioritize fast tests
		for _, test := range result.SelectedTests {
			if test.EstimatedTime > 30*time.Second {
				t.Errorf("Fast priority should not select slow tests, got %v", test.EstimatedTime)
			}
		}
		
		if result.EstimatedTime > request.TimeLimit {
			t.Errorf("Estimated time %v exceeds time limit %v", result.EstimatedTime, request.TimeLimit)
		}
	})
	
	t.Run("Balanced Priority Selection", func(t *testing.T) {
		request := TestSelectionRequest{
			ChangedFiles:    []string{"internal/services/article_service.go"},
			Priority:        PriorityBalanced,
			ParallelWorkers: 4,
			TargetCoverage:  0.9,
		}
		
		result, err := selector.SelectTests(ctx, request)
		if err != nil {
			t.Fatalf("SelectTests failed: %v", err)
		}
		
		if len(result.SelectedTests) == 0 {
			t.Error("Expected at least one test to be selected")
		}
		
		if len(result.Reasoning) == 0 {
			t.Error("Expected reasoning to be provided")
		}
		
		if result.ExecutionPlan == nil {
			t.Error("Expected execution plan to be generated")
		}
	})
	
	t.Run("Critical Priority Selection", func(t *testing.T) {
		request := TestSelectionRequest{
			ChangedFiles:    []string{"internal/api/handlers.go"},
			Priority:        PriorityCritical,
			ParallelWorkers: 1,
		}
		
		result, err := selector.SelectTests(ctx, request)
		if err != nil {
			t.Fatalf("SelectTests failed: %v", err)
		}
		
		// Should only select critical tests (high impact or tagged as critical)
		for _, test := range result.SelectedTests {
			if test.Impact.Overall < 0.8 && !containsString(test.Tags, "critical") {
				t.Errorf("Critical priority should only select critical tests, got test with impact %f", test.Impact.Overall)
			}
		}
	})
}

func TestImpactAnalyzer_AnalyzeImpact(t *testing.T) {
	analyzer := NewImpactAnalyzer()
	
	ctx := context.Background()
	changedFiles := []string{
		"internal/models/user.go",
		"internal/services/auth_service.go",
	}
	
	analysis, err := analyzer.AnalyzeImpact(ctx, changedFiles)
	if err != nil {
		t.Fatalf("AnalyzeImpact failed: %v", err)
	}
	
	if len(analysis.AffectedFiles) == 0 {
		t.Error("Expected affected files to be identified")
	}
	
	if len(analysis.AffectedPackages) == 0 {
		t.Error("Expected affected packages to be identified")
	}
	
	if analysis.RiskLevel == "" {
		t.Error("Expected risk level to be calculated")
	}
	
	// Check that changed files are marked as affected
	for _, file := range changedFiles {
		if !analysis.AffectedFiles[file] {
			t.Errorf("Changed file %s should be marked as affected", file)
		}
	}
}

func TestParallelExecutionOptimizer_OptimizeExecution(t *testing.T) {
	optimizer := NewParallelExecutionOptimizer()
	
	tests := []TestCase{
		{
			ID:            "fast1",
			EstimatedTime: 5 * time.Second,
			ResourceUsage: ResourceUsage{CPU: 20, Memory: 100 * 1024 * 1024},
		},
		{
			ID:            "fast2",
			EstimatedTime: 8 * time.Second,
			ResourceUsage: ResourceUsage{CPU: 25, Memory: 150 * 1024 * 1024},
		},
		{
			ID:            "medium1",
			EstimatedTime: 45 * time.Second,
			ResourceUsage: ResourceUsage{CPU: 50, Memory: 500 * 1024 * 1024},
		},
		{
			ID:            "slow1",
			EstimatedTime: 180 * time.Second,
			ResourceUsage: ResourceUsage{CPU: 80, Memory: 1024 * 1024 * 1024},
		},
	}
	
	ctx := context.Background()
	plan, err := optimizer.OptimizeExecution(ctx, tests, 4)
	if err != nil {
		t.Fatalf("OptimizeExecution failed: %v", err)
	}
	
	if len(plan.Phases) == 0 {
		t.Error("Expected execution phases to be created")
	}
	
	if len(plan.ParallelGroups) == 0 {
		t.Error("Expected parallel groups to be created")
	}
	
	if plan.TotalEstimated == 0 {
		t.Error("Expected total estimated time to be calculated")
	}
	
	// Verify that fast tests are in parallel phases
	for _, phase := range plan.Phases {
		if phase.Name == "fast_parallel" && !phase.Parallel {
			t.Error("Fast phase should be parallel")
		}
	}
}

func TestTestMaintenanceAdvisor_AnalyzeTestSuite(t *testing.T) {
	advisor := NewTestMaintenanceAdvisor()
	
	// Create test cases with maintenance issues
	tests := []TestCase{
		{
			ID:            "flaky_test",
			Name:          "Flaky Test",
			Package:       "internal/services",
			EstimatedTime: 30 * time.Second,
			Flakiness:     0.4, // High flakiness
			SuccessRate:   0.7, // Low success rate
		},
		{
			ID:            "slow_test",
			Name:          "Slow Test",
			Package:       "internal/services",
			EstimatedTime: 600 * time.Second, // Very slow
			Flakiness:     0.1,
			SuccessRate:   0.95,
		},
		{
			ID:            "similar_test1",
			Name:          "User Service Test Create",
			Package:       "internal/services",
			EstimatedTime: 20 * time.Second,
			Tags:          []string{"unit", "user"},
		},
		{
			ID:            "similar_test2",
			Name:          "User Service Test Update",
			Package:       "internal/services",
			EstimatedTime: 25 * time.Second,
			Tags:          []string{"unit", "user"},
		},
		{
			ID:            "similar_test3",
			Name:          "User Service Test Delete",
			Package:       "internal/services",
			EstimatedTime: 22 * time.Second,
			Tags:          []string{"unit", "user"},
		},
	}
	
	// Register tests
	for _, test := range tests {
		advisor.testRegistry.RegisterTest(test)
	}
	
	ctx := context.Background()
	recommendations, err := advisor.AnalyzeTestSuite(ctx)
	if err != nil {
		t.Fatalf("AnalyzeTestSuite failed: %v", err)
	}
	
	if len(recommendations) == 0 {
		t.Error("Expected maintenance recommendations to be generated")
	}
	
	// Check for specific recommendation types
	hasFlakiness := false
	hasOptimization := false
	hasConsolidation := false
	
	for _, rec := range recommendations {
		switch rec.Type {
		case RecommendationCleanup:
			hasFlakiness = true
		case RecommendationOptimize:
			hasOptimization = true
		case RecommendationConsolidate:
			hasConsolidation = true
		}
	}
	
	if !hasFlakiness {
		t.Error("Expected flakiness cleanup recommendation")
	}
	
	if !hasOptimization {
		t.Error("Expected optimization recommendation for slow test")
	}
	
	// Verify recommendations have required fields
	for _, rec := range recommendations {
		if rec.Description == "" {
			t.Error("Recommendation should have description")
		}
		if rec.Impact == "" {
			t.Error("Recommendation should have impact description")
		}
		if len(rec.Actions) == 0 {
			t.Error("Recommendation should have actions")
		}
	}
}

func TestExecutionTimePredictor_PredictExecutionTime(t *testing.T) {
	predictor := NewExecutionTimePredictor()
	
	// Add some historical data
	testID := "test1"
	for i := 0; i < 10; i++ {
		record := ExecutionRecord{
			TestID:    testID,
			Duration:  time.Duration(30+i*2) * time.Second,
			Success:   true,
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
		}
		predictor.AddExecutionRecord(record)
	}
	
	test := &TestCase{
		ID:            testID,
		EstimatedTime: 35 * time.Second,
		ResourceUsage: ResourceUsage{
			CPU:     50,
			Memory:  512 * 1024 * 1024,
			Network: true,
		},
		Flakiness: 0.1,
	}
	
	ctx := context.Background()
	prediction, err := predictor.PredictExecutionTime(ctx, test)
	if err != nil {
		t.Fatalf("PredictExecutionTime failed: %v", err)
	}
	
	if prediction.EstimatedTime == 0 {
		t.Error("Expected non-zero estimated time")
	}
	
	if prediction.MinTime >= prediction.EstimatedTime {
		t.Error("Min time should be less than estimated time")
	}
	
	if prediction.MaxTime <= prediction.EstimatedTime {
		t.Error("Max time should be greater than estimated time")
	}
	
	if prediction.Confidence < 0 || prediction.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", prediction.Confidence)
	}
	
	if len(prediction.Factors) == 0 {
		t.Error("Expected prediction factors to be identified")
	}
}

func TestTestExecutionOptimizer_OptimizeTestExecution(t *testing.T) {
	optimizer := NewTestExecutionOptimizer()
	
	tests := []TestCase{
		{
			ID:            "test1",
			EstimatedTime: 30 * time.Second,
			ResourceUsage: ResourceUsage{CPU: 40, Memory: 256 * 1024 * 1024},
			Impact:        ImpactScore{Overall: 0.8},
		},
		{
			ID:            "test2",
			EstimatedTime: 120 * time.Second,
			ResourceUsage: ResourceUsage{CPU: 80, Memory: 1024 * 1024 * 1024},
			Impact:        ImpactScore{Overall: 0.6},
		},
	}
	
	ctx := context.Background()
	result, err := optimizer.OptimizeTestExecution(ctx, tests)
	if err != nil {
		t.Fatalf("OptimizeTestExecution failed: %v", err)
	}
	
	if result.SchedulingPlan == nil {
		t.Error("Expected scheduling plan to be generated")
	}
	
	if result.CostAnalysis == nil {
		t.Error("Expected cost analysis to be generated")
	}
	
	if result.ResourceUsage == nil {
		t.Error("Expected resource usage analysis to be generated")
	}
	
	if len(result.Recommendations) == 0 {
		t.Error("Expected optimization recommendations to be generated")
	}
	
	// Verify recommendations have required fields
	for _, rec := range result.Recommendations {
		if rec.Description == "" {
			t.Error("Recommendation should have description")
		}
		if rec.ExpectedImpact == "" {
			t.Error("Recommendation should have expected impact")
		}
	}
}

// Helper function
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}