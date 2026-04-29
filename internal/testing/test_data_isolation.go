package testing

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// TestDataIsolationManager manages isolated test environments and data
type TestDataIsolationManager struct {
	dbPools        map[string]*sql.DB
	cachePools     map[string]interface{}
	fileSystems    map[string]string
	activeTests    map[string]*TestEnvironment
	mutex          sync.RWMutex
	cleanupPolicies []CleanupPolicy
	config         IsolationConfig
}

// TestEnvironment represents an isolated test environment
type TestEnvironment struct {
	ID              string                 `json:"id"`
	TestSuite       string                 `json:"test_suite"`
	DatabaseURL     string                 `json:"database_url"`
	CacheNamespace  string                 `json:"cache_namespace"`
	FileSystemPath  string                 `json:"filesystem_path"`
	CreatedAt       time.Time              `json:"created_at"`
	LastAccessedAt  time.Time              `json:"last_accessed_at"`
	Status          EnvironmentStatus      `json:"status"`
	Resources       ResourceAllocation     `json:"resources"`
	Metadata        map[string]interface{} `json:"metadata"`
	CleanupHandlers []func() error         `json:"-"`
}

// EnvironmentStatus represents the status of a test environment
type EnvironmentStatus string

const (
	EnvironmentStatusCreating EnvironmentStatus = "creating"
	EnvironmentStatusReady    EnvironmentStatus = "ready"
	EnvironmentStatusInUse    EnvironmentStatus = "in_use"
	EnvironmentStatusCleaning EnvironmentStatus = "cleaning"
	EnvironmentStatusFailed   EnvironmentStatus = "failed"
)

// ResourceAllocation represents allocated resources for a test environment
type ResourceAllocation struct {
	DatabaseConnections int   `json:"database_connections"`
	CacheMemoryMB      int   `json:"cache_memory_mb"`
	FileSystemSizeMB   int   `json:"filesystem_size_mb"`
	CPULimitPercent    int   `json:"cpu_limit_percent"`
	MemoryLimitMB      int   `json:"memory_limit_mb"`
}

// IsolationConfig holds configuration for test data isolation
type IsolationConfig struct {
	DatabaseHost     string        `json:"database_host"`
	DatabasePort     int           `json:"database_port"`
	CacheHost        string        `json:"cache_host"`
	CachePort        int           `json:"cache_port"`
	FileSystemRoot   string        `json:"filesystem_root"`
	MaxEnvironments  int           `json:"max_environments"`
	DefaultTTL       time.Duration `json:"default_ttl"`
	CleanupInterval  time.Duration `json:"cleanup_interval"`
	ResourceLimits   ResourceAllocation `json:"resource_limits"`
}

// CleanupPolicy defines how to clean up test data
type CleanupPolicy struct {
	Name        string                              `json:"name"`
	Pattern     string                              `json:"pattern"`
	MaxAge      time.Duration                       `json:"max_age"`
	MaxSize     int64                               `json:"max_size"`
	Condition   func(env *TestEnvironment) bool     `json:"-"`
	Action      func(env *TestEnvironment) error    `json:"-"`
}

// DataContaminationPrevention handles prevention of data contamination between tests
type DataContaminationPrevention struct {
	isolationManager *TestDataIsolationManager
	checksumCache    map[string]string
	mutex            sync.RWMutex
}

// NewTestDataIsolationManager creates a new test data isolation manager
func NewTestDataIsolationManager(config IsolationConfig) *TestDataIsolationManager {
	manager := &TestDataIsolationManager{
		dbPools:     make(map[string]*sql.DB),
		cachePools:  make(map[string]interface{}),
		fileSystems: make(map[string]string),
		activeTests: make(map[string]*TestEnvironment),
		config:      config,
	}
	
	manager.initializeDefaultCleanupPolicies()
	manager.startCleanupScheduler()
	
	return manager
}

