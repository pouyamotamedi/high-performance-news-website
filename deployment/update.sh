#!/bin/bash

#############################################
# News Website - Safe Update Script
# Updates code without touching docker-compose.yml or .env
# Usage: ./update.sh [--rebuild]
#############################################

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="$(dirname "$SCRIPT_DIR")"
REBUILD=false

# Parse arguments
if [ "$1" == "--rebuild" ]; then
    REBUILD=true
fi

log_info "=========================================="
log_info "News Website - Safe Update"
log_info "=========================================="
log_info "Install Dir: $INSTALL_DIR"
log_info "Rebuild: $REBUILD"
log_info "=========================================="

# Check if docker-compose.yml exists and backup
if [ ! -f "$SCRIPT_DIR/docker-compose.yml" ]; then
    log_error "docker-compose.yml not found! Run install script first."
    exit 1
fi

if [ ! -f "$SCRIPT_DIR/.env" ]; then
    log_error ".env not found! Run install script first."
    exit 1
fi

# Backup important files
log_info "Backing up configuration files..."
cp "$SCRIPT_DIR/docker-compose.yml" "$SCRIPT_DIR/docker-compose.yml.backup"
cp "$SCRIPT_DIR/.env" "$SCRIPT_DIR/.env.backup"
if [ -f "$SCRIPT_DIR/init-db.sql" ]; then
    cp "$SCRIPT_DIR/init-db.sql" "$SCRIPT_DIR/init-db.sql.backup"
fi

# Pull latest code from git
log_info "Pulling latest code from git..."
cd "$INSTALL_DIR"
if [ -d ".git" ]; then
    git fetch origin
    git reset --hard origin/master 2>/dev/null || git reset --hard origin/main
    log_success "Code updated from git"
else
    log_warning "Not a git repository - skipping git pull"
fi

# Restore configuration files
log_info "Restoring configuration files..."
cp "$SCRIPT_DIR/docker-compose.yml.backup" "$SCRIPT_DIR/docker-compose.yml"
cp "$SCRIPT_DIR/.env.backup" "$SCRIPT_DIR/.env"
if [ -f "$SCRIPT_DIR/init-db.sql.backup" ]; then
    cp "$SCRIPT_DIR/init-db.sql.backup" "$SCRIPT_DIR/init-db.sql"
fi
log_success "Configuration files restored"

# Get project name from docker-compose.yml
PROJECT_NAME=$(grep "container_name:" "$SCRIPT_DIR/docker-compose.yml" | head -1 | sed 's/.*: //' | sed 's/_postgres//' | sed 's/_redis//' | sed 's/_app//' | tr -d ' ')
log_info "Project name: $PROJECT_NAME"

cd "$SCRIPT_DIR"

if [ "$REBUILD" == true ]; then
    log_info "Rebuilding application..."
    docker compose -p "$PROJECT_NAME" build app --no-cache
    log_info "Restarting services..."
    docker compose -p "$PROJECT_NAME" up -d
else
    log_info "Restarting application only..."
    docker compose -p "$PROJECT_NAME" up -d --build app
fi

# Wait for health check
log_info "Waiting for application to be healthy..."
sleep 10

# Check status
if docker compose -p "$PROJECT_NAME" ps | grep -q "healthy"; then
    log_success "Application is healthy!"
else
    log_warning "Application may still be starting. Check with: docker compose -p $PROJECT_NAME ps"
fi

# Cleanup backups
rm -f "$SCRIPT_DIR/docker-compose.yml.backup"
rm -f "$SCRIPT_DIR/.env.backup"
rm -f "$SCRIPT_DIR/init-db.sql.backup"

log_success "=========================================="
log_success "Update complete!"
log_success "=========================================="
echo ""
echo "Commands:"
echo "  Status:  docker compose -p $PROJECT_NAME ps"
echo "  Logs:    docker compose -p $PROJECT_NAME logs -f app"
echo "  Restart: docker compose -p $PROJECT_NAME restart"
