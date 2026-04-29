# High-Performance News Website

A fast, SEO-optimized news website built with Go, PostgreSQL, and Redis.

## Features

- ⚡ High-performance Go backend with Gin framework
- 🗄️ PostgreSQL with table partitioning for scalability
- 🚀 Redis caching for fast response times
- 🔍 Full-text search with MeiliSearch (optional)
- 📱 Responsive design with RTL support
- 🔐 JWT-based authentication
- 📊 Built-in analytics
- 🖼️ Automatic image optimization
- 📧 Email subscription system
- 🔔 Push notifications support

## Quick Start

### One-Command Installation (Production)

```bash
# On your server (Ubuntu/Debian)
curl -sSL https://raw.githubusercontent.com/YOUR_USERNAME/news-website/main/deployment/install.sh | sudo bash -s -- yourdomain.com admin@yourdomain.com
```

### Multi-Site Installation

Install multiple independent sites on one server:

```bash
./deployment/multi-site-install.sh site1.com 1 admin@site1.com
./deployment/multi-site-install.sh site2.com 2 admin@site2.com
./deployment/multi-site-install.sh site3.com 3 admin@site3.com
```

### Local Development

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/news-website.git
cd news-website

# Copy environment file
cp .env.example .env.local

# Start with Docker
docker-compose up -d

# Or run directly (requires Go 1.21+)
go run cmd/server/main.go
```

## Project Structure

```
.
├── cmd/                    # Application entry points
│   └── server/            # Main server application
├── configs/               # Configuration files
├── deployment/            # Docker and deployment scripts
├── internal/              # Private application code
│   ├── api/              # HTTP handlers
│   ├── auth/             # Authentication
│   ├── config/           # Configuration loading
│   ├── models/           # Data models
│   ├── repositories/     # Database access
│   ├── server/           # Server setup
│   ├── services/         # Business logic
│   └── templates/        # Template rendering
├── migrations/            # Database migrations
├── pkg/                   # Public packages
│   ├── cache/            # Cache utilities
│   └── database/         # Database utilities
├── scripts/              # Utility scripts
└── web/                  # Frontend assets
    ├── static/           # CSS, JS, images
    └── templates/        # HTML templates
```

## Configuration

Configuration is loaded from environment variables with `NEWS_` prefix:

```bash
# Server
NEWS_SERVER_PORT=8080
NEWS_SERVER_MODE=release

# Database
NEWS_DATABASE_HOST=localhost
NEWS_DATABASE_PORT=5432
NEWS_DATABASE_DBNAME=newsdb
NEWS_DATABASE_USER=newsapp
NEWS_DATABASE_PASSWORD=your_password

# Cache (Redis)
NEWS_CACHE_HOST=localhost
NEWS_CACHE_PORT=6379

# JWT
NEWS_JWT_SECRET=your_jwt_secret

# Admin
NEWS_ADMIN_EMAIL=admin@example.com
NEWS_ADMIN_PASSWORD=Admin123!
```

## API Endpoints

### Public
- `GET /` - Homepage
- `GET /article/:slug` - Article page
- `GET /category/:slug` - Category page
- `GET /tag/:slug` - Tag page
- `GET /search?q=query` - Search
- `GET /api/articles` - List articles (JSON)

### Admin (requires authentication)
- `GET /admin/login` - Login page
- `GET /admin/dashboard` - Dashboard
- `GET /admin/articles` - Manage articles
- `GET /admin/categories` - Manage categories
- `GET /admin/settings` - Site settings

## Deployment

See [deployment/README.md](deployment/README.md) for detailed deployment instructions.

### Requirements
- Ubuntu 20.04+ or Debian 11+
- 2GB RAM minimum (4GB recommended)
- Docker and Docker Compose
- Domain with DNS pointing to server

## License

MIT License - see [LICENSE](LICENSE) file.
