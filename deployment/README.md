# Deployment Guide

## Quick Start - New Site Installation

```bash
cd deployment
chmod +x install-site.sh
./install-site.sh <domain> <app-port> <db-port> <redis-port> [admin-email]
```

### Examples:

```bash
./install-site.sh enginosys.com 8081 5433 6381 admin@enginosys.com
./install-site.sh cryptonlisys.com 8082 5434 6382 admin@cryptonlisys.com
./install-site.sh technolisys.com 8083 5435 6383 admin@technolisys.com
```

### Port Allocation:

| Site | App Port | DB Port | Redis Port |
|------|----------|---------|------------|
| 1    | 8081     | 5433    | 6381       |
| 2    | 8082     | 5434    | 6382       |
| 3    | 8083     | 5435    | 6383       |

## Updating an Existing Site

```bash
cd /opt/<sitename>-website/deployment
chmod +x update.sh
./update.sh          # Quick update (rebuild app only)
./update.sh --rebuild  # Full rebuild (no cache)
```

## Docker Commands

Each site uses its own project name for isolation:

```bash
# Status
docker compose -p <sitename> ps

# Logs
docker compose -p <sitename> logs -f app

# Restart
docker compose -p <sitename> restart

# Stop
docker compose -p <sitename> down

# Start
docker compose -p <sitename> up -d
```

## Files

| File | Description |
|------|-------------|
| `install-site.sh` | Install a new site (creates everything from scratch) |
| `update.sh` | Update existing site code without touching configs |
| `generate-init-db.sh` | Export schema from production to update init-db.sql |
| `init-db.sql` | Complete database schema (auto-runs on first start) |
| `docker-compose.yml` | Docker services configuration (site-specific) |
| `Dockerfile` | Application build instructions |
| `.env` | Site-specific environment variables (auto-generated) |

## Database Schema Updates

If you change the database schema:

1. Make changes on a running site
2. Export the updated schema:
   ```bash
   ./generate-init-db.sh <sitename>
   ```
3. Commit and push:
   ```bash
   git add deployment/init-db.sql
   git commit -m "Update database schema"
   git push origin master
   ```

## Important Notes

- Each site has its own isolated Docker network, volumes, and containers
- `docker-compose.yml` and `.env` are site-specific and should NOT be overwritten during updates
- The `update.sh` script handles this automatically
- Always use `-p <sitename>` flag with docker compose commands
