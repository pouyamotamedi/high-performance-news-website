import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';
import PerformanceRegressionDetector from './performance-regression-detector.js';

// Initialize regression detector
const regressionDetector = new PerformanceRegressionDetector();

// Custom metrics for regression detection
const homepageResponseTime = new Trend('homepage_response_time');
const apiResponseTime = new Trend('api_response_time');
const articleCreationTime = new Trend('article_creation_time');
const dbQueryTime = new Trend('db_query_time');
const cacheHitRate = new Rate('cache_hit');
const errorRate = new Rate('error_occurred');
const articlesCreated = new Counter('articles_created');
const memoryUsage = new Trend('memory_usage');
const cpuUsage = new Trend('cpu_usage');

export const options = {
  scenarios: {
    // Regression detection test - comprehensive but focused
    regression_detection: {
      executor: 'constant-vus',
      vus: 50,
      duration: '5m',
      tags: { test_type: 'regression_detection' },
    },
  },
  
  thresholds: {
    // Dynamic thresholds based on baseline + tolerance
    'homepage_response_time': [`p(95)<${__ENV.BASELINE_HOMEPAGE_P95 || 500}`],
    'api_response_time': [`p(95)<${__ENV.BASELINE_API_P95 || 100}`],
    'article_creation_time': [`p(95)<${__ENV.BASELINE_ARTICLE_CREATION_P95 || 1000}`],
    'db_query_time': [`p(95)<${__ENV.BASELINE_DB_QUERY_P95 || 10}`],
    'cache_hit': [`rate>${__ENV.BASELINE_CACHE_HIT_RATE || 0.8}`],
    'error_occurred': [`rate<${__ENV.BASELINE_ERROR_RATE || 0.02}`],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data for consistent testing
const testQueries = [
  '/api/v1/articles?limit=20',
  '/api/v1/categories',
  '/api/v1/tags?limit=50',
  '/api/v1/articles/popular?limit=10',
  '/api/v1/search?q=technology&limit=10',
];

function authenticate() {
  const loginData = {
    username: __ENV.TEST_USERNAME || 'testuser',
    password: __ENV.TEST_PASSWORD || 'testpass123',
  };
  
  const response = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify(loginData), {
    headers: { 'Content-Type': 'application/json' },
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

function generateTestArticle() {
  return {
    title: `Regression Test Article ${Date.now()}`,
    content: 'This is a test article for performance regression detection. '.repeat(50),
    excerpt: 'Test article for regression detection',
    author_id: 1,
    category_id: 1,
    status: 'published',
    tags: [1, 2],
  };
}

export default function() {
  // Test homepage performance
  testHomepagePerformance();
  
  // Test API performance
  testAPIPerformance();
  
  // Test article creation performance
  testArticleCreationPerformance();
  
  // Test database query performance
  testDatabaseQueryPerformance();
  
  // Test cache performance
  testCachePerformance();
  
  // Collect system metrics
  collectSystemMetrics();
  
  sleep(1);
}

function testHomepagePerformance() {
  const start = Date.now();
  const response = http.get(`${BASE_URL}/`, {
    tags: { endpoint: 'homepage' },
  });
  const duration = Date.now() - start;
  
  homepageResponseTime.add(duration);
  regressionDetector.recordMetric('homepage_response_time', duration);
  
  const success = check(response, {
    'homepage status is 200': (r) => r.status === 200,
    'homepage loads quickly': (r) => r.timings.duration < 2000,
  });
  
  if (!success) {
    errorRate.add(1);
    regressionDetector.recordMetric('error_occurred', 1);
  } else {
    errorRate.add(0);
    regressionDetector.recordMetric('error_occurred', 0);
  }
}

function testAPIPerformance() {
  const query = testQueries[Math.floor(Math.random() * testQueries.length)];
  
  const start = Date.now();
  const response = http.get(`${BASE_URL}${query}`, {
    tags: { endpoint: 'api' },
  });
  const duration = Date.now() - start;
  
  apiResponseTime.add(duration);
  regressionDetector.recordMetric('api_response_time', duration);
  
  const success = check(response, {
    'API status is 200': (r) => r.status === 200,
    'API responds quickly': (r) => r.timings.duration < 500,
  });
  
  if (!success) {
    errorRate.add(1);
    regressionDetector.recordMetric('error_occurred', 1);
  } else {
    errorRate.add(0);
    regressionDetector.recordMetric('error_occurred', 0);
  }
}

function testArticleCreationPerformance() {
  const token = authenticate();
  if (!token) {
    errorRate.add(1);
    regressionDetector.recordMetric('error_occurred', 1);
    return;
  }
  
  const article = generateTestArticle();
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };
  
  const start = Date.now();
  const response = http.post(`${BASE_URL}/api/v1/articles`, JSON.stringify(article), {
    headers: headers,
    tags: { endpoint: 'article_creation' },
  });
  const duration = Date.now() - start;
  
  articleCreationTime.add(duration);
  regressionDetector.recordMetric('article_creation_time', duration);
  
  const success = check(response, {
    'article creation status is 201': (r) => r.status === 201,
    'article creation completes quickly': (r) => r.timings.duration < 2000,
  });
  
  if (success) {
    articlesCreated.add(1);
    regressionDetector.recordMetric('articles_created', 1);
    errorRate.add(0);
    regressionDetector.recordMetric('error_occurred', 0);
  } else {
    errorRate.add(1);
    regressionDetector.recordMetric('error_occurred', 1);
  }
}

function testDatabaseQueryPerformance() {
  const token = authenticate();
  if (!token) return;
  
  const headers = { 'Authorization': `Bearer ${token}` };
  
  // Test specific database-intensive queries
  const dbQueries = [
    '/api/v1/articles?category=1&limit=50',
    '/api/v1/articles/trending?limit=20',
    '/api/v1/search?q=programming&category=1&limit=10',
    '/api/v1/tags/popular?limit=30',
  ];
  
  const query = dbQueries[Math.floor(Math.random() * dbQueries.length)];
  
  const start = Date.now();
  const response = http.get(`${BASE_URL}${query}`, {
    headers: headers,
    tags: { endpoint: 'database_query' },
  });
  const duration = Date.now() - start;
  
  dbQueryTime.add(duration);
  regressionDetector.recordMetric('db_query_time', duration);
  
  check(response, {
    'database query successful': (r) => r.status === 200,
    'database query fast': (r) => r.timings.duration < 100,
  });
}

function testCachePerformance() {
  // Test cache hit rates by making repeated requests
  const cacheableEndpoints = [
    '/',
    '/api/v1/categories',
    '/api/v1/tags?limit=20',
    '/api/v1/articles?limit=10',
  ];
  
  const endpoint = cacheableEndpoints[Math.floor(Math.random() * cacheableEndpoints.length)];
  
  // First request (likely cache miss)
  const firstResponse = http.get(`${BASE_URL}${endpoint}`, {
    tags: { cache_test: 'first_request' },
  });
  
  // Second request (should be cache hit)
  const secondResponse = http.get(`${BASE_URL}${endpoint}`, {
    tags: { cache_test: 'second_request' },
  });
  
  // Determine if cache hit occurred
  const cacheHit = secondResponse.headers['X-Cache-Status'] === 'HIT' ||
                   secondResponse.timings.duration < firstResponse.timings.duration * 0.7;
  
  cacheHitRate.add(cacheHit ? 1 : 0);
  regressionDetector.recordMetric('cache_hit', cacheHit ? 1 : 0);
  
  check(secondResponse, {
    'cache working': () => cacheHit,
    'cached response fast': (r) => cacheHit ? r.timings.duration < 200 : true,
  });
}

function collectSystemMetrics() {
  // Collect system metrics if available
  const metricsResponse = http.get(`${BASE_URL}/api/v1/system/metrics`);
  
  if (metricsResponse.status === 200) {
    try {
      const metrics = JSON.parse(metricsResponse.body);
      
      if (metrics.memory_usage_mb) {
        memoryUsage.add(metrics.memory_usage_mb);
        regressionDetector.recordMetric('memory_usage', metrics.memory_usage_mb);
      }
      
      if (metrics.cpu_usage_percent) {
        cpuUsage.add(metrics.cpu_usage_percent);
        regressionDetector.recordMetric('cpu_usage', metrics.cpu_usage_percent);
      }
      
      check(metrics, {
        'memory usage reasonable': (m) => m.memory_usage_mb < 1200,
        'CPU usage reasonable': (m) => m.cpu_usage_percent < 85,
      });
    } catch (e) {
      console.log('Could not parse system metrics:', e);
    }
  }
}

export function setup() {
  console.log('Starting performance regression detection test...');
  
  // Load baseline metrics
  const baseline = regressionDetector.loadBaseline();
  console.log('Loaded baseline metrics:', JSON.stringify(baseline, null, 2));
  
  // Verify server accessibility
  const healthResponse = http.get(`${BASE_URL}/health`);
  if (healthResponse.status !== 200) {
    throw new Error(`Server health check failed: ${healthResponse.status}`);
  }
  
  // Verify authentication
  const token = authenticate();
  if (!token) {
    throw new Error('Authentication failed during setup');
  }
  
  console.log('Performance regression test setup completed');
  return { 
    setupComplete: true,
    baseline: baseline,
    startTime: Date.now(),
  };
}

export function teardown(data) {
  console.log('Performance regression detection test completed');
  
  // Generate comprehensive performance report
  const report = regressionDetector.generateReport();
  
  console.log('\n=== PERFORMANCE REGRESSION REPORT ===');
  console.log(`Test Duration: ${data.startTime ? (Date.now() - data.startTime) / 1000 : 'unknown'}s`);
  console.log(`Timestamp: ${report.timestamp}`);
  
  // Print regression summary
  console.log(`\nREGRESSIONS DETECTED: ${report.regressions.count}`);
  if (report.regressions.count > 0) {
    report.regressions.details.forEach(regression => {
      console.log(`  - ${regression.metric}: ${regression.changePercent.toFixed(1)}% change (${regression.severity})`);
      console.log(`    Baseline: ${regression.baseline}, Current: ${regression.current}`);
    });
  }
  
  // Print alerts summary
  console.log(`\nALERTS GENERATED:`);
  console.log(`  Critical: ${report.alerts.critical}`);
  console.log(`  High: ${report.alerts.high}`);
  console.log(`  Medium: ${report.alerts.medium}`);
  console.log(`  Low: ${report.alerts.low}`);
  
  if (report.alerts.details.length > 0) {
    console.log('\nALERT DETAILS:');
    report.alerts.details.forEach(alert => {
      console.log(`  [${alert.severity.toUpperCase()}] ${alert.message}`);
      console.log(`    Recommendation: ${alert.recommendation}`);
    });
  }
  
  // Print trend analysis
  console.log(`\nTREND ANALYSIS:`);
  console.log(`  Improving: ${report.trends.improving.length} metrics`);
  console.log(`  Stable: ${report.trends.stable.length} metrics`);
  console.log(`  Degrading: ${report.trends.degrading.length} metrics`);
  
  // Print recommendations
  if (report.recommendations.length > 0) {
    console.log('\nRECOMMENDATIONS:');
    report.recommendations.forEach(rec => {
      console.log(`  [${rec.priority.toUpperCase()}] ${rec.action}`);
      console.log(`    ${rec.details}`);
      console.log(`    Affected metrics: ${rec.metrics.join(', ')}`);
    });
  }
  
  // Output structured data for CI/CD integration
  console.log('\n=== STRUCTURED REPORT (JSON) ===');
  console.log(JSON.stringify(report, null, 2));
  
  // Determine if test should fail based on regressions
  const criticalRegressions = report.regressions.details.filter(r => r.severity === 'critical');
  const highRegressions = report.regressions.details.filter(r => r.severity === 'high');
  
  if (criticalRegressions.length > 0) {
    console.log('\n❌ TEST FAILED: Critical performance regressions detected');
    // In a real CI/CD system, this would set exit code to fail the build
  } else if (highRegressions.length > 2) {
    console.log('\n⚠️  TEST WARNING: Multiple high-severity regressions detected');
  } else if (report.regressions.count === 0) {
    console.log('\n✅ TEST PASSED: No performance regressions detected');
  } else {
    console.log('\n✅ TEST PASSED: Minor regressions within acceptable limits');
  }
  
  console.log('\n=== END PERFORMANCE REGRESSION REPORT ===');
}