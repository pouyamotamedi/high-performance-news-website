#!/bin/bash

# Security Scan Automation Script
# This script runs comprehensive security scans including OWASP ZAP and Snyk

set -e

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORTS_DIR="${PROJECT_ROOT}/reports/security"
ZAP_CONFIG="${PROJECT_ROOT}/configs/owasp-zap-config.yaml"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if gosec is installed
    if ! command -v gosec &> /dev/null; then
        warning "gosec not found, installing..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi
    
    # Check if golangci-lint is installed
    if ! command -v golangci-lint &> /dev/null; then
        warning "golangci-lint not found, installing..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2
    fi
    
    # Check if Snyk is installed (optional)
    if ! command -v snyk &> /dev/null; then
        warning "Snyk CLI not found. Install with: npm install -g snyk"
        warning "Dependency vulnerability scanning will be skipped"
    fi
    
    success "Prerequisites check completed"
}

# Create reports directory
setup_reports_dir() {
    log "Setting up reports directory..."
    mkdir -p "${REPORTS_DIR}"
    mkdir -p "${REPORTS_DIR}/gosec"
    mkdir -p "${REPORTS_DIR}/snyk"
    mkdir -p "${REPORTS_DIR}/zap"
    mkdir -p "${REPORTS_DIR}/combined"
    success "Reports directory created: ${REPORTS_DIR}"
}

# Run static security analysis with gosec
run_gosec_scan() {
    log "Running gosec static security analysis..."
    
    cd "${PROJECT_ROOT}"
    
    # Run gosec with JSON output
    local gosec_report="${REPORTS_DIR}/gosec/gosec-report-${TIMESTAMP}.json"
    local gosec_html="${REPORTS_DIR}/gosec/gosec-report-${TIMESTAMP}.html"
    
    if gosec -fmt json -out "${gosec_report}" ./...; then
        success "gosec scan completed successfully"
    else
        warning "gosec found security issues (exit code $?)"
    fi
    
    # Generate HTML report
    if gosec -fmt html -out "${gosec_html}" ./...; then
        success "gosec HTML report generated: ${gosec_html}"
    else
        warning "Failed to generate gosec HTML report"
    fi
    
    # Run golangci-lint with security focus
    log "Running golangci-lint with security rules..."
    local golangci_report="${REPORTS_DIR}/gosec/golangci-security-${TIMESTAMP}.json"
    
    if golangci-lint run --out-format json --issues-exit-code 0 > "${golangci_report}"; then
        success "golangci-lint security scan completed"
    else
        warning "golangci-lint found issues"
    fi
}

# Run dependency vulnerability scanning with Snyk
run_snyk_scan() {
    if ! command -v snyk &> /dev/null; then
        warning "Snyk CLI not available, skipping dependency vulnerability scan"
        return 0
    fi
    
    log "Running Snyk dependency vulnerability scan..."
    
    cd "${PROJECT_ROOT}"
    
    # Authenticate Snyk if token is available
    if [[ -n "${SNYK_TOKEN}" ]]; then
        snyk auth "${SNYK_TOKEN}"
    else
        warning "SNYK_TOKEN not set, using existing authentication"
    fi
    
    local snyk_report="${REPORTS_DIR}/snyk/snyk-report-${TIMESTAMP}.json"
    local snyk_html="${REPORTS_DIR}/snyk/snyk-report-${TIMESTAMP}.html"
    
    # Run Snyk test
    if snyk test --json > "${snyk_report}" 2>/dev/null; then
        success "Snyk scan completed - no vulnerabilities found"
    else
        local exit_code=$?
        if [[ $exit_code -eq 1 ]]; then
            warning "Snyk found vulnerabilities"
        else
            error "Snyk scan failed with exit code $exit_code"
        fi
    fi
    
    # Generate HTML report
    if snyk test --json | snyk-to-html -o "${snyk_html}" 2>/dev/null; then
        success "Snyk HTML report generated: ${snyk_html}"
    else
        warning "Failed to generate Snyk HTML report (snyk-to-html may not be installed)"
    fi
    
    # Run Snyk monitor for continuous monitoring (if configured)
    if [[ "${SNYK_MONITOR:-false}" == "true" ]]; then
        log "Running Snyk monitor for continuous monitoring..."
        if snyk monitor; then
            success "Project added to Snyk monitoring"
        else
            warning "Failed to add project to Snyk monitoring"
        fi
    fi
}

