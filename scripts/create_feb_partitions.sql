-- Create February 2026 partitions for metrics tables
-- Run with: sudo -u newsapp psql -d newsdb -f /home/newsapp/news-website/scripts/create_feb_partitions.sql

\echo '=== Creating February 2026 Partitions ==='

-- System metrics partitions
DO $$
DECLARE
    start_date date;
    end_date date;
    partition_name text;
BEGIN
    FOR i IN 1..28 LOOP
        start_date := '2026-02-01'::date + (i-1);
        end_date := start_date + 1;
        partition_name := 'system_metrics_' || to_char(start_date, 'YYYY_MM_DD');
        
        -- Check if partition exists
        IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = partition_name) THEN
            EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF system_metrics FOR VALUES FROM (%L) TO (%L)',
                partition_name, start_date, end_date);
            RAISE NOTICE 'Created partition: %', partition_name;
        END IF;
    END LOOP;
END $$;

-- Database metrics partitions  
DO $$
DECLARE
    start_date date;
    end_date date;
    partition_name text;
BEGIN
    FOR i IN 1..28 LOOP
        start_date := '2026-02-01'::date + (i-1);
        end_date := start_date + 1;
        partition_name := 'database_metrics_history_' || to_char(start_date, 'YYYY_MM_DD');
        
        IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = partition_name) THEN
            EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF database_metrics_history FOR VALUES FROM (%L) TO (%L)',
                partition_name, start_date, end_date);
            RAISE NOTICE 'Created partition: %', partition_name;
        END IF;
    END LOOP;
END $$;

-- Cache metrics partitions
DO $$
DECLARE
    start_date date;
    end_date date;
    partition_name text;
BEGIN
    FOR i IN 1..28 LOOP
        start_date := '2026-02-01'::date + (i-1);
        end_date := start_date + 1;
        partition_name := 'cache_metrics_history_' || to_char(start_date, 'YYYY_MM_DD');
        
        IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = partition_name) THEN
            EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF cache_metrics_history FOR VALUES FROM (%L) TO (%L)',
                partition_name, start_date, end_date);
            RAISE NOTICE 'Created partition: %', partition_name;
        END IF;
    END LOOP;
END $$;

-- Publishing metrics partitions
DO $$
DECLARE
    start_date date;
    end_date date;
    partition_name text;
BEGIN
    FOR i IN 1..28 LOOP
        start_date := '2026-02-01'::date + (i-1);
        end_date := start_date + 1;
        partition_name := 'publishing_metrics_history_' || to_char(start_date, 'YYYY_MM_DD');
        
        IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = partition_name) THEN
            EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF publishing_metrics_history FOR VALUES FROM (%L) TO (%L)',
                partition_name, start_date, end_date);
            RAISE NOTICE 'Created partition: %', partition_name;
        END IF;
    END LOOP;
END $$;

-- Articles partitions for February
DO $$
DECLARE
    start_date date;
    end_date date;
    partition_name text;
BEGIN
    FOR i IN 1..28 LOOP
        start_date := '2026-02-01'::date + (i-1);
        end_date := start_date + 1;
        partition_name := 'articles_' || to_char(start_date, 'YYYY_MM_DD');
        
        IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = partition_name) THEN
            EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF articles FOR VALUES FROM (%L) TO (%L)',
                partition_name, start_date, end_date);
            RAISE NOTICE 'Created partition: %', partition_name;
        END IF;
    END LOOP;
END $$;

-- Article tags partitions for February
DO $$
DECLARE
    start_date date;
    end_date date;
    partition_name text;
BEGIN
    FOR i IN 1..28 LOOP
        start_date := '2026-02-01'::date + (i-1);
        end_date := start_date + 1;
        partition_name := 'article_tags_' || to_char(start_date, 'YYYY_MM_DD');
        
        IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = partition_name) THEN
            EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF article_tags FOR VALUES FROM (%L) TO (%L)',
                partition_name, start_date, end_date);
            RAISE NOTICE 'Created partition: %', partition_name;
        END IF;
    END LOOP;
END $$;

\echo '=== Partition Creation Complete ==='
