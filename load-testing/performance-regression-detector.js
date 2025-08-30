import http from 'k6/http';
import { check } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

// Performance regression detection system
export class PerformanceRegressionDetector {
  constructor(baselineFile = 'performance-baseline.json') {
    this.baselineFile = baselineFile;
    this.currentMetrics = {};
    this.regressions = [];
    this.alerts = [];
  }

  // Load baseline metrics from storage
  loadBaseline() {
    try {
      // In a real implementation, this would load from a file or database
      // For k6, we'll use environment variables or hardcoded values
      return {
        homepage_p95: parseFloat(__ENV.BASELINE_HOMEPAGE_P95) || 500,
        api_p95: parseFloat(__ENV.BASELINE_API_P95) || 100,
        article_creation_p95: parseFloat(__ENV.BASELINE_ARTICLE_CREATION_P95) || 1000,
        db_query_p95: parseFloat(__ENV.BASELINE_DB_QUERY_P95) || 10,
        cache_hit_rate: parseFloat(__ENV.BASELINE_CACHE_HIT_RATE) || 0.8,
        error_rate: parseFloat(__ENV.BASELINE_ERROR_RATE) || 0.02,
        articles_per_minute: parseFloat(__ENV.BASELINE_ARTICLES_PER_MINUTE) || 35,
        memory_usage_mb: parseFloat(__ENV.BASELINE_MEMORY_MB) || 800,
        cpu_usage_percent: parseFloat(__ENV.BASELINE_CPU_PERCENT) || 60,
        timestamp: __ENV.BASELINE_TIMESTAMP || new Date().toISOString(),
      };
    } catch (error) {
      console.log('Warning: Could not load baseline metrics, using defaults');
      return this.getDefaultBaseline();
    }
  }

  getDefaultBaseline() {
    return {
      homepage_p95: 500,
      api_p95: 100,
      article_creation_p95: 1000,
      db_query_p95: 10,
      cache_hit_rate: 0.8,
      error_rate: 0.02,
      articles_per_minute: 35,
      memory_usage_mb: 800,
      cpu_usage_percent: 60,
      timestamp: new Date().toISOString(),
    };
  }

  // Store current performance metrics
  recordMetric(name, value, tags = {}) {
    if (!this.currentMetrics[name]) {
      this.currentMetrics[name] = [];
    }
    
    this.currentMetrics[name].push({
      value: value,
      timestamp: Date.now(),
      tags: tags,
    });
  }

  // Calculate percentiles from recorded metrics
  calculatePercentile(values, percentile) {
    if (values.length === 0) return 0;
    
    const sorted = values.sort((a, b) => a - b);
    const index = Math.ceil((percentile / 100) * sorted.length) - 1;
    return sorted[Math.max(0, index)];
  }

  // Detect performance regressions
  detectRegressions() {
    const baseline = this.loadBaseline();
    const regressions = [];

    // Calculate current metrics
    const currentStats = this.calculateCurrentStats();

    // Define regression thresholds (percentage increase that triggers alert)
    const thresholds = {
      homepage_p95: 0.2, // 20% increase
      api_p95: 0.3, // 30% increase
      article_creation_p95: 0.25, // 25% increase
      db_query_p95: 0.5, // 50% increase
      cache_hit_rate: -0.1, // 10% decrease (negative because lower is worse)
      error_rate: 1.0, // 100% increase (double the error rate)
      articles_per_minute: -0.15, // 15% decrease
      memory_usage_mb: 0.3, // 30% increase
      cpu_usage_percent: 0.4, // 40% increase
    };

    // Check each metric for regressions
    Object.keys(thresholds).forEach(metric => {
      const baselineValue = baseline[metric];
      const currentValue = currentStats[metric];
      const threshold = thresholds[metric];

      if (baselineValue && currentValue !== undefined) {
        let regressionDetected = false;
        let changePercent = 0;

        if (threshold < 0) {
          // For metrics where lower is worse (cache hit rate, throughput)
          changePercent = (baselineValue - currentValue) / baselineValue;
          regressionDetected = changePercent > Math.abs(threshold);
        } else {
          // For metrics where higher is worse (response time, error rate)
          changePercent = (currentValue - baselineValue) / baselineValue;
          regressionDetected = changePercent > threshold;
        }

        if (regressionDetected) {
          const regression = {
            metric: metric,
            baseline: baselineValue,
            current: currentValue,
            changePercent: changePercent * 100,
            threshold: threshold * 100,
            severity: this.calculateSeverity(changePercent, threshold),
            timestamp: new Date().toISOString(),
          };

          regressions.push(regression);
          this.regressions.push(regression);
        }
      }
    });

    return regressions;
  }

