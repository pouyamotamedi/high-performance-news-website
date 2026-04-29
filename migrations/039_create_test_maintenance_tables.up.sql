-- Test Maintenance and Evolution System Tables

-- Test metadata table
CREATE TABLE IF NOT EXISTS test_metadata (
    test_id VARCHAR(255) PRIMARY KEY,
    file_path VARCHAR(500) NOT NULL,
    test_name VARCHAR(255) NOT NULL,
    test_type VARCHAR(50) NOT NULL DEFAULT 'unit',
    dependencies JSONB DEFAULT '[]',
    code_coverage DECIMAL(5,4) DEFAULT 0.0,
    last_modified TIMESTAMP WITH TIME ZONE NOT NULL,
    last_executed TIMESTAMP WITH TIME ZONE,
    execution_count INTEGER DEFAULT 0,
    failure_rate DECIMAL(5,4) DEFAULT 0.0,
    average_runtime_ms BIGINT DEFAULT 0,
    complexity INTEGER DEFAULT 1,
    relationships JSONB DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'active',
    tags JSONB DEFAULT '[]',
    annotations JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Test relationships table
CREATE TABLE IF NOT EXISTS test_relationships (
    id SERIAL PRIMARY KEY,
    source_test_id VARCHAR(255) NOT NULL,
    target_test_id VARCHAR(255) NOT NULL,
    relation_type VARCHAR(50) NOT NULL,
    strength DECIMAL(3,2) NOT NULL DEFAULT 0.0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (source_test_id) REFERENCES test_metadata(test_id) ON DELETE CASCADE,
    FOREIGN KEY (target_test_id) REFERENCES test_metadata(test_id) ON DELETE CASCADE,
    UNIQUE(source_test_id, target_test_id, relation_type)
);

-- Test migrations table
CREATE TABLE IF NOT EXISTS test_migrations (
    migration_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(50),
    status VARCHAR(50) DEFAULT 'pending',
    steps JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error TEXT
);

-- Test lifecycle events table
CREATE TABLE IF NOT EXISTS test_lifecycle_events (
    event_id VARCHAR(255) PRIMARY KEY,
    test_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',
    reason TEXT,
    FOREIGN KEY (test_id) REFERENCES test_metadata(test_id) ON DELETE CASCADE
);

-- Test evolution table
CREATE TABLE IF NOT EXISTS test_evolution (
    test_id VARCHAR(255) PRIMARY KEY,
    changes JSONB DEFAULT '[]',
    metrics JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (test_id) REFERENCES test_metadata(test_id) ON DELETE CASCADE
);

-- Test changes table
CREATE TABLE IF NOT EXISTS test_changes (
    change_id VARCHAR(255) PRIMARY KEY,
    test_id VARCHAR(255) NOT NULL,
    change_type VARCHAR(50) NOT NULL,
    description TEXT,
    author VARCHAR(255),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    impact JSONB DEFAULT '{}',
    reason TEXT,
    FOREIGN KEY (test_id) REFERENCES test_metadata(test_id) ON DELETE CASCADE
);

-- Test metric snapshots table
CREATE TABLE IF NOT EXISTS test_metric_snapshots (
    id SERIAL PRIMARY KEY,
    test_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    coverage DECIMAL(5,4) DEFAULT 0.0,
    runtime_ms BIGINT DEFAULT 0,
    failure_rate DECIMAL(5,4) DEFAULT 0.0,
    execution_count INTEGER DEFAULT 0,
    complexity INTEGER DEFAULT 1,
    metrics_json JSONB DEFAULT '{}',
    FOREIGN KEY (test_id) REFERENCES test_metadata(test_id) ON DELETE CASCADE
);

-- Maintenance schedules table
CREATE TABLE IF NOT EXISTS maintenance_schedules (
    schedule_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    schedule_cron VARCHAR(100) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    last_run TIMESTAMP WITH TIME ZONE,
    next_run TIMESTAMP WITH TIME ZONE,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Test quality metrics table
CREATE TABLE IF NOT EXISTS test_quality_metrics (
    test_id VARCHAR(255) PRIMARY KEY,
    maintainability DECIMAL(3,2) DEFAULT 0.0,
    readability DECIMAL(3,2) DEFAULT 0.0,
    reliability DECIMAL(3,2) DEFAULT 0.0,
    performance DECIMAL(3,2) DEFAULT 0.0,
    coverage DECIMAL(3,2) DEFAULT 0.0,
    overall_quality DECIMAL(3,2) DEFAULT 0.0,
    last_calculated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    trend_direction VARCHAR(20) DEFAULT 'stable',
    FOREIGN KEY (test_id) REFERENCES test_metadata(test_id) ON DELETE CASCADE
);

-- Refactoring opportunities table
CREATE TABLE IF NOT EXISTS refactoring_opportunities (
    opportunity_id VARCHAR(255) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    test_ids JSONB DEFAULT '[]',
    description TEXT,
    benefits JSONB DEFAULT '[]',
    estimated_effort VARCHAR(50),
    priority VARCHAR(20) DEFAULT 'medium',
    auto_applicable BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_test_metadata_status ON test_metadata(status);
CREATE INDEX IF NOT EXISTS idx_test_metadata_type ON test_metadata(test_type);
CREATE INDEX IF NOT EXISTS idx_test_metadata_last_executed ON test_metadata(last_executed);
CREATE INDEX IF NOT EXISTS idx_test_metadata_failure_rate ON test_metadata(failure_rate);
CREATE INDEX IF NOT EXISTS idx_test_metadata_coverage ON test_metadata(code_coverage);

CREATE INDEX IF NOT EXISTS idx_test_relationships_source ON test_relationships(source_test_id);
CREATE INDEX IF NOT EXISTS idx_test_relationships_target ON test_relationships(target_test_id);
CREATE INDEX IF NOT EXISTS idx_test_relationships_type ON test_relationships(relation_type);
CREATE INDEX IF NOT EXISTS idx_test_relationships_strength ON test_relationships(strength);

CREATE INDEX IF NOT EXISTS idx_test_lifecycle_events_test_id ON test_lifecycle_events(test_id);
CREATE INDEX IF NOT EXISTS idx_test_lifecycle_events_type ON test_lifecycle_events(event_type);
CREATE INDEX IF NOT EXISTS idx_test_lifecycle_events_timestamp ON test_lifecycle_events(timestamp);

CREATE INDEX IF NOT EXISTS idx_test_changes_test_id ON test_changes(test_id);
CREATE INDEX IF NOT EXISTS idx_test_changes_type ON test_changes(change_type);
CREATE INDEX IF NOT EXISTS idx_test_changes_timestamp ON test_changes(timestamp);

CREATE INDEX IF NOT EXISTS idx_test_metric_snapshots_test_id ON test_metric_snapshots(test_id);
CREATE INDEX IF NOT EXISTS idx_test_metric_snapshots_timestamp ON test_metric_snapshots(timestamp);

CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_enabled ON maintenance_schedules(enabled);
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_next_run ON maintenance_schedules(next_run);

-- Create GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_test_metadata_dependencies_gin ON test_metadata USING GIN(dependencies);
CREATE INDEX IF NOT EXISTS idx_test_metadata_tags_gin ON test_metadata USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_test_metadata_annotations_gin ON test_metadata USING GIN(annotations);
CREATE INDEX IF NOT EXISTS idx_test_evolution_changes_gin ON test_evolution USING GIN(changes);
CREATE INDEX IF NOT EXISTS idx_test_evolution_metrics_gin ON test_evolution USING GIN(metrics);

-- Add comments for documentation
COMMENT ON TABLE test_metadata IS 'Stores metadata and metrics for all tests in the system';
COMMENT ON TABLE test_relationships IS 'Stores relationships between tests (dependencies, similarities, etc.)';
COMMENT ON TABLE test_migrations IS 'Tracks test framework migrations and upgrades';
COMMENT ON TABLE test_lifecycle_events IS 'Records lifecycle events for tests (created, deprecated, removed, etc.)';
COMMENT ON TABLE test_evolution IS 'Tracks the evolution and change history of tests';
COMMENT ON TABLE test_changes IS 'Records individual changes made to tests';
COMMENT ON TABLE test_metric_snapshots IS 'Stores periodic snapshots of test metrics for trend analysis';
COMMENT ON TABLE maintenance_schedules IS 'Defines scheduled maintenance tasks for the test suite';
COMMENT ON TABLE test_quality_metrics IS 'Stores calculated quality metrics for tests';
COMMENT ON TABLE refactoring_opportunities IS 'Identifies opportunities for test refactoring and improvement';

-- Create views for common queries
CREATE OR REPLACE VIEW active_tests AS
SELECT * FROM test_metadata WHERE status = 'active';

CREATE OR REPLACE VIEW deprecated_tests AS
SELECT * FROM test_metadata WHERE status = 'deprecated';

CREATE OR REPLACE VIEW high_failure_tests AS
SELECT * FROM test_metadata WHERE failure_rate > 0.1 AND status = 'active';

CREATE OR REPLACE VIEW slow_tests AS
SELECT * FROM test_metadata WHERE average_runtime_ms > 10000 AND status = 'active';

CREATE OR REPLACE VIEW low_coverage_tests AS
SELECT * FROM test_metadata WHERE code_coverage < 0.8 AND status = 'active';

CREATE OR REPLACE VIEW test_relationship_summary AS
SELECT 
    source_test_id,
    relation_type,
    COUNT(*) as relationship_count,
    AVG(strength) as avg_strength
FROM test_relationships
GROUP BY source_test_id, relation_type;

-- Create functions for maintenance operations
CREATE OR REPLACE FUNCTION update_test_metadata_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
CREATE TRIGGER update_test_metadata_timestamp_trigger
    BEFORE UPDATE ON test_metadata
    FOR EACH ROW
    EXECUTE FUNCTION update_test_metadata_timestamp();

CREATE TRIGGER update_test_evolution_timestamp_trigger
    BEFORE UPDATE ON test_evolution
    FOR EACH ROW
    EXECUTE FUNCTION update_test_metadata_timestamp();

CREATE TRIGGER update_maintenance_schedules_timestamp_trigger
    BEFORE UPDATE ON maintenance_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_test_metadata_timestamp();