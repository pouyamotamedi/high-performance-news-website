package testing

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// EnvironmentOptimizer optimizes test execution environments
type EnvironmentOptimizer struct {
	db *sql.DB
}

// NewEnvironmentOptimizer creates a new environment optimizer
func NewEnvironmentOptimizer(db *sql.DB) *EnvironmentOptimizer {
	return &EnvironmentOptimizer{db: db}
}

// OptimizeEnvironment optimizes a specific environment for a test
func (e *EnvironmentOptimizer) OptimizeEnvironment(environment, testName, testSuite string) bool {
	log.Printf("Optimizing environment %s for test %s.%s", environment, testSuite, testName)

	// Get environment-specific failure patterns
	failureRate, err := e.getEnvironmentFailureRate(environment, testName, testSuite)
	if err != nil {
		log.Printf("Failed to get failure rate for environment %s: %v", environment, err)
		return false
	}

	if failureRate < 0.3 {
		return true // Environment is already stable
	}

	// Apply environment-specific optimizations
	optimizations := []func(string, string, string) bool{
		e.optimizeResourceLimits,
		e.optimizeNetworkSettings,
		e.optimizeStorageSettings,
		e.optimizeProcessSettings,
	}

	success := true
	for _, optimize := range optimizations {
		if !optimize(environment, testName, testSuite) {
			success = false
		}
	}

	// Record optimization attempt
	e.recordOptimizationAttempt(environment, testName, testSuite, success)

	return success
}

// OptimizeAllEnvironments optimizes all test environments
func (e *EnvironmentOptimizer) OptimizeAllEnvironments() ([]OptimizationAction, error) {
	var actions []OptimizationAction

	// Get all environments with issues
	problematicEnvs, err := e.getProblematicEnvironments()
	if err != nil {
		return nil, fmt.Errorf("failed to get problematic environments: %w", err)
	}

	for _, env := range problematicEnvs {
		envActions := e.optimizeEnvironmentGlobally(env)
		actions = append(actions, envActions...)
	}

	return actions, nil
}

// optimizeResourceLimits optimizes CPU and memory limits for the environment
func (e *EnvironmentOptimizer) optimizeResourceLimits(environment, testName, testSuite string) bool {
	// Get current resource usage patterns
	avgDuration, err := e.getAverageTestDuration(environment, testName, testSuite)
	if err != nil {
		return false
	}

	// Calculate recommended resources based on duration
	var cpuLimit, memoryLimit int
	
	if avgDuration > 60*time.Second {
		// Long-running tests need more resources
		cpuLimit = 2000  // 2 CPU cores
		memoryLimit = 2048 // 2GB RAM
	} else if avgDuration > 30*time.Second {
		// Medium tests
		cpuLimit = 1000  // 1 CPU core
		memoryLimit = 1024 // 1GB RAM
	} else {
		// Fast tests
		cpuLimit = 500   // 0.5 CPU core
		memoryLimit = 512 // 512MB RAM
	}

	// Apply resource limits
	query := `
		INSERT INTO environment_resource_limits (environment, test_name, test_suite, cpu_limit, memory_limit, applied_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (environment, test_name, test_suite) DO UPDATE SET
			cpu_limit = EXCLUDED.cpu_limit,
			memory_limit = EXCLUDED.memory_limit,
			applied_at = NOW()
	`

	_, err = e.db.Exec(query, environment, testName, testSuite, cpuLimit, memoryLimit)
	return err == nil
}

// optimizeNetworkSettings optimizes network configuration for the environment
func (e *EnvironmentOptimizer) optimizeNetworkSettings(environment, testName, testSuite string) bool {
	// Check if test has network-related failures
	hasNetworkIssues, err := e.hasNetworkRelatedFailures(environment, testName, testSuite)
	if err != nil || !hasNetworkIssues {
		return !hasNetworkIssues
	}

	// Apply network optimizations
	networkConfig := map[string]interface{}{
		"connection_timeout":    30,  // 30 seconds
		"read_timeout":         60,  // 60 seconds
		"max_connections":      50,  // 50 concurrent connections
		"keep_alive_timeout":   300, // 5 minutes
		"retry_attempts":       3,   // 3 retry attempts
		"retry_delay":          1,   // 1 second delay between retries
	}

	query := `
		INSERT INTO environment_network_config (environment, test_name, test_suite, config, applied_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (environment, test_name, test_suite) DO UPDATE SET
			config = EXCLUDED.config,
			applied_at = NOW()
	`

	configJSON := fmt.Sprintf(`{
		"connection_timeout": %d,
		"read_timeout": %d,
		"max_connections": %d,
		"keep_alive_timeout": %d,
		"retry_attempts": %d,
		"retry_delay": %d
	}`, networkConfig["connection_timeout"], networkConfig["read_timeout"],
		networkConfig["max_connections"], networkConfig["keep_alive_timeout"],
		networkConfig["retry_attempts"], networkConfig["retry_delay"])

	_, err = e.db.Exec(query, environment, testName, testSuite, configJSON)
	return err == nil
}

