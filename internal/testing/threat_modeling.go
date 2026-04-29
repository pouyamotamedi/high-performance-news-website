package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ThreatModelingEngine performs comprehensive threat modeling and attack simulation
type ThreatModelingEngine struct {
	projectRoot      string
	reportDir        string
	attackSimulator  *AttackSimulator
	complianceChecker *ComplianceChecker
	siemIntegration  *SIEMIntegration
	config           *ThreatModelingConfig
}

// ThreatModelingConfig holds configuration for threat modeling
type ThreatModelingConfig struct {
	EnableAttackSimulation bool              `yaml:"enable_attack_simulation"`
	EnableComplianceCheck  bool              `yaml:"enable_compliance_check"`
	EnableSIEMIntegration  bool              `yaml:"enable_siem_integration"`
	ThreatCategories       []string          `yaml:"threat_categories"`
	AttackVectors          []string          `yaml:"attack_vectors"`
	ComplianceFrameworks   []string          `yaml:"compliance_frameworks"`
	SimulationTimeout      time.Duration     `yaml:"simulation_timeout"`
	ReportFormats          []string          `yaml:"report_formats"`
	CustomThreats          map[string]string `yaml:"custom_threats"`
}

// ThreatModel represents a comprehensive threat model
type ThreatModel struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
	Assets           []Asset               `json:"assets"`
	Threats          []Threat              `json:"threats"`
	Vulnerabilities  []Vulnerability       `json:"vulnerabilities"`
	AttackPaths      []AttackPath          `json:"attack_paths"`
	Mitigations      []Mitigation          `json:"mitigations"`
	RiskAssessment   RiskAssessment        `json:"risk_assessment"`
	ComplianceStatus ComplianceAssessment  `json:"compliance_status"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// Asset represents a system asset that needs protection
type Asset struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Description    string            `json:"description"`
	Criticality    string            `json:"criticality"`
	Owner          string            `json:"owner"`
	Location       string            `json:"location"`
	Dependencies   []string          `json:"dependencies"`
	SecurityLevel  string            `json:"security_level"`
	DataTypes      []string          `json:"data_types"`
	AccessControls []AccessControl   `json:"access_controls"`
	Metadata       map[string]string `json:"metadata"`
}

// Threat represents a potential security threat
type Threat struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Category        string            `json:"category"`
	Description     string            `json:"description"`
	ThreatActor     string            `json:"threat_actor"`
	Motivation      string            `json:"motivation"`
	Capability      string            `json:"capability"`
	Likelihood      string            `json:"likelihood"`
	Impact          string            `json:"impact"`
	RiskScore       float64           `json:"risk_score"`
	STRIDECategory  string            `json:"stride_category"`
	MITREAttackID   string            `json:"mitre_attack_id"`
	TargetAssets    []string          `json:"target_assets"`
	AttackVectors   []string          `json:"attack_vectors"`
	Indicators      []string          `json:"indicators"`
	Metadata        map[string]string `json:"metadata"`
}

// AttackPath represents a sequence of attack steps
type AttackPath struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	StartAsset   string       `json:"start_asset"`
	TargetAsset  string       `json:"target_asset"`
	Steps        []AttackStep `json:"steps"`
	Probability  float64      `json:"probability"`
	Impact       string       `json:"impact"`
	Complexity   string       `json:"complexity"`
	DetectionDifficulty string `json:"detection_difficulty"`
}

// AttackStep represents a single step in an attack path
type AttackStep struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Technique   string            `json:"technique"`
	Tool        string            `json:"tool"`
	Asset       string            `json:"asset"`
	Success     float64           `json:"success_probability"`
	Detection   float64           `json:"detection_probability"`
	Impact      string            `json:"impact"`
	Metadata    map[string]string `json:"metadata"`
}

// Mitigation represents a security control or countermeasure
type Mitigation struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Description    string            `json:"description"`
	Effectiveness  string            `json:"effectiveness"`
	Cost           string            `json:"cost"`
	Complexity     string            `json:"complexity"`
	Status         string            `json:"status"`
	Owner          string            `json:"owner"`
	TargetThreats  []string          `json:"target_threats"`
	TargetAssets   []string          `json:"target_assets"`
	Implementation string            `json:"implementation"`
	Validation     string            `json:"validation"`
	Metadata       map[string]string `json:"metadata"`
}

// AccessControl represents access control mechanisms
type AccessControl struct {
	Type        string   `json:"type"`
	Principals  []string `json:"principals"`
	Permissions []string `json:"permissions"`
	Conditions  []string `json:"conditions"`
}

// RiskAssessment provides overall risk assessment
type RiskAssessment struct {
	OverallRisk      string                    `json:"overall_risk"`
	RiskScore        float64                   `json:"risk_score"`
	CriticalThreats  int                       `json:"critical_threats"`
	HighThreats      int                       `json:"high_threats"`
	MediumThreats    int                       `json:"medium_threats"`
	LowThreats       int                       `json:"low_threats"`
	UnmitigatedRisks []UnmitigatedRisk        `json:"unmitigated_risks"`
	RiskTrends       []RiskTrend              `json:"risk_trends"`
	Recommendations  []string                  `json:"recommendations"`
	NextReview       time.Time                 `json:"next_review"`
}

// UnmitigatedRisk represents risks without adequate controls
type UnmitigatedRisk struct {
	ThreatID    string  `json:"threat_id"`
	AssetID     string  `json:"asset_id"`
	RiskScore   float64 `json:"risk_score"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"`
}

