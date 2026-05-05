-- Rollback: Fix image_variants schema
-- This removes the new columns added for responsive image support

-- Remove new indexes
DROP INDEX IF EXISTS idx_image_variants_size_format;
DROP INDEX IF EXISTS idx_images_hash;

-- Remove new unique constraint
ALTER TABLE image_variants DROP CONSTRAINT IF EXISTS image_variants_image_id_size_format_key;

-- Remove new columns from image_variants
ALTER TABLE image_variants DROP COLUMN IF EXISTS size;
ALTER TABLE image_variants DROP COLUMN IF EXISTS format;
ALTER TABLE image_variants DROP COLUMN IF EXISTS url;
ALTER TABLE image_variants DROP COLUMN IF EXISTS quality;

-- Remove hash column from images
ALTER TABLE images DROP COLUMN IF EXISTS hash;

-- Restore NOT NULL constraints on old columns
ALTER TABLE image_variants ALTER COLUMN variant_name SET NOT NULL;
ALTER TABLE image_variants ALTER COLUMN filename SET NOT NULL;
