import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Database performance metrics
const dbConnectionTime = new Trend('db_connection_time');
const queryExecutionTime = new Trend('query_execution_time');
const connectionPoolUtilization = new Trend('connection_pool_utilization');
const slowQueryCount = new Counter('slow_queries');
const connectionErrors = new Counter('connection_errors');
const deadlockCount = new Counter('deadlocks');

export const options = {
  scenarios: {
    // Test connection pool limits
    connection_pool_stress: {
      executor: 'constant-vus',
      vus: 200, // More than max connections (150)
      duration: '5m',
      tags: { test_type: 'connection_pool' },
    },
    
    // Test query performance under load
    query_performance: {
      executor: 'constant-vus',
      vus: 100,
      duration: '10m',
      tags: { test_type: 'query_performance' },
    },
    
    // Test concurrent writes (potential bottleneck)
    concurrent_writes: {
      executor: 'constant-arrival-rate',
      rate: 50, // 50 writes per minute
      timeUnit: '1m',
      duration: '5m',
      preAllocatedVUs: 10,
      maxVUs: 25,
      tags: { test_type: 'concurrent_writes' },
    },
    
    // Test partition performance
    partition_queries: {
      executor: 'constant-vus',
      vus: 50,
      duration: '5m',
      tags: { test_type: 'partition_queries' },
    },
  },
  
  thresholds: {
    // Database performance requirements
    'db_connection_time': ['p(95)<50'], // Connection under 50ms
    'query_execution_time': ['p(95)<10'], // Queries under 10ms
    'query_execution_time{query_type:indexed}': ['p(95)<5'], // Indexed queries under 5ms
    'query_execution_time{query_type:complex}': ['p(95)<100'], // Complex queries under 100ms
    'connection_pool_utilization': ['p(95)<0.8'], // Pool utilization under 80%
    'slow_queries': ['count<10'], // Less than 10 slow queries
    'connection_errors': ['count<5'], // Less than 5 connection errors
    'deadlocks': ['count<1'], // No deadlocks
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Database test queries categorized by complexity
const testQueries = {
  simple: [
    '/api/v1/articles/1',
    '/api/v1/categories/1',
    '/api/v1/tags/1',
    '/api/v1/users/1',
  ],
  
  indexed: [
    '/api/v1/articles?category=1&limit=10',
    '/api/v1/articles?author=1&limit=10',
    '/api/v1/articles?status=published&limit=20',
    '/api/v1/tags?limit=50',
  ],
  
  complex: [
    '/api/v1/articles?category=1&tags=1,2,3&limit=20',
    '/api/v1/search?q=programming&category=1&limit=10',
    '/api/v1/articles/trending?limit=10',
    '/api/v1/articles/popular?timeframe=week&limit=10',
    '/api/v1/categories/1/articles?include_children=true&limit=20',
  ],
  
  aggregation: [
    '/api/v1/analytics/articles/stats',
    '/api/v1/analytics/categories/performance',
    '/api/v1/analytics/tags/usage',
    '/api/v1/system/metrics',
  ],
};

function authenticate() {
  const loginData = {
    username: __ENV.TEST_USERNAME || 'testuser',
    password: __ENV.TEST_PASSWORD || 'testpass123',
  };
  
  const response = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify(loginData), {
    headers: { 'Content-Type': 'application/json' },
    tags: { endpoint: 'auth' },
  });
  
  if (response.status === 200) {
    try {
      return JSON.parse(response.body).token;
    } catch (e) {
      return null;
    }
  }
  return null;
}

export default function() {
  const scenario = __ENV.K6_SCENARIO || 'query_performance';
  
  switch (scenario) {
    case 'connection_pool':
      testConnectionPool();
      break;
    case 'concurrent_writes':
      testConcurrentWrites();
      break;
    case 'partition_queries':
      testPartitionQueries();
      break;
    default:
      testQueryPerformance();
  }
}

function testConnectionPool() {
  const token = authenticate();
  if (!token) {
    connectionErrors.add(1);
    return;
  }
  
  const headers = { 'Authorization': `Bearer ${token}` };
  
  // Test multiple simultaneous connections
  const connectionStart = Date.now();
  
  // Make multiple concurrent requests to stress connection pool
  const requests = [
    http.get(`${BASE_URL}/api/v1/articles?limit=10`, { headers }),
    http.get(`${BASE_URL}/api/v1/categories`, { headers }),
    http.get(`${BASE_URL}/api/v1/tags?limit=20`, { headers }),
  ];
  
  const connectionTime = Date.now() - connectionStart;
  dbConnectionTime.add(connectionTime);
  
  // Check for connection pool exhaustion
  const poolResponse = http.get(`${BASE_URL}/api/v1/system/db-pool-stats`, { headers });
  if (poolResponse.status === 200) {
    try {
      const poolStats = JSON.parse(poolResponse.body);
      const utilization = poolStats.active_connections / poolStats.max_connections;
      connectionPoolUtilization.add(utilization);
      
      check(poolStats, {
        'connection pool not exhausted': (stats) => stats.active_connections < stats.max_connections,
        'connection pool utilization reasonable': (stats) => utilization < 0.9,
      });
    } catch (e) {
      console.log('Could not parse pool stats:', e);
    }
  }
  
  requests.forEach((response, index) => {
    check(response, {
      [`request ${index} successful`]: (r) => r.status === 200,
      [`request ${index} no connection timeout`]: (r) => r.status !== 504,
    });
    
    if (response.status >= 500) {
      connectionErrors.add(1);
    }
  });
  
  sleep(0.1);
}

function testQueryPerformance() {
  const token = authenticate();
  if (!token) {
    return;
  }
  
  const headers = { 'Authorization': `Bearer ${token}` };
  
  // Test different types of queries
  Object.keys(testQueries).forEach(queryType => {
    const queries = testQueries[queryType];
    const query = queries[Math.floor(Math.random() * queries.length)];
    
    const queryStart = Date.now();
    const response = http.get(`${BASE_URL}${query}`, {
      headers,
      tags: { query_type: queryType },
    });
    const queryTime = Date.now() - queryStart;
    
    queryExecutionTime.add(queryTime, { query_type: queryType });
    
    // Track slow queries
    if (queryTime > 100) {
      slowQueryCount.add(1, { query_type: queryType });
    }
    
    check(response, {
      [`${queryType} query successful`]: (r) => r.status === 200,
      [`${queryType} query fast enough`]: (r) => {
        const maxTime = queryType === 'complex' ? 100 : queryType === 'aggregation' ? 500 : 10;
        return r.timings.duration < maxTime;
      },
    });
    
    // Check for database errors in response
    if (response.status === 500) {
      try {
        const body = JSON.parse(response.body);
        if (body.error && body.error.includes('deadlock')) {
          deadlockCount.add(1);
        }
      } catch (e) {
        // Ignore parsing errors
      }
    }
  });
  
  sleep(0.5);
}

function testConcurrentWrites() {
  const token = authenticate();
  if (!token) {
    return;
  }
  
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };
  
  // Test concurrent article creation (write operations)
  const article = {
    title: `Load Test Article ${Date.now()}`,
    content: 'This is a test article for database load testing.',
    excerpt: 'Test article excerpt',
    author_id: 1,
    category_id: 1,
    status: 'published',
    tags: [1, 2],
  };
  
  const writeStart = Date.now();
  const response = http.post(`${BASE_URL}/api/v1/articles`, JSON.stringify(article), {
    headers,
    tags: { operation: 'write', query_type: 'insert' },
  });
  const writeTime = Date.now() - writeStart;
  
  queryExecutionTime.add(writeTime, { query_type: 'insert' });
  
  check(response, {
    'concurrent write successful': (r) => r.status === 201,
    'concurrent write fast': (r) => r.timings.duration < 1000,
    'no write conflicts': (r) => r.status !== 409,
  });
  
  if (response.status === 201) {
    // Test immediate read consistency
    try {
      const created = JSON.parse(response.body);
      const readResponse = http.get(`${BASE_URL}/api/v1/articles/${created.id}`, {
        headers: { 'Authorization': `Bearer ${token}` },
        tags: { operation: 'read_after_write' },
      });
      
      check(readResponse, {
        'read after write consistent': (r) => r.status === 200,
      });
    } catch (e) {
      console.log('Error testing read consistency:', e);
    }
  }
  
  sleep(0.2);
}

