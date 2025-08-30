package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CICDIntegration provides CI/CD pipeline integration
type CICDIntegration struct {
	reporting *ReportingSystem
	config    CICDConfig
}

// CICDConfig contains CI/CD integration configuration
type CICDConfig struct {
	Enabled           bool     `json:"enabled"`
	FailOnBlock       bool     `json:"fail_on_block"`
	OutputFormats     []string `json:"output_formats"`     // json, junit, csv
	ArtifactPaths     []string `json:"artifact_paths"`     // paths to save reports
	AnnotationSupport bool     `json:"annotation_support"` // GitHub/GitLab annotations
	CommentSupport    bool     `json:"comment_support"`    // PR/MR comments
}

// CICDResult contains the result of CI/CD validation
type CICDResult struct {
	Success         bool     `json:"success"`
	ExitCode        int      `json:"exit_code"`
	Message         string   `json:"message"`
	ReportPaths     []string `json:"report_paths"`
	Annotations     []Annotation `json:"annotations,omitempty"`
	Summary         string   `json:"summary"`
	BlockingReasons []string `json:"blocking_reasons,omitempty"`
}

// Annotation represents a CI/CD annotation (GitHub Actions, GitLab CI, etc.)
type Annotation struct {
	Level    string `json:"level"`    // notice, warning, error
	Title    string `json:"title"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

// NewCICDIntegration creates a new CI/CD integration
func NewCICDIntegration(reporting *ReportingSystem, config CICDConfig) *CICDIntegration {
	return &CICDIntegration{
		reporting: reporting,
		config:    config,
	}
}

// RunValidation runs validation for CI/CD pipeline
func (c *CICDIntegration) RunValidation(filePaths []string) (*CICDResult, error) {
	if !c.config.Enabled {
		return &CICDResult{
			Success:  true,
			ExitCode: 0,
			Message:  "CI/CD validation disabled",
		}, nil
	}

	// Run validation
	report, err := c.reporting.ValidateAndReport(filePaths)
	if err != nil {
		return &CICDResult{
			Success:  false,
			ExitCode: 2,
			Message:  fmt.Sprintf("Validation failed: %v", err),
		}, err
	}

	result := &CICDResult{
		Success:         !report.ShouldBlock || !c.config.FailOnBlock,
		BlockingReasons: report.BlockingReasons,
		Summary:         c.generateSummary(report),
	}

	// Set exit code
	if report.ShouldBlock && c.config.FailOnBlock {
		result.ExitCode = 1
		result.Message = "Code quality checks failed - deployment blocked"
	} else if report.ShouldBlock {
		result.ExitCode = 0
		result.Message = "Code quality issues found but not blocking deployment"
	} else {
		result.ExitCode = 0
		result.Message = "Code quality checks passed"
	}

	// Generate annotations if supported
	if c.config.AnnotationSupport {
		result.Annotations = c.generateAnnotations(report)
	}

	// Save reports in requested formats
	reportPaths, err := c.saveReports(report)
	if err != nil {
		return result, fmt.Errorf("failed to save reports: %w", err)
	}
	result.ReportPaths = reportPaths

	return result, nil
}

// generateSummary creates a summary for CI/CD output
func (c *CICDIntegration) generateSummary(report *AggregatedReport) string {
	var summary strings.Builder
	
	summary.WriteString(fmt.Sprintf("## AI Code Quality Report\n\n"))
	summary.WriteString(fmt.Sprintf("**Overall Score:** %.1f%% (Grade: %s)\n", 
		report.Summary.OverallScore, report.Summary.QualityGrade))
	summary.WriteString(fmt.Sprintf("**Files Analyzed:** %d\n", report.Summary.TotalFiles))
	summary.WriteString(fmt.Sprintf("**Files with Issues:** %d\n", report.Summary.FilesWithIssues))
	summary.WriteString(fmt.Sprintf("**Total Issues:** %d\n\n", report.Summary.TotalIssues))
	
	// Issue breakdown
	summary.WriteString("### Issue Breakdown\n")
	summary.WriteString(fmt.Sprintf("- 🔴 Critical: %d\n", report.Summary.CriticalIssues))
	summary.WriteString(fmt.Sprintf("- 🟠 High: %d\n", report.Summary.HighIssues))
	summary.WriteString(fmt.Sprintf("- 🟡 Medium: %d\n", report.Summary.MediumIssues))
	summary.WriteString(fmt.Sprintf("- 🟢 Low: %d\n", report.Summary.LowIssues))
	summary.WriteString(fmt.Sprintf("- 👁️ Manual Review: %d\n\n", report.Summary.ManualReview))
	
	// Blocking status
	if report.ShouldBlock {
		summary.WriteString("### ⚠️ Deployment Status: BLOCKED\n")
		summary.WriteString("**Blocking Reasons:**\n")
		for _, reason := range report.BlockingReasons {
			summary.WriteString(fmt.Sprintf("- %s\n", reason))
		}
		summary.WriteString("\n")
	} else {
		summary.WriteString("### ✅ Deployment Status: APPROVED\n\n")
	}
	
	// Top categories
	if len(report.CategoryAnalysis.Categories) > 0 {
		summary.WriteString("### Top Issue Categories\n")
		for i, category := range report.CategoryAnalysis.Categories {
			if i >= 5 { // Show top 5
				break
			}
			summary.WriteString(fmt.Sprintf("- **%s**: %d issues (%.1f%%)\n", 
				category.Name, category.Count, category.Percentage))
		}
		summary.WriteString("\n")
	}
	
	// Recommendations
	if len(report.Recommendations) > 0 {
		summary.WriteString("### 🎯 Key Recommendations\n")
		for i, rec := range report.Recommendations {
			if i >= 3 { // Show top 3
				break
			}
			priority := "🔵"
			if rec.Priority == "high" {
				priority = "🔴"
			} else if rec.Priority == "medium" {
				priority = "🟡"
			}
			summary.WriteString(fmt.Sprintf("- %s **%s**: %s\n", priority, rec.Title, rec.Description))
		}
	}
	
	return summary.String()
}

// generateAnnotations creates CI/CD annotations from validation results
func (c *CICDIntegration) generateAnnotations(report *AggregatedReport) []Annotation {
	var annotations []Annotation
	
	// Add summary annotation
	level := "notice"
	if report.Summary.CriticalIssues > 0 {
		level = "error"
	} else if report.Summary.HighIssues > 0 {
		level = "warning"
	}
	
	annotations = append(annotations, Annotation{
		Level:   level,
		Title:   fmt.Sprintf("Code Quality Score: %.1f%%", report.Summary.OverallScore),
		Message: fmt.Sprintf("Found %d issues across %d files", report.Summary.TotalIssues, report.Summary.TotalFiles),
	})
	
	// Add file-specific annotations (limit to prevent spam)
	annotationCount := 0
	maxAnnotations := 50
	
	for _, fileReport := range report.FileReports {
		for _, result := range fileReport.Results {
			if annotationCount >= maxAnnotations {
				annotations = append(annotations, Annotation{
					Level:   "notice",
					Title:   "Additional Issues",
					Message: fmt.Sprintf("... and %d more issues. See full report for details.", 
						report.Summary.TotalIssues-maxAnnotations),
				})
				break
			}
			
			annotationLevel := "notice"
			switch result.Severity {
			case SeverityCritical:
				annotationLevel = "error"
			case SeverityHigh:
				annotationLevel = "warning"
			case SeverityMedium:
				annotationLevel = "warning"
			}
			
			annotations = append(annotations, Annotation{
				Level:   annotationLevel,
				Title:   fmt.Sprintf("[%s] %s", result.Severity, result.RuleName),
				Message: fmt.Sprintf("%s\n\nSuggestion: %s", result.Message, result.Suggestion),
				File:    fileReport.FilePath,
				Line:    result.Line,
				Column:  result.Column,
			})
			
			annotationCount++
		}
		
		if annotationCount >= maxAnnotations {
			break
		}
	}
	
	return annotations
}

// saveReports saves reports in requested formats
func (c *CICDIntegration) saveReports(report *AggregatedReport) ([]string, error) {
	var reportPaths []string
	
	for _, format := range c.config.OutputFormats {
		for _, artifactPath := range c.config.ArtifactPaths {
			// Ensure directory exists
			if err := os.MkdirAll(artifactPath, 0755); err != nil {
				return reportPaths, fmt.Errorf("failed to create artifact directory %s: %w", artifactPath, err)
			}
			
			// Generate filename
			var filename string
			switch format {
			case "json":
				filename = "validation-report.json"
			case "junit":
				filename = "validation-report.xml"
			case "csv":
				filename = "validation-report.csv"
			default:
				continue
			}
			
			filePath := filepath.Join(artifactPath, filename)
			
			// Create file
			file, err := os.Create(filePath)
			if err != nil {
				return reportPaths, fmt.Errorf("failed to create report file %s: %w", filePath, err)
			}
			
			// Export report
			if err := c.reporting.ExportReport(report, format, file); err != nil {
				file.Close()
				return reportPaths, fmt.Errorf("failed to export report to %s: %w", filePath, err)
			}
			
			file.Close()
			reportPaths = append(reportPaths, filePath)
		}
	}
	
	return reportPaths, nil
}

// OutputGitHubActions outputs results in GitHub Actions format
func (c *CICDIntegration) OutputGitHubActions(result *CICDResult) {
	if !c.config.AnnotationSupport {
		return
	}
	
	// Set outputs
	fmt.Printf("::set-output name=success::%t\n", result.Success)
	fmt.Printf("::set-output name=exit-code::%d\n", result.ExitCode)
	fmt.Printf("::set-output name=summary::%s\n", strings.ReplaceAll(result.Summary, "\n", "\\n"))
	
	// Output annotations
	for _, annotation := range result.Annotations {
		if annotation.File != "" {
			fmt.Printf("::%s file=%s,line=%d,col=%d,title=%s::%s\n",
				annotation.Level, annotation.File, annotation.Line, annotation.Column,
				annotation.Title, annotation.Message)
		} else {
			fmt.Printf("::%s title=%s::%s\n", annotation.Level, annotation.Title, annotation.Message)
		}
	}
}

// OutputGitLabCI outputs results in GitLab CI format
func (c *CICDIntegration) OutputGitLabCI(result *CICDResult) {
	// GitLab CI uses different annotation format
	if !c.config.AnnotationSupport {
		return
	}
	
	// Create GitLab code quality report
	gitlabReport := struct {
		Version     string                 `json:"version"`
		Fingerprint string                 `json:"fingerprint"`
		Issues      []GitLabCodeQualityIssue `json:"issues"`
	}{
		Version:     "14.0.0",
		Fingerprint: fmt.Sprintf("%d", len(result.Annotations)),
		Issues:      make([]GitLabCodeQualityIssue, 0),
	}
	
	for _, annotation := range result.Annotations {
		if annotation.File == "" {
			continue
		}
		
		severity := "info"
		switch annotation.Level {
		case "error":
			severity = "critical"
		case "warning":
			severity = "major"
		}
		
		gitlabReport.Issues = append(gitlabReport.Issues, GitLabCodeQualityIssue{
			Description: annotation.Message,
			Fingerprint: fmt.Sprintf("%s-%s-%d", annotation.File, annotation.Title, annotation.Line),
			Severity:    severity,
			Location: GitLabLocation{
				Path: annotation.File,
				Lines: GitLabLines{
					Begin: annotation.Line,
				},
			},
		})
	}
	
	// Save GitLab code quality report
	for _, artifactPath := range c.config.ArtifactPaths {
		reportPath := filepath.Join(artifactPath, "gl-code-quality-report.json")
		file, err := os.Create(reportPath)
		if err != nil {
			continue
		}
		
		json.NewEncoder(file).Encode(gitlabReport)
		file.Close()
	}
}

// GitLabCodeQualityIssue represents a GitLab code quality issue
type GitLabCodeQualityIssue struct {
	Description string        `json:"description"`
	Fingerprint string        `json:"fingerprint"`
	Severity    string        `json:"severity"`
	Location    GitLabLocation `json:"location"`
}

// GitLabLocation represents the location of an issue
type GitLabLocation struct {
	Path  string      `json:"path"`
	Lines GitLabLines `json:"lines"`
}

// GitLabLines represents line information
type GitLabLines struct {
	Begin int `json:"begin"`
}

// GetDefaultCICDConfig returns default CI/CD configuration
func GetDefaultCICDConfig() CICDConfig {
	return CICDConfig{
		Enabled:           true,
		FailOnBlock:       true,
		OutputFormats:     []string{"json", "junit"},
		ArtifactPaths:     []string{"./artifacts", "./reports"},
		AnnotationSupport: true,
		CommentSupport:    false,
	}
}