#!/bin/bash

#############################################
# News Website - Single Site Installation Script
# Creates isolated Docker environment for each site
# Usage: ./install-site.sh <domain> <app-port> <db-port> <redis-port> [email]
# Example: ./install-site.sh cryptonlisys.com 8082 5434 6382 admin@cryptonlisys.com
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

if [ "$EUID" -ne 0 ]; then
    log_error "Please run as root"
    exit 1
fi

if [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ]; then
    echo "Usage: $0 <domain> <app-port> <db-port> <redis-port> [email]"
    echo ""
    echo "Examples:"
    echo "  $0 enginosys.com 8081 5433 6381 admin@enginosys.com"
    echo "  $0 cryptonlisys.com 8082 5434 6382 admin@cryptonlisys.com"
    echo "  $0 technews.com 8083 5435 6383 admin@technews.com"
    echo ""
    echo "Port allocation guide:"
    echo "  Site 1: 8081, 5433, 6381"
    echo "  Site 2: 8082, 5434, 6382"
    echo "  Site 3: 8083, 5435, 6383"
    exit 1
fi

DOMAIN=$1
APP_PORT=$2
DB_PORT=$3
REDIS_PORT=$4
EMAIL=${5:-"admin@$DOMAIN"}

# Site-specific names
SITE_NAME=$(echo "$DOMAIN" | cut -d'.' -f1)
PROJECT_NAME="${SITE_NAME}"
INSTALL_DIR="/opt/${SITE_NAME}-website"

log_info "=========================================="
log_info "Installing News Website"
log_info "=========================================="
log_info "Domain: $DOMAIN"
log_info "Site Name: $SITE_NAME"
log_info "Email: $EMAIL"
log_info "App Port: $APP_PORT"
log_info "DB Port: $DB_PORT"
log_info "Redis Port: $REDIS_PORT"
log_info "Install Dir: $INSTALL_DIR"
log_info "=========================================="

# Check if ports are available
check_port() {
    if ss -tlnp | grep -q ":$1 "; then
        log_warning "Port $1 is already in use"
        ss -tlnp | grep ":$1 "
        read -p "Continue anyway? (y/N): " confirm
        if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
            exit 1
        fi
    fi
}

log_info "Checking port availability..."
check_port $APP_PORT
check_port $DB_PORT
check_port $REDIS_PORT

# Check if directory exists
if [ -d "$INSTALL_DIR" ]; then
    log_warning "Directory $INSTALL_DIR already exists!"
    read -p "Update existing installation? (y/N): " confirm
    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        log_info "Installation cancelled"
        exit 0
    fi
    UPDATE_MODE=true
else
    UPDATE_MODE=false
fi

# Step 1: Ensure Docker is installed
log_info "Step 1/6: Checking Docker..."
if ! command -v docker &> /dev/null; then
    log_info "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker && systemctl start docker
fi
log_success "Docker ready"

# Step 2: Copy source files
log_info "Step 2/6: Copying source files..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_DIR="$(dirname "$SCRIPT_DIR")"

mkdir -p $INSTALL_DIR

# Copy all files except deployment configs if updating
if [ "$UPDATE_MODE" == true ]; then
    # Backup existing configs
    cp "$INSTALL_DIR/deployment/docker-compose.yml" "$INSTALL_DIR/deployment/docker-compose.yml.backup" 2>/dev/null || true
    cp "$INSTALL_DIR/deployment/.env" "$INSTALL_DIR/deployment/.env.backup" 2>/dev/null || true
fi

