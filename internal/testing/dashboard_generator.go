package testing

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"time"
)

// DashboardGenerator generates interactive dashboards for test results
type DashboardGenerator struct {
	config DashboardConfig
}

// DashboardConfig holds configuration for dashboard generation
type DashboardConfig struct {
	OutputDir    string            `json:"output_dir"`
	Title        string            `json:"title"`
	RefreshRate  time.Duration     `json:"refresh_rate"`
	Widgets      []WidgetConfig    `json:"widgets"`
	Themes       map[string]Theme  `json:"themes"`
	DefaultTheme string            `json:"default_theme"`
}

// WidgetConfig defines configuration for dashboard widgets
type WidgetConfig struct {
	ID       string     `json:"id"`
	Type     WidgetType `json:"type"`
	Title    string     `json:"title"`
	Position Position   `json:"position"`
	Size     Size       `json:"size"`
	Config   map[string]interface{} `json:"config"`
}

// WidgetType defines the type of dashboard widget
type WidgetType string

const (
	WidgetTypeMetric     WidgetType = "metric"
	WidgetTypeChart      WidgetType = "chart"
	WidgetTypeTable      WidgetType = "table"
	WidgetTypeProgress   WidgetType = "progress"
	WidgetTypeAlert      WidgetType = "alert"
	WidgetTypeTrend      WidgetType = "trend"
)

// Position defines widget position on dashboard
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Size defines widget size
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Theme defines dashboard theme
type Theme struct {
	Name            string            `json:"name"`
	Colors          map[string]string `json:"colors"`
	BackgroundColor string            `json:"background_color"`
	TextColor       string            `json:"text_color"`
	FontFamily      string            `json:"font_family"`
}

// Dashboard represents a generated dashboard
type Dashboard struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	GeneratedAt time.Time       `json:"generated_at"`
	Widgets     []Widget        `json:"widgets"`
	Data        DashboardData   `json:"data"`
	Config      DashboardConfig `json:"config"`
}

// Widget represents a dashboard widget
type Widget struct {
	ID       string                 `json:"id"`
	Type     WidgetType             `json:"type"`
	Title    string                 `json:"title"`
	Position Position               `json:"position"`
	Size     Size                   `json:"size"`
	Data     map[string]interface{} `json:"data"`
	Status   WidgetStatus           `json:"status"`
}

// WidgetStatus defines widget status
type WidgetStatus string

const (
	WidgetStatusGood    WidgetStatus = "good"
	WidgetStatusWarning WidgetStatus = "warning"
	WidgetStatusError   WidgetStatus = "error"
)

// DashboardData contains all data for dashboard widgets
type DashboardData struct {
	Summary        TestSummary     `json:"summary"`
	QualityMetrics QualityMetrics  `json:"quality_metrics"`
	TrendData      TrendData       `json:"trend_data"`
	KPIs           map[string]KPIStatus `json:"kpis"`
	Alerts         []Alert         `json:"alerts"`
	RecentResults  []PipelineResult `json:"recent_results"`
}

// TrendData contains trend information for charts
type TrendData struct {
	Coverage    []TrendDataPoint `json:"coverage"`
	Security    []TrendDataPoint `json:"security"`
	Performance []TrendDataPoint `json:"performance"`
	Quality     []TrendDataPoint `json:"quality"`
}

