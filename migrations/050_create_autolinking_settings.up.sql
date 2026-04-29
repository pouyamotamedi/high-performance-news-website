-- Create auto-linking settings table
CREATE TABLE IF NOT EXISTS autolinking_settings (
    id SERIAL PRIMARY KEY,
    global_enabled BOOLEAN DEFAULT true,
    content_ingestion_enabled BOOLEAN DEFAULT false,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default settings
INSERT INTO autolinking_settings (global_enabled, content_ingestion_enabled)
VALUES (true, false)
ON CONFLICT DO NOTHING;

-- Add comment
COMMENT ON TABLE autolinking_settings IS 'Stores global auto-linking system configuration';
COMMENT ON COLUMN autolinking_settings.global_enabled IS 'Enable auto-linking globally for new articles';
COMMENT ON COLUMN autolinking_settings.content_ingestion_enabled IS 'Enable auto-linking for Content Ingestion API';
