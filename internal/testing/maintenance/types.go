package maintenance

import (
	"time"
)

// TestSuiteAnalysis represents the complete analysis of a test suite
type TestSuiteAnalysis struct {
	Timestamp   time.Time                    `json:"timestamp"`
	RootPath    string                       `json:"root_path"`
	Tests       map[string]*TestMetadata     `json:"tests"`
	Issues      []TestIssue                  `json:"issues"`
	Suggestions []MaintenanceSuggestion      `json:"suggestions"`
	Metrics     TestSuiteMetrics             `json:"metrics"`
}

// TestSuiteMetrics provides overall metrics about the test suite
type TestSuiteMetrics struct {
	TotalTests          int           `json:"total_tests"`
	ActiveTests         int           `json:"active_tests"`
	DeprecatedTests     int           `json:"deprecated_tests"`
	AverageCoverage     float64       `json:"average_coverage"`
	AverageRuntime      time.Duration `json:"average_runtime"`
	TotalExecutionTime  time.Duration `json:"total_execution_time"`
	HighFailureRateTests int          `json:"high_failure_rate_tests"`
	SlowTests           int           `json:"slow_tests"`
	OutdatedTests       int           `json:"outdated_tests"`
}

// TestIssue represents an issue found during test analysis
type TestIssue struct {
	Type        IssueType     `json:"type"`
	TestID      string        `json:"test_id"`
	Severity    IssueSeverity `json:"severity"`
	Description string        `json:"description"`
	Suggestion  string        `json:"suggestion"`
	DetectedAt  time.Time     `json:"detected_at"`
}

// IssueType defines types of test issues
type IssueType string

const (
	IssueHighFailureRate IssueType = "high_failure_rate"
	IssueSlowExecution   IssueType = "slow_execution"
	IssueOutdated        IssueType = "outdated"
	IssueLowCoverage     IssueType = "low_coverage"
	IssueHighComplexity  IssueType = "high_complexity"
	IssueDuplicate       IssueType = "duplicate"
	IssueUnused          IssueType = "unused"
	IssueFlaky           IssueType = "flaky"
	IssueMissingDeps     IssueType = "missing_dependencies"
)

// IssueSeverity defines severity levels for issues
type IssueSeverity string

const (
	SeverityCritical IssueSeverity = "critical"
	SeverityHigh     IssueSeverity = "high"
	SeverityMedium   IssueSeverity = "medium"
	SeverityLow      IssueSeverity = "low"
)

// MaintenanceSuggestion represents a suggestion for test maintenance
type MaintenanceSuggestion struct {
	Type        SuggestionType `json:"type"`
	Priority    Priority       `json:"priority"`
	Description string         `json:"description"`
	TestIDs     []string       `json:"test_ids"`
	Action      string         `json:"action"`
	EstimatedEffort string     `json:"estimated_effort"`
	Benefits    []string       `json:"benefits"`
	CreatedAt   time.Time      `json:"created_at"`
}

// SuggestionType defines types of maintenance suggestions
type SuggestionType string

const (
	SuggestionConsolidate SuggestionType = "consolidate"
	SuggestionDeprecate   SuggestionType = "deprecate"
	SuggestionOptimize    SuggestionType = "optimize"
	SuggestionRefactor    SuggestionType = "refactor"
	SuggestionUpdate      SuggestionType = "update"
	SuggestionRemove      SuggestionType = "remove"
	SuggestionParallelize SuggestionType = "parallelize"
	SuggestionMock        SuggestionType = "mock"
)

// Priority defines priority levels for suggestions
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
)

