package maintenance

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"time"
)

// RefactoringEngine identifies and suggests refactoring opportunities
type RefactoringEngine struct{}

// NewRefactoringEngine creates a new refactoring engine
func NewRefactoringEngine() *RefactoringEngine {
	return &RefactoringEngine{}
}

// FindOpportunities finds refactoring opportunities in a test file
func (re *RefactoringEngine) FindOpportunities(node *ast.File, fset *token.FileSet, filePath string) []RefactoringOpportunity {
	var opportunities []RefactoringOpportunity

	// Find duplicate code patterns
	duplicateOpps := re.findDuplicateCode(node, fset, filePath)
	opportunities = append(opportunities, duplicateOpps...)

	// Find complex functions that need simplification
	complexityOpps := re.findComplexFunctions(node, fset, filePath)
	opportunities = append(opportunities, complexityOpps...)

	// Find common setup/teardown patterns
	setupOpps := re.findCommonSetupPatterns(node, fset, filePath)
	opportunities = append(opportunities, setupOpps...)

	// Find naming improvements
	namingOpps := re.findNamingImprovements(node, fset, filePath)
	opportunities = append(opportunities, namingOpps...)

	// Find assertion improvements
	assertionOpps := re.findAssertionImprovements(node, fset, filePath)
	opportunities = append(opportunities, assertionOpps...)

	return opportunities
}

// findDuplicateCode finds duplicate code that can be extracted
func (re *RefactoringEngine) findDuplicateCode(node *ast.File, fset *token.FileSet, filePath string) []RefactoringOpportunity {
	var opportunities []RefactoringOpportunity

	// Collect all function bodies for comparison
	functions := re.collectTestFunctions(node)
	
	// Compare functions for similarity
	for i, fn1 := range functions {
		for j, fn2 := range functions {
			if i >= j {
				continue
			}

			similarity := re.calculateFunctionSimilarity(fn1, fn2)
			if similarity > 0.7 { // High similarity threshold
				opportunity := RefactoringOpportunity{
					ID:              fmt.Sprintf("duplicate_%d_%d", i, j),
					Type:            RefactoringRemoveDuplication,
					TestIDs:         []string{fn1.Name.Name, fn2.Name.Name},
					Description:     fmt.Sprintf("Functions %s and %s have high similarity (%.1f%%)", fn1.Name.Name, fn2.Name.Name, similarity*100),
					Benefits:        []string{"Reduce code duplication", "Improve maintainability", "Reduce test suite size"},
					EstimatedEffort: "Medium",
					Priority:        PriorityMedium,
					AutoApplicable:  false,
					CreatedAt:       time.Now(),
				}
				opportunities = append(opportunities, opportunity)
			}
		}
	}

	return opportunities
}

// findComplexFunctions finds functions that are too complex
func (re *RefactoringEngine) findComplexFunctions(node *ast.File, fset *token.FileSet, filePath string) []RefactoringOpportunity {
	var opportunities []RefactoringOpportunity

	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name != nil && strings.HasPrefix(fn.Name.Name, "Test") {
			complexity := re.calculateComplexity(fn)
			lineCount := re.countStatements(fn)

			if complexity > 10 || lineCount > 50 {
				opportunity := RefactoringOpportunity{
					ID:              fmt.Sprintf("complex_%s", fn.Name.Name),
					Type:            RefactoringReduceComplexity,
					TestIDs:         []string{fn.Name.Name},
					Description:     fmt.Sprintf("Function %s is too complex (complexity: %d, statements: %d)", fn.Name.Name, complexity, lineCount),
					Benefits:        []string{"Improve readability", "Easier debugging", "Better maintainability"},
					EstimatedEffort: "High",
					Priority:        PriorityHigh,
					AutoApplicable:  false,
					CreatedAt:       time.Now(),
				}
				opportunities = append(opportunities, opportunity)
			}
		}
		return true
	})

	return opportunities
}

