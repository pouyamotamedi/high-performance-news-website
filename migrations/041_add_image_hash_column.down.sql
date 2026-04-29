-- Remove hash column from images table
DROP INDEX IF EXISTS idx_images_hash;
ALTER TABLE images DROP COLUMN IF EXISTS hash;