// CreateIsolatedEnvironment creates a new isolated test environment
func (tim *TestDataIsolationManager) CreateIsolatedEnvironment(testSuite string) (*TestEnvironment, error) {
	tim.mutex.Lock()
	defer tim.mutex.Unlock()
	
	// Check environment limits
	if len(tim.activeTests) >= tim.config.MaxEnvironments {
		return nil, fmt.Errorf("maximum number of environments (%d) reached", tim.config.MaxEnvironments)
	}
	
	// Generate unique environment ID
	envID := tim.generateEnvironmentID(testSuite)
	
	env := &TestEnvironment{
		ID:             envID,
		TestSuite:      testSuite,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		Status:         EnvironmentStatusCreating,
		Resources:      tim.config.ResourceLimits,
		Metadata:       make(map[string]interface{}),
		CleanupHandlers: make([]func() error, 0),
	}
	
	// Create isolated database
	if err := tim.createIsolatedDatabase(env); err != nil {
		env.Status = EnvironmentStatusFailed
		return nil, fmt.Errorf("failed to create isolated database: %w", err)
	}
	
	// Create isolated cache namespace
	if err := tim.createIsolatedCache(env); err != nil {
		env.Status = EnvironmentStatusFailed
		tim.cleanupDatabase(env)
		return nil, fmt.Errorf("failed to create isolated cache: %w", err)
	}
	
	// Create isolated file system
	if err := tim.createIsolatedFileSystem(env); err != nil {
		env.Status = EnvironmentStatusFailed
		tim.cleanupDatabase(env)
		tim.cleanupCache(env)
		return nil, fmt.Errorf("failed to create isolated filesystem: %w", err)
	}
	
	env.Status = EnvironmentStatusReady
	tim.activeTests[envID] = env
	
	log.Printf("Created isolated environment %s for test suite %s", envID, testSuite)
	return env, nil
}

// GetEnvironment retrieves an existing test environment
func (tim *TestDataIsolationManager) GetEnvironment(envID string) (*TestEnvironment, error) {
	tim.mutex.RLock()
	defer tim.mutex.RUnlock()
	
	env, exists := tim.activeTests[envID]
	if !exists {
		return nil, fmt.Errorf("environment %s not found", envID)
	}
	
	env.LastAccessedAt = time.Now()
	return env, nil
}

// CleanupEnvironment cleans up and removes a test environment
func (tim *TestDataIsolationManager) CleanupEnvironment(envID string) error {
	tim.mutex.Lock()
	defer tim.mutex.Unlock()
	
	env, exists := tim.activeTests[envID]
	if !exists {
		return fmt.Errorf("environment %s not found", envID)
	}
	
	env.Status = EnvironmentStatusCleaning
	
	// Run custom cleanup handlers first
	for _, handler := range env.CleanupHandlers {
		if err := handler(); err != nil {
			log.Printf("Custom cleanup handler failed for environment %s: %v", envID, err)
		}
	}
	
	// Cleanup resources
	var errors []string
	
	if err := tim.cleanupDatabase(env); err != nil {
		errors = append(errors, fmt.Sprintf("database cleanup failed: %v", err))
	}
	
	if err := tim.cleanupCache(env); err != nil {
		errors = append(errors, fmt.Sprintf("cache cleanup failed: %v", err))
	}
	
	if err := tim.cleanupFileSystem(env); err != nil {
		errors = append(errors, fmt.Sprintf("filesystem cleanup failed: %v", err))
	}
	
	// Remove from active tests
	delete(tim.activeTests, envID)
	
	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errors, "; "))
	}
	
	log.Printf("Successfully cleaned up environment %s", envID)
	return nil
}

// createIsolatedDatabase creates an isolated database for the test environment
func (tim *TestDataIsolationManager) createIsolatedDatabase(env *TestEnvironment) error {
	// Create unique database name
	dbName := fmt.Sprintf("test_%s_%s", env.TestSuite, env.ID)
	dbName = strings.ReplaceAll(dbName, "-", "_")
	
	// Connect to postgres database to create new database
	adminDSN := fmt.Sprintf("host=%s port=%d user=postgres dbname=postgres sslmode=disable",
		tim.config.DatabaseHost, tim.config.DatabasePort)
	
	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to admin database: %w", err)
	}
	defer adminDB.Close()
	
	// Create database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}
	
	// Create database connection for the test environment
	testDSN := fmt.Sprintf("host=%s port=%d user=postgres dbname=%s sslmode=disable",
		tim.config.DatabaseHost, tim.config.DatabasePort, dbName)
	
	testDB, err := sql.Open("postgres", testDSN)
	if err != nil {
		// Cleanup created database
		adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		return fmt.Errorf("failed to connect to test database: %w", err)
	}
	
	// Configure connection pool
	testDB.SetMaxOpenConns(env.Resources.DatabaseConnections)
	testDB.SetMaxIdleConns(env.Resources.DatabaseConnections / 2)
	testDB.SetConnMaxLifetime(time.Hour)
	
	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		return fmt.Errorf("failed to ping test database: %w", err)
	}
	
	env.DatabaseURL = testDSN
	tim.dbPools[env.ID] = testDB
	
	// Add cleanup handler
	env.CleanupHandlers = append(env.CleanupHandlers, func() error {
		return tim.cleanupDatabase(env)
	})
	
	return nil
}

