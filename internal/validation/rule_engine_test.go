package validation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"time"
)

func TestRuleEngine_BusinessLogicValidation(t *testing.T) {
	engine := NewRuleEngine()
	
	tests := []struct {
		name           string
		code           string
		expectedIssues int
		expectedCategories []string
	}{
		{
			name: "article creation without slug",
			code: `package main
func CreateArticle(title string) Article {
	article := Article{
		Title: title,
		Content: "some content",
	}
	return article
}`,
			expectedIssues: 1,
			expectedCategories: []string{"business-logic"},
		},
		{
			name: "missing SEO meta tags in HTML generation",
			code: `package main
func generateHTML(article Article) string {
	html := "<html><head><title>" + article.Title + "</title></head></html>"
	return html
}`,
			expectedIssues: 1,
			expectedCategories: []string{"seo"},
		},
		{
			name: "N+1 query problem",
			code: `package main
func loadArticlesWithAuthors(articles []Article) {
	for _, article := range articles {
		author := db.Query("SELECT * FROM authors WHERE id = ?", article.AuthorID)
		article.Author = author
	}
}`,
			expectedIssues: 1,
			expectedCategories: []string{"performance"},
		},
		{
			name: "admin endpoint without auth check",
			code: `package main
func adminHandler(w http.ResponseWriter, r *http.Request) {
	// Delete all articles
	db.Exec("DELETE FROM articles")
}`,
			expectedIssues: 1,
			expectedCategories: []string{"security"},
		},
		{
			name: "proper article creation with all checks",
			code: `package main
func CreateArticle(title string) Article {
	article := Article{
		Title: title,
		Slug: utils.GenerateSlug(title),
		MetaTitle: generateMetaTitle(title),
		MetaDescription: generateMetaDescription(title),
		CanonicalURL: fmt.Sprintf("/articles/%s", slug),
	}
	
	if err := models.ValidateStruct(article); err != nil {
		return Article{}, err
	}
	
	return article
}`,
			expectedIssues: 0,
			expectedCategories: []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			astFile, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Logf("Parse error (expected for some tests): %v", err)
			}
			
			results := engine.ExecuteRules("test.go", tt.code, astFile, fset)
			
			if len(results) != tt.expectedIssues {
				t.Errorf("ExecuteRules() found %d issues, expected %d", len(results), tt.expectedIssues)
				for _, result := range results {
					t.Logf("Issue: %s - %s (%s)", result.RuleName, result.Message, result.Category)
				}
			}
			
			// Check categories
			foundCategories := make(map[string]bool)
			for _, result := range results {
				foundCategories[result.Category] = true
			}
			
			for _, expectedCategory := range tt.expectedCategories {
				if !foundCategories[expectedCategory] {
					t.Errorf("Expected category %s not found in results", expectedCategory)
				}
			}
		})
	}
}

func TestRuleEngine_ValidateBusinessLogic(t *testing.T) {
	engine := NewRuleEngine()
	
	code := `package main

func CreateArticle(title, content string) error {
	// Missing slug generation
	article := Article{
		Title: title,
		Content: content,
	}
	
	// Missing validation
	db.Insert("articles", article)
	return nil
}

func generateHTML(article Article) string {
	// Missing meta tags and schema markup
	return "<html><body>" + article.Content + "</body></html>"
}

func loadUserArticles(userID string) []Article {
	var articles []Article
	// N+1 query problem
	for _, category := range categories {
		categoryArticles := db.Query("SELECT * FROM articles WHERE category_id = " + category.ID)
		articles = append(articles, categoryArticles...)
	}
	return articles
}

func adminDeleteHandler(w http.ResponseWriter, r *http.Request) {
	// Missing auth check
	db.Exec("DELETE FROM articles WHERE id = " + r.FormValue("id"))
}`
	
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "test.go", code, parser.ParseComments)
	if err != nil {
		t.Logf("Parse error: %v", err)
	}
	
	report := engine.ValidateBusinessLogic("test.go", code, astFile, fset)
	
	if len(report.ArticleIssues) == 0 {
		t.Error("Expected article issues to be found")
	}
	
	if len(report.SEOIssues) == 0 {
		t.Error("Expected SEO issues to be found")
	}
	
	if len(report.PerformanceIssues) == 0 {
		t.Error("Expected performance issues to be found")
	}
	
	if len(report.SecurityIssues) == 0 {
		t.Error("Expected security issues to be found")
	}
	
	if report.OverallScore >= 80.0 {
		t.Errorf("Expected low score for problematic code, got %.2f", report.OverallScore)
	}
	
	if report.GeneratedAt.IsZero() {
		t.Error("Report should have generation timestamp")
	}
}

