package testing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UnifiedConfigurationManager manages all testing system configurations
type UnifiedConfigurationManager struct {
	// Configuration storage
	config              *ComprehensiveTestingConfig
	configPath          string
	
	// Component configurations
	orchestratorConfig  *OrchestratorConfig
	environmentConfig   *EnvironmentConfig
	reliabilityConfig   *TestReliabilityConfig
	baselineConfig      *BaselineConfig
	aiConfig            *AITestConfig
	securityConfig      *SecurityConfig
	
	// Dynamic configuration
	dynamicSettings     map[string]interface{}
	configWatchers      []ConfigWatcher
	
	// State management
	mutex               sync.RWMutex
	lastUpdate          time.Time
	configVersion       string
	
	// Database for persistent configuration
	db                  *sql.DB
}

// ConfigWatcher defines interface for configuration change notifications
type ConfigWatcher interface {
	OnConfigChange(section string, oldValue, newValue interface{}) error
}

// ConfigurationValidation represents configuration validation results
type ConfigurationValidation struct {
	Valid       bool                    `json:"valid"`
	Errors      []ConfigValidationError `json:"errors"`
	Warnings    []ConfigValidationWarning `json:"warnings"`
	Suggestions []ConfigSuggestion      `json:"suggestions"`
}

// ConfigValidationError represents a configuration validation error
type ConfigValidationError struct {
	Section     string `json:"section"`
	Field       string `json:"field"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
}

// ConfigValidationWarning represents a configuration validation warning
type ConfigValidationWarning struct {
	Section     string `json:"section"`
	Field       string `json:"field"`
	Message     string `json:"message"`
	Impact      string `json:"impact"`
}

// ConfigSuggestion represents a configuration optimization suggestion
type ConfigSuggestion struct {
	Section     string `json:"section"`
	Field       string `json:"field"`
	Current     interface{} `json:"current"`
	Suggested   interface{} `json:"suggested"`
	Reason      string `json:"reason"`
	Impact      string `json:"impact"`
}

// ConfigurationTemplate represents a configuration template
type ConfigurationTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Config      *ComprehensiveTestingConfig `json:"config"`
	Metadata    TemplateMetadata       `json:"metadata"`
}

// TemplateMetadata contains template metadata
type TemplateMetadata struct {
	Version     string    `json:"version"`
	Author      string    `json:"author"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []string  `json:"tags"`
	UseCase     string    `json:"use_case"`
}

// NewUnifiedConfigurationManager creates a new unified configuration manager
func NewUnifiedConfigurationManager(db *sql.DB, configPath string) (*UnifiedConfigurationManager, error) {
	manager := &UnifiedConfigurationManager{
		db:              db,
		configPath:      configPath,
		dynamicSettings: make(map[string]interface{}),
		configWatchers:  []ConfigWatcher{},
		configVersion:   "1.0.0",
	}
	
	// Load configuration
	if err := manager.LoadConfiguration(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Initialize component configurations
	if err := manager.initializeComponentConfigurations(); err != nil {
		return nil, fmt.Errorf("failed to initialize component configurations: %w", err)
	}
	
	// Create configuration tables if they don't exist
	if err := manager.createConfigurationTables(); err != nil {
		return nil, fmt.Errorf("failed to create configuration tables: %w", err)
	}
	
	return manager, nil
}

// LoadConfiguration loads the comprehensive testing configuration
func (m *UnifiedConfigurationManager) LoadConfiguration() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if config file exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		log.Printf("Configuration file not found, creating default configuration: %s", m.configPath)
		m.config = m.createDefaultConfiguration()
		return m.saveConfiguration()
	}
	
	// Read configuration file
	data, err := ioutil.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}
	
	// Parse configuration
	var config ComprehensiveTestingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}
	
	m.config = &config
	m.lastUpdate = time.Now()
	
	log.Printf("Configuration loaded successfully from %s", m.configPath)
	return nil
}

// SaveConfiguration saves the current configuration
func (m *UnifiedConfigurationManager) SaveConfiguration() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	return m.saveConfiguration()
}

// saveConfiguration internal method to save configuration (requires lock)
func (m *UnifiedConfigurationManager) saveConfiguration() error {
	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create configuration directory: %w", err)
	}
	
	// Marshal configuration
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	
	// Write configuration file
	if err := ioutil.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}
	
	// Store in database
	if err := m.storeConfigurationInDB(); err != nil {
		log.Printf("Warning: Failed to store configuration in database: %v", err)
	}
	
	m.lastUpdate = time.Now()
	log.Printf("Configuration saved successfully to %s", m.configPath)
	
	return nil
}

