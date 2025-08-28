-- Create monitoring tables for performance monitoring system

-- Health checks table
CREATE TABLE IF NOT EXISTS health_checks (
    id BIGSERIAL PRIMARY KEY,
    component VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('healthy', 'degraded', 'unhealthy')),
    message TEXT,
    response_time_ms INTEGER,
    metadata JSONB,
    checked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- System metrics table (partitioned by created_at)
CREATE TABLE IF NOT EXISTS system_metrics (
    id BIGSERIAL,
    cpu_usage DECIMAL(5,2) NOT NULL,
    memory_usage DECIMAL(5,2) NOT NULL,
    memory_total BIGINT NOT NULL,
    memory_used BIGINT NOT NULL,
    disk_usage DECIMAL(5,2) NOT NULL,
    disk_total BIGINT NOT NULL,
    disk_used BIGINT NOT NULL,
    network_bytes_in BIGINT DEFAULT 0,
    network_bytes_out BIGINT DEFAULT 0,
    load_average_1 DECIMAL(10,2) DEFAULT 0,
    load_average_5 DECIMAL(10,2) DEFAULT 0,
    load_average_15 DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Database metrics table (partitioned by created_at)
CREATE TABLE IF NOT EXISTS database_metrics (
    id BIGSERIAL,
    active_connections INTEGER NOT NULL,
    idle_connections INTEGER NOT NULL,
    max_connections INTEGER NOT NULL,
    slow_queries BIGINT DEFAULT 0,
    average_query_time DECIMAL(10,3) DEFAULT 0,
    queries_per_second DECIMAL(10,2) DEFAULT 0,
    cache_hit_ratio DECIMAL(5,4) DEFAULT 0,
    deadlock_count BIGINT DEFAULT 0,
    temp_files_created BIGINT DEFAULT 0,
    checkpoint_write_time DECIMAL(10,3) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Cache metrics table (partitioned by created_at)
CREATE TABLE IF NOT EXISTS cache_metrics (
    id BIGSERIAL,
    hit_count BIGINT NOT NULL DEFAULT 0,
    miss_count BIGINT NOT NULL DEFAULT 0,
    hit_rate DECIMAL(5,4) NOT NULL DEFAULT 0,
    key_count BIGINT NOT NULL DEFAULT 0,
    memory_usage BIGINT NOT NULL DEFAULT 0,
    memory_total BIGINT NOT NULL DEFAULT 0,
    evicted_keys BIGINT DEFAULT 0,
    expired_keys BIGINT DEFAULT 0,
    operations_per_sec DECIMAL(10,2) DEFAULT 0,
    average_latency DECIMAL(10,3) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Publishing metrics table (partitioned by created_at)
CREATE TABLE IF NOT EXISTS publishing_metrics (
    id BIGSERIAL,
    articles_published BIGINT NOT NULL DEFAULT 0,
    publishing_rate DECIMAL(10,2) NOT NULL DEFAULT 0,
    average_publish_time DECIMAL(10,3) DEFAULT 0,
    failed_publications BIGINT DEFAULT 0,
    queued_articles BIGINT DEFAULT 0,
    processing_articles BIGINT DEFAULT 0,
    static_pages_generated BIGINT DEFAULT 0,
    cache_invalidations BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'resolved', 'suppressed')),
    component VARCHAR(100) NOT NULL,
    metric VARCHAR(100) NOT NULL,
    threshold DECIMAL(15,6) NOT NULL,
    current_value DECIMAL(15,6) NOT NULL,
    metadata JSONB,
    triggered_at TIMESTAMP WITH TIME ZONE NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Alert rules table
CREATE TABLE IF NOT EXISTS alert_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    component VARCHAR(100) NOT NULL,
    metric VARCHAR(100) NOT NULL,
    operator VARCHAR(10) NOT NULL CHECK (operator IN ('>', '<', '>=', '<=', '==', '!=')),
    threshold DECIMAL(15,6) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    enabled BOOLEAN NOT NULL DEFAULT true,
    cooldown_minutes INTEGER NOT NULL DEFAULT 15,
    conditions JSONB,
    actions JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- User sessions table for active user tracking
CREATE TABLE IF NOT EXISTS user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) NOT NULL UNIQUE,
    ip_address INET,
    user_agent TEXT,
    last_activity TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_health_checks_component_checked_at ON health_checks (component, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_health_checks_status ON health_checks (status);
CREATE INDEX IF NOT EXISTS idx_health_checks_checked_at ON health_checks (checked_at DESC);

CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts (status);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts (severity);
CREATE INDEX IF NOT EXISTS idx_alerts_component ON alerts (component);
CREATE INDEX IF NOT EXISTS idx_alerts_triggered_at ON alerts (triggered_at DESC);

CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules (enabled);
CREATE INDEX IF NOT EXISTS idx_alert_rules_component ON alert_rules (component);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_last_activity ON user_sessions (last_activity DESC);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions (expires_at);

-- Create daily partitions for metrics tables (current month)
DO $$
DECLARE
    start_date DATE := DATE_TRUNC('month', CURRENT_DATE);
    end_date DATE := start_date + INTERVAL '1 month';
    partition_date DATE := start_date;
    table_name TEXT;
    partition_name TEXT;
BEGIN
    -- Create partitions for each metrics table
    FOREACH table_name IN ARRAY ARRAY['system_metrics', 'database_metrics', 'cache_metrics', 'publishing_metrics']
    LOOP
        partition_date := start_date;
        WHILE partition_date < end_date LOOP
            partition_name := table_name || '_' || TO_CHAR(partition_date, 'YYYY_MM_DD');
            
            EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I 
                           FOR VALUES FROM (%L) TO (%L)',
                          partition_name, table_name, 
                          partition_date, partition_date + INTERVAL '1 day');
            
            -- Create indexes on partitions
            EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON %I (created_at DESC)',
                          'idx_' || partition_name || '_created_at', partition_name);
            
            partition_date := partition_date + INTERVAL '1 day';
        END LOOP;
    END LOOP;
