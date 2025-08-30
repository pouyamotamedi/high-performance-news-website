package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"high-performance-news-website/internal/validation"
)

func main() {
	var (
		files       = flag.String("files", "", "Comma-separated list of Go files to validate")
		dir         = flag.String("dir", "", "Directory to scan for Go files")
		output      = flag.String("output", "./validation-reports", "Output directory for reports")
		format      = flag.String("format", "json", "Export format: json, csv, junit, html")
		cicd        = flag.Bool("cicd", false, "Run in CI/CD mode")
		dashboard   = flag.Bool("dashboard", false, "Start dashboard server")
		port        = flag.Int("port", 8080, "Dashboard server port")
		configFile  = flag.String("config", "", "Configuration file path")
		verbose     = flag.Bool("verbose", false, "Verbose output")
		help        = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Load configuration
	config := loadConfig(*configFile)
	if *output != "./validation-reports" {
		config.OutputDir = *output
	}

	// Create reporting system
	reporting := validation.NewReportingSystem(config)

	if *dashboard {
		startDashboard(reporting, *port)
		return
	}

	// Collect files to validate
	filePaths, err := collectFiles(*files, *dir)
	if err != nil {
		log.Fatalf("Error collecting files: %v", err)
	}

	if len(filePaths) == 0 {
		log.Fatal("No files specified for validation")
	}

	if *verbose {
		fmt.Printf("Validating %d files...\n", len(filePaths))
	}

	if *cicd {
		runCICDMode(reporting, filePaths, *verbose)
	} else {
		runStandardMode(reporting, filePaths, *format, *verbose)
	}
}

func showHelp() {
	fmt.Println(`AI Code Validator - Comprehensive code quality analysis for AI-generated code

Usage:
  ai-validator [options]

Options:
  -files string     Comma-separated list of Go files to validate
  -dir string       Directory to scan for Go files (recursive)
  -output string    Output directory for reports (default: ./validation-reports)
  -format string    Export format: json, csv, junit, html (default: json)
  -cicd            Run in CI/CD mode with exit codes
  -dashboard       Start interactive dashboard server
  -port int        Dashboard server port (default: 8080)
  -config string   Configuration file path
  -verbose         Verbose output
  -help            Show this help

Examples:
  # Validate specific files
  ai-validator -files "main.go,handler.go" -format html

  # Validate entire directory
  ai-validator -dir ./src -output ./reports

  # Run in CI/CD mode
  ai-validator -dir ./src -cicd

  # Start dashboard
  ai-validator -dashboard -port 9090

  # Use custom configuration
  ai-validator -config ./validation-config.json -dir ./src`)
}

func loadConfig(configFile string) validation.ReportingConfig {
	if configFile == "" {
		return validation.GetDefaultConfig()
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("Warning: Could not read config file %s, using defaults: %v", configFile, err)
		return validation.GetDefaultConfig()
	}

	var config validation.ReportingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Warning: Could not parse config file %s, using defaults: %v", configFile, err)
		return validation.GetDefaultConfig()
	}

	return config
}

func collectFiles(files, dir string) ([]string, error) {
	var filePaths []string

	// Add individual files
	if files != "" {
		for _, file := range strings.Split(files, ",") {
			file = strings.TrimSpace(file)
			if file != "" {
				filePaths = append(filePaths, file)
			}
		}
	}

	// Add files from directory
	if dir != "" {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && strings.HasSuffix(path, ".go") {
				// Skip test files and vendor directories
				if !strings.HasSuffix(path, "_test.go") && !strings.Contains(path, "vendor/") {
					filePaths = append(filePaths, path)
				}
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %w", dir, err)
		}
	}

	return filePaths, nil
}

func runStandardMode(reporting *validation.ReportingSystem, filePaths []string, format string, verbose bool) {
	// Run validation
	report, err := reporting.ValidateAndReport(filePaths)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// Print summary
	printSummary(report, verbose)

	// Export in requested format
	if format != "json" { // JSON is already saved by default
		outputPath := filepath.Join(reporting.GetOutputDir(), fmt.Sprintf("validation-report.%s", getFileExtension(format)))
		
		file, err := os.Create(outputPath)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer file.Close()

		if err := reporting.ExportReport(report, format, file); err != nil {
			log.Fatalf("Failed to export report: %v", err)
		}

		fmt.Printf("Report exported to: %s\n", outputPath)
	}

	// Exit with appropriate code
	if report.ShouldBlock {
		fmt.Println("⚠️  Quality issues found that may require attention")
		os.Exit(1)
	} else {
		fmt.Println("✅ Code quality validation passed")
	}
}

