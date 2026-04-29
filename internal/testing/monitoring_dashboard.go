package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// MonitoringDashboard provides real-time monitoring dashboard
type MonitoringDashboard struct {
	executionMonitor *TestExecutionMonitor
	resourceTracker  *ResourceTracker
	metricsCollector *MetricsCollector
	alertManager     *TestAlertManager
	clients          map[string]*DashboardClient
	mu               sync.RWMutex
	upgrader         websocket.Upgrader
	isRunning        bool
	stopChan         chan struct{}
}

// DashboardClient represents a connected dashboard client
type DashboardClient struct {
	ID         string          `json:"id"`
	Connection *websocket.Conn `json:"-"`
	LastSeen   time.Time       `json:"last_seen"`
	UserAgent  string          `json:"user_agent"`
	RemoteAddr string          `json:"remote_addr"`
}

// DashboardData contains all dashboard information
type DashboardData struct {
	Overview        OverviewData        `json:"overview"`
	TestExecution   TestExecutionData   `json:"test_execution"`
	ResourceMetrics ResourceMetricsData `json:"resource_metrics"`
	Alerts          AlertsData          `json:"alerts"`
	Performance     PerformanceData     `json:"performance"`
	Trends          TrendsData          `json:"trends"`
	SystemHealth    SystemHealthData    `json:"system_health"`
	Timestamp       time.Time           `json:"timestamp"`
}

// OverviewData contains high-level overview information
type OverviewData struct {
	ActiveTests       int     `json:"active_tests"`
	CompletedToday    int     `json:"completed_today"`
	FailedToday       int     `json:"failed_today"`
	SuccessRate       float64 `json:"success_rate"`
	ActiveAlerts      int     `json:"active_alerts"`
	CriticalAlerts    int     `json:"critical_alerts"`
	SystemHealth      string  `json:"system_health"`
	OverallScore      float64 `json:"overall_score"`
}

// TestExecutionData contains test execution information
type TestExecutionData struct {
	ActiveExecutions  []TestExecutionInfo `json:"active_executions"`
	RecentCompletions []TestExecutionInfo `json:"recent_completions"`
	QueuedTests       []TestExecutionInfo `json:"queued_tests"`
	ExecutionStats    ExecutionStats      `json:"execution_stats"`
}

// TestExecutionInfo contains information about a test execution
type TestExecutionInfo struct {
	ID           string        `json:"id"`
	TestSuite    string        `json:"test_suite"`
	TestName     string        `json:"test_name"`
	Status       string        `json:"status"`
	Progress     float64       `json:"progress"`
	Duration     time.Duration `json:"duration"`
	Environment  string        `json:"environment"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      *time.Time    `json:"end_time,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// ExecutionStats contains execution statistics
type ExecutionStats struct {
	TotalExecutions   int64         `json:"total_executions"`
	AverageRuntime    time.Duration `json:"average_runtime"`
	MedianRuntime     time.Duration `json:"median_runtime"`
	TestsPerHour      float64       `json:"tests_per_hour"`
	ParallelismFactor float64       `json:"parallelism_factor"`
}

// ResourceMetricsData contains resource utilization information
type ResourceMetricsData struct {
	SystemMetrics   SystemResourceMetrics `json:"system_metrics"`
	ProcessMetrics  ProcessResourceMetrics `json:"process_metrics"`
	DatabaseMetrics DatabaseResourceMetrics `json:"database_metrics"`
	CacheMetrics    CacheResourceMetrics   `json:"cache_metrics"`
	ResourceAlerts  []ResourceAlertInfo    `json:"resource_alerts"`
}

// SystemResourceMetrics contains system-level resource metrics
type SystemResourceMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	LoadAverage   float64 `json:"load_average"`
	NetworkIOKB   float64 `json:"network_io_kb"`
	DiskIOKB      float64 `json:"disk_io_kb"`
}

// ProcessResourceMetrics contains process-level resource metrics
type ProcessResourceMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      float64 `json:"memory_mb"`
	MemoryPercent float64 `json:"memory_percent"`
	NumThreads    int32   `json:"num_threads"`
	NumFDs        int32   `json:"num_fds"`
}

