package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/validation"
	"high-performance-news-website/pkg/database"
)

func main() {
	var (
		configPath   = flag.String("config", "configs/config.yaml", "Path to configuration file")
		sampleSize   = flag.Int("sample-size", 1000, "Number of articles to sample for consistency checking")
		outputFormat = flag.String("output", "json", "Output format: json, table, summary")
		executeRemed = flag.Bool("execute-remediation", false, "Execute high-confidence remediation suggestions automatically")
		scheduleMode = flag.Bool("schedule", false, "Run in scheduled mode (starts scheduler)")
		verbose      = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup database connection
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create consistency checker and reporter
	checker := validation.NewConsistencyChecker(db)
	reporter := validation.NewConsistencyReporter()
	reporter.SetDatabase(db)

	ctx := context.Background()

	if *scheduleMode {
		runScheduledMode(ctx, db, checker, reporter, *verbose)
	} else {
		runSingleCheck(ctx, checker, reporter, *sampleSize, *outputFormat, *executeRemed, *verbose)
	}
}

func runSingleCheck(ctx context.Context, checker *validation.ConsistencyChecker, reporter *validation.ConsistencyReporter, 
	sampleSize int, outputFormat string, executeRemediation bool, verbose bool) {
	
	if verbose {
		log.Printf("Starting consistency check with sample size: %d", sampleSize)
	}

	start := time.Now()

	// Run consistency check
	check, err := checker.ValidateDataConsistency(ctx)
	if err != nil {
		log.Fatalf("Consistency check failed: %v", err)
	}

	duration := time.Since(start)

	if verbose {
		log.Printf("Consistency check completed in %v", duration)
		log.Printf("Found %d issues", len(check.Issues))
	}

	// Process issues through reporter
	if len(check.Issues) > 0 {
		if err := reporter.ProcessIssues(ctx, check.Issues); err != nil {
			log.Printf("Warning: Failed to process issues: %v", err)
		}

		// Execute high-confidence remediation if requested
		if executeRemediation {
			executeHighConfidenceRemediation(ctx, reporter, check.Issues, verbose)
		}
	}

	// Output results
	switch outputFormat {
	case "json":
		outputJSON(check)
	case "table":
		outputTable(check)
	case "summary":
		outputSummary(check)
	default:
		log.Fatalf("Unknown output format: %s", outputFormat)
	}

	// Exit with appropriate code
	if check.Status == validation.CheckStatusFailed {
		os.Exit(1)
	} else if check.Status == validation.CheckStatusWarning {
		os.Exit(2)
	}
}

func runScheduledMode(ctx context.Context, db *database.DB, checker *validation.ConsistencyChecker, 
	reporter *validation.ConsistencyReporter, verbose bool) {
	
	log.Printf("Starting consistency check scheduler...")

	// Create and start scheduler
	scheduler := validation.NewCheckScheduler()
	scheduler.SetDependencies(db, checker, reporter, nil) // No monitoring client for CLI

	if err := scheduler.Start(ctx); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	if verbose {
		schedules := scheduler.GetSchedules()
		log.Printf("Loaded %d schedules:", len(schedules))
		for _, schedule := range schedules {
			log.Printf("  - %s: %s (enabled: %t)", schedule.ID, schedule.Name, schedule.Enabled)
		}
	}

	// Wait for interrupt signal
	log.Printf("Scheduler running. Press Ctrl+C to stop.")
	select {
	case <-ctx.Done():
		log.Printf("Context cancelled, stopping scheduler...")
	}

	scheduler.Stop()
	log.Printf("Scheduler stopped.")
}

func executeHighConfidenceRemediation(ctx context.Context, reporter *validation.ConsistencyReporter, 
	issues []validation.ConsistencyIssue, verbose bool) {
	
	log.Printf("Executing high-confidence remediation suggestions...")

	executedCount := 0
	for _, issue := range issues {
		suggestions, err := reporter.GetRemediationSuggestions(ctx, issue.ID)
		if err != nil {
			if verbose {
				log.Printf("Failed to get suggestions for issue %s: %v", issue.ID, err)
			}
			continue
		}

		for _, suggestion := range suggestions {
			if suggestion.Confidence >= 0.9 { // Only execute very high confidence suggestions
				if err := reporter.ExecuteRemediation(ctx, suggestion.ID); err != nil {
					if verbose {
						log.Printf("Failed to execute remediation %s: %v", suggestion.ID, err)
					}
				} else {
					executedCount++
					if verbose {
						log.Printf("Executed remediation: %s", suggestion.Description)
					}
				}
			}
		}
	}

	log.Printf("Executed %d high-confidence remediation suggestions", executedCount)
}

func outputJSON(check *validation.ConsistencyCheck) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(check); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}
}