// GetConfiguration returns the current configuration
func (m *UnifiedConfigurationManager) GetConfiguration() *ComprehensiveTestingConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Return a deep copy to prevent external modifications
	configJSON, _ := json.Marshal(m.config)
	var configCopy ComprehensiveTestingConfig
	json.Unmarshal(configJSON, &configCopy)
	
	return &configCopy
}

// UpdateConfiguration updates the configuration with validation
func (m *UnifiedConfigurationManager) UpdateConfiguration(newConfig *ComprehensiveTestingConfig) error {
	// Validate configuration
	validation := m.ValidateConfiguration(newConfig)
	if !validation.Valid {
		return fmt.Errorf("configuration validation failed: %v", validation.Errors)
	}
	
	m.mutex.Lock()
	oldConfig := m.config
	m.config = newConfig
	m.mutex.Unlock()
	
	// Notify watchers of changes
	if err := m.notifyConfigurationChange("all", oldConfig, newConfig); err != nil {
		log.Printf("Warning: Failed to notify configuration watchers: %v", err)
	}
	
	// Update component configurations
	if err := m.updateComponentConfigurations(); err != nil {
		log.Printf("Warning: Failed to update component configurations: %v", err)
	}
	
	// Save configuration
	if err := m.SaveConfiguration(); err != nil {
		return fmt.Errorf("failed to save updated configuration: %w", err)
	}
	
	log.Println("Configuration updated successfully")
	return nil
}

// ValidateConfiguration validates a configuration
func (m *UnifiedConfigurationManager) ValidateConfiguration(config *ComprehensiveTestingConfig) ConfigurationValidation {
	validation := ConfigurationValidation{
		Valid:       true,
		Errors:      []ConfigValidationError{},
		Warnings:    []ConfigValidationWarning{},
		Suggestions: []ConfigSuggestion{},
	}
	
	// Validate system configuration
	m.validateSystemConfig(&config.System, &validation)
	
	// Validate environment configuration
	m.validateEnvironmentConfig(&config.Environments, &validation)
	
	// Validate execution configuration
	m.validateExecutionConfig(&config.Execution, &validation)
	
	// Validate quality gates configuration
	m.validateQualityGatesConfig(&config.QualityGates, &validation)
	
	// Validate AI configuration
	m.validateAIConfig(&config.AI, &validation)
	
	// Validate monitoring configuration
	m.validateMonitoringConfig(&config.Monitoring, &validation)
	
	// Validate reporting configuration
	m.validateReportingConfig(&config.Reporting, &validation)
	
	// Validate security configuration
	m.validateSecurityConfig(&config.Security, &validation)
	
	// Validate performance configuration
	m.validatePerformanceConfig(&config.Performance, &validation)
	
	// Validate data management configuration
	m.validateDataManagementConfig(&config.DataManagement, &validation)
	
	// Generate optimization suggestions
	m.generateOptimizationSuggestions(config, &validation)
	
	// Set overall validation status
	validation.Valid = len(validation.Errors) == 0
	
	return validation
}

// GetOrchestratorConfig returns orchestrator-specific configuration
func (m *UnifiedConfigurationManager) GetOrchestratorConfig() *OrchestratorConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.orchestratorConfig
}

// GetEnvironmentConfig returns environment-specific configuration
func (m *UnifiedConfigurationManager) GetEnvironmentConfig() *EnvironmentConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return &m.config.Environments
}

// GetReliabilityConfig returns reliability-specific configuration
func (m *UnifiedConfigurationManager) GetReliabilityConfig() *TestReliabilityConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.reliabilityConfig
}

// GetBaselineConfig returns baseline-specific configuration
func (m *UnifiedConfigurationManager) GetBaselineConfig() *BaselineConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return &m.config.Performance.BaselineManagement
}

// GetAIConfig returns AI-specific configuration
func (m *UnifiedConfigurationManager) GetAIConfig() *AITestConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.aiConfig
}

// GetSecurityConfig returns security-specific configuration
func (m *UnifiedConfigurationManager) GetSecurityConfig() *SecurityConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.securityConfig
}

// RegisterConfigWatcher registers a configuration change watcher
func (m *UnifiedConfigurationManager) RegisterConfigWatcher(watcher ConfigWatcher) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.configWatchers = append(m.configWatchers, watcher)
}

// SetDynamicSetting sets a dynamic configuration setting
func (m *UnifiedConfigurationManager) SetDynamicSetting(key string, value interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	oldValue := m.dynamicSettings[key]
	m.dynamicSettings[key] = value
	
	// Notify watchers
	if err := m.notifyConfigurationChange("dynamic."+key, oldValue, value); err != nil {
		return fmt.Errorf("failed to notify configuration change: %w", err)
	}
	
	log.Printf("Dynamic setting updated: %s = %v", key, value)
	return nil
}

