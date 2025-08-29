package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"high-performance-news-website/internal/api"
	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/services"
	"high-performance-news-website/pkg/cache"
)

func main() {
	log.Println("Starting High-Performance News Website Monitoring System...")

	// Load configuration
	cfg := config.LoadConfig()
	monitoringConfig := config.LoadMonitoringConfig()

	// Initialize database connection
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	// Initialize cache service (mock implementation for example)
	cacheService := &MockCacheService{}

	// Initialize email service (mock implementation for example)
	emailService := &MockEmailService{}

	// Create monitoring integration service
	monitoringService := services.NewMonitoringIntegrationService(
		db,
		cacheService,
		monitoringConfig,
		emailService,
	)

	// Start monitoring system
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := monitoringService.Start(ctx); err != nil {
		log.Fatalf("Failed to start monitoring system: %v", err)
	}

	// Set up HTTP server with monitoring endpoints
	router := setupRouter(monitoringService)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start HTTP server
	go func() {
		log.Println("Starting HTTP server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Start maintenance routine
	go maintenanceRoutine(monitoringService)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down monitoring system...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop monitoring system
	if err := monitoringService.Stop(); err != nil {
		log.Printf("Monitoring system shutdown error: %v", err)
	}

	log.Println("Monitoring system shutdown complete")
}

// setupRouter configures the HTTP router with monitoring endpoints
func setupRouter(monitoringService *services.MonitoringIntegrationService) *gin.Engine {
	router := gin.Default()

	// Create monitoring handlers
	monitoringHandlers := api.NewMonitoringHandlers(
		monitoringService.GetMetricsService(),
		monitoringService.GetHealthService(),
		monitoringService.GetAlertingService(),
	)

	// Register monitoring routes
	monitoringHandlers.RegisterRoutes(router)

	// Additional monitoring endpoints
	v1 := router.Group("/api/v1")
	{
		// System status endpoint
		v1.GET("/system/status", func(c *gin.Context) {
			status, err := monitoringService.GetSystemStatus()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, status)
		})

		// Dashboard data endpoint
		v1.GET("/dashboard", func(c *gin.Context) {
			dashboard, err := monitoringService.GetDashboardData()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dashboard)
		})

		// Test alert endpoint
		v1.POST("/test-alert", func(c *gin.Context) {
			if err := monitoringService.TestAlert(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Test alert sent successfully"})
		})

		// Recent logs endpoint
		v1.GET("/logs", func(c *gin.Context) {
			component := c.Query("component")
			level := services.LogLevel(c.Query("level"))
			limit := 100
			since := time.Now().Add(-24 * time.Hour)

			logs, err := monitoringService.GetRecentLogs(component, level, limit, since)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"logs": logs})
		})

		// Log statistics endpoint
		v1.GET("/logs/stats", func(c *gin.Context) {
			since := time.Now().Add(-24 * time.Hour)
			stats, err := monitoringService.GetLogStatistics(since)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, stats)
		})

		// Remediation actions endpoint
		v1.GET("/remediation/actions", func(c *gin.Context) {
			actions := monitoringService.GetRemediationActions()
			c.JSON(http.StatusOK, gin.H{"actions": actions})
		})

		// Operational runbooks endpoint
		v1.GET("/runbooks", func(c *gin.Context) {
			runbooks := monitoringService.GetOperationalRunbooks()
			c.JSON(http.StatusOK, gin.H{"runbooks": runbooks})
		})

		// Cache management endpoint
		v1.DELETE("/cache", func(c *gin.Context) {
			cacheType := c.DefaultQuery("type", "all")
			cleared, err := monitoringService.ClearCache(cacheType)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"cleared": cleared})
		})

		// Maintenance endpoint
		v1.POST("/maintenance", func(c *gin.Context) {
			if err := monitoringService.PerformMaintenance(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Maintenance completed successfully"})
		})
	}

	return router
}

// maintenanceRoutine performs periodic maintenance tasks
func maintenanceRoutine(monitoringService *services.MonitoringIntegrationService) {
	ticker := time.NewTicker(24 * time.Hour) // Daily maintenance
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Starting scheduled maintenance...")
			if err := monitoringService.PerformMaintenance(); err != nil {
				log.Printf("Maintenance error: %v", err)
			} else {
				log.Println("Scheduled maintenance completed")
			}
		}
	}
}

// MockCacheService is a simple mock implementation for demonstration
type MockCacheService struct{}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	return []byte("mock_value"), nil
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *MockCacheService) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *MockCacheService) Exists(ctx context.Context, key string) bool {
	return true
}

// MockEmailService is a simple mock implementation for demonstration
type MockEmailService struct{}

func (m *MockEmailService) SendEmail(ctx context.Context, to, subject, textBody, htmlBody string) error {
	log.Printf("Mock email sent to %s: %s", to, subject)
	return nil
}