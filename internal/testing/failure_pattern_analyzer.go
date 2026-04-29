package testing

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// FailurePatternAnalyzer analyzes patterns in test failures
type FailurePatternAnalyzer struct {
	db *sql.DB
}

// NewFailurePatternAnalyzer creates a new failure pattern analyzer
func NewFailurePatternAnalyzer(db *sql.DB) *FailurePatternAnalyzer {
	return &FailurePatternAnalyzer{db: db}
}

// AnalyzeFailurePatterns analyzes failure patterns for a test
func (f *FailurePatternAnalyzer) AnalyzeFailurePatterns(testName, testSuite string, executions []TestExecutionRecord) ([]FailurePattern, error) {
	var patterns []FailurePattern

	// Analyze different types of patterns
	intermittentPattern := f.analyzeIntermittentPattern(executions)
	if intermittentPattern != nil {
		patterns = append(patterns, *intermittentPattern)
	}

	consecutivePattern := f.analyzeConsecutivePattern(executions)
	if consecutivePattern != nil {
		patterns = append(patterns, *consecutivePattern)
	}

	environmentPattern := f.analyzeEnvironmentPattern(executions)
	if environmentPattern != nil {
		patterns = append(patterns, *environmentPattern)
	}

	timingPattern := f.analyzeTimingPattern(executions)
	if timingPattern != nil {
		patterns = append(patterns, *timingPattern)
	}

	errorMessagePattern := f.analyzeErrorMessagePattern(executions)
	if errorMessagePattern != nil {
		patterns = append(patterns, *errorMessagePattern)
	}

	return patterns, nil
}

// AnalyzeNewFailure analyzes a new failure for immediate pattern detection
func (f *FailurePatternAnalyzer) AnalyzeNewFailure(execution *TestExecutionRecord) error {
	// Get recent executions for context
	recentExecutions, err := f.getRecentExecutions(execution.TestName, execution.TestSuite, 20)
	if err != nil {
		return fmt.Errorf("failed to get recent executions: %w", err)
	}

	// Add current execution to the list
	recentExecutions = append([]TestExecutionRecord{*execution}, recentExecutions...)

	// Analyze patterns with the new failure
	patterns, err := f.AnalyzeFailurePatterns(execution.TestName, execution.TestSuite, recentExecutions)
	if err != nil {
		return fmt.Errorf("failed to analyze patterns: %w", err)
	}

	// Store detected patterns
	for _, pattern := range patterns {
		if err := f.storeFailurePattern(execution.TestName, execution.TestSuite, pattern); err != nil {
			log.Printf("Failed to store failure pattern: %v", err)
		}
	}

	return nil
}

// analyzeIntermittentPattern detects intermittent failure patterns
func (f *FailurePatternAnalyzer) analyzeIntermittentPattern(executions []TestExecutionRecord) *FailurePattern {
	if len(executions) < 10 {
		return nil
	}

	// Look for alternating pass/fail patterns
	transitions := 0
	consecutiveFailures := 0
	maxConsecutiveFailures := 0
	currentConsecutive := 0

	for i := 1; i < len(executions); i++ {
		prev := executions[i-1].Status
		curr := executions[i].Status

		// Count transitions between pass and fail
		if (prev == "passed" && (curr == "failed" || curr == "error")) ||
		   ((prev == "failed" || prev == "error") && curr == "passed") {
			transitions++
		}

		// Track consecutive failures
		if curr == "failed" || curr == "error" {
			currentConsecutive++
			if currentConsecutive > maxConsecutiveFailures {
				maxConsecutiveFailures = currentConsecutive
			}
		} else {
			if currentConsecutive > 0 {
				consecutiveFailures++
			}
			currentConsecutive = 0
		}
	}

	transitionRate := float64(transitions) / float64(len(executions)-1)

	// Intermittent pattern: high transition rate, but not all failures are consecutive
	if transitionRate > 0.3 && maxConsecutiveFailures < len(executions)/2 {
		var firstSeen, lastSeen time.Time
		for _, exec := range executions {
			if exec.Status == "failed" || exec.Status == "error" {
				if firstSeen.IsZero() || exec.StartTime.Before(firstSeen) {
					firstSeen = exec.StartTime
				}
				if lastSeen.IsZero() || exec.StartTime.After(lastSeen) {
					lastSeen = exec.StartTime
				}
			}
		}

		return &FailurePattern{
			Type:        "intermittent",
			Description: fmt.Sprintf("Test shows intermittent failures with %.1f%% transition rate", transitionRate*100),
			Frequency:   transitionRate,
			FirstSeen:   firstSeen,
			LastSeen:    lastSeen,
			Confidence:  f.calculateConfidence(transitionRate, 0.3, 1.0),
		}
	}

	return nil
}

