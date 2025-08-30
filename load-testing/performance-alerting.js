import http from 'k6/http';

// Performance Alerting System
export class PerformanceAlerting {
  constructor() {
    this.alertChannels = {
      webhook: __ENV.ALERT_WEBHOOK_URL,
      slack: __ENV.SLACK_WEBHOOK_URL,
      email: __ENV.EMAIL_ALERT_ENDPOINT,
      teams: __ENV.TEAMS_WEBHOOK_URL,
    };
    
    this.alertThresholds = {
      critical: {
        response_time_increase: 100, // 100% increase
        error_rate_increase: 200, // 200% increase
        cache_hit_rate_decrease: 30, // 30% decrease
        throughput_decrease: 50, // 50% decrease
      },
      high: {
        response_time_increase: 50, // 50% increase
        error_rate_increase: 100, // 100% increase
        cache_hit_rate_decrease: 20, // 20% decrease
        throughput_decrease: 30, // 30% decrease
      },
      medium: {
        response_time_increase: 25, // 25% increase
        error_rate_increase: 50, // 50% increase
        cache_hit_rate_decrease: 15, // 15% decrease
        throughput_decrease: 20, // 20% decrease
      },
    };
  }

  // Send performance regression alerts
  sendRegressionAlerts(regressions, testMetadata = {}) {
    if (regressions.length === 0) {
      console.log('No regressions detected - no alerts to send');
      return;
    }

    const alertData = this.prepareAlertData(regressions, testMetadata);
    
    // Send to configured alert channels
    Object.keys(this.alertChannels).forEach(channel => {
      if (this.alertChannels[channel]) {
        this.sendAlert(channel, alertData);
      }
    });
  }

  prepareAlertData(regressions, testMetadata) {
    const criticalCount = regressions.filter(r => r.severity === 'critical').length;
    const highCount = regressions.filter(r => r.severity === 'high').length;
    const mediumCount = regressions.filter(r => r.severity === 'medium').length;
    const lowCount = regressions.filter(r => r.severity === 'low').length;

    const alertLevel = criticalCount > 0 ? 'critical' : 
                     highCount > 0 ? 'high' : 
                     mediumCount > 0 ? 'medium' : 'low';

    return {
      timestamp: new Date().toISOString(),
      alertLevel: alertLevel,
      summary: {
        total: regressions.length,
        critical: criticalCount,
        high: highCount,
        medium: mediumCount,
        low: lowCount,
      },
      regressions: regressions,
      testMetadata: {
        environment: __ENV.ENVIRONMENT || 'unknown',
        baseUrl: __ENV.BASE_URL || 'unknown',
        testDuration: testMetadata.duration || 'unknown',
        buildId: __ENV.BUILD_ID || 'unknown',
        commitHash: __ENV.COMMIT_HASH || 'unknown',
        branch: __ENV.BRANCH || 'unknown',
      },
      recommendations: this.generateAlertRecommendations(regressions),
    };
  }

  generateAlertRecommendations(regressions) {
    const recommendations = [];

    // Database performance recommendations
    const dbRegressions = regressions.filter(r => 
      r.metric.includes('db_') || r.metric.includes('query') || r.metric.includes('article_creation')
    );
    if (dbRegressions.length > 0) {
      recommendations.push({
        category: 'Database Performance',
        priority: 'high',
        actions: [
          'Review database query execution plans',
          'Check database connection pool utilization',
          'Verify database indexes are being used effectively',
          'Monitor database server resource usage',
        ],
        affectedMetrics: dbRegressions.map(r => r.metric),
      });
    }

    // API performance recommendations
    const apiRegressions = regressions.filter(r => 
      r.metric.includes('api_') || r.metric.includes('homepage_')
    );
    if (apiRegressions.length > 0) {
      recommendations.push({
        category: 'API Performance',
        priority: 'medium',
        actions: [
          'Review API endpoint response times',
          'Check application server resource usage',
          'Verify cache configuration and hit rates',
          'Review recent code changes for performance impact',
        ],
        affectedMetrics: apiRegressions.map(r => r.metric),
      });
    }

    // Cache performance recommendations
    const cacheRegressions = regressions.filter(r => r.metric.includes('cache'));
    if (cacheRegressions.length > 0) {
      recommendations.push({
        category: 'Cache Performance',
        priority: 'medium',
        actions: [
          'Review cache configuration and TTL settings',
          'Check cache server resource usage and connectivity',
          'Verify cache invalidation strategies',
          'Monitor cache memory usage and eviction rates',
        ],
        affectedMetrics: cacheRegressions.map(r => r.metric),
      });
    }

    // System resource recommendations
    const resourceRegressions = regressions.filter(r => 
      r.metric.includes('memory') || r.metric.includes('cpu')
    );
    if (resourceRegressions.length > 0) {
      recommendations.push({
        category: 'System Resources',
        priority: 'high',
        actions: [
          'Monitor system resource usage trends',
          'Check for memory leaks in application code',
          'Review CPU-intensive operations and algorithms',
          'Consider scaling resources if usage is consistently high',
        ],
        affectedMetrics: resourceRegressions.map(r => r.metric),
      });
    }

    return recommendations;
  }

