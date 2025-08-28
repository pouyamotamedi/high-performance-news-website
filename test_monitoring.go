package main

import (
	"context"
	"log"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/services"
)

func main() {
	log.Println("Testing monitoring system...")
	
	// Load monitoring config
	monitoringConfig := config.LoadMonitoringConfig()
	if err := monitoringConfig.Validate(); err != nil {
		log.Printf("Warning: Invalid monitoring config: %v", err)
	}
	
	// Create metrics service
	metricsService := services.NewMetricsService(nil, nil, monitoringConfig)
	
	// Start monitoring
	ctx := context.Background()
	go metricsService.StartMonitoring(ctx)
	
	log.Println("Monitoring system started, collecting metrics for 10 seconds...")
	time.Sleep(10 * time.Second)
	
	// Get system metrics
	systemMetrics, err := metricsService.GetSystemMetrics()
	if err != nil {
		log.Printf("Error getting system metrics: %v", err)
	} else {
		log.Printf("System Metrics - CPU: %.2f%%, Memory: %.2f%%, Disk: %.2f%%", 
			systemMetrics.CPUUsage, systemMetrics.MemoryUsage, systemMetrics.DiskUsage)
	}
	
	// Get overall health
	health := metricsService.GetOverallHealth()
	log.Printf("Overall system health: %s", health)
	
	// Get active alerts
	alerts := metricsService.GetActiveAlerts()
	log.Printf("Active alerts: %d", len(alerts))
	
	log.Println("Monitoring test completed successfully!")
}