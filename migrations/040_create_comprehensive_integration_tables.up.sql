-- Migration: Create comprehensive integration testing tables
-- Version: 040
-- Description: Creates tables for comprehensive testing integration system

-- Enhanced test results table
CREATE TABLE IF NOT EXISTS enhanced_test_results (
    id SERIAL PRIMARY KEY,
    execution_id VARCHAR(100) UNIQUE NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    duration BIGINT NOT NULL, -- nanoseconds
    status VARCHAR(50) NOT NULL,
    result_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Configuration history table
CREATE TABLE IF NOT EXISTS configuration_history (
    id SERIAL PRIMARY KEY,
    config_version VARCHAR(50) NOT NULL,
    config_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(100) DEFAULT 'system',
    change_description TEXT
);

-- Add config_version column if it doesn't exist (for existing tables)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'configuration_history' 
        AND column_name = 'config_version'
    ) THEN
        ALTER TABLE configuration_history ADD COLUMN config_version VARCHAR(50) NOT NULL DEFAULT 'unknown';
    END IF;
END $$;

-- Configuration templates table
CREATE TABLE IF NOT EXISTS configuration_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    category VARCHAR(50),
    config_data JSONB NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Test reports table
CREATE TABLE IF NOT EXISTS test_reports (
    id SERIAL PRIMARY KEY,
    report_type VARCHAR(50) NOT NULL,
    report_data JSONB NOT NULL,
    generated_at TIMESTAMP DEFAULT NOW(),
    retention_until TIMESTAMP
);

-- Integration metrics table
CREATE TABLE IF NOT EXISTS integration_metrics (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(10,4) NOT NULL,
    metric_unit VARCHAR(20),
    component VARCHAR(100),
    tags JSONB,
    recorded_at TIMESTAMP DEFAULT NOW()
);

-- Component health tracking table
CREATE TABLE IF NOT EXISTS component_health (
    id SERIAL PRIMARY KEY,
    component_name VARCHAR(100) NOT NULL,
    health_status VARCHAR(20) NOT NULL,
    health_score DECIMAL(5,2),
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    metrics JSONB,
    checked_at TIMESTAMP DEFAULT NOW()
);

-- Test execution queue table
CREATE TABLE IF NOT EXISTS test_execution_queue (
    id SERIAL PRIMARY KEY,
    execution_id VARCHAR(100) UNIQUE NOT NULL,
    priority INTEGER DEFAULT 5,
    status VARCHAR(20) DEFAULT 'queued',
    execution_data JSONB NOT NULL,
    scheduled_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3
);

-- AI test generation history table
CREATE TABLE IF NOT EXISTS ai_test_generation_history (
    id SERIAL PRIMARY KEY,
    generation_id VARCHAR(100) UNIQUE NOT NULL,
    test_type VARCHAR(50) NOT NULL,
    prompt_data JSONB NOT NULL,
    generated_tests JSONB NOT NULL,
    success_rate DECIMAL(5,2),
    execution_results JSONB,
    generated_at TIMESTAMP DEFAULT NOW()
);

-- Performance baseline snapshots table
CREATE TABLE IF NOT EXISTS performance_baseline_snapshots (
    id SERIAL PRIMARY KEY,
    snapshot_id VARCHAR(100) UNIQUE NOT NULL,
    baseline_name VARCHAR(100) NOT NULL,
    snapshot_data JSONB NOT NULL,
    metrics_summary JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP
);

-- Security scan results table
CREATE TABLE IF NOT EXISTS security_scan_results (
    id SERIAL PRIMARY KEY,
    scan_id VARCHAR(100) UNIQUE NOT NULL,
    scan_type VARCHAR(50) NOT NULL,
    target_component VARCHAR(100),
    vulnerabilities JSONB,
    compliance_status JSONB,
    risk_score DECIMAL(5,2),
    scan_duration INTEGER, -- seconds
    scanned_at TIMESTAMP DEFAULT NOW()
);

