# Article Repository Testing Guide

This guide explains how to test the article repository implementation to verify it meets all requirements.

## Quick Validation Test

First, run the simple validation script to test core components:

```bash
go run test_article_repository.go
```

This will test:
- ✅ Article model validation
- ✅ Slug generation and validation  
- ✅ SEO data validation
- ✅ Cache key generation patterns

## Unit Tests

Run the comprehensive unit test suite:

```bash
# Run all repository unit tests
go test ./internal/repositories -v

# Run only article repository tests
go test ./internal/repositories -v -run "TestArticleRepository"

# Run with coverage report
go test ./internal/repositories -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Unit Test Coverage

The unit tests cover:
- ✅ **CRUD Operations**: Create, Read, Update, Delete validation
- ✅ **Cache Integration**: Cache hit/miss scenarios with mocks
- ✅ **Error Handling**: Validation errors, database errors, cache failures
- ✅ **Slug Generation**: Various title formats and edge cases
- ✅ **SEO Validation**: Meta titles, descriptions, URLs, schema types
- ✅ **Static Fallback**: File-based fallback mechanism
- ✅ **Performance**: Benchmarks for validation and key generation

## Integration Tests

Integration tests require a real database and cache. Set up the environment first:

### Prerequisites

1. **PostgreSQL Database**:
   ```bash
   # Start PostgreSQL (using Docker)
   docker-compose up -d postgres
   
   # Or use existing PostgreSQL instance
   # Create test database: news_test
   ```

2. **Cache Service** (DragonflyDB or Redis):
   ```bash
   # Start DragonflyDB (using Docker)  
   docker-compose up -d dragonfly
   
   # Or use Redis
   docker-compose up -d redis
   ```

### Run Integration Tests

```bash
# Set environment variables
export INTEGRATION_TEST=1
export DB_HOST=localhost
export DB_USER=postgres
export DB_PASSWORD=password
export DB_NAME=news_test
export CACHE_HOST=localhost

# Run integration tests
go test ./internal/repositories -v -run "Integration"

# Run with race detection
go test ./internal/repositories -v -race -run "Integration"
```

### Integration Test Coverage

The integration tests verify:
- ✅ **Database Operations**: Real CRUD operations with PostgreSQL
- ✅ **Prepared Statements**: All queries use prepared statements
- ✅ **Cache Integration**: Real cache operations with DragonflyDB
- ✅ **Bulk Operations**: PostgreSQL COPY for high-volume inserts
- ✅ **Graceful Degradation**: Cache → Database → Static file fallback
- ✅ **Performance**: Sub-second article creation, sub-100ms cached retrieval
- ✅ **Concurrency**: Race condition testing
- ✅ **Analytics**: View tracking and trending algorithms

## Performance Benchmarks

Run performance benchmarks to verify requirements compliance:

```bash
# Run all benchmarks
go test ./internal/repositories -bench=. -benchmem

# Run specific benchmarks
go test ./internal/repositories -bench=BenchmarkArticleRepository -benchmem

# Run integration benchmarks (requires database)
INTEGRATION_TEST=1 go test ./internal/repositories -bench=Integration -benchmem
```

### Performance Targets

The benchmarks verify these requirements:

| Operation | Target | Requirement |
|-----------|--------|-------------|
| Article Creation | < 1 second | Req 1, 22 |
| Cached Retrieval | < 100ms | Req 22 |
| Database Retrieval | < 500ms | Req 22 |
| Bulk Insert (1000 articles) | < 60 seconds | Req 1.5 |
| Validation | < 1ms | Performance |
| Slug Generation | < 100μs | Performance |

## Manual Testing Scenarios

### Scenario 1: Basic CRUD Operations

```bash
# 1. Start services
docker-compose up -d

# 2. Run database migrations
make migrate-up

# 3. Test article creation
go run cmd/server/main.go &

# 4. Use curl or Postman to test endpoints
curl -X POST http://localhost:8080/api/articles \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Article",
    "content": "Test content...",
    "author_id": 1,
    "category_id": 1,
    "status": "published"
  }'