// DatabaseResourceMetrics contains database resource metrics
type DatabaseResourceMetrics struct {
	ActiveConnections int     `json:"active_connections"`
	MaxConnections    int     `json:"max_connections"`
	ConnectionPercent float64 `json:"connection_percent"`
	SlowQueries       int64   `json:"slow_queries"`
	QueryRate         float64 `json:"query_rate"`
	CacheHitRatio     float64 `json:"cache_hit_ratio"`
}

// CacheResourceMetrics contains cache resource metrics
type CacheResourceMetrics struct {
	HitRate         float64 `json:"hit_rate"`
	MissRate        float64 `json:"miss_rate"`
	KeyCount        int64   `json:"key_count"`
	MemoryUsedMB    float64 `json:"memory_used_mb"`
	ConnectionCount int     `json:"connection_count"`
}

// ResourceAlertInfo contains resource alert information
type ResourceAlertInfo struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Timestamp time.Time `json:"timestamp"`
}

// AlertsData contains alert information
type AlertsData struct {
	ActiveAlerts   []AlertInfo   `json:"active_alerts"`
	RecentAlerts   []AlertInfo   `json:"recent_alerts"`
	AlertSummary   AlertSummary  `json:"alert_summary"`
	AlertTrends    []AlertTrend  `json:"alert_trends"`
}

// AlertInfo contains alert information
type AlertInfo struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Severity  string    `json:"severity"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"`
}

// AlertTrend contains alert trend information
type AlertTrend struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

// PerformanceData contains performance information
type PerformanceData struct {
	ResponseTimes    ResponseTimeData    `json:"response_times"`
	ThroughputData   ThroughputData      `json:"throughput_data"`
	ErrorRates       ErrorRateData       `json:"error_rates"`
	PerformanceTrend PerformanceTrend    `json:"performance_trend"`
}

// ResponseTimeData contains response time metrics
type ResponseTimeData struct {
	Average time.Duration `json:"average"`
	Median  time.Duration `json:"median"`
	P95     time.Duration `json:"p95"`
	P99     time.Duration `json:"p99"`
	History []TimePoint   `json:"history"`
}

// ThroughputData contains throughput metrics
type ThroughputData struct {
	Current     float64     `json:"current"`
	Peak        float64     `json:"peak"`
	Average     float64     `json:"average"`
	History     []TimePoint `json:"history"`
}

// ErrorRateData contains error rate metrics
type ErrorRateData struct {
	Current float64     `json:"current"`
	Average float64     `json:"average"`
	History []TimePoint `json:"history"`
}

// PerformanceTrend contains performance trend information
type PerformanceTrend struct {
	Direction string  `json:"direction"` // "improving", "stable", "degrading"
	Change    float64 `json:"change"`    // Percentage change
	Period    string  `json:"period"`    // Time period for the trend
}

// TimePoint represents a point in time with a value
type TimePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// TrendsData contains trend analysis
type TrendsData struct {
	ExecutionTrend    TrendInfo `json:"execution_trend"`
	PerformanceTrend  TrendInfo `json:"performance_trend"`
	QualityTrend      TrendInfo `json:"quality_trend"`
	ReliabilityTrend  TrendInfo `json:"reliability_trend"`
	ResourceTrend     TrendInfo `json:"resource_trend"`
}

// TrendInfo contains trend information
type TrendInfo struct {
	Direction   string    `json:"direction"`
	Change      float64   `json:"change"`
	Period      string    `json:"period"`
	Confidence  float64   `json:"confidence"`
	Prediction  float64   `json:"prediction"`
	LastUpdated time.Time `json:"last_updated"`
}

// SystemHealthData contains system health information
type SystemHealthData struct {
	OverallStatus   string              `json:"overall_status"`
	ComponentHealth map[string]string   `json:"component_health"`
	HealthScore     float64             `json:"health_score"`
	Uptime          time.Duration       `json:"uptime"`
	LastIncident    *time.Time          `json:"last_incident,omitempty"`
	HealthHistory   []HealthDataPoint   `json:"health_history"`
}

