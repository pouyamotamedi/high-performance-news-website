-- Standardize Daily Partitioning Strategy
-- This migration ensures consistent daily partitioning for both fresh and existing deployments

-- Check if we're dealing with a fresh deployment or existing monthly partitions
DO $$
DECLARE
    monthly_partition_count integer;
    daily_partition_count integer;
BEGIN
    -- Count existing monthly partitions
    SELECT COUNT(*) INTO monthly_partition_count
    FROM pg_tables 
    WHERE tablename ~ '^articles_\d{4}_\d{2}$'
    AND schemaname = 'public';
    
    -- Count existing daily partitions
    SELECT COUNT(*) INTO daily_partition_count
    FROM pg_tables 
    WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$'
    AND schemaname = 'public';
    
    RAISE NOTICE 'Found % monthly partitions and % daily partitions', monthly_partition_count, daily_partition_count;
    
    -- If we have monthly partitions but no daily partitions, this is a legacy deployment
    IF monthly_partition_count > 0 AND daily_partition_count = 0 THEN
        RAISE NOTICE 'Legacy deployment detected - monthly partitions will be preserved';
        RAISE NOTICE 'Daily partitioning is already handled by migration 045';
        
    -- If we have daily partitions, we're already migrated
    ELSIF daily_partition_count > 0 THEN
        RAISE NOTICE 'Daily partitioning already implemented - no action needed';
        
    -- If we have neither, this is a fresh deployment - create daily partitions
    ELSE
        RAISE NOTICE 'Fresh deployment detected - creating initial daily partitions';
        
        -- Create daily partitions for current month and next month
        DECLARE
            current_date_iter date := date_trunc('month', CURRENT_DATE);
            end_date date := current_date_iter + interval '2 months';
            daily_partition_name text;
        BEGIN
            WHILE current_date_iter < end_date LOOP
                daily_partition_name := 'articles_' || to_char(current_date_iter, 'YYYY_MM_DD');
                
                -- Create articles daily partition
                EXECUTE format('CREATE TABLE %I PARTITION OF articles FOR VALUES FROM (%L) TO (%L)',
                    daily_partition_name, current_date_iter, current_date_iter + interval '1 day');
                
                -- Create optimized indexes
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
                
                -- Create article_tags daily partition
                EXECUTE format('CREATE TABLE article_tags_%s PARTITION OF article_tags FOR VALUES FROM (%L) TO (%L)',
                    to_char(current_date_iter, 'YYYY_MM_DD'), current_date_iter, current_date_iter + interval '1 day');
                
                EXECUTE format('CREATE INDEX idx_article_tags_%s_article ON article_tags_%s (article_id)',
                    to_char(current_date_iter, 'YYYY_MM_DD'), to_char(current_date_iter, 'YYYY_MM_DD'));
                EXECUTE format('CREATE INDEX idx_article_tags_%s_tag ON article_tags_%s (tag_id)',
                    to_char(current_date_iter, 'YYYY_MM_DD'), to_char(current_date_iter, 'YYYY_MM_DD'));
                
                -- Create article_views daily partition
                EXECUTE format('CREATE TABLE article_views_%s PARTITION OF article_views FOR VALUES FROM (%L) TO (%L)',
                    to_char(current_date_iter, 'YYYY_MM_DD'), current_date_iter, current_date_iter + interval '1 day');
                
                EXECUTE format('CREATE INDEX idx_article_views_%s_created_brin ON article_views_%s USING BRIN (created_at) WITH (pages_per_range = 32)',
                    to_char(current_date_iter, 'YYYY_MM_DD'), to_char(current_date_iter, 'YYYY_MM_DD'));
                EXECUTE format('CREATE INDEX idx_article_views_%s_article ON article_views_%s (article_id, created_at)',
                    to_char(current_date_iter, 'YYYY_MM_DD'), to_char(current_date_iter, 'YYYY_MM_DD'));
                
                current_date_iter := current_date_iter + interval '1 day';
            END LOOP;
            
            RAISE NOTICE 'Created daily partitions for fresh deployment';
        END;
    END IF;
    
    RAISE NOTICE 'Daily partitioning standardization completed';
END;
$$;