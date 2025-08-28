-- Canonicalization Jobs table for delayed canonical URL processing
CREATE TABLE canonical_jobs (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('tag', 'category', 'url')),
    target_id BIGINT, -- For tag or category targets
    target_url VARCHAR(500), -- For custom URL targets
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processed', 'cancelled', 'failed')),
    admin_override BOOLEAN DEFAULT false,
    created_by BIGINT REFERENCES users(id),
    processed_by BIGINT REFERENCES users(id),
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for canonical_jobs
CREATE INDEX idx_canonical_jobs_article ON canonical_jobs (article_id);
CREATE INDEX idx_canonical_jobs_scheduled ON canonical_jobs (scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_canonical_jobs_status ON canonical_jobs (status, scheduled_at);
CREATE INDEX idx_canonical_jobs_target ON canonical_jobs (target_type, target_id) WHERE target_id IS NOT NULL;
CREATE INDEX idx_canonical_jobs_created_by ON canonical_jobs (created_by) WHERE created_by IS NOT NULL;

-- Add auto_linking column to articles table if not exists
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'articles' AND column_name = 'auto_linking') THEN
        ALTER TABLE articles ADD COLUMN auto_linking BOOLEAN DEFAULT true;
    END IF;
END $$;

-- Function to generate canonical URLs for tags and categories
CREATE OR REPLACE FUNCTION generate_canonical_url(
    target_type VARCHAR(20),
    target_id BIGINT DEFAULT NULL,
    target_url VARCHAR(500) DEFAULT NULL
) RETURNS VARCHAR(500) AS $
DECLARE
    result_url VARCHAR(500);
    tag_slug VARCHAR(100);
    category_slug VARCHAR(100);
    category_path TEXT;
BEGIN
    CASE target_type
        WHEN 'tag' THEN
            SELECT slug INTO tag_slug FROM tags WHERE id = target_id;
            IF tag_slug IS NULL THEN
                RAISE EXCEPTION 'Tag with ID % not found', target_id;
            END IF;
            result_url := '/tag/' || tag_slug;
            
        WHEN 'category' THEN
            -- Build category path for hierarchical categories
            WITH RECURSIVE category_hierarchy AS (
                SELECT id, slug, parent_id, slug as path
                FROM categories 
                WHERE id = target_id
                
                UNION ALL
                
                SELECT c.id, c.slug, c.parent_id, c.slug || '/' || ch.path
                FROM categories c
                INNER JOIN category_hierarchy ch ON c.id = ch.parent_id
            )
            SELECT path INTO category_path 
            FROM category_hierarchy 
            WHERE parent_id IS NULL;
            
            IF category_path IS NULL THEN
                RAISE EXCEPTION 'Category with ID % not found', target_id;
            END IF;
            result_url := '/category/' || category_path;
            
        WHEN 'url' THEN
            IF target_url IS NULL OR target_url = '' THEN
                RAISE EXCEPTION 'Custom URL cannot be empty';
            END IF;
            result_url := target_url;
            
        ELSE
            RAISE EXCEPTION 'Invalid target_type: %', target_type;
    END CASE;
    
    RETURN result_url;
END;
$ LANGUAGE plpgsql;

-- Function to process canonical jobs
CREATE OR REPLACE FUNCTION process_canonical_job(job_id BIGINT, processor_user_id BIGINT DEFAULT NULL)
RETURNS BOOLEAN AS $
DECLARE
    job_record RECORD;
    canonical_url VARCHAR(500);
    article_partition_name TEXT;
BEGIN
    -- Lock and fetch the job
    SELECT * INTO job_record 
    FROM canonical_jobs 
    WHERE id = job_id AND status = 'pending'
    FOR UPDATE SKIP LOCKED;
    
    IF NOT FOUND THEN
        RETURN FALSE; -- Job not found or already processed
    END IF;
    
    -- Check if it's time to process (unless admin override)
    IF NOT job_record.admin_override AND job_record.scheduled_at > NOW() THEN
        RETURN FALSE; -- Not yet time to process
    END IF;
    
    BEGIN
        -- Generate canonical URL
        canonical_url := generate_canonical_url(
            job_record.target_type, 
            job_record.target_id, 
            job_record.target_url
        );
        
        -- Update the article's canonical URL
        -- Note: We need to update across all partitions since we don't know which partition contains the article
        UPDATE articles 
        SET canonical_url = canonical_url, updated_at = NOW()
        WHERE id = job_record.article_id;
        
        -- Mark job as processed
        UPDATE canonical_jobs 
        SET status = 'processed',
            processed_at = NOW(),
            processed_by = processor_user_id,
            updated_at = NOW()
        WHERE id = job_id;
        
        RETURN TRUE;
        
    EXCEPTION WHEN OTHERS THEN
        -- Mark job as failed and increment retry count
        UPDATE canonical_jobs 
        SET status = 'failed',
            error_message = SQLERRM,
            retry_count = retry_count + 1,
            updated_at = NOW()
        WHERE id = job_id;
        
        RETURN FALSE;
    END;