END $$;

-- Create BRIN indexes for time-series data performance
CREATE INDEX IF NOT EXISTS idx_system_metrics_created_at_brin ON system_metrics USING BRIN (created_at) WITH (pages_per_range = 128);
CREATE INDEX IF NOT EXISTS idx_database_metrics_created_at_brin ON database_metrics USING BRIN (created_at) WITH (pages_per_range = 128);
CREATE INDEX IF NOT EXISTS idx_cache_metrics_created_at_brin ON cache_metrics USING BRIN (created_at) WITH (pages_per_range = 128);
CREATE INDEX IF NOT EXISTS idx_publishing_metrics_created_at_brin ON publishing_metrics USING BRIN (created_at) WITH (pages_per_range = 128);

-- Create function to automatically create daily partitions
CREATE OR REPLACE FUNCTION create_daily_monitoring_partitions()
RETURNS void AS $$
DECLARE
    table_name TEXT;
    partition_date DATE;
    partition_name TEXT;
BEGIN
    -- Create partitions for next 7 days
    FOREACH table_name IN ARRAY ARRAY['system_metrics', 'database_metrics', 'cache_metrics', 'publishing_metrics']
    LOOP
        FOR i IN 0..6 LOOP
            partition_date := CURRENT_DATE + i;
            partition_name := table_name || '_' || TO_CHAR(partition_date, 'YYYY_MM_DD');
            
            -- Check if partition already exists
            IF NOT EXISTS (
                SELECT 1 FROM pg_class c 
                JOIN pg_namespace n ON n.oid = c.relnamespace 
                WHERE c.relname = partition_name AND n.nspname = 'public'
            ) THEN
                EXECUTE format('CREATE TABLE %I PARTITION OF %I 
                               FOR VALUES FROM (%L) TO (%L)',
                              partition_name, table_name, 
                              partition_date, partition_date + INTERVAL '1 day');
                
                -- Create index on new partition
                EXECUTE format('CREATE INDEX %I ON %I (created_at DESC)',
                              'idx_' || partition_name || '_created_at', partition_name);
            END IF;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create function to drop old monitoring partitions
