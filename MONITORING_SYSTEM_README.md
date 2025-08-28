# Performance Monitoring System

This document describes the comprehensive performance monitoring system implemented for the high-performance news website. The system provides real-time monitoring, alerting, and performance analytics to ensure optimal system operation.

## Overview

The monitoring system consists of several key components:

- **Metrics Collection**: System, database, cache, and publishing metrics
- **Health Checks**: Component health monitoring with automated checks
- **Alerting System**: Configurable alerts with multiple notification channels
- **Performance Dashboard**: Real-time web dashboard for monitoring
- **Data Persistence**: Historical data storage with automatic cleanup
- **Prometheus Integration**: Metrics export for external monitoring tools

## Features

### 1. Comprehensive Metrics Collection

#### System Metrics
- CPU usage percentage
- Memory usage and availability
- Disk usage and space
- Network I/O statistics
- Load averages (1, 5, 15 minutes)

#### Database Metrics
- Active and idle connections
- Connection pool utilization
- Slow query detection
- Cache hit ratios
- Query performance statistics

#### Cache Metrics
- Hit/miss rates
- Memory usage
- Key count and operations
- Eviction and expiration statistics
- Average latency

#### Publishing Metrics
- Articles published per minute
- Publishing rate trends
- Failed publication tracking
- Queue status monitoring
- Static page generation stats

### 2. Health Check System

The system performs automated health checks on critical components:

- **Database**: Connection health, query performance
- **Cache**: Read/write operations, connectivity
- **System Resources**: CPU, memory, disk thresholds
- **Application**: Service availability and response times

Health checks run at configurable intervals and provide:
- Component status (healthy, degraded, unhealthy)
- Response time measurements
- Detailed error information
- Historical health data

### 3. Intelligent Alerting

#### Alert Types
- **Critical**: System failures, resource exhaustion
- **Warning**: Performance degradation, threshold breaches
- **Info**: Status changes, maintenance notifications

#### Alert Channels
- **Email**: SMTP-based notifications
- **Slack**: Webhook integration with rich formatting
- **Webhook**: Custom HTTP endpoints for integration

#### Alert Features
- Configurable thresholds and cooldown periods
- Alert rule management with conditions
- Automatic alert resolution
- Alert history and analytics
- Rate limiting to prevent spam

### 4. Performance Dashboard

A comprehensive web dashboard provides:

#### Real-time Metrics
- System resource utilization charts
- Database performance graphs
- Cache hit rate visualization
- Publishing rate trends

#### Health Status Overview
- Component health indicators
- Active alert notifications
- System uptime tracking
- Performance trend analysis

#### Interactive Features
- Auto-refresh capabilities
- Drill-down into specific metrics
- Historical data visualization
- Alert management interface

### 5. Data Persistence

#### Database Schema
- Partitioned tables for high-volume metrics
- Automated partition management
- Efficient indexing for time-series data
- Configurable data retention

#### Storage Optimization
- BRIN indexes for time-series performance
- Automatic old data cleanup
- Compressed historical storage
- Partition pruning for queries

## Configuration

### Environment Variables

```bash
# Prometheus Settings
MONITORING_ENABLE_PROMETHEUS=true
MONITORING_PROMETHEUS_PORT=9090
MONITORING_PROMETHEUS_PATH=/metrics

# Health Check Settings
MONITORING_ENABLE_HEALTH_CHECKS=true
MONITORING_HEALTH_CHECK_INTERVAL_SECONDS=30
MONITORING_HEALTH_CHECK_TIMEOUT_SECONDS=5

# Resource Monitoring
MONITORING_ENABLE_RESOURCE_MONITORING=true
MONITORING_RESOURCE_CHECK_INTERVAL_SECONDS=60
MONITORING_CPU_THRESHOLD=80.0
MONITORING_MEMORY_THRESHOLD=85.0
MONITORING_DISK_THRESHOLD=90.0

# Database Monitoring
MONITORING_ENABLE_DB_MONITORING=true
MONITORING_DB_CONNECTION_THRESHOLD=140
MONITORING_SLOW_QUERY_THRESHOLD_MS=1000

# Cache Monitoring
MONITORING_ENABLE_CACHE_MONITORING=true
MONITORING_CACHE_HIT_RATE_THRESHOLD=0.8

# Publishing Monitoring
MONITORING_ENABLE_PUBLISHING_MONITORING=true
MONITORING_PUBLISHING_RATE_THRESHOLD=35.0

# Alerting
MONITORING_ENABLE_ALERTING=true
MONITORING_ALERT_CHECK_INTERVAL_SECONDS=60
MONITORING_ALERT_COOLDOWN_MINUTES=15

# Alert Channels
MONITORING_EMAIL_ALERTING=false
MONITORING_SLACK_ALERTING=false
MONITORING_WEBHOOK_ALERTING=false
MONITORING_SLACK_WEBHOOK_URL=""
MONITORING_ALERT_WEBHOOK_URL=""
MONITORING_ALERT_EMAIL_RECIPIENTS=""

# Data Retention
MONITORING_METRICS_RETENTION_DAYS=30
MONITORING_ALERT_HISTORY_RETENTION_DAYS=90
```

