-- Safe migration from monthly to daily partitions
-- This approach detaches monthly partitions, creates daily partitions, then migrates data

-- Step 1: Create a function to safely migrate by detaching first
CREATE OR REPLACE FUNCTION safe_migrate_to_daily_partitions()
RETURNS void AS $$
DECLARE
    monthly_partition RECORD;
    daily_partition_date date;
    daily_partition_name text;
    data_count integer;
    temp_table_name text;
BEGIN
    -- Process each monthly partition
    FOR monthly_partition IN
        SELECT schemaname, tablename 
        FROM pg_tables 
        WHERE tablename ~ '^articles_\d{4}_\d{2}$'
        AND schemaname = 'public'
    LOOP
        RAISE NOTICE 'Processing monthly partition: %', monthly_partition.tablename;
        
        DECLARE
            year_month text;
            start_date date;
            end_date date;
            current_date_iter date;
        BEGIN
            -- Extract YYYY_MM from partition name
            year_month := regexp_replace(monthly_partition.tablename, '^articles_', '');
            start_date := to_date(year_month || '_01', 'YYYY_MM_DD');
            end_date := start_date + interval '1 month';
            
            RAISE NOTICE 'Monthly partition % covers % to %', monthly_partition.tablename, start_date, end_date;
            
            -- Step 1: Detach the monthly partition from the main table
            EXECUTE format('ALTER TABLE articles DETACH PARTITION %I', monthly_partition.tablename);
            RAISE NOTICE 'Detached monthly partition: %', monthly_partition.tablename;
            
            -- Step 2: Create daily partitions for each day in the month
            current_date_iter := start_date;
            WHILE current_date_iter < end_date LOOP
                daily_partition_name := 'articles_' || to_char(current_date_iter, 'YYYY_MM_DD');
                
                -- Check if daily partition already exists
                IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = daily_partition_name) THEN
                    -- Create daily partition
                    EXECUTE format('CREATE TABLE %I PARTITION OF articles FOR VALUES FROM (%L) TO (%L)',
                        daily_partition_name, current_date_iter, current_date_iter + interval '1 day');
                    
                    -- Create optimized indexes for daily partition
                    EXECUTE format('CREATE INDEX idx_%I_published_at ON %I (published_at DESC) WHERE status = ''published''',
                        daily_partition_name, daily_partition_name);
                    EXECUTE format('CREATE INDEX idx_%I_category ON %I (category_id, published_at DESC)',
                        daily_partition_name, daily_partition_name);
                    EXECUTE format('CREATE INDEX idx_%I_author ON %I (author_id, published_at DESC)',
                        daily_partition_name, daily_partition_name);
                    EXECUTE format('CREATE INDEX idx_%I_slug ON %I (slug) WHERE status = ''published''',
                        daily_partition_name, daily_partition_name);
                    EXECUTE format('CREATE INDEX idx_%I_search ON %I USING gin(to_tsvector(''english'', title || '' '' || content))',
                        daily_partition_name, daily_partition_name);
                    EXECUTE format('CREATE INDEX idx_%I_published_brin ON %I USING BRIN (published_at) WITH (pages_per_range = 32)',
                        daily_partition_name, daily_partition_name);
                    
                    RAISE NOTICE 'Created daily partition: %', daily_partition_name;
                END IF;
                
                current_date_iter := current_date_iter + interval '1 day';
            END LOOP;
            
            -- Step 3: Move data from detached monthly partition to daily partitions
            EXECUTE format('SELECT COUNT(*) FROM %I', monthly_partition.tablename) INTO data_count;
            
            IF data_count > 0 THEN
                RAISE NOTICE 'Moving % records from detached monthly partition %', data_count, monthly_partition.tablename;
                
                -- Insert data into the main partitioned table (PostgreSQL will route to correct daily partitions)
                EXECUTE format('INSERT INTO articles SELECT * FROM %I', monthly_partition.tablename);
                
                RAISE NOTICE 'Successfully moved % records to daily partitions', data_count;
            ELSE
                RAISE NOTICE 'No data found in monthly partition %', monthly_partition.tablename;
            END IF;
            
            -- Step 4: Drop the detached monthly partition
            EXECUTE format('DROP TABLE %I CASCADE', monthly_partition.tablename);
            RAISE NOTICE 'Dropped detached monthly partition: %', monthly_partition.tablename;
            
        EXCEPTION
            WHEN OTHERS THEN
                RAISE WARNING 'Error processing monthly partition %: %', monthly_partition.tablename, SQLERRM;
                -- Try to reattach the partition if something went wrong
                BEGIN
                    EXECUTE format('ALTER TABLE articles ATTACH PARTITION %I FOR VALUES FROM (%L) TO (%L)',
                        monthly_partition.tablename, start_date, end_date);
                    RAISE NOTICE 'Reattached partition % due to error', monthly_partition.tablename;
                EXCEPTION
                    WHEN OTHERS THEN
                        RAISE WARNING 'Could not reattach partition %: %', monthly_partition.tablename, SQLERRM;
                END;
        END;
    END LOOP;
    
    RAISE NOTICE 'Articles monthly to daily partition migration completed';
