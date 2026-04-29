#!/bin/bash

#############################################
# News Website - One-Click Installation Script
# Usage: ./install.sh yourdomain.com [email]
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

if [ -z "$1" ]; then
    echo "Usage: $0 <domain> [email]"
    echo "Example: $0 example.com admin@example.com"
    exit 1
fi

DOMAIN=$1
EMAIL=${2:-"admin@$DOMAIN"}
INSTALL_DIR="/opt/news-website"

log_info "=========================================="
log_info "Installing News Website"
log_info "Domain: $DOMAIN"
log_info "Email: $EMAIL"
log_info "=========================================="

# Step 1: Install Docker
log_info "Step 1/8: Checking Docker..."
if ! command -v docker &> /dev/null; then
    log_info "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker && systemctl start docker
fi
log_success "Docker ready"

# Step 2: Generate secure passwords
log_info "Step 2/8: Generating secure passwords..."
DB_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
JWT_SECRET=$(openssl rand -base64 64 | tr -dc 'a-zA-Z0-9' | head -c 64)
# Password MUST have: uppercase, lowercase, number, special char (min 8 chars)
# Format: Admin + 6 random alphanumeric + !1 = meets all requirements
RANDOM_PART=$(openssl rand -base64 8 | tr -dc 'a-zA-Z0-9' | head -c 6)
ADMIN_PASSWORD="Admin${RANDOM_PART}!1"
log_success "Passwords generated"

# Step 3: Create configuration
log_info "Step 3/8: Creating configuration..."
mkdir -p $INSTALL_DIR/deployment

cat > $INSTALL_DIR/deployment/.env << EOF
DOMAIN=$DOMAIN
PROJECT_NAME=news
DB_NAME=newsdb
DB_USER=newsapp
DB_PASSWORD=$DB_PASSWORD
JWT_SECRET=$JWT_SECRET
ADMIN_EMAIL=$EMAIL
ADMIN_PASSWORD=$ADMIN_PASSWORD
EOF
chmod 600 $INSTALL_DIR/deployment/.env
log_success "Configuration created"

# Step 4: Create nginx config (HTTP only initially for SSL setup)
log_info "Step 4/8: Creating nginx configuration..."
cat > $INSTALL_DIR/deployment/nginx-site.conf << 'NGINXEOF'
server {
    listen 80;
    server_name DOMAIN_PLACEHOLDER;
    
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }
    
    location / {
        proxy_pass http://app:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }
}
NGINXEOF
sed -i "s/DOMAIN_PLACEHOLDER/$DOMAIN/g" $INSTALL_DIR/deployment/nginx-site.conf
log_success "Nginx configuration created"

# Step 5: Prepare database initialization
log_info "Step 5/8: Preparing database..."
if [ -f "$INSTALL_DIR/database_schema.sql" ]; then
    log_info "Using full database schema..."
    cp $INSTALL_DIR/database_schema.sql $INSTALL_DIR/deployment/init-db.sql
else
    log_info "Combining migrations..."
    cat > $INSTALL_DIR/deployment/init-db.sql << 'EOF'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
EOF
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
log_info "Step 6/8: Building and starting services..."
cd $INSTALL_DIR/deployment

log_info "Building application (this may take 2-4 minutes)..."
docker compose build app --quiet 2>/dev/null || docker compose build app

log_info "Starting database and cache..."
docker compose up -d postgres redis

# Wait for postgres to be fully ready
log_info "Waiting for database to initialize..."
POSTGRES_READY=false
for i in {1..60}; do
    if docker compose exec -T postgres pg_isready -U newsapp -d newsdb > /dev/null 2>&1; then
        POSTGRES_READY=true
        log_success "Database is ready"
        break
    fi
    sleep 2
done

if [ "$POSTGRES_READY" != true ]; then
    log_error "PostgreSQL failed to start within 2 minutes"
    docker compose logs postgres | tail -20
    exit 1
fi

# Extra wait for database to fully initialize
sleep 5

log_info "Starting application..."
docker compose up -d app

# Wait for app to connect to database and create admin user
log_info "Waiting for application to initialize..."
ADMIN_CREATED=false
for i in {1..20}; do
    sleep 5
    
    # Check if admin user was created
    USER_COUNT=$(docker compose exec -T postgres psql -U newsapp -d newsdb -t -c "SELECT COUNT(*) FROM users WHERE role='admin';" 2>/dev/null | tr -d ' ' || echo "0")
    
    if [ "$USER_COUNT" -gt 0 ] 2>/dev/null; then
        ADMIN_CREATED=true
        log_success "Admin user created successfully"
        break
    fi
    
    # Check app logs for errors
    APP_LOGS=$(docker compose logs app 2>&1 | tail -10)
    
    # If app failed to connect or create user, restart it
    if echo "$APP_LOGS" | grep -q "mock services\|connection refused\|Failed to initialize database"; then
        log_warning "App not connected to database (attempt $i), restarting..."
        docker compose restart app
        sleep 5
    elif echo "$APP_LOGS" | grep -q "password validation failed"; then
        log_error "Password validation failed - this should not happen with the new password format"
        break
    fi
