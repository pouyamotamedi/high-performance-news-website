-- Add partition management functions for automated partition management
-- This migration adds the missing partition management functions

-- Function 1: Create Daily Partitions (creates partitions for next 7 days)
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

-- Function 2: Drop Old Partitions (removes partitions older than retention period)
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

-- Function 3: Partition Maintenance (combines daily creation and cleanup)
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