END;
$$ LANGUAGE plpgsql;

-- Step 2: Create function to migrate article_tags partitions safely
CREATE OR REPLACE FUNCTION safe_migrate_article_tags_to_daily()
RETURNS void AS $$
DECLARE
    monthly_partition RECORD;
    data_count integer;
BEGIN
    FOR monthly_partition IN
        SELECT schemaname, tablename 
        FROM pg_tables 
        WHERE tablename ~ '^article_tags_\d{4}_\d{2}$'
        AND schemaname = 'public'
    LOOP
        RAISE NOTICE 'Processing article_tags monthly partition: %', monthly_partition.tablename;
        
        DECLARE
            year_month text;
            start_date date;
            end_date date;
            current_date_iter date;
            daily_partition_name text;
        BEGIN
            year_month := regexp_replace(monthly_partition.tablename, '^article_tags_', '');
            start_date := to_date(year_month || '_01', 'YYYY_MM_DD');
            end_date := start_date + interval '1 month';
            
            -- Detach monthly partition
            EXECUTE format('ALTER TABLE article_tags DETACH PARTITION %I', monthly_partition.tablename);
            RAISE NOTICE 'Detached article_tags monthly partition: %', monthly_partition.tablename;
            
            -- Create daily partitions
            current_date_iter := start_date;
            WHILE current_date_iter < end_date LOOP
                daily_partition_name := 'article_tags_' || to_char(current_date_iter, 'YYYY_MM_DD');
                
                IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = daily_partition_name) THEN
                    EXECUTE format('CREATE TABLE %I PARTITION OF article_tags FOR VALUES FROM (%L) TO (%L)',
                        daily_partition_name, current_date_iter, current_date_iter + interval '1 day');
                    
                    EXECUTE format('CREATE INDEX idx_%I_article ON %I (article_id)',
                        daily_partition_name, daily_partition_name);
                    EXECUTE format('CREATE INDEX idx_%I_tag ON %I (tag_id)',
                        daily_partition_name, daily_partition_name);
                    
                    RAISE NOTICE 'Created daily article_tags partition: %', daily_partition_name;
                END IF;
                
                current_date_iter := current_date_iter + interval '1 day';
            END LOOP;
            
            -- Move data
            EXECUTE format('SELECT COUNT(*) FROM %I', monthly_partition.tablename) INTO data_count;
            
            IF data_count > 0 THEN
                EXECUTE format('INSERT INTO article_tags SELECT * FROM %I', monthly_partition.tablename);
                RAISE NOTICE 'Moved % article_tags records to daily partitions', data_count;
            END IF;
            
            -- Drop detached partition
            EXECUTE format('DROP TABLE %I CASCADE', monthly_partition.tablename);
            RAISE NOTICE 'Dropped detached article_tags partition: %', monthly_partition.tablename;
            
        EXCEPTION
            WHEN OTHERS THEN
                RAISE WARNING 'Error processing article_tags partition %: %', monthly_partition.tablename, SQLERRM;
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Step 3: Create function to migrate article_views partitions safely
CREATE OR REPLACE FUNCTION safe_migrate_article_views_to_daily()
RETURNS void AS $$
DECLARE
    monthly_partition RECORD;
    data_count integer;
