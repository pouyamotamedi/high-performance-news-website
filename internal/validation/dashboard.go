package validation

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// DashboardServer serves the quality dashboard web interface
type DashboardServer struct {
	config    DashboardConfig
	reporting *ReportingSystem
	server    *http.Server
}

// NewDashboardServer creates a new dashboard server
func NewDashboardServer(config DashboardConfig, reporting *ReportingSystem) *DashboardServer {
	return &DashboardServer{
		config:    config,
		reporting: reporting,
	}
}

// Start starts the dashboard server
func (d *DashboardServer) Start() error {
	if !d.config.Enabled {
		return nil
	}

	mux := http.NewServeMux()
	
	// API endpoints
	mux.HandleFunc("/api/latest-report", d.handleLatestReport)
	mux.HandleFunc("/api/trends", d.handleTrends)
	mux.HandleFunc("/api/validate", d.handleValidate)
	mux.HandleFunc("/api/export", d.handleExport)
	
	// Static dashboard
	mux.HandleFunc("/", d.handleDashboard)
	mux.HandleFunc("/dashboard", d.handleDashboard)
	
	d.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", d.config.Port),
		Handler: mux,
	}

	log.Printf("Starting AI Code Quality Dashboard on port %d", d.config.Port)
	return d.server.ListenAndServe()
}

// Stop stops the dashboard server
func (d *DashboardServer) Stop() error {
	if d.server != nil {
		return d.server.Close()
	}
	return nil
}

