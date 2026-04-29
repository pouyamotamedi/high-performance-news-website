-- Content Sources table
CREATE TABLE content_sources (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('api', 'webhook', 'manual')),
    api_key VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    rate_limit INTEGER DEFAULT 100 CHECK (rate_limit >= 0 AND rate_limit <= 10000),
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for content_sources
CREATE INDEX idx_content_sources_api_key ON content_sources (api_key) WHERE is_active = true;
CREATE INDEX idx_content_sources_type ON content_sources (type);
CREATE INDEX idx_content_sources_priority ON content_sources (priority DESC);

-- Ingested Content table (simplified without partitioning)
CREATE TABLE ingested_content (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES content_sources(id),
    external_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT,
    author_name VARCHAR(100),
    author_email VARCHAR(255),
    category_name VARCHAR(100),
    tags JSONB DEFAULT '[]',
    published_at TIMESTAMP WITH TIME ZONE,
    source_url VARCHAR(500),
    content_hash VARCHAR(64) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processed', 'rejected', 'duplicate')),
    processed_at TIMESTAMP WITH TIME ZONE,
    article_id BIGINT, -- References articles(id) but no FK constraint due to partitioning
    rejection_reason TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for ingested_content
CREATE INDEX idx_ingested_content_source_external ON ingested_content (source_id, external_id);
CREATE INDEX idx_ingested_content_hash ON ingested_content (content_hash);
CREATE INDEX idx_ingested_content_status ON ingested_content (status, created_at DESC);
CREATE INDEX idx_ingested_content_source_url ON ingested_content (source_url) WHERE source_url IS NOT NULL;
CREATE INDEX idx_ingested_content_article ON ingested_content (article_id) WHERE article_id IS NOT NULL;

-- Content Ingestion Rate Limits table
CREATE TABLE content_ingestion_rate_limits (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES content_sources(id),
    requests_count INTEGER DEFAULT 1,
    window_start TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for rate limit cleanup
CREATE INDEX idx_rate_limits_cleanup ON content_ingestion_rate_limits (created_at);
CREATE INDEX idx_rate_limits_source_window ON content_ingestion_rate_limits (source_id, window_start);

-- Content Processing Statistics table
CREATE TABLE content_processing_stats (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES content_sources(id),
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    total_received INTEGER DEFAULT 0,
    total_processed INTEGER DEFAULT 0,
    total_rejected INTEGER DEFAULT 0,
    total_duplicates INTEGER DEFAULT 0,
    avg_processing_time_ms INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(source_id, date)
);

-- Indexes for processing stats
CREATE INDEX idx_processing_stats_source_date ON content_processing_stats (source_id, date DESC);
CREATE INDEX idx_processing_stats_date ON content_processing_stats (date DESC);

-- Create view for content source analytics
CREATE VIEW content_source_analytics AS
SELECT 
    cs.id,
    cs.name,
    cs.type,
    cs.is_active,
    COUNT(ic.id) as total_content,
    COUNT(CASE WHEN ic.status = 'processed' THEN 1 END) as processed_content,
    COUNT(CASE WHEN ic.status = 'pending' THEN 1 END) as pending_content,
    COUNT(CASE WHEN ic.status = 'rejected' THEN 1 END) as rejected_content,
    COUNT(CASE WHEN ic.status = 'duplicate' THEN 1 END) as duplicate_content,
    MAX(ic.created_at) as last_content_received
FROM content_sources cs
LEFT JOIN ingested_content ic ON cs.id = ic.source_id
GROUP BY cs.id, cs.name, cs.type, cs.is_active;