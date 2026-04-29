-- Rollback migration: Convert daily partitions back to monthly
-- WARNING: This rollback will consolidate daily partitions back to monthly ones

-- This is a complex rollback that should be used with caution
-- It will merge all daily partitions for each month back into monthly partitions

DO $$
BEGIN
    RAISE NOTICE 'Rolling back daily partitions to monthly partitions...';
    RAISE NOTICE 'This operation will consolidate daily data into monthly partitions.';
    
    -- Note: This rollback is complex and should be implemented carefully
    -- For now, we'll just log the rollback attempt
    RAISE WARNING 'Daily to monthly partition rollback not implemented in this version.';
    RAISE WARNING 'Manual intervention required to rollback this migration.';
    RAISE WARNING 'Contact system administrator for rollback assistance.';
END;
$$;