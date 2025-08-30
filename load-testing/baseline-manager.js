import http from 'k6/http';
import { check } from 'k6';

// Baseline Performance Metrics Manager
export class BaselineManager {
  constructor() {
    this.baselineFile = 'performance-baseline.json';
    this.metricsHistory = [];
  }

  // Establish new baseline from current test run
  establishBaseline() {
    console.log('Establishing new performance baseline...');
    
    const baseline = this.measureBaselineMetrics();
    
    // Store baseline (in real implementation, this would save to file/database)
    this.storeBaseline(baseline);
    
    return baseline;
  }

  // Measure comprehensive baseline metrics
  measureBaselineMetrics() {
    const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
    const measurements = {
      timestamp: new Date().toISOString(),
      environment: {
        base_url: BASE_URL,
        test_user: __ENV.TEST_USERNAME || 'testuser',
        k6_version: __ENV.K6_VERSION || 'unknown',
      },
      metrics: {},
    };

    // Authenticate for protected endpoints
    const token = this.authenticate();
    const headers = token ? { 'Authorization': `Bearer ${token}` } : {};

    console.log('Measuring homepage performance...');
    measurements.metrics.homepage = this.measureEndpointPerformance(`${BASE_URL}/`, {}, 10);

    console.log('Measuring API performance...');
    const apiEndpoints = [
      '/api/v1/articles?limit=20',
      '/api/v1/categories',
      '/api/v1/tags?limit=50',
      '/api/v1/articles/popular?limit=10',
    ];
    
    measurements.metrics.api = {};
    apiEndpoints.forEach((endpoint, index) => {
      measurements.metrics.api[`endpoint_${index}`] = this.measureEndpointPerformance(
        `${BASE_URL}${endpoint}`, 
        { headers }, 
        5
      );
    });

    console.log('Measuring search performance...');
    measurements.metrics.search = this.measureEndpointPerformance(
      `${BASE_URL}/api/v1/search?q=technology&limit=10`,
      { headers },
      5
    );

    console.log('Measuring article creation performance...');
    if (token) {
      measurements.metrics.article_creation = this.measureArticleCreationPerformance(BASE_URL, token);
    }

    console.log('Measuring database query performance...');
    measurements.metrics.database = this.measureDatabasePerformance(BASE_URL, headers);

    console.log('Measuring cache performance...');
    measurements.metrics.cache = this.measureCachePerformance(BASE_URL);

    console.log('Measuring system resources...');
    measurements.metrics.system = this.measureSystemResources(BASE_URL, headers);

    // Calculate aggregate metrics
    measurements.aggregates = this.calculateAggregateMetrics(measurements.metrics);

    return measurements;
  }