-- Data consistency validation results table
CREATE TABLE IF NOT EXISTS data_consistency_results (
    id SERIAL PRIMARY KEY,
    validation_id VARCHAR(100) UNIQUE NOT NULL,
    validation_type VARCHAR(50) NOT NULL,
    consistency_score DECIMAL(5,2) NOT NULL,
    issues_found JSONB,
    validation_details JSONB,
    validated_at TIMESTAMP DEFAULT NOW()
);

-- Test environment usage tracking table
CREATE TABLE IF NOT EXISTS test_environment_usage (
    id SERIAL PRIMARY KEY,
    environment_id VARCHAR(100) NOT NULL,
    environment_type VARCHAR(50) NOT NULL,
    resource_allocation JSONB,
    usage_metrics JSONB,
    test_execution_id VARCHAR(100),
    allocated_at TIMESTAMP DEFAULT NOW(),
    released_at TIMESTAMP
);

-- Flaky test quarantine log table
CREATE TABLE IF NOT EXISTS flaky_test_quarantine_log (
    id SERIAL PRIMARY KEY,
    test_name VARCHAR(200) NOT NULL,
    quarantine_reason TEXT NOT NULL,
    flaky_score DECIMAL(5,4) NOT NULL,
    quarantined_at TIMESTAMP DEFAULT NOW(),
    released_at TIMESTAMP,
    auto_remediation_applied BOOLEAN DEFAULT FALSE,
    remediation_details JSONB
);

-- Test optimization suggestions table
CREATE TABLE IF NOT EXISTS test_optimization_suggestions (
    id SERIAL PRIMARY KEY,
    suggestion_id VARCHAR(100) UNIQUE NOT NULL,
    suggestion_type VARCHAR(50) NOT NULL,
    target_component VARCHAR(100),
    current_state JSONB,
    suggested_changes JSONB,
    expected_impact JSONB,
    confidence_score DECIMAL(5,4),
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    applied_at TIMESTAMP
);

-- Predictive insights table
CREATE TABLE IF NOT EXISTS predictive_insights (
    id SERIAL PRIMARY KEY,
    insight_id VARCHAR(100) UNIQUE NOT NULL,
    insight_type VARCHAR(50) NOT NULL,
    prediction JSONB NOT NULL,
    confidence_level DECIMAL(5,4) NOT NULL,
    time_frame VARCHAR(50),
    factors JSONB,
    validation_results JSONB,
    generated_at TIMESTAMP DEFAULT NOW(),
    validated_at TIMESTAMP
);

-- System alerts table
CREATE TABLE IF NOT EXISTS system_alerts (
    id SERIAL PRIMARY KEY,
    alert_id VARCHAR(100) UNIQUE NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    component VARCHAR(100),
    message TEXT NOT NULL,
    details JSONB,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    acknowledged_at TIMESTAMP,
    resolved_at TIMESTAMP
);

