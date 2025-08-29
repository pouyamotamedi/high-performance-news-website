package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

type BackupHandlers struct {
	backupService services.BackupServiceInterface
}

func NewBackupHandlers(backupService services.BackupServiceInterface) *BackupHandlers {
	return &BackupHandlers{
		backupService: backupService,
	}
}

// RegisterBackupRoutes registers all backup-related routes
func (h *BackupHandlers) RegisterRoutes(router *gin.RouterGroup) {
	backup := router.Group("/backup")
	{
		// Backup operations
		backup.POST("/create", h.CreateBackup)
		backup.POST("/full", h.CreateFullBackup)
		backup.POST("/incremental", h.CreateIncrementalBackup)
		backup.GET("/list", h.ListBackups)
		backup.GET("/:id", h.GetBackup)
		backup.DELETE("/:id", h.DeleteBackup)
		
		// Restore operations
		backup.POST("/restore", h.RestoreBackup)
		backup.POST("/restore/point-in-time", h.RestoreToPointInTime)
		backup.GET("/restore/list", h.ListRestoreOperations)
		backup.GET("/restore/:id", h.GetRestoreOperation)
		backup.GET("/recovery-points", h.GetAvailableRecoveryPoints)
		
		// Validation and testing
		backup.POST("/:id/validate", h.ValidateBackup)
		backup.POST("/dr-test", h.RunDisasterRecoveryTest)
		backup.GET("/dr-test/list", h.ListDRTests)
		backup.GET("/dr-test/:id", h.GetDRTestResults)
		
		// Replication
		backup.POST("/:id/replicate", h.ReplicateBackup)
		backup.GET("/:id/replication-status", h.GetReplicationStatus)
		
		// Metrics and monitoring
		backup.GET("/metrics", h.GetBackupMetrics)
		backup.GET("/health", h.GetBackupHealth)
		
		// Maintenance
		backup.POST("/cleanup", h.CleanupOldBackups)
		backup.POST("/archive", h.ArchiveOldBackups)
		
		// Scheduler
		backup.POST("/scheduler/start", h.StartBackupScheduler)
		backup.POST("/scheduler/stop", h.StopBackupScheduler)
		backup.GET("/scheduler/status", h.GetSchedulerStatus)
	}
}

// Backup operations
func (h *BackupHandlers) CreateBackup(c *gin.Context) {
	var request models.BackupRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}
	
	backup, err := h.backupService.CreateBackup(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backup", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"backup": backup})
}

func (h *BackupHandlers) CreateFullBackup(c *gin.Context) {
	backup, err := h.backupService.CreateFullBackup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create full backup", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"backup": backup})
}

func (h *BackupHandlers) CreateIncrementalBackup(c *gin.Context) {
	backup, err := h.backupService.CreateIncrementalBackup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create incremental backup", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"backup": backup})
}

func (h *BackupHandlers) GetBackup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup ID"})
		return
	}
	
	backup, err := h.backupService.GetBackup(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backup not found", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"backup": backup})
}

func (h *BackupHandlers) ListBackups(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	
	if limit > 100 {
		limit = 100
	}
	
	backups, err := h.backupService.ListBackups(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list backups", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"backups": backups,
		"limit":   limit,
		"offset":  offset,
	})
}

func (h *BackupHandlers) DeleteBackup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup ID"})
		return
	}
	
	if err := h.backupService.DeleteBackup(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete backup", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Backup deleted successfully"})
}

// Restore operations
func (h *BackupHandlers) RestoreBackup(c *gin.Context) {
	var request models.RestoreRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}
	
	restore, err := h.backupService.RestoreBackup(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start restore operation", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"restore_operation": restore})
}

