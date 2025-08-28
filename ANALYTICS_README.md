# Analytics System

This document describes the comprehensive analytics system implemented for the high-performance news website.

## Overview

The analytics system provides:
- Article view tracking with IP-based analytics
- Engagement tracking (likes, dislikes, shares, comments)
- User behavior analytics and performance metrics collection
- Reporting system with time-range filtering and export capabilities
- Real-time analytics dashboard
- Data export in multiple formats (CSV, JSON, XLSX, PDF)

## Architecture

### Components

1. **Models** (`internal/models/analytics.go`)
   - Data structures for analytics events
   - Validation logic
   - Export request handling

2. **Repository** (`internal/repositories/analytics_repository.go`)
   - Database operations
   - Bulk data insertion for performance
   - Complex analytics queries

3. **Service** (`internal/services/analytics_service.go`)
   - Business logic
   - Data processing and aggregation
   - Export functionality

4. **API Handlers** (`internal/api/analytics_handlers.go`)
   - HTTP endpoints for tracking and reporting
   - Request validation and response formatting

5. **Configuration** (`internal/config/analytics.go`)
   - Environment-based configuration
   - Privacy and compliance settings

## Database Schema

### Core Tables

#### `article_views` (Partitioned by created_at)
```sql
- id: BIGSERIAL
- article_id: BIGINT (references articles.id)
- ip_address: INET
- user_agent: TEXT
- referer: TEXT
- created_at: TIMESTAMP WITH TIME ZONE
```

#### `article_engagement` (Partitioned by created_at)
```sql
- id: BIGSERIAL
- article_id: BIGINT (references articles.id)
- action: VARCHAR(20) ('like', 'dislike', 'share', 'comment')
- ip_address: INET
- created_at: TIMESTAMP WITH TIME ZONE
```

#### `user_behavior` (Partitioned by created_at)
```sql
- id: BIGSERIAL
- session_id: VARCHAR(255)
- user_id: BIGINT (references users.id, nullable)
- ip_address: INET
- user_agent: TEXT
- page_url: TEXT
- referer: TEXT
- time_on_page: INTEGER (seconds)
- scroll_depth: FLOAT (percentage)
- behavior_data: JSONB (device, browser, OS, geolocation, UTM params)
- created_at: TIMESTAMP WITH TIME ZONE
```

#### `performance_metrics` (Partitioned by created_at)
```sql
- id: BIGSERIAL
- metric_type: VARCHAR(50)
- name: VARCHAR(255)
- value: FLOAT
- unit: VARCHAR(50)
- tags: JSONB
- created_at: TIMESTAMP WITH TIME ZONE
```

#### `analytics_reports`
```sql
- id: BIGSERIAL
- name: VARCHAR(255)
- report_type: VARCHAR(50)
- parameters: JSONB
- data: JSONB
- generated_by: BIGINT (references users.id)
- generated_at: TIMESTAMP WITH TIME ZONE
- expires_at: TIMESTAMP WITH TIME ZONE
```

### Partitioning Strategy

All analytics tables are partitioned by month to ensure optimal performance:
- Automatic partition creation via stored procedure
- BRIN indexes for time-series data
- Efficient data retention and archival

## API Endpoints

### Tracking Endpoints

#### Track Article View
```http
POST /api/analytics/track/view/{articleId}
```
Records a view event for an article.

#### Track Engagement
```http
POST /api/analytics/track/engagement/{articleId}/{action}
```
Records engagement actions: `like`, `dislike`, `share`, `comment`.

#### Track User Behavior
```http
POST /api/analytics/track/behavior
Content-Type: application/json

{
  "session_id": "session123",
  "user_id": 456,
  "page_url": "/article/test",
  "time_on_page": 120,
  "scroll_depth": 85.5
}
```

#### Bulk Track Views
```http
POST /api/analytics/track/views/bulk
Content-Type: application/json

[
  {
    "article_id": 1,
    "ip_address": "192.168.1.1",
    "user_agent": "Mozilla/5.0...",
    "referer": "https://google.com",
    "created_at": "2023-01-01T12:00:00Z"
  }
]
```

### Analytics Endpoints

#### Get Article Analytics
```http
GET /api/analytics/articles/{articleId}?start_date=2023-01-01&end_date=2023-01-31
```

