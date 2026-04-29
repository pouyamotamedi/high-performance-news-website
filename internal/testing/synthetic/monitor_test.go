package synthetic

import (
	"context"
	"testing"
	"time"
)

func TestSyntheticMonitor_Creation(t *testing.T) {
	// Skip browser-based tests in CI/test environments
	t.Skip("Skipping browser-based test - requires Chrome installation and may be blocked by antivirus")
}

func TestTestScheduler_Schedule(t *testing.T) {
	scheduler := NewTestScheduler()

	testFunc := func(ctx context.Context) {
		// Test function implementation
	}

	scheduler.Schedule("test_job", 1*time.Second, testFunc)

	jobs := scheduler.GetJobStatus()
	if len(jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(jobs))
	}

	if job, exists := jobs["test_job"]; !exists {
		t.Error("Expected job 'test_job' to exist")
	} else {
		if job.Name != "test_job" {
			t.Errorf("Expected job name 'test_job', got '%s'", job.Name)
		}
		if job.Interval != 1*time.Second {
			t.Errorf("Expected interval 1s, got %v", job.Interval)
		}
		if !job.Enabled {
			t.Error("Job should be enabled by default")
		}
	}
}

func TestMemoryResultStore_Store(t *testing.T) {
	store := NewMemoryResultStore()

	result := MonitoringResult{
		TestName:  "test_homepage",
		TestType:  "user_journey",
		Status:    StatusPassed,
		Duration:  500 * time.Millisecond,
		Timestamp: time.Now(),
		Metrics:   map[string]float64{"load_time_ms": 500},
	}

	err := store.Store(result)
	if err != nil {
		t.Errorf("Failed to store result: %v", err)
	}

	results, err := store.GetLatestResults(10)
	if err != nil {
		t.Errorf("Failed to get results: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].TestName != "test_homepage" {
		t.Errorf("Expected test name 'test_homepage', got '%s'", results[0].TestName)
	}
}

func TestMemoryResultStore_GetTestSummary(t *testing.T) {
	store := NewMemoryResultStore()

	// Add multiple results for the same test
	for i := 0; i < 5; i++ {
		result := MonitoringResult{
			TestName:  "test_performance",
			TestType:  "performance",
			Status:    StatusPassed,
			Duration:  time.Duration(100+i*50) * time.Millisecond,
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Metrics:   map[string]float64{"response_time": float64(100 + i*50)},
		}
		store.Store(result)
	}

	summary, err := store.GetTestSummary("test_performance", 24*time.Hour)
	if err != nil {
		t.Errorf("Failed to get test summary: %v", err)
	}

	if summary.TestName != "test_performance" {
		t.Errorf("Expected test name 'test_performance', got '%s'", summary.TestName)
	}

	if summary.TotalRuns != 5 {
		t.Errorf("Expected 5 total runs, got %d", summary.TotalRuns)
	}

	if summary.SuccessRate != 100.0 {
		t.Errorf("Expected 100%% success rate, got %.1f%%", summary.SuccessRate)
	}

	if len(summary.Trends) != 5 {
		t.Errorf("Expected 5 trend points, got %d", len(summary.Trends))
	}
}

func TestAlertManager_SendAlert(t *testing.T) {
	alertManager := NewAlertManager()

	result := MonitoringResult{
		TestName:  "test_failure",
		TestType:  "user_journey",
		Status:    StatusFailed,
		Duration:  2 * time.Second,
		Timestamp: time.Now(),
		Errors:    []string{"Page load timeout"},
	}

	err := alertManager.SendAlert(AlertCritical, "Test failed", result)
	if err != nil {
		t.Errorf("Failed to send alert: %v", err)
	}

	alerts := alertManager.GetAlertHistory(10)
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.Level != AlertCritical {
		t.Errorf("Expected alert level 'critical', got '%s'", alert.Level)
	}

	if alert.TestName != "test_failure" {
		t.Errorf("Expected test name 'test_failure', got '%s'", alert.TestName)
	}
}

func TestAlertManager_RateLimiting(t *testing.T) {
	alertManager := NewAlertManager()

	result := MonitoringResult{
		TestName:  "test_rate_limit",
		TestType:  "user_journey",
		Status:    StatusFailed,
		Duration:  1 * time.Second,
		Timestamp: time.Now(),
	}

	// Send first alert
	err := alertManager.SendAlert(AlertHigh, "First alert", result)
	if err != nil {
		t.Errorf("Failed to send first alert: %v", err)
	}

	// Send second alert immediately (should be rate limited)
	err = alertManager.SendAlert(AlertHigh, "Second alert", result)
	if err != nil {
		t.Errorf("Failed to send second alert: %v", err)
	}

	alerts := alertManager.GetAlertHistory(10)
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert due to rate limiting, got %d", len(alerts))
	}
}

