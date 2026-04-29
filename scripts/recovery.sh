#!/bin/bash

# Production Recovery Script for News Website
# This script restores from backups created by backup.sh

set -e

# Configuration
BACKUP_DIR="/home/newsapp/backups"
APP_DIR="/home/newsapp/news-website"
DB_NAME="newsdb"
DB_USER="newsapp"
DB_HOST="127.0.0.1"
DB_PORT="6432"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_prompt() {
    echo -e "${BLUE}[PROMPT]${NC} $1"
}

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Recovery options:
    -d, --database BACKUP_DATE    Restore database from specific backup date
    -a, --application BACKUP_DATE Restore application files from specific backup date
    -c, --configs BACKUP_DATE     Restore configuration files from specific backup date
    -f, --full BACKUP_DATE        Full system restore from specific backup date
    -l, --list                    List available backups
    -h, --help                    Show this help message

BACKUP_DATE format: YYYYMMDD_HHMMSS (e.g., 20231215_143022)

Examples:
    $0 --list                                    # List available backups
    $0 --database 20231215_143022               # Restore only database
    $0 --full 20231215_143022                   # Full system restore
    $0 --application 20231215_143022            # Restore only application files

EOF
}

# List available backups
list_backups() {
    log_info "Available backups:"
    echo
    
    if [[ -d "$BACKUP_DIR" ]]; then
        # Find all manifest files and extract dates
        find "$BACKUP_DIR" -name "manifest_*.txt" -type f | sort -r | while read -r manifest; do
            backup_date=$(basename "$manifest" | sed 's/manifest_\(.*\)\.txt/\1/')
            backup_size=$(du -sh "$BACKUP_DIR" | cut -f1)
            
            echo "Backup Date: $backup_date"
            if [[ -f "$manifest" ]]; then
                echo "  Manifest: $(basename "$manifest")"
                echo "  Files:"
                grep -E "^\- " "$manifest" | sed 's/^/    /'
                echo
            fi
        done
    else
        log_warning "Backup directory not found: $BACKUP_DIR"
    fi
}

