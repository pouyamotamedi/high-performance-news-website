# Performance Baseline Management System

This package provides a comprehensive performance baseline management and regression detection system for the news website application.

## Overview

The system consists of several interconnected components that work together to:

1. **Establish Performance Baselines** - Automatically collect and analyze performance metrics to create statistical baselines
2. **Detect Performance Regressions** - Compare current performance against baselines with intelligent analysis
3. **Provide Capacity Planning** - Analyze resource utilization and provide scaling recommendations
4. **Generate Optimization Suggestions** - Recommend specific actions to improve performance
5. **Automate Baseline Management** - Schedule and manage baseline updates automatically

## Components

### 1. Enhanced Baseline Manager (`enhanced_baseline_manager.go`)

The core component that provides advanced statistical analysis and baseline establishment:

```go
enhancedManager := performance.NewEnhancedBaselineManager(db)

// Establish a comprehensive baseline with statistical analysis
baseline, err := enhancedManager.EstablishEnhancedBaseline(
    "load_test", "v1.0.0", "production", rawMetrics)
```

**Features:**
- Statistical analysis with confidence intervals and outlier detection
- Trend analysis and forecasting
- Capacity planning with resource utilization analysis
- Distribution statistics (skewness, kurtosis, variance)
- Seasonal pattern detection

### 2. Automated Baseline Manager (`automated_baseline_manager.go`)

High-level manager that orchestrates the entire baseline management process:

```go
automatedManager := performance.NewAutomatedBaselineManager(db, enhancedManager)

// Establish automated baseline with validation and recommendations
result, err := automatedManager.EstablishAutomatedBaseline(
    "api_performance", "v2.1.0", "production")
```

**Features:**
- Automated baseline establishment with quality scoring
- Validation engine with configurable rules
- Recommendation generation
- Scheduling engine for automatic updates
- Comprehensive result reporting

### 3. Intelligent Regression Detector (`regression_detection_engine.go`)

Advanced regression detection with confidence scoring and root cause analysis:

```go
detector := performance.NewIntelligentRegressionDetector(db, enhancedManager)

request := performance.RegressionAnalysisRequest{
    TestName:       "load_test",
    CurrentVersion: "v1.1.0",
    Environment:    "production",
    CurrentMetrics: currentMetrics,
    AnalysisOptions: performance.AnalysisOptions{
        UseAdaptiveThresholds: true,
        ConfidenceLevel:       0.95,
        IncludeOptimizations:  true,
        DetailedAnalysis:      true,
    },
}

result, err := detector.DetectRegressions(request)
```

**Features:**
- Confidence scoring for regression detection
- Adaptive thresholds based on historical data
- Root cause analysis with pattern recognition
- Optimization suggestions engine
- Statistical significance testing
- Alert suppression and escalation logic

### 4. Base Baseline Manager (`baseline_manager.go`)

Fundamental baseline operations and storage:

```go
baseManager := performance.NewBaselineManager(db)

// Store a basic baseline
err := baseManager.StoreBaseline(baseline)

// Compare current metrics with baseline
result, err := baseManager.CompareWithBaseline(
    "test_name", "v1.1.0", "production", currentMetrics)
```

## Usage Examples

### Basic Baseline Establishment

```go
package main

import (
    "database/sql"
    "log"
    
    "github.com/your-org/news-website/internal/performance"
    _ "github.com/lib/pq"
)

func main() {
    db, err := sql.Open("postgres", "your-database-url")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create managers
    enhancedManager := performance.NewEnhancedBaselineManager(db)
    automatedManager := performance.NewAutomatedBaselineManager(db, enhancedManager)

    // Establish automated baseline
    result, err := automatedManager.EstablishAutomatedBaseline(
        "homepage_performance", "v1.0.0", "production")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Baseline established with quality score: %.2f", result.QualityScore)
    log.Printf("Recommendations: %d", len(result.Recommendations))
}
```

### Regression Detection

