# Test Execution Monitoring & Quality Dashboard

This module provides comprehensive test execution monitoring, flakiness detection, failure pattern analysis, and a real-time quality metrics dashboard for the news website project.

## Features

### 1. Test Execution Monitoring (`execution_monitor.go`)

- **Real-time Test Tracking**: Records every test execution with detailed metrics
- **Flakiness Detection**: Automatically identifies flaky tests using statistical analysis
- **Failure Pattern Analysis**: Detects common failure patterns (timeouts, connection errors, etc.)
- **Coverage Trend Tracking**: Monitors test coverage changes over time
- **Alerting System**: Generates alerts for test failures, performance issues, and quality degradation

### 2. Quality Metrics Dashboard (`quality_dashboard.go`)

- **Real-time Dashboard**: Web-based dashboard with live metrics updates
- **Quality Gates**: Configurable quality gates with pass/fail status
- **Security Vulnerability Tracking**: Monitors and displays security issues
- **Performance Regression Detection**: Tracks performance degradation over time
- **Interactive Charts**: Coverage trends, failure patterns, and quality metrics visualization

### 3. Monitoring Integration (`monitoring_integration.go`)

- **Seamless Integration**: Works with existing test runners and CI/CD pipelines
- **Enhanced Test Results**: Provides enriched test results with monitoring data
- **Actionable Recommendations**: Generates specific recommendations for quality improvement
- **Git Integration**: Tracks test results by commit and build

## Database Schema

The monitoring system creates the following tables:

```sql
-- Test execution records
CREATE TABLE test_executions (
    id BIGSERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    test_suite VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    duration_ms BIGINT NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    error_message TEXT,
    stack_trace TEXT,
    coverage DECIMAL(5,2),
    environment VARCHAR(100),
    git_commit VARCHAR(40),
    build_id VARCHAR(100),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Flaky test tracking
CREATE TABLE test_flakiness (
    id BIGSERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL UNIQUE,
    test_suite VARCHAR(255) NOT NULL,
    total_runs BIGINT DEFAULT 0,
    failure_count BIGINT DEFAULT 0,
    flakiness_score DECIMAL(5,2) DEFAULT 0,
    last_failure TIMESTAMP WITH TIME ZONE,
    failure_pattern TEXT,
    recommendation TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Failure pattern analysis
CREATE TABLE failure_patterns (
    id BIGSERIAL PRIMARY KEY,
    pattern VARCHAR(500) NOT NULL,
    description TEXT,
    frequency BIGINT DEFAULT 1,
    affected_tests TEXT[],
    first_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    severity VARCHAR(50) DEFAULT 'medium',
    recommendation TEXT
);

-- Coverage trends
CREATE TABLE coverage_trends (
    id BIGSERIAL PRIMARY KEY,
    date DATE NOT NULL,
    test_suite VARCHAR(255),
    coverage_percent DECIMAL(5,2) NOT NULL,
    total_lines BIGINT,
    covered_lines BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Security vulnerabilities
CREATE TABLE security_vulnerabilities (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    component VARCHAR(255),
    cvss_score DECIMAL(3,1),
    first_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'open',
    remediation TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Performance regressions
CREATE TABLE performance_regressions (
    id BIGSERIAL PRIMARY KEY,
    test_name VARCHAR(255) NOT NULL,
    metric VARCHAR(100) NOT NULL,
    baseline_value DECIMAL(15,6) NOT NULL,
    current_value DECIMAL(15,6) NOT NULL,
    regression_percent DECIMAL(5,2) NOT NULL,
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    severity VARCHAR(50) DEFAULT 'medium',
    status VARCHAR(50) DEFAULT 'open'
);
```

## Usage Examples

### Basic Test Execution Tracking

```go
// Setup monitoring
env := NewTestEnvironment(t)
monitor := NewTestExecutionMonitor(env.DB)

// Record a test execution
execution := &TestExecution{
    TestName:     "TestUserLogin",
    TestSuite:    "integration",
    Status:       "passed",
    Duration:     1500, // milliseconds
    StartTime:    time.Now().Add(-2 * time.Second),
    EndTime:      time.Now(),
    Coverage:     95.5,
    Environment:  "test",
    GitCommit:    "abc123",
    BuildID:      "build-456",
}

err := monitor.RecordTestExecution(execution)
if err != nil {
    log.Printf("Failed to record test: %v", err)
}
```

### Using the Monitored Test Runner

```go
// Create monitored test runner
runner := NewMonitoredTestRunner(db)

// Run tests with comprehensive monitoring
ctx := context.Background()
result, err := runner.RunAllTestsWithMonitoring(ctx)
if err != nil {
    log.Printf("Test execution failed: %v", err)
    return
}

// Access enhanced results
fmt.Printf("Tests: %d passed, %d failed\n", result.PassedTests, result.FailedTests)
fmt.Printf("Coverage: %.2f%%\n", result.CoveragePercent)

// Review recommendations
for _, rec := range result.Recommendations {
    fmt.Printf("[%s] %s: %s\n", rec.Priority, rec.Title, rec.Action)
}

// Check quality gates
if result.QualityGateStatus.OverallStatus == "failed" {
    fmt.Println("Quality gates failed - deployment blocked")
}
```