### Threshold Configuration

Default performance thresholds:

```json
{
  "cpu_warning": 70.0,
  "cpu_critical": 85.0,
  "memory_warning": 80.0,
  "memory_critical": 90.0,
  "disk_warning": 85.0,
  "disk_critical": 95.0,
  "db_connections_warning": 120,
  "db_connections_critical": 140,
  "cache_hit_rate_warning": 0.7,
  "cache_hit_rate_critical": 0.5,
  "publishing_rate_warning": 25.0,
  "publishing_rate_critical": 15.0,
  "response_time_warning": 1000.0,
  "response_time_critical": 2000.0
}
```

## API Endpoints

### Health and Status
- `GET /health` - Basic health check
- `GET /health/live` - Kubernetes liveness probe
- `GET /health/ready` - Kubernetes readiness probe
- `GET /metrics` - Prometheus metrics endpoint

### Monitoring API (Admin Only)
- `GET /api/v1/monitoring/dashboard` - Complete dashboard data
- `GET /api/v1/monitoring/overview` - System overview
- `GET /api/v1/monitoring/metrics/system` - System metrics
- `GET /api/v1/monitoring/metrics/database` - Database metrics
- `GET /api/v1/monitoring/metrics/cache` - Cache metrics
- `GET /api/v1/monitoring/metrics/publishing` - Publishing metrics
- `GET /api/v1/monitoring/health/components` - Component health
- `GET /api/v1/monitoring/alerts` - Alert history
- `GET /api/v1/monitoring/alerts/active` - Active alerts
- `POST /api/v1/monitoring/alerts/test` - Send test alert

### Alert Management
- `GET /api/v1/monitoring/alert-rules` - List alert rules
- `POST /api/v1/monitoring/alert-rules` - Create alert rule
- `PUT /api/v1/monitoring/alert-rules/:id` - Update alert rule
- `DELETE /api/v1/monitoring/alert-rules/:id` - Delete alert rule
- `POST /api/v1/monitoring/alerts/:id/resolve` - Resolve alert

### Cache Management
- `POST /api/v1/monitoring/cache/clear` - Clear cache patterns
- `GET /api/v1/monitoring/cache/stats` - Cache statistics

## Usage Examples

### Command Line Tool

The monitoring system includes a CLI tool for testing and management:

```bash
# Check system status
./monitoring -command=status -verbose

# Get detailed metrics
./monitoring -command=metrics -verbose

# Run health checks
./monitoring -command=health -verbose

# Test alerting system
./monitoring -command=alerts -verbose

# Run monitoring test for 60 seconds
./monitoring -command=test -duration=60s -verbose

# Get dashboard data as JSON
./monitoring -command=dashboard -verbose
```

### Programmatic Usage

```go
// Create monitoring services
metricsService := services.NewMetricsService(db, cacheService, config)
healthService := services.NewHealthService(db, cacheService, config, metricsService)
alertingService := services.NewAlertingService(config, emailService)

// Start monitoring
ctx := context.Background()
go metricsService.StartMonitoring(ctx)

// Get system metrics
systemMetrics, err := metricsService.GetSystemMetrics()
if err != nil {
    log.Printf("Error getting system metrics: %v", err)
}

// Perform health check
healthResponse := healthService.PerformHealthCheck(true)
fmt.Printf("System health: %s\n", healthResponse.Status)

// Create alert rule
alertRule := &models.AlertRule{
    Name:        "high_cpu_usage",
    Description: "CPU usage is too high",
    Component:   "system",
    Metric:      "cpu_usage",
    Operator:    ">",
    Threshold:   80.0,
    Severity:    models.AlertSeverityWarning,
    Enabled:     true,
    Cooldown:    15 * time.Minute,
}

err = alertingService.CreateAlertRule(alertRule)
if err != nil {
    log.Printf("Error creating alert rule: %v", err)
}
```