// GetDynamicSetting gets a dynamic configuration setting
func (m *UnifiedConfigurationManager) GetDynamicSetting(key string) (interface{}, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	value, exists := m.dynamicSettings[key]
	return value, exists
}

// CreateConfigurationTemplate creates a configuration template
func (m *UnifiedConfigurationManager) CreateConfigurationTemplate(name, description, category string) (*ConfigurationTemplate, error) {
	template := &ConfigurationTemplate{
		Name:        name,
		Description: description,
		Category:    category,
		Config:      m.GetConfiguration(),
		Metadata: TemplateMetadata{
			Version:   m.configVersion,
			Author:    "system",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      []string{category},
			UseCase:   description,
		},
	}
	
	// Store template in database
	if err := m.storeConfigurationTemplate(template); err != nil {
		return nil, fmt.Errorf("failed to store configuration template: %w", err)
	}
	
	log.Printf("Configuration template created: %s", name)
	return template, nil
}

// LoadConfigurationTemplate loads a configuration template
func (m *UnifiedConfigurationManager) LoadConfigurationTemplate(name string) (*ConfigurationTemplate, error) {
	template, err := m.loadConfigurationTemplate(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration template: %w", err)
	}
	
	return template, nil
}

// ApplyConfigurationTemplate applies a configuration template
func (m *UnifiedConfigurationManager) ApplyConfigurationTemplate(templateName string) error {
	template, err := m.LoadConfigurationTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}
	
	// Validate template configuration
	validation := m.ValidateConfiguration(template.Config)
	if !validation.Valid {
		return fmt.Errorf("template configuration is invalid: %v", validation.Errors)
	}
	
	// Apply template configuration
	if err := m.UpdateConfiguration(template.Config); err != nil {
		return fmt.Errorf("failed to apply template configuration: %w", err)
	}
	
	log.Printf("Configuration template applied: %s", templateName)
	return nil
}

// GetConfigurationHistory returns configuration change history
func (m *UnifiedConfigurationManager) GetConfigurationHistory(limit int) ([]ConfigurationHistoryEntry, error) {
	query := `
		SELECT id, config_version, config_data, created_at, created_by, change_description
		FROM configuration_history 
		ORDER BY created_at DESC 
		LIMIT $1
	`
	
	rows, err := m.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query configuration history: %w", err)
	}
	defer rows.Close()
	
	var history []ConfigurationHistoryEntry
	for rows.Next() {
		var entry ConfigurationHistoryEntry
		var configData string
		
		err := rows.Scan(&entry.ID, &entry.ConfigVersion, &configData, 
			&entry.CreatedAt, &entry.CreatedBy, &entry.ChangeDescription)
		if err != nil {
			continue
		}
		
		// Parse configuration data
		var config ComprehensiveTestingConfig
		if err := json.Unmarshal([]byte(configData), &config); err == nil {
			entry.Config = &config
		}
		
		history = append(history, entry)
	}
	
	return history, nil
}

// ExportConfiguration exports configuration to a file
func (m *UnifiedConfigurationManager) ExportConfiguration(exportPath string) error {
	config := m.GetConfiguration()
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	
	if err := ioutil.WriteFile(exportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}
	
	log.Printf("Configuration exported to: %s", exportPath)
	return nil
}

// ImportConfiguration imports configuration from a file
func (m *UnifiedConfigurationManager) ImportConfiguration(importPath string) error {
	data, err := ioutil.ReadFile(importPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}
	
	var config ComprehensiveTestingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse imported configuration: %w", err)
	}
	
	// Validate imported configuration
	validation := m.ValidateConfiguration(&config)
	if !validation.Valid {
		return fmt.Errorf("imported configuration is invalid: %v", validation.Errors)
	}
	
	// Apply imported configuration
	if err := m.UpdateConfiguration(&config); err != nil {
		return fmt.Errorf("failed to apply imported configuration: %w", err)
	}
	
	log.Printf("Configuration imported from: %s", importPath)
	return nil
}

// Private helper methods

