package validation

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"time"
)

// RuleEngine manages and executes validation rules
type RuleEngine struct {
	rules           []ValidationRule
	customRules     []CustomRule
	businessLogic   BusinessLogicRules
}

// CustomRule represents a user-defined validation rule
type CustomRule struct {
	Name        string
	Description string
	Category    string
	Severity    ValidationSeverity
	Condition   func(content string, astFile *ast.File) []ValidationResult
}

// BusinessLogicRules contains rules specific to the news website business logic
type BusinessLogicRules struct {
	ArticleRules    []ValidationRule
	SEORules        []ValidationRule
	PerformanceRules []ValidationRule
	SecurityRules   []ValidationRule
}

// NewRuleEngine creates a new rule engine with business-specific rules
func NewRuleEngine() *RuleEngine {
	engine := &RuleEngine{}
	engine.initializeBusinessRules()
	return engine
}

// initializeBusinessRules sets up rules specific to the news website
func (r *RuleEngine) initializeBusinessRules() {
	r.businessLogic = BusinessLogicRules{
		ArticleRules: []ValidationRule{
			{
				Name:        "missing-article-validation",
				Description: "Article creation without proper validation",
				Category:    "business-logic",
				Severity:    SeverityHigh,
				Pattern:     regexp.MustCompile(`(?i)insert\s+into\s+articles.*values.*\([^)]*\)(?!.*validate)`),
			},
			{
				Name:        "missing-slug-generation",
				Description: "Article without slug generation",
				Category:    "seo",
				Severity:    SeverityHigh,
				Pattern:     regexp.MustCompile(`(?i)article.*\{[^}]*title[^}]*\}(?!.*slug)`),
			},
			{
				Name:        "missing-canonical-url",
				Description: "Article without canonical URL",
				Category:    "seo",
				Severity:    SeverityMedium,
				Pattern:     regexp.MustCompile(`(?i)article.*\{[^}]*\}(?!.*canonical)`),
			},
		},
		SEORules: []ValidationRule{
			{
				Name:        "missing-meta-tags",
				Description: "Missing SEO meta tags generation",
				Category:    "seo",
				Severity:    SeverityHigh,
				Pattern:     regexp.MustCompile(`(?i)func.*generate.*html.*\{(?!.*meta.*title|.*meta.*description)`),
			},
			{
				Name:        "missing-schema-markup",
				Description: "Missing structured data schema markup",
				Category:    "seo",
				Severity:    SeverityMedium,
				Pattern:     regexp.MustCompile(`(?i)article.*html.*\{(?!.*schema|.*json-ld)`),
			},
			{
				Name:        "invalid-hreflang",
				Description: "Invalid hreflang implementation",
				Category:    "seo",
				Severity:    SeverityMedium,
				Pattern:     regexp.MustCompile(`hreflang.*=.*"[^"]*"(?!.*[a-z]{2}(-[A-Z]{2})?)`),
			},
		},
		PerformanceRules: []ValidationRule{
			{
				Name:        "n-plus-one-query",
				Description: "Potential N+1 query problem",
				Category:    "performance",
				Severity:    SeverityCritical,
				Pattern:     regexp.MustCompile(`(?i)for.*range.*\{[^}]*(?:query|select)[^}]*\}`),
			},
			{
				Name:        "missing-cache-check",
				Description: "Database query without cache check",
				Category:    "performance",
				Severity:    SeverityHigh,
				Pattern:     regexp.MustCompile(`(?i)(?:query|select).*from.*articles(?!.*cache)`),
			},
			{
				Name:        "inefficient-pagination",
				Description: "Inefficient pagination using OFFSET",
				Category:    "performance",
				Severity:    SeverityMedium,
				Pattern:     regexp.MustCompile(`(?i)select.*from.*offset.*\$\d+.*limit`),
			},
		},
		SecurityRules: []ValidationRule{
			{
				Name:        "missing-auth-check",
				Description: "Admin endpoint without authentication check",
				Category:    "security",
				Severity:    SeverityCritical,
				Pattern:     regexp.MustCompile(`(?i)func.*admin.*handler.*\{(?!.*auth|.*permission)`),
			},
			{
				Name:        "missing-rate-limiting",
				Description: "API endpoint without rate limiting",
				Category:    "security",
				Severity:    SeverityHigh,
				Pattern:     regexp.MustCompile(`(?i)func.*api.*handler.*\{(?!.*rate.*limit)`),
			},
			{
				Name:        "unsafe-file-upload",
				Description: "File upload without security validation",
				Category:    "security",
				Severity:    SeverityCritical,
				Pattern:     regexp.MustCompile(`(?i)multipart\.file.*\{(?!.*validate.*type|.*sanitize)`),
			},
		},
	}
	
	// Combine all business rules
	r.rules = append(r.rules, r.businessLogic.ArticleRules...)
	r.rules = append(r.rules, r.businessLogic.SEORules...)
	r.rules = append(r.rules, r.businessLogic.PerformanceRules...)
	r.rules = append(r.rules, r.businessLogic.SecurityRules...)
}

