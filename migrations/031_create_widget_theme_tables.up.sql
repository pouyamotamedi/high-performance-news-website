-- Create widgets table
CREATE TABLE widgets (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    description TEXT,
    config JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create widget_placements table
CREATE TABLE widget_placements (
    id BIGSERIAL PRIMARY KEY,
    widget_id BIGINT NOT NULL REFERENCES widgets(id) ON DELETE CASCADE,
    page_type VARCHAR(50) NOT NULL,
    zone VARCHAR(50) NOT NULL,
    position INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create themes table
CREATE TABLE themes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT false,
    is_default BOOLEAN DEFAULT false,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create template_overrides table
CREATE TABLE template_overrides (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    template_path VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for widgets
CREATE INDEX idx_widgets_type ON widgets(type);
CREATE INDEX idx_widgets_active ON widgets(is_active);
CREATE INDEX idx_widgets_sort_order ON widgets(sort_order);

-- Create indexes for widget_placements
CREATE INDEX idx_widget_placements_widget_id ON widget_placements(widget_id);
CREATE INDEX idx_widget_placements_page_zone ON widget_placements(page_type, zone);
CREATE INDEX idx_widget_placements_active ON widget_placements(is_active);
CREATE INDEX idx_widget_placements_position ON widget_placements(position);

-- Create indexes for themes
CREATE INDEX idx_themes_active ON themes(is_active);
CREATE INDEX idx_themes_default ON themes(is_default);

-- Create indexes for template_overrides
CREATE INDEX idx_template_overrides_path ON template_overrides(template_path);
CREATE INDEX idx_template_overrides_active ON template_overrides(is_active);

-- Ensure only one active theme at a time
CREATE UNIQUE INDEX idx_themes_single_active ON themes(is_active) WHERE is_active = true;

-- Ensure only one default theme at a time
CREATE UNIQUE INDEX idx_themes_single_default ON themes(is_default) WHERE is_default = true;

-- Ensure unique template override paths for active overrides
CREATE UNIQUE INDEX idx_template_overrides_unique_path ON template_overrides(template_path) WHERE is_active = true;

-- Add constraints
ALTER TABLE widget_placements ADD CONSTRAINT chk_widget_placements_page_type 
    CHECK (page_type IN ('homepage', 'article', 'category', 'tag', 'search', 'author', 'global'));

ALTER TABLE widget_placements ADD CONSTRAINT chk_widget_placements_zone 
    CHECK (zone IN ('header', 'sidebar', 'footer', 'content', 'after_title', 'before_content', 'after_content'));

ALTER TABLE widgets ADD CONSTRAINT chk_widgets_type 
    CHECK (type IN ('latest_articles', 'popular_articles', 'trending_articles', 'categories', 'tags', 'search', 'newsletter', 'custom_html', 'advertisement', 'social_media'));

-- Insert default theme
INSERT INTO themes (name, description, is_active, is_default, config) VALUES (
    'Default Theme',
    'Default theme for the news website',
    true,
    true,
    '{
        "colors": {
            "primary": "#3b82f6",
            "secondary": "#64748b",
            "accent": "#f59e0b",
            "background": "#ffffff",
            "surface": "#f8fafc",
            "text": "#1e293b",
            "text_muted": "#64748b",
            "border": "#e2e8f0",
            "success": "#10b981",
            "warning": "#f59e0b",
            "error": "#ef4444",
            "info": "#3b82f6"
        },
        "typography": {
            "font_family": "Inter, system-ui, sans-serif",
            "heading_font": "Inter, system-ui, sans-serif",
            "base_font_size": "16px",
            "line_height": 1.6,
            "heading_weight": "600",
            "body_weight": "400",
            "letter_spacing": "0"
        },
        "layout": {
            "max_width": "1200px",
            "sidebar_width": "300px",
            "header_height": "80px",
            "footer_height": "auto",
            "border_radius": "8px",
            "spacing": "1rem",
            "grid_columns": 12,
            "show_sidebar": true,
            "sidebar_position": "right",
            "header_style": "sticky",
            "footer_style": "static"
        },
        "branding": {
            "site_name": "News Website",
            "site_description": "Your trusted source for news",
            "logo_url": "",
            "favicon_url": "",
            "show_site_name": true,
            "show_description": true
        },
        "custom_css": "",
        "custom_js": ""
    }'
);

-- Insert some default widgets
INSERT INTO widgets (name, type, title, description, config, is_active, sort_order) VALUES 
(
    'Latest Articles Sidebar',
    'latest_articles',
    'Latest Articles',
    'Display the 5 most recent articles in the sidebar',
    '{"article_count": 5, "show_excerpt": false, "show_date": true, "show_author": false, "show_image": true, "cache_enabled": true, "cache_ttl": 900000000000}',
    true,
    10
),
(
    'Popular Articles Sidebar',
    'popular_articles',
    'Popular Articles',
    'Display the 5 most popular articles in the sidebar',
    '{"article_count": 5, "cache_enabled": true, "cache_ttl": 1800000000000}',
    true,
    20
),
(
    'Categories Widget',
    'categories',
    'Categories',
    'Display article categories',
    '{"show_hierarchy": true, "max_depth": 2, "show_count": true, "cache_enabled": true, "cache_ttl": 1800000000000}',
    true,
    30
),
(
    'Search Widget',
    'search',
    'Search',
    'Search form for articles',
    '{"cache_enabled": false}',
    true,
    40
),
(
    'Newsletter Subscription',
    'newsletter',
    'Subscribe to Newsletter',
    'Newsletter subscription form',
    '{"cache_enabled": false}',
    true,
    50
);

-- Insert default widget placements
INSERT INTO widget_placements (widget_id, page_type, zone, position, is_active) VALUES 
(1, 'global', 'sidebar', 1, true),  -- Latest Articles in sidebar
(2, 'global', 'sidebar', 2, true),  -- Popular Articles in sidebar
(3, 'global', 'sidebar', 3, true),  -- Categories in sidebar
(4, 'global', 'header', 1, true),   -- Search in header
(5, 'global', 'footer', 1, true);   -- Newsletter in footer