func (m *UnifiedConfigurationManager) createDefaultConfiguration() *ComprehensiveTestingConfig {
	return &ComprehensiveTestingConfig{
		System: SystemConfig{
			MaxConcurrentEnvironments: 5,
			EnvironmentTimeout:        10 * time.Minute,
			MaxParallelTests:          10,
			TestTimeout:               30 * time.Minute,
			RetryAttempts:             3,
			LogLevel:                  "info",
			EnableDebugMode:           false,
		},
		Environments: EnvironmentConfig{
			DefaultResources: ResourceAllocation{
				Memory:   1024 * 1024 * 1024, // 1GB
				CPUQuota: 50000,               // 50% CPU
			},
			Docker: DockerConfig{
				Registry:       "docker.io",
				NetworkName:    "test-network",
				VolumePrefix:   "test-vol",
				EnableBuildKit: true,
				Images: map[string]string{
					"postgres":   "postgres:15-alpine",
					"redis":      "redis:7-alpine",
					"dragonfly":  "docker.dragonflydb.io/dragonflydb/dragonfly:latest",
				},
			},
			Cleanup: EnvironmentCleanupConfig{
				AutoCleanup:       true,
				CleanupInterval:   1 * time.Hour,
				MaxEnvironmentAge: 4 * time.Hour,
				OrphanedCleanup:   true,
			},
		},
		Execution: ExecutionConfig{
			DefaultPriority: PriorityMedium,
			ParallelExecution: ParallelExecutionConfig{
				Enabled:               true,
				MaxParallelGroups:     3,
				ResourceBalancing:     true,
				DependencyResolution:  true,
				LoadBalancingStrategy: "round_robin",
			},
			TestSelection: TestSelectionConfig{
				IntelligentSelection: true,
				RiskBasedSelection:   true,
				ChangeBasedSelection: true,
				PriorityFilters:      []string{"high", "critical"},
			},
			FailureHandling: FailureHandlingConfig{
				StopOnFirstFailure:  false,
				MaxFailureThreshold: 10,
				RetryStrategy:       "exponential_backoff",
				RetryDelay:          5 * time.Second,
				FlakyTestHandling:   "quarantine",
			},
			Optimization: ExecutionOptimizationConfig{
				Enabled:                 true,
				CacheTestResults:        true,
				PredictiveScheduling:    true,
				ResourceOptimization:    true,
				ExecutionTimeEstimation: true,
			},
		},
		QualityGates: QualityGatesConfig{
			Enabled: true,
			DefaultGates: []QualityGate{
				{
					Name:        "Code Coverage",
					Type:        "coverage",
					Threshold:   95.0,
					Operator:    ">=",
					Description: "Minimum code coverage threshold",
					Critical:    true,
				},
				{
					Name:        "Test Success Rate",
					Type:        "success_rate",
					Threshold:   95.0,
					Operator:    ">=",
					Description: "Minimum test success rate",
					Critical:    true,
				},
			},
			FailureAction: "block_deployment",
		},
		AI: AIConfig{
			Enabled:             true,
			Provider:            "openai",
			Model:               "gpt-4",
			MaxTokens:           2000,
			Temperature:         0.7,
			Timeout:             30 * time.Second,
			MaxRetries:          3,
			ConfidenceThreshold: 0.7,
			TestGeneration: AITestGenerationConfig{
				EdgeCaseGeneration:    true,
				FuzzingGeneration:     true,
				PerformanceGeneration: true,
				SecurityGeneration:    true,
				MaxTestsPerSuite:      50,
			},
			Analysis: AIAnalysisConfig{
				FailureAnalysis:      true,
				PerformanceAnalysis:  true,
				SecurityAnalysis:     true,
				TrendAnalysis:        true,
				RecommendationEngine: true,
			},
		},
		Monitoring: MonitoringConfig{
			Enabled:             true,
			HealthCheckInterval: 30 * time.Second,
			MetricsCollection: MetricsCollectionConfig{
				Enabled:            true,
				CollectionInterval: 60 * time.Second,
				RetentionPeriod:    30 * 24 * time.Hour,
			},
			Alerting: AlertingConfig{
				Enabled: true,
			},
			Dashboard: DashboardConfig{
				Enabled:         true,
				Port:            8080,
				RefreshInterval: 30 * time.Second,
			},
		},
		Reporting: ReportingConfig{
			Enabled:            true,
			GenerationInterval: 24 * time.Hour,
			RetentionPeriod:    30 * 24 * time.Hour,
			OutputFormats:      []string{"html", "json"},
			ReportTypes:        []string{"daily_summary", "quality_trends"},
		},
		Security: SecurityConfig{
			Enabled:   true,
			ScanTypes: []string{"sast", "dependency", "container"},
		},
		Performance: PerformanceConfig{
			Enabled: true,
			BaselineManagement: BaselineConfig{
				AutoEstablishment:   true,
				UpdateStrategy:      "adaptive",
				RetentionPeriod:     30 * 24 * time.Hour,
				ComparisonThreshold: 5.0,
			},
			LoadTesting: LoadTestingConfig{
				MaxVirtualUsers: 1000,
				TestDuration:    10 * time.Minute,
			},
			RegressionDetection: RegressionConfig{
				Enabled:           true,
				Threshold:         10.0,
				WindowSize:        10,
				StatisticalMethod: "t_test",
				AutoAlert:         true,
			},
		},
		DataManagement: DataManagementConfig{
			TestDataGeneration: TestDataGenerationConfig{
				Enabled:             true,
				MultilingualSupport: true,
				Languages:           []string{"en", "fa", "ar"},
				RealisticData:       true,
				Relationships:       true,
			},
			DataAnonymization: DataAnonymizationConfig{
				Enabled:           true,
				ValidationEnabled: true,
			},
			DataVersioning: DataVersioningConfig{
				Enabled:            true,
				VersioningStrategy: "semantic",
				AutoMigration:      true,
			},
			Cleanup: DataCleanupConfig{
				Enabled:             true,
				CleanupInterval:     4 * time.Hour,
				RetentionPeriod:     24 * time.Hour,
				OrphanedDataCleanup: true,
			},
		},
	}
}