### Setting up the Dashboard

```go
// Setup dashboard with Gin router
router := gin.New()
dashboard := NewQualityDashboard(db)
dashboard.SetupRoutes(router)

// Start server
router.Run(":8080")

// Access dashboard at http://localhost:8080/dashboard/
```

### Simple Test Tracking

```go
// For simple test tracking without full integration
tracker := NewTestExecutionTracker(db)

// Track individual test
err := tracker.TrackTestExecution(
    "TestExample",     // test name
    "unit",           // test suite
    "passed",         // status
    2*time.Second,    // duration
    94.5,             // coverage
    "",               // error message
)
```

## Dashboard API Endpoints

The dashboard provides the following REST API endpoints:

- `GET /dashboard/` - Main dashboard page
- `GET /dashboard/api/data` - Complete dashboard data
- `GET /dashboard/api/metrics/{timeRange}` - Test metrics (24h, 7d, 30d)
- `GET /dashboard/api/flaky-tests` - Flaky test information
- `GET /dashboard/api/failure-patterns` - Failure pattern analysis
- `GET /dashboard/api/coverage-trends/{days}` - Coverage trends
- `GET /dashboard/api/security-vulnerabilities` - Security issues
- `GET /dashboard/api/performance-regressions` - Performance regressions
- `GET /dashboard/api/quality-gates` - Quality gate status
- `POST /dashboard/api/quality-gates/evaluate` - Trigger quality gate evaluation
- `GET /dashboard/health` - Dashboard health check

## Quality Gates

The system includes configurable quality gates:

1. **Test Success Rate** (≥95%)
2. **Code Coverage** (≥95%)
3. **Flaky Test Count** (≤5)
4. **Security Vulnerabilities** (0 high/critical)
5. **Performance Regressions** (≤2)

Quality gates can be customized by modifying the `QualityGateManager`.

## Flakiness Detection Algorithm

The flakiness detection uses a sophisticated algorithm that considers:

- **Failure Rate**: Tests with 5-95% failure rate (pure failures aren't flaky)
- **Transition Rate**: Frequency of pass/fail transitions
- **Pattern Analysis**: Identifies intermittent vs. consistent failures

Flakiness Score = TransitionRate × (1 - |FailureRate - 0.5| × 2)

## Failure Pattern Detection

The system automatically detects common failure patterns:

- **Timeout**: Test execution timeouts
- **Connection Error**: Network/database connection issues
- **Memory Error**: Out of memory conditions
- **Race Condition**: Concurrency-related failures
- **Network Error**: Network connectivity problems
- **File System**: File operation failures
- **Assertion Failure**: Test assertion mismatches

## Performance Considerations

- **Async Processing**: Test analysis runs asynchronously to avoid blocking
- **Efficient Sampling**: Uses TABLESAMPLE for large dataset analysis
- **Indexed Queries**: All database queries are optimized with proper indexes
- **Batch Operations**: Supports batch recording for high-volume test suites

## Configuration

Environment variables for configuration:

```bash
# Database connection
TEST_DATABASE_URL=postgres://user:pass@localhost:5432/testdb

# Cache connection (optional)
TEST_CACHE_URL=redis://localhost:6379/1

# Build information
BUILD_ID=build-123
GIT_COMMIT=abc123def

# Dashboard settings
DASHBOARD_PORT=8080
DASHBOARD_REFRESH_INTERVAL=30s
```

## Integration with CI/CD

The monitoring system integrates seamlessly with CI/CD pipelines:

```yaml
# Example GitHub Actions integration
- name: Run Tests with Monitoring
  run: |
    go test -v -race -coverprofile=coverage.out ./...
    
- name: Upload Test Results
  run: |
    # Test results are automatically recorded via monitoring integration
    curl -X POST http://dashboard:8080/dashboard/api/quality-gates/evaluate
```

## Troubleshooting

### Common Issues

1. **Database Connection**: Ensure PostgreSQL is running and accessible
2. **Missing Tables**: Tables are created automatically on first run
3. **Performance**: For large test suites, consider increasing database connection pool
4. **Memory Usage**: Monitor memory usage with high-frequency test execution

### Debugging

Enable debug logging:

```go
log.SetLevel(log.DebugLevel)
```

Check dashboard health:

```bash
curl http://localhost:8080/dashboard/health
```

## Testing

Run the monitoring system tests:

```bash
# Unit tests
go test ./internal/testing/

# Integration tests (requires database)
TEST_DATABASE_URL=postgres://... go test ./internal/testing/

# Benchmarks
go test -bench=. ./internal/testing/
```

## Future Enhancements

Planned improvements:

1. **Machine Learning**: ML-based flakiness prediction
2. **Advanced Analytics**: Trend analysis and forecasting
3. **Integration**: Support for more CI/CD platforms
4. **Notifications**: Slack/email alerts for quality issues
5. **Historical Analysis**: Long-term trend analysis and reporting