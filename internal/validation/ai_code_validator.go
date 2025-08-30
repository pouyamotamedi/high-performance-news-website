package validation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
	"time"
)

// ValidationSeverity represents the severity level of a validation issue
type ValidationSeverity string

const (
	SeverityCritical ValidationSeverity = "critical"
	SeverityHigh     ValidationSeverity = "high"
	SeverityMedium   ValidationSeverity = "medium"
	SeverityLow      ValidationSeverity = "low"
)

// ValidationResult represents a single validation issue
type ValidationResult struct {
	Severity    ValidationSeverity `json:"severity"`
	Category    string            `json:"category"`
	Message     string            `json:"message"`
	File        string            `json:"file"`
	Line        int               `json:"line"`
	Column      int               `json:"column"`
	Suggestion  string            `json:"suggestion"`
	RuleName    string            `json:"rule_name"`
	CodeSnippet string            `json:"code_snippet"`
}

// AICodeValidator validates AI-generated code for common issues
type AICodeValidator struct {
	rules         []ValidationRule
	fileSet       *token.FileSet
	manualReview  []string // Patterns that require manual review
}

// ValidationRule represents a single validation rule
type ValidationRule struct {
	Name        string
	Description string
	Category    string
	Severity    ValidationSeverity
	Pattern     *regexp.Regexp
	ASTChecker  func(*ast.File, *token.FileSet) []ValidationResult
}

// NewAICodeValidator creates a new AI code validator with default rules
func NewAICodeValidator() *AICodeValidator {
	validator := &AICodeValidator{
		fileSet: token.NewFileSet(),
		manualReview: []string{
			`func.*\{[^}]*database.*transaction.*[^}]*\}`,     // Complex database transactions
			`func.*\{[^}]*goroutine.*channel.*[^}]*\}`,        // Complex concurrency patterns
			`func.*\{[^}]*reflect\.[^}]*\}`,                   // Reflection usage
			`func.*\{[^}]*unsafe\.[^}]*\}`,                    // Unsafe operations
			`func.*\{[^}]*(?:mutex|sync\.)[^}]*\}`,            // Synchronization primitives
			`func.*\{[^}]*(?:atomic\.)[^}]*\}`,                // Atomic operations
			`func.*\{[^}]*(?:syscall|os\.)[^}]*\}`,            // System calls
			`func.*\{[^}]*(?:cgo|C\.)[^}]*\}`,                 // CGO usage
			`func.*\{[^}]*(?:crypto|hash)[^}]*\}`,             // Cryptographic operations
			`func.*\{[^}]*(?:json\.Unmarshal|xml\.Unmarshal)[^}]*\}`, // Complex unmarshaling
			`func.*\{[^}]*(?:regexp\.MustCompile|regexp\.Compile)[^}]*\}`, // Regex compilation
			`(?s)func.*\{.*(?:for.*\{.*for.*\{|while.*\{.*while.*\{).*\}`, // Nested loops
			`func.*\{[^}]*(?:panic|recover)[^}]*\}`,           // Panic/recover usage
		},
	}
	
	validator.initializeRules()
	return validator
}