### Dashboard Access

Access the monitoring dashboard at:
- Development: `http://localhost:8080/admin/monitoring`
- Production: `https://yourdomain.com/admin/monitoring`

## Database Schema

The monitoring system uses partitioned tables for efficient storage:

### Tables
- `health_checks` - Health check results
- `system_metrics` - System resource metrics (partitioned by date)
- `database_metrics` - Database performance metrics (partitioned by date)
- `cache_metrics` - Cache performance metrics (partitioned by date)
- `publishing_metrics` - Publishing performance metrics (partitioned by date)
- `alerts` - Alert history and status
- `alert_rules` - Alert rule configurations
- `user_sessions` - Active user tracking
- `monitoring_config` - System configuration

### Automatic Maintenance
- Daily partition creation for next 7 days
- Automatic cleanup of old partitions (30+ days)
- Index maintenance and optimization
- Configuration backup and validation

## Performance Considerations

### Resource Usage
- Monitoring overhead: <2% CPU, <100MB RAM
- Database storage: ~1GB per month for metrics
- Network overhead: Minimal for internal monitoring

### Optimization Features
- Efficient time-series storage with BRIN indexes
- Configurable collection intervals
- Intelligent caching of metrics
- Batch processing for high-volume data

### Scalability
- Horizontal scaling support for multiple instances
- Load balancer health check integration
- Distributed alerting with coordination
- External monitoring system integration

## Troubleshooting

### Common Issues

#### High Resource Usage
```bash
# Check monitoring configuration
./monitoring -command=status -verbose

# Reduce collection frequency
export MONITORING_RESOURCE_CHECK_INTERVAL_SECONDS=120
export MONITORING_HEALTH_CHECK_INTERVAL_SECONDS=60
```

#### Database Connection Issues
```bash
# Check database metrics
./monitoring -command=metrics -verbose

# Verify database connectivity
./monitoring -command=health -component=database -verbose
```

#### Alert Delivery Problems
```bash
# Test alert system
./monitoring -command=alerts -verbose

# Check alert configuration
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/v1/monitoring/config
```

### Log Analysis

Monitor system logs for:
- `UNHEALTHY:` - Component health issues
- `DEGRADED:` - Performance warnings
- `ALERT TRIGGERED:` - Alert activations
- `Error saving` - Persistence issues

### Performance Tuning

1. **Adjust Collection Intervals**
   - Increase intervals for non-critical metrics
   - Use different intervals for different metric types

2. **Optimize Database Queries**
   - Monitor slow query logs
   - Adjust connection pool settings
   - Optimize partition pruning

3. **Configure Alert Thresholds**
   - Set appropriate warning/critical levels
   - Adjust cooldown periods to prevent spam
   - Use conditional alerting for complex scenarios

## Integration

### Prometheus Integration
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'news-website'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### Grafana Dashboard
Import the provided Grafana dashboard configuration for advanced visualization and alerting.

### External Monitoring
The system supports integration with:
- Datadog
- New Relic
- AWS CloudWatch
- Google Cloud Monitoring

## Security

### Access Control
- Admin-only access to monitoring endpoints
- API key authentication for external integrations
- Role-based permissions for alert management

### Data Protection
- Encrypted storage of sensitive configuration
- Secure transmission of alert notifications
- Audit logging for configuration changes

### Network Security
- Internal-only monitoring endpoints
- Rate limiting on public health checks
- Secure webhook configurations

## Maintenance

### Regular Tasks
- Review and update alert thresholds
- Clean up old monitoring data
- Update alert notification channels
- Validate backup and recovery procedures

### Monitoring the Monitor
- Set up external monitoring for the monitoring system
- Configure alerts for monitoring system failures
- Regular testing of alert delivery channels
- Performance benchmarking of monitoring overhead

This monitoring system provides comprehensive visibility into the high-performance news website's operation, ensuring optimal performance and rapid issue detection.