// Alert represents a dashboard alert
type Alert struct {
	ID          string    `json:"id"`
	Type        AlertType `json:"type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Acknowledged bool     `json:"acknowledged"`
}

// AlertType defines alert types
type AlertType string

const (
	AlertTypeCoverage    AlertType = "coverage"
	AlertTypeSecurity    AlertType = "security"
	AlertTypePerformance AlertType = "performance"
	AlertTypeFailure     AlertType = "failure"
)

// NewDashboardGenerator creates a new dashboard generator
func NewDashboardGenerator(config DashboardConfig) *DashboardGenerator {
	return &DashboardGenerator{
		config: config,
	}
}

// GenerateDashboard generates a comprehensive dashboard
func (d *DashboardGenerator) GenerateDashboard(report *TestReport) (*Dashboard, error) {
	log.Printf("Generating dashboard for report: %s", report.ID)
	
	dashboard := &Dashboard{
		ID:          fmt.Sprintf("dashboard-%s", report.ID),
		Title:       d.config.Title,
		GeneratedAt: time.Now(),
		Config:      d.config,
	}
	
	// Prepare dashboard data
	dashboard.Data = d.prepareDashboardData(report)
	
	// Generate widgets
	widgets, err := d.generateWidgets(dashboard.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to generate widgets: %w", err)
	}
	dashboard.Widgets = widgets
	
	// Generate dashboard files
	if err := d.generateDashboardFiles(dashboard); err != nil {
		return nil, fmt.Errorf("failed to generate dashboard files: %w", err)
	}
	
	log.Printf("Dashboard generated successfully: %s", dashboard.ID)
	return dashboard, nil
}

// prepareDashboardData prepares data for dashboard widgets
func (d *DashboardGenerator) prepareDashboardData(report *TestReport) DashboardData {
	data := DashboardData{
		Summary:        report.Summary,
		QualityMetrics: report.QualityMetrics,
		KPIs:           report.KPIStatus,
		RecentResults:  report.PipelineResults,
		Alerts:         d.generateAlerts(report),
	}
	
	// Prepare trend data (simplified - would normally come from historical data)
	data.TrendData = TrendData{
		Coverage: []TrendDataPoint{
			{Timestamp: time.Now().Add(-7 * 24 * time.Hour), Value: report.QualityMetrics.CodeCoverage.Overall - 2},
			{Timestamp: time.Now().Add(-6 * 24 * time.Hour), Value: report.QualityMetrics.CodeCoverage.Overall - 1},
			{Timestamp: time.Now().Add(-5 * 24 * time.Hour), Value: report.QualityMetrics.CodeCoverage.Overall},
			{Timestamp: time.Now().Add(-4 * 24 * time.Hour), Value: report.QualityMetrics.CodeCoverage.Overall + 1},
			{Timestamp: time.Now().Add(-3 * 24 * time.Hour), Value: report.QualityMetrics.CodeCoverage.Overall + 0.5},
			{Timestamp: time.Now().Add(-2 * 24 * time.Hour), Value: report.QualityMetrics.CodeCoverage.Overall + 1.5},
			{Timestamp: time.Now(), Value: report.QualityMetrics.CodeCoverage.Overall + 2},
		},
		Security: []TrendDataPoint{
			{Timestamp: time.Now().Add(-7 * 24 * time.Hour), Value: report.QualityMetrics.SecurityMetrics.SecurityScoreAvg - 5},
			{Timestamp: time.Now().Add(-6 * 24 * time.Hour), Value: report.QualityMetrics.SecurityMetrics.SecurityScoreAvg - 3},
			{Timestamp: time.Now().Add(-5 * 24 * time.Hour), Value: report.QualityMetrics.SecurityMetrics.SecurityScoreAvg - 1},
			{Timestamp: time.Now().Add(-4 * 24 * time.Hour), Value: report.QualityMetrics.SecurityMetrics.SecurityScoreAvg},
			{Timestamp: time.Now().Add(-3 * 24 * time.Hour), Value: report.QualityMetrics.SecurityMetrics.SecurityScoreAvg + 1},
			{Timestamp: time.Now().Add(-2 * 24 * time.Hour), Value: report.QualityMetrics.SecurityMetrics.SecurityScoreAvg + 2},
			{Timestamp: time.Now(), Value: report.QualityMetrics.SecurityMetrics.SecurityScoreAvg + 3},
		},
		Performance: []TrendDataPoint{
			{Timestamp: time.Now().Add(-7 * 24 * time.Hour), Value: report.QualityMetrics.PerformanceMetrics.RegressionPercentage + 2},
			{Timestamp: time.Now().Add(-6 * 24 * time.Hour), Value: report.QualityMetrics.PerformanceMetrics.RegressionPercentage + 1.5},
			{Timestamp: time.Now().Add(-5 * 24 * time.Hour), Value: report.QualityMetrics.PerformanceMetrics.RegressionPercentage + 1},
			{Timestamp: time.Now().Add(-4 * 24 * time.Hour), Value: report.QualityMetrics.PerformanceMetrics.RegressionPercentage + 0.5},
			{Timestamp: time.Now().Add(-3 * 24 * time.Hour), Value: report.QualityMetrics.PerformanceMetrics.RegressionPercentage},
			{Timestamp: time.Now().Add(-2 * 24 * time.Hour), Value: report.QualityMetrics.PerformanceMetrics.RegressionPercentage - 0.5},
			{Timestamp: time.Now(), Value: report.QualityMetrics.PerformanceMetrics.RegressionPercentage - 1},
		},
	}
	
	return data
}

// generateAlerts generates alerts based on report data
func (d *DashboardGenerator) generateAlerts(report *TestReport) []Alert {
	var alerts []Alert
	
	// Coverage alerts
	if report.QualityMetrics.CodeCoverage.Overall < 90 {
		alerts = append(alerts, Alert{
			ID:        "coverage-low",
			Type:      AlertTypeCoverage,
			Severity:  "warning",
			Title:     "Low Code Coverage",
			Message:   fmt.Sprintf("Code coverage is %.1f%%, below recommended 90%%", report.QualityMetrics.CodeCoverage.Overall),
			Timestamp: time.Now(),
		})
	}
	
	// Security alerts
	if report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities > 0 {
		alerts = append(alerts, Alert{
			ID:        "security-critical",
			Type:      AlertTypeSecurity,
			Severity:  "critical",
			Title:     "Critical Security Vulnerabilities",
			Message:   fmt.Sprintf("Found %d critical security vulnerabilities", report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities),
			Timestamp: time.Now(),
		})
	}
	
	// Performance alerts
	if report.QualityMetrics.PerformanceMetrics.RegressionPercentage > 15 {
		alerts = append(alerts, Alert{
			ID:        "performance-regression",
			Type:      AlertTypePerformance,
			Severity:  "warning",
			Title:     "Performance Regression",
			Message:   fmt.Sprintf("Performance regression of %.1f%% detected", report.QualityMetrics.PerformanceMetrics.RegressionPercentage),
			Timestamp: time.Now(),
		})
	}
	
	// Pipeline failure alerts
	failedPipelines := report.Summary.TotalPipelines - report.Summary.SuccessfulPipelines
	if failedPipelines > 0 {
		alerts = append(alerts, Alert{
			ID:        "pipeline-failures",
			Type:      AlertTypeFailure,
			Severity:  "high",
			Title:     "Pipeline Failures",
			Message:   fmt.Sprintf("%d pipeline(s) failed in the last period", failedPipelines),
			Timestamp: time.Now(),
		})
	}
	
	return alerts
}

// generateWidgets generates dashboard widgets
func (d *DashboardGenerator) generateWidgets(data DashboardData) ([]Widget, error) {
	var widgets []Widget
	
	// Success Rate Widget
	widgets = append(widgets, Widget{
		ID:       "success-rate",
		Type:     WidgetTypeMetric,
		Title:    "Pipeline Success Rate",
		Position: Position{X: 0, Y: 0},
		Size:     Size{Width: 3, Height: 2},
		Data: map[string]interface{}{
			"value":  data.Summary.SuccessRate,
			"unit":   "%",
			"target": 95.0,
		},
		Status: d.getMetricStatus(data.Summary.SuccessRate, 95.0, true),
	})
	
	// Code Coverage Widget
	widgets = append(widgets, Widget{
		ID:       "code-coverage",
		Type:     WidgetTypeProgress,
		Title:    "Code Coverage",
		Position: Position{X: 3, Y: 0},
		Size:     Size{Width: 3, Height: 2},
		Data: map[string]interface{}{
			"value":  data.QualityMetrics.CodeCoverage.Overall,
			"target": 95.0,
			"unit":   "%",
		},
		Status: d.getMetricStatus(data.QualityMetrics.CodeCoverage.Overall, 95.0, true),
	})
	
	// Security Score Widget
	widgets = append(widgets, Widget{
		ID:       "security-score",
		Type:     WidgetTypeMetric,
		Title:    "Security Score",
		Position: Position{X: 6, Y: 0},
		Size:     Size{Width: 3, Height: 2},
		Data: map[string]interface{}{
			"value":  data.QualityMetrics.SecurityMetrics.SecurityScoreAvg,
			"unit":   "/100",
			"target": 90.0,
		},
		Status: d.getMetricStatus(data.QualityMetrics.SecurityMetrics.SecurityScoreAvg, 90.0, true),
	})
	
	// Performance Widget
	widgets = append(widgets, Widget{
		ID:       "performance-regression",
		Type:     WidgetTypeMetric,
		Title:    "Performance Regression",
		Position: Position{X: 9, Y: 0},
		Size:     Size{Width: 3, Height: 2},
		Data: map[string]interface{}{
			"value":  data.QualityMetrics.PerformanceMetrics.RegressionPercentage,
			"unit":   "%",
			"target": 5.0,
		},
		Status: d.getMetricStatus(data.QualityMetrics.PerformanceMetrics.RegressionPercentage, 5.0, false),
	})
	
	// Coverage Trend Chart
	widgets = append(widgets, Widget{
		ID:       "coverage-trend",
		Type:     WidgetTypeChart,
		Title:    "Coverage Trend (7 days)",
		Position: Position{X: 0, Y: 2},
		Size:     Size{Width: 6, Height: 4},
		Data: map[string]interface{}{
			"type":   "line",
			"data":   data.TrendData.Coverage,
			"yAxis":  "Coverage %",
			"target": 95.0,
		},
		Status: WidgetStatusGood,
	})
	
	// Security Trend Chart
	widgets = append(widgets, Widget{
		ID:       "security-trend",
		Type:     WidgetTypeChart,
		Title:    "Security Score Trend (7 days)",
		Position: Position{X: 6, Y: 2},
		Size:     Size{Width: 6, Height: 4},
		Data: map[string]interface{}{
			"type":   "line",
			"data":   data.TrendData.Security,
			"yAxis":  "Security Score",
			"target": 90.0,
		},
		Status: WidgetStatusGood,
	})
	
	// Recent Pipeline Results Table
	widgets = append(widgets, Widget{
		ID:       "recent-pipelines",
		Type:     WidgetTypeTable,
		Title:    "Recent Pipeline Results",
		Position: Position{X: 0, Y: 6},
		Size:     Size{Width: 8, Height: 4},
		Data: map[string]interface{}{
			"columns": []string{"Pipeline ID", "Status", "Duration", "Coverage", "Issues"},
			"rows":    d.preparePipelineTableData(data.RecentResults),
		},
		Status: WidgetStatusGood,
	})
	
	// Alerts Widget
	widgets = append(widgets, Widget{
		ID:       "alerts",
		Type:     WidgetTypeAlert,
		Title:    "Active Alerts",
		Position: Position{X: 8, Y: 6},
		Size:     Size{Width: 4, Height: 4},
		Data: map[string]interface{}{
			"alerts": data.Alerts,
		},
		Status: d.getAlertsStatus(data.Alerts),
	})
	
	return widgets, nil
}

// getMetricStatus determines widget status based on metric value
func (d *DashboardGenerator) getMetricStatus(value, target float64, higherIsBetter bool) WidgetStatus {
	var ratio float64
	if higherIsBetter {
		ratio = value / target
	} else {
		ratio = target / value
	}
	
	if ratio >= 1.0 {
		return WidgetStatusGood
	} else if ratio >= 0.8 {
		return WidgetStatusWarning
	} else {
		return WidgetStatusError
	}
}

// getAlertsStatus determines status based on alerts
func (d *DashboardGenerator) getAlertsStatus(alerts []Alert) WidgetStatus {
	if len(alerts) == 0 {
		return WidgetStatusGood
	}
	
	for _, alert := range alerts {
		if alert.Severity == "critical" {
			return WidgetStatusError
		}
	}
	
	return WidgetStatusWarning
}

// preparePipelineTableData prepares data for pipeline results table
func (d *DashboardGenerator) preparePipelineTableData(results []PipelineResult) [][]string {
	var rows [][]string
	
	// Limit to last 10 results
	limit := 10
	if len(results) < limit {
		limit = len(results)
	}
	
	for i := len(results) - limit; i < len(results); i++ {
		result := results[i]
		
		// Extract coverage and issues from stages
		var coverage string = "N/A"
		var issues string = "0"
		
		for _, stage := range result.Stages {
			if stage.Metrics != nil {
				if cov, ok := stage.Metrics["coverage"].(float64); ok {
					coverage = fmt.Sprintf("%.1f%%", cov)
				}
				if criticalIssues, ok := stage.Metrics["critical_issues"].(int); ok {
					issues = fmt.Sprintf("%d", criticalIssues)
				}
			}
		}
		
		row := []string{
			result.ID,
			string(result.Status),
			result.Duration.String(),
			coverage,
			issues,
		}
		rows = append(rows, row)
	}
	
	return rows
}

// generateDashboardFiles generates HTML and supporting files for the dashboard
func (d *DashboardGenerator) generateDashboardFiles(dashboard *Dashboard) error {
	// Ensure output directory exists
	if err := os.MkdirAll(d.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Generate HTML file
	if err := d.generateHTMLDashboard(dashboard); err != nil {
		return fmt.Errorf("failed to generate HTML dashboard: %w", err)
	}
	
	// Generate JSON data file
	if err := d.generateJSONData(dashboard); err != nil {
		return fmt.Errorf("failed to generate JSON data: %w", err)
	}
	
	// Generate CSS file
	if err := d.generateCSS(); err != nil {
		return fmt.Errorf("failed to generate CSS: %w", err)
	}
	
	// Generate JavaScript file
	if err := d.generateJavaScript(); err != nil {
		return fmt.Errorf("failed to generate JavaScript: %w", err)
	}
	
	return nil
}

// generateHTMLDashboard generates the main HTML dashboard file
func (d *DashboardGenerator) generateHTMLDashboard(dashboard *Dashboard) error {
	htmlTemplate := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Test Dashboard</title>
    <link rel="stylesheet" href="dashboard.css">
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <div class="dashboard-container">
        <header class="dashboard-header">
            <h1>{{.Title}}</h1>
            <div class="dashboard-info">
                <span>Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</span>
                <span>Auto-refresh: {{.Config.RefreshRate}}</span>
            </div>
        </header>
        
        <div class="dashboard-grid" id="dashboard-grid">
            {{range .Widgets}}
            <div class="widget widget-{{.Type}} widget-{{.Status}}" 
                 style="grid-column: span {{.Size.Width}}; grid-row: span {{.Size.Height}};"
                 data-widget-id="{{.ID}}">
                <div class="widget-header">
                    <h3>{{.Title}}</h3>
                    <div class="widget-status status-{{.Status}}"></div>
                </div>
                <div class="widget-content" id="widget-{{.ID}}">
                    {{if eq .Type "metric"}}
                        <div class="metric-widget">
                            <div class="metric-value">{{index .Data "value" | printf "%.1f"}}{{index .Data "unit"}}</div>
                            <div class="metric-target">Target: {{index .Data "target" | printf "%.1f"}}{{index .Data "unit"}}</div>
                        </div>
                    {{else if eq .Type "progress"}}
                        <div class="progress-widget">
                            <div class="progress-bar">
                                <div class="progress-fill" style="width: {{index .Data "value" | printf "%.1f"}}%"></div>
                            </div>
                            <div class="progress-text">{{index .Data "value" | printf "%.1f"}}% / {{index .Data "target" | printf "%.1f"}}%</div>
                        </div>
                    {{else if eq .Type "chart"}}
                        <canvas id="chart-{{.ID}}"></canvas>
                    {{else if eq .Type "table"}}
                        <div class="table-widget">
                            <table>
                                <thead>
                                    <tr>
                                        {{range index .Data "columns"}}
                                        <th>{{.}}</th>
                                        {{end}}
                                    </tr>
                                </thead>
                                <tbody>
                                    {{range index .Data "rows"}}
                                    <tr>
                                        {{range .}}
                                        <td>{{.}}</td>
                                        {{end}}
                                    </tr>
                                    {{end}}
                                </tbody>
                            </table>
                        </div>
                    {{else if eq .Type "alert"}}
                        <div class="alert-widget">
                            {{range index .Data "alerts"}}
                            <div class="alert alert-{{.Severity}}">
                                <div class="alert-title">{{.Title}}</div>
                                <div class="alert-message">{{.Message}}</div>
                                <div class="alert-time">{{.Timestamp.Format "15:04:05"}}</div>
                            </div>
                            {{end}}
                        </div>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>
    </div>
    
    <script src="dashboard.js"></script>
    <script>
        // Initialize dashboard with data
        const dashboardData = {{.Data | toJSON}};
        initializeDashboard(dashboardData);
    </script>
</body>
</html>`
	
	tmpl, err := template.New("dashboard").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return err
	}
	
	file, err := os.Create(filepath.Join(d.config.OutputDir, "dashboard.html"))
	if err != nil {
		return err
	}
	defer file.Close()
	
	return tmpl.Execute(file, dashboard)
}

