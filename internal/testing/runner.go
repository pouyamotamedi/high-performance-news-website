package testing

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestRunner provides comprehensive test execution capabilities
type TestRunner struct {
	Config          *TestConfig
	CoverageTarget  float64
	ParallelTests   bool
	VerboseOutput   bool
	FailFast        bool
	TestTimeout     time.Duration
	BenchmarkTime   time.Duration
	OutputDir       string
}

// NewTestRunner creates a new test runner with default configuration
func NewTestRunner() *TestRunner {
	return &TestRunner{
		Config:          GetTestConfig(),
		CoverageTarget:  95.0, // 95% coverage target as per requirements
		ParallelTests:   true,
		VerboseOutput:   false,
		FailFast:        false,
		TestTimeout:     30 * time.Minute,
		BenchmarkTime:   10 * time.Second,
		OutputDir:       "./test-results",
	}
}

// TestRunResult represents the result of a test run
type TestRunResult struct {
	Success         bool
	TotalTests      int
	PassedTests     int
	FailedTests     int
	SkippedTests    int
	Duration        time.Duration
	CoveragePercent float64
	Output          string
	Errors          []string
}

// RunAllTests runs the complete test suite with coverage tracking
func (tr *TestRunner) RunAllTests(ctx context.Context) (*TestRunResult, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(tr.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	result := &TestRunResult{
		Errors: make([]string, 0),
	}

	startTime := time.Now()

	// Run unit tests with coverage
	unitResult, err := tr.runUnitTests(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Unit tests failed: %v", err))
	} else {
		result.TotalTests += unitResult.TotalTests
		result.PassedTests += unitResult.PassedTests
		result.FailedTests += unitResult.FailedTests
		result.SkippedTests += unitResult.SkippedTests
		result.CoveragePercent = unitResult.CoveragePercent
		result.Output += unitResult.Output
	}

	// Run integration tests if database is available
	if tr.isDatabaseAvailable() {
		integrationResult, err := tr.runIntegrationTests(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Integration tests failed: %v", err))
		} else {
			result.TotalTests += integrationResult.TotalTests
			result.PassedTests += integrationResult.PassedTests
			result.FailedTests += integrationResult.FailedTests
			result.SkippedTests += integrationResult.SkippedTests
			result.Output += integrationResult.Output
		}
	}

	result.Duration = time.Since(startTime)
	result.Success = len(result.Errors) == 0 && result.CoveragePercent >= tr.CoverageTarget

	// Generate coverage report
	if err := tr.generateCoverageReport(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Coverage report generation failed: %v", err))
	}

	return result, nil
}

// runUnitTests runs unit tests with coverage tracking
func (tr *TestRunner) runUnitTests(ctx context.Context) (*TestRunResult, error) {
	args := []string{"test"}
	
	if tr.VerboseOutput {
		args = append(args, "-v")
	}
	
	if tr.ParallelTests {
		args = append(args, fmt.Sprintf("-parallel=%d", runtime.NumCPU()))
	}
	
	if tr.FailFast {
		args = append(args, "-failfast")
	}

	// Add coverage flags
	coverageFile := filepath.Join(tr.OutputDir, "coverage.out")
	args = append(args, "-race", "-coverprofile="+coverageFile, "-covermode=atomic")

	// Add timeout
	args = append(args, "-timeout="+tr.TestTimeout.String())

	// Test packages
	packages := []string{
		"./internal/models/...",
		"./internal/repositories/...",
		"./internal/services/...",
		"./internal/api/...",
		"./internal/auth/...",
		"./internal/validation/...",
		"./pkg/...",
	}
	args = append(args, packages...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1") // Enable CGO for race detector
	
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	result := &TestRunResult{
		Output: outputStr,
	}

	// Parse test results
	tr.parseTestOutput(outputStr, result)

	// Calculate coverage
	if err == nil {
		coverage, coverageErr := tr.calculateCoverage(coverageFile)
		if coverageErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Coverage calculation failed: %v", coverageErr))
		} else {
			result.CoveragePercent = coverage
		}
	}

	if err != nil {
		return result, fmt.Errorf("unit tests failed: %w", err)
	}

	return result, nil
}

// runIntegrationTests runs integration tests
func (tr *TestRunner) runIntegrationTests(ctx context.Context) (*TestRunResult, error) {
	args := []string{"test"}
	
	if tr.VerboseOutput {
		args = append(args, "-v")
	}

	// Add integration test tags
	args = append(args, "-tags=integration")
	args = append(args, "-timeout="+tr.TestTimeout.String())

	// Integration test packages
	packages := []string{
		"./internal/integration/...",
		"./internal/repositories/...",
	}
	args = append(args, packages...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = append(os.Environ(), 
		"CGO_ENABLED=1",
		"TEST_DATABASE_URL="+tr.Config.DatabaseURL,
		"TEST_CACHE_URL="+tr.Config.CacheURL,
	)
	
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	result := &TestRunResult{
		Output: outputStr,
	}

	// Parse test results
	tr.parseTestOutput(outputStr, result)

	if err != nil {
		return result, fmt.Errorf("integration tests failed: %w", err)
	}

	return result, nil
}

// RunBenchmarks runs performance benchmarks
func (tr *TestRunner) RunBenchmarks(ctx context.Context) (*TestRunResult, error) {
	args := []string{"test", "-bench=.", "-benchmem", "-benchtime=" + tr.BenchmarkTime.String()}
	
	if tr.VerboseOutput {
		args = append(args, "-v")
	}

	// Benchmark packages
	packages := []string{
		"./internal/models/...",
		"./internal/repositories/...",
		"./internal/services/...",
		"./pkg/...",
	}
	args = append(args, packages...)

	cmd := exec.CommandContext(ctx, "go", args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	result := &TestRunResult{
		Output: outputStr,
	}

	// Save benchmark results
	benchmarkFile := filepath.Join(tr.OutputDir, "benchmark.txt")
	if writeErr := os.WriteFile(benchmarkFile, output, 0644); writeErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to save benchmark results: %v", writeErr))
	}

	if err != nil {
		return result, fmt.Errorf("benchmarks failed: %w", err)
	}

	return result, nil
}

// calculateCoverage calculates test coverage from coverage file
func (tr *TestRunner) calculateCoverage(coverageFile string) (float64, error) {
	cmd := exec.Command("go", "tool", "cover", "-func="+coverageFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to calculate coverage: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "total:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				coverageStr := strings.TrimSuffix(parts[2], "%")
				var coverage float64
				if _, err := fmt.Sscanf(coverageStr, "%f", &coverage); err == nil {
					return coverage, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("could not parse coverage from output")
}

// generateCoverageReport generates HTML coverage report
func (tr *TestRunner) generateCoverageReport() error {
	coverageFile := filepath.Join(tr.OutputDir, "coverage.out")
	htmlFile := filepath.Join(tr.OutputDir, "coverage.html")

	cmd := exec.Command("go", "tool", "cover", "-html="+coverageFile, "-o", htmlFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate HTML coverage report: %w", err)
	}

	return nil
}

// parseTestOutput parses go test output to extract test results
func (tr *TestRunner) parseTestOutput(output string, result *TestRunResult) {
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.Contains(line, "PASS:") {
			result.PassedTests++
		} else if strings.Contains(line, "FAIL:") {
			result.FailedTests++
		} else if strings.Contains(line, "SKIP:") {
			result.SkippedTests++
		}
		
		// Count total tests from summary lines
		if strings.Contains(line, "ok") && strings.Contains(line, "coverage:") {
			// This is a package summary line
			result.TotalTests++
		}
	}
}

// isDatabaseAvailable checks if test database is available
func (tr *TestRunner) isDatabaseAvailable() bool {
	env := NewTestEnvironment(&testing.T{})
	defer env.TearDown()
	return env.HasDatabase()
}

// ValidateCoverage validates that coverage meets the minimum requirement
func (tr *TestRunner) ValidateCoverage(coverage float64) error {
	if coverage < tr.CoverageTarget {
		return fmt.Errorf("coverage %.2f%% is below target %.2f%%", coverage, tr.CoverageTarget)
	}
	return nil
}

// GenerateTestReport generates a comprehensive test report
func (tr *TestRunner) GenerateTestReport(result *TestRunResult) error {
	reportFile := filepath.Join(tr.OutputDir, "test-report.txt")
	
	report := fmt.Sprintf(`Test Execution Report
=====================

Execution Time: %v
Total Tests: %d
Passed: %d
Failed: %d
Skipped: %d
Success Rate: %.2f%%
Coverage: %.2f%%
Target Coverage: %.2f%%
Coverage Met: %t

Errors:
%s

Output:
%s
`,
		result.Duration,
		result.TotalTests,
		result.PassedTests,
		result.FailedTests,
		result.SkippedTests,
		float64(result.PassedTests)/float64(result.TotalTests)*100,
		result.CoveragePercent,
		tr.CoverageTarget,
		result.CoveragePercent >= tr.CoverageTarget,
		strings.Join(result.Errors, "\n"),
		result.Output,
	)

	if err := os.WriteFile(reportFile, []byte(report), 0644); err != nil {
		return fmt.Errorf("failed to write test report: %w", err)
	}

	return nil
}

// TestExecutor provides methods for executing specific test types
type TestExecutor struct {
	Runner *TestRunner
}

// NewTestExecutor creates a new test executor
func NewTestExecutor() *TestExecutor {
	return &TestExecutor{
		Runner: NewTestRunner(),
	}
}

// ExecuteUnitTests executes only unit tests
func (te *TestExecutor) ExecuteUnitTests(ctx context.Context) error {
	result, err := te.Runner.runUnitTests(ctx)
	if err != nil {
		return err
	}

	if err := te.Runner.ValidateCoverage(result.CoveragePercent); err != nil {
		return err
	}

	return te.Runner.GenerateTestReport(result)
}

// ExecuteIntegrationTests executes only integration tests
func (te *TestExecutor) ExecuteIntegrationTests(ctx context.Context) error {
	if !te.Runner.isDatabaseAvailable() {
		return fmt.Errorf("database not available for integration tests")
	}

	result, err := te.Runner.runIntegrationTests(ctx)
	if err != nil {
		return err
	}

	return te.Runner.GenerateTestReport(result)
}

// ExecuteAllTests executes the complete test suite
func (te *TestExecutor) ExecuteAllTests(ctx context.Context) error {
	result, err := te.Runner.RunAllTests(ctx)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("test suite failed: %s", strings.Join(result.Errors, "; "))
	}

	return te.Runner.GenerateTestReport(result)
}

// ExecuteBenchmarks executes performance benchmarks
func (te *TestExecutor) ExecuteBenchmarks(ctx context.Context) error {
	result, err := te.Runner.RunBenchmarks(ctx)
	if err != nil {
		return err
	}

	return te.Runner.GenerateTestReport(result)
}