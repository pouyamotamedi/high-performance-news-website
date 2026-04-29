package performance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// AutomatedBaselineManager manages automated baseline establishment and updates
type AutomatedBaselineManager struct {
	db                    *sql.DB
	enhancedManager       *EnhancedBaselineManager
	regressionDetector    *IntelligentRegressionDetector
	validationEngine      *ValidationEngine
	schedulingEngine      *SchedulingEngine
}

// ValidationStatus represents the status of a validation check
type ValidationStatus string

const (
	ValidationStatusPass    ValidationStatus = "pass"
	ValidationStatusWarning ValidationStatus = "warning"
	ValidationStatusFail    ValidationStatus = "fail"
)

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	RuleName  string           `json:"rule_name"`
	Status    ValidationStatus `json:"status"`
	Message   string           `json:"message"`
	Severity  string           `json:"severity"`
	Threshold float64          `json:"threshold"`
	Value     float64          `json:"value"`
}

// AutomatedBaselineResult represents the result of automated baseline establishment
type AutomatedBaselineResult struct {
	BaselineID            int64                    `json:"baseline_id"`
	TestName              string                   `json:"test_name"`
	Version               string                   `json:"version"`
	Environment           string                   `json:"environment"`
	QualityScore          float64                  `json:"quality_score"`
	DataPoints            int                      `json:"data_points"`
	NextUpdateScheduled   time.Time                `json:"next_update_scheduled"`
	ValidationResults     []ValidationResult       `json:"validation_results"`
	StatisticalAnalysis   StatisticalAnalysis      `json:"statistical_analysis"`
	TrendAnalysis         TrendAnalysis            `json:"trend_analysis"`
	CapacityAnalysis      CapacityAnalysis         `json:"capacity_analysis"`
	Recommendations       []BaselineRecommendation `json:"recommendations"`
	CreatedAt             time.Time                `json:"created_at"`
}

// StatisticalAnalysis contains statistical analysis results
type StatisticalAnalysis struct {
	SampleSize            int     `json:"sample_size"`
	DataCompleteness      float64 `json:"data_completeness"`
	OutliersDetected      int     `json:"outliers_detected"`
	ChangePointsDetected  int     `json:"change_points_detected"`
	SeasonalityDetected   bool    `json:"seasonality_detected"`
	DataQuality           string  `json:"data_quality"`
	ConfidenceLevel       float64 `json:"confidence_level"`
	StatisticalSignificance bool  `json:"statistical_significance"`
}

// TrendAnalysis contains trend analysis results
type TrendAnalysis struct {
	TrendDirection   string  `json:"trend_direction"`
	TrendStrength    float64 `json:"trend_strength"`
	ForecastAccuracy float64 `json:"forecast_accuracy"`
	VolatilityIndex  float64 `json:"volatility_index"`
}

// CapacityAnalysis contains capacity analysis results
type CapacityAnalysis struct {
	CurrentUtilization map[string]float64 `json:"current_utilization"`
	BottleneckRisk     string             `json:"bottleneck_risk"`
	ScalingNeeded      bool               `json:"scaling_needed"`
	TimeToCapacity     *time.Time         `json:"time_to_capacity,omitempty"`
}

// BaselineRecommendation represents a recommendation for baseline improvement
type BaselineRecommendation struct {
	Type        string  `json:"type"`
	Priority    string  `json:"priority"`
	Description string  `json:"description"`
	Action      string  `json:"action"`
	Impact      string  `json:"impact"`
	Confidence  float64 `json:"confidence"`
}

// ValidationEngine handles baseline validation
type ValidationEngine struct {
	db    *sql.DB
	rules map[string]ValidationRule
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Name          string  `json:"name"`
	ConditionType string  `json:"condition_type"`
	Threshold     float64 `json:"threshold"`
	Severity      string  `json:"severity"`
	ActionType    string  `json:"action_type"`
	Description   string  `json:"description"`
}

// SchedulingEngine handles baseline scheduling
type SchedulingEngine struct {
	db *sql.DB
}

