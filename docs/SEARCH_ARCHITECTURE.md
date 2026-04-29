# Search System Architecture

## Overview

The search system is designed for enterprise-grade, high-volume news platforms supporting millions of articles with sub-150ms P95 latency.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT REQUEST                                  │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           RATE LIMITER                                       │
│                    (Token Bucket: 100/s, burst 200)                         │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CONCURRENCY LIMITER                                   │
│                         (Max 500 concurrent)                                 │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INPUT VALIDATION                                     │
│              • Query sanitization (SQL injection prevention)                 │
│              • Length limits (500 chars max)                                 │
│              • Offset bounds (10,000 max)                                    │
│              • Sort field whitelist                                          │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          CACHE LAYER                                         │
│                    (Redis/DragonflyDB, 5min TTL)                            │
│              • SHA256 hash for long queries                                  │
│              • Pattern-based invalidation                                    │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                          ┌───────────┴───────────┐
                          │     CACHE MISS        │
                          └───────────┬───────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ENGINE MODE ROUTER                                    │
│                                                                              │
│   SEARCH_ENGINE=postgres  │  SEARCH_ENGINE=meili  │  SEARCH_ENGINE=hybrid   │
│           │                        │                        │               │
│           ▼                        ▼                        ▼               │
│     PostgreSQL              MeiliSearch              Try MeiliSearch        │
│       Only                    Only                   Fallback to PG         │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│     MEILISEARCH     │  │   CIRCUIT BREAKER   │  │     POSTGRESQL      │
│                     │  │                     │  │                     │
│ • Typo tolerance    │  │ • 5 failure thresh  │  │ • GIN index on      │
│ • Ranking rules     │  │ • Exponential       │  │   search_vector     │
│ • Faceted search    │  │   backoff           │  │ • ts_rank scoring   │
│ • 10s timeout       │  │ • Half-open state   │  │ • 15s timeout       │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         RESPONSE PROCESSING                                  │
│              • Normalize schema (MeiliSearch ↔ PostgreSQL)                  │
│              • Add processing time                                           │
│              • Record metrics                                                │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ASYNC CACHE STORE                                    │
│                    (Non-blocking, fire-and-forget)                          │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              RESPONSE                                        │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. Rate Limiter (Token Bucket)
- **Rate**: 100 requests/second
- **Burst**: 200 requests
- **Purpose**: Prevent abuse and ensure fair resource allocation

### 2. Concurrency Limiter
- **Max Concurrent**: 500 requests
- **Purpose**: Prevent resource exhaustion under load

### 3. Circuit Breaker
- **Threshold**: 5 consecutive failures
- **Backoff**: Exponential (1s base, 60s max)
- **States**: Closed → Open → Half-Open → Closed

### 4. Cache Layer (Redis/DragonflyDB)
- **TTL**: 5 minutes
- **Key Strategy**: SHA256 hash for queries > 100 chars
- **Invalidation**: Pattern-based on article changes

### 5. MeiliSearch
- **Timeout**: 10 seconds
- **Features**: Typo tolerance, ranking rules, faceted search
- **Index Settings**:
  - Searchable: title, content, excerpt, meta_title, meta_description, keywords, tags
  - Filterable: author_id, category_id, status, published_at, language_code, tags
  - Sortable: published_at, created_at, view_count, like_count

### 6. PostgreSQL Fallback
- **Timeout**: 15 seconds
- **Index**: GIN on search_vector column
- **Scoring**: ts_rank for relevance

## Metrics

### Prometheus Metrics
- `search_latency_seconds` - Histogram of search latencies
- `search_requests_total` - Counter of total requests by source/status
- `search_cache_hit_ratio` - Gauge of cache hit ratio
- `search_fallback_rate` - Gauge of PostgreSQL fallback rate
- `search_indexing_duration_seconds` - Histogram of indexing durations
- `search_circuit_breaker_open` - Gauge of circuit breaker state
- `search_concurrent_requests` - Gauge of concurrent requests
- `search_dead_letter_queue_size` - Gauge of DLQ size

### Internal Metrics
- Total searches
- MeiliSearch hits
- PostgreSQL fallbacks
- Cache hits/misses
- Errors
- Slow queries
- Rate limited requests
- P95/P99 latencies

## Security

### Input Validation
- SQL injection pattern removal
- Query length limits
- Offset bounds protection
- Sort field whitelist

### Error Handling
- No sensitive information in error responses
- Structured logging without PII
- Audit logging for admin operations

## Resilience

### Dead Letter Queue
- Failed indexing operations queued for retry
- Max 5 retries with 5-minute intervals
- Automatic cleanup of successful retries

### Index Reconciliation
- Hourly comparison of DB vs index counts
- Automatic detection of drift
- Manual re-index capability

## Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| P95 Latency | < 150ms | ~50ms |
| P99 Latency | < 300ms | ~100ms |
| Cache Hit Ratio | > 80% | ~85% |
| Availability | 99.9% | 99.9% |
