-- Add engagement columns to articles table
-- Note: These columns might already exist in some partitions, so we use IF NOT EXISTS where possible

-- For partitioned tables, we need to add columns to each partition
DO $
DECLARE
    partition_record RECORD;
BEGIN
    -- First, try to add columns to the parent table
    -- This will apply to new partitions automatically
    BEGIN
        ALTER TABLE articles ADD COLUMN IF NOT EXISTS like_count INTEGER DEFAULT 0;
        ALTER TABLE articles ADD COLUMN IF NOT EXISTS dislike_count INTEGER DEFAULT 0;
    EXCEPTION
        WHEN OTHERS THEN
            -- Columns might already exist, continue
            NULL;
    END;
    
    -- Add columns to existing partitions
    FOR partition_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE tablename LIKE 'articles_%'
        AND schemaname = 'public'
    LOOP
        BEGIN
            EXECUTE format('ALTER TABLE %I.%I ADD COLUMN IF NOT EXISTS like_count INTEGER DEFAULT 0',
                partition_record.schemaname, 
                partition_record.tablename);
            EXECUTE format('ALTER TABLE %I.%I ADD COLUMN IF NOT EXISTS dislike_count INTEGER DEFAULT 0',
                partition_record.schemaname, 
                partition_record.tablename);
        EXCEPTION
            WHEN duplicate_column THEN
                -- Column already exists, skip
                NULL;
            WHEN OTHERS THEN
                -- Log error but continue with other partitions
                RAISE NOTICE 'Could not add engagement columns to partition %: %', partition_record.tablename, SQLERRM;
        END;
    END LOOP;
END $;

-- Add constraints to ensure non-negative values (on parent table)
DO $
BEGIN
    BEGIN
        ALTER TABLE articles ADD CONSTRAINT check_like_count_non_negative CHECK (like_count >= 0);
    EXCEPTION
        WHEN duplicate_object THEN
            -- Constraint already exists
            NULL;
    END;
    
    BEGIN
        ALTER TABLE articles ADD CONSTRAINT check_dislike_count_non_negative CHECK (dislike_count >= 0);
    EXCEPTION
        WHEN duplicate_object THEN
            -- Constraint already exists
            NULL;
    END;
END $;

-- Create indexes for performance on existing partitions
DO $
DECLARE
    partition_record RECORD;
BEGIN
    FOR partition_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE tablename LIKE 'articles_%'
        AND schemaname = 'public'
    LOOP
        BEGIN
            EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_like_count ON %I.%I(like_count)',
                partition_record.tablename,
                partition_record.schemaname, 
                partition_record.tablename);
            EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_dislike_count ON %I.%I(dislike_count)',
                partition_record.tablename,
                partition_record.schemaname, 
                partition_record.tablename);
        EXCEPTION
            WHEN OTHERS THEN
                -- Log error but continue with other partitions
                RAISE NOTICE 'Could not create indexes on partition %: %', partition_record.tablename, SQLERRM;
        END;
    END LOOP;
END $;

-- Comments for documentation
COMMENT ON COLUMN articles.like_count IS 'Number of likes for this article';
COMMENT ON COLUMN articles.dislike_count IS 'Number of dislikes for this article';