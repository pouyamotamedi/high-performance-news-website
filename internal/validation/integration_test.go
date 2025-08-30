package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAICodeValidator_Integration(t *testing.T) {
	validator := NewAICodeValidator()
	
	// Test with a realistic AI-generated code sample
	code := `package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

// TODO: implement proper error handling
func createArticle(title, content string) error {
	// Hardcoded database connection
	db, err := sql.Open("postgres", "localhost:5432")
	
	// SQL injection vulnerability
	query := "INSERT INTO articles (title, content) VALUES ('" + title + "', '" + content + "')"
	db.Exec(query)
	
	// Missing transaction for multiple operations
	db.Exec("UPDATE article_count SET count = count + 1")
	
	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Missing context timeout
	// Missing input validation
	title := r.FormValue("title")
	
	// Unsafe type assertion
	data := r.Context().Value("user").(User)
	
	// Inefficient string concatenation in loop
	result := ""
	for i := 0; i < 100; i++ {
		result += fmt.Sprintf("item %d ", i)
	}
	
	// Goroutine without cleanup
	go func() {
		for {
			processData()
		}
	}()
}

type User struct {
	ID   int
	Name string
}

func processData() {
	// Implementation
}
`
	
	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	
	err := os.WriteFile(tmpFile, []byte(code), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	
	// Validate the file
	results, err := validator.ValidateFile(tmpFile)
	if err != nil {
		t.Fatalf("ValidateFile() error = %v", err)
	}
	
	// Should find multiple issues
	if len(results) == 0 {
		t.Error("Expected to find validation issues in problematic code")
	}
	
	// Check for specific issue categories
	categories := make(map[string]int)
	for _, result := range results {
		categories[result.Category]++
		t.Logf("Found issue: %s - %s (%s)", result.RuleName, result.Message, result.Category)
	}
	
	// Should find issues in multiple categories
	expectedCategories := []string{"maintainability", "security", "reliability"}
	for _, category := range expectedCategories {
		if categories[category] == 0 {
			t.Logf("Warning: No issues found in category %s", category)
		}
	}
	
	// Generate report
	report, err := validator.GenerateReport(tmpFile)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}
	
	if report.Summary.TotalIssues == 0 {
		t.Error("Report should show total issues > 0")
	}
	
	t.Logf("Validation complete: %d total issues found", report.Summary.TotalIssues)
	t.Logf("Critical: %d, High: %d, Medium: %d, Low: %d", 
		report.Summary.CriticalIssues, 
		report.Summary.HighIssues,
		report.Summary.MediumIssues,
		report.Summary.LowIssues)
}

func TestAICodeValidator_ManualReviewDetection(t *testing.T) {
	validator := NewAICodeValidator()
	
	// Test complex patterns that should trigger manual review
	complexCode := `package main

import (
	"reflect"
	"sync"
	"unsafe"
)

func complexFunction() {
	// Reflection usage
	v := reflect.ValueOf(obj)
	v.MethodByName("Process").Call(nil)
	
	// Unsafe operations
	ptr := unsafe.Pointer(&data)
	result := (*int)(ptr)
	
	// Mutex synchronization
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()
	
	// Database transaction
	tx := database.Begin()
	defer tx.Rollback()
}
`
	
	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "complex.go")
	
	err := os.WriteFile(tmpFile, []byte(complexCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	
	// Validate the file
	results, err := validator.ValidateFile(tmpFile)
	if err != nil {
		t.Fatalf("ValidateFile() error = %v", err)
	}
	
	// Should trigger manual review
	foundManualReview := false
	for _, result := range results {
		if result.Category == "manual-review" {
			foundManualReview = true
			break
		}
	}
	
	if !foundManualReview {
		t.Error("Expected manual review to be triggered for complex code patterns")
	}
}

func TestAICodeValidator_BusinessLogicPatterns(t *testing.T) {
	validator := NewAICodeValidator()
	
	// Test business logic specific to the news website
	newsCode := `package main

func createNewsArticle(title, content string) error {
	// Missing slug generation
	article := Article{
		Title:   title,
		Content: content,
	}
	
	// Missing SEO metadata
	// Missing canonical URL
	
	// Insert without validation
	db.Insert("articles", article)
	
	return nil
}

func publishArticle(id int) {
	// Multiple database operations without transaction
	db.Exec("UPDATE articles SET status = 'published' WHERE id = ?", id)
	db.Exec("INSERT INTO article_history (article_id, action) VALUES (?, 'published')", id)
}

type Article struct {
	ID      int
	Title   string
	Content string
}
`
	
	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "news.go")
	
	err := os.WriteFile(tmpFile, []byte(newsCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	
	// Validate the file
	results, err := validator.ValidateFile(tmpFile)
	if err != nil {
		t.Fatalf("ValidateFile() error = %v", err)
	}
	
	// Should find business logic issues
	if len(results) == 0 {
		t.Error("Expected to find business logic issues")
	}
	
	for _, result := range results {
		t.Logf("Business logic issue: %s - %s", result.RuleName, result.Message)
	}
}