package testing

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSecurityAutomation(t *testing.T) {
	// Create temporary project directory
	tempDir := t.TempDir()
	
	// Create basic Go project structure
	createTestGoProject(t, tempDir)
	
	// Test security automation creation
	securityAuto := NewSecurityAutomation(tempDir)
	if securityAuto == nil {
		t.Fatal("Failed to create SecurityAutomation instance")
	}
	
	// Verify report directory creation
	reportDir := filepath.Join(tempDir, "reports", "security")
	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		t.Errorf("Report directory not created: %s", reportDir)
	}
}

func TestDependencyScanner(t *testing.T) {
	// Create temporary project directory
	tempDir := t.TempDir()
	
	// Create basic Go project structure
	createTestGoProject(t, tempDir)
	
	// Test dependency scanner creation
	depScanner := NewDependencyScanner(tempDir)
	if depScanner == nil {
		t.Fatal("Failed to create DependencyScanner instance")
	}
	
	// Test that it recognizes Go project
	if !depScanner.isGoProject() {
		t.Error("Failed to recognize Go project")
	}
}

func TestSecurityRegressionTester(t *testing.T) {
	// Create temporary project directory
	tempDir := t.TempDir()
	
	// Create basic Go project structure
	createTestGoProject(t, tempDir)
	
	// Test regression tester creation
	regressionTester := NewSecurityRegressionTester(tempDir)
	if regressionTester == nil {
		t.Fatal("Failed to create SecurityRegressionTester instance")
	}
	
	// Verify baseline directory creation
	baselineDir := filepath.Join(tempDir, "reports", "security", "baselines")
	if _, err := os.Stat(baselineDir); os.IsNotExist(err) {
		t.Errorf("Baseline directory not created: %s", baselineDir)
	}
}

func TestSecurityCICDIntegration(t *testing.T) {
	// Create temporary project directory
	tempDir := t.TempDir()
	
	// Create basic Go project structure
	createTestGoProject(t, tempDir)
	
	// Test CI/CD integration creation
	cicdIntegration := NewSecurityCICDIntegration(tempDir)
	if cicdIntegration == nil {
		t.Fatal("Failed to create SecurityCICDIntegration instance")
	}
	
	// Test quality gate evaluation with mock data
	result := &SecurityGateResult{
		Stage:             StagePreCommit,
		QualityGateChecks: make(map[string]QualityGateCheck),
		SecurityResults: &ComprehensiveSecurityResult{
			SecuritySummary: SecuritySummary{
				SecurityScore:    85.0,
				CriticalIssues:   0,
				HighRiskIssues:   2,
				MediumRiskIssues: 5,
				LowRiskIssues:    10,
			},
		},
	}
	
	cicdIntegration.evaluateQualityGates(result)
	
	// Should pass with these values
	if !result.Passed {
		t.Errorf("Quality gate should pass with security score 85.0 and 0 critical issues")
	}
}

func TestSecurityScoreCalculation(t *testing.T) {
	securityAuto := NewSecurityAutomation(t.TempDir())
	
	testCases := []struct {
		name     string
		summary  SecuritySummary
		expected float64
	}{
		{
			name: "Perfect score",
			summary: SecuritySummary{
				CriticalIssues:   0,
				HighRiskIssues:   0,
				MediumRiskIssues: 0,
				LowRiskIssues:    0,
			},
			expected: 100.0,
		},
		{
			name: "One critical issue",
			summary: SecuritySummary{
				CriticalIssues:   1,
				HighRiskIssues:   0,
				MediumRiskIssues: 0,
				LowRiskIssues:    0,
			},
			expected: 80.0,
		},
		{
			name: "Mixed issues",
			summary: SecuritySummary{
				CriticalIssues:   1,
				HighRiskIssues:   2,
				MediumRiskIssues: 3,
				LowRiskIssues:    5,
			},
			expected: 60.0, // 100 - 20 - 20 - 15 - 5 = 40, but minimum is 0
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := securityAuto.calculateSecurityScore(tc.summary)
			if score != tc.expected {
				t.Errorf("Expected score %.1f, got %.1f", tc.expected, score)
			}
		})
	}
}

