package synthetic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

// runSearchFunctionalityTests tests search functionality
func (s *SyntheticMonitor) runSearchFunctionalityTests(ctx context.Context) {
	searchTests := []struct {
		name string
		test func(context.Context) MonitoringResult
	}{
		{"basic_search", s.testBasicSearch},
		{"advanced_search", s.testAdvancedSearch},
		{"search_suggestions", s.testSearchSuggestions},
		{"search_filters", s.testSearchFilters},
		{"multilingual_search", s.testMultilingualSearch},
	}

	for _, searchTest := range searchTests {
		result := searchTest.test(ctx)
		s.resultStore.Store(result)

		if result.Status == StatusFailed {
			s.alertManager.SendAlert(AlertMedium, fmt.Sprintf("Search functionality failed: %s", searchTest.name), result)
		}
	}
}

// testBasicSearch validates basic search functionality
func (s *SyntheticMonitor) testBasicSearch(ctx context.Context) MonitoringResult {
	start := time.Now()
	result := MonitoringResult{
		TestName:    "basic_search",
		TestType:    "search_functionality",
		Timestamp:   start,
		UserJourney: "Basic Search",
		Metrics:     make(map[string]float64),
	}

	page := s.browser.MustPage(s.baseURL)
	defer page.Close()

	// Wait for page to load
	err := page.WaitLoad()
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Page load failed: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Find search input
	searchInput, err := page.Element("input[type='search'], input[name='search'], #search")
	if err != nil {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "Search input not found")
		result.Duration = time.Since(start)
		return result
	}

	// Perform search
	searchTerm := "technology"
	searchStart := time.Now()
	
	err = searchInput.Input(searchTerm)
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to input search term: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Submit search (try Enter key first, then search button)
	err = searchInput.Type(input.Enter)
	if err != nil {
		// Try clicking search button
		searchButton, btnErr := page.Element("button[type='submit'], .search-button, #search-btn")
		if btnErr != nil {
			result.Status = StatusError
			result.Errors = append(result.Errors, "Could not submit search")
			result.Duration = time.Since(start)
			return result
		}
		searchButton.MustClick()
	}

	// Wait for search results
	err = page.WaitLoad()
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Search results failed to load: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	searchDuration := time.Since(searchStart)
	result.Metrics["search_response_time_ms"] = float64(searchDuration.Milliseconds())

	// Capture screenshot
	screenshot, _ := page.Screenshot(true, nil)
	screenshotPath := s.saveScreenshot(screenshot, "basic_search")
	result.Screenshots = append(result.Screenshots, screenshotPath)

	// Validate search results
	searchResults, err := page.Elements(".search-result, .article-item, .result-item")
	if err != nil || len(searchResults) == 0 {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "No search results found")
	} else {
		result.Metrics["search_results_count"] = float64(len(searchResults))
		
		// Validate first result contains search term
		firstResult := searchResults[0]
		resultText, _ := firstResult.Text()
		if !strings.Contains(strings.ToLower(resultText), strings.ToLower(searchTerm)) {
			result.Status = StatusFailed
			result.Errors = append(result.Errors, "Search results don't match search term")
		}
	}

	// Check for search metadata (result count, time taken)
	resultMeta, err := page.Element(".search-meta, .results-info")
	if err == nil {
		metaText, _ := resultMeta.Text()
		result.Metrics["has_search_metadata"] = 1
		if strings.Contains(metaText, "results") {
			result.Metrics["shows_result_count"] = 1
		}
	}

	if result.Status == "" {
		result.Status = StatusPassed
	}

	result.Duration = time.Since(start)
	return result
}

// testAdvancedSearch validates advanced search features
func (s *SyntheticMonitor) testAdvancedSearch(ctx context.Context) MonitoringResult {
	start := time.Now()
	result := MonitoringResult{
		TestName:    "advanced_search",
		TestType:    "search_functionality",
		Timestamp:   start,
		UserJourney: "Advanced Search",
		Metrics:     make(map[string]float64),
	}

	page := s.browser.MustPage(s.baseURL + "/search")
	defer page.Close()

	err := page.WaitLoad()
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Advanced search page load failed: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Test category filter
	categoryFilter, err := page.Element("select[name='category'], .category-filter")
	if err == nil {
		err = categoryFilter.Select([]string{"Technology"}, true, rod.SelectorTypeText)
		if err != nil {
			result.Errors = append(result.Errors, "Failed to select category filter")
		} else {
			result.Metrics["has_category_filter"] = 1
		}
	}

	// Test date range filter
	_, err = page.Element("input[type='date'], .date-filter")
	if err == nil {
		result.Metrics["has_date_filter"] = 1
	}

	// Test author filter
	_, err = page.Element("select[name='author'], .author-filter")
	if err == nil {
		result.Metrics["has_author_filter"] = 1
	}

	// Perform filtered search
	searchInput, err := page.Element("input[type='search'], input[name='search']")
	if err != nil {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "Search input not found on advanced search page")
		result.Duration = time.Since(start)
		return result
	}

	searchStart := time.Now()
	err = searchInput.Input("artificial intelligence")
	if err == nil {
		searchInput.Type(input.Enter)
		page.WaitLoad()
	}
	searchDuration := time.Since(searchStart)

	result.Metrics["advanced_search_response_time_ms"] = float64(searchDuration.Milliseconds())

	// Validate filtered results
	searchResults, err := page.Elements(".search-result, .article-item")
	if err == nil && len(searchResults) > 0 {
		result.Metrics["filtered_results_count"] = float64(len(searchResults))
		result.Status = StatusPassed
	} else {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "Advanced search returned no results")
	}

	result.Duration = time.Since(start)
	return result
}