func (m *UnifiedConfigurationManager) initializeComponentConfigurations() error {
	// Initialize orchestrator configuration
	m.orchestratorConfig = &OrchestratorConfig{
		MaxConcurrentEnvironments: m.config.System.MaxConcurrentEnvironments,
		EnvironmentTimeout:        m.config.System.EnvironmentTimeout,
		MaxParallelTests:          m.config.System.MaxParallelTests,
		TestTimeout:               m.config.System.TestTimeout,
		RetryAttempts:             m.config.System.RetryAttempts,
		MinCoverageThreshold:      95.0,
		MaxFlakinessThreshold:     5.0,
		PerformanceRegressionLimit: 10.0,
		EnableAITestGeneration:    m.config.AI.Enabled,
		AIConfidenceThreshold:     m.config.AI.ConfidenceThreshold,
		HealthCheckInterval:       m.config.Monitoring.HealthCheckInterval,
		MetricsCollectionInterval: m.config.Monitoring.MetricsCollection.CollectionInterval,
		ReportGenerationInterval:  m.config.Reporting.GenerationInterval,
		RetainReportsFor:          m.config.Reporting.RetentionPeriod,
	}
	
	// Initialize reliability configuration
	m.reliabilityConfig = &TestReliabilityConfig{
		FlakyThreshold:        0.3,
		MinExecutions:         5,
		WindowSize:            50,
		QuarantineThreshold:   0.5,
		ReintegrationCooldown: 24 * time.Hour,
		MaxQuarantineDuration: 7 * 24 * time.Hour,
		EnableAutoRemediation: true,
		NotificationEnabled:   true,
	}
	
	// Initialize AI configuration
	m.aiConfig = &AITestConfig{
		LLMProvider: m.config.AI.Provider,
		Model:       m.config.AI.Model,
		APIKey:      m.config.AI.APIKey,
		MaxTokens:   m.config.AI.MaxTokens,
		Temperature: m.config.AI.Temperature,
		Timeout:     m.config.AI.Timeout,
		MaxRetries:  m.config.AI.MaxRetries,
	}
	
	// Initialize security configuration
	m.securityConfig = &SecurityConfig{
		EnabledScanners: m.config.Security.ScanTypes,
		ScanTimeout:     10 * time.Minute,
		MaxConcurrentScans: 3,
		ReportFormat:    "json",
		AutoRemediation: false,
	}
	
	return nil
}

func (m *UnifiedConfigurationManager) updateComponentConfigurations() error {
	// Update orchestrator configuration
	m.orchestratorConfig.MaxConcurrentEnvironments = m.config.System.MaxConcurrentEnvironments
	m.orchestratorConfig.EnvironmentTimeout = m.config.System.EnvironmentTimeout
	m.orchestratorConfig.MaxParallelTests = m.config.System.MaxParallelTests
	m.orchestratorConfig.TestTimeout = m.config.System.TestTimeout
	m.orchestratorConfig.RetryAttempts = m.config.System.RetryAttempts
	m.orchestratorConfig.EnableAITestGeneration = m.config.AI.Enabled
	m.orchestratorConfig.AIConfidenceThreshold = m.config.AI.ConfidenceThreshold
	m.orchestratorConfig.HealthCheckInterval = m.config.Monitoring.HealthCheckInterval
	m.orchestratorConfig.MetricsCollectionInterval = m.config.Monitoring.MetricsCollection.CollectionInterval
	m.orchestratorConfig.ReportGenerationInterval = m.config.Reporting.GenerationInterval
	m.orchestratorConfig.RetainReportsFor = m.config.Reporting.RetentionPeriod
	
	// Update AI configuration
	m.aiConfig.LLMProvider = m.config.AI.Provider
	m.aiConfig.Model = m.config.AI.Model
	m.aiConfig.APIKey = m.config.AI.APIKey
	m.aiConfig.MaxTokens = m.config.AI.MaxTokens
	m.aiConfig.Temperature = m.config.AI.Temperature
	m.aiConfig.Timeout = m.config.AI.Timeout
	m.aiConfig.MaxRetries = m.config.AI.MaxRetries
	
	// Update security configuration
	m.securityConfig.EnabledScanners = m.config.Security.ScanTypes
	
	return nil
}