// generateJSONData generates JSON data file for the dashboard
func (d *DashboardGenerator) generateJSONData(dashboard *Dashboard) error {
	file, err := os.Create(filepath.Join(d.config.OutputDir, "dashboard-data.json"))
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(dashboard)
}

// generateCSS generates CSS file for dashboard styling
func (d *DashboardGenerator) generateCSS() error {
	css := `
/* Dashboard CSS */
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    background-color: #f5f5f5;
    color: #333;
}

.dashboard-container {
    max-width: 1400px;
    margin: 0 auto;
    padding: 20px;
}

.dashboard-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: white;
    padding: 20px;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    margin-bottom: 20px;
}

.dashboard-header h1 {
    color: #2c3e50;
}

.dashboard-info {
    display: flex;
    gap: 20px;
    font-size: 14px;
    color: #666;
}

.dashboard-grid {
    display: grid;
    grid-template-columns: repeat(12, 1fr);
    gap: 20px;
    grid-auto-rows: 120px;
}

.widget {
    background: white;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    padding: 16px;
    transition: transform 0.2s, box-shadow 0.2s;
}

.widget:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 8px rgba(0,0,0,0.15);
}

.widget-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
}

.widget-header h3 {
    font-size: 16px;
    color: #2c3e50;
}

.widget-status {
    width: 12px;
    height: 12px;
    border-radius: 50%;
}

.status-good { background-color: #27ae60; }
.status-warning { background-color: #f39c12; }
.status-error { background-color: #e74c3c; }

.widget-content {
    height: calc(100% - 40px);
    display: flex;
    flex-direction: column;
    justify-content: center;
}

.metric-widget {
    text-align: center;
}

.metric-value {
    font-size: 2.5em;
    font-weight: bold;
    color: #2c3e50;
}

.metric-target {
    font-size: 0.9em;
    color: #666;
    margin-top: 8px;
}

.progress-widget {
    text-align: center;
}

.progress-bar {
    width: 100%;
    height: 20px;
    background-color: #ecf0f1;
    border-radius: 10px;
    overflow: hidden;
    margin-bottom: 10px;
}

.progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #27ae60, #2ecc71);
    transition: width 0.3s ease;
}

.progress-text {
    font-weight: bold;
    color: #2c3e50;
}

.table-widget {
    overflow: auto;
    height: 100%;
}

.table-widget table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
}

.table-widget th,
.table-widget td {
    padding: 8px;
    text-align: left;
    border-bottom: 1px solid #ecf0f1;
}

.table-widget th {
    background-color: #f8f9fa;
    font-weight: bold;
}

.alert-widget {
    overflow-y: auto;
    height: 100%;
}

.alert {
    padding: 8px;
    margin-bottom: 8px;
    border-radius: 4px;
    border-left: 4px solid;
}

.alert-critical {
    background-color: #fdf2f2;
    border-left-color: #e74c3c;
}

.alert-warning {
    background-color: #fefbf3;
    border-left-color: #f39c12;
}

.alert-high {
    background-color: #fff5f5;
    border-left-color: #e53e3e;
}

.alert-title {
    font-weight: bold;
    font-size: 12px;
}

.alert-message {
    font-size: 11px;
    margin: 4px 0;
}

.alert-time {
    font-size: 10px;
    color: #666;
}

/* Responsive design */
@media (max-width: 768px) {
    .dashboard-grid {
        grid-template-columns: 1fr;
    }
    
    .widget {
        grid-column: span 1 !important;
    }
}
`
	
	file, err := os.Create(filepath.Join(d.config.OutputDir, "dashboard.css"))
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = file.WriteString(css)
	return err
}

