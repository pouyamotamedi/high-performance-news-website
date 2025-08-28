package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// CanonicalExample demonstrates the usage of the canonicalization system
func CanonicalExample(db *sql.DB) {
	// Initialize the canonical manager
	cm := NewCanonicalManager(db)

	// Example user ID (admin)
	userID := uint64(1)

	fmt.Println("=== Canonicalization System Example ===")

	// Example 1: Schedule a canonical job for a tag with 48-hour delay
	fmt.Println("\n1. Scheduling canonical job for tag with 48-hour delay...")
	tagTarget := NewTagTarget(1) // Assuming tag ID 1 exists
	jobID1, err := cm.ScheduleCanonicalJob(123, tagTarget, &userID, false)
	if err != nil {
		log.Printf("Error scheduling tag job: %v", err)
	} else {
		fmt.Printf("   Scheduled job ID: %d\n", jobID1)
	}

	// Example 2: Schedule a canonical job for a category with admin override
	fmt.Println("\n2. Scheduling canonical job for category with admin override...")
	categoryTarget := NewCategoryTarget(1) // Assuming category ID 1 exists
	jobID2, err := cm.ScheduleCanonicalJob(124, categoryTarget, &userID, true)
	if err != nil {
		log.Printf("Error scheduling category job: %v", err)
	} else {
		fmt.Printf("   Scheduled job ID: %d (immediate processing)\n", jobID2)
	}

	// Example 3: Schedule a canonical job for a custom URL
	fmt.Println("\n3. Scheduling canonical job for custom URL...")
	urlTarget := NewURLTarget("/special/landing-page")
	jobID3, err := cm.ScheduleCanonicalJob(125, urlTarget, &userID, false)
	if err != nil {
		log.Printf("Error scheduling URL job: %v", err)
	} else {
		fmt.Printf("   Scheduled job ID: %d\n", jobID3)
	}

	// Example 4: Generate canonical URL for preview
	fmt.Println("\n4. Generating canonical URL for preview...")
	previewTarget := NewTagTarget(1)
	canonicalURL, err := cm.GenerateCanonicalURL(previewTarget)
	if err != nil {
		log.Printf("Error generating canonical URL: %v", err)
	} else {
		fmt.Printf("   Generated URL: %s\n", canonicalURL)
	}

	// Example 5: Get pending jobs
	fmt.Println("\n5. Getting pending jobs...")
	pendingJobs, err := cm.GetPendingJobs()
	if err != nil {
		log.Printf("Error getting pending jobs: %v", err)
	} else {
		fmt.Printf("   Found %d pending jobs\n", len(pendingJobs))
		for _, job := range pendingJobs {
			fmt.Printf("   - Job %d: Article %d -> %s (admin_override: %t)\n",
				job.ID, job.ArticleID, job.TargetType, job.AdminOverride)
		}
	}

	// Example 6: Process pending jobs manually
	fmt.Println("\n6. Processing pending jobs manually...")
	processed, err := cm.ProcessPendingJobs(&userID)
	if err != nil {
		log.Printf("Error processing jobs: %v", err)
	} else {
		fmt.Printf("   Processed %d jobs\n", processed)
	}

	// Example 7: Get jobs for a specific article
	fmt.Println("\n7. Getting jobs for article 123...")
	articleJobs, err := cm.GetJobsByArticle(123)
	if err != nil {
		log.Printf("Error getting article jobs: %v", err)
	} else {
		fmt.Printf("   Found %d jobs for article 123\n", len(articleJobs))
		for _, job := range articleJobs {
			fmt.Printf("   - Job %d: %s -> %s (status: %s)\n",
				job.ID, job.TargetType, getTargetDescription(job), job.Status)
		}
	}

	// Example 8: Get job statistics
	fmt.Println("\n8. Getting job statistics...")
	stats, err := cm.GetJobStats()
	if err != nil {
		log.Printf("Error getting stats: %v", err)
	} else {
		fmt.Printf("   Job statistics (last 30 days):\n")
		for status, count := range stats {
			fmt.Printf("   - %s: %d\n", status, count)
		}
	}

	// Example 9: Start background processor (in production)
	fmt.Println("\n9. Starting background processor...")
	fmt.Printf("   Background processor would run every 5 minutes\n")
	fmt.Printf("   In production, call: cm.StartJobProcessor(5*time.Minute, &userID)\n")

	// Example 10: Cleanup old jobs (admin operation)
	fmt.Println("\n10. Cleaning up old jobs...")
	deletedCount, err := cm.CleanupOldJobs()
	if err != nil {
		log.Printf("Error cleaning up jobs: %v", err)
	} else {
		fmt.Printf("   Cleaned up %d old jobs\n", deletedCount)
	}

	fmt.Println("\n=== Example Complete ===")
}

