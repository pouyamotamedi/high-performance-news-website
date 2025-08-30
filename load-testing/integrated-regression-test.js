import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';
import PerformanceRegressionDetector from './performance-regression-detector.js';
import PerformanceAlerting from './performance-alerting.js';
import PerformanceTrendAnalyzer from './performance-trend-analyzer.js';

// Initialize components
const regressionDetector = new PerformanceRegressionDetector();
const alerting = new PerformanceAlerting();
const trendAnalyzer = new PerformanceTrendAnalyzer();

// Custom metrics
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
    // Comprehensive regression test
    regression_test: {
      executor: 'constant-vus',
      vus: parseInt(__ENV.TEST_VUS) || 25,
      duration: __ENV.TEST_DURATION || '5m',
      tags: { test_type: 'integrated_regression' },
    },
  },
  
  // Dynamic thresholds based on baseline
  thresholds: {
    'homepage_response_time': [`p(95)<${__ENV.BASELINE_HOMEPAGE_P95 || 500}`],
    'api_response_time': [`p(95)<${__ENV.BASELINE_API_P95 || 100}`],
    'article_creation_time': [`p(95)<${__ENV.BASELINE_ARTICLE_CREATION_P95 || 1000}`],
    'db_query_time': [`p(95)<${__ENV.BASELINE_DB_QUERY_P95 || 10}`],
    'cache_hit': [`rate>${__ENV.BASELINE_CACHE_HIT_RATE || 0.8}`],
    'error_occurred': [`rate<${__ENV.BASELINE_ERROR_RATE || 0.02}`],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test scenarios with weighted distribution
const testScenarios = [
  { name: 'homepage', weight: 0.3, func: testHomepage },
  { name: 'api_calls', weight: 0.4, func: testAPIEndpoints },
  { name: 'article_creation', weight: 0.1, func: testArticleCreation },
  { name: 'database_queries', weight: 0.15, func: testDatabaseQueries },
  { name: 'cache_operations', weight: 0.05, func: testCacheOperations },
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

export default function() {
  // Select test scenario based on weights
  const scenario = selectWeightedScenario();
  scenario.func();
  
  // Collect system metrics periodically
  if (Math.random() < 0.1) { // 10% of iterations
    collectSystemMetrics();
  }
  
  sleep(0.5);
}

function selectWeightedScenario() {
  const random = Math.random();
  let cumulativeWeight = 0;
  
  for (const scenario of testScenarios) {
    cumulativeWeight += scenario.weight;
    if (random <= cumulativeWeight) {
      return scenario;
    }
  }
  
  return testScenarios[0]; // Fallback
}

function testHomepage() {
  const start = Date.now();
  const response = http.get(`${BASE_URL}/`, {
    tags: { endpoint: 'homepage', scenario: 'homepage' },
  });
  const duration = Date.now() - start;
  
  homepageResponseTime.add(duration);
  regressionDetector.recordMetric('homepage_response_time', duration, { endpoint: 'homepage' });
  
  const success = check(response, {
    'homepage status is 200': (r) => r.status === 200,
    'homepage loads quickly': (r) => r.timings.duration < 2000,
    'homepage has content': (r) => r.body && r.body.length > 1000,
  });
  
  errorRate.add(success ? 0 : 1);
  regressionDetector.recordMetric('error_occurred', success ? 0 : 1);
}

function testAPIEndpoints() {
  const endpoints = [
    '/api/v1/articles?limit=20',
    '/api/v1/categories',
    '/api/v1/tags?limit=50',
    '/api/v1/articles/popular?limit=10',
    '/api/v1/search?q=technology&limit=10',
  ];
  
  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  
  const start = Date.now();
  const response = http.get(`${BASE_URL}${endpoint}`, {
    tags: { endpoint: 'api', scenario: 'api_calls', api_endpoint: endpoint },
  });
  const duration = Date.now() - start;
  
  apiResponseTime.add(duration);
  regressionDetector.recordMetric('api_response_time', duration, { endpoint: endpoint });
  
  const success = check(response, {
    'API status is 200': (r) => r.status === 200,
    'API responds quickly': (r) => r.timings.duration < 500,
    'API returns valid JSON': (r) => {
      try {
        JSON.parse(r.body);
        return true;
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(success ? 0 : 1);
  regressionDetector.recordMetric('error_occurred', success ? 0 : 1);
}

function testArticleCreation() {
  const token = authenticate();
  if (!token) {
    errorRate.add(1);
    regressionDetector.recordMetric('error_occurred', 1);
    return;
  }
  
  const article = {
    title: `Regression Test Article ${Date.now()}`,
    content: 'This is a comprehensive test article for performance regression detection. '.repeat(100),
    excerpt: 'Test article for integrated regression testing',
    author_id: 1,
    category_id: 1,
    status: 'published',
    tags: [1, 2],
    meta_title: 'Test Article - Performance Testing',
    meta_description: 'Test article for performance regression detection',
  };
  
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };
  
  const start = Date.now();
  const response = http.post(`${BASE_URL}/api/v1/articles`, JSON.stringify(article), {
    headers: headers,
    tags: { endpoint: 'article_creation', scenario: 'article_creation' },
  });
  const duration = Date.now() - start;
  
  articleCreationTime.add(duration);
  regressionDetector.recordMetric('article_creation_time', duration);
  
  const success = check(response, {
    'article creation status is 201': (r) => r.status === 201,
    'article creation completes quickly': (r) => r.timings.duration < 2000,
    'article has valid response': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id && body.slug;
      } catch (e) {
        return false;
      }
    },
  });
  
  if (success) {
    articlesCreated.add(1);
    regressionDetector.recordMetric('articles_created', 1);
    
    // Test immediate retrieval for consistency
    try {
      const created = JSON.parse(response.body);
      const retrievalResponse = http.get(`${BASE_URL}/api/v1/articles/${created.id}`, {
        headers: { 'Authorization': `Bearer ${token}` },
        tags: { endpoint: 'article_retrieval', scenario: 'article_creation' },
      });
      
      check(retrievalResponse, {
        'created article retrievable': (r) => r.status === 200,
      });
    } catch (e) {
      // Ignore retrieval test errors
    }
  }
  
  errorRate.add(success ? 0 : 1);
  regressionDetector.recordMetric('error_occurred', success ? 0 : 1);
}

function testDatabaseQueries() {
  const token = authenticate();
  if (!token) return;
  
  const headers = { 'Authorization': `Bearer ${token}` };
  
  const dbQueries = [
    '/api/v1/articles?category=1&limit=50',
    '/api/v1/articles/trending?limit=20',
    '/api/v1/search?q=programming&category=1&limit=10',
    '/api/v1/tags/popular?limit=30',
    '/api/v1/articles?author=1&limit=25',
  ];
  
  const query = dbQueries[Math.floor(Math.random() * dbQueries.length)];
  
  const start = Date.now();
  const response = http.get(`${BASE_URL}${query}`, {
    headers: headers,
    tags: { endpoint: 'database_query', scenario: 'database_queries', query_type: query },
  });
  const duration = Date.now() - start;
  
  dbQueryTime.add(duration);
  regressionDetector.recordMetric('db_query_time', duration, { query_type: query });
  
  const success = check(response, {
    'database query successful': (r) => r.status === 200,
    'database query fast': (r) => r.timings.duration < 200,
    'database query returns data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body.articles) || Array.isArray(body.tags) || body.articles;
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(success ? 0 : 1);
  regressionDetector.recordMetric('error_occurred', success ? 0 : 1);
}

function testCacheOperations() {
  const cacheableEndpoints = [
    '/',
    '/api/v1/categories',
    '/api/v1/tags?limit=20',
    '/api/v1/articles?limit=10',
  ];
  
  const endpoint = cacheableEndpoints[Math.floor(Math.random() * cacheableEndpoints.length)];
  
  // First request (potential cache miss)
  const firstResponse = http.get(`${BASE_URL}${endpoint}`, {
    tags: { cache_test: 'first_request', scenario: 'cache_operations' },
  });
  
  // Second request (should be cache hit)
  const secondResponse = http.get(`${BASE_URL}${endpoint}`, {
    tags: { cache_test: 'second_request', scenario: 'cache_operations' },
  });
  
  // Determine cache effectiveness
  const cacheHit = secondResponse.headers['X-Cache-Status'] === 'HIT' ||
                   secondResponse.timings.duration < firstResponse.timings.duration * 0.7;
  
  cacheHitRate.add(cacheHit ? 1 : 0);
  regressionDetector.recordMetric('cache_hit', cacheHit ? 1 : 0, { endpoint: endpoint });
  
  check(secondResponse, {
    'cache operation successful': (r) => r.status === 200,
    'cache hit detected': () => cacheHit,
    'cached response fast': (r) => cacheHit ? r.timings.duration < 200 : true,
  });
}

function collectSystemMetrics() {
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
        'memory usage reasonable': (m) => m.memory_usage_mb < 1500,
        'CPU usage reasonable': (m) => m.cpu_usage_percent < 90,
        'database connections healthy': (m) => m.db_connections_active < m.db_connections_max * 0.8,
      });
    } catch (e) {
      console.log('Could not parse system metrics:', e);
    }
  }
}

export function setup() {
  console.log('=== INTEGRATED PERFORMANCE REGRESSION TEST ===');
  console.log(`Target: ${BASE_URL}`);
  console.log(`VUs: ${__ENV.TEST_VUS || 25}`);
  console.log(`Duration: ${__ENV.TEST_DURATION || '5m'}`);
  console.log(`Environment: ${__ENV.ENVIRONMENT || 'development'}`);
  
  // Load baseline for comparison
  const baseline = regressionDetector.loadBaseline();
  console.log('Loaded baseline metrics for comparison');
  
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
  
  console.log('✅ Setup completed successfully');
  
  return { 
    setupComplete: true,
    baseline: baseline,
    startTime: Date.now(),
    testMetadata: {
      environment: __ENV.ENVIRONMENT || 'development',
      buildId: __ENV.BUILD_ID || 'unknown',
      commitHash: __ENV.COMMIT_HASH || 'unknown',
      branch: __ENV.BRANCH || 'unknown',
      testDuration: __ENV.TEST_DURATION || '5m',
    },
  };
}

export function teardown(data) {
  console.log('\n=== INTEGRATED REGRESSION TEST RESULTS ===');
  
  const testDuration = data.startTime ? (Date.now() - data.startTime) / 1000 : 0;
  console.log(`Test completed in ${testDuration}s`);
  
  // Generate comprehensive regression report
  const regressionReport = regressionDetector.generateReport();
  
  // Print summary
  console.log(`\nREGRESSION SUMMARY:`);
  console.log(`- Total regressions: ${regressionReport.regressions.count}`);
  console.log(`- Critical: ${regressionReport.alerts.critical}`);
  console.log(`- High: ${regressionReport.alerts.high}`);
  console.log(`- Medium: ${regressionReport.alerts.medium}`);
  console.log(`- Low: ${regressionReport.alerts.low}`);
  
  // Print trend analysis if historical data available
  if (__ENV.ENABLE_TREND_ANALYSIS === 'true') {
    console.log('\n=== TREND ANALYSIS ===');
    // In a real implementation, this would load historical data
    const mockHistoricalData = {
      homepage_p95: [
        { timestamp: new Date(Date.now() - 7*24*60*60*1000).toISOString(), value: 450 },
        { timestamp: new Date(Date.now() - 6*24*60*60*1000).toISOString(), value: 460 },
        { timestamp: new Date(Date.now() - 5*24*60*60*1000).toISOString(), value: 470 },
        { timestamp: new Date().toISOString(), value: regressionReport.current.homepage_p95 || 500 },
      ],
    };
    
    const trendAnalysis = trendAnalyzer.analyzeTrends(mockHistoricalData);
    console.log(`Trend analysis: ${trendAnalysis.summary.improving.length} improving, ${trendAnalysis.summary.degrading.length} degrading`);
  }
  
  // Send alerts if regressions detected
  if (regressionReport.regressions.count > 0) {
    console.log('\n📧 Sending performance regression alerts...');
    alerting.sendRegressionAlerts(regressionReport.regressions.details, data.testMetadata);
  } else {
    console.log('\n✅ No regressions detected');
    if (__ENV.SEND_SUCCESS_NOTIFICATIONS === 'true') {
      alerting.sendSuccessNotification(data.testMetadata);
    }
  }
  
  // Output structured data for CI/CD integration
  console.log('\n=== CI/CD INTEGRATION DATA ===');
  const cicdData = {
    test_result: regressionReport.regressions.count === 0 ? 'PASS' : 'FAIL',
    regression_count: regressionReport.regressions.count,
    critical_regressions: regressionReport.alerts.critical,
    high_regressions: regressionReport.alerts.high,
    test_duration_seconds: testDuration,
    timestamp: new Date().toISOString(),
    environment: data.testMetadata.environment,
    build_id: data.testMetadata.buildId,
  };
  
  console.log(JSON.stringify(cicdData, null, 2));
  
  // Determine exit status for CI/CD
  const criticalRegressions = regressionReport.alerts.critical;
  const highRegressions = regressionReport.alerts.high;
  
  if (criticalRegressions > 0) {
    console.log('\n❌ TEST FAILED: Critical performance regressions detected');
    console.log('Recommendation: Block deployment and investigate immediately');
  } else if (highRegressions > 2) {
    console.log('\n⚠️  TEST WARNING: Multiple high-severity regressions detected');
    console.log('Recommendation: Review before deployment');
  } else if (regressionReport.regressions.count === 0) {
    console.log('\n✅ TEST PASSED: No performance regressions detected');
  } else {
    console.log('\n✅ TEST PASSED: Minor regressions within acceptable limits');
  }
  
  console.log('\n=== END INTEGRATED REGRESSION TEST ===');
}