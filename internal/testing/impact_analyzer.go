package testing

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ImpactAnalyzer analyzes the impact of code changes on test selection
type ImpactAnalyzer struct {
	fileSet       *token.FileSet
	dependencyMap map[string][]string // file -> dependencies
	callGraph     map[string][]string // function -> called functions
}

// ImpactAnalysis contains the results of impact analysis
type ImpactAnalysis struct {
	AffectedFiles    map[string]bool     `json:"affected_files"`
	AffectedPackages map[string]bool     `json:"affected_packages"`
	AffectedFunctions map[string]bool    `json:"affected_functions"`
	ImpactScore      map[string]float64  `json:"impact_score"`
	ChangeTypes      map[string]ImpactChangeType `json:"change_types"`
	RiskLevel        RiskLevel           `json:"risk_level"`
}

// ImpactScore represents different types of impact scores
type ImpactScore struct {
	Overall     float64 `json:"overall"`
	Functional  float64 `json:"functional"`
	Performance float64 `json:"performance"`
	Security    float64 `json:"security"`
	Integration float64 `json:"integration"`
}

// ImpactChangeType represents the type of change made to a file
type ImpactChangeType string

const (
	ImpactChangeTypeAddition    ImpactChangeType = "addition"
	ImpactChangeTypeModification ImpactChangeType = "modification"
	ImpactChangeTypeDeletion    ImpactChangeType = "deletion"
	ImpactChangeTypeRename      ImpactChangeType = "rename"
)

// RiskLevel represents the risk level of changes
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// NewImpactAnalyzer creates a new impact analyzer
func NewImpactAnalyzer() *ImpactAnalyzer {
	return &ImpactAnalyzer{
		fileSet:       token.NewFileSet(),
		dependencyMap: make(map[string][]string),
		callGraph:     make(map[string][]string),
	}
}

// AnalyzeImpact analyzes the impact of code changes
func (a *ImpactAnalyzer) AnalyzeImpact(ctx context.Context, changedFiles []string) (*ImpactAnalysis, error) {
	log.Printf("Analyzing impact of %d changed files", len(changedFiles))
	
	analysis := &ImpactAnalysis{
		AffectedFiles:     make(map[string]bool),
		AffectedPackages:  make(map[string]bool),
		AffectedFunctions: make(map[string]bool),
		ImpactScore:       make(map[string]float64),
		ChangeTypes:       make(map[string]ChangeType),
	}
	
	// Step 1: Build dependency graph if not already built
	if len(a.dependencyMap) == 0 {
		if err := a.buildDependencyGraph(ctx); err != nil {
			return nil, fmt.Errorf("failed to build dependency graph: %w", err)
		}
	}
	
	// Step 2: Analyze each changed file
	for _, file := range changedFiles {
		if err := a.analyzeFileImpact(ctx, file, analysis); err != nil {
			log.Printf("Warning: failed to analyze file %s: %v", file, err)
			continue
		}
	}
	
	// Step 3: Calculate transitive dependencies
	a.calculateTransitiveDependencies(analysis)
	
	// Step 4: Determine overall risk level
	analysis.RiskLevel = a.calculateRiskLevel(analysis)
	
	log.Printf("Impact analysis complete: %d affected files, %d affected packages, risk level: %s", 
		len(analysis.AffectedFiles), len(analysis.AffectedPackages), analysis.RiskLevel)
	
	return analysis, nil
}

// buildDependencyGraph builds a dependency graph of the codebase
func (a *ImpactAnalyzer) buildDependencyGraph(ctx context.Context) error {
	log.Println("Building dependency graph...")
	
	return filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip non-Go files and test files for dependency analysis
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		
		// Skip vendor and .git directories
		if strings.Contains(path, "vendor/") || strings.Contains(path, ".git/") {
			return nil
		}
		
		// Parse the file to extract dependencies
		dependencies, err := a.extractDependencies(path)
		if err != nil {
			log.Printf("Warning: failed to extract dependencies from %s: %v", path, err)
			return nil
		}
		
		a.dependencyMap[path] = dependencies
		return nil
	})
}

// extractDependencies extracts dependencies from a Go file
func (a *ImpactAnalyzer) extractDependencies(filePath string) ([]string, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	// Parse the Go file
	file, err := parser.ParseFile(a.fileSet, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	
	var dependencies []string
	
	// Extract import dependencies
	for _, imp := range file.Imports {
		if imp.Path != nil {
			importPath := strings.Trim(imp.Path.Value, "\"")
			dependencies = append(dependencies, importPath)
		}
	}
	
	// Extract function call dependencies (simplified)
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					funcCall := fmt.Sprintf("%s.%s", ident.Name, sel.Sel.Name)
					dependencies = append(dependencies, funcCall)
				}
			}
		}
		return true
	})
	
	return dependencies, nil
}

// analyzeFileImpact analyzes the impact of a single file change
func (a *ImpactAnalyzer) analyzeFileImpact(ctx context.Context, filePath string, analysis *ImpactAnalysis) error {
	// Mark file as affected
	analysis.AffectedFiles[filePath] = true
	
	// Mark package as affected
	packagePath := filepath.Dir(filePath)
	analysis.AffectedPackages[packagePath] = true
	
	// Determine change type
	changeType := a.determineChangeType(filePath)
	analysis.ChangeTypes[filePath] = changeType
	
	// Calculate impact score
	impactScore := a.calculateFileImpactScore(filePath, changeType)
	analysis.ImpactScore[filePath] = impactScore
	
	// Analyze specific file types for higher impact
	if strings.Contains(filePath, "repository") || strings.Contains(filePath, "service") {
		// Repository and service changes have higher impact
		analysis.ImpactScore[filePath] *= 1.5
	}
	
	if strings.Contains(filePath, "auth") || strings.Contains(filePath, "security") {
		// Security-related changes have critical impact
		analysis.ImpactScore[filePath] *= 2.0
	}
	
	if strings.Contains(filePath, "config") || strings.Contains(filePath, "main.go") {
		// Configuration and main files have high impact
		analysis.ImpactScore[filePath] *= 1.8
	}
	
	return nil
}