Returns comprehensive analytics for a specific article:
```json
{
  "article_id": 123,
  "title": "Test Article",
  "slug": "test-article",
  "published_at": "2023-01-01T10:00:00Z",
  "view_count": 1000,
  "unique_views": 750,
  "like_count": 50,
  "dislike_count": 5,
  "share_count": 25,
  "comment_count": 15,
  "avg_time_on_page": 120.5,
  "avg_scroll_depth": 75.2,
  "bounce_rate": 0.35,
  "engagement_rate": 0.095,
  "top_referers": ["google.com", "facebook.com"],
  "top_countries": ["US", "UK", "CA"],
  "device_breakdown": {
    "desktop": 600,
    "mobile": 350,
    "tablet": 50
  }
}
```

#### Get Dashboard Metrics
```http
GET /api/analytics/dashboard?start_date=2023-01-01&end_date=2023-01-31
```

Returns key metrics for the analytics dashboard.

#### Get Top Articles
```http
GET /api/analytics/articles/top?limit=10&start_date=2023-01-01&end_date=2023-01-31
```

#### Get Traffic Sources
```http
GET /api/analytics/traffic-sources?start_date=2023-01-01&end_date=2023-01-31
```

### Reporting Endpoints

#### Generate Report
```http
POST /api/analytics/reports
Content-Type: application/json

{
  "report_type": "article_performance",
  "parameters": {
    "start_date": "2023-01-01T00:00:00Z",
    "end_date": "2023-01-31T23:59:59Z",
    "filters": {
      "category_id": 1
    },
    "group_by": ["date"],
    "metrics": ["views", "engagements"],
    "limit": 100
  }
}
```

#### Get Report
```http
GET /api/analytics/reports/{reportId}
```

#### Export Data
```http
POST /api/analytics/export
Content-Type: application/json

{
  "report_type": "engagement",
  "parameters": {
    "start_date": "2023-01-01T00:00:00Z",
    "end_date": "2023-01-31T23:59:59Z"
  },
  "format": "csv"
}
```

Supported formats: `csv`, `json`, `xlsx`, `pdf`

## Configuration

### Environment Variables

```bash
# Tracking settings
ANALYTICS_ENABLE_TRACKING=true
ANALYTICS_ENABLE_IP_TRACKING=true
ANALYTICS_ENABLE_USER_AGENT_TRACKING=true
ANALYTICS_ENABLE_BEHAVIOR_TRACKING=true
ANALYTICS_ENABLE_PERFORMANCE_TRACKING=true

# Data retention (days)
ANALYTICS_VIEW_DATA_RETENTION_DAYS=365
ANALYTICS_BEHAVIOR_DATA_RETENTION_DAYS=90
ANALYTICS_REPORT_RETENTION_DAYS=30

# Performance settings
ANALYTICS_BATCH_SIZE=1000
ANALYTICS_FLUSH_INTERVAL_SECONDS=30
ANALYTICS_MAX_CONCURRENT_REQUESTS=10

# Export settings
ANALYTICS_MAX_EXPORT_ROWS=100000
ANALYTICS_EXPORT_TIMEOUT_SECONDS=300

# Privacy settings
ANALYTICS_ANONYMIZE_IPS=false
ANALYTICS_RESPECT_DO_NOT_TRACK=true
ANALYTICS_COOKIE_CONSENT_REQUIRED=false

# Geolocation
ANALYTICS_ENABLE_GEOLOCATION=false
ANALYTICS_GEOLOCATION_PROVIDER=maxmind
ANALYTICS_GEOLOCATION_API_KEY=your_api_key

# Real-time analytics
ANALYTICS_ENABLE_REAL_TIME=false
ANALYTICS_REAL_TIME_BUFFER_SIZE=1000

# Alerting
ANALYTICS_ENABLE_ALERTING=false
ANALYTICS_HIGH_TRAFFIC_THRESHOLD=10000
ANALYTICS_LOW_ENGAGEMENT_THRESHOLD=0.02
ANALYTICS_HIGH_BOUNCE_RATE_THRESHOLD=0.8
ANALYTICS_SLOW_RESPONSE_THRESHOLD=2000.0
```

## Usage Examples

### Basic Tracking

```go
// Track article view
err := analyticsService.TrackArticleView(ctx, articleID, request)

// Track engagement
err := analyticsService.TrackEngagement(ctx, articleID, models.ActionLike, request)

// Track user behavior
err := analyticsService.TrackUserBehavior(ctx, sessionID, &userID, pageURL, timeOnPage, scrollDepth, request)
```

### Analytics Retrieval

