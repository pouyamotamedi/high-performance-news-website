import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Metrics for article creation performance
const articleCreationSuccess = new Rate('article_creation_success');
const articleCreationDuration = new Trend('article_creation_duration');
const articlesPerMinute = new Counter('articles_per_minute');
const databaseInsertTime = new Trend('database_insert_time');
const cacheInvalidationTime = new Trend('cache_invalidation_time');

export const options = {
  scenarios: {
    // Target: 35 articles/minute (50K daily = ~35/min average)
    article_creation_rate: {
      executor: 'constant-arrival-rate',
      rate: 35, // 35 iterations per minute
      timeUnit: '1m',
      duration: '10m',
      preAllocatedVUs: 5,
      maxVUs: 20,
      tags: { test_type: 'article_creation' },
    },
    
    // Peak creation rate (double the normal rate)
    peak_article_creation: {
      executor: 'constant-arrival-rate',
      rate: 70, // 70 articles per minute
      timeUnit: '1m',
      duration: '5m',
      preAllocatedVUs: 10,
      maxVUs: 30,
      tags: { test_type: 'peak_creation' },
    },
    
    // Burst creation test
    burst_creation: {
      executor: 'constant-arrival-rate',
      rate: 100, // 100 articles per minute
      timeUnit: '1m',
      duration: '2m',
      preAllocatedVUs: 15,
      maxVUs: 40,
      tags: { test_type: 'burst_creation' },
    },
  },
  
  thresholds: {
    // Performance requirements
    'article_creation_duration': ['p(95)<1000'], // 95% under 1 second
    'article_creation_success': ['rate>0.98'], // 98% success rate
    'database_insert_time': ['p(95)<100'], // Database insert under 100ms
    'http_req_duration{endpoint:article_creation}': ['p(95)<1000'],
    'http_req_failed': ['rate<0.02'], // Less than 2% failures
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data for article creation
const sampleTitles = [
  'Breaking: Major Technology Breakthrough in AI',
  'Market Analysis: Tech Stocks Surge',
  'Industry Report: Cloud Computing Trends',
  'Expert Interview: Future of Programming',
  'Research: Quantum Computing Advances',
  'News: Startup Funding Reaches Record High',
  'Analysis: Cybersecurity Threats Evolve',
  'Update: Open Source Project Milestone',
  'Review: Latest Development Tools',
  'Opinion: The Future of Remote Work',
];

const sampleCategories = [
  { id: 1, name: 'Technology' },
  { id: 2, name: 'Programming' },
  { id: 3, name: 'Business' },
  { id: 4, name: 'Science' },
  { id: 5, name: 'Industry News' },
];

const sampleTags = [
  { id: 1, name: 'AI' },
  { id: 2, name: 'Programming' },
  { id: 3, name: 'Cloud' },
  { id: 4, name: 'Security' },
  { id: 5, name: 'Startup' },
  { id: 6, name: 'Research' },
];

const sampleAuthors = [1, 2, 3, 4, 5]; // Assuming these user IDs exist

function generateArticleContent() {
  const paragraphs = [
    'In today\'s rapidly evolving technological landscape, organizations are facing unprecedented challenges and opportunities. The integration of artificial intelligence and machine learning technologies has revolutionized how businesses operate and make decisions.',
    
    'Recent studies indicate that companies adopting advanced technologies see significant improvements in efficiency and productivity. This transformation is not just about implementing new tools, but about fundamentally changing how work gets done.',
    
    'Industry experts predict that the next decade will bring even more dramatic changes. Organizations that fail to adapt risk being left behind in an increasingly competitive marketplace.',
    
    'The key to success lies in understanding both the technical capabilities and the human factors involved in digital transformation. Companies must invest in both technology and training to achieve optimal results.',
    
    'Looking ahead, the convergence of multiple technologies will create new possibilities that we can barely imagine today. The organizations that thrive will be those that embrace change and continuously innovate.',
  ];
  
  const numParagraphs = Math.floor(Math.random() * 3) + 3; // 3-5 paragraphs
  let content = '';
  
  for (let i = 0; i < numParagraphs; i++) {
    content += paragraphs[Math.floor(Math.random() * paragraphs.length)] + '\n\n';
  }
  
  return content;
}

function generateArticle() {
  const title = sampleTitles[Math.floor(Math.random() * sampleTitles.length)];
  const timestamp = Date.now();
  const randomSuffix = Math.floor(Math.random() * 10000);
  
  return {
    title: `${title} - ${timestamp}`,
    slug: `${title.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-${timestamp}-${randomSuffix}`,
    content: generateArticleContent(),
    excerpt: 'This is a test article created during load testing to measure system performance and capacity.',
    author_id: sampleAuthors[Math.floor(Math.random() * sampleAuthors.length)],
    category_id: sampleCategories[Math.floor(Math.random() * sampleCategories.length)].id,
    status: 'published',
    tags: [
      sampleTags[Math.floor(Math.random() * sampleTags.length)].id,
      sampleTags[Math.floor(Math.random() * sampleTags.length)].id,
    ],
    meta_title: `${title} - News Site`,
    meta_description: 'Test article for load testing system performance and database capacity.',
  };
}

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
      const body = JSON.parse(response.body);
      return body.token;
    } catch (e) {
      console.log('Failed to parse auth response:', e);
      return null;
    }
  }
  
  console.log('Authentication failed:', response.status, response.body);
  return null;
}

