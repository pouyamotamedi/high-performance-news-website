# Load Testing Framework

This directory contains a comprehensive load testing framework for the high-performance news website, designed to test the system's ability to handle 50,000+ daily articles with 100+ concurrent users while maintaining sub-2-second response times.

## Overview

The load testing framework validates the performance requirements specified in Requirement 22:

- **Article publishing**: < 1 second
- **Homepage (cached)**: < 500ms
- **Homepage (dynamic)**: < 2 seconds
- **Search queries**: < 200ms
- **API requests**: < 100ms
- **Database queries**: < 10ms
- **Static files**: < 50ms
- **Concurrent users**: 10,000+ simultaneous users
- **Daily articles**: 50,000+ articles per day
- **Peak publishing**: 1000 articles per minute

## Test Scenarios

### 1. Performance Baseline (`performance-baseline.js`)
Establishes baseline performance metrics for:
- Database connection times
- Cache hit rates
- Query execution times
- System resource usage
- API response times

**Purpose**: Create performance baselines before load testing to identify regressions.

### 2. Article Creation Test (`article-creation-test.js`)
Tests article creation at target rates:
- **Normal rate**: 35 articles/minute (50K daily target)
- **Peak rate**: 70 articles/minute (double normal)
- **Burst rate**: 100 articles/minute (stress test)

**Purpose**: Validate the system can handle the required article publishing volume.

### 3. Database Bottleneck Test (`database-bottleneck-test.js`)
Identifies database performance bottlenecks:
- Connection pool stress testing (200 VUs vs 150 max connections)
- Query performance under load
- Concurrent write operations
- Partition query performance

**Purpose**: Ensure database can handle high-volume operations without degradation.

### 4. Comprehensive Load Test (`k6-setup.js`)
Mixed workload simulation:
- 100 concurrent users (normal load)
- Peak load ramping (up to 1000 users)
- Mixed read/write operations
- Real-world usage patterns

**Purpose**: Validate overall system performance under realistic load conditions.

## Prerequisites

### Required Software
1. **k6 Load Testing Tool**
   ```bash
   # Install k6 (see https://k6.io/docs/getting-started/installation/)
   # Windows (using Chocolatey)
   choco install k6
   
   # macOS (using Homebrew)
   brew install k6
   
   # Linux (using package manager)
   sudo apt-get install k6
   ```

2. **curl** (for server health checks)
   - Usually pre-installed on most systems
   - Windows: Available in Windows 10+ or install via package manager

### Server Requirements
1. **Running Application Server**
   - Server must be accessible at configured URL
   - Health endpoint (`/health`) must be available
   - API endpoints must be functional

2. **Test User Account**
   - Username: `testuser` (configurable)
   - Password: `testpass123` (configurable)
   - Must have permissions to create articles

3. **Database Setup**
   - PostgreSQL with partitioning enabled
   - Connection pooling configured (PgBouncer)
   - Test data (categories, tags, users) populated

## Usage

### Quick Start

```bash
# Linux/macOS
cd load-testing
./run-tests.sh

# Windows
cd load-testing
run-tests.bat
```

### Individual Test Scenarios

```bash
# Run only baseline test
./run-tests.sh baseline

# Run only article creation test
./run-tests.sh articles

# Run only database bottleneck test
./run-tests.sh database

# Run only comprehensive load test
./run-tests.sh comprehensive
```

### Custom Configuration

```bash
# Test against different server
export BASE_URL=https://staging.example.com
./run-tests.sh

# Use different credentials
export TEST_USERNAME=admin
export TEST_PASSWORD=admin123
./run-tests.sh

# Custom output directory
export OUTPUT_DIR=/path/to/results
./run-tests.sh
```

### Windows Examples

```cmd
REM Test against different server
set BASE_URL=https://staging.example.com
run-tests.bat

REM Use different credentials
set TEST_USERNAME=admin
set TEST_PASSWORD=admin123
run-tests.bat
```

## Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8080` | Server URL to test |
| `TEST_USERNAME` | `testuser` | Test user username |
| `TEST_PASSWORD` | `testpass123` | Test user password |
| `OUTPUT_DIR` | `./results` | Results output directory |

### Test Parameters

Each test script includes configurable parameters:

