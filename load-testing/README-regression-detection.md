# Performance Regression Detection System

This document describes the comprehensive performance regression detection system implemented for the news website load testing infrastructure.

## Overview

The performance regression detection system provides:

- **Baseline Performance Metrics Storage**: Establish and maintain performance baselines
- **Automated Regression Detection**: Compare current performance against baselines
- **Performance Trend Analysis**: Analyze performance trends over time
- **Intelligent Alerting**: Send alerts when regressions are detected
- **CI/CD Integration**: Integrate with build pipelines for automated testing

## Components

### 1. Performance Regression Detector (`performance-regression-detector.js`)

Core component that:
- Loads baseline metrics from environment variables or configuration
- Records current performance metrics during test execution
- Detects regressions by comparing current vs baseline performance
- Calculates regression severity (critical, high, medium, low)
- Generates comprehensive performance reports

### 2. Baseline Manager (`baseline-manager.js`)

Manages performance baselines:
- Establishes new baselines by measuring comprehensive metrics
- Validates baseline quality before storage
- Provides utilities for baseline comparison
- Outputs environment variables for CI/CD integration

### 3. Performance Alerting (`performance-alerting.js`)

Handles alert notifications:
- Supports multiple channels (Slack, Teams, Webhook, Email)
- Generates contextual alert messages with recommendations
- Provides severity-based alert routing
- Includes success notifications for clean test runs

### 4. Performance Trend Analyzer (`performance-trend-analyzer.js`)

Analyzes performance trends:
- Performs statistical analysis on historical metrics
- Detects trend directions (improving, stable, degrading)
- Identifies high volatility metrics
- Generates trend-based recommendations

### 5. Integrated Regression Test (`integrated-regression-test.js`)

Comprehensive test that:
- Combines all regression detection components
- Runs realistic load scenarios with weighted distribution
- Collects comprehensive performance metrics
- Generates detailed regression reports
- Integrates with CI/CD pipelines

## Usage

### 1. Establish Performance Baseline

First, establish a performance baseline when your system is performing optimally:

```bash
# Run baseline establishment
k6 run load-testing/establish-baseline.js \
  --env BASE_URL=http://localhost:8080 \
  --env TEST_USERNAME=testuser \
  --env TEST_PASSWORD=testpass123 \
  --env ENVIRONMENT=production

# Copy the output environment variables to your CI/CD configuration
```

### 2. Run Regression Detection Tests

Run regression detection tests as part of your CI/CD pipeline:

```bash
# Run integrated regression test
k6 run load-testing/integrated-regression-test.js \
  --env BASE_URL=http://localhost:8080 \
  --env TEST_USERNAME=testuser \
  --env TEST_PASSWORD=testpass123 \
  --env BASELINE_HOMEPAGE_P95=500 \
  --env BASELINE_API_P95=100 \
  --env BASELINE_ARTICLE_CREATION_P95=1000 \
  --env BASELINE_DB_QUERY_P95=10 \
  --env BASELINE_CACHE_HIT_RATE=0.8 \
  --env BASELINE_ERROR_RATE=0.02 \
  --env TEST_VUS=25 \
  --env TEST_DURATION=5m \
  --env ENVIRONMENT=staging \
  --env BUILD_ID=$BUILD_ID \
  --env COMMIT_HASH=$COMMIT_HASH \
  --env BRANCH=$BRANCH
```

### 3. Configure Alerting

Set up alert channels by configuring environment variables:

```bash
# Slack integration
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"

# Teams integration
export TEAMS_WEBHOOK_URL="https://outlook.office.com/webhook/YOUR/TEAMS/WEBHOOK"

# Generic webhook
export ALERT_WEBHOOK_URL="https://your-monitoring-system.com/webhooks/performance"

# Email alerts
export EMAIL_ALERT_ENDPOINT="https://your-email-service.com/send"
export ALERT_EMAIL_RECIPIENTS="devops@company.com,performance@company.com"
```

### 4. Enable Trend Analysis

Enable trend analysis for historical performance tracking:

