package validation

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReportingSystem_ValidateAndReport(t *testing.T) {
	// Create temporary directory for reports
	tmpDir := t.TempDir()
	
	config := ReportingConfig{
		OutputDir: tmpDir,
		BlockingThresholds: BlockingThresholds{
			CriticalIssues: 0,
			HighIssues:     3,
			TotalScore:     70,
			ManualReview:   5,
		},
		TrendTracking: TrendTrackingConfig{
			Enabled:       true,
			RetentionDays: 30,
		},
	}
	
	reporting := NewReportingSystem(config)
	
	// Create test files with issues
	testFiles := []string{
		createTestFileWithIssues(t, "critical"),
		createTestFileWithIssues(t, "high"),
		createTestFileWithIssues(t, "medium"),
	}
	
	// Run validation and reporting
	report, err := reporting.ValidateAndReport(testFiles)
	if err != nil {
		t.Fatalf("ValidateAndReport() error = %v", err)
	}
	
	// Verify report structure
	if report == nil {
		t.Fatal("Expected non-nil report")
	}
	
	if report.Summary.TotalFiles != len(testFiles) {
		t.Errorf("Expected %d files, got %d", len(testFiles), report.Summary.TotalFiles)
	}
	
	if report.Summary.TotalIssues == 0 {
		t.Error("Expected issues to be found")
	}
	
	if len(report.FileReports) != len(testFiles) {
		t.Errorf("Expected %d file reports, got %d", len(testFiles), len(report.FileReports))
	}
	
	// Verify blocking logic
	if !report.ShouldBlock {
		t.Error("Expected report to block deployment due to critical issues")
	}
	
	if len(report.BlockingReasons) == 0 {
		t.Error("Expected blocking reasons to be provided")
	}
	
	// Verify recommendations
	if len(report.Recommendations) == 0 {
		t.Error("Expected recommendations to be generated")
	}
	
	// Verify files were saved
	jsonPath := filepath.Join(tmpDir, "latest-report.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("Expected JSON report to be saved")
	}
	
	htmlPath := filepath.Join(tmpDir, "validation-report.html")
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		t.Error("Expected HTML report to be saved")
	}
}

func TestReportingSystem_CalculateOverallScore(t *testing.T) {
	config := GetDefaultConfig()
	reporting := NewReportingSystem(config)
	
	tests := []struct {
		name     string
		summary  AggregationSummary
		expected float64
	}{
		{
			name: "perfect score",
			summary: AggregationSummary{
				TotalFiles: 10,
			},
			expected: 100.0,
		},
		{
			name: "critical issues",
			summary: AggregationSummary{
				TotalFiles:     10,
				CriticalIssues: 1,
			},
			expected: 80.0, // Should be significantly penalized
		},
		{
			name: "mixed issues",
			summary: AggregationSummary{
				TotalFiles:   10,
				HighIssues:   2,
				MediumIssues: 3,
				LowIssues:    5,
			},
			expected: 76.0, // Should be moderately penalized
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := reporting.calculateOverallScore(tt.summary)
			
			// Allow some tolerance for floating point comparison
			tolerance := 5.0
			if score < tt.expected-tolerance || score > tt.expected+tolerance {
				t.Errorf("calculateOverallScore() = %.2f, expected ~%.2f", score, tt.expected)
			}
		})
	}
}

func TestReportingSystem_ShouldBlockDeployment(t *testing.T) {
	tests := []struct {
		name       string
		config     BlockingThresholds
		summary    AggregationSummary
		shouldBlock bool
	}{
		{
			name: "critical issues should block",
			config: BlockingThresholds{
				CriticalIssues: 0,
			},
			summary: AggregationSummary{
				CriticalIssues: 1,
			},
			shouldBlock: true,
		},
		{
			name: "high issues threshold",
			config: BlockingThresholds{
				HighIssues: 3,
			},
			summary: AggregationSummary{
				HighIssues: 4,
			},
			shouldBlock: true,
		},
		{
			name: "low score should block",
			config: BlockingThresholds{
				TotalScore: 70,
			},
			summary: AggregationSummary{
				OverallScore: 65,
			},
			shouldBlock: true,
		},
		{
			name: "good quality should not block",
			config: BlockingThresholds{
				CriticalIssues: 0,
				HighIssues:     3,
				TotalScore:     70,
			},
			summary: AggregationSummary{
				CriticalIssues: 0,
				HighIssues:     2,
				OverallScore:   85,
			},
			shouldBlock: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ReportingConfig{
				BlockingThresholds: tt.config,
			}
			reporting := NewReportingSystem(config)
			
			report := &AggregatedReport{
				Summary: tt.summary,
			}
			
			shouldBlock, reasons := reporting.shouldBlockDeployment(report)
			
			if shouldBlock != tt.shouldBlock {
				t.Errorf("shouldBlockDeployment() = %v, expected %v", shouldBlock, tt.shouldBlock)
			}
			
			if tt.shouldBlock && len(reasons) == 0 {
				t.Error("Expected blocking reasons when deployment should be blocked")
			}
		})
	}
}

