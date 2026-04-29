# News Website - Docker Deployment

## Quick Installation

### Prerequisites
- A server with Ubuntu 20.04+ or Debian 11+
- A domain name pointing to your server's IP
- Root access to the server

### One-Command Installation

```bash
# On your server, run:
curl -sSL https://raw.githubusercontent.com/your-repo/install.sh | sudo bash -s -- yourdomain.com your@email.com
```

Or manually:

```bash
# 1. Clone/upload the project to your server
scp -r . root@your-server:/opt/news-website/

# 2. SSH into your server
ssh root@your-server

# 3. Run the installation script
cd /opt/news-website/deployment
chmod +x install.sh
./install.sh yourdomain.com your@email.com
```

## What Gets Installed

- **PostgreSQL 15**: Database
- **Dragonfly**: Redis-compatible cache (faster than Redis)
- **Go Application**: The news website
- **Nginx**: Reverse proxy with SSL
- **Certbot**: Automatic SSL certificates

## Management Commands

After installation, these scripts are available:

```bash
# Check status
/opt/news-website/status.sh

# Update the application
/opt/news-website/update.sh

# Create backup
/opt/news-website/backup.sh

# View logs
cd /opt/news-website/deployment && docker compose logs -f

# Restart all services
cd /opt/news-website/deployment && docker compose restart

# Stop all services
cd /opt/news-website/deployment && docker compose down

# Start all services
cd /opt/news-website/deployment && docker compose up -d
```

## Updating the Application

When you have new code changes:

1. Upload new files to `/opt/news-website/`
2. Run `/opt/news-website/update.sh`

The update script will:
- Create a backup
- Rebuild the application
- Restart services
- Keep your data intact

## Backup & Restore

### Create Backup
```bash
/opt/news-website/backup.sh
```

Backups are saved to `/opt/news-website/backups/`

### Restore Database
```bash
cd /opt/news-website/deployment
docker compose exec -T postgres psql -U newsapp newsdb < /path/to/backup.sql
```

## SSL Certificate

SSL certificates are automatically renewed via cron job.
To manually renew:

```bash
cd /opt/news-website/deployment
docker compose run --rm certbot renew
docker compose restart nginx
```

## Troubleshooting

### Check service status
```bash
cd /opt/news-website/deployment
docker compose ps
```

### View logs
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f app
docker compose logs -f postgres
docker compose logs -f nginx
```

### Restart a service
```bash
docker compose restart app
```

### Database connection
```bash
docker compose exec postgres psql -U newsapp newsdb
```

## Environment Variables

Configuration is stored in `/opt/news-website/deployment/.env`

Key variables:
- `DOMAIN`: Your domain name
- `DB_PASSWORD`: Database password
- `JWT_SECRET`: JWT signing secret
- `ADMIN_EMAIL`: Admin login email
- `ADMIN_PASSWORD`: Admin login password

## Ports Used

- 80: HTTP (redirects to HTTPS)
- 443: HTTPS
- 5432: PostgreSQL (internal only)
- 6379: Redis (internal only)
