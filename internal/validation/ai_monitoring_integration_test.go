package validation

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// TestAIMonitoringSystemIntegration tests the complete AI monitoring system
func TestAIMonitoringSystemIntegration(t *testing.T) {
	// Create test database connection (mock)
	db := &sql.DB{}
	
	// Initialize the complete monitoring system
	monitor := NewAICodeMonitor(db)
	performanceTracker := NewAIPerformanceTracker()
	alertManager := NewAIAlertManager()
	
	// Start monitoring
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start AI code monitor: %v", err)
	}
	defer monitor.Stop()
	
	// Simulate AI-generated code execution patterns
	t.Run("SimulateAICodeExecution", func(t *testing.T) {
		// Simulate database queries with varying performance
		queries := []struct {
			query    string
			duration time.Duration
			isAI     bool
		}{
			{"SELECT * FROM articles WHERE id = $1", 45 * time.Millisecond, true},
			{"SELECT * FROM articles ORDER BY published_at DESC LIMIT 10", 120 * time.Millisecond, true},
			{"SELECT COUNT(*) FROM articles", 25 * time.Millisecond, false},
			{"SELECT * FROM articles JOIN categories ON articles.category_id = categories.id WHERE articles.status = 'published'", 180 * time.Millisecond, true},
			{"INSERT INTO articles (title, content) VALUES ($1, $2)", 35 * time.Millisecond, true},
		}
		
		for i, query := range queries {
			monitor.RecordDatabaseQuery(query.query, query.duration, 1, nil)
			performanceTracker.TrackDatabaseQuery(query.query, query.duration, 1, query.isAI)
			
			// Simulate API requests
			endpoint := "/api/articles"
			if i%2 == 0 {
				endpoint = "/api/categories"
			}
			
			apiDuration := time.Duration(150+i*20) * time.Millisecond
			performanceTracker.TrackAPIRequest(endpoint, "GET", apiDuration, 200, 1024, 2048, query.isAI)
			monitor.RecordResponseTime(apiDuration)
		}
		
		// Simulate errors with different handling patterns
		errorScenarios := []struct {
			errorType string
			handled   bool
			recovered bool
			isAI      bool
		}{
			{"database_connection_error", true, true, true},
			{"validation_error", true, false, true},
			{"timeout_error", false, false, true},
			{"parsing_error", true, true, false},
			{"network_error", false, false, true},
		}
		
		for _, scenario := range errorScenarios {
			monitor.RecordError(scenario.errorType, scenario.handled, scenario.recovered)
			performanceTracker.TrackError(scenario.errorType, "Error occurred", "api_handler", scenario.handled, scenario.recovered, scenario.isAI)
		}
		
		// Simulate business logic violations
		violations := []string{
			"data_consistency",
			"business_rule_validation",
			"data_consistency",
			"authorization_check",
			"input_validation",
		}
		
		for _, violation := range violations {
			monitor.RecordBusinessLogicViolation(violation)
		}
		
		// Simulate cache operations
		cacheOps := []struct {
			operation string
			hit       bool
			duration  time.Duration
			isAI      bool
		}{
			{"GET", true, 2 * time.Millisecond, true},
			{"SET", false, 5 * time.Millisecond, true},
			{"GET", false, 15 * time.Millisecond, true},
			{"DELETE", false, 3 * time.Millisecond, false},
			{"GET", true, 1 * time.Millisecond, true},
		}
		
		for i, op := range cacheOps {
			key := "article_" + string(rune(i))
			performanceTracker.TrackCacheOperation(op.operation, key, op.hit, op.duration, 1024, 5*time.Minute, op.isAI)
		}
	})
	
	// Allow some time for monitoring to collect data
	time.Sleep(2 * time.Second)
	
	// Test metrics collection
	t.Run("VerifyMetricsCollection", func(t *testing.T) {
		metrics := monitor.GetMetrics()
		
		if metrics.DatabaseQueries.TotalQueries == 0 {
			t.Error("Expected database queries to be recorded")
		}
		
		if metrics.ErrorHandling.TotalErrors == 0 {
			t.Error("Expected errors to be recorded")
		}
		
		if metrics.BusinessLogic.LogicViolations == 0 {
			t.Error("Expected business logic violations to be recorded")
		}
		
		if len(metrics.Performance.ResponseTimes) == 0 {
			t.Error("Expected response times to be recorded")
		}
		
		t.Logf("Collected metrics: %d queries, %d errors, %d violations, %d response times",
			metrics.DatabaseQueries.TotalQueries,
			metrics.ErrorHandling.TotalErrors,
			metrics.BusinessLogic.LogicViolations,
			len(metrics.Performance.ResponseTimes))
	})
	
	// Test anomaly detection
	t.Run("TestAnomalyDetection", func(t *testing.T) {
		anomalies := monitor.anomalyDetector.DetectAnomalies()
		
		t.Logf("Detected %d anomalies", len(anomalies))
		
		for _, anomaly := range anomalies {
			t.Logf("Anomaly: %s - %s (Severity: %s, Confidence: %.2f)",
				anomaly.Type, anomaly.Description, anomaly.Severity, anomaly.Confidence)
			
			// Verify anomaly structure
			if anomaly.Description == "" {
				t.Error("Anomaly should have a description")
			}
			
			if anomaly.Recommendation == "" {
				t.Error("Anomaly should have a recommendation")
			}
			
			if anomaly.DetectedAt.IsZero() {
				t.Error("Anomaly should have a detection timestamp")
			}
		}
	})
	
	// Test performance regression detection
	t.Run("TestRegressionDetection", func(t *testing.T) {
		// Update performance tracker memory usage to trigger regression
		performanceTracker.TrackMemoryUsage()
		
		regressions := performanceTracker.DetectRegressions()
		
		t.Logf("Detected %d performance regressions", len(regressions))
		
		for _, regression := range regressions {
			t.Logf("Regression: %s - %s (Severity: %s, Change: %.2f%%)",
				regression.RuleName, regression.Description, regression.Severity, regression.PercentChange)
			
			// Verify regression structure
			if regression.Description == "" {
				t.Error("Regression should have a description")
			}
			
			if regression.Recommendation == "" {
				t.Error("Regression should have a recommendation")
			}
			
			if regression.DetectedAt.IsZero() {
				t.Error("Regression should have a detection timestamp")
			}
		}
	})
	
	// Test alert system
	t.Run("TestAlertSystem", func(t *testing.T) {
		// Get anomalies and regressions
		anomalies := monitor.anomalyDetector.DetectAnomalies()
		regressions := performanceTracker.DetectRegressions()
		
		// Send alerts
		if len(anomalies) > 0 {
			alertManager.SendAlerts(anomalies)
		}
		
		if len(regressions) > 0 {
			alertManager.SendRegressionAlerts(regressions)
		}
		
		// Check alert history
		alertHistory := alertManager.GetAlertHistory(10)
		
		t.Logf("Generated %d alerts", len(alertHistory))
		
		for _, alert := range alertHistory {
			t.Logf("Alert: %s - %s (Severity: %s, Type: %s)",
				alert.Title, alert.Description, alert.Severity, alert.Type)
			
			// Verify alert structure
			if alert.Title == "" {
				t.Error("Alert should have a title")
			}
			
			if alert.Description == "" {
				t.Error("Alert should have a description")
			}
			
			if alert.Source == "" {
				t.Error("Alert should have a source")
			}
			
			if alert.Timestamp.IsZero() {
				t.Error("Alert should have a timestamp")
			}
		}
	})
	
	// Test baseline updates
	t.Run("TestBaselineUpdates", func(t *testing.T) {
		// Update baselines based on collected data
		monitor.anomalyDetector.UpdateBaselines()
		err := performanceTracker.UpdateBaselines(ctx)
		if err != nil {
			t.Errorf("Failed to update performance baselines: %v", err)
		}
		
		// Verify baselines were updated
		baselines := monitor.anomalyDetector.GetBaselines()
		if baselines.LastUpdated.IsZero() {
			t.Error("Baselines should have been updated")
		}
		
		perfBaselines := performanceTracker.GetBaselines()
		if perfBaselines.EstablishedAt.IsZero() {
			t.Error("Performance baselines should have been updated")
		}
		
		t.Logf("Updated baselines - Query time: %.2fms, Error rate: %.2f%%, Memory: %d bytes",
			baselines.QueryExecutionTime, baselines.ErrorRate, baselines.MemoryAllocation)
	})
	
	// Test trend analysis
	t.Run("TestTrendAnalysis", func(t *testing.T) {
		// Get trend analysis for different metrics
		queryTrend := performanceTracker.GetTrendAnalysis("query_time")
		responseTrend := performanceTracker.GetTrendAnalysis("response_time")
		memoryTrend := performanceTracker.GetTrendAnalysis("memory_usage")
		
		if queryTrend != nil {
			t.Logf("Query time trend: %d data points", len(queryTrend))
		}
		
		if responseTrend != nil {
			t.Logf("Response time trend: %d data points", len(responseTrend))
		}
		
		if memoryTrend != nil {
			t.Logf("Memory usage trend: %d data points", len(memoryTrend))
		}
		
		// Verify historical data is being collected
		historicalData := monitor.anomalyDetector.GetHistoricalData()
		if len(historicalData.QueryTimes) == 0 {
			t.Error("Expected historical query time data to be collected")
		}
		
		if len(historicalData.Timestamps) == 0 {
			t.Error("Expected historical timestamp data to be collected")
		}
	})
	
	// Test alert acknowledgment and resolution
	t.Run("TestAlertManagement", func(t *testing.T) {
		alertHistory := alertManager.GetAlertHistory(5)
		
		if len(alertHistory) > 0 {
			alert := alertHistory[0]
			
			// Test acknowledgment
			err := alertManager.AcknowledgeAlert(alert.ID, "test_user", "Investigating the issue")
			if err != nil {
				t.Errorf("Failed to acknowledge alert: %v", err)
			}
			
			// Test resolution
			err = alertManager.ResolveAlert(alert.ID, "test_user")
			if err != nil {
				t.Errorf("Failed to resolve alert: %v", err)
			}
			
			// Verify alert was updated
			updatedHistory := alertManager.GetAlertHistory(5)
			if len(updatedHistory) > 0 {
				updatedAlert := updatedHistory[0]
				if updatedAlert.ID == alert.ID {
					if !updatedAlert.Resolved {
						t.Error("Alert should be marked as resolved")
					}
					
					if len(updatedAlert.Acknowledgments) == 0 {
						t.Error("Alert should have acknowledgments")
					}
				}
			}
		}
	})
}

