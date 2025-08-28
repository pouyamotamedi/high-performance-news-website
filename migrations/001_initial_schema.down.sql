-- Drop tables in reverse order to handle foreign key constraints

-- Drop partitioned tables (this will also drop all partitions)
DROP TABLE IF EXISTS article_engagement CASCADE;
DROP TABLE IF EXISTS article_views CASCADE;
DROP TABLE IF EXISTS article_tags CASCADE;
DROP TABLE IF EXISTS articles CASCADE;

-- Drop regular tables
DROP TABLE IF EXISTS tags CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop extensions (only if no other objects depend on them)
-- Note: We don't drop extensions as they might be used by other databases
-- DROP EXTENSION IF EXISTS "pg_trgm";
-- DROP EXTENSION IF EXISTS "uuid-ossp";