# Database Package

This package provides a comprehensive database management system for the high-performance news website, designed to handle 50,000+ daily articles with optimal performance.

## Features

- **High-Performance Connection Pool**: Optimized for 150 max connections with 40 idle connections
- **PgBouncer Integration**: Transaction-mode connection pooling for production environments
- **Prepared Statements**: Pre-compiled SQL statements for maximum performance
- **Migration System**: Up/down migrations with golang-migrate
- **Automatic Partitioning**: Daily and monthly partitions with BRIN indexes
- **Partition Management**: Automated partition creation and cleanup
- **Comprehensive Testing**: Unit, integration, and benchmark tests

## Architecture

### Components

1. **Connection Manager** (`connection.go`): Handles database connections and prepared statements
2. **Migration System** (`migrate.go`): Database schema versioning and migrations
3. **Partition Manager** (`partition.go`): Automated partition management for high-volume data
4. **Database Manager** (`manager.go`): Orchestrates all database components

### Database Schema

The system uses PostgreSQL 15+ with the following key features:

- **Partitioned Tables**: Articles, article_tags, article_views, article_engagement
- **BRIN Indexes**: Optimized for time-series data with 50K+ daily inserts
- **Composite Indexes**: Multi-column indexes for complex queries
- **Full-Text Search**: PostgreSQL trgm extension for search functionality

## Usage

### Basic Setup

```go
import (
    "high-performance-news-website/internal/config"
    "high-performance-news-website/pkg/database"
)

// Load configuration
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Create database manager
manager, err := database.NewManager(&cfg.Database, "./migrations")
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

// Initialize database (run migrations and create partitions)
if err := manager.Initialize(); err != nil {
    log.Fatal(err)
}
```

### Direct Connection

```go
// Direct PostgreSQL connection
db, err := database.NewConnection(&cfg.Database)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// PgBouncer connection (recommended for production)
db, err := database.NewPgBouncerConnection(&cfg.Database)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### Using Prepared Statements

```go
// Get a prepared statement
stmt, err := db.GetPreparedStatement(database.StmtGetArticle)
if err != nil {
    log.Fatal(err)
}

// Execute the prepared statement
var article Article
err = stmt.QueryRow("article-slug").Scan(
    &article.ID, &article.Title, &article.Content, // ... other fields
)
```

### Migration Management

```go
// Run all migrations up
err := manager.Migrate("up", 0)

// Run specific number of migrations
err := manager.Migrate("up", 2)

// Rollback migrations
err := manager.Migrate("down", 1)

// Force migration version (use with caution)
err := manager.ForceVersion(1)
```

### Partition Management

```go
// Create daily partitions for next 7 days
err := manager.CreateDailyPartitions()

// Drop partitions older than 30 days
err := manager.DropOldPartitions(30)

// Run full maintenance (creates partitions + drops old ones)
err := manager.RunMaintenance()

// Get partition information
partitions, err := manager.PartitionManager.GetPartitionInfo()
```

## Configuration

### Database Configuration

```yaml
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: news_website
  sslmode: disable
  max_conns: 150        # Maximum open connections
  min_conns: 40         # Minimum idle connections
  use_pgbouncer: false  # Enable PgBouncer
  pgbouncer_port: 6432  # PgBouncer port
```

### PgBouncer Configuration

For production environments, use PgBouncer with transaction mode:

```ini
[databases]
news_website = host=postgres port=5432 dbname=news_website user=postgres password=postgres

