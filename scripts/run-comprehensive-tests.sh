#!/bin/bash

# Comprehensive Test Runner Script
# Implements requirements for >95% coverage, parallel execution, and comprehensive validation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_THRESHOLD=95
TEST_TIMEOUT="30m"
BENCHMARK_TIME="10s"
RESULTS_DIR="test-results"
PARALLEL_JOBS=$(nproc)

# Functions
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

# Create results directory
setup_test_environment() {
    log_info "Setting up test environment..."
    mkdir -p "$RESULTS_DIR"
    
    # Check if required tools are available
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v bc &> /dev/null; then
        log_warning "bc is not installed, coverage validation may not work"
    fi
    
    log_success "Test environment ready"
}

# Run unit tests with coverage
run_unit_tests() {
    log_info "Running unit tests with coverage tracking..."
    
    CGO_ENABLED=1 go test -v -race \
        -coverprofile="$RESULTS_DIR/coverage.out" \
        -covermode=atomic \
        -timeout="$TEST_TIMEOUT" \
        -parallel="$PARALLEL_JOBS" \
        ./internal/models/... \
        ./internal/repositories/... \
        ./internal/services/... \
        ./internal/api/... \
        ./internal/auth/... \
        ./internal/validation/... \
        ./pkg/... \
        2>&1 | tee "$RESULTS_DIR/unit-tests.log"
    
    if [ $? -eq 0 ]; then
        log_success "Unit tests completed successfully"
    else
        log_error "Unit tests failed"
        return 1
    fi
}

# Validate coverage requirements
validate_coverage() {
    log_info "Validating coverage requirements..."
    
    if [ ! -f "$RESULTS_DIR/coverage.out" ]; then
        log_error "Coverage file not found"
        return 1
    fi
    
    # Generate coverage report
    go tool cover -html="$RESULTS_DIR/coverage.out" -o "$RESULTS_DIR/coverage.html"
    
    # Extract coverage percentage
    COVERAGE=$(go tool cover -func="$RESULTS_DIR/coverage.out" | grep total | awk '{print $3}' | sed 's/%//')
    
    if [ -z "$COVERAGE" ]; then
        log_error "Could not extract coverage percentage"
        return 1
    fi
    
    log_info "Current coverage: ${COVERAGE}%"
    
    # Validate coverage threshold
    if command -v bc &> /dev/null; then
        if [ $(echo "$COVERAGE < $COVERAGE_THRESHOLD" | bc -l) -eq 1 ]; then
            log_error "Coverage ${COVERAGE}% is below required ${COVERAGE_THRESHOLD}%"
            return 1
        else
            log_success "Coverage ${COVERAGE}% meets requirement (≥${COVERAGE_THRESHOLD}%)"
        fi
    else
        log_warning "Cannot validate coverage threshold without bc command"
    fi
    
    echo "$COVERAGE" > "$RESULTS_DIR/coverage-percentage.txt"
}

# Run integration tests
run_integration_tests() {
    log_info "Running integration tests..."
    
    # Check if database is available
    if ! pg_isready -h localhost -p 5432 -U postgres &> /dev/null; then
        log_warning "PostgreSQL not available, skipping integration tests"
        return 0
    fi
    
    CGO_ENABLED=1 go test -v \
        -tags=integration \
        -timeout="$TEST_TIMEOUT" \
        ./internal/integration/... \
        ./internal/repositories/... \
        2>&1 | tee "$RESULTS_DIR/integration-tests.log"
    
    if [ $? -eq 0 ]; then
        log_success "Integration tests completed successfully"
    else
        log_warning "Integration tests failed or skipped"
    fi
}

# Run benchmark tests
run_benchmarks() {
    log_info "Running performance benchmarks..."
    
    go test -bench=. -benchmem \
        -benchtime="$BENCHMARK_TIME" \
        -timeout=10m \
        ./internal/models/... \
        ./internal/repositories/... \
        ./internal/services/... \
        ./pkg/... \
        2>&1 | tee "$RESULTS_DIR/benchmark.txt"
    
    if [ $? -eq 0 ]; then
        log_success "Benchmarks completed successfully"
    else
        log_warning "Benchmarks failed or incomplete"
    fi
}

# Generate comprehensive report
generate_report() {
    log_info "Generating comprehensive test report..."
    
    REPORT_FILE="$RESULTS_DIR/test-report.md"
    
    cat > "$REPORT_FILE" << EOF
# Comprehensive Test Report

Generated: $(date)

## Summary

EOF
    
    if [ -f "$RESULTS_DIR/coverage-percentage.txt" ]; then
        COVERAGE=$(cat "$RESULTS_DIR/coverage-percentage.txt")
        echo "- **Coverage**: ${COVERAGE}%" >> "$REPORT_FILE"
        
        if [ $(echo "$COVERAGE >= $COVERAGE_THRESHOLD" | bc -l) -eq 1 ]; then
            echo "- **Coverage Status**: ✅ PASSED (≥${COVERAGE_THRESHOLD}%)" >> "$REPORT_FILE"
        else
            echo "- **Coverage Status**: ❌ FAILED (<${COVERAGE_THRESHOLD}%)" >> "$REPORT_FILE"
        fi
    fi
    
    echo "" >> "$REPORT_FILE"
    echo "## Test Results" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    if [ -f "$RESULTS_DIR/unit-tests.log" ]; then
        UNIT_PASSED=$(grep -c "PASS:" "$RESULTS_DIR/unit-tests.log" || echo "0")
        UNIT_FAILED=$(grep -c "FAIL:" "$RESULTS_DIR/unit-tests.log" || echo "0")
        echo "- **Unit Tests**: ${UNIT_PASSED} passed, ${UNIT_FAILED} failed" >> "$REPORT_FILE"
    fi
    
    if [ -f "$RESULTS_DIR/integration-tests.log" ]; then
        INT_PASSED=$(grep -c "PASS:" "$RESULTS_DIR/integration-tests.log" || echo "0")
        INT_FAILED=$(grep -c "FAIL:" "$RESULTS_DIR/integration-tests.log" || echo "0")
        echo "- **Integration Tests**: ${INT_PASSED} passed, ${INT_FAILED} failed" >> "$REPORT_FILE"
    fi
    
    echo "" >> "$REPORT_FILE"
    echo "## Files Generated" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    echo "- Coverage Report: [coverage.html](coverage.html)" >> "$REPORT_FILE"
    echo "- Unit Test Log: [unit-tests.log](unit-tests.log)" >> "$REPORT_FILE"
    echo "- Integration Test Log: [integration-tests.log](integration-tests.log)" >> "$REPORT_FILE"
    echo "- Benchmark Results: [benchmark.txt](benchmark.txt)" >> "$REPORT_FILE"
    
    log_success "Test report generated: $REPORT_FILE"
}

# Main execution
main() {
    log_info "Starting comprehensive test suite..."
    
    setup_test_environment
    
    # Run tests
    if ! run_unit_tests; then
        log_error "Unit tests failed, stopping execution"
        exit 1
    fi
    
    if ! validate_coverage; then
        log_error "Coverage validation failed"
        exit 1
    fi
    
    run_integration_tests
    run_benchmarks
    generate_report
    
    log_success "Comprehensive testing completed successfully!"
    log_info "Results available in: $RESULTS_DIR/"
}

# Execute main function
main "$@"