package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAICodeValidator_ValidateFile(t *testing.T) {
	validator := NewAICodeValidator()
	
	tests := []struct {
		name           string
		code           string
		expectedIssues int
		expectedRules  []string
	}{
		{
			name: "missing error handling",
			code: `package main
func main() {
	result, err := db.Query("SELECT * FROM articles")
	data, err := file.Read(buffer)
}`,
			expectedIssues: 2,
			expectedRules:  []string{"potential-unhandled-error"},
		},
		{
			name: "hardcoded values",
			code: `package main
func connect() {
	db.Connect("localhost:5432")
	auth := "password"
}`,
			expectedIssues: 2,
			expectedRules:  []string{"hardcoded-values"},
		},
		{
			name: "SQL injection risk",
			code: `package main
func getUser(id string) {
	query := "SELECT * FROM users WHERE id = " + id
	db.Query(query)
}`,
			expectedIssues: 1,
			expectedRules:  []string{"sql-injection-risk"},
		},
		{
			name: "missing context timeout",
			code: `package main
func handler(w http.ResponseWriter, r *http.Request) {
	// Process request without timeout
	processRequest(r)
}`,
			expectedIssues: 1,
			expectedRules:  []string{"missing-context"},
		},
		{
			name: "missing input validation",
			code: `package main
func handler(w http.ResponseWriter, r *http.Request) {
	userID := r.FormValue("user_id")
	processUser(userID)
}`,
			expectedIssues: 1,
			expectedRules:  []string{"missing-validation"},
		},
		{
			name: "AI-generated TODO comments",
			code: `package main
func processData() {
	// TODO: implement proper error handling
	// FIXME: add validation logic
	data := getData()
}`,
			expectedIssues: 2,
			expectedRules:  []string{"ai-generated-todo"},
		},
		{
			name: "inefficient string concatenation",
			code: `package main
func buildString() string {
	result := ""
	for i := 0; i < 1000; i++ {
		result += fmt.Sprintf("item %d", i)
	}
	return result
}`,
			expectedIssues: 1,
			expectedRules:  []string{"inefficient-string-concat"},
		},
		{
			name: "unsafe type assertion",
			code: `package main
func processInterface(data interface{}) {
	str := data.(string)
	fmt.Println(str)
}`,
			expectedIssues: 1,
			expectedRules:  []string{"unsafe-type-assertion"},
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
			expectedIssues: 1,
			expectedRules:  []string{"goroutine-leak"},
		},
		{
			name: "missing transaction for multiple operations",
			code: `package main
func updateUserData(userID int) {
	db.Exec("UPDATE users SET last_login = NOW() WHERE id = ?", userID)
	db.Exec("INSERT INTO user_activity (user_id, action) VALUES (?, 'login')", userID)
}`,
			expectedIssues: 1,
			expectedRules:  []string{"missing-transaction"},
		},
		{
			name: "good code with proper patterns",
			code: `package main
import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), config.Timeout)
	defer cancel()
	
	userID := r.FormValue("user_id")
	if err := validateUserID(userID); err != nil {
		log.Printf("Invalid user ID: %v", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	
	user, err := db.QueryContext(ctx, "SELECT id, name FROM users WHERE id = $1", userID)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(user)
}`,
			expectedIssues: 0,
			expectedRules:  []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := createTempFile(t, tt.code)
			defer os.Remove(tmpFile)
			
			results, err := validator.ValidateFile(tmpFile)
			if err != nil {
				t.Fatalf("ValidateFile() error = %v", err)
			}
			
			if len(results) != tt.expectedIssues {
				t.Errorf("ValidateFile() found %d issues, expected %d", len(results), tt.expectedIssues)
				for _, result := range results {
					t.Logf("Issue: %s - %s", result.RuleName, result.Message)
				}
			}
			
			// Check if expected rules are found
			foundRules := make(map[string]bool)
			for _, result := range results {
				foundRules[result.RuleName] = true
			}
			
			for _, expectedRule := range tt.expectedRules {
				if !foundRules[expectedRule] {
					t.Errorf("Expected rule %s not found in results", expectedRule)
				}
			}
		})
	}
}

