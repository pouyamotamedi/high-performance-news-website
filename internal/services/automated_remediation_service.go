package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/cache"
)

// RemediationAction represents an automated remediation action
type RemediationAction struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	AlertName       string                 `json:"alert_name"`
	Component       string                 `json:"component"`
	ActionType      string                 `json:"action_type"`
	Parameters      map[string]interface{} `json:"parameters"`
	Enabled         bool                   `json:"enabled"`
	MaxRetries      int                    `json:"max_retries"`
	CooldownMinutes int                    `json:"cooldown_minutes"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// RemediationExecution represents an execution of a remediation action
type RemediationExecution struct {
	ID          uint64                 `json:"id"`
	ActionID    string                 `json:"action_id"`
	AlertName   string                 `json:"alert_name"`
	Status      string                 `json:"status"` // pending, running, completed, failed, skipped
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at"`
	Output      string                 `json:"output"`
	Error       string                 `json:"error"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

// AutomatedRemediationService handles automated remediation actions
type AutomatedRemediationService struct {
	db             *sql.DB
	cache          cache.CacheService
	config         *config.MonitoringConfig
	metricsService *MetricsService
	alertingService *AlertingService
	
	// Remediation actions registry
	actions map[string]*RemediationAction
	mutex   sync.RWMutex
	
	// Execution tracking
	lastExecution map[string]time.Time
	execMutex     sync.RWMutex
	
	// Control channels
	stopChannel chan struct{}
	isRunning   bool
}

// NewAutomatedRemediationService creates a new AutomatedRemediationService
func NewAutomatedRemediationService(
	db *sql.DB,
	cache cache.CacheService,
	config *config.MonitoringConfig,
	metricsService *MetricsService,
	alertingService *AlertingService,
) *AutomatedRemediationService {
	service := &AutomatedRemediationService{
		db:              db,
		cache:           cache,
		config:          config,
		metricsService:  metricsService,
		alertingService: alertingService,
		actions:         make(map[string]*RemediationAction),
		lastExecution:   make(map[string]time.Time),
		stopChannel:     make(chan struct{}),
	}
	
	// Initialize default remediation actions
	service.initializeDefaultActions()
	
	return service
}

// StartRemediationService starts the remediation service
func (ars *AutomatedRemediationService) StartRemediationService(ctx context.Context) {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()
	
	if ars.isRunning {
		return
	}
	
	log.Println("Starting automated remediation service...")
	ars.isRunning = true
	
	// Load remediation actions from database
	go ars.loadRemediationActions()
	
	log.Println("Automated remediation service started")
}

// StopRemediationService stops the remediation service
func (ars *AutomatedRemediationService) StopRemediationService() {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()
	
	if !ars.isRunning {
		return
	}
	
	log.Println("Stopping automated remediation service...")
	close(ars.stopChannel)
	ars.isRunning = false
	log.Println("Automated remediation service stopped")
}

// ProcessAlert processes an alert and executes appropriate remediation actions
func (ars *AutomatedRemediationService) ProcessAlert(alert *models.Alert) error {
	ars.mutex.RLock()
	defer ars.mutex.RUnlock()
	
	// Find matching remediation actions
	var matchingActions []*RemediationAction
	for _, action := range ars.actions {
		if action.Enabled && ars.matchesAlert(action, alert) {
			matchingActions = append(matchingActions, action)
		}
	}
	
	// Execute matching actions
	for _, action := range matchingActions {
		if ars.canExecute(action) {
			go ars.executeAction(action, alert)
		}
	}
	
	return nil
}

// matchesAlert checks if a remediation action matches an alert
func (ars *AutomatedRemediationService) matchesAlert(action *RemediationAction, alert *models.Alert) bool {
	// Check alert name match
	if action.AlertName != "" && action.AlertName != alert.Name {
		return false
	}
	
	// Check component match
	if action.Component != "" && action.Component != alert.Component {
		return false
	}
	
	return true
}

// canExecute checks if an action can be executed (cooldown check)
func (ars *AutomatedRemediationService) canExecute(action *RemediationAction) bool {
	ars.execMutex.RLock()
	defer ars.execMutex.RUnlock()
	
	lastExec, exists := ars.lastExecution[action.ID]
	if !exists {
		return true
	}
	
	cooldown := time.Duration(action.CooldownMinutes) * time.Minute
	return time.Since(lastExec) >= cooldown
}

// executeAction executes a remediation action
func (ars *AutomatedRemediationService) executeAction(action *RemediationAction, alert *models.Alert) {
	log.Printf("Executing remediation action: %s for alert: %s", action.Name, alert.Name)
	
	execution := &RemediationExecution{
		ActionID:  action.ID,
		AlertName: alert.Name,
		Status:    "running",
		StartedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
	
	// Save execution record
	if err := ars.saveExecution(execution); err != nil {
		log.Printf("Error saving remediation execution: %v", err)
		return
	}
	
	// Update last execution time
	ars.execMutex.Lock()
	ars.lastExecution[action.ID] = time.Now()
	ars.execMutex.Unlock()
	
	// Execute the action
	var err error
	switch action.ActionType {
	case "clear_cache":
		err = ars.executeClearCache(action, alert, execution)
	case "disk_cleanup":
		err = ars.executeDiskCleanup(action, alert, execution)
	case "restart_service":
		err = ars.executeRestartService(action, alert, execution)
	case "kill_process":
		err = ars.executeKillProcess(action, alert, execution)
	case "reset_connections":
		err = ars.executeResetConnections(action, alert, execution)
	default:
		err = fmt.Errorf("unknown action type: %s", action.ActionType)
	}
	
	// Update execution status
	now := time.Now()
	execution.CompletedAt = &now
	if err != nil {
		execution.Status = "failed"
		execution.Error = err.Error()
		log.Printf("Remediation action failed: %s - %v", action.Name, err)
	} else {
		execution.Status = "completed"
		log.Printf("Remediation action completed: %s", action.Name)
	}
	
	// Update execution record
	ars.updateExecution(execution)
}

// executeClearCache clears application cache
func (ars *AutomatedRemediationService) executeClearCache(action *RemediationAction, alert *models.Alert, execution *RemediationExecution) error {
	if ars.cache == nil {
		return fmt.Errorf("cache service not available")
	}
	
	cacheType := "all"
	if ct, ok := action.Parameters["cache_type"].(string); ok {
		cacheType = ct
	}
	
	ctx := context.Background()
	switch cacheType {
	case "all":
		// Clear all cache patterns
		patterns := []string{"article:*", "homepage:*", "category:*", "tag:*", "rss:*"}
		for _, pattern := range patterns {
			if err := ars.cache.DeletePattern(ctx, pattern); err != nil {
				return fmt.Errorf("failed to clear cache pattern %s: %v", pattern, err)
			}
		}
	default:
		if err := ars.cache.DeletePattern(ctx, cacheType+":*"); err != nil {
			return fmt.Errorf("failed to clear cache type %s: %v", cacheType, err)
		}
	}
	
	execution.Output = fmt.Sprintf("Cleared cache type: %s", cacheType)
	return nil
}

// executeDiskCleanup performs disk cleanup
func (ars *AutomatedRemediationService) executeDiskCleanup(action *RemediationAction, alert *models.Alert, execution *RemediationExecution) error {
	var commands []string
	
	// Default cleanup commands
	if paths, ok := action.Parameters["cleanup_paths"].([]interface{}); ok {
		for _, path := range paths {
			if pathStr, ok := path.(string); ok {
				commands = append(commands, fmt.Sprintf("find %s -type f -mtime +7 -delete", pathStr))
			}
		}
	} else {
		// Default cleanup paths
		commands = []string{
			"find /tmp -type f -mtime +7 -delete",
			"find /var/log -name '*.log.*' -mtime +30 -delete",
			"find /var/cache -type f -mtime +7 -delete",
		}
	}
	
	var output []string
	for _, cmd := range commands {
		result, err := exec.Command("sh", "-c", cmd).CombinedOutput()
		if err != nil {
			return fmt.Errorf("cleanup command failed: %s - %v", cmd, err)
		}
		output = append(output, fmt.Sprintf("Executed: %s\nOutput: %s", cmd, string(result)))
	}
	
	execution.Output = fmt.Sprintf("Disk cleanup completed:\n%s", fmt.Sprintf("%v", output))
	return nil
}

// executeRestartService restarts a system service
func (ars *AutomatedRemediationService) executeRestartService(action *RemediationAction, alert *models.Alert, execution *RemediationExecution) error {
	serviceName, ok := action.Parameters["service_name"].(string)
	if !ok {
		return fmt.Errorf("service_name parameter required")
	}
	
	// Safety check - only allow specific services
	allowedServices := map[string]bool{
		"nginx":      true,
		"postgresql": true,
		"dragonfly":  true,
	}
	
	if !allowedServices[serviceName] {
		return fmt.Errorf("service restart not allowed for: %s", serviceName)
	}
	
	cmd := exec.Command("systemctl", "restart", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart service %s: %v - %s", serviceName, err, string(output))
	}
	
	execution.Output = fmt.Sprintf("Restarted service: %s\nOutput: %s", serviceName, string(output))
	return nil
}

// executeKillProcess kills a specific process
func (ars *AutomatedRemediationService) executeKillProcess(action *RemediationAction, alert *models.Alert, execution *RemediationExecution) error {
	processName, ok := action.Parameters["process_name"].(string)
	if !ok {
		return fmt.Errorf("process_name parameter required")
	}
	
	// Safety check - find and kill process
	cmd := exec.Command("pkill", "-f", processName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to kill process %s: %v - %s", processName, err, string(output))
	}
	
	execution.Output = fmt.Sprintf("Killed process: %s\nOutput: %s", processName, string(output))
	return nil
}

// executeResetConnections resets database connections
func (ars *AutomatedRemediationService) executeResetConnections(action *RemediationAction, alert *models.Alert, execution *RemediationExecution) error {
	if ars.db == nil {
		return fmt.Errorf("database not available")
	}
	
	// Kill idle connections
	query := `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE state = 'idle'
		AND state_change < NOW() - INTERVAL '10 minutes'
		AND pid <> pg_backend_pid()
	`
	
	result, err := ars.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to reset connections: %v", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	execution.Output = fmt.Sprintf("Reset %d idle database connections", rowsAffected)
	return nil
}

// initializeDefaultActions sets up default remediation actions
func (ars *AutomatedRemediationService) initializeDefaultActions() {
	defaultActions := []*RemediationAction{
		{
			ID:              "clear_cache_high_memory",
			Name:            "Clear Cache on High Memory",
			Description:     "Clear application cache when memory usage is high",
			AlertName:       "high_memory_usage",
			Component:       "system",
			ActionType:      "clear_cache",
			Parameters:      map[string]interface{}{"cache_type": "all"},
			Enabled:         true,
			MaxRetries:      3,
			CooldownMinutes: 15,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "disk_cleanup_high_disk",
			Name:            "Disk Cleanup on High Disk Usage",
			Description:     "Clean up temporary files when disk usage is high",
			AlertName:       "high_disk_usage",
			Component:       "system",
			ActionType:      "disk_cleanup",
			Parameters:      map[string]interface{}{},
			Enabled:         true,
			MaxRetries:      3,
			CooldownMinutes: 30,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "reset_db_connections",
			Name:            "Reset Database Connections",
			Description:     "Reset idle database connections when connection limit is reached",
			AlertName:       "high_db_connections",
			Component:       "database",
			ActionType:      "reset_connections",
			Parameters:      map[string]interface{}{},
			Enabled:         true,
			MaxRetries:      3,
			CooldownMinutes: 10,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}
	
	for _, action := range defaultActions {
		ars.actions[action.ID] = action
	}
}

// loadRemediationActions loads remediation actions from database
func (ars *AutomatedRemediationService) loadRemediationActions() {
	if ars.db == nil {
		return
	}
	
	query := `
		SELECT id, name, description, alert_name, component, action_type,
		       parameters, enabled, max_retries, cooldown_minutes,
		       created_at, updated_at
		FROM remediation_actions
		WHERE enabled = true
	`
	
	rows, err := ars.db.Query(query)
	if err != nil {
		log.Printf("Error loading remediation actions: %v", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		action := &RemediationAction{}
		var parametersJSON []byte
		
		err := rows.Scan(
			&action.ID, &action.Name, &action.Description,
			&action.AlertName, &action.Component, &action.ActionType,
			&parametersJSON, &action.Enabled, &action.MaxRetries,
			&action.CooldownMinutes, &action.CreatedAt, &action.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning remediation action: %v", err)
			continue
		}
		
		// Parse parameters JSON
		if len(parametersJSON) > 0 {
			if err := json.Unmarshal(parametersJSON, &action.Parameters); err != nil {
				log.Printf("Error parsing parameters for action %s: %v", action.ID, err)
				action.Parameters = make(map[string]interface{})
			}
		} else {
			action.Parameters = make(map[string]interface{})
		}
		
		ars.mutex.Lock()
		ars.actions[action.ID] = action
		ars.mutex.Unlock()
	}
	
	log.Printf("Loaded %d remediation actions from database", len(ars.actions))
}

// saveExecution saves a remediation execution to database
func (ars *AutomatedRemediationService) saveExecution(execution *RemediationExecution) error {
	if ars.db == nil {
		return nil
	}
	
	metadataJSON, _ := json.Marshal(execution.Metadata)
	
	query := `
		INSERT INTO remediation_executions 
		(action_id, alert_name, status, started_at, output, error, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	
	err := ars.db.QueryRow(
		query, execution.ActionID, execution.AlertName, execution.Status,
		execution.StartedAt, execution.Output, execution.Error,
		metadataJSON, execution.CreatedAt,
	).Scan(&execution.ID)
	
	return err
}

