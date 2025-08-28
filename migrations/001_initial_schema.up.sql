-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create users table
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'reporter',
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    bio TEXT,
    avatar VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create categories table
CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    parent_id BIGINT REFERENCES categories(id),
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create tags table
CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    keywords JSONB,
    color VARCHAR(7) DEFAULT '#000000',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create articles table (partitioned by published_at)
CREATE TABLE articles (
    id BIGSERIAL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT,
    author_id BIGINT NOT NULL REFERENCES users(id),
    category_id BIGINT NOT NULL REFERENCES categories(id),
    status VARCHAR(20) DEFAULT 'draft',
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    view_count BIGINT DEFAULT 0,
    like_count BIGINT DEFAULT 0,
    dislike_count BIGINT DEFAULT 0,
    meta_title VARCHAR(60),
    meta_description VARCHAR(160),
    canonical_url VARCHAR(500),
    schema_type VARCHAR(50) DEFAULT 'NewsArticle',
    PRIMARY KEY (id, published_at)
) PARTITION BY RANGE (published_at);

-- Create article_tags junction table (partitioned by created_at)
CREATE TABLE article_tags (
    article_id BIGINT,
    tag_id BIGINT REFERENCES tags(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (article_id, tag_id, created_at)
) PARTITION BY RANGE (created_at);

-- Create indexes for performance
CREATE INDEX idx_users_username ON users (username);
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_role ON users (role);

CREATE INDEX idx_categories_slug ON categories (slug);
CREATE INDEX idx_categories_parent ON categories (parent_id);

CREATE INDEX idx_tags_slug ON tags (slug);
CREATE INDEX idx_tags_keywords ON tags USING gin(keywords);

-- Note: Article indexes will be created per partition
-- This is just the template for future partitions

-- Create article_views table (partitioned by created_at)
CREATE TABLE article_views (
    id BIGSERIAL,
    article_id BIGINT NOT NULL,
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create article_engagement table (partitioned by created_at)
CREATE TABLE article_engagement (
    id BIGSERIAL,
    article_id BIGINT NOT NULL,
    action VARCHAR(20) NOT NULL, -- 'like', 'dislike', 'share'
    ip_address INET,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create initial partitions for current month
-- Articles partitions
DO $$
DECLARE
    start_date date := date_trunc('month', CURRENT_DATE);
    end_date date := start_date + interval '1 month';
    partition_name text := 'articles_' || to_char(start_date, 'YYYY_MM');
BEGIN
    EXECUTE format('CREATE TABLE %I PARTITION OF articles FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
    
    -- Create indexes on the partition
    EXECUTE format('CREATE INDEX idx_%I_published_at ON %I (published_at DESC) WHERE status = ''published''',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_category ON %I (category_id, published_at DESC)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_author ON %I (author_id, published_at DESC)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_slug ON %I (slug) WHERE status = ''published''',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_search ON %I USING gin(to_tsvector(''english'', title || '' '' || content))',
        partition_name, partition_name);
    
    -- Create BRIN index for time-series performance
    EXECUTE format('CREATE INDEX idx_%I_published_brin ON %I USING BRIN (published_at) WITH (pages_per_range = 128)',
        partition_name, partition_name);
END $$;

-- Article_tags partitions
DO $$
DECLARE
    start_date date := date_trunc('month', CURRENT_DATE);
    end_date date := start_date + interval '1 month';
    partition_name text := 'article_tags_' || to_char(start_date, 'YYYY_MM');
BEGIN
    EXECUTE format('CREATE TABLE %I PARTITION OF article_tags FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
    
    -- Create indexes on the partition
    EXECUTE format('CREATE INDEX idx_%I_article ON %I (article_id)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_tag ON %I (tag_id)',
        partition_name, partition_name);
END $$;

-- Article_views partitions
DO $$
DECLARE
    start_date date := date_trunc('month', CURRENT_DATE);
    end_date date := start_date + interval '1 month';
    partition_name text := 'article_views_' || to_char(start_date, 'YYYY_MM');
BEGIN
    EXECUTE format('CREATE TABLE %I PARTITION OF article_views FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
    
    -- Create BRIN index for high-volume analytics data
    EXECUTE format('CREATE INDEX idx_%I_created_brin ON %I USING BRIN (created_at) WITH (pages_per_range = 64)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_article ON %I (article_id, created_at)',
        partition_name, partition_name);
END $$;

-- Article_engagement partitions
DO $$
DECLARE
    start_date date := date_trunc('month', CURRENT_DATE);
    end_date date := start_date + interval '1 month';
    partition_name text := 'article_engagement_' || to_char(start_date, 'YYYY_MM');
BEGIN
    EXECUTE format('CREATE TABLE %I PARTITION OF article_engagement FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
    
    -- Create BRIN index for high-volume analytics data
    EXECUTE format('CREATE INDEX idx_%I_created_brin ON %I USING BRIN (created_at) WITH (pages_per_range = 64)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_article ON %I (article_id, created_at)',
        partition_name, partition_name);
END $$;