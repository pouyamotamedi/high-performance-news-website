import http from 'k6/http';

// Performance Trend Analysis System
export class PerformanceTrendAnalyzer {
  constructor() {
    this.trendWindow = parseInt(__ENV.TREND_WINDOW_DAYS) || 30; // 30 days default
    this.trendThresholds = {
      degradation: 0.05, // 5% degradation over trend window
      improvement: 0.05, // 5% improvement over trend window
      volatility: 0.2, // 20% coefficient of variation indicates high volatility
    };
  }

  // Analyze performance trends from historical data
  analyzeTrends(historicalMetrics) {
    const trends = {
      timestamp: new Date().toISOString(),
      analysis_period_days: this.trendWindow,
      metrics: {},
      summary: {
        improving: [],
        stable: [],
        degrading: [],
        volatile: [],
      },
      recommendations: [],
    };

    // Analyze each metric
    Object.keys(historicalMetrics).forEach(metricName => {
      const metricData = historicalMetrics[metricName];
      const trendAnalysis = this.analyzeMetricTrend(metricName, metricData);
      
      trends.metrics[metricName] = trendAnalysis;
      
      // Categorize trend
      if (trendAnalysis.volatility > this.trendThresholds.volatility) {
        trends.summary.volatile.push(metricName);
      } else if (trendAnalysis.trend_direction === 'improving') {
        trends.summary.improving.push(metricName);
      } else if (trendAnalysis.trend_direction === 'degrading') {
        trends.summary.degrading.push(metricName);
      } else {
        trends.summary.stable.push(metricName);
      }
    });

    // Generate recommendations based on trends
    trends.recommendations = this.generateTrendRecommendations(trends);

    return trends;
  }

  analyzeMetricTrend(metricName, metricData) {
    if (!metricData || metricData.length < 2) {
      return {
        error: 'Insufficient data for trend analysis',
        data_points: metricData ? metricData.length : 0,
      };
    }

    // Sort data by timestamp
    const sortedData = metricData.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
    const values = sortedData.map(d => d.value);
    const timestamps = sortedData.map(d => new Date(d.timestamp).getTime());

    // Calculate basic statistics
    const mean = values.reduce((sum, val) => sum + val, 0) / values.length;
    const variance = values.reduce((sum, val) => sum + Math.pow(val - mean, 2), 0) / values.length;
    const stdDev = Math.sqrt(variance);
    const coefficientOfVariation = mean !== 0 ? stdDev / mean : 0;

    // Calculate linear trend using least squares regression
    const trendStats = this.calculateLinearTrend(timestamps, values);
    
    // Determine trend direction and significance
    const trendDirection = this.determineTrendDirection(metricName, trendStats.slope, trendStats.rSquared);
    
    // Calculate recent vs historical comparison
    const recentComparison = this.calculateRecentComparison(values);

    return {
      data_points: values.length,
      time_range: {
        start: sortedData[0].timestamp,
        end: sortedData[sortedData.length - 1].timestamp,
      },
      statistics: {
        mean: mean,
        min: Math.min(...values),
        max: Math.max(...values),
        std_dev: stdDev,
        coefficient_of_variation: coefficientOfVariation,
      },
      trend: {
        slope: trendStats.slope,
        r_squared: trendStats.rSquared,
        direction: trendDirection,
        significance: trendStats.rSquared > 0.5 ? 'significant' : 'weak',
      },
      volatility: coefficientOfVariation,
      recent_comparison: recentComparison,
      alerts: this.generateMetricAlerts(metricName, trendDirection, coefficientOfVariation, recentComparison),
    };
  }

  calculateLinearTrend(timestamps, values) {
    const n = timestamps.length;
    if (n < 2) return { slope: 0, rSquared: 0 };

    // Normalize timestamps to avoid large numbers
    const minTime = Math.min(...timestamps);
    const normalizedTimes = timestamps.map(t => (t - minTime) / (1000 * 60 * 60 * 24)); // Convert to days

    // Calculate linear regression
    const sumX = normalizedTimes.reduce((sum, x) => sum + x, 0);
    const sumY = values.reduce((sum, y) => sum + y, 0);
    const sumXY = normalizedTimes.reduce((sum, x, i) => sum + x * values[i], 0);
    const sumXX = normalizedTimes.reduce((sum, x) => sum + x * x, 0);
    const sumYY = values.reduce((sum, y) => sum + y * y, 0);

    const slope = (n * sumXY - sumX * sumY) / (n * sumXX - sumX * sumX);
    const intercept = (sumY - slope * sumX) / n;

    // Calculate R-squared
    const meanY = sumY / n;
    const ssRes = values.reduce((sum, y, i) => {
      const predicted = slope * normalizedTimes[i] + intercept;
      return sum + Math.pow(y - predicted, 2);
    }, 0);
    const ssTot = values.reduce((sum, y) => sum + Math.pow(y - meanY, 2), 0);
    const rSquared = ssTot !== 0 ? 1 - (ssRes / ssTot) : 0;

    return { slope, intercept, rSquared };
  }