// optimizeStorageSettings optimizes storage configuration for the environment
func (e *EnvironmentOptimizer) optimizeStorageSettings(environment, testName, testSuite string) bool {
	// Check if test has storage-related issues
	hasStorageIssues, err := e.hasStorageRelatedFailures(environment, testName, testSuite)
	if err != nil || !hasStorageIssues {
		return !hasStorageIssues
	}

	// Apply storage optimizations
	storageConfig := map[string]interface{}{
		"disk_space_limit":     "10GB",
		"temp_dir_cleanup":     true,
		"io_timeout":          30,
		"max_file_handles":    1000,
		"cache_size":          "1GB",
	}

	query := `
		INSERT INTO environment_storage_config (environment, test_name, test_suite, config, applied_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (environment, test_name, test_suite) DO UPDATE SET
			config = EXCLUDED.config,
			applied_at = NOW()
	`

	configJSON := fmt.Sprintf(`{
		"disk_space_limit": "%s",
		"temp_dir_cleanup": %t,
		"io_timeout": %d,
		"max_file_handles": %d,
		"cache_size": "%s"
	}`, storageConfig["disk_space_limit"], storageConfig["temp_dir_cleanup"],
		storageConfig["io_timeout"], storageConfig["max_file_handles"],
		storageConfig["cache_size"])

	_, err = e.db.Exec(query, environment, testName, testSuite, configJSON)
	return err == nil
}

// optimizeProcessSettings optimizes process configuration for the environment
func (e *EnvironmentOptimizer) optimizeProcessSettings(environment, testName, testSuite string) bool {
	// Apply process optimizations
	processConfig := map[string]interface{}{
		"max_processes":        10,
		"process_timeout":      300, // 5 minutes
		"cleanup_on_exit":      true,
		"resource_monitoring":  true,
		"auto_restart":         true,
	}

	query := `
		INSERT INTO environment_process_config (environment, test_name, test_suite, config, applied_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (environment, test_name, test_suite) DO UPDATE SET
			config = EXCLUDED.config,
			applied_at = NOW()
	`

	configJSON := fmt.Sprintf(`{
		"max_processes": %d,
		"process_timeout": %d,
		"cleanup_on_exit": %t,
		"resource_monitoring": %t,
		"auto_restart": %t
	}`, processConfig["max_processes"], processConfig["process_timeout"],
		processConfig["cleanup_on_exit"], processConfig["resource_monitoring"],
		processConfig["auto_restart"])

	_, err = e.db.Exec(query, environment, testName, testSuite, configJSON)
	return err == nil
}

// optimizeEnvironmentGlobally applies global optimizations to an environment
func (e *EnvironmentOptimizer) optimizeEnvironmentGlobally(environment string) []OptimizationAction {
	var actions []OptimizationAction

	// Get environment statistics
	stats, err := e.getEnvironmentStatistics(environment)
	if err != nil {
		log.Printf("Failed to get statistics for environment %s: %v", environment, err)
		return actions
	}

	// Apply global resource optimization
	if stats.AvgCPUUsage > 80 {
		action := OptimizationAction{
			Type:        "environment_cpu",
			Description: fmt.Sprintf("Increased CPU allocation for %s environment", environment),
			Applied:     e.increaseCPUAllocation(environment),
			Timestamp:   time.Now(),
		}
		actions = append(actions, action)
	}

	// Apply global memory optimization
	if stats.AvgMemoryUsage > 85 {
		action := OptimizationAction{
			Type:        "environment_memory",
			Description: fmt.Sprintf("Increased memory allocation for %s environment", environment),
			Applied:     e.increaseMemoryAllocation(environment),
			Timestamp:   time.Now(),
		}
		actions = append(actions, action)
	}

	// Apply network optimization
	if stats.NetworkErrorRate > 0.1 {
		action := OptimizationAction{
			Type:        "environment_network",
			Description: fmt.Sprintf("Optimized network configuration for %s environment", environment),
			Applied:     e.optimizeGlobalNetworkSettings(environment),
			Timestamp:   time.Now(),
		}
		actions = append(actions, action)
	}

	return actions
}

// Helper methods