```

### Scenario 2: High-Volume Testing

```bash
# Test bulk article creation
go run scripts/bulk_test.go  # Create this script to test 1000+ articles

# Monitor performance
go run scripts/performance_monitor.go  # Monitor database and cache metrics
```

### Scenario 3: Graceful Degradation

```bash
# 1. Start with all services
docker-compose up -d

# 2. Test normal operation (cache + database)
curl http://localhost:8080/api/articles/test-article

# 3. Stop cache service
docker-compose stop dragonfly

# 4. Test database fallback
curl http://localhost:8080/api/articles/test-article

# 5. Stop database
docker-compose stop postgres

# 6. Test static file fallback (if configured)
curl http://localhost:8080/api/articles/test-article
```

## Test Data Setup

### Sample Articles for Testing

```sql
-- Insert test users
INSERT INTO users (username, email, password_hash, role) VALUES
('admin', 'admin@example.com', 'hashed_password', 'admin'),
('editor', 'editor@example.com', 'hashed_password', 'editor'),
('reporter', 'reporter@example.com', 'hashed_password', 'reporter');

-- Insert test categories
INSERT INTO categories (name, slug, description) VALUES
('Technology', 'technology', 'Technology news and articles'),
('Sports', 'sports', 'Sports news and updates'),
('Politics', 'politics', 'Political news and analysis');

-- Insert test articles (use the repository for this)
```

## Troubleshooting

### Common Issues

1. **Database Connection Errors**:
   ```bash
   # Check PostgreSQL is running
   docker-compose ps postgres
   
   # Check connection
   psql -h localhost -U postgres -d news_test
   ```

2. **Cache Connection Errors**:
   ```bash
   # Check DragonflyDB is running
   docker-compose ps dragonfly
   
   # Test connection
   redis-cli -h localhost -p 6379 ping
   ```

3. **Test Failures**:
   ```bash
   # Run with verbose output
   go test ./internal/repositories -v -failfast
   
   # Check specific test
   go test ./internal/repositories -v -run "TestSpecificFunction"
   ```

4. **Performance Issues**:
   ```bash
   # Profile CPU usage
   go test ./internal/repositories -cpuprofile=cpu.prof -bench=.
   go tool pprof cpu.prof
   
   # Profile memory usage  
   go test ./internal/repositories -memprofile=mem.prof -bench=.
   go tool pprof mem.prof
   ```

## Continuous Integration

For CI/CD pipelines, use this test sequence:

```bash
#!/bin/bash
# ci_test.sh

set -e

echo "🧪 Running Article Repository Tests..."

# 1. Unit tests (no external dependencies)
echo "Running unit tests..."
go test ./internal/repositories -v -short

# 2. Integration tests (requires services)
if [ "$CI_INTEGRATION_TESTS" = "true" ]; then
    echo "Starting test services..."
    docker-compose -f docker-compose.test.yml up -d
    
    echo "Waiting for services..."
    sleep 10
    
    echo "Running integration tests..."
    INTEGRATION_TEST=1 go test ./internal/repositories -v
    
    echo "Cleaning up..."
    docker-compose -f docker-compose.test.yml down
fi

# 3. Performance benchmarks
echo "Running benchmarks..."
go test ./internal/repositories -bench=. -benchmem -short

echo "✅ All tests passed!"
```

## Success Criteria

The article repository implementation is considered successful when:

- ✅ All unit tests pass (100% coverage for critical paths)
- ✅ All integration tests pass with real database/cache
- ✅ Performance benchmarks meet requirements (< 1s creation, < 100ms retrieval)
- ✅ Bulk operations handle 1000+ articles efficiently
- ✅ Graceful degradation works through all fallback layers
- ✅ No race conditions under concurrent load
- ✅ Memory usage remains stable under load
- ✅ Error handling covers all edge cases

Run the complete test suite to verify all requirements are met!