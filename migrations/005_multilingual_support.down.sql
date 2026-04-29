-- Drop multilingual support (reverse of 005_multilingual_support.up.sql)

-- Drop views
DROP VIEW IF EXISTS article_translations;
DROP VIEW IF EXISTS category_translations;
DROP VIEW IF EXISTS tag_translations;

-- Drop functions
DROP FUNCTION IF EXISTS create_translation_group(VARCHAR(20), BIGINT[]);
DROP FUNCTION IF EXISTS get_articles_by_language(VARCHAR(5), VARCHAR(5), INTEGER, INTEGER);
DROP FUNCTION IF EXISTS update_translation_group_updated_at();
DROP FUNCTION IF EXISTS add_article_language_constraints();
DROP FUNCTION IF EXISTS create_article_partition_with_language_support(TEXT, DATE, DATE);

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_update_translation_group_updated_at ON translation_groups;

-- Remove language-aware unique constraints from partitions
DO $
DECLARE
    partition_record RECORD;
BEGIN
    FOR partition_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE tablename LIKE 'articles_%'
        AND schemaname = 'public'
    LOOP
        BEGIN
            EXECUTE format('ALTER TABLE %I.%I DROP CONSTRAINT IF EXISTS %I',
                partition_record.schemaname, 
                partition_record.tablename,
                partition_record.tablename || '_slug_language_unique');
        EXCEPTION
            WHEN OTHERS THEN
                -- Ignore errors if constraint doesn't exist
                NULL;
        END;
    END LOOP;
END;
$;

-- Drop foreign key constraints
ALTER TABLE articles DROP CONSTRAINT IF EXISTS fk_articles_translation_group;
ALTER TABLE articles DROP CONSTRAINT IF EXISTS fk_articles_language;
ALTER TABLE categories DROP CONSTRAINT IF EXISTS fk_categories_translation_group;
ALTER TABLE categories DROP CONSTRAINT IF EXISTS fk_categories_language;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS fk_tags_translation_group;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS fk_tags_language;

-- Drop indexes
DROP INDEX IF EXISTS idx_articles_language;
DROP INDEX IF EXISTS idx_articles_translation_group;
DROP INDEX IF EXISTS idx_categories_language;
DROP INDEX IF EXISTS idx_categories_translation_group;
DROP INDEX IF EXISTS idx_tags_language;
DROP INDEX IF EXISTS idx_tags_translation_group;

-- Restore original unique constraints
ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_slug_language_unique;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS tags_slug_language_unique;

ALTER TABLE categories ADD CONSTRAINT categories_slug_key UNIQUE (slug);
ALTER TABLE tags ADD CONSTRAINT tags_slug_key UNIQUE (slug);

-- Remove language columns
ALTER TABLE articles DROP COLUMN IF EXISTS language_code;
ALTER TABLE articles DROP COLUMN IF EXISTS translation_group_id;
ALTER TABLE categories DROP COLUMN IF EXISTS language_code;
ALTER TABLE categories DROP COLUMN IF EXISTS translation_group_id;
ALTER TABLE tags DROP COLUMN IF EXISTS language_code;
ALTER TABLE tags DROP COLUMN IF EXISTS translation_group_id;

-- Drop tables
DROP TABLE IF EXISTS translation_groups;
DROP TABLE IF EXISTS languages;