// HealthDataPoint represents a health measurement point
type HealthDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
	Status    string    `json:"status"`
}

// NewMonitoringDashboard creates a new monitoring dashboard
func NewMonitoringDashboard(executionMonitor *TestExecutionMonitor, resourceTracker *ResourceTracker, metricsCollector *MetricsCollector, alertManager *TestAlertManager) *MonitoringDashboard {
	return &MonitoringDashboard{
		executionMonitor: executionMonitor,
		resourceTracker:  resourceTracker,
		metricsCollector: metricsCollector,
		alertManager:     alertManager,
		clients:          make(map[string]*DashboardClient),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		stopChan: make(chan struct{}),
	}
}

// Start begins the monitoring dashboard
func (d *MonitoringDashboard) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isRunning {
		return fmt.Errorf("monitoring dashboard is already running")
	}

	d.isRunning = true

	// Start dashboard update goroutines
	go d.broadcastUpdates(ctx)
	go d.cleanupClients(ctx)

	log.Printf("Monitoring dashboard started")
	return nil
}

// Stop stops the monitoring dashboard
func (d *MonitoringDashboard) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isRunning {
		return
	}

	close(d.stopChan)

	// Close all client connections
	for _, client := range d.clients {
		client.Connection.Close()
	}
	d.clients = make(map[string]*DashboardClient)

	d.isRunning = false

	log.Printf("Monitoring dashboard stopped")
}

// RegisterRoutes registers dashboard HTTP routes
func (d *MonitoringDashboard) RegisterRoutes(router *gin.Engine) {
	dashboard := router.Group("/dashboard")
	{
		dashboard.GET("/", d.serveDashboard)
		dashboard.GET("/ws", d.handleWebSocket)
		dashboard.GET("/api/data", d.getDashboardData)
		dashboard.GET("/api/executions", d.getExecutions)
		dashboard.GET("/api/metrics", d.getMetrics)
		dashboard.GET("/api/alerts", d.getAlerts)
		dashboard.GET("/api/resources", d.getResources)
		dashboard.POST("/api/alerts/:id/acknowledge", d.acknowledgeAlert)
		dashboard.POST("/api/alerts/:id/resolve", d.resolveAlert)
	}

	// Serve static dashboard files
	router.Static("/dashboard/static", "./web/dashboard")
}

// serveDashboard serves the main dashboard page
func (d *MonitoringDashboard) serveDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Test Execution Monitoring Dashboard",
	})
}

// handleWebSocket handles WebSocket connections for real-time updates
func (d *MonitoringDashboard) handleWebSocket(c *gin.Context) {
	conn, err := d.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	client := &DashboardClient{
		ID:         generateClientID(),
		Connection: conn,
		LastSeen:   time.Now(),
		UserAgent:  c.Request.UserAgent(),
		RemoteAddr: c.ClientIP(),
	}

	d.mu.Lock()
	d.clients[client.ID] = client
	d.mu.Unlock()

	log.Printf("Dashboard client connected: %s from %s", client.ID, client.RemoteAddr)

	// Send initial data
	d.sendDashboardData(client)

	// Handle client messages
	go d.handleClientMessages(client)
}

// getDashboardData returns complete dashboard data
func (d *MonitoringDashboard) getDashboardData(c *gin.Context) {
	data := d.collectDashboardData()
	c.JSON(http.StatusOK, data)
}

// getExecutions returns test execution data
func (d *MonitoringDashboard) getExecutions(c *gin.Context) {
	activeExecutions := d.executionMonitor.GetActiveExecutions()
	executionHistory := d.executionMonitor.GetExecutionHistory(50)

	c.JSON(http.StatusOK, gin.H{
		"active_executions": activeExecutions,
		"execution_history": executionHistory,
	})
}

// getMetrics returns metrics data
func (d *MonitoringDashboard) getMetrics(c *gin.Context) {
	aggregatedMetrics := d.metricsCollector.GetAggregatedMetrics()
	allMetrics := d.metricsCollector.GetAllMetrics()

	c.JSON(http.StatusOK, gin.H{
		"aggregated_metrics": aggregatedMetrics,
		"all_metrics":        allMetrics,
	})
}

