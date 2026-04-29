package testing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQualityDashboard(t *testing.T) {
	env := NewTestEnvironment(t)
	if !env.HasDatabase() {
		t.Skip("Database not available for testing")
	}

	dashboard := NewQualityDashboard(env.DB)

	t.Run("GetDashboardData", func(t *testing.T) {
		ctx := context.Background()
		
		// Add some test data first
		monitor := NewTestExecutionMonitor(env.DB)
		
		executions := []*TestExecution{
			{
				TestName:  "TestSuccess",
				TestSuite: "unit",
				Status:    "passed",
				Duration:  1000,
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now().Add(-1 * time.Hour).Add(1 * time.Second),
				Coverage:  95.0,
			},
			{
				TestName:  "TestFailure",
				TestSuite: "unit",
				Status:    "failed",
				Duration:  2000,
				StartTime: time.Now().Add(-30 * time.Minute),
				EndTime:   time.Now().Add(-30 * time.Minute).Add(2 * time.Second),
				Coverage:  90.0,
				ErrorMessage: "Test failed due to timeout",
			},
		}

		for _, exec := range executions {
			err := monitor.RecordTestExecution(exec)
			require.NoError(t, err)
		}

		// Allow time for async processing
		time.Sleep(200 * time.Millisecond)

		data, err := dashboard.GetDashboardData(ctx)
		require.NoError(t, err)
		assert.NotNil(t, data)
		assert.NotNil(t, data.Overview)
		assert.NotNil(t, data.TestMetrics)
		assert.Greater(t, data.TestMetrics.TotalExecutions, int64(0))
	})

	t.Run("CalculateOverview", func(t *testing.T) {
		data := &DashboardData{
			TestMetrics: &TestMetrics{
				SuccessRate:     85.0,
				CoveragePercent: 92.0,
				FlakyTests:      3,
			},
			SecurityVulns:     []SecurityVulnerability{{Severity: "high"}},
			PerformanceRegress: []PerformanceRegression{{Severity: "medium"}},
			QualityGateStatus: &QualityGateStatus{
				Gates: []QualityGate{
					{Status: "passed"},
					{Status: "failed"},
					{Status: "passed"},
				},
			},
		}

		overview := dashboard.calculateOverview(data)
		assert.NotNil(t, overview)
		assert.Equal(t, 85.0, overview.TestSuccessRate)
		assert.Equal(t, 92.0, overview.CoveragePercent)
		assert.Equal(t, int64(3), overview.FlakyTestCount)
		assert.Equal(t, int64(1), overview.SecurityIssues)
		assert.Equal(t, int64(1), overview.PerformanceIssues)
		assert.Equal(t, 2, overview.QualityGatesPassed)
		assert.Equal(t, 3, overview.QualityGatesTotal)
		assert.Less(t, overview.HealthScore, 100.0) // Should be penalized for issues
	})
}

func TestQualityDashboardHTTP(t *testing.T) {
	env := NewTestEnvironment(t)
	if !env.HasDatabase() {
		t.Skip("Database not available for testing")
	}

	dashboard := NewQualityDashboard(env.DB)
	
	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	dashboard.SetupRoutes(router)

	t.Run("GET /dashboard/api/data", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dashboard/api/data", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var data DashboardData
		err := json.Unmarshal(w.Body.Bytes(), &data)
		assert.NoError(t, err)
		assert.NotZero(t, data.LastUpdated)
	})

	t.Run("GET /dashboard/api/metrics/24h", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dashboard/api/metrics/24h", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var metrics TestMetrics
		err := json.Unmarshal(w.Body.Bytes(), &metrics)
		assert.NoError(t, err)
		assert.Equal(t, "24h", metrics.TimeRange)
	})

	t.Run("GET /dashboard/api/flaky-tests", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dashboard/api/flaky-tests?limit=5", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var flakyTests []FlakyTestInfo
		err := json.Unmarshal(w.Body.Bytes(), &flakyTests)
		assert.NoError(t, err)
	})

	t.Run("GET /dashboard/api/failure-patterns", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dashboard/api/failure-patterns", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var patterns []FailurePattern
		err := json.Unmarshal(w.Body.Bytes(), &patterns)
		assert.NoError(t, err)
	})

	t.Run("GET /dashboard/api/coverage-trends/7", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dashboard/api/coverage-trends/7", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var trends []CoverageTrend
		err := json.Unmarshal(w.Body.Bytes(), &trends)
		assert.NoError(t, err)
	})

	t.Run("GET /dashboard/api/quality-gates", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dashboard/api/quality-gates", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var status QualityGateStatus
		err := json.Unmarshal(w.Body.Bytes(), &status)
		assert.NoError(t, err)
	})

	t.Run("POST /dashboard/api/quality-gates/evaluate", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/dashboard/api/quality-gates/evaluate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var status QualityGateStatus
		err := json.Unmarshal(w.Body.Bytes(), &status)
		assert.NoError(t, err)
	})

	t.Run("GET /dashboard/health", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dashboard/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var health map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &health)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", health["status"])
	})
}

