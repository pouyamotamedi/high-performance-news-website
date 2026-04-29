package testing

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// RemediationEngine generates suggestions for fixing flaky tests
type RemediationEngine struct {
	db *sql.DB
}

// NewRemediationEngine creates a new remediation engine
func NewRemediationEngine() *RemediationEngine {
	return &RemediationEngine{}
}

// SetDatabase sets the database connection for the remediation engine
func (r *RemediationEngine) SetDatabase(db *sql.DB) {
	r.db = db
}

// GenerateRemediationSuggestions generates remediation suggestions for a flaky test
func (r *RemediationEngine) GenerateRemediationSuggestions(testName, testSuite string) ([]RemediationSuggestion, error) {
	var suggestions []RemediationSuggestion

	// Get failure patterns for the test
	patterns, err := r.getFailurePatterns(testName, testSuite)
	if err != nil {
		return nil, fmt.Errorf("failed to get failure patterns: %w", err)
	}

	// Get recent executions for analysis
	executions, err := r.getRecentExecutions(testName, testSuite, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent executions: %w", err)
	}

	// Generate suggestions based on patterns
	for _, pattern := range patterns {
		patternSuggestions := r.generateSuggestionsForPattern(pattern, executions)
		suggestions = append(suggestions, patternSuggestions...)
	}

	// Generate general suggestions based on execution analysis
	generalSuggestions := r.generateGeneralSuggestions(executions)
	suggestions = append(suggestions, generalSuggestions...)

	// Remove duplicates and sort by priority
	suggestions = r.deduplicateAndSort(suggestions)

	return suggestions, nil
}

// generateSuggestionsForPattern generates suggestions based on specific failure patterns
func (r *RemediationEngine) generateSuggestionsForPattern(pattern FailurePattern, executions []TestExecutionRecord) []RemediationSuggestion {
	var suggestions []RemediationSuggestion
	now := time.Now()

	switch pattern.Type {
	case "intermittent":
		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "timing",
			Priority:    "high",
			Description: "Intermittent failures often indicate race conditions or timing issues",
			Action:      "Add explicit waits, synchronization, or increase timeouts in test code",
			Confidence:  0.8,
			Examples: []string{
				"Replace Thread.sleep() with explicit waits for conditions",
				"Use WebDriverWait instead of implicit waits",
				"Add retry mechanisms for network operations",
				"Implement proper synchronization for async operations",
			},
			CreatedAt: now,
		})

		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "dependency",
			Priority:    "medium",
			Description: "Intermittent failures may be caused by external dependencies",
			Action:      "Mock external dependencies or add proper error handling",
			Confidence:  0.6,
			Examples: []string{
				"Mock HTTP calls to external services",
				"Use test doubles for database connections",
				"Implement circuit breaker patterns",
				"Add proper cleanup between test runs",
			},
			CreatedAt: now,
		})

	case "consecutive":
		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "environment",
			Priority:    "high",
			Description: "Consecutive failures suggest persistent environment issues",
			Action:      "Check test environment setup and resource availability",
			Confidence:  0.9,
			Examples: []string{
				"Verify database connection pool settings",
				"Check memory and CPU resource limits",
				"Ensure proper test data cleanup",
				"Validate test environment configuration",
			},
			CreatedAt: now,
		})

		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "resource",
			Priority:    "medium",
			Description: "Resource contention may cause consecutive failures",
			Action:      "Implement proper resource management and cleanup",
			Confidence:  0.7,
			Examples: []string{
				"Add proper connection pooling",
				"Implement resource cleanup in teardown methods",
				"Use isolated test environments",
				"Add resource monitoring and alerts",
			},
			CreatedAt: now,
		})

	case "environment":
		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "environment",
			Priority:    "high",
			Description: "Environment-specific failures indicate configuration issues",
			Action:      "Review and standardize environment configurations",
			Confidence:  0.9,
			Examples: []string{
				"Standardize environment variables across all environments",
				"Use infrastructure as code for consistent setup",
				"Implement environment health checks",
				"Add environment-specific test configurations",
			},
			CreatedAt: now,
		})

	case "timing":
		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "timing",
			Priority:    "high",
			Description: "Time-based failures suggest load or scheduling issues",
			Action:      "Investigate system load patterns and adjust test scheduling",
			Confidence:  0.8,
			Examples: []string{
				"Distribute test execution across different time periods",
				"Implement load balancing for test execution",
				"Add system resource monitoring during tests",
				"Use dedicated test environments during peak hours",
			},
			CreatedAt: now,
		})

	case "error_message":
		suggestions = append(suggestions, r.generateErrorMessageSuggestions(pattern, executions, now)...)
	}

	return suggestions
}

