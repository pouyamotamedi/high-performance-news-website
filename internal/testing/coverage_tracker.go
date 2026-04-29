package testing

import (
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// CoverageTracker tracks code coverage trends over time
type CoverageTracker struct {
	db *sql.DB
}

// NewCoverageTracker creates a new coverage tracker
func NewCoverageTracker(db *sql.DB) *CoverageTracker {
	tracker := &CoverageTracker{db: db}
	tracker.initializeTables()
	return tracker
}

// initializeTables creates the coverage tracking tables
func (c *CoverageTracker) initializeTables() error {
	query := `
		CREATE TABLE IF NOT EXISTS coverage_trends (
			id BIGSERIAL PRIMARY KEY,
			date DATE NOT NULL,
			coverage_percent DECIMAL(5,2) NOT NULL,
			test_count BIGINT NOT NULL,
			lines_total BIGINT NOT NULL,
			lines_covered BIGINT NOT NULL,
			branch_coverage DECIMAL(5,2) DEFAULT 0,
			function_coverage DECIMAL(5,2) DEFAULT 0,
			package_name VARCHAR(255),
			commit_hash VARCHAR(64),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(date, package_name)
		)
	`

	_, err := c.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create coverage_trends table: %w", err)
	}

	// Create index for efficient querying
	indexQuery := `CREATE INDEX IF NOT EXISTS idx_coverage_trends_date ON coverage_trends(date DESC)`
	_, err = c.db.Exec(indexQuery)
	
	return err
}

// RecordCoverage records coverage data for a specific date
func (c *CoverageTracker) RecordCoverage(coverage *CoverageTrend) error {
	query := `
		INSERT INTO coverage_trends (date, coverage_percent, test_count, lines_total, lines_covered, 
			branch_coverage, function_coverage, package_name, commit_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (date, package_name) DO UPDATE SET
			coverage_percent = EXCLUDED.coverage_percent,
			test_count = EXCLUDED.test_count,
			lines_total = EXCLUDED.lines_total,
			lines_covered = EXCLUDED.lines_covered,
			branch_coverage = EXCLUDED.branch_coverage,
			function_coverage = EXCLUDED.function_coverage,
			commit_hash = EXCLUDED.commit_hash
	`

	_, err := c.db.Exec(query, coverage.Date, coverage.CoveragePercent, coverage.TestCount,
		coverage.LinesTotal, coverage.LinesCovered, 0, 0, "", "")
	
	if err != nil {
		return fmt.Errorf("failed to record coverage: %w", err)
	}

	return nil
}

// GetCoverageTrends returns coverage trends for the specified number of days
func (c *CoverageTracker) GetCoverageTrends(days int) ([]CoverageTrend, error) {
	query := `
		SELECT date, coverage_percent, test_count, lines_total, lines_covered
		FROM coverage_trends
		WHERE date >= CURRENT_DATE - INTERVAL '%d days'
		AND package_name IS NULL OR package_name = ''
		ORDER BY date DESC
		LIMIT $1
	`

	rows, err := c.db.Query(fmt.Sprintf(query, days), days)
	if err != nil {
		return nil, fmt.Errorf("failed to get coverage trends: %w", err)
	}
	defer rows.Close()

	var trends []CoverageTrend
	for rows.Next() {
		var trend CoverageTrend
		
		err := rows.Scan(&trend.Date, &trend.CoveragePercent, &trend.TestCount,
			&trend.LinesTotal, &trend.LinesCovered)
		if err != nil {
			continue
		}

		trends = append(trends, trend)
	}

	return trends, nil
}

// GetCurrentCoverage gets the current coverage by running go test with coverage
func (c *CoverageTracker) GetCurrentCoverage() (*CoverageTrend, error) {
	// Run go test with coverage
	cmd := exec.Command("go", "test", "-coverprofile=coverage.out", "./...")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to run coverage tests: %v\nOutput: %s", err, string(output))
		return nil, fmt.Errorf("failed to run coverage tests: %w", err)
	}

	// Parse coverage output
	coverage, err := c.parseCoverageOutput(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse coverage output: %w", err)
	}

	// Get detailed coverage info
	cmd = exec.Command("go", "tool", "cover", "-func=coverage.out")
	funcOutput, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to get function coverage: %v", err)
	} else {
		c.parseFunctionCoverage(string(funcOutput), coverage)
	}

	coverage.Date = time.Now()
	return coverage, nil
}