func runCICDMode(reporting *validation.ReportingSystem, filePaths []string, verbose bool) {
	// Setup CI/CD integration
	cicdConfig := validation.GetDefaultCICDConfig()
	cicdConfig.ArtifactPaths = []string{reporting.GetOutputDir()}
	
	cicd := validation.NewCICDIntegration(reporting, cicdConfig)

	// Run validation
	result, err := cicd.RunValidation(filePaths)
	if err != nil {
		log.Fatalf("CI/CD validation failed: %v", err)
	}

	// Output results
	fmt.Println(result.Summary)

	if verbose {
		fmt.Printf("\nReport files created:\n")
		for _, path := range result.ReportPaths {
			fmt.Printf("  - %s\n", path)
		}
	}

	// Output CI/CD annotations (GitHub Actions format)
	cicd.OutputGitHubActions(result)

	// Exit with appropriate code
	os.Exit(result.ExitCode)
}

func startDashboard(reporting *validation.ReportingSystem, port int) {
	dashboardConfig := validation.DashboardConfig{
		Enabled:     true,
		Port:        port,
		Title:       "AI Code Quality Dashboard",
		RefreshRate: 30,
	}

	dashboard := validation.NewDashboardServer(dashboardConfig, reporting)

	fmt.Printf("Starting AI Code Quality Dashboard on http://localhost:%d\n", port)
	fmt.Println("Press Ctrl+C to stop")

	if err := dashboard.Start(); err != nil {
		log.Fatalf("Dashboard server failed: %v", err)
	}
}

func printSummary(report *validation.AggregatedReport, verbose bool) {
	fmt.Printf("\n🔍 AI Code Quality Report\n")
	fmt.Printf("========================\n\n")

	// Overall score
	grade := report.Summary.QualityGrade
	gradeEmoji := getGradeEmoji(grade)
	fmt.Printf("Overall Score: %.1f%% %s (Grade: %s)\n", 
		report.Summary.OverallScore, gradeEmoji, grade)

	// File statistics
	fmt.Printf("Files Analyzed: %d\n", report.Summary.TotalFiles)
	fmt.Printf("Files with Issues: %d\n", report.Summary.FilesWithIssues)
	fmt.Printf("Total Issues: %d\n\n", report.Summary.TotalIssues)

	// Issue breakdown
	fmt.Printf("Issue Breakdown:\n")
	fmt.Printf("  🔴 Critical: %d\n", report.Summary.CriticalIssues)
	fmt.Printf("  🟠 High: %d\n", report.Summary.HighIssues)
	fmt.Printf("  🟡 Medium: %d\n", report.Summary.MediumIssues)
	fmt.Printf("  🟢 Low: %d\n", report.Summary.LowIssues)
	fmt.Printf("  👁️  Manual Review: %d\n\n", report.Summary.ManualReview)

	// Blocking status
	if report.ShouldBlock {
		fmt.Printf("🚫 Deployment Status: BLOCKED\n")
		fmt.Printf("Blocking Reasons:\n")
		for _, reason := range report.BlockingReasons {
			fmt.Printf("  - %s\n", reason)
		}
		fmt.Println()
	} else {
		fmt.Printf("✅ Deployment Status: APPROVED\n\n")
	}

	if verbose {
		// Top categories
		if len(report.CategoryAnalysis.Categories) > 0 {
			fmt.Printf("Top Issue Categories:\n")
			for i, category := range report.CategoryAnalysis.Categories {
				if i >= 5 {
					break
				}
				fmt.Printf("  %d. %s: %d issues (%.1f%%)\n", 
					i+1, category.Name, category.Count, category.Percentage)
			}
			fmt.Println()
		}

		// Top rules
		if len(report.RuleAnalysis.Rules) > 0 {
			fmt.Printf("Most Common Issues:\n")
			for i, rule := range report.RuleAnalysis.Rules {
				if i >= 5 {
					break
				}
				fmt.Printf("  %d. %s: %d occurrences (%s)\n", 
					i+1, rule.Name, rule.Count, rule.Severity)
			}
			fmt.Println()
		}
	}

	// Key recommendations
	if len(report.Recommendations) > 0 {
		fmt.Printf("🎯 Key Recommendations:\n")
		for i, rec := range report.Recommendations {
			if i >= 3 {
				break
			}
			priority := getPriorityEmoji(rec.Priority)
			fmt.Printf("  %s %s: %s\n", priority, rec.Title, rec.Description)
		}
		fmt.Println()
	}

	// Report location
	fmt.Printf("📊 Detailed reports saved to: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
}

func getGradeEmoji(grade string) string {
	switch grade {
	case "A":
		return "🌟"
	case "B":
		return "👍"
	case "C":
		return "👌"
	case "D":
		return "⚠️"
	case "F":
		return "❌"
	default:
		return "❓"
	}
}

func getPriorityEmoji(priority string) string {
	switch priority {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🔵"
	default:
		return "⚪"
	}
}

func getFileExtension(format string) string {
	switch format {
	case "csv":
		return "csv"
	case "junit":
		return "xml"
	case "html":
		return "html"
	default:
		return "json"
	}
}

// GetOutputDir returns the output directory (helper for testing)
func (r *validation.ReportingSystem) GetOutputDir() string {
	return r.config.OutputDir
}