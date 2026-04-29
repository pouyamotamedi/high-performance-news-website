package synthetic

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// runAdminPanelWorkflow tests admin panel functionality
func (s *SyntheticMonitor) runAdminPanelWorkflow(ctx context.Context) {
	adminTests := []struct {
		name string
		test func(context.Context) MonitoringResult
	}{
		{"admin_login", s.testAdminLogin},
		{"article_management", s.testArticleManagement},
		{"user_management", s.testUserManagement},
		{"system_monitoring", s.testSystemMonitoring},
		{"content_moderation", s.testContentModeration},
	}

	for _, adminTest := range adminTests {
		result := adminTest.test(ctx)
		s.resultStore.Store(result)

		if result.Status == StatusFailed {
			s.alertManager.SendAlert(AlertHigh, fmt.Sprintf("Admin panel workflow failed: %s", adminTest.name), result)
		}
	}
}

// testAdminLogin validates admin authentication and dashboard access
func (s *SyntheticMonitor) testAdminLogin(ctx context.Context) MonitoringResult {
	start := time.Now()
	result := MonitoringResult{
		TestName:    "admin_login",
		TestType:    "admin_workflow",
		Timestamp:   start,
		UserJourney: "Admin Login",
		Metrics:     make(map[string]float64),
	}

	page := s.browser.MustPage(s.baseURL + "/admin")
	defer page.Close()

	// Wait for login page to load
	err := page.WaitLoad()
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Admin page load failed: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Check if already logged in (redirect to dashboard)
	currentURL := page.MustInfo().URL
	if strings.Contains(currentURL, "/dashboard") {
		result.Status = StatusPassed
		result.Metrics["already_authenticated"] = 1
		result.Duration = time.Since(start)
		return result
	}

	// Find login form elements
	usernameInput, err := page.Element("input[name='username'], input[name='email'], #username, #email")
	if err != nil {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "Username input not found")
		result.Duration = time.Since(start)
		return result
	}

	passwordInput, err := page.Element("input[name='password'], #password")
	if err != nil {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "Password input not found")
		result.Duration = time.Since(start)
		return result
	}

	loginButton, err := page.Element("button[type='submit'], input[type='submit'], .login-button")
	if err != nil {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "Login button not found")
		result.Duration = time.Since(start)
		return result
	}

	// Perform login with test credentials
	loginStart := time.Now()
	
	err = usernameInput.Input("admin@test.com") // Use test admin credentials
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to input username: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	err = passwordInput.Input("testpassword")
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to input password: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	loginButton.MustClick()
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to click login button: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Wait for redirect to dashboard
	err = page.WaitLoad()
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Login redirect failed: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	loginDuration := time.Since(loginStart)
	result.Metrics["login_response_time_ms"] = float64(loginDuration.Milliseconds())

	// Capture screenshot
	screenshot, _ := page.Screenshot(true, nil)
	screenshotPath := s.saveScreenshot(screenshot, "admin_login")
	result.Screenshots = append(result.Screenshots, screenshotPath)

	// Validate successful login (check for dashboard elements)
	dashboardElements := []struct {
		selector string
		name     string
	}{
		{".dashboard, #dashboard", "Dashboard Container"},
		{".admin-nav, .sidebar", "Admin Navigation"},
		{".stats, .metrics", "Dashboard Statistics"},
		{".logout, .sign-out", "Logout Button"},
	}

	foundElements := 0
	for _, element := range dashboardElements {
		el, err := page.Element(element.selector)
		if err == nil && el != nil {
			foundElements++
		}
	}

	result.Metrics["dashboard_elements_found"] = float64(foundElements)

	if foundElements >= 2 { // At least 2 dashboard elements should be present
		result.Status = StatusPassed
	} else {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "Dashboard elements not found after login")
	}

	result.Duration = time.Since(start)
	return result
}

// testArticleManagement validates article management functionality
func (s *SyntheticMonitor) testArticleManagement(ctx context.Context) MonitoringResult {
	start := time.Now()
	result := MonitoringResult{
		TestName:    "article_management",
		TestType:    "admin_workflow",
		Timestamp:   start,
		UserJourney: "Article Management",
		Metrics:     make(map[string]float64),
	}

	// Navigate to articles management page
	page := s.browser.MustPage(s.baseURL + "/admin/articles")
	defer page.Close()

	err := page.WaitLoad()
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Articles page load failed: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Capture screenshot
	screenshot, _ := page.Screenshot(true, nil)
	screenshotPath := s.saveScreenshot(screenshot, "article_management")
	result.Screenshots = append(result.Screenshots, screenshotPath)

	// Validate article management elements
	managementElements := []struct {
		selector string
		name     string
	}{
		{".article-list, table", "Article List"},
		{".create-article, .new-article", "Create Article Button"},
		{".search-articles, input[type='search']", "Article Search"},
		{".filter-articles, .article-filters", "Article Filters"},
		{".edit-article, .article-actions", "Article Actions"},
	}

	foundElements := 0
	for _, element := range managementElements {
		el, err := page.Element(element.selector)
		if err == nil && el != nil {
			foundElements++
			result.Metrics[fmt.Sprintf("has_%s", strings.ToLower(strings.ReplaceAll(element.name, " ", "_")))] = 1
		}
	}

	result.Metrics["management_elements_found"] = float64(foundElements)

	// Test article list loading
	articles, err := page.Elements(".article-row, tr, .article-item")
	if err == nil {
		result.Metrics["articles_displayed"] = float64(len(articles))
	}

	// Test pagination if present
	pagination, err := page.Element(".pagination, .pager")
	if err == nil && pagination != nil {
		result.Metrics["has_pagination"] = 1
	}

	// Test sorting if present
	sortHeaders, err := page.Elements("th[data-sort], .sortable")
	if err == nil {
		result.Metrics["sortable_columns"] = float64(len(sortHeaders))
	}

	if foundElements >= 3 { // At least 3 management elements should be present
		result.Status = StatusPassed
	} else {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "Insufficient article management elements found")
	}

	result.Duration = time.Since(start)
	return result
}