# Run OWASP ZAP scan (if ZAP is running)
run_zap_scan() {
    log "Checking if OWASP ZAP is available..."
    
    local zap_url="${ZAP_API_URL:-http://localhost:8080}"
    local zap_api_key="${ZAP_API_KEY}"
    
    if [[ -z "${zap_api_key}" ]]; then
        warning "ZAP_API_KEY not set, skipping OWASP ZAP scan"
        return 0
    fi
    
    # Check if ZAP is running
    if ! curl -s "${zap_url}/JSON/core/view/version/?apikey=${zap_api_key}" > /dev/null; then
        warning "OWASP ZAP is not running at ${zap_url}, skipping dynamic security scan"
        return 0
    fi
    
    log "Running OWASP ZAP dynamic security scan..."
    
    local target_url="${TARGET_URL:-http://localhost:8080}"
    local zap_report="${REPORTS_DIR}/zap/zap-report-${TIMESTAMP}"
    
    # Create new ZAP session
    local session_name="security-scan-${TIMESTAMP}"
    curl -s "${zap_url}/JSON/core/action/newSession/?apikey=${zap_api_key}&name=${session_name}" > /dev/null
    
    # Spider the target
    log "Spidering target application..."
    local spider_id=$(curl -s "${zap_url}/JSON/spider/action/scan/?apikey=${zap_api_key}&url=${target_url}" | jq -r '.scan')
    
    # Wait for spider to complete
    while true; do
        local spider_status=$(curl -s "${zap_url}/JSON/spider/view/status/?apikey=${zap_api_key}&scanId=${spider_id}" | jq -r '.status')
        if [[ "${spider_status}" == "100" ]]; then
            break
        fi
        log "Spider progress: ${spider_status}%"
        sleep 5
    done
    success "Spidering completed"
    
    # Run active scan
    log "Running active security scan..."
    local scan_id=$(curl -s "${zap_url}/JSON/ascan/action/scan/?apikey=${zap_api_key}&url=${target_url}" | jq -r '.scan')
    
    # Wait for active scan to complete
    while true; do
        local scan_status=$(curl -s "${zap_url}/JSON/ascan/view/status/?apikey=${zap_api_key}&scanId=${scan_id}" | jq -r '.status')
        if [[ "${scan_status}" == "100" ]]; then
            break
        fi
        log "Active scan progress: ${scan_status}%"
        sleep 10
    done
    success "Active scan completed"
    
    # Generate reports
    log "Generating ZAP reports..."
    
    # JSON report
    curl -s "${zap_url}/JSON/core/view/alerts/?apikey=${zap_api_key}" > "${zap_report}.json"
    
    # HTML report
    curl -s "${zap_url}/OTHER/core/other/htmlreport/?apikey=${zap_api_key}" > "${zap_report}.html"
    
    # XML report
    curl -s "${zap_url}/OTHER/core/other/xmlreport/?apikey=${zap_api_key}" > "${zap_report}.xml"
    
    success "ZAP reports generated: ${zap_report}.*"
}

