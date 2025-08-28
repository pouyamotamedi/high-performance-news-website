-- Drop triggers
DROP TRIGGER IF EXISTS trigger_facebook_instant_articles_updated_at ON facebook_instant_articles;
DROP TRIGGER IF EXISTS trigger_social_media_posts_updated_at ON social_media_posts;
DROP TRIGGER IF EXISTS trigger_social_media_credentials_updated_at ON social_media_credentials;

-- Drop functions
DROP FUNCTION IF EXISTS update_facebook_instant_articles_updated_at();
DROP FUNCTION IF EXISTS update_social_media_posts_updated_at();
DROP FUNCTION IF EXISTS update_social_media_credentials_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_social_webhooks_created_at;
DROP INDEX IF EXISTS idx_social_webhooks_platform_processed;
DROP INDEX IF EXISTS idx_social_posts_platform_status;
DROP INDEX IF EXISTS idx_social_posts_status_scheduled;
DROP INDEX IF EXISTS idx_social_posts_article_platform;
DROP INDEX IF EXISTS idx_social_credentials_platform_active;

-- Drop tables
DROP TABLE IF EXISTS facebook_instant_articles;
DROP TABLE IF EXISTS social_media_webhooks;
DROP TABLE IF EXISTS social_media_posts;
DROP TABLE IF EXISTS social_media_credentials;