```go
// Get article analytics
analytics, err := analyticsService.GetArticleAnalytics(ctx, articleID, startDate, endDate)

// Get dashboard metrics
metrics, err := analyticsService.GetDashboardMetrics(ctx, startDate, endDate)

// Get top articles
articles, err := analyticsService.GetTopArticles(ctx, startDate, endDate, limit)
```

### Report Generation

```go
// Generate report
params := models.ReportParameters{
    StartDate: startDate,
    EndDate:   endDate,
    Metrics:   []string{"views", "engagements"},
    Limit:     100,
}

report, err := analyticsService.GenerateReport(ctx, models.ReportTypeArticlePerformance, params, userID)

// Export data
exportReq := models.ExportRequest{
    ReportType: models.ReportTypeEngagement,
    Parameters: params,
    Format:     models.ExportFormatCSV,
}

data, contentType, err := analyticsService.ExportData(ctx, exportReq)
```

## Performance Considerations

### Database Optimization

1. **Partitioning**: All analytics tables are partitioned by month
2. **Indexing**: Strategic indexes on frequently queried columns
3. **BRIN Indexes**: For time-series data to reduce index size
4. **Bulk Operations**: Use COPY for high-volume data insertion

### Caching Strategy

1. **Dashboard Metrics**: Cache for 5-15 minutes
2. **Article Analytics**: Cache for 1 hour
3. **Reports**: Cache generated reports for 24 hours
4. **Top Articles**: Cache for 30 minutes

### Batch Processing

1. **View Tracking**: Batch insert every 30 seconds
2. **Engagement Tracking**: Real-time for immediate feedback
3. **Behavior Data**: Batch process every minute
4. **Performance Metrics**: Batch insert every 10 seconds

## Privacy and Compliance

### GDPR Compliance

1. **IP Anonymization**: Optional IP address anonymization
2. **Data Retention**: Configurable retention periods
3. **Right to Deletion**: Soft delete with anonymization
4. **Consent Management**: Cookie consent integration

### Privacy Features

1. **Do Not Track**: Respect DNT header
2. **Cookie Consent**: Require explicit consent
3. **Data Minimization**: Collect only necessary data
4. **Secure Storage**: Encrypted sensitive data

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Data Ingestion Rate**: Views/engagements per second
2. **Query Performance**: Average response times
3. **Storage Growth**: Database size and partition health
4. **Error Rates**: Failed tracking requests

### Alert Conditions

1. **High Traffic**: Unusual traffic spikes
2. **Low Engagement**: Declining engagement rates
3. **High Bounce Rate**: Increasing bounce rates
4. **Slow Responses**: API response time degradation

## Testing

### Unit Tests

- Service layer logic testing
- Data validation testing
- Export functionality testing
- Privacy compliance testing

### Integration Tests

- Database operations testing
- API endpoint testing
- Bulk operations testing
- Report generation testing

### Performance Tests

- High-volume data insertion
- Complex query performance
- Export operation timing
- Concurrent request handling

## Deployment

### Database Migration

```bash
# Run analytics table migrations
migrate -path migrations -database "postgres://..." up
```

### Service Configuration

1. Set environment variables
2. Configure database connections
3. Set up monitoring and alerting
4. Configure caching layer

### Monitoring Setup

1. Set up Prometheus metrics
2. Configure Grafana dashboards
3. Set up alert rules
4. Monitor key performance indicators

## Troubleshooting

### Common Issues

1. **High Memory Usage**: Reduce batch sizes
2. **Slow Queries**: Check partition pruning
3. **Missing Data**: Verify tracking configuration
4. **Export Timeouts**: Reduce export row limits

### Debug Commands

```bash
# Check partition health
SELECT schemaname, tablename, attname, n_distinct, correlation 
FROM pg_stats WHERE tablename LIKE 'article_views_%';

# Monitor query performance
SELECT query, mean_time, calls FROM pg_stat_statements 
WHERE query LIKE '%article_views%' ORDER BY mean_time DESC;

# Check data retention
SELECT 
  schemaname, 
  tablename, 
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables WHERE tablename LIKE 'article_%' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

## Future Enhancements

1. **Real-time Analytics**: WebSocket-based live updates
2. **Machine Learning**: Predictive analytics and recommendations
3. **A/B Testing**: Integrated experimentation framework
4. **Advanced Segmentation**: User cohort analysis
5. **Custom Dashboards**: User-configurable analytics views