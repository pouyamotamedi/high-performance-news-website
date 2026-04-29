package synthetic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// SyntheticMonitoringService provides the main service interface
type SyntheticMonitoringService struct {
	monitor             *SyntheticMonitor
	continuousValidator *ContinuousValidator
	config              *ServiceConfig
	httpServer          *http.Server
}

// ServiceConfig contains configuration for the synthetic monitoring service
type ServiceConfig struct {
	BaseURL        string        `json:"base_url"`
	Port           int           `json:"port"`
	EnableWebUI    bool          `json:"enable_web_ui"`
	EnableAPI      bool          `json:"enable_api"`
	MonitoringInterval time.Duration `json:"monitoring_interval"`
	WebhookURL     string        `json:"webhook_url"`
	EmailConfig    *EmailConfig  `json:"email_config"`
}

// NewSyntheticMonitoringService creates a new synthetic monitoring service
func NewSyntheticMonitoringService(config *ServiceConfig) (*SyntheticMonitoringService, error) {
	monitor, err := NewSyntheticMonitor(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create synthetic monitor: %w", err)
	}

	// Configure alerting
	if config.WebhookURL != "" {
		monitor.alertManager.ConfigureWebhook(config.WebhookURL)
	}
	if config.EmailConfig != nil {
		monitor.alertManager.ConfigureEmail(*config.EmailConfig)
	}

	continuousValidator := NewContinuousValidator(monitor)

	service := &SyntheticMonitoringService{
		monitor:             monitor,
		continuousValidator: continuousValidator,
		config:              config,
	}

	// Set up HTTP server if enabled
	if config.EnableAPI || config.EnableWebUI {
		service.setupHTTPServer()
	}

	return service, nil
}

// Start begins the synthetic monitoring service
func (s *SyntheticMonitoringService) Start(ctx context.Context) error {
	log.Println("Starting Synthetic Monitoring Service...")

	// Start the main monitoring
	go func() {
		if err := s.monitor.StartMonitoring(ctx); err != nil {
			log.Printf("Monitoring error: %v", err)
		}
	}()

	// Start continuous validation
	go func() {
		if err := s.continuousValidator.StartContinuousValidation(ctx); err != nil {
			log.Printf("Continuous validation error: %v", err)
		}
	}()

	// Start HTTP server if enabled
	if s.httpServer != nil {
		go func() {
			log.Printf("Starting HTTP server on port %d", s.config.Port)
			if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP server error: %v", err)
			}
		}()
	}

	log.Println("Synthetic Monitoring Service started successfully")
	return nil
}

// Stop gracefully shuts down the service
func (s *SyntheticMonitoringService) Stop(ctx context.Context) error {
	log.Println("Stopping Synthetic Monitoring Service...")

	// Stop HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Stop monitoring
	s.monitor.testScheduler.Stop()

	// Close browser
	if err := s.monitor.Close(); err != nil {
		log.Printf("Monitor cleanup error: %v", err)
	}

	log.Println("Synthetic Monitoring Service stopped")
	return nil
}

// setupHTTPServer configures the HTTP server for API and Web UI
func (s *SyntheticMonitoringService) setupHTTPServer() {
	mux := http.NewServeMux()

	if s.config.EnableAPI {
		s.setupAPIRoutes(mux)
	}

	if s.config.EnableWebUI {
		s.setupWebUIRoutes(mux)
	}

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: mux,
	}
}

// setupAPIRoutes configures API endpoints
func (s *SyntheticMonitoringService) setupAPIRoutes(mux *http.ServeMux) {
	// Get latest results
	mux.HandleFunc("/api/results", s.handleGetResults)
	
	// Get test summary
	mux.HandleFunc("/api/summary", s.handleGetSummary)
	
	// Get job status
	mux.HandleFunc("/api/jobs", s.handleGetJobs)
	
	// Get alerts
	mux.HandleFunc("/api/alerts", s.handleGetAlerts)
	
	// Health check
	mux.HandleFunc("/api/health", s.handleHealthCheck)
	
	// Trigger manual test
	mux.HandleFunc("/api/test", s.handleManualTest)
}

// setupWebUIRoutes configures Web UI endpoints
func (s *SyntheticMonitoringService) setupWebUIRoutes(mux *http.ServeMux) {
	// Dashboard
	mux.HandleFunc("/", s.handleDashboard)
	
	// Results page
	mux.HandleFunc("/results", s.handleResultsPage)
	
	// Jobs page
	mux.HandleFunc("/jobs", s.handleJobsPage)
	
	// Alerts page
	mux.HandleFunc("/alerts", s.handleAlertsPage)
}

// API Handlers

