#!/bin/bash

#############################################
# News Website - Multi-Site Installation Script
# Installs multiple independent instances on one server
# Usage: ./multi-site-install.sh <domain> <site-number> [email]
# Example: ./multi-site-install.sh cryptonlisys.com 1 admin@example.com
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

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <domain> <site-number> [email]"
    echo ""
    echo "Site numbers and their ports:"
    echo "  1 = ports 8081, 5433, 6381"
    echo "  2 = ports 8082, 5434, 6382"
    echo "  3 = ports 8083, 5435, 6383"
    echo ""
    echo "Examples:"
    echo "  $0 enginosys.com 1 admin@enginosys.com"
    echo "  $0 cryptonlisys.com 2 admin@cryptonlisys.com"
    echo "  $0 technolisys.com 3 admin@technolisys.com"
    exit 1
fi

DOMAIN=$1
SITE_NUM=$2
EMAIL=${3:-"admin@$DOMAIN"}

# Calculate ports based on site number
APP_PORT=$((8080 + SITE_NUM))
DB_PORT=$((5432 + SITE_NUM))
REDIS_PORT=$((6380 + SITE_NUM))

# Site-specific names
SITE_NAME=$(echo "$DOMAIN" | cut -d'.' -f1)
PROJECT_NAME="${SITE_NAME}"
INSTALL_DIR="/opt/${SITE_NAME}-website"

log_info "=========================================="
log_info "Installing News Website - Multi-Site Mode"
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
        log_error "Port $1 is already in use!"
        ss -tlnp | grep ":$1 "
        exit 1
    fi
}

log_info "Checking port availability..."
check_port $APP_PORT
check_port $DB_PORT
check_port $REDIS_PORT
log_success "All ports available"

# Check if directory exists
if [ -d "$INSTALL_DIR" ]; then
    log_warning "Directory $INSTALL_DIR already exists!"
    read -p "Do you want to continue and overwrite? (y/N): " confirm
    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        log_info "Installation cancelled"
        exit 0
    fi
fi

# Step 1: Ensure Docker is installed
log_info "Step 1/7: Checking Docker..."
if ! command -v docker &> /dev/null; then
    log_info "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker && systemctl start docker
fi
log_success "Docker ready"

# Step 2: Copy source files
log_info "Step 2/7: Copying source files..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_DIR="$(dirname "$SCRIPT_DIR")"