// generateErrorMessageSuggestions generates suggestions based on error message patterns
func (r *RemediationEngine) generateErrorMessageSuggestions(pattern FailurePattern, executions []TestExecutionRecord, now time.Time) []RemediationSuggestion {
	var suggestions []RemediationSuggestion

	errorMsg := strings.ToLower(pattern.Description)

	// Analyze common error patterns
	if strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "timed out") {
		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "timing",
			Priority:    "high",
			Description: "Timeout errors indicate insufficient wait times or slow operations",
			Action:      "Increase timeout values or optimize slow operations",
			Confidence:  0.9,
			Examples: []string{
				"Increase HTTP request timeouts",
				"Optimize database queries",
				"Use connection pooling",
				"Implement progressive timeout strategies",
			},
			CreatedAt: now,
		})
	}

	if strings.Contains(errorMsg, "connection") || strings.Contains(errorMsg, "network") {
		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "dependency",
			Priority:    "high",
			Description: "Connection errors suggest network or service availability issues",
			Action:      "Implement connection retry logic and health checks",
			Confidence:  0.8,
			Examples: []string{
				"Add exponential backoff for connection retries",
				"Implement health checks for external services",
				"Use connection pooling with proper validation",
				"Add circuit breaker patterns for external calls",
			},
			CreatedAt: now,
		})
	}

	if strings.Contains(errorMsg, "memory") || strings.Contains(errorMsg, "out of memory") {
		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "resource",
			Priority:    "high",
			Description: "Memory errors indicate resource management issues",
			Action:      "Optimize memory usage and implement proper cleanup",
			Confidence:  0.9,
			Examples: []string{
				"Implement proper resource cleanup in test teardown",
				"Reduce test data size or use streaming",
				"Add memory monitoring and limits",
				"Use memory-efficient data structures",
			},
			CreatedAt: now,
		})
	}

	if strings.Contains(errorMsg, "deadlock") || strings.Contains(errorMsg, "lock") {
		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "timing",
			Priority:    "high",
			Description: "Deadlock errors indicate concurrency issues",
			Action:      "Review and fix concurrent access patterns",
			Confidence:  0.9,
			Examples: []string{
				"Implement proper locking order",
				"Use timeout-based locks",
				"Reduce lock scope and duration",
				"Consider lock-free data structures",
			},
			CreatedAt: now,
		})
	}

	if strings.Contains(errorMsg, "assertion") || strings.Contains(errorMsg, "expected") {
		suggestions = append(suggestions, RemediationSuggestion{
			Type:        "timing",
			Priority:    "medium",
			Description: "Assertion failures may indicate timing or state issues",
			Action:      "Add proper state verification and wait conditions",
			Confidence:  0.6,
			Examples: []string{
				"Use eventually consistent assertions",
				"Add explicit waits for state changes",
				"Implement proper test data setup",
				"Use more robust assertion patterns",
			},
			CreatedAt: now,
		})
	}

	return suggestions
}

// generateGeneralSuggestions generates general suggestions based on execution analysis
func (r *RemediationEngine) generateGeneralSuggestions(executions []TestExecutionRecord) []RemediationSuggestion {
	var suggestions []RemediationSuggestion
	now := time.Now()

	if len(executions) < 10 {
		return suggestions // Need sufficient data
	}

	// Analyze execution duration variance
	durations := make([]float64, len(executions))
	for i, exec := range executions {
		durations[i] = float64(exec.Duration.Nanoseconds())
	}

	variance := calculateVariance(durations)
	mean := calculateMean(durations)
	
	if mean > 0 {
		coefficientOfVariation := variance / (mean * mean)
		
		if coefficientOfVariation > 0.5 { // High variance in execution time
			suggestions = append(suggestions, RemediationSuggestion{
				Type:        "timing",
				Priority:    "medium",
				Description: "High variance in execution time suggests inconsistent performance",
				Action:      "Investigate and stabilize test execution performance",
				Confidence:  0.7,
				Examples: []string{
					"Profile test execution to identify bottlenecks",
					"Use consistent test data sizes",
					"Implement proper resource warming",
					"Add performance monitoring to tests",
				},
				CreatedAt: now,
			})
		}
	}

	// Analyze failure distribution
	failuresByEnvironment := make(map[string]int)
	totalByEnvironment := make(map[string]int)

	for _, exec := range executions {
		if exec.Environment != "" {
			totalByEnvironment[exec.Environment]++
			if exec.Status == "failed" || exec.Status == "error" {
				failuresByEnvironment[exec.Environment]++
			}
		}
	}

	// Check for environment-specific issues
	for env, failures := range failuresByEnvironment {
		total := totalByEnvironment[env]
		if total >= 5 && float64(failures)/float64(total) > 0.5 {
			suggestions = append(suggestions, RemediationSuggestion{
				Type:        "environment",
				Priority:    "high",
				Description: fmt.Sprintf("High failure rate in %s environment", env),
				Action:      fmt.Sprintf("Review and fix %s environment configuration", env),
				Confidence:  0.8,
				Examples: []string{
					"Check environment-specific configuration",
					"Verify resource availability in environment",
					"Compare with stable environments",
					"Implement environment health monitoring",
				},
				CreatedAt: now,
			})
		}
	}

	// Check for recent degradation
	if len(executions) >= 20 {
		recentExecutions := executions[:10]
		olderExecutions := executions[10:20]

		recentFailures := 0
		olderFailures := 0

		for _, exec := range recentExecutions {
			if exec.Status == "failed" || exec.Status == "error" {
				recentFailures++
			}
		}

		for _, exec := range olderExecutions {
			if exec.Status == "failed" || exec.Status == "error" {
				olderFailures++
			}
		}

		recentFailureRate := float64(recentFailures) / float64(len(recentExecutions))
		olderFailureRate := float64(olderFailures) / float64(len(olderExecutions))

		if recentFailureRate > olderFailureRate+0.2 { // Significant increase in failures
			suggestions = append(suggestions, RemediationSuggestion{
				Type:        "regression",
				Priority:    "high",
				Description: "Recent increase in failure rate detected",
				Action:      "Investigate recent changes that may have introduced instability",
				Confidence:  0.8,
				Examples: []string{
					"Review recent code changes",
					"Check for infrastructure changes",
					"Analyze dependency updates",
					"Compare with previous stable versions",
				},
				CreatedAt: now,
			})
		}
	}

	return suggestions
}

