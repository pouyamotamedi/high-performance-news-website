-- Create social media credentials table
CREATE TABLE IF NOT EXISTS social_media_credentials (
    id BIGSERIAL PRIMARY KEY,
    platform VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    credentials JSONB NOT NULL,
    is_active BOOLEAN DEFAULT true,
    last_rotated TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_platform_name UNIQUE(platform, name)
);

-- Create social media posts table
CREATE TABLE IF NOT EXISTS social_media_posts (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL, -- References articles(id) - no FK constraint due to partitioning
    platform VARCHAR(50) NOT NULL,
    credential_id BIGINT NOT NULL REFERENCES social_media_credentials(id) ON DELETE CASCADE,
    post_id VARCHAR(255), -- Platform-specific post ID
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    content JSONB NOT NULL,
    scheduled_at TIMESTAMP,
    posted_at TIMESTAMP,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT valid_status CHECK (status IN ('pending', 'scheduled', 'posted', 'failed', 'retrying'))
);

-- Create social media webhooks table
CREATE TABLE IF NOT EXISTS social_media_webhooks (
    id BIGSERIAL PRIMARY KEY,
    platform VARCHAR(50) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    post_id VARCHAR(255),
    payload JSONB NOT NULL,
    signature VARCHAR(500),
    verified BOOLEAN DEFAULT false,
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create Facebook Instant Articles table
CREATE TABLE IF NOT EXISTS facebook_instant_articles (
    article_id BIGINT PRIMARY KEY, -- References articles(id) - no FK constraint due to partitioning
    instant_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    html TEXT NOT NULL,
    published_url VARCHAR(500),
    last_synced TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_social_credentials_platform_active ON social_media_credentials(platform, is_active);
CREATE INDEX IF NOT EXISTS idx_social_posts_article_platform ON social_media_posts(article_id, platform);
CREATE INDEX IF NOT EXISTS idx_social_posts_status_scheduled ON social_media_posts(status, scheduled_at) WHERE status = 'scheduled';
CREATE INDEX IF NOT EXISTS idx_social_posts_platform_status ON social_media_posts(platform, status);
CREATE INDEX IF NOT EXISTS idx_social_webhooks_platform_processed ON social_media_webhooks(platform, processed);
CREATE INDEX IF NOT EXISTS idx_social_webhooks_created_at ON social_media_webhooks(created_at);

-- Add updated_at trigger for social_media_credentials
CREATE OR REPLACE FUNCTION update_social_media_credentials_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_social_media_credentials_updated_at
    BEFORE UPDATE ON social_media_credentials
    FOR EACH ROW
    EXECUTE FUNCTION update_social_media_credentials_updated_at();

-- Add updated_at trigger for social_media_posts
CREATE OR REPLACE FUNCTION update_social_media_posts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_social_media_posts_updated_at
    BEFORE UPDATE ON social_media_posts
    FOR EACH ROW
    EXECUTE FUNCTION update_social_media_posts_updated_at();

-- Add updated_at trigger for facebook_instant_articles
CREATE OR REPLACE FUNCTION update_facebook_instant_articles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_facebook_instant_articles_updated_at
    BEFORE UPDATE ON facebook_instant_articles
    FOR EACH ROW
    EXECUTE FUNCTION update_facebook_instant_articles_updated_at();
-- Note: Foreign key constraints to articles table are not used because articles is partitioned.
-- Application logic should ensure referential integrity.
-- Indexes on article_id columns provide performance for joins.

-- Comments for documentation
COMMENT ON TABLE social_media_posts IS 'Social media posts linked to articles (no FK due to partitioning)';
COMMENT ON TABLE facebook_instant_articles IS 'Facebook Instant Articles linked to articles (no FK due to partitioning)';
COMMENT ON COLUMN social_media_posts.article_id IS 'References articles.id (enforced by application)';
COMMENT ON COLUMN facebook_instant_articles.article_id IS 'References articles.id (enforced by application)';