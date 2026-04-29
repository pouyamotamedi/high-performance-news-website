package performance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"
)

// BaselineManager handles performance baseline storage and comparison
type BaselineManager struct {
	db *sql.DB
}

// PerformanceBaseline represents a stored performance baseline
type PerformanceBaseline struct {
	ID          int64                  `json:"id"`
	TestName    string                 `json:"test_name"`
	Version     string                 `json:"version"`
	Metrics     map[string]MetricData  `json:"metrics"`
	Environment string                 `json:"environment"`
	CreatedAt   time.Time             `json:"created_at"`
	IsActive    bool                  `json:"is_active"`
}

// MetricData represents performance metric statistics
type MetricData struct {
	Mean      float64 `json:"mean"`
	P95       float64 `json:"p95"`
	P99       float64 `json:"p99"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Count     int64   `json:"count"`
	StdDev    float64 `json:"std_dev"`
	Unit      string  `json:"unit"`
	Threshold float64 `json:"threshold"`
}

// RegressionResult represents the result of regression detection
type RegressionResult struct {
	TestName        string                    `json:"test_name"`
	CurrentVersion  string                    `json:"current_version"`
	BaselineVersion string                    `json:"baseline_version"`
	Regressions     []MetricRegression        `json:"regressions"`
	Improvements    []MetricImprovement       `json:"improvements"`
	OverallStatus   RegressionStatus          `json:"overall_status"`
	ComparedAt      time.Time                 `json:"compared_at"`
	Summary         RegressionSummary         `json:"summary"`
}

// MetricRegression represents a performance regression
type MetricRegression struct {
	MetricName      string  `json:"metric_name"`
	BaselineValue   float64 `json:"baseline_value"`
	CurrentValue    float64 `json:"current_value"`
	PercentChange   float64 `json:"percent_change"`
	Severity        string  `json:"severity"`
	Threshold       float64 `json:"threshold"`
	Unit            string  `json:"unit"`
}

// MetricImprovement represents a performance improvement
type MetricImprovement struct {
	MetricName      string  `json:"metric_name"`
	BaselineValue   float64 `json:"baseline_value"`
	CurrentValue    float64 `json:"current_value"`
	PercentChange   float64 `json:"percent_change"`
	Unit            string  `json:"unit"`
}

// RegressionSummary provides overall regression analysis
type RegressionSummary struct {
	TotalMetrics        int     `json:"total_metrics"`
	RegressedMetrics    int     `json:"regressed_metrics"`
	ImprovedMetrics     int     `json:"improved_metrics"`
	CriticalRegressions int     `json:"critical_regressions"`
	OverallScore        float64 `json:"overall_score"` // 0-100, higher is better
}

type RegressionStatus string

const (
	StatusPass     RegressionStatus = "pass"
	StatusWarning  RegressionStatus = "warning"
	StatusFail     RegressionStatus = "fail"
	StatusCritical RegressionStatus = "critical"
)

// NewBaselineManager creates a new baseline manager
func NewBaselineManager(db *sql.DB) *BaselineManager {
	return &BaselineManager{db: db}
}

// StoreBaseline stores a new performance baseline
func (bm *BaselineManager) StoreBaseline(baseline *PerformanceBaseline) error {
	// Deactivate previous baselines for the same test
	_, err := bm.db.Exec(`
		UPDATE performance_baselines 
		SET is_active = false 
		WHERE test_name = $1 AND environment = $2 AND is_active = true`,
		baseline.TestName, baseline.Environment)
	if err != nil {
		return fmt.Errorf("failed to deactivate previous baselines: %w", err)
	}

	// Store new baseline
	metricsJSON, err := json.Marshal(baseline.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	err = bm.db.QueryRow(`
		INSERT INTO performance_baselines (test_name, version, metrics, environment, is_active, created_at)
		VALUES ($1, $2, $3, $4, true, $5)
		RETURNING id`,
		baseline.TestName, baseline.Version, metricsJSON, baseline.Environment, time.Now()).
		Scan(&baseline.ID)

	if err != nil {
		return fmt.Errorf("failed to store baseline: %w", err)
	}

	log.Printf("Stored performance baseline for %s version %s", baseline.TestName, baseline.Version)
	return nil
}

// GetActiveBaseline retrieves the active baseline for a test
func (bm *BaselineManager) GetActiveBaseline(testName, environment string) (*PerformanceBaseline, error) {
	var baseline PerformanceBaseline
	var metricsJSON []byte

	err := bm.db.QueryRow(`
		SELECT id, test_name, version, metrics, environment, created_at, is_active
		FROM performance_baselines
		WHERE test_name = $1 AND environment = $2 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1`,
		testName, environment).
		Scan(&baseline.ID, &baseline.TestName, &baseline.Version, &metricsJSON,
			&baseline.Environment, &baseline.CreatedAt, &baseline.IsActive)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active baseline found for test %s in environment %s", testName, environment)
		}
		return nil, fmt.Errorf("failed to get baseline: %w", err)
	}

	err = json.Unmarshal(metricsJSON, &baseline.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	return &baseline, nil
}

// CompareWithBaseline compares current metrics with baseline and detects regressions
func (bm *BaselineManager) CompareWithBaseline(testName, currentVersion, environment string, currentMetrics map[string]MetricData) (*RegressionResult, error) {
	baseline, err := bm.GetActiveBaseline(testName, environment)
	if err != nil {
		return nil, fmt.Errorf("failed to get baseline: %w", err)
	}

	result := &RegressionResult{
		TestName:        testName,
		CurrentVersion:  currentVersion,
		BaselineVersion: baseline.Version,
		ComparedAt:      time.Now(),
		Regressions:     []MetricRegression{},
		Improvements:    []MetricImprovement{},
	}

	totalMetrics := 0
	regressedMetrics := 0
	improvedMetrics := 0
	criticalRegressions := 0
	overallScore := 100.0

	// Compare each metric
	for metricName, currentData := range currentMetrics {
		baselineData, exists := baseline.Metrics[metricName]
		if !exists {
			log.Printf("Warning: metric %s not found in baseline", metricName)
			continue
		}

		totalMetrics++

		// Calculate percentage change (using P95 as primary comparison metric)
		percentChange := ((currentData.P95 - baselineData.P95) / baselineData.P95) * 100

		// Determine if this is a regression or improvement
		// For response time metrics, higher is worse
		// For throughput metrics, lower is worse
		isResponseTimeMetric := isResponseTimeMetric(metricName)
		
		var regressionThreshold float64 = 10.0 // 10% degradation threshold
		if baselineData.Threshold > 0 {
			regressionThreshold = baselineData.Threshold
		}

		if isResponseTimeMetric {
			// For response time: increase is bad
			if percentChange > regressionThreshold {
				severity := bm.calculateSeverity(percentChange, regressionThreshold)
				regression := MetricRegression{
					MetricName:    metricName,
					BaselineValue: baselineData.P95,
					CurrentValue:  currentData.P95,
					PercentChange: percentChange,
					Severity:      severity,
					Threshold:     regressionThreshold,
					Unit:          currentData.Unit,
				}
				result.Regressions = append(result.Regressions, regression)
				regressedMetrics++

				if severity == "critical" {
					criticalRegressions++
				}

				// Reduce overall score based on severity
				scoreReduction := bm.calculateScoreReduction(percentChange, regressionThreshold)
				overallScore -= scoreReduction
			} else if percentChange < -5.0 { // 5% improvement threshold
				improvement := MetricImprovement{
					MetricName:    metricName,
					BaselineValue: baselineData.P95,
					CurrentValue:  currentData.P95,
					PercentChange: percentChange,
					Unit:          currentData.Unit,
				}
				result.Improvements = append(result.Improvements, improvement)
				improvedMetrics++
			}
		} else {
			// For throughput: decrease is bad
			if percentChange < -regressionThreshold {
				severity := bm.calculateSeverity(math.Abs(percentChange), regressionThreshold)
				regression := MetricRegression{
					MetricName:    metricName,
					BaselineValue: baselineData.P95,
					CurrentValue:  currentData.P95,
					PercentChange: percentChange,
					Severity:      severity,
					Threshold:     regressionThreshold,
					Unit:          currentData.Unit,
				}
				result.Regressions = append(result.Regressions, regression)
				regressedMetrics++

				if severity == "critical" {
					criticalRegressions++
				}

				scoreReduction := bm.calculateScoreReduction(math.Abs(percentChange), regressionThreshold)
				overallScore -= scoreReduction
			} else if percentChange > 5.0 { // 5% improvement threshold
				improvement := MetricImprovement{
					MetricName:    metricName,
					BaselineValue: baselineData.P95,
					CurrentValue:  currentData.P95,
					PercentChange: percentChange,
					Unit:          currentData.Unit,
				}
				result.Improvements = append(result.Improvements, improvement)
				improvedMetrics++
			}
		}
	}

	// Ensure score doesn't go below 0
	if overallScore < 0 {
		overallScore = 0
	}

	result.Summary = RegressionSummary{
		TotalMetrics:        totalMetrics,
		RegressedMetrics:    regressedMetrics,
		ImprovedMetrics:     improvedMetrics,
		CriticalRegressions: criticalRegressions,
		OverallScore:        overallScore,
	}

	// Determine overall status
	result.OverallStatus = bm.determineOverallStatus(criticalRegressions, regressedMetrics, totalMetrics, overallScore)

	// Store comparison result
	err = bm.storeRegressionResult(result)
	if err != nil {
		log.Printf("Warning: failed to store regression result: %v", err)
	}

	return result, nil
}

// calculateSeverity determines the severity of a regression
func (bm *BaselineManager) calculateSeverity(percentChange, threshold float64) string {
	if percentChange > threshold*3 { // 3x threshold
		return "critical"
	} else if percentChange > threshold*2 { // 2x threshold
		return "high"
	} else if percentChange > threshold*1.5 { // 1.5x threshold
		return "medium"
	}
	return "low"
}

// calculateScoreReduction calculates how much to reduce the overall score
func (bm *BaselineManager) calculateScoreReduction(percentChange, threshold float64) float64 {
	if percentChange > threshold*3 {
		return 25.0 // Critical regression
	} else if percentChange > threshold*2 {
		return 15.0 // High regression
	} else if percentChange > threshold*1.5 {
		return 10.0 // Medium regression
	}
	return 5.0 // Low regression
}

// determineOverallStatus determines the overall regression status
func (bm *BaselineManager) determineOverallStatus(critical, regressed, total int, score float64) RegressionStatus {
	if critical > 0 {
		return StatusCritical
	}
	
	if score < 70 {
		return StatusFail
	}
	
	if regressed > 0 || score < 85 {
		return StatusWarning
	}
	
	return StatusPass
}

// isResponseTimeMetric determines if a metric represents response time (higher is worse)
func isResponseTimeMetric(metricName string) bool {
	responseTimeMetrics := []string{
		"http_req_duration", "article_creation_duration", "database_query_duration",
		"api_response_time", "db_connection_time", "query_execution_time",
		"cache_invalidation_time", "database_insert_time",
	}
	
	for _, rtMetric := range responseTimeMetrics {
		if metricName == rtMetric || 
		   fmt.Sprintf("%s_", rtMetric) == metricName[:len(rtMetric)+1] {
			return true
		}
	}
	return false
}

// storeRegressionResult stores the regression analysis result
func (bm *BaselineManager) storeRegressionResult(result *RegressionResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal regression result: %w", err)
	}

	_, err = bm.db.Exec(`
		INSERT INTO performance_regression_results 
		(test_name, current_version, baseline_version, result_data, overall_status, overall_score, compared_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		result.TestName, result.CurrentVersion, result.BaselineVersion,
		resultJSON, string(result.OverallStatus), result.Summary.OverallScore, result.ComparedAt)

	return err
}

