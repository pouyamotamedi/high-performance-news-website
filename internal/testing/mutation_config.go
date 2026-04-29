package testing

import (
	"encoding/json"
	"os"
	"time"
)

// DefaultMutationConfig returns a default configuration for mutation testing
func DefaultMutationConfig() *MutationConfig {
	return &MutationConfig{
		TargetPackages: []string{
			"./internal/models",
			"./internal/auth",
			"./internal/services",
			"./internal/repositories",
		},
		TestPackages: []string{
			"./internal/models",
			"./internal/auth", 
			"./internal/services",
			"./internal/repositories",
		},
		ExcludePatterns: []string{
			".*_test\\.go$",
			".*mock.*\\.go$",
			".*generated.*\\.go$",
			".*/vendor/.*",
		},
		MutationTypes: []string{
			"ConditionalBoundaryMutator",
			"ArithmeticOperatorMutator",
			"LogicalOperatorMutator",
			"ReturnValueMutator",
			"NullCheckMutator",
			"SecurityMutator",
			"PerformanceMutator",
		},
		MinMutationScore: 80.0,
		Timeout:         30 * time.Second,
		MaxConcurrency:  4,
		CriticalFunctions: []string{
			// Business Logic Functions
			"ValidateUser",
			"ValidateArticle", 
			"ProcessPayment",
			"CalculateScore",
			"GenerateSlug",
			"ValidateInput",
			
			// Content Management
			"PublishArticle",
			"ArchiveArticle",
			"UpdateArticle",
			"DeleteArticle",
			"CreateArticle",
		},
		SecurityFunctions: []string{
			// Authentication Functions
			"HashPassword",
			"ComparePassword",
			"GenerateToken",
			"ValidateToken",
			"RefreshToken",
			
			// Authorization Functions
			"CheckPermission",
			"HasRole",
			"CanAccess",
			"IsAuthorized",
			
			// Input Validation
			"SanitizeInput",
			"ValidateEmail",
			"ValidateUsername",
			"EscapeHTML",
			"ValidateURL",
		},
		PerformanceFunctions: []string{
			// Database Operations
			"Query",
			"QueryRow", 
			"Exec",
			"Prepare",
			"Begin",
			"Commit",
			"Rollback",
			
			// Cache Operations
			"Get",
			"Set",
			"Del",
			"Exists",
			"Expire",
			
			// Search Operations
			"Search",
			"Index",
			"BulkIndex",
			
			// File Operations
			"ReadFile",
			"WriteFile",
			"ProcessImage",
			"GenerateStatic",
		},
	}
}

// LoadMutationConfig loads configuration from a JSON file
func LoadMutationConfig(filename string) (*MutationConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	var config MutationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// SaveMutationConfig saves configuration to a JSON file
func (mc *MutationConfig) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}

// CriticalCodeConfig defines configuration for critical code identification
type CriticalCodeConfig struct {
	BusinessLogicPatterns []string `json:"business_logic_patterns"`
	SecurityPatterns      []string `json:"security_patterns"`
	PerformancePatterns   []string `json:"performance_patterns"`
	CriticalFiles         []string `json:"critical_files"`
	CriticalPackages      []string `json:"critical_packages"`
}