CREATE OR REPLACE FUNCTION drop_old_monitoring_partitions()
RETURNS void AS $$
DECLARE
    table_name TEXT;
    partition_date DATE;
    partition_name TEXT;
    retention_days INTEGER := 30; -- Keep 30 days of data
BEGIN
    FOREACH table_name IN ARRAY ARRAY['system_metrics', 'database_metrics', 'cache_metrics', 'publishing_metrics']
    LOOP
        partition_date := CURRENT_DATE - retention_days;
        partition_name := table_name || '_' || TO_CHAR(partition_date, 'YYYY_MM_DD');
        
        -- Check if partition exists and drop it
        IF EXISTS (
            SELECT 1 FROM pg_class c 
            JOIN pg_namespace n ON n.oid = c.relnamespace 
            WHERE c.relname = partition_name AND n.nspname = 'public'
        ) THEN
            EXECUTE format('DROP TABLE IF EXISTS %I', partition_name);
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create monitoring configuration table
CREATE TABLE IF NOT EXISTS monitoring_config (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Insert default monitoring configuration
INSERT INTO monitoring_config (config_key, config_value, description) VALUES
('thresholds', '{
    "cpu_warning": 70.0,
    "cpu_critical": 85.0,
    "memory_warning": 80.0,
    "memory_critical": 90.0,
    "disk_warning": 85.0,
    "disk_critical": 95.0,
    "db_connections_warning": 120,
    "db_connections_critical": 140,
    "cache_hit_rate_warning": 0.7,
    "cache_hit_rate_critical": 0.5,
    "publishing_rate_warning": 25.0,
    "publishing_rate_critical": 15.0,
    "response_time_warning": 1000.0,
    "response_time_critical": 2000.0
}', 'Performance monitoring thresholds'),
('retention', '{
    "metrics_days": 30,
    "alerts_days": 90,
    "health_checks_days": 7
}', 'Data retention settings'),
('alerting', '{
    "enabled": true,
    "cooldown_minutes": 15,
    "email_enabled": false,
    "slack_enabled": false,
    "webhook_enabled": false
}', 'Alerting configuration')
ON CONFLICT (config_key) DO NOTHING;

-- Create view for latest metrics
CREATE OR REPLACE VIEW latest_system_metrics AS
SELECT DISTINCT ON (DATE_TRUNC('hour', created_at))
    *
FROM system_metrics
WHERE created_at >= NOW() - INTERVAL '24 hours'
ORDER BY DATE_TRUNC('hour', created_at) DESC, created_at DESC;

CREATE OR REPLACE VIEW latest_database_metrics AS
SELECT DISTINCT ON (DATE_TRUNC('hour', created_at))
    *
FROM database_metrics
WHERE created_at >= NOW() - INTERVAL '24 hours'
ORDER BY DATE_TRUNC('hour', created_at) DESC, created_at DESC;

CREATE OR REPLACE VIEW latest_cache_metrics AS
SELECT DISTINCT ON (DATE_TRUNC('hour', created_at))
    *
FROM cache_metrics
WHERE created_at >= NOW() - INTERVAL '24 hours'
ORDER BY DATE_TRUNC('hour', created_at) DESC, created_at DESC;

CREATE OR REPLACE VIEW latest_publishing_metrics AS
SELECT DISTINCT ON (DATE_TRUNC('hour', created_at))
    *
FROM publishing_metrics
WHERE created_at >= NOW() - INTERVAL '24 hours'
ORDER BY DATE_TRUNC('hour', created_at) DESC, created_at DESC;