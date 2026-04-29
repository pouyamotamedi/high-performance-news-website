# Search System Scaling Guide

## Resource Estimation

### Memory Requirements

| Article Count | PostgreSQL | MeiliSearch | Redis Cache | Total RAM |
|---------------|------------|-------------|-------------|-----------|
| 100K          | 2 GB       | 1 GB        | 512 MB      | 4 GB      |
| 500K          | 8 GB       | 4 GB        | 2 GB        | 16 GB     |
| 1M            | 16 GB      | 8 GB        | 4 GB        | 32 GB     |
| 5M            | 64 GB      | 32 GB       | 16 GB       | 128 GB    |

### CPU Requirements

| Concurrent Searches | CPU Cores (App) | CPU Cores (DB) | CPU Cores (Meili) |
|---------------------|-----------------|----------------|-------------------|
| 100                 | 2               | 2              | 2                 |
| 500                 | 4               | 4              | 4                 |
| 1000                | 8               | 8              | 8                 |
| 5000                | 16              | 16             | 16                |

### Storage Requirements

| Article Count | PostgreSQL | MeiliSearch Index | Total Storage |
|---------------|------------|-------------------|---------------|
| 100K          | 5 GB       | 2 GB              | 10 GB         |
| 500K          | 25 GB      | 10 GB             | 50 GB         |
| 1M            | 50 GB      | 20 GB             | 100 GB        |
| 5M            | 250 GB     | 100 GB            | 500 GB        |

## Horizontal Scaling

### Application Layer

```
                    ┌─────────────────┐
                    │  Load Balancer  │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
         ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│   App Node 1    │ │   App Node 2    │ │   App Node 3    │
│   (Search)      │ │   (Search)      │ │   (Search)      │
└─────────────────┘ └─────────────────┘ └─────────────────┘
         │                   │                   │
         └───────────────────┼───────────────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
         ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  Redis Cluster  │ │   PostgreSQL    │ │   MeiliSearch   │
│   (3 nodes)     │ │   (Primary +    │ │   (Primary +    │
│                 │ │    Replicas)    │ │    Replicas)    │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

### PostgreSQL Scaling

1. **Read Replicas**: Route search queries to read replicas
2. **Connection Pooling**: Use PgBouncer with 200+ connections
3. **Partitioning**: Partition articles by published_at date

```sql
-- Create partitioned table
CREATE TABLE articles_partitioned (
    LIKE articles INCLUDING ALL
) PARTITION BY RANGE (published_at);

-- Create monthly partitions
CREATE TABLE articles_2024_01 PARTITION OF articles_partitioned
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

### MeiliSearch Scaling

1. **Sharding**: Split index by category or language
2. **Replicas**: Use MeiliSearch Cloud for automatic scaling
3. **Dedicated Nodes**: Separate indexing from search nodes

### Redis/DragonflyDB Scaling

1. **Cluster Mode**: 3+ nodes for high availability
2. **Memory Optimization**: Use hash compression
3. **Eviction Policy**: allkeys-lru for cache

```bash
# DragonflyDB cluster setup
dragonfly --cluster_mode=emulated --cluster_announce_ip=node1
```

## Performance Optimization

### Query Optimization

```sql
-- Ensure proper index usage
CREATE INDEX CONCURRENTLY idx_articles_search_optimized 
ON articles USING gin(search_vector) 
WHERE status = 'published';

-- Partial index for recent articles
CREATE INDEX CONCURRENTLY idx_articles_recent_search 
ON articles USING gin(search_vector) 
WHERE status = 'published' 
AND published_at > NOW() - INTERVAL '30 days';
```

### Cache Optimization

```go
// Increase cache TTL for stable queries
config.CacheTTL = 15 * time.Minute

// Use cache warming for popular queries
func WarmCache(popularQueries []string) {
    for _, query := range popularQueries {
        go searchService.Search(SearchRequest{Query: query})
    }
}
```

### MeiliSearch Optimization

```json
{
  "rankingRules": [
    "words",
    "typo",
    "proximity",
    "attribute",
    "published_at:desc",
    "view_count:desc",
    "exactness"
  ],
  "typoTolerance": {
    "enabled": true,
    "minWordSizeForTypos": {
      "oneTypo": 4,
      "twoTypos": 8
    }
  }
}
```

## Load Testing

### k6 Load Test Script

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '1m', target: 100 },  // Ramp up
    { duration: '5m', target: 500 },  // Sustained load
    { duration: '1m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<150', 'p(99)<300'],
    http_req_failed: ['rate<0.01'],
  },
};

const queries = ['news', 'technology', 'sports', 'politics', 'health'];

export default function () {
  const query = queries[Math.floor(Math.random() * queries.length)];
  const res = http.get(`http://localhost:8080/api/v1/search?q=${query}`);
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 150ms': (r) => r.timings.duration < 150,
  });
  
  sleep(0.1);
}
```

### Running Load Tests

```bash
# Install k6
brew install k6

# Run load test
k6 run load-test.js

# Run with more VUs
k6 run --vus 500 --duration 5m load-test.js
```

## Monitoring Alerts

### Prometheus Alert Rules

```yaml
groups:
  - name: search
    rules:
      - alert: SearchLatencyHigh
        expr: histogram_quantile(0.95, search_latency_seconds) > 0.15
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Search P95 latency above 150ms"

      - alert: SearchErrorRateHigh
        expr: rate(search_requests_total{status="error"}[5m]) > 0.01
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Search error rate above 1%"

      - alert: CircuitBreakerOpen
        expr: search_circuit_breaker_open == 1
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "MeiliSearch circuit breaker is open"

      - alert: DeadLetterQueueGrowing
        expr: search_dead_letter_queue_size > 100
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Dead letter queue has more than 100 items"
```

## Operational Checklist

### Pre-Go-Live

- [ ] PostgreSQL search_vector column created
- [ ] GIN index created and verified
- [ ] MeiliSearch index configured (if using)
- [ ] Redis/DragonflyDB cluster healthy
- [ ] Initial index build completed
- [ ] Load testing passed (P95 < 150ms)
- [ ] Monitoring dashboards configured
- [ ] Alert rules configured
- [ ] Backup strategy verified
- [ ] Rollback plan documented

### Daily Operations

- [ ] Check search health endpoint
- [ ] Review P95/P99 latencies
- [ ] Check dead letter queue size
- [ ] Verify cache hit ratio > 80%
- [ ] Review error rates

### Weekly Operations

- [ ] Review index reconciliation results
- [ ] Check disk usage trends
- [ ] Review slow query logs
- [ ] Update popular query cache warming list