  calculateSeverity(changePercent, threshold) {
    const ratio = Math.abs(changePercent) / Math.abs(threshold);
    
    if (ratio >= 2.0) return 'critical';
    if (ratio >= 1.5) return 'high';
    if (ratio >= 1.0) return 'medium';
    return 'low';
  }

  calculateCurrentStats() {
    const stats = {};

    // Calculate stats from recorded metrics
    Object.keys(this.currentMetrics).forEach(metricName => {
      const values = this.currentMetrics[metricName].map(m => m.value);
      
      switch (metricName) {
        case 'homepage_response_time':
          stats.homepage_p95 = this.calculatePercentile(values, 95);
          break;
        case 'api_response_time':
          stats.api_p95 = this.calculatePercentile(values, 95);
          break;
        case 'article_creation_time':
          stats.article_creation_p95 = this.calculatePercentile(values, 95);
          break;
        case 'db_query_time':
          stats.db_query_p95 = this.calculatePercentile(values, 95);
          break;
        case 'cache_hit':
          stats.cache_hit_rate = values.reduce((a, b) => a + b, 0) / values.length;
          break;
        case 'error_occurred':
          stats.error_rate = values.reduce((a, b) => a + b, 0) / values.length;
          break;
        case 'articles_created':
          stats.articles_per_minute = values.length; // Assuming 1-minute test
          break;
        case 'memory_usage':
          stats.memory_usage_mb = values[values.length - 1] || 0; // Latest value
          break;
        case 'cpu_usage':
          stats.cpu_usage_percent = values[values.length - 1] || 0; // Latest value
          break;
      }
    });

    return stats;
  }

  // Generate performance alerts
  generateAlerts(regressions) {
    const alerts = [];

    regressions.forEach(regression => {
      let alertMessage = '';
      let recommendation = '';

      switch (regression.metric) {
        case 'homepage_p95':
          alertMessage = `Homepage response time increased by ${regression.changePercent.toFixed(1)}%`;
          recommendation = 'Check cache configuration and static file serving';
          break;
        case 'api_p95':
          alertMessage = `API response time increased by ${regression.changePercent.toFixed(1)}%`;
          recommendation = 'Review database query performance and connection pooling';
          break;
        case 'article_creation_p95':
          alertMessage = `Article creation time increased by ${regression.changePercent.toFixed(1)}%`;
          recommendation = 'Check database write performance and cache invalidation';
          break;
        case 'db_query_p95':
          alertMessage = `Database query time increased by ${regression.changePercent.toFixed(1)}%`;
          recommendation = 'Review query plans and database indexes';
          break;
        case 'cache_hit_rate':
          alertMessage = `Cache hit rate decreased by ${Math.abs(regression.changePercent).toFixed(1)}%`;
          recommendation = 'Check cache configuration and TTL settings';
          break;
        case 'error_rate':
          alertMessage = `Error rate increased by ${regression.changePercent.toFixed(1)}%`;
          recommendation = 'Review application logs and error handling';
          break;
        case 'articles_per_minute':
          alertMessage = `Article creation throughput decreased by ${Math.abs(regression.changePercent).toFixed(1)}%`;
          recommendation = 'Check system resources and database performance';
          break;
        case 'memory_usage_mb':
          alertMessage = `Memory usage increased by ${regression.changePercent.toFixed(1)}%`;
          recommendation = 'Check for memory leaks and optimize memory usage';
          break;
        case 'cpu_usage_percent':
          alertMessage = `CPU usage increased by ${regression.changePercent.toFixed(1)}%`;
          recommendation = 'Review CPU-intensive operations and optimize algorithms';
          break;
      }

      const alert = {
        severity: regression.severity,
        metric: regression.metric,
        message: alertMessage,
        recommendation: recommendation,
        baseline: regression.baseline,
        current: regression.current,
        changePercent: regression.changePercent,
        timestamp: regression.timestamp,
      };

      alerts.push(alert);
      this.alerts.push(alert);
    });

    return alerts;
  }

