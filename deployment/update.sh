#!/bin/bash

#############################################
# News Website - Update Script
# Updates the application without losing data
#############################################

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

INSTALL_DIR="/opt/news-website"
cd $INSTALL_DIR/deployment

# Use docker compose (v2) or docker-compose (v1)
if docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
else
    COMPOSE_CMD="docker-compose"
fi

log_info "Creating backup before update..."
$INSTALL_DIR/backup.sh 2>/dev/null || true

log_info "Pulling latest images..."
$COMPOSE_CMD pull postgres redis nginx

log_info "Rebuilding application..."
$COMPOSE_CMD build --no-cache app

log_info "Restarting services..."
$COMPOSE_CMD up -d

log_info "Cleaning up old images..."
docker image prune -f

log_success "Update complete!"
echo ""
$COMPOSE_CMD ps
