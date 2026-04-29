#!/bin/bash

# OWASP ZAP Automation Script for Security Testing
# This script automates the setup and execution of OWASP ZAP security scans

set -e

# Configuration
ZAP_VERSION="2.14.0"
ZAP_PORT="${ZAP_PORT:-8080}"
TARGET_URL="${TARGET_URL:-http://localhost:8080}"
ZAP_API_KEY="${ZAP_API_KEY:-$(openssl rand -hex 16)}"
REPORT_DIR="./reports/security/zap"
CONFIG_FILE="./configs/owasp-zap-config.yaml"

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

# Check if ZAP is installed
check_zap_installation() {
    log_info "Checking OWASP ZAP installation..."
    
    if command -v zap.sh &> /dev/null; then
        log_success "OWASP ZAP found in PATH"
        return 0
    elif [ -f "/opt/zaproxy/zap.sh" ]; then
        log_success "OWASP ZAP found at /opt/zaproxy/"
        ZAP_CMD="/opt/zaproxy/zap.sh"
        return 0
    elif [ -f "./zap/zap.sh" ]; then
        log_success "OWASP ZAP found in local directory"
        ZAP_CMD="./zap/zap.sh"
        return 0
    else
        log_error "OWASP ZAP not found. Please install ZAP or set ZAP_HOME environment variable."
        return 1
    fi
}

# Download and install ZAP if not present
install_zap() {
    log_info "Installing OWASP ZAP..."
    
    # Create zap directory
    mkdir -p ./zap
    cd ./zap
    
    # Download ZAP
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        ZAP_DOWNLOAD_URL="https://github.com/zaproxy/zaproxy/releases/download/v${ZAP_VERSION}/ZAP_${ZAP_VERSION}_Linux.tar.gz"
        log_info "Downloading ZAP for Linux..."
        curl -L -o zap.tar.gz "$ZAP_DOWNLOAD_URL"
        tar -xzf zap.tar.gz --strip-components=1
        rm zap.tar.gz
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        ZAP_DOWNLOAD_URL="https://github.com/zaproxy/zaproxy/releases/download/v${ZAP_VERSION}/ZAP_${ZAP_VERSION}_mac.dmg"
        log_info "Downloading ZAP for macOS..."
        curl -L -o zap.dmg "$ZAP_DOWNLOAD_URL"
        # Note: DMG installation requires manual steps on macOS
        log_warning "Please manually install the downloaded DMG file"
        return 1
    else
        log_error "Unsupported operating system: $OSTYPE"
        return 1
    fi
    
    cd ..
    ZAP_CMD="./zap/zap.sh"
    log_success "OWASP ZAP installed successfully"
}

# Start ZAP daemon
start_zap_daemon() {
    log_info "Starting OWASP ZAP daemon..."
    
    # Check if ZAP is already running
    if curl -s "http://localhost:${ZAP_PORT}/JSON/core/view/version/" > /dev/null 2>&1; then
        log_warning "ZAP daemon is already running on port ${ZAP_PORT}"
        return 0
    fi
    
    # Create report directory
    mkdir -p "$REPORT_DIR"
    
    # Start ZAP in daemon mode
    ${ZAP_CMD:-zap.sh} -daemon -port "$ZAP_PORT" -config api.key="$ZAP_API_KEY" \
        -config api.addrs.addr.name="*" \
        -config api.addrs.addr.regex=true \
        -config spider.maxDepth=5 \
        -config spider.maxChildren=10 \
        -config scanner.maxRuleDurationInMins=5 \
        -config scanner.maxScanDurationInMins=30 \
        > "$REPORT_DIR/zap-daemon.log" 2>&1 &
    
    ZAP_PID=$!
    echo $ZAP_PID > "$REPORT_DIR/zap.pid"
    
    # Wait for ZAP to start
    log_info "Waiting for ZAP daemon to start..."
    for i in {1..30}; do
        if curl -s "http://localhost:${ZAP_PORT}/JSON/core/view/version/?apikey=${ZAP_API_KEY}" > /dev/null 2>&1; then
            log_success "ZAP daemon started successfully on port ${ZAP_PORT}"
            return 0
        fi
        sleep 2
    done
    
    log_error "Failed to start ZAP daemon"
    return 1
}

# Stop ZAP daemon
stop_zap_daemon() {
    log_info "Stopping OWASP ZAP daemon..."
    
    if [ -f "$REPORT_DIR/zap.pid" ]; then
        ZAP_PID=$(cat "$REPORT_DIR/zap.pid")
        if kill -0 "$ZAP_PID" 2>/dev/null; then
            kill "$ZAP_PID"
            rm "$REPORT_DIR/zap.pid"
            log_success "ZAP daemon stopped"
        else
            log_warning "ZAP daemon was not running"
            rm -f "$REPORT_DIR/zap.pid"
        fi
    else
        # Try to stop via API
        curl -s "http://localhost:${ZAP_PORT}/JSON/core/action/shutdown/?apikey=${ZAP_API_KEY}" > /dev/null 2>&1 || true
        log_info "Sent shutdown command to ZAP daemon"
    fi
}