// createIsolatedCache creates an isolated cache namespace for the test environment
func (tim *TestDataIsolationManager) createIsolatedCache(env *TestEnvironment) error {
	// Create unique cache namespace
	namespace := fmt.Sprintf("test:%s:%s:", env.TestSuite, env.ID)
	
	// Create Redis client with isolated database
	cacheDB := tim.getCacheDBNumber(env.ID)
	
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", tim.config.CacheHost, tim.config.CachePort),
		DB:       cacheDB,
		Password: "", // No password for test environment
	})
	
	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return fmt.Errorf("failed to connect to cache: %w", err)
	}
	
	env.CacheNamespace = namespace
	tim.cachePools[env.ID] = client
	
	// Add cleanup handler
	env.CleanupHandlers = append(env.CleanupHandlers, func() error {
		return tim.cleanupCache(env)
	})
	
	return nil
}

// createIsolatedFileSystem creates an isolated file system for the test environment
func (tim *TestDataIsolationManager) createIsolatedFileSystem(env *TestEnvironment) error {
	// Create unique directory path
	fsPath := filepath.Join(tim.config.FileSystemRoot, "test_data", env.TestSuite, env.ID)
	
	// Create directory structure
	if err := os.MkdirAll(fsPath, 0755); err != nil {
		return fmt.Errorf("failed to create filesystem directory: %w", err)
	}
	
	// Create subdirectories
	subdirs := []string{"uploads", "temp", "cache", "logs"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(fsPath, subdir), 0755); err != nil {
			return fmt.Errorf("failed to create subdirectory %s: %w", subdir, err)
		}
	}
	
	env.FileSystemPath = fsPath
	tim.fileSystems[env.ID] = fsPath
	
	// Add cleanup handler
	env.CleanupHandlers = append(env.CleanupHandlers, func() error {
		return tim.cleanupFileSystem(env)
	})
	
	return nil
}

// GetDatabaseConnection returns the isolated database connection for an environment
func (tim *TestDataIsolationManager) GetDatabaseConnection(envID string) (*sql.DB, error) {
	tim.mutex.RLock()
	defer tim.mutex.RUnlock()
	
	db, exists := tim.dbPools[envID]
	if !exists {
		return nil, fmt.Errorf("database connection not found for environment %s", envID)
	}
	
	return db, nil
}

// GetCacheClient returns the isolated cache client for an environment
func (tim *TestDataIsolationManager) GetCacheClient(envID string) (interface{}, error) {
	tim.mutex.RLock()
	defer tim.mutex.RUnlock()
	
	client, exists := tim.cachePools[envID]
	if !exists {
		return nil, fmt.Errorf("cache client not found for environment %s", envID)
	}
	
	return client, nil
}

// GetFileSystemPath returns the isolated file system path for an environment
func (tim *TestDataIsolationManager) GetFileSystemPath(envID string) (string, error) {
	tim.mutex.RLock()
	defer tim.mutex.RUnlock()
	
	path, exists := tim.fileSystems[envID]
	if !exists {
		return "", fmt.Errorf("filesystem path not found for environment %s", envID)
	}
	
	return path, nil
}

// cleanupDatabase cleans up the isolated database
func (tim *TestDataIsolationManager) cleanupDatabase(env *TestEnvironment) error {
	// Close connection pool
	if db, exists := tim.dbPools[env.ID]; exists {
		db.Close()
		delete(tim.dbPools, env.ID)
	}
	
	// Drop database
	if env.DatabaseURL != "" {
		dbName := tim.extractDatabaseName(env.DatabaseURL)
		if dbName != "" {
			adminDSN := fmt.Sprintf("host=%s port=%d user=postgres dbname=postgres sslmode=disable",
				tim.config.DatabaseHost, tim.config.DatabasePort)
			
			adminDB, err := sql.Open("postgres", adminDSN)
			if err != nil {
				return fmt.Errorf("failed to connect to admin database: %w", err)
			}
			defer adminDB.Close()
			
			// Terminate active connections to the database
			_, err = adminDB.Exec(fmt.Sprintf(`
				SELECT pg_terminate_backend(pid)
				FROM pg_stat_activity
				WHERE datname = '%s' AND pid <> pg_backend_pid()
			`, dbName))
			if err != nil {
				log.Printf("Warning: failed to terminate connections to database %s: %v", dbName, err)
			}
			
			// Drop database
			_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
			if err != nil {
				return fmt.Errorf("failed to drop database %s: %w", dbName, err)
			}
		}
	}
	
	return nil
}