// handleLatestReport serves the latest validation report
func (d *DashboardServer) handleLatestReport(w http.ResponseWriter, r *http.Request) {
	latestPath := filepath.Join(d.reporting.config.OutputDir, "latest-report.json")
	
	data, err := os.ReadFile(latestPath)
	if err != nil {
		http.Error(w, "Latest report not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// handleTrends serves quality trends data
func (d *DashboardServer) handleTrends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if err := json.NewEncoder(w).Encode(d.reporting.trends); err != nil {
		http.Error(w, "Failed to encode trends", http.StatusInternalServerError)
		return
	}
}

// handleValidate triggers validation of specified files
func (d *DashboardServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		Files []string `json:"files"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	report, err := d.reporting.ValidateAndReport(request.Files)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// handleExport exports reports in various formats
func (d *DashboardServer) handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	
	// Load latest report
	latestPath := filepath.Join(d.reporting.config.OutputDir, "latest-report.json")
	data, err := os.ReadFile(latestPath)
	if err != nil {
		http.Error(w, "Latest report not found", http.StatusNotFound)
		return
	}
	
	var report AggregatedReport
	if err := json.Unmarshal(data, &report); err != nil {
		http.Error(w, "Failed to parse report", http.StatusInternalServerError)
		return
	}
	
	// Set appropriate content type and filename
	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=validation-report.csv")
	case "junit":
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", "attachment; filename=validation-report.xml")
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=validation-report.json")
	}
	
	if err := d.reporting.ExportReport(&report, format, w); err != nil {
		http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleDashboard serves the main dashboard interface
func (d *DashboardServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	dashboardHTML := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Live Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #f8f9fa; }
        .header { background: #343a40; color: white; padding: 1rem 2rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header h1 { display: inline-block; }
        .refresh-info { float: right; font-size: 0.9rem; opacity: 0.8; }
        .container { max-width: 1400px; margin: 2rem auto; padding: 0 2rem; }
        .dashboard-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 2rem; margin-bottom: 2rem; }
        .card { background: white; border-radius: 8px; padding: 1.5rem; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .score-card { text-align: center; }
        .score { font-size: 4rem; font-weight: bold; margin: 1rem 0; }
        .grade-A { color: #28a745; }
        .grade-B { color: #17a2b8; }
        .grade-C { color: #ffc107; }
        .grade-D { color: #fd7e14; }
        .grade-F { color: #dc3545; }
        .metrics-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1rem; }
        .metric { text-align: center; padding: 1rem; background: #f8f9fa; border-radius: 6px; }
        .metric-value { font-size: 2rem; font-weight: bold; margin-bottom: 0.5rem; }
        .critical { color: #dc3545; }
        .high { color: #fd7e14; }
        .medium { color: #ffc107; }
        .low { color: #28a745; }
        .chart-container { height: 300px; margin: 1rem 0; }
        .loading { text-align: center; padding: 2rem; color: #6c757d; }
        .error { background: #f8d7da; color: #721c24; padding: 1rem; border-radius: 6px; margin: 1rem 0; }
        .success { background: #d4edda; color: #155724; padding: 1rem; border-radius: 6px; margin: 1rem 0; }
        .actions { margin: 2rem 0; }
        .btn { padding: 0.75rem 1.5rem; border: none; border-radius: 6px; cursor: pointer; font-size: 1rem; margin-right: 1rem; }
        .btn-primary { background: #007bff; color: white; }
        .btn-secondary { background: #6c757d; color: white; }
        .btn:hover { opacity: 0.9; }
        .trends-chart { width: 100%; height: 200px; background: #f8f9fa; border-radius: 6px; display: flex; align-items: center; justify-content: center; }
        .file-list { max-height: 300px; overflow-y: auto; }
        .file-item { padding: 0.5rem; border-bottom: 1px solid #dee2e6; display: flex; justify-content: between; }
        .file-issues { font-weight: bold; }
        .timestamp { color: #6c757d; font-size: 0.9rem; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.Title}}</h1>
        <div class="refresh-info">
            Auto-refresh: {{.RefreshRate}}s | Last updated: <span id="lastUpdate">Loading...</span>
        </div>
    </div>

    <div class="container">
        <div id="loading" class="loading">Loading dashboard data...</div>
        <div id="error" class="error" style="display: none;"></div>
        
        <div id="dashboard" style="display: none;">
            <div class="dashboard-grid">
                <div class="card score-card">
                    <h2>Overall Quality Score</h2>
                    <div id="score" class="score">-</div>
                    <div>Grade: <span id="grade">-</span></div>
                    <div class="timestamp">Generated: <span id="reportTime">-</span></div>
                </div>
                
                <div class="card">
                    <h2>Issue Metrics</h2>
                    <div class="metrics-grid">
                        <div class="metric">
                            <div id="totalIssues" class="metric-value">-</div>
                            <div>Total Issues</div>
                        </div>
                        <div class="metric">
                            <div id="criticalIssues" class="metric-value critical">-</div>
                            <div>Critical</div>
                        </div>
                        <div class="metric">
                            <div id="highIssues" class="metric-value high">-</div>
                            <div>High</div>
                        </div>
                        <div class="metric">
                            <div id="manualReview" class="metric-value">-</div>
                            <div>Manual Review</div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="dashboard-grid">
                <div class="card">
                    <h2>Quality Trends</h2>
                    <div id="trendsChart" class="trends-chart">
                        Trends visualization would go here
                    </div>
                </div>
                
                <div class="card">
                    <h2>Files with Issues</h2>
                    <div id="fileList" class="file-list">
                        Loading file data...
                    </div>
                </div>
            </div>

            <div class="card">
                <h2>Actions</h2>
                <div class="actions">
                    <button class="btn btn-primary" onclick="refreshData()">Refresh Data</button>
                    <button class="btn btn-secondary" onclick="exportReport('json')">Export JSON</button>
                    <button class="btn btn-secondary" onclick="exportReport('csv')">Export CSV</button>
                    <button class="btn btn-secondary" onclick="exportReport('junit')">Export JUnit</button>
                </div>
            </div>
        </div>
    </div>

    <script>
        let refreshInterval;
        
        async function loadDashboardData() {
            try {
                const response = await fetch('/api/latest-report');
                if (!response.ok) {
                    throw new Error('Failed to load report data');
                }
                
                const data = await response.json();
                updateDashboard(data);
                
                document.getElementById('loading').style.display = 'none';
                document.getElementById('error').style.display = 'none';
                document.getElementById('dashboard').style.display = 'block';
                
            } catch (error) {
                document.getElementById('loading').style.display = 'none';
                document.getElementById('error').style.display = 'block';
                document.getElementById('error').textContent = 'Error loading dashboard: ' + error.message;
            }
        }
        
        function updateDashboard(data) {
            // Update score and grade
            document.getElementById('score').textContent = data.summary.overall_score.toFixed(1) + '%';
            document.getElementById('score').className = 'score grade-' + data.summary.quality_grade;
            document.getElementById('grade').textContent = data.summary.quality_grade;
            document.getElementById('reportTime').textContent = new Date(data.generated_at).toLocaleString();
            
            // Update metrics
            document.getElementById('totalIssues').textContent = data.summary.total_issues;
            document.getElementById('criticalIssues').textContent = data.summary.critical_issues;
            document.getElementById('highIssues').textContent = data.summary.high_issues;
            document.getElementById('manualReview').textContent = data.summary.manual_review;
            
            // Update file list
            const fileList = document.getElementById('fileList');
            fileList.innerHTML = '';
            
            data.file_reports.forEach(file => {
                if (file.summary.total_issues > 0) {
                    const fileItem = document.createElement('div');
                    fileItem.className = 'file-item';
                    fileItem.innerHTML = \`
                        <div>
                            <div>\${file.file_path}</div>
                            <div class="timestamp">\${file.summary.total_issues} issues</div>
                        </div>
                        <div class="file-issues">
                            <span class="critical">\${file.summary.critical_issues}</span> |
                            <span class="high">\${file.summary.high_issues}</span> |
                            <span class="medium">\${file.summary.medium_issues}</span> |
                            <span class="low">\${file.summary.low_issues}</span>
                        </div>
                    \`;
                    fileList.appendChild(fileItem);
                }
            });
            
            if (fileList.children.length === 0) {
                fileList.innerHTML = '<div class="success">No files with issues found!</div>';
            }
            
            // Update last update time
            document.getElementById('lastUpdate').textContent = new Date().toLocaleString();
        }
        
        function refreshData() {
            document.getElementById('loading').style.display = 'block';
            document.getElementById('dashboard').style.display = 'none';
            loadDashboardData();
        }
        
        function exportReport(format) {
            window.open('/api/export?format=' + format, '_blank');
        }
        
        // Initialize dashboard
        document.addEventListener('DOMContentLoaded', function() {
            loadDashboardData();
            
            // Set up auto-refresh
            refreshInterval = setInterval(loadDashboardData, {{.RefreshRate}} * 1000);
        });
        
        // Cleanup on page unload
        window.addEventListener('beforeunload', function() {
            if (refreshInterval) {
                clearInterval(refreshInterval);
            }
        });
    </script>
</body>
</html>`

	tmpl, err := template.New("dashboard").Parse(dashboardHTML)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, d.config); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}