```bash
k6 run load-testing/integrated-regression-test.js \
  --env ENABLE_TREND_ANALYSIS=true \
  --env TREND_WINDOW_DAYS=30 \
  # ... other environment variables
```

## Configuration

### Environment Variables

#### Required Variables
- `BASE_URL`: Target application URL
- `TEST_USERNAME`: Test user username
- `TEST_PASSWORD`: Test user password

#### Baseline Metrics
- `BASELINE_HOMEPAGE_P95`: Homepage 95th percentile response time (ms)
- `BASELINE_API_P95`: API 95th percentile response time (ms)
- `BASELINE_ARTICLE_CREATION_P95`: Article creation 95th percentile time (ms)
- `BASELINE_DB_QUERY_P95`: Database query 95th percentile time (ms)
- `BASELINE_CACHE_HIT_RATE`: Expected cache hit rate (0.0-1.0)
- `BASELINE_ERROR_RATE`: Expected error rate (0.0-1.0)
- `BASELINE_MEMORY_MB`: Expected memory usage (MB)
- `BASELINE_CPU_PERCENT`: Expected CPU usage (%)
- `BASELINE_TIMESTAMP`: Baseline establishment timestamp

#### Test Configuration
- `TEST_VUS`: Number of virtual users (default: 25)
- `TEST_DURATION`: Test duration (default: 5m)
- `ENVIRONMENT`: Environment name (development, staging, production)
- `BUILD_ID`: CI/CD build identifier
- `COMMIT_HASH`: Git commit hash
- `BRANCH`: Git branch name

#### Alert Configuration
- `SLACK_WEBHOOK_URL`: Slack webhook URL
- `TEAMS_WEBHOOK_URL`: Microsoft Teams webhook URL
- `ALERT_WEBHOOK_URL`: Generic webhook URL
- `EMAIL_ALERT_ENDPOINT`: Email service endpoint
- `ALERT_EMAIL_RECIPIENTS`: Comma-separated email addresses
- `SEND_SUCCESS_NOTIFICATIONS`: Send notifications for successful tests (true/false)

#### Trend Analysis
- `ENABLE_TREND_ANALYSIS`: Enable trend analysis (true/false)
- `TREND_WINDOW_DAYS`: Analysis window in days (default: 30)

## Regression Detection Thresholds

### Critical Regressions (Block Deployment)
- Response time increase: >100%
- Error rate increase: >200%
- Cache hit rate decrease: >30%
- Throughput decrease: >50%

### High Regressions (Review Required)
- Response time increase: >50%
- Error rate increase: >100%
- Cache hit rate decrease: >20%
- Throughput decrease: >30%

### Medium Regressions (Monitor)
- Response time increase: >25%
- Error rate increase: >50%
- Cache hit rate decrease: >15%
- Throughput decrease: >20%

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Performance Regression Test

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  performance-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install k6
        run: |
          sudo gpg -k
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      
      - name: Start application
        run: |
          # Start your application here
          docker-compose up -d
          sleep 30
      
      - name: Run performance regression test
        env:
          BASE_URL: http://localhost:8080
          TEST_USERNAME: ${{ secrets.TEST_USERNAME }}
          TEST_PASSWORD: ${{ secrets.TEST_PASSWORD }}
          BASELINE_HOMEPAGE_P95: 500
          BASELINE_API_P95: 100
          BASELINE_ARTICLE_CREATION_P95: 1000
          BASELINE_DB_QUERY_P95: 10
          BASELINE_CACHE_HIT_RATE: 0.8
          BASELINE_ERROR_RATE: 0.02
          ENVIRONMENT: ci
          BUILD_ID: ${{ github.run_id }}
          COMMIT_HASH: ${{ github.sha }}
          BRANCH: ${{ github.ref_name }}
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
        run: |
          k6 run load-testing/integrated-regression-test.js
      
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: performance-test-results
          path: |
            *.json
            *.log
