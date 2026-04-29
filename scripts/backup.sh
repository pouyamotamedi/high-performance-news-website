#!/bin/bash

# Production Backup Script for News Website
# This script creates comprehensive backups of database, application files, and configurations

set -e

# Configuration
BACKUP_DIR="/home/newsapp/backups"
APP_DIR="/home/newsapp/news-website"
DB_NAME="newsdb"
DB_USER="newsapp"
DB_HOST="127.0.0.1"
DB_PORT="6432"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=7

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
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

# Create backup directory
mkdir -p "$BACKUP_DIR"/{database,application,configs}

log_info "Starting backup process at $(date)"

# 1. Database backup
log_info "Creating database backup..."
if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$BACKUP_DIR/database/db_backup_$DATE.sql"; then
    gzip "$BACKUP_DIR/database/db_backup_$DATE.sql"
    log_info "Database backup completed: db_backup_$DATE.sql.gz"
else
    log_error "Database backup failed"
    exit 1
fi

# 2. Application files backup
log_info "Creating application files backup..."
if tar -czf "$BACKUP_DIR/application/app_backup_$DATE.tar.gz" -C "$APP_DIR" \
    --exclude='*.log' \
    --exclude='tmp/*' \
    --exclude='cache/*' \
    --exclude='.git/*' \
    .; then
    log_info "Application backup completed: app_backup_$DATE.tar.gz"
else
    log_error "Application backup failed"
    exit 1
fi

# 3. Configuration files backup
log_info "Creating configuration backup..."
if tar -czf "$BACKUP_DIR/configs/config_backup_$DATE.tar.gz" \
    /etc/nginx/sites-available/newsapp \
    /etc/systemd/system/newsapp.service \
    "$APP_DIR/.env.production" \
    /etc/logrotate.d/newsapp 2>/dev/null; then
    log_info "Configuration backup completed: config_backup_$DATE.tar.gz"
else
    log_warning "Some configuration files may not have been backed up"
fi

# 4. Create backup manifest
log_info "Creating backup manifest..."
cat > "$BACKUP_DIR/manifest_$DATE.txt" << EOF
Backup Manifest - $DATE
========================

Backup Date: $(date)
Server: $(hostname)
Database: $DB_NAME
Application Directory: $APP_DIR

Files Created:
- database/db_backup_$DATE.sql.gz ($(du -h "$BACKUP_DIR/database/db_backup_$DATE.sql.gz" | cut -f1))
- application/app_backup_$DATE.tar.gz ($(du -h "$BACKUP_DIR/application/app_backup_$DATE.tar.gz" | cut -f1))
- configs/config_backup_$DATE.tar.gz ($(du -h "$BACKUP_DIR/configs/config_backup_$DATE.tar.gz" | cut -f1))

Total Backup Size: $(du -sh "$BACKUP_DIR" | cut -f1)

Checksums:
$(cd "$BACKUP_DIR" && find . -name "*_$DATE.*" -type f -exec sha256sum {} \;)
EOF

# 5. Cleanup old backups
log_info "Cleaning up old backups (older than $RETENTION_DAYS days)..."
find "$BACKUP_DIR" -name "*backup_*.gz" -mtime +$RETENTION_DAYS -delete
find "$BACKUP_DIR" -name "manifest_*.txt" -mtime +$RETENTION_DAYS -delete

# 6. Verify backup integrity
log_info "Verifying backup integrity..."
if gzip -t "$BACKUP_DIR/database/db_backup_$DATE.sql.gz" && \
   tar -tzf "$BACKUP_DIR/application/app_backup_$DATE.tar.gz" >/dev/null && \
   tar -tzf "$BACKUP_DIR/configs/config_backup_$DATE.tar.gz" >/dev/null; then
    log_info "Backup integrity verification passed"
else
    log_error "Backup integrity verification failed"
    exit 1
fi

# 7. Optional: Upload to remote storage (uncomment and configure as needed)
# if command -v aws >/dev/null 2>&1; then
#     log_info "Uploading to S3..."
#     aws s3 sync "$BACKUP_DIR" s3://your-backup-bucket/$(hostname)/
# fi

log_info "Backup process completed successfully at $(date)"
log_info "Backup location: $BACKUP_DIR"
log_info "Manifest file: $BACKUP_DIR/manifest_$DATE.txt"

# Send notification (optional)
# echo "Backup completed successfully on $(hostname) at $(date)" | mail -s "Backup Success" admin@yourdomain.com