// analyzeConsecutivePattern detects consecutive failure patterns
func (f *FailurePatternAnalyzer) analyzeConsecutivePattern(executions []TestExecutionRecord) *FailurePattern {
	if len(executions) < 5 {
		return nil
	}

	maxConsecutive := 0
	currentConsecutive := 0
	totalFailures := 0
	var firstConsecutiveStart, lastConsecutiveEnd time.Time

	for _, exec := range executions {
		if exec.Status == "failed" || exec.Status == "error" {
			if currentConsecutive == 0 {
				firstConsecutiveStart = exec.StartTime
			}
			currentConsecutive++
			totalFailures++
			lastConsecutiveEnd = exec.StartTime
			
			if currentConsecutive > maxConsecutive {
				maxConsecutive = currentConsecutive
			}
		} else {
			currentConsecutive = 0
		}
	}

	// Consecutive pattern: significant consecutive failures
	if maxConsecutive >= 3 && maxConsecutive < len(executions) {
		consecutiveRatio := float64(maxConsecutive) / float64(len(executions))
		
		return &FailurePattern{
			Type:        "consecutive",
			Description: fmt.Sprintf("Test shows consecutive failures (max %d in a row)", maxConsecutive),
			Frequency:   consecutiveRatio,
			FirstSeen:   firstConsecutiveStart,
			LastSeen:    lastConsecutiveEnd,
			Confidence:  f.calculateConfidence(consecutiveRatio, 0.2, 0.8),
		}
	}

	return nil
}

// analyzeEnvironmentPattern detects environment-specific failure patterns
func (f *FailurePatternAnalyzer) analyzeEnvironmentPattern(executions []TestExecutionRecord) *FailurePattern {
	envStats := make(map[string]struct{ total, failures int })

	for _, exec := range executions {
		if exec.Environment == "" {
			continue
		}

		stats := envStats[exec.Environment]
		stats.total++
		if exec.Status == "failed" || exec.Status == "error" {
			stats.failures++
		}
		envStats[exec.Environment] = stats
	}

	if len(envStats) < 2 {
		return nil // Need multiple environments to detect pattern
	}

	// Find environments with significantly different failure rates
	var problematicEnvs []string
	var maxFailureRate float64
	var firstSeen, lastSeen time.Time

	for env, stats := range envStats {
		if stats.total < 3 {
			continue // Need sufficient data
		}

		failureRate := float64(stats.failures) / float64(stats.total)
		if failureRate > 0.5 { // More than 50% failure rate
			problematicEnvs = append(problematicEnvs, env)
			if failureRate > maxFailureRate {
				maxFailureRate = failureRate
			}

			// Find time range for this environment's failures
			for _, exec := range executions {
				if exec.Environment == env && (exec.Status == "failed" || exec.Status == "error") {
					if firstSeen.IsZero() || exec.StartTime.Before(firstSeen) {
						firstSeen = exec.StartTime
					}
					if lastSeen.IsZero() || exec.StartTime.After(lastSeen) {
						lastSeen = exec.StartTime
					}
				}
			}
		}
	}

	if len(problematicEnvs) > 0 {
		return &FailurePattern{
			Type:        "environment",
			Description: fmt.Sprintf("Test fails more frequently in environments: %s", strings.Join(problematicEnvs, ", ")),
			Frequency:   maxFailureRate,
			FirstSeen:   firstSeen,
			LastSeen:    lastSeen,
			Confidence:  f.calculateConfidence(maxFailureRate, 0.5, 1.0),
		}
	}

	return nil
}

