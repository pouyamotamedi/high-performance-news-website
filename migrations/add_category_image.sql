-- Add image field to categories table
ALTER TABLE categories ADD COLUMN image_url VARCHAR(500);
ALTER TABLE categories ADD COLUMN image_alt_text VARCHAR(200);

-- Add comment for the new columns
COMMENT ON COLUMN categories.image_url IS 'URL path to category image';
COMMENT ON COLUMN categories.image_alt_text IS 'Alt text for category image accessibility';

-- Create index for better performance when filtering by image
CREATE INDEX idx_categories_image_url ON categories(image_url) WHERE image_url IS NOT NULL;