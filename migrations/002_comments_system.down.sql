-- Drop views
DROP VIEW IF EXISTS moderation_queue;
DROP VIEW IF EXISTS comment_stats;

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_update_comment_updated_at ON comments;

-- Drop functions
DROP FUNCTION IF EXISTS update_comment_updated_at();
DROP FUNCTION IF EXISTS drop_old_comment_partitions();
DROP FUNCTION IF EXISTS create_monthly_comment_partitions();

-- Drop tables (this will also drop all partitions)
DROP TABLE IF EXISTS comment_moderation_log CASCADE;
DROP TABLE IF EXISTS comments CASCADE;