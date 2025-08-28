-- Initialize PostgreSQL with required extensions and optimizations

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create a user for the application (optional, using postgres for simplicity in dev)
-- CREATE USER news_app WITH PASSWORD 'news_app_password';
-- GRANT ALL PRIVILEGES ON DATABASE news_website TO news_app;

-- Set some performance-related settings for the session
-- These will be overridden by postgresql.conf in production
SET shared_preload_libraries = 'pg_stat_statements';
SET track_activity_query_size = 2048;
SET pg_stat_statements.track = 'all';