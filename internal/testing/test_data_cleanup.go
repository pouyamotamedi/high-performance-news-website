package testing

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TestDataCleanupManager manages cleanup and archival of test data
type TestDataCleanupManager struct {
	isolationManager *TestDataIsolationManager
	archiveStorage   ArchiveStorage
	cleanupPolicies  []CleanupPolicy
	archivePolicies  []ArchivePolicy
	mutex            sync.RWMutex
	config           CleanupConfig
}

// CleanupConfig holds configuration for test data cleanup
type CleanupConfig struct {
	ArchiveDirectory    string        `json:"archive_directory"`
	MaxArchiveSize      int64         `json:"max_archive_size_bytes"`
	CompressionEnabled  bool          `json:"compression_enabled"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	CleanupBatchSize    int           `json:"cleanup_batch_size"`
	ParallelWorkers     int           `json:"parallel_workers"`
	VerifyBeforeDelete  bool          `json:"verify_before_delete"`
}

// ArchivePolicy defines how to archive test data
type ArchivePolicy struct {
	Name            string                              `json:"name"`
	Pattern         string                              `json:"pattern"`
	MaxAge          time.Duration                       `json:"max_age"`
	CompressionType string                              `json:"compression_type"`
	Condition       func(env *TestEnvironment) bool     `json:"-"`
	ArchiveAction   func(env *TestEnvironment) error    `json:"-"`
}

// ArchiveStorage interface for different archive storage backends
type ArchiveStorage interface {
	Store(key string, data []byte) error
	Retrieve(key string) ([]byte, error)
	Delete(key string) error
	List(pattern string) ([]string, error)
	Size(key string) (int64, error)
}

// FileSystemArchiveStorage implements ArchiveStorage using filesystem
type FileSystemArchiveStorage struct {
	basePath string
	mutex    sync.RWMutex
}

// ArchiveMetadata holds metadata about archived test data
type ArchiveMetadata struct {
	ArchiveID       string                 `json:"archive_id"`
	OriginalEnvID   string                 `json:"original_env_id"`
	TestSuite       string                 `json:"test_suite"`
	ArchivedAt      time.Time              `json:"archived_at"`
	OriginalSize    int64                  `json:"original_size_bytes"`
	CompressedSize  int64                  `json:"compressed_size_bytes"`
	CompressionType string                 `json:"compression_type"`
	DataTypes       []string               `json:"data_types"`
	Checksum        string                 `json:"checksum"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// CleanupResult holds the result of a cleanup operation
type CleanupResult struct {
	EnvironmentsProcessed int                    `json:"environments_processed"`
	EnvironmentsArchived  int                    `json:"environments_archived"`
	EnvironmentsDeleted   int                    `json:"environments_deleted"`
	DataSizeArchived      int64                  `json:"data_size_archived_bytes"`
	DataSizeDeleted       int64                  `json:"data_size_deleted_bytes"`
	Duration              time.Duration          `json:"duration"`
	Errors                []string               `json:"errors"`
	ArchivedItems         []ArchiveMetadata      `json:"archived_items"`
}

// NewTestDataCleanupManager creates a new test data cleanup manager
func NewTestDataCleanupManager(isolationManager *TestDataIsolationManager, config CleanupConfig) *TestDataCleanupManager {
	manager := &TestDataCleanupManager{
		isolationManager: isolationManager,
		archiveStorage:   NewFileSystemArchiveStorage(config.ArchiveDirectory),
		cleanupPolicies:  make([]CleanupPolicy, 0),
		archivePolicies:  make([]ArchivePolicy, 0),
		config:           config,
	}
	
	manager.initializeDefaultPolicies()
	return manager
}

// NewFileSystemArchiveStorage creates a new filesystem-based archive storage
func NewFileSystemArchiveStorage(basePath string) *FileSystemArchiveStorage {
	// Ensure archive directory exists
	os.MkdirAll(basePath, 0755)
	
	return &FileSystemArchiveStorage{
		basePath: basePath,
	}
}

// Store stores data in the filesystem archive
func (fs *FileSystemArchiveStorage) Store(key string, data []byte) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	
	filePath := filepath.Join(fs.basePath, key)
	
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	
	// Write data to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}
	
	return nil
}

