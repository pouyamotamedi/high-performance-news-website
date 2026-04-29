package testing

import (
	"go/ast"
	"go/parser"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMutationTester_Creation(t *testing.T) {
	config := DefaultMutationConfig()
	tester := NewMutationTester(config)
	
	assert.NotNil(t, tester)
	assert.Equal(t, config, tester.config)
	assert.NotNil(t, tester.fileSet)
	assert.NotEmpty(t, tester.mutators)
	assert.NotNil(t, tester.testRunner)
	assert.NotNil(t, tester.reporter)
}

func TestMutationConfig_Default(t *testing.T) {
	config := DefaultMutationConfig()
	
	assert.NotEmpty(t, config.TargetPackages)
	assert.NotEmpty(t, config.TestPackages)
	assert.NotEmpty(t, config.MutationTypes)
	assert.Greater(t, config.MinMutationScore, 0.0)
	assert.Greater(t, config.Timeout, time.Duration(0))
	assert.Greater(t, config.MaxConcurrency, 0)
	assert.NotEmpty(t, config.CriticalFunctions)
	assert.NotEmpty(t, config.SecurityFunctions)
	assert.NotEmpty(t, config.PerformanceFunctions)
}

func TestConditionalBoundaryMutator(t *testing.T) {
	mutator := &ConditionalBoundaryMutator{}
	
	tests := []struct {
		name     string
		code     string
		expected string
		canMutate bool
	}{
		{
			name:     "less than to less equal",
			code:     "x < y",
			expected: "<=",
			canMutate: true,
		},
		{
			name:     "greater than to greater equal",
			code:     "x > y",
			expected: ">=",
			canMutate: true,
		},
		{
			name:     "equal to not equal",
			code:     "x == y",
			expected: "!=",
			canMutate: true,
		},
		{
			name:     "addition not mutatable",
			code:     "x + y",
			canMutate: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			canMutate := mutator.CanMutate(expr)
			assert.Equal(t, tt.canMutate, canMutate)
			
			if canMutate {
				mutated, err := mutator.Mutate(expr)
				require.NoError(t, err)
				
				if binExpr, ok := mutated.(*ast.BinaryExpr); ok {
					assert.Contains(t, tt.expected, binExpr.Op.String())
				}
			}
		})
	}
}

func TestArithmeticOperatorMutator(t *testing.T) {
	mutator := &ArithmeticOperatorMutator{}
	
	tests := []struct {
		name      string
		code      string
		canMutate bool
	}{
		{
			name:      "addition",
			code:      "x + y",
			canMutate: true,
		},
		{
			name:      "subtraction",
			code:      "x - y",
			canMutate: true,
		},
		{
			name:      "multiplication",
			code:      "x * y",
			canMutate: true,
		},
		{
			name:      "division",
			code:      "x / y",
			canMutate: true,
		},
		{
			name:      "comparison not mutatable",
			code:      "x == y",
			canMutate: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			canMutate := mutator.CanMutate(expr)
			assert.Equal(t, tt.canMutate, canMutate)
			
			if canMutate {
				mutated, err := mutator.Mutate(expr)
				require.NoError(t, err)
				assert.NotNil(t, mutated)
			}
		})
	}
}

func TestLogicalOperatorMutator(t *testing.T) {
	mutator := &LogicalOperatorMutator{}
	
	tests := []struct {
		name      string
		code      string
		canMutate bool
	}{
		{
			name:      "logical and",
			code:      "x && y",
			canMutate: true,
		},
		{
			name:      "logical or",
			code:      "x || y",
			canMutate: true,
		},
		{
			name:      "logical not",
			code:      "!x",
			canMutate: true,
		},
		{
			name:      "bitwise and not mutatable",
			code:      "x & y",
			canMutate: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			canMutate := mutator.CanMutate(expr)
			assert.Equal(t, tt.canMutate, canMutate)
			
			if canMutate {
				mutated, err := mutator.Mutate(expr)
				require.NoError(t, err)
				assert.NotNil(t, mutated)
			}
		})
	}
}

func TestSecurityMutator(t *testing.T) {
	mutator := &SecurityMutator{}
	
	tests := []struct {
		name      string
		code      string
		canMutate bool
	}{
		{
			name:      "bcrypt function call",
			code:      "bcrypt.GenerateFromPassword(password, cost)",
			canMutate: true,
		},
		{
			name:      "validate token call",
			code:      "ValidateToken(token)",
			canMutate: true,
		},
		{
			name:      "regular function call",
			code:      "fmt.Println(message)",
			canMutate: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			canMutate := mutator.CanMutate(expr)
			assert.Equal(t, tt.canMutate, canMutate)
		})
	}
}

