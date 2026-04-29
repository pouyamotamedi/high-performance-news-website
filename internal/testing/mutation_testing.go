package testing

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// MutationTester implements mutation testing for Go code
type MutationTester struct {
	config        *MutationConfig
	fileSet       *token.FileSet
	mutators      []Mutator
	testRunner    MutationTestRunner
	reporter      *MutationReporter
	workDir       string
}

// MutationConfig defines configuration for mutation testing
type MutationConfig struct {
	TargetPackages     []string          `json:"target_packages"`
	TestPackages       []string          `json:"test_packages"`
	ExcludePatterns    []string          `json:"exclude_patterns"`
	MutationTypes      []string          `json:"mutation_types"`
	MinMutationScore   float64           `json:"min_mutation_score"`
	Timeout            time.Duration     `json:"timeout"`
	MaxConcurrency     int               `json:"max_concurrency"`
	CriticalFunctions  []string          `json:"critical_functions"`
	SecurityFunctions  []string          `json:"security_functions"`
	PerformanceFunctions []string        `json:"performance_functions"`
}

// MutationResult represents the result of a single mutation test
type MutationResult struct {
	ID              string        `json:"id"`
	FilePath        string        `json:"file_path"`
	Function        string        `json:"function"`
	MutationType    string        `json:"mutation_type"`
	LineNumber      int           `json:"line_number"`
	OriginalCode    string        `json:"original_code"`
	MutatedCode     string        `json:"mutated_code"`
	TestsPassed     bool          `json:"tests_passed"`
	TestOutput      string        `json:"test_output"`
	ExecutionTime   time.Duration `json:"execution_time"`
	Killed          bool          `json:"killed"`
	Category        string        `json:"category"` // business_logic, security, performance
}

// MutationReport contains comprehensive mutation testing results
type MutationReport struct {
	Timestamp         time.Time         `json:"timestamp"`
	TotalMutations    int               `json:"total_mutations"`
	KilledMutations   int               `json:"killed_mutations"`
	SurvivedMutations int               `json:"survived_mutations"`
	MutationScore     float64           `json:"mutation_score"`
	Results           []MutationResult  `json:"results"`
	CategoryScores    map[string]float64 `json:"category_scores"`
	WeakTests         []WeakTestReport  `json:"weak_tests"`
	Recommendations   []string          `json:"recommendations"`
}

// WeakTestReport identifies tests that failed to catch mutations
type WeakTestReport struct {
	TestFunction    string   `json:"test_function"`
	MissedMutations []string `json:"missed_mutations"`
	Severity        string   `json:"severity"`
	Suggestions     []string `json:"suggestions"`
}

// Mutator interface for different types of mutations
type Mutator interface {
	Name() string
	CanMutate(node ast.Node) bool
	Mutate(node ast.Node) (ast.Node, error)
	Category() string
}

// MutationTestRunner interface for executing tests during mutation testing
type MutationTestRunner interface {
	RunTests(packagePath string, timeout time.Duration) (bool, string, error)
}

// NewMutationTester creates a new mutation tester
func NewMutationTester(config *MutationConfig) *MutationTester {
	return &MutationTester{
		config:     config,
		fileSet:    token.NewFileSet(),
		mutators:   createMutators(),
		testRunner: &GoTestRunner{},
		reporter:   NewMutationReporter(),
		workDir:    createTempWorkDir(),
	}
}

// RunMutationTesting executes comprehensive mutation testing
func (mt *MutationTester) RunMutationTesting() (*MutationReport, error) {
	log.Println("Starting mutation testing...")
	
	report := &MutationReport{
		Timestamp:      time.Now(),
		CategoryScores: make(map[string]float64),
		Results:        []MutationResult{},
		WeakTests:      []WeakTestReport{},
	}
	
	// Discover target files
	targetFiles, err := mt.discoverTargetFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to discover target files: %w", err)
	}
	
	log.Printf("Found %d target files for mutation testing", len(targetFiles))
	
	// Generate mutations for each file
	for _, filePath := range targetFiles {
		mutations, err := mt.generateMutationsForFile(filePath)
		if err != nil {
			log.Printf("Error generating mutations for %s: %v", filePath, err)
			continue
		}
		
		// Execute mutations
		for _, mutation := range mutations {
			result := mt.executeMutation(mutation)
			report.Results = append(report.Results, result)
		}
	}
	
	// Calculate scores and analyze results
	mt.calculateMutationScores(report)
	mt.analyzeWeakTests(report)
	mt.generateRecommendations(report)
	
	return report, nil
}

