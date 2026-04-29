-- Test Reliability Tracking System Tables
-- This migration creates all tables needed for the comprehensive test reliability tracking system

-- Core test execution tracking
CREATE TABLE IF NOT EXISTS test_executions (
    id SERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('passed', 'failed', 'error', 'skipped')),
    duration BIGINT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    error_message TEXT,
    environment VARCHAR(100),
    build_id VARCHAR(100),
    commit_hash VARCHAR(100),
    branch VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_test_executions_test_name ON test_executions(test_name, test_suite);
CREATE INDEX IF NOT EXISTS idx_test_executions_start_time ON test_executions(start_time);
CREATE INDEX IF NOT EXISTS idx_test_executions_status ON test_executions(status);
CREATE INDEX IF NOT EXISTS idx_test_executions_environment ON test_executions(environment);

-- Test reliability metrics storage
CREATE TABLE IF NOT EXISTS test_reliability_metrics (
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    reliability_score FLOAT NOT NULL CHECK (reliability_score >= 0 AND reliability_score <= 1),
    flakiness_score FLOAT NOT NULL CHECK (flakiness_score >= 0 AND flakiness_score <= 1),
    stability_trend VARCHAR(50) CHECK (stability_trend IN ('improving', 'stable', 'degrading')),
    total_executions BIGINT DEFAULT 0,
    successful_executions BIGINT DEFAULT 0,
    failed_executions BIGINT DEFAULT 0,
    error_executions BIGINT DEFAULT 0,
    skipped_executions BIGINT DEFAULT 0,
    average_duration BIGINT DEFAULT 0,
    duration_variance FLOAT DEFAULT 0,
    failure_patterns JSONB,
    environment_impact JSONB,
    time_of_day_impact JSONB,
    recent_performance JSONB,
    last_updated TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (test_name, test_suite)
);

-- Test quarantine management
CREATE TABLE IF NOT EXISTS test_quarantine (
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    quarantined_at TIMESTAMP NOT NULL,
    reintegrated_at TIMESTAMP,
    reason TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'quarantined' CHECK (status IN ('quarantined', 'reintegrated')),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (test_name, test_suite)
);

-- Failure pattern tracking
CREATE TABLE IF NOT EXISTS test_failure_patterns (
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    pattern_type VARCHAR(100) NOT NULL CHECK (pattern_type IN ('intermittent', 'consecutive', 'environment', 'timing', 'error_message')),
    description TEXT,
    frequency FLOAT CHECK (frequency >= 0 AND frequency <= 1),
    first_seen TIMESTAMP,
    last_seen TIMESTAMP,
    confidence FLOAT CHECK (confidence >= 0 AND confidence <= 1),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (test_name, test_suite, pattern_type)
);

-- Index for pattern queries
CREATE INDEX IF NOT EXISTS idx_test_failure_patterns_last_seen ON test_failure_patterns(last_seen);
CREATE INDEX IF NOT EXISTS idx_test_failure_patterns_confidence ON test_failure_patterns(confidence);