// AddCustomRule adds a custom validation rule
func (r *RuleEngine) AddCustomRule(rule CustomRule) {
	r.customRules = append(r.customRules, rule)
}

// ExecuteRules runs all validation rules against the provided content and AST
func (r *RuleEngine) ExecuteRules(filePath, content string, astFile *ast.File, fset *token.FileSet) []ValidationResult {
	var results []ValidationResult
	
	// Execute regex-based rules
	for _, rule := range r.rules {
		if rule.Pattern != nil {
			matches := rule.Pattern.FindAllStringIndex(content, -1)
			for _, match := range matches {
				line, column := getLineColumn(content, match[0])
				snippet := getCodeSnippet(content, match[0], match[1])
				
				results = append(results, ValidationResult{
					Severity:    rule.Severity,
					Category:    rule.Category,
					Message:     rule.Description,
					File:        filePath,
					Line:        line,
					Column:      column,
					RuleName:    rule.Name,
					CodeSnippet: snippet,
					Suggestion:  r.getSuggestionForRule(rule.Name),
				})
			}
		}
	}
	
	// Execute AST-based rules
	if astFile != nil {
		for _, rule := range r.rules {
			if rule.ASTChecker != nil {
				astResults := rule.ASTChecker(astFile, fset)
				results = append(results, astResults...)
			}
		}
	}
	
	// Execute custom rules
	for _, rule := range r.customRules {
		customResults := rule.Condition(content, astFile)
		results = append(results, customResults...)
	}
	
	return results
}

// getSuggestionForRule returns business-specific suggestions
func (r *RuleEngine) getSuggestionForRule(ruleName string) string {
	suggestions := map[string]string{
		"missing-article-validation": "Add article validation: if err := models.ValidateStruct(article); err != nil { return err }",
		"missing-slug-generation":    "Generate slug from title: article.Slug = utils.GenerateSlug(article.Title)",
		"missing-canonical-url":      "Set canonical URL: article.CanonicalURL = fmt.Sprintf(\"/articles/%s\", article.Slug)",
		"missing-meta-tags":          "Add meta tags: <meta name=\"description\" content=\"{{.MetaDescription}}\">",
		"missing-schema-markup":      "Add JSON-LD schema: <script type=\"application/ld+json\">{{.SchemaMarkup}}</script>",
		"invalid-hreflang":          "Use valid hreflang format: hreflang=\"en-US\" or hreflang=\"fa\"",
		"n-plus-one-query":          "Use JOIN or preload related data: db.Preload(\"Author\").Find(&articles)",
		"missing-cache-check":        "Check cache first: if cached := cache.Get(key); cached != nil { return cached }",
		"inefficient-pagination":     "Use cursor-based pagination: WHERE id > $1 ORDER BY id LIMIT $2",
		"missing-auth-check":         "Add authentication: if !auth.IsAdmin(r) { return http.StatusUnauthorized }",
		"missing-rate-limiting":      "Add rate limiting: if !rateLimiter.Allow(clientIP) { return http.StatusTooManyRequests }",
		"unsafe-file-upload":         "Validate file type: if !isAllowedFileType(file.Header.Get(\"Content-Type\")) { return error }",
	}
	
	if suggestion, exists := suggestions[ruleName]; exists {
		return suggestion
	}
	
	return "Review this code pattern for business logic compliance"
}

// ValidateBusinessLogic performs comprehensive business logic validation
func (r *RuleEngine) ValidateBusinessLogic(filePath, content string, astFile *ast.File, fset *token.FileSet) BusinessLogicReport {
	report := BusinessLogicReport{
		FilePath:    filePath,
		GeneratedAt: time.Now(),
	}
	
	// Check article-related business logic
	articleResults := r.validateArticleLogic(content, astFile, fset)
	report.ArticleIssues = articleResults
	
	// Check SEO compliance
	seoResults := r.validateSEOCompliance(content, astFile, fset)
	report.SEOIssues = seoResults
	
	// Check performance patterns
	perfResults := r.validatePerformancePatterns(content, astFile, fset)
	report.PerformanceIssues = perfResults
	
	// Check security patterns
	secResults := r.validateSecurityPatterns(content, astFile, fset)
	report.SecurityIssues = secResults
	
	// Calculate overall score
	report.OverallScore = r.calculateBusinessLogicScore(report)
	
	return report
}

// validateArticleLogic checks article-specific business logic
func (r *RuleEngine) validateArticleLogic(content string, astFile *ast.File, fset *token.FileSet) []ValidationResult {
	var results []ValidationResult
	
	// Check for proper article creation flow
	if strings.Contains(content, "CreateArticle") || strings.Contains(content, "NewArticle") {
		// Ensure slug generation
		if !strings.Contains(content, "GenerateSlug") && !strings.Contains(content, "slug") {
			results = append(results, ValidationResult{
				Severity:   SeverityHigh,
				Category:   "business-logic",
				Message:    "Article creation without slug generation",
				RuleName:   "missing-slug-in-creation",
				Suggestion: "Add slug generation: article.Slug = utils.GenerateSlug(article.Title)",
			})
		}
		
		// Ensure SEO metadata
		if !strings.Contains(content, "MetaTitle") && !strings.Contains(content, "MetaDescription") {
			results = append(results, ValidationResult{
				Severity:   SeverityMedium,
				Category:   "seo",
				Message:    "Article creation without SEO metadata",
				RuleName:   "missing-seo-metadata",
				Suggestion: "Add SEO metadata generation for MetaTitle and MetaDescription",
			})
		}
	}
	
	return results
}