func (m *UnifiedConfigurationManager) notifyConfigurationChange(section string, oldValue, newValue interface{}) error {
	for _, watcher := range m.configWatchers {
		if err := watcher.OnConfigChange(section, oldValue, newValue); err != nil {
			log.Printf("Configuration watcher error: %v", err)
		}
	}
	return nil
}

// Validation methods for different configuration sections

func (m *UnifiedConfigurationManager) validateSystemConfig(config *SystemConfig, validation *ConfigurationValidation) {
	if config.MaxConcurrentEnvironments <= 0 {
		validation.Errors = append(validation.Errors, ConfigValidationError{
			Section:  "system",
			Field:    "max_concurrent_environments",
			Message:  "Must be greater than 0",
			Severity: "error",
		})
	}
	
	if config.MaxParallelTests <= 0 {
		validation.Errors = append(validation.Errors, ConfigValidationError{
			Section:  "system",
			Field:    "max_parallel_tests",
			Message:  "Must be greater than 0",
			Severity: "error",
		})
	}
	
	if config.TestTimeout <= 0 {
		validation.Errors = append(validation.Errors, ConfigValidationError{
			Section:  "system",
			Field:    "test_timeout",
			Message:  "Must be greater than 0",
			Severity: "error",
		})
	}
	
	// Suggestions
	if config.MaxConcurrentEnvironments > 10 {
		validation.Suggestions = append(validation.Suggestions, ConfigSuggestion{
			Section:   "system",
			Field:     "max_concurrent_environments",
			Current:   config.MaxConcurrentEnvironments,
			Suggested: 10,
			Reason:    "High number of concurrent environments may cause resource contention",
			Impact:    "Better resource utilization",
		})
	}
}

func (m *UnifiedConfigurationManager) validateEnvironmentConfig(config *EnvironmentConfig, validation *ConfigurationValidation) {
	if config.DefaultResources.Memory <= 0 {
		validation.Errors = append(validation.Errors, ConfigValidationError{
			Section:  "environments",
			Field:    "default_resources.memory",
			Message:  "Must be greater than 0",
			Severity: "error",
		})
	}
	
	if config.DefaultResources.CPUQuota <= 0 {
		validation.Errors = append(validation.Errors, ConfigValidationError{
			Section:  "environments",
			Field:    "default_resources.cpu_quota",
			Message:  "Must be greater than 0",
			Severity: "error",
		})
	}
	
	// Warnings
	if config.DefaultResources.Memory < 512*1024*1024 { // 512MB
		validation.Warnings = append(validation.Warnings, ConfigValidationWarning{
			Section: "environments",
			Field:   "default_resources.memory",
			Message: "Memory allocation is quite low",
			Impact:  "May cause test failures due to insufficient memory",
		})
	}
}

func (m *UnifiedConfigurationManager) validateExecutionConfig(config *ExecutionConfig, validation *ConfigurationValidation) {
	if config.ParallelExecution.MaxParallelGroups <= 0 {
		validation.Errors = append(validation.Errors, ConfigValidationError{
			Section:  "execution",
			Field:    "parallel_execution.max_parallel_groups",
			Message:  "Must be greater than 0",
			Severity: "error",
		})
	}
	
	if config.FailureHandling.MaxFailureThreshold <= 0 {
		validation.Errors = append(validation.Errors, ConfigValidationError{
			Section:  "execution",
			Field:    "failure_handling.max_failure_threshold",
			Message:  "Must be greater than 0",
			Severity: "error",
		})
	}
}

func (m *UnifiedConfigurationManager) validateQualityGatesConfig(config *QualityGatesConfig, validation *ConfigurationValidation) {
	for i, gate := range config.DefaultGates {
		if gate.Name == "" {
			validation.Errors = append(validation.Errors, ConfigValidationError{
				Section:  "quality_gates",
				Field:    fmt.Sprintf("default_gates[%d].name", i),
				Message:  "Gate name cannot be empty",
				Severity: "error",
			})
		}
		
		if gate.Threshold < 0 || gate.Threshold > 100 {
			validation.Warnings = append(validation.Warnings, ConfigValidationWarning{
				Section: "quality_gates",
				Field:   fmt.Sprintf("default_gates[%d].threshold", i),
				Message: "Threshold should typically be between 0 and 100",
				Impact:  "May cause unexpected gate behavior",
			})
		}
	}
}