-- Remediation suggestions storage
CREATE TABLE IF NOT EXISTS test_remediation_suggestions (
    id SERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    suggestion JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for remediation queries
CREATE INDEX IF NOT EXISTS idx_test_remediation_suggestions_test ON test_remediation_suggestions(test_name, test_suite);
CREATE INDEX IF NOT EXISTS idx_test_remediation_suggestions_created_at ON test_remediation_suggestions(created_at);

-- Remediation attempt tracking
CREATE TABLE IF NOT EXISTS test_remediation_attempts (
    id SERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    remediation_type VARCHAR(100),
    success BOOLEAN DEFAULT FALSE,
    attempted_at TIMESTAMP DEFAULT NOW()
);

-- Environment adjustment tracking
CREATE TABLE IF NOT EXISTS test_environment_adjustments (
    id SERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    adjustment_type VARCHAR(100) NOT NULL,
    adjustment_value VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP DEFAULT NOW()
);

-- Environment resource limits
CREATE TABLE IF NOT EXISTS environment_resource_limits (
    environment VARCHAR(100) NOT NULL,
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    cpu_limit INTEGER,
    memory_limit INTEGER,
    applied_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (environment, test_name, test_suite)
);

-- Environment network configuration
CREATE TABLE IF NOT EXISTS environment_network_config (
    environment VARCHAR(100) NOT NULL,
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    config JSONB,
    applied_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (environment, test_name, test_suite)
);

-- Environment storage configuration
CREATE TABLE IF NOT EXISTS environment_storage_config (
    environment VARCHAR(100) NOT NULL,
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    config JSONB,
    applied_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (environment, test_name, test_suite)
);

-- Environment process configuration
CREATE TABLE IF NOT EXISTS environment_process_config (
    environment VARCHAR(100) NOT NULL,
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    config JSONB,
    applied_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (environment, test_name, test_suite)
);

-- Global environment configuration
CREATE TABLE IF NOT EXISTS environment_global_config (
    environment VARCHAR(100) NOT NULL,
    config_type VARCHAR(100) NOT NULL,
    config_value TEXT,
    applied_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (environment, config_type)
);

-- Environment optimization attempt tracking
CREATE TABLE IF NOT EXISTS environment_optimization_attempts (
    id SERIAL PRIMARY KEY,
    environment VARCHAR(100) NOT NULL,
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    success BOOLEAN DEFAULT FALSE,
    attempted_at TIMESTAMP DEFAULT NOW()
);

-- Flakiness tracking (legacy compatibility)
CREATE TABLE IF NOT EXISTS test_flakiness (
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) DEFAULT 'default',
    flakiness_score FLOAT NOT NULL CHECK (flakiness_score >= 0 AND flakiness_score <= 1),
    total_runs BIGINT DEFAULT 0,
    failure_count BIGINT DEFAULT 0,
    last_failure TIMESTAMP,
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'quarantined', 'fixed')),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (test_name)
);

-- Additional indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_test_reliability_metrics_flakiness ON test_reliability_metrics(flakiness_score);
CREATE INDEX IF NOT EXISTS idx_test_reliability_metrics_reliability ON test_reliability_metrics(reliability_score);
CREATE INDEX IF NOT EXISTS idx_test_quarantine_status ON test_quarantine(status);
CREATE INDEX IF NOT EXISTS idx_test_quarantine_quarantined_at ON test_quarantine(quarantined_at);
CREATE INDEX IF NOT EXISTS idx_environment_resource_limits_env ON environment_resource_limits(environment);
CREATE INDEX IF NOT EXISTS idx_test_flakiness_score ON test_flakiness(flakiness_score);

-- Comments for documentation
COMMENT ON TABLE test_executions IS 'Stores individual test execution records for reliability analysis';
COMMENT ON TABLE test_reliability_metrics IS 'Aggregated reliability metrics and analysis for each test';
COMMENT ON TABLE test_quarantine IS 'Tracks quarantined tests and their reintegration status';
COMMENT ON TABLE test_failure_patterns IS 'Detected failure patterns for intelligent remediation';
COMMENT ON TABLE test_remediation_suggestions IS 'AI-generated suggestions for fixing flaky tests';
COMMENT ON TABLE environment_resource_limits IS 'Environment-specific resource limits for test optimization';
COMMENT ON TABLE environment_global_config IS 'Global configuration settings per environment';

-- Create a view for easy flaky test monitoring
CREATE OR REPLACE VIEW flaky_tests_summary AS
SELECT 
    trm.test_name,
    trm.test_suite,
    trm.flakiness_score,
    trm.reliability_score,
    trm.stability_trend,
    trm.total_executions,
    trm.failed_executions,
    CASE 
        WHEN tq.status = 'quarantined' THEN 'quarantined'
        WHEN trm.flakiness_score > 0.5 THEN 'critical'
        WHEN trm.flakiness_score > 0.3 THEN 'warning'
        ELSE 'stable'
    END as status,
    tq.quarantined_at,
    trm.last_updated
FROM test_reliability_metrics trm
LEFT JOIN test_quarantine tq ON trm.test_name = tq.test_name AND trm.test_suite = tq.test_suite
WHERE trm.flakiness_score > 0.1
ORDER BY trm.flakiness_score DESC;

COMMENT ON VIEW flaky_tests_summary IS 'Summary view of flaky tests with status classification';