func TestRuleEngine_AddCustomRule(t *testing.T) {
	engine := NewRuleEngine()
	
	// Add custom rule for checking specific business logic
	customRule := CustomRule{
		Name:        "missing-audit-log",
		Description: "Database modification without audit logging",
		Category:    "compliance",
		Severity:    SeverityHigh,
		Condition: func(content string, astFile *ast.File) []ValidationResult {
			var results []ValidationResult
			
			// Check for INSERT/UPDATE/DELETE without audit logging
			if strings.Contains(content, "INSERT") || strings.Contains(content, "UPDATE") || strings.Contains(content, "DELETE") {
				if !strings.Contains(content, "auditLog") && !strings.Contains(content, "logChange") {
					results = append(results, ValidationResult{
						Severity:   SeverityHigh,
						Category:   "compliance",
						Message:    "Database modification without audit logging",
						RuleName:   "missing-audit-log",
						Suggestion: "Add audit logging: auditLog.LogChange(operation, table, recordID)",
					})
				}
			}
			
			return results
		},
	}
	
	engine.AddCustomRule(customRule)
	
	code := `package main
func updateArticle(id int, title string) {
	db.Exec("UPDATE articles SET title = ? WHERE id = ?", title, id)
}`
	
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "test.go", code, parser.ParseComments)
	if err != nil {
		t.Logf("Parse error: %v", err)
	}
	
	results := engine.ExecuteRules("test.go", code, astFile, fset)
	
	// Should find the custom rule violation
	found := false
	for _, result := range results {
		if result.RuleName == "missing-audit-log" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Custom rule 'missing-audit-log' was not triggered")
	}
}

func TestRuleEngine_SEOValidation(t *testing.T) {
	engine := NewRuleEngine()
	
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name: "circular canonical reference",
			code: `package main
func setCanonical(article *Article) {
	article.CanonicalURL = article.CanonicalURL + "/canonical"
}`,
			expected: true,
		},
		{
			name: "proper canonical handling",
			code: `package main
func setCanonical(article *Article) {
	article.CanonicalURL = fmt.Sprintf("/articles/%s", article.Slug)
}`,
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			astFile, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Logf("Parse error: %v", err)
			}
			
			results := engine.validateSEOCompliance(tt.code, astFile, fset)
			
			hasCircularCanonical := false
			for _, result := range results {
				if result.RuleName == "circular-canonical" {
					hasCircularCanonical = true
					break
				}
			}
			
			if hasCircularCanonical != tt.expected {
				t.Errorf("validateSEOCompliance() circular canonical = %v, expected %v", hasCircularCanonical, tt.expected)
			}
		})
	}
}

func TestRuleEngine_PerformanceValidation(t *testing.T) {
	engine := NewRuleEngine()
	
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name: "query in loop",
			code: `package main
func loadData() {
	for _, id := range ids {
		user := db.Query("SELECT * FROM users WHERE id = ?", id)
		users = append(users, user)
	}
}`,
			expected: true,
		},
		{
			name: "proper batch query",
			code: `package main
func loadData() {
	users := db.Query("SELECT * FROM users WHERE id IN (?)", ids)
	return users
}`,
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			astFile, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Logf("Parse error: %v", err)
			}
			
			results := engine.validatePerformancePatterns(tt.code, astFile, fset)
			
			hasQueryInLoop := false
			for _, result := range results {
				if result.RuleName == "query-in-loop" {
					hasQueryInLoop = true
					break
				}
			}
			
			if hasQueryInLoop != tt.expected {
				t.Errorf("validatePerformancePatterns() query in loop = %v, expected %v", hasQueryInLoop, tt.expected)
			}
		})
	}
}

func TestRuleEngine_SecurityValidation(t *testing.T) {
	engine := NewRuleEngine()
	
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name: "SQL injection vulnerability",
			code: `package main
func getUser(id string) {
	query := "SELECT * FROM users WHERE id = " + id
	db.Query(query)
}`,
			expected: true,
		},
		{
			name: "parameterized query",
			code: `package main
func getUser(id string) {
	db.Query("SELECT * FROM users WHERE id = $1", id)
}`,
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			astFile, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Logf("Parse error: %v", err)
			}
			
			results := engine.validateSecurityPatterns(tt.code, astFile, fset)
			
			hasSQLInjection := false
			for _, result := range results {
				if result.RuleName == "sql-injection-risk" {
					hasSQLInjection = true
					break
				}
			}
			
			if hasSQLInjection != tt.expected {
				t.Errorf("validateSecurityPatterns() SQL injection = %v, expected %v", hasSQLInjection, tt.expected)
			}
		})
	}
}