func (m *UnifiedConfigurationManager) validateAIConfig(config *AIConfig, validation *ConfigurationValidation) {
	if config.Enabled {
		if config.Provider == "" {
			validation.Errors = append(validation.Errors, ConfigValidationError{
				Section:  "ai",
				Field:    "provider",
				Message:  "AI provider must be specified when AI is enabled",
				Severity: "error",
			})
		}
		
		if config.Model == "" {
			validation.Errors = append(validation.Errors, ConfigValidationError{
				Section:  "ai",
				Field:    "model",
				Message:  "AI model must be specified when AI is enabled",
				Severity: "error",
			})
		}
		
		if config.MaxTokens <= 0 {
			validation.Errors = append(validation.Errors, ConfigValidationError{
				Section:  "ai",
				Field:    "max_tokens",
				Message:  "Max tokens must be greater than 0",
				Severity: "error",
			})
		}
		
		if config.Temperature < 0 || config.Temperature > 2 {
			validation.Warnings = append(validation.Warnings, ConfigValidationWarning{
				Section: "ai",
				Field:   "temperature",
				Message: "Temperature should typically be between 0 and 2",
				Impact:  "May affect AI response quality",
			})
		}
	}
}

func (m *UnifiedConfigurationManager) validateMonitoringConfig(config *MonitoringConfig, validation *ConfigurationValidation) {
	if config.Enabled {
		if config.HealthCheckInterval <= 0 {
			validation.Errors = append(validation.Errors, ConfigValidationError{
				Section:  "monitoring",
				Field:    "health_check_interval",
				Message:  "Health check interval must be greater than 0",
				Severity: "error",
			})
		}
		
		if config.MetricsCollection.CollectionInterval <= 0 {
			validation.Errors = append(validation.Errors, ConfigValidationError{
				Section:  "monitoring",
				Field:    "metrics_collection.collection_interval",
				Message:  "Metrics collection interval must be greater than 0",
				Severity: "error",
			})
		}
	}
}

func (m *UnifiedConfigurationManager) validateReportingConfig(config *ReportingConfig, validation *ConfigurationValidation) {
	if config.Enabled {
		if config.GenerationInterval <= 0 {
			validation.Errors = append(validation.Errors, ConfigValidationError{
				Section:  "reporting",
				Field:    "generation_interval",
				Message:  "Report generation interval must be greater than 0",
				Severity: "error",
			})
		}
		
		if len(config.OutputFormats) == 0 {
			validation.Warnings = append(validation.Warnings, ConfigValidationWarning{
				Section: "reporting",
				Field:   "output_formats",
				Message: "No output formats specified",
				Impact:  "Reports may not be generated in desired formats",
			})
		}
	}
}

func (m *UnifiedConfigurationManager) validateSecurityConfig(config *SecurityConfig, validation *ConfigurationValidation) {
	if config.Enabled {
		if len(config.ScanTypes) == 0 {
			validation.Warnings = append(validation.Warnings, ConfigValidationWarning{
				Section: "security",
				Field:   "scan_types",
				Message: "No security scan types specified",
				Impact:  "Security scanning may not be effective",
			})
		}
	}
}

func (m *UnifiedConfigurationManager) validatePerformanceConfig(config *PerformanceConfig, validation *ConfigurationValidation) {
	if config.Enabled {
		if config.BaselineManagement.ComparisonThreshold <= 0 {
			validation.Errors = append(validation.Errors, ConfigValidationError{
				Section:  "performance",
				Field:    "baseline_management.comparison_threshold",
				Message:  "Comparison threshold must be greater than 0",
				Severity: "error",
			})
		}
		
		if config.LoadTesting.MaxVirtualUsers <= 0 {
			validation.Errors = append(validation.Errors, ConfigValidationError{
				Section:  "performance",
				Field:    "load_testing.max_virtual_users",
				Message:  "Max virtual users must be greater than 0",
				Severity: "error",
			})
		}
	}
}

func (m *UnifiedConfigurationManager) validateDataManagementConfig(config *DataManagementConfig, validation *ConfigurationValidation) {
	if config.TestDataGeneration.Enabled {
		if len(config.TestDataGeneration.Languages) == 0 {
			validation.Warnings = append(validation.Warnings, ConfigValidationWarning{
				Section: "data_management",
				Field:   "test_data_generation.languages",
				Message: "No languages specified for test data generation",
				Impact:  "Multilingual testing may not be effective",
			})
		}
	}
}

