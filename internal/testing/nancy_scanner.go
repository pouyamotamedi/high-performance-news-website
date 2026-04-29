package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// NancyScanner implements Nancy vulnerability scanning (OSS Index)
type NancyScanner struct {
	projectRoot string
}

// Name returns the scanner name
func (n *NancyScanner) Name() string {
	return "nancy"
}

// IsAvailable checks if Nancy is available
func (n *NancyScanner) IsAvailable() bool {
	// Check if nancy command exists
	if _, err := exec.LookPath("nancy"); err != nil {
		// Try to install nancy if not found
		if err := n.installNancy(); err != nil {
			return false
		}
	}
	return true
}

// installNancy installs Nancy if not available
func (n *NancyScanner) installNancy() error {
	cmd := exec.Command("go", "install", "github.com/sonatypecommunity/nancy@latest")
	return cmd.Run()
}

// Scan performs Nancy vulnerability scanning
func (n *NancyScanner) Scan(ctx context.Context, projectPath string) (*DependencyScanResult, error) {
	result := &DependencyScanResult{
		ScannerName: "nancy",
		Status:      "running",
		StartTime:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// First, generate go.list file for Nancy
	listCmd := exec.CommandContext(ctx, "go", "list", "-json", "-deps", "./...")
	listCmd.Dir = projectPath
	
	listOutput, err := listCmd.Output()
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("failed to generate dependency list: %v", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	// Run Nancy scan
	nancyCmd := exec.CommandContext(ctx, "nancy", "sleuth", "--output", "json")
	nancyCmd.Dir = projectPath
	nancyCmd.Stdin = strings.NewReader(string(listOutput))

	output, err := nancyCmd.Output()
	if err != nil {
		// Nancy returns non-zero exit code when vulnerabilities are found
		// This is expected behavior, so we continue processing if we have output
		if len(output) == 0 {
			result.Status = "failed"
			result.ErrorMessage = fmt.Sprintf("nancy scan failed: %v", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, err
		}
	}

	// Parse Nancy results
	vulnerabilities, packagesScanned, err := n.parseNancyOutput(output)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("failed to parse results: %v", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	result.Vulnerabilities = vulnerabilities
	result.PackagesScanned = packagesScanned
	result.VulnerabilitiesFound = len(vulnerabilities)
	result.Status = "completed"
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// parseNancyOutput parses Nancy JSON output
func (n *NancyScanner) parseNancyOutput(output []byte) ([]Vulnerability, int, error) {
	var nancyResult struct {
		Audited []struct {
			Coordinates string `json:"coordinates"`
			Reference   string `json:"reference"`
			Vulnerabilities []struct {
				ID          string  `json:"id"`
				Title       string  `json:"title"`
				Description string  `json:"description"`
				CVSS        float64 `json:"cvssScore"`
				CVE         string  `json:"cve"`
				Reference   string  `json:"reference"`
				Excluded    bool    `json:"excluded"`
			} `json:"vulnerabilities"`
			InvalidSemVer bool `json:"invalidSemVer"`
		} `json:"audited"`
		Exclusions []struct {
			Coordinates string `json:"coordinates"`
			Justification string `json:"justification"`
		} `json:"exclusions"`
		Invalid []struct {
			Coordinates string `json:"coordinates"`
			Reference   string `json:"reference"`
		} `json:"invalid"`
		Num_Audited     int `json:"num_audited"`
		Num_Vulnerable  int `json:"num_vulnerable"`
		Version         string `json:"version"`
	}

	if err := json.Unmarshal(output, &nancyResult); err != nil {
		return nil, 0, fmt.Errorf("failed to parse Nancy JSON: %w", err)
	}

	var vulnerabilities []Vulnerability
	packageSet := make(map[string]bool)

	for _, audited := range nancyResult.Audited {
		packageSet[audited.Coordinates] = true
		
		for _, vuln := range audited.Vulnerabilities {
			if vuln.Excluded {
				continue // Skip excluded vulnerabilities
			}

			// Parse package name and version from coordinates
			// Format is typically: pkg:golang/github.com/package/name@version
			parts := strings.Split(audited.Coordinates, "@")
			packageName := audited.Coordinates
			version := ""
			if len(parts) == 2 {
				packageName = parts[0]
				version = parts[1]
				// Remove pkg:golang/ prefix if present
				if strings.HasPrefix(packageName, "pkg:golang/") {
					packageName = strings.TrimPrefix(packageName, "pkg:golang/")
				}
			}

			// Determine severity based on CVSS score
			severity := "low"
			if vuln.CVSS >= 9.0 {
				severity = "critical"
			} else if vuln.CVSS >= 7.0 {
				severity = "high"
			} else if vuln.CVSS >= 4.0 {
				severity = "medium"
			}

			var references []string
			if vuln.Reference != "" {
				references = append(references, vuln.Reference)
			}
			if audited.Reference != "" {
				references = append(references, audited.Reference)
			}

			vulnerability := Vulnerability{
				ID:          vuln.ID,
				Title:       vuln.Title,
				Description: vuln.Description,
				Severity:    severity,
				CVSS:        vuln.CVSS,
				Package:     packageName,
				Version:     version,
				References:  references,
				Scanner:     "nancy",
				Upgradable:  false, // Nancy doesn't provide fix information
				Patchable:   false,
				DependencyPath: []string{packageName},
				Metadata: map[string]string{
					"coordinates":     audited.Coordinates,
					"cve":            vuln.CVE,
					"invalid_semver": fmt.Sprintf("%t", audited.InvalidSemVer),
				},
			}

			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	return vulnerabilities, len(packageSet), nil
}