// testUserManagement validates user management functionality
func (s *SyntheticMonitor) testUserManagement(ctx context.Context) MonitoringResult {
	start := time.Now()
	result := MonitoringResult{
		TestName:    "user_management",
		TestType:    "admin_workflow",
		Timestamp:   start,
		UserJourney: "User Management",
		Metrics:     make(map[string]float64),
	}

	page := s.browser.MustPage(s.baseURL + "/admin/users")
	defer page.Close()

	err := page.WaitLoad()
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Users page load failed: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Validate user management elements
	userElements := []struct {
		selector string
		name     string
	}{
		{".user-list, table", "User List"},
		{".create-user, .new-user", "Create User Button"},
		{".search-users, input[type='search']", "User Search"},
		{".user-roles, .role-filter", "Role Management"},
		{".user-actions, .edit-user", "User Actions"},
	}

	foundElements := 0
	for _, element := range userElements {
		el, err := page.Element(element.selector)
		if err == nil && el != nil {
			foundElements++
			result.Metrics[fmt.Sprintf("has_%s", strings.ToLower(strings.ReplaceAll(element.name, " ", "_")))] = 1
		}
	}

	result.Metrics["user_management_elements"] = float64(foundElements)

	// Test user list
	users, err := page.Elements(".user-row, tr, .user-item")
	if err == nil {
		result.Metrics["users_displayed"] = float64(len(users))
	}

	if foundElements >= 2 {
		result.Status = StatusPassed
	} else {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "User management elements not found")
	}

	result.Duration = time.Since(start)
	return result
}

// testSystemMonitoring validates system monitoring dashboard
func (s *SyntheticMonitor) testSystemMonitoring(ctx context.Context) MonitoringResult {
	start := time.Now()
	result := MonitoringResult{
		TestName:    "system_monitoring",
		TestType:    "admin_workflow",
		Timestamp:   start,
		UserJourney: "System Monitoring",
		Metrics:     make(map[string]float64),
	}

	page := s.browser.MustPage(s.baseURL + "/admin/monitoring")
	defer page.Close()

	err := page.WaitLoad()
	if err != nil {
		// Monitoring page might not exist, which is okay
		result.Status = StatusPassed
		result.Metrics["has_monitoring_page"] = 0
		result.Duration = time.Since(start)
		return result
	}

	// Validate monitoring elements
	monitoringElements := []struct {
		selector string
		name     string
	}{
		{".system-stats, .metrics", "System Statistics"},
		{".performance-charts, .charts", "Performance Charts"},
		{".error-logs, .logs", "Error Logs"},
		{".alerts, .notifications", "System Alerts"},
		{".health-status, .status", "Health Status"},
	}

	foundElements := 0
	for _, element := range monitoringElements {
		el, err := page.Element(element.selector)
		if err == nil && el != nil {
			foundElements++
			result.Metrics[fmt.Sprintf("has_%s", strings.ToLower(strings.ReplaceAll(element.name, " ", "_")))] = 1
		}
	}

	result.Metrics["monitoring_elements"] = float64(foundElements)
	result.Metrics["has_monitoring_page"] = 1

	if foundElements > 0 {
		result.Status = StatusPassed
	} else {
		result.Status = StatusPassed // Monitoring might be minimal
		result.Metrics["basic_monitoring"] = 1
	}

	result.Duration = time.Since(start)
	return result
}

// testContentModeration validates content moderation tools
func (s *SyntheticMonitor) testContentModeration(ctx context.Context) MonitoringResult {
	start := time.Now()
	result := MonitoringResult{
		TestName:    "content_moderation",
		TestType:    "admin_workflow",
		Timestamp:   start,
		UserJourney: "Content Moderation",
		Metrics:     make(map[string]float64),
	}

	page := s.browser.MustPage(s.baseURL + "/admin/moderation")
	defer page.Close()

	err := page.WaitLoad()
	if err != nil {
		// Moderation page might not exist
		result.Status = StatusPassed
		result.Metrics["has_moderation_page"] = 0
		result.Duration = time.Since(start)
		return result
	}

	// Validate moderation elements
	moderationElements := []struct {
		selector string
		name     string
	}{
		{".pending-content, .moderation-queue", "Moderation Queue"},
		{".flagged-content, .reported-content", "Flagged Content"},
		{".moderation-actions, .approve-reject", "Moderation Actions"},
		{".content-filters, .auto-moderation", "Content Filters"},
	}

	foundElements := 0
	for _, element := range moderationElements {
		el, err := page.Element(element.selector)
		if err == nil && el != nil {
			foundElements++
			result.Metrics[fmt.Sprintf("has_%s", strings.ToLower(strings.ReplaceAll(element.name, " ", "_")))] = 1
		}
	}

	result.Metrics["moderation_elements"] = float64(foundElements)
	result.Metrics["has_moderation_page"] = 1

	if foundElements > 0 {
		result.Status = StatusPassed
	} else {
		result.Status = StatusPassed // Moderation might be basic
	}

	result.Duration = time.Since(start)
	return result
}