  sendAlert(channel, alertData) {
    try {
      switch (channel) {
        case 'webhook':
          this.sendWebhookAlert(alertData);
          break;
        case 'slack':
          this.sendSlackAlert(alertData);
          break;
        case 'teams':
          this.sendTeamsAlert(alertData);
          break;
        case 'email':
          this.sendEmailAlert(alertData);
          break;
        default:
          console.log(`Unknown alert channel: ${channel}`);
      }
    } catch (error) {
      console.log(`Failed to send alert via ${channel}:`, error);
    }
  }

  sendWebhookAlert(alertData) {
    const webhookUrl = this.alertChannels.webhook;
    if (!webhookUrl) return;

    const payload = {
      alert_type: 'performance_regression',
      severity: alertData.alertLevel,
      timestamp: alertData.timestamp,
      summary: alertData.summary,
      environment: alertData.testMetadata.environment,
      build_id: alertData.testMetadata.buildId,
      regressions: alertData.regressions.map(r => ({
        metric: r.metric,
        change_percent: r.changePercent,
        severity: r.severity,
        baseline: r.baseline,
        current: r.current,
      })),
      recommendations: alertData.recommendations,
    };

    const response = http.post(webhookUrl, JSON.stringify(payload), {
      headers: { 'Content-Type': 'application/json' },
      tags: { alert_channel: 'webhook' },
    });

    if (response.status >= 200 && response.status < 300) {
      console.log('✅ Webhook alert sent successfully');
    } else {
      console.log(`❌ Webhook alert failed: ${response.status}`);
    }
  }

  sendSlackAlert(alertData) {
    const slackUrl = this.alertChannels.slack;
    if (!slackUrl) return;

    const color = this.getAlertColor(alertData.alertLevel);
    const emoji = this.getAlertEmoji(alertData.alertLevel);
    
    const regressionSummary = alertData.regressions
      .slice(0, 5) // Show top 5 regressions
      .map(r => `• ${r.metric}: ${r.changePercent.toFixed(1)}% change (${r.severity})`)
      .join('\n');

    const payload = {
      text: `${emoji} Performance Regression Alert`,
      attachments: [
        {
          color: color,
          title: `${alertData.summary.total} Performance Regressions Detected`,
          fields: [
            {
              title: 'Environment',
              value: alertData.testMetadata.environment,
              short: true,
            },
            {
              title: 'Build ID',
              value: alertData.testMetadata.buildId,
              short: true,
            },
            {
              title: 'Severity Breakdown',
              value: `Critical: ${alertData.summary.critical}, High: ${alertData.summary.high}, Medium: ${alertData.summary.medium}, Low: ${alertData.summary.low}`,
              short: false,
            },
            {
              title: 'Top Regressions',
              value: regressionSummary,
              short: false,
            },
          ],
          footer: 'Performance Monitoring',
          ts: Math.floor(Date.now() / 1000),
        },
      ],
    };

    const response = http.post(slackUrl, JSON.stringify(payload), {
      headers: { 'Content-Type': 'application/json' },
      tags: { alert_channel: 'slack' },
    });

    if (response.status >= 200 && response.status < 300) {
      console.log('✅ Slack alert sent successfully');
    } else {
      console.log(`❌ Slack alert failed: ${response.status}`);
    }
  }