// TestAIMonitoringPerformance tests the performance of the monitoring system
func TestAIMonitoringPerformance(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	performanceTracker := NewAIPerformanceTracker()
	
	// Test high-volume data recording
	t.Run("HighVolumeDataRecording", func(t *testing.T) {
		start := time.Now()
		
		// Record 1000 database queries
		for i := 0; i < 1000; i++ {
			query := "SELECT * FROM articles WHERE id = $1"
			duration := time.Duration(50+i%100) * time.Millisecond
			monitor.RecordDatabaseQuery(query, duration, 1, nil)
			performanceTracker.TrackDatabaseQuery(query, duration, 1, true)
		}
		
		// Record 1000 API requests
		for i := 0; i < 1000; i++ {
			endpoint := "/api/articles"
			duration := time.Duration(100+i%200) * time.Millisecond
			performanceTracker.TrackAPIRequest(endpoint, "GET", duration, 200, 1024, 2048, true)
			monitor.RecordResponseTime(duration)
		}
		
		// Record 500 errors
		for i := 0; i < 500; i++ {
			errorType := "test_error"
			handled := i%2 == 0
			recovered := i%3 == 0
			monitor.RecordError(errorType, handled, recovered)
			performanceTracker.TrackError(errorType, "Test error", "test_context", handled, recovered, true)
		}
		
		elapsed := time.Since(start)
		t.Logf("Recorded 2500 events in %v (%.2f events/ms)", elapsed, float64(2500)/float64(elapsed.Nanoseconds()/1e6))
		
		// Verify data was recorded correctly
		metrics := monitor.GetMetrics()
		if metrics.DatabaseQueries.TotalQueries != 1000 {
			t.Errorf("Expected 1000 database queries, got %d", metrics.DatabaseQueries.TotalQueries)
		}
		
		if metrics.ErrorHandling.TotalErrors != 500 {
			t.Errorf("Expected 500 errors, got %d", metrics.ErrorHandling.TotalErrors)
		}
		
		if len(metrics.Performance.ResponseTimes) != 1000 {
			t.Errorf("Expected 1000 response times, got %d", len(metrics.Performance.ResponseTimes))
		}
	})
	
	// Test anomaly detection performance
	t.Run("AnomalyDetectionPerformance", func(t *testing.T) {
		start := time.Now()
		
		// Run anomaly detection multiple times
		for i := 0; i < 100; i++ {
			_ = monitor.anomalyDetector.DetectAnomalies()
		}
		
		elapsed := time.Since(start)
		t.Logf("Performed 100 anomaly detections in %v (%.2fms per detection)", elapsed, float64(elapsed.Nanoseconds()/1e6)/100)
	})
	
	// Test regression detection performance
	t.Run("RegressionDetectionPerformance", func(t *testing.T) {
		start := time.Now()
		
		// Run regression detection multiple times
		for i := 0; i < 100; i++ {
			_ = performanceTracker.DetectRegressions()
		}
		
		elapsed := time.Since(start)
		t.Logf("Performed 100 regression detections in %v (%.2fms per detection)", elapsed, float64(elapsed.Nanoseconds()/1e6)/100)
	})
}