func outputTable(check *validation.ConsistencyCheck) {
	fmt.Printf("Consistency Check Results\n")
	fmt.Printf("========================\n")
	fmt.Printf("Check ID: %s\n", check.ID)
	fmt.Printf("Status: %s\n", check.Status)
	fmt.Printf("Duration: %v\n", check.Duration)
	fmt.Printf("Sample Size: %d\n", check.SampleSize)
	fmt.Printf("Issues Found: %d\n", len(check.Issues))
	fmt.Printf("\n")

	if len(check.Issues) > 0 {
		fmt.Printf("Issues:\n")
		fmt.Printf("-------\n")
		fmt.Printf("%-20s %-10s %-50s\n", "Type", "Severity", "Description")
		fmt.Printf("%-20s %-10s %-50s\n", "----", "--------", "-----------")

		for _, issue := range check.Issues {
			description := issue.Description
			if len(description) > 47 {
				description = description[:47] + "..."
			}
			fmt.Printf("%-20s %-10s %-50s\n", issue.Type, issue.Severity, description)
		}
	}
}

func outputSummary(check *validation.ConsistencyCheck) {
	fmt.Printf("Consistency Check Summary\n")
	fmt.Printf("========================\n")
	fmt.Printf("Status: %s\n", check.Status)
	fmt.Printf("Duration: %v\n", check.Duration)
	fmt.Printf("Sample Size: %d\n", check.SampleSize)
	fmt.Printf("Total Issues: %d\n", len(check.Issues))

	// Count issues by severity
	severityCounts := make(map[string]int)
	typeCounts := make(map[string]int)

	for _, issue := range check.Issues {
		severityCounts[issue.Severity]++
		typeCounts[issue.Type]++
	}

	if len(severityCounts) > 0 {
		fmt.Printf("\nIssues by Severity:\n")
		for severity, count := range severityCounts {
			fmt.Printf("  %s: %d\n", severity, count)
		}
	}

	if len(typeCounts) > 0 {
		fmt.Printf("\nTop Issue Types:\n")
		// Sort by count (simplified)
		for issueType, count := range typeCounts {
			if count >= 5 { // Only show types with 5+ occurrences
				fmt.Printf("  %s: %d\n", issueType, count)
			}
		}
	}

	// Show metadata if available
	if check.Metadata != nil {
		fmt.Printf("\nCheck Metadata:\n")
		if actualSize, ok := check.Metadata["actual_sample_size"]; ok {
			fmt.Printf("  Actual Sample Size: %v\n", actualSize)
		}
		if refIssues, ok := check.Metadata["referential_issues"]; ok {
			fmt.Printf("  Referential Issues: %v\n", refIssues)
		}
		if multiIssues, ok := check.Metadata["multilingual_issues"]; ok {
			fmt.Printf("  Multilingual Issues: %v\n", multiIssues)
		}
		if seoIssues, ok := check.Metadata["seo_issues"]; ok {
			fmt.Printf("  SEO Issues: %v\n", seoIssues)
		}
	}

	// Provide recommendations
	fmt.Printf("\nRecommendations:\n")
	if check.Status == validation.CheckStatusFailed {
		fmt.Printf("  - Address high-severity issues immediately\n")
		fmt.Printf("  - Review manual remediation queue\n")
	} else if check.Status == validation.CheckStatusWarning {
		fmt.Printf("  - Review and fix medium-severity issues\n")
		fmt.Printf("  - Consider running automated remediation\n")
	} else {
		fmt.Printf("  - No immediate action required\n")
		fmt.Printf("  - Continue regular monitoring\n")
	}
}