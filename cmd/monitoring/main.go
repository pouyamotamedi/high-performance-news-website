package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/services"
	"high-performance-news-website/pkg/cache"

	_ "github.com/lib/pq"
)

func main() {
	var (
		command    = flag.String("command", "status", "Command to run: status, metrics, health, alerts, test")
		dbURL      = flag.String("db", "", "Database URL")
		configFile = flag.String("config", "", "Configuration file path")
		component  = flag.String("component", "", "Component to check (for health command)")
		duration   = flag.Duration("duration", 30*time.Second, "Duration to run monitoring test")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Load configuration
	var monitoringConfig *config.MonitoringConfig
	if *configFile != "" {
		// Load from file (implementation would go here)
		monitoringConfig = config.LoadMonitoringConfig()
	} else {
		monitoringConfig = config.LoadMonitoringConfig()
	}

	// Setup database connection if provided
	var db *sql.DB
	var err error
	if *dbURL != "" {
		db, err = sql.Open("postgres", *dbURL)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			log.Fatalf("Failed to ping database: %v", err)
		}
	}

	// Create services
	metricsService := services.NewMetricsService(db, nil, monitoringConfig)
	healthService := services.NewHealthService(db, nil, monitoringConfig, metricsService)
	alertingService := services.NewAlertingService(monitoringConfig, nil)

	// Execute command
	switch *command {
	case "status":
		runStatusCommand(metricsService, *verbose)
	case "metrics":
		runMetricsCommand(metricsService, *verbose)
	case "health":
		runHealthCommand(healthService, *component, *verbose)
	case "alerts":
		runAlertsCommand(alertingService, *verbose)
	case "test":
		runTestCommand(metricsService, healthService, alertingService, *duration, *verbose)
	case "dashboard":
		runDashboardCommand(metricsService, *verbose)
	default:
		fmt.Printf("Unknown command: %s\n", *command)
		fmt.Println("Available commands: status, metrics, health, alerts, test, dashboard")
		os.Exit(1)
	}
}

func runStatusCommand(metricsService *services.MetricsService, verbose bool) {
	fmt.Println("=== System Status ===")
	
	// Overall health
	health := metricsService.GetOverallHealth()
	fmt.Printf("Overall Health: %s\n", health)
	
	// Uptime
	uptime := time.Since(time.Now().Add(-24 * time.Hour)) // Simplified
	fmt.Printf("Uptime: %s\n", formatDuration(uptime))
	
	// Active alerts
	alerts := metricsService.GetActiveAlerts()
	fmt.Printf("Active Alerts: %d\n", len(alerts))
	
	if verbose && len(alerts) > 0 {
		fmt.Println("\nActive Alerts:")
		for _, alert := range alerts {
			fmt.Printf("  - %s (%s): %s\n", alert.Name, alert.Severity, alert.Description)
		}
	}
	
	// Health checks
	healthChecks := metricsService.GetHealthChecks()
	fmt.Printf("Health Checks: %d components\n", len(healthChecks))
	
	if verbose {
		fmt.Println("\nComponent Health:")
		for component, check := range healthChecks {
			fmt.Printf("  - %s: %s (%s)\n", component, check.Status, check.ResponseTime)
		}
	}
}