// Retrieve retrieves data from the filesystem archive
func (fs *FileSystemArchiveStorage) Retrieve(key string) ([]byte, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	
	filePath := filepath.Join(fs.basePath, key)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	return data, nil
}

// Delete deletes data from the filesystem archive
func (fs *FileSystemArchiveStorage) Delete(key string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	
	filePath := filepath.Join(fs.basePath, key)
	
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", filePath, err)
	}
	
	return nil
}

// List lists files matching a pattern in the archive
func (fs *FileSystemArchiveStorage) List(pattern string) ([]string, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	
	var matches []string
	
	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			relPath, err := filepath.Rel(fs.basePath, path)
			if err != nil {
				return err
			}
			
			// Simple pattern matching (in production, use proper glob matching)
			if strings.Contains(relPath, pattern) {
				matches = append(matches, relPath)
			}
		}
		
		return nil
	})
	
	return matches, err
}

// Size returns the size of a file in the archive
func (fs *FileSystemArchiveStorage) Size(key string) (int64, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	
	filePath := filepath.Join(fs.basePath, key)
	
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	
	return info.Size(), nil
}

// initializeDefaultPolicies sets up default cleanup and archive policies
func (tcm *TestDataCleanupManager) initializeDefaultPolicies() {
	// Default cleanup policies
	tcm.cleanupPolicies = []CleanupPolicy{
		{
			Name:   "expired_environments",
			MaxAge: 24 * time.Hour,
			Condition: func(env *TestEnvironment) bool {
				return time.Since(env.CreatedAt) > 24*time.Hour
			},
			Action: func(env *TestEnvironment) error {
				return tcm.isolationManager.CleanupEnvironment(env.ID)
			},
		},
		{
			Name:   "failed_environments",
			MaxAge: time.Hour,
			Condition: func(env *TestEnvironment) bool {
				return env.Status == EnvironmentStatusFailed && time.Since(env.CreatedAt) > time.Hour
			},
			Action: func(env *TestEnvironment) error {
				return tcm.isolationManager.CleanupEnvironment(env.ID)
			},
		},
		{
			Name:   "unused_environments",
			MaxAge: 4 * time.Hour,
			Condition: func(env *TestEnvironment) bool {
				return time.Since(env.LastAccessedAt) > 4*time.Hour
			},
			Action: func(env *TestEnvironment) error {
				return tcm.isolationManager.CleanupEnvironment(env.ID)
			},
		},
	}
	
	// Default archive policies
	tcm.archivePolicies = []ArchivePolicy{
		{
			Name:            "completed_test_environments",
			MaxAge:          2 * time.Hour,
			CompressionType: "gzip",
			Condition: func(env *TestEnvironment) bool {
				return env.Status == EnvironmentStatusReady && time.Since(env.LastAccessedAt) > 2*time.Hour
			},
			ArchiveAction: func(env *TestEnvironment) error {
				return tcm.archiveEnvironment(env)
			},
		},
		{
			Name:            "large_test_datasets",
			MaxAge:          time.Hour,
			CompressionType: "gzip",
			Condition: func(env *TestEnvironment) bool {
				// Archive environments with large datasets
				return env.Resources.FileSystemSizeMB > 100 // > 100MB
			},
			ArchiveAction: func(env *TestEnvironment) error {
				return tcm.archiveEnvironment(env)
			},
		},
	}
}

