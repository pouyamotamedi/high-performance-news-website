package testing

import (
	"context"
	"testing"
	"time"
)

func TestTask24Implementation_SelectIntelligentTests(t *testing.T) {
	impl := NewTask24Implementation()
	
	ctx := context.Background()
	
	t.Run("Fast Priority Selection", func(t *testing.T) {
		request := Task24SelectionRequest{
			ChangedFiles:    []string{"internal/models/user.go", "internal/services/auth.go"},
			TimeLimit:       2 * time.Minute,
			Priority:        "fast",
			ParallelWorkers: 2,
			TargetCoverage:  0.8,
		}
		
		result, err := impl.SelectIntelligentTests(ctx, request)
		if err != nil {
			t.Fatalf("SelectIntelligentTests failed: %v", err)
		}
		
		if len(result.SelectedTests) == 0 {
			t.Error("Expected at least one test to be selected")
		}
		
		// Should prioritize fast tests
		for _, test := range result.SelectedTests {
			if test.EstimatedTime > 60*time.Second {
				t.Errorf("Fast priority should not select slow tests, got %v", test.EstimatedTime)
			}
		}
		
		if result.EstimatedTime > request.TimeLimit {
			t.Errorf("Estimated time %v exceeds time limit %v", result.EstimatedTime, request.TimeLimit)
		}
		
		if len(result.Reasoning) == 0 {
			t.Error("Expected reasoning to be provided")
		}
		
		if result.ExecutionPlan == nil {
			t.Error("Expected execution plan to be generated")
		}
	})
	
	t.Run("Balanced Priority Selection", func(t *testing.T) {
		request := Task24SelectionRequest{
			ChangedFiles:    []string{"internal/services/article_service.go"},
			Priority:        "balanced",
			ParallelWorkers: 4,
			TargetCoverage:  0.9,
		}
		
		result, err := impl.SelectIntelligentTests(ctx, request)
		if err != nil {
			t.Fatalf("SelectIntelligentTests failed: %v", err)
		}
		
		if len(result.SelectedTests) == 0 {
			t.Error("Expected at least one test to be selected")
		}
		
		if result.ExpectedCoverage < 0 || result.ExpectedCoverage > 1 {
			t.Errorf("Expected coverage should be between 0 and 1, got %f", result.ExpectedCoverage)
		}
	})
	
	t.Run("Critical Priority Selection", func(t *testing.T) {
		request := Task24SelectionRequest{
			ChangedFiles:    []string{"internal/auth/security.go"},
			Priority:        "critical",
			ParallelWorkers: 1,
		}
		
		result, err := impl.SelectIntelligentTests(ctx, request)
		if err != nil {
			t.Fatalf("SelectIntelligentTests failed: %v", err)
		}
		
		// Should only select critical tests (high impact)
		for _, test := range result.SelectedTests {
			if test.ImpactScore < 0.8 {
				t.Errorf("Critical priority should only select high-impact tests, got impact %f", test.ImpactScore)
			}
		}
	})
	
	t.Run("Complete Priority Selection", func(t *testing.T) {
		request := Task24SelectionRequest{
			ChangedFiles:    []string{"internal/models/article.go", "internal/services/content.go"},
			Priority:        "complete",
			ParallelWorkers: 8,
		}
		
		result, err := impl.SelectIntelligentTests(ctx, request)
		if err != nil {
			t.Fatalf("SelectIntelligentTests failed: %v", err)
		}
		
		// Complete priority should select all affected tests
		if len(result.SelectedTests) != len(request.ChangedFiles) {
			t.Errorf("Complete priority should select all affected tests, expected %d, got %d", 
				len(request.ChangedFiles), len(result.SelectedTests))
		}
	})
}

