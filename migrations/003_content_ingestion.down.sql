-- Drop triggers
DROP TRIGGER IF EXISTS update_content_sources_updated_at ON content_sources;
DROP TRIGGER IF EXISTS update_ingested_content_updated_at ON ingested_content;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS create_monthly_ingested_content_partition(DATE);
DROP FUNCTION IF EXISTS cleanup_old_ingested_content_partitions(INTEGER);

-- Drop tables (this will also drop all partitions)
DROP TABLE IF EXISTS content_ingestion_rate_limits CASCADE;
DROP TABLE IF EXISTS ingested_content CASCADE;
DROP TABLE IF EXISTS content_sources CASCADE;

-- Note: pg_trgm extension is not dropped as it might be used by other parts of the system