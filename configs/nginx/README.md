# Nginx Configuration for Static-First Serving

This directory contains nginx configuration files for implementing static-first serving with dynamic fallback.

## Files

- `static-first.conf` - Main nginx configuration for static-first serving

## Configuration Overview

The nginx configuration implements a static-first strategy where:

1. **Static Assets** - Served directly with long cache headers
2. **Homepage** - Tries static HTML first, falls back to dynamic
3. **Article Pages** - Tries static HTML first, falls back to dynamic
4. **Category/Tag Pages** - Tries static HTML with pagination support
5. **API Endpoints** - Always served dynamically with rate limiting
6. **Admin Endpoints** - Always served dynamically with strict rate limiting

## Key Features

### Static-First Strategy
- Static files are served directly from `/static-html/` directory
- Dynamic fallback to Go application when static files don't exist
- Proper cache headers for both static and dynamic content

### Rate Limiting
- API endpoints: 10 requests/second
- Static content: 100 requests/second
- Admin endpoints: 2 requests/second (stricter)

### Performance Optimizations
- Gzip compression for all text content
- Long cache headers for static assets (1 year)
- Shorter cache headers for dynamic content (5-30 minutes)
- Connection keepalive and pooling

### Security Headers
- X-Frame-Options: SAMEORIGIN
- X-Content-Type-Options: nosniff
- X-XSS-Protection: 1; mode=block
- Referrer-Policy: strict-origin-when-cross-origin

## Directory Structure

The nginx configuration expects the following directory structure:

```
/var/www/
├── static/                 # Static assets (CSS, JS, images)
├── static-html/           # Generated static HTML files
│   ├── index.html         # Homepage (default language)
│   ├── en/
│   │   └── index.html     # English homepage
│   ├── ar/
│   │   └── index.html     # Arabic homepage
│   ├── articles/
│   │   └── [slug]/
│   │       └── index.html # Article pages
│   ├── categories/
│   │   └── [slug]/
│   │       ├── index.html      # Category page 1
│   │       └── page-[n].html   # Category pagination
│   └── tags/
│       └── [slug]/
│           ├── index.html      # Tag page 1
│           └── page-[n].html   # Tag pagination
└── error-pages/           # Custom error pages
    ├── 404.html
    └── 50x.html
```

## Usage

1. Copy `static-first.conf` to your nginx configuration directory
2. Update the `upstream backend` section with your Go application address
3. Update file paths to match your deployment structure
4. Test the configuration: `nginx -t`
5. Reload nginx: `systemctl reload nginx`

## Monitoring

The configuration includes:
- Access logging for all requests
- Separate error logging
- Health check endpoint at `/health`
- Headers to identify static vs dynamic serving

## SSL/HTTPS

The configuration includes commented SSL configuration. Uncomment and configure:
- SSL certificate paths
- SSL protocols and ciphers
- Redirect HTTP to HTTPS if needed