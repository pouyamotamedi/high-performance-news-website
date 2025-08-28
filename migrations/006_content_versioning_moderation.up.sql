-- Content Versioning and Moderation System Migration

-- Create article_versions table for version history tracking
CREATE TABLE article_versions (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    version_number INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT,
    author_id BIGINT NOT NULL REFERENCES users(id),
    category_id BIGINT NOT NULL REFERENCES categories(id),
    status VARCHAR(20) NOT NULL,
    published_at TIMESTAMP WITH TIME ZONE,
    meta_title VARCHAR(60),
    meta_description VARCHAR(160),
    canonical_url VARCHAR(500),
    schema_type VARCHAR(50) DEFAULT 'NewsArticle',
    language_code VARCHAR(2) NOT NULL DEFAULT 'fa',
    translation_group_id BIGINT,
    auto_linking BOOLEAN DEFAULT true,
    change_summary TEXT, -- Summary of changes made in this version
    created_by BIGINT NOT NULL REFERENCES users(id), -- Who created this version
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(article_id, version_number)
);

-- Create moderation_queue table for content approval workflow
CREATE TABLE moderation_queue (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    article_version_id BIGINT REFERENCES article_versions(id),
    content_type VARCHAR(20) NOT NULL DEFAULT 'article', -- 'article', 'comment', etc.
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected', 'flagged'
    priority INTEGER DEFAULT 1, -- 1=low, 2=medium, 3=high, 4=urgent
    submitted_by BIGINT NOT NULL REFERENCES users(id),
    assigned_to BIGINT REFERENCES users(id), -- Moderator assigned to review
    ai_quality_score DECIMAL(3,2), -- AI quality score (0.00-1.00)
    ai_feedback JSONB, -- AI analysis results
    moderator_notes TEXT,
    rejection_reason TEXT,
    auto_approved BOOLEAN DEFAULT false, -- Whether approved automatically by AI
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by BIGINT REFERENCES users(id)
);

-- Create content_quality_checks table for AI analysis results
CREATE TABLE content_quality_checks (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    article_version_id BIGINT REFERENCES article_versions(id),
    ai_provider VARCHAR(20) NOT NULL, -- 'openai', 'anthropic'
    quality_score DECIMAL(3,2) NOT NULL, -- Overall quality score (0.00-1.00)
    grammar_score DECIMAL(3,2), -- Grammar quality (0.00-1.00)
    readability_score DECIMAL(3,2), -- Readability score (0.00-1.00)
    appropriateness_score DECIMAL(3,2), -- Content appropriateness (0.00-1.00)
    issues_found JSONB, -- Array of issues found
    suggestions JSONB, -- Array of improvement suggestions
    flagged_content JSONB, -- Specific content flagged for review
    processing_time_ms INTEGER, -- Time taken for AI analysis
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create moderation_actions table for audit trail
CREATE TABLE moderation_actions (
    id BIGSERIAL PRIMARY KEY,
    moderation_queue_id BIGINT NOT NULL REFERENCES moderation_queue(id),
    action VARCHAR(20) NOT NULL, -- 'approve', 'reject', 'flag', 'assign', 'reassign'
    performed_by BIGINT NOT NULL REFERENCES users(id),
    notes TEXT,
    previous_status VARCHAR(20),
    new_status VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create bulk_moderation_jobs table for high-volume processing
CREATE TABLE bulk_moderation_jobs (
    id BIGSERIAL PRIMARY KEY,
    job_name VARCHAR(255) NOT NULL,
    job_type VARCHAR(50) NOT NULL, -- 'bulk_approve', 'bulk_reject', 'bulk_ai_check'
    criteria JSONB NOT NULL, -- Selection criteria for bulk operation
    total_items INTEGER DEFAULT 0,
    processed_items INTEGER DEFAULT 0,
    successful_items INTEGER DEFAULT 0,
    failed_items INTEGER DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
    created_by BIGINT NOT NULL REFERENCES users(id),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    error_log TEXT
);

-- Create indexes for performance
CREATE INDEX idx_article_versions_article_id ON article_versions (article_id, version_number DESC);
CREATE INDEX idx_article_versions_author ON article_versions (author_id, created_at DESC);
CREATE INDEX idx_article_versions_created_by ON article_versions (created_by, created_at DESC);

CREATE INDEX idx_moderation_queue_status ON moderation_queue (status, priority DESC, submitted_at ASC);
CREATE INDEX idx_moderation_queue_assigned ON moderation_queue (assigned_to, status, submitted_at ASC);
CREATE INDEX idx_moderation_queue_article ON moderation_queue (article_id, submitted_at DESC);
CREATE INDEX idx_moderation_queue_submitted_by ON moderation_queue (submitted_by, submitted_at DESC);

CREATE INDEX idx_content_quality_checks_article ON content_quality_checks (article_id, created_at DESC);
CREATE INDEX idx_content_quality_checks_score ON content_quality_checks (quality_score, created_at DESC);
CREATE INDEX idx_content_quality_checks_provider ON content_quality_checks (ai_provider, created_at DESC);

CREATE INDEX idx_moderation_actions_queue ON moderation_actions (moderation_queue_id, created_at DESC);
CREATE INDEX idx_moderation_actions_performed_by ON moderation_actions (performed_by, created_at DESC);

CREATE INDEX idx_bulk_moderation_jobs_status ON bulk_moderation_jobs (status, created_at DESC);
CREATE INDEX idx_bulk_moderation_jobs_created_by ON bulk_moderation_jobs (created_by, created_at DESC);

-- Add version tracking trigger to articles table
CREATE OR REPLACE FUNCTION create_article_version()
RETURNS TRIGGER AS $$
DECLARE
    next_version INTEGER;
BEGIN
    -- Get next version number
    SELECT COALESCE(MAX(version_number), 0) + 1 
    INTO next_version 
    FROM article_versions 
    WHERE article_id = NEW.id;
    
    -- Insert new version record
    INSERT INTO article_versions (
        article_id, version_number, title, slug, content, excerpt,
        author_id, category_id, status, published_at,
        meta_title, meta_description, canonical_url, schema_type,
        language_code, translation_group_id, auto_linking,
        change_summary, created_by
    ) VALUES (
        NEW.id, next_version, NEW.title, NEW.slug, NEW.content, NEW.excerpt,
        NEW.author_id, NEW.category_id, NEW.status, NEW.published_at,
        NEW.meta_title, NEW.meta_description, NEW.canonical_url, NEW.schema_type,
        COALESCE(NEW.language_code, 'fa'), NEW.translation_group_id, COALESCE(NEW.auto_linking, true),
        'Automatic version created on article update', NEW.author_id
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic version creation (only on UPDATE, not INSERT)
-- We'll handle initial version creation manually in the application
CREATE TRIGGER trigger_create_article_version
    AFTER UPDATE ON articles
    FOR EACH ROW
    WHEN (OLD.* IS DISTINCT FROM NEW.*)
    EXECUTE FUNCTION create_article_version();

-- Add moderation status to articles table
ALTER TABLE articles ADD COLUMN moderation_status VARCHAR(20) DEFAULT 'approved';
ALTER TABLE articles ADD COLUMN moderation_notes TEXT;
ALTER TABLE articles ADD COLUMN last_moderated_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE articles ADD COLUMN last_moderated_by BIGINT REFERENCES users(id);

-- Create index for moderation status
CREATE INDEX idx_articles_moderation_status ON articles (moderation_status, created_at DESC);