func TestReportingSystem_ExportReport(t *testing.T) {
	config := GetDefaultConfig()
	reporting := NewReportingSystem(config)
	
	// Create sample report
	report := &AggregatedReport{
		Summary: AggregationSummary{
			TotalFiles:     5,
			TotalIssues:    10,
			CriticalIssues: 1,
			HighIssues:     2,
			MediumIssues:   3,
			LowIssues:      4,
			OverallScore:   75.5,
			QualityGrade:   "C",
		},
		FileReports: []ValidationReport{
			{
				FilePath: "test1.go",
				Results: []ValidationResult{
					{
						Severity:   SeverityCritical,
						Category:   "security",
						Message:    "SQL injection risk",
						File:       "test1.go",
						Line:       10,
						Column:     5,
						RuleName:   "sql-injection-risk",
						Suggestion: "Use parameterized queries",
					},
				},
			},
		},
		GeneratedAt: time.Now(),
	}
	
	tests := []struct {
		name   string
		format string
	}{
		{"JSON export", "json"},
		{"CSV export", "csv"},
		{"JUnit export", "junit"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			
			err := reporting.ExportReport(report, tt.format, &buf)
			if err != nil {
				t.Fatalf("ExportReport() error = %v", err)
			}
			
			output := buf.String()
			if len(output) == 0 {
				t.Error("Expected non-empty export output")
			}
			
			// Verify format-specific content
			switch tt.format {
			case "json":
				var jsonData map[string]interface{}
				if err := json.Unmarshal(buf.Bytes(), &jsonData); err != nil {
					t.Errorf("Invalid JSON output: %v", err)
				}
			case "csv":
				if !strings.Contains(output, "File,Rule,Severity") {
					t.Error("CSV output missing expected headers")
				}
			case "junit":
				if !strings.Contains(output, "<?xml version") {
					t.Error("JUnit output missing XML declaration")
				}
				if !strings.Contains(output, "testsuites") {
					t.Error("JUnit output missing testsuites element")
				}
			}
		})
	}
}

func TestReportingSystem_GenerateRecommendations(t *testing.T) {
	config := GetDefaultConfig()
	reporting := NewReportingSystem(config)
	
	report := &AggregatedReport{
		Summary: AggregationSummary{
			CriticalIssues: 2,
			OverallScore:   65,
			ManualReview:   3,
		},
		CategoryAnalysis: CategoryAnalysis{
			Categories: []CategoryStats{
				{
					Name:       "security",
					Count:      10,
					Percentage: 50,
				},
			},
		},
	}
	
	recommendations := reporting.generateRecommendations(report)
	
	if len(recommendations) == 0 {
		t.Error("Expected recommendations to be generated")
	}
	
	// Check for critical issue recommendation
	foundCritical := false
	for _, rec := range recommendations {
		if rec.Priority == "high" && strings.Contains(rec.Title, "Critical") {
			foundCritical = true
			break
		}
	}
	
	if !foundCritical {
		t.Error("Expected high priority recommendation for critical issues")
	}
	
	// Verify recommendation structure
	for _, rec := range recommendations {
		if rec.Title == "" {
			t.Error("Recommendation missing title")
		}
		if rec.Description == "" {
			t.Error("Recommendation missing description")
		}
		if rec.Action == "" {
			t.Error("Recommendation missing action")
		}
		if rec.Priority == "" {
			t.Error("Recommendation missing priority")
		}
	}
}

