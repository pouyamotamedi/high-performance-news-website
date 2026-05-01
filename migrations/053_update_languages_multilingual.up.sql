-- Migration 053: Update languages for multilingual support
-- Remove Persian (fa), add German (de), French (fr), Spanish (es)
-- Set English (en) as default language

-- Step 1: Update existing content from Persian to English
-- This must be done BEFORE removing the Persian language due to foreign key constraints

-- Update articles language_code from 'fa' to 'en'
UPDATE articles SET language_code = 'en' WHERE language_code = 'fa';

-- Update categories language_code from 'fa' to 'en'
UPDATE categories SET language_code = 'en' WHERE language_code = 'fa';

-- Update tags language_code from 'fa' to 'en'
UPDATE tags SET language_code = 'en' WHERE language_code = 'fa';

-- Step 2: Add new languages (German, French, Spanish)
INSERT INTO languages (code, name, native_name, direction, is_active, sort_order)
VALUES 
    ('de', 'German', 'Deutsch', 'ltr', true, 3),
    ('fr', 'French', 'Français', 'ltr', true, 4),
    ('es', 'Spanish', 'Español', 'ltr', true, 5)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    native_name = EXCLUDED.native_name,
    direction = EXCLUDED.direction,
    is_active = EXCLUDED.is_active,
    sort_order = EXCLUDED.sort_order;

-- Step 3: Activate English and Arabic, update sort orders
UPDATE languages SET is_active = true, sort_order = 1 WHERE code = 'en';
UPDATE languages SET is_active = true, sort_order = 5 WHERE code = 'ar';

-- Step 4: Remove Persian language (after content has been migrated)
DELETE FROM languages WHERE code = 'fa';

-- Step 5: Update the default language in any configuration tables if they exist
-- This handles the case where there might be a site_settings or config table
DO $$
BEGIN
    -- Check if site_settings table exists and has default_language column
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'site_settings' AND column_name = 'default_language'
    ) THEN
        UPDATE site_settings SET default_language = 'en' WHERE default_language = 'fa';
    END IF;
    
    -- Check if configurations table exists
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'configurations'
    ) THEN
        UPDATE configurations 
        SET value = '"en"' 
        WHERE key = 'default_language' AND value = '"fa"';
    END IF;
END $$;

-- Step 6: Update the get_articles_by_language function to use 'en' as default
CREATE OR REPLACE FUNCTION get_articles_by_language(
    p_language_code VARCHAR(5),
    p_fallback_language VARCHAR(5) DEFAULT 'en',
    p_limit INTEGER DEFAULT 20,
    p_offset INTEGER DEFAULT 0
)
RETURNS TABLE (
    id BIGINT,
    title VARCHAR(255),
    slug VARCHAR(255),
    excerpt TEXT,
    language_code VARCHAR(5),
    published_at TIMESTAMP WITH TIME ZONE,
    is_fallback BOOLEAN
) AS $func$
BEGIN
    RETURN QUERY
    WITH language_articles AS (
        -- Get articles in requested language
        SELECT a.id, a.title, a.slug, a.excerpt, a.language_code, a.published_at, false as is_fallback
        FROM articles a
        WHERE a.language_code = p_language_code 
        AND a.status = 'published'
        ORDER BY a.published_at DESC
        LIMIT p_limit OFFSET p_offset
    ),
    fallback_articles AS (
        -- Get fallback articles for missing content
        SELECT a.id, a.title, a.slug, a.excerpt, a.language_code, a.published_at, true as is_fallback
        FROM articles a
        WHERE a.language_code = p_fallback_language 
        AND a.status = 'published'
        AND NOT EXISTS (
            SELECT 1 FROM language_articles la WHERE la.id = a.id
        )
        ORDER BY a.published_at DESC
        LIMIT (p_limit - (SELECT COUNT(*) FROM language_articles))
    )
    SELECT * FROM language_articles
    UNION ALL
    SELECT * FROM fallback_articles
    ORDER BY published_at DESC;
END;
$func$ LANGUAGE plpgsql;

-- Step 7: Create helper function to get language direction
CREATE OR REPLACE FUNCTION get_language_direction(p_language_code VARCHAR(5))
RETURNS VARCHAR(3) AS $func$
DECLARE
    v_direction VARCHAR(3);
BEGIN
    SELECT direction INTO v_direction
    FROM languages
    WHERE code = p_language_code;
    
    IF v_direction IS NULL THEN
        RETURN 'ltr'; -- Default to LTR
    END IF;
    
    RETURN v_direction;
END;
$func$ LANGUAGE plpgsql;

-- Step 8: Create function to get active languages
CREATE OR REPLACE FUNCTION get_active_languages()
RETURNS TABLE (
    code VARCHAR(5),
    name VARCHAR(50),
    native_name VARCHAR(50),
    direction VARCHAR(3),
    sort_order INTEGER
) AS $func$
BEGIN
    RETURN QUERY
    SELECT l.code, l.name, l.native_name, l.direction, l.sort_order
    FROM languages l
    WHERE l.is_active = true
    ORDER BY l.sort_order, l.name;
END;
$func$ LANGUAGE plpgsql;

-- Step 9: Create function to validate language code
CREATE OR REPLACE FUNCTION is_valid_language_code(p_language_code VARCHAR(5))
RETURNS BOOLEAN AS $func$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM languages 
        WHERE code = p_language_code AND is_active = true
    );
END;
$func$ LANGUAGE plpgsql;

-- Step 10: Add comments for documentation
COMMENT ON FUNCTION get_articles_by_language IS 'Get articles by language with English fallback';
COMMENT ON FUNCTION get_language_direction IS 'Get text direction (ltr/rtl) for a language code';
COMMENT ON FUNCTION get_active_languages IS 'Get all active languages ordered by sort_order';
COMMENT ON FUNCTION is_valid_language_code IS 'Check if a language code is valid and active';

-- Verify the migration
DO $$
DECLARE
    lang_count INTEGER;
    fa_count INTEGER;
BEGIN
    -- Check that we have the expected languages
    SELECT COUNT(*) INTO lang_count FROM languages WHERE is_active = true;
    IF lang_count != 5 THEN
        RAISE WARNING 'Expected 5 active languages, found %', lang_count;
    END IF;
    
    -- Check that Persian is removed
    SELECT COUNT(*) INTO fa_count FROM languages WHERE code = 'fa';
    IF fa_count > 0 THEN
        RAISE WARNING 'Persian language should have been removed';
    END IF;
    
    RAISE NOTICE 'Migration 053 completed successfully. Active languages: %', lang_count;
END $$;