  // Generate performance trend analysis
  generateTrendAnalysis() {
    const currentStats = this.calculateCurrentStats();
    const baseline = this.loadBaseline();
    
    const trends = {
      improving: [],
      stable: [],
      degrading: [],
      summary: {
        totalMetrics: Object.keys(currentStats).length,
        improvingCount: 0,
        stableCount: 0,
        degradingCount: 0,
      },
    };

    Object.keys(currentStats).forEach(metric => {
      const current = currentStats[metric];
      const baselineValue = baseline[metric];
      
      if (baselineValue && current !== undefined) {
        const changePercent = (current - baselineValue) / baselineValue * 100;
        
        // Determine if metric is "lower is better" or "higher is better"
        const lowerIsBetter = ['homepage_p95', 'api_p95', 'article_creation_p95', 
                              'db_query_p95', 'error_rate', 'memory_usage_mb', 'cpu_usage_percent'];
        
        let trend = 'stable';
        if (Math.abs(changePercent) > 5) { // 5% threshold for trend detection
          if (lowerIsBetter.includes(metric)) {
            trend = changePercent < 0 ? 'improving' : 'degrading';
          } else {
            trend = changePercent > 0 ? 'improving' : 'degrading';
          }
        }

        trends[trend].push({
          metric: metric,
          baseline: baselineValue,
          current: current,
          changePercent: changePercent,
        });

        trends.summary[`${trend}Count`]++;
      }
    });

    return trends;
  }

  // Save current metrics as new baseline
  updateBaseline() {
    const currentStats = this.calculateCurrentStats();
    const newBaseline = {
      ...currentStats,
      timestamp: new Date().toISOString(),
    };

    // In a real implementation, this would save to a file or database
    console.log('New baseline metrics:', JSON.stringify(newBaseline, null, 2));
    
    // For k6, we can output this to be captured by the test runner
    return newBaseline;
  }

  // Generate comprehensive performance report
  generateReport() {
    const regressions = this.detectRegressions();
    const alerts = this.generateAlerts(regressions);
    const trends = this.generateTrendAnalysis();
    const currentStats = this.calculateCurrentStats();
    const baseline = this.loadBaseline();

    const report = {
      timestamp: new Date().toISOString(),
      testDuration: __ENV.K6_DURATION || 'unknown',
      baseline: baseline,
      current: currentStats,
      regressions: {
        count: regressions.length,
        details: regressions,
      },
      alerts: {
        critical: alerts.filter(a => a.severity === 'critical').length,
        high: alerts.filter(a => a.severity === 'high').length,
        medium: alerts.filter(a => a.severity === 'medium').length,
        low: alerts.filter(a => a.severity === 'low').length,
        details: alerts,
      },
      trends: trends,
      recommendations: this.generateRecommendations(regressions, trends),
    };

    return report;
  }

  generateRecommendations(regressions, trends) {
    const recommendations = [];

    // Critical regressions
    const criticalRegressions = regressions.filter(r => r.severity === 'critical');
    if (criticalRegressions.length > 0) {
      recommendations.push({
        priority: 'critical',
        action: 'Immediate investigation required',
        details: `${criticalRegressions.length} critical performance regressions detected`,
        metrics: criticalRegressions.map(r => r.metric),
      });
    }

    // Multiple degrading trends
    if (trends.degrading.length > trends.improving.length) {
      recommendations.push({
        priority: 'high',
        action: 'Performance optimization needed',
        details: `${trends.degrading.length} metrics showing degradation vs ${trends.improving.length} improving`,
        metrics: trends.degrading.map(t => t.metric),
      });
    }

    // Database performance issues
    const dbMetrics = regressions.filter(r => 
      r.metric.includes('db_') || r.metric.includes('query') || r.metric.includes('article_creation')
    );
    if (dbMetrics.length > 0) {
      recommendations.push({
        priority: 'medium',
        action: 'Database performance review',
        details: 'Database-related metrics showing performance issues',
        metrics: dbMetrics.map(r => r.metric),
      });
    }

    // Resource usage concerns
    const resourceMetrics = regressions.filter(r => 
      r.metric.includes('memory') || r.metric.includes('cpu')
    );
    if (resourceMetrics.length > 0) {
      recommendations.push({
        priority: 'medium',
        action: 'Resource usage optimization',
        details: 'System resource usage has increased significantly',
        metrics: resourceMetrics.map(r => r.metric),
      });
    }

    return recommendations;
  }
}

// Export for use in other k6 scripts
export default PerformanceRegressionDetector;