func (s *SyntheticMonitoringService) handleGetResults(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	results, err := s.monitor.resultStore.GetLatestResults(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *SyntheticMonitoringService) handleGetSummary(w http.ResponseWriter, r *http.Request) {
	testName := r.URL.Query().Get("test")
	if testName == "" {
		http.Error(w, "test parameter required", http.StatusBadRequest)
		return
	}

	duration := 24 * time.Hour
	if d := r.URL.Query().Get("duration"); d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			duration = parsed
		}
	}

	summary, err := s.monitor.resultStore.GetTestSummary(testName, duration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (s *SyntheticMonitoringService) handleGetJobs(w http.ResponseWriter, r *http.Request) {
	status := s.monitor.testScheduler.GetJobStatus()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *SyntheticMonitoringService) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if alertManager, ok := s.monitor.alertManager.(*SimpleAlertManager); ok {
		alerts := alertManager.GetAlertHistory(limit)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(alerts)
	} else {
		http.Error(w, "Alert history not available", http.StatusInternalServerError)
	}
}

func (s *SyntheticMonitoringService) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"uptime":    time.Since(time.Now()), // This would be actual uptime in production
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (s *SyntheticMonitoringService) handleManualTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	testName := r.URL.Query().Get("test")
	if testName == "" {
		http.Error(w, "test parameter required", http.StatusBadRequest)
		return
	}

	// Trigger manual test based on test name
	ctx := context.Background()
	switch testName {
	case "critical_user_journeys":
		go s.monitor.runCriticalUserJourneys(ctx)
	case "article_publishing":
		go s.monitor.runArticlePublishingWorkflow(ctx)
	case "search_functionality":
		go s.monitor.runSearchFunctionalityTests(ctx)
	case "admin_panel":
		go s.monitor.runAdminPanelWorkflow(ctx)
	default:
		http.Error(w, "Unknown test name", http.StatusBadRequest)
		return
	}

	response := map[string]string{
		"status":  "triggered",
		"test":    testName,
		"message": "Test execution started",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Web UI Handlers (simplified HTML responses)

func (s *SyntheticMonitoringService) handleDashboard(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Synthetic Monitoring Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .card { border: 1px solid #ddd; padding: 20px; margin: 10px 0; border-radius: 5px; }
        .status-passed { color: green; }
        .status-failed { color: red; }
        .status-error { color: orange; }
    </style>
</head>
<body>
    <h1>Synthetic Monitoring Dashboard</h1>
    <div class="card">
        <h2>Quick Actions</h2>
        <button onclick="triggerTest('critical_user_journeys')">Run User Journey Tests</button>
        <button onclick="triggerTest('search_functionality')">Run Search Tests</button>
        <button onclick="triggerTest('admin_panel')">Run Admin Tests</button>
    </div>
    <div class="card">
        <h2>Recent Results</h2>
        <div id="results">Loading...</div>
    </div>
    <script>
        function triggerTest(testName) {
            fetch('/api/test?test=' + testName, {method: 'POST'})
                .then(response => response.json())
                .then(data => alert('Test triggered: ' + data.message));
        }
        
        function loadResults() {
            fetch('/api/results?limit=10')
                .then(response => response.json())
                .then(data => {
                    const resultsDiv = document.getElementById('results');
                    resultsDiv.innerHTML = data.map(result => 
                        '<div class="status-' + result.status + '">' +
                        result.test_name + ' - ' + result.status + ' (' + result.duration + ')' +
                        '</div>'
                    ).join('');
                });
        }
        
        loadResults();
        setInterval(loadResults, 30000); // Refresh every 30 seconds
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *SyntheticMonitoringService) handleResultsPage(w http.ResponseWriter, r *http.Request) {
	// Simplified results page
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h1>Test Results</h1><p>Detailed results would be shown here.</p>"))
}

func (s *SyntheticMonitoringService) handleJobsPage(w http.ResponseWriter, r *http.Request) {
	// Simplified jobs page
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h1>Scheduled Jobs</h1><p>Job status would be shown here.</p>"))
}

func (s *SyntheticMonitoringService) handleAlertsPage(w http.ResponseWriter, r *http.Request) {
	// Simplified alerts page
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h1>Alerts</h1><p>Alert history would be shown here.</p>"))
}

// GetMetrics returns current monitoring metrics
func (s *SyntheticMonitoringService) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})
	
	// Get job status
	jobStatus := s.monitor.testScheduler.GetJobStatus()
	metrics["scheduled_jobs"] = len(jobStatus)
	
	// Get result count
	if memStore, ok := s.monitor.resultStore.(*MemoryResultStore); ok {
		metrics["total_results"] = memStore.GetResultsCount()
	}
	
	// Get alert stats
	if alertManager, ok := s.monitor.alertManager.(*SimpleAlertManager); ok {
		alertStats := alertManager.GetAlertStats(24 * time.Hour)
		metrics["alerts_24h"] = alertStats.Total
	}
	
	return metrics
}