func TestAccessibilityValidator_ValidateAccessibility(t *testing.T) {
	validator := NewAccessibilityValidator()

	// Test HTML with accessibility issues
	htmlContent := `
	<html>
		<head><title>Test Page</title></head>
		<body>
			<h1>Main Heading</h1>
			<img src="test.jpg">
			<p>Some content</p>
		</body>
	</html>
	`

	results := validator.ValidateAccessibility(htmlContent)

	if len(results) == 0 {
		t.Error("Expected validation results, got none")
	}

	// Check for image alt text validation
	var altTextResult *ValidationResult
	for _, result := range results {
		if result.RuleID == "1.1.1" {
			altTextResult = &result
			break
		}
	}

	if altTextResult == nil {
		t.Error("Expected alt text validation result")
	} else if altTextResult.Passed {
		t.Error("Expected alt text validation to fail (image without alt)")
	}
}

func TestMobileExperienceValidator_ValidateMobileExperience(t *testing.T) {
	validator := NewMobileExperienceValidator()

	// Test metrics with mobile issues
	pageMetrics := map[string]interface{}{
		"has_viewport_meta": false,
		"scroll_width":      400.0,
		"viewport_width":    375.0,
		"small_touch_targets": 3.0,
	}

	results := validator.ValidateMobileExperience(pageMetrics)

	if len(results) == 0 {
		t.Error("Expected validation results, got none")
	}

	// Check viewport validation
	var viewportResult *ValidationResult
	for _, result := range results {
		if result.RuleID == "mobile.viewport" {
			viewportResult = &result
			break
		}
	}

	if viewportResult == nil {
		t.Error("Expected viewport validation result")
	} else if viewportResult.Passed {
		t.Error("Expected viewport validation to fail (no viewport meta)")
	}

	// Check scroll validation
	var scrollResult *ValidationResult
	for _, result := range results {
		if result.RuleID == "mobile.scroll" {
			scrollResult = &result
			break
		}
	}

	if scrollResult == nil {
		t.Error("Expected scroll validation result")
	} else if scrollResult.Passed {
		t.Error("Expected scroll validation to fail (horizontal scrolling)")
	}
}

func TestPerformanceValidator_CheckRegression(t *testing.T) {
	validator := NewPerformanceValidator()

	// Set up baseline
	responses := []time.Duration{
		100 * time.Millisecond,
		150 * time.Millisecond,
		200 * time.Millisecond,
		120 * time.Millisecond,
		180 * time.Millisecond,
	}

	validator.UpdateBaseline("test_performance", responses)

	// Test normal response (should not be regression)
	normalResponse := 150 * time.Millisecond
	if validator.CheckRegression("test_performance", normalResponse) {
		t.Error("Normal response should not be flagged as regression")
	}

	// Test slow response (should be regression)
	slowResponse := 500 * time.Millisecond
	if !validator.CheckRegression("test_performance", slowResponse) {
		t.Error("Slow response should be flagged as regression")
	}

	// Test unknown test (should not be regression)
	if validator.CheckRegression("unknown_test", slowResponse) {
		t.Error("Unknown test should not be flagged as regression")
	}
}

func TestSyntheticMonitoringService_Creation(t *testing.T) {
	// Skip browser-based tests in CI/test environments
	t.Skip("Skipping browser-based test - requires Chrome installation and may be blocked by antivirus")
}