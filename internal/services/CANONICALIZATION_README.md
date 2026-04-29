# Canonicalization System

The canonicalization system provides delayed canonical URL processing with admin override capabilities for the high-performance news website. This system implements requirement 8 (Automated Canonicalization) and requirement 41 from the project specifications.

## Overview

The canonicalization system allows articles to have their canonical URLs set to point to tags, categories, or custom URLs after a configurable delay (default 48 hours). This helps with SEO by consolidating page authority and preventing duplicate content issues.

## Key Features

- **48-Hour Delay**: Canonical URLs are applied 48 hours after scheduling by default
- **Admin Override**: Administrators can bypass the delay for immediate processing
- **Multiple Target Types**: Support for tag, category, and custom URL targets
- **Hierarchical Categories**: Automatic path generation for nested categories
- **Job Queue Processing**: Database-backed job storage with retry mechanisms
- **Background Processing**: Automatic job processing with configurable intervals
- **Comprehensive API**: Full REST API for job management and monitoring

## Architecture

### Database Schema

The system uses a `canonical_jobs` table to store scheduled canonicalization tasks:

```sql
CREATE TABLE canonical_jobs (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('tag', 'category', 'url')),
    target_id BIGINT, -- For tag or category targets
    target_url VARCHAR(500), -- For custom URL targets
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processed', 'cancelled', 'failed')),
    admin_override BOOLEAN DEFAULT false,
    created_by BIGINT REFERENCES users(id),
    processed_by BIGINT REFERENCES users(id),
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Core Components

1. **CanonicalManager**: Main service class handling job scheduling and processing
2. **Database Functions**: PostgreSQL functions for URL generation and job processing
3. **API Handlers**: REST API endpoints for job management
4. **Background Processor**: Automatic job processing with configurable intervals

## Usage

### Basic Usage

```go
// Create canonical manager
cm := services.NewCanonicalManager(db)

// Schedule a canonical job for a tag
target := services.NewTagTarget(tagID)
jobID, err := cm.ScheduleCanonicalJob(articleID, target, &userID, false)

// Schedule with admin override (immediate processing)
jobID, err := cm.ScheduleCanonicalJob(articleID, target, &userID, true)

// Process pending jobs manually
processed, err := cm.ProcessPendingJobs(&userID)

// Start background processor
cm.StartJobProcessor(5*time.Minute, &userID)
```

### Target Types

#### Tag Target
```go
target := services.NewTagTarget(123)
// Generates: /tag/tag-slug
```

#### Category Target
```go
target := services.NewCategoryTarget(456)
// Generates: /category/parent-slug/child-slug (for hierarchical categories)
// Generates: /category/category-slug (for root categories)
```

#### Custom URL Target
```go
target := services.NewURLTarget("/custom/canonical/path")
// Uses the exact URL provided
```

### API Endpoints

#### Schedule Canonical Job
```http
POST /api/v1/canonical/jobs
Content-Type: application/json

{
    "article_id": 123,
    "target": {
        "type": "tag",
        "id": 456
    },
    "admin_override": false
}
```

#### Get Pending Jobs
```http
GET /api/v1/canonical/jobs/pending
```

#### Get Jobs by Article
```http
GET /api/v1/canonical/jobs/article/123
```

#### Process Pending Jobs
```http
POST /api/v1/canonical/jobs/process
```

#### Cancel Job
```http
DELETE /api/v1/canonical/jobs/123
```

#### Retry Failed Job
```http
POST /api/v1/canonical/jobs/123/retry
Content-Type: application/json

{
    "admin_override": true
}
```

#### Generate Canonical URL (Preview)
```http
POST /api/v1/canonical/generate-url
Content-Type: application/json