func (m *UnifiedConfigurationManager) generateOptimizationSuggestions(config *ComprehensiveTestingConfig, validation *ConfigurationValidation) {
	// Suggest optimizations based on configuration analysis
	
	// Performance optimization suggestions
	if config.System.MaxParallelTests > config.System.MaxConcurrentEnvironments*2 {
		validation.Suggestions = append(validation.Suggestions, ConfigSuggestion{
			Section:   "system",
			Field:     "max_parallel_tests",
			Current:   config.System.MaxParallelTests,
			Suggested: config.System.MaxConcurrentEnvironments * 2,
			Reason:    "Too many parallel tests relative to available environments",
			Impact:    "Better resource utilization and reduced contention",
		})
	}
	
	// AI optimization suggestions
	if config.AI.Enabled && config.AI.MaxTokens > 4000 {
		validation.Suggestions = append(validation.Suggestions, ConfigSuggestion{
			Section:   "ai",
			Field:     "max_tokens",
			Current:   config.AI.MaxTokens,
			Suggested: 2000,
			Reason:    "High token count may increase costs and latency",
			Impact:    "Reduced AI costs and faster response times",
		})
	}
	
	// Monitoring optimization suggestions
	if config.Monitoring.HealthCheckInterval < 10*time.Second {
		validation.Suggestions = append(validation.Suggestions, ConfigSuggestion{
			Section:   "monitoring",
			Field:     "health_check_interval",
			Current:   config.Monitoring.HealthCheckInterval,
			Suggested: 30 * time.Second,
			Reason:    "Very frequent health checks may cause unnecessary overhead",
			Impact:    "Reduced system overhead while maintaining adequate monitoring",
		})
	}
}

// Database operations

func (m *UnifiedConfigurationManager) createConfigurationTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS configuration_history (
			id SERIAL PRIMARY KEY,
			config_version VARCHAR(50) NOT NULL,
			config_data JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			created_by VARCHAR(100) DEFAULT 'system',
			change_description TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS configuration_templates (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			category VARCHAR(50),
			config_data JSONB NOT NULL,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_config_history_created_at ON configuration_history(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_config_templates_category ON configuration_templates(category)`,
	}
	
	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create configuration table: %w", err)
		}
	}
	
	return nil
}

func (m *UnifiedConfigurationManager) storeConfigurationInDB() error {
	configJSON, err := json.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	
	query := `
		INSERT INTO configuration_history (config_version, config_data, change_description)
		VALUES ($1, $2, $3)
	`
	
	_, err = m.db.Exec(query, m.configVersion, string(configJSON), "Configuration update")
	return err
}

func (m *UnifiedConfigurationManager) storeConfigurationTemplate(template *ConfigurationTemplate) error {
	configJSON, err := json.Marshal(template.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal template configuration: %w", err)
	}
	
	metadataJSON, err := json.Marshal(template.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal template metadata: %w", err)
	}
	
	query := `
		INSERT INTO configuration_templates (name, description, category, config_data, metadata)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (name) DO UPDATE SET
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			config_data = EXCLUDED.config_data,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`
	
	_, err = m.db.Exec(query, template.Name, template.Description, template.Category,
		string(configJSON), string(metadataJSON))
	
	return err
}

func (m *UnifiedConfigurationManager) loadConfigurationTemplate(name string) (*ConfigurationTemplate, error) {
	query := `
		SELECT name, description, category, config_data, metadata
		FROM configuration_templates
		WHERE name = $1
	`
	
	var template ConfigurationTemplate
	var configData, metadataData string
	
	err := m.db.QueryRow(query, name).Scan(
		&template.Name, &template.Description, &template.Category,
		&configData, &metadataData)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}
	
	// Parse configuration data
	var config ComprehensiveTestingConfig
	if err := json.Unmarshal([]byte(configData), &config); err != nil {
		return nil, fmt.Errorf("failed to parse template configuration: %w", err)
	}
	template.Config = &config
	
	// Parse metadata
	var metadata TemplateMetadata
	if err := json.Unmarshal([]byte(metadataData), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse template metadata: %w", err)
	}
	template.Metadata = metadata
	
	return &template, nil
}

// Supporting data structures

type ConfigurationHistoryEntry struct {
	ID                int                         `json:"id"`
	ConfigVersion     string                      `json:"config_version"`
	Config            *ComprehensiveTestingConfig `json:"config"`
	CreatedAt         time.Time                   `json:"created_at"`
	CreatedBy         string                      `json:"created_by"`
	ChangeDescription string                      `json:"change_description"`
}