func (e *EnvironmentOptimizer) getEnvironmentFailureRate(environment, testName, testSuite string) (float64, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status IN ('failed', 'error')) as failures
		FROM test_executions
		WHERE environment = $1 AND test_name = $2 AND test_suite = $3
		  AND start_time >= $4
	`

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	var total, failures int
	err := e.db.QueryRow(query, environment, testName, testSuite, cutoff).Scan(&total, &failures)
	if err != nil {
		return 0, err
	}

	if total == 0 {
		return 0, nil
	}

	return float64(failures) / float64(total), nil
}

func (e *EnvironmentOptimizer) getAverageTestDuration(environment, testName, testSuite string) (time.Duration, error) {
	query := `
		SELECT AVG(duration) as avg_duration
		FROM test_executions
		WHERE environment = $1 AND test_name = $2 AND test_suite = $3
		  AND start_time >= $4 AND status = 'passed'
	`

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	var avgDurationNs sql.NullInt64
	err := e.db.QueryRow(query, environment, testName, testSuite, cutoff).Scan(&avgDurationNs)
	if err != nil {
		return 0, err
	}

	if !avgDurationNs.Valid {
		return 30 * time.Second, nil // Default duration
	}

	return time.Duration(avgDurationNs.Int64), nil
}

func (e *EnvironmentOptimizer) hasNetworkRelatedFailures(environment, testName, testSuite string) (bool, error) {
	query := `
		SELECT COUNT(*) > 0
		FROM test_executions
		WHERE environment = $1 AND test_name = $2 AND test_suite = $3
		  AND start_time >= $4
		  AND status IN ('failed', 'error')
		  AND (error_message ILIKE '%connection%' OR 
		       error_message ILIKE '%network%' OR 
		       error_message ILIKE '%timeout%')
	`

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	var hasNetworkIssues bool
	err := e.db.QueryRow(query, environment, testName, testSuite, cutoff).Scan(&hasNetworkIssues)
	return hasNetworkIssues, err
}

func (e *EnvironmentOptimizer) hasStorageRelatedFailures(environment, testName, testSuite string) (bool, error) {
	query := `
		SELECT COUNT(*) > 0
		FROM test_executions
		WHERE environment = $1 AND test_name = $2 AND test_suite = $3
		  AND start_time >= $4
		  AND status IN ('failed', 'error')
		  AND (error_message ILIKE '%disk%' OR 
		       error_message ILIKE '%storage%' OR 
		       error_message ILIKE '%file%' OR
		       error_message ILIKE '%space%')
	`

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	var hasStorageIssues bool
	err := e.db.QueryRow(query, environment, testName, testSuite, cutoff).Scan(&hasStorageIssues)
	return hasStorageIssues, err
}

func (e *EnvironmentOptimizer) getProblematicEnvironments() ([]string, error) {
	query := `
		SELECT environment, 
		       COUNT(*) as total,
		       COUNT(*) FILTER (WHERE status IN ('failed', 'error')) as failures
		FROM test_executions
		WHERE start_time >= $1
		GROUP BY environment
		HAVING COUNT(*) >= 10 
		   AND COUNT(*) FILTER (WHERE status IN ('failed', 'error')) * 1.0 / COUNT(*) > 0.3
	`

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	rows, err := e.db.Query(query, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var environments []string
	for rows.Next() {
		var env string
		var total, failures int
		if err := rows.Scan(&env, &total, &failures); err != nil {
			continue
		}
		environments = append(environments, env)
	}

	return environments, nil
}

func (e *EnvironmentOptimizer) getEnvironmentStatistics(environment string) (*EnvironmentStatistics, error) {
	// This would typically query monitoring systems for actual resource usage
	// For now, return mock statistics
	return &EnvironmentStatistics{
		Environment:      environment,
		AvgCPUUsage:     75.0,
		AvgMemoryUsage:  80.0,
		NetworkErrorRate: 0.05,
		DiskUsage:       60.0,
	}, nil
}

func (e *EnvironmentOptimizer) increaseCPUAllocation(environment string) bool {
	query := `
		INSERT INTO environment_global_config (environment, config_type, config_value, applied_at)
		VALUES ($1, 'cpu_allocation', '2.0', NOW())
		ON CONFLICT (environment, config_type) DO UPDATE SET
			config_value = EXCLUDED.config_value,
			applied_at = NOW()
	`

	_, err := e.db.Exec(query, environment)
	return err == nil
}

func (e *EnvironmentOptimizer) increaseMemoryAllocation(environment string) bool {
	query := `
		INSERT INTO environment_global_config (environment, config_type, config_value, applied_at)
		VALUES ($1, 'memory_allocation', '4GB', NOW())
		ON CONFLICT (environment, config_type) DO UPDATE SET
			config_value = EXCLUDED.config_value,
			applied_at = NOW()
	`

	_, err := e.db.Exec(query, environment)
	return err == nil
}

func (e *EnvironmentOptimizer) optimizeGlobalNetworkSettings(environment string) bool {
	query := `
		INSERT INTO environment_global_config (environment, config_type, config_value, applied_at)
		VALUES ($1, 'network_optimization', 'enabled', NOW())
		ON CONFLICT (environment, config_type) DO UPDATE SET
			config_value = EXCLUDED.config_value,
			applied_at = NOW()
	`

	_, err := e.db.Exec(query, environment)
	return err == nil
}

func (e *EnvironmentOptimizer) recordOptimizationAttempt(environment, testName, testSuite string, success bool) {
	query := `
		INSERT INTO environment_optimization_attempts (environment, test_name, test_suite, success, attempted_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := e.db.Exec(query, environment, testName, testSuite, success)
	if err != nil {
		log.Printf("Failed to record optimization attempt: %v", err)
	}
}

// EnvironmentStatistics represents statistics for an environment
type EnvironmentStatistics struct {
	Environment      string  `json:"environment"`
	AvgCPUUsage     float64 `json:"avg_cpu_usage"`
	AvgMemoryUsage  float64 `json:"avg_memory_usage"`
	NetworkErrorRate float64 `json:"network_error_rate"`
	DiskUsage       float64 `json:"disk_usage"`
}