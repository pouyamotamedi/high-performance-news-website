package testing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example functions that would be targets for mutation testing
// These represent typical business logic, security, and performance code

// Business Logic Example - Article validation
func ValidateArticle(title, content string, authorID uint64) error {
	if len(title) == 0 {
		return ErrInvalidTitle
	}
	if len(title) > 255 {
		return ErrTitleTooLong
	}
	if len(content) < 100 {
		return ErrContentTooShort
	}
	if authorID == 0 {
		return ErrInvalidAuthor
	}
	return nil
}

// Security Example - Password validation
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	if len(password) > 128 {
		return ErrPasswordTooLong
	}
	
	hasUpper := false
	hasLower := false
	hasDigit := false
	
	for _, char := range password {
		if char >= 'A' && char <= 'Z' {
			hasUpper = true
		} else if char >= 'a' && char <= 'z' {
			hasLower = true
		} else if char >= '0' && char <= '9' {
			hasDigit = true
		}
	}
	
	if !hasUpper || !hasLower || !hasDigit {
		return ErrPasswordTooWeak
	}
	
	return nil
}

// Performance Example - Batch processing
func ProcessArticlesBatch(articles []Article, batchSize int) (int, error) {
	if batchSize <= 0 {
		return 0, ErrInvalidBatchSize
	}
	
	processed := 0
	for i := 0; i < len(articles); i += batchSize {
		end := i + batchSize
		if end > len(articles) {
			end = len(articles)
		}
		
		batch := articles[i:end]
		for _, article := range batch {
			if err := processArticle(article); err != nil {
				return processed, err
			}
			processed++
		}
	}
	
	return processed, nil
}

// Helper types and functions for examples
type Article struct {
	ID      uint64
	Title   string
	Content string
}