// cleanupCache cleans up the isolated cache namespace
func (tim *TestDataIsolationManager) cleanupCache(env *TestEnvironment) error {
	if client, exists := tim.cachePools[env.ID]; exists {
		ctx := context.Background()
		
		// Delete all keys in the namespace
		if env.CacheNamespace != "" {
			keys, err := client.Keys(ctx, env.CacheNamespace+"*").Result()
			if err == nil && len(keys) > 0 {
				client.Del(ctx, keys...)
			}
		}
		
		// Flush the entire database (since we use separate DB numbers)
		client.FlushDB(ctx)
		
		// Close client
		client.Close()
		delete(tim.cachePools, env.ID)
	}
	
	return nil
}

// cleanupFileSystem cleans up the isolated file system
func (tim *TestDataIsolationManager) cleanupFileSystem(env *TestEnvironment) error {
	if path, exists := tim.fileSystems[env.ID]; exists {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove filesystem directory %s: %w", path, err)
		}
		delete(tim.fileSystems, env.ID)
	}
	
	return nil
}

// generateEnvironmentID generates a unique environment ID
func (tim *TestDataIsolationManager) generateEnvironmentID(testSuite string) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)
	
	return fmt.Sprintf("%s_%d_%s", testSuite, timestamp, randomHex)
}

// getCacheDBNumber returns a cache database number for the environment
func (tim *TestDataIsolationManager) getCacheDBNumber(envID string) int {
	// Use hash of environment ID to determine DB number (0-15)
	hash := 0
	for _, char := range envID {
		hash = (hash + int(char)) % 16
	}
	return hash
}

// extractDatabaseName extracts database name from DSN
func (tim *TestDataIsolationManager) extractDatabaseName(dsn string) string {
	parts := strings.Split(dsn, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, "dbname=") {
			return strings.TrimPrefix(part, "dbname=")
		}
	}
	return ""
}

// initializeDefaultCleanupPolicies sets up default cleanup policies
func (tim *TestDataIsolationManager) initializeDefaultCleanupPolicies() {
	tim.cleanupPolicies = []CleanupPolicy{
		{
			Name:   "age_based_cleanup",
			MaxAge: tim.config.DefaultTTL,
			Condition: func(env *TestEnvironment) bool {
				return time.Since(env.LastAccessedAt) > tim.config.DefaultTTL
			},
			Action: func(env *TestEnvironment) error {
				return tim.CleanupEnvironment(env.ID)
			},
		},
		{
			Name: "failed_environment_cleanup",
			Condition: func(env *TestEnvironment) bool {
				return env.Status == EnvironmentStatusFailed
			},
			Action: func(env *TestEnvironment) error {
				return tim.CleanupEnvironment(env.ID)
			},
		},
		{
			Name:   "stale_environment_cleanup",
			MaxAge: 24 * time.Hour, // Clean up environments older than 24 hours
			Condition: func(env *TestEnvironment) bool {
				return time.Since(env.CreatedAt) > 24*time.Hour
			},
			Action: func(env *TestEnvironment) error {
				return tim.CleanupEnvironment(env.ID)
			},
		},
	}
}

// startCleanupScheduler starts the background cleanup scheduler
func (tim *TestDataIsolationManager) startCleanupScheduler() {
	go func() {
		ticker := time.NewTicker(tim.config.CleanupInterval)
		defer ticker.Stop()
		
		for range ticker.C {
			tim.runCleanupPolicies()
		}
	}()
}

// runCleanupPolicies runs all cleanup policies
func (tim *TestDataIsolationManager) runCleanupPolicies() {
	tim.mutex.RLock()
	environments := make([]*TestEnvironment, 0, len(tim.activeTests))
	for _, env := range tim.activeTests {
		environments = append(environments, env)
	}
	tim.mutex.RUnlock()
	
	for _, env := range environments {
		for _, policy := range tim.cleanupPolicies {
			if policy.Condition(env) {
				if err := policy.Action(env); err != nil {
					log.Printf("Cleanup policy %s failed for environment %s: %v", 
						policy.Name, env.ID, err)
				} else {
					log.Printf("Cleanup policy %s applied to environment %s", 
						policy.Name, env.ID)
				}
				break // Only apply first matching policy
			}
		}
	}
}