// getAlerts returns alert data
func (d *MonitoringDashboard) getAlerts(c *gin.Context) {
	activeAlerts := d.alertManager.GetActiveAlerts()
	alertHistory := d.alertManager.GetAlertHistory(100)
	alertSummary := d.alertManager.GetAlertSummary()

	c.JSON(http.StatusOK, gin.H{
		"active_alerts": activeAlerts,
		"alert_history": alertHistory,
		"alert_summary": alertSummary,
	})
}

// getResources returns resource data
func (d *MonitoringDashboard) getResources(c *gin.Context) {
	currentUsage := d.resourceTracker.GetCurrentUsage()
	detailedMetrics, _ := d.resourceTracker.GetDetailedMetrics()
	usageHistory := d.resourceTracker.GetUsageHistory(100)
	activeAlerts := d.resourceTracker.GetActiveAlerts()

	c.JSON(http.StatusOK, gin.H{
		"current_usage":    currentUsage,
		"detailed_metrics": detailedMetrics,
		"usage_history":    usageHistory,
		"active_alerts":    activeAlerts,
	})
}

// acknowledgeAlert acknowledges an alert
func (d *MonitoringDashboard) acknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")
	acknowledgedBy := c.DefaultQuery("acknowledged_by", "dashboard_user")

	err := d.alertManager.AcknowledgeAlert(alertID, acknowledgedBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged"})
}

// resolveAlert resolves an alert
func (d *MonitoringDashboard) resolveAlert(c *gin.Context) {
	alertID := c.Param("id")

	err := d.alertManager.ResolveAlert(alertID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert resolved"})
}

// collectDashboardData collects all dashboard data
func (d *MonitoringDashboard) collectDashboardData() *DashboardData {
	// Collect data from all components
	dashboardData := d.executionMonitor.GetDashboardData()
	aggregatedMetrics := d.metricsCollector.GetAggregatedMetrics()
	alertSummary := d.alertManager.GetAlertSummary()
	resourceUsage := d.resourceTracker.GetCurrentUsage()
	detailedMetrics, _ := d.resourceTracker.GetDetailedMetrics()

	// Build overview data
	overview := OverviewData{
		ActiveTests:    dashboardData.ActiveTests,
		CompletedToday: dashboardData.CompletedTests,
		FailedToday:    dashboardData.FailedTests,
		SuccessRate:    float64(dashboardData.CompletedTests) / float64(dashboardData.CompletedTests+dashboardData.FailedTests),
		ActiveAlerts:   alertSummary.ActiveAlerts,
		CriticalAlerts: alertSummary.CriticalAlerts,
		SystemHealth:   d.calculateSystemHealth(),
		OverallScore:   d.calculateOverallScore(),
	}

	// Build test execution data
	activeExecutions := d.executionMonitor.GetActiveExecutions()
	executionHistory := d.executionMonitor.GetExecutionHistory(10)

	testExecutionData := TestExecutionData{
		ActiveExecutions:  d.convertToExecutionInfo(activeExecutions),
		RecentCompletions: d.convertHistoryToExecutionInfo(executionHistory),
		QueuedTests:       make([]TestExecutionInfo, 0), // Placeholder
		ExecutionStats: ExecutionStats{
			TotalExecutions:   aggregatedMetrics.TestExecution.TotalExecutions,
			AverageRuntime:    aggregatedMetrics.TestExecution.AverageRuntime,
			MedianRuntime:     aggregatedMetrics.TestExecution.MedianRuntime,
			TestsPerHour:      aggregatedMetrics.TestExecution.TestsPerHour,
			ParallelismFactor: aggregatedMetrics.TestExecution.ParallelismFactor,
		},
	}

	// Build resource metrics data
	resourceMetricsData := ResourceMetricsData{
		SystemMetrics: SystemResourceMetrics{
			CPUPercent:    resourceUsage.CPUPercent,
			MemoryPercent: 0, // Would need total memory to calculate
			DiskPercent:   0, // Would need disk info
			LoadAverage:   0, // Placeholder
			NetworkIOKB:   resourceUsage.NetworkIOKB,
			DiskIOKB:      resourceUsage.DiskIOKB,
		},
		ProcessMetrics: ProcessResourceMetrics{
			CPUPercent:    0, // Would get from detailed metrics
			MemoryMB:      resourceUsage.MemoryMB,
			MemoryPercent: 0,
			NumThreads:    0,
			NumFDs:        0,
		},
		DatabaseMetrics: DatabaseResourceMetrics{
			ActiveConnections: resourceUsage.DatabaseConns,
			MaxConnections:    150, // Placeholder
			ConnectionPercent: float64(resourceUsage.DatabaseConns) / 150.0 * 100,
			SlowQueries:       0,
			QueryRate:         0,
			CacheHitRatio:     0,
		},
		CacheMetrics: CacheResourceMetrics{
			HitRate:         float64(resourceUsage.CacheHits) / float64(resourceUsage.CacheHits+resourceUsage.CacheMisses),
			MissRate:        float64(resourceUsage.CacheMisses) / float64(resourceUsage.CacheHits+resourceUsage.CacheMisses),
			KeyCount:        0,
			MemoryUsedMB:    0,
			ConnectionCount: 0,
		},
		ResourceAlerts: d.convertResourceAlerts(d.resourceTracker.GetActiveAlerts()),
	}

	// Build alerts data
	activeAlerts := d.alertManager.GetActiveAlerts()
	alertHistory := d.alertManager.GetAlertHistory(20)

	alertsData := AlertsData{
		ActiveAlerts: d.convertToAlertInfo(activeAlerts),
		RecentAlerts: d.convertHistoryToAlertInfo(alertHistory),
		AlertSummary: alertSummary,
		AlertTrends:  d.calculateAlertTrends(),
	}

	return &DashboardData{
		Overview:        overview,
		TestExecution:   testExecutionData,
		ResourceMetrics: resourceMetricsData,
		Alerts:          alertsData,
		Performance:     d.buildPerformanceData(aggregatedMetrics),
		Trends:          d.buildTrendsData(aggregatedMetrics),
		SystemHealth:    d.buildSystemHealthData(detailedMetrics),
		Timestamp:       time.Now(),
	}
}

// broadcastUpdates broadcasts dashboard updates to all connected clients
func (d *MonitoringDashboard) broadcastUpdates(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopChan:
			return
		case <-ticker.C:
			d.sendUpdatesToAllClients()
		}
	}
}