- **Virtual Users (VUs)**: Number of concurrent users
- **Duration**: Test duration
- **Rate**: Request rate (for rate-based tests)
- **Thresholds**: Performance thresholds for pass/fail

## Output and Results

### Generated Files

After running tests, the following files are created in the output directory:

- `baseline-results.json` - Detailed baseline metrics
- `baseline-summary.json` - Baseline test summary
- `article-creation-results.json` - Article creation test data
- `article-creation-summary.json` - Article creation summary
- `database-results.json` - Database performance data
- `database-summary.json` - Database test summary
- `comprehensive-results.json` - Comprehensive test data
- `comprehensive-summary.json` - Comprehensive test summary
- `test-report.md` - Human-readable test report

### Metrics Collected

#### Performance Metrics
- HTTP request duration (p95, p99)
- Database query execution time
- Cache hit rates
- Connection pool utilization
- Memory and CPU usage

#### Business Metrics
- Articles created per minute
- Article creation success rate
- Search query performance
- User session handling

#### Error Metrics
- HTTP error rates
- Database connection errors
- Timeout occurrences
- Failed operations

## Performance Thresholds

The tests include built-in thresholds based on Requirement 22:

```javascript
thresholds: {
  'http_req_duration': ['p(95)<2000'],           // 95% under 2s
  'http_req_duration{endpoint:homepage}': ['p(95)<500'],  // Homepage under 500ms
  'http_req_duration{endpoint:api}': ['p(95)<100'],       // API under 100ms
  'http_req_duration{endpoint:search}': ['p(95)<200'],    // Search under 200ms
  'article_creation_success': ['rate>0.95'],              // 95% success rate
  'database_query_duration': ['p(95)<10'],                // DB queries under 10ms
}
```

## Interpreting Results

### Success Criteria

A test passes if:
1. All HTTP requests complete successfully (< 5% error rate)
2. Response times meet performance thresholds
3. No critical system errors occur
4. Resource utilization remains reasonable

### Common Issues

#### High Response Times
- **Cause**: Database bottlenecks, insufficient caching
- **Solution**: Check database indexes, cache configuration

#### Connection Errors
- **Cause**: Connection pool exhaustion
- **Solution**: Increase pool size, optimize connection usage

#### Memory Issues
- **Cause**: Memory leaks, insufficient resources
- **Solution**: Profile application, increase server resources

## Automated Performance Regression Testing

### CI/CD Integration

The load testing framework can be integrated into CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
name: Performance Tests
on: [push, pull_request]

jobs:
  performance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Install k6
        run: |
          sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      - name: Run baseline tests
        run: |
          cd load-testing
          ./run-tests.sh baseline
      - name: Upload results
        uses: actions/upload-artifact@v2
        with:
          name: performance-results
          path: load-testing/results/
```

### Performance Monitoring

Set up continuous performance monitoring:

1. **Baseline Tracking**: Run baseline tests regularly
2. **Threshold Alerts**: Alert when performance degrades
3. **Trend Analysis**: Track performance over time
4. **Capacity Planning**: Use results for scaling decisions

## Troubleshooting

### Common Setup Issues

1. **k6 not found**
   - Install k6 from https://k6.io/docs/getting-started/installation/

2. **Server not accessible**
   - Verify server is running: `curl http://localhost:8080/health`
   - Check firewall settings
   - Verify URL configuration

3. **Authentication failures**
   - Verify test user exists in database
   - Check username/password configuration
   - Ensure user has required permissions

4. **Database connection issues**
   - Verify database is running
   - Check connection pool configuration
   - Ensure test data exists (categories, tags, users)

### Performance Issues

1. **Tests timing out**
   - Reduce concurrent users
   - Increase test duration
   - Check server resources

2. **High error rates**
   - Check server logs
   - Verify database connectivity
   - Monitor system resources

3. **Inconsistent results**
   - Run tests multiple times
   - Check for background processes
   - Ensure stable test environment

## Contributing

When adding new tests:

1. Follow existing naming conventions
2. Include proper error handling
3. Add meaningful metrics
4. Document test purpose and thresholds
5. Update this README with new test information

## Support

For issues with the load testing framework:

1. Check server logs for errors
2. Verify all prerequisites are met
3. Run individual test scenarios to isolate issues
4. Review generated test reports for detailed metrics