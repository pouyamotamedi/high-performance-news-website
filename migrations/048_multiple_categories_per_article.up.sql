-- Migration: Enable multiple categories per article
-- This migration creates a junction table for article-category relationships

-- Create article_categories junction table (partitioned by created_at)
CREATE TABLE article_categories (
    article_id BIGINT,
    category_id BIGINT REFERENCES categories(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (article_id, category_id, created_at)
) PARTITION BY RANGE (created_at);

-- Create initial partition for current month
DO $$
DECLARE
    start_date date := date_trunc('month', CURRENT_DATE);
    end_date date := start_date + interval '1 month';
    partition_name text := 'article_categories_' || to_char(start_date, 'YYYY_MM');
BEGIN
    EXECUTE format('CREATE TABLE %I PARTITION OF article_categories FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
    
    -- Create indexes on the partition
    EXECUTE format('CREATE INDEX idx_%I_article ON %I (article_id)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_category ON %I (category_id)',
        partition_name, partition_name);
END $$;

-- Migrate existing data from articles.category_id to article_categories
INSERT INTO article_categories (article_id, category_id, created_at)
SELECT id, category_id, created_at
FROM articles 
WHERE category_id IS NOT NULL AND category_id > 0;

-- Add comment for documentation
COMMENT ON TABLE article_categories IS 'Junction table for many-to-many relationship between articles and categories';

-- Note: We keep the category_id column in articles for backward compatibility
-- It can be removed in a future migration after ensuring all code uses the junction table