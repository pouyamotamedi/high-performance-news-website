-- Create spam_settings table for configurable spam detection
CREATE TABLE IF NOT EXISTS spam_settings (
    id INTEGER PRIMARY KEY DEFAULT 1,
    keywords JSONB NOT NULL DEFAULT '["viagra", "casino", "lottery", "winner", "congratulations", "click here", "free money", "buy now", "limited time", "act now"]',
    threshold DECIMAL(3,2) NOT NULL DEFAULT 0.5,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT single_row CHECK (id = 1)
);

-- Insert default settings
INSERT INTO spam_settings (id, keywords, threshold) 
VALUES (1, '["viagra", "casino", "lottery", "winner", "congratulations", "click here", "free money", "buy now", "limited time", "act now"]', 0.5)
ON CONFLICT (id) DO NOTHING;