// RunCleanup runs the cleanup process for all environments
func (tcm *TestDataCleanupManager) RunCleanup() (*CleanupResult, error) {
	startTime := time.Now()
	
	result := &CleanupResult{
		Errors:        make([]string, 0),
		ArchivedItems: make([]ArchiveMetadata, 0),
	}
	
	// Get all active environments
	stats := tcm.isolationManager.GetEnvironmentStats()
	totalEnvs := stats["total_environments"].(int)
	result.EnvironmentsProcessed = totalEnvs
	
	log.Printf("Starting cleanup process for %d environments", totalEnvs)
	
	// Run archive policies first
	if err := tcm.runArchivePolicies(result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("archive policies failed: %v", err))
	}
	
	// Run cleanup policies
	if err := tcm.runCleanupPolicies(result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("cleanup policies failed: %v", err))
	}
	
	// Clean up old archives
	if err := tcm.cleanupOldArchives(result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("archive cleanup failed: %v", err))
	}
	
	result.Duration = time.Since(startTime)
	
	log.Printf("Cleanup completed: %d archived, %d deleted, %d errors in %v",
		result.EnvironmentsArchived, result.EnvironmentsDeleted, len(result.Errors), result.Duration)
	
	return result, nil
}

// runArchivePolicies runs all archive policies
func (tcm *TestDataCleanupManager) runArchivePolicies(result *CleanupResult) error {
	tcm.isolationManager.mutex.RLock()
	environments := make([]*TestEnvironment, 0)
	for _, env := range tcm.isolationManager.activeTests {
		environments = append(environments, env)
	}
	tcm.isolationManager.mutex.RUnlock()
	
	for _, env := range environments {
		for _, policy := range tcm.archivePolicies {
			if policy.Condition(env) {
				if err := policy.ArchiveAction(env); err != nil {
					result.Errors = append(result.Errors, 
						fmt.Sprintf("archive policy %s failed for env %s: %v", policy.Name, env.ID, err))
				} else {
					result.EnvironmentsArchived++
					log.Printf("Archived environment %s using policy %s", env.ID, policy.Name)
				}
				break // Only apply first matching policy
			}
		}
	}
	
	return nil
}

// runCleanupPolicies runs all cleanup policies
func (tcm *TestDataCleanupManager) runCleanupPolicies(result *CleanupResult) error {
	tcm.isolationManager.mutex.RLock()
	environments := make([]*TestEnvironment, 0)
	for _, env := range tcm.isolationManager.activeTests {
		environments = append(environments, env)
	}
	tcm.isolationManager.mutex.RUnlock()
	
	for _, env := range environments {
		for _, policy := range tcm.cleanupPolicies {
			if policy.Condition(env) {
				if err := policy.Action(env); err != nil {
					result.Errors = append(result.Errors, 
						fmt.Sprintf("cleanup policy %s failed for env %s: %v", policy.Name, env.ID, err))
				} else {
					result.EnvironmentsDeleted++
					log.Printf("Cleaned up environment %s using policy %s", env.ID, policy.Name)
				}
				break // Only apply first matching policy
			}
		}
	}
	
	return nil
}

// archiveEnvironment archives a test environment
func (tcm *TestDataCleanupManager) archiveEnvironment(env *TestEnvironment) error {
	log.Printf("Archiving environment %s", env.ID)
	
	// Collect environment data
	envData, err := tcm.collectEnvironmentData(env)
	if err != nil {
		return fmt.Errorf("failed to collect environment data: %w", err)
	}
	
	// Create archive metadata
	metadata := ArchiveMetadata{
		ArchiveID:       fmt.Sprintf("archive_%s_%d", env.ID, time.Now().Unix()),
		OriginalEnvID:   env.ID,
		TestSuite:       env.TestSuite,
		ArchivedAt:      time.Now(),
		OriginalSize:    int64(len(envData)),
		CompressionType: "gzip",
		DataTypes:       []string{"database", "cache", "filesystem"},
		Metadata: map[string]interface{}{
			"environment_status": env.Status,
			"created_at":        env.CreatedAt,
			"last_accessed_at":  env.LastAccessedAt,
			"resources":         env.Resources,
		},
	}
	
	// Compress data if enabled
	var finalData []byte
	if tcm.config.CompressionEnabled {
		compressedData, err := tcm.compressData(envData)
		if err != nil {
			return fmt.Errorf("failed to compress data: %w", err)
		}
		finalData = compressedData
		metadata.CompressedSize = int64(len(compressedData))
	} else {
		finalData = envData
		metadata.CompressedSize = metadata.OriginalSize
	}
	
	// Calculate checksum
	metadata.Checksum = tcm.calculateChecksum(finalData)
	
	// Store archive
	archiveKey := fmt.Sprintf("%s/%s.archive", env.TestSuite, metadata.ArchiveID)
	if err := tcm.archiveStorage.Store(archiveKey, finalData); err != nil {
		return fmt.Errorf("failed to store archive: %w", err)
	}
	
	// Store metadata
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	metadataKey := fmt.Sprintf("%s/%s.metadata", env.TestSuite, metadata.ArchiveID)
	if err := tcm.archiveStorage.Store(metadataKey, metadataJSON); err != nil {
		return fmt.Errorf("failed to store metadata: %w", err)
	}
	
	log.Printf("Environment %s archived as %s (compressed %d -> %d bytes)", 
		env.ID, metadata.ArchiveID, metadata.OriginalSize, metadata.CompressedSize)
	
	return nil
}