var (
	ErrInvalidTitle     = &ValidationError{"title cannot be empty"}
	ErrTitleTooLong     = &ValidationError{"title too long"}
	ErrContentTooShort  = &ValidationError{"content too short"}
	ErrInvalidAuthor    = &ValidationError{"invalid author"}
	ErrPasswordTooShort = &ValidationError{"password too short"}
	ErrPasswordTooLong  = &ValidationError{"password too long"}
	ErrPasswordTooWeak  = &ValidationError{"password too weak"}
	ErrInvalidBatchSize = &ValidationError{"invalid batch size"}
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func processArticle(article Article) error {
	// Simulate processing
	if article.ID == 0 {
		return &ValidationError{"invalid article ID"}
	}
	return nil
}

// Test cases that demonstrate mutation testing effectiveness

func TestValidateArticle_MutationTargets(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		content   string
		authorID  uint64
		expectErr bool
	}{
		{
			name:      "valid article",
			title:     "Valid Title",
			content:   "This is a valid content that is longer than 100 characters to meet the minimum requirement for article content validation.",
			authorID:  1,
			expectErr: false,
		},
		{
			name:      "empty title",
			title:     "",
			content:   "Valid content that is long enough to pass validation requirements for article content length checking.",
			authorID:  1,
			expectErr: true,
		},
		{
			name:      "title too long",
			title:     "This is a very long title that exceeds the maximum allowed length of 255 characters for article titles in our system and should trigger a validation error when processed by the ValidateArticle function because it violates the length constraint",
			content:   "Valid content that is long enough to pass validation requirements for article content length checking.",
			authorID:  1,
			expectErr: true,
		},
		{
			name:      "content too short",
			title:     "Valid Title",
			content:   "Short",
			authorID:  1,
			expectErr: true,
		},
		{
			name:      "invalid author",
			title:     "Valid Title",
			content:   "This is a valid content that is longer than 100 characters to meet the minimum requirement for article content validation.",
			authorID:  0,
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArticle(tt.title, tt.content, tt.authorID)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword_MutationTargets(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		expectErr bool
	}{
		{
			name:      "valid password",
			password:  "ValidPass123",
			expectErr: false,
		},
		{
			name:      "too short",
			password:  "Short1",
			expectErr: true,
		},
		{
			name:      "too long",
			password:  "ThisPasswordIsWayTooLongAndExceedsTheMaximumAllowedLengthOf128CharactersWhichShouldTriggerAValidationErrorInOurSystem123456789",
			expectErr: true,
		},
		{
			name:      "no uppercase",
			password:  "lowercase123",
			expectErr: true,
		},
		{
			name:      "no lowercase",
			password:  "UPPERCASE123",
			expectErr: true,
		},
		{
			name:      "no digits",
			password:  "NoDigitsHere",
			expectErr: true,
		},
		{
			name:      "minimum valid",
			password:  "Aa1bcdef",
			expectErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessArticlesBatch_MutationTargets(t *testing.T) {
	articles := []Article{
		{ID: 1, Title: "Article 1"},
		{ID: 2, Title: "Article 2"},
		{ID: 3, Title: "Article 3"},
		{ID: 4, Title: "Article 4"},
		{ID: 5, Title: "Article 5"},
	}
	
	tests := []struct {
		name         string
		articles     []Article
		batchSize    int
		expectErr    bool
		expectedCount int
	}{
		{
			name:         "valid batch processing",
			articles:     articles,
			batchSize:    2,
			expectErr:    false,
			expectedCount: 5,
		},
		{
			name:         "batch size zero",
			articles:     articles,
			batchSize:    0,
			expectErr:    true,
			expectedCount: 0,
		},
		{
			name:         "negative batch size",
			articles:     articles,
			batchSize:    -1,
			expectErr:    true,
			expectedCount: 0,
		},
		{
			name:         "empty articles",
			articles:     []Article{},
			batchSize:    2,
			expectErr:    false,
			expectedCount: 0,
		},
		{
			name:         "batch size larger than articles",
			articles:     articles,
			batchSize:    10,
			expectErr:    false,
			expectedCount: 5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := ProcessArticlesBatch(tt.articles, tt.batchSize)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, count)
			}
		})
	}
}

// Example of running mutation testing on the above functions
func TestMutationTesting_ExampleUsage(t *testing.T) {
	// This test demonstrates how to configure and run mutation testing
	// on specific functions
	
	config := &MutationConfig{
		TargetPackages: []string{"./internal/testing"},
		TestPackages:   []string{"./internal/testing"},
		ExcludePatterns: []string{
			".*_test\\.go$",
			".*mock.*\\.go$",
		},
		MutationTypes: []string{
			"ConditionalBoundaryMutator",
			"ArithmeticOperatorMutator",
			"LogicalOperatorMutator",
			"ReturnValueMutator",
		},
		MinMutationScore: 80.0,
		Timeout:         10 * time.Second,
		MaxConcurrency:  2,
		CriticalFunctions: []string{
			"ValidateArticle",
			"ValidatePassword",
		},
		SecurityFunctions: []string{
			"ValidatePassword",
		},
		PerformanceFunctions: []string{
			"ProcessArticlesBatch",
		},
	}
	
	tester := NewMutationTester(config)
	
	// In a real scenario, you would run the full mutation testing
	// For this test, we just verify the configuration is valid
	assert.NotNil(t, tester)
	assert.Equal(t, config, tester.config)
	
	// Verify mutators are created
	assert.NotEmpty(t, tester.mutators)
	
	// Verify each mutator type is present
	mutatorNames := make(map[string]bool)
	for _, mutator := range tester.mutators {
		mutatorNames[mutator.Name()] = true
	}
	
	for _, expectedMutator := range config.MutationTypes {
		assert.True(t, mutatorNames[expectedMutator], 
			"Expected mutator %s not found", expectedMutator)
	}
}

// Example of test quality analysis
func TestTestQualityAnalysis_Example(t *testing.T) {
	// Simulate mutation results for our example functions
	results := []MutationResult{
		{
			FilePath:     "internal/testing/mutation_example_test.go",
			Function:     "ValidateArticle",
			MutationType: "ConditionalBoundaryMutator",
			Category:     "business_logic",
			Killed:       true,
			LineNumber:   15,
		},
		{
			FilePath:     "internal/testing/mutation_example_test.go",
			Function:     "ValidateArticle",
			MutationType: "ArithmeticOperatorMutator",
			Category:     "business_logic",
			Killed:       false, // This mutation survived - weak test
			LineNumber:   18,
		},
		{
			FilePath:     "internal/testing/mutation_example_test.go",
			Function:     "ValidatePassword",
			MutationType: "SecurityMutator",
			Category:     "security",
			Killed:       true,
			LineNumber:   35,
		},
		{
			FilePath:     "internal/testing/mutation_example_test.go",
			Function:     "ProcessArticlesBatch",
			MutationType: "PerformanceMutator",
			Category:     "performance",
			Killed:       false, // This mutation survived - weak test
			LineNumber:   65,
		},
	}
	
	analyzer := NewTestQualityAnalyzer(results)
	report := analyzer.AnalyzeTestEffectiveness()
	
	require.NotNil(t, report)
	
	// Check function analysis
	validateArticleKey := "internal/testing/mutation_example_test.go::ValidateArticle"
	validateArticleAnalysis := report.FunctionAnalysis[validateArticleKey]
	require.NotNil(t, validateArticleAnalysis)
	
	assert.Equal(t, 2, validateArticleAnalysis.TotalMutations)
	assert.Equal(t, 1, validateArticleAnalysis.KilledMutations)
	assert.Equal(t, 50.0, validateArticleAnalysis.EffectivenessScore)
	assert.Equal(t, "fair", validateArticleAnalysis.QualityLevel)
	assert.Contains(t, validateArticleAnalysis.SurvivedMutations, "ArithmeticOperatorMutator")
	
	// Check category analysis
	businessLogicAnalysis := report.CategoryAnalysis["business_logic"]
	require.NotNil(t, businessLogicAnalysis)
	assert.Equal(t, 50.0, businessLogicAnalysis.EffectivenessScore)
	
	securityAnalysis := report.CategoryAnalysis["security"]
	require.NotNil(t, securityAnalysis)
	assert.Equal(t, 100.0, securityAnalysis.EffectivenessScore)
	
	performanceAnalysis := report.CategoryAnalysis["performance"]
	require.NotNil(t, performanceAnalysis)
	assert.Equal(t, 0.0, performanceAnalysis.EffectivenessScore)
}

// Benchmark to demonstrate performance testing integration
func BenchmarkProcessArticlesBatch(b *testing.B) {
	articles := make([]Article, 1000)
	for i := range articles {
		articles[i] = Article{
			ID:    uint64(i + 1),
			Title: "Benchmark Article",
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ProcessArticlesBatch(articles, 50)
		if err != nil {
			b.Fatal(err)
		}
	}
}