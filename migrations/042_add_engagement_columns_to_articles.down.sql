-- Remove engagement columns from articles table

-- Remove constraints first
ALTER TABLE articles DROP CONSTRAINT IF EXISTS check_like_count_non_negative;
ALTER TABLE articles DROP CONSTRAINT IF EXISTS check_dislike_count_non_negative;

-- Remove indexes from partitions
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
            EXECUTE format('DROP INDEX IF EXISTS idx_%I_like_count',
                partition_record.tablename);
            EXECUTE format('DROP INDEX IF EXISTS idx_%I_dislike_count',
                partition_record.tablename);
        EXCEPTION
            WHEN OTHERS THEN
                -- Continue with other partitions
                NULL;
        END;
    END LOOP;
END $;

-- Remove columns from partitions
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
            EXECUTE format('ALTER TABLE %I.%I DROP COLUMN IF EXISTS like_count',
                partition_record.schemaname, 
                partition_record.tablename);
            EXECUTE format('ALTER TABLE %I.%I DROP COLUMN IF EXISTS dislike_count',
                partition_record.schemaname, 
                partition_record.tablename);
        EXCEPTION
            WHEN OTHERS THEN
                -- Continue with other partitions
                NULL;
        END;
    END LOOP;
END $;

-- Remove columns from parent table
ALTER TABLE articles DROP COLUMN IF EXISTS like_count;
ALTER TABLE articles DROP COLUMN IF EXISTS dislike_count;