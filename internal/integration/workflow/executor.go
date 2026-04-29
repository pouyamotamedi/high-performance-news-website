package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// WorkflowExecutor executes workflow actions
type WorkflowExecutor struct {
	client *http.Client
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor() *WorkflowExecutor {
	return &WorkflowExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute executes a workflow
func (we *WorkflowExecutor) Execute(ctx context.Context, workflow *Workflow, execution *WorkflowExecution) {
	execution.Status = ExecutionStatusRunning
	execution.StartTime = time.Now()

	log.Printf("Starting execution of workflow %s (%s)", workflow.Name, execution.ID)

	// Execute actions sequentially
	for i, action := range workflow.Actions {
		actionCtx, cancel := context.WithTimeout(ctx, action.Timeout)
		if action.Timeout == 0 {
			actionCtx, cancel = context.WithTimeout(ctx, workflow.Config.ExecutionTimeout)
		}

		result := we.executeAction(actionCtx, action, execution.TriggerData)
		result.ActionIndex = i
		execution.Results = append(execution.Results, result)

		cancel()

		// If action failed and no retries, stop execution
		if result.Status == ExecutionStatusFailed && action.Retries == 0 {
			execution.Status = ExecutionStatusFailed
			execution.Error = result.Error
			break
		}

		// Handle retries
		if result.Status == ExecutionStatusFailed && action.Retries > 0 {
			retryResult := we.retryAction(ctx, action, execution.TriggerData, workflow.Config.RetryPolicy)
			if retryResult.Status == ExecutionStatusFailed {
				execution.Status = ExecutionStatusFailed
				execution.Error = retryResult.Error
				break
			}
			execution.Results[i] = retryResult
		}
	}

	// Set final status if not already failed
	if execution.Status != ExecutionStatusFailed {
		execution.Status = ExecutionStatusCompleted
	}

	execution.EndTime = time.Now()
	execution.Duration = execution.EndTime.Sub(execution.StartTime)

	// Update workflow stats
	we.updateWorkflowStats(workflow, execution)

	log.Printf("Completed execution of workflow %s (%s) in %v with status %s",
		workflow.Name, execution.ID, execution.Duration, execution.Status)
}

// executeAction executes a single workflow action
func (we *WorkflowExecutor) executeAction(ctx context.Context, action WorkflowAction, triggerData map[string]interface{}) ActionResult {
	result := ActionResult{
		Status:    ExecutionStatusRunning,
		StartTime: time.Now(),
	}

	switch action.Type {
	case ActionTypeSendEvent:
		result = we.executeSendEvent(ctx, action, triggerData)
	case ActionTypeCreateIssue:
		result = we.executeCreateIssue(ctx, action, triggerData)
	case ActionTypeSendMessage:
		result = we.executeSendMessage(ctx, action, triggerData)
	case ActionTypeRunScript:
		result = we.executeRunScript(ctx, action, triggerData)
	case ActionTypeHTTPRequest:
		result = we.executeHTTPRequest(ctx, action, triggerData)
	case ActionTypeDelay:
		result = we.executeDelay(ctx, action, triggerData)
	case ActionTypeConditional:
		result = we.executeConditional(ctx, action, triggerData)
	default:
		result.Status = ExecutionStatusFailed
		result.Error = fmt.Sprintf("unknown action type: %s", action.Type)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// executeSendEvent executes a send event action
func (we *WorkflowExecutor) executeSendEvent(ctx context.Context, action WorkflowAction, triggerData map[string]interface{}) ActionResult {
	result := ActionResult{Status: ExecutionStatusRunning}

	// This would integrate with the integration manager to send events
	// For now, we'll simulate it
	eventData := we.processTemplate(action.Template, triggerData)
	
	result.Output = map[string]interface{}{
		"event_sent": true,
		"event_data": eventData,
	}
	result.Status = ExecutionStatusCompleted

	return result
}

// executeCreateIssue executes a create issue action
func (we *WorkflowExecutor) executeCreateIssue(ctx context.Context, action WorkflowAction, triggerData map[string]interface{}) ActionResult {
	result := ActionResult{Status: ExecutionStatusRunning}

	// Process template for issue creation
	issueData := we.processTemplate(action.Template, triggerData)
	
	// This would integrate with bug tracking systems
	result.Output = map[string]interface{}{
		"issue_created": true,
		"issue_data":    issueData,
	}
	result.Status = ExecutionStatusCompleted

	return result
}

// executeSendMessage executes a send message action
func (we *WorkflowExecutor) executeSendMessage(ctx context.Context, action WorkflowAction, triggerData map[string]interface{}) ActionResult {
	result := ActionResult{Status: ExecutionStatusRunning}

	message := we.processTemplate(action.Template, triggerData)
	
	// This would integrate with communication systems
	result.Output = map[string]interface{}{
		"message_sent": true,
		"message":      message,
	}
	result.Status = ExecutionStatusCompleted

	return result
}

// executeRunScript executes a run script action
func (we *WorkflowExecutor) executeRunScript(ctx context.Context, action WorkflowAction, triggerData map[string]interface{}) ActionResult {
	result := ActionResult{Status: ExecutionStatusRunning}

	scriptPath, ok := action.Parameters["script"].(string)
	if !ok {
		result.Status = ExecutionStatusFailed
		result.Error = "script parameter is required"
		return result
	}

	// Prepare arguments
	var args []string
	if argsParam, ok := action.Parameters["args"].([]interface{}); ok {
		for _, arg := range argsParam {
			if argStr, ok := arg.(string); ok {
				args = append(args, we.processTemplate(argStr, triggerData))
			}
		}
	}

	// Execute script
	cmd := exec.CommandContext(ctx, scriptPath, args...)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = fmt.Sprintf("script execution failed: %v", err)
		result.Output = map[string]interface{}{
			"output": string(output),
		}
		return result
	}

	result.Status = ExecutionStatusCompleted
	result.Output = map[string]interface{}{
		"output":     string(output),
		"exit_code":  cmd.ProcessState.ExitCode(),
	}

	return result
}

// executeHTTPRequest executes an HTTP request action
func (we *WorkflowExecutor) executeHTTPRequest(ctx context.Context, action WorkflowAction, triggerData map[string]interface{}) ActionResult {
	result := ActionResult{Status: ExecutionStatusRunning}

	url, ok := action.Parameters["url"].(string)
	if !ok {
		result.Status = ExecutionStatusFailed
		result.Error = "url parameter is required"
		return result
	}

	method, ok := action.Parameters["method"].(string)
	if !ok {
		method = "GET"
	}

	// Process URL template
	url = we.processTemplate(url, triggerData)

	// Prepare request body
	var body io.Reader
	if bodyData, ok := action.Parameters["body"]; ok {
		if bodyStr, ok := bodyData.(string); ok {
			processedBody := we.processTemplate(bodyStr, triggerData)
			body = strings.NewReader(processedBody)
		} else {
			jsonBody, err := json.Marshal(bodyData)
			if err != nil {
				result.Status = ExecutionStatusFailed
				result.Error = fmt.Sprintf("failed to marshal request body: %v", err)
				return result
			}
			body = bytes.NewReader(jsonBody)
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	// Add headers
	if headers, ok := action.Parameters["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if valueStr, ok := value.(string); ok {
				req.Header.Set(key, we.processTemplate(valueStr, triggerData))
			}
		}
	}

	// Execute request
	resp, err := we.client.Do(req)
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = fmt.Sprintf("failed to read response: %v", err)
		return result
	}

	result.Status = ExecutionStatusCompleted
	result.Output = map[string]interface{}{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		"body":        string(responseBody),
	}

	return result
}

// executeDelay executes a delay action
func (we *WorkflowExecutor) executeDelay(ctx context.Context, action WorkflowAction, triggerData map[string]interface{}) ActionResult {
	result := ActionResult{Status: ExecutionStatusRunning}

	durationStr, ok := action.Parameters["duration"].(string)
	if !ok {
		result.Status = ExecutionStatusFailed
		result.Error = "duration parameter is required"
		return result
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = fmt.Sprintf("invalid duration: %v", err)
		return result
	}

	select {
	case <-ctx.Done():
		result.Status = ExecutionStatusCancelled
		result.Error = "delay cancelled"
	case <-time.After(duration):
		result.Status = ExecutionStatusCompleted
		result.Output = map[string]interface{}{
			"delayed_for": duration.String(),
		}
	}

	return result
}

// executeConditional executes a conditional action
func (we *WorkflowExecutor) executeConditional(ctx context.Context, action WorkflowAction, triggerData map[string]interface{}) ActionResult {
	result := ActionResult{Status: ExecutionStatusRunning}

	// This would evaluate conditions and execute nested actions
	// For now, we'll simulate it
	result.Status = ExecutionStatusCompleted
	result.Output = map[string]interface{}{
		"condition_evaluated": true,
	}

	return result
}

// retryAction retries a failed action
func (we *WorkflowExecutor) retryAction(ctx context.Context, action WorkflowAction, triggerData map[string]interface{}, retryPolicy RetryPolicy) ActionResult {
	var lastResult ActionResult
	delay := retryPolicy.InitialDelay

	for attempt := 0; attempt < action.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				lastResult.Status = ExecutionStatusCancelled
				lastResult.Error = "retry cancelled"
				return lastResult
			case <-time.After(delay):
			}

			// Exponential backoff
			delay = time.Duration(float64(delay) * retryPolicy.BackoffMultiplier)
			if delay > retryPolicy.MaxDelay {
				delay = retryPolicy.MaxDelay
			}
		}

		actionCtx, cancel := context.WithTimeout(ctx, action.Timeout)
		lastResult = we.executeAction(actionCtx, action, triggerData)
		cancel()

		if lastResult.Status == ExecutionStatusCompleted {
			break
		}
	}

	return lastResult
}

