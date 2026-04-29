-- Rollback migration: Remove tag and category validation enhancements

-- Drop the trigger and function
DROP TRIGGER IF EXISTS tr_validate_tag_keywords ON tags;
DROP FUNCTION IF EXISTS trigger_validate_tag_keywords();
DROP FUNCTION IF EXISTS validate_tag_keyword_uniqueness(JSONB, BIGINT);

-- Drop unique indexes
DROP INDEX IF EXISTS idx_categories_name_unique_ci;
DROP INDEX IF EXISTS idx_tags_name_unique_ci;

-- Note: We don't remove image_url and image_alt_text columns as they might be used elsewhere
-- and removing columns can be destructive. If needed, they can be removed in a separate migration.