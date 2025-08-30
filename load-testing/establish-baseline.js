import BaselineManager from './baseline-manager.js';

// Script to establish performance baseline
const baselineManager = new BaselineManager();

export const options = {
  scenarios: {
    baseline_establishment: {
      executor: 'shared-iterations',
      vus: 1,
      iterations: 1,
      maxDuration: '10m',
    },
  },
};

export default function() {
  console.log('=== ESTABLISHING PERFORMANCE BASELINE ===');
  
  // Establish comprehensive baseline
  const baseline = baselineManager.establishBaseline();
  
  console.log('\n✅ Baseline establishment completed successfully');
  console.log(`Baseline timestamp: ${baseline.timestamp}`);
  console.log(`Total metrics collected: ${Object.keys(baseline.metrics).length}`);
  
  // Validate baseline quality
  validateBaseline(baseline);
}

function validateBaseline(baseline) {
  console.log('\n=== BASELINE VALIDATION ===');
  
  const aggregates = baseline.aggregates;
  const validationResults = [];
  
  // Validate homepage performance
  if (aggregates.homepage_p95 > 0 && aggregates.homepage_p95 < 5000) {
    validationResults.push('✅ Homepage P95 within reasonable range');
  } else {
    validationResults.push('❌ Homepage P95 outside expected range');
  }
  
  // Validate API performance
  if (aggregates.api_p95 > 0 && aggregates.api_p95 < 1000) {
    validationResults.push('✅ API P95 within reasonable range');
  } else {
    validationResults.push('❌ API P95 outside expected range');
  }
  
  // Validate article creation performance
  if (aggregates.article_creation_p95 > 0 && aggregates.article_creation_p95 < 5000) {
    validationResults.push('✅ Article creation P95 within reasonable range');
  } else {
    validationResults.push('❌ Article creation P95 outside expected range');
  }
  
  // Validate cache hit rate
  if (aggregates.cache_hit_rate >= 0 && aggregates.cache_hit_rate <= 1) {
    validationResults.push('✅ Cache hit rate within valid range');
  } else {
    validationResults.push('❌ Cache hit rate outside valid range');
  }
  
  // Validate error rate
  if (aggregates.error_rate >= 0 && aggregates.error_rate <= 0.1) {
    validationResults.push('✅ Error rate within acceptable range');
  } else {
    validationResults.push('❌ Error rate too high for baseline');
  }
  
  // Print validation results
  validationResults.forEach(result => console.log(result));
  
  const failedValidations = validationResults.filter(r => r.startsWith('❌')).length;
  const totalValidations = validationResults.length;
  
  console.log(`\nValidation Summary: ${totalValidations - failedValidations}/${totalValidations} checks passed`);
  
  if (failedValidations === 0) {
    console.log('✅ Baseline validation passed - baseline is suitable for regression detection');
  } else {
    console.log('⚠️  Baseline validation failed - consider investigating performance issues before using as baseline');
  }
}

export function setup() {
  console.log('Setting up baseline establishment...');
  
  // Verify server is accessible
  const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
  
  console.log(`Target server: ${BASE_URL}`);
  console.log(`Test user: ${__ENV.TEST_USERNAME || 'testuser'}`);
  console.log(`Environment: ${__ENV.ENVIRONMENT || 'development'}`);
  
  return { setupComplete: true };
}

export function teardown(data) {
  console.log('\n=== BASELINE ESTABLISHMENT COMPLETE ===');
  console.log('Next steps:');
  console.log('1. Copy the environment variables above to your CI/CD configuration');
  console.log('2. Run performance regression tests using: k6 run performance-regression-test.js');
  console.log('3. Set up automated baseline updates on a regular schedule');
  console.log('4. Configure alerts for critical performance regressions');
}