// collectEnvironmentData collects all data from a test environment
func (tcm *TestDataCleanupManager) collectEnvironmentData(env *TestEnvironment) ([]byte, error) {
	envData := make(map[string]interface{})
	
	// Collect database data
	if dbData, err := tcm.collectDatabaseData(env); err == nil {
		envData["database"] = dbData
	} else {
		log.Printf("Warning: failed to collect database data for env %s: %v", env.ID, err)
	}
	
	// Collect cache data
	if cacheData, err := tcm.collectCacheData(env); err == nil {
		envData["cache"] = cacheData
	} else {
		log.Printf("Warning: failed to collect cache data for env %s: %v", env.ID, err)
	}
	
	// Collect filesystem data
	if fsData, err := tcm.collectFileSystemData(env); err == nil {
		envData["filesystem"] = fsData
	} else {
		log.Printf("Warning: failed to collect filesystem data for env %s: %v", env.ID, err)
	}
	
	// Add environment metadata
	envData["environment"] = map[string]interface{}{
		"id":               env.ID,
		"test_suite":       env.TestSuite,
		"created_at":       env.CreatedAt,
		"last_accessed_at": env.LastAccessedAt,
		"status":           env.Status,
		"resources":        env.Resources,
		"metadata":         env.Metadata,
	}
	
	// Serialize to JSON
	jsonData, err := json.Marshal(envData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal environment data: %w", err)
	}
	
	return jsonData, nil
}

// collectDatabaseData collects database data from an environment
func (tcm *TestDataCleanupManager) collectDatabaseData(env *TestEnvironment) (map[string]interface{}, error) {
	db, err := tcm.isolationManager.GetDatabaseConnection(env.ID)
	if err != nil {
		return nil, err
	}
	
	data := make(map[string]interface{})
	
	// Get table list
	rows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}
	
	// Get row counts for each table
	tableCounts := make(map[string]int)
	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err == nil {
			tableCounts[table] = count
		}
	}
	
	data["tables"] = tables
	data["table_counts"] = tableCounts
	data["total_tables"] = len(tables)
	
	return data, nil
}

// collectCacheData collects cache data from an environment
func (tcm *TestDataCleanupManager) collectCacheData(env *TestEnvironment) (map[string]interface{}, error) {
	client, err := tcm.isolationManager.GetCacheClient(env.ID)
	if err != nil {
		return nil, err
	}
	
	ctx := context.Background()
	data := make(map[string]interface{})
	
	// Get all keys in the namespace
	keys, err := client.Keys(ctx, env.CacheNamespace+"*").Result()
	if err != nil {
		return nil, err
	}
	
	// Get key types and TTLs
	keyInfo := make(map[string]map[string]interface{})
	for _, key := range keys {
		info := make(map[string]interface{})
		
		// Get key type
		keyType, err := client.Type(ctx, key).Result()
		if err == nil {
			info["type"] = keyType
		}
		
		// Get TTL
		ttl, err := client.TTL(ctx, key).Result()
		if err == nil {
			info["ttl"] = ttl.Seconds()
		}
		
		keyInfo[key] = info
	}
	
	data["keys"] = keys
	data["key_info"] = keyInfo
	data["total_keys"] = len(keys)
	
	return data, nil
}

