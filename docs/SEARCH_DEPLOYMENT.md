# Search System Deployment Guide

## Prerequisites

- Go 1.21+
- PostgreSQL 14+ with pg_trgm extension
- Redis/DragonflyDB 6+
- MeiliSearch 1.12+ (optional but recommended)

## Environment Variables

Add these to your `.env.local` file:

```bash
# Search Engine Mode
NEWS_SEARCH_ENABLED=true

# MeiliSearch Configuration
NEWS_SEARCH_MEILISEARCH_URL=http://localhost:7700
NEWS_SEARCH_MEILISEARCH_API_KEY=your-master-key
NEWS_SEARCH_INDEX_NAME=articles

# PostgreSQL Configuration (already in main config)
NEWS_DATABASE_HOST=localhost
NEWS_DATABASE_PORT=5432
NEWS_DATABASE_USER=newsapp
NEWS_DATABASE_PASSWORD=your-password
NEWS_DATABASE_DBNAME=newsdb

# Redis/DragonflyDB Configuration (already in main config)
NEWS_CACHE_HOST=localhost
NEWS_CACHE_PORT=6379
```

## Quick Start

### Automated MeiliSearch Installation

```bash
# Run the installation script (as root)
sudo ./deploy/install-meilisearch.sh

# Or with a specific master key
sudo ./deploy/install-meilisearch.sh "your-secure-master-key"
```

The script will:
1. Download MeiliSearch v1.12.0
2. Create data directory
3. Set up systemd service
4. Start MeiliSearch
5. Output the configuration to add to `.env.local`

### Manual MeiliSearch Installation

```bash
# Download MeiliSearch binary
curl -L -o meilisearch https://github.com/meilisearch/meilisearch/releases/download/v1.12.0/meilisearch-linux-amd64
chmod +x meilisearch
sudo mv meilisearch /usr/local/bin/

# Create data directory
sudo mkdir -p /home/newsapp/meili_data
sudo chown newsapp:newsapp /home/newsapp/meili_data

# Create systemd service
sudo cp deploy/meilisearch.service /etc/systemd/system/
# Edit the service file to set your MEILI_MASTER_KEY

sudo systemctl daemon-reload
sudo systemctl enable meilisearch
sudo systemctl start meilisearch
```

## Deployment Without Docker

### 1. PostgreSQL Setup

```sql
-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Add search_vector column (if not exists)
ALTER TABLE articles ADD COLUMN IF NOT EXISTS search_vector tsvector
GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(excerpt, '')), 'B') ||
    setweight(to_tsvector('simple', coalesce(content, '')), 'C')
) STORED;

-- Create GIN index
CREATE INDEX IF NOT EXISTS idx_articles_search_vector 
ON articles USING gin(search_vector);

-- Create composite index for filters
CREATE INDEX IF NOT EXISTS idx_articles_search_filters 
ON articles(status, category_id, author_id, published_at DESC);
```

### 2. MeiliSearch Setup (Optional)

```bash
# Install MeiliSearch
curl -L https://install.meilisearch.com | sh

# Start MeiliSearch
./meilisearch --master-key="your-master-key" --env="production"

# Or as a systemd service
sudo tee /etc/systemd/system/meilisearch.service << EOF
[Unit]
Description=MeiliSearch
After=network.target

[Service]
Type=simple
User=meilisearch
ExecStart=/usr/local/bin/meilisearch --master-key="your-master-key" --env="production"
Restart=always

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable meilisearch
sudo systemctl start meilisearch
```

### 3. Application Deployment

```bash
# Build the application
CGO_ENABLED=1 go build -o newsapp ./cmd/server

# Run with environment variables
export SEARCH_ENGINE=hybrid
export MEILISEARCH_URL=http://localhost:7700
./newsapp
```

## Deployment With Docker

### docker-compose.yml

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SEARCH_ENGINE=hybrid
      - MEILISEARCH_URL=http://meilisearch:7700
      - MEILISEARCH_API_KEY=${MEILISEARCH_API_KEY}
      - DATABASE_URL=postgres://newsapp:password@postgres:5432/newsdb
      - REDIS_URL=redis://dragonfly:6379
    depends_on:
      - postgres
      - dragonfly
      - meilisearch

  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: newsapp
      POSTGRES_PASSWORD: password
      POSTGRES_DB: newsdb
    volumes:
      - postgres_data:/var/lib/postgresql/data

  dragonfly:
    image: docker.dragonflydb.io/dragonflydb/dragonfly:latest
    ports:
      - "6379:6379"
    volumes:
      - dragonfly_data:/data

  meilisearch:
    image: getmeili/meilisearch:v1.6
    environment:
      MEILI_MASTER_KEY: ${MEILISEARCH_API_KEY}
      MEILI_ENV: production
    volumes:
      - meilisearch_data:/meili_data

volumes:
  postgres_data:
  dragonfly_data:
  meilisearch_data:
```

## Initial Index Build

```bash
# Using CLI command
./newsapp search reindex --all

# Or via API (requires admin authentication)
curl -X POST http://localhost:8080/api/v1/admin/search/reindex \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Health Checks

```bash
# Basic health check
curl http://localhost:8080/api/v1/search/health

# Detailed health check
curl http://localhost:8080/api/v1/search/health/detailed

# Prometheus metrics
curl http://localhost:8080/metrics
```

## Monitoring Setup

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'newsapp'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### Grafana Dashboard

Import the provided dashboard from `docs/grafana/search-dashboard.json`

## Troubleshooting

### MeiliSearch Connection Issues

```bash
# Check MeiliSearch health
curl http://localhost:7700/health

# Check index status
curl http://localhost:7700/indexes/articles/stats \
  -H "Authorization: Bearer $MEILISEARCH_API_KEY"
```

### PostgreSQL Performance

```sql
-- Check index usage
EXPLAIN ANALYZE SELECT * FROM articles 
WHERE search_vector @@ plainto_tsquery('simple', 'test')
AND status = 'published'
LIMIT 20;

-- Should show "Index Scan using idx_articles_search_vector"
```

### Cache Issues

```bash
# Check Redis connection
redis-cli ping

# Check cache keys
redis-cli KEYS "search:*"

# Clear search cache
redis-cli KEYS "search:*" | xargs redis-cli DEL
```
