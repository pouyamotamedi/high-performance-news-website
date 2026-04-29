-- Create article interactions table for tracking user engagement
CREATE TABLE IF NOT EXISTS article_interactions (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL, -- References articles(id) - no FK constraint due to partitioning
    ip_address INET NOT NULL,
    interaction_type VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure valid interaction types
    CONSTRAINT valid_interaction_type CHECK (interaction_type IN ('like', 'dislike', 'bookmark', 'view')),
    
    -- Unique constraint to prevent duplicate interactions per IP per article
    CONSTRAINT unique_article_ip_interaction UNIQUE(article_id, ip_address, interaction_type)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_article_interactions_article_id ON article_interactions(article_id);
CREATE INDEX IF NOT EXISTS idx_article_interactions_ip_address ON article_interactions(ip_address);
CREATE INDEX IF NOT EXISTS idx_article_interactions_type ON article_interactions(interaction_type);
CREATE INDEX IF NOT EXISTS idx_article_interactions_created_at ON article_interactions(created_at);
CREATE INDEX IF NOT EXISTS idx_article_interactions_article_ip ON article_interactions(article_id, ip_address);

-- Add updated_at trigger for article_interactions
CREATE OR REPLACE FUNCTION update_article_interactions_updated_at()
RETURNS TRIGGER AS $
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_article_interactions_updated_at
    BEFORE UPDATE ON article_interactions
    FOR EACH ROW
    EXECUTE FUNCTION update_article_interactions_updated_at();

-- Comments for documentation
COMMENT ON TABLE article_interactions IS 'User interactions with articles (likes, dislikes, bookmarks) tracked by IP address';
COMMENT ON COLUMN article_interactions.article_id IS 'References articles.id (enforced by application due to partitioning)';
COMMENT ON COLUMN article_interactions.ip_address IS 'Client IP address for rate limiting and tracking';
COMMENT ON COLUMN article_interactions.interaction_type IS 'Type of interaction: like, dislike, bookmark, or view';
COMMENT ON CONSTRAINT unique_article_ip_interaction ON article_interactions IS 'Prevents duplicate interactions per IP per article';