function testPartitionQueries() {
  const token = authenticate();
  if (!token) {
    return;
  }
  
  const headers = { 'Authorization': `Bearer ${token}` };
  
  // Test queries that should benefit from partitioning
  const partitionQueries = [
    // Recent articles (should hit current partition)
    '/api/v1/articles?published_after=2024-01-01&limit=20',
    // Older articles (should hit older partitions)
    '/api/v1/articles?published_before=2023-12-31&limit=20',
    // Date range queries (might span partitions)
    '/api/v1/articles?published_after=2023-12-01&published_before=2024-01-31&limit=50',
  ];
  
  partitionQueries.forEach((query, index) => {
    const queryStart = Date.now();
    const response = http.get(`${BASE_URL}${query}`, {
      headers,
      tags: { query_type: 'partition', partition_test: `test_${index}` },
    });
    const queryTime = Date.now() - queryStart;
    
    queryExecutionTime.add(queryTime, { query_type: 'partition' });
    
    check(response, {
      [`partition query ${index} successful`]: (r) => r.status === 200,
      [`partition query ${index} uses pruning`]: (r) => r.timings.duration < 50, // Should be fast with pruning
    });
    
    // Check if query plan shows partition pruning (if available)
    const explainResponse = http.get(`${BASE_URL}/api/v1/system/explain?query=${encodeURIComponent(query)}`, {
      headers,
    });
    
    if (explainResponse.status === 200) {
      try {
        const plan = JSON.parse(explainResponse.body);
        check(plan, {
          [`partition query ${index} uses pruning in plan`]: (p) => 
            p.plan && p.plan.includes('Partition') && !p.plan.includes('Seq Scan'),
        });
      } catch (e) {
        // Ignore if explain endpoint not available
      }
    }
  });
  
  sleep(0.3);
}

