#!/bin/bash

# Load Testing Runner Script
# This script runs various load tests for the high-performance news website

set -e

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
TEST_USERNAME="${TEST_USERNAME:-testuser}"
TEST_PASSWORD="${TEST_PASSWORD:-testpass123}"
OUTPUT_DIR="${OUTPUT_DIR:-./results}"
ENVIRONMENT="${ENVIRONMENT:-development}"
K6_VERSION="${K6_VERSION:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if k6 is installed
check_k6() {
    if ! command -v k6 &> /dev/null; then
        log_error "k6 is not installed. Please install k6 first."
        log_info "Visit: https://k6.io/docs/getting-started/installation/"
        exit 1
    fi
    
    log_success "k6 is installed: $(k6 version)"
}

# Check if server is accessible
check_server() {
    log_info "Checking server accessibility at $BASE_URL"
    
    if curl -f -s "$BASE_URL/health" > /dev/null; then
        log_success "Server is accessible"
    else
        log_error "Server is not accessible at $BASE_URL"
        log_info "Please ensure the server is running and accessible"
        exit 1
    fi
}

# Create output directory
setup_output_dir() {
    mkdir -p "$OUTPUT_DIR"
    log_info "Results will be saved to: $OUTPUT_DIR"
}

# Run performance baseline test
run_baseline_test() {
    log_info "Running performance baseline test..."
    
    k6 run \
        --env BASE_URL="$BASE_URL" \
        --env TEST_USERNAME="$TEST_USERNAME" \
        --env TEST_PASSWORD="$TEST_PASSWORD" \
        --out json="$OUTPUT_DIR/baseline-results.json" \
        --summary-export="$OUTPUT_DIR/baseline-summary.json" \
        performance-baseline.js
    
    if [ $? -eq 0 ]; then
        log_success "Baseline test completed successfully"
    else
        log_error "Baseline test failed"
        return 1
    fi
}

# Run article creation test
run_article_creation_test() {
    log_info "Running article creation test (35 articles/minute target)..."
    
    k6 run \
        --env BASE_URL="$BASE_URL" \
        --env TEST_USERNAME="$TEST_USERNAME" \
        --env TEST_PASSWORD="$TEST_PASSWORD" \
        --out json="$OUTPUT_DIR/article-creation-results.json" \
        --summary-export="$OUTPUT_DIR/article-creation-summary.json" \
        article-creation-test.js
    
    if [ $? -eq 0 ]; then
        log_success "Article creation test completed successfully"
    else
        log_error "Article creation test failed"
        return 1
    fi
}

# Run database bottleneck test
run_database_test() {
    log_info "Running database bottleneck test..."
    
    k6 run \
        --env BASE_URL="$BASE_URL" \
        --env TEST_USERNAME="$TEST_USERNAME" \
        --env TEST_PASSWORD="$TEST_PASSWORD" \
        --env K6_SCENARIO="query_performance" \
        --out json="$OUTPUT_DIR/database-results.json" \
        --summary-export="$OUTPUT_DIR/database-summary.json" \
        database-bottleneck-test.js
    
    if [ $? -eq 0 ]; then
        log_success "Database test completed successfully"
    else
        log_error "Database test failed"
        return 1
    fi
}

# Run comprehensive load test
run_comprehensive_test() {
    log_info "Running comprehensive load test (100 concurrent users)..."
    
    k6 run \
        --env BASE_URL="$BASE_URL" \
        --env TEST_USERNAME="$TEST_USERNAME" \
        --env TEST_PASSWORD="$TEST_PASSWORD" \
        --out json="$OUTPUT_DIR/comprehensive-results.json" \
        --summary-export="$OUTPUT_DIR/comprehensive-summary.json" \
        k6-setup.js
    
    if [ $? -eq 0 ]; then
        log_success "Comprehensive test completed successfully"
    else
        log_error "Comprehensive test failed"
        return 1
    fi
}

# Establish performance baseline
establish_baseline() {
    log_info "Establishing performance baseline..."
    
    k6 run \
        --env BASE_URL="$BASE_URL" \
        --env TEST_USERNAME="$TEST_USERNAME" \
        --env TEST_PASSWORD="$TEST_PASSWORD" \
        --env ENVIRONMENT="$ENVIRONMENT" \
        --out json="$OUTPUT_DIR/baseline-establishment.json" \
        establish-baseline.js
    
    if [ $? -eq 0 ]; then
        log_success "Baseline establishment completed"
        log_info "Copy the baseline environment variables from the output above"
    else
        log_error "Baseline establishment failed"
        return 1
    fi
}