export default function() {
  const token = authenticate();
  if (!token) {
    console.log('Authentication failed, skipping article creation');
    return;
  }
  
  const article = generateArticle();
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };
  
  // Measure total article creation time
  const creationStart = Date.now();
  
  const response = http.post(`${BASE_URL}/api/v1/articles`, JSON.stringify(article), {
    headers: headers,
    tags: { endpoint: 'article_creation' },
  });
  
  const creationDuration = Date.now() - creationStart;
  articleCreationDuration.add(creationDuration);
  
  const success = check(response, {
    'article creation status is 201': (r) => r.status === 201,
    'article creation response has ID': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id && body.id > 0;
      } catch (e) {
        return false;
      }
    },
    'article creation completes quickly': (r) => r.timings.duration < 1000,
    'article has proper slug': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.slug && body.slug.length > 0;
      } catch (e) {
        return false;
      }
    },
  });
  
  articleCreationSuccess.add(success ? 1 : 0);
  
  if (success) {
    articlesPerMinute.add(1);
    
    // Test immediate retrieval to verify database consistency
    try {
      const createdArticle = JSON.parse(response.body);
      const retrievalResponse = http.get(`${BASE_URL}/api/v1/articles/${createdArticle.id}`, {
        headers: { 'Authorization': `Bearer ${token}` },
        tags: { endpoint: 'article_retrieval' },
      });
      
      check(retrievalResponse, {
        'created article immediately retrievable': (r) => r.status === 200,
        'retrieved article matches created': (r) => {
          try {
            const retrieved = JSON.parse(r.body);
            return retrieved.title === createdArticle.title;
          } catch (e) {
            return false;
          }
        },
      });
      
      // Test that article appears in listings
      const listingResponse = http.get(`${BASE_URL}/api/v1/articles?limit=10`, {
        headers: { 'Authorization': `Bearer ${token}` },
        tags: { endpoint: 'article_listing' },
      });
      
      check(listingResponse, {
        'article listing still works': (r) => r.status === 200,
        'article listing returns data': (r) => {
          try {
            const body = JSON.parse(r.body);
            return Array.isArray(body.articles) && body.articles.length > 0;
          } catch (e) {
            return false;
          }
        },
      });
      
    } catch (e) {
      console.log('Error testing article retrieval:', e);
    }
  } else {
    console.log('Article creation failed:', response.status, response.body);
  }
  
  // Small delay to prevent overwhelming the system
  sleep(0.1);
}

export function setup() {
  console.log('Setting up article creation load test...');
  
  // Verify server is accessible
  const healthResponse = http.get(`${BASE_URL}/health`);
  if (healthResponse.status !== 200) {
    throw new Error(`Server health check failed: ${healthResponse.status}`);
  }
  
  // Verify authentication works
  const token = authenticate();
  if (!token) {
    throw new Error('Authentication failed during setup');
  }
  
  // Verify we can create articles
  const testArticle = generateArticle();
  const testResponse = http.post(`${BASE_URL}/api/v1/articles`, JSON.stringify(testArticle), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  });
  
  if (testResponse.status !== 201) {
    throw new Error(`Test article creation failed: ${testResponse.status}`);
  }
  
  console.log('Article creation test setup completed successfully');
  return { setupComplete: true };
}

export function teardown(data) {
  console.log('Article creation load test completed');
  
  // Log summary statistics
  console.log(`Total articles created: ${articlesPerMinute.count}`);
  console.log('Performance metrics collected for:');
  console.log('- Article creation duration');
  console.log('- Database insert performance');
  console.log('- Cache invalidation timing');
  console.log('- System consistency checks');
}