// validateSEOCompliance checks SEO-related patterns
func (r *RuleEngine) validateSEOCompliance(content string, astFile *ast.File, fset *token.FileSet) []ValidationResult {
	var results []ValidationResult
	
	// Check for canonical URL handling
	if strings.Contains(content, "canonical") {
		// Ensure no circular references
		circularPattern := regexp.MustCompile(`canonical.*=.*canonical`)
		if circularPattern.MatchString(content) {
			results = append(results, ValidationResult{
				Severity:   SeverityCritical,
				Category:   "seo",
				Message:    "Potential circular canonical URL reference",
				RuleName:   "circular-canonical",
				Suggestion: "Ensure canonical URLs don't reference each other in a loop",
			})
		}
	}
	
	return results
}

// validatePerformancePatterns checks for performance anti-patterns
func (r *RuleEngine) validatePerformancePatterns(content string, astFile *ast.File, fset *token.FileSet) []ValidationResult {
	var results []ValidationResult
	
	// Check for database queries in loops
	loopQueryPattern := regexp.MustCompile(`(?i)for.*\{[^}]*(?:db\.query|db\.exec|\.find)[^}]*\}`)
	if loopQueryPattern.MatchString(content) {
		results = append(results, ValidationResult{
			Severity:   SeverityCritical,
			Category:   "performance",
			Message:    "Database query inside loop - potential N+1 problem",
			RuleName:   "query-in-loop",
			Suggestion: "Move query outside loop or use batch operations",
		})
	}
	
	return results
}

// validateSecurityPatterns checks for security issues
func (r *RuleEngine) validateSecurityPatterns(content string, astFile *ast.File, fset *token.FileSet) []ValidationResult {
	var results []ValidationResult
	
	// Check for SQL injection patterns
	sqlInjectionPattern := regexp.MustCompile(`(?i)(?:query|exec)\s*\([^)]*\+[^)]*\)`)
	if sqlInjectionPattern.MatchString(content) {
		results = append(results, ValidationResult{
			Severity:   SeverityCritical,
			Category:   "security",
			Message:    "Potential SQL injection vulnerability",
			RuleName:   "sql-injection-risk",
			Suggestion: "Use parameterized queries with placeholders",
		})
	}
	
	return results
}

// calculateBusinessLogicScore calculates an overall business logic compliance score
func (r *RuleEngine) calculateBusinessLogicScore(report BusinessLogicReport) float64 {
	totalIssues := len(report.ArticleIssues) + len(report.SEOIssues) + 
	              len(report.PerformanceIssues) + len(report.SecurityIssues)
	
	if totalIssues == 0 {
		return 100.0
	}
	
	// Weight different issue types
	criticalWeight := 10.0
	highWeight := 5.0
	mediumWeight := 2.0
	lowWeight := 1.0
	
	var weightedScore float64
	allIssues := append(report.ArticleIssues, report.SEOIssues...)
	allIssues = append(allIssues, report.PerformanceIssues...)
	allIssues = append(allIssues, report.SecurityIssues...)
	
	for _, issue := range allIssues {
		switch issue.Severity {
		case SeverityCritical:
			weightedScore += criticalWeight
		case SeverityHigh:
			weightedScore += highWeight
		case SeverityMedium:
			weightedScore += mediumWeight
		case SeverityLow:
			weightedScore += lowWeight
		}
	}
	
	// Calculate score (100 - penalty)
	maxPossibleScore := float64(totalIssues) * criticalWeight
	score := 100.0 - (weightedScore/maxPossibleScore)*100.0
	
	if score < 0 {
		score = 0
	}
	
	return score
}

// BusinessLogicReport contains business logic validation results
type BusinessLogicReport struct {
	FilePath          string             `json:"file_path"`
	ArticleIssues     []ValidationResult `json:"article_issues"`
	SEOIssues         []ValidationResult `json:"seo_issues"`
	PerformanceIssues []ValidationResult `json:"performance_issues"`
	SecurityIssues    []ValidationResult `json:"security_issues"`
	OverallScore      float64            `json:"overall_score"`
	GeneratedAt       time.Time          `json:"generated_at"`
}

// Helper functions (moved from ai_code_validator.go to avoid duplication)
func getLineColumn(content string, offset int) (int, int) {
	line := 1
	column := 1
	
	for i := 0; i < offset && i < len(content); i++ {
		if content[i] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	
	return line, column
}

func getCodeSnippet(content string, start, end int) string {
	// Find the start of the line
	lineStart := start
	for lineStart > 0 && content[lineStart-1] != '\n' {
		lineStart--
	}
	
	// Find the end of the line
	lineEnd := end
	for lineEnd < len(content) && content[lineEnd] != '\n' {
		lineEnd++
	}
	
	return strings.TrimSpace(content[lineStart:lineEnd])
}