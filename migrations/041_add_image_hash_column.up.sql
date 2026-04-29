-- Add hash column to images table for deduplication
-- This allows detecting duplicate uploads by comparing SHA256 hashes

ALTER TABLE images ADD COLUMN IF NOT EXISTS hash VARCHAR(64);

-- Create index for fast hash lookups
CREATE INDEX IF NOT EXISTS idx_images_hash ON images(hash);

-- Add comment explaining the column
COMMENT ON COLUMN images.hash IS 'SHA256 hash of image content for deduplication';
