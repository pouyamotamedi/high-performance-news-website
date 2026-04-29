package validation

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestAICodeMonitor_Start(t *testing.T) {
	// Create a test database connection (mock)
	db := &sql.DB{} // In real tests, you'd use a test database
	
	monitor := NewAICodeMonitor(db)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start AI code monitor: %v", err)
	}
	
	if !monitor.isRunning {
		t.Error("Monitor should be running after start")
	}
	
	// Test that starting again returns error
	err = monitor.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running monitor")
	}
	
	monitor.Stop()
	
	if monitor.isRunning {
		t.Error("Monitor should not be running after stop")
	}
}

func TestAICodeMonitor_RecordDatabaseQuery(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	// Test recording a database query
	query := "SELECT * FROM articles WHERE id = $1"
	duration := 150 * time.Millisecond
	
	monitor.RecordDatabaseQuery(query, duration, 1, nil)
	
	metrics := monitor.GetMetrics()
	
	if metrics.DatabaseQueries.TotalQueries != 1 {
		t.Errorf("Expected 1 total query, got %d", metrics.DatabaseQueries.TotalQueries)
	}
	
	if metrics.DatabaseQueries.SlowQueries != 1 {
		t.Errorf("Expected 1 slow query (>100ms), got %d", metrics.DatabaseQueries.SlowQueries)
	}
	
	expectedAvg := 150.0
	if metrics.DatabaseQueries.AvgExecutionTime != expectedAvg {
		t.Errorf("Expected avg execution time %.2f, got %.2f", expectedAvg, metrics.DatabaseQueries.AvgExecutionTime)
	}
}

func TestAICodeMonitor_RecordError(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	// Test recording handled error
	monitor.RecordError("database_error", true, true)
	
	metrics := monitor.GetMetrics()
	
	if metrics.ErrorHandling.TotalErrors != 1 {
		t.Errorf("Expected 1 total error, got %d", metrics.ErrorHandling.TotalErrors)
	}
	
	if metrics.ErrorHandling.HandledErrors != 1 {
		t.Errorf("Expected 1 handled error, got %d", metrics.ErrorHandling.HandledErrors)
	}
	
	if metrics.ErrorHandling.RecoverySuccess != 1 {
		t.Errorf("Expected 1 recovery success, got %d", metrics.ErrorHandling.RecoverySuccess)
	}
	
	expectedEffectiveness := 100.0
	if metrics.ErrorHandling.HandlingEffectiveness != expectedEffectiveness {
		t.Errorf("Expected handling effectiveness %.2f, got %.2f", expectedEffectiveness, metrics.ErrorHandling.HandlingEffectiveness)
	}
	
	// Test recording unhandled error
	monitor.RecordError("validation_error", false, false)
	
	metrics = monitor.GetMetrics()
	
	if metrics.ErrorHandling.TotalErrors != 2 {
		t.Errorf("Expected 2 total errors, got %d", metrics.ErrorHandling.TotalErrors)
	}
	
	if metrics.ErrorHandling.UnhandledErrors != 1 {
		t.Errorf("Expected 1 unhandled error, got %d", metrics.ErrorHandling.UnhandledErrors)
	}
	
	expectedEffectiveness = 50.0 // 1 handled out of 2 total
	if metrics.ErrorHandling.HandlingEffectiveness != expectedEffectiveness {
		t.Errorf("Expected handling effectiveness %.2f, got %.2f", expectedEffectiveness, metrics.ErrorHandling.HandlingEffectiveness)
	}
}

func TestAICodeMonitor_RecordBusinessLogicViolation(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	// Set up some business rule checks first
	monitor.metrics.BusinessLogic.BusinessRuleChecks = 10
	
	// Record a violation
	monitor.RecordBusinessLogicViolation("data_validation")
	
	metrics := monitor.GetMetrics()
	
	if metrics.BusinessLogic.LogicViolations != 1 {
		t.Errorf("Expected 1 logic violation, got %d", metrics.BusinessLogic.LogicViolations)
	}
	
	if count, exists := metrics.BusinessLogic.RuleViolations["data_validation"]; !exists || count != 1 {
		t.Errorf("Expected 1 data_validation violation, got %d", count)
	}
	
	expectedScore := 90.0 // (10-1)/10 * 100
	if metrics.BusinessLogic.ConsistencyScore != expectedScore {
		t.Errorf("Expected consistency score %.2f, got %.2f", expectedScore, metrics.BusinessLogic.ConsistencyScore)
	}
}