done

# If admin still not created, try one more restart
if [ "$ADMIN_CREATED" != true ]; then
    log_warning "Admin user not created yet, forcing restart..."
    docker compose restart app
    sleep 20
    
    USER_COUNT=$(docker compose exec -T postgres psql -U newsapp -d newsdb -t -c "SELECT COUNT(*) FROM users WHERE role='admin';" 2>/dev/null | tr -d ' ' || echo "0")
    if [ "$USER_COUNT" -gt 0 ] 2>/dev/null; then
        ADMIN_CREATED=true
        log_success "Admin user created after restart"
    else
        log_warning "Admin user may not have been created - check logs with: docker compose logs app"
    fi
fi

log_info "Starting nginx..."
docker compose up -d nginx
sleep 3

log_success "All services started"

# Step 7: SSL Certificate
log_info "Step 7/8: Setting up SSL certificate..."

# Install dig if not available
if ! command -v dig &> /dev/null; then
    apt-get update -qq && apt-get install -y -qq dnsutils > /dev/null 2>&1
fi

# Get IPs for comparison (force IPv4)
RESOLVED_IP=$(dig +short $DOMAIN A | grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$' | head -1)
SERVER_IP=$(curl -4 -s --connect-timeout 5 ifconfig.me 2>/dev/null || curl -4 -s --connect-timeout 5 icanhazip.com 2>/dev/null || hostname -I | awk '{print $1}')

SSL_SUCCESS=false

log_info "Domain resolves to: $RESOLVED_IP"
log_info "Server IP: $SERVER_IP"

if [ "$RESOLVED_IP" != "$SERVER_IP" ] && [ -n "$RESOLVED_IP" ]; then
    log_warning "DNS mismatch detected (possibly Cloudflare proxy)"
    log_warning "Skipping Let's Encrypt - configure SSL via Cloudflare or update DNS"
    log_info "To use Let's Encrypt later, ensure DNS A record points directly to $SERVER_IP"
else
    # Install certbot on host
    log_info "Installing certbot..."
    apt-get update -qq && apt-get install -y -qq certbot > /dev/null 2>&1
    
    # Stop nginx to free port 80
    log_info "Stopping nginx for certificate request..."
    docker compose stop nginx > /dev/null 2>&1
    sleep 2
    
    # Request certificate
    log_info "Requesting SSL certificate from Let's Encrypt..."
    if certbot certonly --standalone \
        --email "$EMAIL" \
        --agree-tos \
        --no-eff-email \
        --non-interactive \
        -d "$DOMAIN" 2>&1; then
        SSL_SUCCESS=true
        log_success "SSL certificate obtained"
    else
        log_warning "SSL certificate request failed - site will run on HTTP"
    fi
fi

# Update nginx config based on SSL status
if [ "$SSL_SUCCESS" = true ]; then
    log_info "Configuring nginx with HTTPS..."
    cat > $INSTALL_DIR/deployment/nginx-site.conf << NGINXEOF
server {
    listen 80;
    server_name $DOMAIN;
    
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }
    
    location / {
        return 301 https://\$host\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN;
    
    ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    
    location / {
        proxy_pass http://app:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_cache_bypass \$http_upgrade;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }
    
    location /static/ {
        proxy_pass http://app:8080;
        proxy_cache_valid 200 1d;
        expires 1d;
        add_header Cache-Control "public, immutable";
    }
}
NGINXEOF
    log_success "HTTPS configuration applied"
fi

# Restart nginx with new config
log_info "Starting nginx..."
docker compose up -d nginx
sleep 3

# Verify nginx config
if docker compose exec -T nginx nginx -t > /dev/null 2>&1; then
    log_success "Nginx configuration valid"
else
    log_warning "Nginx configuration test failed - check logs"
fi

# Final health check
log_info "Running final health checks..."
sleep 5

SITE_URL="https://$DOMAIN"
if [ "$SSL_SUCCESS" != true ]; then
    SITE_URL="http://$DOMAIN"
fi

# Check if site is accessible
HTTP_CODE=$(curl -4 -s -o /dev/null -w "%{http_code}" --connect-timeout 10 -k "$SITE_URL" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
    log_success "Site is accessible (HTTP $HTTP_CODE)"
elif [ "$HTTP_CODE" = "301" ] || [ "$HTTP_CODE" = "302" ]; then
    log_success "Site is accessible (redirect to HTTPS)"
else
    log_warning "Site returned HTTP $HTTP_CODE - may need manual verification"
fi

# Step 8: Create management scripts
log_info "Step 8/8: Creating management scripts..."

cat > $INSTALL_DIR/update.sh << 'EOF'
#!/bin/bash
cd /opt/news-website/deployment
docker compose pull
docker compose build app
docker compose up -d
echo "Update complete!"
EOF
chmod +x $INSTALL_DIR/update.sh

cat > $INSTALL_DIR/status.sh << 'EOF'
#!/bin/bash
cd /opt/news-website/deployment
echo "=== Service Status ==="
docker compose ps
echo ""
echo "=== Health Check ==="
curl -s http://localhost:8080/health 2>/dev/null | python3 -m json.tool 2>/dev/null || curl -s http://localhost:8080/health
EOF
chmod +x $INSTALL_DIR/status.sh

cat > $INSTALL_DIR/logs.sh << 'EOF'
#!/bin/bash
cd /opt/news-website/deployment
docker compose logs -f --tail=100 app
EOF
chmod +x $INSTALL_DIR/logs.sh

cat > $INSTALL_DIR/restart.sh << 'EOF'
#!/bin/bash
cd /opt/news-website/deployment
echo "Restarting services..."
docker compose restart app
sleep 10
docker compose restart nginx
echo "Restart complete!"
docker compose ps
EOF
chmod +x $INSTALL_DIR/restart.sh

cat > $INSTALL_DIR/backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/news-website/backups"
mkdir -p $BACKUP_DIR
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
cd /opt/news-website/deployment
echo "Creating database backup..."
docker compose exec -T postgres pg_dump -U newsapp newsdb > $BACKUP_DIR/db_$TIMESTAMP.sql
echo "Backup saved to: $BACKUP_DIR/db_$TIMESTAMP.sql"
EOF
chmod +x $INSTALL_DIR/backup.sh

# SSL renewal cron job
if [ "$SSL_SUCCESS" = true ]; then
    cat > $INSTALL_DIR/renew-ssl.sh << 'EOF'
#!/bin/bash
certbot renew --quiet
cd /opt/news-website/deployment && docker compose exec -T nginx nginx -s reload
EOF
    chmod +x $INSTALL_DIR/renew-ssl.sh
    echo "0 3 * * * root /opt/news-website/renew-ssl.sh >> /var/log/ssl-renew.log 2>&1" > /etc/cron.d/news-ssl-renew
    log_success "SSL auto-renewal configured"
fi

# Save credentials
cat > $INSTALL_DIR/CREDENTIALS.txt << EOF
==========================================
   News Website - Installation Complete
==========================================

Domain: $DOMAIN
Website: $SITE_URL
Admin Panel: $SITE_URL/admin/login

------------------------------------------
   Admin Login Credentials
------------------------------------------
Email: $EMAIL
Password: $ADMIN_PASSWORD

------------------------------------------
   Database Credentials (internal use)
------------------------------------------
Database: newsdb
Username: newsapp
Password: $DB_PASSWORD

------------------------------------------
   Security
------------------------------------------
JWT Secret: $JWT_SECRET
SSL: $([ "$SSL_SUCCESS" = true ] && echo "Enabled (Let's Encrypt)" || echo "Disabled (HTTP only)")

------------------------------------------
   Management Commands
------------------------------------------
Status:  /opt/news-website/status.sh
Logs:    /opt/news-website/logs.sh
Restart: /opt/news-website/restart.sh
Update:  /opt/news-website/update.sh
Backup:  /opt/news-website/backup.sh

==========================================
EOF
chmod 600 $INSTALL_DIR/CREDENTIALS.txt

# Final summary
echo ""
echo "=============================================="
log_success "Installation Complete!"
echo "=============================================="
echo ""
echo -e "Website:  ${GREEN}$SITE_URL${NC}"
echo -e "Admin:    ${GREEN}$SITE_URL/admin/login${NC}"
echo ""
echo -e "Email:    ${YELLOW}$EMAIL${NC}"
echo -e "Password: ${YELLOW}$ADMIN_PASSWORD${NC}"
echo ""
if [ "$ADMIN_CREATED" = true ]; then
    echo -e "${GREEN}✓ Admin user created successfully${NC}"
else
    echo -e "${YELLOW}⚠ Admin user may need manual verification${NC}"
    echo "  Check with: docker compose exec postgres psql -U newsapp -d newsdb -c \"SELECT email FROM users;\""
fi
echo ""
echo "Credentials saved to: $INSTALL_DIR/CREDENTIALS.txt"
echo "=============================================="