// findCommonSetupPatterns finds common setup/teardown patterns that can be extracted
func (re *RefactoringEngine) findCommonSetupPatterns(node *ast.File, fset *token.FileSet, filePath string) []RefactoringOpportunity {
	var opportunities []RefactoringOpportunity

	// Collect setup patterns from all test functions
	setupPatterns := make(map[string][]string)
	
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name != nil && strings.HasPrefix(fn.Name.Name, "Test") {
			setup := re.extractSetupPattern(fn)
			if setup != "" {
				setupPatterns[setup] = append(setupPatterns[setup], fn.Name.Name)
			}
		}
		return true
	})

	// Find patterns used by multiple tests
	for pattern, testNames := range setupPatterns {
		if len(testNames) >= 3 { // Used by 3 or more tests
			opportunity := RefactoringOpportunity{
				ID:              fmt.Sprintf("setup_%s", strings.ReplaceAll(pattern, " ", "_")),
				Type:            RefactoringOptimizeSetup,
				TestIDs:         testNames,
				Description:     fmt.Sprintf("Common setup pattern found in %d tests: %s", len(testNames), pattern),
				Benefits:        []string{"Reduce code duplication", "Centralize setup logic", "Improve consistency"},
				EstimatedEffort: "Medium",
				Priority:        PriorityMedium,
				AutoApplicable:  true,
				CreatedAt:       time.Now(),
			}
			opportunities = append(opportunities, opportunity)
		}
	}

	return opportunities
}

// findNamingImprovements finds naming improvements
func (re *RefactoringEngine) findNamingImprovements(node *ast.File, fset *token.FileSet, filePath string) []RefactoringOpportunity {
	var opportunities []RefactoringOpportunity

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name != nil && strings.HasPrefix(x.Name.Name, "Test") {
				if !re.isGoodTestName(x.Name.Name) {
					opportunity := RefactoringOpportunity{
						ID:              fmt.Sprintf("naming_%s", x.Name.Name),
						Type:            RefactoringImproveNaming,
						TestIDs:         []string{x.Name.Name},
						Description:     fmt.Sprintf("Test function %s has unclear naming", x.Name.Name),
						Benefits:        []string{"Improve test readability", "Better documentation", "Clearer intent"},
						EstimatedEffort: "Low",
						Priority:        PriorityLow,
						AutoApplicable:  false,
						CreatedAt:       time.Now(),
					}
					opportunities = append(opportunities, opportunity)
				}
			}
		case *ast.GenDecl:
			for _, spec := range x.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						if !re.isGoodVariableName(name.Name) {
							opportunity := RefactoringOpportunity{
								ID:              fmt.Sprintf("var_naming_%s", name.Name),
								Type:            RefactoringImproveNaming,
								TestIDs:         []string{name.Name},
								Description:     fmt.Sprintf("Variable %s has unclear naming", name.Name),
								Benefits:        []string{"Improve code readability", "Better self-documentation"},
								EstimatedEffort: "Low",
								Priority:        PriorityLow,
								AutoApplicable:  false,
								CreatedAt:       time.Now(),
							}
							opportunities = append(opportunities, opportunity)
						}
					}
				}
			}
		}
		return true
	})

	return opportunities
}

// findAssertionImprovements finds assertion improvements
func (re *RefactoringEngine) findAssertionImprovements(node *ast.File, fset *token.FileSet, filePath string) []RefactoringOpportunity {
	var opportunities []RefactoringOpportunity

	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name != nil && strings.HasPrefix(fn.Name.Name, "Test") {
			improvements := re.analyzeAssertions(fn)
			for _, improvement := range improvements {
				opportunity := RefactoringOpportunity{
					ID:              fmt.Sprintf("assertion_%s_%d", fn.Name.Name, len(opportunities)),
					Type:            RefactoringSimplifyLogic,
					TestIDs:         []string{fn.Name.Name},
					Description:     improvement,
					Benefits:        []string{"Better error messages", "Clearer test intent", "Improved debugging"},
					EstimatedEffort: "Low",
					Priority:        PriorityMedium,
					AutoApplicable:  true,
					CreatedAt:       time.Now(),
				}
				opportunities = append(opportunities, opportunity)
			}
		}
		return true
	})

	return opportunities
}