// RiskTrend tracks risk changes over time
type RiskTrend struct {
	Date      time.Time `json:"date"`
	RiskScore float64   `json:"risk_score"`
	Category  string    `json:"category"`
	Change    string    `json:"change"`
}

// ComplianceAssessment tracks regulatory compliance
type ComplianceAssessment struct {
	Frameworks       []ComplianceFramework `json:"frameworks"`
	OverallScore     float64               `json:"overall_score"`
	ComplianceGaps   []ComplianceGap       `json:"compliance_gaps"`
	CertificationStatus string             `json:"certification_status"`
	LastAudit        time.Time             `json:"last_audit"`
	NextAudit        time.Time             `json:"next_audit"`
	AuditFindings    []AuditFinding        `json:"audit_findings"`
}

// ComplianceFramework represents a regulatory framework
type ComplianceFramework struct {
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Score        float64             `json:"score"`
	Status       string              `json:"status"`
	Controls     []ComplianceControl `json:"controls"`
	LastAssessed time.Time           `json:"last_assessed"`
}

// ComplianceControl represents a specific compliance control
type ComplianceControl struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	Evidence     []string  `json:"evidence"`
	LastTested   time.Time `json:"last_tested"`
	NextTest     time.Time `json:"next_test"`
	Owner        string    `json:"owner"`
	Deficiencies []string  `json:"deficiencies"`
}