mkdir -p $INSTALL_DIR
cp -r "$SOURCE_DIR"/* $INSTALL_DIR/
log_success "Source files copied"

# Step 3: Generate secure passwords
log_info "Step 3/7: Generating secure passwords..."
DB_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
JWT_SECRET=$(openssl rand -base64 64 | tr -dc 'a-zA-Z0-9' | head -c 64)
RANDOM_PART=$(openssl rand -base64 8 | tr -dc 'a-zA-Z0-9' | head -c 6)
ADMIN_PASSWORD="Admin${RANDOM_PART}!1"
log_success "Passwords generated"

# Step 4: Create configuration
log_info "Step 4/7: Creating configuration..."

cat > $INSTALL_DIR/deployment/.env << EOF
DOMAIN=$DOMAIN
PROJECT_NAME=$PROJECT_NAME
SITE_NUM=$SITE_NUM
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

# Create multi-site docker-compose
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

networks:
  ${PROJECT_NAME}_network:
    driver: bridge
EOF

log_success "Configuration created"

# Step 5: Prepare database
log_info "Step 5/7: Preparing database..."
if [ -f "$INSTALL_DIR/database_schema.sql" ]; then
    cp $INSTALL_DIR/database_schema.sql $INSTALL_DIR/deployment/init-db.sql
else
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
log_info "Step 6/7: Building and starting services..."
cd $INSTALL_DIR/deployment

log_info "Building application (this may take 2-4 minutes)..."
docker compose -p ${PROJECT_NAME} build app --quiet 2>/dev/null || docker compose -p ${PROJECT_NAME} build app

log_info "Starting database and cache..."
docker compose -p ${PROJECT_NAME} up -d postgres redis

log_info "Waiting for database to initialize..."
POSTGRES_READY=false
for i in {1..60}; do
    if docker compose -p ${PROJECT_NAME} exec -T postgres pg_isready -U ${SITE_NAME}app -d ${SITE_NAME}db > /dev/null 2>&1; then
        POSTGRES_READY=true
        log_success "Database is ready"
        break
    fi
    sleep 2
done

if [ "$POSTGRES_READY" != true ]; then
    log_error "PostgreSQL failed to start"
    docker compose -p ${PROJECT_NAME} logs postgres | tail -20
    exit 1
fi

# Fix PostgreSQL authentication method (scram-sha-256 -> md5)
log_info "Configuring PostgreSQL authentication..."
docker compose -p ${PROJECT_NAME} exec -T postgres sh -c "sed -i 's/scram-sha-256/md5/g' /var/lib/postgresql/data/pg_hba.conf" 2>/dev/null || true
docker compose -p ${PROJECT_NAME} exec -T postgres psql -U ${SITE_NAME}app -d ${SITE_NAME}db -c "SELECT pg_reload_conf();" > /dev/null 2>&1 || true

# Reset password with md5 encoding
docker compose -p ${PROJECT_NAME} exec -T postgres psql -U ${SITE_NAME}app -d ${SITE_NAME}db -c "ALTER USER ${SITE_NAME}app WITH PASSWORD '${DB_PASSWORD}';" > /dev/null 2>&1 || true

# Ensure init-db.sql is executed
log_info "Ensuring database schema is initialized..."
docker compose -p ${PROJECT_NAME} exec -T postgres psql -U ${SITE_NAME}app -d ${SITE_NAME}db -f /docker-entrypoint-initdb.d/init.sql > /dev/null 2>&1 || true

sleep 5

log_info "Starting application..."
docker compose -p ${PROJECT_NAME} up -d app

log_info "Waiting for application to initialize..."
ADMIN_CREATED=false
for i in {1..20}; do
    sleep 5
    USER_COUNT=$(docker compose -p ${PROJECT_NAME} exec -T postgres psql -U ${SITE_NAME}app -d ${SITE_NAME}db -t -c "SELECT COUNT(*) FROM users WHERE role='admin';" 2>/dev/null | tr -d ' ' || echo "0")
    if [ "$USER_COUNT" -gt 0 ] 2>/dev/null; then
        ADMIN_CREATED=true
        log_success "Admin user created successfully"
        break
    fi
done

if [ "$ADMIN_CREATED" != true ]; then
    log_warning "Restarting app to ensure database connection..."
    docker compose -p ${PROJECT_NAME} restart app
    sleep 15
fi

log_success "Application started on port $APP_PORT"

# Step 7: Configure nginx
log_info "Step 7/7: Configuring nginx..."

# Create nginx config for this site
cat > /etc/nginx/sites-available/${DOMAIN}.conf << NGINXEOF
server {
    listen 80;
    server_name ${DOMAIN} www.${DOMAIN};

    location / {
        proxy_pass http://127.0.0.1:${APP_PORT};
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
        
        # WebSocket support
        proxy_set_header Connection "upgrade";
    }

    location /static/ {
        proxy_pass http://127.0.0.1:${APP_PORT};
        proxy_cache_valid 200 1d;
        expires 1d;
        add_header Cache-Control "public, immutable";
    }

    location /uploads/ {
        proxy_pass http://127.0.0.1:${APP_PORT};
        proxy_cache_valid 200 7d;
        expires 7d;
    }
}
NGINXEOF

# Enable the site
ln -sf /etc/nginx/sites-available/${DOMAIN}.conf /etc/nginx/sites-enabled/

# Test nginx config
if nginx -t; then
    systemctl reload nginx
    log_success "Nginx configured and reloaded"
else
    log_error "Nginx configuration error!"
    exit 1
fi

# Try to get SSL certificate
log_info "Requesting SSL certificate..."
if command -v certbot &> /dev/null; then
    if certbot --nginx -d ${DOMAIN} -d www.${DOMAIN} --non-interactive --agree-tos --email ${EMAIL} 2>&1; then
        log_success "SSL certificate obtained"
        SSL_SUCCESS=true
    else
        log_warning "SSL certificate failed - site will run on HTTP"
        log_info "You can try manually later: certbot --nginx -d ${DOMAIN}"
        SSL_SUCCESS=false
    fi
else
    log_info "Installing certbot..."
    apt-get update -qq && apt-get install -y -qq certbot python3-certbot-nginx > /dev/null 2>&1
    if certbot --nginx -d ${DOMAIN} -d www.${DOMAIN} --non-interactive --agree-tos --email ${EMAIL} 2>&1; then
        log_success "SSL certificate obtained"
        SSL_SUCCESS=true
    else
        log_warning "SSL certificate failed - site will run on HTTP"
        SSL_SUCCESS=false
    fi
fi

# Create management scripts
cat > $INSTALL_DIR/status.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")/deployment"
echo "=== Service Status ==="
docker compose ps
echo ""
echo "=== Health Check ==="
source .env
curl -s http://localhost:${APP_PORT}/health 2>/dev/null | python3 -m json.tool 2>/dev/null || curl -s http://localhost:${APP_PORT}/health
EOF
chmod +x $INSTALL_DIR/status.sh

cat > $INSTALL_DIR/logs.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")/deployment"
docker compose logs -f --tail=100 app
EOF
chmod +x $INSTALL_DIR/logs.sh

cat > $INSTALL_DIR/restart.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")/deployment"
echo "Restarting services..."
docker compose restart
echo "Restart complete!"
docker compose ps
EOF
chmod +x $INSTALL_DIR/restart.sh

cat > $INSTALL_DIR/backup.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")/deployment"
source .env
BACKUP_DIR="$(dirname "$0")/backups"
mkdir -p $BACKUP_DIR
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
echo "Creating database backup..."
docker compose exec -T postgres pg_dump -U ${DB_USER} ${DB_NAME} > $BACKUP_DIR/db_$TIMESTAMP.sql
echo "Backup saved to: $BACKUP_DIR/db_$TIMESTAMP.sql"
EOF
chmod +x $INSTALL_DIR/backup.sh

# Save credentials
SITE_URL="https://${DOMAIN}"
[ "$SSL_SUCCESS" != true ] && SITE_URL="http://${DOMAIN}"

cat > $INSTALL_DIR/CREDENTIALS.txt << EOF
==========================================
   ${DOMAIN} - Installation Complete
==========================================

Domain: ${DOMAIN}
Website: ${SITE_URL}
Admin Panel: ${SITE_URL}/admin/login

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
Database Password: ${DB_PASSWORD}
JWT Secret: ${JWT_SECRET}

------------------------------------------
   Management Commands
------------------------------------------
Status:  ${INSTALL_DIR}/status.sh
Logs:    ${INSTALL_DIR}/logs.sh
Restart: ${INSTALL_DIR}/restart.sh
Backup:  ${INSTALL_DIR}/backup.sh

Docker:  cd ${INSTALL_DIR}/deployment && docker compose [command]

==========================================
EOF
chmod 600 $INSTALL_DIR/CREDENTIALS.txt

# Final summary
echo ""
echo "=============================================="
log_success "Installation Complete!"
echo "=============================================="
echo ""
echo -e "Website:  ${GREEN}${SITE_URL}${NC}"
echo -e "Admin:    ${GREEN}${SITE_URL}/admin/login${NC}"
echo ""
echo -e "Email:    ${YELLOW}${EMAIL}${NC}"
echo -e "Password: ${YELLOW}${ADMIN_PASSWORD}${NC}"
echo ""
echo "Credentials saved to: ${INSTALL_DIR}/CREDENTIALS.txt"
echo "=============================================="