// updateExecution updates a remediation execution in database
func (ars *AutomatedRemediationService) updateExecution(execution *RemediationExecution) error {
	if ars.db == nil {
		return nil
	}
	
	metadataJSON, _ := json.Marshal(execution.Metadata)
	
	query := `
		UPDATE remediation_executions 
		SET status = $1, completed_at = $2, output = $3, error = $4, metadata = $5
		WHERE id = $6
	`
	
	_, err := ars.db.Exec(
		query, execution.Status, execution.CompletedAt, execution.Output,
		execution.Error, metadataJSON, execution.ID,
	)
	
	return err
}

// GetRemediationActions returns all remediation actions
func (ars *AutomatedRemediationService) GetRemediationActions() map[string]*RemediationAction {
	ars.mutex.RLock()
	defer ars.mutex.RUnlock()
	
	actions := make(map[string]*RemediationAction)
	for k, v := range ars.actions {
		actions[k] = v
	}
	
	return actions
}

// GetRemediationExecutions returns recent remediation executions
func (ars *AutomatedRemediationService) GetRemediationExecutions(limit int) ([]*RemediationExecution, error) {
	if ars.db == nil {
		return []*RemediationExecution{}, nil
	}
	
	query := `
		SELECT id, action_id, alert_name, status, started_at, completed_at,
		       output, error, metadata, created_at
		FROM remediation_executions
		ORDER BY created_at DESC
		LIMIT $1
	`
	
	rows, err := ars.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var executions []*RemediationExecution
	for rows.Next() {
		execution := &RemediationExecution{}
		var metadataJSON []byte
		
		err := rows.Scan(
			&execution.ID, &execution.ActionID, &execution.AlertName,
			&execution.Status, &execution.StartedAt, &execution.CompletedAt,
			&execution.Output, &execution.Error, &metadataJSON,
			&execution.CreatedAt,
		)
		if err != nil {
			continue
		}
		
		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &execution.Metadata)
		} else {
			execution.Metadata = make(map[string]interface{})
		}
		
		executions = append(executions, execution)
	}
	
	return executions, nil
}