func TestPerformanceMutator(t *testing.T) {
	mutator := &PerformanceMutator{}
	
	tests := []struct {
		name      string
		code      string
		canMutate bool
	}{
		{
			name:      "make function call",
			code:      "make([]int, 10, 20)",
			canMutate: true,
		},
		{
			name:      "database query",
			code:      "db.Query(sql, args...)",
			canMutate: true,
		},
		{
			name:      "cache get",
			code:      "cache.Get(key)",
			canMutate: true,
		},
		{
			name:      "regular function call",
			code:      "fmt.Println(message)",
			canMutate: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			canMutate := mutator.CanMutate(expr)
			assert.Equal(t, tt.canMutate, canMutate)
		})
	}
}

func TestNullCheckMutator(t *testing.T) {
	mutator := &NullCheckMutator{}
	
	tests := []struct {
		name      string
		code      string
		canMutate bool
	}{
		{
			name:      "nil equality check",
			code:      "x == nil",
			canMutate: true,
		},
		{
			name:      "nil inequality check",
			code:      "x != nil",
			canMutate: true,
		},
		{
			name:      "regular equality check",
			code:      "x == y",
			canMutate: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			canMutate := mutator.CanMutate(expr)
			assert.Equal(t, tt.canMutate, canMutate)
			
			if canMutate {
				mutated, err := mutator.Mutate(expr)
				require.NoError(t, err)
				assert.NotNil(t, mutated)
			}
		})
	}
}

func TestMutationReport_CalculateScores(t *testing.T) {
	tester := NewMutationTester(DefaultMutationConfig())
	
	report := &MutationReport{
		Results: []MutationResult{
			{Category: "business_logic", Killed: true},
			{Category: "business_logic", Killed: false},
			{Category: "security", Killed: true},
			{Category: "security", Killed: true},
			{Category: "performance", Killed: false},
		},
		CategoryScores: make(map[string]float64),
	}
	
	tester.calculateMutationScores(report)
	
	assert.Equal(t, 5, report.TotalMutations)
	assert.Equal(t, 3, report.KilledMutations)
	assert.Equal(t, 2, report.SurvivedMutations)
	assert.Equal(t, 60.0, report.MutationScore)
	
	// Check category scores
	assert.Equal(t, 50.0, report.CategoryScores["business_logic"])  // 1/2 killed
	assert.Equal(t, 100.0, report.CategoryScores["security"])      // 2/2 killed
	assert.Equal(t, 0.0, report.CategoryScores["performance"])     // 0/1 killed
}

func TestMutationReport_AnalyzeWeakTests(t *testing.T) {
	tester := NewMutationTester(DefaultMutationConfig())
	
	report := &MutationReport{
		Results: []MutationResult{
			{
				FilePath:     "test.go",
				Function:     "TestFunction",
				MutationType: "ConditionalBoundaryMutator",
				Killed:       false,
			},
			{
				FilePath:     "test.go",
				Function:     "TestFunction",
				MutationType: "ArithmeticOperatorMutator",
				Killed:       false,
			},
			{
				FilePath:     "test.go",
				Function:     "AnotherFunction",
				MutationType: "LogicalOperatorMutator",
				Killed:       true,
			},
		},
	}
	
	tester.analyzeWeakTests(report)
	
	assert.Len(t, report.WeakTests, 1)
	
	weakTest := report.WeakTests[0]
	assert.Equal(t, "test.go::TestFunction", weakTest.TestFunction)
	assert.Len(t, weakTest.MissedMutations, 2)
	assert.Contains(t, weakTest.MissedMutations, "ConditionalBoundaryMutator")
	assert.Contains(t, weakTest.MissedMutations, "ArithmeticOperatorMutator")
	assert.NotEmpty(t, weakTest.Suggestions)
}