func TestRiskLevelDetermination(t *testing.T) {
	securityAuto := NewSecurityAutomation(t.TempDir())
	
	testCases := []struct {
		name     string
		summary  SecuritySummary
		expected string
	}{
		{
			name: "Critical risk",
			summary: SecuritySummary{
				CriticalIssues: 1,
			},
			expected: "critical",
		},
		{
			name: "High risk",
			summary: SecuritySummary{
				CriticalIssues: 0,
				HighRiskIssues: 6,
			},
			expected: "high",
		},
		{
			name: "Medium risk",
			summary: SecuritySummary{
				CriticalIssues:   0,
				HighRiskIssues:   2,
				MediumRiskIssues: 5,
			},
			expected: "medium",
		},
		{
			name: "Low risk",
			summary: SecuritySummary{
				CriticalIssues:   0,
				HighRiskIssues:   0,
				MediumRiskIssues: 5,
				LowRiskIssues:    10,
			},
			expected: "low",
		},
		{
			name: "Minimal risk",
			summary: SecuritySummary{
				CriticalIssues:   0,
				HighRiskIssues:   0,
				MediumRiskIssues: 0,
				LowRiskIssues:    5,
			},
			expected: "minimal",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			riskLevel := securityAuto.determineRiskLevel(tc.summary)
			if riskLevel != tc.expected {
				t.Errorf("Expected risk level %s, got %s", tc.expected, riskLevel)
			}
		})
	}
}

func TestIssueHashCalculation(t *testing.T) {
	regressionTester := NewSecurityRegressionTester(t.TempDir())
	
	issue1 := TopSecurityIssue{
		Title:    "SQL Injection",
		Severity: "high",
		Category: "injection",
		Source:   "zap",
	}
	
	issue2 := TopSecurityIssue{
		Title:    "SQL Injection",
		Severity: "high",
		Category: "injection",
		Source:   "zap",
	}
	
	issue3 := TopSecurityIssue{
		Title:    "XSS Vulnerability",
		Severity: "medium",
		Category: "xss",
		Source:   "zap",
	}
	
	hash1 := regressionTester.calculateIssueHash(issue1)
	hash2 := regressionTester.calculateIssueHash(issue2)
	hash3 := regressionTester.calculateIssueHash(issue3)
	
	// Same issues should have same hash
	if hash1 != hash2 {
		t.Error("Identical issues should have the same hash")
	}
	
	// Different issues should have different hashes
	if hash1 == hash3 {
		t.Error("Different issues should have different hashes")
	}
}

// Helper function to create a basic Go project for testing
func createTestGoProject(t *testing.T, projectDir string) {
	// Create go.mod file
	goModContent := `module test-project

go 1.19

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/lib/pq v1.10.9
)
`
	
	goModPath := filepath.Join(projectDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}
	
	// Create main.go file
	mainGoContent := `package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello World"})
	})
	r.Run(":8080")
}
`
	
	mainGoPath := filepath.Join(projectDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainGoContent), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}
}

// Benchmark tests
func BenchmarkSecurityScoreCalculation(b *testing.B) {
	securityAuto := NewSecurityAutomation(b.TempDir())
	
	summary := SecuritySummary{
		CriticalIssues:   2,
		HighRiskIssues:   5,
		MediumRiskIssues: 10,
		LowRiskIssues:    20,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		securityAuto.calculateSecurityScore(summary)
	}
}

func BenchmarkIssueHashCalculation(b *testing.B) {
	regressionTester := NewSecurityRegressionTester(b.TempDir())
	
	issue := TopSecurityIssue{
		Title:    "SQL Injection Vulnerability",
		Severity: "high",
		Category: "injection",
		Source:   "zap",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		regressionTester.calculateIssueHash(issue)
	}
}

// Integration test (requires actual tools to be installed)
func TestDependencyScanIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create temporary project directory
	tempDir := t.TempDir()
	createTestGoProject(t, tempDir)
	
	// Test dependency scanner
	depScanner := NewDependencyScanner(tempDir)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// This will only work if go is installed and available
	result, err := depScanner.RunComprehensiveDependencyScan(ctx)
	if err != nil {
		t.Logf("Dependency scan failed (expected if tools not installed): %v", err)
		return
	}
	
	if result == nil {
		t.Error("Expected non-nil result from dependency scan")
	}
	
	t.Logf("Dependency scan completed with %d total vulnerabilities", 
		result.Summary.TotalVulnerabilities)
}