  authenticate() {
    const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
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

  measureEndpointPerformance(url, options = {}, iterations = 5) {
    const measurements = [];
    
    for (let i = 0; i < iterations; i++) {
      const start = Date.now();
      const response = http.get(url, options);
      const totalTime = Date.now() - start;
      
      measurements.push({
        status: response.status,
        duration: response.timings.duration,
        totalTime: totalTime,
        size: response.body ? response.body.length : 0,
        success: response.status >= 200 && response.status < 300,
      });
      
      // Small delay between measurements
      if (i < iterations - 1) {
        http.batch([]); // Small delay
      }
    }
    
    return this.calculateStatistics(measurements);
  }

  measureArticleCreationPerformance(baseUrl, token) {
    const measurements = [];
    const headers = {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    };
    
    for (let i = 0; i < 3; i++) { // 3 article creations for baseline
      const article = {
        title: `Baseline Test Article ${Date.now()}-${i}`,
        content: 'This is a baseline test article for performance measurement. '.repeat(100),
        excerpt: 'Baseline test article excerpt',
        author_id: 1,
        category_id: 1,
        status: 'published',
        tags: [1, 2],
      };
      
      const start = Date.now();
      const response = http.post(`${baseUrl}/api/v1/articles`, JSON.stringify(article), { headers });
      const totalTime = Date.now() - start;
      
      measurements.push({
        status: response.status,
        duration: response.timings.duration,
        totalTime: totalTime,
        success: response.status === 201,
      });
      
      // Clean up created article if possible
      if (response.status === 201) {
        try {
          const created = JSON.parse(response.body);
          // Note: In a real system, you might want to clean up test articles
        } catch (e) {
          // Ignore cleanup errors
        }
      }
    }
    
    return this.calculateStatistics(measurements);
  }

  measureDatabasePerformance(baseUrl, headers) {
    const dbQueries = [
      '/api/v1/articles?category=1&limit=50',
      '/api/v1/articles/trending?limit=20',
      '/api/v1/tags/popular?limit=30',
      '/api/v1/search?q=programming&category=1&limit=10',
    ];
    
    const measurements = {};
    
    dbQueries.forEach((query, index) => {
      const queryMeasurements = [];
      
      for (let i = 0; i < 3; i++) {
        const start = Date.now();
        const response = http.get(`${baseUrl}${query}`, { headers });
        const totalTime = Date.now() - start;
        
        queryMeasurements.push({
          status: response.status,
          duration: response.timings.duration,
          totalTime: totalTime,
          success: response.status === 200,
        });
      }
      
      measurements[`query_${index}`] = this.calculateStatistics(queryMeasurements);
    });
    
    return measurements;
  }

  measureCachePerformance(baseUrl) {
    const cacheableEndpoints = [
      '/',
      '/api/v1/categories',
      '/api/v1/tags?limit=20',
    ];
    
    const cacheMetrics = {};
    
    cacheableEndpoints.forEach((endpoint, index) => {
      // First request (cache miss)
      const firstResponse = http.get(`${baseUrl}${endpoint}`);
      
      // Second request (should be cache hit)
      const secondResponse = http.get(`${baseUrl}${endpoint}`);
      
      const cacheHit = secondResponse.headers['X-Cache-Status'] === 'HIT' ||
                       secondResponse.timings.duration < firstResponse.timings.duration * 0.7;
      
      cacheMetrics[`endpoint_${index}`] = {
        first_request_duration: firstResponse.timings.duration,
        second_request_duration: secondResponse.timings.duration,
        cache_hit_detected: cacheHit,
        cache_improvement: cacheHit ? 
          ((firstResponse.timings.duration - secondResponse.timings.duration) / firstResponse.timings.duration) : 0,
      };
    });
    
    // Calculate overall cache hit rate
    const hitCount = Object.values(cacheMetrics).filter(m => m.cache_hit_detected).length;
    const totalTests = Object.keys(cacheMetrics).length;
    
    return {
      endpoints: cacheMetrics,
      overall_hit_rate: hitCount / totalTests,
      total_tests: totalTests,
      hits: hitCount,
    };
  }

  measureSystemResources(baseUrl, headers) {
    const response = http.get(`${baseUrl}/api/v1/system/metrics`, { headers });
    
    if (response.status === 200) {
      try {
        const metrics = JSON.parse(response.body);
        return {
          memory_usage_mb: metrics.memory_usage_mb || 0,
          cpu_usage_percent: metrics.cpu_usage_percent || 0,
          db_connections_active: metrics.db_connections_active || 0,
          db_connections_max: metrics.db_connections_max || 0,
          cache_memory_mb: metrics.cache_memory_mb || 0,
          goroutines: metrics.goroutines || 0,
          timestamp: new Date().toISOString(),
        };
      } catch (e) {
        console.log('Could not parse system metrics:', e);
      }
    }
    
    return {
      memory_usage_mb: 0,
      cpu_usage_percent: 0,
      db_connections_active: 0,
      db_connections_max: 0,
      cache_memory_mb: 0,
      goroutines: 0,
      error: 'System metrics not available',
    };
  }

  calculateStatistics(measurements) {
    if (measurements.length === 0) {
      return { error: 'No measurements available' };
    }
    
    const durations = measurements.map(m => m.duration).filter(d => d !== undefined);
    const totalTimes = measurements.map(m => m.totalTime).filter(t => t !== undefined);
    const successCount = measurements.filter(m => m.success).length;
    
    durations.sort((a, b) => a - b);
    totalTimes.sort((a, b) => a - b);
    
    return {
      count: measurements.length,
      success_rate: successCount / measurements.length,
      duration: {
        min: Math.min(...durations),
        max: Math.max(...durations),
        avg: durations.reduce((a, b) => a + b, 0) / durations.length,
        p50: this.percentile(durations, 50),
        p95: this.percentile(durations, 95),
        p99: this.percentile(durations, 99),
      },
      total_time: {
        min: Math.min(...totalTimes),
        max: Math.max(...totalTimes),
        avg: totalTimes.reduce((a, b) => a + b, 0) / totalTimes.length,
        p95: this.percentile(totalTimes, 95),
      },
    };
  }

  percentile(values, p) {
    if (values.length === 0) return 0;
    const index = Math.ceil((p / 100) * values.length) - 1;
    return values[Math.max(0, index)];
  }

  calculateAggregateMetrics(metrics) {
    const aggregates = {
      homepage_p95: metrics.homepage?.duration?.p95 || 0,
      api_p95: 0,
      article_creation_p95: metrics.article_creation?.duration?.p95 || 0,
      db_query_p95: 0,
      cache_hit_rate: metrics.cache?.overall_hit_rate || 0,
      error_rate: 0,
      memory_usage_mb: metrics.system?.memory_usage_mb || 0,
      cpu_usage_percent: metrics.system?.cpu_usage_percent || 0,
    };
    
    // Calculate API aggregate
    if (metrics.api) {
      const apiP95s = Object.values(metrics.api)
        .map(endpoint => endpoint.duration?.p95)
        .filter(p95 => p95 !== undefined);
      
      if (apiP95s.length > 0) {
        aggregates.api_p95 = apiP95s.reduce((a, b) => a + b, 0) / apiP95s.length;
      }
    }
    
    // Calculate database aggregate
    if (metrics.database) {
      const dbP95s = Object.values(metrics.database)
        .map(query => query.duration?.p95)
        .filter(p95 => p95 !== undefined);
      
      if (dbP95s.length > 0) {
        aggregates.db_query_p95 = dbP95s.reduce((a, b) => a + b, 0) / dbP95s.length;
      }
    }
    
    // Calculate overall error rate
    const allMeasurements = [];
    if (metrics.homepage) allMeasurements.push(metrics.homepage);
    if (metrics.api) allMeasurements.push(...Object.values(metrics.api));
    if (metrics.search) allMeasurements.push(metrics.search);
    if (metrics.article_creation) allMeasurements.push(metrics.article_creation);
    
    if (allMeasurements.length > 0) {
      const totalSuccessRate = allMeasurements.reduce((sum, m) => sum + (m.success_rate || 0), 0);
      const avgSuccessRate = totalSuccessRate / allMeasurements.length;
      aggregates.error_rate = 1 - avgSuccessRate;
    }
    
    return aggregates;
  }

  storeBaseline(baseline) {
    // In a real implementation, this would save to a file or database
    console.log('Storing baseline metrics...');
    console.log('=== BASELINE METRICS ===');
    console.log(JSON.stringify(baseline, null, 2));
    console.log('=== END BASELINE METRICS ===');
    
    // Output environment variables for easy CI/CD integration
    console.log('\n=== ENVIRONMENT VARIABLES FOR CI/CD ===');
    const aggregates = baseline.aggregates;
    console.log(`export BASELINE_HOMEPAGE_P95=${aggregates.homepage_p95}`);
    console.log(`export BASELINE_API_P95=${aggregates.api_p95}`);
    console.log(`export BASELINE_ARTICLE_CREATION_P95=${aggregates.article_creation_p95}`);
    console.log(`export BASELINE_DB_QUERY_P95=${aggregates.db_query_p95}`);
    console.log(`export BASELINE_CACHE_HIT_RATE=${aggregates.cache_hit_rate}`);
    console.log(`export BASELINE_ERROR_RATE=${aggregates.error_rate}`);
    console.log(`export BASELINE_MEMORY_MB=${aggregates.memory_usage_mb}`);
    console.log(`export BASELINE_CPU_PERCENT=${aggregates.cpu_usage_percent}`);
    console.log(`export BASELINE_TIMESTAMP="${baseline.timestamp}"`);
    console.log('=== END ENVIRONMENT VARIABLES ===');
    
    return baseline;
  }

  // Compare current metrics against baseline
  compareWithBaseline(currentMetrics, baselineMetrics) {
    const comparison = {
      timestamp: new Date().toISOString(),
      baseline_timestamp: baselineMetrics.timestamp,
      comparisons: {},
      regressions: [],
      improvements: [],
    };
    
    const currentAgg = currentMetrics.aggregates;
    const baselineAgg = baselineMetrics.aggregates;
    
    Object.keys(baselineAgg).forEach(metric => {
      if (currentAgg[metric] !== undefined) {
        const current = currentAgg[metric];
        const baseline = baselineAgg[metric];
        const changePercent = baseline !== 0 ? ((current - baseline) / baseline) * 100 : 0;
        
        comparison.comparisons[metric] = {
          baseline: baseline,
          current: current,
          change_percent: changePercent,
          change_absolute: current - baseline,
        };
        
        // Determine if this is a regression or improvement
        const isRegression = this.isRegression(metric, changePercent);
        const isImprovement = this.isImprovement(metric, changePercent);
        
        if (isRegression) {
          comparison.regressions.push({
            metric: metric,
            change_percent: changePercent,
            severity: this.calculateRegressionSeverity(metric, changePercent),
          });
        } else if (isImprovement) {
          comparison.improvements.push({
            metric: metric,
            change_percent: changePercent,
          });
        }
      }
    });
    
    return comparison;
  }

  isRegression(metric, changePercent) {
    // Define what constitutes a regression for each metric type
    const regressionThresholds = {
      homepage_p95: 20, // 20% increase is a regression
      api_p95: 30, // 30% increase is a regression
      article_creation_p95: 25, // 25% increase is a regression
      db_query_p95: 50, // 50% increase is a regression
      cache_hit_rate: -10, // 10% decrease is a regression
      error_rate: 100, // 100% increase (doubling) is a regression
      memory_usage_mb: 30, // 30% increase is a regression
      cpu_usage_percent: 40, // 40% increase is a regression
    };
    
    const threshold = regressionThresholds[metric];
    if (threshold === undefined) return false;
    
    if (threshold < 0) {
      // For metrics where lower is better (cache hit rate)
      return changePercent < threshold;
    } else {
      // For metrics where higher is worse (response times, error rates)
      return changePercent > threshold;
    }
  }

  isImprovement(metric, changePercent) {
    // Define what constitutes an improvement for each metric type
    const improvementThresholds = {
      homepage_p95: -10, // 10% decrease is an improvement
      api_p95: -15, // 15% decrease is an improvement
      article_creation_p95: -10, // 10% decrease is an improvement
      db_query_p95: -20, // 20% decrease is an improvement
      cache_hit_rate: 5, // 5% increase is an improvement
      error_rate: -25, // 25% decrease is an improvement
      memory_usage_mb: -15, // 15% decrease is an improvement
      cpu_usage_percent: -20, // 20% decrease is an improvement
    };
    
    const threshold = improvementThresholds[metric];
    if (threshold === undefined) return false;
    
    if (threshold > 0) {
      // For metrics where higher is better (cache hit rate)
      return changePercent > threshold;
    } else {
      // For metrics where lower is better (response times, error rates)
      return changePercent < threshold;
    }
  }

  calculateRegressionSeverity(metric, changePercent) {
    const absChange = Math.abs(changePercent);
    
    if (absChange >= 100) return 'critical';
    if (absChange >= 50) return 'high';
    if (absChange >= 25) return 'medium';
    return 'low';
  }
}

export default BaselineManager;