// initializeRules sets up the default validation rules
func (v *AICodeValidator) initializeRules() {
	v.rules = []ValidationRule{
		{
			Name:        "missing-error-handling",
			Description: "Function call without error handling",
			Category:    "error-handling",
			Severity:    SeverityCritical,
			Pattern:     nil, // Use AST checker only for this rule
			ASTChecker:  v.checkMissingErrorHandling,
		},
		{
			Name:        "hardcoded-values",
			Description: "Hardcoded values that should be configurable",
			Category:    "maintainability",
			Severity:    SeverityMedium,
			Pattern:     regexp.MustCompile(`"(?:localhost|127\.0\.0\.1|password|secret|key|token|admin|root|test123)"`),
		},
		{
			Name:        "inefficient-db-query",
			Description: "Potentially inefficient database query pattern",
			Category:    "performance",
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`(?i)select\s+\*\s+from.*where.*=.*\$\d+.*order\s+by.*limit`),
		},
		{
			Name:        "sql-injection-risk",
			Description: "Potential SQL injection vulnerability",
			Category:    "security",
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`(?i)(?:query|exec)\s*\(\s*"[^"]*"\s*\+`),
		},
		{
			Name:        "missing-context",
			Description: "HTTP handler without context timeout",
			Category:    "reliability",
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`func.*\(.*http\.ResponseWriter.*\*http\.Request.*\).*\{`),
		},
		{
			Name:        "unbounded-slice",
			Description: "Slice append without capacity check",
			Category:    "performance",
			Severity:    SeverityMedium,
			Pattern:     regexp.MustCompile(`append\([^,]+,.*\)`),
		},
		{
			Name:        "missing-validation",
			Description: "User input without validation",
			Category:    "security",
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`(?:r\.FormValue|r\.PostFormValue|r\.URL\.Query)\([^)]+\)`),
		},
		// Enhanced AI-specific patterns
		{
			Name:        "ai-generated-todo",
			Description: "AI-generated TODO comments that need attention",
			Category:    "maintainability",
			Severity:    SeverityMedium,
			Pattern:     regexp.MustCompile(`(?i)//\s*(?:todo|fixme|hack|xxx).*(?:implement|add|fix|complete)`),
		},
		{
			Name:        "missing-nil-check",
			Description: "Potential nil pointer dereference",
			Category:    "reliability",
			Severity:    SeverityHigh,
			Pattern:     nil, // Use AST checker only for this rule
			ASTChecker:  v.checkNilPointerAccess,
		},
		{
			Name:        "inefficient-string-concat",
			Description: "Inefficient string concatenation in loop",
			Category:    "performance",
			Severity:    SeverityMedium,
			Pattern:     regexp.MustCompile(`(?i)for.*\{[^}]*\w+\s*\+=\s*[^}]*\}`),
		},
		{
			Name:        "missing-transaction",
			Description: "Multiple database operations without transaction",
			Category:    "reliability",
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`(?i)(?:insert|update|delete).*(?:insert|update|delete)`),
		},
		{
			Name:        "hardcoded-timeout",
			Description: "Hardcoded timeout values",
			Category:    "maintainability",
			Severity:    SeverityLow,
			Pattern:     regexp.MustCompile(`time\.(?:Second|Minute|Hour)\s*\*\s*\d+`),
		},
		{
			Name:        "missing-logging",
			Description: "Error handling without logging",
			Category:    "observability",
			Severity:    SeverityMedium,
			Pattern:     regexp.MustCompile(`if\s+err\s*!=\s*nil\s*\{[^}]*return[^}]*\}`),
		},
		{
			Name:        "goroutine-leak",
			Description: "Goroutine without proper cleanup",
			Category:    "reliability",
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`go\s+func\s*\([^)]*\)\s*\{`),
		},
		{
			Name:        "unsafe-type-assertion",
			Description: "Type assertion without ok check",
			Category:    "reliability",
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`\w+\.\([^)]+\)`),
		},
	}
}

// ValidateFile validates a single Go file for AI-generated code issues
func (v *AICodeValidator) ValidateFile(filePath string) ([]ValidationResult, error) {
	var results []ValidationResult
	
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	contentStr := string(content)
	
	// Parse the file for AST analysis
	astFile, err := parser.ParseFile(v.fileSet, filePath, content, parser.ParseComments)
	if err != nil {
		// If parsing fails, still run regex-based checks
		results = append(results, ValidationResult{
			Severity: SeverityHigh,
			Category: "syntax",
			Message:  fmt.Sprintf("Failed to parse Go file: %v", err),
			File:     filePath,
			RuleName: "parse-error",
		})
	}
	
	// Apply regex-based rules
	for _, rule := range v.rules {
		if rule.Pattern != nil {
			matches := rule.Pattern.FindAllStringIndex(contentStr, -1)
			for _, match := range matches {
				line, column := v.getLineColumn(contentStr, match[0])
				snippet := v.getCodeSnippet(contentStr, match[0], match[1])
				
				results = append(results, ValidationResult{
					Severity:    rule.Severity,
					Category:    rule.Category,
					Message:     rule.Description,
					File:        filePath,
					Line:        line,
					Column:      column,
					RuleName:    rule.Name,
					CodeSnippet: snippet,
					Suggestion:  v.getSuggestion(rule.Name),
				})
			}
		}
	}
	
	// Apply AST-based rules
	if astFile != nil {
		for _, rule := range v.rules {
			if rule.ASTChecker != nil {
				astResults := rule.ASTChecker(astFile, v.fileSet)
				results = append(results, astResults...)
			}
		}
	}
	
	// Check for manual review requirements
	if v.requiresManualReview(contentStr) {
		results = append(results, ValidationResult{
			Severity:   SeverityHigh,
			Category:   "manual-review",
			Message:    "Complex AI-generated code requires manual review",
			File:       filePath,
			RuleName:   "ai-manual-review",
			Suggestion: "This code contains complex patterns that should be reviewed by a senior developer",
		})
	}
	
	return results, nil
}