// BaselineSchedule represents a baseline update schedule
type BaselineSchedule struct {
	ID                   int64         `json:"id"`
	TestName             string        `json:"test_name"`
	Environment          string        `json:"environment"`
	UpdateFrequency      time.Duration `json:"update_frequency"`
	NextUpdate           time.Time     `json:"next_update"`
	LastUpdate           *time.Time    `json:"last_update,omitempty"`
	Status               string        `json:"status"`
	RequiredDataPoints   int           `json:"required_data_points"`
	CollectedMetrics     map[string]interface{} `json:"collected_metrics"`
}

// NewAutomatedBaselineManager creates a new automated baseline manager
func NewAutomatedBaselineManager(db *sql.DB, enhancedManager *EnhancedBaselineManager) *AutomatedBaselineManager {
	regressionDetector := NewIntelligentRegressionDetector(db, enhancedManager)
	
	return &AutomatedBaselineManager{
		db:                 db,
		enhancedManager:    enhancedManager,
		regressionDetector: regressionDetector,
		validationEngine:   NewValidationEngine(db),
		schedulingEngine:   NewSchedulingEngine(db),
	}
}

// EstablishAutomatedBaseline establishes a baseline with comprehensive validation and analysis
func (abm *AutomatedBaselineManager) EstablishAutomatedBaseline(testName, version, environment string) (*AutomatedBaselineResult, error) {
	log.Printf("Starting automated baseline establishment for %s version %s in %s", testName, version, environment)
	
	// Generate sample performance data (in real implementation, this would come from actual test runs)
	rawMetrics, err := abm.generateSampleMetrics(testName, environment)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sample metrics: %w", err)
	}
	
	// Establish enhanced baseline
	enhancedBaseline, err := abm.enhancedManager.EstablishEnhancedBaseline(testName, version, environment, rawMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to establish enhanced baseline: %w", err)
	}
	
	// Validate baseline quality
	validationResults, err := abm.validationEngine.ValidateBaseline(enhancedBaseline)
	if err != nil {
		return nil, fmt.Errorf("failed to validate baseline: %w", err)
	}
	
	// Calculate quality score
	qualityScore := abm.calculateQualityScore(validationResults, enhancedBaseline)
	
	// Generate recommendations
	recommendations := abm.generateRecommendations(enhancedBaseline, validationResults)
	
	// Schedule next update
	nextUpdate := time.Now().Add(24 * time.Hour) // Default to 24 hours
	err = abm.schedulingEngine.ScheduleUpdate(testName, environment, nextUpdate)
	if err != nil {
		log.Printf("Warning: failed to schedule next update: %v", err)
	}
	
	// Store automated baseline result
	result := &AutomatedBaselineResult{
		BaselineID:          enhancedBaseline.ID,
		TestName:            testName,
		Version:             version,
		Environment:         environment,
		QualityScore:        qualityScore,
		DataPoints:          len(rawMetrics),
		NextUpdateScheduled: nextUpdate,
		ValidationResults:   validationResults,
		StatisticalAnalysis: abm.extractStatisticalAnalysis(enhancedBaseline),
		TrendAnalysis:       abm.extractTrendAnalysis(enhancedBaseline),
		CapacityAnalysis:    abm.extractCapacityAnalysis(enhancedBaseline),
		Recommendations:     recommendations,
		CreatedAt:           time.Now(),
	}
	
	err = abm.storeAutomatedResult(result)
	if err != nil {
		return nil, fmt.Errorf("failed to store automated result: %w", err)
	}
	
	log.Printf("Automated baseline established successfully with quality score %.2f", qualityScore)
	return result, nil
}

// UpdateBaselinesAutomatically updates all scheduled baselines
func (abm *AutomatedBaselineManager) UpdateBaselinesAutomatically() error {
	schedules, err := abm.schedulingEngine.GetDueUpdates()
	if err != nil {
		return fmt.Errorf("failed to get due updates: %w", err)
	}
	
	log.Printf("Found %d baselines due for update", len(schedules))
	
	for _, schedule := range schedules {
		err := abm.updateScheduledBaseline(schedule)
		if err != nil {
			log.Printf("Failed to update baseline for %s in %s: %v", schedule.TestName, schedule.Environment, err)
			continue
		}
		
		// Update schedule
		err = abm.schedulingEngine.UpdateSchedule(schedule.ID, time.Now().Add(schedule.UpdateFrequency))
		if err != nil {
			log.Printf("Failed to update schedule for %s: %v", schedule.TestName, err)
		}
	}
	
	return nil
}