// ComplianceGap represents a compliance deficiency
type ComplianceGap struct {
	Framework   string    `json:"framework"`
	ControlID   string    `json:"control_id"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	DueDate     time.Time `json:"due_date"`
	Owner       string    `json:"owner"`
	Status      string    `json:"status"`
}

// AuditFinding represents an audit finding
type AuditFinding struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Evidence    string    `json:"evidence"`
	Status      string    `json:"status"`
	DueDate     time.Time `json:"due_date"`
	Owner       string    `json:"owner"`
}

// ThreatModelingResult represents the result of threat modeling analysis
type ThreatModelingResult struct {
	ThreatModel        ThreatModel           `json:"threat_model"`
	AttackSimulations  []AttackSimulation    `json:"attack_simulations"`
	ComplianceResults  []ComplianceResult    `json:"compliance_results"`
	SIEMIntegration    SIEMIntegrationResult `json:"siem_integration"`
	ExecutionTime      time.Duration         `json:"execution_time"`
	ReportPaths        []string              `json:"report_paths"`
	Recommendations    []string              `json:"recommendations"`
	Status             string                `json:"status"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// NewThreatModelingEngine creates a new threat modeling engine
func NewThreatModelingEngine(projectRoot string) *ThreatModelingEngine {
	reportDir := filepath.Join(projectRoot, "reports", "security", "threat-modeling")
	os.MkdirAll(reportDir, 0755)

	config := &ThreatModelingConfig{
		EnableAttackSimulation: true,
		EnableComplianceCheck:  true,
		EnableSIEMIntegration:  true,
		ThreatCategories: []string{
			"spoofing", "tampering", "repudiation", "information_disclosure",
			"denial_of_service", "elevation_of_privilege",
		},
		AttackVectors: []string{
			"web_application", "database", "network", "social_engineering",
			"physical", "insider_threat", "supply_chain",
		},
		ComplianceFrameworks: []string{"GDPR", "CCPA", "SOX", "PCI-DSS", "ISO27001"},
		SimulationTimeout:    30 * time.Minute,
		ReportFormats:        []string{"json", "html", "pdf"},
		CustomThreats:        make(map[string]string),
	}

	return &ThreatModelingEngine{
		projectRoot:       projectRoot,
		reportDir:         reportDir,
		attackSimulator:   NewAttackSimulator(projectRoot),
		complianceChecker: NewComplianceChecker(projectRoot),
		siemIntegration:   NewSIEMIntegration(projectRoot),
		config:            config,
	}
}

// RunComprehensiveThreatModeling performs complete threat modeling analysis
func (t *ThreatModelingEngine) RunComprehensiveThreatModeling(ctx context.Context) (*ThreatModelingResult, error) {
	start := time.Now()
	
	log.Println("Starting comprehensive threat modeling analysis...")

	result := &ThreatModelingResult{
		Status:   "running",
		Metadata: make(map[string]interface{}),
	}

	// Create timeout context
	modelingCtx, cancel := context.WithTimeout(ctx, t.config.SimulationTimeout)
	defer cancel()

	// Step 1: Build threat model
	log.Println("Building threat model...")
	threatModel, err := t.buildThreatModel(modelingCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build threat model: %w", err)
	}
	result.ThreatModel = *threatModel

	// Step 2: Run attack simulations
	if t.config.EnableAttackSimulation {
		log.Println("Running attack simulations...")
		simulations, err := t.attackSimulator.RunAttackSimulations(modelingCtx, threatModel)
		if err != nil {
			log.Printf("Attack simulation failed: %v", err)
			result.Metadata["attack_simulation_error"] = err.Error()
		} else {
			result.AttackSimulations = simulations
		}
	}

	// Step 3: Check compliance
	if t.config.EnableComplianceCheck {
		log.Println("Checking compliance...")
		complianceResults, err := t.complianceChecker.CheckCompliance(modelingCtx, threatModel)
		if err != nil {
			log.Printf("Compliance check failed: %v", err)
			result.Metadata["compliance_check_error"] = err.Error()
		} else {
			result.ComplianceResults = complianceResults
		}
	}

	// Step 4: SIEM integration
	if t.config.EnableSIEMIntegration {
		log.Println("Integrating with SIEM...")
		siemResult, err := t.siemIntegration.IntegrateWithSIEM(modelingCtx, threatModel)
		if err != nil {
			log.Printf("SIEM integration failed: %v", err)
			result.Metadata["siem_integration_error"] = err.Error()
		} else {
			result.SIEMIntegration = *siemResult
		}
	}

	// Generate recommendations
	result.Recommendations = t.generateThreatModelingRecommendations(result)

	// Generate reports
	reportPaths, err := t.generateThreatModelingReports(result)
	if err != nil {
		log.Printf("Failed to generate some reports: %v", err)
	}
	result.ReportPaths = reportPaths

	result.ExecutionTime = time.Since(start)
	result.Status = "completed"

	log.Printf("Threat modeling analysis completed in %v", result.ExecutionTime)
	return result, nil
}

// buildThreatModel creates a comprehensive threat model for the system
func (t *ThreatModelingEngine) buildThreatModel(ctx context.Context) (*ThreatModel, error) {
	model := &ThreatModel{
		ID:          fmt.Sprintf("threat-model-%d", time.Now().Unix()),
		Name:        "News Website Threat Model",
		Description: "Comprehensive threat model for high-performance news website",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Identify system assets
	model.Assets = t.identifySystemAssets()

	// Identify threats using STRIDE methodology
	model.Threats = t.identifyThreats(model.Assets)

	// Identify vulnerabilities
	model.Vulnerabilities = t.identifyVulnerabilities(model.Assets)

	// Map attack paths
	model.AttackPaths = t.mapAttackPaths(model.Assets, model.Threats)

	// Identify mitigations
	model.Mitigations = t.identifyMitigations(model.Threats)

	// Perform risk assessment
	model.RiskAssessment = t.performRiskAssessment(model.Threats, model.Mitigations)

	// Assess compliance
	model.ComplianceStatus = t.assessCompliance(model.Assets, model.Threats, model.Mitigations)

	return model, nil
}

// identifySystemAssets identifies critical system assets
func (t *ThreatModelingEngine) identifySystemAssets() []Asset {
	return []Asset{
		{
			ID:          "web-application",
			Name:        "Web Application",
			Type:        "application",
			Description: "Main news website application",
			Criticality: "high",
			Owner:       "development-team",
			Location:    "cloud",
			SecurityLevel: "confidential",
			DataTypes:   []string{"user_data", "content", "analytics"},
			AccessControls: []AccessControl{
				{Type: "authentication", Principals: []string{"users", "admins"}, Permissions: []string{"read", "write"}},
				{Type: "authorization", Principals: []string{"editors"}, Permissions: []string{"publish", "moderate"}},
			},
		},
		{
			ID:          "database",
			Name:        "PostgreSQL Database",
			Type:        "database",
			Description: "Primary database storing articles and user data",
			Criticality: "critical",
			Owner:       "database-team",
			Location:    "cloud",
			SecurityLevel: "restricted",
			DataTypes:   []string{"pii", "content", "metadata"},
			AccessControls: []AccessControl{
				{Type: "network", Principals: []string{"application"}, Permissions: []string{"connect"}},
				{Type: "database", Principals: []string{"app_user"}, Permissions: []string{"select", "insert", "update"}},
			},
		},
		{
			ID:          "cache-layer",
			Name:        "DragonflyDB Cache",
			Type:        "cache",
			Description: "High-performance caching layer",
			Criticality: "medium",
			Owner:       "infrastructure-team",
			Location:    "cloud",
			SecurityLevel: "internal",
			DataTypes:   []string{"cached_content", "session_data"},
		},
		{
			ID:          "cdn",
			Name:        "Content Delivery Network",
			Type:        "infrastructure",
			Description: "CDN for static content delivery",
			Criticality: "medium",
			Owner:       "infrastructure-team",
			Location:    "global",
			SecurityLevel: "public",
			DataTypes:   []string{"static_content", "images"},
		},
		{
			ID:          "admin-panel",
			Name:        "Administrative Panel",
			Type:        "application",
			Description: "Administrative interface for content management",
			Criticality: "high",
			Owner:       "development-team",
			Location:    "cloud",
			SecurityLevel: "restricted",
			DataTypes:   []string{"admin_data", "system_config"},
			AccessControls: []AccessControl{
				{Type: "mfa", Principals: []string{"admins"}, Permissions: []string{"full_access"}},
			},
		},
	}
}

// identifyThreats identifies potential threats using STRIDE methodology
func (t *ThreatModelingEngine) identifyThreats(assets []Asset) []Threat {
	threats := []Threat{
		// Spoofing threats
		{
			ID:             "T001",
			Name:           "User Identity Spoofing",
			Category:       "spoofing",
			Description:    "Attacker impersonates legitimate user",
			ThreatActor:    "external_attacker",
			Motivation:     "unauthorized_access",
			Capability:     "medium",
			Likelihood:     "medium",
			Impact:         "high",
			RiskScore:      7.5,
			STRIDECategory: "spoofing",
			MITREAttackID:  "T1078",
			TargetAssets:   []string{"web-application", "admin-panel"},
			AttackVectors:  []string{"credential_stuffing", "phishing", "session_hijacking"},
			Indicators:     []string{"unusual_login_patterns", "multiple_failed_logins"},
		},
		// Tampering threats
		{
			ID:             "T002",
			Name:           "Content Tampering",
			Category:       "tampering",
			Description:    "Unauthorized modification of news content",
			ThreatActor:    "insider_threat",
			Motivation:     "misinformation",
			Capability:     "high",
			Likelihood:     "low",
			Impact:         "critical",
			RiskScore:      8.0,
			STRIDECategory: "tampering",
			MITREAttackID:  "T1565",
			TargetAssets:   []string{"database", "web-application"},
			AttackVectors:  []string{"sql_injection", "privilege_escalation", "insider_access"},
			Indicators:     []string{"unauthorized_content_changes", "audit_log_anomalies"},
		},
		// Information Disclosure threats
		{
			ID:             "T003",
			Name:           "Personal Data Exposure",
			Category:       "information_disclosure",
			Description:    "Unauthorized access to user personal data",
			ThreatActor:    "external_attacker",
			Motivation:     "data_theft",
			Capability:     "medium",
			Likelihood:     "medium",
			Impact:         "critical",
			RiskScore:      8.5,
			STRIDECategory: "information_disclosure",
			MITREAttackID:  "T1005",
			TargetAssets:   []string{"database", "cache-layer"},
			AttackVectors:  []string{"sql_injection", "cache_poisoning", "api_abuse"},
			Indicators:     []string{"unusual_data_access", "large_data_exports"},
		},
		// Denial of Service threats
		{
			ID:             "T004",
			Name:           "Application DDoS",
			Category:       "denial_of_service",
			Description:    "Distributed denial of service attack on web application",
			ThreatActor:    "external_attacker",
			Motivation:     "disruption",
			Capability:     "medium",
			Likelihood:     "high",
			Impact:         "medium",
			RiskScore:      6.0,
			STRIDECategory: "denial_of_service",
			MITREAttackID:  "T1498",
			TargetAssets:   []string{"web-application", "cdn"},
			AttackVectors:  []string{"volumetric_attack", "application_layer_attack", "botnet"},
			Indicators:     []string{"traffic_spikes", "response_time_degradation"},
		},
		// Elevation of Privilege threats
		{
			ID:             "T005",
			Name:           "Admin Privilege Escalation",
			Category:       "elevation_of_privilege",
			Description:    "Unauthorized elevation to administrative privileges",
			ThreatActor:    "insider_threat",
			Motivation:     "unauthorized_access",
			Capability:     "high",
			Likelihood:     "low",
			Impact:         "critical",
			RiskScore:      7.0,
			STRIDECategory: "elevation_of_privilege",
			MITREAttackID:  "T1068",
			TargetAssets:   []string{"admin-panel", "database"},
			AttackVectors:  []string{"privilege_escalation", "configuration_exploit", "insider_access"},
			Indicators:     []string{"privilege_changes", "unauthorized_admin_actions"},
		},
	}

	// Calculate risk scores based on likelihood and impact
	for i := range threats {
		threats[i].RiskScore = t.calculateThreatRiskScore(threats[i])
	}

	return threats
}

// identifyVulnerabilities identifies system vulnerabilities
func (t *ThreatModelingEngine) identifyVulnerabilities(assets []Asset) []Vulnerability {
	return []Vulnerability{
		{
			ID:           "V001",
			Title:        "Weak Authentication Mechanisms",
			Severity:     "high",
			CVSS:         7.5,
			Package:      "authentication-system",
			Description:  "Insufficient password complexity requirements",
			Scanner:      "threat-modeling",
		},
		{
			ID:           "V002",
			Title:        "Insufficient Input Validation",
			Severity:     "medium",
			CVSS:         6.0,
			Package:      "web-application",
			Description:  "Potential for injection attacks",
			Scanner:      "threat-modeling",
		},
		{
			ID:           "V003",
			Title:        "Inadequate Session Management",
			Severity:     "high",
			CVSS:         7.0,
			Package:      "session-management",
			Description:  "Session tokens not properly secured",
			Scanner:      "threat-modeling",
		},
	}
}

// mapAttackPaths identifies potential attack paths
func (t *ThreatModelingEngine) mapAttackPaths(assets []Asset, threats []Threat) []AttackPath {
	return []AttackPath{
		{
			ID:          "AP001",
			Name:        "External to Database",
			Description: "Attack path from external access to database compromise",
			StartAsset:  "web-application",
			TargetAsset: "database",
			Steps: []AttackStep{
				{
					ID:          "AS001",
					Name:        "Initial Access",
					Description: "Gain initial access to web application",
					Technique:   "credential_stuffing",
					Asset:       "web-application",
					Success:     0.3,
					Detection:   0.7,
					Impact:      "low",
				},
				{
					ID:          "AS002",
					Name:        "Privilege Escalation",
					Description: "Escalate privileges within application",
					Technique:   "privilege_escalation",
					Asset:       "web-application",
					Success:     0.4,
					Detection:   0.6,
					Impact:      "medium",
				},
				{
					ID:          "AS003",
					Name:        "Database Access",
					Description: "Access database through application",
					Technique:   "sql_injection",
					Asset:       "database",
					Success:     0.5,
					Detection:   0.8,
					Impact:      "critical",
				},
			},
			Probability:         0.06, // 0.3 * 0.4 * 0.5
			Impact:             "critical",
			Complexity:         "medium",
			DetectionDifficulty: "medium",
		},
	}
}

// identifyMitigations identifies security controls and mitigations
func (t *ThreatModelingEngine) identifyMitigations(threats []Threat) []Mitigation {
	return []Mitigation{
		{
			ID:             "M001",
			Name:           "Multi-Factor Authentication",
			Type:           "preventive",
			Description:    "Implement MFA for all user accounts",
			Effectiveness:  "high",
			Cost:           "medium",
			Complexity:     "medium",
			Status:         "planned",
			Owner:          "security-team",
			TargetThreats:  []string{"T001", "T005"},
			Implementation: "Deploy MFA solution with TOTP/SMS backup",
			Validation:     "Penetration testing and user acceptance testing",
		},
		{
			ID:             "M002",
			Name:           "Input Validation Framework",
			Type:           "preventive",
			Description:    "Comprehensive input validation and sanitization",
			Effectiveness:  "high",
			Cost:           "low",
			Complexity:     "low",
			Status:         "implemented",
			Owner:          "development-team",
			TargetThreats:  []string{"T002", "T003"},
			Implementation: "Server-side validation with whitelist approach",
			Validation:     "Automated security testing and code review",
		},
		{
			ID:             "M003",
			Name:           "Rate Limiting and DDoS Protection",
			Type:           "preventive",
			Description:    "Implement rate limiting and DDoS protection",
			Effectiveness:  "medium",
			Cost:           "medium",
			Complexity:     "medium",
			Status:         "implemented",
			Owner:          "infrastructure-team",
			TargetThreats:  []string{"T004"},
			Implementation: "CDN-based DDoS protection with rate limiting",
			Validation:     "Load testing and DDoS simulation",
		},
	}
}

// performRiskAssessment calculates overall risk assessment
func (t *ThreatModelingEngine) performRiskAssessment(threats []Threat, mitigations []Mitigation) RiskAssessment {
	assessment := RiskAssessment{
		UnmitigatedRisks: []UnmitigatedRisk{},
		RiskTrends:       []RiskTrend{},
		Recommendations:  []string{},
		NextReview:       time.Now().AddDate(0, 3, 0), // 3 months
	}

	// Count threats by severity
	for _, threat := range threats {
		switch {
		case threat.RiskScore >= 9.0:
			assessment.CriticalThreats++
		case threat.RiskScore >= 7.0:
			assessment.HighThreats++
		case threat.RiskScore >= 4.0:
			assessment.MediumThreats++
		default:
			assessment.LowThreats++
		}
	}

	// Calculate overall risk score
	totalRisk := 0.0
	for _, threat := range threats {
		totalRisk += threat.RiskScore
	}
	assessment.RiskScore = totalRisk / float64(len(threats))

	// Determine overall risk level
	switch {
	case assessment.RiskScore >= 8.0:
		assessment.OverallRisk = "critical"
	case assessment.RiskScore >= 6.0:
		assessment.OverallRisk = "high"
	case assessment.RiskScore >= 4.0:
		assessment.OverallRisk = "medium"
	default:
		assessment.OverallRisk = "low"
	}

	// Generate recommendations
	if assessment.CriticalThreats > 0 {
		assessment.Recommendations = append(assessment.Recommendations,
			"URGENT: Address critical threats immediately")
	}
	if assessment.HighThreats > 3 {
		assessment.Recommendations = append(assessment.Recommendations,
			"Prioritize high-risk threat mitigation")
	}

	return assessment
}

// assessCompliance performs compliance assessment
func (t *ThreatModelingEngine) assessCompliance(assets []Asset, threats []Threat, mitigations []Mitigation) ComplianceAssessment {
	assessment := ComplianceAssessment{
		Frameworks:          []ComplianceFramework{},
		ComplianceGaps:      []ComplianceGap{},
		CertificationStatus: "in_progress",
		LastAudit:           time.Now().AddDate(0, -6, 0),
		NextAudit:           time.Now().AddDate(0, 6, 0),
		AuditFindings:       []AuditFinding{},
	}

	// Assess GDPR compliance
	gdprFramework := ComplianceFramework{
		Name:         "GDPR",
		Version:      "2018",
		Score:        75.0,
		Status:       "partial_compliance",
		LastAssessed: time.Now(),
		Controls: []ComplianceControl{
			{
				ID:          "GDPR-32",
				Name:        "Security of Processing",
				Description: "Implement appropriate technical and organizational measures",
				Status:      "implemented",
				LastTested:  time.Now().AddDate(0, -1, 0),
				NextTest:    time.Now().AddDate(0, 2, 0),
				Owner:       "security-team",
			},
			{
				ID:          "GDPR-25",
				Name:        "Data Protection by Design",
				Description: "Implement data protection by design and by default",
				Status:      "partial",
				LastTested:  time.Now().AddDate(0, -2, 0),
				NextTest:    time.Now().AddDate(0, 1, 0),
				Owner:       "development-team",
				Deficiencies: []string{"Privacy impact assessments needed"},
			},
		},
	}
	assessment.Frameworks = append(assessment.Frameworks, gdprFramework)

	// Calculate overall compliance score
	totalScore := 0.0
	for _, framework := range assessment.Frameworks {
		totalScore += framework.Score
	}
	assessment.OverallScore = totalScore / float64(len(assessment.Frameworks))

	return assessment
}

// calculateThreatRiskScore calculates risk score for a threat
func (t *ThreatModelingEngine) calculateThreatRiskScore(threat Threat) float64 {
	likelihoodScore := t.getLikelihoodScore(threat.Likelihood)
	impactScore := t.getImpactScore(threat.Impact)
	return (likelihoodScore + impactScore) / 2.0 * 10.0
}

// getLikelihoodScore converts likelihood to numeric score
func (t *ThreatModelingEngine) getLikelihoodScore(likelihood string) float64 {
	switch strings.ToLower(likelihood) {
	case "very_low":
		return 0.1
	case "low":
		return 0.3
	case "medium":
		return 0.5
	case "high":
		return 0.7
	case "very_high":
		return 0.9
	default:
		return 0.5
	}
}

// getImpactScore converts impact to numeric score
func (t *ThreatModelingEngine) getImpactScore(impact string) float64 {
	switch strings.ToLower(impact) {
	case "very_low":
		return 0.1
	case "low":
		return 0.3
	case "medium":
		return 0.5
	case "high":
		return 0.7
	case "critical":
		return 0.9
	default:
		return 0.5
	}
}

// generateThreatModelingRecommendations generates actionable recommendations
func (t *ThreatModelingEngine) generateThreatModelingRecommendations(result *ThreatModelingResult) []string {
	var recommendations []string

	// Risk-based recommendations
	if result.ThreatModel.RiskAssessment.CriticalThreats > 0 {
		recommendations = append(recommendations,
			"CRITICAL: Implement immediate mitigations for critical threats")
	}

	if result.ThreatModel.RiskAssessment.HighThreats > 3 {
		recommendations = append(recommendations,
			"HIGH: Prioritize mitigation of high-risk threats")
	}

	// Compliance-based recommendations
	if result.ThreatModel.ComplianceStatus.OverallScore < 80.0 {
		recommendations = append(recommendations,
			"Improve compliance posture to meet regulatory requirements")
	}

	// Attack simulation recommendations
	for _, simulation := range result.AttackSimulations {
		if simulation.Success {
			recommendations = append(recommendations,
				fmt.Sprintf("Address vulnerabilities exposed by %s simulation", simulation.AttackType))
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"Threat model is comprehensive - maintain current security posture")
	}

	return recommendations
}

// generateThreatModelingReports generates comprehensive reports
func (t *ThreatModelingEngine) generateThreatModelingReports(result *ThreatModelingResult) ([]string, error) {
	timestamp := time.Now().Format("20060102-150405")
	var reportPaths []string

	// Generate JSON report
	jsonPath := filepath.Join(t.reportDir, fmt.Sprintf("threat-model-report-%s.json", timestamp))
	if err := t.generateJSONReport(result, jsonPath); err != nil {
		log.Printf("Failed to generate JSON report: %v", err)
	} else {
		reportPaths = append(reportPaths, jsonPath)
	}

	// Generate HTML report
	htmlPath := filepath.Join(t.reportDir, fmt.Sprintf("threat-model-report-%s.html", timestamp))
	if err := t.generateHTMLReport(result, htmlPath); err != nil {
		log.Printf("Failed to generate HTML report: %v", err)
	} else {
		reportPaths = append(reportPaths, htmlPath)
	}

	return reportPaths, nil
}

// generateJSONReport creates a JSON report
func (t *ThreatModelingEngine) generateJSONReport(result *ThreatModelingResult, reportPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(reportPath, data, 0644)
}

// generateHTMLReport creates an HTML report
func (t *ThreatModelingEngine) generateHTMLReport(result *ThreatModelingResult, reportPath string) error {
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Threat Model Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; }
        .threat { border: 1px solid #ddd; margin: 10px 0; padding: 15px; border-radius: 5px; }
        .critical { border-left: 5px solid #d32f2f; }
        .high { border-left: 5px solid #f57c00; }
        .medium { border-left: 5px solid #fbc02d; }
        .low { border-left: 5px solid #388e3c; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Threat Model Report</h1>
        <p><strong>Model:</strong> %s</p>
        <p><strong>Generated:</strong> %s</p>
        <p><strong>Overall Risk:</strong> %s</p>
        <p><strong>Risk Score:</strong> %.1f</p>
    </div>
    
    <div class="section">
        <h2>Risk Summary</h2>
        <p><strong>Critical Threats:</strong> %d</p>
        <p><strong>High Threats:</strong> %d</p>
        <p><strong>Medium Threats:</strong> %d</p>
        <p><strong>Low Threats:</strong> %d</p>
    </div>
    
    <div class="section">
        <h2>Compliance Status</h2>
        <p><strong>Overall Score:</strong> %.1f%%</p>
        <p><strong>Status:</strong> %s</p>
    </div>
</body>
</html>`,
		result.ThreatModel.Name,
		time.Now().Format("2006-01-02 15:04:05"),
		result.ThreatModel.RiskAssessment.OverallRisk,
		result.ThreatModel.RiskAssessment.RiskScore,
		result.ThreatModel.RiskAssessment.CriticalThreats,
		result.ThreatModel.RiskAssessment.HighThreats,
		result.ThreatModel.RiskAssessment.MediumThreats,
		result.ThreatModel.RiskAssessment.LowThreats,
		result.ThreatModel.ComplianceStatus.OverallScore,
		result.ThreatModel.ComplianceStatus.CertificationStatus,
	)

	return os.WriteFile(reportPath, []byte(htmlContent), 0644)
}