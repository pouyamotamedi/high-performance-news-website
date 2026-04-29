-- Comprehensive monitoring system tables
-- This migration creates all necessary tables for the monitoring and alerting system

-- Log entries table for log aggregation (partitioned by timestamp)
CREATE TABLE IF NOT EXISTS log_entries (
    id BIGSERIAL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    level VARCHAR(20) NOT NULL,
    component VARCHAR(100) NOT NULL,
    message TEXT NOT NULL,
    context JSONB DEFAULT '{}',
    source VARCHAR(500),
    trace_id VARCHAR(100),
    user_id BIGINT,
    request_id VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Partition log_entries by timestamp for better performance
CREATE TABLE IF NOT EXISTS log_entries_2024_01 PARTITION OF log_entries
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE IF NOT EXISTS log_entries_2024_02 PARTITION OF log_entries
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

CREATE TABLE IF NOT EXISTS log_entries_2024_03 PARTITION OF log_entries
FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');

-- Add more partitions as needed...

-- Indexes for log_entries
CREATE INDEX IF NOT EXISTS idx_log_entries_timestamp ON log_entries (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_log_entries_level ON log_entries (level);
CREATE INDEX IF NOT EXISTS idx_log_entries_component ON log_entries (component);
CREATE INDEX IF NOT EXISTS idx_log_entries_trace_id ON log_entries (trace_id) WHERE trace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_log_entries_user_id ON log_entries (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_log_entries_request_id ON log_entries (request_id) WHERE request_id IS NOT NULL;

-- Full-text search index for log messages
CREATE INDEX IF NOT EXISTS idx_log_entries_message_fts ON log_entries USING gin(to_tsvector('english', message));

-- Alert rules table
CREATE TABLE IF NOT EXISTS alert_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    component VARCHAR(100) NOT NULL,
    metric VARCHAR(100) NOT NULL,
    operator VARCHAR(10) NOT NULL CHECK (operator IN ('>', '<', '>=', '<=', '==', '!=')),
    threshold DECIMAL(10,2) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    enabled BOOLEAN DEFAULT true,
    cooldown_minutes INTEGER DEFAULT 15,
    conditions JSONB DEFAULT '{}',
    actions JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for alert_rules
CREATE INDEX IF NOT EXISTS idx_alert_rules_component ON alert_rules (component);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules (enabled);

-- Remediation actions table
CREATE TABLE IF NOT EXISTS remediation_actions (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    alert_name VARCHAR(255) NOT NULL,
    component VARCHAR(100) NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    parameters JSONB DEFAULT '{}',
    enabled BOOLEAN DEFAULT true,
    max_retries INTEGER DEFAULT 3,
    cooldown_minutes INTEGER DEFAULT 15,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for remediation_actions
CREATE INDEX IF NOT EXISTS idx_remediation_actions_alert_name ON remediation_actions (alert_name);
CREATE INDEX IF NOT EXISTS idx_remediation_actions_component ON remediation_actions (component);
CREATE INDEX IF NOT EXISTS idx_remediation_actions_enabled ON remediation_actions (enabled);

-- Remediation executions table
CREATE TABLE IF NOT EXISTS remediation_executions (
    id BIGSERIAL PRIMARY KEY,
    action_id VARCHAR(255) NOT NULL,
    alert_name VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped')),
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    output TEXT,
    error TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for remediation_executions
CREATE INDEX IF NOT EXISTS idx_remediation_executions_action_id ON remediation_executions (action_id);
CREATE INDEX IF NOT EXISTS idx_remediation_executions_alert_name ON remediation_executions (alert_name);
CREATE INDEX IF NOT EXISTS idx_remediation_executions_status ON remediation_executions (status);
CREATE INDEX IF NOT EXISTS idx_remediation_executions_created_at ON remediation_executions (created_at DESC);

-- Operational runbooks table
CREATE TABLE IF NOT EXISTS operational_runbooks (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    severity VARCHAR(20) CHECK (severity IN ('info', 'warning', 'critical')),
    alert_names JSONB DEFAULT '[]',
    components JSONB DEFAULT '[]',
    steps JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for operational_runbooks
CREATE INDEX IF NOT EXISTS idx_operational_runbooks_category ON operational_runbooks (category);
CREATE INDEX IF NOT EXISTS idx_operational_runbooks_severity ON operational_runbooks (severity);
CREATE INDEX IF NOT EXISTS idx_operational_runbooks_enabled ON operational_runbooks (enabled);

-- Runbook executions table
CREATE TABLE IF NOT EXISTS runbook_executions (
    id BIGSERIAL PRIMARY KEY,
    runbook_id VARCHAR(255) NOT NULL,
    alert_name VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'paused')),
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    step_results JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for runbook_executions
CREATE INDEX IF NOT EXISTS idx_runbook_executions_runbook_id ON runbook_executions (runbook_id);
CREATE INDEX IF NOT EXISTS idx_runbook_executions_alert_name ON runbook_executions (alert_name);
CREATE INDEX IF NOT EXISTS idx_runbook_executions_status ON runbook_executions (status);
CREATE INDEX IF NOT EXISTS idx_runbook_executions_created_at ON runbook_executions (created_at DESC);

-- System metrics history table (partitioned by date)
CREATE TABLE IF NOT EXISTS system_metrics_history (
    id BIGSERIAL,
    cpu_usage DECIMAL(5,2),
    memory_usage DECIMAL(5,2),
    memory_total BIGINT,
    memory_used BIGINT,
    disk_usage DECIMAL(5,2),
    disk_total BIGINT,
    disk_used BIGINT,
    network_bytes_in BIGINT,
    network_bytes_out BIGINT,
    load_average_1 DECIMAL(5,2),
    load_average_5 DECIMAL(5,2),
    load_average_15 DECIMAL(5,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions for system_metrics_history
CREATE TABLE IF NOT EXISTS system_metrics_history_2024_01 PARTITION OF system_metrics_history
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE IF NOT EXISTS system_metrics_history_2024_02 PARTITION OF system_metrics_history
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Database metrics history table (partitioned by date)
CREATE TABLE IF NOT EXISTS database_metrics_history (
    id BIGSERIAL,
    active_connections INTEGER,
    idle_connections INTEGER,
    max_connections INTEGER,
    slow_queries BIGINT,
    average_query_time DECIMAL(10,3),
    queries_per_second DECIMAL(10,2),
    cache_hit_ratio DECIMAL(5,4),
    deadlock_count BIGINT,
    temp_files_created BIGINT,
    checkpoint_write_time DECIMAL(10,3),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions for database_metrics_history
CREATE TABLE IF NOT EXISTS database_metrics_history_2024_01 PARTITION OF database_metrics_history
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE IF NOT EXISTS database_metrics_history_2024_02 PARTITION OF database_metrics_history
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Cache metrics history table (partitioned by date)
CREATE TABLE IF NOT EXISTS cache_metrics_history (
    id BIGSERIAL,
    hit_count BIGINT,
    miss_count BIGINT,
    hit_rate DECIMAL(5,4),
    key_count BIGINT,
    memory_usage BIGINT,
    memory_total BIGINT,
    evicted_keys BIGINT,
    expired_keys BIGINT,
    operations_per_sec DECIMAL(10,2),
    average_latency DECIMAL(10,3),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions for cache_metrics_history
CREATE TABLE IF NOT EXISTS cache_metrics_history_2024_01 PARTITION OF cache_metrics_history
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE IF NOT EXISTS cache_metrics_history_2024_02 PARTITION OF cache_metrics_history
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Publishing metrics history table (partitioned by date)
CREATE TABLE IF NOT EXISTS publishing_metrics_history (
    id BIGSERIAL,
    articles_published BIGINT,
    publishing_rate DECIMAL(10,2),
    average_publish_time DECIMAL(10,3),
    failed_publications BIGINT,
    queued_articles BIGINT,
    processing_articles BIGINT,
    static_pages_generated BIGINT,
    cache_invalidations BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions for publishing_metrics_history
CREATE TABLE IF NOT EXISTS publishing_metrics_history_2024_01 PARTITION OF publishing_metrics_history
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE IF NOT EXISTS publishing_metrics_history_2024_02 PARTITION OF publishing_metrics_history
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Alert history table for tracking all alerts
CREATE TABLE IF NOT EXISTS alert_history (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('active', 'resolved', 'suppressed')),
    component VARCHAR(100) NOT NULL,
    metric VARCHAR(100) NOT NULL,
    threshold DECIMAL(10,2),
    current_value DECIMAL(10,2),
    metadata JSONB DEFAULT '{}',
    triggered_at TIMESTAMP WITH TIME ZONE NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for alert_history
CREATE INDEX IF NOT EXISTS idx_alert_history_name ON alert_history (name);
CREATE INDEX IF NOT EXISTS idx_alert_history_severity ON alert_history (severity);
CREATE INDEX IF NOT EXISTS idx_alert_history_status ON alert_history (status);
CREATE INDEX IF NOT EXISTS idx_alert_history_component ON alert_history (component);
CREATE INDEX IF NOT EXISTS idx_alert_history_triggered_at ON alert_history (triggered_at DESC);

-- Monitoring configuration table
CREATE TABLE IF NOT EXISTS monitoring_configuration (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(255) NOT NULL UNIQUE,
    config_value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default monitoring configuration
INSERT INTO monitoring_configuration (config_key, config_value, description) VALUES
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
}', 'Default metric thresholds for alerting')
ON CONFLICT (config_key) DO NOTHING;

-- Create functions for automatic partition management
CREATE OR REPLACE FUNCTION create_daily_monitoring_partitions()
RETURNS void AS $$
DECLARE
    start_date date;
    end_date date;
    table_name text;
    partition_name text;
BEGIN
    -- Create partitions for the next 7 days
    FOR i IN 0..6 LOOP
        start_date := CURRENT_DATE + i;
        end_date := start_date + 1;
        
        -- Create partitions for each metrics table
        FOREACH table_name IN ARRAY ARRAY['log_entries', 'system_metrics_history', 'database_metrics_history', 'cache_metrics_history', 'publishing_metrics_history'] LOOP
            partition_name := table_name || '_' || to_char(start_date, 'YYYY_MM_DD');
            
            -- Check if partition already exists
            IF NOT EXISTS (
                SELECT 1 FROM pg_class c
                JOIN pg_namespace n ON n.oid = c.relnamespace
                WHERE c.relname = partition_name AND n.nspname = 'public'
            ) THEN
                EXECUTE format('CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                    partition_name, table_name, start_date, end_date);
                
                RAISE NOTICE 'Created partition: %', partition_name;
            END IF;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create function to drop old partitions
CREATE OR REPLACE FUNCTION drop_old_monitoring_partitions()
RETURNS void AS $$
DECLARE
    partition_record record;
    cutoff_date date;
BEGIN
    -- Drop partitions older than 30 days
    cutoff_date := CURRENT_DATE - 30;
    
    FOR partition_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE schemaname = 'public'
        AND (
            tablename LIKE 'log_entries_%'
            OR tablename LIKE 'system_metrics_history_%'
            OR tablename LIKE 'database_metrics_history_%'
            OR tablename LIKE 'cache_metrics_history_%'
            OR tablename LIKE 'publishing_metrics_history_%'
        )
        AND tablename ~ '\d{4}_\d{2}_\d{2}$'
        AND to_date(right(tablename, 10), 'YYYY_MM_DD') < cutoff_date
    LOOP
        EXECUTE format('DROP TABLE IF EXISTS %I.%I', partition_record.schemaname, partition_record.tablename);
        RAISE NOTICE 'Dropped old partition: %', partition_record.tablename;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create initial partitions
SELECT create_daily_monitoring_partitions();

-- Create triggers for automatic updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers to relevant tables
CREATE TRIGGER update_alert_rules_updated_at
    BEFORE UPDATE ON alert_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_remediation_actions_updated_at
    BEFORE UPDATE ON remediation_actions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_operational_runbooks_updated_at
    BEFORE UPDATE ON operational_runbooks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alert_history_updated_at
    BEFORE UPDATE ON alert_history
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_monitoring_configuration_updated_at
    BEFORE UPDATE ON monitoring_configuration
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create views for easier querying
CREATE OR REPLACE VIEW active_alerts AS
SELECT * FROM alert_history
WHERE status = 'active'
ORDER BY triggered_at DESC;

CREATE OR REPLACE VIEW recent_remediation_executions AS
SELECT * FROM remediation_executions
WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'
ORDER BY created_at DESC;

CREATE OR REPLACE VIEW recent_runbook_executions AS
SELECT * FROM runbook_executions
WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'
ORDER BY created_at DESC;

CREATE OR REPLACE VIEW system_health_summary AS
SELECT
    'system' as component,
    CASE
        WHEN cpu_usage >= 85 OR memory_usage >= 90 OR disk_usage >= 95 THEN 'unhealthy'
        WHEN cpu_usage >= 70 OR memory_usage >= 80 OR disk_usage >= 85 THEN 'degraded'
        ELSE 'healthy'
    END as status,
    cpu_usage,
    memory_usage,
    disk_usage,
    created_at
FROM system_metrics_history
WHERE created_at >= NOW() - INTERVAL '5 minutes'
ORDER BY created_at DESC
LIMIT 1;

-- Grant necessary permissions (adjust as needed for your user setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO your_app_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO your_app_user;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO your_app_user;