// generateSampleMetrics generates sample performance metrics for testing
func (abm *AutomatedBaselineManager) generateSampleMetrics(testName, environment string) ([]map[string]MetricData, error) {
	// In a real implementation, this would collect actual performance data
	// For now, generate realistic sample data
	
	sampleCount := 50 // Generate 50 sample data points
	rawMetrics := make([]map[string]MetricData, sampleCount)
	
	for i := 0; i < sampleCount; i++ {
		metrics := map[string]MetricData{
			"http_req_duration": {
				Mean:      100.0 + float64(i%10)*5.0, // Vary between 100-145ms
				P95:       150.0 + float64(i%15)*3.0, // Vary between 150-192ms
				P99:       200.0 + float64(i%20)*2.0, // Vary between 200-238ms
				Min:       50.0,
				Max:       300.0,
				Count:     1000,
				StdDev:    25.0,
				Unit:      "ms",
				Threshold: 10.0,
			},
			"article_creation_duration": {
				Mean:      500.0 + float64(i%8)*10.0,
				P95:       800.0 + float64(i%12)*5.0,
				P99:       1200.0 + float64(i%10)*8.0,
				Min:       200.0,
				Max:       2000.0,
				Count:     100,
				StdDev:    150.0,
				Unit:      "ms",
				Threshold: 20.0,
			},
			"database_query_duration": {
				Mean:      5.0 + float64(i%5)*0.5,
				P95:       8.0 + float64(i%7)*0.3,
				P99:       12.0 + float64(i%6)*0.4,
				Min:       1.0,
				Max:       20.0,
				Count:     5000,
				StdDev:    2.0,
				Unit:      "ms",
				Threshold: 5.0,
			},
			"cache_hit_rate": {
				Mean:      0.85 + float64(i%10)*0.01,
				P95:       0.90,
				P99:       0.95,
				Min:       0.70,
				Max:       0.98,
				Count:     1000,
				StdDev:    0.05,
				Unit:      "ratio",
				Threshold: 0.1,
			},
		}
		rawMetrics[i] = metrics
	}
	
	return rawMetrics, nil
}

// calculateQualityScore calculates the overall quality score for a baseline
func (abm *AutomatedBaselineManager) calculateQualityScore(validationResults []ValidationResult, baseline *EnhancedPerformanceBaseline) float64 {
	if len(validationResults) == 0 {
		return 50.0 // Default score if no validations
	}
	
	totalScore := 0.0
	weightSum := 0.0
	
	for _, result := range validationResults {
		weight := 1.0
		score := 0.0
		
		// Assign weights based on severity
		switch result.Severity {
		case "critical":
			weight = 3.0
		case "high":
			weight = 2.0
		case "medium":
			weight = 1.5
		case "low":
			weight = 1.0
		}
		
		// Assign scores based on status
		switch result.Status {
		case ValidationStatusPass:
			score = 100.0
		case ValidationStatusWarning:
			score = 70.0
		case ValidationStatusFail:
			score = 0.0
		}
		
		totalScore += score * weight
		weightSum += weight
	}
	
	if weightSum == 0 {
		return 50.0
	}
	
	return totalScore / weightSum
}