func TestReportingSystem_TrendTracking(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := ReportingConfig{
		OutputDir: tmpDir,
		TrendTracking: TrendTrackingConfig{
			Enabled:       true,
			RetentionDays: 30,
		},
	}
	
	reporting := NewReportingSystem(config)
	
	// Create multiple reports to establish trends
	testFile := createTestFileWithIssues(t, "medium")
	
	// First report
	report1, err := reporting.ValidateAndReport([]string{testFile})
	if err != nil {
		t.Fatalf("First validation error: %v", err)
	}
	
	// Second report (simulate improvement)
	// For this test, we'll manually manipulate the trends
	reporting.trends.DataPoints = append(reporting.trends.DataPoints, QualityDataPoint{
		Timestamp:    time.Now().Add(-time.Hour),
		OverallScore: 60,
		TotalIssues:  20,
	})
	
	report2, err := reporting.ValidateAndReport([]string{testFile})
	if err != nil {
		t.Fatalf("Second validation error: %v", err)
	}
	
	// Verify trends are tracked
	if len(reporting.trends.DataPoints) < 2 {
		t.Error("Expected at least 2 trend data points")
	}
	
	// Verify trend analysis
	if report2.Trends == nil {
		t.Error("Expected trends to be included in report")
	}
	
	if report2.Trends.Summary.ScoreTrend == "" {
		t.Error("Expected score trend to be analyzed")
	}
}

func TestReportingSystem_QualityGrade(t *testing.T) {
	config := GetDefaultConfig()
	reporting := NewReportingSystem(config)
	
	tests := []struct {
		score    float64
		expected string
	}{
		{95, "A"},
		{85, "B"},
		{75, "C"},
		{65, "D"},
		{45, "F"},
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("score_%.0f", tt.score), func(t *testing.T) {
			grade := reporting.getQualityGrade(tt.score)
			if grade != tt.expected {
				t.Errorf("getQualityGrade(%.0f) = %s, expected %s", tt.score, grade, tt.expected)
			}
		})
	}
}

// Helper function to create test files with specific issue types
func createTestFileWithIssues(t *testing.T, issueType string) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	
	var code string
	switch issueType {
	case "critical":
		code = `package main
func badFunction() {
	db.Query("SELECT * FROM users WHERE id = " + userID)
	password := "hardcoded_password"
}`
	case "high":
		code = `package main
func handler(w http.ResponseWriter, r *http.Request) {
	userID := r.FormValue("user_id")
	processUser(userID)
}`
	case "medium":
		code = `package main
func buildString() string {
	result := ""
	for i := 0; i < 100; i++ {
		result += fmt.Sprintf("item %d", i)
	}
	return result
}`
	default:
		code = `package main
func goodFunction() {
	return "no issues"
}`
	}
	
	err := os.WriteFile(tmpFile, []byte(code), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	return tmpFile
}

// Benchmark tests
func BenchmarkReportingSystem_ValidateAndReport(b *testing.B) {
	tmpDir := b.TempDir()
	config := ReportingConfig{
		OutputDir: tmpDir,
		TrendTracking: TrendTrackingConfig{
			Enabled: false, // Disable for benchmarking
		},
	}
	
	reporting := NewReportingSystem(config)
	
	// Create test files
	testFiles := make([]string, 10)
	for i := 0; i < 10; i++ {
		testFiles[i] = createTestFileWithIssues(b, "medium")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reporting.ValidateAndReport(testFiles)
		if err != nil {
			b.Fatalf("ValidateAndReport() error = %v", err)
		}
	}
}

func BenchmarkReportingSystem_ExportReport(b *testing.B) {
	config := GetDefaultConfig()
	reporting := NewReportingSystem(config)
	
	// Create large sample report
	report := &AggregatedReport{
		Summary: AggregationSummary{
			TotalFiles:   100,
			TotalIssues:  500,
			OverallScore: 75,
		},
		FileReports: make([]ValidationReport, 100),
	}
	
	// Populate with sample data
	for i := 0; i < 100; i++ {
		report.FileReports[i] = ValidationReport{
			FilePath: fmt.Sprintf("file%d.go", i),
			Results: []ValidationResult{
				{
					Severity: SeverityMedium,
					Category: "performance",
					Message:  "Sample issue",
					Line:     10,
					RuleName: "sample-rule",
				},
			},
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := reporting.ExportReport(report, "json", &buf)
		if err != nil {
			b.Fatalf("ExportReport() error = %v", err)
		}
	}
}