// collectFileSystemData collects filesystem data from an environment
func (tcm *TestDataCleanupManager) collectFileSystemData(env *TestEnvironment) (map[string]interface{}, error) {
	fsPath, err := tcm.isolationManager.GetFileSystemPath(env.ID)
	if err != nil {
		return nil, err
	}
	
	data := make(map[string]interface{})
	
	// Walk directory tree and collect file info
	var totalSize int64
	var fileCount int
	var dirCount int
	
	err = filepath.Walk(fsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			dirCount++
		} else {
			fileCount++
			totalSize += info.Size()
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	data["path"] = fsPath
	data["total_size_bytes"] = totalSize
	data["file_count"] = fileCount
	data["directory_count"] = dirCount
	
	return data, nil
}

// compressData compresses data using gzip
func (tcm *TestDataCleanupManager) compressData(data []byte) ([]byte, error) {
	var compressed strings.Builder
	writer := gzip.NewWriter(&compressed)
	
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	
	if err := writer.Close(); err != nil {
		return nil, err
	}
	
	return []byte(compressed.String()), nil
}

// decompressData decompresses gzip data
func (tcm *TestDataCleanupManager) decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	
	return decompressed, nil
}

// calculateChecksum calculates a simple checksum for data integrity
func (tcm *TestDataCleanupManager) calculateChecksum(data []byte) string {
	// Simple checksum implementation (in production, use proper hashing like SHA-256)
	sum := 0
	for _, b := range data {
		sum += int(b)
	}
	return fmt.Sprintf("%x", sum)
}

// cleanupOldArchives removes old archives based on retention policy
func (tcm *TestDataCleanupManager) cleanupOldArchives(result *CleanupResult) error {
	// List all archives
	archives, err := tcm.archiveStorage.List(".metadata")
	if err != nil {
		return fmt.Errorf("failed to list archives: %w", err)
	}
	
	cutoffTime := time.Now().Add(-tcm.config.RetentionPeriod)
	
	for _, archiveKey := range archives {
		// Load metadata
		metadataData, err := tcm.archiveStorage.Retrieve(archiveKey)
		if err != nil {
			continue
		}
		
		var metadata ArchiveMetadata
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			continue
		}
		
		// Check if archive is old enough to delete
		if metadata.ArchivedAt.Before(cutoffTime) {
			// Delete archive data
			archiveDataKey := strings.Replace(archiveKey, ".metadata", ".archive", 1)
			if err := tcm.archiveStorage.Delete(archiveDataKey); err != nil {
				result.Errors = append(result.Errors, 
					fmt.Sprintf("failed to delete archive data %s: %v", archiveDataKey, err))
			}
			
			// Delete metadata
			if err := tcm.archiveStorage.Delete(archiveKey); err != nil {
				result.Errors = append(result.Errors, 
					fmt.Sprintf("failed to delete archive metadata %s: %v", archiveKey, err))
			} else {
				result.DataSizeDeleted += metadata.CompressedSize
				log.Printf("Deleted old archive %s (archived at %v)", metadata.ArchiveID, metadata.ArchivedAt)
			}
		}
	}
	
	return nil
}

