-- Migration 053 Down: Revert language changes
-- Restore Persian (fa), remove German (de), French (fr), Spanish (es)

-- Step 1: Re-add Persian language
INSERT INTO languages (code, name, native_name, direction, is_active, sort_order)
VALUES ('fa', 'Persian', 'فارسی', 'rtl', true, 1)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    native_name = EXCLUDED.native_name,
    direction = EXCLUDED.direction,
    is_active = EXCLUDED.is_active,
    sort_order = EXCLUDED.sort_order;

-- Step 2: Update content back to Persian
UPDATE articles SET language_code = 'fa' WHERE language_code = 'en';
UPDATE categories SET language_code = 'fa' WHERE language_code = 'en';
UPDATE tags SET language_code = 'fa' WHERE language_code = 'en';

-- Step 3: Remove new languages
DELETE FROM languages WHERE code IN ('de', 'fr', 'es');

-- Step 4: Deactivate English and Arabic
UPDATE languages SET is_active = false WHERE code = 'en';
UPDATE languages SET is_active = false WHERE code = 'ar';

-- Step 5: Restore original function with Persian default
CREATE OR REPLACE FUNCTION get_articles_by_language(
    p_language_code VARCHAR(5),
    p_fallback_language VARCHAR(5) DEFAULT 'fa',
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
        SELECT a.id, a.title, a.slug, a.excerpt, a.language_code, a.published_at, false as is_fallback
        FROM articles a
        WHERE a.language_code = p_language_code 
        AND a.status = 'published'
        ORDER BY a.published_at DESC
        LIMIT p_limit OFFSET p_offset
    ),
    fallback_articles AS (
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

-- Step 6: Drop helper functions
DROP FUNCTION IF EXISTS get_language_direction(VARCHAR);
DROP FUNCTION IF EXISTS get_active_languages();
DROP FUNCTION IF EXISTS is_valid_language_code(VARCHAR);
