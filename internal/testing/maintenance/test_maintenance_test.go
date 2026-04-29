package maintenance

import (
	"database/sql"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestMaintenanceManager(t *testing.T) {
	// Skip if no database URL provided
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not provided, skipping database tests")
	}

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)
	defer db.Close()

	// Create test maintenance manager
	tmm := NewTestMaintenanceManager(db)
	require.NotNil(t, tmm)

	t.Run("AnalyzeTestSuite", func(t *testing.T) {
		// Create a temporary test file
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "example_test.go")
		
		testContent := `package example

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestExample(t *testing.T) {
	result := 2 + 2
	assert.Equal(t, 4, result)
}

func TestAnotherExample(t *testing.T) {
	t.Parallel()
	
	value := "hello"
	assert.NotEmpty(t, value)
}
`
		
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// Analyze the test suite
		analysis, err := tmm.AnalyzeTestSuite(tempDir)
		require.NoError(t, err)
		require.NotNil(t, analysis)

		// Verify analysis results
		assert.Equal(t, tempDir, analysis.RootPath)
		assert.True(t, len(analysis.Tests) > 0)
		
		// Check that tests were found
		foundTests := false
		for testID := range analysis.Tests {
			if testID == "example_test.go::TestExample" || testID == "example_test.go::TestAnotherExample" {
				foundTests = true
				break
			}
		}
		assert.True(t, foundTests, "Expected to find test functions")
	})

	t.Run("TestRelationshipManager", func(t *testing.T) {
		// Test relationship management
		testID1 := "test1"
		testID2 := "test2"
		
		relationships := []TestRelation{
			{
				Type:       RelationSimilarTo,
				TargetTest: testID2,
				Strength:   0.8,
			},
		}

		// Update relationships
		err := tmm.RelationshipManager().UpdateRelationships(testID1, relationships)
		require.NoError(t, err)

		// Retrieve relationships
		retrieved, err := tmm.RelationshipManager().GetRelationships(testID1)
		require.NoError(t, err)
		assert.Len(t, retrieved, 1)
		assert.Equal(t, RelationSimilarTo, retrieved[0].Type)
		assert.Equal(t, testID2, retrieved[0].TargetTest)
		assert.Equal(t, 0.8, retrieved[0].Strength)
	})

	t.Run("TestLifecycleManager", func(t *testing.T) {
		testID := "lifecycle_test"
		
		// Create test
		err := tmm.LifecycleManager().CreateTest(testID, "Initial test creation", nil)
		require.NoError(t, err)

		// Activate test
		err = tmm.LifecycleManager().ActivateTest(testID, "Test is ready for use")
		require.NoError(t, err)

		// Deprecate test
		err = tmm.LifecycleManager().DeprecateTest(testID, "Test is outdated", nil)
		require.NoError(t, err)

		// Get lifecycle history
		events, err := tmm.LifecycleManager().GetTestLifecycle(testID)
		require.NoError(t, err)
		assert.Len(t, events, 3) // Created, Activated, Deprecated

		// Verify event types
		eventTypes := make(map[LifecycleEvent]bool)
		for _, event := range events {
			eventTypes[event.EventType] = true
		}
		assert.True(t, eventTypes[EventCreated])
		assert.True(t, eventTypes[EventActivated])
		assert.True(t, eventTypes[EventDeprecated])
	})

	t.Run("TestEvolutionTracker", func(t *testing.T) {
		testID := "evolution_test"
		
		// Track a change
		impact := Impact{
			CoverageChange:   0.1,
			RuntimeChange:    -100 * time.Millisecond,
			StabilityChange:  0.05,
			ComplexityChange: -1,
		}
		
		err := tmm.EvolutionTracker().TrackTestChange(
			testID, 
			ChangeOptimized, 
			"Optimized test performance", 
			"developer", 
			"Performance improvement",
			impact,
		)
		require.NoError(t, err)

		// Record metric snapshot
		metrics := TestMetricSnapshot{
			Timestamp:      time.Now(),
			Coverage:       0.85,
			Runtime:        500 * time.Millisecond,
			FailureRate:    0.02,
			ExecutionCount: 100,
			Complexity:     5,
		}
		
		err = tmm.EvolutionTracker().RecordMetricSnapshot(testID, metrics)
		require.NoError(t, err)

		// Get evolution history
		evolution, err := tmm.EvolutionTracker().GetTestEvolution(testID)
		require.NoError(t, err)
		assert.Equal(t, testID, evolution.TestID)
		assert.Len(t, evolution.Changes, 1)
		assert.Len(t, evolution.Metrics, 1)

		// Verify change details
		change := evolution.Changes[0]
		assert.Equal(t, ChangeOptimized, change.Type)
		assert.Equal(t, "Optimized test performance", change.Description)
		assert.Equal(t, "developer", change.Author)
	})
}

