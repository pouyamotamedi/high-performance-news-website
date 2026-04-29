package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"high-performance-news-website/internal/integration/interfaces"
)

// WorkflowAutomation manages automated workflows and custom integrations
type WorkflowAutomation struct {
	workflows map[string]*Workflow
	triggers  map[string][]string // event type -> workflow names
	executor  *WorkflowExecutor
	mu        sync.RWMutex
}

// Workflow represents an automated workflow
type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	Triggers    []WorkflowTrigger      `json:"triggers"`
	Actions     []WorkflowAction       `json:"actions"`
	Conditions  []WorkflowCondition    `json:"conditions"`
	Config      WorkflowConfig         `json:"config"`
	Stats       WorkflowStats          `json:"stats"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// WorkflowTrigger defines what triggers a workflow
type WorkflowTrigger struct {
	Type       TriggerType            `json:"type"`
	EventType  interfaces.EventType  `json:"event_type,omitempty"`
	Schedule   string                 `json:"schedule,omitempty"` // cron expression
	Webhook    string                 `json:"webhook,omitempty"`
	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// WorkflowAction defines an action to perform
type WorkflowAction struct {
	Type        ActionType             `json:"type"`
	Integration string                 `json:"integration,omitempty"`
	Template    string                 `json:"template,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
	Timeout     time.Duration          `json:"timeout"`
	Retries     int                    `json:"retries"`
}

// WorkflowCondition defines conditions for workflow execution
type WorkflowCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// WorkflowConfig holds workflow configuration
type WorkflowConfig struct {
	MaxConcurrentExecutions int           `json:"max_concurrent_executions"`
	ExecutionTimeout        time.Duration `json:"execution_timeout"`
	RetryPolicy             RetryPolicy   `json:"retry_policy"`
	NotificationSettings    Notification  `json:"notification_settings"`
}

// WorkflowStats holds workflow execution statistics
type WorkflowStats struct {
	TotalExecutions    int64     `json:"total_executions"`
	SuccessfulRuns     int64     `json:"successful_runs"`
	FailedRuns         int64     `json:"failed_runs"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	LastExecution      time.Time `json:"last_execution"`
	LastSuccess        time.Time `json:"last_success"`
	LastFailure        time.Time `json:"last_failure"`
}

// TriggerType defines the type of workflow trigger
type TriggerType string

const (
	TriggerTypeEvent    TriggerType = "event"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeWebhook  TriggerType = "webhook"
	TriggerTypeManual   TriggerType = "manual"
)

// ActionType defines the type of workflow action
type ActionType string

const (
	ActionTypeSendEvent      ActionType = "send_event"
	ActionTypeCreateIssue    ActionType = "create_issue"
	ActionTypeSendMessage    ActionType = "send_message"
	ActionTypeRunScript      ActionType = "run_script"
	ActionTypeHTTPRequest    ActionType = "http_request"
	ActionTypeDelay          ActionType = "delay"
	ActionTypeConditional    ActionType = "conditional"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffMultiplier float64     `json:"backoff_multiplier"`
}

// Notification defines notification settings
type Notification struct {
	OnSuccess bool     `json:"on_success"`
	OnFailure bool     `json:"on_failure"`
	Channels  []string `json:"channels"`
}

// WorkflowExecution represents a workflow execution instance
type WorkflowExecution struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	Status      ExecutionStatus        `json:"status"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Duration    time.Duration          `json:"duration"`
	TriggerData map[string]interface{} `json:"trigger_data"`
	Results     []ActionResult         `json:"results"`
	Error       string                 `json:"error,omitempty"`
}