// analyzeTimingPattern detects timing-related failure patterns
func (f *FailurePatternAnalyzer) analyzeTimingPattern(executions []TestExecutionRecord) *FailurePattern {
	if len(executions) < 10 {
		return nil
	}

	// Analyze time-of-day patterns
	hourStats := make(map[int]struct{ total, failures int })
	
	for _, exec := range executions {
		hour := exec.StartTime.Hour()
		stats := hourStats[hour]
		stats.total++
		if exec.Status == "failed" || exec.Status == "error" {
			stats.failures++
		}
		hourStats[hour] = stats
	}

	// Find hours with high failure rates
	var problematicHours []int
	var maxFailureRate float64
	var firstSeen, lastSeen time.Time

	for hour, stats := range hourStats {
		if stats.total < 3 {
			continue
		}

		failureRate := float64(stats.failures) / float64(stats.total)
		if failureRate > 0.6 { // More than 60% failure rate at this hour
			problematicHours = append(problematicHours, hour)
			if failureRate > maxFailureRate {
				maxFailureRate = failureRate
			}

			// Find time range
			for _, exec := range executions {
				if exec.StartTime.Hour() == hour && (exec.Status == "failed" || exec.Status == "error") {
					if firstSeen.IsZero() || exec.StartTime.Before(firstSeen) {
						firstSeen = exec.StartTime
					}
					if lastSeen.IsZero() || exec.StartTime.After(lastSeen) {
						lastSeen = exec.StartTime
					}
				}
			}
		}
	}

	if len(problematicHours) > 0 {
		hourStrings := make([]string, len(problematicHours))
		for i, hour := range problematicHours {
			hourStrings[i] = fmt.Sprintf("%02d:00", hour)
		}

		return &FailurePattern{
			Type:        "timing",
			Description: fmt.Sprintf("Test fails more frequently during hours: %s", strings.Join(hourStrings, ", ")),
			Frequency:   maxFailureRate,
			FirstSeen:   firstSeen,
			LastSeen:    lastSeen,
			Confidence:  f.calculateConfidence(maxFailureRate, 0.6, 1.0),
		}
	}

	return nil
}

// analyzeErrorMessagePattern detects patterns in error messages
func (f *FailurePatternAnalyzer) analyzeErrorMessagePattern(executions []TestExecutionRecord) *FailurePattern {
	errorMessages := make(map[string]int)
	var firstSeen, lastSeen time.Time

	for _, exec := range executions {
		if exec.Status == "failed" || exec.Status == "error" {
			if exec.ErrorMessage != "" {
				// Normalize error message (remove dynamic parts)
				normalized := f.normalizeErrorMessage(exec.ErrorMessage)
				errorMessages[normalized]++

				if firstSeen.IsZero() || exec.StartTime.Before(firstSeen) {
					firstSeen = exec.StartTime
				}
				if lastSeen.IsZero() || exec.StartTime.After(lastSeen) {
					lastSeen = exec.StartTime
				}
			}
		}
	}

	if len(errorMessages) == 0 {
		return nil
	}

	// Find the most common error pattern
	var mostCommonError string
	var maxCount int
	totalFailures := 0

	for msg, count := range errorMessages {
		totalFailures += count
		if count > maxCount {
			maxCount = count
			mostCommonError = msg
		}
	}

	if maxCount >= 3 && totalFailures > 0 {
		frequency := float64(maxCount) / float64(totalFailures)
		
		if frequency > 0.5 { // More than 50% of failures have the same error pattern
			return &FailurePattern{
				Type:        "error_message",
				Description: fmt.Sprintf("Common error pattern: %s", f.truncateErrorMessage(mostCommonError)),
				Frequency:   frequency,
				FirstSeen:   firstSeen,
				LastSeen:    lastSeen,
				Confidence:  f.calculateConfidence(frequency, 0.5, 1.0),
			}
		}
	}

	return nil
}

