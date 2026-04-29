#!/bin/bash

# Deployment Verification Script
# This script checks if all components are properly configured and running

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOMAIN="${1:-localhost}"
APP_USER="newsapp"
DB_NAME="newsdb"

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Function to print test results
print_test() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    case $status in
        "PASS")
            echo -e "${GREEN}✓${NC} $test_name: ${GREEN}PASSED${NC} $message"
            ((PASSED++))
            ;;
        "FAIL")
            echo -e "${RED}✗${NC} $test_name: ${RED}FAILED${NC} $message"
            ((FAILED++))
            ;;
        "WARN")
            echo -e "${YELLOW}⚠${NC} $test_name: ${YELLOW}WARNING${NC} $message"
            ((WARNINGS++))
            ;;
        "INFO")
            echo -e "${BLUE}ℹ${NC} $test_name: ${BLUE}INFO${NC} $message"
            ;;
    esac
}

# Function to check if service is running
check_service() {
    local service_name="$1"
    if systemctl is-active --quiet "$service_name"; then
        print_test "Service: $service_name" "PASS" "is running"
        return 0
    else
        print_test "Service: $service_name" "FAIL" "is not running"
        return 1
    fi
}

# Function to check if port is listening
check_port() {
    local port="$1"
    local service="$2"
    if netstat -tuln | grep -q ":$port "; then
        print_test "Port: $port ($service)" "PASS" "is listening"
        return 0
    else
        print_test "Port: $port ($service)" "FAIL" "is not listening"
        return 1
    fi
}

# Function to check file exists and permissions
check_file() {
    local file_path="$1"
    local expected_owner="$2"
    local expected_perms="$3"
    
    if [[ -f "$file_path" ]]; then
        local actual_owner=$(stat -c '%U:%G' "$file_path")
        local actual_perms=$(stat -c '%a' "$file_path")
        
        if [[ "$actual_owner" == "$expected_owner" && "$actual_perms" == "$expected_perms" ]]; then
            print_test "File: $file_path" "PASS" "exists with correct ownership and permissions"
        else
            print_test "File: $file_path" "WARN" "exists but ownership ($actual_owner) or permissions ($actual_perms) may be incorrect"
        fi
        return 0
    else
        print_test "File: $file_path" "FAIL" "does not exist"
        return 1
    fi
}

# Function to check directory exists
check_directory() {
    local dir_path="$1"
    local expected_owner="$2"
    
    if [[ -d "$dir_path" ]]; then
        local actual_owner=$(stat -c '%U:%G' "$dir_path")
        if [[ "$actual_owner" == "$expected_owner" ]]; then
            print_test "Directory: $dir_path" "PASS" "exists with correct ownership"
        else
            print_test "Directory: $dir_path" "WARN" "exists but ownership ($actual_owner) may be incorrect"
        fi
        return 0
    else
        print_test "Directory: $dir_path" "FAIL" "does not exist"
        return 1
    fi
}

# Function to check database connectivity
check_database() {
    local db_host="$1"
    local db_port="$2"
    local db_user="$3"
    local db_name="$4"
    
    if sudo -u "$db_user" psql -h "$db_host" -p "$db_port" -d "$db_name" -c "SELECT 1;" &>/dev/null; then
        print_test "Database: $db_name" "PASS" "connection successful"
        return 0
    else
        print_test "Database: $db_name" "FAIL" "connection failed"
        return 1
    fi
}

# Function to check HTTP/HTTPS connectivity
check_web() {
    local url="$1"
    local expected_code="$2"
    
    local response_code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    
    if [[ "$response_code" == "$expected_code" ]]; then
        print_test "Web: $url" "PASS" "returned HTTP $response_code"
        return 0
    else
        print_test "Web: $url" "FAIL" "returned HTTP $response_code (expected $expected_code)"
        return 1
    fi
}

# Function to check SSL certificate
check_ssl() {
    local domain="$1"
    
    if echo | openssl s_client -connect "$domain:443" -servername "$domain" 2>/dev/null | openssl x509 -noout -dates &>/dev/null; then
        local expiry=$(echo | openssl s_client -connect "$domain:443" -servername "$domain" 2>/dev/null | openssl x509 -noout -enddate | cut -d= -f2)
        print_test "SSL Certificate: $domain" "PASS" "valid (expires: $expiry)"
        return 0
    else
        print_test "SSL Certificate: $domain" "FAIL" "invalid or not found"
        return 1
    fi
}

