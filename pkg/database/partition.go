package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type PartitionManager struct {
	db              *sql.DB
	retentionDays   int
	schedulerStop   chan bool
	schedulerActive bool
}

func NewPartitionManager(db *sql.DB) *PartitionManager {
	return &PartitionManager{
		db:            db,
		retentionDays: 30, // Default 30 days retention
		schedulerStop: make(chan bool),
	}
}

// SetRetentionDays configures the retention period for old partitions
func (pm *PartitionManager) SetRetentionDays(days int) {
	if days > 0 {
		pm.retentionDays = days
	}
}

// CreateDailyPartitions creates daily partitions for the next 7 days with proper error handling
func (pm *PartitionManager) CreateDailyPartitions() error {
	// First, verify that the articles table exists and is partitioned
	var isPartitioned bool
	err := pm.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_partitioned_table pt
			JOIN pg_class c ON pt.partrelid = c.oid
			WHERE c.relname = 'articles'
		)
	`).Scan(&isPartitioned)
	if err != nil {
		return fmt.Errorf("failed to check if articles table is partitioned: %w", err)
	}
	if !isPartitioned {
		return fmt.Errorf("articles table is not partitioned - cannot create daily partitions")
	}

	createPartitionSQL := `
		CREATE OR REPLACE FUNCTION create_daily_partitions()
		RETURNS TABLE(partition_name text, status text, error_message text) AS $$
		DECLARE
			start_date date;
			end_date date;
			part_name text;
			created_count integer := 0;
			error_count integer := 0;
		BEGIN
			-- Create partitions for next 7 days
			FOR i IN 0..6 LOOP
				start_date := CURRENT_DATE + i;
				end_date := start_date + interval '1 day';
				part_name := 'articles_' || to_char(start_date, 'YYYY_MM_DD');
				
				-- Check if partition already exists
				IF NOT EXISTS (
					SELECT 1 FROM pg_class WHERE relname = part_name
				) THEN
					BEGIN
						-- Create the partition
						EXECUTE format('CREATE TABLE %I PARTITION OF articles FOR VALUES FROM (%L) TO (%L)',
							part_name, start_date, end_date);
						
						-- Create optimized indexes for daily partitions
						EXECUTE format('CREATE INDEX idx_%I_published_at ON %I (published_at DESC) WHERE status = ''published''',
							part_name, part_name);
						EXECUTE format('CREATE INDEX idx_%I_category ON %I (category_id, published_at DESC)',
							part_name, part_name);
						EXECUTE format('CREATE INDEX idx_%I_author ON %I (author_id, published_at DESC)',
							part_name, part_name);
						EXECUTE format('CREATE INDEX idx_%I_slug ON %I (slug) WHERE status = ''published''',
							part_name, part_name);
						
						-- BRIN index for time-series performance on daily partitions
						EXECUTE format('CREATE INDEX idx_%I_published_brin ON %I USING BRIN (published_at) WITH (pages_per_range = 32)',
							part_name, part_name);
						
						-- Full-text search index
						EXECUTE format('CREATE INDEX idx_%I_search ON %I USING gin(to_tsvector(''english'', title || '' '' || content))',
							part_name, part_name);
							
						created_count := created_count + 1;
						partition_name := part_name;
						status := 'created';
						error_message := NULL;
						RETURN NEXT;
						
					EXCEPTION
						WHEN duplicate_table THEN
							partition_name := part_name;
							status := 'exists';
							error_message := 'Partition already exists';
							RETURN NEXT;
						WHEN OTHERS THEN
							error_count := error_count + 1;
							partition_name := part_name;
							status := 'error';
							error_message := SQLERRM;
							RETURN NEXT;
					END;
				ELSE
					partition_name := part_name;
					status := 'exists';
					error_message := 'Partition already exists';
					RETURN NEXT;
				END IF;
			END LOOP;
			
			-- Return summary
			partition_name := 'SUMMARY';
			status := format('Created: %s, Errors: %s', created_count, error_count);
			error_message := NULL;
			RETURN NEXT;
		END;
		$$ LANGUAGE plpgsql;
	`

	// Create the function
	if _, err := pm.db.Exec(createPartitionSQL); err != nil {
		return fmt.Errorf("failed to create daily partitions function: %w", err)
	}

	// Execute the function and collect results
	rows, err := pm.db.Query("SELECT * FROM create_daily_partitions()")
	if err != nil {
		return fmt.Errorf("failed to execute daily partitions creation: %w", err)
	}
	defer rows.Close()

	var hasErrors bool
	var createdCount, errorCount int

	for rows.Next() {
		var partitionName, status, errorMessage sql.NullString
		if err := rows.Scan(&partitionName, &status, &errorMessage); err != nil {
			return fmt.Errorf("failed to scan partition creation results: %w", err)
		}

		if partitionName.String == "SUMMARY" {
			log.Printf("Daily partition creation summary: %s", status.String)
		} else if status.String == "created" {
			createdCount++
			log.Printf("Created daily partition: %s", partitionName.String)
		} else if status.String == "error" {
			errorCount++
			hasErrors = true
			log.Printf("Failed to create partition %s: %s", partitionName.String, errorMessage.String)
		} else if status.String == "exists" {
			log.Printf("Daily partition already exists: %s", partitionName.String)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error reading partition creation results: %w", err)
	}

	if hasErrors {
		return fmt.Errorf("daily partition creation completed with %d errors", errorCount)
	}

	log.Printf("Daily partitions created successfully: %d new partitions", createdCount)
	return nil
}

// DropOldPartitions removes partitions older than the specified retention period
func (pm *PartitionManager) DropOldPartitions(retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = pm.retentionDays // Use configured retention period
	}

	dropPartitionsSQL := `
		CREATE OR REPLACE FUNCTION drop_old_partitions(retention_days integer DEFAULT 30)
		RETURNS TABLE(partition_name text, status text, error_message text) AS $$
		DECLARE
			partition_record RECORD;
			cutoff_date date;
			dropped_count integer := 0;
			error_count integer := 0;
		BEGIN
			cutoff_date := CURRENT_DATE - retention_days;
			
			-- Find and drop old article partitions
			FOR partition_record IN
				SELECT schemaname, tablename 
				FROM pg_tables 
				WHERE tablename ~ '^articles_\d{4}_\d{2}(_\d{2})?$'
				AND schemaname = 'public'
			LOOP
				-- Extract date from partition name and check if it's old enough
				DECLARE
					partition_date date;
					date_part text;
				BEGIN
					-- Extract date part from partition name (articles_YYYY_MM or articles_YYYY_MM_DD)
					date_part := regexp_replace(partition_record.tablename, '^articles_', '');
					
					-- Handle both monthly (YYYY_MM) and daily (YYYY_MM_DD) partitions
					IF date_part ~ '^\d{4}_\d{2}_\d{2}$' THEN
						-- Daily partition format
						partition_date := to_date(date_part, 'YYYY_MM_DD');
					ELSIF date_part ~ '^\d{4}_\d{2}$' THEN
						-- Monthly partition format - use first day of month
						partition_date := to_date(date_part || '_01', 'YYYY_MM_DD');
					ELSE
						CONTINUE; -- Skip if format doesn't match
					END IF;
					
					IF partition_date < cutoff_date THEN
						BEGIN
							EXECUTE format('DROP TABLE IF EXISTS %I.%I CASCADE',
								partition_record.schemaname, partition_record.tablename);
							dropped_count := dropped_count + 1;
							partition_name := partition_record.tablename;
							status := 'dropped';
							error_message := NULL;
							RETURN NEXT;
						EXCEPTION
							WHEN OTHERS THEN
								error_count := error_count + 1;
								partition_name := partition_record.tablename;
								status := 'error';
								error_message := SQLERRM;
								RETURN NEXT;
						END;
					END IF;
				EXCEPTION
					WHEN OTHERS THEN
						error_count := error_count + 1;
						partition_name := partition_record.tablename;
						status := 'error';
						error_message := 'Failed to process partition: ' || SQLERRM;
						RETURN NEXT;
				END;
			END LOOP;
			
			-- Drop old article_tags partitions
			FOR partition_record IN
				SELECT schemaname, tablename 
				FROM pg_tables 
				WHERE tablename ~ '^article_tags_\d{4}_\d{2}$'
				AND schemaname = 'public'
			LOOP
				DECLARE
					partition_date date;
					date_part text;
				BEGIN
					date_part := regexp_replace(partition_record.tablename, '^article_tags_', '');
					partition_date := to_date(date_part || '_01', 'YYYY_MM_DD');
					
					IF partition_date < cutoff_date THEN
						BEGIN
							EXECUTE format('DROP TABLE IF EXISTS %I.%I CASCADE',
								partition_record.schemaname, partition_record.tablename);
							dropped_count := dropped_count + 1;
							partition_name := partition_record.tablename;
							status := 'dropped';
							error_message := NULL;
							RETURN NEXT;
						EXCEPTION
							WHEN OTHERS THEN
								error_count := error_count + 1;
								partition_name := partition_record.tablename;
								status := 'error';
								error_message := SQLERRM;
								RETURN NEXT;
						END;
					END IF;
				EXCEPTION
					WHEN OTHERS THEN
						error_count := error_count + 1;
						partition_name := partition_record.tablename;
						status := 'error';
						error_message := 'Failed to process article_tags partition: ' || SQLERRM;
						RETURN NEXT;
				END;
			END LOOP;
			
			-- Drop old article_views partitions
			FOR partition_record IN
				SELECT schemaname, tablename 
				FROM pg_tables 
				WHERE tablename ~ '^article_views_\d{4}_\d{2}$'
				AND schemaname = 'public'
			LOOP
				DECLARE
					partition_date date;
					date_part text;
				BEGIN
					date_part := regexp_replace(partition_record.tablename, '^article_views_', '');
					partition_date := to_date(date_part || '_01', 'YYYY_MM_DD');
					
					IF partition_date < cutoff_date THEN
						BEGIN
							EXECUTE format('DROP TABLE IF EXISTS %I.%I CASCADE',
								partition_record.schemaname, partition_record.tablename);
							dropped_count := dropped_count + 1;
							partition_name := partition_record.tablename;
							status := 'dropped';
							error_message := NULL;
							RETURN NEXT;
						EXCEPTION
							WHEN OTHERS THEN
								error_count := error_count + 1;
								partition_name := partition_record.tablename;
								status := 'error';
								error_message := SQLERRM;
								RETURN NEXT;
						END;
					END IF;
				EXCEPTION
					WHEN OTHERS THEN
						error_count := error_count + 1;
						partition_name := partition_record.tablename;
						status := 'error';
						error_message := 'Failed to process article_views partition: ' || SQLERRM;
						RETURN NEXT;
				END;
			END LOOP;
			
			-- Return summary
			partition_name := 'SUMMARY';
			status := format('Dropped: %s, Errors: %s', dropped_count, error_count);
			error_message := NULL;
			RETURN NEXT;
		END;
		$$ LANGUAGE plpgsql;
	`

	// Create the function
	if _, err := pm.db.Exec(dropPartitionsSQL); err != nil {
		return fmt.Errorf("failed to create drop partitions function: %w", err)
	}

	// Execute the function and collect results
	rows, err := pm.db.Query("SELECT * FROM drop_old_partitions($1)", retentionDays)
	if err != nil {
		return fmt.Errorf("failed to execute partition cleanup: %w", err)
	}
	defer rows.Close()

	var hasErrors bool
	var droppedCount, errorCount int

	for rows.Next() {
		var partitionName, status, errorMessage sql.NullString
		if err := rows.Scan(&partitionName, &status, &errorMessage); err != nil {
			return fmt.Errorf("failed to scan partition cleanup results: %w", err)
		}

		if partitionName.String == "SUMMARY" {
			log.Printf("Partition cleanup summary: %s", status.String)
		} else if status.String == "dropped" {
			droppedCount++
			log.Printf("Dropped old partition: %s", partitionName.String)
		} else if status.String == "error" {
			errorCount++
			hasErrors = true
			log.Printf("Failed to drop partition %s: %s", partitionName.String, errorMessage.String)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error reading partition cleanup results: %w", err)
	}

	if hasErrors {
		log.Printf("Partition cleanup completed with %d errors", errorCount)
		// Don't return error for cleanup failures, just log them
	}

	log.Printf("Old partitions cleanup completed: %d partitions dropped (retention: %d days)", droppedCount, retentionDays)
	return nil
}

// SchedulePartitionMaintenance sets up automatic partition maintenance
func (pm *PartitionManager) SchedulePartitionMaintenance() error {
	// Create a function that can be called by cron or scheduled jobs
	maintenanceSQL := `
		CREATE OR REPLACE FUNCTION partition_maintenance()
		RETURNS void AS $$
		BEGIN
			-- Create partitions for next 7 days
			PERFORM create_daily_partitions();
			
			-- Drop partitions older than 30 days
			PERFORM drop_old_partitions(30);
			
			-- Log maintenance completion
			RAISE NOTICE 'Partition maintenance completed at %', NOW();
		END;
		$$ LANGUAGE plpgsql;
	`

	if _, err := pm.db.Exec(maintenanceSQL); err != nil {
		return fmt.Errorf("failed to create partition maintenance function: %w", err)
	}

	log.Println("Partition maintenance function created successfully")
	return nil
}

// RunMaintenance executes the partition maintenance function
func (pm *PartitionManager) RunMaintenance() error {
	if _, err := pm.db.Exec("SELECT partition_maintenance()"); err != nil {
		return fmt.Errorf("failed to run partition maintenance: %w", err)
	}

	log.Println("Partition maintenance executed successfully")
	return nil
}

// GetPartitionInfo returns information about existing partitions
func (pm *PartitionManager) GetPartitionInfo() ([]PartitionInfo, error) {
	query := `
		SELECT 
			schemaname,
			tablename,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
			(SELECT count(*) FROM information_schema.tables WHERE table_name = t.tablename) as exists
		FROM pg_tables t
		WHERE tablename ~ '^(articles|article_tags|article_views)_\d{4}_\d{2}(_\d{2})?$'
		AND schemaname = 'public'
		ORDER BY tablename;
	`

	rows, err := pm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get partition info: %w", err)
	}
	defer rows.Close()

	var partitions []PartitionInfo
	for rows.Next() {
		var p PartitionInfo
		if err := rows.Scan(&p.Schema, &p.Name, &p.Size, &p.Exists); err != nil {
			return nil, fmt.Errorf("failed to scan partition info: %w", err)
		}
		partitions = append(partitions, p)
	}

	return partitions, nil
}

type PartitionInfo struct {
	Schema string
	Name   string
	Size   string
	Exists bool
}

// StartPartitionScheduler starts a goroutine that runs partition maintenance daily
func (pm *PartitionManager) StartPartitionScheduler() {
	if pm.schedulerActive {
		log.Println("Partition scheduler is already running")
		return
	}

	pm.schedulerActive = true
	go func() {
		defer func() {
			pm.schedulerActive = false
		}()

		ticker := time.NewTicker(24 * time.Hour) // Run daily
		defer ticker.Stop()

		// Run maintenance immediately on start
		if err := pm.RunMaintenance(); err != nil {
			log.Printf("Initial partition maintenance failed: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := pm.RunMaintenance(); err != nil {
					log.Printf("Scheduled partition maintenance failed: %v", err)
				}
			case <-pm.schedulerStop:
				log.Println("Partition scheduler stopped")
				return
			}
		}
	}()

	log.Println("Partition scheduler started (runs daily)")
}

// StopPartitionScheduler stops the partition maintenance scheduler
func (pm *PartitionManager) StopPartitionScheduler() {
	if pm.schedulerActive {
		pm.schedulerStop <- true
		log.Println("Partition scheduler stop signal sent")
	}
}

// IsSchedulerActive returns whether the partition scheduler is currently running
func (pm *PartitionManager) IsSchedulerActive() bool {
	return pm.schedulerActive
}