// deduplicateAndSort removes duplicate suggestions and sorts by priority
func (r *RemediationEngine) deduplicateAndSort(suggestions []RemediationSuggestion) []RemediationSuggestion {
	seen := make(map[string]bool)
	var unique []RemediationSuggestion

	for _, suggestion := range suggestions {
		key := fmt.Sprintf("%s:%s:%s", suggestion.Type, suggestion.Priority, suggestion.Description)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, suggestion)
		}
	}

	// Sort by priority (high -> medium -> low) and then by confidence
	priorityOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
	
	for i := 0; i < len(unique)-1; i++ {
		for j := i + 1; j < len(unique); j++ {
			iPriority := priorityOrder[unique[i].Priority]
			jPriority := priorityOrder[unique[j].Priority]
			
			if iPriority < jPriority || 
			   (iPriority == jPriority && unique[i].Confidence < unique[j].Confidence) {
				unique[i], unique[j] = unique[j], unique[i]
			}
		}
	}

	return unique
}

// Helper methods for database access

func (r *RemediationEngine) getFailurePatterns(testName, testSuite string) ([]FailurePattern, error) {
	if r.db == nil {
		return []FailurePattern{}, nil // Return empty if no database
	}

	query := `
		SELECT pattern_type, description, frequency, first_seen, last_seen, confidence
		FROM test_failure_patterns
		WHERE test_name = $1 AND test_suite = $2
		  AND last_seen >= $3
		ORDER BY confidence DESC
	`

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	rows, err := r.db.Query(query, testName, testSuite, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []FailurePattern
	for rows.Next() {
		var pattern FailurePattern
		err := rows.Scan(&pattern.Type, &pattern.Description, &pattern.Frequency,
			&pattern.FirstSeen, &pattern.LastSeen, &pattern.Confidence)
		if err != nil {
			continue
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

func (r *RemediationEngine) getRecentExecutions(testName, testSuite string, limit int) ([]TestExecutionRecord, error) {
	if r.db == nil {
		return []TestExecutionRecord{}, nil // Return empty if no database
	}

	query := `
		SELECT test_name, test_suite, status, duration, start_time, end_time,
			   error_message, environment, build_id, commit_hash, branch
		FROM test_executions
		WHERE test_name = $1 AND test_suite = $2
		  AND start_time >= $3
		ORDER BY start_time DESC
		LIMIT $4
	`

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	rows, err := r.db.Query(query, testName, testSuite, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []TestExecutionRecord
	for rows.Next() {
		var exec TestExecutionRecord
		var duration int64

		err := rows.Scan(&exec.TestName, &exec.TestSuite, &exec.Status,
			&duration, &exec.StartTime, &exec.EndTime, &exec.ErrorMessage,
			&exec.Environment, &exec.BuildID, &exec.CommitHash, &exec.Branch)
		if err != nil {
			continue
		}

		exec.Duration = time.Duration(duration)
		executions = append(executions, exec)
	}

	return executions, nil
}

// GenerateRemediationReport generates a comprehensive remediation report
func (r *RemediationEngine) GenerateRemediationReport(flakyTests []TestReliabilityMetrics) (*RemediationReport, error) {
	report := &RemediationReport{
		GeneratedAt:   time.Now(),
		TotalTests:    len(flakyTests),
		Suggestions:   make(map[string][]RemediationSuggestion),
		Summary:       make(map[string]int),
	}

	for _, test := range flakyTests {
		suggestions, err := r.GenerateRemediationSuggestions(test.TestName, test.TestSuite)
		if err != nil {
			continue
		}

		testKey := fmt.Sprintf("%s.%s", test.TestSuite, test.TestName)
		report.Suggestions[testKey] = suggestions

		// Update summary counts
		for _, suggestion := range suggestions {
			report.Summary[suggestion.Type]++
		}
	}

	return report, nil
}

// RemediationReport represents a comprehensive remediation report
type RemediationReport struct {
	GeneratedAt time.Time                           `json:"generated_at"`
	TotalTests  int                                 `json:"total_tests"`
	Suggestions map[string][]RemediationSuggestion  `json:"suggestions"`
	Summary     map[string]int                      `json:"summary"`
}