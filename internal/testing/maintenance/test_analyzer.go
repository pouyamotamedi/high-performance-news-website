package maintenance

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TestAnalyzer analyzes test files and extracts metadata
type TestAnalyzer struct {
	fileSet *token.FileSet
}

// NewTestAnalyzer creates a new test analyzer
func NewTestAnalyzer() *TestAnalyzer {
	return &TestAnalyzer{
		fileSet: token.NewFileSet(),
	}
}

// AnalyzeTestFile analyzes a single test file and extracts test metadata
func (ta *TestAnalyzer) AnalyzeTestFile(filePath string) ([]*TestMetadata, error) {
	// Parse the Go file
	node, err := parser.ParseFile(ta.fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	var tests []*TestMetadata

	// Extract file-level information
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Find all test functions
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if ta.isTestFunction(x) {
				test := ta.extractTestMetadata(x, filePath, fileInfo.ModTime())
				tests = append(tests, test)
			}
		}
		return true
	})

	// Analyze file-level dependencies
	dependencies := ta.extractDependencies(node)

	// Update all tests with file-level information
	for _, test := range tests {
		test.Dependencies = append(test.Dependencies, dependencies...)
		test.Tags = ta.extractTags(filePath)
		test.Annotations = ta.extractAnnotations(filePath)
	}

	return tests, nil
}

// isTestFunction checks if a function is a test function
func (ta *TestAnalyzer) isTestFunction(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}

	name := fn.Name.Name
	return strings.HasPrefix(name, "Test") || 
		   strings.HasPrefix(name, "Benchmark") || 
		   strings.HasPrefix(name, "Example")
}

// extractTestMetadata extracts metadata from a test function
func (ta *TestAnalyzer) extractTestMetadata(fn *ast.FuncDecl, filePath string, lastModified time.Time) *TestMetadata {
	testName := fn.Name.Name
	testID := fmt.Sprintf("%s::%s", filepath.Base(filePath), testName)

	test := &TestMetadata{
		ID:           testID,
		FilePath:     filePath,
		TestName:     testName,
		TestType:     ta.determineTestType(testName),
		LastModified: lastModified,
		Status:       StatusActive,
		Complexity:   ta.calculateComplexity(fn),
		Dependencies: []string{},
		Tags:         []string{},
		Annotations:  make(map[string]string),
	}

	// Extract test-specific information from comments
	if fn.Doc != nil {
		test.Annotations = ta.parseTestComments(fn.Doc.Text())
	}

	return test
}

// determineTestType determines the type of test based on naming and content
func (ta *TestAnalyzer) determineTestType(testName string) string {
	switch {
	case strings.HasPrefix(testName, "TestUnit"):
		return "unit"
	case strings.HasPrefix(testName, "TestIntegration"):
		return "integration"
	case strings.HasPrefix(testName, "TestE2E") || strings.HasPrefix(testName, "TestEndToEnd"):
		return "e2e"
	case strings.HasPrefix(testName, "TestPerformance") || strings.HasPrefix(testName, "TestLoad"):
		return "performance"
	case strings.HasPrefix(testName, "TestSecurity"):
		return "security"
	case strings.HasPrefix(testName, "Benchmark"):
		return "benchmark"
	case strings.HasPrefix(testName, "Example"):
		return "example"
	default:
		return "unit" // Default to unit test
	}
}

// calculateComplexity calculates the cyclomatic complexity of a test function
func (ta *TestAnalyzer) calculateComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // Base complexity

	ast.Inspect(fn, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
}

// extractDependencies extracts import dependencies from the file
func (ta *TestAnalyzer) extractDependencies(node *ast.File) []string {
	var dependencies []string

	for _, imp := range node.Imports {
		if imp.Path != nil {
			// Remove quotes from import path
			path := strings.Trim(imp.Path.Value, `"`)
			dependencies = append(dependencies, path)
		}
	}

	return dependencies
}