func TestAICodeValidator_RequiresManualReview(t *testing.T) {
	validator := NewAICodeValidator()
	
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name: "complex database transaction",
			code: `func processArticles() {
				tx := database.Begin()
				defer tx.Rollback()
				// Complex transaction logic
			}`,
			expected: true,
		},
		{
			name: "complex concurrency pattern",
			code: `func processData() {
				ch := make(chan data)
				go func() {
					for item := range ch {
						goroutine.Process(item)
					}
				}()
			}`,
			expected: true,
		},
		{
			name: "reflection usage",
			code: `func dynamicCall() {
				v := reflect.ValueOf(obj)
				v.MethodByName("Process").Call(nil)
			}`,
			expected: true,
		},
		{
			name: "unsafe operations",
			code: `func unsafeOperation() {
				ptr := unsafe.Pointer(&data)
				result := (*int)(ptr)
			}`,
			expected: true,
		},
		{
			name: "mutex synchronization",
			code: `func criticalSection() {
				var mu sync.Mutex
				mu.Lock()
				defer mu.Unlock()
			}`,
			expected: true,
		},
		{
			name: "atomic operations",
			code: `func atomicIncrement() {
				atomic.AddInt64(&counter, 1)
			}`,
			expected: true,
		},
		{
			name: "cryptographic operations",
			code: `func hashData() {
				hasher := crypto.SHA256.New()
				hasher.Write(data)
			}`,
			expected: true,
		},
		{
			name: "panic/recover usage",
			code: `func riskyOperation() {
				defer func() {
					if r := recover(); r != nil {
						panic(r)
					}
				}()
			}`,
			expected: true,
		},
		{
			name: "simple function",
			code: `func simpleAdd(a, b int) int {
				return a + b
			}`,
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.requiresManualReview(tt.code)
			if result != tt.expected {
				t.Errorf("requiresManualReview() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestAICodeValidator_GenerateReport(t *testing.T) {
	validator := NewAICodeValidator()
	
	code := `package main
import "database/sql"

func badFunction() {
	db.Query("SELECT * FROM users WHERE id = " + userID)
	password := "hardcoded_password"
	file.Read(buffer)
}`
	
	tmpFile := createTempFile(t, code)
	defer os.Remove(tmpFile)
	
	report, err := validator.GenerateReport(tmpFile)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}
	
	if report == nil {
		t.Fatal("GenerateReport() returned nil report")
	}
	
	if report.Summary.TotalIssues == 0 {
		t.Error("Expected issues to be found in bad code")
	}
	
	if report.Summary.CriticalIssues == 0 {
		t.Error("Expected critical issues to be found")
	}
	
	if report.FilePath != tmpFile {
		t.Errorf("Report file path = %s, expected %s", report.FilePath, tmpFile)
	}
	
	if report.GeneratedAt.IsZero() {
		t.Error("Report should have generation timestamp")
	}
}

func TestValidationReport_ShouldBlockDeployment(t *testing.T) {
	tests := []struct {
		name     string
		summary  ValidationSummary
		expected bool
	}{
		{
			name: "critical issues should block",
			summary: ValidationSummary{
				CriticalIssues: 1,
				HighIssues:     2,
			},
			expected: true,
		},
		{
			name: "many high issues should block",
			summary: ValidationSummary{
				CriticalIssues: 0,
				HighIssues:     4,
			},
			expected: true,
		},
		{
			name: "few high issues should not block",
			summary: ValidationSummary{
				CriticalIssues: 0,
				HighIssues:     2,
				MediumIssues:   5,
			},
			expected: false,
		},
		{
			name: "only medium/low issues should not block",
			summary: ValidationSummary{
				CriticalIssues: 0,
				HighIssues:     0,
				MediumIssues:   10,
				LowIssues:      20,
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &ValidationReport{
				Summary: tt.summary,
			}
			
			result := report.ShouldBlockDeployment()
			if result != tt.expected {
				t.Errorf("ShouldBlockDeployment() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestAICodeValidator_GetSuggestion(t *testing.T) {
	validator := NewAICodeValidator()
	
	tests := []struct {
		ruleName string
		expected string
	}{
		{
			ruleName: "missing-error-handling",
			expected: "Add proper error handling: if err != nil { return err }",
		},
		{
			ruleName: "sql-injection-risk",
			expected: "Use parameterized queries with placeholders ($1, $2, etc.) instead of string concatenation",
		},
		{
			ruleName: "unknown-rule",
			expected: "Review this code pattern for potential issues",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.ruleName, func(t *testing.T) {
			result := validator.getSuggestion(tt.ruleName)
			if result != tt.expected {
				t.Errorf("getSuggestion(%s) = %s, expected %s", tt.ruleName, result, tt.expected)
			}
		})
	}
}

func TestAICodeValidator_LineColumnCalculation(t *testing.T) {
	validator := NewAICodeValidator()
	
	content := `line 1
line 2
line 3 with issue here
line 4`
	
	// Find position of "issue"
	offset := strings.Index(content, "issue")
	if offset == -1 {
		t.Fatal("Could not find 'issue' in test content")
	}
	
	line, column := validator.getLineColumn(content, offset)
	
	expectedLine := 3
	expectedColumn := 12 // "issue" starts at column 12 in line 3
	
	if line != expectedLine {
		t.Errorf("getLineColumn() line = %d, expected %d", line, expectedLine)
	}
	
	if column != expectedColumn {
		t.Errorf("getLineColumn() column = %d, expected %d", column, expectedColumn)
	}
}

func TestAICodeValidator_CodeSnippetExtraction(t *testing.T) {
	validator := NewAICodeValidator()
	
	content := `func main() {
	db.Query("SELECT * FROM users")
	fmt.Println("done")
}`
	
	// Find the Query call
	start := strings.Index(content, "db.Query")
	end := start + len("db.Query")
	
	snippet := validator.getCodeSnippet(content, start, end)
	
	expected := `db.Query("SELECT * FROM users")`
	if snippet != expected {
		t.Errorf("getCodeSnippet() = %s, expected %s", snippet, expected)
	}
}

func TestAICodeValidator_PerformanceWithLargeFile(t *testing.T) {
	validator := NewAICodeValidator()
	
	// Create a large Go file with multiple issues
	var codeBuilder strings.Builder
	codeBuilder.WriteString("package main\n\n")
	
	// Add 1000 functions with various issues
	for i := 0; i < 1000; i++ {
		codeBuilder.WriteString(fmt.Sprintf(`func function%d() {
	db.Query("SELECT * FROM table%d WHERE id = " + userID)
	password := "hardcoded%d"
	file.Read(buffer)
}

`, i, i, i))
	}
	
	tmpFile := createTempFile(t, codeBuilder.String())
	defer os.Remove(tmpFile)
	
	start := time.Now()
	results, err := validator.ValidateFile(tmpFile)
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("ValidateFile() error = %v", err)
	}
	
	// Should complete within reasonable time (< 5 seconds for 1000 functions)
	if duration > 5*time.Second {
		t.Errorf("Validation took too long: %v", duration)
	}
	
	// Should find multiple issues
	if len(results) == 0 {
		t.Error("Expected to find issues in large file with problems")
	}
	
	t.Logf("Validated large file with %d functions in %v, found %d issues", 
		1000, duration, len(results))
}

func TestAICodeValidator_ConcurrentValidation(t *testing.T) {
	validator := NewAICodeValidator()
	
	// Test concurrent validation of multiple files
	codes := []string{
		`package main
func test1() { db.Query("SELECT * FROM users WHERE id = " + id) }`,
		`package main  
func test2() { password := "hardcoded" }`,
		`package main
func test3() { file.Read(buffer) }`,
	}
	
	var tmpFiles []string
	for _, code := range codes {
		tmpFile := createTempFile(t, code)
		tmpFiles = append(tmpFiles, tmpFile)
		defer os.Remove(tmpFile)
	}
	
	// Validate files concurrently
	results := make(chan []ValidationResult, len(tmpFiles))
	errors := make(chan error, len(tmpFiles))
	
	for _, file := range tmpFiles {
		go func(f string) {
			res, err := validator.ValidateFile(f)
			if err != nil {
				errors <- err
				return
			}
			results <- res
		}(file)
	}
	
	// Collect results
	var allResults []ValidationResult
	for range tmpFiles {
		select {
		case res := <-results:
			allResults = append(allResults, res...)
		case err := <-errors:
			t.Fatalf("Concurrent validation error: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Concurrent validation timed out")
		}
	}
	
	if len(allResults) == 0 {
		t.Error("Expected to find issues in concurrent validation")
	}
}

// Helper function to create temporary files for testing
func createTempFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	
	return tmpFile
}

// Helper function to create temporary files for benchmarking
func createTempFileForBenchmark(b *testing.B, content string) string {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	
	return tmpFile
}

// Benchmark tests
func BenchmarkAICodeValidator_ValidateFile(b *testing.B) {
	validator := NewAICodeValidator()
	
	code := `package main
import "database/sql"

func problematicFunction() {
	db.Query("SELECT * FROM users WHERE id = " + userID)
	password := "hardcoded_password"
	file.Read(buffer)
	
	for _, user := range users {
		db.Query("SELECT * FROM profiles WHERE user_id = " + user.ID)
	}
}`
	
	tmpFile := createTempFileForBenchmark(b, code)
	defer os.Remove(tmpFile)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateFile(tmpFile)
		if err != nil {
			b.Fatalf("ValidateFile() error = %v", err)
		}
	}
}

func BenchmarkAICodeValidator_PatternMatching(b *testing.B) {
	validator := NewAICodeValidator()
	
	// Large code sample with multiple patterns
	var codeBuilder strings.Builder
	for i := 0; i < 100; i++ {
		codeBuilder.WriteString(fmt.Sprintf(`
func function%d() {
	db.Query("SELECT * FROM table WHERE id = " + id)
	password := "secret%d"
	if err := someCall(); err != nil {
		return err
	}
}`, i, i))
	}
	
	content := codeBuilder.String()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, rule := range validator.rules {
			if rule.Pattern != nil {
				rule.Pattern.FindAllStringIndex(content, -1)
			}
		}
	}
}