// DefaultCriticalCodeConfig returns default critical code configuration
func DefaultCriticalCodeConfig() *CriticalCodeConfig {
	return &CriticalCodeConfig{
		BusinessLogicPatterns: []string{
			// Validation patterns
			"Validate.*",
			"Check.*",
			"Verify.*",
			".*Validation.*",
			
			// Business rule patterns
			"Calculate.*",
			"Process.*",
			"Generate.*",
			"Transform.*",
			"Convert.*",
			
			// State management patterns
			"Update.*Status",
			"Change.*State",
			"Transition.*",
		},
		SecurityPatterns: []string{
			// Authentication patterns
			".*Auth.*",
			".*Login.*",
			".*Password.*",
			".*Token.*",
			".*Session.*",
			
			// Authorization patterns
			".*Permission.*",
			".*Role.*",
			".*Access.*",
			"Can.*",
			"Has.*",
			"Is.*Authorized",
			
			// Input sanitization patterns
			"Sanitize.*",
			"Escape.*",
			"Clean.*",
			"Filter.*",
		},
		PerformancePatterns: []string{
			// Database patterns
			".*Query.*",
			".*Exec.*",
			".*Prepare.*",
			".*Transaction.*",
			
			// Cache patterns
			".*Cache.*",
			".*Get.*",
			".*Set.*",
			".*Del.*",
			
			// Batch processing patterns
			".*Batch.*",
			".*Bulk.*",
			".*Process.*Async",
			
			// Resource management patterns
			".*Pool.*",
			".*Connection.*",
			".*Resource.*",
		},
		CriticalFiles: []string{
			"internal/auth/auth.go",
			"internal/models/user.go",
			"internal/models/article.go",
			"internal/services/user_service.go",
			"internal/services/article_service.go",
			"internal/services/auth_service.go",
			"internal/repositories/user_repository.go",
			"internal/repositories/article_repository.go",
		},
		CriticalPackages: []string{
			"internal/auth",
			"internal/models",
			"internal/services",
			"internal/repositories",
		},
	}
}

// MutationTestSuite represents a complete mutation testing suite configuration
type MutationTestSuite struct {
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	MutationConfig    *MutationConfig     `json:"mutation_config"`
	CriticalConfig    *CriticalCodeConfig `json:"critical_config"`
	ReportingConfig   *ReportingConfig    `json:"reporting_config"`
	ScheduleConfig    *ScheduleConfig     `json:"schedule_config"`
}

// ReportingConfig defines how mutation testing results should be reported
type ReportingConfig struct {
	OutputFormats     []string `json:"output_formats"`     // json, html, csv, junit
	OutputDirectory   string   `json:"output_directory"`
	IncludeTrends     bool     `json:"include_trends"`
	EmailReports      bool     `json:"email_reports"`
	EmailRecipients   []string `json:"email_recipients"`
	SlackWebhook      string   `json:"slack_webhook"`
	FailureThreshold  float64  `json:"failure_threshold"`  // Fail CI if score below this
}

// ScheduleConfig defines when mutation testing should run
type ScheduleConfig struct {
	Enabled           bool     `json:"enabled"`
	CronExpression    string   `json:"cron_expression"`
	RunOnCommit       bool     `json:"run_on_commit"`
	RunOnPullRequest  bool     `json:"run_on_pull_request"`
	RunNightly        bool     `json:"run_nightly"`
	TargetBranches    []string `json:"target_branches"`
}

// DefaultMutationTestSuite returns a default mutation test suite configuration
func DefaultMutationTestSuite() *MutationTestSuite {
	return &MutationTestSuite{
		Name:        "Comprehensive Mutation Testing Suite",
		Description: "Mutation testing for critical business logic, security, and performance code",
		MutationConfig: DefaultMutationConfig(),
		CriticalConfig: DefaultCriticalCodeConfig(),
		ReportingConfig: &ReportingConfig{
			OutputFormats:    []string{"json", "html", "csv"},
			OutputDirectory:  "mutation_reports",
			IncludeTrends:    true,
			EmailReports:     false,
			FailureThreshold: 75.0,
		},
		ScheduleConfig: &ScheduleConfig{
			Enabled:          true,
			RunOnCommit:      false, // Too expensive for every commit
			RunOnPullRequest: true,
			RunNightly:       true,
			CronExpression:   "0 2 * * *", // 2 AM daily
			TargetBranches:   []string{"main", "develop"},
		},
	}
}

// LoadMutationTestSuite loads a complete test suite configuration
func LoadMutationTestSuite(filename string) (*MutationTestSuite, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	var suite MutationTestSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, err
	}
	
	return &suite, nil
}

// SaveToFile saves the mutation test suite configuration
func (mts *MutationTestSuite) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(mts, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}