// extractTags extracts build tags and other tags from the file
func (ta *TestAnalyzer) extractTags(filePath string) []string {
	var tags []string

	file, err := os.Open(filePath)
	if err != nil {
		return tags
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Only check the first few lines for build tags
	for scanner.Scan() && lineNum < 10 {
		line := strings.TrimSpace(scanner.Text())
		lineNum++

		// Check for build tags
		if strings.HasPrefix(line, "//go:build") || strings.HasPrefix(line, "// +build") {
			buildTags := ta.parseBuildTags(line)
			tags = append(tags, buildTags...)
		}

		// Check for custom test tags in comments
		if strings.Contains(line, "@tag:") {
			customTags := ta.parseCustomTags(line)
			tags = append(tags, customTags...)
		}

		// Stop at first non-comment, non-empty line
		if !strings.HasPrefix(line, "//") && !strings.HasPrefix(line, "/*") && line != "" {
			break
		}
	}

	// Add tags based on file path
	pathTags := ta.extractPathTags(filePath)
	tags = append(tags, pathTags...)

	return tags
}

// parseBuildTags parses build tags from a build constraint line
func (ta *TestAnalyzer) parseBuildTags(line string) []string {
	var tags []string

	// Remove comment prefix and build directive
	line = strings.TrimPrefix(line, "//go:build")
	line = strings.TrimPrefix(line, "// +build")
	line = strings.TrimSpace(line)

	// Simple parsing - split by spaces and logical operators
	re := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)
	matches := re.FindAllString(line, -1)

	for _, match := range matches {
		// Filter out logical operators
		if match != "AND" && match != "OR" && match != "NOT" {
			tags = append(tags, "build:"+match)
		}
	}

	return tags
}

// parseCustomTags parses custom tags from comments
func (ta *TestAnalyzer) parseCustomTags(line string) []string {
	var tags []string

	re := regexp.MustCompile(`@tag:([a-zA-Z0-9_-]+)`)
	matches := re.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		if len(match) > 1 {
			tags = append(tags, match[1])
		}
	}

	return tags
}

// extractPathTags extracts tags based on file path patterns
func (ta *TestAnalyzer) extractPathTags(filePath string) []string {
	var tags []string

	// Normalize path separators
	normalizedPath := filepath.ToSlash(filePath)

	// Add tags based on directory structure
	if strings.Contains(normalizedPath, "/integration/") {
		tags = append(tags, "integration")
	}
	if strings.Contains(normalizedPath, "/unit/") {
		tags = append(tags, "unit")
	}
	if strings.Contains(normalizedPath, "/e2e/") {
		tags = append(tags, "e2e")
	}
	if strings.Contains(normalizedPath, "/performance/") {
		tags = append(tags, "performance")
	}
	if strings.Contains(normalizedPath, "/security/") {
		tags = append(tags, "security")
	}

	// Add module/package tags
	parts := strings.Split(normalizedPath, "/")
	for i, part := range parts {
		if part == "internal" && i+1 < len(parts) {
			tags = append(tags, "module:"+parts[i+1])
		}
	}

	return tags
}

// extractAnnotations extracts annotations from the file
func (ta *TestAnalyzer) extractAnnotations(filePath string) map[string]string {
	annotations := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		return annotations
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for annotation patterns in comments
		if strings.HasPrefix(line, "//") {
			ta.parseAnnotationLine(line, annotations)
		}
	}

	return annotations
}

// parseAnnotationLine parses annotations from a comment line
func (ta *TestAnalyzer) parseAnnotationLine(line string, annotations map[string]string) {
	// Remove comment prefix
	content := strings.TrimPrefix(line, "//")
	content = strings.TrimSpace(content)

	// Parse different annotation patterns
	patterns := []struct {
		regex *regexp.Regexp
		key   string
	}{
		{regexp.MustCompile(`@timeout:(\d+[smh]?)`), "timeout"},
		{regexp.MustCompile(`@parallel:(\w+)`), "parallel"},
		{regexp.MustCompile(`@requires:(.+)`), "requires"},
		{regexp.MustCompile(`@author:(.+)`), "author"},
		{regexp.MustCompile(`@since:(.+)`), "since"},
		{regexp.MustCompile(`@category:(.+)`), "category"},
		{regexp.MustCompile(`@priority:(\w+)`), "priority"},
		{regexp.MustCompile(`@flaky:(\w+)`), "flaky"},
	}

	for _, pattern := range patterns {
		matches := pattern.regex.FindStringSubmatch(content)
		if len(matches) > 1 {
			annotations[pattern.key] = strings.TrimSpace(matches[1])
		}
	}
}

// parseTestComments parses test-specific information from function comments
func (ta *TestAnalyzer) parseTestComments(docText string) map[string]string {
	annotations := make(map[string]string)

	lines := strings.Split(docText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		ta.parseAnnotationLine("// "+line, annotations)
	}

	return annotations
}

// AnalyzeTestExecution analyzes test execution data and updates metadata
func (ta *TestAnalyzer) AnalyzeTestExecution(testID string, executionData *TestExecutionData) *TestMetadata {
	// This would typically load existing metadata and update it
	// For now, create a basic metadata structure
	test := &TestMetadata{
		ID:             testID,
		LastExecuted:   executionData.Timestamp,
		ExecutionCount: executionData.ExecutionCount,
		FailureRate:    executionData.FailureRate,
		AverageRuntime: executionData.AverageRuntime,
		CodeCoverage:   executionData.Coverage,
	}

	return test
}