// processTemplate processes a template string with trigger data
func (we *WorkflowExecutor) processTemplate(template string, data map[string]interface{}) string {
	result := template

	// Simple template processing - replace {{key}} with values
	for key, value := range data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		if valueStr, ok := value.(string); ok {
			result = strings.ReplaceAll(result, placeholder, valueStr)
		} else {
			// Convert to JSON for complex types
			if jsonBytes, err := json.Marshal(value); err == nil {
				result = strings.ReplaceAll(result, placeholder, string(jsonBytes))
			}
		}
	}

	return result
}

// updateWorkflowStats updates workflow execution statistics
func (we *WorkflowExecutor) updateWorkflowStats(workflow *Workflow, execution *WorkflowExecution) {
	workflow.Stats.TotalExecutions++
	workflow.Stats.LastExecution = execution.EndTime

	if execution.Status == ExecutionStatusCompleted {
		workflow.Stats.SuccessfulRuns++
		workflow.Stats.LastSuccess = execution.EndTime
	} else {
		workflow.Stats.FailedRuns++
		workflow.Stats.LastFailure = execution.EndTime
	}

	// Update average execution time
	if workflow.Stats.TotalExecutions > 0 {
		totalTime := workflow.Stats.AverageExecutionTime * time.Duration(workflow.Stats.TotalExecutions-1)
		totalTime += execution.Duration
		workflow.Stats.AverageExecutionTime = totalTime / time.Duration(workflow.Stats.TotalExecutions)
	}
}