```

### Jenkins Pipeline Example

```groovy
pipeline {
    agent any
    
    environment {
        BASE_URL = 'http://localhost:8080'
        TEST_USERNAME = credentials('test-username')
        TEST_PASSWORD = credentials('test-password')
        BASELINE_HOMEPAGE_P95 = '500'
        BASELINE_API_P95 = '100'
        BASELINE_ARTICLE_CREATION_P95 = '1000'
        BASELINE_DB_QUERY_P95 = '10'
        BASELINE_CACHE_HIT_RATE = '0.8'
        BASELINE_ERROR_RATE = '0.02'
        SLACK_WEBHOOK_URL = credentials('slack-webhook-url')
    }
    
    stages {
        stage('Setup') {
            steps {
                sh 'docker-compose up -d'
                sleep 30
            }
        }
        
        stage('Performance Regression Test') {
            steps {
                script {
                    env.BUILD_ID = env.BUILD_NUMBER
                    env.COMMIT_HASH = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
                    env.BRANCH = env.BRANCH_NAME
                    env.ENVIRONMENT = 'ci'
                }
                
                sh '''
                    k6 run load-testing/integrated-regression-test.js \
                        --out json=performance-results.json
                '''
            }
        }
    }
    
    post {
        always {
            archiveArtifacts artifacts: '*.json', fingerprint: true
            sh 'docker-compose down'
        }
        
        failure {
            emailext (
                subject: "Performance Regression Test Failed - Build ${env.BUILD_NUMBER}",
                body: "Performance regression test failed. Check the build logs for details.",
                to: "devops@company.com"
            )
        }
    }
}
```

## Monitoring and Maintenance

### Regular Baseline Updates

Update baselines regularly to account for legitimate performance changes:

```bash
# Monthly baseline update
k6 run load-testing/establish-baseline.js \
  --env BASE_URL=https://production.example.com \
  --env ENVIRONMENT=production

# Update CI/CD configuration with new baseline values
```

### Performance Trend Monitoring

Set up regular trend analysis to identify gradual performance degradation:

```bash
# Weekly trend analysis
k6 run load-testing/integrated-regression-test.js \
  --env ENABLE_TREND_ANALYSIS=true \
  --env TREND_WINDOW_DAYS=7 \
  --env TEST_DURATION=10m
```

### Alert Tuning

Adjust alert thresholds based on your system's characteristics:

1. Monitor false positive rates
2. Adjust thresholds in `performance-regression-detector.js`
3. Update severity calculations based on business impact
4. Fine-tune alert channels and recipients

## Troubleshooting

### Common Issues

1. **Authentication Failures**
   - Verify `TEST_USERNAME` and `TEST_PASSWORD`
   - Check if test user has required permissions
   - Ensure authentication endpoint is accessible

2. **Baseline Loading Issues**
   - Verify baseline environment variables are set
   - Check baseline timestamp format
   - Ensure baseline values are reasonable

3. **Alert Delivery Failures**
   - Verify webhook URLs are accessible
   - Check authentication credentials for alert channels
   - Test webhook endpoints manually

4. **High False Positive Rates**
   - Review baseline establishment conditions
   - Adjust regression thresholds
   - Consider system warm-up time

### Debug Mode

Enable debug logging for troubleshooting:

```bash
k6 run load-testing/integrated-regression-test.js \
  --env DEBUG=true \
  --env LOG_LEVEL=debug
```

## Best Practices

1. **Baseline Quality**
   - Establish baselines during optimal system performance
   - Use consistent test conditions
   - Update baselines after significant system changes

2. **Test Environment**
   - Use dedicated test environments
   - Ensure consistent resource allocation
   - Minimize external dependencies

3. **Alert Management**
   - Configure appropriate alert channels for different severities
   - Include actionable recommendations in alerts
   - Set up escalation procedures for critical regressions

4. **Continuous Improvement**
   - Regularly review and update thresholds
   - Analyze trend data for optimization opportunities
   - Incorporate feedback from development teams

## Support

For issues or questions about the performance regression detection system:

1. Check the troubleshooting section above
2. Review test logs and output
3. Consult with the DevOps or Performance Engineering team
4. Create an issue in the project repository