package repositories

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
	"high-performance-news-website/pkg/database"
)

// PartitionManager handles automatic partition creation for all partitioned tables
type PartitionManager struct {
	db *database.DB
}

// NewPartitionManager creates a new partition manager
func NewPartitionManager(db *database.DB) *PartitionManager {
	return &PartitionManager{
		db: db,
	}
}

// PartitionedTable represents a table that needs partitioning
type PartitionedTable struct {
	TableName    string
	DateColumn   string
	PartitionKey string // Format for partition naming (e.g., "2006_01_02")
}

// GetPartitionedTables returns all tables that need partitioning
func (pm *PartitionManager) GetPartitionedTables() []PartitionedTable {
	return []PartitionedTable{
		{
			TableName:    "articles",
			DateColumn:   "published_at",
			PartitionKey: "2006_01_02",
		},
		{
			TableName:    "system_metrics",
			DateColumn:   "timestamp",
			PartitionKey: "2006_01_02",
		},
		{
			TableName:    "database_metrics",
			DateColumn:   "timestamp",
			PartitionKey: "2006_01_02",
		},
		{
			TableName:    "cache_metrics",
			DateColumn:   "timestamp",
			PartitionKey: "2006_01_02",
		},
		{
			TableName:    "publishing_metrics",
			DateColumn:   "timestamp",
			PartitionKey: "2006_01_02",
		},
	}
}

// EnsurePartitionExists creates a partition for the given table and date if it doesn't exist
func (pm *PartitionManager) EnsurePartitionExists(ctx context.Context, tableName string, targetDate time.Time) error {
	// Format partition name
	partitionName := fmt.Sprintf("%s_%s", tableName, targetDate.Format("2006_01_02"))
	
	// Check if partition already exists
	var exists bool
	err := pm.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_class WHERE relname = $1
		)
	`, partitionName).Scan(&exists)
	
	if err != nil {
		return fmt.Errorf("failed to check partition existence: %w", err)
	}
	
	if exists {
		return nil // Partition already exists
	}
	
	// Create the partition
	startDate := targetDate.Format("2006-01-02")
	endDate := targetDate.AddDate(0, 0, 1).Format("2006-01-02")
	
	createSQL := fmt.Sprintf(`
		CREATE TABLE %s PARTITION OF %s 
		FOR VALUES FROM ('%s') TO ('%s')
	`, partitionName, tableName, startDate, endDate)
	
	_, err = pm.db.ExecContext(ctx, createSQL)
	if err != nil {
		// Check if it's a "relation already exists" error (race condition)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "42P07" {
			// Partition was created by another process, that's fine
			return nil
		}
		return fmt.Errorf("failed to create partition %s: %w", partitionName, err)
	}
	
	log.Printf("Created partition %s for date %s", partitionName, startDate)
	return nil
}

// EnsurePartitionsForDate creates partitions for all partitioned tables for a specific date
func (pm *PartitionManager) EnsurePartitionsForDate(ctx context.Context, targetDate time.Time) error {
	tables := pm.GetPartitionedTables()
	
	for _, table := range tables {
		if err := pm.EnsurePartitionExists(ctx, table.TableName, targetDate); err != nil {
			log.Printf("Warning: Failed to create partition for %s: %v", table.TableName, err)
			// Continue with other tables
		}
	}
	
	return nil
}

// CreateDailyPartitions creates partitions for the next N days for all tables
func (pm *PartitionManager) CreateDailyPartitions(ctx context.Context, days int) error {
	now := time.Now()
	
	for i := 0; i < days; i++ {
		targetDate := now.AddDate(0, 0, i)
		if err := pm.EnsurePartitionsForDate(ctx, targetDate); err != nil {
			log.Printf("Warning: Failed to create partitions for date %s: %v", targetDate.Format("2006-01-02"), err)
		}
	}
	
	return nil
}

// CleanupOldPartitions removes partitions older than the specified number of days
func (pm *PartitionManager) CleanupOldPartitions(ctx context.Context, retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	tables := pm.GetPartitionedTables()
	
	for _, table := range tables {
		if err := pm.cleanupTablePartitions(ctx, table.TableName, cutoffDate); err != nil {
			log.Printf("Warning: Failed to cleanup partitions for %s: %v", table.TableName, err)
		}
	}
	
	return nil
}

// cleanupTablePartitions removes old partitions for a specific table
func (pm *PartitionManager) cleanupTablePartitions(ctx context.Context, tableName string, cutoffDate time.Time) error {
	// Get all partitions for this table
	query := `
		SELECT schemaname, tablename 
		FROM pg_tables 
		WHERE tablename LIKE $1 
		AND schemaname = 'public'
	`
	
	rows, err := pm.db.QueryContext(ctx, query, tableName+"_%")
	if err != nil {
		return fmt.Errorf("failed to query partitions: %w", err)
	}
	defer rows.Close()
	
	var partitionsToDelete []string
	
	for rows.Next() {
		var schemaName, partitionName string
		if err := rows.Scan(&schemaName, &partitionName); err != nil {
			continue
		}
		
		// Extract date from partition name (e.g., articles_2024_01_01)
		if len(partitionName) >= len(tableName)+11 { // tablename + "_YYYY_MM_DD"
			dateStr := partitionName[len(tableName)+1:] // Remove "tablename_"
			if partitionDate, err := time.Parse("2006_01_02", dateStr); err == nil {
				if partitionDate.Before(cutoffDate) {
					partitionsToDelete = append(partitionsToDelete, partitionName)
				}
			}
		}
	}
	
	// Delete old partitions
	for _, partitionName := range partitionsToDelete {
		dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", partitionName)
		if _, err := pm.db.ExecContext(ctx, dropSQL); err != nil {
			log.Printf("Warning: Failed to drop partition %s: %v", partitionName, err)
		} else {
			log.Printf("Dropped old partition: %s", partitionName)
		}
	}
	
	return nil
}