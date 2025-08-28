-- Email subscribers and newsletter system tables
CREATE TABLE email_subscribers (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, confirmed, unsubscribed, bounced
    confirmation_token VARCHAR(255),
    unsubscribe_token VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    preferences JSONB DEFAULT '{}', -- Category preferences, frequency, etc.
    source VARCHAR(50) DEFAULT 'website', -- website, api, import
    confirmed_at TIMESTAMP WITH TIME ZONE,
    unsubscribed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE email_campaigns (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    template_id BIGINT,
    content TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, scheduled, sending, sent, cancelled
    scheduled_at TIMESTAMP WITH TIME ZONE,
    sent_at TIMESTAMP WITH TIME ZONE,
    recipient_count INTEGER DEFAULT 0,
    sent_count INTEGER DEFAULT 0,
    delivered_count INTEGER DEFAULT 0,
    opened_count INTEGER DEFAULT 0,
    clicked_count INTEGER DEFAULT 0,
    bounced_count INTEGER DEFAULT 0,
    unsubscribed_count INTEGER DEFAULT 0,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE email_templates (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    html_content TEXT NOT NULL,
    text_content TEXT,
    template_type VARCHAR(50) NOT NULL, -- newsletter, welcome, confirmation, notification
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE email_sends (
    id BIGSERIAL,
    campaign_id BIGINT NOT NULL,
    subscriber_id BIGINT NOT NULL,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, sent, delivered, bounced, failed
    external_id VARCHAR(255), -- SendGrid/Mailgun message ID
    sent_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    opened_at TIMESTAMP WITH TIME ZONE,
    clicked_at TIMESTAMP WITH TIME ZONE,
    bounced_at TIMESTAMP WITH TIME ZONE,
    bounce_reason TEXT,
    unsubscribed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at),
    FOREIGN KEY (campaign_id) REFERENCES email_campaigns(id),
    FOREIGN KEY (subscriber_id) REFERENCES email_subscribers(id)
) PARTITION BY RANGE (created_at);

-- Create initial partition for email sends
CREATE TABLE email_sends_2024_01 PARTITION OF email_sends
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Indexes for performance
CREATE INDEX idx_email_subscribers_email ON email_subscribers (email);
CREATE INDEX idx_email_subscribers_status ON email_subscribers (status);
CREATE INDEX idx_email_subscribers_created ON email_subscribers (created_at);
CREATE INDEX idx_email_campaigns_status ON email_campaigns (status);
CREATE INDEX idx_email_campaigns_scheduled ON email_campaigns (scheduled_at) WHERE status = 'scheduled';
CREATE INDEX idx_email_sends_campaign ON email_sends (campaign_id);
CREATE INDEX idx_email_sends_subscriber ON email_sends (subscriber_id);
CREATE INDEX idx_email_sends_status ON email_sends (status);
CREATE INDEX idx_email_sends_external_id ON email_sends (external_id) WHERE external_id IS NOT NULL;

-- BRIN index for time-series data
CREATE INDEX idx_email_sends_created_brin ON email_sends 
USING BRIN (created_at) WITH (pages_per_range = 64);

-- Insert default email templates
INSERT INTO email_templates (name, subject, html_content, text_content, template_type) VALUES
('Welcome Email', 'Welcome to {{site_name}}!', 
 '<h1>Welcome {{first_name}}!</h1><p>Thank you for subscribing to our newsletter.</p><p><a href="{{unsubscribe_url}}">Unsubscribe</a></p>',
 'Welcome {{first_name}}! Thank you for subscribing to our newsletter. Unsubscribe: {{unsubscribe_url}}',
 'welcome'),
('Confirmation Email', 'Please confirm your subscription to {{site_name}}',
 '<h1>Confirm Your Subscription</h1><p>Please click the link below to confirm your subscription:</p><p><a href="{{confirmation_url}}">Confirm Subscription</a></p>',
 'Please confirm your subscription by visiting: {{confirmation_url}}',
 'confirmation'),
('Newsletter Template', '{{subject}}',
 '<h1>{{subject}}</h1>{{content}}<hr><p><a href="{{unsubscribe_url}}">Unsubscribe</a></p>',
 '{{subject}}\n\n{{content}}\n\nUnsubscribe: {{unsubscribe_url}}',
 'newsletter');