// RestoreEnvironment restores a test environment from archive
func (tcm *TestDataCleanupManager) RestoreEnvironment(archiveID string) (*TestEnvironment, error) {
	log.Printf("Restoring environment from archive %s", archiveID)
	
	// Find archive by ID
	archives, err := tcm.archiveStorage.List(archiveID)
	if err != nil {
		return nil, fmt.Errorf("failed to list archives: %w", err)
	}
	
	var metadataKey string
	for _, key := range archives {
		if strings.Contains(key, archiveID) && strings.HasSuffix(key, ".metadata") {
			metadataKey = key
			break
		}
	}
	
	if metadataKey == "" {
		return nil, fmt.Errorf("archive %s not found", archiveID)
	}
	
	// Load metadata
	metadataData, err := tcm.archiveStorage.Retrieve(metadataKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve metadata: %w", err)
	}
	
	var metadata ArchiveMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	
	// Load archive data
	archiveDataKey := strings.Replace(metadataKey, ".metadata", ".archive", 1)
	archiveData, err := tcm.archiveStorage.Retrieve(archiveDataKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve archive data: %w", err)
	}
	
	// Verify checksum
	if tcm.calculateChecksum(archiveData) != metadata.Checksum {
		return nil, fmt.Errorf("archive data checksum mismatch")
	}
	
	// Decompress if needed
	var envData []byte
	if metadata.CompressionType == "gzip" {
		envData, err = tcm.decompressData(archiveData)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress data: %w", err)
		}
	} else {
		envData = archiveData
	}
	
	// Parse environment data
	var envDataMap map[string]interface{}
	if err := json.Unmarshal(envData, &envDataMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environment data: %w", err)
	}
	
	// Create new environment
	newEnv, err := tcm.isolationManager.CreateIsolatedEnvironment(metadata.TestSuite)
	if err != nil {
		return nil, fmt.Errorf("failed to create new environment: %w", err)
	}
	
	// Restore data to new environment
	if err := tcm.restoreEnvironmentData(newEnv, envDataMap); err != nil {
		// Cleanup failed environment
		tcm.isolationManager.CleanupEnvironment(newEnv.ID)
		return nil, fmt.Errorf("failed to restore environment data: %w", err)
	}
	
	log.Printf("Environment restored from archive %s as new environment %s", archiveID, newEnv.ID)
	return newEnv, nil
}

// restoreEnvironmentData restores data to a test environment
func (tcm *TestDataCleanupManager) restoreEnvironmentData(env *TestEnvironment, data map[string]interface{}) error {
	// Restore database data
	if dbData, exists := data["database"]; exists {
		if err := tcm.restoreDatabaseData(env, dbData.(map[string]interface{})); err != nil {
			return fmt.Errorf("failed to restore database data: %w", err)
		}
	}
	
	// Restore cache data
	if cacheData, exists := data["cache"]; exists {
		if err := tcm.restoreCacheData(env, cacheData.(map[string]interface{})); err != nil {
			return fmt.Errorf("failed to restore cache data: %w", err)
		}
	}
	
	// Restore filesystem data
	if fsData, exists := data["filesystem"]; exists {
		if err := tcm.restoreFileSystemData(env, fsData.(map[string]interface{})); err != nil {
			return fmt.Errorf("failed to restore filesystem data: %w", err)
		}
	}
	
	return nil
}

// restoreDatabaseData restores database data (placeholder implementation)
func (tcm *TestDataCleanupManager) restoreDatabaseData(env *TestEnvironment, data map[string]interface{}) error {
	// In a real implementation, this would restore actual table data
	// For now, we just log the restoration
	log.Printf("Restoring database data for environment %s", env.ID)
	return nil
}

// restoreCacheData restores cache data (placeholder implementation)
func (tcm *TestDataCleanupManager) restoreCacheData(env *TestEnvironment, data map[string]interface{}) error {
	// In a real implementation, this would restore actual cache keys
	// For now, we just log the restoration
	log.Printf("Restoring cache data for environment %s", env.ID)
	return nil
}

// restoreFileSystemData restores filesystem data (placeholder implementation)
func (tcm *TestDataCleanupManager) restoreFileSystemData(env *TestEnvironment, data map[string]interface{}) error {
	// In a real implementation, this would restore actual files
	// For now, we just log the restoration
	log.Printf("Restoring filesystem data for environment %s", env.ID)
	return nil
}