// TestAIMonitoringConcurrency tests concurrent access to the monitoring system
func TestAIMonitoringConcurrency(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	performanceTracker := NewAIPerformanceTracker()
	alertManager := NewAIAlertManager()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	defer monitor.Stop()
	
	// Test concurrent data recording
	t.Run("ConcurrentDataRecording", func(t *testing.T) {
		done := make(chan bool, 4)
		
		// Goroutine 1: Record database queries
		go func() {
			for i := 0; i < 200; i++ {
				query := "SELECT * FROM test"
				duration := time.Duration(50+i%50) * time.Millisecond
				monitor.RecordDatabaseQuery(query, duration, 1, nil)
				performanceTracker.TrackDatabaseQuery(query, duration, 1, true)
			}
			done <- true
		}()
		
		// Goroutine 2: Record errors
		go func() {
			for i := 0; i < 200; i++ {
				monitor.RecordError("concurrent_error", i%2 == 0, i%3 == 0)
				performanceTracker.TrackError("concurrent_error", "Test", "context", i%2 == 0, i%3 == 0, true)
			}
			done <- true
		}()
		
		// Goroutine 3: Record response times
		go func() {
			for i := 0; i < 200; i++ {
				duration := time.Duration(100+i%100) * time.Millisecond
				monitor.RecordResponseTime(duration)
				performanceTracker.TrackAPIRequest("/test", "GET", duration, 200, 1024, 2048, true)
			}
			done <- true
		}()
		
		// Goroutine 4: Detect anomalies and send alerts
		go func() {
			for i := 0; i < 50; i++ {
				anomalies := monitor.anomalyDetector.DetectAnomalies()
				if len(anomalies) > 0 {
					alertManager.SendAlerts(anomalies)
				}
				
				regressions := performanceTracker.DetectRegressions()
				if len(regressions) > 0 {
					alertManager.SendRegressionAlerts(regressions)
				}
				
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()
		
		// Wait for all goroutines to complete
		for i := 0; i < 4; i++ {
			<-done
		}
		
		// Verify data integrity
		metrics := monitor.GetMetrics()
		if metrics.DatabaseQueries.TotalQueries != 200 {
			t.Errorf("Expected 200 database queries, got %d", metrics.DatabaseQueries.TotalQueries)
		}
		
		if metrics.ErrorHandling.TotalErrors != 200 {
			t.Errorf("Expected 200 errors, got %d", metrics.ErrorHandling.TotalErrors)
		}
		
		if len(metrics.Performance.ResponseTimes) != 200 {
			t.Errorf("Expected 200 response times, got %d", len(metrics.Performance.ResponseTimes))
		}
		
		// Check that alerts were generated
		alertHistory := alertManager.GetAlertHistory(10)
		t.Logf("Generated %d alerts during concurrent testing", len(alertHistory))
	})
}

// TestAIMonitoringEdgeCases tests edge cases and error conditions
func TestAIMonitoringEdgeCases(t *testing.T) {
	db := &sql.DB{}
	monitor := NewAICodeMonitor(db)
	performanceTracker := NewAIPerformanceTracker()
	alertManager := NewAIAlertManager()
	
	t.Run("EmptyMetrics", func(t *testing.T) {
		// Test anomaly detection with empty metrics
		anomalies := monitor.anomalyDetector.DetectAnomalies()
		// Should not crash, may return empty or minimal anomalies
		t.Logf("Detected %d anomalies with empty metrics", len(anomalies))
	})
	
	t.Run("ExtremeValues", func(t *testing.T) {
		// Test with extreme values
		monitor.RecordDatabaseQuery("SELECT * FROM test", 10*time.Second, 1000000, nil) // Very slow query
		monitor.RecordResponseTime(30 * time.Second) // Very slow response
		
		// Should handle extreme values gracefully
		metrics := monitor.GetMetrics()
		if metrics.DatabaseQueries.MaxExecutionTime <= 0 {
			t.Error("Should record extreme execution time")
		}
	})
	
	t.Run("InvalidAlertOperations", func(t *testing.T) {
		// Test invalid alert operations
		err := alertManager.AcknowledgeAlert("nonexistent_id", "user", "comment")
		if err == nil {
			t.Error("Expected error for nonexistent alert ID")
		}
		
		err = alertManager.ResolveAlert("nonexistent_id", "user")
		if err == nil {
			t.Error("Expected error for nonexistent alert ID")
		}
	})
	
	t.Run("MemoryLimits", func(t *testing.T) {
		// Test memory limits by adding many data points
		for i := 0; i < 2000; i++ {
			monitor.RecordDatabaseQuery("SELECT * FROM test", 50*time.Millisecond, 1, nil)
			monitor.RecordResponseTime(100 * time.Millisecond)
			performanceTracker.TrackDatabaseQuery("SELECT * FROM test", 50*time.Millisecond, 1, true)
		}
		
		// Verify that memory limits are respected
		metrics := monitor.GetMetrics()
		if len(metrics.Performance.ResponseTimes) > 100 {
			t.Errorf("Response times should be limited to 100, got %d", len(metrics.Performance.ResponseTimes))
		}
		
		currentMetrics := performanceTracker.GetCurrentMetrics()
		if len(currentMetrics.DatabaseQueries) > 1000 {
			t.Errorf("Database queries should be limited to 1000, got %d", len(currentMetrics.DatabaseQueries))
		}
	})
}