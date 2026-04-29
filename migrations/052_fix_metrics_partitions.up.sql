-- Migration 052: Fix metrics table partitions
-- This migration:
-- 1. Creates missing partitions for system_metrics, database_metrics, cache_metrics, publishing_metrics
-- 2. Updates the partition creation function to target the correct tables (not _history variants)

-- Drop the old function that targets wrong tables
DROP FUNCTION IF EXISTS create_daily_monitoring_partitions();

-- Create new function that targets the correct tables
CREATE OR REPLACE FUNCTION create_daily_monitoring_partitions()
RETURNS void AS $$
DECLARE
    start_date date;
    end_date date;
    table_name text;
    partition_name text;
BEGIN
    -- Create partitions for the next 14 days (gives buffer for weekends/holidays)
    FOR i IN 0..13 LOOP
        start_date := CURRENT_DATE + i;
        end_date := start_date + 1;

        -- Create partitions for each metrics table (the actual tables, not _history)
        FOREACH table_name IN ARRAY ARRAY['system_metrics', 'database_metrics', 'cache_metrics', 'publishing_metrics'] LOOP
            partition_name := table_name || '_' || to_char(start_date, 'YYYY_MM_DD');

            -- Check if partition already exists
            IF NOT EXISTS (
                SELECT 1 FROM pg_class c
                JOIN pg_namespace n ON n.oid = c.relnamespace
                WHERE c.relname = partition_name AND n.nspname = 'public'
            ) THEN
                BEGIN
                    EXECUTE format('CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                        partition_name, table_name, start_date, end_date);
                    RAISE NOTICE 'Created partition: %', partition_name;
                EXCEPTION
                    WHEN duplicate_table THEN
                        -- Ignore if already exists (race condition)
                        NULL;
                    WHEN OTHERS THEN
                        RAISE WARNING 'Failed to create partition %: %', partition_name, SQLERRM;
                END;
            END IF;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create partitions for current date through next 14 days
SELECT create_daily_monitoring_partitions();

-- Also create partitions for any missing dates between last partition and today
DO $$
DECLARE
    table_name text;
    last_partition_date date;
    current_date_val date := CURRENT_DATE;
    partition_date date;
    partition_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY['system_metrics', 'database_metrics', 'cache_metrics', 'publishing_metrics'] LOOP
        -- Find the latest partition date for this table
        SELECT MAX(to_date(right(tablename, 10), 'YYYY_MM_DD'))
        INTO last_partition_date
        FROM pg_tables
        WHERE tablename LIKE table_name || '_%'
        AND tablename ~ '\d{4}_\d{2}_\d{2}$';
        
        IF last_partition_date IS NOT NULL AND last_partition_date < current_date_val THEN
            -- Create missing partitions between last partition and today
            partition_date := last_partition_date + 1;
            WHILE partition_date <= current_date_val LOOP
                partition_name := table_name || '_' || to_char(partition_date, 'YYYY_MM_DD');
                
                IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = partition_name) THEN
                    BEGIN
                        EXECUTE format('CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                            partition_name, table_name, partition_date, partition_date + 1);
                        RAISE NOTICE 'Created missing partition: %', partition_name;
                    EXCEPTION
                        WHEN duplicate_table THEN NULL;
                        WHEN OTHERS THEN
                            RAISE WARNING 'Failed to create partition %: %', partition_name, SQLERRM;
                    END;
                END IF;
                
                partition_date := partition_date + 1;
            END LOOP;
        END IF;
    END LOOP;
END $$;

-- Add comment documenting the fix
COMMENT ON FUNCTION create_daily_monitoring_partitions() IS 
'Creates daily partitions for metrics tables (system_metrics, database_metrics, cache_metrics, publishing_metrics). 
Should be called regularly to ensure partitions exist for upcoming dates. 
The application also creates partitions on-demand when writing metrics.';
