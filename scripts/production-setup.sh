#!/bin/bash

# Production Setup Script for News Website
# This script automates the initial server setup process

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration variables (modify these)
DOMAIN=""
SERVER_IP=""
DB_PASSWORD=""
JWT_SECRET=""
CSRF_SECRET=""
APP_USER="newsapp"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if running as root
check_root() {
    if [[ $EUID -eq 0 ]]; then
        print_status "Running as root - OK"
    else
        print_error "This script must be run as root"
        exit 1
    fi
}

# Function to validate configuration
validate_config() {
    if [[ -z "$DOMAIN" ]]; then
        print_error "DOMAIN variable is not set. Please edit the script and set your domain."
        exit 1
    fi
    
    if [[ -z "$DB_PASSWORD" ]]; then
        print_error "DB_PASSWORD variable is not set. Please edit the script and set a secure password."
        exit 1
    fi
    
    if [[ -z "$JWT_SECRET" ]]; then
        print_error "JWT_SECRET variable is not set. Please edit the script and set a secure secret."
        exit 1
    fi
    
    if [[ -z "$CSRF_SECRET" ]]; then
        print_error "CSRF_SECRET variable is not set. Please edit the script and set a secure secret."
        exit 1
    fi
}

# Function to update system packages
update_system() {
    print_status "Updating system packages..."
    apt update && apt upgrade -y
    apt install -y curl wget git unzip software-properties-common apt-transport-https ca-certificates gnupg lsb-release
}

# Function to install Go
install_go() {
    print_status "Installing Go..."
    cd /tmp
    wget -q https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
    
    # Add Go to PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    echo 'export GOPATH=$HOME/go' >> /etc/profile
    echo 'export PATH=$PATH:$GOPATH/bin' >> /etc/profile
    
    # Verify installation
    /usr/local/go/bin/go version
    print_status "Go installed successfully"
}

# Function to install PostgreSQL
install_postgresql() {
    print_status "Installing PostgreSQL..."
    apt install -y postgresql postgresql-contrib postgresql-client
    systemctl start postgresql
    systemctl enable postgresql
    print_status "PostgreSQL installed and started"
}

# Function to install Nginx
install_nginx() {
    print_status "Installing Nginx..."
    apt install -y nginx
    systemctl start nginx
    systemctl enable nginx
    print_status "Nginx installed and started"
}

# Function to configure firewall
configure_firewall() {
    print_status "Configuring firewall..."
    apt install -y ufw
    ufw default deny incoming
    ufw default allow outgoing
    ufw allow 22/tcp
    ufw allow 80/tcp
    ufw allow 443/tcp
    ufw --force enable
    print_status "Firewall configured"
}

# Function to create application user
create_app_user() {
    print_status "Creating application user: $APP_USER"
    
    # Create user if it doesn't exist
    if ! id "$APP_USER" &>/dev/null; then
        adduser --disabled-password --gecos "" $APP_USER
        usermod -aG sudo $APP_USER
    fi
    
    # Set up SSH directory
    mkdir -p /home/$APP_USER/.ssh
    if [[ -f ~/.ssh/authorized_keys ]]; then
        cp ~/.ssh/authorized_keys /home/$APP_USER/.ssh/
        chown -R $APP_USER:$APP_USER /home/$APP_USER/.ssh
        chmod 700 /home/$APP_USER/.ssh
        chmod 600 /home/$APP_USER/.ssh/authorized_keys
    fi
    
    print_status "Application user created"
}

# Function to configure PostgreSQL
configure_postgresql() {
    print_status "Configuring PostgreSQL..."
    
    # Create database and user
    sudo -u postgres psql << EOF
CREATE DATABASE newsdb;
CREATE USER $APP_USER WITH ENCRYPTED PASSWORD '$DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE newsdb TO $APP_USER;
ALTER USER $APP_USER CREATEDB;
EOF

    # Configure PostgreSQL for production
    PG_VERSION=$(sudo -u postgres psql -t -c "SELECT version();" | grep -oP '\d+\.\d+' | head -1)
    PG_CONFIG="/etc/postgresql/$PG_VERSION/main/postgresql.conf"
    
    # Backup original config
    cp "$PG_CONFIG" "$PG_CONFIG.backup"
    
    # Add performance tuning
    cat >> "$PG_CONFIG" << EOF

# Performance tuning
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
max_connections = 200
EOF

    systemctl restart postgresql
    print_status "PostgreSQL configured"
}