-- System validation results table
CREATE TABLE IF NOT EXISTS system_validation_results (
    id SERIAL PRIMARY KEY,
    validation_id VARCHAR(100) UNIQUE NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    overall_status VARCHAR(50) NOT NULL,
    overall_score DECIMAL(5,2) NOT NULL,
    system_health_score DECIMAL(5,2),
    reliability_score DECIMAL(5,2),
    security_score DECIMAL(5,2),
    performance_score DECIMAL(5,2),
    validation_duration BIGINT, -- nanoseconds
    components_validated INTEGER,
    tests_executed INTEGER,
    validation_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- System optimization results table
CREATE TABLE IF NOT EXISTS system_optimization_results (
    id SERIAL PRIMARY KEY,
    optimization_id VARCHAR(100) UNIQUE NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL,
    optimizations_applied INTEGER DEFAULT 0,
    optimization_duration BIGINT, -- nanoseconds
    baseline_metrics JSONB,
    optimization_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for better performance

-- Enhanced test results indexes
CREATE INDEX IF NOT EXISTS idx_enhanced_test_results_execution_id ON enhanced_test_results(execution_id);
CREATE INDEX IF NOT EXISTS idx_enhanced_test_results_status ON enhanced_test_results(status);
CREATE INDEX IF NOT EXISTS idx_enhanced_test_results_start_time ON enhanced_test_results(start_time);
CREATE INDEX IF NOT EXISTS idx_enhanced_test_results_duration ON enhanced_test_results(duration);

-- Configuration history indexes
CREATE INDEX IF NOT EXISTS idx_config_history_created_at ON configuration_history(created_at);
CREATE INDEX IF NOT EXISTS idx_config_history_version ON configuration_history(config_version);

-- Configuration templates indexes
CREATE INDEX IF NOT EXISTS idx_config_templates_category ON configuration_templates(category);
CREATE INDEX IF NOT EXISTS idx_config_templates_name ON configuration_templates(name);

-- Test reports indexes
CREATE INDEX IF NOT EXISTS idx_test_reports_type ON test_reports(report_type);
CREATE INDEX IF NOT EXISTS idx_test_reports_generated_at ON test_reports(generated_at);
CREATE INDEX IF NOT EXISTS idx_test_reports_retention ON test_reports(retention_until);

-- Integration metrics indexes
CREATE INDEX IF NOT EXISTS idx_integration_metrics_name ON integration_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_integration_metrics_component ON integration_metrics(component);
CREATE INDEX IF NOT EXISTS idx_integration_metrics_recorded_at ON integration_metrics(recorded_at);
CREATE INDEX IF NOT EXISTS idx_integration_metrics_tags ON integration_metrics USING GIN(tags);

-- Component health indexes
CREATE INDEX IF NOT EXISTS idx_component_health_name ON component_health(component_name);
CREATE INDEX IF NOT EXISTS idx_component_health_status ON component_health(health_status);
CREATE INDEX IF NOT EXISTS idx_component_health_checked_at ON component_health(checked_at);

-- Test execution queue indexes
CREATE INDEX IF NOT EXISTS idx_test_execution_queue_status ON test_execution_queue(status);
CREATE INDEX IF NOT EXISTS idx_test_execution_queue_priority ON test_execution_queue(priority);
CREATE INDEX IF NOT EXISTS idx_test_execution_queue_scheduled_at ON test_execution_queue(scheduled_at);

-- AI test generation history indexes
CREATE INDEX IF NOT EXISTS idx_ai_test_generation_type ON ai_test_generation_history(test_type);
CREATE INDEX IF NOT EXISTS idx_ai_test_generation_generated_at ON ai_test_generation_history(generated_at);
CREATE INDEX IF NOT EXISTS idx_ai_test_generation_success_rate ON ai_test_generation_history(success_rate);

-- Performance baseline snapshots indexes
CREATE INDEX IF NOT EXISTS idx_performance_baseline_name ON performance_baseline_snapshots(baseline_name);
CREATE INDEX IF NOT EXISTS idx_performance_baseline_created_at ON performance_baseline_snapshots(created_at);
CREATE INDEX IF NOT EXISTS idx_performance_baseline_expires_at ON performance_baseline_snapshots(expires_at);

-- Security scan results indexes
CREATE INDEX IF NOT EXISTS idx_security_scan_type ON security_scan_results(scan_type);
CREATE INDEX IF NOT EXISTS idx_security_scan_component ON security_scan_results(target_component);
CREATE INDEX IF NOT EXISTS idx_security_scan_risk_score ON security_scan_results(risk_score);
CREATE INDEX IF NOT EXISTS idx_security_scan_scanned_at ON security_scan_results(scanned_at);

-- Data consistency results indexes
CREATE INDEX IF NOT EXISTS idx_data_consistency_type ON data_consistency_results(validation_type);
CREATE INDEX IF NOT EXISTS idx_data_consistency_score ON data_consistency_results(consistency_score);
CREATE INDEX IF NOT EXISTS idx_data_consistency_validated_at ON data_consistency_results(validated_at);

-- Test environment usage indexes
CREATE INDEX IF NOT EXISTS idx_test_env_usage_env_id ON test_environment_usage(environment_id);
CREATE INDEX IF NOT EXISTS idx_test_env_usage_type ON test_environment_usage(environment_type);
CREATE INDEX IF NOT EXISTS idx_test_env_usage_execution_id ON test_environment_usage(test_execution_id);
CREATE INDEX IF NOT EXISTS idx_test_env_usage_allocated_at ON test_environment_usage(allocated_at);

-- Flaky test quarantine log indexes
CREATE INDEX IF NOT EXISTS idx_flaky_test_quarantine_test_name ON flaky_test_quarantine_log(test_name);
CREATE INDEX IF NOT EXISTS idx_flaky_test_quarantine_score ON flaky_test_quarantine_log(flaky_score);
CREATE INDEX IF NOT EXISTS idx_flaky_test_quarantine_quarantined_at ON flaky_test_quarantine_log(quarantined_at);

-- Test optimization suggestions indexes
CREATE INDEX IF NOT EXISTS idx_test_optimization_type ON test_optimization_suggestions(suggestion_type);
CREATE INDEX IF NOT EXISTS idx_test_optimization_component ON test_optimization_suggestions(target_component);
CREATE INDEX IF NOT EXISTS idx_test_optimization_status ON test_optimization_suggestions(status);
CREATE INDEX IF NOT EXISTS idx_test_optimization_confidence ON test_optimization_suggestions(confidence_score);

-- Predictive insights indexes
CREATE INDEX IF NOT EXISTS idx_predictive_insights_type ON predictive_insights(insight_type);
CREATE INDEX IF NOT EXISTS idx_predictive_insights_confidence ON predictive_insights(confidence_level);
CREATE INDEX IF NOT EXISTS idx_predictive_insights_generated_at ON predictive_insights(generated_at);

-- System alerts indexes
CREATE INDEX IF NOT EXISTS idx_system_alerts_type ON system_alerts(alert_type);
CREATE INDEX IF NOT EXISTS idx_system_alerts_severity ON system_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_system_alerts_component ON system_alerts(component);
CREATE INDEX IF NOT EXISTS idx_system_alerts_status ON system_alerts(status);
CREATE INDEX IF NOT EXISTS idx_system_alerts_created_at ON system_alerts(created_at);

-- System validation results indexes
CREATE INDEX IF NOT EXISTS idx_system_validation_results_validation_id ON system_validation_results(validation_id);
CREATE INDEX IF NOT EXISTS idx_system_validation_results_timestamp ON system_validation_results(timestamp);
CREATE INDEX IF NOT EXISTS idx_system_validation_results_status ON system_validation_results(overall_status);
CREATE INDEX IF NOT EXISTS idx_system_validation_results_score ON system_validation_results(overall_score);

-- System optimization results indexes
CREATE INDEX IF NOT EXISTS idx_system_optimization_results_optimization_id ON system_optimization_results(optimization_id);
CREATE INDEX IF NOT EXISTS idx_system_optimization_results_timestamp ON system_optimization_results(timestamp);
CREATE INDEX IF NOT EXISTS idx_system_optimization_results_status ON system_optimization_results(status);

-- Create composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_enhanced_test_results_status_time ON enhanced_test_results(status, start_time);
CREATE INDEX IF NOT EXISTS idx_integration_metrics_component_time ON integration_metrics(component, recorded_at);
CREATE INDEX IF NOT EXISTS idx_component_health_name_time ON component_health(component_name, checked_at);
CREATE INDEX IF NOT EXISTS idx_security_scan_type_time ON security_scan_results(scan_type, scanned_at);
CREATE INDEX IF NOT EXISTS idx_system_alerts_severity_status ON system_alerts(severity, status);

-- Add constraints and triggers

-- Add check constraints
ALTER TABLE enhanced_test_results 
ADD CONSTRAINT chk_enhanced_test_results_status 
CHECK (status IN ('running', 'success', 'failed', 'warning', 'cancelled'));

ALTER TABLE test_execution_queue 
ADD CONSTRAINT chk_test_execution_queue_status 
CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled'));

ALTER TABLE test_execution_queue 
ADD CONSTRAINT chk_test_execution_queue_priority 
CHECK (priority >= 1 AND priority <= 10);

ALTER TABLE component_health 
ADD CONSTRAINT chk_component_health_status 
CHECK (health_status IN ('healthy', 'degraded', 'unhealthy', 'unknown'));

ALTER TABLE component_health 
ADD CONSTRAINT chk_component_health_score 
CHECK (health_score >= 0 AND health_score <= 100);

ALTER TABLE system_alerts 
ADD CONSTRAINT chk_system_alerts_severity 
CHECK (severity IN ('low', 'medium', 'high', 'critical'));

ALTER TABLE system_alerts 
ADD CONSTRAINT chk_system_alerts_status 
CHECK (status IN ('active', 'acknowledged', 'resolved', 'suppressed'));

-- Create trigger function for updating timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add update triggers for tables with updated_at columns
CREATE TRIGGER update_enhanced_test_results_updated_at 
    BEFORE UPDATE ON enhanced_test_results 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_configuration_templates_updated_at 
    BEFORE UPDATE ON configuration_templates 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function for automatic report cleanup
CREATE OR REPLACE FUNCTION cleanup_expired_reports()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM test_reports 
    WHERE retention_until IS NOT NULL 
    AND retention_until < NOW();
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create function for automatic baseline cleanup
CREATE OR REPLACE FUNCTION cleanup_expired_baselines()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM performance_baseline_snapshots 
    WHERE expires_at IS NOT NULL 
    AND expires_at < NOW();
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create function for metrics aggregation
CREATE OR REPLACE FUNCTION aggregate_integration_metrics(
    component_name VARCHAR(100),
    metric_name VARCHAR(100),
    time_window INTERVAL DEFAULT '1 hour'
)
RETURNS TABLE(
    avg_value DECIMAL(10,4),
    min_value DECIMAL(10,4),
    max_value DECIMAL(10,4),
    count_values BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        AVG(im.metric_value) as avg_value,
        MIN(im.metric_value) as min_value,
        MAX(im.metric_value) as max_value,
        COUNT(im.metric_value) as count_values
    FROM integration_metrics im
    WHERE im.component = component_name
    AND im.metric_name = aggregate_integration_metrics.metric_name
    AND im.recorded_at >= NOW() - time_window;
END;
$$ LANGUAGE plpgsql;

-- Create function for health score calculation
CREATE OR REPLACE FUNCTION calculate_overall_health_score()
RETURNS DECIMAL(5,2) AS $$
DECLARE
    overall_score DECIMAL(5,2);
BEGIN
    SELECT AVG(health_score) INTO overall_score
    FROM component_health
    WHERE checked_at >= NOW() - INTERVAL '1 hour'
    AND health_score IS NOT NULL;
    
    RETURN COALESCE(overall_score, 0);
END;
$$ LANGUAGE plpgsql;

-- Insert default configuration templates
INSERT INTO configuration_templates (name, description, category, config_data, metadata) VALUES
('default-comprehensive', 'Default comprehensive testing configuration', 'default', 
 '{"system": {"max_concurrent_environments": 5, "max_parallel_tests": 10}, "ai": {"enabled": true}}',
 '{"version": "1.0", "author": "system", "tags": ["default", "comprehensive"]}'),
('performance-focused', 'Performance-focused testing configuration', 'performance',
 '{"system": {"max_concurrent_environments": 3, "max_parallel_tests": 5}, "performance": {"enabled": true}}',
 '{"version": "1.0", "author": "system", "tags": ["performance", "baseline"]}'),
('security-focused', 'Security-focused testing configuration', 'security',
 '{"security": {"enabled": true, "scan_types": ["sast", "dependency", "container"]}}',
 '{"version": "1.0", "author": "system", "tags": ["security", "compliance"]}')
ON CONFLICT (name) DO NOTHING;

-- Insert initial system alerts for monitoring
INSERT INTO system_alerts (alert_id, alert_type, severity, component, message, details) VALUES
('system-init-001', 'system', 'low', 'database', 'Comprehensive integration system initialized', 
 '{"migration_version": "040", "tables_created": 16, "indexes_created": 35}')
ON CONFLICT (alert_id) DO NOTHING;

-- Create views for common queries

-- View for recent test execution summary
CREATE OR REPLACE VIEW recent_test_executions AS
SELECT 
    execution_id,
    status,
    start_time,
    end_time,
    duration / 1000000000.0 as duration_seconds,
    (result_data->>'test_results')::jsonb as test_results_summary,
    created_at
FROM enhanced_test_results
WHERE start_time >= NOW() - INTERVAL '7 days'
ORDER BY start_time DESC;

-- View for component health dashboard
CREATE OR REPLACE VIEW component_health_dashboard AS
SELECT 
    component_name,
    health_status,
    health_score,
    error_count,
    last_error,
    checked_at,
    CASE 
        WHEN checked_at >= NOW() - INTERVAL '5 minutes' THEN 'current'
        WHEN checked_at >= NOW() - INTERVAL '1 hour' THEN 'recent'
        ELSE 'stale'
    END as freshness
FROM component_health
WHERE checked_at >= NOW() - INTERVAL '24 hours';

-- View for active alerts summary
CREATE OR REPLACE VIEW active_alerts_summary AS
SELECT 
    alert_type,
    severity,
    component,
    COUNT(*) as alert_count,
    MIN(created_at) as oldest_alert,
    MAX(created_at) as newest_alert
FROM system_alerts
WHERE status = 'active'
GROUP BY alert_type, severity, component
ORDER BY 
    CASE severity 
        WHEN 'critical' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
    END,
    alert_count DESC;

-- View for performance metrics trends
CREATE OR REPLACE VIEW performance_metrics_trends AS
SELECT 
    component,
    metric_name,
    DATE_TRUNC('hour', recorded_at) as hour,
    AVG(metric_value) as avg_value,
    MIN(metric_value) as min_value,
    MAX(metric_value) as max_value,
    COUNT(*) as sample_count
FROM integration_metrics
WHERE recorded_at >= NOW() - INTERVAL '24 hours'
GROUP BY component, metric_name, DATE_TRUNC('hour', recorded_at)
ORDER BY component, metric_name, hour;

-- Grant appropriate permissions (commented out - roles need to be created first)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO news_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO news_user;
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO news_readonly;

-- Add comments for documentation
COMMENT ON TABLE enhanced_test_results IS 'Stores comprehensive test execution results with enhanced analysis';
COMMENT ON TABLE configuration_history IS 'Tracks configuration changes over time';
COMMENT ON TABLE configuration_templates IS 'Stores reusable configuration templates';
COMMENT ON TABLE integration_metrics IS 'Stores system and component metrics for monitoring';
COMMENT ON TABLE component_health IS 'Tracks health status of system components';
COMMENT ON TABLE predictive_insights IS 'Stores AI-generated predictive insights and forecasts';
COMMENT ON TABLE system_alerts IS 'Manages system alerts and notifications';

COMMENT ON FUNCTION cleanup_expired_reports() IS 'Automatically removes expired test reports';
COMMENT ON FUNCTION calculate_overall_health_score() IS 'Calculates overall system health score';
COMMENT ON FUNCTION aggregate_integration_metrics(VARCHAR, VARCHAR, INTERVAL) IS 'Aggregates metrics for a component over a time window';

COMMENT ON VIEW recent_test_executions IS 'Shows recent test executions with summary information';
COMMENT ON VIEW component_health_dashboard IS 'Provides current component health status for dashboards';
COMMENT ON VIEW active_alerts_summary IS 'Summarizes active alerts by type and severity';
COMMENT ON VIEW performance_metrics_trends IS 'Shows performance metrics trends over time';