// TestEvolution represents the evolution history of a test
type TestEvolution struct {
	TestID      string                 `json:"test_id"`
	Changes     []TestChange           `json:"changes"`
	Metrics     []TestMetricSnapshot   `json:"metrics"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TestChange represents a change made to a test
type TestChange struct {
	ID          string     `json:"id"`
	Type        ChangeType `json:"type"`
	Description string     `json:"description"`
	Author      string     `json:"author"`
	Timestamp   time.Time  `json:"timestamp"`
	Impact      Impact     `json:"impact"`
	Reason      string     `json:"reason"`
}

// ChangeType defines types of test changes
type ChangeType string

const (
	ChangeCreated     ChangeType = "created"
	ChangeModified    ChangeType = "modified"
	ChangeDeprecated  ChangeType = "deprecated"
	ChangeRemoved     ChangeType = "removed"
	ChangeOptimized   ChangeType = "optimized"
	ChangeRefactored  ChangeType = "refactored"
	ChangeMigrated    ChangeType = "migrated"
)

// Impact represents the impact of a change
type Impact struct {
	CoverageChange  float64       `json:"coverage_change"`
	RuntimeChange   time.Duration `json:"runtime_change"`
	StabilityChange float64       `json:"stability_change"`
	ComplexityChange int          `json:"complexity_change"`
}

// TestMetricSnapshot represents metrics at a point in time
type TestMetricSnapshot struct {
	Timestamp      time.Time     `json:"timestamp"`
	Coverage       float64       `json:"coverage"`
	Runtime        time.Duration `json:"runtime"`
	FailureRate    float64       `json:"failure_rate"`
	ExecutionCount int           `json:"execution_count"`
	Complexity     int           `json:"complexity"`
}

// TestMigration represents a test framework migration
type TestMigration struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Status      MigrationStatus   `json:"status"`
	Steps       []MigrationStep   `json:"steps"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// MigrationStatus defines migration status
type MigrationStatus string

const (
	MigrationPending    MigrationStatus = "pending"
	MigrationRunning    MigrationStatus = "running"
	MigrationCompleted  MigrationStatus = "completed"
	MigrationFailed     MigrationStatus = "failed"
	MigrationRolledBack MigrationStatus = "rolled_back"
)

// MigrationStep represents a step in a migration
type MigrationStep struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Status      StepStatus  `json:"status"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Error       string      `json:"error,omitempty"`
	Rollback    string      `json:"rollback,omitempty"`
}

// StepStatus defines migration step status
type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
	StepSkipped   StepStatus = "skipped"
)

// TestLifecycleEvent represents an event in a test's lifecycle
type TestLifecycleEvent struct {
	ID        string          `json:"id"`
	TestID    string          `json:"test_id"`
	EventType LifecycleEvent  `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
	Reason    string          `json:"reason"`
}

// LifecycleEvent defines types of lifecycle events
type LifecycleEvent string

const (
	EventCreated     LifecycleEvent = "created"
	EventActivated   LifecycleEvent = "activated"
	EventDeprecated  LifecycleEvent = "deprecated"
	EventObsoleted   LifecycleEvent = "obsoleted"
	EventQuarantined LifecycleEvent = "quarantined"
	EventRemoved     LifecycleEvent = "removed"
	EventMigrated    LifecycleEvent = "migrated"
)

// MaintenanceSchedule represents a scheduled maintenance task
type MaintenanceSchedule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        MaintenanceType        `json:"type"`
	Schedule    string                 `json:"schedule"` // Cron expression
	Enabled     bool                   `json:"enabled"`
	LastRun     *time.Time             `json:"last_run,omitempty"`
	NextRun     *time.Time             `json:"next_run,omitempty"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// MaintenanceType defines types of maintenance tasks
type MaintenanceType string

const (
	MaintenanceAnalysis     MaintenanceType = "analysis"
	MaintenanceCleanup      MaintenanceType = "cleanup"
	MaintenanceOptimization MaintenanceType = "optimization"
	MaintenanceDeprecation  MaintenanceType = "deprecation"
	MaintenanceMigration    MaintenanceType = "migration"
	MaintenanceValidation   MaintenanceType = "validation"
)

// QualityMetrics represents quality metrics for tests
type QualityMetrics struct {
	TestID              string    `json:"test_id"`
	Maintainability     float64   `json:"maintainability"`
	Readability         float64   `json:"readability"`
	Reliability         float64   `json:"reliability"`
	Performance         float64   `json:"performance"`
	Coverage            float64   `json:"coverage"`
	OverallQuality      float64   `json:"overall_quality"`
	LastCalculated      time.Time `json:"last_calculated"`
	TrendDirection      string    `json:"trend_direction"` // "improving", "degrading", "stable"
}

// RefactoringOpportunity represents an opportunity for test refactoring
type RefactoringOpportunity struct {
	ID              string             `json:"id"`
	Type            RefactoringType    `json:"type"`
	TestIDs         []string           `json:"test_ids"`
	Description     string             `json:"description"`
	Benefits        []string           `json:"benefits"`
	EstimatedEffort string             `json:"estimated_effort"`
	Priority        Priority           `json:"priority"`
	AutoApplicable  bool               `json:"auto_applicable"`
	CreatedAt       time.Time          `json:"created_at"`
}

// RefactoringType defines types of refactoring opportunities
type RefactoringType string

const (
	RefactoringExtractCommon    RefactoringType = "extract_common"
	RefactoringRemoveDuplication RefactoringType = "remove_duplication"
	RefactoringSimplifyLogic    RefactoringType = "simplify_logic"
	RefactoringImproveNaming    RefactoringType = "improve_naming"
	RefactoringReduceComplexity RefactoringType = "reduce_complexity"
	RefactoringOptimizeSetup    RefactoringType = "optimize_setup"
)