// sendUpdatesToAllClients sends updates to all connected clients
func (d *MonitoringDashboard) sendUpdatesToAllClients() {
	d.mu.RLock()
	clients := make([]*DashboardClient, 0, len(d.clients))
	for _, client := range d.clients {
		clients = append(clients, client)
	}
	d.mu.RUnlock()

	data := d.collectDashboardData()

	for _, client := range clients {
		d.sendDashboardData(client)
		_ = data // Use data in actual implementation
	}
}

// sendDashboardData sends dashboard data to a specific client
func (d *MonitoringDashboard) sendDashboardData(client *DashboardClient) {
	data := d.collectDashboardData()

	message := map[string]interface{}{
		"type": "dashboard_update",
		"data": data,
	}

	if err := client.Connection.WriteJSON(message); err != nil {
		log.Printf("Failed to send data to client %s: %v", client.ID, err)
		d.removeClient(client.ID)
	}
}

// handleClientMessages handles messages from dashboard clients
func (d *MonitoringDashboard) handleClientMessages(client *DashboardClient) {
	defer d.removeClient(client.ID)

	for {
		var message map[string]interface{}
		err := client.Connection.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", client.ID, err)
			}
			break
		}

		client.LastSeen = time.Now()

		// Handle different message types
		if msgType, ok := message["type"].(string); ok {
			switch msgType {
			case "ping":
				d.sendPong(client)
			case "subscribe":
				// Handle subscription requests
			case "unsubscribe":
				// Handle unsubscription requests
			}
		}
	}
}