// parseCoverageOutput parses the output from go test -cover
func (c *CoverageTracker) parseCoverageOutput(output string) (*CoverageTrend, error) {
	lines := strings.Split(output, "\n")
	
	coverage := &CoverageTrend{
		Date: time.Now(),
	}

	testCount := int64(0)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Count test runs
		if strings.Contains(line, "RUN") || strings.Contains(line, "PASS") {
			if strings.Contains(line, "Test") {
				testCount++
			}
		}
		
		// Parse coverage percentage
		if strings.Contains(line, "coverage:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "coverage:" && i+1 < len(parts) {
					coverageStr := strings.TrimSuffix(parts[i+1], "%")
					if percent, err := strconv.ParseFloat(coverageStr, 64); err == nil {
						coverage.CoveragePercent = percent
					}
				}
			}
		}
	}

	coverage.TestCount = testCount
	return coverage, nil
}

// parseFunctionCoverage parses function coverage details
func (c *CoverageTracker) parseFunctionCoverage(output string, coverage *CoverageTrend) {
	lines := strings.Split(output, "\n")
	
	totalLines := int64(0)
	coveredLines := int64(0)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total:") {
			continue
		}
		
		// Parse function coverage line: filename:function coverage%
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			// Extract coverage percentage
			coverageStr := parts[len(parts)-1]
			coverageStr = strings.TrimSuffix(coverageStr, "%")
			
			if percent, err := strconv.ParseFloat(coverageStr, 64); err == nil {
				// Estimate lines (this is approximate)
				estimatedLines := int64(10) // Average function size
				totalLines += estimatedLines
				coveredLines += int64(float64(estimatedLines) * percent / 100.0)
			}
		}
	}
	
	coverage.LinesTotal = totalLines
	coverage.LinesCovered = coveredLines
}

// UpdateDailyCoverage updates the daily coverage record
func (c *CoverageTracker) UpdateDailyCoverage() error {
	coverage, err := c.GetCurrentCoverage()
	if err != nil {
		return fmt.Errorf("failed to get current coverage: %w", err)
	}

	return c.RecordCoverage(coverage)
}

// GetCoverageByPackage returns coverage trends by package
func (c *CoverageTracker) GetCoverageByPackage(packageName string, days int) ([]CoverageTrend, error) {
	query := `
		SELECT date, coverage_percent, test_count, lines_total, lines_covered
		FROM coverage_trends
		WHERE package_name = $1
		AND date >= CURRENT_DATE - INTERVAL '%d days'
		ORDER BY date DESC
		LIMIT $2
	`

	rows, err := c.db.Query(fmt.Sprintf(query, days), packageName, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get package coverage trends: %w", err)
	}
	defer rows.Close()

	var trends []CoverageTrend
	for rows.Next() {
		var trend CoverageTrend
		
		err := rows.Scan(&trend.Date, &trend.CoveragePercent, &trend.TestCount,
			&trend.LinesTotal, &trend.LinesCovered)
		if err != nil {
			continue
		}

		trends = append(trends, trend)
	}

	return trends, nil
}