  determineTrendDirection(metricName, slope, rSquared) {
    // For metrics where lower is better
    const lowerIsBetter = [
      'homepage_p95', 'api_p95', 'article_creation_p95', 'db_query_p95',
      'error_rate', 'memory_usage_mb', 'cpu_usage_percent'
    ];

    const isSignificant = rSquared > 0.3; // 30% of variance explained
    const slopeThreshold = 0.01; // Minimum slope to consider as trend

    if (!isSignificant || Math.abs(slope) < slopeThreshold) {
      return 'stable';
    }

    if (lowerIsBetter.includes(metricName)) {
      return slope < 0 ? 'improving' : 'degrading';
    } else {
      // For metrics where higher is better (cache_hit_rate, throughput)
      return slope > 0 ? 'improving' : 'degrading';
    }
  }

  calculateRecentComparison(values) {
    if (values.length < 4) return { error: 'Insufficient data' };

    // Compare recent 25% of data with earlier 25%
    const quarterSize = Math.floor(values.length / 4);
    const recentValues = values.slice(-quarterSize);
    const earlierValues = values.slice(0, quarterSize);

    const recentMean = recentValues.reduce((sum, val) => sum + val, 0) / recentValues.length;
    const earlierMean = earlierValues.reduce((sum, val) => sum + val, 0) / earlierValues.length;

    const changePercent = earlierMean !== 0 ? ((recentMean - earlierMean) / earlierMean) * 100 : 0;

    return {
      recent_mean: recentMean,
      earlier_mean: earlierMean,
      change_percent: changePercent,
      recent_period_size: quarterSize,
      comparison_period_size: quarterSize,
    };
  }

  generateMetricAlerts(metricName, trendDirection, volatility, recentComparison) {
    const alerts = [];

    // High volatility alert
    if (volatility > this.trendThresholds.volatility) {
      alerts.push({
        type: 'high_volatility',
        severity: 'medium',
        message: `${metricName} shows high volatility (CV: ${(volatility * 100).toFixed(1)}%)`,
        recommendation: 'Investigate causes of performance variability',
      });
    }

    // Degradation trend alert
    if (trendDirection === 'degrading') {
      alerts.push({
        type: 'degradation_trend',
        severity: 'high',
        message: `${metricName} shows degrading trend over time`,
        recommendation: 'Review recent changes and optimize performance',
      });
    }

    // Recent performance change alert
    if (recentComparison.change_percent && Math.abs(recentComparison.change_percent) > 10) {
      const changeType = recentComparison.change_percent > 0 ? 'increase' : 'decrease';
      const severity = Math.abs(recentComparison.change_percent) > 25 ? 'high' : 'medium';
      
      alerts.push({
        type: 'recent_change',
        severity: severity,
        message: `${metricName} shows ${Math.abs(recentComparison.change_percent).toFixed(1)}% ${changeType} in recent period`,
        recommendation: 'Review recent deployments and system changes',
      });
    }

    return alerts;
  }

  generateTrendRecommendations(trends) {
    const recommendations = [];

    // Overall system health recommendation
    const totalMetrics = Object.keys(trends.metrics).length;
    const degradingCount = trends.summary.degrading.length;
    const volatileCount = trends.summary.volatile.length;

    if (degradingCount > totalMetrics * 0.3) {
      recommendations.push({
        priority: 'critical',
        category: 'System Health',
        title: 'Multiple metrics showing degradation',
        description: `${degradingCount} out of ${totalMetrics} metrics are degrading`,
        actions: [
          'Conduct comprehensive performance review',
          'Review recent deployments and changes',
          'Consider system resource scaling',
          'Implement performance optimization initiatives',
        ],
        affected_metrics: trends.summary.degrading,
      });
    }

    if (volatileCount > totalMetrics * 0.2) {
      recommendations.push({
        priority: 'high',
        category: 'System Stability',
        title: 'High performance volatility detected',
        description: `${volatileCount} metrics showing high volatility`,
        actions: [
          'Investigate intermittent performance issues',
          'Review system resource allocation',
          'Check for external dependencies causing instability',
          'Implement performance monitoring alerts',
        ],
        affected_metrics: trends.summary.volatile,
      });
    }

    // Database-specific recommendations
    const dbMetrics = Object.keys(trends.metrics).filter(m => 
      m.includes('db_') || m.includes('query') || m.includes('article_creation')
    );
    const degradingDbMetrics = dbMetrics.filter(m => trends.summary.degrading.includes(m));

    if (degradingDbMetrics.length > 0) {
      recommendations.push({
        priority: 'high',
        category: 'Database Performance',
        title: 'Database performance degradation',
        description: 'Database-related metrics showing performance decline',
        actions: [
          'Review database query performance and execution plans',
          'Check database server resource utilization',
          'Analyze database connection pool configuration',
          'Consider database optimization and indexing',
        ],
        affected_metrics: degradingDbMetrics,
      });
    }

    // Cache performance recommendations
    const cacheMetrics = Object.keys(trends.metrics).filter(m => m.includes('cache'));
    const degradingCacheMetrics = cacheMetrics.filter(m => trends.summary.degrading.includes(m));

    if (degradingCacheMetrics.length > 0) {
      recommendations.push({
        priority: 'medium',
        category: 'Cache Performance',
        title: 'Cache performance issues detected',
        description: 'Cache-related metrics showing degradation',
        actions: [
          'Review cache configuration and TTL settings',
          'Check cache server health and connectivity',
          'Analyze cache hit rates and eviction patterns',
          'Consider cache warming strategies',
        ],
        affected_metrics: degradingCacheMetrics,
      });
    }

    // Positive trend recognition
    if (trends.summary.improving.length > trends.summary.degrading.length) {
      recommendations.push({
        priority: 'info',
        category: 'Performance Improvement',
        title: 'Positive performance trends detected',
        description: `${trends.summary.improving.length} metrics showing improvement`,
        actions: [
          'Document successful optimizations for future reference',
          'Consider applying similar improvements to other areas',
          'Monitor to ensure improvements are sustained',
        ],
        affected_metrics: trends.summary.improving,
      });
    }

    return recommendations;
  }

