-- Rollback migration: Remove content ingestion enhancements

-- Drop partition maintenance function
DROP FUNCTION IF EXISTS partition_maintenance();

-- Drop article_tags partition creation function
DROP FUNCTION IF EXISTS create_article_tags_daily_partitions();

-- Drop image_variants table
DROP TABLE IF EXISTS image_variants CASCADE;

-- Drop foreign key constraint
ALTER TABLE articles DROP CONSTRAINT IF EXISTS articles_featured_image_id_fkey;

-- Drop indexes
DROP INDEX IF EXISTS idx_articles_featured_image;
DROP INDEX IF EXISTS idx_articles_focus_keyword;

-- Remove columns from articles table
ALTER TABLE articles DROP COLUMN IF EXISTS featured_image_id;
ALTER TABLE articles DROP COLUMN IF EXISTS focus_keyword;

-- Drop images table
DROP TABLE IF EXISTS images CASCADE;