// determineChangeType determines the type of change for a file
func (a *ImpactAnalyzer) determineChangeType(filePath string) ChangeType {
	// In a real implementation, this would use git diff or similar
	// For now, we'll assume modification for existing files
	if _, err := os.Stat(filePath); err == nil {
		return ChangeTypeModification
	}
	return ChangeTypeAddition
}

// calculateFileImpactScore calculates the impact score for a file
func (a *ImpactAnalyzer) calculateFileImpactScore(filePath string, changeType ChangeType) float64 {
	baseScore := 0.5
	
	// Adjust based on change type
	switch changeType {
	case ChangeTypeAddition:
		baseScore = 0.3 // New files have lower initial impact
	case ChangeTypeModification:
		baseScore = 0.7 // Modified files have higher impact
	case ChangeTypeDeletion:
		baseScore = 0.9 // Deleted files have high impact
	case ChangeTypeRename:
		baseScore = 0.6 // Renamed files have medium impact
	}
	
	// Adjust based on file type and location
	if strings.HasSuffix(filePath, "_test.go") {
		baseScore *= 0.5 // Test files have lower impact on other tests
	}
	
	if strings.Contains(filePath, "internal/") {
		baseScore *= 1.2 // Internal packages have higher impact
	}
	
	if strings.Contains(filePath, "cmd/") {
		baseScore *= 1.5 // Command packages have higher impact
	}
	
	// Check dependency count
	if deps, exists := a.dependencyMap[filePath]; exists {
		// Files with more dependencies have higher impact
		depFactor := 1.0 + float64(len(deps))*0.1
		baseScore *= depFactor
	}
	
	// Ensure score is between 0 and 1
	if baseScore > 1.0 {
		baseScore = 1.0
	}
	
	return baseScore
}

// calculateTransitiveDependencies calculates transitive dependencies
func (a *ImpactAnalyzer) calculateTransitiveDependencies(analysis *ImpactAnalysis) {
	// Find all files that depend on the affected files
	for filePath := range analysis.AffectedFiles {
		a.findDependentFiles(filePath, analysis, make(map[string]bool))
	}
}

// findDependentFiles recursively finds files that depend on the given file
func (a *ImpactAnalyzer) findDependentFiles(targetFile string, analysis *ImpactAnalysis, visited map[string]bool) {
	if visited[targetFile] {
		return
	}
	visited[targetFile] = true
	
	// Look for files that import or depend on the target file
	for filePath, dependencies := range a.dependencyMap {
		for _, dep := range dependencies {
			// Check if this file depends on the target file
			if strings.Contains(dep, filepath.Base(targetFile)) || 
			   strings.Contains(dep, filepath.Dir(targetFile)) {
				
				if !analysis.AffectedFiles[filePath] {
					analysis.AffectedFiles[filePath] = true
					analysis.AffectedPackages[filepath.Dir(filePath)] = true
					
					// Reduce impact score for transitive dependencies
					if originalScore, exists := analysis.ImpactScore[targetFile]; exists {
						analysis.ImpactScore[filePath] = originalScore * 0.7
					} else {
						analysis.ImpactScore[filePath] = 0.3
					}
					
					// Recursively find dependencies of this file
					a.findDependentFiles(filePath, analysis, visited)
				}
			}
		}
	}
}

// calculateRiskLevel calculates the overall risk level of the changes
func (a *ImpactAnalyzer) calculateRiskLevel(analysis *ImpactAnalysis) RiskLevel {
	totalScore := 0.0
	maxScore := 0.0
	
	for _, score := range analysis.ImpactScore {
		totalScore += score
		if score > maxScore {
			maxScore = score
		}
	}
	
	avgScore := totalScore / float64(len(analysis.ImpactScore))
	
	// Determine risk level based on average and maximum scores
	if maxScore >= 0.9 || avgScore >= 0.8 {
		return RiskLevelCritical
	} else if maxScore >= 0.7 || avgScore >= 0.6 {
		return RiskLevelHigh
	} else if maxScore >= 0.5 || avgScore >= 0.4 {
		return RiskLevelMedium
	}
	
	return RiskLevelLow
}

// GetAffectedTests returns tests that are affected by the given files
func (a *ImpactAnalyzer) GetAffectedTests(ctx context.Context, changedFiles []string) ([]string, error) {
	var affectedTests []string
	
	// Find test files that directly test the changed files
	for _, file := range changedFiles {
		// Look for corresponding test file
		testFile := strings.Replace(file, ".go", "_test.go", 1)
		if _, err := os.Stat(testFile); err == nil {
			affectedTests = append(affectedTests, testFile)
		}
		
		// Look for integration tests in the same package
		packageDir := filepath.Dir(file)
		testPattern := filepath.Join(packageDir, "*_test.go")
		matches, err := filepath.Glob(testPattern)
		if err == nil {
			affectedTests = append(affectedTests, matches...)
		}
	}
	
	// Remove duplicates
	testMap := make(map[string]bool)
	var uniqueTests []string
	for _, test := range affectedTests {
		if !testMap[test] {
			testMap[test] = true
			uniqueTests = append(uniqueTests, test)
		}
	}
	
	return uniqueTests, nil
}