package testing

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// MutationReporter handles reporting and analysis of mutation testing results
type MutationReporter struct {
	outputDir string
}

// NewMutationReporter creates a new mutation reporter
func NewMutationReporter() *MutationReporter {
	return &MutationReporter{
		outputDir: "mutation_reports",
	}
}

// GenerateReport creates comprehensive mutation testing reports
func (mr *MutationReporter) GenerateReport(report *MutationReport) error {
	// Ensure output directory exists
	if err := os.MkdirAll(mr.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Generate JSON report
	if err := mr.generateJSONReport(report); err != nil {
		return fmt.Errorf("failed to generate JSON report: %w", err)
	}
	
	// Generate HTML report
	if err := mr.generateHTMLReport(report); err != nil {
		return fmt.Errorf("failed to generate HTML report: %w", err)
	}
	
	// Generate CSV report for data analysis
	if err := mr.generateCSVReport(report); err != nil {
		return fmt.Errorf("failed to generate CSV report: %w", err)
	}
	
	// Generate trend analysis if historical data exists
	if err := mr.generateTrendAnalysis(report); err != nil {
		return fmt.Errorf("failed to generate trend analysis: %w", err)
	}
	
	return nil
}

// generateJSONReport creates a detailed JSON report
func (mr *MutationReporter) generateJSONReport(report *MutationReport) error {
	filename := filepath.Join(mr.outputDir, fmt.Sprintf("mutation_report_%s.json", 
		report.Timestamp.Format("2006-01-02_15-04-05")))
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// generateHTMLReport creates a visual HTML report
func (mr *MutationReporter) generateHTMLReport(report *MutationReport) error {
	filename := filepath.Join(mr.outputDir, fmt.Sprintf("mutation_report_%s.html", 
		report.Timestamp.Format("2006-01-02_15-04-05")))
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	tmpl := template.Must(template.New("report").Parse(htmlReportTemplate))
	
	data := struct {
		*MutationReport
		GeneratedAt string
	}{
		MutationReport: report,
		GeneratedAt:    time.Now().Format("2006-01-02 15:04:05"),
	}
	
	return tmpl.Execute(file, data)
}

// generateCSVReport creates a CSV report for data analysis
func (mr *MutationReporter) generateCSVReport(report *MutationReport) error {
	filename := filepath.Join(mr.outputDir, fmt.Sprintf("mutation_data_%s.csv", 
		report.Timestamp.Format("2006-01-02_15-04-05")))
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write CSV header
	fmt.Fprintln(file, "ID,FilePath,Function,MutationType,Category,LineNumber,Killed,ExecutionTime,TestsPassed")
	
	// Write data rows
	for _, result := range report.Results {
		fmt.Fprintf(file, "%s,%s,%s,%s,%s,%d,%t,%s,%t\n",
			result.ID,
			result.FilePath,
			result.Function,
			result.MutationType,
			result.Category,
			result.LineNumber,
			result.Killed,
			result.ExecutionTime.String(),
			result.TestsPassed,
		)
	}
	
	return nil
}

// generateTrendAnalysis creates trend analysis from historical data
func (mr *MutationReporter) generateTrendAnalysis(currentReport *MutationReport) error {
	// Load historical reports
	historicalReports, err := mr.loadHistoricalReports()
	if err != nil {
		return err
	}
	
	// Add current report to history
	historicalReports = append(historicalReports, currentReport)
	
	// Generate trend analysis
	trendData := mr.analyzeTrends(historicalReports)
	
	// Save trend analysis
	filename := filepath.Join(mr.outputDir, "mutation_trends.json")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(trendData)
}

// TrendAnalysis represents mutation testing trends over time
type TrendAnalysis struct {
	Reports          []TrendDataPoint `json:"reports"`
	OverallTrend     string          `json:"overall_trend"`
	CategoryTrends   map[string]string `json:"category_trends"`
	Recommendations  []string        `json:"recommendations"`
}

// TrendDataPoint represents a single point in the trend analysis
type TrendDataPoint struct {
	Timestamp      time.Time         `json:"timestamp"`
	MutationScore  float64          `json:"mutation_score"`
	CategoryScores map[string]float64 `json:"category_scores"`
	TotalMutations int              `json:"total_mutations"`
	WeakTestCount  int              `json:"weak_test_count"`
}

// loadHistoricalReports loads previous mutation reports for trend analysis
func (mr *MutationReporter) loadHistoricalReports() ([]*MutationReport, error) {
	var reports []*MutationReport
	
	files, err := filepath.Glob(filepath.Join(mr.outputDir, "mutation_report_*.json"))
	if err != nil {
		return reports, err
	}
	
	for _, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			continue
		}
		
		var report MutationReport
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&report); err != nil {
			file.Close()
			continue
		}
		
		reports = append(reports, &report)
		file.Close()
	}
	
	// Sort by timestamp
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Timestamp.Before(reports[j].Timestamp)
	})
	
	return reports, nil
}

