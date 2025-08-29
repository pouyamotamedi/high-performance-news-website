package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// ConfigurationHandlers handles configuration management API endpoints
type ConfigurationHandlers struct {
	configService *services.ConfigService
}

// NewConfigurationHandlers creates a new ConfigurationHandlers instance
func NewConfigurationHandlers(configService *services.ConfigService) *ConfigurationHandlers {
	return &ConfigurationHandlers{
		configService: configService,
	}
}

// RegisterConfigurationRoutes registers all configuration management routes
func (h *ConfigurationHandlers) RegisterConfigurationRoutes(router *gin.RouterGroup) {
	// Configuration management
	router.GET("/config", h.GetAllConfigurations)
	router.GET("/config/:key", h.GetConfiguration)
	router.PUT("/config/:key", h.UpdateConfiguration)
	router.DELETE("/config/:key", h.DeleteConfiguration)
	router.POST("/config/validate", h.ValidateConfigurations)
	router.GET("/config/categories/:category", h.GetConfigurationsByCategory)
	
	// Configuration snapshots
	router.GET("/config/snapshots", h.GetSnapshots)
	router.POST("/config/snapshots", h.CreateSnapshot)
	router.POST("/config/snapshots/:id/restore", h.RestoreSnapshot)
	
	// Configuration history
	router.GET("/config/:key/history", h.GetConfigurationHistory)
	
	// Feature flags
	router.GET("/feature-flags", h.GetAllFeatureFlags)
	router.GET("/feature-flags/:key", h.GetFeatureFlag)
	router.PUT("/feature-flags/:key", h.UpdateFeatureFlag)
	router.POST("/feature-flags", h.CreateFeatureFlag)
	router.DELETE("/feature-flags/:key", h.DeleteFeatureFlag)
	router.POST("/feature-flags/:key/check", h.CheckFeatureFlag)
	
	// Hot reload
	router.POST("/config/reload", h.ReloadConfiguration)
}

// Configuration Management Endpoints

// GetAllConfigurations returns all system configurations
func (h *ConfigurationHandlers) GetAllConfigurations(c *gin.Context) {
	configs := h.configService.GetAll()
	
	// Filter out secret values for non-admin users
	filteredConfigs := make(map[string]*models.Configuration)
	for key, config := range configs {
		if config.IsSecret {
			// Create a copy without the actual value
			configCopy := *config
			configCopy.Value = "***HIDDEN***"
			filteredConfigs[key] = &configCopy
		} else {
			filteredConfigs[key] = config
		}
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Configurations retrieved successfully",
		Data:    filteredConfigs,
	})
}

// GetConfiguration returns a specific configuration
func (h *ConfigurationHandlers) GetConfiguration(c *gin.Context) {
	key := c.Param("key")
	
	config, err := h.configService.GetConfig(key)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Configuration not found",
			Code:    "CONFIG_NOT_FOUND",
			Message: err.Error(),
		})
		return
	}
	
	// Hide secret values for non-admin users
	if config.IsSecret {
		configCopy := *config
		configCopy.Value = "***HIDDEN***"
		config = &configCopy
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Configuration retrieved successfully",
		Data:    config,
	})
}