func (h *BackupHandlers) RestoreToPointInTime(c *gin.Context) {
	var request struct {
		TargetTime     string `json:"target_time" binding:"required"`
		TargetDatabase string `json:"target_database" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}
	
	targetTime, err := time.Parse(time.RFC3339, request.TargetTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target time format. Use RFC3339 format"})
		return
	}
	
	restore, err := h.backupService.RestoreToPointInTime(targetTime, request.TargetDatabase)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start point-in-time restore", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"restore_operation": restore})
}

func (h *BackupHandlers) GetRestoreOperation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid restore operation ID"})
		return
	}
	
	restore, err := h.backupService.GetRestoreOperation(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restore operation not found", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"restore_operation": restore})
}

func (h *BackupHandlers) ListRestoreOperations(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	
	if limit > 100 {
		limit = 100
	}
	
	operations, err := h.backupService.ListRestoreOperations(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list restore operations", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"restore_operations": operations,
		"limit":              limit,
		"offset":             offset,
	})
}

func (h *BackupHandlers) GetAvailableRecoveryPoints(c *gin.Context) {
	recoveryPoints, err := h.backupService.GetAvailableRecoveryPoints()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recovery points", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"recovery_points": recoveryPoints})
}

// Validation and testing
func (h *BackupHandlers) ValidateBackup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup ID"})
		return
	}
	
	validation, err := h.backupService.ValidateBackup(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start backup validation", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"validation": validation})
}

func (h *BackupHandlers) RunDisasterRecoveryTest(c *gin.Context) {
	var request struct {
		TestName string `json:"test_name" binding:"required"`
		BackupID uint64 `json:"backup_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}
	
	test, err := h.backupService.RunDisasterRecoveryTest(request.TestName, request.BackupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start DR test", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"dr_test": test})
}

func (h *BackupHandlers) GetDRTestResults(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DR test ID"})
		return
	}
	
	test, err := h.backupService.GetDRTestResults(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DR test not found", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"dr_test": test})
}

func (h *BackupHandlers) ListDRTests(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	
	if limit > 100 {
		limit = 100
	}
	
	tests, err := h.backupService.ListDRTests(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list DR tests", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"dr_tests": tests,
		"limit":    limit,
		"offset":   offset,
	})
}

// Replication
func (h *BackupHandlers) ReplicateBackup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup ID"})
		return
	}
	
	var request struct {
		TargetName string `json:"target_name" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}
	
	replication, err := h.backupService.ReplicateBackup(id, request.TargetName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start replication", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"replication": replication})
}

func (h *BackupHandlers) GetReplicationStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup ID"})
		return
	}
	
	replications, err := h.backupService.GetReplicationStatus(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get replication status", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"replications": replications})
}

// Metrics and monitoring
func (h *BackupHandlers) GetBackupMetrics(c *gin.Context) {
	metrics, err := h.backupService.GetBackupMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get backup metrics", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

func (h *BackupHandlers) GetBackupHealth(c *gin.Context) {
	health, err := h.backupService.GetBackupHealth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get backup health", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"health": health})
}

// Maintenance
func (h *BackupHandlers) CleanupOldBackups(c *gin.Context) {
	if err := h.backupService.CleanupOldBackups(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup old backups", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Old backups cleaned up successfully"})
}

func (h *BackupHandlers) ArchiveOldBackups(c *gin.Context) {
	var request struct {
		OlderThanDays int `json:"older_than_days" binding:"required,min=1"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}
	
	olderThan := time.Now().AddDate(0, 0, -request.OlderThanDays)
	if err := h.backupService.ArchiveOldBackups(olderThan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive old backups", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Old backups archived successfully"})
}

// Scheduler
func (h *BackupHandlers) StartBackupScheduler(c *gin.Context) {
	if err := h.backupService.StartBackupScheduler(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start backup scheduler", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Backup scheduler started successfully"})
}

func (h *BackupHandlers) StopBackupScheduler(c *gin.Context) {
	if err := h.backupService.StopBackupScheduler(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop backup scheduler", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Backup scheduler stopped successfully"})
}

func (h *BackupHandlers) GetSchedulerStatus(c *gin.Context) {
	status, err := h.backupService.GetSchedulerStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get scheduler status", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"scheduler_status": status})
}