// analyzeTrends analyzes trends in mutation testing results
func (mr *MutationReporter) analyzeTrends(reports []*MutationReport) *TrendAnalysis {
	if len(reports) == 0 {
		return &TrendAnalysis{}
	}
	
	trendData := &TrendAnalysis{
		Reports:        make([]TrendDataPoint, len(reports)),
		CategoryTrends: make(map[string]string),
	}
	
	// Convert reports to trend data points
	for i, report := range reports {
		trendData.Reports[i] = TrendDataPoint{
			Timestamp:      report.Timestamp,
			MutationScore:  report.MutationScore,
			CategoryScores: report.CategoryScores,
			TotalMutations: report.TotalMutations,
			WeakTestCount:  len(report.WeakTests),
		}
	}
	
	// Analyze overall trend
	if len(reports) >= 2 {
		firstScore := reports[0].MutationScore
		lastScore := reports[len(reports)-1].MutationScore
		
		if lastScore > firstScore+5 {
			trendData.OverallTrend = "improving"
		} else if lastScore < firstScore-5 {
			trendData.OverallTrend = "declining"
		} else {
			trendData.OverallTrend = "stable"
		}
	}
	
	// Analyze category trends
	categories := []string{"business_logic", "security", "performance"}
	for _, category := range categories {
		if len(reports) >= 2 {
			firstScore := reports[0].CategoryScores[category]
			lastScore := reports[len(reports)-1].CategoryScores[category]
			
			if lastScore > firstScore+5 {
				trendData.CategoryTrends[category] = "improving"
			} else if lastScore < firstScore-5 {
				trendData.CategoryTrends[category] = "declining"
			} else {
				trendData.CategoryTrends[category] = "stable"
			}
		}
	}
	
	// Generate trend-based recommendations
	trendData.Recommendations = mr.generateTrendRecommendations(trendData)
	
	return trendData
}

// generateTrendRecommendations creates recommendations based on trend analysis
func (mr *MutationReporter) generateTrendRecommendations(trendData *TrendAnalysis) []string {
	var recommendations []string
	
	if trendData.OverallTrend == "declining" {
		recommendations = append(recommendations, 
			"Overall mutation score is declining. Review recent code changes and strengthen tests.")
	}
	
	for category, trend := range trendData.CategoryTrends {
		if trend == "declining" {
			switch category {
			case "security":
				recommendations = append(recommendations, 
					"Security mutation score is declining. Focus on security test coverage.")
			case "business_logic":
				recommendations = append(recommendations, 
					"Business logic mutation score is declining. Add more edge case tests.")
			case "performance":
				recommendations = append(recommendations, 
					"Performance mutation score is declining. Add performance validation tests.")
			}
		}
	}
	
	// Check for consistent weak test patterns
	if len(trendData.Reports) >= 3 {
		recentWeakTests := 0
		for i := len(trendData.Reports) - 3; i < len(trendData.Reports); i++ {
			recentWeakTests += trendData.Reports[i].WeakTestCount
		}
		
		if recentWeakTests > 10 {
			recommendations = append(recommendations, 
				"Consistently high number of weak tests detected. Consider test review sessions.")
		}
	}
	
	return recommendations
}

