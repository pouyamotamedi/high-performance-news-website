-- Create consistency validation tables for data integrity monitoring

-- Table for storing consistency issues
CREATE TABLE consistency_issues (
    id VARCHAR(50) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high')),
    article_id BIGINT,
    category_id BIGINT,
    tag_id BIGINT,
    user_id BIGINT,
    details JSONB,
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'resolved', 'ignored')),
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Table for storing remediation suggestions
CREATE TABLE remediation_suggestions (
    id VARCHAR(50) PRIMARY KEY,
    issue_id VARCHAR(50) NOT NULL REFERENCES consistency_issues(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    action TEXT NOT NULL,
    sql_query TEXT,
    parameters JSONB,
    confidence DECIMAL(3,2) CHECK (confidence >= 0.0 AND confidence <= 1.0),
    executed BOOLEAN DEFAULT false,
    executed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Table for manual review queue
CREATE TABLE manual_review_queue (
    id VARCHAR(50) PRIMARY KEY,
    issue_id VARCHAR(50) NOT NULL REFERENCES consistency_issues(id) ON DELETE CASCADE,
    priority VARCHAR(20) NOT NULL CHECK (priority IN ('low', 'medium', 'high')),
    assigned_to VARCHAR(100),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled')),
    notes TEXT,
    context JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Table for consistency check schedules
CREATE TABLE consistency_schedules (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    cron_expression VARCHAR(50),
    interval_seconds INTEGER,
    enabled BOOLEAN DEFAULT true,
    sample_size INTEGER DEFAULT 1000,
    check_types JSONB,
    last_run TIMESTAMP WITH TIME ZONE,
    next_run TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT check_schedule_timing CHECK (
        (cron_expression IS NOT NULL AND interval_seconds IS NULL) OR
        (cron_expression IS NULL AND interval_seconds IS NOT NULL)
    )
);

-- Table for storing check results
CREATE TABLE consistency_check_results (
    id VARCHAR(50) PRIMARY KEY,
    schedule_id VARCHAR(50) REFERENCES consistency_schedules(id) ON DELETE SET NULL,
    check_id VARCHAR(50),
    status VARCHAR(20) NOT NULL CHECK (status IN ('passed', 'warning', 'failed')),
    issues_found INTEGER DEFAULT 0,
    duration_ms BIGINT,
    sample_size INTEGER,
    metadata JSONB,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Table for tracking consistency trends
CREATE TABLE consistency_trends (
    id BIGSERIAL PRIMARY KEY,
    date DATE NOT NULL,
    issue_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    count INTEGER DEFAULT 0,
    resolved INTEGER DEFAULT 0,
    new_issues INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(date, issue_type, severity)
);

-- Table for consistency alerts
CREATE TABLE consistency_alerts (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance

-- Consistency issues indexes
CREATE INDEX idx_consistency_issues_type ON consistency_issues (type);
CREATE INDEX idx_consistency_issues_severity ON consistency_issues (severity);
CREATE INDEX idx_consistency_issues_status ON consistency_issues (status);
CREATE INDEX idx_consistency_issues_article ON consistency_issues (article_id) WHERE article_id IS NOT NULL;
CREATE INDEX idx_consistency_issues_category ON consistency_issues (category_id) WHERE category_id IS NOT NULL;
CREATE INDEX idx_consistency_issues_tag ON consistency_issues (tag_id) WHERE tag_id IS NOT NULL;
CREATE INDEX idx_consistency_issues_user ON consistency_issues (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_consistency_issues_created ON consistency_issues (created_at DESC);
CREATE INDEX idx_consistency_issues_details ON consistency_issues USING gin(details);

-- Remediation suggestions indexes
CREATE INDEX idx_remediation_suggestions_issue ON remediation_suggestions (issue_id);
CREATE INDEX idx_remediation_suggestions_confidence ON remediation_suggestions (confidence DESC);
CREATE INDEX idx_remediation_suggestions_executed ON remediation_suggestions (executed, created_at);

-- Manual review queue indexes
CREATE INDEX idx_manual_review_priority ON manual_review_queue (priority, created_at);
CREATE INDEX idx_manual_review_status ON manual_review_queue (status);
CREATE INDEX idx_manual_review_assigned ON manual_review_queue (assigned_to) WHERE assigned_to IS NOT NULL;

-- Schedule indexes
CREATE INDEX idx_consistency_schedules_enabled ON consistency_schedules (enabled, next_run) WHERE enabled = true;
CREATE INDEX idx_consistency_schedules_last_run ON consistency_schedules (last_run);

-- Check results indexes
CREATE INDEX idx_consistency_check_results_schedule ON consistency_check_results (schedule_id, created_at DESC);
CREATE INDEX idx_consistency_check_results_status ON consistency_check_results (status, created_at DESC);
CREATE INDEX idx_consistency_check_results_created ON consistency_check_results (created_at DESC);

-- Trends indexes
CREATE INDEX idx_consistency_trends_date ON consistency_trends (date DESC);
CREATE INDEX idx_consistency_trends_type ON consistency_trends (issue_type, date DESC);
CREATE INDEX idx_consistency_trends_severity ON consistency_trends (severity, date DESC);

-- Alerts indexes
CREATE INDEX idx_consistency_alerts_acknowledged ON consistency_alerts (acknowledged, created_at DESC);
CREATE INDEX idx_consistency_alerts_type ON consistency_alerts (type, created_at DESC);

-- Create trigger function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_consistency_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at
CREATE TRIGGER trigger_consistency_issues_updated_at
    BEFORE UPDATE ON consistency_issues
    FOR EACH ROW
    EXECUTE FUNCTION update_consistency_updated_at();

CREATE TRIGGER trigger_manual_review_queue_updated_at
    BEFORE UPDATE ON manual_review_queue
    FOR EACH ROW
    EXECUTE FUNCTION update_consistency_updated_at();

CREATE TRIGGER trigger_consistency_schedules_updated_at
    BEFORE UPDATE ON consistency_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_consistency_updated_at();

-- Create view for consistency dashboard
CREATE VIEW consistency_dashboard AS
SELECT 
    ci.type as issue_type,
    ci.severity,
    COUNT(*) as total_issues,
    COUNT(*) FILTER (WHERE ci.status = 'open') as open_issues,
    COUNT(*) FILTER (WHERE ci.status = 'resolved') as resolved_issues,
    COUNT(rs.id) as suggestions_count,
    COUNT(rs.id) FILTER (WHERE rs.executed = true) as executed_suggestions,
    COUNT(mrq.id) as manual_review_count,
    COUNT(mrq.id) FILTER (WHERE mrq.status = 'pending') as pending_review,
    MAX(ci.created_at) as last_occurrence
FROM consistency_issues ci
LEFT JOIN remediation_suggestions rs ON ci.id = rs.issue_id
LEFT JOIN manual_review_queue mrq ON ci.id = mrq.issue_id
WHERE ci.created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY ci.type, ci.severity
ORDER BY total_issues DESC;

-- Create view for recent check performance
CREATE VIEW consistency_check_performance AS
SELECT 
    cs.name as schedule_name,
    cs.enabled,
    cs.sample_size,
    cs.last_run,
    ccr.status as last_status,
    ccr.issues_found as last_issues_found,
    ccr.duration_ms as last_duration_ms,
    AVG(ccr.duration_ms) OVER (
        PARTITION BY cs.id 
        ORDER BY ccr.created_at 
        ROWS BETWEEN 9 PRECEDING AND CURRENT ROW
    ) as avg_duration_ms,
    AVG(ccr.issues_found) OVER (
        PARTITION BY cs.id 
        ORDER BY ccr.created_at 
        ROWS BETWEEN 9 PRECEDING AND CURRENT ROW
    ) as avg_issues_found
FROM consistency_schedules cs
LEFT JOIN LATERAL (
    SELECT status, issues_found, duration_ms, created_at
    FROM consistency_check_results
    WHERE schedule_id = cs.id
    ORDER BY created_at DESC
    LIMIT 1
) ccr ON true
ORDER BY cs.name;

-- Create function to get consistency health score
CREATE OR REPLACE FUNCTION get_consistency_health_score()
RETURNS TABLE (
    overall_score DECIMAL(5,2),
    total_issues INTEGER,
    high_severity_issues INTEGER,
    resolved_rate DECIMAL(5,2),
    avg_resolution_time_hours DECIMAL(10,2)
) AS $$
DECLARE
    v_total_issues INTEGER;
    v_high_severity_issues INTEGER;
    v_resolved_issues INTEGER;
    v_avg_resolution_hours DECIMAL(10,2);
    v_overall_score DECIMAL(5,2);
    v_resolved_rate DECIMAL(5,2);
BEGIN
    -- Get issue counts from last 30 days
    SELECT 
        COUNT(*),
        COUNT(*) FILTER (WHERE severity = 'high'),
        COUNT(*) FILTER (WHERE status = 'resolved')
    INTO v_total_issues, v_high_severity_issues, v_resolved_issues
    FROM consistency_issues
    WHERE created_at >= CURRENT_DATE - INTERVAL '30 days';
    
    -- Calculate resolved rate
    IF v_total_issues > 0 THEN
        v_resolved_rate := (v_resolved_issues::DECIMAL / v_total_issues) * 100;
    ELSE
        v_resolved_rate := 100.0;
    END IF;
    
    -- Calculate average resolution time
    SELECT AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600)
    INTO v_avg_resolution_hours
    FROM consistency_issues
    WHERE status = 'resolved' 
    AND resolved_at IS NOT NULL
    AND created_at >= CURRENT_DATE - INTERVAL '30 days';
    
    -- Calculate overall health score (0-100)
    v_overall_score := GREATEST(0, 
        100 - 
        (v_high_severity_issues * 10) - 
        (GREATEST(0, v_total_issues - 50) * 0.5) -
        (CASE WHEN v_resolved_rate < 80 THEN (80 - v_resolved_rate) * 0.5 ELSE 0 END)
    );
    
    RETURN QUERY SELECT 
        v_overall_score,
        v_total_issues,
        v_high_severity_issues,
        v_resolved_rate,
        COALESCE(v_avg_resolution_hours, 0.0);
END;
$$ LANGUAGE plpgsql;

-- Create function to cleanup old consistency data
CREATE OR REPLACE FUNCTION cleanup_old_consistency_data(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
    cutoff_date TIMESTAMP WITH TIME ZONE;
BEGIN
    cutoff_date := NOW() - (retention_days || ' days')::INTERVAL;
    
    -- Delete old resolved issues
    DELETE FROM consistency_issues 
    WHERE status = 'resolved' 
    AND resolved_at < cutoff_date;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Delete old check results
    DELETE FROM consistency_check_results 
    WHERE created_at < cutoff_date;
    
    -- Delete old trends (keep monthly aggregates)
    DELETE FROM consistency_trends 
    WHERE date < cutoff_date - INTERVAL '1 year';
    
    -- Delete old acknowledged alerts
    DELETE FROM consistency_alerts 
    WHERE acknowledged = true 
    AND acknowledged_at < cutoff_date;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON TABLE consistency_issues IS 'Stores data consistency issues found during validation checks';
COMMENT ON TABLE remediation_suggestions IS 'Automated suggestions for fixing consistency issues';
COMMENT ON TABLE manual_review_queue IS 'Queue for issues requiring manual review and intervention';
COMMENT ON TABLE consistency_schedules IS 'Schedules for automated consistency checks';
COMMENT ON TABLE consistency_check_results IS 'Results of scheduled consistency validation runs';
COMMENT ON TABLE consistency_trends IS 'Historical trend data for consistency issues';
COMMENT ON TABLE consistency_alerts IS 'Alerts generated for consistency issues';

COMMENT ON VIEW consistency_dashboard IS 'Dashboard view showing consistency issue summary by type and severity';
COMMENT ON VIEW consistency_check_performance IS 'Performance metrics for scheduled consistency checks';

COMMENT ON FUNCTION get_consistency_health_score() IS 'Calculates overall consistency health score (0-100)';
COMMENT ON FUNCTION cleanup_old_consistency_data(INTEGER) IS 'Cleans up old consistency data beyond retention period';