// checkMissingErrorHandling uses AST to find function calls without error handling
func (v *AICodeValidator) checkMissingErrorHandling(file *ast.File, fset *token.FileSet) []ValidationResult {
	var results []ValidationResult
	
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			// Check for assignments that might return errors
			if len(node.Lhs) >= 2 {
				// Check if the last assignment target is named "err"
				if ident, ok := node.Lhs[len(node.Lhs)-1].(*ast.Ident); ok && ident.Name == "err" {
					// Look for the next statement to see if error is handled
					// This is a simplified check - in practice, you'd need more sophisticated analysis
					pos := fset.Position(node.Pos())
					results = append(results, ValidationResult{
						Severity:   SeverityMedium,
						Category:   "error-handling",
						Message:    "Potential unhandled error - verify error handling",
						File:       pos.Filename,
						Line:       pos.Line,
						Column:     pos.Column,
						RuleName:   "potential-unhandled-error",
						Suggestion: "Ensure error is properly handled with if err != nil check",
					})
				}
			}
		}
		return true
	})
	
	return results
}

// checkNilPointerAccess uses AST to find potential nil pointer dereferences
func (v *AICodeValidator) checkNilPointerAccess(file *ast.File, fset *token.FileSet) []ValidationResult {
	var results []ValidationResult
	
	// Only check for specific patterns that are commonly problematic
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			// Only flag potential issues for variables that could be nil
			if ident, ok := node.X.(*ast.Ident); ok {
				// Skip standard library packages and common safe patterns
				if v.isLikelySafeAccess(ident.Name, node.Sel.Name) {
					return true
				}
				
				// Only flag if it looks like a pointer that could be nil
				if v.isLikelyPointerAccess(ident.Name) {
					pos := fset.Position(node.Pos())
					results = append(results, ValidationResult{
						Severity:    SeverityLow, // Reduced severity to avoid noise
						Category:    "reliability",
						Message:     fmt.Sprintf("Consider nil check for %s.%s", ident.Name, node.Sel.Name),
						File:        pos.Filename,
						Line:        pos.Line,
						Column:      pos.Column,
						RuleName:    "potential-nil-access",
						Suggestion:  fmt.Sprintf("Add nil check: if %s != nil { %s.%s }", ident.Name, ident.Name, node.Sel.Name),
					})
				}
			}
		}
		return true
	})
	
	return results
}

// isLikelySafeAccess checks if the access is likely safe (standard library, etc.)
func (v *AICodeValidator) isLikelySafeAccess(varName, fieldName string) bool {
	safePatterns := []string{
		"http", "fmt", "log", "json", "time", "context", "strings", "os", "io",
		"db", "tx", "err", "req", "resp", "w", "r",
	}
	
	for _, pattern := range safePatterns {
		if varName == pattern {
			return true
		}
	}
	
	return false
}

// isLikelyPointerAccess checks if the variable name suggests it could be a pointer
func (v *AICodeValidator) isLikelyPointerAccess(varName string) bool {
	pointerPatterns := []string{
		"user", "article", "config", "client", "conn", "obj", "data", "item",
	}
	
	for _, pattern := range pointerPatterns {
		if strings.Contains(strings.ToLower(varName), pattern) {
			return true
		}
	}
	
	return false
}

