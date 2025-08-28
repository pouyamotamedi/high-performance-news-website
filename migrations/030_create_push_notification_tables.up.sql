-- Push notification subscription table
CREATE TABLE push_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    user_agent TEXT,
    ip_address INET,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(endpoint)
);

-- Push notification table
CREATE TABLE push_notifications (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    icon VARCHAR(500),
    badge VARCHAR(500),
    image VARCHAR(500),
    url VARCHAR(500),
    data JSONB,
    target_type VARCHAR(20) DEFAULT 'all',
    target_value VARCHAR(255),
    scheduled_at TIMESTAMP WITH TIME ZONE,
    sent_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'pending',
    total_sent INTEGER DEFAULT 0,
    total_delivered INTEGER DEFAULT 0,
    total_clicked INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Push delivery tracking table (partitioned by created_at for performance)
CREATE TABLE push_deliveries (
    id BIGSERIAL,
    notification_id BIGINT NOT NULL REFERENCES push_notifications(id) ON DELETE CASCADE,
    subscription_id BIGINT NOT NULL REFERENCES push_subscriptions(id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'sent',
    error_message TEXT,
    delivered_at TIMESTAMP WITH TIME ZONE,
    clicked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Push notification templates
CREATE TABLE push_templates (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    icon VARCHAR(500),
    badge VARCHAR(500),
    image VARCHAR(500),
    url VARCHAR(500),
    variables JSONB, -- Array of template variables
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User notification preferences
CREATE TABLE notification_preferences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    subscription_id BIGINT NOT NULL REFERENCES push_subscriptions(id) ON DELETE CASCADE,
    breaking_news BOOLEAN DEFAULT true,
    category_updates BOOLEAN DEFAULT true,
    tag_updates BOOLEAN DEFAULT false,
    author_updates BOOLEAN DEFAULT false,
    preferred_categories JSONB, -- Array of category IDs
    preferred_tags JSONB, -- Array of tag IDs
    preferred_authors JSONB, -- Array of author IDs
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(subscription_id)
);

-- Create initial partition for push_deliveries (current month)
CREATE TABLE push_deliveries_2024_01 PARTITION OF push_deliveries
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Indexes for performance
CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions(user_id);
CREATE INDEX idx_push_subscriptions_active ON push_subscriptions(is_active) WHERE is_active = true;
CREATE INDEX idx_push_subscriptions_created_at ON push_subscriptions(created_at);

CREATE INDEX idx_push_notifications_status ON push_notifications(status);
CREATE INDEX idx_push_notifications_scheduled ON push_notifications(scheduled_at) WHERE scheduled_at IS NOT NULL;
CREATE INDEX idx_push_notifications_target ON push_notifications(target_type, target_value);
CREATE INDEX idx_push_notifications_created_at ON push_notifications(created_at);

CREATE INDEX idx_push_deliveries_notification ON push_deliveries(notification_id);
CREATE INDEX idx_push_deliveries_subscription ON push_deliveries(subscription_id);
CREATE INDEX idx_push_deliveries_status ON push_deliveries(status);
CREATE INDEX idx_push_deliveries_created_brin ON push_deliveries USING BRIN (created_at);

CREATE INDEX idx_push_templates_active ON push_templates(is_active) WHERE is_active = true;
CREATE INDEX idx_push_templates_name ON push_templates(name);

CREATE INDEX idx_notification_preferences_user ON notification_preferences(user_id);
CREATE INDEX idx_notification_preferences_subscription ON notification_preferences(subscription_id);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_push_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_push_subscriptions_updated_at BEFORE UPDATE ON push_subscriptions FOR EACH ROW EXECUTE FUNCTION update_push_updated_at_column();
CREATE TRIGGER update_push_notifications_updated_at BEFORE UPDATE ON push_notifications FOR EACH ROW EXECUTE FUNCTION update_push_updated_at_column();
CREATE TRIGGER update_push_deliveries_updated_at BEFORE UPDATE ON push_deliveries FOR EACH ROW EXECUTE FUNCTION update_push_updated_at_column();
CREATE TRIGGER update_push_templates_updated_at BEFORE UPDATE ON push_templates FOR EACH ROW EXECUTE FUNCTION update_push_updated_at_column();
CREATE TRIGGER update_notification_preferences_updated_at BEFORE UPDATE ON notification_preferences FOR EACH ROW EXECUTE FUNCTION update_push_updated_at_column();