# Function to check firewall rules
check_firewall() {
    if command -v ufw &> /dev/null; then
        if ufw status | grep -q "Status: active"; then
            print_test "Firewall: UFW" "PASS" "is active"
            
            # Check specific rules
            if ufw status | grep -q "22/tcp"; then
                print_test "Firewall: SSH (22)" "PASS" "rule exists"
            else
                print_test "Firewall: SSH (22)" "WARN" "rule not found"
            fi
            
            if ufw status | grep -q "80/tcp"; then
                print_test "Firewall: HTTP (80)" "PASS" "rule exists"
            else
                print_test "Firewall: HTTP (80)" "WARN" "rule not found"
            fi
            
            if ufw status | grep -q "443/tcp"; then
                print_test "Firewall: HTTPS (443)" "PASS" "rule exists"
            else
                print_test "Firewall: HTTPS (443)" "WARN" "rule not found"
            fi
        else
            print_test "Firewall: UFW" "WARN" "is installed but not active"
        fi
    else
        print_test "Firewall: UFW" "WARN" "is not installed"
    fi
}

# Function to check system resources
check_resources() {
    # Check disk space
    local disk_usage=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
    if [[ $disk_usage -lt 80 ]]; then
        print_test "Disk Usage" "PASS" "${disk_usage}% used"
    elif [[ $disk_usage -lt 90 ]]; then
        print_test "Disk Usage" "WARN" "${disk_usage}% used"
    else
        print_test "Disk Usage" "FAIL" "${disk_usage}% used (critical)"
    fi
    
    # Check memory usage
    local mem_usage=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
    if [[ $mem_usage -lt 80 ]]; then
        print_test "Memory Usage" "PASS" "${mem_usage}% used"
    elif [[ $mem_usage -lt 90 ]]; then
        print_test "Memory Usage" "WARN" "${mem_usage}% used"
    else
        print_test "Memory Usage" "FAIL" "${mem_usage}% used (critical)"
    fi
    
    # Check load average
    local load_avg=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//')
    local cpu_cores=$(nproc)
    local load_percentage=$(echo "$load_avg * 100 / $cpu_cores" | bc -l | cut -d. -f1)
    
    if [[ $load_percentage -lt 70 ]]; then
        print_test "System Load" "PASS" "${load_avg} (${load_percentage}% of ${cpu_cores} cores)"
    elif [[ $load_percentage -lt 90 ]]; then
        print_test "System Load" "WARN" "${load_avg} (${load_percentage}% of ${cpu_cores} cores)"
    else
        print_test "System Load" "FAIL" "${load_avg} (${load_percentage}% of ${cpu_cores} cores)"
    fi
}

# Function to check log files
check_logs() {
    local log_files=(
        "/var/log/newsapp/app.log"
        "/var/log/nginx/access.log"
        "/var/log/nginx/error.log"
        "/var/log/postgresql/postgresql-*.log"
    )
    
    for log_file in "${log_files[@]}"; do
        if ls $log_file &>/dev/null; then
            local size=$(du -h $log_file 2>/dev/null | cut -f1 | head -1)
            print_test "Log File: $log_file" "PASS" "exists (size: $size)"
        else
            print_test "Log File: $log_file" "WARN" "not found"
        fi
    done
}

# Function to check cron jobs
check_cron() {
    # Check root cron for certbot renewal
    if crontab -l 2>/dev/null | grep -q certbot; then
        print_test "Cron: SSL Renewal" "PASS" "certbot renewal job exists"
    else
        print_test "Cron: SSL Renewal" "WARN" "certbot renewal job not found"
    fi
    
    # Check app user cron for backups
    if sudo -u $APP_USER crontab -l 2>/dev/null | grep -q backup; then
        print_test "Cron: Database Backup" "PASS" "backup job exists"
    else
        print_test "Cron: Database Backup" "WARN" "backup job not found"
    fi
}

