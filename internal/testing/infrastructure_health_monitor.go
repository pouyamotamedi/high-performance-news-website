package testing

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

// InfrastructureHealthMonitor provides comprehensive health monitoring and predictive maintenance
type InfrastructureHealthMonitor struct {
	metrics         map[string]*HealthMetric
	thresholds      map[string]HealthThreshold
	predictors      map[string]*PredictiveModel
	alerts          []HealthAlert
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	monitorInterval time.Duration
}

// HealthMetric represents a monitored health metric
type HealthMetric struct {
	Name        string                 `json:"name"`
	Type        MetricType            `json:"type"`
	Value       float64               `json:"value"`
	Unit        string                `json:"unit"`
	Timestamp   time.Time             `json:"timestamp"`
	History     []HistoricalValue     `json:"history"`
	Trend       TrendDirection        `json:"trend"`
	Status      HealthStatus          `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// HistoricalValue stores historical metric data
type HistoricalValue struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthThreshold defines alert thresholds for metrics
type HealthThreshold struct {
	MetricName    string  `json:"metric_name"`
	WarningLevel  float64 `json:"warning_level"`
	CriticalLevel float64 `json:"critical_level"`
	Direction     string  `json:"direction"` // "above" or "below"
}

// PredictiveModel provides predictive maintenance capabilities
type PredictiveModel struct {
	MetricName       string                 `json:"metric_name"`
	Algorithm        string                 `json:"algorithm"`
	TrainingData     []HistoricalValue     `json:"training_data"`
	Predictions      []PredictedValue      `json:"predictions"`
	Accuracy         float64               `json:"accuracy"`
	LastTrained      time.Time             `json:"last_trained"`
	NextMaintenance  *time.Time            `json:"next_maintenance,omitempty"`
}

// PredictedValue represents a predicted metric value
type PredictedValue struct {
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	Confidence float64   `json:"confidence"`
}

// HealthAlert represents a health monitoring alert
type HealthAlert struct {
	ID          string                 `json:"id"`
	MetricName  string                 `json:"metric_name"`
	Level       AlertLevel            `json:"level"`
	Message     string                `json:"message"`
	Value       float64               `json:"value"`
	Threshold   float64               `json:"threshold"`
	Timestamp   time.Time             `json:"timestamp"`
	Resolved    bool                  `json:"resolved"`
	ResolvedAt  *time.Time            `json:"resolved_at,omitempty"`
	Context     map[string]interface{} `json:"context"`
}

type MetricType string

const (
	MetricTypeCPU        MetricType = "cpu"
	MetricTypeMemory     MetricType = "memory"
	MetricTypeDisk       MetricType = "disk"
	MetricTypeNetwork    MetricType = "network"
	MetricTypeDatabase   MetricType = "database"
	MetricTypeCache      MetricType = "cache"
	MetricTypeLatency    MetricType = "latency"
	MetricTypeThroughput MetricType = "throughput"
	MetricTypeErrors     MetricType = "errors"
)

type HealthStatus string

const (
	HealthStatusHealthy  HealthStatus = "healthy"
	HealthStatusWarning  HealthStatus = "warning"
	HealthStatusCritical HealthStatus = "critical"
	HealthStatusUnknown  HealthStatus = "unknown"
)

type TrendDirection string

const (
	TrendUp    TrendDirection = "up"
	TrendDown  TrendDirection = "down"
	TrendFlat  TrendDirection = "flat"
)

type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// NewInfrastructureHealthMonitor creates a new health monitor
func NewInfrastructureHealthMonitor() *InfrastructureHealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	ihm := &InfrastructureHealthMonitor{
		metrics:         make(map[string]*HealthMetric),
		thresholds:      make(map[string]HealthThreshold),
		predictors:      make(map[string]*PredictiveModel),
		alerts:          make([]HealthAlert, 0),
		ctx:             ctx,
		cancel:          cancel,
		monitorInterval: 10 * time.Second,
	}
	
	// Initialize default metrics and thresholds
	ihm.initializeDefaultMetrics()
	ihm.initializeDefaultThresholds()
	
	return ihm
}

// Start begins health monitoring
func (ihm *InfrastructureHealthMonitor) Start(ctx context.Context) error {
	log.Println("Starting Infrastructure Health Monitor...")
	
	// Start monitoring loop
	go ihm.monitoringLoop()
	
	// Start predictive maintenance
	go ihm.predictiveMaintenanceLoop()
	
	log.Println("Infrastructure Health Monitor started")
	return nil
}

// Stop gracefully shuts down the health monitor
func (ihm *InfrastructureHealthMonitor) Stop() error {
	log.Println("Stopping Infrastructure Health Monitor...")
	ihm.cancel()
	return nil
}

// monitoringLoop runs the main monitoring cycle
func (ihm *InfrastructureHealthMonitor) monitoringLoop() {
	ticker := time.NewTicker(ihm.monitorInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ihm.ctx.Done():
			return
		case <-ticker.C:
			ihm.collectMetrics()
			ihm.analyzeMetrics()
			ihm.checkThresholds()
		}
	}
}

// predictiveMaintenanceLoop runs predictive analysis
func (ihm *InfrastructureHealthMonitor) predictiveMaintenanceLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ihm.ctx.Done():
			return
		case <-ticker.C:
			ihm.runPredictiveAnalysis()
		}
	}
}

// collectMetrics gathers current system metrics
func (ihm *InfrastructureHealthMonitor) collectMetrics() {
	now := time.Now()
	
	// Collect CPU metrics
	ihm.updateMetric("cpu_usage", ihm.getCPUUsage(), "percent", now)
	
	// Collect memory metrics
	ihm.updateMetric("memory_usage", ihm.getMemoryUsage(), "percent", now)
	ihm.updateMetric("memory_allocated", ihm.getMemoryAllocated(), "MB", now)
	
	// Collect disk metrics
	ihm.updateMetric("disk_usage", ihm.getDiskUsage(), "percent", now)
	ihm.updateMetric("disk_io", ihm.getDiskIO(), "ops/sec", now)
	
	// Collect network metrics
	ihm.updateMetric("network_latency", ihm.getNetworkLatency(), "ms", now)
	ihm.updateMetric("network_throughput", ihm.getNetworkThroughput(), "MB/s", now)
	
	// Collect application-specific metrics
	ihm.updateMetric("database_connections", ihm.getDatabaseConnections(), "count", now)
	ihm.updateMetric("cache_hit_rate", ihm.getCacheHitRate(), "percent", now)
	ihm.updateMetric("error_rate", ihm.getErrorRate(), "percent", now)
}

// updateMetric updates a metric with a new value
func (ihm *InfrastructureHealthMonitor) updateMetric(name string, value float64, unit string, timestamp time.Time) {
	ihm.mu.Lock()
	defer ihm.mu.Unlock()
	
	metric, exists := ihm.metrics[name]
	if !exists {
		metric = &HealthMetric{
			Name:     name,
			Type:     ihm.getMetricType(name),
			Unit:     unit,
			History:  make([]HistoricalValue, 0),
			Status:   HealthStatusUnknown,
			Metadata: make(map[string]interface{}),
		}
		ihm.metrics[name] = metric
	}
	
	// Update current value
	metric.Value = value
	metric.Timestamp = timestamp
	
	// Add to history (keep last 100 values)
	metric.History = append(metric.History, HistoricalValue{
		Value:     value,
		Timestamp: timestamp,
	})
	
	if len(metric.History) > 100 {
		metric.History = metric.History[1:]
	}
	
	// Update trend
	metric.Trend = ihm.calculateTrend(metric.History)
	
	// Update status based on thresholds
	metric.Status = ihm.calculateStatus(name, value)
}

// getMetricType determines the metric type based on name
func (ihm *InfrastructureHealthMonitor) getMetricType(name string) MetricType {
	switch {
	case contains(name, "cpu"):
		return MetricTypeCPU
	case contains(name, "memory"):
		return MetricTypeMemory
	case contains(name, "disk"):
		return MetricTypeDisk
	case contains(name, "network"):
		return MetricTypeNetwork
	case contains(name, "database"):
		return MetricTypeDatabase
	case contains(name, "cache"):
		return MetricTypeCache
	case contains(name, "latency"):
		return MetricTypeLatency
	case contains(name, "throughput"):
		return MetricTypeThroughput
	case contains(name, "error"):
		return MetricTypeErrors
	default:
		return MetricType("custom")
	}
}

// calculateTrend analyzes the trend in historical data
func (ihm *InfrastructureHealthMonitor) calculateTrend(history []HistoricalValue) TrendDirection {
	if len(history) < 3 {
		return TrendFlat
	}
	
	// Simple trend calculation using last 10 values
	start := len(history) - 10
	if start < 0 {
		start = 0
	}
	
	recentValues := history[start:]
	if len(recentValues) < 2 {
		return TrendFlat
	}
	
	firstHalf := recentValues[:len(recentValues)/2]
	secondHalf := recentValues[len(recentValues)/2:]
	
	firstAvg := ihm.calculateAverage(firstHalf)
	secondAvg := ihm.calculateAverage(secondHalf)
	
	diff := secondAvg - firstAvg
	threshold := firstAvg * 0.05 // 5% threshold
	
	if diff > threshold {
		return TrendUp
	} else if diff < -threshold {
		return TrendDown
	}
	
	return TrendFlat
}

// calculateAverage calculates the average of historical values
func (ihm *InfrastructureHealthMonitor) calculateAverage(values []HistoricalValue) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v.Value
	}
	
	return sum / float64(len(values))
}

// calculateStatus determines health status based on thresholds
func (ihm *InfrastructureHealthMonitor) calculateStatus(metricName string, value float64) HealthStatus {
	threshold, exists := ihm.thresholds[metricName]
	if !exists {
		return HealthStatusHealthy
	}
	
	if threshold.Direction == "above" {
		if value >= threshold.CriticalLevel {
			return HealthStatusCritical
		} else if value >= threshold.WarningLevel {
			return HealthStatusWarning
		}
	} else { // "below"
		if value <= threshold.CriticalLevel {
			return HealthStatusCritical
		} else if value <= threshold.WarningLevel {
			return HealthStatusWarning
		}
	}
	
	return HealthStatusHealthy
}

// checkThresholds checks all metrics against their thresholds
func (ihm *InfrastructureHealthMonitor) checkThresholds() {
	ihm.mu.RLock()
	defer ihm.mu.RUnlock()
	
	for name, metric := range ihm.metrics {
		if metric.Status == HealthStatusWarning || metric.Status == HealthStatusCritical {
			ihm.createAlert(name, metric)
		}
	}
}

// createAlert creates a new health alert
func (ihm *InfrastructureHealthMonitor) createAlert(metricName string, metric *HealthMetric) {
	threshold := ihm.thresholds[metricName]
	
	alertLevel := AlertLevelWarning
	thresholdValue := threshold.WarningLevel
	
	if metric.Status == HealthStatusCritical {
		alertLevel = AlertLevelCritical
		thresholdValue = threshold.CriticalLevel
	}
	
	alert := HealthAlert{
		ID:         fmt.Sprintf("%s_%d", metricName, time.Now().Unix()),
		MetricName: metricName,
		Level:      alertLevel,
		Message:    fmt.Sprintf("Metric %s is %s: %.2f %s", metricName, metric.Status, metric.Value, metric.Unit),
		Value:      metric.Value,
		Threshold:  thresholdValue,
		Timestamp:  time.Now(),
		Resolved:   false,
		Context: map[string]interface{}{
			"trend":     metric.Trend,
			"threshold": threshold,
		},
	}
	
	ihm.alerts = append(ihm.alerts, alert)
	
	log.Printf("Health alert created: %s - %s", alert.ID, alert.Message)
}

// Metric collection methods (these would integrate with actual monitoring systems)
func (ihm *InfrastructureHealthMonitor) getCPUUsage() float64 {
	// This would integrate with actual CPU monitoring
	// For now, return a simulated value
	return 25.0 + (float64(time.Now().Second()) * 0.5)
}

func (ihm *InfrastructureHealthMonitor) getMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Calculate memory usage percentage (simplified)
	const maxMemory = 1024 * 1024 * 1024 // 1GB
	return float64(m.Alloc) / maxMemory * 100
}

func (ihm *InfrastructureHealthMonitor) getMemoryAllocated() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / 1024 / 1024 // MB
}

func (ihm *InfrastructureHealthMonitor) getDiskUsage() float64 {
	// This would check actual disk usage
	return 45.0 // Simulated value
}

func (ihm *InfrastructureHealthMonitor) getDiskIO() float64 {
	// This would measure actual disk I/O
	return 150.0 // Simulated value
}

func (ihm *InfrastructureHealthMonitor) getNetworkLatency() float64 {
	// This would measure actual network latency
	return 5.0 + (float64(time.Now().Second()%10) * 0.5)
}

func (ihm *InfrastructureHealthMonitor) getNetworkThroughput() float64 {
	// This would measure actual network throughput
	return 50.0 // Simulated value
}

func (ihm *InfrastructureHealthMonitor) getDatabaseConnections() float64 {
	// This would check actual database connection count
	return 15.0 // Simulated value
}

func (ihm *InfrastructureHealthMonitor) getCacheHitRate() float64 {
	// This would check actual cache hit rate
	return 85.0 + (float64(time.Now().Second()%20) * 0.1)
}

func (ihm *InfrastructureHealthMonitor) getErrorRate() float64 {
	// This would check actual error rate
	return 0.5 // Simulated value
}

// initializeDefaultMetrics sets up default metrics
func (ihm *InfrastructureHealthMonitor) initializeDefaultMetrics() {
	metrics := []string{
		"cpu_usage", "memory_usage", "memory_allocated",
		"disk_usage", "disk_io", "network_latency",
		"network_throughput", "database_connections",
		"cache_hit_rate", "error_rate",
	}
	
	for _, name := range metrics {
		ihm.metrics[name] = &HealthMetric{
			Name:     name,
			Type:     ihm.getMetricType(name),
			History:  make([]HistoricalValue, 0),
			Status:   HealthStatusUnknown,
			Metadata: make(map[string]interface{}),
		}
	}
}

// initializeDefaultThresholds sets up default alert thresholds
func (ihm *InfrastructureHealthMonitor) initializeDefaultThresholds() {
	ihm.thresholds["cpu_usage"] = HealthThreshold{
		MetricName:    "cpu_usage",
		WarningLevel:  70.0,
		CriticalLevel: 90.0,
		Direction:     "above",
	}
	
	ihm.thresholds["memory_usage"] = HealthThreshold{
		MetricName:    "memory_usage",
		WarningLevel:  80.0,
		CriticalLevel: 95.0,
		Direction:     "above",
	}
	
	ihm.thresholds["disk_usage"] = HealthThreshold{
		MetricName:    "disk_usage",
		WarningLevel:  80.0,
		CriticalLevel: 95.0,
		Direction:     "above",
	}
	
	ihm.thresholds["network_latency"] = HealthThreshold{
		MetricName:    "network_latency",
		WarningLevel:  100.0,
		CriticalLevel: 500.0,
		Direction:     "above",
	}
	
	ihm.thresholds["cache_hit_rate"] = HealthThreshold{
		MetricName:    "cache_hit_rate",
		WarningLevel:  70.0,
		CriticalLevel: 50.0,
		Direction:     "below",
	}
	
	ihm.thresholds["error_rate"] = HealthThreshold{
		MetricName:    "error_rate",
		WarningLevel:  1.0,
		CriticalLevel: 5.0,
		Direction:     "above",
	}
}

// GetOverallHealthScore calculates an overall health score
func (ihm *InfrastructureHealthMonitor) GetOverallHealthScore() float64 {
	ihm.mu.RLock()
	defer ihm.mu.RUnlock()
	
	if len(ihm.metrics) == 0 {
		return 0.0
	}
	
	totalScore := 0.0
	count := 0
	
	for _, metric := range ihm.metrics {
		score := 0.0
		switch metric.Status {
		case HealthStatusHealthy:
			score = 100.0
		case HealthStatusWarning:
			score = 60.0
		case HealthStatusCritical:
			score = 20.0
		case HealthStatusUnknown:
			score = 50.0
		}
		
		totalScore += score
		count++
	}
	
	return totalScore / float64(count)
}

// runPredictiveAnalysis performs predictive maintenance analysis
func (ihm *InfrastructureHealthMonitor) runPredictiveAnalysis() {
	ihm.mu.Lock()
	defer ihm.mu.Unlock()
	
	for name, metric := range ihm.metrics {
		if len(metric.History) < 10 {
			continue // Need more data for prediction
		}
		
		predictor := ihm.getOrCreatePredictor(name)
		ihm.updatePredictiveModel(predictor, metric.History)
		ihm.generatePredictions(predictor)
	}
}

// getOrCreatePredictor gets or creates a predictive model for a metric
func (ihm *InfrastructureHealthMonitor) getOrCreatePredictor(metricName string) *PredictiveModel {
	if predictor, exists := ihm.predictors[metricName]; exists {
		return predictor
	}
	
	predictor := &PredictiveModel{
		MetricName:   metricName,
		Algorithm:    "linear_regression",
		TrainingData: make([]HistoricalValue, 0),
		Predictions:  make([]PredictedValue, 0),
		Accuracy:     0.0,
		LastTrained:  time.Now(),
	}
	
	ihm.predictors[metricName] = predictor
	return predictor
}

// updatePredictiveModel updates the predictive model with new data
func (ihm *InfrastructureHealthMonitor) updatePredictiveModel(predictor *PredictiveModel, history []HistoricalValue) {
	// Simple linear regression for trend prediction
	predictor.TrainingData = history
	predictor.LastTrained = time.Now()
	
	// Calculate accuracy based on recent predictions vs actual values
	if len(predictor.Predictions) > 0 {
		predictor.Accuracy = ihm.calculatePredictionAccuracy(predictor, history)
	}
}

// generatePredictions generates future predictions
func (ihm *InfrastructureHealthMonitor) generatePredictions(predictor *PredictiveModel) {
	if len(predictor.TrainingData) < 5 {
		return
	}
	
	// Simple linear trend prediction
	recentData := predictor.TrainingData[len(predictor.TrainingData)-5:]
	trend := ihm.calculateLinearTrend(recentData)
	
	predictor.Predictions = make([]PredictedValue, 0)
	
	// Generate predictions for next 24 hours
	lastValue := recentData[len(recentData)-1].Value
	lastTime := recentData[len(recentData)-1].Timestamp
	
	for i := 1; i <= 24; i++ {
		predictedValue := lastValue + (trend * float64(i))
		confidence := 0.8 - (float64(i) * 0.02) // Decreasing confidence over time
		
		if confidence < 0.3 {
			confidence = 0.3
		}
		
		predictor.Predictions = append(predictor.Predictions, PredictedValue{
			Value:      predictedValue,
			Timestamp:  lastTime.Add(time.Duration(i) * time.Hour),
			Confidence: confidence,
		})
	}
	
	// Check if maintenance is needed
	ihm.checkMaintenanceNeeded(predictor)
}

// calculateLinearTrend calculates a simple linear trend
func (ihm *InfrastructureHealthMonitor) calculateLinearTrend(data []HistoricalValue) float64 {
	if len(data) < 2 {
		return 0
	}
	
	// Simple slope calculation
	firstValue := data[0].Value
	lastValue := data[len(data)-1].Value
	timeSpan := data[len(data)-1].Timestamp.Sub(data[0].Timestamp).Hours()
	
	if timeSpan == 0 {
		return 0
	}
	
	return (lastValue - firstValue) / timeSpan
}

// calculatePredictionAccuracy calculates how accurate recent predictions were
func (ihm *InfrastructureHealthMonitor) calculatePredictionAccuracy(predictor *PredictiveModel, actualHistory []HistoricalValue) float64 {
	// This would compare recent predictions with actual values
	// For now, return a simulated accuracy
	return 0.75 // 75% accuracy
}

// checkMaintenanceNeeded determines if predictive maintenance is needed
func (ihm *InfrastructureHealthMonitor) checkMaintenanceNeeded(predictor *PredictiveModel) {
	threshold, exists := ihm.thresholds[predictor.MetricName]
	if !exists {
		return
	}
	
	// Check if any predictions exceed thresholds
	for _, prediction := range predictor.Predictions {
		if prediction.Confidence > 0.6 {
			if (threshold.Direction == "above" && prediction.Value > threshold.WarningLevel) ||
			   (threshold.Direction == "below" && prediction.Value < threshold.WarningLevel) {
				
				maintenanceTime := prediction.Timestamp
				predictor.NextMaintenance = &maintenanceTime
				
				log.Printf("Predictive maintenance recommended for %s at %v (predicted value: %.2f)",
					predictor.MetricName, maintenanceTime, prediction.Value)
				break
			}
		}
	}
}