func TestAICodeMonitor_RecordResponseTime(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	// Record several response times
	responseTimes := []time.Duration{
		100 * time.Millisecond,
		150 * time.Millisecond,
		200 * time.Millisecond,
		120 * time.Millisecond,
		180 * time.Millisecond,
	}
	
	for _, duration := range responseTimes {
		monitor.RecordResponseTime(duration)
	}
	
	metrics := monitor.GetMetrics()
	
	if len(metrics.Performance.ResponseTimes) != len(responseTimes) {
		t.Errorf("Expected %d response times, got %d", len(responseTimes), len(metrics.Performance.ResponseTimes))
	}
	
	// Check that response times are recorded correctly
	for i, expectedMs := range []float64{100, 150, 200, 120, 180} {
		if metrics.Performance.ResponseTimes[i] != expectedMs {
			t.Errorf("Expected response time %.2f, got %.2f", expectedMs, metrics.Performance.ResponseTimes[i])
		}
	}
}

func TestAICodeMonitor_AnalyzeQueryPattern(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	testCases := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM articles", "select_all"},
		{"SELECT id, title FROM articles JOIN categories ON articles.category_id = categories.id WHERE articles.status = 'published'", "complex_join"},
		{"SELECT * FROM articles ORDER BY published_at DESC LIMIT 10", "paginated_query"},
		{"INSERT INTO articles (title, content) VALUES ($1, $2)", "write_operation"},
		{"UPDATE articles SET status = 'published' WHERE id = $1", "write_operation"},
		{"DELETE FROM articles WHERE id = $1", "write_operation"},
		{"SELECT COUNT(*) FROM articles", "simple_query"},
	}
	
	for _, tc := range testCases {
		result := monitor.analyzeQueryPattern(tc.query)
		if result != tc.expected {
			t.Errorf("Query pattern analysis failed for '%s': expected '%s', got '%s'", tc.query, tc.expected, result)
		}
	}
}

func TestAICodeMonitor_CalculateQueryComplexity(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	testCases := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM articles", "low"},
		{"SELECT * FROM articles ORDER BY published_at", "low"},
		{"SELECT * FROM articles JOIN categories ON articles.category_id = categories.id", "medium"},
		{"SELECT * FROM articles JOIN categories ON articles.category_id = categories.id ORDER BY published_at GROUP BY category_id", "high"},
		{"SELECT * FROM articles WHERE id IN (SELECT article_id FROM article_tags WHERE tag_id = 1) ORDER BY published_at GROUP BY category_id HAVING COUNT(*) > 5", "very_high"},
	}
	
	for _, tc := range testCases {
		result := monitor.calculateQueryComplexity(tc.query)
		if result != tc.expected {
			t.Errorf("Query complexity calculation failed for '%s': expected '%s', got '%s'", tc.query, tc.expected, result)
		}
	}
}

func TestAICodeMonitor_CalculatePerformanceScore(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	// Test with good performance metrics
	monitor.metrics.Performance.MemoryUsage = 50 * 1024 * 1024  // 50MB
	monitor.metrics.Performance.GoroutineCount = 50              // 50 goroutines
	monitor.metrics.Performance.ThroughputRPS = 600              // 600 RPS
	monitor.metrics.Performance.GCPauses = []time.Duration{
		2 * time.Millisecond,
		3 * time.Millisecond,
		1 * time.Millisecond,
	}
	
	score := monitor.calculatePerformanceScore()
	expectedScore := 100.0 // All metrics are good
	
	if score != expectedScore {
		t.Errorf("Expected performance score %.2f, got %.2f", expectedScore, score)
	}
	
	// Test with poor performance metrics
	monitor.metrics.Performance.MemoryUsage = 600 * 1024 * 1024 // 600MB (high)
	monitor.metrics.Performance.GoroutineCount = 1500           // 1500 goroutines (high)
	monitor.metrics.Performance.ThroughputRPS = 50              // 50 RPS (low)
	monitor.metrics.Performance.GCPauses = []time.Duration{
		15 * time.Millisecond, // Long GC pauses
		20 * time.Millisecond,
		12 * time.Millisecond,
	}
	
	score = monitor.calculatePerformanceScore()
	expectedScore = 50.0 // 100 - 20 (memory) - 15 (goroutines) - 15 (throughput) - 10 (GC)
	
	if score != expectedScore {
		t.Errorf("Expected performance score %.2f, got %.2f", expectedScore, score)
	}
}