func TestMutationReport_GenerateRecommendations(t *testing.T) {
	config := DefaultMutationConfig()
	config.MinMutationScore = 80.0
	tester := NewMutationTester(config)
	
	report := &MutationReport{
		MutationScore: 70.0, // Below minimum
		CategoryScores: map[string]float64{
			"security":        60.0, // Low security score
			"business_logic":  85.0, // Good score
			"performance":     75.0, // Below 80
		},
		WeakTests: []WeakTestReport{
			{TestFunction: "test1"},
			{TestFunction: "test2"},
		},
	}
	
	tester.generateRecommendations(report)
	
	assert.NotEmpty(t, report.Recommendations)
	
	// Check that recommendations address the issues
	recommendationText := strings.Join(report.Recommendations, " ")
	assert.Contains(t, recommendationText, "below target")
	assert.Contains(t, recommendationText, "Security")
	assert.Contains(t, recommendationText, "Performance")
	assert.Contains(t, recommendationText, "weak test")
}

func TestGoTestRunner(t *testing.T) {
	runner := &GoTestRunner{}
	
	// Test with a simple package (this might fail if the package doesn't exist)
	// In a real scenario, you'd test with a known good package
	passed, output, err := runner.RunTests(".", 5*time.Second)
	
	// We can't assert specific results since it depends on the environment
	// But we can check that the runner returns reasonable values
	assert.NotNil(t, output)
	// err might be nil or not nil depending on test results
	_ = passed
	_ = err
}

func TestMutationTestSuite_DefaultConfiguration(t *testing.T) {
	suite := DefaultMutationTestSuite()
	
	assert.NotEmpty(t, suite.Name)
	assert.NotEmpty(t, suite.Description)
	assert.NotNil(t, suite.MutationConfig)
	assert.NotNil(t, suite.CriticalConfig)
	assert.NotNil(t, suite.ReportingConfig)
	assert.NotNil(t, suite.ScheduleConfig)
	
	// Check that critical code patterns are defined
	assert.NotEmpty(t, suite.CriticalConfig.BusinessLogicPatterns)
	assert.NotEmpty(t, suite.CriticalConfig.SecurityPatterns)
	assert.NotEmpty(t, suite.CriticalConfig.PerformancePatterns)
	
	// Check reporting configuration
	assert.NotEmpty(t, suite.ReportingConfig.OutputFormats)
	assert.Greater(t, suite.ReportingConfig.FailureThreshold, 0.0)
	
	// Check schedule configuration
	assert.NotEmpty(t, suite.ScheduleConfig.CronExpression)
	assert.NotEmpty(t, suite.ScheduleConfig.TargetBranches)
}

func TestTestQualityAnalyzer(t *testing.T) {
	results := []MutationResult{
		{
			FilePath:     "test1.go",
			Function:     "Function1",
			Category:     "business_logic",
			MutationType: "ConditionalBoundaryMutator",
			Killed:       true,
		},
		{
			FilePath:     "test1.go",
			Function:     "Function1",
			Category:     "business_logic",
			MutationType: "ArithmeticOperatorMutator",
			Killed:       false,
		},
		{
			FilePath:     "test2.go",
			Function:     "Function2",
			Category:     "security",
			MutationType: "SecurityMutator",
			Killed:       true,
		},
	}
	
	analyzer := NewTestQualityAnalyzer(results)
	report := analyzer.AnalyzeTestEffectiveness()
	
	assert.NotNil(t, report)
	assert.Len(t, report.FunctionAnalysis, 2)
	assert.Len(t, report.CategoryAnalysis, 2)
	
	// Check function analysis
	func1Key := "test1.go::Function1"
	func1Analysis := report.FunctionAnalysis[func1Key]
	assert.NotNil(t, func1Analysis)
	assert.Equal(t, 2, func1Analysis.TotalMutations)
	assert.Equal(t, 1, func1Analysis.KilledMutations)
	assert.Equal(t, 50.0, func1Analysis.EffectivenessScore)
	assert.Equal(t, "fair", func1Analysis.QualityLevel)
	
	// Check category analysis
	businessLogicAnalysis := report.CategoryAnalysis["business_logic"]
	assert.NotNil(t, businessLogicAnalysis)
	assert.Equal(t, 2, businessLogicAnalysis.TotalMutations)
	assert.Equal(t, 1, businessLogicAnalysis.KilledMutations)
	assert.Equal(t, 50.0, businessLogicAnalysis.EffectivenessScore)
	
	securityAnalysis := report.CategoryAnalysis["security"]
	assert.NotNil(t, securityAnalysis)
	assert.Equal(t, 1, securityAnalysis.TotalMutations)
	assert.Equal(t, 1, securityAnalysis.KilledMutations)
	assert.Equal(t, 100.0, securityAnalysis.EffectivenessScore)
}