package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPatternDetectionAccuracy tests the accuracy of AI code pattern detection
func TestPatternDetectionAccuracy(t *testing.T) {
	validator := NewAICodeValidator()
	
	testCases := []struct {
		name           string
		code           string
		expectedRules  []string
		shouldNotFind  []string
	}{
		{
			name: "missing error handling patterns",
			code: `package main
func processData() {
	result, err := db.Query("SELECT * FROM users")
	// err is assigned but not checked
	data := result.Scan()
}`,
			expectedRules: []string{"potential-unhandled-error"},
			shouldNotFind: []string{"sql-injection-risk"},
		},
		{
			name: "hardcoded values detection",
			code: `package main
func connect() {
	conn := db.Connect("localhost:5432")
	password := "secret"
	token := "admin"
}`,
			expectedRules: []string{"hardcoded-values"},
			shouldNotFind: []string{"missing-error-handling"},
		},
		{
			name: "SQL injection patterns",
			code: `package main
func getUser(id string) {
	db.Query("SELECT * FROM users WHERE id = " + id)
}`,
			expectedRules: []string{"sql-injection-risk"},
			shouldNotFind: []string{"hardcoded-values"},
		},
		{
			name: "inefficient database queries",
			code: `package main
func loadUsers() {
	for _, id := range userIDs {
		result += "user " + id
	}
}`,
			expectedRules: []string{"inefficient-string-concat"},
			shouldNotFind: []string{"sql-injection-risk"},
		},
		{
			name: "unsafe type assertions",
			code: `package main
func processInterface(data interface{}) {
	user := data.(User)
	fmt.Println(user.Name)
}`,
			expectedRules: []string{"unsafe-type-assertion"},
			shouldNotFind: []string{"hardcoded-values"},
		},
		{
			name: "goroutine without cleanup",
			code: `package main
func startWorker() {
	go func() {
		for {
			doWork()
		}
	}()
}`,
			expectedRules: []string{"goroutine-leak"},
			shouldNotFind: []string{"missing-error-handling"},
		},
		{
			name: "AI-generated TODO comments",
			code: `package main
func processData() {
	// TODO: implement proper validation
	// FIXME: add error handling
	data := getData()
}`,
			expectedRules: []string{"ai-generated-todo"},
			shouldNotFind: []string{"sql-injection-risk"},
		},
		{
			name: "good code should not trigger false positives",
			code: `package main

func goodFunction() {
	// Simple function that should not trigger any issues
	result := calculateSum(1, 2)
	return result
}

func calculateSum(a, b int) int {
	return a + b
}`,
			expectedRules: []string{},
			shouldNotFind: []string{"sql-injection-risk", "hardcoded-values", "unsafe-type-assertion"},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			
			err := os.WriteFile(tmpFile, []byte(tc.code), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			
			// Validate the file
			results, err := validator.ValidateFile(tmpFile)
			if err != nil {
				t.Fatalf("ValidateFile() error = %v", err)
			}
			
			// Check expected rules are found
			foundRules := make(map[string]bool)
			for _, result := range results {
				foundRules[result.RuleName] = true
			}
			
			for _, expectedRule := range tc.expectedRules {
				if !foundRules[expectedRule] {
					t.Errorf("Expected rule %s not found in results", expectedRule)
				}
			}
			
			// Check rules that should not be found
			for _, shouldNotFind := range tc.shouldNotFind {
				if foundRules[shouldNotFind] {
					t.Errorf("Rule %s should not be found but was detected", shouldNotFind)
				}
			}
			
			// Log all found rules for debugging
			if len(results) > 0 {
				t.Logf("Found rules: ")
				for _, result := range results {
					t.Logf("  - %s: %s", result.RuleName, result.Message)
				}
			}
		})
	}
}

// TestPatternDetectionPerformance tests the performance of pattern detection
func TestPatternDetectionPerformance(t *testing.T) {
	validator := NewAICodeValidator()
	
	// Generate a large code file with multiple patterns
	largeCode := `package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

`
	
	// Add 100 functions with various patterns
	for i := 0; i < 100; i++ {
		largeCode += fmt.Sprintf(`
func function%d() {
	// TODO: implement this function
	db.Query("SELECT * FROM table WHERE id = " + userID)
	password := "hardcoded%d"
	
	for j := 0; j < 10; j++ {
		result += fmt.Sprintf("item %%d", j)
	}
	
	go func() {
		for {
			doWork()
		}
	}()
	
	data := interface{}(nil)
	user := data.(User)
}
`, i, i)
	}
	
	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.go")
	
	err := os.WriteFile(tmpFile, []byte(largeCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	
	// Measure validation time
	start := time.Now()
	results, err := validator.ValidateFile(tmpFile)
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("ValidateFile() error = %v", err)
	}
	
	// Should complete within reasonable time (< 2 seconds for 100 functions)
	if duration > 2*time.Second {
		t.Errorf("Validation took too long: %v", duration)
	}
	
	// Should find multiple issues
	if len(results) == 0 {
		t.Error("Expected to find issues in large file with problems")
	}
	
	t.Logf("Validated large file with 100 functions in %v, found %d issues", 
		duration, len(results))
	
	// Check issue distribution
	categories := make(map[string]int)
	for _, result := range results {
		categories[result.Category]++
	}
	
	t.Logf("Issue distribution:")
	for category, count := range categories {
		t.Logf("  %s: %d", category, count)
	}
}

// TestPatternDetectionEdgeCases tests edge cases and corner scenarios
func TestPatternDetectionEdgeCases(t *testing.T) {
	validator := NewAICodeValidator()
	
	edgeCases := []struct {
		name string
		code string
		desc string
	}{
		{
			name: "empty file",
			code: `package main`,
			desc: "Should handle empty files gracefully",
		},
		{
			name: "syntax errors",
			code: `package main
func broken( {
	// Syntax error in function declaration
}`,
			desc: "Should handle syntax errors gracefully",
		},
		{
			name: "very long lines",
			code: `package main
func longLine() {
	query := "SELECT * FROM very_long_table_name_that_goes_on_and_on_and_on_and_on_and_on_and_on_and_on_and_on_and_on_and_on_and_on_and_on WHERE very_long_column_name_that_also_goes_on_and_on = " + userInput
}`,
			desc: "Should handle very long lines",
		},
		{
			name: "nested patterns",
			code: `package main
func nested() {
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			result += fmt.Sprintf("item %d-%d", i, j)
			go func() {
				db.Query("SELECT * FROM nested WHERE id = " + id)
			}()
		}
	}
}`,
			desc: "Should detect patterns in nested structures",
		},
		{
			name: "unicode and special characters",
			code: `package main
func unicode() {
	// TODO: implement unicode handling
	message := "Hello 世界"
	password := "пароль"
}`,
			desc: "Should handle unicode characters",
		},
	}
	
	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "edge.go")
			
			err := os.WriteFile(tmpFile, []byte(tc.code), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			
			// Should not panic or error
			results, err := validator.ValidateFile(tmpFile)
			if err != nil {
				t.Logf("Expected error for %s: %v", tc.desc, err)
			}
			
			t.Logf("%s: Found %d issues", tc.desc, len(results))
			for _, result := range results {
				t.Logf("  - %s: %s", result.RuleName, result.Message)
			}
		})
	}
}