```go
// Create regression detector
detector := performance.NewIntelligentRegressionDetector(db, enhancedManager)

// Current performance metrics
currentMetrics := map[string]performance.MetricData{
    "http_req_duration": {
        Mean:   125.0,  // 25% increase from baseline
        P95:    187.5,  // 25% increase from baseline
        P99:    250.0,  // 25% increase from baseline
        Count:  1000,
        StdDev: 30.0,
        Unit:   "ms",
    },
}

// Analyze for regressions
request := performance.RegressionAnalysisRequest{
    TestName:       "api_performance",
    CurrentVersion: "v1.1.0",
    Environment:    "production",
    CurrentMetrics: currentMetrics,
    AnalysisOptions: performance.AnalysisOptions{
        UseAdaptiveThresholds: true,
        ConfidenceLevel:       0.95,
        IncludeOptimizations:  true,
        DetailedAnalysis:      true,
    },
}

result, err := detector.DetectRegressions(request)
if err != nil {
    log.Fatal(err)
}

// Check results
if len(result.Regressions) > 0 {
    log.Printf("Detected %d regressions with %.1f%% confidence", 
        len(result.Regressions), result.ConfidenceScore*100)
    
    for _, regression := range result.Regressions {
        log.Printf("- %s: %.1f%% increase (%s severity)", 
            regression.MetricName, regression.PercentChange, regression.Severity)
    }
    
    // Root cause analysis
    for _, hypothesis := range result.RootCauseAnalysis {
        log.Printf("Root cause: %s (confidence: %.1f%%)", 
            hypothesis.Hypothesis, hypothesis.Confidence*100)
    }
    
    // Optimization suggestions
    for _, suggestion := range result.OptimizationSuggestions {
        log.Printf("Suggestion: %s (%s priority)", 
            suggestion.Suggestion, suggestion.Priority)
    }
}
```

### Scheduled Baseline Updates

```go
// Update all scheduled baselines
err := automatedManager.UpdateBaselinesAutomatically()
if err != nil {
    log.Printf("Failed to update baselines: %v", err)
}

// Schedule a specific baseline for updates
schedulingEngine := automatedManager.SchedulingEngine
nextUpdate := time.Now().Add(24 * time.Hour)
err = schedulingEngine.ScheduleUpdate("api_performance", "production", nextUpdate)
if err != nil {
    log.Printf("Failed to schedule update: %v", err)
}
```

## CLI Tool

The system includes a command-line tool for baseline management:

```bash
# Establish a new baseline
./automated-baseline -test "load_test" -version "v1.0.0" -env "production" -action "establish"

# Update all scheduled baselines
./automated-baseline -action "update"

# Show status of baselines
./automated-baseline -action "status" -json

# Schedule baseline updates
./automated-baseline -test "api_test" -env "staging" -action "schedule"
```

### CLI Options

- `-test`: Test name for baseline operations
- `-version`: Version identifier (default: "auto")
- `-env`: Environment (development, staging, production)
- `-action`: Action to perform (establish, update, schedule, status)
- `-db`: Database URL (or use DATABASE_URL env var)
- `-verbose`: Enable verbose logging
- `-json`: Output results in JSON format

## Database Schema

The system uses several database tables:

### Core Tables
- `performance_baselines`: Stores baseline metrics and metadata
- `performance_regression_results`: Stores regression analysis results

### Automated Management Tables
- `automated_baseline_results`: Results of automated baseline establishment
- `baseline_schedules`: Schedules for automated baseline updates
- `baseline_validation_rules`: Rules for validating baseline quality
- `baseline_quality_metrics`: Detailed quality metrics for baselines
- `capacity_forecasts`: Capacity planning forecasts
- `performance_trend_forecasts`: Performance trend analysis results
- `baseline_recommendations`: Actionable recommendations

## Configuration

### Environment Variables

```bash
# Database connection
DATABASE_URL="postgres://user:pass@localhost/dbname?sslmode=disable"

# Baseline thresholds (optional)
BASELINE_CONFIDENCE_THRESHOLD=0.80
BASELINE_SAMPLE_SIZE_MIN=30
BASELINE_UPDATE_FREQUENCY=24h

# Regression detection (optional)
REGRESSION_CONFIDENCE_THRESHOLD=0.70
REGRESSION_ALERT_THRESHOLD=0.15
```