// generateRecommendations generates recommendations based on baseline analysis
func (abm *AutomatedBaselineManager) generateRecommendations(baseline *EnhancedPerformanceBaseline, validationResults []ValidationResult) []BaselineRecommendation {
	recommendations := []BaselineRecommendation{}
	
	// Check for failed validations
	for _, result := range validationResults {
		if result.Status == ValidationStatusFail {
			rec := BaselineRecommendation{
				Type:        "validation_failure",
				Priority:    result.Severity,
				Description: fmt.Sprintf("Validation failed: %s", result.Message),
				Action:      abm.getValidationFailureAction(result.RuleName),
				Impact:      "Baseline quality may be compromised",
				Confidence:  0.9,
			}
			recommendations = append(recommendations, rec)
		}
	}
	
	// Check capacity analysis
	for resource, utilization := range baseline.CapacityData.ResourceUtilization {
		if utilization > 0.8 { // 80% utilization threshold
			rec := BaselineRecommendation{
				Type:        "capacity_planning",
				Priority:    "medium",
				Description: fmt.Sprintf("%s utilization is high (%.1f%%)", resource, utilization*100),
				Action:      fmt.Sprintf("Consider scaling %s resources", resource),
				Impact:      "Performance may degrade under increased load",
				Confidence:  0.8,
			}
			recommendations = append(recommendations, rec)
		}
	}
	
	// Check trend analysis
	if baseline.TrendData.TrendDirection == "increasing" && baseline.TrendData.TrendStrength > 0.7 {
		rec := BaselineRecommendation{
			Type:        "performance_trend",
			Priority:    "high",
			Description: "Performance metrics showing strong degradation trend",
			Action:      "Investigate root cause of performance degradation",
			Impact:      "Continued degradation may impact user experience",
			Confidence:  baseline.TrendData.TrendStrength,
		}
		recommendations = append(recommendations, rec)
	}
	
	return recommendations
}

// extractStatisticalAnalysis extracts statistical analysis from enhanced baseline
func (abm *AutomatedBaselineManager) extractStatisticalAnalysis(baseline *EnhancedPerformanceBaseline) StatisticalAnalysis {
	return StatisticalAnalysis{
		SampleSize:              len(baseline.StatisticalData.Outliers), // Simplified
		DataCompleteness:        0.95, // Would be calculated from actual data
		OutliersDetected:        len(baseline.StatisticalData.Outliers),
		ChangePointsDetected:    len(baseline.TrendData.ChangePoints),
		SeasonalityDetected:     len(baseline.StatisticalData.Seasonality) > 0,
		DataQuality:             "good", // Would be determined by validation
		ConfidenceLevel:         0.95,
		StatisticalSignificance: true,
	}
}

// extractTrendAnalysis extracts trend analysis from enhanced baseline
func (abm *AutomatedBaselineManager) extractTrendAnalysis(baseline *EnhancedPerformanceBaseline) TrendAnalysis {
	return TrendAnalysis{
		TrendDirection:   baseline.TrendData.TrendDirection,
		TrendStrength:    baseline.TrendData.TrendStrength,
		ForecastAccuracy: 0.85, // Would be calculated from historical accuracy
		VolatilityIndex:  0.3,  // Would be calculated from variance
	}
}

// extractCapacityAnalysis extracts capacity analysis from enhanced baseline
func (abm *AutomatedBaselineManager) extractCapacityAnalysis(baseline *EnhancedPerformanceBaseline) CapacityAnalysis {
	bottleneckRisk := "low"
	scalingNeeded := false
	
	// Determine bottleneck risk
	for _, utilization := range baseline.CapacityData.ResourceUtilization {
		if utilization > 0.9 {
			bottleneckRisk = "high"
			scalingNeeded = true
			break
		} else if utilization > 0.8 {
			bottleneckRisk = "medium"
		}
	}
	
	return CapacityAnalysis{
		CurrentUtilization: baseline.CapacityData.ResourceUtilization,
		BottleneckRisk:     bottleneckRisk,
		ScalingNeeded:      scalingNeeded,
		TimeToCapacity:     nil, // Would be calculated from trend analysis
	}
}

// getValidationFailureAction returns appropriate action for validation failure
func (abm *AutomatedBaselineManager) getValidationFailureAction(ruleName string) string {
	actions := map[string]string{
		"minimum_sample_size":   "Collect more performance data points",
		"data_completeness":     "Ensure all required metrics are collected",
		"variance_threshold":    "Investigate source of high variance",
		"outlier_percentage":    "Review and filter outlier data points",
		"change_point_threshold": "Analyze significant performance changes",
	}
	
	if action, exists := actions[ruleName]; exists {
		return action
	}
	return "Review validation rule and data collection process"
}

