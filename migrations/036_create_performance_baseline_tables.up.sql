-- Performance Baseline and Regression Detection Tables
-- Migration: 036_create_performance_baseline_tables.up.sql

-- Performance baselines storage
CREATE TABLE IF NOT EXISTS performance_baselines (
    id BIGSERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    version VARCHAR(100) NOT NULL,
    metrics JSONB NOT NULL,
    environment VARCHAR(50) NOT NULL DEFAULT 'development',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Performance regression results storage
CREATE TABLE IF NOT EXISTS performance_regression_results (
    id BIGSERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    current_version VARCHAR(100) NOT NULL,
    baseline_version VARCHAR(100) NOT NULL,
    result_data JSONB NOT NULL,
    overall_status VARCHAR(20) NOT NULL CHECK (overall_status IN ('pass', 'warning', 'fail', 'critical')),
    overall_score DECIMAL(5,2) NOT NULL DEFAULT 0.00 CHECK (overall_score >= 0 AND overall_score <= 100),
    compared_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Performance metrics history for trend analysis
CREATE TABLE IF NOT EXISTS performance_metrics_history (
    id BIGSERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    metric_name VARCHAR(255) NOT NULL,
    metric_value DECIMAL(12,4) NOT NULL,
    metric_unit VARCHAR(20),
    environment VARCHAR(50) NOT NULL DEFAULT 'development',
    version VARCHAR(100),
    measured_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Performance alerts log
CREATE TABLE IF NOT EXISTS performance_alerts (
    id BIGSERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    alert_type VARCHAR(50) NOT NULL CHECK (alert_type IN ('regression', 'improvement', 'trend', 'threshold')),
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    message TEXT NOT NULL,
    details JSONB,
    environment VARCHAR(50) NOT NULL DEFAULT 'development',
    version VARCHAR(100),
    resolved BOOLEAN NOT NULL DEFAULT false,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_performance_baselines_test_env_active 
    ON performance_baselines(test_name, environment, is_active);

CREATE INDEX IF NOT EXISTS idx_performance_baselines_created_at 
    ON performance_baselines(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_performance_regression_results_test_name 
    ON performance_regression_results(test_name);

CREATE INDEX IF NOT EXISTS idx_performance_regression_results_compared_at 
    ON performance_regression_results(compared_at DESC);

CREATE INDEX IF NOT EXISTS idx_performance_regression_results_status 
    ON performance_regression_results(overall_status);

CREATE INDEX IF NOT EXISTS idx_performance_metrics_history_test_metric 
    ON performance_metrics_history(test_name, metric_name);

CREATE INDEX IF NOT EXISTS idx_performance_metrics_history_measured_at 
    ON performance_metrics_history(measured_at DESC);

CREATE INDEX IF NOT EXISTS idx_performance_metrics_history_env_version 
    ON performance_metrics_history(environment, version);

CREATE INDEX IF NOT EXISTS idx_performance_alerts_test_severity 
    ON performance_alerts(test_name, severity);

CREATE INDEX IF NOT EXISTS idx_performance_alerts_resolved 
    ON performance_alerts(resolved, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_performance_alerts_created_at 
    ON performance_alerts(created_at DESC);

-- GIN indexes for JSONB columns for efficient querying
CREATE INDEX IF NOT EXISTS idx_performance_baselines_metrics_gin 
    ON performance_baselines USING GIN(metrics);

CREATE INDEX IF NOT EXISTS idx_performance_regression_results_data_gin 
    ON performance_regression_results USING GIN(result_data);

CREATE INDEX IF NOT EXISTS idx_performance_alerts_details_gin 
    ON performance_alerts USING GIN(details);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_performance_baselines_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_performance_baselines_updated_at
    BEFORE UPDATE ON performance_baselines
    FOR EACH ROW
    EXECUTE FUNCTION update_performance_baselines_updated_at();

-- Comments for documentation
COMMENT ON TABLE performance_baselines IS 'Stores performance baseline metrics for regression detection';
COMMENT ON TABLE performance_regression_results IS 'Stores results of performance regression analysis';
COMMENT ON TABLE performance_metrics_history IS 'Historical performance metrics for trend analysis';
COMMENT ON TABLE performance_alerts IS 'Performance alerts and notifications log';

COMMENT ON COLUMN performance_baselines.metrics IS 'JSONB containing performance metrics data (mean, p95, p99, etc.)';
COMMENT ON COLUMN performance_baselines.is_active IS 'Whether this baseline is currently active for comparisons';
COMMENT ON COLUMN performance_regression_results.result_data IS 'JSONB containing detailed regression analysis results';
COMMENT ON COLUMN performance_regression_results.overall_score IS 'Overall performance score (0-100, higher is better)';
COMMENT ON COLUMN performance_metrics_history.metric_value IS 'The measured value of the performance metric';
COMMENT ON COLUMN performance_alerts.details IS 'JSONB containing additional alert details and context';