END;
$ LANGUAGE plpgsql;

-- Function to schedule canonical job with 48-hour delay
CREATE OR REPLACE FUNCTION schedule_canonical_job(
    p_article_id BIGINT,
    p_target_type VARCHAR(20),
    p_target_id BIGINT DEFAULT NULL,
    p_target_url VARCHAR(500) DEFAULT NULL,
    p_created_by BIGINT DEFAULT NULL,
    p_admin_override BOOLEAN DEFAULT FALSE
) RETURNS BIGINT AS $
DECLARE
    job_id BIGINT;
    scheduled_time TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Calculate scheduled time (48 hours from now, or immediate if admin override)
    IF p_admin_override THEN
        scheduled_time := NOW();
    ELSE
        scheduled_time := NOW() + INTERVAL '48 hours';
    END IF;
    
    -- Validate target parameters
    IF p_target_type IN ('tag', 'category') AND p_target_id IS NULL THEN
        RAISE EXCEPTION 'target_id is required for target_type %', p_target_type;
    END IF;
    
    IF p_target_type = 'url' AND (p_target_url IS NULL OR p_target_url = '') THEN
        RAISE EXCEPTION 'target_url is required for target_type url';
    END IF;
    
    -- Cancel any existing pending jobs for this article
    UPDATE canonical_jobs 
    SET status = 'cancelled', updated_at = NOW()
    WHERE article_id = p_article_id AND status = 'pending';
    
    -- Insert new job
    INSERT INTO canonical_jobs (
        article_id, target_type, target_id, target_url, 
        scheduled_at, admin_override, created_by
    ) VALUES (
        p_article_id, p_target_type, p_target_id, p_target_url,
        scheduled_time, p_admin_override, p_created_by
    ) RETURNING id INTO job_id;
    
    RETURN job_id;
END;
$ LANGUAGE plpgsql;

-- Function to clean up old canonical jobs (older than 30 days)
CREATE OR REPLACE FUNCTION cleanup_old_canonical_jobs()
RETURNS INTEGER AS $
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM canonical_jobs 
    WHERE status IN ('processed', 'cancelled', 'failed')
    AND updated_at < NOW() - INTERVAL '30 days';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$ LANGUAGE plpgsql;

-- Add trigger to update updated_at timestamp
CREATE TRIGGER update_canonical_jobs_updated_at
    BEFORE UPDATE ON canonical_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create view for pending canonical jobs ready for processing
CREATE VIEW pending_canonical_jobs AS
SELECT 
    cj.*,
    a.title as article_title,
    a.slug as article_slug,
    CASE 
        WHEN cj.target_type = 'tag' THEN t.name
        WHEN cj.target_type = 'category' THEN c.name
        ELSE NULL
    END as target_name,
    CASE 
        WHEN cj.target_type = 'tag' THEN t.slug
        WHEN cj.target_type = 'category' THEN c.slug
        ELSE NULL
    END as target_slug
FROM canonical_jobs cj
LEFT JOIN articles a ON cj.article_id = a.id
LEFT JOIN tags t ON cj.target_type = 'tag' AND cj.target_id = t.id
LEFT JOIN categories c ON cj.target_type = 'category' AND cj.target_id = c.id
WHERE cj.status = 'pending'
AND (cj.admin_override = true OR cj.scheduled_at <= NOW())
ORDER BY cj.admin_override DESC, cj.scheduled_at ASC;

-- Comments for documentation
COMMENT ON TABLE canonical_jobs IS 'Jobs for delayed canonicalization processing with 48-hour delay';
COMMENT ON COLUMN canonical_jobs.target_type IS 'Type of canonical target: tag, category, or url';
COMMENT ON COLUMN canonical_jobs.target_id IS 'ID of target tag or category (NULL for url type)';
COMMENT ON COLUMN canonical_jobs.target_url IS 'Custom canonical URL (for url type)';
COMMENT ON COLUMN canonical_jobs.admin_override IS 'If true, process immediately instead of waiting 48 hours';
COMMENT ON COLUMN canonical_jobs.retry_count IS 'Number of processing attempts (max 3 before permanent failure)';

COMMENT ON FUNCTION generate_canonical_url IS 'Generates canonical URLs for tags, categories, or custom URLs';
COMMENT ON FUNCTION process_canonical_job IS 'Processes a single canonical job and updates the article';
COMMENT ON FUNCTION schedule_canonical_job IS 'Schedules a new canonical job with 48-hour delay or immediate processing';
COMMENT ON FUNCTION cleanup_old_canonical_jobs IS 'Removes old processed/cancelled/failed canonical jobs';