-- Add language support to existing tables
-- Default language is Persian (fa) as per requirements

-- Add language columns to articles table
ALTER TABLE articles ADD COLUMN language_code VARCHAR(5) DEFAULT 'fa' NOT NULL;
ALTER TABLE articles ADD COLUMN translation_group_id BIGINT;

-- Add language columns to categories table
ALTER TABLE categories ADD COLUMN language_code VARCHAR(5) DEFAULT 'fa' NOT NULL;
ALTER TABLE categories ADD COLUMN translation_group_id BIGINT;

-- Add language columns to tags table
ALTER TABLE tags ADD COLUMN language_code VARCHAR(5) DEFAULT 'fa' NOT NULL;
ALTER TABLE tags ADD COLUMN translation_group_id BIGINT;

-- Create translation groups table for managing article relationships
CREATE TABLE translation_groups (
    id BIGSERIAL PRIMARY KEY,
    group_type VARCHAR(20) NOT NULL CHECK (group_type IN ('article', 'category', 'tag')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create language configuration table
CREATE TABLE languages (
    code VARCHAR(5) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    native_name VARCHAR(50) NOT NULL,
    direction VARCHAR(3) NOT NULL CHECK (direction IN ('ltr', 'rtl')),
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert supported languages (Persian default, English and Arabic optional)
INSERT INTO languages (code, name, native_name, direction, is_active, sort_order) VALUES
('fa', 'Persian', 'فارسی', 'rtl', true, 1),
('en', 'English', 'English', 'ltr', false, 2),
('ar', 'Arabic', 'العربية', 'rtl', false, 3);

-- Add foreign key constraints for translation groups
ALTER TABLE articles ADD CONSTRAINT fk_articles_translation_group 
    FOREIGN KEY (translation_group_id) REFERENCES translation_groups(id) ON DELETE SET NULL;

ALTER TABLE categories ADD CONSTRAINT fk_categories_translation_group 
    FOREIGN KEY (translation_group_id) REFERENCES translation_groups(id) ON DELETE SET NULL;

ALTER TABLE tags ADD CONSTRAINT fk_tags_translation_group 
    FOREIGN KEY (translation_group_id) REFERENCES translation_groups(id) ON DELETE SET NULL;

-- Add foreign key constraints for language codes
ALTER TABLE articles ADD CONSTRAINT fk_articles_language 
    FOREIGN KEY (language_code) REFERENCES languages(code);

ALTER TABLE categories ADD CONSTRAINT fk_categories_language 
    FOREIGN KEY (language_code) REFERENCES languages(code);

ALTER TABLE tags ADD CONSTRAINT fk_tags_language 
    FOREIGN KEY (language_code) REFERENCES languages(code);

-- Create indexes for language-aware queries
CREATE INDEX idx_articles_language ON articles (language_code, published_at DESC) WHERE status = 'published';
CREATE INDEX idx_articles_translation_group ON articles (translation_group_id) WHERE translation_group_id IS NOT NULL;
CREATE INDEX idx_categories_language ON categories (language_code, sort_order);
CREATE INDEX idx_categories_translation_group ON categories (translation_group_id) WHERE translation_group_id IS NOT NULL;
CREATE INDEX idx_tags_language ON tags (language_code);
CREATE INDEX idx_tags_translation_group ON tags (translation_group_id) WHERE translation_group_id IS NOT NULL;

-- Update unique constraints to include language_code
-- Drop existing unique constraints
ALTER TABLE categories DROP CONSTRAINT categories_slug_key;
ALTER TABLE tags DROP CONSTRAINT tags_slug_key;

-- Add new unique constraints with language_code
ALTER TABLE categories ADD CONSTRAINT categories_slug_language_unique UNIQUE (slug, language_code);
ALTER TABLE tags ADD CONSTRAINT tags_slug_language_unique UNIQUE (slug, language_code);

-- For articles, we need to handle the partitioned table differently
-- Create a function to add language-aware unique constraints to article partitions
CREATE OR REPLACE FUNCTION add_article_language_constraints()
RETURNS VOID AS $$
DECLARE
    partition_record RECORD;
BEGIN
    -- Add unique constraint for slug + language_code to each existing partition
    FOR partition_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE tablename LIKE 'articles_%'
        AND schemaname = 'public'
    LOOP
        BEGIN
            EXECUTE format('ALTER TABLE %I.%I ADD CONSTRAINT %I UNIQUE (slug, language_code)',
                partition_record.schemaname, 
                partition_record.tablename,
                partition_record.tablename || '_slug_language_unique');
        EXCEPTION
            WHEN duplicate_table THEN
                -- Constraint already exists, skip
                NULL;
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Apply language constraints to existing partitions
SELECT add_article_language_constraints();

-- Create function to automatically add language constraints to new partitions
CREATE OR REPLACE FUNCTION create_article_partition_with_language_support(
    partition_name TEXT,
    start_date DATE,
    end_date DATE
)
RETURNS VOID AS $$
BEGIN
    -- Create the partition
    EXECUTE format('CREATE TABLE %I PARTITION OF articles FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
    
    -- Add language-aware unique constraint
    EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I UNIQUE (slug, language_code)',
        partition_name, partition_name || '_slug_language_unique');
    
    -- Create standard indexes
    EXECUTE format('CREATE INDEX idx_%I_published_at ON %I (published_at DESC) WHERE status = ''published''',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_category ON %I (category_id, published_at DESC)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_author ON %I (author_id, published_at DESC)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_slug ON %I (slug) WHERE status = ''published''',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_search ON %I USING gin(to_tsvector(''english'', title || '' '' || content))',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_published_brin ON %I USING BRIN (published_at) WITH (pages_per_range = 128)',
        partition_name, partition_name);
    
    -- Add language-specific indexes
    EXECUTE format('CREATE INDEX idx_%I_language ON %I (language_code, published_at DESC) WHERE status = ''published''',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_translation_group ON %I (translation_group_id) WHERE translation_group_id IS NOT NULL',
        partition_name, partition_name);
END;
$$ LANGUAGE plpgsql;

-- Create view for getting articles with their translations
CREATE VIEW article_translations AS
SELECT 
    a.id,
    a.title,
    a.slug,
    a.language_code,
    a.translation_group_id,
    a.published_at,
    a.status,
    tg.group_type,
    l.name as language_name,
    l.native_name as language_native_name,
    l.direction as language_direction,
    -- Get all translations in the same group
    ARRAY_AGG(
        JSON_BUILD_OBJECT(
            'id', ta.id,
            'title', ta.title,
            'slug', ta.slug,
            'language_code', ta.language_code,
            'language_name', tl.name,
            'language_native_name', tl.native_name
        ) ORDER BY tl.sort_order
    ) FILTER (WHERE ta.id != a.id) as translations
FROM articles a
LEFT JOIN translation_groups tg ON a.translation_group_id = tg.id
LEFT JOIN languages l ON a.language_code = l.code
LEFT JOIN articles ta ON a.translation_group_id = ta.translation_group_id
LEFT JOIN languages tl ON ta.language_code = tl.code
WHERE a.status = 'published'
GROUP BY a.id, a.title, a.slug, a.language_code, a.translation_group_id, 
         a.published_at, a.status, tg.group_type, l.name, l.native_name, l.direction;

-- Create view for getting categories with their translations
CREATE VIEW category_translations AS
SELECT 
    c.id,
    c.name,
    c.slug,
    c.language_code,
    c.translation_group_id,
    c.parent_id,
    c.sort_order,
    l.name as language_name,
    l.native_name as language_native_name,
    l.direction as language_direction,
    -- Get all translations in the same group
    ARRAY_AGG(
        JSON_BUILD_OBJECT(
            'id', tc.id,
            'name', tc.name,
            'slug', tc.slug,
            'language_code', tc.language_code,
            'language_name', tl.name,
            'language_native_name', tl.native_name
        ) ORDER BY tl.sort_order
    ) FILTER (WHERE tc.id != c.id) as translations
FROM categories c
LEFT JOIN translation_groups tg ON c.translation_group_id = tg.id
LEFT JOIN languages l ON c.language_code = l.code
LEFT JOIN categories tc ON c.translation_group_id = tc.translation_group_id
LEFT JOIN languages tl ON tc.language_code = tl.code
GROUP BY c.id, c.name, c.slug, c.language_code, c.translation_group_id, 
         c.parent_id, c.sort_order, l.name, l.native_name, l.direction;

-- Create view for getting tags with their translations
CREATE VIEW tag_translations AS
SELECT 
    t.id,
    t.name,
    t.slug,
    t.language_code,
    t.translation_group_id,
    t.keywords,
    t.color,
    l.name as language_name,
    l.native_name as language_native_name,
    l.direction as language_direction,
    -- Get all translations in the same group
    ARRAY_AGG(
        JSON_BUILD_OBJECT(
            'id', tt.id,
            'name', tt.name,
            'slug', tt.slug,
            'language_code', tt.language_code,
            'language_name', tl.name,
            'language_native_name', tl.native_name
        ) ORDER BY tl.sort_order
    ) FILTER (WHERE tt.id != t.id) as translations
FROM tags t
LEFT JOIN translation_groups tg ON t.translation_group_id = tg.id
LEFT JOIN languages l ON t.language_code = l.code
LEFT JOIN tags tt ON t.translation_group_id = tt.translation_group_id
LEFT JOIN languages tl ON tt.language_code = tl.code
GROUP BY t.id, t.name, t.slug, t.language_code, t.translation_group_id, 
         t.keywords, t.color, l.name, l.native_name, l.direction;

-- Create function to create translation group and link content
CREATE OR REPLACE FUNCTION create_translation_group(
    p_group_type VARCHAR(20),
    p_content_ids BIGINT[]
)
RETURNS BIGINT AS $$
DECLARE
    v_group_id BIGINT;
    v_content_id BIGINT;
BEGIN
    -- Create new translation group
    INSERT INTO translation_groups (group_type)
    VALUES (p_group_type)
    RETURNING id INTO v_group_id;
    
    -- Link content to the group
    FOREACH v_content_id IN ARRAY p_content_ids
    LOOP
        CASE p_group_type
            WHEN 'article' THEN
                UPDATE articles SET translation_group_id = v_group_id WHERE id = v_content_id;
            WHEN 'category' THEN
                UPDATE categories SET translation_group_id = v_group_id WHERE id = v_content_id;
            WHEN 'tag' THEN
                UPDATE tags SET translation_group_id = v_group_id WHERE id = v_content_id;
        END CASE;
    END LOOP;
    
    RETURN v_group_id;
END;
$$ LANGUAGE plpgsql;

-- Create function to get content by language with fallback
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
) AS $$
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
$$ LANGUAGE plpgsql;

-- Create trigger function to update translation group updated_at
CREATE OR REPLACE FUNCTION update_translation_group_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for translation_groups
CREATE TRIGGER trigger_update_translation_group_updated_at
    BEFORE UPDATE ON translation_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_translation_group_updated_at();

-- Add comments for documentation
COMMENT ON TABLE translation_groups IS 'Groups for managing translations of articles, categories, and tags';
COMMENT ON TABLE languages IS 'Supported languages with RTL/LTR direction information';

COMMENT ON COLUMN articles.language_code IS 'Language code (fa=Persian, en=English, ar=Arabic)';
COMMENT ON COLUMN articles.translation_group_id IS 'Links articles that are translations of each other';
COMMENT ON COLUMN categories.language_code IS 'Language code for category names and descriptions';
COMMENT ON COLUMN categories.translation_group_id IS 'Links categories that are translations of each other';
COMMENT ON COLUMN tags.language_code IS 'Language code for tag names and keywords';
COMMENT ON COLUMN tags.translation_group_id IS 'Links tags that are translations of each other';

COMMENT ON VIEW article_translations IS 'Articles with their available translations';
COMMENT ON VIEW category_translations IS 'Categories with their available translations';
COMMENT ON VIEW tag_translations IS 'Tags with their available translations';