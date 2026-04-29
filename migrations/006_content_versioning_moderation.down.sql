-- Rollback Content Versioning and Moderation System Migration

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_create_article_version ON articles;
DROP FUNCTION IF EXISTS create_article_version();

-- Remove moderation columns from articles table
ALTER TABLE articles DROP COLUMN IF EXISTS moderation_status;
ALTER TABLE articles DROP COLUMN IF EXISTS moderation_notes;
ALTER TABLE articles DROP COLUMN IF EXISTS last_moderated_at;
ALTER TABLE articles DROP COLUMN IF EXISTS last_moderated_by;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS bulk_moderation_jobs;
DROP TABLE IF EXISTS moderation_actions;
DROP TABLE IF EXISTS content_quality_checks;
DROP TABLE IF EXISTS moderation_queue;
DROP TABLE IF EXISTS article_versions;