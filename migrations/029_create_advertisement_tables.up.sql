-- Advertisement Management System Migration
-- Requirements: 11, 28

-- Advertisement campaigns table
CREATE TABLE advertisement_campaigns (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    advertiser_name VARCHAR(255) NOT NULL,
    advertiser_email VARCHAR(255),
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE,
    budget_total DECIMAL(10,2),
    budget_daily DECIMAL(10,2),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'draft')),
    priority INTEGER DEFAULT 1 CHECK (priority >= 1 AND priority <= 10),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Advertisement slots/placements table
CREATE TABLE advertisement_slots (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    page_type VARCHAR(50) NOT NULL CHECK (page_type IN ('homepage', 'article', 'category', 'tag', 'search', 'all')),
    position VARCHAR(50) NOT NULL CHECK (position IN ('header', 'sidebar', 'content-top', 'content-middle', 'content-bottom', 'footer', 'floating')),
    width INTEGER,
    height INTEGER,
    is_active BOOLEAN DEFAULT true,
    lazy_load BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Advertisement creatives table
CREATE TABLE advertisement_creatives (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL REFERENCES advertisement_campaigns(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('image', 'html', 'script', 'video')),
    content TEXT NOT NULL, -- Image URL, HTML content, or script code
    alt_text VARCHAR(255),
    click_url VARCHAR(1000),
    width INTEGER,
    height INTEGER,
    file_size INTEGER, -- in bytes for performance tracking
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Advertisement targeting rules table
CREATE TABLE advertisement_targeting (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL REFERENCES advertisement_campaigns(id) ON DELETE CASCADE,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('category', 'tag', 'page_type', 'device', 'time')),
    target_value VARCHAR(255) NOT NULL,
    is_include BOOLEAN DEFAULT true, -- true for include, false for exclude
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Advertisement placements (many-to-many between campaigns and slots)
CREATE TABLE advertisement_placements (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL REFERENCES advertisement_campaigns(id) ON DELETE CASCADE,
    slot_id BIGINT NOT NULL REFERENCES advertisement_slots(id) ON DELETE CASCADE,
    creative_id BIGINT NOT NULL REFERENCES advertisement_creatives(id) ON DELETE CASCADE,
    weight INTEGER DEFAULT 1, -- For A/B testing and rotation
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(campaign_id, slot_id, creative_id)
);

-- Advertisement impressions tracking (partitioned by date for performance)
CREATE TABLE advertisement_impressions (
    id BIGSERIAL,
    placement_id BIGINT NOT NULL,
    campaign_id BIGINT NOT NULL,
    slot_id BIGINT NOT NULL,
    creative_id BIGINT NOT NULL,
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    page_url TEXT,
    device_type VARCHAR(20), -- mobile, tablet, desktop
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Advertisement clicks tracking (partitioned by date for performance)
CREATE TABLE advertisement_clicks (
    id BIGSERIAL,
    placement_id BIGINT NOT NULL,
    campaign_id BIGINT NOT NULL,
    slot_id BIGINT NOT NULL,
    creative_id BIGINT NOT NULL,
    impression_id BIGINT, -- Link to impression if available
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    page_url TEXT,
    device_type VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create initial partitions for current month
CREATE TABLE advertisement_impressions_2024_01 PARTITION OF advertisement_impressions
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE advertisement_clicks_2024_01 PARTITION OF advertisement_clicks
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Indexes for performance
CREATE INDEX idx_campaigns_status_dates ON advertisement_campaigns (status, start_date, end_date);
CREATE INDEX idx_campaigns_priority ON advertisement_campaigns (priority DESC, created_at DESC);

CREATE INDEX idx_slots_page_position ON advertisement_slots (page_type, position, is_active);
CREATE INDEX idx_slots_active ON advertisement_slots (is_active, created_at DESC);

CREATE INDEX idx_creatives_campaign ON advertisement_creatives (campaign_id, is_active);
CREATE INDEX idx_creatives_type ON advertisement_creatives (type, is_active);

CREATE INDEX idx_targeting_campaign ON advertisement_targeting (campaign_id, target_type);
CREATE INDEX idx_targeting_value ON advertisement_targeting (target_type, target_value);

CREATE INDEX idx_placements_campaign ON advertisement_placements (campaign_id, is_active);
CREATE INDEX idx_placements_slot ON advertisement_placements (slot_id, is_active);
CREATE INDEX idx_placements_weight ON advertisement_placements (weight DESC, created_at);

-- BRIN indexes for time-series data (impressions and clicks)
CREATE INDEX idx_impressions_created_brin ON advertisement_impressions 
USING BRIN (created_at) WITH (pages_per_range = 128);

CREATE INDEX idx_clicks_created_brin ON advertisement_clicks 
USING BRIN (created_at) WITH (pages_per_range = 128);

-- Regular indexes for analytics queries
CREATE INDEX idx_impressions_campaign_date ON advertisement_impressions (campaign_id, created_at);
CREATE INDEX idx_impressions_placement_date ON advertisement_impressions (placement_id, created_at);
CREATE INDEX idx_impressions_device ON advertisement_impressions (device_type, created_at);

CREATE INDEX idx_clicks_campaign_date ON advertisement_clicks (campaign_id, created_at);
CREATE INDEX idx_clicks_placement_date ON advertisement_clicks (placement_id, created_at);
CREATE INDEX idx_clicks_device ON advertisement_clicks (device_type, created_at);

-- Insert default advertisement slots
INSERT INTO advertisement_slots (name, slug, description, page_type, position, width, height, lazy_load) VALUES
('Homepage Header Banner', 'homepage-header', 'Large banner at the top of homepage', 'homepage', 'header', 728, 90, false),
('Homepage Sidebar Top', 'homepage-sidebar-top', 'Sidebar advertisement at top', 'homepage', 'sidebar', 300, 250, true),
('Homepage Sidebar Bottom', 'homepage-sidebar-bottom', 'Sidebar advertisement at bottom', 'homepage', 'sidebar', 300, 600, true),
('Article Header Banner', 'article-header', 'Banner above article content', 'article', 'content-top', 728, 90, false),
('Article Content Middle', 'article-content-middle', 'Advertisement within article content', 'article', 'content-middle', 300, 250, true),
('Article Sidebar', 'article-sidebar', 'Sidebar advertisement on article pages', 'article', 'sidebar', 300, 250, true),
('Category Header', 'category-header', 'Banner on category pages', 'category', 'header', 728, 90, false),
('Category Sidebar', 'category-sidebar', 'Sidebar on category pages', 'category', 'sidebar', 300, 250, true),
('Tag Header', 'tag-header', 'Banner on tag pages', 'tag', 'header', 728, 90, false),
('Tag Sidebar', 'tag-sidebar', 'Sidebar on tag pages', 'tag', 'sidebar', 300, 250, true),
('Mobile Banner', 'mobile-banner', 'Mobile-optimized banner', 'all', 'content-top', 320, 50, false),
('Floating Ad', 'floating-ad', 'Floating advertisement', 'all', 'floating', 300, 250, true);