// collectTestFunctions collects all test functions from the AST
func (re *RefactoringEngine) collectTestFunctions(node *ast.File) []*ast.FuncDecl {
	var functions []*ast.FuncDecl

	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name != nil && strings.HasPrefix(fn.Name.Name, "Test") {
			functions = append(functions, fn)
		}
		return true
	})

	return functions
}

// calculateFunctionSimilarity calculates similarity between two functions
func (re *RefactoringEngine) calculateFunctionSimilarity(fn1, fn2 *ast.FuncDecl) float64 {
	// Simplified similarity calculation based on structure
	
	// Compare parameter counts
	paramSimilarity := 0.0
	if fn1.Type.Params != nil && fn2.Type.Params != nil {
		if len(fn1.Type.Params.List) == len(fn2.Type.Params.List) {
			paramSimilarity = 1.0
		}
	}

	// Compare statement counts
	stmtCount1 := re.countStatements(fn1)
	stmtCount2 := re.countStatements(fn2)
	stmtSimilarity := 1.0 - float64(abs(stmtCount1-stmtCount2))/float64(max(stmtCount1, stmtCount2))

	// Compare complexity
	complexity1 := re.calculateComplexity(fn1)
	complexity2 := re.calculateComplexity(fn2)
	complexitySimilarity := 1.0 - float64(abs(complexity1-complexity2))/float64(max(complexity1, complexity2))

	// Weighted average
	return (paramSimilarity*0.2 + stmtSimilarity*0.4 + complexitySimilarity*0.4)
}

// calculateComplexity calculates cyclomatic complexity
func (re *RefactoringEngine) calculateComplexity(fn *ast.FuncDecl) int {
	complexity := 1

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

// countStatements counts the number of statements in a function
func (re *RefactoringEngine) countStatements(fn *ast.FuncDecl) int {
	count := 0

	if fn.Body != nil {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if _, ok := n.(ast.Stmt); ok {
				count++
			}
			return true
		})
	}

	return count
}

// extractSetupPattern extracts common setup patterns from a function
func (re *RefactoringEngine) extractSetupPattern(fn *ast.FuncDecl) string {
	var patterns []string

	if fn.Body != nil && len(fn.Body.List) > 0 {
		// Look at the first few statements for setup patterns
		for i, stmt := range fn.Body.List {
			if i >= 3 { // Only check first 3 statements
				break
			}

			pattern := re.identifyStatementPattern(stmt)
			if pattern != "" {
				patterns = append(patterns, pattern)
			}
		}
	}

	if len(patterns) > 0 {
		return strings.Join(patterns, " + ")
	}

	return ""
}

// identifyStatementPattern identifies the pattern of a statement
func (re *RefactoringEngine) identifyStatementPattern(stmt ast.Stmt) string {
	switch x := stmt.(type) {
	case *ast.AssignStmt:
		if len(x.Lhs) > 0 && len(x.Rhs) > 0 {
			// Look for common assignment patterns
			if callExpr, ok := x.Rhs[0].(*ast.CallExpr); ok {
				if ident, ok := callExpr.Fun.(*ast.Ident); ok {
					return fmt.Sprintf("assign_%s", ident.Name)
				}
			}
		}
	case *ast.ExprStmt:
		if callExpr, ok := x.X.(*ast.CallExpr); ok {
			if ident, ok := callExpr.Fun.(*ast.Ident); ok {
				return fmt.Sprintf("call_%s", ident.Name)
			}
		}
	}

	return ""
}