# Create new ZAP session
create_zap_session() {
    local session_name="security-scan-$(date +%Y%m%d-%H%M%S)"
    
    log_info "Creating ZAP session: $session_name"
    
    curl -s "http://localhost:${ZAP_PORT}/JSON/core/action/newSession/?apikey=${ZAP_API_KEY}&name=${session_name}" > /dev/null
    
    if [ $? -eq 0 ]; then
        log_success "ZAP session created: $session_name"
        echo "$session_name"
    else
        log_error "Failed to create ZAP session"
        return 1
    fi
}

# Configure ZAP context
configure_zap_context() {
    local context_name="news-website-context"
    
    log_info "Configuring ZAP context: $context_name"
    
    # Create context
    curl -s "http://localhost:${ZAP_PORT}/JSON/context/action/newContext/?apikey=${ZAP_API_KEY}&contextName=${context_name}" > /dev/null
    
    # Include URLs in context
    curl -s "http://localhost:${ZAP_PORT}/JSON/context/action/includeInContext/?apikey=${ZAP_API_KEY}&contextName=${context_name}&regex=${TARGET_URL}.*" > /dev/null
    
    # Exclude static resources
    curl -s "http://localhost:${ZAP_PORT}/JSON/context/action/excludeFromContext/?apikey=${ZAP_API_KEY}&contextName=${context_name}&regex=${TARGET_URL}/static/.*" > /dev/null
    curl -s "http://localhost:${ZAP_PORT}/JSON/context/action/excludeFromContext/?apikey=${ZAP_API_KEY}&contextName=${context_name}&regex=${TARGET_URL}/favicon.ico" > /dev/null
    
    log_success "ZAP context configured"
}

# Run ZAP spider
run_zap_spider() {
    log_info "Starting ZAP spider scan on $TARGET_URL"
    
    # Start spider
    local spider_id=$(curl -s "http://localhost:${ZAP_PORT}/JSON/spider/action/scan/?apikey=${ZAP_API_KEY}&url=${TARGET_URL}&maxChildren=10&recurse=true&contextName=news-website-context" | jq -r '.scan')
    
    if [ "$spider_id" = "null" ] || [ -z "$spider_id" ]; then
        log_error "Failed to start spider scan"
        return 1
    fi
    
    log_info "Spider scan started with ID: $spider_id"
    
    # Wait for spider to complete
    while true; do
        local status=$(curl -s "http://localhost:${ZAP_PORT}/JSON/spider/view/status/?apikey=${ZAP_API_KEY}&scanId=${spider_id}" | jq -r '.status')
        local progress=$(curl -s "http://localhost:${ZAP_PORT}/JSON/spider/view/status/?apikey=${ZAP_API_KEY}&scanId=${spider_id}" | jq -r '.status')
        
        if [ "$status" = "100" ]; then
            log_success "Spider scan completed"
            break
        fi
        
        log_info "Spider progress: ${status}%"
        sleep 5
    done
    
    # Get spider results
    local urls_found=$(curl -s "http://localhost:${ZAP_PORT}/JSON/core/view/urls/?apikey=${ZAP_API_KEY}" | jq '.urls | length')
    log_success "Spider found $urls_found URLs"
}

# Run ZAP active scan
run_zap_active_scan() {
    log_info "Starting ZAP active scan on $TARGET_URL"
    
    # Start active scan
    local scan_id=$(curl -s "http://localhost:${ZAP_PORT}/JSON/ascan/action/scan/?apikey=${ZAP_API_KEY}&url=${TARGET_URL}&recurse=true&inScopeOnly=false&scanPolicyName=Default%20Policy&method=GET&postData=" | jq -r '.scan')
    
    if [ "$scan_id" = "null" ] || [ -z "$scan_id" ]; then
        log_error "Failed to start active scan"
        return 1
    fi
    
    log_info "Active scan started with ID: $scan_id"
    
    # Wait for active scan to complete
    while true; do
        local status=$(curl -s "http://localhost:${ZAP_PORT}/JSON/ascan/view/status/?apikey=${ZAP_API_KEY}&scanId=${scan_id}" | jq -r '.status')
        
        if [ "$status" = "100" ]; then
            log_success "Active scan completed"
            break
        fi
        
        log_info "Active scan progress: ${status}%"
        sleep 10
    done
}

