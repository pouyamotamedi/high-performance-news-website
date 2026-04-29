-- Automated Baseline Management Tables
-- Migration: 038_create_automated_baseline_tables.up.sql

-- Automated baseline results storage
CREATE TABLE IF NOT EXISTS automated_baseline_results (
    id BIGSERIAL PRIMARY KEY,
    baseline_id BIGINT NOT NULL REFERENCES performance_baselines(id) ON DELETE CASCADE,
    test_name VARCHAR(255) NOT NULL,
    environment VARCHAR(100) NOT NULL,
    version VARCHAR(100) NOT NULL,
    established_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    data_points INTEGER NOT NULL,
    quality_score DECIMAL(5,2) NOT NULL CHECK (quality_score >= 0 AND quality_score <= 100),
    next_update_scheduled TIMESTAMP WITH TIME ZONE NOT NULL,
    result_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Baseline update schedules
CREATE TABLE IF NOT EXISTS baseline_schedules (
    id BIGSERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    environment VARCHAR(100) NOT NULL,
    update_frequency INTERVAL NOT NULL DEFAULT '24 hours',
    next_update TIMESTAMP WITH TIME ZONE NOT NULL,
    last_update TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'failed')),
    required_data_points INTEGER NOT NULL DEFAULT 50,
    collected_metrics JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(test_name, environment)
);

