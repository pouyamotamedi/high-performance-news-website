package testing

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestSecurityAutomationIntegration(t *testing.T) {
	// Skip if running in CI without proper setup
	if os.Getenv("CI") == "true" && os.Getenv("SECURITY_INTEGRATION_TEST") != "true" {
		t.Skip("Skipping security integration test in CI")
	}

	projectRoot := "../../" // Adjust path as needed
	automation := NewSecurityAutomation(projectRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Run("ComprehensiveSecurityScan", func(t *testing.T) {
		result, err := automation.RunComprehensiveSecurityScan(ctx)
		if err != nil {
			t.Fatalf("Comprehensive security scan failed: %v", err)
		}

		if result == nil {
			t.Fatal("Security scan result is nil")
		}

		if result.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", result.Status)
		}

		if result.Duration == 0 {
			t.Error("Expected non-zero duration")
		}

		// Verify dependency scanning ran
		if result.DependencyResults == nil {
			t.Error("Expected dependency scan results")
		} else {
			t.Logf("Dependency scan found %d vulnerabilities", result.DependencyResults.Summary.TotalVulnerabilities)
		}

		// Verify security scenarios ran
		if len(result.ScenarioResults) == 0 {
			t.Error("Expected security scenario results")
		} else {
			t.Logf("Ran %d security test scenarios", len(result.ScenarioResults))
		}

		// Verify patch management report generated
		if result.PatchManagement == nil {
			t.Error("Expected patch management report")
		} else {
			t.Logf("Generated patch management report with %d recommendations", len(result.PatchManagement.Recommendations))
		}

		t.Logf("Security scan completed in %v", result.Duration)
		t.Logf("Total security issues: %d", result.SecuritySummary.TotalIssues)
		t.Logf("Security score: %.1f", result.SecuritySummary.SecurityScore)
	})
}

func TestZAPAutomation(t *testing.T) {
	// Skip if ZAP is not available
	if os.Getenv("ZAP_API_KEY") == "" {
		t.Skip("Skipping ZAP test - ZAP_API_KEY not set")
	}

	projectRoot := "../../"
	zapAutomation := NewZAPAutomation(projectRoot)

	// Test ZAP availability check
	t.Run("ZAPAvailability", func(t *testing.T) {
		available := zapAutomation.isZAPRunning()
		if !available {
			t.Skip("ZAP is not running, skipping ZAP tests")
		}
		t.Log("ZAP is available and running")
	})
}

func TestDependencyScanner(t *testing.T) {
	projectRoot := "../../"
	scanner := NewDependencyScanner(projectRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("DependencyScan", func(t *testing.T) {
		result, err := scanner.RunComprehensiveDependencyScan(ctx)
		if err != nil {
			t.Fatalf("Dependency scan failed: %v", err)
		}

		if result == nil {
			t.Fatal("Dependency scan result is nil")
		}

		if result.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", result.Status)
		}

		t.Logf("Dependency scan completed in %v", result.Duration)
		t.Logf("Total vulnerabilities: %d", result.Summary.TotalVulnerabilities)
		t.Logf("Scanners used: %d", len(result.ScannerResults))

		// Verify at least one scanner ran
		if len(result.ScannerResults) == 0 {
			t.Error("Expected at least one scanner to run")
		}

		for scannerName, scannerResult := range result.ScannerResults {
			t.Logf("Scanner %s: %s, found %d vulnerabilities", 
				scannerName, scannerResult.Status, scannerResult.VulnerabilitiesFound)
		}
	})
}

func TestSecurityTestScenarios(t *testing.T) {
	scenarios := NewSecurityTestScenarios("http://localhost:8080")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("SecurityScenarios", func(t *testing.T) {
		results := scenarios.RunAllScenarios(ctx)

		if len(results) == 0 {
			t.Fatal("Expected security scenario results")
		}

		passedCount := 0
		failedCount := 0
		errorCount := 0

		for _, result := range results {
			switch result.Status {
			case "passed":
				passedCount++
			case "failed":
				failedCount++
			case "error":
				errorCount++
			}
		}

		t.Logf("Security scenarios: %d total, %d passed, %d failed, %d errors", 
			len(results), passedCount, failedCount, errorCount)

		// Log any failed scenarios for investigation
		for _, result := range results {
			if result.Status == "failed" {
				t.Logf("Failed scenario %s: %s", result.ScenarioID, result.Message)
			}
		}
	})
}

func TestSecurityPatchManager(t *testing.T) {
	projectRoot := "../../"
	patchManager := NewSecurityPatchManager(projectRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create sample vulnerabilities for testing
	sampleVulns := []Vulnerability{
		{
			ID:           "CVE-2023-1234",
			Title:        "Test vulnerability",
			Severity:     "high",
			CVSS:         7.5,
			Package:      "github.com/example/vulnerable",
			Version:      "1.0.0",
			FixedVersion: "1.2.3",
			Upgradable:   true,
			Scanner:      "test",
		},
		{
			ID:           "CVE-2023-5678",
			Title:        "Critical test vulnerability",
			Severity:     "critical",
			CVSS:         9.8,
			Package:      "github.com/example/critical",
			Version:      "2.0.0",
			FixedVersion: "2.1.0",
			Upgradable:   true,
			Scanner:      "test",
		},
	}

	t.Run("PatchManagementReport", func(t *testing.T) {
		report, err := patchManager.GeneratePatchManagementReport(ctx, sampleVulns)
		if err != nil {
			t.Fatalf("Patch management report failed: %v", err)
		}

		if report == nil {
			t.Fatal("Patch management report is nil")
		}

		if len(report.Recommendations) != len(sampleVulns) {
			t.Errorf("Expected %d recommendations, got %d", len(sampleVulns), len(report.Recommendations))
		}

		if len(report.ActionPlan) == 0 {
			t.Error("Expected action plan to be generated")
		}

		t.Logf("Generated patch report with %d recommendations", len(report.Recommendations))
		t.Logf("Risk score: %.1f", report.Summary.RiskScore)
		t.Logf("Compliance score: %.1f%%", report.ComplianceStatus.ComplianceScore)

		// Verify critical vulnerability gets immediate priority
		for _, rec := range report.Recommendations {
			if rec.Severity == "critical" && rec.Priority != PatchPriorityImmediate {
				t.Errorf("Critical vulnerability should have immediate priority, got %s", rec.Priority)
			}
		}
	})
}