// GetRecentRegressionResults gets recent regression results for analysis
func (bm *BaselineManager) GetRecentRegressionResults(testName string, limit int) ([]RegressionResult, error) {
	rows, err := bm.db.Query(`
		SELECT test_name, current_version, baseline_version, result_data, overall_status, overall_score, compared_at
		FROM performance_regression_results
		WHERE test_name = $1
		ORDER BY compared_at DESC
		LIMIT $2`,
		testName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query regression results: %w", err)
	}
	defer rows.Close()

	var results []RegressionResult
	for rows.Next() {
		var result RegressionResult
		var resultJSON []byte
		var status string

		err := rows.Scan(&result.TestName, &result.CurrentVersion, &result.BaselineVersion,
			&resultJSON, &status, &result.Summary.OverallScore, &result.ComparedAt)
		if err != nil {
			continue
		}

		result.OverallStatus = RegressionStatus(status)

		err = json.Unmarshal(resultJSON, &result)
		if err != nil {
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// UpdateBaseline updates an existing baseline with new data
func (bm *BaselineManager) UpdateBaseline(testName, version, environment string, newMetrics map[string]MetricData) error {
	// Get current baseline
	baseline, err := bm.GetActiveBaseline(testName, environment)
	if err != nil {
		// If no baseline exists, create a new one
		newBaseline := &PerformanceBaseline{
			TestName:    testName,
			Version:     version,
			Metrics:     newMetrics,
			Environment: environment,
		}
		return bm.StoreBaseline(newBaseline)
	}

	// Merge metrics (weighted average with existing baseline)
	mergedMetrics := make(map[string]MetricData)
	for metricName, newData := range newMetrics {
		if existingData, exists := baseline.Metrics[metricName]; exists {
			// Calculate weighted average (70% existing, 30% new)
			merged := MetricData{
				Mean:      existingData.Mean*0.7 + newData.Mean*0.3,
				P95:       existingData.P95*0.7 + newData.P95*0.3,
				P99:       existingData.P99*0.7 + newData.P99*0.3,
				Min:       math.Min(existingData.Min, newData.Min),
				Max:       math.Max(existingData.Max, newData.Max),
				Count:     existingData.Count + newData.Count,
				StdDev:    math.Sqrt((existingData.StdDev*existingData.StdDev*0.7) + (newData.StdDev*newData.StdDev*0.3)),
				Unit:      newData.Unit,
				Threshold: existingData.Threshold, // Keep existing threshold
			}
			mergedMetrics[metricName] = merged
		} else {
			// New metric, add as-is
			mergedMetrics[metricName] = newData
		}
	}

	// Add any existing metrics not in new data
	for metricName, existingData := range baseline.Metrics {
		if _, exists := mergedMetrics[metricName]; !exists {
			mergedMetrics[metricName] = existingData
		}
	}

	// Create updated baseline
	updatedBaseline := &PerformanceBaseline{
		TestName:    testName,
		Version:     version,
		Metrics:     mergedMetrics,
		Environment: environment,
	}

	return bm.StoreBaseline(updatedBaseline)
}

// DeleteBaseline removes a baseline
func (bm *BaselineManager) DeleteBaseline(testName, environment string) error {
	_, err := bm.db.Exec(`
		DELETE FROM performance_baselines 
		WHERE test_name = $1 AND environment = $2`,
		testName, environment)
	
	if err != nil {
		return fmt.Errorf("failed to delete baseline: %w", err)
	}

	log.Printf("Deleted baseline for %s in environment %s", testName, environment)
	return nil
}

// ListBaselines returns all baselines for a test
func (bm *BaselineManager) ListBaselines(testName string) ([]PerformanceBaseline, error) {
	rows, err := bm.db.Query(`
		SELECT id, test_name, version, metrics, environment, created_at, is_active
		FROM performance_baselines
		WHERE test_name = $1
		ORDER BY created_at DESC`,
		testName)
	if err != nil {
		return nil, fmt.Errorf("failed to list baselines: %w", err)
	}
	defer rows.Close()

	var baselines []PerformanceBaseline
	for rows.Next() {
		var baseline PerformanceBaseline
		var metricsJSON []byte

		err := rows.Scan(&baseline.ID, &baseline.TestName, &baseline.Version, 
			&metricsJSON, &baseline.Environment, &baseline.CreatedAt, &baseline.IsActive)
		if err != nil {
			continue
		}

		err = json.Unmarshal(metricsJSON, &baseline.Metrics)
		if err != nil {
			continue
		}

		baselines = append(baselines, baseline)
	}

	return baselines, nil
}

// GetPerformanceTrend analyzes performance trends over time
func (bm *BaselineManager) GetPerformanceTrend(testName string, metricName string, days int) ([]TrendPoint, error) {
	rows, err := bm.db.Query(`
		SELECT compared_at, overall_score, result_data
		FROM performance_regression_results
		WHERE test_name = $1 AND compared_at > NOW() - INTERVAL '%d days'
		ORDER BY compared_at ASC`,
		testName, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get trend data: %w", err)
	}
	defer rows.Close()

	var trendPoints []TrendPoint
	for rows.Next() {
		var timestamp time.Time
		var score float64
		var resultJSON []byte

		err := rows.Scan(&timestamp, &score, &resultJSON)
		if err != nil {
			continue
		}

		var result RegressionResult
		err = json.Unmarshal(resultJSON, &result)
		if err != nil {
			continue
		}

		// Find the specific metric value
		var metricValue float64
		for _, regression := range result.Regressions {
			if regression.MetricName == metricName {
				metricValue = regression.CurrentValue
				break
			}
		}
		for _, improvement := range result.Improvements {
			if improvement.MetricName == metricName {
				metricValue = improvement.CurrentValue
				break
			}
		}

		trendPoint := TrendPoint{
			Timestamp:   timestamp,
			MetricValue: metricValue,
			Score:       score,
		}
		trendPoints = append(trendPoints, trendPoint)
	}

	return trendPoints, nil
}

// TrendPoint represents a point in performance trend analysis
type TrendPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	MetricValue float64   `json:"metric_value"`
	Score       float64   `json:"score"`
}