# Function to check application health
check_app_health() {
    if [[ "$DOMAIN" != "localhost" ]]; then
        # Check health endpoint
        if curl -s "https://$DOMAIN/health" | grep -q "OK\|healthy\|success"; then
            print_test "Application: Health Check" "PASS" "health endpoint responding"
        else
            print_test "Application: Health Check" "FAIL" "health endpoint not responding correctly"
        fi
        
        # Check main page
        if curl -s "https://$DOMAIN/" | grep -q "<html\|<!DOCTYPE"; then
            print_test "Application: Main Page" "PASS" "main page loading"
        else
            print_test "Application: Main Page" "FAIL" "main page not loading correctly"
        fi
    else
        print_test "Application: Health Check" "INFO" "skipped (localhost domain)"
    fi
}

# Main verification function
main() {
    echo -e "${BLUE}=== Production Deployment Verification ===${NC}"
    echo "Domain: $DOMAIN"
    echo "App User: $APP_USER"
    echo "Database: $DB_NAME"
    echo
    
    # System Services
    echo -e "${BLUE}--- System Services ---${NC}"
    check_service "nginx"
    check_service "postgresql"
    check_service "pgbouncer"
    check_service "newsapp"
    echo
    
    # Network Ports
    echo -e "${BLUE}--- Network Ports ---${NC}"
    check_port "80" "HTTP"
    check_port "443" "HTTPS"
    check_port "5432" "PostgreSQL"
    check_port "6432" "PgBouncer"
    check_port "8080" "Application"
    echo
    
    # Configuration Files
    echo -e "${BLUE}--- Configuration Files ---${NC}"
    check_file "/etc/nginx/sites-available/newsapp" "root:root" "644"
    check_file "/etc/systemd/system/newsapp.service" "root:root" "644"
    check_file "/home/$APP_USER/news-website/.env.production" "$APP_USER:$APP_USER" "600"
    check_file "/home/$APP_USER/backup.sh" "$APP_USER:$APP_USER" "755"
    check_file "/etc/logrotate.d/newsapp" "root:root" "644"
    echo
    
    # Directories
    echo -e "${BLUE}--- Directories ---${NC}"
    check_directory "/home/$APP_USER/news-website" "$APP_USER:$APP_USER"
    check_directory "/home/$APP_USER/backups" "$APP_USER:$APP_USER"
    check_directory "/var/log/newsapp" "$APP_USER:$APP_USER"
    echo
    
    # Database Connectivity
    echo -e "${BLUE}--- Database Connectivity ---${NC}"
    check_database "127.0.0.1" "5432" "$APP_USER" "$DB_NAME"
    check_database "127.0.0.1" "6432" "$APP_USER" "$DB_NAME"
    echo
    
    # Web Connectivity
    if [[ "$DOMAIN" != "localhost" ]]; then
        echo -e "${BLUE}--- Web Connectivity ---${NC}"
        check_web "http://$DOMAIN" "301"  # Should redirect to HTTPS
        check_web "https://$DOMAIN" "200"
        check_web "https://www.$DOMAIN" "200"
        echo
        
        # SSL Certificate
        echo -e "${BLUE}--- SSL Certificate ---${NC}"
        check_ssl "$DOMAIN"
        echo
    fi
    
    # Firewall
    echo -e "${BLUE}--- Firewall ---${NC}"
    check_firewall
    echo
    
    # System Resources
    echo -e "${BLUE}--- System Resources ---${NC}"
    check_resources
    echo
    
    # Log Files
    echo -e "${BLUE}--- Log Files ---${NC}"
    check_logs
    echo
    
    # Cron Jobs
    echo -e "${BLUE}--- Cron Jobs ---${NC}"
    check_cron
    echo
    
    # Application Health
    echo -e "${BLUE}--- Application Health ---${NC}"
    check_app_health
    echo
    
    # Summary
    echo -e "${BLUE}=== Verification Summary ===${NC}"
    echo -e "${GREEN}Passed: $PASSED${NC}"
    echo -e "${YELLOW}Warnings: $WARNINGS${NC}"
    echo -e "${RED}Failed: $FAILED${NC}"
    echo
    
    if [[ $FAILED -eq 0 ]]; then
        if [[ $WARNINGS -eq 0 ]]; then
            echo -e "${GREEN}✓ All checks passed! Deployment appears to be successful.${NC}"
            exit 0
        else
            echo -e "${YELLOW}⚠ Deployment completed with warnings. Please review the warnings above.${NC}"
            exit 1
        fi
    else
        echo -e "${RED}✗ Deployment has issues. Please fix the failed checks above.${NC}"
        exit 2
    fi
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}This script must be run as root (use sudo)${NC}"
    exit 1
fi

# Run main function
main "$@"