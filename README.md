# High Performance News Website

A ultra-high-performance news website built with Go, capable of handling 50,000+ daily articles with exceptional speed, SEO optimization, and scalability.

## Features

- **High Performance**: Sub-2-second page loads, optimized for 50K+ daily articles
- **SEO Optimized**: Structured data, fast indexing, AI-ready content
- **Mobile-First**: Progressive enhancement, Core Web Vitals compliance
- **Security Focused**: Comprehensive protection against web vulnerabilities
- **Easy Deployment**: Zero-touch deployment from desktop to server
- **Modular Architecture**: Extensible plugin system for future features

## Technology Stack

- **Backend**: Go with Gin framework
- **Database**: PostgreSQL 15+ with partitioning
- **Cache**: DragonflyDB (Redis-compatible)
- **Connection Pooling**: PgBouncer
- **Search**: MeiliSearch (planned)
- **Frontend**: Server-side rendering with static generation

## Quick Start

### Prerequisites

- **Go 1.21 or higher** - [Download and install Go](https://golang.org/dl/)
- **Docker Desktop** - [Download Docker Desktop for Windows](https://docs.docker.com/desktop/windows/)
- **Make** (optional) - For Windows: [Install via Chocolatey](https://chocolatey.org/packages/make) or use WSL

### Installing Go on Windows

1. Download Go from https://golang.org/dl/
2. Run the installer (.msi file)
3. Restart your terminal/PowerShell
4. Verify installation: `go version`

If `go` command is not recognized, you may need to:
- Restart your computer
- Add Go to your PATH manually: `C:\Program Files\Go\bin`

### Development Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd high-performance-news-website
   ```

2. **Install development tools**
   ```bash
   make install-tools
   ```

3. **Set up environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Start development services**
   ```bash
   make docker-up
   ```

5. **Run database migrations**
   ```bash
   make migrate-up
   ```

6. **Start development server with hot reload**
   ```bash
   make dev
   ```

The application will be available at `http://localhost:8080`

### Available Services

- **Application**: http://localhost:8080
- **Database Admin (Adminer)**: http://localhost:8081
- **PostgreSQL**: localhost:5432
- **PgBouncer**: localhost:6432
- **DragonflyDB**: localhost:6379

## Project Structure

```
├── cmd/                    # Application entrypoints
│   └── server/            # Main server application
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   └── server/           # HTTP server setup
├── pkg/                   # Public library code
│   ├── database/         # Database connection utilities
│   └── cache/            # Cache client utilities
├── web/                   # Web assets and templates
│   ├── static/           # Static files (CSS, JS, images)
│   └── templates/        # HTML templates
├── migrations/            # Database migrations
├── configs/              # Configuration files
├── scripts/              # Utility scripts
└── docs/                 # Documentation
```

## Development Commands

```bash
# Development
make dev                   # Start development server with hot reload
make run                   # Run application without hot reload
make build                 # Build application binary

# Quality Assurance
make test                  # Run tests
make test-coverage         # Run tests with coverage report
make lint                  # Run linter
make fmt                   # Format code
make vet                   # Run go vet
make quality              # Run all quality checks

# Database
make migrate-up           # Run migrations up
make migrate-down         # Run migrations down
make migrate-create NAME=migration_name  # Create new migration

# Docker
make docker-up            # Start Docker services
make docker-down          # Stop Docker services
make docker-logs          # Show Docker logs

# Setup
make dev-setup            # Complete development environment setup
make deps                 # Install/update dependencies
make install-tools        # Install development tools
```

## Configuration

The application uses a hierarchical configuration system:

1. **Default values** (in code)
2. **Configuration file** (`configs/config.yaml`)
3. **Environment variables** (prefixed with `NEWS_`)

### Environment Variables

All configuration can be overridden with environment variables:

```bash
NEWS_SERVER_HOST=0.0.0.0
NEWS_SERVER_PORT=8080
NEWS_DATABASE_HOST=localhost
NEWS_DATABASE_PORT=5432
# ... see .env.example for full list
```

## Database

The application uses PostgreSQL with the following optimizations:

- **Partitioning**: Articles table partitioned by publication date
- **Connection Pooling**: PgBouncer for efficient connection management
- **Indexing**: Optimized indexes including BRIN for time-series data
- **Extensions**: pg_trgm for full-text search, uuid-ossp for UUIDs

## Caching

DragonflyDB (Redis-compatible) is used for high-performance caching:

- **Multi-layer caching**: Application, database query, and static file caching
- **Intelligent invalidation**: Cache invalidation based on content relationships
- **High throughput**: Optimized for 50K+ articles per day

## Performance Targets

- **Page Load Time**: < 2 seconds on 3G connections
- **Article Publishing**: < 1 second per article
- **Homepage Load**: < 500ms (cached), < 2 seconds (dynamic)
- **Search Response**: < 200ms
- **API Response**: < 100ms
- **Database Queries**: < 10ms (indexed queries)
- **Concurrent Users**: 10,000+ simultaneous users
- **Daily Articles**: 50,000+ articles per day
- **Peak Publishing**: 1,000 articles per minute

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run quality checks: `make quality`
5. Submit a pull request

## License

[License information to be added]

## Support

[Support information to be added]