// ExecutionStatus defines workflow execution status
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// ActionResult represents the result of an action execution
type ActionResult struct {
	ActionIndex int                    `json:"action_index"`
	Status      ExecutionStatus        `json:"status"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Duration    time.Duration          `json:"duration"`
	Output      map[string]interface{} `json:"output"`
	Error       string                 `json:"error,omitempty"`
}

// NewWorkflowAutomation creates a new workflow automation system
func NewWorkflowAutomation() *WorkflowAutomation {
	return &WorkflowAutomation{
		workflows: make(map[string]*Workflow),
		triggers:  make(map[string][]string),
		executor:  NewWorkflowExecutor(),
	}
}

// RegisterWorkflow registers a new workflow
func (wa *WorkflowAutomation) RegisterWorkflow(workflow *Workflow) error {
	wa.mu.Lock()
	defer wa.mu.Unlock()

	if workflow.ID == "" {
		workflow.ID = fmt.Sprintf("workflow_%d", time.Now().UnixNano())
	}

	workflow.CreatedAt = time.Now()
	workflow.UpdatedAt = time.Now()

	// Set default config values
	if workflow.Config.MaxConcurrentExecutions == 0 {
		workflow.Config.MaxConcurrentExecutions = 1
	}
	if workflow.Config.ExecutionTimeout == 0 {
		workflow.Config.ExecutionTimeout = 5 * time.Minute
	}

	wa.workflows[workflow.ID] = workflow

	// Register triggers
	for _, trigger := range workflow.Triggers {
		if trigger.Type == TriggerTypeEvent {
			eventType := string(trigger.EventType)
			wa.triggers[eventType] = append(wa.triggers[eventType], workflow.ID)
		}
	}

	log.Printf("Registered workflow: %s (%s)", workflow.Name, workflow.ID)
	return nil
}

// UnregisterWorkflow removes a workflow
func (wa *WorkflowAutomation) UnregisterWorkflow(workflowID string) error {
	wa.mu.Lock()
	defer wa.mu.Unlock()

	workflow, exists := wa.workflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow %s not found", workflowID)
	}

	// Remove triggers
	for _, trigger := range workflow.Triggers {
		if trigger.Type == TriggerTypeEvent {
			eventType := string(trigger.EventType)
			if workflows, exists := wa.triggers[eventType]; exists {
				for i, id := range workflows {
					if id == workflowID {
						wa.triggers[eventType] = append(workflows[:i], workflows[i+1:]...)
						break
					}
				}
			}
		}
	}

	delete(wa.workflows, workflowID)
	log.Printf("Unregistered workflow: %s", workflowID)
	return nil
}

// TriggerWorkflows triggers workflows based on an event
func (wa *WorkflowAutomation) TriggerWorkflows(ctx context.Context, event interfaces.Event) error {
	wa.mu.RLock()
	eventType := string(event.Type)
	workflowIDs := wa.triggers[eventType]
	wa.mu.RUnlock()

	if len(workflowIDs) == 0 {
		return nil // No workflows to trigger
	}

	var errors []error
	for _, workflowID := range workflowIDs {
		if err := wa.ExecuteWorkflow(ctx, workflowID, map[string]interface{}{
			"event": event,
		}); err != nil {
			errors = append(errors, fmt.Errorf("failed to execute workflow %s: %w", workflowID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("workflow execution errors: %v", errors)
	}

	return nil
}

// ExecuteWorkflow executes a workflow manually
func (wa *WorkflowAutomation) ExecuteWorkflow(ctx context.Context, workflowID string, triggerData map[string]interface{}) error {
	wa.mu.RLock()
	workflow, exists := wa.workflows[workflowID]
	wa.mu.RUnlock()

	if !exists {
		return fmt.Errorf("workflow %s not found", workflowID)
	}

	if !workflow.Enabled {
		return fmt.Errorf("workflow %s is disabled", workflowID)
	}

	// Check conditions
	if !wa.evaluateConditions(workflow.Conditions, triggerData) {
		log.Printf("Workflow %s conditions not met, skipping execution", workflowID)
		return nil
	}

	// Create execution context
	execution := &WorkflowExecution{
		ID:          fmt.Sprintf("exec_%d", time.Now().UnixNano()),
		WorkflowID:  workflowID,
		Status:      ExecutionStatusPending,
		StartTime:   time.Now(),
		TriggerData: triggerData,
	}

	// Execute workflow
	go wa.executor.Execute(ctx, workflow, execution)

	return nil
}

// evaluateConditions evaluates workflow conditions
func (wa *WorkflowAutomation) evaluateConditions(conditions []WorkflowCondition, data map[string]interface{}) bool {
	for _, condition := range conditions {
		if !wa.evaluateCondition(condition, data) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition
func (wa *WorkflowAutomation) evaluateCondition(condition WorkflowCondition, data map[string]interface{}) bool {
	value, exists := data[condition.Field]
	if !exists {
		return false
	}

	switch condition.Operator {
	case "equals":
		return value == condition.Value
	case "not_equals":
		return value != condition.Value
	case "contains":
		if str, ok := value.(string); ok {
			if substr, ok := condition.Value.(string); ok {
				return contains(str, substr)
			}
		}
	case "greater_than":
		return compareNumbers(value, condition.Value) > 0
	case "less_than":
		return compareNumbers(value, condition.Value) < 0
	case "exists":
		return exists
	}

	return false
}

// GetWorkflow returns a workflow by ID
func (wa *WorkflowAutomation) GetWorkflow(workflowID string) (*Workflow, error) {
	wa.mu.RLock()
	defer wa.mu.RUnlock()

	workflow, exists := wa.workflows[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}

	return workflow, nil
}

// GetAllWorkflows returns all workflows
func (wa *WorkflowAutomation) GetAllWorkflows() map[string]*Workflow {
	wa.mu.RLock()
	defer wa.mu.RUnlock()

	workflows := make(map[string]*Workflow)
	for id, workflow := range wa.workflows {
		workflows[id] = workflow
	}

	return workflows
}

// UpdateWorkflow updates an existing workflow
func (wa *WorkflowAutomation) UpdateWorkflow(workflowID string, updates *Workflow) error {
	wa.mu.Lock()
	defer wa.mu.Unlock()

	workflow, exists := wa.workflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow %s not found", workflowID)
	}

	// Update fields
	if updates.Name != "" {
		workflow.Name = updates.Name
	}
	if updates.Description != "" {
		workflow.Description = updates.Description
	}
	workflow.Enabled = updates.Enabled
	if len(updates.Triggers) > 0 {
		workflow.Triggers = updates.Triggers
	}
	if len(updates.Actions) > 0 {
		workflow.Actions = updates.Actions
	}
	if len(updates.Conditions) > 0 {
		workflow.Conditions = updates.Conditions
	}

	workflow.UpdatedAt = time.Now()

	return nil
}

// Helper functions
func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[:len(substr)] == substr
}

func compareNumbers(a, b interface{}) int {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)
	
	if !aOk || !bOk {
		return 0
	}
	
	if aFloat > bFloat {
		return 1
	} else if aFloat < bFloat {
		return -1
	}
	return 0
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}