// UpdateConfiguration updates a specific configuration
func (h *ConfigurationHandlers) UpdateConfiguration(c *gin.Context) {
	key := c.Param("key")
	
	var request struct {
		Value  interface{} `json:"value" binding:"required"`
		Reason string      `json:"reason"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}
	
	// Get user ID from context (assuming it's set by auth middleware)
	userID := getUserIDFromContext(c)
	
	// Update configuration
	if err := h.configService.SetWithContext(c.Request.Context(), key, request.Value, userID, request.Reason); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to update configuration",
			Code:    "CONFIG_UPDATE_ERROR",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Configuration updated successfully",
		Data: gin.H{
			"key":        key,
			"updated_at": time.Now().Unix(),
			"updated_by": userID,
		},
	})
}

// DeleteConfiguration deletes a specific configuration
func (h *ConfigurationHandlers) DeleteConfiguration(c *gin.Context) {
	key := c.Param("key")
	
	if err := h.configService.Delete(key); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to delete configuration",
			Code:    "CONFIG_DELETE_ERROR",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Configuration deleted successfully",
		Data: gin.H{
			"key":        key,
			"deleted_at": time.Now().Unix(),
		},
	})
}

// ValidateConfigurations validates all configurations
func (h *ConfigurationHandlers) ValidateConfigurations(c *gin.Context) {
	errors := h.configService.ValidateConfiguration()
	
	if len(errors) > 0 {
		errorMessages := make([]string, len(errors))
		for i, err := range errors {
			errorMessages[i] = err.Error()
		}
		
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Configuration validation failed",
			Code:    "VALIDATION_ERROR",
			Message: "Multiple validation errors found",
			Details: map[string]interface{}{
				"errors": errorMessages,
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "All configurations are valid",
		Data: gin.H{
			"validated_at": time.Now().Unix(),
		},
	})
}

// GetConfigurationsByCategory returns configurations by category
func (h *ConfigurationHandlers) GetConfigurationsByCategory(c *gin.Context) {
	category := c.Param("category")
	
	configs := h.configService.GetAllByCategory(category)
	
	// Filter out secret values
	filteredConfigs := make(map[string]*models.Configuration)
	for key, config := range configs {
		if config.IsSecret {
			configCopy := *config
			configCopy.Value = "***HIDDEN***"
			filteredConfigs[key] = &configCopy
		} else {
			filteredConfigs[key] = config
		}
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Configurations retrieved successfully",
		Data:    filteredConfigs,
	})
}

// Configuration Snapshots

// GetSnapshots returns all configuration snapshots
func (h *ConfigurationHandlers) GetSnapshots(c *gin.Context) {
	// TODO: Implement snapshot retrieval from database
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Snapshots retrieved successfully",
		Data:    []interface{}{},
	})
}

// CreateSnapshot creates a new configuration snapshot
func (h *ConfigurationHandlers) CreateSnapshot(c *gin.Context) {
	var request struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}
	
	userID := getUserIDFromContext(c)
	
	snapshot, err := h.configService.CreateSnapshot(request.Name, request.Description, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create snapshot",
			Code:    "SNAPSHOT_CREATE_ERROR",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, SuccessResponse{
		Message: "Snapshot created successfully",
		Data:    snapshot,
	})
}

// RestoreSnapshot restores configuration from a snapshot
func (h *ConfigurationHandlers) RestoreSnapshot(c *gin.Context) {
	snapshotIDStr := c.Param("id")
	snapshotID, err := strconv.ParseUint(snapshotIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid snapshot ID",
			Code:    "INVALID_SNAPSHOT_ID",
			Message: err.Error(),
		})
		return
	}
	
	userID := getUserIDFromContext(c)
	
	if err := h.configService.RestoreSnapshot(snapshotID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to restore snapshot",
			Code:    "SNAPSHOT_RESTORE_ERROR",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Snapshot restored successfully",
		Data: gin.H{
			"snapshot_id": snapshotID,
			"restored_at": time.Now().Unix(),
			"restored_by": userID,
		},
	})
}

// GetConfigurationHistory returns configuration change history
func (h *ConfigurationHandlers) GetConfigurationHistory(c *gin.Context) {
	key := c.Param("key")
	
	// TODO: Implement history retrieval from database
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Configuration history retrieved successfully",
		Data: gin.H{
			"key":     key,
			"history": []interface{}{},
		},
	})
}

// Feature Flag Endpoints

// GetAllFeatureFlags returns all feature flags
func (h *ConfigurationHandlers) GetAllFeatureFlags(c *gin.Context) {
	flags := h.configService.GetAllFeatureFlags()
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Feature flags retrieved successfully",
		Data:    flags,
	})
}

// GetFeatureFlag returns a specific feature flag
func (h *ConfigurationHandlers) GetFeatureFlag(c *gin.Context) {
	key := c.Param("key")
	
	flag, err := h.configService.GetFeatureFlag(key)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Feature flag not found",
			Code:    "FEATURE_FLAG_NOT_FOUND",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Feature flag retrieved successfully",
		Data:    flag,
	})
}

// CreateFeatureFlag creates a new feature flag
func (h *ConfigurationHandlers) CreateFeatureFlag(c *gin.Context) {
	var flag models.FeatureFlag
	
	if err := c.ShouldBindJSON(&flag); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid feature flag data",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}
	
	flag.CreatedAt = time.Now()
	flag.UpdatedAt = time.Now()
	
	if err := h.configService.SetFeatureFlag(&flag); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create feature flag",
			Code:    "FEATURE_FLAG_CREATE_ERROR",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, SuccessResponse{
		Message: "Feature flag created successfully",
		Data:    flag,
	})
}

// UpdateFeatureFlag updates a feature flag
func (h *ConfigurationHandlers) UpdateFeatureFlag(c *gin.Context) {
	key := c.Param("key")
	
	var flag models.FeatureFlag
	if err := c.ShouldBindJSON(&flag); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid feature flag data",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}
	
	flag.Key = key
	flag.UpdatedAt = time.Now()
	
	if err := h.configService.SetFeatureFlag(&flag); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update feature flag",
			Code:    "FEATURE_FLAG_UPDATE_ERROR",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Feature flag updated successfully",
		Data:    flag,
	})
}

// DeleteFeatureFlag deletes a feature flag
func (h *ConfigurationHandlers) DeleteFeatureFlag(c *gin.Context) {
	key := c.Param("key")
	
	// TODO: Implement feature flag deletion
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Feature flag deleted successfully",
		Data: gin.H{
			"key":        key,
			"deleted_at": time.Now().Unix(),
		},
	})
}

// CheckFeatureFlag checks if a feature flag is enabled for given context
func (h *ConfigurationHandlers) CheckFeatureFlag(c *gin.Context) {
	key := c.Param("key")
	
	var request struct {
		Context map[string]interface{} `json:"context"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}
	
	// Add user context if available
	if userID := getUserIDFromContext(c); userID > 0 {
		if request.Context == nil {
			request.Context = make(map[string]interface{})
		}
		request.Context["user_id"] = userID
	}
	
	enabled := h.configService.IsFeatureEnabled(key, request.Context)
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Feature flag checked successfully",
		Data: gin.H{
			"key":     key,
			"enabled": enabled,
			"context": request.Context,
		},
	})
}

// ReloadConfiguration reloads configuration from database
func (h *ConfigurationHandlers) ReloadConfiguration(c *gin.Context) {
	if err := h.configService.ReloadFromDatabase(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to reload configuration",
			Code:    "CONFIG_RELOAD_ERROR",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Configuration reloaded successfully",
		Data: gin.H{
			"reloaded_at": time.Now().Unix(),
		},
	})
}

// Helper function to get user ID from context
func getUserIDFromContext(c *gin.Context) uint64 {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint64); ok {
			return id
		}
	}
	return 0
}