// TestQualityAnalyzer provides detailed analysis of test quality
type TestQualityAnalyzer struct {
	mutationResults []MutationResult
}

// NewTestQualityAnalyzer creates a new test quality analyzer
func NewTestQualityAnalyzer(results []MutationResult) *TestQualityAnalyzer {
	return &TestQualityAnalyzer{
		mutationResults: results,
	}
}

// AnalyzeTestEffectiveness analyzes how effective tests are at catching mutations
func (tqa *TestQualityAnalyzer) AnalyzeTestEffectiveness() *TestEffectivenessReport {
	report := &TestEffectivenessReport{
		FunctionAnalysis: make(map[string]*FunctionTestQuality),
		CategoryAnalysis: make(map[string]*CategoryTestQuality),
	}
	
	// Analyze by function
	functionStats := make(map[string]*FunctionTestQuality)
	for _, result := range tqa.mutationResults {
		key := fmt.Sprintf("%s::%s", result.FilePath, result.Function)
		
		if functionStats[key] == nil {
			functionStats[key] = &FunctionTestQuality{
				FunctionName: result.Function,
				FilePath:     result.FilePath,
			}
		}
		
		stats := functionStats[key]
		stats.TotalMutations++
		if result.Killed {
			stats.KilledMutations++
		} else {
			stats.SurvivedMutations = append(stats.SurvivedMutations, result.MutationType)
		}
	}
	
	// Calculate effectiveness scores
	for key, stats := range functionStats {
		if stats.TotalMutations > 0 {
			stats.EffectivenessScore = float64(stats.KilledMutations) / float64(stats.TotalMutations) * 100
		}
		
		// Determine quality level
		if stats.EffectivenessScore >= 90 {
			stats.QualityLevel = "excellent"
		} else if stats.EffectivenessScore >= 75 {
			stats.QualityLevel = "good"
		} else if stats.EffectivenessScore >= 50 {
			stats.QualityLevel = "fair"
		} else {
			stats.QualityLevel = "poor"
		}
		
		report.FunctionAnalysis[key] = stats
	}
	
	// Analyze by category
	categoryStats := make(map[string]*CategoryTestQuality)
	for _, result := range tqa.mutationResults {
		if categoryStats[result.Category] == nil {
			categoryStats[result.Category] = &CategoryTestQuality{
				Category: result.Category,
			}
		}
		
		stats := categoryStats[result.Category]
		stats.TotalMutations++
		if result.Killed {
			stats.KilledMutations++
		}
	}
	
	for category, stats := range categoryStats {
		if stats.TotalMutations > 0 {
			stats.EffectivenessScore = float64(stats.KilledMutations) / float64(stats.TotalMutations) * 100
		}
		report.CategoryAnalysis[category] = stats
	}
	
	return report
}

// TestEffectivenessReport contains detailed test effectiveness analysis
type TestEffectivenessReport struct {
	FunctionAnalysis map[string]*FunctionTestQuality `json:"function_analysis"`
	CategoryAnalysis map[string]*CategoryTestQuality `json:"category_analysis"`
}

// FunctionTestQuality represents test quality for a specific function
type FunctionTestQuality struct {
	FunctionName        string   `json:"function_name"`
	FilePath           string   `json:"file_path"`
	TotalMutations     int      `json:"total_mutations"`
	KilledMutations    int      `json:"killed_mutations"`
	SurvivedMutations  []string `json:"survived_mutations"`
	EffectivenessScore float64  `json:"effectiveness_score"`
	QualityLevel       string   `json:"quality_level"`
}

// CategoryTestQuality represents test quality for a category
type CategoryTestQuality struct {
	Category           string  `json:"category"`
	TotalMutations     int     `json:"total_mutations"`
	KilledMutations    int     `json:"killed_mutations"`
	EffectivenessScore float64 `json:"effectiveness_score"`
}