// isGoodTestName checks if a test name is descriptive
func (re *RefactoringEngine) isGoodTestName(name string) bool {
	// Check for descriptive test names
	if len(name) < 10 {
		return false
	}

	// Should describe what is being tested
	badPatterns := []string{"Test1", "Test2", "TestFunc", "TestMethod"}
	for _, pattern := range badPatterns {
		if strings.Contains(name, pattern) {
			return false
		}
	}

	// Should follow TestXxxYyy pattern
	if !strings.HasPrefix(name, "Test") {
		return false
	}

	// Should have meaningful words
	words := re.splitCamelCase(name[4:]) // Remove "Test" prefix
	if len(words) < 2 {
		return false
	}

	return true
}

// isGoodVariableName checks if a variable name is descriptive
func (re *RefactoringEngine) isGoodVariableName(name string) bool {
	// Check for meaningful variable names
	if len(name) < 2 {
		return false
	}

	badNames := []string{"a", "b", "c", "x", "y", "z", "temp", "tmp", "data", "val", "var"}
	for _, badName := range badNames {
		if name == badName {
			return false
		}
	}

	return true
}

// analyzeAssertions analyzes assertions in a test function
func (re *RefactoringEngine) analyzeAssertions(fn *ast.FuncDecl) []string {
	var improvements []string

	ast.Inspect(fn, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			// Look for assertion improvements
			if improvement := re.suggestAssertionImprovement(callExpr); improvement != "" {
				improvements = append(improvements, improvement)
			}
		}
		return true
	})

	return improvements
}

// suggestAssertionImprovement suggests improvements for assertions
func (re *RefactoringEngine) suggestAssertionImprovement(callExpr *ast.CallExpr) string {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		switch selExpr.Sel.Name {
		case "True":
			if len(callExpr.Args) >= 2 {
				// Suggest more specific assertions
				return "Consider using more specific assertion instead of assert.True"
			}
		case "False":
			if len(callExpr.Args) >= 2 {
				return "Consider using more specific assertion instead of assert.False"
			}
		case "Equal":
			if len(callExpr.Args) >= 3 {
				// Check argument order
				return "Verify assertion argument order (expected, actual)"
			}
		}
	}

	return ""
}

// splitCamelCase splits a camelCase string into words
func (re *RefactoringEngine) splitCamelCase(s string) []string {
	var words []string
	var currentWord strings.Builder

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}
		currentWord.WriteRune(r)
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// ApplyRefactoring applies a refactoring opportunity
func (re *RefactoringEngine) ApplyRefactoring(opportunity RefactoringOpportunity, filePath string) error {
	if !opportunity.AutoApplicable {
		return fmt.Errorf("refactoring %s is not auto-applicable", opportunity.ID)
	}

	switch opportunity.Type {
	case RefactoringOptimizeSetup:
		return re.applySetupOptimization(opportunity, filePath)
	case RefactoringSimplifyLogic:
		return re.applyLogicSimplification(opportunity, filePath)
	default:
		return fmt.Errorf("unsupported refactoring type: %s", opportunity.Type)
	}
}

// applySetupOptimization applies setup optimization refactoring
func (re *RefactoringEngine) applySetupOptimization(opportunity RefactoringOpportunity, filePath string) error {
	// This would involve:
	// 1. Extracting common setup code
	// 2. Creating a helper function
	// 3. Updating all affected tests
	// For now, this is a placeholder
	return fmt.Errorf("setup optimization not yet implemented")
}

// applyLogicSimplification applies logic simplification refactoring
func (re *RefactoringEngine) applyLogicSimplification(opportunity RefactoringOpportunity, filePath string) error {
	// This would involve:
	// 1. Analyzing the specific logic issue
	// 2. Applying the appropriate transformation
	// 3. Updating the file
	// For now, this is a placeholder
	return fmt.Errorf("logic simplification not yet implemented")
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}