-- Migration: Add GIN indexes for PostgreSQL full-text search fallback
-- This enables efficient full-text search when MeiliSearch is unavailable
-- Date: 2026-02-26

-- Create a generated column for full-text search vector (PostgreSQL 12+)
-- This pre-computes the tsvector for faster searches
ALTER TABLE articles 
ADD COLUMN IF NOT EXISTS search_vector tsvector 
GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(excerpt, '')), 'B') ||
    setweight(to_tsvector('simple', COALESCE(content, '')), 'C')
) STORED;

-- Create GIN index on the search vector for fast full-text search
CREATE INDEX IF NOT EXISTS idx_articles_search_vector 
ON articles USING GIN (search_vector);

-- Create additional indexes for common filter combinations
CREATE INDEX IF NOT EXISTS idx_articles_status_published_at 
ON articles (status, published_at DESC) 
WHERE status = 'published';

CREATE INDEX IF NOT EXISTS idx_articles_category_published 
ON articles (category_id, published_at DESC) 
WHERE status = 'published';

CREATE INDEX IF NOT EXISTS idx_articles_author_published 
ON articles (author_id, published_at DESC) 
WHERE status = 'published';

CREATE INDEX IF NOT EXISTS idx_articles_language_published 
ON articles (language_code, published_at DESC) 
WHERE status = 'published';

-- Composite index for common search patterns
CREATE INDEX IF NOT EXISTS idx_articles_search_filters 
ON articles (status, category_id, author_id, published_at DESC);

-- Comment explaining the migration
COMMENT ON COLUMN articles.search_vector IS 'Pre-computed tsvector for PostgreSQL full-text search fallback when MeiliSearch is unavailable';