func TestTask24Implementation_OptimizeTestExecution(t *testing.T) {
	impl := NewTask24Implementation()
	
	tests := []Task24TestInfo{
		{
			ID:            "test1",
			Name:          "Fast Unit Test",
			EstimatedTime: 15 * time.Second,
			ImpactScore:   0.8,
			ResourceUsage: Task24ResourceUsage{CPU: 30, Memory: 100 * 1024 * 1024},
		},
		{
			ID:            "test2",
			Name:          "High CPU Test",
			EstimatedTime: 45 * time.Second,
			ImpactScore:   0.7,
			ResourceUsage: Task24ResourceUsage{CPU: 80, Memory: 500 * 1024 * 1024}, // High CPU
		},
		{
			ID:            "test3",
			Name:          "High Memory Test",
			EstimatedTime: 60 * time.Second,
			ImpactScore:   0.6,
			ResourceUsage: Task24ResourceUsage{CPU: 40, Memory: 3 * 1024 * 1024 * 1024}, // 3GB memory
		},
		{
			ID:            "test4",
			Name:          "Network Test",
			EstimatedTime: 30 * time.Second,
			ImpactScore:   0.5,
			ResourceUsage: Task24ResourceUsage{CPU: 25, Memory: 200 * 1024 * 1024, Network: true},
		},
	}
	
	ctx := context.Background()
	result, err := impl.OptimizeTestExecution(ctx, tests)
	if err != nil {
		t.Fatalf("OptimizeTestExecution failed: %v", err)
	}
	
	if len(result.Bottlenecks) == 0 {
		t.Error("Expected bottlenecks to be detected")
	}
	
	if result.ResourceAnalysis == nil {
		t.Error("Expected resource analysis to be generated")
	}
	
	if len(result.Recommendations) == 0 {
		t.Error("Expected optimization recommendations to be generated")
	}
	
	// Verify bottleneck detection
	foundCPUBottleneck := false
	foundMemoryBottleneck := false
	
	for _, bottleneck := range result.Bottlenecks {
		switch bottleneck.Type {
		case "cpu_contention":
			foundCPUBottleneck = true
		case "memory_pressure":
			foundMemoryBottleneck = true
		}
	}
	
	if !foundCPUBottleneck {
		t.Error("Expected CPU contention bottleneck to be detected")
	}
	
	if !foundMemoryBottleneck {
		t.Error("Expected memory pressure bottleneck to be detected")
	}
	
	// Verify resource analysis
	if result.ResourceAnalysis.CPUUtilization <= 0 {
		t.Error("Expected positive CPU utilization")
	}
	
	if result.ResourceAnalysis.Efficiency < 0 || result.ResourceAnalysis.Efficiency > 1 {
		t.Errorf("Efficiency should be between 0 and 1, got %f", result.ResourceAnalysis.Efficiency)
	}
	
	// Verify recommendations have required fields
	for _, rec := range result.Recommendations {
		if rec.Type == "" {
			t.Error("Recommendation should have type")
		}
		if rec.Description == "" {
			t.Error("Recommendation should have description")
		}
		if rec.ExpectedImpact == "" {
			t.Error("Recommendation should have expected impact")
		}
		if rec.Priority == "" {
			t.Error("Recommendation should have priority")
		}
	}
}

func TestTask24ImpactAnalyzer_AnalyzeCodeImpact(t *testing.T) {
	analyzer := NewTask24ImpactAnalyzer()
	
	ctx := context.Background()
	changedFiles := []string{
		"internal/models/user.go",
		"internal/services/auth_service.go",
		"internal/api/handlers.go",
	}
	
	affectedTests, err := analyzer.AnalyzeCodeImpact(ctx, changedFiles)
	if err != nil {
		t.Fatalf("AnalyzeCodeImpact failed: %v", err)
	}
	
	if len(affectedTests) != len(changedFiles) {
		t.Errorf("Expected %d affected tests, got %d", len(changedFiles), len(affectedTests))
	}
	
	// Verify test information
	for i, test := range affectedTests {
		if test.ID == "" {
			t.Error("Test should have ID")
		}
		if test.Name == "" {
			t.Error("Test should have name")
		}
		if test.EstimatedTime == 0 {
			t.Error("Test should have estimated time")
		}
		if test.ImpactScore < 0 || test.ImpactScore > 1 {
			t.Errorf("Impact score should be between 0 and 1, got %f", test.ImpactScore)
		}
		
		// Impact should decrease for later tests (as implemented)
		if i > 0 && test.ImpactScore >= affectedTests[i-1].ImpactScore {
			t.Error("Impact score should decrease for later tests")
		}
	}
}