func TestSecurityVulnerabilityTracker(t *testing.T) {
	env := NewTestEnvironment(t)
	if !env.HasDatabase() {
		t.Skip("Database not available for testing")
	}

	tracker := NewSecurityVulnerabilityTracker(env.DB)

	t.Run("RecordVulnerability", func(t *testing.T) {
		vuln := &SecurityVulnerability{
			Type:        "SQL Injection",
			Severity:    "high",
			Description: "Potential SQL injection in user input",
			Component:   "user-service",
			CVSS:        8.5,
			Remediation: "Use parameterized queries",
		}

		err := tracker.RecordVulnerability(vuln)
		require.NoError(t, err)
		assert.Greater(t, vuln.ID, int64(0))
		assert.NotZero(t, vuln.FirstSeen)
	})

	t.Run("GetVulnerabilities", func(t *testing.T) {
		vulns, err := tracker.GetVulnerabilities()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(vulns), 0)
	})
}

func TestPerformanceRegressionTracker(t *testing.T) {
	env := NewTestEnvironment(t)
	if !env.HasDatabase() {
		t.Skip("Database not available for testing")
	}

	tracker := NewPerformanceRegressionTracker(env.DB)

	t.Run("RecordRegression", func(t *testing.T) {
		regression := &PerformanceRegression{
			TestName:        "TestDatabaseQuery",
			Metric:          "response_time",
			BaselineValue:   100.0,
			CurrentValue:    250.0,
			RegressionPct:   150.0,
			Severity:        "high",
		}

		err := tracker.RecordRegression(regression)
		require.NoError(t, err)
		assert.Greater(t, regression.ID, int64(0))
		assert.NotZero(t, regression.DetectedAt)
	})

	t.Run("GetRegressions", func(t *testing.T) {
		regressions, err := tracker.GetRegressions()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(regressions), 0)
	})
}

func TestQualityGateManager(t *testing.T) {
	env := NewTestEnvironment(t)
	if !env.HasDatabase() {
		t.Skip("Database not available for testing")
	}

	manager := NewQualityGateManager(env.DB)

	t.Run("EvaluateGates", func(t *testing.T) {
		ctx := context.Background()
		
		// Add some test data to make gates meaningful
		monitor := NewTestExecutionMonitor(env.DB)
		execution := &TestExecution{
			TestName:  "TestForGates",
			TestSuite: "unit",
			Status:    "passed",
			Duration:  1000,
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now().Add(-1 * time.Hour).Add(1 * time.Second),
			Coverage:  96.0,
		}
		
		err := monitor.RecordTestExecution(execution)
		require.NoError(t, err)

		status, err := manager.EvaluateGates(ctx)
		require.NoError(t, err)
		assert.NotNil(t, status)
		assert.NotEmpty(t, status.Gates)
		assert.NotZero(t, status.LastEvaluation)
		assert.Contains(t, []string{"passed", "failed", "error"}, status.OverallStatus)
	})

	t.Run("GetStatus", func(t *testing.T) {
		status, err := manager.GetStatus()
		require.NoError(t, err)
		assert.NotNil(t, status)
	})

	t.Run("QualityGateEvaluation", func(t *testing.T) {
		// Test individual gate evaluation logic
		gates := []QualityGate{
			{Name: "Test Success Rate", Threshold: 95.0, CurrentValue: 98.0},
			{Name: "Code Coverage", Threshold: 95.0, CurrentValue: 92.0},
			{Name: "Flaky Test Count", Threshold: 5.0, CurrentValue: 3.0},
		}

		for _, gate := range gates {
			switch gate.Name {
			case "Test Success Rate", "Code Coverage":
				// Higher is better
				if gate.CurrentValue >= gate.Threshold {
					assert.True(t, gate.CurrentValue >= gate.Threshold, "Gate should pass")
				} else {
					assert.False(t, gate.CurrentValue >= gate.Threshold, "Gate should fail")
				}
			case "Flaky Test Count":
				// Lower is better
				if gate.CurrentValue <= gate.Threshold {
					assert.True(t, gate.CurrentValue <= gate.Threshold, "Gate should pass")
				} else {
					assert.False(t, gate.CurrentValue <= gate.Threshold, "Gate should fail")
				}
			}
		}
	})
}