# Function to install PgBouncer
install_pgbouncer() {
    print_status "Installing and configuring PgBouncer..."
    apt install -y pgbouncer
    
    # Configure PgBouncer
    cat > /etc/pgbouncer/pgbouncer.ini << EOF
[databases]
newsdb = host=127.0.0.1 port=5432 dbname=newsdb

[pgbouncer]
listen_port = 6432
listen_addr = 127.0.0.1
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
logfile = /var/log/postgresql/pgbouncer.log
pidfile = /var/run/postgresql/pgbouncer.pid
admin_users = $APP_USER
pool_mode = transaction
server_reset_query = DISCARD ALL
max_client_conn = 200
default_pool_size = 25
reserve_pool_size = 5
EOF

    # Create user list
    echo "\"$APP_USER\" \"$DB_PASSWORD\"" > /etc/pgbouncer/userlist.txt
    chown postgres:postgres /etc/pgbouncer/userlist.txt
    chmod 600 /etc/pgbouncer/userlist.txt
    
    systemctl start pgbouncer
    systemctl enable pgbouncer
    print_status "PgBouncer installed and configured"
}

# Function to install Docker
install_docker() {
    print_status "Installing Docker..."
    # Install required packages
    apt install -y apt-transport-https ca-certificates curl gnupg lsb-release
    
    # Add Docker's official GPG key
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
    
    # Add Docker repository
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list
    
    # Update and install Docker
    apt update
    apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    # Add users to docker group
    usermod -aG docker $APP_USER
    
    # Start and enable Docker
    systemctl start docker
    systemctl enable docker
    
    print_status "Docker installed and configured"
}

# Function to install migration tool
install_migration_tool() {
    print_status "Installing database migration tool..."
    cd /tmp
    curl -L https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz | tar xvz
    mv migrate /usr/local/bin/migrate
    chmod +x /usr/local/bin/migrate
    print_status "Migration tool installed"
}

# Function to install Certbot
install_certbot() {
    print_status "Installing Certbot..."
    apt install -y certbot python3-certbot-nginx
    print_status "Certbot installed"
}

# Function to create Nginx configuration
create_nginx_config() {
    print_status "Creating Nginx configuration..."
    
    cat > /etc/nginx/sites-available/newsapp << EOF
server {
    listen 80;
    server_name $DOMAIN www.$DOMAIN;
    
    # Allow Certbot challenges
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }
    
    # Redirect all other HTTP traffic to HTTPS
    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN www.$DOMAIN;
    
    # SSL certificate paths (will be configured by Certbot)
    ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;
    
    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin";
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';";
    
    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml+rss application/json;
    
    # Static files
    location /static/ {
        alias /home/$APP_USER/news-website/web/static/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
    
    # Proxy to Go application
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }
    
    # Health check endpoint
    location /health {
        proxy_pass http://127.0.0.1:8080/health;
        access_log off;
    }
}
EOF

    # Enable the site
    ln -sf /etc/nginx/sites-available/newsapp /etc/nginx/sites-enabled/
    rm -f /etc/nginx/sites-enabled/default
    
    # Test configuration
    nginx -t
    systemctl reload nginx
    
    print_status "Nginx configuration created"
}

# Function to create application directory structure
create_app_structure() {
    print_status "Creating application directory structure..."
    
    sudo -u $APP_USER mkdir -p /home/$APP_USER/news-website/{cmd/server,internal,web/static,web/templates,migrations}
    sudo -u $APP_USER mkdir -p /home/$APP_USER/backups
    mkdir -p /var/log/newsapp
    chown $APP_USER:$APP_USER /var/log/newsapp
    
    print_status "Application directory structure created"
}

# Function to create environment file
create_env_file() {
    print_status "Creating environment configuration..."
    
    sudo -u $APP_USER tee /home/$APP_USER/news-website/.env.production << EOF
# Database Configuration
DB_HOST=127.0.0.1
DB_PORT=6432
DB_NAME=newsdb
DB_USER=$APP_USER
DB_PASSWORD=$DB_PASSWORD
DB_SSLMODE=disable

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=127.0.0.1
ENVIRONMENT=production

# Security
JWT_SECRET=$JWT_SECRET
CSRF_SECRET=$CSRF_SECRET

# Logging
LOG_LEVEL=info
LOG_FILE=/var/log/newsapp/app.log

# Cache Configuration
REDIS_URL=redis://127.0.0.1:6379/0

# Static Files
STATIC_DIR=/home/$APP_USER/news-website/web/static
TEMPLATES_DIR=/home/$APP_USER/news-website/web/templates
EOF

    chmod 600 /home/$APP_USER/news-website/.env.production
    print_status "Environment configuration created"
}

