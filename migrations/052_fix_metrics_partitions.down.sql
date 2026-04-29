-- Migration 052 down: Revert metrics partition fix
-- Note: This does NOT drop the created partitions as that would lose data
-- It only reverts the function to its previous (broken) state

DROP FUNCTION IF EXISTS create_daily_monitoring_partitions();

-- Restore the old (broken) function that targets _history tables
CREATE OR REPLACE FUNCTION create_daily_monitoring_partitions()
RETURNS void AS $$
DECLARE
    start_date date;
    end_date date;
    table_name text;
    partition_name text;
BEGIN
    FOR i IN 0..6 LOOP
        start_date := CURRENT_DATE + i;
        end_date := start_date + 1;

        FOREACH table_name IN ARRAY ARRAY['log_entries', 'system_metrics_history', 'database_metrics_history', 'cache_metrics_history', 'publishing_metrics_history'] LOOP
            partition_name := table_name || '_' || to_char(start_date, 'YYYY_MM_DD');

            IF NOT EXISTS (
                SELECT 1 FROM pg_class c
                JOIN pg_namespace n ON n.oid = c.relnamespace
                WHERE c.relname = partition_name AND n.nspname = 'public'
            ) THEN
                EXECUTE format('CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                    partition_name, table_name, start_date, end_date);
                RAISE NOTICE 'Created partition: %', partition_name;
            END IF;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;