// sendPong sends a pong response to a client
func (d *MonitoringDashboard) sendPong(client *DashboardClient) {
	message := map[string]interface{}{
		"type": "pong",
		"timestamp": time.Now(),
	}

	client.Connection.WriteJSON(message)
}

// removeClient removes a client from the dashboard
func (d *MonitoringDashboard) removeClient(clientID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if client, exists := d.clients[clientID]; exists {
		client.Connection.Close()
		delete(d.clients, clientID)
		log.Printf("Dashboard client disconnected: %s", clientID)
	}
}

// cleanupClients periodically cleans up inactive clients
func (d *MonitoringDashboard) cleanupClients(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopChan:
			return
		case <-ticker.C:
			d.performClientCleanup()
		}
	}
}

// performClientCleanup removes inactive clients
func (d *MonitoringDashboard) performClientCleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)
	toRemove := make([]string, 0)

	for id, client := range d.clients {
		if client.LastSeen.Before(cutoff) {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		if client, exists := d.clients[id]; exists {
			client.Connection.Close()
			delete(d.clients, id)
			log.Printf("Removed inactive dashboard client: %s", id)
		}
	}
}

// Helper methods for data conversion and calculation

func (d *MonitoringDashboard) calculateSystemHealth() string {
	// This would calculate overall system health based on various metrics
	return "healthy"
}

func (d *MonitoringDashboard) calculateOverallScore() float64 {
	// This would calculate an overall quality/health score
	return 0.95
}

func (d *MonitoringDashboard) convertToExecutionInfo(executions map[string]*TestExecution) []TestExecutionInfo {
	info := make([]TestExecutionInfo, 0, len(executions))
	for _, execution := range executions {
		info = append(info, TestExecutionInfo{
			ID:           execution.ID,
			TestSuite:    execution.TestSuite,
			TestName:     execution.TestName,
			Status:       string(execution.Status),
			Progress:     execution.Progress,
			Duration:     execution.Duration,
			Environment:  execution.Environment,
			StartTime:    execution.StartTime,
			EndTime:      execution.EndTime,
			ErrorMessage: execution.ErrorMessage,
		})
	}
	return info
}

func (d *MonitoringDashboard) convertHistoryToExecutionInfo(history []TestExecution) []TestExecutionInfo {
	info := make([]TestExecutionInfo, len(history))
	for i, execution := range history {
		info[i] = TestExecutionInfo{
			ID:           execution.ID,
			TestSuite:    execution.TestSuite,
			TestName:     execution.TestName,
			Status:       string(execution.Status),
			Progress:     execution.Progress,
			Duration:     execution.Duration,
			Environment:  execution.Environment,
			StartTime:    execution.StartTime,
			EndTime:      execution.EndTime,
			ErrorMessage: execution.ErrorMessage,
		}
	}
	return info
}

func (d *MonitoringDashboard) convertResourceAlerts(alerts []ResourceAlert) []ResourceAlertInfo {
	info := make([]ResourceAlertInfo, len(alerts))
	for i, alert := range alerts {
		info[i] = ResourceAlertInfo{
			Type:      alert.Type,
			Message:   alert.Message,
			Severity:  alert.Severity,
			Value:     alert.Value,
			Threshold: alert.Threshold,
			Timestamp: alert.Timestamp,
		}
	}
	return info
}

func (d *MonitoringDashboard) convertToAlertInfo(alerts map[string]*TestAlert) []AlertInfo {
	info := make([]AlertInfo, 0, len(alerts))
	for _, alert := range alerts {
		info = append(info, AlertInfo{
			ID:        alert.ID,
			Type:      string(alert.Type),
			Severity:  string(alert.Severity),
			Title:     alert.Title,
			Message:   alert.Message,
			Status:    string(alert.Status),
			CreatedAt: alert.CreatedAt,
			Source:    alert.Source,
		})
	}
	return info
}

func (d *MonitoringDashboard) convertHistoryToAlertInfo(history []TestAlert) []AlertInfo {
	info := make([]AlertInfo, len(history))
	for i, alert := range history {
		info[i] = AlertInfo{
			ID:        alert.ID,
			Type:      string(alert.Type),
			Severity:  string(alert.Severity),
			Title:     alert.Title,
			Message:   alert.Message,
			Status:    string(alert.Status),
			CreatedAt: alert.CreatedAt,
			Source:    alert.Source,
		}
	}
	return info
}