{
    "target": {
        "type": "category",
        "id": 789
    }
}
```

#### Get Job Statistics
```http
GET /api/v1/canonical/stats
```

#### Cleanup Old Jobs
```http
POST /api/v1/canonical/cleanup
```

## Configuration

### Environment Variables

- `CANONICAL_PROCESSOR_INTERVAL`: Background processor interval (default: 5 minutes)
- `CANONICAL_DELAY_HOURS`: Default delay in hours (default: 48)
- `CANONICAL_MAX_RETRIES`: Maximum retry attempts (default: 3)
- `CANONICAL_CLEANUP_DAYS`: Days to keep old jobs (default: 30)

### Database Configuration

The system uses PostgreSQL functions for optimal performance:

- `schedule_canonical_job()`: Schedules new jobs with validation
- `process_canonical_job()`: Processes individual jobs
- `generate_canonical_url()`: Generates URLs for different target types
- `cleanup_old_canonical_jobs()`: Removes old processed jobs

## Job Processing

### Job States

1. **pending**: Job is scheduled but not yet ready for processing
2. **processed**: Job has been successfully processed
3. **cancelled**: Job was cancelled before processing
4. **failed**: Job processing failed (can be retried)

### Processing Flow

1. Job is scheduled with 48-hour delay (or immediate if admin override)
2. Background processor checks for ready jobs every 5 minutes
3. Ready jobs are processed in order of priority (admin override first)
4. Article's canonical URL is updated upon successful processing
5. Failed jobs can be retried up to 3 times
6. Old jobs are automatically cleaned up after 30 days

### Error Handling

- Invalid target IDs result in job failure with descriptive error messages
- Database errors are logged and jobs are marked as failed
- Failed jobs can be retried with exponential backoff
- Admin override allows bypassing retry limits

## Security

### Authorization

- **Job Scheduling**: Any authenticated user can schedule jobs
- **Admin Override**: Requires admin or editor role
- **Job Processing**: Requires admin or editor role
- **Job Cancellation**: Requires admin or editor role
- **Job Retry**: Requires admin or editor role
- **Cleanup**: Requires admin role

### Input Validation

- All target types are validated before job creation
- Target IDs are verified to exist in the database
- Custom URLs are validated for format and length
- SQL injection protection through prepared statements

## Monitoring

### Job Statistics

The system provides comprehensive statistics:

```json
{
    "stats": {
        "pending": 15,
        "processed": 1250,
        "cancelled": 5,
        "failed": 3
    }
}
```

### Logging

- Job scheduling events are logged with user and target information
- Processing results are logged with timing information
- Errors are logged with full stack traces
- Background processor activity is logged

### Health Checks

- Database connectivity is verified before processing
- Job queue depth is monitored
- Processing performance metrics are tracked

## Performance

### Optimizations

- Database indexes on frequently queried columns
- Prepared statements for repeated queries
- Batch processing for multiple jobs
- BRIN indexes for time-series data
- Connection pooling for database access

### Scalability

- Jobs are processed using database-level locking (FOR UPDATE SKIP LOCKED)
- Multiple processor instances can run concurrently
- Partitioning support for high-volume job storage
- Automatic cleanup prevents table bloat

## Testing

### Unit Tests

- Complete test coverage for all service methods
- Mock database testing for isolated unit tests
- Error condition testing for edge cases
- Input validation testing

### Integration Tests

- End-to-end workflow testing
- Database integration testing
- API endpoint testing
- Background processor testing

### Load Testing

- High-volume job scheduling
- Concurrent processing testing
- Database performance under load
- Memory usage monitoring

## Deployment

### Database Migration

Run the canonicalization migration:

```bash
migrate -path migrations -database "postgres://..." up
```

### Service Initialization

```go
// Initialize canonical manager
cm := services.NewCanonicalManager(db)

// Start background processor
cm.StartJobProcessor(5*time.Minute, nil)

// Register API routes
handlers := api.NewCanonicalHandlers(cm)
api.RegisterCanonicalRoutes(router, handlers)
```

### Monitoring Setup

- Set up alerts for failed job counts
- Monitor job queue depth
- Track processing performance
- Set up log aggregation

## Troubleshooting

### Common Issues

1. **Jobs not processing**: Check background processor is running
2. **High failure rate**: Verify target IDs exist in database
3. **Slow processing**: Check database performance and indexes
4. **Memory issues**: Verify job cleanup is running regularly

### Debug Commands

```sql
-- Check pending jobs
SELECT * FROM pending_canonical_jobs;

-- Check job statistics
SELECT status, COUNT(*) FROM canonical_jobs GROUP BY status;

-- Check failed jobs
SELECT * FROM canonical_jobs WHERE status = 'failed' ORDER BY updated_at DESC;

-- Manual job processing
SELECT process_canonical_job(job_id, user_id);
```

## Future Enhancements

- Support for bulk job scheduling
- Advanced scheduling options (specific dates/times)
- Job priority levels
- Webhook notifications for job completion
- Integration with content management workflows
- A/B testing for canonical URL effectiveness