func TestTestAnalyzer(t *testing.T) {
	analyzer := NewTestAnalyzer()
	require.NotNil(t, analyzer)

	t.Run("AnalyzeTestFile", func(t *testing.T) {
		// Create a temporary test file
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "sample_test.go")
		
		testContent := `package sample

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

// TestSample tests the sample function
func TestSample(t *testing.T) {
	t.Parallel()
	
	// Test basic functionality
	result := add(2, 3)
	assert.Equal(t, 5, result)
	
	// Test edge case
	result = add(0, 0)
	assert.Equal(t, 0, result)
}

func add(a, b int) int {
	return a + b
}
`
		
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// Analyze the test file
		tests, err := analyzer.AnalyzeTestFile(testFile)
		require.NoError(t, err)
		require.Len(t, tests, 1)

		test := tests[0]
		assert.Equal(t, "sample_test.go::TestSample", test.ID)
		assert.Equal(t, testFile, test.FilePath)
		assert.Equal(t, "TestSample", test.TestName)
		assert.Equal(t, "unit", test.TestType)
		assert.True(t, test.Complexity > 0)
		assert.Contains(t, test.Dependencies, "testing")
		assert.Contains(t, test.Dependencies, "github.com/stretchr/testify/assert")
	})

	t.Run("DetectTestPatterns", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "patterns_test.go")
		
		testContent := `package patterns

import (
	"testing"
)

func TestTableDriven(t *testing.T) {
	testCases := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero", 0, 0},
		{"positive", 5, 25},
		{"negative", -3, 9},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := square(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestWithSetup(t *testing.T) {
	// Setup
	cleanup := setupTest()
	defer cleanup()
	
	// Test logic here
}

func square(x int) int {
	return x * x
}

func setupTest() func() {
	return func() {}
}
`
		
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// Detect patterns
		patterns, err := analyzer.DetectTestPatterns(testFile)
		require.NoError(t, err)
		
		// Should detect table-driven and setup/teardown patterns
		patternTypes := make(map[string]bool)
		for _, pattern := range patterns {
			patternTypes[pattern.Type] = true
		}
		
		assert.True(t, patternTypes["table_driven"], "Should detect table-driven pattern")
		assert.True(t, patternTypes["setup_teardown"], "Should detect setup/teardown pattern")
	})
}

func TestQualityAnalyzer(t *testing.T) {
	analyzer := NewQualityAnalyzer()
	require.NotNil(t, analyzer)

	t.Run("AnalyzeFile", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "quality_test.go")
		
		testContent := `package quality

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

// TestGoodQuality demonstrates good test quality
func TestGoodQuality(t *testing.T) {
	t.Parallel()
	
	// Test with clear naming and good structure
	calculator := NewCalculator()
	
	result := calculator.Add(2, 3)
	assert.Equal(t, 5, result, "Addition should work correctly")
	
	// Test edge case
	result = calculator.Add(0, 0)
	assert.Equal(t, 0, result, "Adding zeros should return zero")
}

type Calculator struct{}

func NewCalculator() *Calculator {
	return &Calculator{}
}

func (c *Calculator) Add(a, b int) int {
	return a + b
}
`
		
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// Parse the file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
		require.NoError(t, err)

		// Analyze quality
		metrics := analyzer.AnalyzeFile(node, fset, testContent)
		require.NotNil(t, metrics)

		// Verify metrics are reasonable
		assert.True(t, metrics.Maintainability >= 0 && metrics.Maintainability <= 1)
		assert.True(t, metrics.Readability >= 0 && metrics.Readability <= 1)
		assert.True(t, metrics.Reliability >= 0 && metrics.Reliability <= 1)
		assert.True(t, metrics.Performance >= 0 && metrics.Performance <= 1)
		assert.True(t, metrics.OverallQuality >= 0 && metrics.OverallQuality <= 1)
		assert.Equal(t, "stable", metrics.TrendDirection)
	})
}