// requiresManualReview checks if the code contains patterns requiring manual review
func (v *AICodeValidator) requiresManualReview(content string) bool {
	for _, pattern := range v.manualReview {
		matched, _ := regexp.MatchString(pattern, content)
		if matched {
			return true
		}
	}
	return false
}

// getLineColumn calculates line and column numbers for a byte offset
func (v *AICodeValidator) getLineColumn(content string, offset int) (int, int) {
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

// getCodeSnippet extracts a code snippet around the issue
func (v *AICodeValidator) getCodeSnippet(content string, start, end int) string {
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

// getSuggestion returns a suggestion for fixing the issue
func (v *AICodeValidator) getSuggestion(ruleName string) string {
	suggestions := map[string]string{
		"missing-error-handling":     "Add proper error handling: if err != nil { return err }",
		"hardcoded-values":           "Move hardcoded values to configuration files or environment variables",
		"inefficient-db-query":       "Consider using specific column names instead of SELECT *, add appropriate indexes",
		"sql-injection-risk":         "Use parameterized queries with placeholders ($1, $2, etc.) instead of string concatenation",
		"missing-context":            "Add context with timeout: ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)",
		"unbounded-slice":            "Pre-allocate slice capacity: make([]Type, 0, expectedSize)",
		"missing-validation":         "Validate user input before processing: if err := validateInput(value); err != nil { ... }",
		"ai-generated-todo":          "Replace TODO with actual implementation or create a proper issue tracker item",
		"missing-nil-check":          "Add nil check before accessing: if obj != nil { obj.Field }",
		"inefficient-string-concat":  "Use strings.Builder for efficient string concatenation in loops",
		"missing-transaction":        "Wrap multiple database operations in a transaction: tx, err := db.Begin()",
		"hardcoded-timeout":          "Move timeout values to configuration: timeout := config.GetTimeout()",
		"missing-logging":            "Add logging for error cases: log.Printf(\"Error: %v\", err)",
		"goroutine-leak":             "Add proper cleanup: defer cancel() or use context.Done() channel",
		"unsafe-type-assertion":      "Use safe type assertion: if val, ok := obj.(Type); ok { ... }",
		"potential-unhandled-error":  "Ensure error is properly handled with if err != nil check",
		"potential-nil-access":       "Add nil check before accessing pointer fields",
	}
	
	if suggestion, exists := suggestions[ruleName]; exists {
		return suggestion
	}
	
	return "Review this code pattern for potential issues"
}

// ValidationReport aggregates validation results
type ValidationReport struct {
	FilePath     string             `json:"file_path"`
	Results      []ValidationResult `json:"results"`
	Summary      ValidationSummary  `json:"summary"`
	GeneratedAt  time.Time          `json:"generated_at"`
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalIssues    int `json:"total_issues"`
	CriticalIssues int `json:"critical_issues"`
	HighIssues     int `json:"high_issues"`
	MediumIssues   int `json:"medium_issues"`
	LowIssues      int `json:"low_issues"`
	ManualReview   int `json:"manual_review"`
}

// GenerateReport creates a validation report for a file
func (v *AICodeValidator) GenerateReport(filePath string) (*ValidationReport, error) {
	results, err := v.ValidateFile(filePath)
	if err != nil {
		return nil, err
	}
	
	summary := ValidationSummary{}
	for _, result := range results {
		summary.TotalIssues++
		switch result.Severity {
		case SeverityCritical:
			summary.CriticalIssues++
		case SeverityHigh:
			summary.HighIssues++
		case SeverityMedium:
			summary.MediumIssues++
		case SeverityLow:
			summary.LowIssues++
		}
		
		if result.Category == "manual-review" {
			summary.ManualReview++
		}
	}
	
	return &ValidationReport{
		FilePath:    filePath,
		Results:     results,
		Summary:     summary,
		GeneratedAt: time.Now(),
	}, nil
}

// ShouldBlockDeployment determines if validation results should block deployment
func (r *ValidationReport) ShouldBlockDeployment() bool {
	return r.Summary.CriticalIssues > 0 || r.Summary.HighIssues > 3
}