func TestTask24TimePredictor_PredictTime(t *testing.T) {
	predictor := NewTask24TimePredictor()
	
	// Add some historical data
	testID := "test1"
	for i := 0; i < 5; i++ {
		record := Task24ExecutionRecord{
			TestID:    testID,
			Duration:  time.Duration(30+i*2) * time.Second,
			Success:   true,
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
		}
		predictor.historicalData[testID] = append(predictor.historicalData[testID], record)
	}
	
	test := &Task24TestInfo{
		ID:            testID,
		EstimatedTime: 35 * time.Second,
		ResourceUsage: Task24ResourceUsage{
			CPU:     50,
			Memory:  512 * 1024 * 1024,
			Network: true,
		},
		Flakiness: 0.1,
	}
	
	ctx := context.Background()
	prediction, err := predictor.PredictTime(ctx, test)
	if err != nil {
		t.Fatalf("PredictTime failed: %v", err)
	}
	
	if prediction == 0 {
		t.Error("Expected non-zero predicted time")
	}
	
	// Should be influenced by historical data and factors
	if prediction < 30*time.Second || prediction > 60*time.Second {
		t.Errorf("Prediction seems unreasonable: %v", prediction)
	}
}

func TestTask24Implementation_CreateExecutionPlan(t *testing.T) {
	impl := NewTask24Implementation()
	
	tests := []Task24TestInfo{
		{
			ID:            "fast1",
			EstimatedTime: 10 * time.Second,
			ResourceUsage: Task24ResourceUsage{CPU: 20},
		},
		{
			ID:            "fast2",
			EstimatedTime: 15 * time.Second,
			ResourceUsage: Task24ResourceUsage{CPU: 25},
		},
		{
			ID:            "medium1",
			EstimatedTime: 90 * time.Second,
			ResourceUsage: Task24ResourceUsage{CPU: 50},
		},
		{
			ID:            "slow1",
			EstimatedTime: 300 * time.Second,
			ResourceUsage: Task24ResourceUsage{CPU: 80},
		},
	}
	
	plan := impl.createExecutionPlan(tests, 4)
	
	if len(plan.Phases) == 0 {
		t.Error("Expected execution phases to be created")
	}
	
	if len(plan.ParallelGroups) == 0 {
		t.Error("Expected parallel groups to be created")
	}
	
	if plan.TotalEstimated == 0 {
		t.Error("Expected total estimated time to be calculated")
	}
	
	// Verify phase characteristics
	for _, phase := range plan.Phases {
		if phase.Name == "" {
			t.Error("Phase should have name")
		}
		if len(phase.Tests) == 0 {
			t.Error("Phase should have tests")
		}
		if phase.MaxWorkers <= 0 {
			t.Error("Phase should have positive max workers")
		}
		
		// Fast tests should be parallel
		if phase.Name == "fast_parallel" && !phase.Parallel {
			t.Error("Fast phase should be parallel")
		}
		
		// Slow tests should be sequential
		if phase.Name == "slow_sequential" && phase.Parallel {
			t.Error("Slow phase should be sequential")
		}
	}
	
	// Verify resource needs
	if plan.ResourceNeeds.TotalCPU <= 0 {
		t.Error("Expected positive total CPU")
	}
	
	if plan.ResourceNeeds.Workers <= 0 {
		t.Error("Expected positive worker count")
	}
}

func TestTask24ExecutionOptimizer_DetectBottlenecks(t *testing.T) {
	optimizer := NewTask24ExecutionOptimizer()
	
	tests := []Task24TestInfo{
		{
			ID:            "high_cpu_test",
			ResourceUsage: Task24ResourceUsage{CPU: 85}, // High CPU
		},
		{
			ID:            "high_memory_test",
			ResourceUsage: Task24ResourceUsage{Memory: 3 * 1024 * 1024 * 1024}, // 3GB
		},
		{
			ID:            "network_test1",
			ResourceUsage: Task24ResourceUsage{Network: true},
		},
		{
			ID:            "network_test2",
			ResourceUsage: Task24ResourceUsage{Network: true},
		},
		{
			ID:            "network_test3",
			ResourceUsage: Task24ResourceUsage{Network: true},
		},
		{
			ID:            "network_test4",
			ResourceUsage: Task24ResourceUsage{Network: true},
		},
	}
	
	ctx := context.Background()
	bottlenecks, err := optimizer.DetectBottlenecks(ctx, tests)
	if err != nil {
		t.Fatalf("DetectBottlenecks failed: %v", err)
	}
	
	if len(bottlenecks) == 0 {
		t.Error("Expected bottlenecks to be detected")
	}
	
	// Verify specific bottlenecks
	foundCPU := false
	foundMemory := false
	foundNetwork := false
	
	for _, bottleneck := range bottlenecks {
		switch bottleneck.Type {
		case "cpu_contention":
			foundCPU = true
			if len(bottleneck.TestIDs) == 0 {
				t.Error("CPU bottleneck should have test IDs")
			}
		case "memory_pressure":
			foundMemory = true
			if len(bottleneck.TestIDs) == 0 {
				t.Error("Memory bottleneck should have test IDs")
			}
		case "network_dependency":
			foundNetwork = true
			if len(bottleneck.TestIDs) < 4 {
				t.Error("Network bottleneck should have multiple test IDs")
			}
		}
		
		// Verify bottleneck fields
		if bottleneck.Description == "" {
			t.Error("Bottleneck should have description")
		}
		if bottleneck.Severity == "" {
			t.Error("Bottleneck should have severity")
		}
	}
	
	if !foundCPU {
		t.Error("Expected CPU contention bottleneck")
	}
	if !foundMemory {
		t.Error("Expected memory pressure bottleneck")
	}
	if !foundNetwork {
		t.Error("Expected network dependency bottleneck")
	}
}

