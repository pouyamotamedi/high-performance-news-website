-- Drop article interactions table and related objects

-- Drop trigger first
DROP TRIGGER IF EXISTS trigger_article_interactions_updated_at ON article_interactions;

-- Drop function
DROP FUNCTION IF EXISTS update_article_interactions_updated_at();

-- Drop indexes (they will be dropped automatically with the table, but explicit for clarity)
DROP INDEX IF EXISTS idx_article_interactions_article_id;
DROP INDEX IF EXISTS idx_article_interactions_ip_address;
DROP INDEX IF EXISTS idx_article_interactions_type;
DROP INDEX IF EXISTS idx_article_interactions_created_at;
DROP INDEX IF EXISTS idx_article_interactions_article_ip;

-- Drop the table
DROP TABLE IF EXISTS article_interactions;