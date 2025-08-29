-- Create configuration management tables

-- Main configuration table
CREATE TABLE configurations (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'string',
    category VARCHAR(100) NOT NULL DEFAULT 'general',
    description TEXT,
    is_secret BOOLEAN DEFAULT FALSE,
    validation JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Feature flags table
CREATE TABLE feature_flags (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT FALSE,
    rollout JSONB,
    conditions JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Configuration history table for audit trail
CREATE TABLE configuration_history (
    id BIGSERIAL PRIMARY KEY,
    config_key VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by BIGINT,
    change_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Configuration snapshots table
CREATE TABLE configuration_snapshots (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    created_by BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_configurations_category ON configurations(category);
CREATE INDEX idx_configurations_type ON configurations(type);
CREATE INDEX idx_configurations_updated_at ON configurations(updated_at);

CREATE INDEX idx_feature_flags_enabled ON feature_flags(enabled);
CREATE INDEX idx_feature_flags_updated_at ON feature_flags(updated_at);

CREATE INDEX idx_configuration_history_config_key ON configuration_history(config_key);
CREATE INDEX idx_configuration_history_created_at ON configuration_history(created_at);
CREATE INDEX idx_configuration_history_changed_by ON configuration_history(changed_by);

CREATE INDEX idx_configuration_snapshots_created_by ON configuration_snapshots(created_by);
CREATE INDEX idx_configuration_snapshots_created_at ON configuration_snapshots(created_at);

-- Insert default configurations
INSERT INTO configurations (key, value, type, category, description) VALUES
-- Site settings
('site_name', 'High Performance News Website', 'string', 'site', 'Website name'),
('site_description', 'A fast and scalable news platform', 'string', 'site', 'Website description'),
('site_url', 'https://example.com', 'string', 'site', 'Website URL'),
('site_logo', '', 'string', 'site', 'Website logo URL'),
('site_favicon', '', 'string', 'site', 'Website favicon URL'),

-- Performance settings
('cache_ttl', '3600', 'int', 'performance', 'Cache TTL in seconds'),
('static_generation', 'true', 'bool', 'performance', 'Enable static HTML generation'),
('compression_enabled', 'true', 'bool', 'performance', 'Enable compression'),

-- Feature settings
('comments_enabled', 'true', 'bool', 'features', 'Enable comments system'),
('registration_enabled', 'false', 'bool', 'features', 'Enable user registration'),
('search_enabled', 'true', 'bool', 'features', 'Enable search functionality'),
('analytics_enabled', 'false', 'bool', 'features', 'Enable analytics tracking'),
('social_sharing', 'true', 'bool', 'features', 'Enable social sharing'),
('newsletter_enabled', 'false', 'bool', 'features', 'Enable newsletter'),

-- Appearance settings
('theme', 'default', 'string', 'appearance', 'Website theme'),
('primary_color', '#007bff', 'string', 'appearance', 'Primary color'),
('secondary_color', '#6c757d', 'string', 'appearance', 'Secondary color'),

-- Content settings
('articles_per_page', '10', 'int', 'content', 'Articles per page'),
('excerpt_length', '150', 'int', 'content', 'Article excerpt length'),
('allow_comments', 'true', 'bool', 'content', 'Allow comments on articles'),
('moderate_comments', 'true', 'bool', 'content', 'Moderate comments before publishing'),

-- SEO settings
('meta_title', '', 'string', 'seo', 'Default meta title'),
('meta_description', '', 'string', 'seo', 'Default meta description'),
('meta_keywords', '', 'string', 'seo', 'Default meta keywords'),

-- Analytics settings
('google_analytics', '', 'string', 'analytics', 'Google Analytics tracking ID'),

-- System settings
('admin_email', '', 'string', 'system', 'Administrator email');

-- Insert default feature flags
INSERT INTO feature_flags (key, name, description, enabled) VALUES
('new_editor', 'New Article Editor', 'Enable the new rich text editor for articles', false),
('advanced_search', 'Advanced Search', 'Enable advanced search with filters', false),
('dark_mode', 'Dark Mode', 'Enable dark mode theme option', false),
('ai_content_suggestions', 'AI Content Suggestions', 'Enable AI-powered content suggestions', false),
('real_time_notifications', 'Real-time Notifications', 'Enable real-time push notifications', false),
('beta_features', 'Beta Features', 'Enable access to beta features', false);