# Verify backup files exist
verify_backup_files() {
    local backup_date="$1"
    local check_db="$2"
    local check_app="$3"
    local check_config="$4"
    
    local missing_files=()
    
    if [[ "$check_db" == "true" ]]; then
        if [[ ! -f "$BACKUP_DIR/database/db_backup_${backup_date}.sql.gz" ]]; then
            missing_files+=("database/db_backup_${backup_date}.sql.gz")
        fi
    fi
    
    if [[ "$check_app" == "true" ]]; then
        if [[ ! -f "$BACKUP_DIR/application/app_backup_${backup_date}.tar.gz" ]]; then
            missing_files+=("application/app_backup_${backup_date}.tar.gz")
        fi
    fi
    
    if [[ "$check_config" == "true" ]]; then
        if [[ ! -f "$BACKUP_DIR/configs/config_backup_${backup_date}.tar.gz" ]]; then
            missing_files+=("configs/config_backup_${backup_date}.tar.gz")
        fi
    fi
    
    if [[ ${#missing_files[@]} -gt 0 ]]; then
        log_error "Missing backup files:"
        printf '%s\n' "${missing_files[@]}" | sed 's/^/  - /'
        return 1
    fi
    
    return 0
}

# Restore database
restore_database() {
    local backup_date="$1"
    local backup_file="$BACKUP_DIR/database/db_backup_${backup_date}.sql.gz"
    
    log_info "Restoring database from $backup_file..."
    
    # Stop application service
    log_info "Stopping application service..."
    sudo systemctl stop newsapp || true
    
    # Create a backup of current database
    log_info "Creating safety backup of current database..."
    pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "/tmp/pre_restore_backup_$(date +%Y%m%d_%H%M%S).sql" || true
    
    # Drop and recreate database
    log_warning "Dropping and recreating database..."
    sudo -u postgres psql << EOF
DROP DATABASE IF EXISTS ${DB_NAME}_temp;
CREATE DATABASE ${DB_NAME}_temp;
EOF
    
    # Restore from backup
    log_info "Restoring database from backup..."
    if gunzip -c "$backup_file" | psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "${DB_NAME}_temp"; then
        # Swap databases
        sudo -u postgres psql << EOF
ALTER DATABASE $DB_NAME RENAME TO ${DB_NAME}_old;
ALTER DATABASE ${DB_NAME}_temp RENAME TO $DB_NAME;
DROP DATABASE ${DB_NAME}_old;
EOF
        log_info "Database restore completed successfully"
    else
        log_error "Database restore failed"
        return 1
    fi
}

# Restore application files
restore_application() {
    local backup_date="$1"
    local backup_file="$BACKUP_DIR/application/app_backup_${backup_date}.tar.gz"
    
    log_info "Restoring application files from $backup_file..."
    
    # Stop application service
    log_info "Stopping application service..."
    sudo systemctl stop newsapp || true
    
    # Create safety backup of current application
    log_info "Creating safety backup of current application..."
    tar -czf "/tmp/pre_restore_app_$(date +%Y%m%d_%H%M%S).tar.gz" -C "$APP_DIR" . || true
    
    # Restore application files
    log_info "Extracting application backup..."
    if tar -xzf "$backup_file" -C "$APP_DIR"; then
        # Fix permissions
        chown -R newsapp:newsapp "$APP_DIR"
        chmod +x "$APP_DIR/news-server" 2>/dev/null || true
        log_info "Application files restore completed successfully"
    else
        log_error "Application files restore failed"
        return 1
    fi
}

# Restore configuration files
restore_configs() {
    local backup_date="$1"
    local backup_file="$BACKUP_DIR/configs/config_backup_${backup_date}.tar.gz"
    
    log_info "Restoring configuration files from $backup_file..."
    
    # Create safety backup of current configs
    log_info "Creating safety backup of current configurations..."
    tar -czf "/tmp/pre_restore_config_$(date +%Y%m%d_%H%M%S).tar.gz" \
        /etc/nginx/sites-available/newsapp \
        /etc/systemd/system/newsapp.service \
        "$APP_DIR/.env.production" \
        /etc/logrotate.d/newsapp 2>/dev/null || true
    
    # Restore configuration files
    log_info "Extracting configuration backup..."
    if tar -xzf "$backup_file" -C /; then
        # Reload systemd and nginx
        sudo systemctl daemon-reload
        sudo nginx -t && sudo systemctl reload nginx
        log_info "Configuration files restore completed successfully"
    else
        log_error "Configuration files restore failed"
        return 1
    fi
}

# Main recovery function
perform_recovery() {
    local backup_date="$1"
    local restore_db="$2"
    local restore_app="$3"
    local restore_config="$4"
    
    log_info "Starting recovery process for backup date: $backup_date"
    
    # Verify backup files exist
    if ! verify_backup_files "$backup_date" "$restore_db" "$restore_app" "$restore_config"; then
        log_error "Backup verification failed"
        return 1
    fi
    
    # Confirm recovery
    log_prompt "This will restore from backup date: $backup_date"
    log_prompt "Components to restore:"
    [[ "$restore_db" == "true" ]] && echo "  - Database"
    [[ "$restore_app" == "true" ]] && echo "  - Application files"
    [[ "$restore_config" == "true" ]] && echo "  - Configuration files"
    echo
    read -p "Are you sure you want to proceed? (yes/no): " -r
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log_info "Recovery cancelled"
        return 0
    fi
    
    # Perform recovery steps
    if [[ "$restore_config" == "true" ]]; then
        restore_configs "$backup_date" || return 1
    fi
    
    if [[ "$restore_app" == "true" ]]; then
        restore_application "$backup_date" || return 1
    fi
    
    if [[ "$restore_db" == "true" ]]; then
        restore_database "$backup_date" || return 1
    fi
    
    # Start services
    log_info "Starting services..."
    sudo systemctl start newsapp
    
    # Wait for service to start
    sleep 5
    
    # Verify service is running
    if sudo systemctl is-active --quiet newsapp; then
        log_info "Application service started successfully"
    else
        log_error "Application service failed to start"
        sudo systemctl status newsapp
        return 1
    fi
    
    log_info "Recovery completed successfully!"
    log_info "Please verify application functionality"
}

# Parse command line arguments
BACKUP_DATE=""
RESTORE_DB=false
RESTORE_APP=false
RESTORE_CONFIG=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--database)
            BACKUP_DATE="$2"
            RESTORE_DB=true
            shift 2
            ;;
        -a|--application)
            BACKUP_DATE="$2"
            RESTORE_APP=true
            shift 2
            ;;
        -c|--configs)
            BACKUP_DATE="$2"
            RESTORE_CONFIG=true
            shift 2
            ;;
        -f|--full)
            BACKUP_DATE="$2"
            RESTORE_DB=true
            RESTORE_APP=true
            RESTORE_CONFIG=true
            shift 2
            ;;
        -l|--list)
            list_backups
            exit 0
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

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

# Validate arguments
if [[ -z "$BACKUP_DATE" ]]; then
    log_error "Backup date is required"
    usage
    exit 1
fi

if [[ "$RESTORE_DB" == false && "$RESTORE_APP" == false && "$RESTORE_CONFIG" == false ]]; then
    log_error "At least one restore option must be specified"
    usage
    exit 1
fi

# Perform recovery
perform_recovery "$BACKUP_DATE" "$RESTORE_DB" "$RESTORE_APP" "$RESTORE_CONFIG"