func TestTask24ExecutionOptimizer_AnalyzeResourceUsage(t *testing.T) {
	optimizer := NewTask24ExecutionOptimizer()
	
	tests := []Task24TestInfo{
		{
			ID:            "test1",
			ResourceUsage: Task24ResourceUsage{CPU: 40, Memory: 512 * 1024 * 1024},
		},
		{
			ID:            "test2",
			ResourceUsage: Task24ResourceUsage{CPU: 60, Memory: 1024 * 1024 * 1024},
		},
		{
			ID:            "test3",
			ResourceUsage: Task24ResourceUsage{CPU: 20, Memory: 256 * 1024 * 1024},
		},
	}
	
	ctx := context.Background()
	analysis, err := optimizer.AnalyzeResourceUsage(ctx, tests)
	if err != nil {
		t.Fatalf("AnalyzeResourceUsage failed: %v", err)
	}
	
	if analysis.CPUUtilization <= 0 {
		t.Error("Expected positive CPU utilization")
	}
	
	if analysis.MemoryUtilization < 0 {
		t.Error("Expected non-negative memory utilization")
	}
	
	if analysis.Efficiency < 0 || analysis.Efficiency > 1 {
		t.Errorf("Efficiency should be between 0 and 1, got %f", analysis.Efficiency)
	}
	
	// Verify calculated values
	expectedCPU := (40.0 + 60.0 + 20.0) / 3.0
	if analysis.CPUUtilization != expectedCPU {
		t.Errorf("Expected CPU utilization %f, got %f", expectedCPU, analysis.CPUUtilization)
	}
}

func TestTask24Implementation_GenerateOptimizationRecommendations(t *testing.T) {
	impl := NewTask24Implementation()
	
	bottlenecks := []Task24Bottleneck{
		{
			Type:     "cpu_contention",
			Severity: "high",
			TestIDs:  []string{"test1", "test2"},
		},
		{
			Type:     "memory_pressure",
			Severity: "medium",
			TestIDs:  []string{"test3"},
		},
	}
	
	resourceAnalysis := &Task24ResourceAnalysis{
		CPUUtilization: 25.0, // Low CPU utilization
		Efficiency:     0.5,  // Low efficiency
	}
	
	recommendations := impl.generateOptimizationRecommendations(bottlenecks, resourceAnalysis)
	
	if len(recommendations) == 0 {
		t.Error("Expected optimization recommendations to be generated")
	}
	
	// Verify recommendation types
	foundParallel := false
	foundResource := false
	foundScheduling := false
	
	for _, rec := range recommendations {
		switch rec.Type {
		case "parallel_optimization":
			foundParallel = true
		case "resource_optimization":
			foundResource = true
		case "scheduling_optimization":
			foundScheduling = true
		}
		
		// Verify recommendation fields
		if rec.Description == "" {
			t.Error("Recommendation should have description")
		}
		if rec.Priority == "" {
			t.Error("Recommendation should have priority")
		}
		if rec.ExpectedImpact == "" {
			t.Error("Recommendation should have expected impact")
		}
	}
	
	if !foundParallel {
		t.Error("Expected parallel optimization recommendation")
	}
	if !foundResource {
		t.Error("Expected resource optimization recommendation")
	}
	if !foundScheduling {
		t.Error("Expected scheduling optimization recommendation")
	}
}