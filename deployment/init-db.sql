-- News Website Database Initialization
-- This script runs automatically when the database container starts for the first time

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Set timezone
SET timezone = 'UTC';

-- ============================================
-- Images and Media Tables
-- ============================================

-- Create images table for storing article images
CREATE TABLE IF NOT EXISTS images (
    id BIGSERIAL PRIMARY KEY,
    original_url VARCHAR(500) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    alt_text TEXT,
    caption TEXT,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    hash VARCHAR(64),  -- Content hash for deduplication
    article_id BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for images table
CREATE INDEX IF NOT EXISTS idx_images_article_id ON images (article_id);
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images (created_at);
CREATE INDEX IF NOT EXISTS idx_images_hash ON images (hash) WHERE hash IS NOT NULL;

-- Create image_variants table for responsive images
CREATE TABLE IF NOT EXISTS image_variants (
    id BIGSERIAL PRIMARY KEY,
    image_id BIGINT NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    size VARCHAR(50) NOT NULL,        -- e.g., 'thumbnail', 'small', 'medium', 'large', 'xlarge'
    format VARCHAR(20) NOT NULL,      -- e.g., 'webp', 'jpeg', 'avif'
    url TEXT NOT NULL,                -- URL path to the variant
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    file_size BIGINT NOT NULL,
    quality INTEGER DEFAULT 85,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (image_id, size, format)
);

-- Create indexes for image variants
CREATE INDEX IF NOT EXISTS idx_image_variants_image_id ON image_variants (image_id);
CREATE INDEX IF NOT EXISTS idx_image_variants_size_format ON image_variants (size, format);

-- Add comments
COMMENT ON TABLE images IS 'Stores article images including featured images and inline images';
COMMENT ON TABLE image_variants IS 'Stores responsive image variants in multiple sizes and formats (WebP, JPEG, AVIF)';
COMMENT ON COLUMN images.hash IS 'SHA-256 hash of image content for deduplication';
COMMENT ON COLUMN image_variants.size IS 'Size category: thumbnail, small, medium, large, xlarge';
COMMENT ON COLUMN image_variants.format IS 'Image format: webp, jpeg, avif';