// generateJavaScript generates JavaScript file for dashboard interactivity
func (d *DashboardGenerator) generateJavaScript() error {
	js := `
// Dashboard JavaScript
function initializeDashboard(data) {
    console.log('Initializing dashboard with data:', data);
    
    // Initialize charts
    initializeCharts(data.trend_data);
    
    // Set up auto-refresh
    setInterval(refreshDashboard, 30000); // Refresh every 30 seconds
}

function initializeCharts(trendData) {
    // Coverage trend chart
    const coverageCtx = document.getElementById('chart-coverage-trend');
    if (coverageCtx) {
        new Chart(coverageCtx, {
            type: 'line',
            data: {
                labels: trendData.coverage.map(point => new Date(point.timestamp).toLocaleDateString()),
                datasets: [{
                    label: 'Coverage %',
                    data: trendData.coverage.map(point => point.value),
                    borderColor: '#3498db',
                    backgroundColor: 'rgba(52, 152, 219, 0.1)',
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: false,
                        min: Math.min(...trendData.coverage.map(p => p.value)) - 5,
                        max: 100
                    }
                }
            }
        });
    }
    
    // Security trend chart
    const securityCtx = document.getElementById('chart-security-trend');
    if (securityCtx) {
        new Chart(securityCtx, {
            type: 'line',
            data: {
                labels: trendData.security.map(point => new Date(point.timestamp).toLocaleDateString()),
                datasets: [{
                    label: 'Security Score',
                    data: trendData.security.map(point => point.value),
                    borderColor: '#e74c3c',
                    backgroundColor: 'rgba(231, 76, 60, 0.1)',
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: false,
                        min: 0,
                        max: 100
                    }
                }
            }
        });
    }
}

function refreshDashboard() {
    // Fetch updated data and refresh widgets
    fetch('dashboard-data.json')
        .then(response => response.json())
        .then(data => {
            updateWidgets(data);
        })
        .catch(error => {
            console.error('Failed to refresh dashboard:', error);
        });
}

function updateWidgets(data) {
    // Update metric widgets
    updateMetricWidgets(data);
    
    // Update progress widgets
    updateProgressWidgets(data);
    
    // Update alerts
    updateAlerts(data.data.alerts);
}

function updateMetricWidgets(data) {
    // This would update metric values in real-time
    console.log('Updating metric widgets');
}

function updateProgressWidgets(data) {
    // This would update progress bars in real-time
    console.log('Updating progress widgets');
}

function updateAlerts(alerts) {
    // This would update alerts in real-time
    console.log('Updating alerts:', alerts);
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    console.log('Dashboard DOM loaded');
});
`
	
	file, err := os.Create(filepath.Join(d.config.OutputDir, "dashboard.js"))
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = file.WriteString(js)
	return err
}