// normalizeErrorMessage removes dynamic parts from error messages to find patterns
func (f *FailurePatternAnalyzer) normalizeErrorMessage(message string) string {
	// Remove timestamps
	timestampRegex := regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`)
	normalized := timestampRegex.ReplaceAllString(message, "[TIMESTAMP]")

	// Remove numbers that might be dynamic (IDs, ports, etc.)
	numberRegex := regexp.MustCompile(`\b\d{3,}\b`)
	normalized = numberRegex.ReplaceAllString(normalized, "[NUMBER]")

	// Remove file paths
	pathRegex := regexp.MustCompile(`[/\\][^\s]+\.(go|js|py|java|cpp|c)\b`)
	normalized = pathRegex.ReplaceAllString(normalized, "[FILEPATH]")

	// Remove memory addresses
	addressRegex := regexp.MustCompile(`0x[0-9a-fA-F]+`)
	normalized = addressRegex.ReplaceAllString(normalized, "[ADDRESS]")

	// Remove UUIDs
	uuidRegex := regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	normalized = uuidRegex.ReplaceAllString(normalized, "[UUID]")

	return strings.TrimSpace(normalized)
}

// truncateErrorMessage truncates error message for display
func (f *FailurePatternAnalyzer) truncateErrorMessage(message string) string {
	if len(message) <= 100 {
		return message
	}
	return message[:97] + "..."
}

// calculateConfidence calculates confidence score based on frequency and thresholds
func (f *FailurePatternAnalyzer) calculateConfidence(frequency, minThreshold, maxThreshold float64) float64 {
	if frequency < minThreshold {
		return 0.0
	}
	if frequency >= maxThreshold {
		return 1.0
	}

	// Linear interpolation between thresholds
	range_ := maxThreshold - minThreshold
	position := frequency - minThreshold
	return position / range_
}

// getRecentExecutions gets recent executions for pattern analysis
func (f *FailurePatternAnalyzer) getRecentExecutions(testName, testSuite string, limit int) ([]TestExecutionRecord, error) {
	query := `
		SELECT test_name, test_suite, status, duration, start_time, end_time,
			   error_message, environment, build_id, commit_hash, branch
		FROM test_executions
		WHERE test_name = $1 AND test_suite = $2
		  AND start_time >= $3
		ORDER BY start_time DESC
		LIMIT $4
	`

	cutoff := time.Now().Add(-72 * time.Hour) // Last 3 days for pattern analysis
	rows, err := f.db.Query(query, testName, testSuite, cutoff, limit)
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

// storeFailurePattern stores a detected failure pattern
func (f *FailurePatternAnalyzer) storeFailurePattern(testName, testSuite string, pattern FailurePattern) error {
	query := `
		INSERT INTO test_failure_patterns (
			test_name, test_suite, pattern_type, description, frequency,
			first_seen, last_seen, confidence, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (test_name, test_suite, pattern_type) DO UPDATE SET
			description = EXCLUDED.description,
			frequency = EXCLUDED.frequency,
			last_seen = EXCLUDED.last_seen,
			confidence = EXCLUDED.confidence,
			updated_at = NOW()
	`

	_, err := f.db.Exec(query, testName, testSuite, pattern.Type, pattern.Description,
		pattern.Frequency, pattern.FirstSeen, pattern.LastSeen, pattern.Confidence)

	return err
}

// GetFailurePatternsForTest retrieves stored failure patterns for a test
func (f *FailurePatternAnalyzer) GetFailurePatternsForTest(testName, testSuite string) ([]FailurePattern, error) {
	query := `
		SELECT pattern_type, description, frequency, first_seen, last_seen, confidence
		FROM test_failure_patterns
		WHERE test_name = $1 AND test_suite = $2
		  AND last_seen >= $3
		ORDER BY confidence DESC, frequency DESC
	`

	cutoff := time.Now().Add(-7 * 24 * time.Hour) // Patterns from last 7 days
	rows, err := f.db.Query(query, testName, testSuite, cutoff)
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

// CleanupOldPatterns removes old failure patterns
func (f *FailurePatternAnalyzer) CleanupOldPatterns() error {
	query := `
		DELETE FROM test_failure_patterns
		WHERE last_seen < $1
	`

	cutoff := time.Now().Add(-30 * 24 * time.Hour) // Remove patterns older than 30 days
	_, err := f.db.Exec(query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup old patterns: %w", err)
	}

	return nil
}