// GetArchiveStats returns statistics about archived data
func (tcm *TestDataCleanupManager) GetArchiveStats() (map[string]interface{}, error) {
	archives, err := tcm.archiveStorage.List(".metadata")
	if err != nil {
		return nil, fmt.Errorf("failed to list archives: %w", err)
	}
	
	stats := map[string]interface{}{
		"total_archives":     len(archives),
		"total_size_bytes":   int64(0),
		"by_test_suite":      make(map[string]int),
		"by_compression":     make(map[string]int),
		"oldest_archive":     time.Now(),
		"newest_archive":     time.Time{},
	}
	
	var totalSize int64
	suiteCounts := make(map[string]int)
	compressionCounts := make(map[string]int)
	oldestTime := time.Now()
	var newestTime time.Time
	
	for _, archiveKey := range archives {
		// Load metadata
		metadataData, err := tcm.archiveStorage.Retrieve(archiveKey)
		if err != nil {
			continue
		}
		
		var metadata ArchiveMetadata
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			continue
		}
		
		totalSize += metadata.CompressedSize
		suiteCounts[metadata.TestSuite]++
		compressionCounts[metadata.CompressionType]++
		
		if metadata.ArchivedAt.Before(oldestTime) {
			oldestTime = metadata.ArchivedAt
		}
		if metadata.ArchivedAt.After(newestTime) {
			newestTime = metadata.ArchivedAt
		}
	}
	
	stats["total_size_bytes"] = totalSize
	stats["by_test_suite"] = suiteCounts
	stats["by_compression"] = compressionCounts
	stats["oldest_archive"] = oldestTime
	stats["newest_archive"] = newestTime
	
	return stats, nil
}

// AddCleanupPolicy adds a custom cleanup policy
func (tcm *TestDataCleanupManager) AddCleanupPolicy(policy CleanupPolicy) {
	tcm.mutex.Lock()
	defer tcm.mutex.Unlock()
	
	tcm.cleanupPolicies = append(tcm.cleanupPolicies, policy)
}

// AddArchivePolicy adds a custom archive policy
func (tcm *TestDataCleanupManager) AddArchivePolicy(policy ArchivePolicy) {
	tcm.mutex.Lock()
	defer tcm.mutex.Unlock()
	
	tcm.archivePolicies = append(tcm.archivePolicies, policy)
}

// ValidateArchiveIntegrity validates the integrity of archived data
func (tcm *TestDataCleanupManager) ValidateArchiveIntegrity() error {
	archives, err := tcm.archiveStorage.List(".metadata")
	if err != nil {
		return fmt.Errorf("failed to list archives: %w", err)
	}
	
	var errors []string
	
	for _, archiveKey := range archives {
		// Load metadata
		metadataData, err := tcm.archiveStorage.Retrieve(archiveKey)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to retrieve metadata %s: %v", archiveKey, err))
			continue
		}
		
		var metadata ArchiveMetadata
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			errors = append(errors, fmt.Sprintf("failed to unmarshal metadata %s: %v", archiveKey, err))
			continue
		}
		
		// Load and verify archive data
		archiveDataKey := strings.Replace(archiveKey, ".metadata", ".archive", 1)
		archiveData, err := tcm.archiveStorage.Retrieve(archiveDataKey)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to retrieve archive data %s: %v", archiveDataKey, err))
			continue
		}
		
		// Verify checksum
		if tcm.calculateChecksum(archiveData) != metadata.Checksum {
			errors = append(errors, fmt.Sprintf("checksum mismatch for archive %s", metadata.ArchiveID))
		}
		
		// Verify size
		if int64(len(archiveData)) != metadata.CompressedSize {
			errors = append(errors, fmt.Sprintf("size mismatch for archive %s", metadata.ArchiveID))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("archive integrity validation failed: %s", strings.Join(errors, "; "))
	}
	
	log.Printf("Archive integrity validation passed for %d archives", len(archives))
	return nil
}