export function setup() {
  console.log('Setting up database bottleneck tests...');
  
  // Verify database connectivity
  const healthResponse = http.get(`${BASE_URL}/health`);
  if (healthResponse.status !== 200) {
    throw new Error(`Database health check failed: ${healthResponse.status}`);
  }
  
  // Verify authentication
  const token = authenticate();
  if (!token) {
    throw new Error('Authentication failed during setup');
  }
  
  // Check database pool status
  const poolResponse = http.get(`${BASE_URL}/api/v1/system/db-pool-stats`, {
    headers: { 'Authorization': `Bearer ${token}` },
  });
  
  if (poolResponse.status === 200) {
    try {
      const poolStats = JSON.parse(poolResponse.body);
      console.log(`Database pool: ${poolStats.active_connections}/${poolStats.max_connections} connections`);
      
      if (poolStats.active_connections > poolStats.max_connections * 0.8) {
        console.warn('Warning: Database pool utilization is high before test start');
      }
    } catch (e) {
      console.log('Could not parse pool stats during setup');
    }
  }
  
  console.log('Database bottleneck test setup completed');
  return { setupComplete: true };
}

export function teardown(data) {
  console.log('Database bottleneck tests completed');
  
  // Log performance summary
  console.log('Database Performance Summary:');
  console.log(`- Slow queries detected: ${slowQueryCount.count}`);
  console.log(`- Connection errors: ${connectionErrors.count}`);
  console.log(`- Deadlocks detected: ${deadlockCount.count}`);
  console.log('- Connection pool utilization: measured');
  console.log('- Query execution times: measured');
  console.log('- Partition query performance: measured');
}