func (d *MonitoringDashboard) calculateAlertTrends() []AlertTrend {
	// This would calculate alert trends over time
	trends := make([]AlertTrend, 24)
	for i := 0; i < 24; i++ {
		trends[i] = AlertTrend{
			Hour:  i,
			Count: 0, // Placeholder
		}
	}
	return trends
}

func (d *MonitoringDashboard) buildPerformanceData(metrics *AggregatedMetrics) PerformanceData {
	return PerformanceData{
		ResponseTimes: ResponseTimeData{
			Average: metrics.Performance.ResponseTimes.Average,
			Median:  metrics.Performance.ResponseTimes.Median,
			P95:     metrics.Performance.ResponseTimes.P95,
			P99:     metrics.Performance.ResponseTimes.P99,
			History: make([]TimePoint, 0), // Placeholder
		},
		ThroughputData: ThroughputData{
			Current: metrics.Performance.Throughput.RequestsPerSecond,
			Peak:    metrics.Performance.Throughput.PeakThroughput,
			Average: metrics.Performance.Throughput.RequestsPerSecond,
			History: make([]TimePoint, 0), // Placeholder
		},
		ErrorRates: ErrorRateData{
			Current: metrics.Infrastructure.ErrorRate,
			Average: metrics.Infrastructure.ErrorRate,
			History: make([]TimePoint, 0), // Placeholder
		},
		PerformanceTrend: PerformanceTrend{
			Direction: metrics.Trends.PerformanceTrend,
			Change:    0, // Placeholder
			Period:    "1h",
		},
	}
}

func (d *MonitoringDashboard) buildTrendsData(metrics *AggregatedMetrics) TrendsData {
	return TrendsData{
		ExecutionTrend: TrendInfo{
			Direction:   metrics.Trends.ExecutionTrend,
			Change:      0, // Placeholder
			Period:      "1h",
			Confidence:  0.95,
			Prediction:  0,
			LastUpdated: time.Now(),
		},
		PerformanceTrend: TrendInfo{
			Direction:   metrics.Trends.PerformanceTrend,
			Change:      0,
			Period:      "1h",
			Confidence:  0.95,
			Prediction:  0,
			LastUpdated: time.Now(),
		},
		QualityTrend: TrendInfo{
			Direction:   metrics.Trends.QualityTrend,
			Change:      0,
			Period:      "1h",
			Confidence:  0.95,
			Prediction:  0,
			LastUpdated: time.Now(),
		},
		ReliabilityTrend: TrendInfo{
			Direction:   metrics.Trends.ReliabilityTrend,
			Change:      0,
			Period:      "1h",
			Confidence:  0.95,
			Prediction:  0,
			LastUpdated: time.Now(),
		},
		ResourceTrend: TrendInfo{
			Direction:   "stable",
			Change:      0,
			Period:      "1h",
			Confidence:  0.95,
			Prediction:  0,
			LastUpdated: time.Now(),
		},
	}
}

func (d *MonitoringDashboard) buildSystemHealthData(metrics *ResourceMetrics) SystemHealthData {
	if metrics == nil {
		return SystemHealthData{
			OverallStatus:   "unknown",
			ComponentHealth: make(map[string]string),
			HealthScore:     0,
			Uptime:          0,
			HealthHistory:   make([]HealthDataPoint, 0),
		}
	}

	return SystemHealthData{
		OverallStatus: "healthy",
		ComponentHealth: map[string]string{
			"database": "healthy",
			"cache":    "healthy",
			"system":   "healthy",
		},
		HealthScore:   0.95,
		Uptime:        24 * time.Hour, // Placeholder
		HealthHistory: make([]HealthDataPoint, 0),
	}
}

// Utility functions

func generateClientID() string {
	return fmt.Sprintf("client_%d_%d", time.Now().UnixNano(), time.Now().Unix()%1000)
}