# Generate ZAP reports
generate_zap_reports() {
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local report_base="$REPORT_DIR/zap-report-$timestamp"
    
    log_info "Generating ZAP reports..."
    
    # Generate HTML report
    curl -s "http://localhost:${ZAP_PORT}/OTHER/core/other/htmlreport/?apikey=${ZAP_API_KEY}" > "${report_base}.html"
    
    # Generate JSON report
    curl -s "http://localhost:${ZAP_PORT}/JSON/core/view/alerts/?apikey=${ZAP_API_KEY}" > "${report_base}.json"
    
    # Generate XML report
    curl -s "http://localhost:${ZAP_PORT}/OTHER/core/other/xmlreport/?apikey=${ZAP_API_KEY}" > "${report_base}.xml"
    
    # Generate summary
    local total_alerts=$(curl -s "http://localhost:${ZAP_PORT}/JSON/core/view/numberOfAlerts/?apikey=${ZAP_API_KEY}" | jq -r '.numberOfAlerts')
    local high_alerts=$(curl -s "http://localhost:${ZAP_PORT}/JSON/core/view/alertsSummary/?apikey=${ZAP_API_KEY}" | jq -r '.alertsSummary.High // 0')
    local medium_alerts=$(curl -s "http://localhost:${ZAP_PORT}/JSON/core/view/alertsSummary/?apikey=${ZAP_API_KEY}" | jq -r '.alertsSummary.Medium // 0')
    local low_alerts=$(curl -s "http://localhost:${ZAP_PORT}/JSON/core/view/alertsSummary/?apikey=${ZAP_API_KEY}" | jq -r '.alertsSummary.Low // 0')
    
    cat > "${report_base}-summary.txt" << EOF
OWASP ZAP Security Scan Summary
==============================
Scan Date: $(date)
Target URL: $TARGET_URL
Total Alerts: $total_alerts

Alert Breakdown:
- High Risk: $high_alerts
- Medium Risk: $medium_alerts
- Low Risk: $low_alerts

Reports Generated:
- HTML Report: ${report_base}.html
- JSON Report: ${report_base}.json
- XML Report: ${report_base}.xml
EOF
    
    log_success "Reports generated in $REPORT_DIR"
    log_info "Total alerts found: $total_alerts (High: $high_alerts, Medium: $medium_alerts, Low: $low_alerts)"
    
    # Return exit code based on findings
    if [ "$high_alerts" -gt 0 ]; then
        log_error "High risk vulnerabilities found!"
        return 2
    elif [ "$medium_alerts" -gt 5 ]; then
        log_warning "Multiple medium risk vulnerabilities found!"
        return 1
    fi
    
    return 0
}

# Main execution function
run_security_scan() {
    log_info "Starting OWASP ZAP security scan..."
    
    # Check if target is accessible
    if ! curl -s --max-time 10 "$TARGET_URL" > /dev/null; then
        log_error "Target URL $TARGET_URL is not accessible"
        return 1
    fi
    
    # Check ZAP installation
    if ! check_zap_installation; then
        if [ "${AUTO_INSTALL_ZAP:-false}" = "true" ]; then
            install_zap || return 1
        else
            log_error "Please install OWASP ZAP or set AUTO_INSTALL_ZAP=true"
            return 1
        fi
    fi
    
    # Start ZAP daemon
    start_zap_daemon || return 1
    
    # Set up cleanup trap
    trap 'stop_zap_daemon' EXIT
    
    # Create session and configure context
    create_zap_session || return 1
    configure_zap_context || return 1
    
    # Run scans
    run_zap_spider || return 1
    run_zap_active_scan || return 1
    
    # Generate reports
    generate_zap_reports
    local scan_result=$?
    
    # Stop ZAP daemon
    stop_zap_daemon
    
    return $scan_result
}

# Script usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

OWASP ZAP Automation Script for Security Testing

OPTIONS:
    -u, --url URL           Target URL to scan (default: http://localhost:8080)
    -p, --port PORT         ZAP daemon port (default: 8080)
    -k, --api-key KEY       ZAP API key (default: auto-generated)
    -r, --report-dir DIR    Report output directory (default: ./reports/security/zap)
    -i, --install           Auto-install ZAP if not found
    -s, --spider-only       Run spider scan only
    -a, --active-only       Run active scan only (requires previous spider)
    -q, --quiet             Quiet mode
    -h, --help              Show this help message

ENVIRONMENT VARIABLES:
    ZAP_PORT                ZAP daemon port
    ZAP_API_KEY             ZAP API key
    TARGET_URL              Target URL to scan
    AUTO_INSTALL_ZAP        Auto-install ZAP if not found (true/false)

EXAMPLES:
    $0                                          # Scan localhost:8080
    $0 -u https://example.com                   # Scan specific URL
    $0 -u http://localhost:3000 -p 8081         # Use different ZAP port
    $0 -s                                       # Spider scan only
    $0 -i -u https://staging.example.com        # Auto-install ZAP and scan

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--url)
            TARGET_URL="$2"
            shift 2
            ;;
        -p|--port)
            ZAP_PORT="$2"
            shift 2
            ;;
        -k|--api-key)
            ZAP_API_KEY="$2"
            shift 2
            ;;
        -r|--report-dir)
            REPORT_DIR="$2"
            shift 2
            ;;
        -i|--install)
            AUTO_INSTALL_ZAP=true
            shift
            ;;
        -s|--spider-only)
            SPIDER_ONLY=true
            shift
            ;;
        -a|--active-only)
            ACTIVE_ONLY=true
            shift
            ;;
        -q|--quiet)
            QUIET=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Check for required tools
if ! command -v curl &> /dev/null; then
    log_error "curl is required but not installed"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    log_error "jq is required but not installed"
    exit 1
fi

# Run the security scan
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    run_security_scan
    exit $?
fi