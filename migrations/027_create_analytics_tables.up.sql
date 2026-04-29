-- Create user_behavior table (partitioned by created_at)
CREATE TABLE user_behavior (
    id BIGSERIAL,
    session_id VARCHAR(255) NOT NULL,
    user_id BIGINT REFERENCES users(id),
    ip_address INET NOT NULL,
    user_agent TEXT,
    page_url TEXT NOT NULL,
    referer TEXT,
    time_on_page INTEGER DEFAULT 0,
    scroll_depth FLOAT DEFAULT 0,
    behavior_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create performance_metrics table (partitioned by created_at)
CREATE TABLE performance_metrics (
    id BIGSERIAL,
    metric_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    value FLOAT NOT NULL,
    unit VARCHAR(50),
    tags JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create analytics_reports table
CREATE TABLE analytics_reports (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    report_type VARCHAR(50) NOT NULL,
    parameters JSONB NOT NULL,
    data JSONB NOT NULL,
    generated_by BIGINT NOT NULL REFERENCES users(id),
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Create initial partitions for current month
-- User behavior partitions
DO $$
DECLARE
    start_date date := date_trunc('month', CURRENT_DATE);
    end_date date := start_date + interval '1 month';
    partition_name text := 'user_behavior_' || to_char(start_date, 'YYYY_MM');
BEGIN
    EXECUTE format('CREATE TABLE %I PARTITION OF user_behavior FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
    
    -- Create indexes on the partition
    EXECUTE format('CREATE INDEX idx_%I_session ON %I (session_id, created_at)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_user ON %I (user_id, created_at) WHERE user_id IS NOT NULL',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_page ON %I (page_url, created_at)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_behavior_gin ON %I USING gin(behavior_data)',
        partition_name, partition_name);
    
    -- Create BRIN index for time-series performance
    EXECUTE format('CREATE INDEX idx_%I_created_brin ON %I USING BRIN (created_at) WITH (pages_per_range = 64)',
        partition_name, partition_name);
END $$;

-- Performance metrics partitions
DO $$
DECLARE
    start_date date := date_trunc('month', CURRENT_DATE);
    end_date date := start_date + interval '1 month';
    partition_name text := 'performance_metrics_' || to_char(start_date, 'YYYY_MM');
BEGIN
    EXECUTE format('CREATE TABLE %I PARTITION OF performance_metrics FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
    
    -- Create indexes on the partition
    EXECUTE format('CREATE INDEX idx_%I_type ON %I (metric_type, created_at)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_name ON %I (name, created_at)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX idx_%I_tags_gin ON %I USING gin(tags)',
        partition_name, partition_name);
    
    -- Create BRIN index for time-series performance
    EXECUTE format('CREATE INDEX idx_%I_created_brin ON %I USING BRIN (created_at) WITH (pages_per_range = 64)',
        partition_name, partition_name);
END $$;

-- Create indexes for analytics_reports
CREATE INDEX idx_analytics_reports_type ON analytics_reports (report_type, generated_at DESC);
CREATE INDEX idx_analytics_reports_user ON analytics_reports (generated_by, generated_at DESC);
CREATE INDEX idx_analytics_reports_expires ON analytics_reports (expires_at) WHERE expires_at IS NOT NULL;

-- Create function to automatically create monthly partitions
CREATE OR REPLACE FUNCTION create_monthly_analytics_partitions()
RETURNS void AS $$
DECLARE
    start_date date;
    end_date date;
    partition_name text;
BEGIN
    -- Create partitions for next month
    start_date := date_trunc('month', CURRENT_DATE + interval '1 month');
    end_date := start_date + interval '1 month';
    
    -- User behavior partition
    partition_name := 'user_behavior_' || to_char(start_date, 'YYYY_MM');
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF user_behavior FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
    
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_session ON %I (session_id, created_at)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_user ON %I (user_id, created_at) WHERE user_id IS NOT NULL',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_page ON %I (page_url, created_at)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_behavior_gin ON %I USING gin(behavior_data)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_created_brin ON %I USING BRIN (created_at) WITH (pages_per_range = 64)',
        partition_name, partition_name);
    
    -- Performance metrics partition
    partition_name := 'performance_metrics_' || to_char(start_date, 'YYYY_MM');
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF performance_metrics FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
    
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_type ON %I (metric_type, created_at)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_name ON %I (name, created_at)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_tags_gin ON %I USING gin(tags)',
        partition_name, partition_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_created_brin ON %I USING BRIN (created_at) WITH (pages_per_range = 64)',
        partition_name, partition_name);
END;
$$ LANGUAGE plpgsql;

-- Create a scheduled job to create partitions monthly (requires pg_cron extension)
-- SELECT cron.schedule('create-analytics-partitions', '0 0 1 * *', 'SELECT create_monthly_analytics_partitions();');