// GetCoverageDrops returns significant coverage drops
func (c *CoverageTracker) GetCoverageDrops(threshold float64) ([]CoverageAlert, error) {
	query := `
		WITH coverage_changes AS (
			SELECT 
				date,
				coverage_percent,
				LAG(coverage_percent) OVER (ORDER BY date) as prev_coverage,
				coverage_percent - LAG(coverage_percent) OVER (ORDER BY date) as coverage_change
			FROM coverage_trends
			WHERE date >= CURRENT_DATE - INTERVAL '30 days'
			AND (package_name IS NULL OR package_name = '')
			ORDER BY date
		)
		SELECT date, coverage_percent, prev_coverage, coverage_change
		FROM coverage_changes
		WHERE coverage_change < -$1
		ORDER BY date DESC
	`

	rows, err := c.db.Query(query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get coverage drops: %w", err)
	}
	defer rows.Close()

	var alerts []CoverageAlert
	for rows.Next() {
		var alert CoverageAlert
		var prevCoverage sql.NullFloat64
		var coverageChange sql.NullFloat64
		
		err := rows.Scan(&alert.Date, &alert.CurrentCoverage, &prevCoverage, &coverageChange)
		if err != nil {
			continue
		}

		if prevCoverage.Valid {
			alert.PreviousCoverage = prevCoverage.Float64
		}
		if coverageChange.Valid {
			alert.CoverageChange = coverageChange.Float64
		}

		alert.Severity = c.assessCoverageDropSeverity(alert.CoverageChange)
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// assessCoverageDropSeverity assesses the severity of a coverage drop
func (c *CoverageTracker) assessCoverageDropSeverity(drop float64) string {
	if drop <= -10.0 {
		return "critical"
	} else if drop <= -5.0 {
		return "high"
	} else if drop <= -2.0 {
		return "medium"
	}
	return "low"
}

// CoverageAlert represents a coverage alert
type CoverageAlert struct {
	Date             time.Time `json:"date"`
	CurrentCoverage  float64   `json:"current_coverage"`
	PreviousCoverage float64   `json:"previous_coverage"`
	CoverageChange   float64   `json:"coverage_change"`
	Severity         string    `json:"severity"`
}

// GetCoverageSummary returns a summary of coverage statistics
func (c *CoverageTracker) GetCoverageSummary() (*CoverageSummary, error) {
	query := `
		SELECT 
			AVG(coverage_percent) as avg_coverage,
			MIN(coverage_percent) as min_coverage,
			MAX(coverage_percent) as max_coverage,
			COUNT(*) as data_points
		FROM coverage_trends
		WHERE date >= CURRENT_DATE - INTERVAL '30 days'
		AND (package_name IS NULL OR package_name = '')
	`

	var summary CoverageSummary
	var avgCoverage, minCoverage, maxCoverage sql.NullFloat64
	
	err := c.db.QueryRow(query).Scan(&avgCoverage, &minCoverage, &maxCoverage, &summary.DataPoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get coverage summary: %w", err)
	}

	if avgCoverage.Valid {
		summary.AverageCoverage = avgCoverage.Float64
	}
	if minCoverage.Valid {
		summary.MinCoverage = minCoverage.Float64
	}
	if maxCoverage.Valid {
		summary.MaxCoverage = maxCoverage.Float64
	}

	// Get trend direction
	trendQuery := `
		SELECT coverage_percent
		FROM coverage_trends
		WHERE date >= CURRENT_DATE - INTERVAL '7 days'
		AND (package_name IS NULL OR package_name = '')
		ORDER BY date
		LIMIT 2
	`

	rows, err := c.db.Query(trendQuery)
	if err == nil {
		defer rows.Close()
		
		var coverages []float64
		for rows.Next() {
			var coverage float64
			if err := rows.Scan(&coverage); err == nil {
				coverages = append(coverages, coverage)
			}
		}
		
		if len(coverages) >= 2 {
			if coverages[len(coverages)-1] > coverages[0] {
				summary.Trend = "increasing"
			} else if coverages[len(coverages)-1] < coverages[0] {
				summary.Trend = "decreasing"
			} else {
				summary.Trend = "stable"
			}
		}
	}

	return &summary, nil
}

// CoverageSummary represents coverage summary statistics
type CoverageSummary struct {
	AverageCoverage float64 `json:"average_coverage"`
	MinCoverage     float64 `json:"min_coverage"`
	MaxCoverage     float64 `json:"max_coverage"`
	DataPoints      int64   `json:"data_points"`
	Trend           string  `json:"trend"` // "increasing", "decreasing", "stable"
}