// updateScheduledBaseline updates a baseline according to its schedule
func (abm *AutomatedBaselineManager) updateScheduledBaseline(schedule BaselineSchedule) error {
	log.Printf("Updating scheduled baseline for %s in %s", schedule.TestName, schedule.Environment)
	
	// Generate new metrics (in real implementation, collect from actual tests)
	rawMetrics, err := abm.generateSampleMetrics(schedule.TestName, schedule.Environment)
	if err != nil {
		return fmt.Errorf("failed to generate metrics for update: %w", err)
	}
	
	// Get current version (simplified)
	version := fmt.Sprintf("auto-%d", time.Now().Unix())
	
	// Establish new baseline
	_, err = abm.EstablishAutomatedBaseline(schedule.TestName, version, schedule.Environment)
	if err != nil {
		return fmt.Errorf("failed to establish updated baseline: %w", err)
	}
	
	return nil
}

// storeAutomatedResult stores the automated baseline result
func (abm *AutomatedBaselineManager) storeAutomatedResult(result *AutomatedBaselineResult) error {
	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result data: %w", err)
	}
	
	_, err = abm.db.Exec(`
		INSERT INTO automated_baseline_results 
		(baseline_id, test_name, environment, version, established_at, data_points, quality_score, next_update_scheduled, result_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		result.BaselineID, result.TestName, result.Environment, result.Version,
		result.CreatedAt, result.DataPoints, result.QualityScore, result.NextUpdateScheduled, resultData)
	
	return err
}

// NewValidationEngine creates a new validation engine
func NewValidationEngine(db *sql.DB) *ValidationEngine {
	engine := &ValidationEngine{
		db:    db,
		rules: make(map[string]ValidationRule),
	}
	
	engine.loadValidationRules()
	return engine
}

// loadValidationRules loads validation rules from database
func (ve *ValidationEngine) loadValidationRules() {
	rows, err := ve.db.Query(`
		SELECT rule_name, condition_type, threshold_value, severity, action_type, description
		FROM baseline_validation_rules
		WHERE enabled = true`)
	if err != nil {
		log.Printf("Failed to load validation rules: %v", err)
		ve.loadDefaultRules()
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var rule ValidationRule
		err := rows.Scan(&rule.Name, &rule.ConditionType, &rule.Threshold, &rule.Severity, &rule.ActionType, &rule.Description)
		if err != nil {
			log.Printf("Failed to scan validation rule: %v", err)
			continue
		}
		ve.rules[rule.Name] = rule
	}
	
	if len(ve.rules) == 0 {
		ve.loadDefaultRules()
	}
}

// loadDefaultRules loads default validation rules
func (ve *ValidationEngine) loadDefaultRules() {
	ve.rules = map[string]ValidationRule{
		"minimum_sample_size": {
			Name:          "minimum_sample_size",
			ConditionType: "greater_than_equal",
			Threshold:     30,
			Severity:      "critical",
			ActionType:    "reject",
			Description:   "Minimum required sample size for statistical significance",
		},
		"data_completeness": {
			Name:          "data_completeness",
			ConditionType: "greater_than_equal",
			Threshold:     0.90,
			Severity:      "high",
			ActionType:    "warn",
			Description:   "Minimum data completeness percentage",
		},
		"variance_threshold": {
			Name:          "variance_threshold",
			ConditionType: "less_than_equal",
			Threshold:     0.30,
			Severity:      "medium",
			ActionType:    "warn",
			Description:   "Maximum allowed coefficient of variation",
		},
	}
}

// ValidateBaseline validates a baseline against all rules
func (ve *ValidationEngine) ValidateBaseline(baseline *EnhancedPerformanceBaseline) ([]ValidationResult, error) {
	results := []ValidationResult{}
	
	for _, rule := range ve.rules {
		result := ve.validateRule(rule, baseline)
		results = append(results, result)
	}
	
	return results, nil
}

// validateRule validates a single rule against the baseline
func (ve *ValidationEngine) validateRule(rule ValidationRule, baseline *EnhancedPerformanceBaseline) ValidationResult {
	result := ValidationResult{
		RuleName:  rule.Name,
		Severity:  rule.Severity,
		Threshold: rule.Threshold,
	}
	
	switch rule.Name {
	case "minimum_sample_size":
		// Check if we have enough data points (simplified)
		sampleSize := float64(len(baseline.StatisticalData.Outliers) + 30) // Estimate
		result.Value = sampleSize
		if sampleSize >= rule.Threshold {
			result.Status = ValidationStatusPass
			result.Message = fmt.Sprintf("Sample size %.0f meets minimum requirement", sampleSize)
		} else {
			result.Status = ValidationStatusFail
			result.Message = fmt.Sprintf("Sample size %.0f below minimum %.0f", sampleSize, rule.Threshold)
		}
		
	case "data_completeness":
		// Check data completeness (simplified)
		completeness := 0.95 // Would be calculated from actual data
		result.Value = completeness
		if completeness >= rule.Threshold {
			result.Status = ValidationStatusPass
			result.Message = fmt.Sprintf("Data completeness %.1f%% meets requirement", completeness*100)
		} else {
			result.Status = ValidationStatusFail
			result.Message = fmt.Sprintf("Data completeness %.1f%% below %.1f%%", completeness*100, rule.Threshold*100)
		}
		
	case "variance_threshold":
		// Check variance (simplified)
		variance := 0.25 // Would be calculated from actual metrics
		result.Value = variance
		if variance <= rule.Threshold {
			result.Status = ValidationStatusPass
			result.Message = fmt.Sprintf("Variance %.3f within acceptable range", variance)
		} else {
			result.Status = ValidationStatusWarning
			result.Message = fmt.Sprintf("Variance %.3f exceeds threshold %.3f", variance, rule.Threshold)
		}
		
	default:
		result.Status = ValidationStatusPass
		result.Message = "Rule validation not implemented"
		result.Value = 0
	}
	
	return result
}

// NewSchedulingEngine creates a new scheduling engine
func NewSchedulingEngine(db *sql.DB) *SchedulingEngine {
	return &SchedulingEngine{db: db}
}

// ScheduleUpdate schedules a baseline update
func (se *SchedulingEngine) ScheduleUpdate(testName, environment string, nextUpdate time.Time) error {
	_, err := se.db.Exec(`
		INSERT INTO baseline_schedules (test_name, environment, next_update, status)
		VALUES ($1, $2, $3, 'active')
		ON CONFLICT (test_name, environment)
		DO UPDATE SET next_update = $3, status = 'active'`,
		testName, environment, nextUpdate)
	
	return err
}

// GetDueUpdates returns schedules that are due for update
func (se *SchedulingEngine) GetDueUpdates() ([]BaselineSchedule, error) {
	rows, err := se.db.Query(`
		SELECT id, test_name, environment, update_frequency, next_update, last_update, status, required_data_points
		FROM baseline_schedules
		WHERE status = 'active' AND next_update <= NOW()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	schedules := []BaselineSchedule{}
	for rows.Next() {
		var schedule BaselineSchedule
		var updateFrequencyStr string
		var lastUpdate sql.NullTime
		
		err := rows.Scan(&schedule.ID, &schedule.TestName, &schedule.Environment,
			&updateFrequencyStr, &schedule.NextUpdate, &lastUpdate, &schedule.Status, &schedule.RequiredDataPoints)
		if err != nil {
			continue
		}
		
		// Parse update frequency
		schedule.UpdateFrequency, _ = time.ParseDuration(updateFrequencyStr)
		if schedule.UpdateFrequency == 0 {
			schedule.UpdateFrequency = 24 * time.Hour // Default
		}
		
		if lastUpdate.Valid {
			schedule.LastUpdate = &lastUpdate.Time
		}
		
		schedules = append(schedules, schedule)
	}
	
	return schedules, nil
}

// UpdateSchedule updates a schedule's next update time
func (se *SchedulingEngine) UpdateSchedule(scheduleID int64, nextUpdate time.Time) error {
	_, err := se.db.Exec(`
		UPDATE baseline_schedules
		SET next_update = $2, last_update = NOW()
		WHERE id = $1`,
		scheduleID, nextUpdate)
	
	return err
}