-- Baseline validation rules
CREATE TABLE IF NOT EXISTS baseline_validation_rules (
    id BIGSERIAL PRIMARY KEY,
    rule_name VARCHAR(255) NOT NULL UNIQUE,
    condition_type VARCHAR(100) NOT NULL,
    threshold_value DECIMAL(10,4) NOT NULL,
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    action_type VARCHAR(100) NOT NULL CHECK (action_type IN ('warn', 'reject', 'auto_fix')),
    enabled BOOLEAN NOT NULL DEFAULT true,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Baseline update triggers
CREATE TABLE IF NOT EXISTS baseline_update_triggers (
    id BIGSERIAL PRIMARY KEY,
    trigger_name VARCHAR(255) NOT NULL,
    trigger_type VARCHAR(100) NOT NULL CHECK (trigger_type IN ('time', 'data_volume', 'variance', 'regression')),
    condition_expression VARCHAR(500) NOT NULL,
    threshold_value DECIMAL(10,4) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    test_name VARCHAR(255),
    environment VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Baseline quality metrics
CREATE TABLE IF NOT EXISTS baseline_quality_metrics (
    id BIGSERIAL PRIMARY KEY,
    baseline_id BIGINT NOT NULL REFERENCES performance_baselines(id) ON DELETE CASCADE,
    automated_result_id BIGINT REFERENCES automated_baseline_results(id) ON DELETE CASCADE,
    sample_size INTEGER NOT NULL,
    data_completeness DECIMAL(5,4) NOT NULL CHECK (data_completeness >= 0 AND data_completeness <= 1),
    outliers_detected INTEGER NOT NULL DEFAULT 0,
    change_points_detected INTEGER NOT NULL DEFAULT 0,
    seasonality_detected BOOLEAN NOT NULL DEFAULT false,
    data_quality VARCHAR(50) NOT NULL CHECK (data_quality IN ('excellent', 'good', 'fair', 'poor')),
    confidence_level DECIMAL(5,4) NOT NULL DEFAULT 0.95,
    statistical_significance BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Capacity planning forecasts
CREATE TABLE IF NOT EXISTS capacity_forecasts (
    id BIGSERIAL PRIMARY KEY,
    automated_result_id BIGINT NOT NULL REFERENCES automated_baseline_results(id) ON DELETE CASCADE,
    resource_name VARCHAR(255) NOT NULL,
    current_utilization DECIMAL(5,4) NOT NULL CHECK (current_utilization >= 0 AND current_utilization <= 1),
    predicted_utilization DECIMAL(5,4) NOT NULL CHECK (predicted_utilization >= 0),
    time_to_capacity TIMESTAMP WITH TIME ZONE,
    bottleneck_risk VARCHAR(50) NOT NULL CHECK (bottleneck_risk IN ('low', 'medium', 'high')),
    scaling_recommendation TEXT,
    confidence DECIMAL(5,4) NOT NULL DEFAULT 0.8,
    forecast_horizon INTERVAL NOT NULL DEFAULT '7 days',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Performance trend forecasts
CREATE TABLE IF NOT EXISTS performance_trend_forecasts (
    id BIGSERIAL PRIMARY KEY,
    automated_result_id BIGINT NOT NULL REFERENCES automated_baseline_results(id) ON DELETE CASCADE,
    metric_name VARCHAR(255) NOT NULL,
    trend_direction VARCHAR(50) NOT NULL CHECK (trend_direction IN ('increasing', 'decreasing', 'stable')),
    trend_strength DECIMAL(5,4) NOT NULL CHECK (trend_strength >= 0 AND trend_strength <= 1),
    predicted_value DECIMAL(15,6) NOT NULL,
    confidence_lower DECIMAL(15,6) NOT NULL,
    confidence_upper DECIMAL(15,6) NOT NULL,
    confidence_level DECIMAL(5,4) NOT NULL DEFAULT 0.95,
    forecast_accuracy DECIMAL(5,4) NOT NULL DEFAULT 0.8,
    forecast_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Baseline recommendations
CREATE TABLE IF NOT EXISTS baseline_recommendations (
    id BIGSERIAL PRIMARY KEY,
    automated_result_id BIGINT NOT NULL REFERENCES automated_baseline_results(id) ON DELETE CASCADE,
    recommendation_type VARCHAR(100) NOT NULL,
    priority VARCHAR(50) NOT NULL CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    description TEXT NOT NULL,
    recommended_action TEXT NOT NULL,
    expected_impact TEXT,
    confidence DECIMAL(5,4) NOT NULL DEFAULT 0.8,
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'dismissed')),
    assigned_to VARCHAR(255),
    due_date TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_automated_baseline_results_test_env 
    ON automated_baseline_results(test_name, environment);

CREATE INDEX IF NOT EXISTS idx_automated_baseline_results_established_at 
    ON automated_baseline_results(established_at DESC);

CREATE INDEX IF NOT EXISTS idx_automated_baseline_results_quality_score 
    ON automated_baseline_results(quality_score DESC);

CREATE INDEX IF NOT EXISTS idx_baseline_schedules_next_update 
    ON baseline_schedules(next_update ASC) WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_baseline_schedules_test_env 
    ON baseline_schedules(test_name, environment);

CREATE INDEX IF NOT EXISTS idx_baseline_validation_rules_enabled 
    ON baseline_validation_rules(enabled) WHERE enabled = true;

CREATE INDEX IF NOT EXISTS idx_baseline_update_triggers_enabled 
    ON baseline_update_triggers(enabled) WHERE enabled = true;

CREATE INDEX IF NOT EXISTS idx_baseline_quality_metrics_baseline_id 
    ON baseline_quality_metrics(baseline_id);

CREATE INDEX IF NOT EXISTS idx_capacity_forecasts_result_id 
    ON capacity_forecasts(automated_result_id);

CREATE INDEX IF NOT EXISTS idx_capacity_forecasts_resource_utilization 
    ON capacity_forecasts(resource_name, predicted_utilization DESC);

CREATE INDEX IF NOT EXISTS idx_performance_trend_forecasts_result_id 
    ON performance_trend_forecasts(automated_result_id);

CREATE INDEX IF NOT EXISTS idx_performance_trend_forecasts_metric 
    ON performance_trend_forecasts(metric_name, forecast_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_baseline_recommendations_result_id 
    ON baseline_recommendations(automated_result_id);

CREATE INDEX IF NOT EXISTS idx_baseline_recommendations_priority_status 
    ON baseline_recommendations(priority, status);

-- GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_automated_baseline_results_data_gin 
    ON automated_baseline_results USING GIN(result_data);

CREATE INDEX IF NOT EXISTS idx_baseline_schedules_metrics_gin 
    ON baseline_schedules USING GIN(collected_metrics);

-- Triggers to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_automated_baseline_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_automated_baseline_results_updated_at
    BEFORE UPDATE ON automated_baseline_results
    FOR EACH ROW
    EXECUTE FUNCTION update_automated_baseline_updated_at();

CREATE TRIGGER trigger_baseline_schedules_updated_at
    BEFORE UPDATE ON baseline_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_automated_baseline_updated_at();

CREATE TRIGGER trigger_baseline_validation_rules_updated_at
    BEFORE UPDATE ON baseline_validation_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_automated_baseline_updated_at();

CREATE TRIGGER trigger_baseline_update_triggers_updated_at
    BEFORE UPDATE ON baseline_update_triggers
    FOR EACH ROW
    EXECUTE FUNCTION update_automated_baseline_updated_at();

CREATE TRIGGER trigger_baseline_recommendations_updated_at
    BEFORE UPDATE ON baseline_recommendations
    FOR EACH ROW
    EXECUTE FUNCTION update_automated_baseline_updated_at();

-- Insert default validation rules
INSERT INTO baseline_validation_rules (rule_name, condition_type, threshold_value, severity, action_type, description) VALUES
('minimum_sample_size', 'greater_than_equal', 30, 'critical', 'reject', 'Minimum required sample size for statistical significance'),
('data_completeness', 'greater_than_equal', 0.90, 'high', 'warn', 'Minimum data completeness percentage'),
('variance_threshold', 'less_than_equal', 0.30, 'medium', 'warn', 'Maximum allowed coefficient of variation'),
('outlier_percentage', 'less_than_equal', 0.10, 'medium', 'warn', 'Maximum allowed percentage of outliers'),
('change_point_threshold', 'less_than_equal', 5, 'medium', 'warn', 'Maximum allowed number of change points');

-- Insert default update triggers
INSERT INTO baseline_update_triggers (trigger_name, trigger_type, condition_expression, threshold_value, enabled) VALUES
('daily_update', 'time', 'every 24 hours', 24.0, true),
('data_volume_trigger', 'data_volume', 'new_data_points >= threshold', 100.0, true),
('variance_spike_trigger', 'variance', 'variance_increase > threshold', 0.40, true),
('regression_trigger', 'regression', 'performance_degradation > threshold', 0.25, true);

-- Comments for documentation
COMMENT ON TABLE automated_baseline_results IS 'Results of automated baseline establishment with quality metrics and recommendations';
COMMENT ON TABLE baseline_schedules IS 'Schedules for automated baseline updates';
COMMENT ON TABLE baseline_validation_rules IS 'Rules for validating baseline quality and data integrity';
COMMENT ON TABLE baseline_update_triggers IS 'Triggers that initiate automated baseline updates';
COMMENT ON TABLE baseline_quality_metrics IS 'Detailed quality metrics for established baselines';
COMMENT ON TABLE capacity_forecasts IS 'Capacity planning forecasts and scaling recommendations';
COMMENT ON TABLE performance_trend_forecasts IS 'Performance trend analysis and forecasting results';
COMMENT ON TABLE baseline_recommendations IS 'Actionable recommendations for baseline and performance improvements';

COMMENT ON COLUMN automated_baseline_results.quality_score IS 'Overall quality score (0-100) based on validation results and statistical analysis';
COMMENT ON COLUMN automated_baseline_results.result_data IS 'JSONB containing detailed automated baseline establishment results';
COMMENT ON COLUMN baseline_schedules.collected_metrics IS 'JSONB containing collected performance metrics for baseline establishment';
COMMENT ON COLUMN baseline_quality_metrics.data_completeness IS 'Percentage of expected metrics present in the data (0.0-1.0)';
COMMENT ON COLUMN capacity_forecasts.current_utilization IS 'Current resource utilization as a percentage (0.0-1.0)';
COMMENT ON COLUMN capacity_forecasts.predicted_utilization IS 'Predicted resource utilization (can exceed 1.0 for over-capacity)';
COMMENT ON COLUMN performance_trend_forecasts.trend_strength IS 'Strength of the detected trend (0.0-1.0)';
COMMENT ON COLUMN baseline_recommendations.confidence IS 'Confidence level in the recommendation (0.0-1.0)';