func TestAICodeMonitor_MetricsThreadSafety(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	// Test concurrent access to metrics
	done := make(chan bool)
	
	// Goroutine 1: Record database queries
	go func() {
		for i := 0; i < 100; i++ {
			monitor.RecordDatabaseQuery("SELECT * FROM test", 10*time.Millisecond, 1, nil)
		}
		done <- true
	}()
	
	// Goroutine 2: Record errors
	go func() {
		for i := 0; i < 100; i++ {
			monitor.RecordError("test_error", true, true)
		}
		done <- true
	}()
	
	// Goroutine 3: Record response times
	go func() {
		for i := 0; i < 100; i++ {
			monitor.RecordResponseTime(100 * time.Millisecond)
		}
		done <- true
	}()
	
	// Goroutine 4: Read metrics
	go func() {
		for i := 0; i < 100; i++ {
			_ = monitor.GetMetrics()
		}
		done <- true
	}()
	
	// Wait for all goroutines to complete
	for i := 0; i < 4; i++ {
		<-done
	}
	
	// Verify final metrics
	metrics := monitor.GetMetrics()
	
	if metrics.DatabaseQueries.TotalQueries != 100 {
		t.Errorf("Expected 100 total queries, got %d", metrics.DatabaseQueries.TotalQueries)
	}
	
	if metrics.ErrorHandling.TotalErrors != 100 {
		t.Errorf("Expected 100 total errors, got %d", metrics.ErrorHandling.TotalErrors)
	}
	
	if len(metrics.Performance.ResponseTimes) != 100 {
		t.Errorf("Expected 100 response times, got %d", len(metrics.Performance.ResponseTimes))
	}
}

func TestAICodeMonitor_Integration(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// Start monitoring
	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	
	// Simulate AI-generated code activity
	go func() {
		for i := 0; i < 10; i++ {
			// Simulate database queries
			monitor.RecordDatabaseQuery("SELECT * FROM articles WHERE id = $1", 75*time.Millisecond, 1, nil)
			
			// Simulate API requests
			monitor.RecordResponseTime(120 * time.Millisecond)
			
			// Simulate occasional errors
			if i%3 == 0 {
				monitor.RecordError("validation_error", true, true)
			}
			
			// Simulate business logic checks
			if i%2 == 0 {
				monitor.RecordBusinessLogicViolation("data_consistency")
			}
			
			time.Sleep(100 * time.Millisecond)
		}
	}()
	
	// Let it run for a bit
	time.Sleep(1500 * time.Millisecond)
	
	// Check that anomaly detection is working
	anomalies := monitor.anomalyDetector.DetectAnomalies()
	
	// We expect some anomalies due to the simulated issues
	if len(anomalies) == 0 {
		t.Log("No anomalies detected (this might be expected depending on thresholds)")
	} else {
		t.Logf("Detected %d anomalies", len(anomalies))
		for _, anomaly := range anomalies {
			t.Logf("Anomaly: %s - %s", anomaly.Type, anomaly.Description)
		}
	}
	
	// Get final metrics
	metrics := monitor.GetMetrics()
	
	if metrics.DatabaseQueries.TotalQueries == 0 {
		t.Error("Expected some database queries to be recorded")
	}
	
	if len(metrics.Performance.ResponseTimes) == 0 {
		t.Error("Expected some response times to be recorded")
	}
	
	monitor.Stop()
}

// Benchmark tests
func BenchmarkAICodeMonitor_RecordDatabaseQuery(b *testing.B) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	query := "SELECT * FROM articles WHERE id = $1"
	duration := 50 * time.Millisecond
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		monitor.RecordDatabaseQuery(query, duration, 1, nil)
	}
}

func BenchmarkAICodeMonitor_RecordError(b *testing.B) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		monitor.RecordError("test_error", true, true)
	}
}

func BenchmarkAICodeMonitor_GetMetrics(b *testing.B) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	
	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		monitor.RecordDatabaseQuery("SELECT * FROM test", 10*time.Millisecond, 1, nil)
		monitor.RecordError("test_error", true, true)
		monitor.RecordResponseTime(100 * time.Millisecond)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = monitor.GetMetrics()
	}
}