func TestRefactoringEngine(t *testing.T) {
	engine := NewRefactoringEngine()
	require.NotNil(t, engine)

	t.Run("FindOpportunities", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "refactor_test.go")
		
		testContent := `package refactor

import (
	"testing"
)

func TestA(t *testing.T) {
	// Setup
	x := 1
	y := 2
	
	// Test
	result := x + y
	if result != 3 {
		t.Error("Expected 3")
	}
}

func TestB(t *testing.T) {
	// Setup (similar to TestA)
	x := 1
	y := 2
	
	// Test
	result := x + y
	if result != 3 {
		t.Error("Expected 3")
	}
}

func TestVeryComplexFunction(t *testing.T) {
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			if i > 5 {
				if j > 5 {
					if i+j > 15 {
						// Very nested logic
						t.Log("Complex condition met")
					}
				}
			}
		}
	}
}
`
		
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// Parse the file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
		require.NoError(t, err)

		// Find refactoring opportunities
		opportunities := engine.FindOpportunities(node, fset, testFile)
		require.NotEmpty(t, opportunities)

		// Should find opportunities for duplicate code and complexity
		opportunityTypes := make(map[RefactoringType]bool)
		for _, opp := range opportunities {
			opportunityTypes[opp.Type] = true
		}
		
		// Should detect complexity issues
		assert.True(t, opportunityTypes[RefactoringReduceComplexity], "Should detect complexity issues")
	})
}

func TestMaintenanceScheduler(t *testing.T) {
	// Skip if no database URL provided
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not provided, skipping database tests")
	}

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)
	defer db.Close()

	scheduler := NewMaintenanceScheduler(db)
	require.NotNil(t, scheduler)

	t.Run("ScheduleTask", func(t *testing.T) {
		schedule := MaintenanceSchedule{
			ID:       "test_schedule",
			Name:     "Test Analysis",
			Type:     MaintenanceAnalysis,
			Schedule: "0 0 * * *", // Daily at midnight
			Enabled:  true,
			Config: map[string]interface{}{
				"test_path": ".",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		task := &AnalysisTask{
			db:     db,
			config: schedule.Config,
		}

		err := scheduler.ScheduleTask(schedule, task)
		require.NoError(t, err)

		// Verify task was scheduled
		schedules, err := scheduler.GetScheduledTasks()
		require.NoError(t, err)
		
		found := false
		for _, s := range schedules {
			if s.ID == "test_schedule" {
				found = true
				assert.Equal(t, "Test Analysis", s.Name)
				assert.Equal(t, MaintenanceAnalysis, s.Type)
				assert.True(t, s.Enabled)
				break
			}
		}
		assert.True(t, found, "Scheduled task should be found")

		// Cleanup
		err = scheduler.DeleteSchedule("test_schedule")
		require.NoError(t, err)
	})
}

// Benchmark tests for performance validation
func BenchmarkTestAnalysis(b *testing.B) {
	analyzer := NewTestAnalyzer()
	
	// Create a sample test file content
	testContent := `package benchmark

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestBenchmark(t *testing.T) {
	t.Parallel()
	
	for i := 0; i < 100; i++ {
		result := i * 2
		assert.Equal(t, i*2, result)
	}
}
`
	
	// Create temporary file
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "benchmark_test.go")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeTestFile(testFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQualityAnalysis(b *testing.B) {
	analyzer := NewQualityAnalyzer()
	
	testContent := `package benchmark

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestQualityBenchmark(t *testing.T) {
	calculator := NewCalculator()
	
	result := calculator.Add(2, 3)
	assert.Equal(t, 5, result)
	
	result = calculator.Multiply(4, 5)
	assert.Equal(t, 20, result)
}

type Calculator struct{}

func NewCalculator() *Calculator {
	return &Calculator{}
}

func (c *Calculator) Add(a, b int) int {
	return a + b
}

func (c *Calculator) Multiply(a, b int) int {
	return a * b
}
`
	
	// Parse the content
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "benchmark.go", testContent, parser.ParseComments)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = analyzer.AnalyzeFile(node, fset, testContent)
	}
}