// TestExecutionData represents execution data for a test
type TestExecutionData struct {
	Timestamp      time.Time     `json:"timestamp"`
	ExecutionCount int           `json:"execution_count"`
	FailureRate    float64       `json:"failure_rate"`
	AverageRuntime time.Duration `json:"average_runtime"`
	Coverage       float64       `json:"coverage"`
	Success        bool          `json:"success"`
	Error          string        `json:"error,omitempty"`
}

// AnalyzeCodeCoverage analyzes code coverage for tests
func (ta *TestAnalyzer) AnalyzeCodeCoverage(coverageFile string) (map[string]float64, error) {
	coverage := make(map[string]float64)

	file, err := os.Open(coverageFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open coverage file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip mode line
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		// Parse coverage line: filename:startLine.startCol,endLine.endCol numStmt count
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			filePart := parts[0]
			stmtCount, _ := strconv.Atoi(parts[1])
			execCount, _ := strconv.Atoi(parts[2])

			// Extract filename
			colonIndex := strings.LastIndex(filePart, ":")
			if colonIndex > 0 {
				filename := filePart[:colonIndex]
				
				if stmtCount > 0 {
					if execCount > 0 {
						coverage[filename] = 1.0 // Covered
					} else {
						coverage[filename] = 0.0 // Not covered
					}
				}
			}
		}
	}

	return coverage, scanner.Err()
}

// DetectTestPatterns detects common patterns in test code
func (ta *TestAnalyzer) DetectTestPatterns(filePath string) ([]TestPattern, error) {
	var patterns []TestPattern

	node, err := parser.ParseFile(ta.fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Detect various patterns
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if ta.isTestFunction(x) {
				// Detect setup/teardown patterns
				if ta.hasSetupTeardown(x) {
					patterns = append(patterns, TestPattern{
						Type:        "setup_teardown",
						Description: "Test uses setup/teardown pattern",
						Location:    ta.getNodeLocation(x),
					})
				}

				// Detect table-driven tests
				if ta.isTableDriven(x) {
					patterns = append(patterns, TestPattern{
						Type:        "table_driven",
						Description: "Table-driven test pattern",
						Location:    ta.getNodeLocation(x),
					})
				}

				// Detect parallel tests
				if ta.isParallelTest(x) {
					patterns = append(patterns, TestPattern{
						Type:        "parallel",
						Description: "Parallel test execution",
						Location:    ta.getNodeLocation(x),
					})
				}
			}
		}
		return true
	})

	return patterns, nil
}

// TestPattern represents a detected pattern in test code
type TestPattern struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Location    string `json:"location"`
}

// hasSetupTeardown checks if a test function has setup/teardown pattern
func (ta *TestAnalyzer) hasSetupTeardown(fn *ast.FuncDecl) bool {
	hasDefer := false
	
	ast.Inspect(fn, func(n ast.Node) bool {
		if _, ok := n.(*ast.DeferStmt); ok {
			hasDefer = true
			return false
		}
		return true
	})

	return hasDefer
}

// isTableDriven checks if a test uses table-driven pattern
func (ta *TestAnalyzer) isTableDriven(fn *ast.FuncDecl) bool {
	hasTestCases := false
	hasLoop := false

	ast.Inspect(fn, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			// Look for test cases variable declaration
			if x.Tok == token.VAR {
				for _, spec := range x.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range valueSpec.Names {
							if strings.Contains(strings.ToLower(name.Name), "test") ||
							   strings.Contains(strings.ToLower(name.Name), "case") {
								hasTestCases = true
							}
						}
					}
				}
			}
		case *ast.RangeStmt:
			hasLoop = true
		}
		return true
	})

	return hasTestCases && hasLoop
}

// isParallelTest checks if a test is marked for parallel execution
func (ta *TestAnalyzer) isParallelTest(fn *ast.FuncDecl) bool {
	hasParallel := false

	ast.Inspect(fn, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				if selExpr.Sel.Name == "Parallel" {
					hasParallel = true
					return false
				}
			}
		}
		return true
	})

	return hasParallel
}

// getNodeLocation gets the location of an AST node
func (ta *TestAnalyzer) getNodeLocation(node ast.Node) string {
	pos := ta.fileSet.Position(node.Pos())
	return fmt.Sprintf("%s:%d:%d", pos.Filename, pos.Line, pos.Column)
}