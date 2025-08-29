-- Create backup tables for disaster recovery system

-- Main backups table
CREATE TABLE IF NOT EXISTS backups (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(20) NOT NULL CHECK (type IN ('full', 'incremental', 'wal')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'validated', 'replicated')),
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT DEFAULT 0,
    checksum VARCHAR(64),
    compressed BOOLEAN DEFAULT false,
    encrypted BOOLEAN DEFAULT false,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_msg TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for backups table
CREATE INDEX idx_backups_type ON backups(type);
CREATE INDEX idx_backups_status ON backups(status);
CREATE INDEX idx_backups_created_at ON backups(created_at DESC);
CREATE INDEX idx_backups_completed_at ON backups(completed_at DESC) WHERE completed_at IS NOT NULL;

-- Backup replications table
CREATE TABLE IF NOT EXISTS backup_replications (
    id BIGSERIAL PRIMARY KEY,
    backup_id BIGINT NOT NULL REFERENCES backups(id) ON DELETE CASCADE,
    target_name VARCHAR(100) NOT NULL,
    target_location VARCHAR(500) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    replication_size BIGINT DEFAULT 0,
    replication_time BIGINT DEFAULT 0, -- milliseconds
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_msg TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for backup_replications table
CREATE INDEX idx_backup_replications_backup_id ON backup_replications(backup_id);
CREATE INDEX idx_backup_replications_status ON backup_replications(status);
CREATE INDEX idx_backup_replications_target ON backup_replications(target_name);

-- Backup validations table
CREATE TABLE IF NOT EXISTS backup_validations (
    id BIGSERIAL PRIMARY KEY,
    backup_id BIGINT NOT NULL REFERENCES backups(id) ON DELETE CASCADE,
    validation_type VARCHAR(50) NOT NULL, -- checksum, restore_test, integrity
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    validation_time BIGINT DEFAULT 0, -- milliseconds
    result JSONB,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_msg TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for backup_validations table
CREATE INDEX idx_backup_validations_backup_id ON backup_validations(backup_id);
CREATE INDEX idx_backup_validations_type ON backup_validations(validation_type);
CREATE INDEX idx_backup_validations_status ON backup_validations(status);

-- Restore operations table
CREATE TABLE IF NOT EXISTS restore_operations (
    id BIGSERIAL PRIMARY KEY,
    backup_id BIGINT NOT NULL REFERENCES backups(id),
    restore_type VARCHAR(50) NOT NULL, -- full, partial, point_in_time
    target_database VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    restore_time BIGINT DEFAULT 0, -- milliseconds
    records_restored BIGINT DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_msg TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for restore_operations table
CREATE INDEX idx_restore_operations_backup_id ON restore_operations(backup_id);
CREATE INDEX idx_restore_operations_status ON restore_operations(status);
CREATE INDEX idx_restore_operations_created_at ON restore_operations(created_at DESC);

-- Disaster recovery tests table
CREATE TABLE IF NOT EXISTS disaster_recovery_tests (
    id BIGSERIAL PRIMARY KEY,
    test_name VARCHAR(200) NOT NULL,
    backup_id BIGINT NOT NULL REFERENCES backups(id),
    test_type VARCHAR(50) NOT NULL, -- full_restore, partial_restore, point_in_time
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    test_environment VARCHAR(100),
    recovery_time BIGINT DEFAULT 0, -- milliseconds
    data_integrity BOOLEAN DEFAULT false,
    test_results JSONB,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_msg TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for disaster_recovery_tests table
CREATE INDEX idx_disaster_recovery_tests_backup_id ON disaster_recovery_tests(backup_id);
CREATE INDEX idx_disaster_recovery_tests_status ON disaster_recovery_tests(status);
CREATE INDEX idx_disaster_recovery_tests_created_at ON disaster_recovery_tests(created_at DESC);
CREATE INDEX idx_disaster_recovery_tests_test_type ON disaster_recovery_tests(test_type);

-- Create a view for backup metrics
CREATE OR REPLACE VIEW backup_metrics_view AS
SELECT 
    COUNT(*) as total_backups,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful_backups,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_backups,
    COALESCE(SUM(file_size), 0) as total_backup_size,
    COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000), 0) as average_backup_time,
    MAX(completed_at) as last_backup_time,
    MAX(CASE WHEN status = 'completed' THEN completed_at END) as last_successful_backup
FROM backups;

-- Create a function to automatically update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
CREATE TRIGGER update_backups_updated_at BEFORE UPDATE ON backups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_backup_replications_updated_at BEFORE UPDATE ON backup_replications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_backup_validations_updated_at BEFORE UPDATE ON backup_validations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_restore_operations_updated_at BEFORE UPDATE ON restore_operations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_disaster_recovery_tests_updated_at BEFORE UPDATE ON disaster_recovery_tests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create a function for automatic backup cleanup
CREATE OR REPLACE FUNCTION cleanup_old_backups(retention_days INTEGER DEFAULT 30)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete backup records older than retention period
    DELETE FROM backups 
    WHERE created_at < NOW() - INTERVAL '1 day' * retention_days;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create a function to get backup health status
CREATE OR REPLACE FUNCTION get_backup_health()
RETURNS JSONB AS $$
DECLARE
    result JSONB;
    recent_backup_count INTEGER;
    failed_backup_count INTEGER;
    last_successful_backup TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Check for recent successful backups (last 48 hours)
    SELECT COUNT(*) INTO recent_backup_count
    FROM backups 
    WHERE status = 'completed' 
    AND created_at > NOW() - INTERVAL '48 hours';
    
    -- Check for recent failed backups (last 24 hours)
    SELECT COUNT(*) INTO failed_backup_count
    FROM backups 
    WHERE status = 'failed' 
    AND created_at > NOW() - INTERVAL '24 hours';
    
    -- Get last successful backup time
    SELECT MAX(completed_at) INTO last_successful_backup
    FROM backups 
    WHERE status = 'completed';
    
    -- Build result JSON
    result := jsonb_build_object(
        'healthy', CASE 
            WHEN recent_backup_count > 0 AND failed_backup_count = 0 THEN true 
            ELSE false 
        END,
        'recent_successful_backups', recent_backup_count,
        'recent_failed_backups', failed_backup_count,
        'last_successful_backup', last_successful_backup,
        'check_time', NOW()
    );
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Insert initial backup configuration if needed
-- This would typically be handled by the application configuration
COMMENT ON TABLE backups IS 'Main backup records table for disaster recovery system';
COMMENT ON TABLE backup_replications IS 'Cross-region backup replication tracking';
COMMENT ON TABLE backup_validations IS 'Backup validation and integrity check results';
COMMENT ON TABLE restore_operations IS 'Database restore operation tracking';
COMMENT ON TABLE disaster_recovery_tests IS 'Disaster recovery test execution results';