  // Generate comprehensive trend report
  generateTrendReport(trends) {
    const report = {
      executive_summary: this.generateExecutiveSummary(trends),
      detailed_analysis: trends,
      action_items: this.prioritizeActionItems(trends.recommendations),
      monitoring_recommendations: this.generateMonitoringRecommendations(trends),
    };

    return report;
  }

  generateExecutiveSummary(trends) {
    const totalMetrics = Object.keys(trends.metrics).length;
    const summary = trends.summary;

    return {
      overview: `Performance trend analysis of ${totalMetrics} metrics over ${trends.analysis_period_days} days`,
      key_findings: [
        `${summary.improving.length} metrics showing improvement`,
        `${summary.stable.length} metrics remaining stable`,
        `${summary.degrading.length} metrics showing degradation`,
        `${summary.volatile.length} metrics showing high volatility`,
      ],
      overall_health: this.calculateOverallHealth(summary, totalMetrics),
      priority_actions: trends.recommendations
        .filter(r => r.priority === 'critical' || r.priority === 'high')
        .length,
    };
  }

  calculateOverallHealth(summary, totalMetrics) {
    const improvingRatio = summary.improving.length / totalMetrics;
    const degradingRatio = summary.degrading.length / totalMetrics;
    const volatileRatio = summary.volatile.length / totalMetrics;

    if (degradingRatio > 0.3 || volatileRatio > 0.3) {
      return 'poor';
    } else if (degradingRatio > 0.15 || volatileRatio > 0.15) {
      return 'fair';
    } else if (improvingRatio > 0.3) {
      return 'excellent';
    } else {
      return 'good';
    }
  }

  prioritizeActionItems(recommendations) {
    const priorityOrder = { critical: 0, high: 1, medium: 2, low: 3, info: 4 };
    
    return recommendations
      .sort((a, b) => priorityOrder[a.priority] - priorityOrder[b.priority])
      .map((rec, index) => ({
        id: index + 1,
        priority: rec.priority,
        category: rec.category,
        title: rec.title,
        description: rec.description,
        actions: rec.actions,
        affected_metrics: rec.affected_metrics,
        estimated_effort: this.estimateEffort(rec),
        expected_impact: this.estimateImpact(rec),
      }));
  }

  estimateEffort(recommendation) {
    const effortMap = {
      'System Health': 'high',
      'Database Performance': 'medium',
      'Cache Performance': 'low',
      'System Stability': 'medium',
      'Performance Improvement': 'low',
    };
    
    return effortMap[recommendation.category] || 'medium';
  }

  estimateImpact(recommendation) {
    const impactMap = {
      critical: 'high',
      high: 'high',
      medium: 'medium',
      low: 'low',
      info: 'low',
    };
    
    return impactMap[recommendation.priority] || 'medium';
  }

  generateMonitoringRecommendations(trends) {
    const recommendations = [];

    // Metrics that need closer monitoring
    const volatileMetrics = trends.summary.volatile;
    const degradingMetrics = trends.summary.degrading;

    if (volatileMetrics.length > 0) {
      recommendations.push({
        type: 'increased_monitoring',
        metrics: volatileMetrics,
        frequency: 'every 5 minutes',
        reason: 'High volatility detected',
        alert_thresholds: 'Tighten alert thresholds by 20%',
      });
    }

    if (degradingMetrics.length > 0) {
      recommendations.push({
        type: 'trend_monitoring',
        metrics: degradingMetrics,
        frequency: 'daily trend analysis',
        reason: 'Degradation trend detected',
        alert_thresholds: 'Set up trend-based alerts',
      });
    }

    // New metrics to consider monitoring
    recommendations.push({
      type: 'additional_metrics',
      suggestions: [
        'Database connection pool utilization',
        'Cache memory usage and eviction rates',
        'Application garbage collection metrics',
        'External API response times',
        'Queue processing latency',
      ],
      reason: 'Improve observability of system components',
    });

    return recommendations;
  }
}

export default PerformanceTrendAnalyzer;