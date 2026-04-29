package testing

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestTestExecutionMonitor(t *testing.T) {
	// This test requires a test database
	// Skip if no test database is available
	db, err := sql.Open("postgres", "postgres://testuser:testpass@localhost/testdb?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not accessible:", err)
	}

	monitor := NewTestExecutionMonitor(db)

	t.Run("RecordTestExecution", func(t *testing.T) {
		execution := &TestExecutionRecord{
			TestSuite:    "unit_tests",
			TestName:     "TestExample",
			Status:       "passed",
			Duration:     100 * time.Millisecond,
			StartTime:    time.Now().Add(-1 * time.Minute),
			EndTime:      time.Now(),
			Coverage:     95.5,
			Environment:  "test",
			Branch:       "main",
			CommitHash:   "abc123",
		}

		err := monitor.RecordTestExecution(execution)
		if err != nil {
			t.Fatalf("Failed to record test execution: %v", err)
		}

		if execution.ID == 0 {
			t.Error("Expected execution ID to be set")
		}
	})

	t.Run("GetTestMetrics", func(t *testing.T) {
		metrics, err := monitor.GetTestMetrics("24h")
		if err != nil {
			t.Fatalf("Failed to get test metrics: %v", err)
		}

		if metrics == nil {
			t.Error("Expected metrics to be returned")
		}

		if metrics.TimeRange != "24h" {
			t.Errorf("Expected time range '24h', got '%s'", metrics.TimeRange)
		}
	})

	t.Run("StartStopMonitoring", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := monitor.StartMonitoring(ctx)
		if err != nil {
			t.Fatalf("Failed to start monitoring: %v", err)
		}

		// Let it run briefly
		time.Sleep(100 * time.Millisecond)

		monitor.StopMonitoring()
	})
}

func TestFlakyTestDetector(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://testuser:testpass@localhost/testdb?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not accessible:", err)
	}

	detector := NewFlakyTestDetector(db)

	t.Run("UpdateTestFlakiness", func(t *testing.T) {
		execution := &TestExecutionRecord{
			TestSuite: "unit_tests",
			TestName:  "TestFlaky",
			Status:    "failed",
			StartTime: time.Now(),
		}

		err := detector.UpdateTestFlakiness(execution)
		if err != nil {
			t.Fatalf("Failed to update test flakiness: %v", err)
		}
	})

	t.Run("GetQuarantinedTests", func(t *testing.T) {
		tests, err := detector.GetQuarantinedTests()
		if err != nil {
			t.Fatalf("Failed to get quarantined tests: %v", err)
		}

		// Should return empty slice if no quarantined tests
		if tests == nil {
			t.Error("Expected non-nil slice")
		}
	})
}

func TestFailurePatternAnalyzer(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://testuser:testpass@localhost/testdb?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not accessible:", err)
	}

	analyzer := NewFailurePatternAnalyzer(db)

	t.Run("AnalyzePatterns", func(t *testing.T) {
		err := analyzer.AnalyzePatterns()
		if err != nil {
			t.Fatalf("Failed to analyze patterns: %v", err)
		}
	})

	t.Run("GetFailurePatterns", func(t *testing.T) {
		patterns, err := analyzer.GetFailurePatterns(10)
		if err != nil {
			t.Fatalf("Failed to get failure patterns: %v", err)
		}

		if patterns == nil {
			t.Error("Expected non-nil slice")
		}
	})
}

func TestCoverageTracker(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://testuser:testpass@localhost/testdb?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not accessible:", err)
	}

	tracker := NewCoverageTracker(db)

	t.Run("RecordCoverage", func(t *testing.T) {
		coverage := &CoverageTrend{
			Date:            time.Now(),
			CoveragePercent: 95.5,
			TestCount:       100,
			LinesTotal:      1000,
			LinesCovered:    955,
		}

		err := tracker.RecordCoverage(coverage)
		if err != nil {
			t.Fatalf("Failed to record coverage: %v", err)
		}
	})

	t.Run("GetCoverageTrends", func(t *testing.T) {
		trends, err := tracker.GetCoverageTrends(7)
		if err != nil {
			t.Fatalf("Failed to get coverage trends: %v", err)
		}

		if trends == nil {
			t.Error("Expected non-nil slice")
		}
	})
}

func TestQualityDashboard(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://testuser:testpass@localhost/testdb?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not accessible:", err)
	}

	dashboard := NewQualityDashboard(db)

	t.Run("GetDashboardData", func(t *testing.T) {
		ctx := context.Background()
		data, err := dashboard.GetDashboardData(ctx)
		if err != nil {
			t.Fatalf("Failed to get dashboard data: %v", err)
		}

		if data == nil {
			t.Error("Expected dashboard data to be returned")
		}

		if data.Overview == nil {
			t.Error("Expected overview to be calculated")
		}
	})
}

// Benchmark tests
func BenchmarkRecordTestExecution(b *testing.B) {
	db, err := sql.Open("postgres", "postgres://testuser:testpass@localhost/testdb?sslmode=disable")
	if err != nil {
		b.Skip("Test database not available:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		b.Skip("Test database not accessible:", err)
	}

	monitor := NewTestExecutionMonitor(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execution := &TestExecutionRecord{
			TestSuite:   "benchmark_tests",
			TestName:    "BenchmarkTest",
			Status:      "passed",
			Duration:    time.Duration(i) * time.Millisecond,
			StartTime:   time.Now().Add(-1 * time.Minute),
			EndTime:     time.Now(),
			Coverage:    95.5,
			Environment: "test",
		}

		monitor.RecordTestExecution(execution)
	}
}

func BenchmarkGetTestMetrics(b *testing.B) {
	db, err := sql.Open("postgres", "postgres://testuser:testpass@localhost/testdb?sslmode=disable")
	if err != nil {
		b.Skip("Test database not available:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		b.Skip("Test database not accessible:", err)
	}

	monitor := NewTestExecutionMonitor(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.GetTestMetrics("24h")
	}
}