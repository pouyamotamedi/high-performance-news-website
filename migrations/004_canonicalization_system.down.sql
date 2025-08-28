-- Drop views
DROP VIEW IF EXISTS pending_canonical_jobs;

-- Drop triggers
DROP TRIGGER IF EXISTS update_canonical_jobs_updated_at ON canonical_jobs;

-- Drop functions
DROP FUNCTION IF EXISTS cleanup_old_canonical_jobs();
DROP FUNCTION IF EXISTS schedule_canonical_job(BIGINT, VARCHAR(20), BIGINT, VARCHAR(500), BIGINT, BOOLEAN);
DROP FUNCTION IF EXISTS process_canonical_job(BIGINT, BIGINT);
DROP FUNCTION IF EXISTS generate_canonical_url(VARCHAR(20), BIGINT, VARCHAR(500));

-- Drop table
DROP TABLE IF EXISTS canonical_jobs CASCADE;

-- Remove auto_linking column from articles table if it was added by this migration
-- Note: We don't remove it if it already existed before this migration
-- This is a conservative approach to avoid data loss