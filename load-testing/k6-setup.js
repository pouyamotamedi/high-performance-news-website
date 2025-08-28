import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const articleCreationRate = new Rate('article_creation_success');
const articleCreationDuration = new Trend('article_creation_duration');
const databaseQueryDuration = new Trend('database_query_duration');
const apiResponseTime = new Trend('api_response_time');
const errorRate = new Rate('error_rate');
const articleCreationCounter = new Counter('articles_created');

// Test configuration for different scenarios
export const options = {
  scenarios: {
    // Normal load: 100 concurrent users
    normal_load: {
      executor: 'constant-vus',
      vus: 100,
      duration: '5m',
      tags: { test_type: 'normal_load' },
    },
    
    // Article creation load: 35 articles/minute (50K daily target)
    article_creation: {
      executor: 'constant-arrival-rate',
      rate: 35, // 35 articles per minute
      timeUnit: '1m',
      duration: '10m',
      preAllocatedVUs: 10,
      maxVUs: 50,
      tags: { test_type: 'article_creation' },
    },
    
    // Peak load simulation
    peak_load: {
      executor: 'ramping-vus',
      startVUs: 100,
      stages: [
        { duration: '2m', target: 500 },
        { duration: '5m', target: 1000 },
        { duration: '2m', target: 100 },
      ],
      tags: { test_type: 'peak_load' },
    },
    
    // Database stress test
    database_stress: {
      executor: 'constant-vus',
      vus: 50,
      duration: '3m',
      tags: { test_type: 'database_stress' },
    },
  },
  
  thresholds: {
    // Performance requirements from task 22
    'http_req_duration': ['p(95)<2000'], // 95% of requests under 2s
    'http_req_duration{endpoint:homepage}': ['p(95)<500'], // Homepage under 500ms
    'http_req_duration{endpoint:api}': ['p(95)<100'], // API under 100ms
    'http_req_duration{endpoint:search}': ['p(95)<200'], // Search under 200ms
    'article_creation_success': ['rate>0.95'], // 95% success rate
    'error_rate': ['rate<0.05'], // Less than 5% errors
    'database_query_duration': ['p(95)<10'], // Database queries under 10ms
  },
};

// Base URL - should be configured via environment
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data generators
function generateArticle() {
  const titles = [
    'Breaking: Major Technology Breakthrough',
    'Latest Updates in Software Development',
    'Industry Analysis: Market Trends',
    'Expert Opinion: Future Predictions',
    'Research Findings: Scientific Discovery',
  ];
  
  const categories = [1, 2, 3, 4, 5]; // Assuming categories exist
  const authors = [1, 2, 3]; // Assuming users exist
  
  return {
    title: titles[Math.floor(Math.random() * titles.length)] + ' ' + Date.now(),
    content: generateContent(),
    excerpt: 'This is a test article excerpt for load testing purposes.',
    author_id: authors[Math.floor(Math.random() * authors.length)],
    category_id: categories[Math.floor(Math.random() * categories.length)],
    status: 'published',
    tags: [1, 2, 3], // Assuming tags exist
  };
}

function generateContent() {
  const paragraphs = [
    'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.',
    'Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.',
    'Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.',
    'Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.',
  ];
  
  let content = '';
  const numParagraphs = Math.floor(Math.random() * 5) + 3; // 3-7 paragraphs
  
  for (let i = 0; i < numParagraphs; i++) {
    content += paragraphs[Math.floor(Math.random() * paragraphs.length)] + '\n\n';
  }
  
  return content;
}

// Authentication helper
function authenticate() {
  const loginData = {
    username: 'testuser',
    password: 'testpass123',
  };
  
  const response = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify(loginData), {
    headers: { 'Content-Type': 'application/json' },
    tags: { endpoint: 'auth' },
  });
  
  if (response.status === 200) {
    const token = JSON.parse(response.body).token;
    return token;
  }
  
  return null;
}

// Main test scenarios
export default function() {
  const scenario = __ENV.K6_SCENARIO || 'normal_load';
  
  switch (scenario) {
    case 'article_creation':
      testArticleCreation();
      break;
    case 'database_stress':
      testDatabaseOperations();
      break;
    case 'peak_load':
      testPeakLoad();
      break;
    default:
      testNormalLoad();
  }
}