# Generate combined security report
generate_combined_report() {
    log "Generating combined security report..."
    
    local combined_report="${REPORTS_DIR}/combined/security-report-${TIMESTAMP}.json"
    local combined_html="${REPORTS_DIR}/combined/security-report-${TIMESTAMP}.html"
    
    # Create combined JSON report structure
    cat > "${combined_report}" << EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "scan_type": "comprehensive_security_scan",
  "reports": {
    "gosec": "${REPORTS_DIR}/gosec/gosec-report-${TIMESTAMP}.json",
    "snyk": "${REPORTS_DIR}/snyk/snyk-report-${TIMESTAMP}.json",
    "zap": "${REPORTS_DIR}/zap/zap-report-${TIMESTAMP}.json"
  },
  "summary": {
    "total_scans": 3,
    "completed_scans": 0,
    "failed_scans": 0
  }
}
EOF
    
    # Generate combined HTML report
    cat > "${combined_html}" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Comprehensive Security Scan Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .critical { border-left: 5px solid #d32f2f; }
        .high { border-left: 5px solid #f57c00; }
        .medium { border-left: 5px solid #fbc02d; }
        .low { border-left: 5px solid #388e3c; }
        .info { border-left: 5px solid #1976d2; }
        .report-link { display: inline-block; margin: 5px; padding: 10px; background-color: #e3f2fd; border-radius: 3px; text-decoration: none; }
        .report-link:hover { background-color: #bbdefb; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Comprehensive Security Scan Report</h1>
        <p><strong>Timestamp:</strong> TIMESTAMP_PLACEHOLDER</p>
        <p><strong>Scan Type:</strong> Comprehensive Security Analysis</p>
    </div>
    
    <div class="section">
        <h2>Individual Reports</h2>
        <a href="gosec-report-TIMESTAMP_PLACEHOLDER.html" class="report-link">📊 Static Analysis (gosec)</a>
        <a href="snyk-report-TIMESTAMP_PLACEHOLDER.html" class="report-link">🔍 Dependency Scan (Snyk)</a>
        <a href="zap-report-TIMESTAMP_PLACEHOLDER.html" class="report-link">🕷️ Dynamic Analysis (ZAP)</a>
    </div>
    
    <div class="section">
        <h2>Scan Summary</h2>
        <p>This comprehensive security scan includes:</p>
        <ul>
            <li><strong>Static Code Analysis:</strong> gosec and golangci-lint security rules</li>
            <li><strong>Dependency Vulnerability Scanning:</strong> Snyk analysis of Go modules</li>
            <li><strong>Dynamic Application Security Testing:</strong> OWASP ZAP web application scan</li>
        </ul>
    </div>
    
    <div class="section">
        <h2>Next Steps</h2>
        <ol>
            <li>Review individual reports for detailed findings</li>
            <li>Prioritize critical and high-severity issues</li>
            <li>Create remediation plan for identified vulnerabilities</li>
            <li>Re-run scans after fixes to verify resolution</li>
        </ol>
    </div>
</body>
</html>
EOF
    
    # Replace timestamp placeholder
    sed -i "s/TIMESTAMP_PLACEHOLDER/${TIMESTAMP}/g" "${combined_html}"
    
    success "Combined security report generated: ${combined_html}"
}

# Send security alerts if configured
send_security_alerts() {
    log "Checking for security alert configuration..."
    
    # Check if Go security alerting is available
    if [[ -f "${PROJECT_ROOT}/internal/testing/security_alerting.go" ]]; then
        log "Running Go-based security alerting..."
        cd "${PROJECT_ROOT}"
        
        # Build and run security alerting tool
        if go run internal/testing/security_alerting.go "${REPORTS_DIR}/combined/security-report-${TIMESTAMP}.json"; then
            success "Security alerts sent successfully"
        else
            warning "Failed to send security alerts"
        fi
    fi
    
    # Send webhook notification if configured
    if [[ -n "${SECURITY_WEBHOOK_URL}" ]]; then
        log "Sending webhook notification..."
        
        local webhook_payload=$(cat << EOF
{
    "text": "Security scan completed",
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "report_url": "${REPORTS_DIR}/combined/security-report-${TIMESTAMP}.html",
    "scan_type": "comprehensive_security_scan"
}
EOF
)
        
        if curl -X POST -H "Content-Type: application/json" -d "${webhook_payload}" "${SECURITY_WEBHOOK_URL}"; then
            success "Webhook notification sent"
        else
            warning "Failed to send webhook notification"
        fi
    fi
}

# Main execution
main() {
    log "Starting comprehensive security scan..."
    
    check_prerequisites
    setup_reports_dir
    
    # Run security scans
    run_gosec_scan
    run_snyk_scan
    run_zap_scan
    
    # Generate reports
    generate_combined_report
    
    # Send alerts
    send_security_alerts
    
    success "Comprehensive security scan completed!"
    success "Reports available in: ${REPORTS_DIR}"
    
    # Print summary
    echo ""
    echo "=== SECURITY SCAN SUMMARY ==="
    echo "Timestamp: ${TIMESTAMP}"
    echo "Reports Directory: ${REPORTS_DIR}"
    echo "Combined Report: ${REPORTS_DIR}/combined/security-report-${TIMESTAMP}.html"
    echo ""
    echo "Individual Reports:"
    echo "- Static Analysis: ${REPORTS_DIR}/gosec/"
    echo "- Dependency Scan: ${REPORTS_DIR}/snyk/"
    echo "- Dynamic Analysis: ${REPORTS_DIR}/zap/"
    echo ""
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h          Show this help message"
        echo "  --gosec-only        Run only gosec static analysis"
        echo "  --snyk-only         Run only Snyk dependency scan"
        echo "  --zap-only          Run only OWASP ZAP dynamic scan"
        echo ""
        echo "Environment Variables:"
        echo "  ZAP_API_KEY         OWASP ZAP API key"
        echo "  ZAP_API_URL         OWASP ZAP API URL (default: http://localhost:8080)"
        echo "  SNYK_TOKEN          Snyk authentication token"
        echo "  TARGET_URL          Target application URL (default: http://localhost:8080)"
        echo "  SECURITY_WEBHOOK_URL Webhook URL for notifications"
 