# Function to create systemd service
create_systemd_service() {
    print_status "Creating systemd service..."
    
    cat > /etc/systemd/system/newsapp.service << EOF
[Unit]
Description=News Website Application
After=network.target postgresql.service pgbouncer.service
Wants=postgresql.service pgbouncer.service

[Service]
Type=simple
User=$APP_USER
Group=$APP_USER
WorkingDirectory=/home/$APP_USER/news-website
ExecStart=/home/$APP_USER/news-website/news-server
EnvironmentFile=/home/$APP_USER/news-website/.env.production
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=newsapp

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/log/newsapp /tmp

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable newsapp
    
    print_status "Systemd service created"
}

# Function to create backup script
create_backup_script() {
    print_status "Creating backup script..."
    
    cat > /home/$APP_USER/backup.sh << 'EOF'
#!/bin/bash

# Configuration
BACKUP_DIR="/home/newsapp/backups"
DB_NAME="newsdb"
DB_USER="newsapp"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p $BACKUP_DIR

# Database backup
pg_dump -h 127.0.0.1 -p 5432 -U $DB_USER -d $DB_NAME -f "$BACKUP_DIR/db_backup_$DATE.sql"

# Compress backup
gzip "$BACKUP_DIR/db_backup_$DATE.sql"

# Keep only last 7 days of backups
find $BACKUP_DIR -name "db_backup_*.sql.gz" -mtime +7 -delete

echo "Backup completed: db_backup_$DATE.sql.gz"
EOF

    chmod +x /home/$APP_USER/backup.sh
    chown $APP_USER:$APP_USER /home/$APP_USER/backup.sh
    
    # Add to crontab
    sudo -u $APP_USER crontab -l 2>/dev/null | { cat; echo "0 2 * * * /home/$APP_USER/backup.sh"; } | sudo -u $APP_USER crontab -
    
    print_status "Backup script created and scheduled"
}

# Function to create logrotate configuration
create_logrotate() {
    print_status "Creating log rotation configuration..."
    
    cat > /etc/logrotate.d/newsapp << EOF
/var/log/newsapp/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 $APP_USER $APP_USER
    postrotate
        systemctl reload newsapp
    endscript
}
EOF

    print_status "Log rotation configured"
}

# Function to display final instructions
display_final_instructions() {
    print_status "Setup completed successfully!"
    echo
    print_warning "IMPORTANT: Complete these manual steps:"
    echo
    echo "1. Configure DNS A record for $DOMAIN to point to your server IP"
    echo "2. Wait for DNS propagation (use 'dig $DOMAIN' to check)"
    echo "3. Obtain SSL certificate:"
    echo "   sudo certbot --nginx -d $DOMAIN -d www.$DOMAIN"
    echo
    echo "4. Upload your application code to /home/$APP_USER/news-website/"
    echo "5. Build the application:"
    echo "   sudo -u $APP_USER -i"
    echo "   cd /home/$APP_USER/news-website"
    echo "   go build -o news-server ./cmd/server"
    echo
    echo "6. Run database migrations:"
    echo "   ./migrate -path ./migrations -database \"postgres://$APP_USER:$DB_PASSWORD@127.0.0.1:6432/newsdb?sslmode=disable\" up"
    echo
    echo "7. Start the application:"
    echo "   sudo systemctl start newsapp"
    echo
    echo "8. Check application status:"
    echo "   sudo systemctl status newsapp"
    echo
    print_status "Configuration files created:"
    echo "- Nginx config: /etc/nginx/sites-available/newsapp"
    echo "- Systemd service: /etc/systemd/system/newsapp.service"
    echo "- Environment file: /home/$APP_USER/news-website/.env.production"
    echo "- Backup script: /home/$APP_USER/backup.sh"
    echo
    print_warning "Remember to:"
    echo "- Change default passwords"
    echo "- Review and customize configuration files"
    echo "- Test all functionality after deployment"
    echo "- Set up monitoring and alerting"
}

# Main execution
main() {
    print_status "Starting production setup for News Website..."
    
    check_root
    validate_config
    
    update_system
    install_go
    install_docker
    install_postgresql
    install_nginx
    configure_firewall
    create_app_user
    configure_postgresql
    install_pgbouncer
    install_migration_tool
    install_certbot
    create_nginx_config
    create_app_structure
    create_env_file
    create_systemd_service
    create_backup_script
    create_logrotate
    
    display_final_instructions
}

# Check if script is being sourced or executed
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Script is being executed directly
    if [[ $# -eq 0 ]]; then
        print_error "Please edit the script and set the configuration variables at the top before running."
        print_error "Required variables: DOMAIN, DB_PASSWORD, JWT_SECRET, CSRF_SECRET"
        exit 1
    fi
    
    main "$@"
fi