// discoverTargetFiles finds all Go files in target packages
func (mt *MutationTester) discoverTargetFiles() ([]string, error) {
	var files []string
	
	for _, pkg := range mt.config.TargetPackages {
		err := filepath.WalkDir(pkg, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			
			if !d.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
				// Check exclude patterns
				excluded := false
				for _, pattern := range mt.config.ExcludePatterns {
					if matched, _ := regexp.MatchString(pattern, path); matched {
						excluded = true
						break
					}
				}
				
				if !excluded {
					files = append(files, path)
				}
			}
			
			return nil
		})
		
		if err != nil {
			return nil, err
		}
	}
	
	return files, nil
}

// generateMutationsForFile creates mutations for a specific file
func (mt *MutationTester) generateMutationsForFile(filePath string) ([]Mutation, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	// Parse the file
	file, err := parser.ParseFile(mt.fileSet, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	
	var mutations []Mutation
	
	// Walk the AST and generate mutations
	ast.Inspect(file, func(n ast.Node) bool {
		for _, mutator := range mt.mutators {
			if mutator.CanMutate(n) {
				mutation := Mutation{
					FilePath:     filePath,
					Mutator:      mutator,
					Node:         n,
					Position:     mt.fileSet.Position(n.Pos()),
					Category:     mutator.Category(),
				}
				
				// Determine if this is a critical function
				if mt.isCriticalFunction(n, filePath) {
					mutation.Priority = "high"
				} else {
					mutation.Priority = "normal"
				}
				
				mutations = append(mutations, mutation)
			}
		}
		return true
	})
	
	return mutations, nil
}

// executeMutation runs a single mutation test
func (mt *MutationTester) executeMutation(mutation Mutation) MutationResult {
	start := time.Now()
	
	result := MutationResult{
		ID:           generateMutationID(mutation),
		FilePath:     mutation.FilePath,
		Function:     mt.extractFunctionName(mutation.Node),
		MutationType: mutation.Mutator.Name(),
		LineNumber:   mutation.Position.Line,
		Category:     mutation.Category,
	}
	
	// Create mutated version of the file
	mutatedCode, err := mt.createMutatedFile(mutation)
	if err != nil {
		result.TestOutput = fmt.Sprintf("Failed to create mutation: %v", err)
		result.ExecutionTime = time.Since(start)
		return result
	}
	
	result.OriginalCode = mt.extractOriginalCode(mutation)
	result.MutatedCode = mutatedCode
	
	// Write mutated file to work directory
	workFilePath := filepath.Join(mt.workDir, filepath.Base(mutation.FilePath))
	if err := os.WriteFile(workFilePath, []byte(mutatedCode), 0644); err != nil {
		result.TestOutput = fmt.Sprintf("Failed to write mutated file: %v", err)
		result.ExecutionTime = time.Since(start)
		return result
	}
	
	// Run tests against mutated code
	packagePath := filepath.Dir(mutation.FilePath)
	testsPassed, output, err := mt.testRunner.RunTests(packagePath, mt.config.Timeout)
	
	result.TestsPassed = testsPassed
	result.TestOutput = output
	result.Killed = !testsPassed // Mutation is "killed" if tests fail
	result.ExecutionTime = time.Since(start)
	
	if err != nil {
		result.TestOutput += fmt.Sprintf("\nTest execution error: %v", err)
	}
	
	return result
}

// isCriticalFunction determines if a function is marked as critical
func (mt *MutationTester) isCriticalFunction(node ast.Node, filePath string) bool {
	funcName := mt.extractFunctionName(node)
	if funcName == "" {
		return false
	}
	
	// Check against critical function lists
	for _, criticalFunc := range mt.config.CriticalFunctions {
		if strings.Contains(funcName, criticalFunc) {
			return true
		}
	}
	
	for _, securityFunc := range mt.config.SecurityFunctions {
		if strings.Contains(funcName, securityFunc) {
			return true
		}
	}
	
	for _, perfFunc := range mt.config.PerformanceFunctions {
		if strings.Contains(funcName, perfFunc) {
			return true
		}
	}
	
	// Check file path for critical packages
	if strings.Contains(filePath, "/auth/") || strings.Contains(filePath, "/security/") {
		return true
	}
	
	return false
}

// Mutation represents a single code mutation
type Mutation struct {
	FilePath string
	Mutator  Mutator
	Node     ast.Node
	Position token.Position
	Category string
	Priority string
}

// createMutatedFile generates the mutated version of a file
func (mt *MutationTester) createMutatedFile(mutation Mutation) (string, error) {
	src, err := os.ReadFile(mutation.FilePath)
	if err != nil {
		return "", err
	}
	
	// Parse the file
	file, err := parser.ParseFile(mt.fileSet, mutation.FilePath, src, parser.ParseComments)
	if err != nil {
		return "", err
	}
	
	// Apply mutation
	_, err = mutation.Mutator.Mutate(mutation.Node)
	if err != nil {
		return "", err
	}
	
	// Replace the node in the AST
	// This is a simplified approach - in practice, you'd need more sophisticated AST manipulation
	// For now, we'll return the original source with a comment indicating mutation
	
	// Format the mutated AST back to source code
	var buf bytes.Buffer
	if err := format.Node(&buf, mt.fileSet, file); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

// extractFunctionName extracts the function name from an AST node
func (mt *MutationTester) extractFunctionName(node ast.Node) string {
	switch n := node.(type) {
	case *ast.FuncDecl:
		if n.Name != nil {
			return n.Name.Name
		}
	case *ast.CallExpr:
		if ident, ok := n.Fun.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// extractOriginalCode extracts the original code snippet
func (mt *MutationTester) extractOriginalCode(mutation Mutation) string {
	src, err := os.ReadFile(mutation.FilePath)
	if err != nil {
		return ""
	}
	
	lines := strings.Split(string(src), "\n")
	if mutation.Position.Line > 0 && mutation.Position.Line <= len(lines) {
		return strings.TrimSpace(lines[mutation.Position.Line-1])
	}
	
	return ""
}

// calculateMutationScores computes mutation scores by category
func (mt *MutationTester) calculateMutationScores(report *MutationReport) {
	report.TotalMutations = len(report.Results)
	
	categoryStats := make(map[string]struct {
		total  int
		killed int
	})
	
	for _, result := range report.Results {
		if result.Killed {
			report.KilledMutations++
		} else {
			report.SurvivedMutations++
		}
		
		stats := categoryStats[result.Category]
		stats.total++
		if result.Killed {
			stats.killed++
		}
		categoryStats[result.Category] = stats
	}
	
	// Calculate overall mutation score
	if report.TotalMutations > 0 {
		report.MutationScore = float64(report.KilledMutations) / float64(report.TotalMutations) * 100
	}
	
	// Calculate category scores
	for category, stats := range categoryStats {
		if stats.total > 0 {
			report.CategoryScores[category] = float64(stats.killed) / float64(stats.total) * 100
		}
	}
}

// analyzeWeakTests identifies tests that failed to catch mutations
func (mt *MutationTester) analyzeWeakTests(report *MutationReport) {
	testFailures := make(map[string][]string)
	
	for _, result := range report.Results {
		if !result.Killed {
			// This mutation survived, indicating a weak test
			testKey := fmt.Sprintf("%s::%s", result.FilePath, result.Function)
			testFailures[testKey] = append(testFailures[testKey], result.MutationType)
		}
	}
	
	for testKey, missedMutations := range testFailures {
		severity := "medium"
		if len(missedMutations) > 3 {
			severity = "high"
		}
		
		weakTest := WeakTestReport{
			TestFunction:    testKey,
			MissedMutations: missedMutations,
			Severity:        severity,
			Suggestions:     mt.generateTestSuggestions(testKey, missedMutations),
		}
		
		report.WeakTests = append(report.WeakTests, weakTest)
	}
}

// generateTestSuggestions provides recommendations for improving weak tests
func (mt *MutationTester) generateTestSuggestions(testKey string, missedMutations []string) []string {
	var suggestions []string
	
	for _, mutation := range missedMutations {
		switch mutation {
		case "ConditionalBoundaryMutator":
			suggestions = append(suggestions, "Add boundary condition tests (==, !=, <, >, <=, >=)")
		case "ArithmeticOperatorMutator":
			suggestions = append(suggestions, "Test arithmetic operations with edge cases (zero, negative, overflow)")
		case "LogicalOperatorMutator":
			suggestions = append(suggestions, "Add tests for logical operator combinations (&&, ||, !)")
		case "ReturnValueMutator":
			suggestions = append(suggestions, "Verify return values in different scenarios")
		case "NullCheckMutator":
			suggestions = append(suggestions, "Add nil/null pointer tests")
		}
	}
	
	// Remove duplicates
	uniqueSuggestions := make([]string, 0, len(suggestions))
	seen := make(map[string]bool)
	for _, suggestion := range suggestions {
		if !seen[suggestion] {
			uniqueSuggestions = append(uniqueSuggestions, suggestion)
			seen[suggestion] = true
		}
	}
	
	return uniqueSuggestions
}

// generateRecommendations creates overall recommendations for improving test quality
func (mt *MutationTester) generateRecommendations(report *MutationReport) {
	var recommendations []string
	
	if report.MutationScore < mt.config.MinMutationScore {
		recommendations = append(recommendations, 
			fmt.Sprintf("Mutation score (%.1f%%) is below target (%.1f%%). Consider adding more comprehensive tests.", 
				report.MutationScore, mt.config.MinMutationScore))
	}
	
	// Category-specific recommendations
	for category, score := range report.CategoryScores {
		if score < 80 {
			switch category {
			case "security":
				recommendations = append(recommendations, 
					fmt.Sprintf("Security function mutation score (%.1f%%) is low. Add more security-focused tests.", score))
			case "business_logic":
				recommendations = append(recommendations, 
					fmt.Sprintf("Business logic mutation score (%.1f%%) is low. Add more edge case tests.", score))
			case "performance":
				recommendations = append(recommendations, 
					fmt.Sprintf("Performance function mutation score (%.1f%%) is low. Add performance validation tests.", score))
			}
		}
	}
	
	if len(report.WeakTests) > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Found %d weak test functions. Review and strengthen these tests.", len(report.WeakTests)))
	}
	
	report.Recommendations = recommendations
}

// GoTestRunner implements MutationTestRunner for Go tests
type GoTestRunner struct{}

// RunTests executes Go tests for a package
func (gtr *GoTestRunner) RunTests(packagePath string, timeout time.Duration) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "go", "test", "-v", packagePath)
	output, err := cmd.CombinedOutput()
	
	if ctx.Err() == context.DeadlineExceeded {
		return false, "Test execution timed out", ctx.Err()
	}
	
	// Tests pass if exit code is 0
	testsPassed := err == nil
	
	return testsPassed, string(output), err
}

// Helper functions
func generateMutationID(mutation Mutation) string {
	return fmt.Sprintf("%s:%d:%s", 
		filepath.Base(mutation.FilePath), 
		mutation.Position.Line, 
		mutation.Mutator.Name())
}

func createTempWorkDir() string {
	workDir, err := os.MkdirTemp("", "mutation_testing_*")
	if err != nil {
		log.Fatalf("Failed to create work directory: %v", err)
	}
	return workDir
}