### Validation Rules

The system includes configurable validation rules:

```sql
INSERT INTO baseline_validation_rules (rule_name, condition_type, threshold_value, severity, action_type, description) VALUES
('minimum_sample_size', 'greater_than_equal', 30, 'critical', 'reject', 'Minimum required sample size'),
('data_completeness', 'greater_than_equal', 0.90, 'high', 'warn', 'Minimum data completeness'),
('variance_threshold', 'less_than_equal', 0.30, 'medium', 'warn', 'Maximum allowed variance');
```

## Testing

### Unit Tests

```bash
# Run all performance package tests
go test ./internal/performance/

# Run with coverage
go test -cover ./internal/performance/

# Run specific test
go test -run TestAutomatedBaselineManager ./internal/performance/
```

### Integration Tests

```bash
# Run integration tests (requires test database)
go test -run TestIntegration ./internal/performance/

# Set up test database
export DATABASE_URL="postgres://test:test@localhost/test_db?sslmode=disable"
```

### Load Testing Integration

The system integrates with k6 load testing:

```bash
# Run load test with baseline establishment
k6 run load-testing/performance-baseline-automation.js

# Run regression detection during load test
k6 run load-testing/integrated-regression-test.js
```

## Monitoring and Alerting

### Metrics Tracked

- **Response Time Metrics**: Mean, P95, P99 response times
- **Throughput Metrics**: Requests per second, articles per minute
- **Error Metrics**: Error rates, failure counts
- **Resource Metrics**: CPU usage, memory usage, database connections
- **Cache Metrics**: Hit rates, miss rates, eviction rates

### Alert Levels

- **Critical**: >50% performance degradation, system instability
- **High**: 25-50% performance degradation, resource exhaustion
- **Medium**: 15-25% performance degradation, capacity concerns
- **Low**: 5-15% performance degradation, optimization opportunities

### Alert Channels

- Slack notifications for team channels
- Email alerts for critical issues
- Webhook integration for monitoring systems
- Dashboard updates for real-time visibility

## Best Practices

### Baseline Establishment

1. **Sufficient Sample Size**: Ensure at least 30 data points for statistical significance
2. **Stable Conditions**: Establish baselines during stable system conditions
3. **Representative Load**: Use realistic load patterns that match production
4. **Regular Updates**: Update baselines after significant system changes

### Regression Detection

1. **Appropriate Thresholds**: Set thresholds based on business impact
2. **Confidence Scoring**: Use confidence scores to reduce false positives
3. **Root Cause Analysis**: Enable detailed analysis for critical regressions
4. **Alert Fatigue**: Use alert suppression to prevent notification spam

### Capacity Planning

1. **Resource Monitoring**: Track CPU, memory, and database utilization
2. **Growth Projections**: Use trend analysis for capacity forecasting
3. **Scaling Triggers**: Set up automated scaling based on utilization
4. **Bottleneck Identification**: Monitor for resource constraints

## Troubleshooting

### Common Issues

1. **Insufficient Sample Size**
   - Increase test duration or frequency
   - Check data collection processes
   - Verify database connectivity

2. **High False Positive Rate**
   - Adjust regression thresholds
   - Enable adaptive thresholds
   - Review baseline quality

3. **Missing Baselines**
   - Check database connectivity
   - Verify baseline establishment process
   - Review validation rules

4. **Performance Degradation**
   - Check system resources
   - Review recent deployments
   - Analyze trend data

### Debug Mode

Enable debug logging for troubleshooting:

```bash
export LOG_LEVEL=debug
./automated-baseline -verbose -test "debug_test" -action "establish"
```

## Contributing

When contributing to the performance baseline system:

1. **Add Tests**: Include unit tests for new functionality
2. **Update Documentation**: Keep README and code comments current
3. **Follow Patterns**: Use existing patterns for consistency
4. **Performance Impact**: Consider the performance impact of changes
5. **Database Migrations**: Include necessary schema changes

## License

This performance baseline management system is part of the news website project and follows the same licensing terms.