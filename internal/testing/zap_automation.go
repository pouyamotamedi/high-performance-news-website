package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ZAPAutomation handles OWASP ZAP dynamic security testing automation
type ZAPAutomation struct {
	apiURL       string
	apiKey       string
	httpClient   *http.Client
	projectRoot  string
	reportDir    string
	config       *ZAPConfig
}

// ZAPConfig holds ZAP automation configuration
type ZAPConfig struct {
	TargetURL           string        `yaml:"target_url"`
	SpiderTimeout       time.Duration `yaml:"spider_timeout"`
	ActiveScanTimeout   time.Duration `yaml:"active_scan_timeout"`
	MaxSpiderDepth      int           `yaml:"max_spider_depth"`
	MaxActiveScanRules  int           `yaml:"max_active_scan_rules"`
	ExcludeURLPatterns  []string      `yaml:"exclude_url_patterns"`
	IncludeURLPatterns  []string      `yaml:"include_url_patterns"`
	AuthenticationMode  string        `yaml:"authentication_mode"`
	SessionManagement   bool          `yaml:"session_management"`
	CustomPolicies      []string      `yaml:"custom_policies"`
	ReportFormats       []string      `yaml:"report_formats"`
}

// ZAPScanResult represents comprehensive ZAP scan results
type ZAPScanResult struct {
	SessionID        string                 `json:"session_id"`
	TargetURL        string                 `json:"target_url"`
	StartTime        time.Time              `json:"start_time"`
	EndTime          time.Time              `json:"end_time"`
	Duration         time.Duration          `json:"duration"`
	Status           string                 `json:"status"`
	SpiderResults    *ZAPSpiderResult       `json:"spider_results"`
	ActiveScanResults *ZAPActiveScanResult  `json:"active_scan_results"`
	Alerts           []ZAPAlert             `json:"alerts"`
	Summary          ZAPScanSummary         `json:"summary"`
	ReportPaths      []string               `json:"report_paths"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ZAPSpiderResult contains spider scan results
type ZAPSpiderResult struct {
	SpiderID      string        `json:"spider_id"`
	URLsFound     int           `json:"urls_found"`
	Duration      time.Duration `json:"duration"`
	Status        string        `json:"status"`
	Progress      int           `json:"progress"`
	ErrorMessages []string      `json:"error_messages"`
}

// ZAPActiveScanResult contains active scan results
type ZAPActiveScanResult struct {
	ScanID        string        `json:"scan_id"`
	Duration      time.Duration `json:"duration"`
	Status        string        `json:"status"`
	Progress      int           `json:"progress"`
	AlertsRaised  int           `json:"alerts_raised"`
	ErrorMessages []string      `json:"error_messages"`
}

// ZAPAlert represents a security alert from ZAP
type ZAPAlert struct {
	ID          string   `json:"id"`
	Alert       string   `json:"alert"`
	Risk        string   `json:"risk"`
	Confidence  string   `json:"confidence"`
	Description string   `json:"description"`
	Solution    string   `json:"solution"`
	Reference   string   `json:"reference"`
	CWEId       string   `json:"cwe_id"`
	WASCId      string   `json:"wasc_id"`
	URL         string   `json:"url"`
	Method      string   `json:"method"`
	Evidence    string   `json:"evidence"`
	Instances   []string `json:"instances"`
}

// ZAPScanSummary provides scan overview
type ZAPScanSummary struct {
	TotalAlerts     int `json:"total_alerts"`
	HighRiskAlerts  int `json:"high_risk_alerts"`
	MediumRiskAlerts int `json:"medium_risk_alerts"`
	LowRiskAlerts   int `json:"low_risk_alerts"`
	InfoAlerts      int `json:"info_alerts"`
	URLsScanned     int `json:"urls_scanned"`
	ScanCoverage    float64 `json:"scan_coverage"`
}

// NewZAPAutomation creates a new ZAP automation instance
func NewZAPAutomation(projectRoot string) *ZAPAutomation {
	config := &ZAPConfig{
		TargetURL:          getEnvOrDefault("ZAP_TARGET_URL", "http://localhost:8080"),
		SpiderTimeout:      5 * time.Minute,
		ActiveScanTimeout:  15 * time.Minute,
		MaxSpiderDepth:     5,
		MaxActiveScanRules: 100,
		ExcludeURLPatterns: []string{"/logout", "/admin/delete"},
		IncludeURLPatterns: []string{".*"},
		AuthenticationMode: "none",
		SessionManagement:  false,
		ReportFormats:      []string{"json", "html", "xml"},
	}

	return &ZAPAutomation{
		apiURL:      getEnvOrDefault("ZAP_API_URL", "http://localhost:8080"),
		apiKey:      os.Getenv("ZAP_API_KEY"),
		projectRoot: projectRoot,
		reportDir:   filepath.Join(projectRoot, "reports", "security", "zap"),
		config:      config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RunComprehensiveZAPScan performs a full ZAP security scan
func (z *ZAPAutomation) RunComprehensiveZAPScan(ctx context.Context) (*ZAPScanResult, error) {
	if z.apiKey == "" {
		return nil, fmt.Errorf("ZAP_API_KEY not set")
	}

	// Ensure report directory exists
	if err := os.MkdirAll(z.reportDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create ZAP report directory: %w", err)
	}

	result := &ZAPScanResult{
		TargetURL: z.config.TargetURL,
		StartTime: time.Now(),
		Status:    "running",
		Metadata:  make(map[string]interface{}),
	}

	log.Printf("Starting comprehensive ZAP scan for: %s", z.config.TargetURL)

	// Check if ZAP is running
	if !z.isZAPRunning() {
		return nil, fmt.Errorf("OWASP ZAP is not running at %s", z.apiURL)
	}

	// Create new session
	sessionID, err := z.createNewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create ZAP session: %w", err)
	}
	result.SessionID = sessionID

	// Configure ZAP settings
	if err := z.configureZAPSettings(); err != nil {
		log.Printf("Warning: failed to configure ZAP settings: %v", err)
	}

	// Run spider scan
	spiderResult, err := z.runSpiderScan(ctx)
	if err != nil {
		log.Printf("Spider scan failed: %v", err)
		result.Metadata["spider_error"] = err.Error()
	} else {
		result.SpiderResults = spiderResult
	}

	// Run active scan
	activeScanResult, err := z.runActiveScan(ctx)
	if err != nil {
		log.Printf("Active scan failed: %v", err)
		result.Metadata["active_scan_error"] = err.Error()
	} else {
		result.ActiveScanResults = activeScanResult
	}

	// Get alerts
	alerts, err := z.getAlerts()
	if err != nil {
		log.Printf("Failed to get alerts: %v", err)
		result.Metadata["alerts_error"] = err.Error()
	} else {
		result.Alerts = alerts
	}

	// Calculate summary
	result.Summary = z.calculateScanSummary(result)
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Status = "completed"

	// Generate reports
	reportPaths, err := z.generateReports(result)
	if err != nil {
		log.Printf("Failed to generate some reports: %v", err)
	}
	result.ReportPaths = reportPaths

	log.Printf("ZAP scan completed in %v with %d alerts", result.Duration, result.Summary.TotalAlerts)

	return result, nil
}

// isZAPRunning checks if ZAP is accessible
func (z *ZAPAutomation) isZAPRunning() bool {
	url := fmt.Sprintf("%s/JSON/core/view/version/?apikey=%s", z.apiURL, z.apiKey)
	resp, err := z.httpClient.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// createNewSession creates a new ZAP session
func (z *ZAPAutomation) createNewSession() (string, error) {
	sessionName := fmt.Sprintf("security-scan-%d", time.Now().Unix())
	url := fmt.Sprintf("%s/JSON/core/action/newSession/?apikey=%s&name=%s", z.apiURL, z.apiKey, sessionName)
	
	resp, err := z.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create session, status: %d", resp.StatusCode)
	}

	return sessionName, nil
}

// configureZAPSettings configures ZAP for optimal scanning
func (z *ZAPAutomation) configureZAPSettings() error {
	// Set spider max depth
	url := fmt.Sprintf("%s/JSON/spider/action/setOptionMaxDepth/?apikey=%s&Integer=%d", 
		z.apiURL, z.apiKey, z.config.MaxSpiderDepth)
	if _, err := z.httpClient.Get(url); err != nil {
		return fmt.Errorf("failed to set spider max depth: %w", err)
	}

	// Configure excluded URLs
	for _, pattern := range z.config.ExcludeURLPatterns {
		url := fmt.Sprintf("%s/JSON/core/action/excludeFromProxy/?apikey=%s&regex=%s", 
			z.apiURL, z.apiKey, pattern)
		if _, err := z.httpClient.Get(url); err != nil {
			log.Printf("Warning: failed to exclude URL pattern %s: %v", pattern, err)
		}
	}

	return nil
}

// runSpiderScan performs spider scanning
func (z *ZAPAutomation) runSpiderScan(ctx context.Context) (*ZAPSpiderResult, error) {
	log.Println("Starting ZAP spider scan...")
	
	// Start spider
	url := fmt.Sprintf("%s/JSON/spider/action/scan/?apikey=%s&url=%s", 
		z.apiURL, z.apiKey, z.config.TargetURL)
	
	resp, err := z.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var spiderResponse struct {
		Scan string `json:"scan"`
	}
	if err := json.Unmarshal(body, &spiderResponse); err != nil {
		return nil, err
	}

	spiderID := spiderResponse.Scan
	result := &ZAPSpiderResult{
		SpiderID: spiderID,
		Status:   "running",
	}

	startTime := time.Now()
	timeout := time.After(z.config.SpiderTimeout)

	// Monitor spider progress
	for {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-timeout:
			result.Status = "timeout"
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("spider scan timed out after %v", z.config.SpiderTimeout)
		default:
			progress, err := z.getSpiderProgress(spiderID)
			if err != nil {
				log.Printf("Error getting spider progress: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			result.Progress = progress
			if progress >= 100 {
				result.Status = "completed"
				result.Duration = time.Since(startTime)
				
				// Get URLs found
				urlsFound, err := z.getSpiderURLsFound(spiderID)
				if err != nil {
					log.Printf("Error getting spider URLs: %v", err)
				} else {
					result.URLsFound = urlsFound
				}
				
				log.Printf("Spider scan completed: %d URLs found", result.URLsFound)
				return result, nil
			}

			log.Printf("Spider progress: %d%%", progress)
			time.Sleep(5 * time.Second)
		}
	}
}

// runActiveScan performs active security scanning
func (z *ZAPAutomation) runActiveScan(ctx context.Context) (*ZAPActiveScanResult, error) {
	log.Println("Starting ZAP active scan...")
	
	// Start active scan
	url := fmt.Sprintf("%s/JSON/ascan/action/scan/?apikey=%s&url=%s", 
		z.apiURL, z.apiKey, z.config.TargetURL)
	
	resp, err := z.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var scanResponse struct {
		Scan string `json:"scan"`
	}
	if err := json.Unmarshal(body, &scanResponse); err != nil {
		return nil, err
	}

	scanID := scanResponse.Scan
	result := &ZAPActiveScanResult{
		ScanID: scanID,
		Status: "running",
	}

	startTime := time.Now()
	timeout := time.After(z.config.ActiveScanTimeout)

	// Monitor active scan progress
	for {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-timeout:
			result.Status = "timeout"
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("active scan timed out after %v", z.config.ActiveScanTimeout)
		default:
			progress, err := z.getActiveScanProgress(scanID)
			if err != nil {
				log.Printf("Error getting active scan progress: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			result.Progress = progress
			if progress >= 100 {
				result.Status = "completed"
				result.Duration = time.Since(startTime)
				log.Printf("Active scan completed")
				return result, nil
			}

			log.Printf("Active scan progress: %d%%", progress)
			time.Sleep(10 * time.Second)
		}
	}
}

// getSpiderProgress gets spider scan progress
func (z *ZAPAutomation) getSpiderProgress(spiderID string) (int, error) {
	url := fmt.Sprintf("%s/JSON/spider/view/status/?apikey=%s&scanId=%s", 
		z.apiURL, z.apiKey, spiderID)
	
	resp, err := z.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var progressResponse struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &progressResponse); err != nil {
		return 0, err
	}

	var progress int
	fmt.Sscanf(progressResponse.Status, "%d", &progress)
	return progress, nil
}

// getActiveScanProgress gets active scan progress
func (z *ZAPAutomation) getActiveScanProgress(scanID string) (int, error) {
	url := fmt.Sprintf("%s/JSON/ascan/view/status/?apikey=%s&scanId=%s", 
		z.apiURL, z.apiKey, scanID)
	
	resp, err := z.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var progressResponse struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &progressResponse); err != nil {
		return 0, err
	}

	var progress int
	fmt.Sscanf(progressResponse.Status, "%d", &progress)
	return progress, nil
}

// getSpiderURLsFound gets number of URLs found by spider
func (z *ZAPAutomation) getSpiderURLsFound(spiderID string) (int, error) {
	url := fmt.Sprintf("%s/JSON/spider/view/results/?apikey=%s&scanId=%s", 
		z.apiURL, z.apiKey, spiderID)
	
	resp, err := z.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var resultsResponse struct {
		Results []string `json:"results"`
	}
	if err := json.Unmarshal(body, &resultsResponse); err != nil {
		return 0, err
	}

	return len(resultsResponse.Results), nil
}

// getAlerts retrieves all security alerts
func (z *ZAPAutomation) getAlerts() ([]ZAPAlert, error) {
	url := fmt.Sprintf("%s/JSON/core/view/alerts/?apikey=%s", z.apiURL, z.apiKey)
	
	resp, err := z.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var alertsResponse struct {
		Alerts []struct {
			Alert       string `json:"alert"`
			Risk        string `json:"risk"`
			Confidence  string `json:"confidence"`
			Description string `json:"description"`
			Solution    string `json:"solution"`
			Reference   string `json:"reference"`
			CWEId       string `json:"cweid"`
			WASCId      string `json:"wascid"`
			URL         string `json:"url"`
			Method      string `json:"method"`
			Evidence    string `json:"evidence"`
			Instances   []struct {
				URI    string `json:"uri"`
				Method string `json:"method"`
			} `json:"instances"`
		} `json:"alerts"`
	}

	if err := json.Unmarshal(body, &alertsResponse); err != nil {
		return nil, err
	}

	var alerts []ZAPAlert
	for i, alert := range alertsResponse.Alerts {
		var instances []string
		for _, instance := range alert.Instances {
			instances = append(instances, fmt.Sprintf("%s %s", instance.Method, instance.URI))
		}

		zapAlert := ZAPAlert{
			ID:          fmt.Sprintf("ZAP-%d", i+1),
			Alert:       alert.Alert,
			Risk:        strings.ToLower(alert.Risk),
			Confidence:  strings.ToLower(alert.Confidence),
			Description: alert.Description,
			Solution:    alert.Solution,
			Reference:   alert.Reference,
			CWEId:       alert.CWEId,
			WASCId:      alert.WASCId,
			URL:         alert.URL,
			Method:      alert.Method,
			Evidence:    alert.Evidence,
			Instances:   instances,
		}
		alerts = append(alerts, zapAlert)
	}

	return alerts, nil
}

// calculateScanSummary calculates scan summary statistics
func (z *ZAPAutomation) calculateScanSummary(result *ZAPScanResult) ZAPScanSummary {
	summary := ZAPScanSummary{}
	
	for _, alert := range result.Alerts {
		summary.TotalAlerts++
		switch alert.Risk {
		case "high":
			summary.HighRiskAlerts++
		case "medium":
			summary.MediumRiskAlerts++
		case "low":
			summary.LowRiskAlerts++
		case "informational":
			summary.InfoAlerts++
		}
	}

	if result.SpiderResults != nil {
		summary.URLsScanned = result.SpiderResults.URLsFound
	}

	// Calculate scan coverage (simplified)
	if summary.URLsScanned > 0 {
		summary.ScanCoverage = float64(summary.TotalAlerts) / float64(summary.URLsScanned) * 100
		if summary.ScanCoverage > 100 {
			summary.ScanCoverage = 100
		}
	}

	return summary
}

// generateReports generates ZAP scan reports in multiple formats
func (z *ZAPAutomation) generateReports(result *ZAPScanResult) ([]string, error) {
	timestamp := result.StartTime.Format("20060102-150405")
	var reportPaths []string

	// Generate JSON report
	jsonPath := filepath.Join(z.reportDir, fmt.Sprintf("zap-scan-%s.json", timestamp))
	if err := z.generateJSONReport(result, jsonPath); err != nil {
		log.Printf("Failed to generate JSON report: %v", err)
	} else {
		reportPaths = append(reportPaths, jsonPath)
	}

	// Generate HTML report from ZAP
	htmlPath := filepath.Join(z.reportDir, fmt.Sprintf("zap-scan-%s.html", timestamp))
	if err := z.generateHTMLReportFromZAP(htmlPath); err != nil {
		log.Printf("Failed to generate HTML report from ZAP: %v", err)
	} else {
		reportPaths = append(reportPaths, htmlPath)
	}

	// Generate XML report from ZAP
	xmlPath := filepath.Join(z.reportDir, fmt.Sprintf("zap-scan-%s.xml", timestamp))
	if err := z.generateXMLReportFromZAP(xmlPath); err != nil {
		log.Printf("Failed to generate XML report from ZAP: %v", err)
	} else {
		reportPaths = append(reportPaths, xmlPath)
	}

	return reportPaths, nil
}

// generateJSONReport generates a JSON report
func (z *ZAPAutomation) generateJSONReport(result *ZAPScanResult, reportPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(reportPath, data, 0644)
}

// generateHTMLReportFromZAP generates HTML report using ZAP API
func (z *ZAPAutomation) generateHTMLReportFromZAP(reportPath string) error {
	url := fmt.Sprintf("%s/OTHER/core/other/htmlreport/?apikey=%s", z.apiURL, z.apiKey)
	
	resp, err := z.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return os.WriteFile(reportPath, body, 0644)
}

// generateXMLReportFromZAP generates XML report using ZAP API
func (z *ZAPAutomation) generateXMLReportFromZAP(reportPath string) error {
	url := fmt.Sprintf("%s/OTHER/core/other/xmlreport/?apikey=%s", z.apiURL, z.apiKey)
	
	resp, err := z.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return os.WriteFile(reportPath, body, 0644)
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}