// getTargetDescription returns a human-readable description of the job target
func getTargetDescription(job CanonicalJob) string {
	switch job.TargetType {
	case "tag":
		if job.TargetID != nil {
			return fmt.Sprintf("tag ID %d", *job.TargetID)
		}
		return "tag (unknown ID)"
	case "category":
		if job.TargetID != nil {
			return fmt.Sprintf("category ID %d", *job.TargetID)
		}
		return "category (unknown ID)"
	case "url":
		if job.TargetURL != nil {
			return fmt.Sprintf("URL %s", *job.TargetURL)
		}
		return "URL (unknown)"
	default:
		return "unknown target"
	}
}

// DemoCanonicalWorkflow demonstrates a complete canonicalization workflow
func DemoCanonicalWorkflow(db *sql.DB) {
	cm := NewCanonicalManager(db)
	userID := uint64(1)

	fmt.Println("=== Canonical Workflow Demo ===")

	// Step 1: Content creator publishes an article
	fmt.Println("\n1. Article published - scheduling canonicalization...")
	articleID := uint64(100)
	target := NewTagTarget(1) // Canonicalize to main tag

	jobID, err := cm.ScheduleCanonicalJob(articleID, target, &userID, false)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("   Job %d scheduled for 48-hour delay\n", jobID)

	// Step 2: Editor decides to override the delay
	fmt.Println("\n2. Editor overrides delay for immediate canonicalization...")
	adminJobID, err := cm.ScheduleCanonicalJob(articleID, target, &userID, true)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("   Admin job %d scheduled for immediate processing\n", adminJobID)

	// Step 3: Background processor runs (simulated)
	fmt.Println("\n3. Background processor running...")
	processed, err := cm.ProcessPendingJobs(&userID)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("   Processed %d jobs\n", processed)

	// Step 4: Check job status
	fmt.Println("\n4. Checking job status...")
	jobs, err := cm.GetJobsByArticle(articleID)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	for _, job := range jobs {
		fmt.Printf("   Job %d: %s (created: %s)\n",
			job.ID, job.Status, job.CreatedAt.Format(time.RFC3339))
		if job.ProcessedAt != nil {
			fmt.Printf("     Processed at: %s\n", job.ProcessedAt.Format(time.RFC3339))
		}
		if job.ErrorMessage != nil {
			fmt.Printf("     Error: %s\n", *job.ErrorMessage)
		}
	}

	fmt.Println("\n=== Workflow Demo Complete ===")
}

// DemoErrorHandling demonstrates error handling scenarios
func DemoErrorHandling(db *sql.DB) {
	cm := NewCanonicalManager(db)
	userID := uint64(1)

	fmt.Println("=== Error Handling Demo ===")

	// Example 1: Invalid tag ID
	fmt.Println("\n1. Testing invalid tag ID...")
	invalidTarget := NewTagTarget(99999) // Non-existent tag
	jobID, err := cm.ScheduleCanonicalJob(200, invalidTarget, &userID, true)
	if err != nil {
		fmt.Printf("   Expected error during scheduling: %v\n", err)
	} else {
		fmt.Printf("   Job %d scheduled (will fail during processing)\n", jobID)

		// Try to process the job
		success, err := cm.ProcessJob(jobID, &userID)
		if err != nil {
			fmt.Printf("   Processing error: %v\n", err)
		} else if !success {
			fmt.Printf("   Job processing failed as expected\n")
		}

		// Check job status
		jobs, _ := cm.GetJobsByArticle(200)
		for _, job := range jobs {
			if job.ID == jobID {
				fmt.Printf("   Job status: %s\n", job.Status)
				if job.ErrorMessage != nil {
					fmt.Printf("   Error message: %s\n", *job.ErrorMessage)
				}
				fmt.Printf("   Retry count: %d\n", job.RetryCount)
			}
		}

		// Retry the failed job
		fmt.Printf("   Retrying failed job...\n")
		err = cm.RetryFailedJob(jobID, false)
		if err != nil {
			fmt.Printf("   Retry error: %v\n", err)
		} else {
			fmt.Printf("   Job queued for retry\n")
		}
	}

	// Example 2: Invalid target type
	fmt.Println("\n2. Testing invalid target type...")
	invalidTypeTarget := CanonicalTarget{Type: "invalid"}
	_, err = cm.ScheduleCanonicalJob(201, invalidTypeTarget, &userID, false)
	if err != nil {
		fmt.Printf("   Expected validation error: %v\n", err)
	}

	// Example 3: Missing required fields
	fmt.Println("\n3. Testing missing required fields...")
	incompleteTarget := CanonicalTarget{Type: "tag"} // Missing ID
	_, err = cm.ScheduleCanonicalJob(202, incompleteTarget, &userID, false)
	if err != nil {
		fmt.Printf("   Expected validation error: %v\n", err)
	}

	fmt.Println("\n=== Error Handling Demo Complete ===")
}