# Run performance regression test
run_regression_test() {
    log_info "Running performance regression detection test..."
    
    # Check if baseline exists
    if [[ -z "$BASELINE_HOMEPAGE_P95" ]]; then
        log_warning "No baseline metrics found. Establishing baseline first..."
        establish_baseline
        if [ $? -ne 0 ]; then
            return 1
        fi
        log_info "Baseline established. Please set the baseline environment variables and run again."
        log_info "Example:"
        log_info "export BASELINE_HOMEPAGE_P95=500"
        log_info "export BASELINE_API_P95=100"
        log_info "export BASELINE_ARTICLE_CREATION_P95=1000"
        log_info "export BASELINE_DB_QUERY_P95=10"
        log_info "export BASELINE_CACHE_HIT_RATE=0.8"
        log_info "export BASELINE_ERROR_RATE=0.02"
        return 0
    fi
    
    k6 run \
        --env BASE_URL="$BASE_URL" \
        --env TEST_USERNAME="$TEST_USERNAME" \
        --env TEST_PASSWORD="$TEST_PASSWORD" \
        --env BASELINE_HOMEPAGE_P95="$BASELINE_HOMEPAGE_P95" \
        --env BASELINE_API_P95="$BASELINE_API_P95" \
        --env BASELINE_ARTICLE_CREATION_P95="$BASELINE_ARTICLE_CREATION_P95" \
        --env BASELINE_DB_QUERY_P95="$BASELINE_DB_QUERY_P95" \
        --env BASELINE_CACHE_HIT_RATE="$BASELINE_CACHE_HIT_RATE" \
        --env BASELINE_ERROR_RATE="$BASELINE_ERROR_RATE" \
        --env ENVIRONMENT="$ENVIRONMENT" \
        --env BUILD_ID="$BUILD_ID" \
        --env COMMIT_HASH="$COMMIT_HASH" \
        --env BRANCH="$BRANCH" \
        --out json="$OUTPUT_DIR/regression-results.json" \
        --summary-export="$OUTPUT_DIR/regression-summary.json" \
        integrated-regression-test.js
    
    if [ $? -eq 0 ]; then
        log_success "Performance regression test completed - no regressions detected"
        generate_regression_report
    else
        log_error "Performance regression test failed - regressions detected!"
        generate_regression_report
        return 1
    fi
}

# Generate regression test report
generate_regression_report() {
    log_info "Generating regression test report..."
    
    cat > "$OUTPUT_DIR/regression-report.md" << EOF
# Performance Regression Test Report

Generated on: $(date)
Server: $BASE_URL
Environment: $ENVIRONMENT
Build ID: $BUILD_ID
Commit: $COMMIT_HASH
Branch: $BRANCH

## Regression Test Results

### Test Configuration
- **Baseline Homepage P95**: ${BASELINE_HOMEPAGE_P95}ms
- **Baseline API P95**: ${BASELINE_API_P95}ms  
- **Baseline Article Creation P95**: ${BASELINE_ARTICLE_CREATION_P95}ms
- **Baseline DB Query P95**: ${BASELINE_DB_QUERY_P95}ms
- **Baseline Cache Hit Rate**: $BASELINE_CACHE_HIT_RATE
- **Baseline Error Rate**: $BASELINE_ERROR_RATE

### Results Files
- **Regression Results**: regression-results.json
- **Regression Summary**: regression-summary.json

Check the test output above for detailed regression analysis and recommendations.

EOF
    
    log_success "Regression test report generated: $OUTPUT_DIR/regression-report.md"
}