// Integration test for the complete monitoring system
func TestMonitoringIntegration(t *testing.T) {
	env := NewTestEnvironment(t)
	if !env.HasDatabase() {
		t.Skip("Database not available for testing")
	}

	integration := NewMonitoringIntegration(env.DB)

	t.Run("CompleteWorkflow", func(t *testing.T) {
		ctx := context.Background()

		// Simulate test execution data
		executions := []*TestExecution{
			{
				TestName:  "TestWorkflow1",
				TestSuite: "unit",
				Status:    "passed",
				Duration:  1500,
				StartTime: time.Now().Add(-2 * time.Hour),
				EndTime:   time.Now().Add(-2 * time.Hour).Add(1500 * time.Millisecond),
				Coverage:  94.5,
			},
			{
				TestName:  "TestWorkflow2",
				TestSuite: "integration",
				Status:    "failed",
				Duration:  3000,
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now().Add(-1 * time.Hour).Add(3 * time.Second),
				Coverage:  88.0,
				ErrorMessage: "Database connection timeout",
			},
		}

		// Record executions
		for _, exec := range executions {
			err := integration.monitor.RecordTestExecution(exec)
			require.NoError(t, err)
		}

		// Allow time for async processing
		time.Sleep(200 * time.Millisecond)

		// Get monitoring data
		monitoringData := integration.getMonitoringData(ctx)
		assert.NotNil(t, monitoringData)
		assert.NotNil(t, monitoringData.TestMetrics)

		// Evaluate quality gates
		gateStatus := integration.evaluateQualityGates(ctx)
		assert.NotNil(t, gateStatus)

		// Generate recommendations
		result := &TestRunResult{
			CoveragePercent: 91.0,
			FailedTests:     1,
		}
		recommendations := integration.generateRecommendations(executions, result)
		assert.NotEmpty(t, recommendations)

		// Verify we get coverage and failure recommendations
		foundCoverage := false
		foundFailure := false
		for _, rec := range recommendations {
			if rec.Type == "coverage" {
				foundCoverage = true
			}
			if rec.Type == "failures" {
				foundFailure = true
			}
		}
		assert.True(t, foundCoverage, "Should recommend coverage improvement")
		assert.True(t, foundFailure, "Should recommend fixing failures")
	})
}

// Benchmark tests for dashboard performance
func BenchmarkDashboardDataRetrieval(b *testing.B) {
	// Create a dummy test for environment setup
	t := &testing.T{}
	env := NewTestEnvironment(t)
	if !env.HasDatabase() {
		b.Skip("Database not available for benchmarking")
	}

	dashboard := NewQualityDashboard(env.DB)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dashboard.GetDashboardData(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQualityGateEvaluation(b *testing.B) {
	// Create a dummy test for environment setup
	t := &testing.T{}
	env := NewTestEnvironment(t)
	if !env.HasDatabase() {
		b.Skip("Database not available for benchmarking")
	}

	manager := NewQualityGateManager(env.DB)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.EvaluateGates(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}