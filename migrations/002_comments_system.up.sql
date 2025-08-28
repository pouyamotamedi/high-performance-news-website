-- Create comments table (simplified without partitioning for now)
CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    parent_id BIGINT, -- Self-referencing for threading
    content TEXT NOT NULL,
    author_name VARCHAR(100) NOT NULL,
    author_email VARCHAR(255) NOT NULL,
    author_ip INET,
    user_agent TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    spam_score DECIMAL(3,2) DEFAULT 0.0,
    moderated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    moderated_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create comment_moderation_log table (simplified without partitioning)
CREATE TABLE comment_moderation_log (
    id BIGSERIAL PRIMARY KEY,
    comment_id BIGINT NOT NULL,
    action VARCHAR(20) NOT NULL,
    moderator_id BIGINT NOT NULL REFERENCES users(id),
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_comments_article_status ON comments (article_id, status, created_at);
CREATE INDEX idx_comments_parent ON comments (parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_comments_status ON comments (status, created_at);
CREATE INDEX idx_comments_user ON comments (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_comments_moderation ON comments (moderated_by, moderated_at) WHERE moderated_by IS NOT NULL;
CREATE INDEX idx_comments_spam_score ON comments (spam_score) WHERE spam_score > 0;
CREATE INDEX idx_comments_author_email ON comments (author_email);

-- Full-text search index for content
CREATE INDEX idx_comments_content_search ON comments USING gin(to_tsvector('english', content));

-- Indexes for moderation log
CREATE INDEX idx_comment_moderation_log_comment ON comment_moderation_log (comment_id, created_at);
CREATE INDEX idx_comment_moderation_log_moderator ON comment_moderation_log (moderator_id, created_at);

-- Create comment statistics view
CREATE VIEW comment_stats AS
SELECT 
    article_id,
    COUNT(*) as total_comments,
    COUNT(CASE WHEN status = 'approved' THEN 1 END) as approved_comments,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_comments,
    COUNT(CASE WHEN status = 'rejected' THEN 1 END) as rejected_comments,
    COUNT(CASE WHEN spam_score > 0.5 THEN 1 END) as spam_comments,
    MAX(created_at) as last_comment_at
FROM comments 
GROUP BY article_id;