# Copy source files
rsync -av --exclude='.git' --exclude='deployment/docker-compose.yml' --exclude='deployment/.env' "$SOURCE_DIR"/ $INSTALL_DIR/ 2>/dev/null || cp -r "$SOURCE_DIR"/* $INSTALL_DIR/

if [ "$UPDATE_MODE" == true ]; then
    # Restore configs
    cp "$INSTALL_DIR/deployment/docker-compose.yml.backup" "$INSTALL_DIR/deployment/docker-compose.yml" 2>/dev/null || true
    cp "$INSTALL_DIR/deployment/.env.backup" "$INSTALL_DIR/deployment/.env" 2>/dev/null || true
    rm -f "$INSTALL_DIR/deployment/docker-compose.yml.backup" "$INSTALL_DIR/deployment/.env.backup"
    log_success "Source files updated (configs preserved)"
else
    log_success "Source files copied"
fi

# Step 3: Generate secure passwords (only for new installations)
if [ "$UPDATE_MODE" != true ]; then
    log_info "Step 3/6: Generating secure passwords..."
    DB_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
    JWT_SECRET=$(openssl rand -base64 64 | tr -dc 'a-zA-Z0-9' | head -c 64)
    RANDOM_PART=$(openssl rand -base64 8 | tr -dc 'a-zA-Z0-9' | head -c 6)
    ADMIN_PASSWORD="Admin${RANDOM_PART}!1"
    log_success "Passwords generated"

    # Step 4: Create configuration
    log_info "Step 4/6: Creating configuration..."

    cat > $INSTALL_DIR/deployment/.env << EOF
DOMAIN=$DOMAIN
PROJECT_NAME=$PROJECT_NAME
APP_PORT=$APP_PORT
DB_PORT=$DB_PORT
REDIS_PORT=$REDIS_PORT
DB_NAME=${SITE_NAME}db
DB_USER=${SITE_NAME}app
DB_PASSWORD=$DB_PASSWORD
JWT_SECRET=$JWT_SECRET
ADMIN_EMAIL=$EMAIL
ADMIN_PASSWORD=$ADMIN_PASSWORD
EOF
    chmod 600 $INSTALL_DIR/deployment/.env
else
    log_info "Step 3/6: Skipping password generation (update mode)"
    log_info "Step 4/6: Preserving existing configuration"
    source $INSTALL_DIR/deployment/.env
fi

# Create site-specific docker-compose.yml (always recreate to ensure correct format)
cat > $INSTALL_DIR/deployment/docker-compose.yml << EOF
services:
  postgres:
    image: postgres:15-alpine
    container_name: ${PROJECT_NAME}_postgres
    environment:
      POSTGRES_DB: ${SITE_NAME}db
      POSTGRES_USER: ${SITE_NAME}app
      POSTGRES_PASSWORD: \${DB_PASSWORD}
    volumes:
      - ${PROJECT_NAME}_postgres_data:/var/lib/postgresql/data
      - ./init-db.sql:/docker-entrypoint-initdb.d/init.sql:ro
    ports:
      - "${DB_PORT}:5432"
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${SITE_NAME}app -d ${SITE_NAME}db"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - ${PROJECT_NAME}_network

  redis:
    image: redis:7-alpine
    container_name: ${PROJECT_NAME}_redis
    command: redis-server --maxmemory 128mb --maxmemory-policy allkeys-lru
    volumes:
      - ${PROJECT_NAME}_redis_data:/data
    ports:
      - "${REDIS_PORT}:6379"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - ${PROJECT_NAME}_network

  app:
    build:
      context: ..
      dockerfile: deployment/Dockerfile
    container_name: ${PROJECT_NAME}_app
    environment:
      - NEWS_SERVER_MODE=release
      - NEWS_APP_ENVIRONMENT=production
      - NEWS_APP_BASE_URL=https://${DOMAIN}
      - NEWS_DATABASE_HOST=postgres
      - NEWS_DATABASE_PORT=5432
      - NEWS_DATABASE_DBNAME=${SITE_NAME}db
      - NEWS_DATABASE_USER=${SITE_NAME}app
      - NEWS_DATABASE_PASSWORD=\${DB_PASSWORD}
      - NEWS_DATABASE_SSLMODE=disable
      - NEWS_CACHE_HOST=redis
      - NEWS_CACHE_PORT=6379
      - NEWS_JWT_SECRET=\${JWT_SECRET}
      - NEWS_ADMIN_EMAIL=\${ADMIN_EMAIL}
      - NEWS_ADMIN_PASSWORD=\${ADMIN_PASSWORD}
      - NEWS_SEARCH_ENABLED=false
    volumes:
      - ${PROJECT_NAME}_uploads:/app/uploads
      - ${PROJECT_NAME}_logs:/app/logs
      - ${PROJECT_NAME}_app_data:/app/data
    ports:
      - "${APP_PORT}:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 10s
      retries: 5
      start_period: 30s
    networks:
      - ${PROJECT_NAME}_network

volumes:
  ${PROJECT_NAME}_postgres_data:
  ${PROJECT_NAME}_redis_data:
  ${PROJECT_NAME}_uploads:
  ${PROJECT_NAME}_logs:
  ${PROJECT_NAME}_app_data:

networks:
  ${PROJECT_NAME}_network:
    driver: bridge
EOF

log_success "Configuration created"

# Step 5: Prepare database init script
log_info "Step 5/6: Preparing database..."
if [ -f "$INSTALL_DIR/database_schema.sql" ]; then
    cp $INSTALL_DIR/database_schema.sql $INSTALL_DIR/deployment/init-db.sql
elif [ -d "$INSTALL_DIR/migrations" ]; then
    cat > $INSTALL_DIR/deployment/init-db.sql << 'SQLEOF'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
SQLEOF
    for migration in $INSTALL_DIR/migrations/*_*.up.sql; do
        if [ -f "$migration" ]; then
            echo "" >> $INSTALL_DIR/deployment/init-db.sql
            echo "-- Migration: $(basename $migration)" >> $INSTALL_DIR/deployment/init-db.sql
            cat "$migration" >> $INSTALL_DIR/deployment/init-db.sql
        fi
    done
fi
log_success "Database prepared"

# Step 6: Build and start services
log_info "Step 6/6: Building and starting services..."
cd $INSTALL_DIR/deployment

# Stop any existing containers with same name
docker rm -f ${PROJECT_NAME}_postgres ${PROJECT_NAME}_redis ${PROJECT_NAME}_app 2>/dev/null || true

log_info "Building application (this may take 2-4 minutes)..."
docker compose -p ${PROJECT_NAME} build app

log_info "Starting services..."
docker compose -p ${PROJECT_NAME} up -d

log_info "Waiting for services to be healthy..."
sleep 15

# Check status
if docker compose -p ${PROJECT_NAME} ps | grep -q "healthy"; then
    log_success "All services are healthy!"
else
    log_warning "Services may still be starting..."
    docker compose -p ${PROJECT_NAME} ps
fi

# Create management scripts
cat > $INSTALL_DIR/status.sh << 'STATUSEOF'
#!/bin/bash
cd "$(dirname "$0")/deployment"
source .env
echo "=== Service Status ==="
docker compose -p $PROJECT_NAME ps
echo ""
echo "=== Health Check ==="
curl -s http://localhost:${APP_PORT}/health 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "Waiting for app..."
STATUSEOF
chmod +x $INSTALL_DIR/status.sh

cat > $INSTALL_DIR/logs.sh << 'LOGSEOF'
#!/bin/bash
cd "$(dirname "$0")/deployment"
source .env
docker compose -p $PROJECT_NAME logs -f --tail=100 app
LOGSEOF
chmod +x $INSTALL_DIR/logs.sh

cat > $INSTALL_DIR/restart.sh << 'RESTARTEOF'
#!/bin/bash
cd "$(dirname "$0")/deployment"
source .env
echo "Restarting services..."
docker compose -p $PROJECT_NAME restart
echo "Restart complete!"
docker compose -p $PROJECT_NAME ps
RESTARTEOF
chmod +x $INSTALL_DIR/restart.sh

# Save credentials (only for new installations)
if [ "$UPDATE_MODE" != true ]; then
    cat > $INSTALL_DIR/CREDENTIALS.txt << EOF
==========================================
   ${DOMAIN} - Installation Complete
==========================================

Website: https://${DOMAIN}
Admin Panel: https://${DOMAIN}/admin/login

------------------------------------------
   Admin Login Credentials
------------------------------------------
Email: ${EMAIL}
Password: ${ADMIN_PASSWORD}

------------------------------------------
   Technical Details
------------------------------------------
Install Directory: ${INSTALL_DIR}
App Port: ${APP_PORT}
Database Port: ${DB_PORT}
Redis Port: ${REDIS_PORT}
Database Name: ${SITE_NAME}db
Database User: ${SITE_NAME}app

------------------------------------------
   Docker Commands
------------------------------------------
Status:  docker compose -p ${PROJECT_NAME} ps
Logs:    docker compose -p ${PROJECT_NAME} logs -f app
Restart: docker compose -p ${PROJECT_NAME} restart
Stop:    docker compose -p ${PROJECT_NAME} down
Start:   docker compose -p ${PROJECT_NAME} up -d

==========================================
EOF
    chmod 600 $INSTALL_DIR/CREDENTIALS.txt
fi

# Final summary
echo ""
echo "=============================================="
log_success "Installation Complete!"
echo "=============================================="
echo ""
echo -e "Website:  ${GREEN}https://${DOMAIN}${NC}"
echo -e "Admin:    ${GREEN}https://${DOMAIN}/admin/login${NC}"
echo ""
if [ "$UPDATE_MODE" != true ]; then
    echo -e "Email:    ${YELLOW}${EMAIL}${NC}"
    echo -e "Password: ${YELLOW}${ADMIN_PASSWORD}${NC}"
    echo ""
    echo "Credentials saved to: ${INSTALL_DIR}/CREDENTIALS.txt"
fi
echo ""
echo "Docker commands:"
echo "  docker compose -p ${PROJECT_NAME} ps"
echo "  docker compose -p ${PROJECT_NAME} logs -f app"
echo "  docker compose -p ${PROJECT_NAME} restart"
echo "=============================================="
