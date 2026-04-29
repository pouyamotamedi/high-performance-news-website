-- Migration: Enhance tag and category validation for Task 8
-- This migration adds constraints and indexes to support the enhanced validation system

-- Add unique constraint for category names (case-insensitive)
-- Note: We use a functional index for case-insensitive uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_name_unique_ci 
ON categories (LOWER(name), language_code);

-- Add unique constraint for tag names (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name_unique_ci 
ON tags (LOWER(name), language_code);

-- Add image columns to categories if they don't exist
-- (These might already exist from previous migrations)
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'categories' AND column_name = 'image_url') THEN
        ALTER TABLE categories ADD COLUMN image_url VARCHAR(500);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'categories' AND column_name = 'image_alt_text') THEN
        ALTER TABLE categories ADD COLUMN image_alt_text VARCHAR(200);
    END IF;
END $$;

-- Ensure keywords column has proper GIN index for efficient searching
-- (This should already exist but let's make sure)
CREATE INDEX IF NOT EXISTS idx_tags_keywords_gin ON tags USING GIN(keywords);

-- Add a function to validate keyword uniqueness across tags
-- This function will be used by the application for validation
CREATE OR REPLACE FUNCTION validate_tag_keyword_uniqueness(
    p_keywords JSONB,
    p_exclude_tag_id BIGINT DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    keyword_text TEXT;
    existing_tag_name TEXT;
BEGIN
    -- Check each keyword against existing tags
    FOR keyword_text IN SELECT jsonb_array_elements_text(p_keywords) LOOP
        -- Normalize the keyword (lowercase, trimmed)
        keyword_text := LOWER(TRIM(keyword_text));
        
        -- Skip empty keywords
        IF keyword_text = '' THEN
            CONTINUE;
        END IF;
        
        -- Check if this keyword exists in any other tag
        SELECT t.name INTO existing_tag_name
        FROM tags t
        WHERE t.id != COALESCE(p_exclude_tag_id, 0)
          AND t.keywords ? keyword_text;
        
        -- If found, return false (not unique)
        IF existing_tag_name IS NOT NULL THEN
            RAISE EXCEPTION 'Keyword "%" is already used by tag "%"', keyword_text, existing_tag_name;
        END IF;
    END LOOP;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Add a trigger to automatically validate keyword uniqueness on insert/update
CREATE OR REPLACE FUNCTION trigger_validate_tag_keywords()
RETURNS TRIGGER AS $$
BEGIN
    -- Validate keyword uniqueness
    PERFORM validate_tag_keyword_uniqueness(NEW.keywords, NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger (drop first if exists)
DROP TRIGGER IF EXISTS tr_validate_tag_keywords ON tags;
CREATE TRIGGER tr_validate_tag_keywords
    BEFORE INSERT OR UPDATE ON tags
    FOR EACH ROW
    EXECUTE FUNCTION trigger_validate_tag_keywords();

-- Add comments for documentation
COMMENT ON FUNCTION validate_tag_keyword_uniqueness(JSONB, BIGINT) IS 
'Validates that keywords are unique across all tags. Used for Task 8 keyword validation.';

COMMENT ON TRIGGER tr_validate_tag_keywords ON tags IS 
'Automatically validates keyword uniqueness when inserting or updating tags.';

COMMENT ON INDEX idx_categories_name_unique_ci IS 
'Ensures category names are unique per language (case-insensitive).';

COMMENT ON INDEX idx_tags_name_unique_ci IS 
'Ensures tag names are unique per language (case-insensitive).';