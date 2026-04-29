package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GovulncheckScanner implements Go vulnerability scanning using govulncheck
type GovulncheckScanner struct {
	projectRoot string
}

// Name returns the scanner name
func (g *GovulncheckScanner) Name() string {
	return "govulncheck"
}

// IsAvailable checks if govulncheck is available
func (g *GovulncheckScanner) IsAvailable() bool {
	// Check if govulncheck command exists
	if _, err := exec.LookPath("govulncheck"); err != nil {
		// Try to install govulncheck if not found
		if err := g.installGovulncheck(); err != nil {
			return false
		}
	}
	return true
}

// installGovulncheck installs govulncheck if not available
func (g *GovulncheckScanner) installGovulncheck() error {
	cmd := exec.Command("go", "install", "golang.org/x/vuln/cmd/govulncheck@latest")
	return cmd.Run()
}

// Scan performs govulncheck vulnerability scanning
func (g *GovulncheckScanner) Scan(ctx context.Context, projectPath string) (*DependencyScanResult, error) {
	result := &DependencyScanResult{
		ScannerName: "govulncheck",
		Status:      "running",
		StartTime:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Run govulncheck with JSON output
	cmd := exec.CommandContext(ctx, "govulncheck", "-json", "./...")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		// govulncheck returns non-zero exit code when vulnerabilities are found
		// This is expected behavior, so we continue processing if we have output
		if len(output) == 0 {
			result.Status = "failed"
			result.ErrorMessage = fmt.Sprintf("govulncheck scan failed: %v", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, err
		}
	}

	// Parse govulncheck results
	vulnerabilities, packagesScanned, err := g.parseGovulncheckOutput(output)
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

// parseGovulncheckOutput parses govulncheck JSON output
func (g *GovulncheckScanner) parseGovulncheckOutput(output []byte) ([]Vulnerability, int, error) {
	lines := strings.Split(string(output), "\n")
	var vulnerabilities []Vulnerability
	packageSet := make(map[string]bool)

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry struct {
			Type string `json:"type"`
			Data struct {
				OSV struct {
					ID       string `json:"id"`
					Summary  string `json:"summary"`
					Details  string `json:"details"`
					Severity []struct {
						Type  string  `json:"type"`
						Score string  `json:"score"`
					} `json:"severity"`
					References []struct {
						Type string `json:"type"`
						URL  string `json:"url"`
					} `json:"references"`
					DatabaseSpecific struct {
						URL string `json:"url"`
					} `json:"database_specific"`
					Affected []struct {
						Package struct {
							Name      string `json:"name"`
							Ecosystem string `json:"ecosystem"`
						} `json:"package"`
						Ranges []struct {
							Type   string `json:"type"`
							Events []struct {
								Introduced string `json:"introduced"`
								Fixed      string `json:"fixed"`
							} `json:"events"`
						} `json:"ranges"`
						Versions         []string `json:"versions"`
						EcosystemSpecific struct {
							Imports []struct {
								Path    string   `json:"path"`
								GOOS    []string `json:"goos"`
								GOARCH  []string `json:"goarch"`
								Symbols []string `json:"symbols"`
							} `json:"imports"`
						} `json:"ecosystem_specific"`
					} `json:"affected"`
					SchemaVersion string    `json:"schema_version"`
					Modified      time.Time `json:"modified"`
					Published     time.Time `json:"published"`
				} `json:"osv"`
				ModulePath string `json:"module_path"`
				Version    string `json:"version"`
			} `json:"data"`
		}

		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}

		if entry.Type == "vulnerability" {
			osv := entry.Data.OSV
			
			// Determine severity
			severity := "medium" // default
			cvssScore := 0.0
			for _, sev := range osv.Severity {
				if sev.Type == "CVSS_V3" {
					// Parse CVSS score from string like "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
					if strings.Contains(sev.Score, "CVSS:") {
						// Simplified CVSS parsing - in production, use proper CVSS library
						if strings.Contains(sev.Score, "C:H") || strings.Contains(sev.Score, "I:H") || strings.Contains(sev.Score, "A:H") {
							severity = "high"
							cvssScore = 7.5
						} else if strings.Contains(sev.Score, "C:M") || strings.Contains(sev.Score, "I:M") || strings.Contains(sev.Score, "A:M") {
							severity = "medium"
							cvssScore = 5.0
						} else {
							severity = "low"
							cvssScore = 2.5
						}
					}
				}
			}

			var references []string
			for _, ref := range osv.References {
				references = append(references, ref.URL)
			}

			// Process affected packages
			for _, affected := range osv.Affected {
				packageName := affected.Package.Name
				packageSet[packageName] = true

				var fixedVersion string
				for _, r := range affected.Ranges {
					for _, event := range r.Events {
						if event.Fixed != "" {
							fixedVersion = event.Fixed
							break
						}
					}
					if fixedVersion != "" {
						break
					}
				}

				vulnerability := Vulnerability{
					ID:           osv.ID,
					Title:        osv.Summary,
					Description:  osv.Details,
					Severity:     severity,
					CVSS:         cvssScore,
					Package:      packageName,
					Version:      entry.Data.Version,
					FixedVersion: fixedVersion,
					References:   references,
					PublishedDate: osv.Published,
					ModifiedDate:  osv.Modified,
					Scanner:      "govulncheck",
					Upgradable:   fixedVersion != "",
					Patchable:    fixedVersion != "",
					DependencyPath: []string{entry.Data.ModulePath},
					Metadata: map[string]string{
						"module_path": entry.Data.ModulePath,
						"ecosystem":   affected.Package.Ecosystem,
					},
				}

				vulnerabilities = append(vulnerabilities, vulnerability)
			}
		}
	}

	return vulnerabilities, len(packageSet), nil
}