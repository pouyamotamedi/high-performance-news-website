package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SnykScanner implements Snyk vulnerability scanning
type SnykScanner struct {
	token       string
	projectRoot string
}

// Name returns the scanner name
func (s *SnykScanner) Name() string {
	return "snyk"
}

// IsAvailable checks if Snyk CLI is available and configured
func (s *SnykScanner) IsAvailable() bool {
	// Check if snyk command exists
	if _, err := exec.LookPath("snyk"); err != nil {
		return false
	}
	
	// Check if token is available
	return s.token != ""
}

// Scan performs Snyk vulnerability scanning
func (s *SnykScanner) Scan(ctx context.Context, projectPath string) (*DependencyScanResult, error) {
	result := &DependencyScanResult{
		ScannerName: "snyk",
		Status:      "running",
		StartTime:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Authenticate with Snyk
	if err := s.authenticate(); err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("authentication failed: %v", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	// Run Snyk test
	cmd := exec.CommandContext(ctx, "snyk", "test", "--json", "--all-projects")
	cmd.Dir = projectPath
	cmd.Env = append(os.Environ(), "SNYK_TOKEN="+s.token)

	output, err := cmd.Output()
	if err != nil {
		// Snyk returns non-zero exit code when vulnerabilities are found
		// This is expected behavior, so we continue processing
		if len(output) == 0 {
			result.Status = "failed"
			result.ErrorMessage = fmt.Sprintf("snyk scan failed: %v", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, err
		}
	}

	// Parse Snyk results
	vulnerabilities, packagesScanned, err := s.parseSnykOutput(output)
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

// authenticate authenticates with Snyk
func (s *SnykScanner) authenticate() error {
	cmd := exec.Command("snyk", "auth", s.token)
	cmd.Env = append(os.Environ(), "SNYK_TOKEN="+s.token)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("authentication failed: %v, output: %s", err, string(output))
	}
	
	return nil
}

// parseSnykOutput parses Snyk JSON output
func (s *SnykScanner) parseSnykOutput(output []byte) ([]Vulnerability, int, error) {
	var snykResult struct {
		Vulnerabilities []struct {
			ID          string  `json:"id"`
			Title       string  `json:"title"`
			Description string  `json:"description"`
			Severity    string  `json:"severity"`
			CVSSv3      string  `json:"CVSSv3"`
			CVSS        float64 `json:"cvssScore"`
			CWE         []string `json:"cwe"`
			References  []struct {
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"references"`
			PublicationTime    time.Time `json:"publicationTime"`
			ModificationTime   time.Time `json:"modificationTime"`
			PackageName        string    `json:"packageName"`
			Version            string    `json:"version"`
			NearestFixedInVersion string `json:"nearestFixedInVersion"`
			From               []string  `json:"from"`
			UpgradePath        []interface{} `json:"upgradePath"`
			IsUpgradable       bool      `json:"isUpgradable"`
			IsPatchable        bool      `json:"isPatchable"`
		} `json:"vulnerabilities"`
		DependencyCount int `json:"dependencyCount"`
		PackageManager  string `json:"packageManager"`
	}

	if err := json.Unmarshal(output, &snykResult); err != nil {
		return nil, 0, fmt.Errorf("failed to parse Snyk JSON: %w", err)
	}

	var vulnerabilities []Vulnerability
	for _, vuln := range snykResult.Vulnerabilities {
		var references []string
		for _, ref := range vuln.References {
			references = append(references, ref.URL)
		}

		vulnerability := Vulnerability{
			ID:             vuln.ID,
			Title:          vuln.Title,
			Description:    vuln.Description,
			Severity:       strings.ToLower(vuln.Severity),
			CVSS:           vuln.CVSS,
			CWE:            vuln.CWE,
			Package:        vuln.PackageName,
			Version:        vuln.Version,
			FixedVersion:   vuln.NearestFixedInVersion,
			References:     references,
			PublishedDate:  vuln.PublicationTime,
			ModifiedDate:   vuln.ModificationTime,
			Scanner:        "snyk",
			Upgradable:     vuln.IsUpgradable,
			Patchable:      vuln.IsPatchable,
			DependencyPath: vuln.From,
			Metadata: map[string]string{
				"package_manager": snykResult.PackageManager,
				"cvss_v3":        vuln.CVSSv3,
			},
		}

		vulnerabilities = append(vulnerabilities, vulnerability)
	}

	return vulnerabilities, snykResult.DependencyCount, nil
}