# Data Consistency Validation System

This package implements a comprehensive data consistency validation system for the high-performance news website. It uses a sample-based approach to efficiently check data integrity across large datasets.

## Features

- **Sample-based validation**: Uses PostgreSQL TABLESAMPLE for efficient random sampling
- **Referential integrity checking**: Validates foreign key relationships
- **Multilingual consistency**: Ensures translation groups are consistent
- **SEO metadata validation**: Checks meta titles, descriptions, and schema markup
- **Automated remediation**: Provides suggestions and can execute high-confidence fixes
- **Scheduling system**: Runs checks automatically on configurable schedules
- **Comprehensive reporting**: Tracks trends and generates alerts

## Components

### ConsistencyChecker

The main validation engine that performs data consistency checks.

```go
checker := validation.NewConsistencyChecker(db)
check, err := checker.ValidateDataConsistency(ctx)
```

### ConsistencyReporter

Handles issue reporting, remediation suggestions, and manual review queue management.

```go
reporter := validation.NewConsistencyReporter()
reporter.SetDatabase(db)
err := reporter.ProcessIssues(ctx, check.Issues)
```

### CheckScheduler

Manages automated scheduling of consistency checks.

```go
scheduler := validation.NewCheckScheduler()
scheduler.SetDependencies(db, checker, reporter, monitoringClient)
err := scheduler.Start(ctx)
```

## Issue Types

### Referential Integrity Issues

- `broken_author_reference`: Article references non-existent author
- `broken_category_reference`: Article references non-existent category
- `broken_translation_group_reference`: Article references non-existent translation group
- `orphaned_article_tag`: Article has tag reference to non-existent tag

### Multilingual Consistency Issues

- `translation_status_inconsistency`: Articles in translation group have inconsistent publication status
- `duplicate_language_in_translation_group`: Multiple articles with same language in translation group

### SEO Metadata Issues

- `missing_meta_title`: Article missing meta title
- `missing_meta_description`: Article missing meta description
- `invalid_schema_type`: Article has invalid schema type
- `invalid_canonical_url`: Article has malformed canonical URL
- `meta_title_too_long`: Meta title exceeds 60 characters
- `meta_description_too_long`: Meta description exceeds 160 characters

## Usage

### Command Line Tool

Run a single consistency check:

```bash
go run cmd/consistency-checker/main.go -sample-size 1000 -output summary
```

Run with automatic remediation:

```bash
go run cmd/consistency-checker/main.go -execute-remediation -verbose
```

Start the scheduler:

```bash
go run cmd/consistency-checker/main.go -schedule -verbose
```

### Programmatic Usage

```go
package main

import (
    "context"
    "log"
    
    "high-performance-news-website/internal/validation"
    "high-performance-news-website/pkg/database"
)

func main() {
    // Setup database connection
    db, err := database.New(config.Database)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create consistency checker
    checker := validation.NewConsistencyChecker(db)
    
    // Run consistency check
    ctx := context.Background()
    check, err := checker.ValidateDataConsistency(ctx)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d issues in %v", len(check.Issues), check.Duration)
    
    // Process issues
    reporter := validation.NewConsistencyReporter()
    reporter.SetDatabase(db)
    
    if len(check.Issues) > 0 {
        err = reporter.ProcessIssues(ctx, check.Issues)
        if err != nil {
            log.Printf("Failed to process issues: %v", err)
        }
    }
}
```

## Database Schema

The system requires the following tables (created by migration `035_create_consistency_validation_tables.up.sql`):

- `consistency_issues`: Stores detected issues
- `remediation_suggestions`: Automated fix suggestions
- `manual_review_queue`: Issues requiring manual review
- `consistency_schedules`: Automated check schedules
- `consistency_check_results`: Historical check results
- `consistency_trends`: Trend tracking data
- `consistency_alerts`: System alerts

## Configuration

### Default Schedules

The system creates three default schedules:

1. **Hourly Quick Check**: Sample size 100, referential integrity only
2. **Daily Full Check**: Sample size 1000, all check types
3. **Weekly Comprehensive Check**: Sample size 5000, all check types

### Remediation Confidence Levels

- **0.9-1.0**: High confidence - safe for automatic execution
- **0.7-0.9**: Medium confidence - review recommended
- **0.5-0.7**: Low confidence - manual review required
- **<0.5**: Very low confidence - manual intervention needed

## Performance Characteristics

### Sample Sizes and Performance

- **100 articles**: ~1-2 seconds
- **1,000 articles**: ~5-10 seconds  
- **5,000 articles**: ~20-30 seconds
- **10,000+ articles**: ~45-60 seconds

### Memory Usage

The system uses efficient sampling to keep memory usage low:
- Processes articles in batches
- Uses database cursors for large result sets
- Minimal in-memory storage of issue data

## Monitoring and Alerting

### Metrics

The system can send metrics to monitoring systems:

- `consistency_check.duration`: Check execution time
- `consistency_check.issues_found`: Number of issues detected
- `consistency_check.sample_size`: Articles processed

### Alerts

Automatic alerts are generated for:

- High number of issues (>50 total)
- Many high-severity issues (>10)
- Issue type spikes (>50 of same type)
- Check failures or timeouts

## Best Practices

### Running Checks

1. **Start small**: Begin with sample size 100-500 for initial validation
2. **Schedule appropriately**: Run comprehensive checks during low-traffic periods
3. **Monitor performance**: Track check duration and adjust sample sizes
4. **Review manually**: Always review high-confidence remediation before execution

### Issue Management

1. **Prioritize by severity**: Address high-severity issues first
2. **Batch remediation**: Group similar issues for efficient fixing
3. **Validate fixes**: Run follow-up checks after remediation
4. **Track trends**: Monitor issue patterns over time

### Database Maintenance

1. **Regular cleanup**: Use `cleanup_old_consistency_data()` function
2. **Index maintenance**: Ensure indexes on large tables are optimized
3. **Partition management**: Consider partitioning for high-volume installations

## Troubleshooting

### Common Issues

**Slow performance**:
- Reduce sample size
- Check database indexes
- Run during low-traffic periods

**Too many false positives**:
- Review validation rules
- Adjust confidence thresholds
- Update test data quality

**Missing issues**:
- Increase sample size
- Check sampling randomness
- Verify validation logic

### Debugging

Enable verbose logging:

```bash
go run cmd/consistency-checker/main.go -verbose
```

Check database performance:

```sql
-- Check sampling performance
EXPLAIN ANALYZE SELECT * FROM articles TABLESAMPLE SYSTEM (1) LIMIT 1000;

-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan 
FROM pg_stat_user_indexes 
WHERE schemaname = 'public' 
ORDER BY idx_scan DESC;
```

## Testing

Run the test suite:

```bash
# Unit tests
go test ./internal/validation -v

# Integration tests
go test ./internal/validation -v -tags=integration

# Performance tests
go test ./internal/validation -v -run=Performance
```

## Contributing

When adding new validation rules:

1. Add the issue type to the appropriate validation function
2. Create remediation suggestions with appropriate confidence levels
3. Add comprehensive tests
4. Update documentation
5. Consider performance impact on large datasets

## License

This consistency validation system is part of the high-performance news website project.