func TestRuleEngine_CalculateBusinessLogicScore(t *testing.T) {
	engine := NewRuleEngine()
	
	tests := []struct {
		name     string
		report   BusinessLogicReport
		expected float64
	}{
		{
			name: "perfect code",
			report: BusinessLogicReport{
				ArticleIssues:     []ValidationResult{},
				SEOIssues:         []ValidationResult{},
				PerformanceIssues: []ValidationResult{},
				SecurityIssues:    []ValidationResult{},
			},
			expected: 100.0,
		},
		{
			name: "one critical issue",
			report: BusinessLogicReport{
				SecurityIssues: []ValidationResult{
					{Severity: SeverityCritical},
				},
			},
			expected: 0.0, // Single critical issue should result in 0 score
		},
		{
			name: "mixed severity issues",
			report: BusinessLogicReport{
				ArticleIssues: []ValidationResult{
					{Severity: SeverityHigh},
					{Severity: SeverityMedium},
				},
				SEOIssues: []ValidationResult{
					{Severity: SeverityLow},
				},
			},
			expected: 20.0, // Should be calculated based on weighted scoring
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.calculateBusinessLogicScore(tt.report)
			
			// Allow some tolerance for floating point comparison
			tolerance := 5.0
			if score < tt.expected-tolerance || score > tt.expected+tolerance {
				t.Errorf("calculateBusinessLogicScore() = %.2f, expected ~%.2f", score, tt.expected)
			}
		})
	}
}

func TestRuleEngine_GetSuggestionForRule(t *testing.T) {
	engine := NewRuleEngine()
	
	tests := []struct {
		ruleName string
		contains string
	}{
		{
			ruleName: "missing-article-validation",
			contains: "ValidateStruct",
		},
		{
			ruleName: "missing-slug-generation",
			contains: "GenerateSlug",
		},
		{
			ruleName: "n-plus-one-query",
			contains: "JOIN",
		},
		{
			ruleName: "missing-auth-check",
			contains: "IsAdmin",
		},
		{
			ruleName: "unknown-rule",
			contains: "business logic compliance",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.ruleName, func(t *testing.T) {
			suggestion := engine.getSuggestionForRule(tt.ruleName)
			
			if !strings.Contains(suggestion, tt.contains) {
				t.Errorf("getSuggestionForRule(%s) = %s, should contain %s", 
					tt.ruleName, suggestion, tt.contains)
			}
		})
	}
}

func TestRuleEngine_ConcurrentExecution(t *testing.T) {
	engine := NewRuleEngine()
	
	codes := []string{
		`package main
func test1() { db.Query("SELECT * FROM users WHERE id = " + id) }`,
		`package main  
func test2() { article := Article{Title: "test"} }`,
		`package main
func test3() { 
	for _, user := range users {
		db.Query("SELECT * FROM profiles WHERE user_id = ?", user.ID)
	}
}`,
	}
	
	// Execute rules concurrently
	results := make(chan []ValidationResult, len(codes))
	
	for i, code := range codes {
		go func(idx int, c string) {
			fset := token.NewFileSet()
			astFile, err := parser.ParseFile(fset, "test.go", c, parser.ParseComments)
			if err != nil {
				t.Logf("Parse error for code %d: %v", idx, err)
			}
			
			res := engine.ExecuteRules("test.go", c, astFile, fset)
			results <- res
		}(i, code)
	}
	
	// Collect results
	var allResults []ValidationResult
	for i := 0; i < len(codes); i++ {
		select {
		case res := <-results:
			allResults = append(allResults, res...)
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent rule execution timed out")
		}
	}
	
	if len(allResults) == 0 {
		t.Error("Expected to find issues in concurrent execution")
	}
}

// Benchmark tests
func BenchmarkRuleEngine_ExecuteRules(b *testing.B) {
	engine := NewRuleEngine()
	
	code := `package main
import "database/sql"

func problematicFunction() {
	// Multiple issues for comprehensive testing
	db.Query("SELECT * FROM users WHERE id = " + userID)
	article := Article{Title: "test"}
	
	for _, user := range users {
		db.Query("SELECT * FROM profiles WHERE user_id = " + user.ID)
	}
	
	adminHandler := func(w http.ResponseWriter, r *http.Request) {
		db.Exec("DELETE FROM articles")
	}
}`
	
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "test.go", code, parser.ParseComments)
	if err != nil {
		b.Fatalf("Parse error: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.ExecuteRules("test.go", code, astFile, fset)
	}
}

func BenchmarkRuleEngine_ValidateBusinessLogic(b *testing.B) {
	engine := NewRuleEngine()
	
	// Large code sample with multiple business logic patterns
	var codeBuilder strings.Builder
	codeBuilder.WriteString("package main\n\n")
	
	for i := 0; i < 50; i++ {
		codeBuilder.WriteString(fmt.Sprintf(`
func createArticle%d(title string) {
	article := Article{Title: title}
	db.Insert("articles", article)
}

func generateHTML%d(article Article) string {
	return "<html>" + article.Content + "</html>"
}

func loadData%d() {
	for _, id := range ids {
		db.Query("SELECT * FROM table WHERE id = " + id)
	}
}
`, i, i, i))
	}
	
	code := codeBuilder.String()
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "test.go", code, parser.ParseComments)
	if err != nil {
		b.Fatalf("Parse error: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.ValidateBusinessLogic("test.go", code, astFile, fset)
	}
}