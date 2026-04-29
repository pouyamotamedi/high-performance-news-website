-- Rollback migration: Remove multiple categories support

-- Drop the article_categories table and its partitions
DROP TABLE IF EXISTS article_categories CASCADE;

-- Note: We don't restore the single category_id constraint as it may cause data loss
-- The category_id column in articles table is preserved for backward compatibility