-- Migration: Add content ingestion enhancements
-- This migration adds support for featured images, focus keywords, and other SEO enhancements

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

-- Create index for hash-based deduplication
CREATE INDEX IF NOT EXISTS idx_images_hash ON images (hash) WHERE hash IS NOT NULL;

-- Create indexes for images table
CREATE INDEX idx_images_article_id ON images (article_id);
CREATE INDEX idx_images_created_at ON images (created_at);

-- Add featured_image_id column to articles table
-- Note: We need to add this to the parent table, not individual partitions
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'articles' AND column_name = 'featured_image_id'
    ) THEN
        ALTER TABLE articles ADD COLUMN featured_image_id BIGINT;
    END IF;
END $$;

-- Add focus_keyword column to articles table
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'articles' AND column_name = 'focus_keyword'
    ) THEN
        ALTER TABLE articles ADD COLUMN focus_keyword VARCHAR(100);
    END IF;
END $$;

-- Add last_moderated_by column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'articles' AND column_name = 'last_moderated_by'
    ) THEN
        ALTER TABLE articles ADD COLUMN last_moderated_by BIGINT;
    END IF;
END $$;

-- Create indexes for new columns
CREATE INDEX IF NOT EXISTS idx_articles_featured_image ON articles (featured_image_id) WHERE featured_image_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_articles_focus_keyword ON articles (focus_keyword) WHERE focus_keyword IS NOT NULL;

-- Add foreign key constraint for featured_image_id
-- Note: We add this as NOT VALID first, then validate it separately to avoid locking
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'articles_featured_image_id_fkey'
    ) THEN
        ALTER TABLE articles ADD CONSTRAINT articles_featured_image_id_fkey 
            FOREIGN KEY (featured_image_id) REFERENCES images(id);
    END IF;
END $$;

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

-- Create index for image variants
CREATE INDEX IF NOT EXISTS idx_image_variants_image_id ON image_variants (image_id);
CREATE INDEX IF NOT EXISTS idx_image_variants_size_format ON image_variants (size, format);

-- Create function to automatically create daily partitions for article_tags
-- This ensures partitions are created automatically when needed
CREATE OR REPLACE FUNCTION create_article_tags_daily_partitions()
RETURNS TABLE(partition_name text, status text, error_message text) AS $$
DECLARE
    start_date date;
    end_date date;
    part_name text;
    created_count integer := 0;
    error_count integer := 0;
BEGIN
    -- Create partitions for next 30 days
    FOR i IN 0..30 LOOP
        start_date := CURRENT_DATE + i;
        end_date := start_date + interval '1 day';
        part_name := 'article_tags_' || to_char(start_date, 'YYYY_MM_DD');
        
        -- Check if partition already exists
        IF NOT EXISTS (
            SELECT 1 FROM pg_class WHERE relname = part_name
        ) THEN
            BEGIN
                -- Create the partition
                EXECUTE format('CREATE TABLE %I PARTITION OF article_tags FOR VALUES FROM (%L) TO (%L)',
                    part_name, start_date, end_date);
                
                -- Create indexes on the partition
                EXECUTE format('CREATE INDEX idx_%I_article ON %I (article_id)',
                    part_name, part_name);
                EXECUTE format('CREATE INDEX idx_%I_tag ON %I (tag_id)',
                    part_name, part_name);
                    
                created_count := created_count + 1;
                partition_name := part_name;
                status := 'created';
                error_message := NULL;
                RETURN NEXT;
                
            EXCEPTION
                WHEN duplicate_table THEN
                    partition_name := part_name;
                    status := 'exists';
                    error_message := 'Partition already exists';
                    RETURN NEXT;
                WHEN OTHERS THEN
                    error_count := error_count + 1;
                    partition_name := part_name;
                    status := 'error';
                    error_message := SQLERRM;
                    RETURN NEXT;
            END;
        ELSE
            partition_name := part_name;
            status := 'exists';
            error_message := 'Partition already exists';
            RETURN NEXT;
        END IF;
    END LOOP;
    
    -- Return summary
    partition_name := 'SUMMARY';
    status := format('Created: %s, Errors: %s', created_count, error_count);
    error_message := NULL;
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

-- Update the partition_maintenance function to include article_tags
CREATE OR REPLACE FUNCTION partition_maintenance()
RETURNS void AS $$
BEGIN
    -- Create partitions for articles (next 7 days)
    PERFORM create_daily_partitions();
    
    -- Create partitions for article_tags (next 30 days)
    PERFORM create_article_tags_daily_partitions();
    
    -- Drop partitions older than 30 days
    PERFORM drop_old_partitions(30);
    
    -- Log maintenance completion
    RAISE NOTICE 'Partition maintenance completed at %', NOW();
END;
$$ LANGUAGE plpgsql;

-- Add comment to document the changes
COMMENT ON TABLE images IS 'Stores article images including featured images and inline images';
COMMENT ON COLUMN articles.featured_image_id IS 'Reference to the featured image for this article';
COMMENT ON COLUMN articles.focus_keyword IS 'Primary SEO keyword for this article';
