-- Migration: Fix image_variants schema to match new responsive image system
-- This migration updates existing tables to support the new image variant structure

-- Add hash column to images table if not exists
ALTER TABLE images ADD COLUMN IF NOT EXISTS hash VARCHAR(64);
CREATE INDEX IF NOT EXISTS idx_images_hash ON images (hash) WHERE hash IS NOT NULL;

-- Add new columns to image_variants table
ALTER TABLE image_variants ADD COLUMN IF NOT EXISTS size VARCHAR(50);
ALTER TABLE image_variants ADD COLUMN IF NOT EXISTS format VARCHAR(20);
ALTER TABLE image_variants ADD COLUMN IF NOT EXISTS url TEXT;
ALTER TABLE image_variants ADD COLUMN IF NOT EXISTS quality INTEGER DEFAULT 85;

-- Make variant_name nullable (old schema) since we now use size+format
ALTER TABLE image_variants ALTER COLUMN variant_name DROP NOT NULL;
ALTER TABLE image_variants ALTER COLUMN filename DROP NOT NULL;

-- Create new unique constraint if not exists (size + format combination)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'image_variants_image_id_size_format_key'
    ) THEN
        -- First, remove any duplicates that might exist
        DELETE FROM image_variants a USING image_variants b
        WHERE a.id > b.id 
        AND a.image_id = b.image_id 
        AND a.size = b.size 
        AND a.format = b.format
        AND a.size IS NOT NULL 
        AND a.format IS NOT NULL;
        
        -- Then create the unique constraint
        ALTER TABLE image_variants ADD CONSTRAINT image_variants_image_id_size_format_key 
            UNIQUE (image_id, size, format);
    END IF;
EXCEPTION
    WHEN duplicate_table THEN NULL;
    WHEN duplicate_object THEN NULL;
END $$;

-- Create index for size and format
CREATE INDEX IF NOT EXISTS idx_image_variants_size_format ON image_variants (size, format);

-- Add comment to document the changes
COMMENT ON TABLE image_variants IS 'Stores responsive image variants in multiple sizes and formats (WebP, JPEG, AVIF)';
COMMENT ON COLUMN image_variants.size IS 'Size category: thumbnail, small, medium, large, xlarge';
COMMENT ON COLUMN image_variants.format IS 'Image format: webp, jpeg, avif';
COMMENT ON COLUMN image_variants.url IS 'URL path to the variant file';
COMMENT ON COLUMN image_variants.quality IS 'Compression quality (1-100)';