function testNormalLoad() {
  // Test homepage loading
  const homepageStart = Date.now();
  const homepageResponse = http.get(`${BASE_URL}/`, {
    tags: { endpoint: 'homepage' },
  });
  
  check(homepageResponse, {
    'homepage status is 200': (r) => r.status === 200,
    'homepage loads in <2s': (r) => r.timings.duration < 2000,
    'homepage cached loads in <500ms': (r) => r.headers['X-Cache-Status'] === 'HIT' ? r.timings.duration < 500 : true,
  });
  
  apiResponseTime.add(homepageResponse.timings.duration, { endpoint: 'homepage' });
  
  // Test article listing
  const articlesResponse = http.get(`${BASE_URL}/api/v1/articles?limit=20`, {
    tags: { endpoint: 'api' },
  });
  
  check(articlesResponse, {
    'articles API status is 200': (r) => r.status === 200,
    'articles API responds in <100ms': (r) => r.timings.duration < 100,
  });
  
  apiResponseTime.add(articlesResponse.timings.duration, { endpoint: 'api' });
  
  // Test search functionality
  const searchResponse = http.get(`${BASE_URL}/api/v1/search?q=technology&limit=10`, {
    tags: { endpoint: 'search' },
  });
  
  check(searchResponse, {
    'search API status is 200': (r) => r.status === 200,
    'search responds in <200ms': (r) => r.timings.duration < 200,
  });
  
  apiResponseTime.add(searchResponse.timings.duration, { endpoint: 'search' });
  
  sleep(1);
}

function testArticleCreation() {
  const token = authenticate();
  if (!token) {
    errorRate.add(1);
    return;
  }
  
  const article = generateArticle();
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
  
  const success = check(response, {
    'article creation status is 201': (r) => r.status === 201,
    'article creation completes in <1s': (r) => r.timings.duration < 1000,
  });
  
  articleCreationRate.add(success);
  articleCreationDuration.add(duration);
  apiResponseTime.add(response.timings.duration, { endpoint: 'article_creation' });
  
  if (success) {
    articleCreationCounter.add(1);
  } else {
    errorRate.add(1);
  }
  
  // Test immediate retrieval of created article
  if (response.status === 201) {
    const createdArticle = JSON.parse(response.body);
    const retrieveResponse = http.get(`${BASE_URL}/api/v1/articles/${createdArticle.id}`, {
      headers: { 'Authorization': `Bearer ${token}` },
      tags: { endpoint: 'article_retrieval' },
    });
    
    check(retrieveResponse, {
      'article retrieval status is 200': (r) => r.status === 200,
      'article retrieval is fast': (r) => r.timings.duration < 100,
    });
  }
  
  sleep(0.5);
}

function testDatabaseOperations() {
  const token = authenticate();
  if (!token) {
    errorRate.add(1);
    return;
  }
  
  const headers = { 'Authorization': `Bearer ${token}` };
  
  // Test various database-intensive operations
  const operations = [
    // Category operations
    () => http.get(`${BASE_URL}/api/v1/categories`, { headers, tags: { endpoint: 'categories' } }),
    () => http.get(`${BASE_URL}/api/v1/categories/1/articles`, { headers, tags: { endpoint: 'category_articles' } }),
    
    // Tag operations
    () => http.get(`${BASE_URL}/api/v1/tags`, { headers, tags: { endpoint: 'tags' } }),
    () => http.get(`${BASE_URL}/api/v1/tags/popular`, { headers, tags: { endpoint: 'popular_tags' } }),
    
    // Article operations
    () => http.get(`${BASE_URL}/api/v1/articles?category=1&limit=50`, { headers, tags: { endpoint: 'filtered_articles' } }),
    () => http.get(`${BASE_URL}/api/v1/articles/trending`, { headers, tags: { endpoint: 'trending_articles' } }),
    
    // Search operations
    () => http.get(`${BASE_URL}/api/v1/search?q=programming&category=1`, { headers, tags: { endpoint: 'filtered_search' } }),
  ];
  
  // Execute random operations
  const operation = operations[Math.floor(Math.random() * operations.length)];
  const start = Date.now();
  const response = operation();
  const dbDuration = Date.now() - start;
  
  check(response, {
    'database operation successful': (r) => r.status === 200,
    'database query fast': (r) => r.timings.duration < 50,
  });
  
  databaseQueryDuration.add(dbDuration);
  
  if (response.status !== 200) {
    errorRate.add(1);
  }
  
  sleep(0.1);
}

function testPeakLoad() {
  // Simulate mixed load during peak traffic
  const operations = [
    () => testNormalLoad(),
    () => testArticleCreation(),
    () => testDatabaseOperations(),
  ];
  
  // Weighted selection (more reads than writes)
  const weights = [0.7, 0.1, 0.2]; // 70% normal load, 10% article creation, 20% database stress
  const random = Math.random();
  
  if (random < weights[0]) {
    testNormalLoad();
  } else if (random < weights[0] + weights[1]) {
    testArticleCreation();
  } else {
    testDatabaseOperations();
  }
}

// Setup function - runs once per VU
export function setup() {
  console.log('Starting load test setup...');
  
  // Verify server is accessible
  const healthCheck = http.get(`${BASE_URL}/health`);
  if (healthCheck.status !== 200) {
    throw new Error(`Server not accessible: ${healthCheck.status}`);
  }
  
  console.log('Server health check passed');
  return { serverReady: true };
}

// Teardown function - runs once after all VUs finish
export function teardown(data) {
  console.log('Load test completed');
  
  // Could send results to monitoring system here
  console.log(`Articles created during test: ${articleCreationCounter.count}`);
}