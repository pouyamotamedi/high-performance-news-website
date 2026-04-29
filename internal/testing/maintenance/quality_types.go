package maintenance

import (
	"time"
)

// QualityAnalysisReport represents a comprehensive quality analysis report
type QualityAnalysisReport struct {
	Timestamp        time.Time                      `json:"timestamp"`
	TestPath         string                         `json:"test_path"`
	QualityScores    map[string]*QualityMetrics     `json:"quality_scores"`
	Issues           []QualityIssue                 `json:"issues"`
	Opportunities    []RefactoringOpportunity       `json:"opportunities"`
	OverallMetrics   *OverallQualityMetrics         `json:"overall_metrics"`
	Recommendations  []QualityRecommendation        `json:"recommendations"`
}

// OverallQualityMetrics represents overall quality metrics for the test suite
type OverallQualityMetrics struct {
	AverageMaintainability float64 `json:"average_maintainability"`
	AverageReadability     float64 `json:"average_readability"`
	AverageReliability     float64 `json:"average_reliability"`
	AveragePerformance     float64 `json:"average_performance"`
	AverageCoverage        float64 `json:"average_coverage"`
	OverallQuality         float64 `json:"overall_quality"`
	TotalTests             int     `json:"total_tests"`
	HighQualityTests       int     `json:"high_quality_tests"`
	LowQualityTests        int     `json:"low_quality_tests"`
	QualityDistribution    map[string]int `json:"quality_distribution"`
}

// QualityIssue represents a quality issue found in tests
type QualityIssue struct {
	ID          string        `json:"id"`
	Type        QualityIssueType `json:"type"`
	Severity    IssueSeverity `json:"severity"`
	TestID      string        `json:"test_id"`
	FilePath    string        `json:"file_path"`
	LineNumber  int           `json:"line_number"`
	Description string        `json:"description"`
	Impact      string        `json:"impact"`
	Suggestion  string        `json:"suggestion"`
	DetectedAt  time.Time     `json:"detected_at"`
}

// QualityIssueType defines types of quality issues
type QualityIssueType string

const (
	QualityIssueMaintainability QualityIssueType = "maintainability"
	QualityIssueReadability     QualityIssueType = "readability"
	QualityIssueReliability     QualityIssueType = "reliability"
	QualityIssuePerformance     QualityIssueType = "performance"
	QualityIssueComplexity      QualityIssueType = "complexity"
	QualityIssueNaming          QualityIssueType = "naming"
	QualityIssueDocumentation   QualityIssueType = "documentation"
	QualityIssueDuplication     QualityIssueType = "duplication"
	QualityIssueAssertion       QualityIssueType = "assertion"
	QualityIssueStructure       QualityIssueType = "structure"
)