// HTML template for mutation testing report
const htmlReportTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Mutation Testing Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .summary { display: flex; justify-content: space-around; margin: 20px 0; }
        .metric { text-align: center; padding: 15px; background-color: #e8f4f8; border-radius: 5px; }
        .metric h3 { margin: 0; color: #2c3e50; }
        .metric .value { font-size: 2em; font-weight: bold; color: #3498db; }
        .section { margin: 30px 0; }
        .category-scores { display: flex; justify-content: space-around; margin: 20px 0; }
        .category { text-align: center; padding: 15px; background-color: #f8f9fa; border-radius: 5px; }
        .weak-tests { background-color: #fff3cd; padding: 15px; border-radius: 5px; border-left: 4px solid #ffc107; }
        .recommendations { background-color: #d1ecf1; padding: 15px; border-radius: 5px; border-left: 4px solid #17a2b8; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f2f2f2; }
        .killed { color: #28a745; }
        .survived { color: #dc3545; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Mutation Testing Report</h1>
        <p>Generated: {{.GeneratedAt}}</p>
        <p>Test Run: {{.Timestamp.Format "2006-01-02 15:04:05"}}</p>
    </div>

    <div class="summary">
        <div class="metric">
            <h3>Mutation Score</h3>
            <div class="value">{{printf "%.1f%%" .MutationScore}}</div>
        </div>
        <div class="metric">
            <h3>Total Mutations</h3>
            <div class="value">{{.TotalMutations}}</div>
        </div>
        <div class="metric">
            <h3>Killed</h3>
            <div class="value">{{.KilledMutations}}</div>
        </div>
        <div class="metric">
            <h3>Survived</h3>
            <div class="value">{{.SurvivedMutations}}</div>
        </div>
    </div>

    <div class="section">
        <h2>Category Scores</h2>
        <div class="category-scores">
            {{range $category, $score := .CategoryScores}}
            <div class="category">
                <h3>{{$category}}</h3>
                <div class="value">{{printf "%.1f%%" $score}}</div>
            </div>
            {{end}}
        </div>
    </div>

    {{if .WeakTests}}
    <div class="section">
        <h2>Weak Tests</h2>
        <div class="weak-tests">
            <p>The following tests failed to catch mutations and may need strengthening:</p>
            {{range .WeakTests}}
            <div style="margin: 10px 0;">
                <strong>{{.TestFunction}}</strong> ({{.Severity}} severity)
                <ul>
                    {{range .MissedMutations}}
                    <li>{{.}}</li>
                    {{end}}
                </ul>
                <p><em>Suggestions:</em></p>
                <ul>
                    {{range .Suggestions}}
                    <li>{{.}}</li>
                    {{end}}
                </ul>
            </div>
            {{end}}
        </div>
    </div>
    {{end}}

    {{if .Recommendations}}
    <div class="section">
        <h2>Recommendations</h2>
        <div class="recommendations">
            <ul>
                {{range .Recommendations}}
                <li>{{.}}</li>
                {{end}}
            </ul>
        </div>
    </div>
    {{end}}

    <div class="section">
        <h2>Detailed Results</h2>
        <table>
            <thead>
                <tr>
                    <th>File</th>
                    <th>Function</th>
                    <th>Mutation Type</th>
                    <th>Category</th>
                    <th>Line</th>
                    <th>Status</th>
                    <th>Execution Time</th>
                </tr>
            </thead>
            <tbody>
                {{range .Results}}
                <tr>
                    <td>{{.FilePath}}</td>
                    <td>{{.Function}}</td>
                    <td>{{.MutationType}}</td>
                    <td>{{.Category}}</td>
                    <td>{{.LineNumber}}</td>
                    <td class="{{if .Killed}}killed{{else}}survived{{end}}">
                        {{if .Killed}}KILLED{{else}}SURVIVED{{end}}
                    </td>
                    <td>{{.ExecutionTime}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</body>
</html>
`