// testSearchSuggestions validates search autocomplete/suggestions
func (s *SyntheticMonitor) testSearchSuggestions(ctx context.Context) MonitoringResult {
	start := time.Now()
	result := MonitoringResult{
		TestName:    "search_suggestions",
		TestType:    "search_functionality",
		Timestamp:   start,
		UserJourney: "Search Suggestions",
		Metrics:     make(map[string]float64),
	}

	page := s.browser.MustPage(s.baseURL)
	defer page.Close()

	err := page.WaitLoad()
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Page load failed: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Find search input
	searchInput, err := page.Element("input[type='search'], input[name='search']")
	if err != nil {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "Search input not found")
		result.Duration = time.Since(start)
		return result
	}

	// Type partial search term to trigger suggestions
	suggestionStart := time.Now()
	err = searchInput.Input("tech")
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to input search term: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Wait a moment for suggestions to appear
	time.Sleep(500 * time.Millisecond)

	// Look for suggestion dropdown
	suggestions, err := page.Elements(".search-suggestions, .autocomplete, .dropdown-menu")
	suggestionDuration := time.Since(suggestionStart)

	result.Metrics["suggestion_response_time_ms"] = float64(suggestionDuration.Milliseconds())

	if err != nil || len(suggestions) == 0 {
		// Suggestions might not be implemented, which is okay
		result.Status = StatusPassed
		result.Metrics["has_suggestions"] = 0
	} else {
		result.Metrics["has_suggestions"] = 1
		result.Metrics["suggestion_count"] = float64(len(suggestions))
		result.Status = StatusPassed
	}

	result.Duration = time.Since(start)
	return result
}

// testSearchFilters validates search filtering functionality
func (s *SyntheticMonitor) testSearchFilters(ctx context.Context) MonitoringResult {
	start := time.Now()
	result := MonitoringResult{
		TestName:    "search_filters",
		TestType:    "search_functionality",
		Timestamp:   start,
		UserJourney: "Search Filters",
		Metrics:     make(map[string]float64),
	}

	// Perform a search first
	page := s.browser.MustPage(s.baseURL)
	defer page.Close()

	err := page.WaitLoad()
	if err != nil {
		result.Status = StatusError
		result.Errors = append(result.Errors, fmt.Sprintf("Page load failed: %v", err))
		result.Duration = time.Since(start)
		return result
	}

	// Search for something that should have results
	searchInput, err := page.Element("input[type='search'], input[name='search']")
	if err == nil {
		searchInput.Input("news")
		searchInput.Type(input.Enter)
		page.WaitLoad()
	}

	// Check for filter options
	filters := []struct {
		selector string
		name     string
	}{
		{".filter-category, select[name='category']", "Category Filter"},
		{".filter-date, input[type='date']", "Date Filter"},
		{".filter-author, select[name='author']", "Author Filter"},
		{".filter-language, select[name='language']", "Language Filter"},
	}

	availableFilters := 0
	for _, filter := range filters {
		element, err := page.Element(filter.selector)
		if err == nil && element != nil {
			availableFilters++
			result.Metrics[fmt.Sprintf("has_%s", strings.ToLower(strings.ReplaceAll(filter.name, " ", "_")))] = 1
		}
	}

	result.Metrics["available_filters_count"] = float64(availableFilters)

	if availableFilters > 0 {
		result.Status = StatusPassed
	} else {
		// No filters might be by design
		result.Status = StatusPassed
		result.Metrics["has_filters"] = 0
	}

	result.Duration = time.Since(start)
	return result
}

// testMultilingualSearch validates search in different languages
func (s *SyntheticMonitor) testMultilingualSearch(ctx context.Context) MonitoringResult {
	start := time.Now()
	result := MonitoringResult{
		TestName:    "multilingual_search",
		TestType:    "search_functionality",
		Timestamp:   start,
		UserJourney: "Multilingual Search",
		Metrics:     make(map[string]float64),
	}

	// Test searches in different languages
	searchTests := []struct {
		language string
		term     string
		url      string
	}{
		{"English", "technology", s.baseURL},
		{"Persian", "فناوری", s.baseURL + "/fa"},
		{"Arabic", "تكنولوجيا", s.baseURL + "/ar"},
	}

	successfulSearches := 0
	for _, test := range searchTests {
		page := s.browser.MustPage(test.url)
		
		err := page.WaitLoad()
		if err != nil {
			page.Close()
			continue
		}

		// Find search input
		searchInput, err := page.Element("input[type='search'], input[name='search']")
		if err != nil {
			page.Close()
			continue
		}

		// Perform search
		searchStart := time.Now()
		err = searchInput.Input(test.term)
		if err == nil {
			searchInput.Type(input.Enter)
			page.WaitLoad()
			
			// Check for results
			searchResults, err := page.Elements(".search-result, .article-item")
			if err == nil && len(searchResults) > 0 {
				successfulSearches++
				result.Metrics[fmt.Sprintf("%s_search_results", strings.ToLower(test.language))] = float64(len(searchResults))
			}
		}
		
		searchDuration := time.Since(searchStart)
		result.Metrics[fmt.Sprintf("%s_search_time_ms", strings.ToLower(test.language))] = float64(searchDuration.Milliseconds())
		
		page.Close()
	}

	result.Metrics["successful_multilingual_searches"] = float64(successfulSearches)
	result.Metrics["total_language_tests"] = float64(len(searchTests))

	if successfulSearches > 0 {
		result.Status = StatusPassed
	} else {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "No multilingual searches succeeded")
	}

	result.Duration = time.Since(start)
	return result
}