  sendTeamsAlert(alertData) {
    const teamsUrl = this.alertChannels.teams;
    if (!teamsUrl) return;

    const color = this.getAlertColor(alertData.alertLevel);
    
    const regressionSummary = alertData.regressions
      .slice(0, 5)
      .map(r => `- **${r.metric}**: ${r.changePercent.toFixed(1)}% change (${r.severity})`)
      .join('\n');

    const payload = {
      '@type': 'MessageCard',
      '@context': 'http://schema.org/extensions',
      themeColor: color,
      summary: 'Performance Regression Alert',
      sections: [
        {
          activityTitle: '🚨 Performance Regression Alert',
          activitySubtitle: `${alertData.summary.total} regressions detected in ${alertData.testMetadata.environment}`,
          facts: [
            {
              name: 'Environment',
              value: alertData.testMetadata.environment,
            },
            {
              name: 'Build ID',
              value: alertData.testMetadata.buildId,
            },
            {
              name: 'Critical',
              value: alertData.summary.critical.toString(),
            },
            {
              name: 'High',
              value: alertData.summary.high.toString(),
            },
            {
              name: 'Medium',
              value: alertData.summary.medium.toString(),
            },
          ],
          text: `**Top Regressions:**\n${regressionSummary}`,
        },
      ],
    };

    const response = http.post(teamsUrl, JSON.stringify(payload), {
      headers: { 'Content-Type': 'application/json' },
      tags: { alert_channel: 'teams' },
    });

    if (response.status >= 200 && response.status < 300) {
      console.log('✅ Teams alert sent successfully');
    } else {
      console.log(`❌ Teams alert failed: ${response.status}`);
    }
  }

  sendEmailAlert(alertData) {
    const emailUrl = this.alertChannels.email;
    if (!emailUrl) return;

    const subject = `Performance Regression Alert - ${alertData.summary.total} issues detected`;
    const body = this.generateEmailBody(alertData);

    const payload = {
      to: __ENV.ALERT_EMAIL_RECIPIENTS || 'devops@company.com',
      subject: subject,
      body: body,
      priority: alertData.alertLevel === 'critical' ? 'high' : 'normal',
    };

    const response = http.post(emailUrl, JSON.stringify(payload), {
      headers: { 'Content-Type': 'application/json' },
      tags: { alert_channel: 'email' },
    });

    if (response.status >= 200 && response.status < 300) {
      console.log('✅ Email alert sent successfully');
    } else {
      console.log(`❌ Email alert failed: ${response.status}`);
    }
  }

  generateEmailBody(alertData) {
    let body = `Performance Regression Alert\n`;
    body += `================================\n\n`;
    body += `Timestamp: ${alertData.timestamp}\n`;
    body += `Environment: ${alertData.testMetadata.environment}\n`;
    body += `Build ID: ${alertData.testMetadata.buildId}\n`;
    body += `Branch: ${alertData.testMetadata.branch}\n\n`;
    
    body += `Summary:\n`;
    body += `- Total Regressions: ${alertData.summary.total}\n`;
    body += `- Critical: ${alertData.summary.critical}\n`;
    body += `- High: ${alertData.summary.high}\n`;
    body += `- Medium: ${alertData.summary.medium}\n`;
    body += `- Low: ${alertData.summary.low}\n\n`;
    
    body += `Regression Details:\n`;
    body += `-------------------\n`;
    alertData.regressions.forEach(r => {
      body += `${r.metric}:\n`;
      body += `  - Change: ${r.changePercent.toFixed(1)}%\n`;
      body += `  - Severity: ${r.severity}\n`;
      body += `  - Baseline: ${r.baseline}\n`;
      body += `  - Current: ${r.current}\n\n`;
    });
    
    if (alertData.recommendations.length > 0) {
      body += `Recommendations:\n`;
      body += `---------------\n`;
      alertData.recommendations.forEach(rec => {
        body += `${rec.category} (${rec.priority}):\n`;
        rec.actions.forEach(action => {
          body += `  - ${action}\n`;
        });
        body += `  Affected metrics: ${rec.affectedMetrics.join(', ')}\n\n`;
      });
    }
    
    return body;
  }

  getAlertColor(level) {
    const colors = {
      critical: '#FF0000',
      high: '#FF6600',
      medium: '#FFAA00',
      low: '#FFDD00',
    };
    return colors[level] || '#808080';
  }

  getAlertEmoji(level) {
    const emojis = {
      critical: '🚨',
      high: '⚠️',
      medium: '⚡',
      low: 'ℹ️',
    };
    return emojis[level] || '📊';
  }

  // Send success notification when no regressions are detected
  sendSuccessNotification(testMetadata = {}) {
    if (!__ENV.SEND_SUCCESS_NOTIFICATIONS) return;

    const successData = {
      timestamp: new Date().toISOString(),
      message: 'Performance test completed successfully - no regressions detected',
      testMetadata: testMetadata,
    };

    // Send to webhook only for success notifications
    if (this.alertChannels.webhook) {
      const payload = {
        alert_type: 'performance_success',
        severity: 'info',
        ...successData,
      };

      const response = http.post(this.alertChannels.webhook, JSON.stringify(payload), {
        headers: { 'Content-Type': 'application/json' },
        tags: { alert_channel: 'webhook', alert_type: 'success' },
      });

      if (response.status >= 200 && response.status < 300) {
        console.log('✅ Success notification sent');
      }
    }
  }
}

export default PerformanceAlerting;