BEGIN
    FOR monthly_partition IN
        SELECT schemaname, tablename 
        FROM pg_tables 
        WHERE tablename ~ '^article_views_\d{4}_\d{2}$'
        AND schemaname = 'public'
    LOOP
        RAISE NOTICE 'Processing article_views monthly partition: %', monthly_partition.tablename;
        
        DECLARE
            year_month text;
            start_date date;
            end_date date;
            current_date_iter date;
            daily_partition_name text;
        BEGIN
            year_month := regexp_replace(monthly_partition.tablename, '^article_views_', '');
            start_date := to_date(year_month || '_01', 'YYYY_MM_DD');
            end_date := start_date + interval '1 month';
            
            -- Detach monthly partition
            EXECUTE format('ALTER TABLE article_views DETACH PARTITION %I', monthly_partition.tablename);
            RAISE NOTICE 'Detached article_views monthly partition: %', monthly_partition.tablename;
            
            -- Create daily partitions
            current_date_iter := start_date;
            WHILE current_date_iter < end_date LOOP
                daily_partition_name := 'article_views_' || to_char(current_date_iter, 'YYYY_MM_DD');
                
                IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = daily_partition_name) THEN
                    EXECUTE format('CREATE TABLE %I PARTITION OF article_views FOR VALUES FROM (%L) TO (%L)',
                        daily_partition_name, current_date_iter, current_date_iter + interval '1 day');
                    
                    EXECUTE format('CREATE INDEX idx_%I_created_brin ON %I USING BRIN (created_at) WITH (pages_per_range = 32)',
                        daily_partition_name, daily_partition_name);
                    EXECUTE format('CREATE INDEX idx_%I_article ON %I (article_id, created_at)',
                        daily_partition_name, daily_partition_name);
                    
                    RAISE NOTICE 'Created daily article_views partition: %', daily_partition_name;
                END IF;
                
                current_date_iter := current_date_iter + interval '1 day';
            END LOOP;
            
            -- Move data
            EXECUTE format('SELECT COUNT(*) FROM %I', monthly_partition.tablename) INTO data_count;
            
            IF data_count > 0 THEN
                EXECUTE format('INSERT INTO article_views SELECT * FROM %I', monthly_partition.tablename);
                RAISE NOTICE 'Moved % article_views records to daily partitions', data_count;
            END IF;
            
            -- Drop detached partition
            EXECUTE format('DROP TABLE %I CASCADE', monthly_partition.tablename);
            RAISE NOTICE 'Dropped detached article_views partition: %', monthly_partition.tablename;
            
        EXCEPTION
            WHEN OTHERS THEN
                RAISE WARNING 'Error processing article_views partition %: %', monthly_partition.tablename, SQLERRM;
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Step 4: Execute the safe migration
DO $$
BEGIN
    RAISE NOTICE 'Starting SAFE migration from monthly to daily partitions...';
    RAISE NOTICE 'This migration will detach monthly partitions before creating daily ones.';
    
    -- Migrate articles partitions
    PERFORM safe_migrate_to_daily_partitions();
    
    -- Migrate article_tags partitions
    PERFORM safe_migrate_article_tags_to_daily();
    
    -- Migrate article_views partitions
    PERFORM safe_migrate_article_views_to_daily();
    
    RAISE NOTICE 'Safe migration completed successfully!';
END;
$$;

-- Step 5: Clean up migration functions
DROP FUNCTION IF EXISTS safe_migrate_to_daily_partitions();
DROP FUNCTION IF EXISTS safe_migrate_article_tags_to_daily();
DROP FUNCTION IF EXISTS safe_migrate_article_views_to_daily();

-- Step 6: Create additional daily partitions for the next few days
SELECT create_daily_partitions();

-- Display final status
SELECT 
    'Safe daily partition migration completed!' as status,
    COUNT(*) as daily_partitions_created
FROM pg_tables 
WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$'
AND schemaname = 'public';