# Generate test report
generate_report() {
    log_info "Generating test report..."
    
    cat > "$OUTPUT_DIR/test-report.md" << EOF
# Load Testing Report

Generated on: $(date)
Server: $BASE_URL
Test Duration: Various scenarios

## Test Results Summary

### Performance Baseline
- **File**: baseline-results.json
- **Purpose**: Establish performance baselines for database operations and API responses
- **Key Metrics**: Database connection time, cache hit rate, query execution time

### Article Creation Test
- **File**: article-creation-results.json  
- **Purpose**: Test article creation at 35 articles/minute rate (50K daily target)
- **Key Metrics**: Article creation success rate, creation duration, database insert time

### Database Bottleneck Test
- **File**: database-results.json
- **Purpose**: Identify bottlenecks in database queries and connection handling
- **Key Metrics**: Connection pool utilization, query execution time, slow query count

### Comprehensive Load Test
- **File**: comprehensive-results.json
- **Purpose**: Test overall system performance with 100 concurrent users
- **Key Metrics**: Response times, error rates, throughput

## Performance Requirements (from Requirement 22)

| Metric | Target | Status |
|--------|--------|--------|
| Article publishing | < 1 second | Check results |
| Homepage (cached) | < 500ms | Check results |
| Homepage (dynamic) | < 2 seconds | Check results |
| Search queries | < 200ms | Check results |
| API requests | < 100ms | Check results |
| Database queries | < 10ms | Check results |
| Static files | < 50ms | Check results |
| Concurrent users | 10,000+ | Check results |
| Daily articles | 50,000+ | Check results |
| Peak publishing | 1000/minute | Check results |

## Files Generated

EOF

    # List all generated files
    for file in "$OUTPUT_DIR"/*.json; do
        if [ -f "$file" ]; then
            echo "- $(basename "$file")" >> "$OUTPUT_DIR/test-report.md"
        fi
    done
    
    log_success "Test report generated: $OUTPUT_DIR/test-report.md"
}

# Main execution
main() {
    log_info "Starting load testing framework for high-performance news website"
    log_info "Target: 50K daily articles, 100 concurrent users, sub-2s response times"
    
    # Pre-flight checks
    check_k6
    check_server
    setup_output_dir
    
    # Run tests based on arguments
    case "${1:-regression}" in
        "regression")
            run_regression_test
            ;;
        "establish-baseline")
            establish_baseline
            ;;
        "baseline")
            run_baseline_test
            ;;
        "articles")
            run_article_creation_test
            ;;
        "database")
            run_database_test
            ;;
        "comprehensive")
            run_comprehensive_test
            ;;
        "all")
            log_info "Running all traditional test scenarios..."
            run_baseline_test && \
            run_article_creation_test && \
            run_database_test && \
            run_comprehensive_test
            ;;
        *)
            log_error "Unknown test scenario: $1"
            log_info "Usage: $0 [regression|establish-baseline|baseline|articles|database|comprehensive|all]"
            exit 1
            ;;
    esac
    
    # Generate report if any tests were run
    if [ $? -eq 0 ]; then
        generate_report
        log_success "All tests completed successfully!"
        log_info "Check results in: $OUTPUT_DIR"
    else
        log_error "Some tests failed. Check the output above for details."
        exit 1
    fi
}

# Help function
show_help() {
    cat << EOF
Load Testing Framework for High-Performance News Website

Usage: $0 [OPTION] [TEST_SCENARIO]

Test Scenarios:
  regression    - Run performance regression detection test (default)
  establish-baseline - Establish new performance baseline
  baseline      - Run performance baseline measurements
  articles      - Test article creation at 35/minute rate
  database      - Test database bottlenecks and connection handling
  comprehensive - Run comprehensive load test with 100 concurrent users
  all           - Run all traditional test scenarios

Options:
  -h, --help    - Show this help message

Environment Variables:
  BASE_URL      - Server URL (default: http://localhost:8080)
  TEST_USERNAME - Test user username (default: testuser)
  TEST_PASSWORD - Test user password (default: testpass123)
  OUTPUT_DIR    - Results output directory (default: ./results)
  ENVIRONMENT   - Environment name (default: development)

Baseline Variables (for regression testing):
  BASELINE_HOMEPAGE_P95        - Homepage 95th percentile (ms)
  BASELINE_API_P95             - API 95th percentile (ms)
  BASELINE_ARTICLE_CREATION_P95 - Article creation 95th percentile (ms)
  BASELINE_DB_QUERY_P95        - Database query 95th percentile (ms)
  BASELINE_CACHE_HIT_RATE      - Cache hit rate (0.0-1.0)
  BASELINE_ERROR_RATE          - Error rate (0.0-1.0)

CI/CD Variables:
  BUILD_ID      - Build identifier
  COMMIT_HASH   - Git commit hash
  BRANCH        - Git branch name

Examples:
  $0                          # Run all tests
  $0 baseline                 # Run only baseline test
  $0 articles                 # Run only article creation test
  BASE_URL=https://example.com $0 comprehensive  # Test remote server

Requirements:
  - k6 load testing tool installed
  - Server running and accessible
  - Test user account configured
  - Sufficient system resources for load generation

Performance Targets (Requirement 22):
  - Article publishing: < 1 second
  - Homepage (cached): < 500ms  
  - Homepage (dynamic): < 2 seconds
  - Search queries: < 200ms
  - API requests: < 100ms
  - Database queries: < 10ms
  - Support 10,000+ concurrent users
  - Handle 50,000+ articles per day
  - Peak publishing: 1000 articles/minute

EOF
}

# Handle help option
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    show_help
    exit 0
fi

# Run main function
main "$@"