// QualityRecommendation represents a quality improvement recommendation
type QualityRecommendation struct {
	ID              string                 `json:"id"`
	Type            RecommendationType     `json:"type"`
	Priority        Priority               `json:"priority"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Benefits        []string               `json:"benefits"`
	EstimatedEffort string                 `json:"estimated_effort"`
	AffectedTests   []string               `json:"affected_tests"`
	ActionItems     []ActionItem           `json:"action_items"`
	CreatedAt       time.Time              `json:"created_at"`
}

// RecommendationType defines types of quality recommendations
type RecommendationType string

const (
	RecommendationRefactor     RecommendationType = "refactor"
	RecommendationOptimize     RecommendationType = "optimize"
	RecommendationStandardize  RecommendationType = "standardize"
	RecommendationConsolidate  RecommendationType = "consolidate"
	RecommendationDocument     RecommendationType = "document"
	RecommendationModernize    RecommendationType = "modernize"
)

// ActionItem represents a specific action to take
type ActionItem struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	AssignedTo  string    `json:"assigned_to,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// QualityTrend represents quality trends over time
type QualityTrend struct {
	TestID      string                 `json:"test_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Metrics     *QualityMetrics        `json:"metrics"`
	Changes     []QualityChange        `json:"changes"`
	Predictions []QualityPrediction    `json:"predictions"`
}

// QualityChange represents a change in quality metrics
type QualityChange struct {
	Metric      string    `json:"metric"`
	OldValue    float64   `json:"old_value"`
	NewValue    float64   `json:"new_value"`
	Change      float64   `json:"change"`
	ChangeType  string    `json:"change_type"` // "improvement", "degradation", "stable"
	Timestamp   time.Time `json:"timestamp"`
}

// QualityPrediction represents predicted quality trends
type QualityPrediction struct {
	Metric         string    `json:"metric"`
	PredictedValue float64   `json:"predicted_value"`
	Confidence     float64   `json:"confidence"`
	TimeHorizon    time.Duration `json:"time_horizon"`
	Factors        []string  `json:"factors"`
}

// QualityBenchmark represents quality benchmarks and targets
type QualityBenchmark struct {
	Metric      string  `json:"metric"`
	Target      float64 `json:"target"`
	Minimum     float64 `json:"minimum"`
	Excellent   float64 `json:"excellent"`
	Industry    float64 `json:"industry_average"`
	Description string  `json:"description"`
}

// QualityReport represents a comprehensive quality report
type QualityReport struct {
	GeneratedAt     time.Time                `json:"generated_at"`
	ReportPeriod    ReportPeriod            `json:"report_period"`
	Summary         QualitySummary          `json:"summary"`
	Trends          []QualityTrend          `json:"trends"`
	Benchmarks      []QualityBenchmark      `json:"benchmarks"`
	TopIssues       []QualityIssue          `json:"top_issues"`
	Improvements    []QualityImprovement    `json:"improvements"`
	Recommendations []QualityRecommendation `json:"recommendations"`
}

// ReportPeriod defines the time period for a report
type ReportPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Duration  string    `json:"duration"`
}

// QualitySummary provides a summary of quality metrics
type QualitySummary struct {
	TotalTests           int                    `json:"total_tests"`
	QualityScore         float64                `json:"quality_score"`
	QualityGrade         string                 `json:"quality_grade"`
	IssuesFound          int                    `json:"issues_found"`
	IssuesResolved       int                    `json:"issues_resolved"`
	TrendDirection       string                 `json:"trend_direction"`
	MetricBreakdown      map[string]float64     `json:"metric_breakdown"`
	CategoryScores       map[string]float64     `json:"category_scores"`
}

// QualityImprovement represents an improvement made to test quality
type QualityImprovement struct {
	ID              string    `json:"id"`
	TestID          string    `json:"test_id"`
	Type            string    `json:"type"`
	Description     string    `json:"description"`
	MetricsBefore   *QualityMetrics `json:"metrics_before"`
	MetricsAfter    *QualityMetrics `json:"metrics_after"`
	ImpactScore     float64   `json:"impact_score"`
	ImplementedAt   time.Time `json:"implemented_at"`
	ImplementedBy   string    `json:"implemented_by"`
}

// QualityAlert represents a quality alert
type QualityAlert struct {
	ID          string           `json:"id"`
	Type        QualityAlertType `json:"type"`
	Severity    AlertSeverity    `json:"severity"`
	TestID      string           `json:"test_id"`
	Metric      string           `json:"metric"`
	Threshold   float64          `json:"threshold"`
	ActualValue float64          `json:"actual_value"`
	Message     string           `json:"message"`
	TriggeredAt time.Time        `json:"triggered_at"`
	Resolved    bool             `json:"resolved"`
	ResolvedAt  *time.Time       `json:"resolved_at,omitempty"`
}

// QualityAlertType defines types of quality alerts
type QualityAlertType string

const (
	AlertQualityDegradation QualityAlertType = "quality_degradation"
	AlertThresholdBreach    QualityAlertType = "threshold_breach"
	AlertTrendAlert         QualityAlertType = "trend_alert"
	AlertAnomalyDetection   QualityAlertType = "anomaly_detection"
)

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// QualityDashboard represents dashboard data for quality metrics
type QualityDashboard struct {
	LastUpdated     time.Time              `json:"last_updated"`
	OverallScore    float64                `json:"overall_score"`
	ScoreHistory    []ScorePoint           `json:"score_history"`
	MetricCards     []MetricCard           `json:"metric_cards"`
	RecentIssues    []QualityIssue         `json:"recent_issues"`
	ActiveAlerts    []QualityAlert         `json:"active_alerts"`
	TopTests        []TestQualityInfo      `json:"top_tests"`
	BottomTests     []TestQualityInfo      `json:"bottom_tests"`
	Recommendations []QualityRecommendation `json:"recommendations"`
}

// ScorePoint represents a point in quality score history
type ScorePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
	Metric    string    `json:"metric"`
}

// MetricCard represents a metric card for the dashboard
type MetricCard struct {
	Title       string  `json:"title"`
	Value       float64 `json:"value"`
	Target      float64 `json:"target"`
	Trend       string  `json:"trend"`
	TrendValue  float64 `json:"trend_value"`
	Status      string  `json:"status"`
	Description string  `json:"description"`
}

// TestQualityInfo represents quality information for a specific test
type TestQualityInfo struct {
	TestID      string          `json:"test_id"`
	TestName    string          `json:"test_name"`
	FilePath    string          `json:"file_path"`
	Quality     *QualityMetrics `json:"quality"`
	Rank        int             `json:"rank"`
	Issues      []QualityIssue  `json:"issues"`
}

// QualityConfiguration represents configuration for quality analysis
type QualityConfiguration struct {
	Thresholds      map[string]float64     `json:"thresholds"`
	Weights         map[string]float64     `json:"weights"`
	EnabledChecks   []string               `json:"enabled_checks"`
	CustomRules     []CustomQualityRule    `json:"custom_rules"`
	AlertSettings   AlertSettings          `json:"alert_settings"`
	ReportSettings  ReportSettings         `json:"report_settings"`
}

// CustomQualityRule represents a custom quality rule
type CustomQualityRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Pattern     string                 `json:"pattern"`
	Severity    IssueSeverity          `json:"severity"`
	Category    QualityIssueType       `json:"category"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
}

// AlertSettings represents settings for quality alerts
type AlertSettings struct {
	Enabled           bool                   `json:"enabled"`
	Channels          []string               `json:"channels"`
	Thresholds        map[string]float64     `json:"thresholds"`
	Cooldown          time.Duration          `json:"cooldown"`
	EscalationRules   []EscalationRule       `json:"escalation_rules"`
}

// EscalationRule represents an alert escalation rule
type EscalationRule struct {
	Condition   string        `json:"condition"`
	Delay       time.Duration `json:"delay"`
	Recipients  []string      `json:"recipients"`
	Actions     []string      `json:"actions"`
}

// ReportSettings represents settings for quality reports
type ReportSettings struct {
	Frequency       string   `json:"frequency"`
	Recipients      []string `json:"recipients"`
	IncludeTrends   bool     `json:"include_trends"`
	IncludeDetails  bool     `json:"include_details"`
	Format          string   `json:"format"`
	Template        string   `json:"template"`
}