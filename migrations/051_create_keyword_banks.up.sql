-- Create keyword_banks table for custom keyword banks with dedicated URLs
CREATE TABLE IF NOT EXISTS keyword_banks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    url VARCHAR(500) NOT NULL,
    keywords JSONB NOT NULL DEFAULT '[]'::jsonb,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index for faster keyword lookups
CREATE INDEX idx_keyword_banks_active ON keyword_banks(is_active);
CREATE INDEX idx_keyword_banks_keywords ON keyword_banks USING gin(keywords);

COMMENT ON TABLE keyword_banks IS 'Custom keyword banks with dedicated URLs for auto-linking';
COMMENT ON COLUMN keyword_banks.keywords IS 'JSON array of keywords for this bank';
COMMENT ON COLUMN keyword_banks.url IS 'Target URL for links (e.g., https://example.com/page)';
