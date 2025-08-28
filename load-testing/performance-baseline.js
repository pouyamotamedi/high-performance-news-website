import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

// Performance baseline metrics
const dbConnectionTime = new Trend('db_connection_time');
const cacheHitRate = new Rate('cache_hit_rate');
const queryExecutionTime = new Trend('query_execution_time');
const memoryUsage = new Trend('memory_usage');
const cpuUsage = new Trend('cpu_usage');

export const options = {
  scenarios: {
    baseline_measurement: {
      executor: 'constant-vus',
      vus: 1,
      duration: '2m',
      tags: { test_type: 'baseline' },
    },
  },
  
  thresholds: {
    // Baseline performance requirements
    'http_req_duration{endpoint:homepage}': ['p(95)<500'],
    'http_req_duration{endpoint:article}': ['p(95)<200'],
    'http_req_duration{endpoint:api}': ['p(95)<100'],
    'db_connection_time': ['p(95)<50'],
    'query_execution_time': ['p(95)<10'],
    'cache_hit_rate': ['rate>0.8'], // 80% cache hit rate
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function() {
  measureDatabasePerformance();
  measureCachePerformance();
  measureAPIPerformance();
  measureSystemResources();
  
  sleep(1);
}

function measureDatabasePerformance() {
  // Test database connection time
  const dbStart = Date.now();
  const dbHealthResponse = http.get(`${BASE_URL}/api/v1/system/db-health`);
  const dbConnTime = Date.now() - dbStart;
  
  dbConnectionTime.add(dbConnTime);
  
  check(dbHealthResponse, {
    'database connection healthy': (r) => r.status === 200,
    'database connection fast': (r) => r.timings.duration < 50,
  });
  
  // Test query performance with different operations
  const queries = [
    // Simple article lookup
    `${BASE_URL}/api/v1/articles/1`,
    // Category with articles
    `${BASE_URL}/api/v1/categories/1/articles?limit=10`,
    // Tag operations
    `${BASE_URL}/api/v1/tags/1/articles?limit=10`,
    // Search query
    `${BASE_URL}/api/v1/search?q=test&limit=10`,
    // Popular articles
    `${BASE_URL}/api/v1/articles/popular?limit=10`,
  ];
  
  queries.forEach((url, index) => {
    const queryStart = Date.now();
    const response = http.get(url, {
      tags: { query_type: `query_${index}` },
    });
    const queryTime = Date.now() - queryStart;
    
    queryExecutionTime.add(queryTime, { query_type: `query_${index}` });
    
    check(response, {
      [`query ${index} successful`]: (r) => r.status === 200,
      [`query ${index} fast`]: (r) => r.timings.duration < 10,
    });
  });
}

function measureCachePerformance() {
  // Test cache hit rates for different content types
  const cacheableEndpoints = [
    { url: `${BASE_URL}/`, type: 'homepage' },
    { url: `${BASE_URL}/api/v1/articles?limit=20`, type: 'articles_list' },
    { url: `${BASE_URL}/api/v1/categories`, type: 'categories' },
    { url: `${BASE_URL}/api/v1/tags/popular`, type: 'popular_tags' },
  ];
  
  cacheableEndpoints.forEach(endpoint => {
    // First request (should miss cache)
    const firstResponse = http.get(endpoint.url, {
      tags: { cache_test: 'first_request', endpoint_type: endpoint.type },
    });
    
    // Second request (should hit cache)
    const secondResponse = http.get(endpoint.url, {
      tags: { cache_test: 'second_request', endpoint_type: endpoint.type },
    });
    
    const cacheHit = secondResponse.headers['X-Cache-Status'] === 'HIT' ||
                     secondResponse.timings.duration < firstResponse.timings.duration * 0.5;
    
    cacheHitRate.add(cacheHit ? 1 : 0, { endpoint_type: endpoint.type });
    
    check(secondResponse, {
      [`${endpoint.type} cache working`]: () => cacheHit,
      [`${endpoint.type} cached response fast`]: (r) => cacheHit ? r.timings.duration < 100 : true,
    });
  });
}

function measureAPIPerformance() {
  // Test different API endpoints for baseline performance
  const apiEndpoints = [
    { url: `${BASE_URL}/api/v1/articles`, method: 'GET', expected: 200, maxTime: 100 },
    { url: `${BASE_URL}/api/v1/categories`, method: 'GET', expected: 200, maxTime: 50 },
    { url: `${BASE_URL}/api/v1/tags`, method: 'GET', expected: 200, maxTime: 50 },
    { url: `${BASE_URL}/api/v1/search?q=test`, method: 'GET', expected: 200, maxTime: 200 },
    { url: `${BASE_URL}/api/v1/system/health`, method: 'GET', expected: 200, maxTime: 50 },
  ];
  
  apiEndpoints.forEach((endpoint, index) => {
    const response = http.get(endpoint.url, {
      tags: { endpoint: 'api', api_type: `api_${index}` },
    });
    
    check(response, {
      [`API ${index} status correct`]: (r) => r.status === endpoint.expected,
      [`API ${index} response time acceptable`]: (r) => r.timings.duration < endpoint.maxTime,
    });
  });
}

function measureSystemResources() {
  // Get system metrics if available
  const metricsResponse = http.get(`${BASE_URL}/api/v1/system/metrics`);
  
  if (metricsResponse.status === 200) {
    try {
      const metrics = JSON.parse(metricsResponse.body);
      
      if (metrics.memory_usage_mb) {
        memoryUsage.add(metrics.memory_usage_mb);
      }
      
      if (metrics.cpu_usage_percent) {
        cpuUsage.add(metrics.cpu_usage_percent);
      }
      
      check(metrics, {
        'memory usage reasonable': (m) => m.memory_usage_mb < 1000, // Less than 1GB
        'CPU usage reasonable': (m) => m.cpu_usage_percent < 80, // Less than 80%
        'database connections healthy': (m) => m.db_connections_active < 100,
        'cache memory usage reasonable': (m) => m.cache_memory_mb < 500,
      });
    } catch (e) {
      console.log('Could not parse system metrics:', e);
    }
  }
}

export function setup() {
  console.log('Starting performance baseline measurement...');
  
  // Verify all required endpoints are available
  const requiredEndpoints = [
    `${BASE_URL}/health`,
    `${BASE_URL}/api/v1/articles`,
    `${BASE_URL}/api/v1/categories`,
    `${BASE_URL}/api/v1/tags`,
  ];
  
  for (const endpoint of requiredEndpoints) {
    const response = http.get(endpoint);
    if (response.status >= 400) {
      throw new Error(`Required endpoint not available: ${endpoint} (status: ${response.status})`);
    }
  }
  
  console.log('All required endpoints are available');
  return { baselineReady: true };
}

export function teardown(data) {
  console.log('Performance baseline measurement completed');
  
  // Log baseline results
  console.log('Baseline Performance Summary:');
  console.log('- Database connection time: measured');
  console.log('- Cache hit rate: measured');
  console.log('- Query execution time: measured');
  console.log('- API response times: measured');
  console.log('- System resource usage: measured');
}