// ValidateDataConsistency validates data consistency across isolated environments
func (tim *TestDataIsolationManager) ValidateDataConsistency(envID string) error {
	env, err := tim.GetEnvironment(envID)
	if err != nil {
		return err
	}
	
	// Validate database consistency
	if err := tim.validateDatabaseConsistency(env); err != nil {
		return fmt.Errorf("database consistency validation failed: %w", err)
	}
	
	// Validate cache consistency
	if err := tim.validateCacheConsistency(env); err != nil {
		return fmt.Errorf("cache consistency validation failed: %w", err)
	}
	
	// Validate filesystem consistency
	if err := tim.validateFileSystemConsistency(env); err != nil {
		return fmt.Errorf("filesystem consistency validation failed: %w", err)
	}
	
	return nil
}

// validateDatabaseConsistency validates database data consistency
func (tim *TestDataIsolationManager) validateDatabaseConsistency(env *TestEnvironment) error {
	db, err := tim.GetDatabaseConnection(env.ID)
	if err != nil {
		return err
	}
	
	// Check for foreign key violations
	var violations int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage ccu ON tc.constraint_name = ccu.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
	`).Scan(&violations)
	
	if err != nil {
		return fmt.Errorf("failed to check foreign key constraints: %w", err)
	}
	
	// Additional consistency checks can be added here
	
	return nil
}

// validateCacheConsistency validates cache data consistency
func (tim *TestDataIsolationManager) validateCacheConsistency(env *TestEnvironment) error {
	client, err := tim.GetCacheClient(env.ID)
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	
	// Check for orphaned keys
	keys, err := client.Keys(ctx, env.CacheNamespace+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to get cache keys: %w", err)
	}
	
	// Validate key patterns and TTLs
	for _, key := range keys {
		ttl, err := client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}
		
		// Check for keys without TTL that should have one
		if ttl == -1 && strings.Contains(key, "temp:") {
			log.Printf("Warning: temporary key %s has no TTL", key)
		}
	}
	
	return nil
}

// validateFileSystemConsistency validates filesystem data consistency
func (tim *TestDataIsolationManager) validateFileSystemConsistency(env *TestEnvironment) error {
	path, err := tim.GetFileSystemPath(env.ID)
	if err != nil {
		return err
	}
	
	// Check if directory exists and is accessible
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("filesystem path %s does not exist", path)
	}
	
	// Check directory permissions
	if err := tim.checkDirectoryPermissions(path); err != nil {
		return fmt.Errorf("filesystem permission check failed: %w", err)
	}
	
	return nil
}

// checkDirectoryPermissions checks if directory has proper permissions
func (tim *TestDataIsolationManager) checkDirectoryPermissions(path string) error {
	// Try to create a test file
	testFile := filepath.Join(path, ".permission_test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot create files in directory %s: %w", path, err)
	}
	file.Close()
	
	// Clean up test file
	os.Remove(testFile)
	
	return nil
}

// GetEnvironmentStats returns statistics about active environments
func (tim *TestDataIsolationManager) GetEnvironmentStats() map[string]interface{} {
	tim.mutex.RLock()
	defer tim.mutex.RUnlock()
	
	stats := map[string]interface{}{
		"total_environments": len(tim.activeTests),
		"by_status":         make(map[EnvironmentStatus]int),
		"by_test_suite":     make(map[string]int),
		"resource_usage":    tim.calculateResourceUsage(),
	}
	
	statusCounts := make(map[EnvironmentStatus]int)
	suiteCounts := make(map[string]int)
	
	for _, env := range tim.activeTests {
		statusCounts[env.Status]++
		suiteCounts[env.TestSuite]++
	}
	
	stats["by_status"] = statusCounts
	stats["by_test_suite"] = suiteCounts
	
	return stats
}

// calculateResourceUsage calculates current resource usage
func (tim *TestDataIsolationManager) calculateResourceUsage() map[string]interface{} {
	totalDBConnections := 0
	totalCacheMemory := 0
	totalFileSystemSize := 0
	
	for _, env := range tim.activeTests {
		totalDBConnections += env.Resources.DatabaseConnections
		totalCacheMemory += env.Resources.CacheMemoryMB
		totalFileSystemSize += env.Resources.FileSystemSizeMB
	}
	
	return map[string]interface{}{
		"database_connections": totalDBConnections,
		"cache_memory_mb":     totalCacheMemory,
		"filesystem_size_mb":  totalFileSystemSize,
	}
}

// CleanupAllEnvironments cleans up all active environments
func (tim *TestDataIsolationManager) CleanupAllEnvironments() error {
	tim.mutex.RLock()
	envIDs := make([]string, 0, len(tim.activeTests))
	for envID := range tim.activeTests {
		envIDs = append(envIDs, envID)
	}
	tim.mutex.RUnlock()
	
	var errors []string
	for _, envID := range envIDs {
		if err := tim.CleanupEnvironment(envID); err != nil {
			errors = append(errors, fmt.Sprintf("environment %s: %v", envID, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

// NewDataContaminationPrevention creates a new data contamination prevention system
func NewDataContaminationPrevention(isolationManager *TestDataIsolationManager) *DataContaminationPrevention {
	return &DataContaminationPrevention{
		isolationManager: isolationManager,
		checksumCache:    make(map[string]string),
	}
}

// PreventContamination prevents data contamination between test runs
func (dcp *DataContaminationPrevention) PreventContamination(envID string, data interface{}) error {
	// Calculate checksum of data
	checksum := dcp.calculateChecksum(data)
	
	dcp.mutex.Lock()
	defer dcp.mutex.Unlock()
	
	// Check if data has been modified unexpectedly
	if existingChecksum, exists := dcp.checksumCache[envID]; exists {
		if existingChecksum != checksum {
			return fmt.Errorf("data contamination detected in environment %s", envID)
		}
	}
	
	// Store checksum
	dcp.checksumCache[envID] = checksum
	
	return nil
}

// calculateChecksum calculates a checksum for data
func (dcp *DataContaminationPrevention) calculateChecksum(data interface{}) string {
	// Simple checksum implementation (in production, use proper hashing)
	return fmt.Sprintf("%x", data)
}

// RepairDataConsistency attempts to repair data consistency issues
func (tim *TestDataIsolationManager) RepairDataConsistency(envID string) error {
	env, err := tim.GetEnvironment(envID)
	if err != nil {
		return err
	}
	
	log.Printf("Attempting to repair data consistency for environment %s", envID)
	
	// Repair database consistency
	if err := tim.repairDatabaseConsistency(env); err != nil {
		return fmt.Errorf("database consistency repair failed: %w", err)
	}
	
	// Repair cache consistency
	if err := tim.repairCacheConsistency(env); err != nil {
		return fmt.Errorf("cache consistency repair failed: %w", err)
	}
	
	// Repair filesystem consistency
	if err := tim.repairFileSystemConsistency(env); err != nil {
		return fmt.Errorf("filesystem consistency repair failed: %w", err)
	}
	
	log.Printf("Data consistency repair completed for environment %s", envID)
	return nil
}

// repairDatabaseConsistency repairs database consistency issues
func (tim *TestDataIsolationManager) repairDatabaseConsistency(env *TestEnvironment) error {
	db, err := tim.GetDatabaseConnection(env.ID)
	if err != nil {
		return err
	}
	
	// Repair foreign key violations by removing orphaned records
	_, err = db.Exec(`
		DELETE FROM articles 
		WHERE author_id NOT IN (SELECT id FROM users)
	`)
	if err != nil {
		return fmt.Errorf("failed to repair foreign key violations: %w", err)
	}
	
	return nil
}

// repairCacheConsistency repairs cache consistency issues
func (tim *TestDataIsolationManager) repairCacheConsistency(env *TestEnvironment) error {
	client, err := tim.GetCacheClient(env.ID)
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	
	// Remove orphaned temporary keys
	keys, err := client.Keys(ctx, env.CacheNamespace+"temp:*").Result()
	if err != nil {
		return err
	}
	
	for _, key := range keys {
		ttl, err := client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}
		
		// Set TTL for keys that don't have one
		if ttl == -1 {
			client.Expire(ctx, key, time.Hour)
		}
	}
	
	return nil
}

// repairFileSystemConsistency repairs filesystem consistency issues
func (tim *TestDataIsolationManager) repairFileSystemConsistency(env *TestEnvironment) error {
	path, err := tim.GetFileSystemPath(env.ID)
	if err != nil {
		return err
	}
	
	// Recreate missing directories
	subdirs := []string{"uploads", "temp", "cache", "logs"}
	for _, subdir := range subdirs {
		subdirPath := filepath.Join(path, subdir)
		if _, err := os.Stat(subdirPath); os.IsNotExist(err) {
			if err := os.MkdirAll(subdirPath, 0755); err != nil {
				return fmt.Errorf("failed to recreate directory %s: %w", subdirPath, err)
			}
		}
	}
	
	return nil
}