[pgbouncer]
pool_mode = transaction
max_client_conn = 200
default_pool_size = 50
min_pool_size = 10
reserve_pool_size = 5
server_lifetime = 3600
server_idle_timeout = 600
```

## Performance Optimizations

### Connection Pool Settings

- **Max Connections**: 150 (leaves 50 for maintenance)
- **Idle Connections**: 40 (balance between performance and resources)
- **Connection Lifetime**: 1 hour (prevents stale connections)
- **Idle Timeout**: 10 minutes (cleanup unused connections)

### Partitioning Strategy

- **Articles Table**: Partitioned by `published_at` (daily/monthly)
- **Analytics Tables**: Partitioned by `created_at` (monthly)
- **BRIN Indexes**: Used for time-series columns (pages_per_range optimized)
- **Composite Indexes**: Multi-column indexes for common query patterns

### Prepared Statements

Pre-compiled statements for frequently used queries:

- `StmtGetArticle`: Retrieve article by slug
- `StmtGetHomepage`: Get latest articles for homepage
- `StmtGetCategory`: Get articles by category with pagination
- `StmtInsertView`: Record article view for analytics
- `StmtInsertArticle`: Insert new article
- `StmtUpdateArticle`: Update existing article

## Testing

### Running Tests

```bash
# Run all database tests
go test ./pkg/database -v

# Run specific test
go test ./pkg/database -v -run TestNewConnection

# Run benchmarks
go test ./pkg/database -v -bench=.

# Run integration tests (requires PostgreSQL)
go test ./pkg/database -v -run TestDatabaseIntegration
```

### Test Database CLI

Use the included CLI tool for testing:

```bash
# Test database connection
go run cmd/dbtest/main.go -action=test

# Initialize database
go run cmd/dbtest/main.go -action=init

# Run migrations
go run cmd/dbtest/main.go -action=migrate -migrate-dir=up

# Manage partitions
go run cmd/dbtest/main.go -action=partition -retention=30

# Show database stats
go run cmd/dbtest/main.go -action=stats

# Test with PgBouncer
go run cmd/dbtest/main.go -action=test -pgbouncer=true
```

## Monitoring and Health Checks

### Health Check

```go
// Check database health
if err := manager.Health(); err != nil {
    log.Printf("Database health check failed: %v", err)
}
```

### Statistics

```go
// Get comprehensive database statistics
stats := manager.GetStats()
fmt.Printf("Migration Version: %d\n", stats.MigrationVersion)
fmt.Printf("Partition Count: %d\n", stats.PartitionCount)
fmt.Printf("Using PgBouncer: %t\n", stats.UsePgBouncer)
```

### Connection Pool Monitoring

```go
// Get connection pool statistics
dbStats := db.GetStats()
fmt.Printf("Open Connections: %d\n", dbStats.OpenConnections)
fmt.Printf("In Use: %d\n", dbStats.InUse)
fmt.Printf("Idle: %d\n", dbStats.Idle)
```

## Production Deployment

### Recommended Setup

1. **Use PgBouncer**: Enable transaction-mode connection pooling
2. **Enable Partition Scheduler**: Automatic daily partition management
3. **Monitor Connection Pool**: Track connection usage and performance
4. **Regular Maintenance**: Schedule partition cleanup and statistics updates

### Performance Targets

- **Article Creation**: < 1 second per article
- **Homepage Load**: < 500ms (cached), < 2 seconds (dynamic)
- **Database Queries**: < 10ms for indexed queries
- **Concurrent Users**: 10,000+ simultaneous connections
- **Daily Volume**: 50,000+ articles per day

### Scaling Considerations

- **Vertical Scaling**: Increase connection pool size as needed
- **Horizontal Scaling**: Read replicas for read-heavy workloads
- **Partition Management**: Adjust retention period based on storage capacity
- **Index Optimization**: Monitor and optimize indexes based on query patterns

## Troubleshooting

### Common Issues

1. **Connection Pool Exhaustion**: Increase `max_conns` or enable PgBouncer
2. **Slow Queries**: Check partition pruning and index usage
3. **Migration Failures**: Use `ForceVersion()` to recover from dirty state
4. **Partition Errors**: Ensure proper date ranges and no overlapping partitions

### Debug Mode

Enable detailed logging for troubleshooting:

```go
// Enable query logging in PostgreSQL
// log_statement = 'all'
// log_min_duration_statement = 0
```

### Performance Analysis

```sql
-- Check slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- Check partition sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE tablename LIKE 'articles_%'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```