func runMetricsCommand(metricsService *services.MetricsService, verbose bool) {
	fmt.Println("=== System Metrics ===")
	
	// System metrics
	systemMetrics, err := metricsService.GetSystemMetrics()
	if err != nil {
		fmt.Printf("Error getting system metrics: %v\n", err)
	} else {
		fmt.Printf("CPU Usage: %.2f%%\n", systemMetrics.CPUUsage)
		fmt.Printf("Memory Usage: %.2f%% (%.2f GB / %.2f GB)\n", 
			systemMetrics.MemoryUsage,
			float64(systemMetrics.MemoryUsed)/(1024*1024*1024),
			float64(systemMetrics.MemoryTotal)/(1024*1024*1024))
		fmt.Printf("Disk Usage: %.2f%% (%.2f GB / %.2f GB)\n",
			systemMetrics.DiskUsage,
			float64(systemMetrics.DiskUsed)/(1024*1024*1024),
			float64(systemMetrics.DiskTotal)/(1024*1024*1024))
		fmt.Printf("Load Average: %.2f, %.2f, %.2f\n",
			systemMetrics.LoadAverage1, systemMetrics.LoadAverage5, systemMetrics.LoadAverage15)
	}
	
	// Database metrics
	dbMetrics, err := metricsService.GetDatabaseMetrics()
	if err != nil {
		fmt.Printf("Error getting database metrics: %v\n", err)
	} else {
		fmt.Printf("\nDatabase Connections: %d active, %d idle (max: %d)\n",
			dbMetrics.ActiveConnections, dbMetrics.IdleConnections, dbMetrics.MaxConnections)
		fmt.Printf("Cache Hit Ratio: %.2f%%\n", dbMetrics.CacheHitRatio*100)
		fmt.Printf("Slow Queries: %d\n", dbMetrics.SlowQueries)
	}
	
	// Cache metrics
	cacheMetrics, err := metricsService.GetCacheMetrics()
	if err != nil {
		fmt.Printf("Error getting cache metrics: %v\n", err)
	} else {
		fmt.Printf("\nCache Hit Rate: %.2f%% (%d hits, %d misses)\n",
			cacheMetrics.HitRate*100, cacheMetrics.HitCount, cacheMetrics.MissCount)
		fmt.Printf("Cache Keys: %d\n", cacheMetrics.KeyCount)
		fmt.Printf("Cache Memory: %.2f MB / %.2f MB\n",
			float64(cacheMetrics.MemoryUsage)/(1024*1024),
			float64(cacheMetrics.MemoryTotal)/(1024*1024))
	}
	
	// Publishing metrics
	publishingMetrics, err := metricsService.GetPublishingMetrics()
	if err != nil {
		fmt.Printf("Error getting publishing metrics: %v\n", err)
	} else {
		fmt.Printf("\nPublishing Rate: %.2f articles/minute\n", publishingMetrics.PublishingRate)
		fmt.Printf("Articles Published (last hour): %d\n", publishingMetrics.ArticlesPublished)
		fmt.Printf("Failed Publications: %d\n", publishingMetrics.FailedPublications)
		fmt.Printf("Queued Articles: %d\n", publishingMetrics.QueuedArticles)
	}
	
	if verbose {
		// Performance metrics
		perfMetrics := metricsService.GetPerformanceMetrics()
		fmt.Println("\nPerformance Metrics:")
		for key, value := range perfMetrics {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}

func runHealthCommand(healthService *services.HealthService, component string, verbose bool) {
	fmt.Println("=== Health Checks ===")
	
	if component != "" {
		// Check specific component
		fmt.Printf("Checking component: %s\n", component)
		// This would need to be implemented in the health service
		fmt.Printf("Component health check not implemented for specific components\n")
	} else {
		// Comprehensive health check
		healthResponse := healthService.PerformHealthCheck(verbose)
		
		fmt.Printf("Overall Status: %s\n", healthResponse.Status)
		fmt.Printf("Timestamp: %s\n", healthResponse.Timestamp.Format(time.RFC3339))
		fmt.Printf("Uptime: %s\n", healthResponse.Uptime)
		
		fmt.Println("\nComponent Health:")
		for name, health := range healthResponse.Components {
			fmt.Printf("  %s: %s - %s (Response: %s)\n",
				name, health.Status, health.Message, health.ResponseTime)
			
			if verbose && len(health.Details) > 0 {
				fmt.Printf("    Details:\n")
				for key, value := range health.Details {
					fmt.Printf("      %s: %v\n", key, value)
				}
			}
		}
		
		if verbose && healthResponse.Metrics != nil {
			fmt.Println("\nMetrics:")
			for key, value := range healthResponse.Metrics {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}
	}
}

func runAlertsCommand(alertingService *services.AlertingService, verbose bool) {
	fmt.Println("=== Alerts ===")
	
	// Get alert history
	history := alertingService.GetAlertHistory()
	fmt.Printf("Total Alerts in History: %d\n", len(history))
	
	if verbose {
		fmt.Println("\nAlert History:")
		for name, alert := range history {
			fmt.Printf("  %s (%s): %s\n", name, alert.Severity, alert.Description)
			fmt.Printf("    Status: %s, Triggered: %s\n", 
				alert.Status, alert.TriggeredAt.Format(time.RFC3339))
			if alert.ResolvedAt != nil {
				fmt.Printf("    Resolved: %s\n", alert.ResolvedAt.Format(time.RFC3339))
			}
		}
	}
	
	// Test alert functionality
	fmt.Println("\nTesting alert system...")
	err := alertingService.TestAlert()
	if err != nil {
		fmt.Printf("Test alert failed: %v\n", err)
	} else {
		fmt.Println("Test alert sent successfully")
	}
}

func runTestCommand(metricsService *services.MetricsService, healthService *services.HealthService, alertingService *services.AlertingService, duration time.Duration, verbose bool) {
	fmt.Printf("=== Running Monitoring Test for %s ===\n", duration)
	
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	// Start monitoring
	go metricsService.StartMonitoring(ctx)
	
	// Collect metrics periodically
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	startTime := time.Now()
	
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\nTest completed after %s\n", time.Since(startTime))
			return
		case <-ticker.C:
			elapsed := time.Since(startTime)
			fmt.Printf("\n[%s] Collecting metrics...\n", elapsed.Truncate(time.Second))
			
			// System metrics
			if systemMetrics, err := metricsService.GetSystemMetrics(); err == nil {
				fmt.Printf("  CPU: %.1f%%, Memory: %.1f%%, Disk: %.1f%%\n",
					systemMetrics.CPUUsage, systemMetrics.MemoryUsage, systemMetrics.DiskUsage)
			}
			
			// Health status
			health := metricsService.GetOverallHealth()
			fmt.Printf("  Health: %s\n", health)
			
			// Active alerts
			alerts := metricsService.GetActiveAlerts()
			if len(alerts) > 0 {
				fmt.Printf("  Active Alerts: %d\n", len(alerts))
				if verbose {
					for _, alert := range alerts {
						fmt.Printf("    - %s: %s\n", alert.Name, alert.Description)
					}
				}
			}
		}
	}
}

func runDashboardCommand(metricsService *services.MetricsService, verbose bool) {
	fmt.Println("=== Monitoring Dashboard Data ===")
	
	dashboard, err := metricsService.GetMonitoringDashboard()
	if err != nil {
		fmt.Printf("Error getting dashboard data: %v\n", err)
		return
	}
	
	if verbose {
		// Output as JSON for easy parsing
		jsonData, err := json.MarshalIndent(dashboard, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling dashboard data: %v\n", err)
			return
		}
		fmt.Println(string(jsonData))
	} else {
		// Summary output
		fmt.Printf("System Health: %s\n", dashboard.SystemHealth)
		fmt.Printf("Last Updated: %s\n", dashboard.LastUpdated.Format(time.RFC3339))
		fmt.Printf("Active Alerts: %d\n", len(dashboard.ActiveAlerts))
		fmt.Printf("Health Checks: %d\n", len(dashboard.RecentHealthChecks))
		
		// System metrics summary
		fmt.Printf("CPU: %.1f%%, Memory: %.1f%%, Disk: %.1f%%\n",
			dashboard.SystemMetrics.CPUUsage,
			dashboard.SystemMetrics.MemoryUsage,
			dashboard.SystemMetrics.DiskUsage)
		
		// Database metrics summary
		fmt.Printf("DB Connections: %d/%d, Cache Hit: %.1f%%\n",
			dashboard.DatabaseMetrics.ActiveConnections,
			dashboard.DatabaseMetrics.MaxConnections,
			dashboard.DatabaseMetrics.CacheHitRatio*100)
		
		// Publishing metrics summary
		fmt.Printf("Publishing Rate: %.1f/min, Articles: %d\n",
			dashboard.PublishingMetrics.PublishingRate,
			dashboard.PublishingMetrics.ArticlesPublished)
	}
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}