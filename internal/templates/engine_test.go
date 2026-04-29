package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewTemplateEngine(t *testing.T) {
	engine := NewTemplateEngine(true)
	
	if engine == nil {
		t.Fatal("Expected template engine to be created")
	}
	
	if engine.devMode != true {
		t.Error("Expected dev mode to be true")
	}
	
	if engine.templates == nil {
		t.Error("Expected templates map to be initialized")
	}
	
	if engine.funcMap == nil {
		t.Error("Expected funcMap to be initialized")
	}
}

func TestTemplateFunctions(t *testing.T) {
	engine := NewTemplateEngine(false)
	
	tests := []struct {
		name     string
		function string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "truncate function",
			function: "truncate",
			input:    map[string]interface{}{"text": "Hello World", "length": 5},
			expected: "Hello...",
		},
		{
			name:     "join function",
			function: "join",
			input:    map[string]interface{}{"sep": ", ", "items": []string{"a", "b", "c"}},
			expected: "a, b, c",
		},
		{
			name:     "upper function",
			function: "upper",
			input:    "hello",
			expected: "HELLO",
		},
		{
			name:     "lower function",
			function: "lower",
			input:    "HELLO",
			expected: "hello",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := engine.funcMap[tt.function]
			if !exists {
				t.Fatalf("Function %s not found in funcMap", tt.function)
			}
			
			// Test the function based on its signature
			switch tt.function {
			case "truncate":
				args := tt.input.(map[string]interface{})
				result := fn.(func(string, int) string)(args["text"].(string), args["length"].(int))
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			case "join":
				args := tt.input.(map[string]interface{})
				result := fn.(func(string, []string) string)(args["sep"].(string), args["items"].([]string))
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			case "upper", "lower":
				result := fn.(func(string) string)(tt.input.(string))
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestTimeAgoFunction(t *testing.T) {
	engine := NewTemplateEngine(false)
	timeAgoFn := engine.funcMap["timeAgo"].(func(time.Time) string)
	
	now := time.Now()
	
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "2 minutes ago",
			time:     now.Add(-2 * time.Minute),
			expected: "2 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "2 hours ago",
			time:     now.Add(-2 * time.Hour),
			expected: "2 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "2 days ago",
			time:     now.Add(-48 * time.Hour),
			expected: "2 days ago",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeAgoFn(tt.time)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMathFunctions(t *testing.T) {
	engine := NewTemplateEngine(false)
	
	tests := []struct {
		name     string
		function string
		a, b     int
		expected int
	}{
		{"add", "add", 5, 3, 8},
		{"subtract", "subtract", 5, 3, 2},
		{"multiply", "multiply", 5, 3, 15},
		{"divide", "divide", 6, 3, 2},
		{"divide by zero", "divide", 5, 0, 0},
		{"mod", "mod", 7, 3, 1},
		{"mod by zero", "mod", 5, 0, 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := engine.funcMap[tt.function].(func(int, int) int)
			result := fn(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestComparisonFunctions(t *testing.T) {
	engine := NewTemplateEngine(false)
	
	tests := []struct {
		name     string
		function string
		a, b     interface{}
		expected bool
	}{
		{"eq true", "eq", 5, 5, true},
		{"eq false", "eq", 5, 3, false},
		{"ne true", "ne", 5, 3, true},
		{"ne false", "ne", 5, 5, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.function {
			case "eq":
				fn := engine.funcMap[tt.function].(func(interface{}, interface{}) bool)
				result := fn(tt.a, tt.b)
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			case "ne":
				fn := engine.funcMap[tt.function].(func(interface{}, interface{}) bool)
				result := fn(tt.a, tt.b)
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestIntComparisonFunctions(t *testing.T) {
	engine := NewTemplateEngine(false)
	
	tests := []struct {
		name     string
		function string
		a, b     int
		expected bool
	}{
		{"lt true", "lt", 3, 5, true},
		{"lt false", "lt", 5, 3, false},
		{"le true equal", "le", 5, 5, true},
		{"le true less", "le", 3, 5, true},
		{"le false", "le", 5, 3, false},
		{"gt true", "gt", 5, 3, true},
		{"gt false", "gt", 3, 5, false},
		{"ge true equal", "ge", 5, 5, true},
		{"ge true greater", "ge", 5, 3, true},
		{"ge false", "ge", 3, 5, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := engine.funcMap[tt.function].(func(int, int) bool)
			result := fn(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestStringFunctions(t *testing.T) {
	engine := NewTemplateEngine(false)
	
	tests := []struct {
		name     string
		function string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "contains true",
			function: "contains",
			input:    map[string]interface{}{"s": "hello world", "substr": "world"},
			expected: true,
		},
		{
			name:     "contains false",
			function: "contains",
			input:    map[string]interface{}{"s": "hello world", "substr": "foo"},
			expected: false,
		},
		{
			name:     "hasPrefix true",
			function: "hasPrefix",
			input:    map[string]interface{}{"s": "hello world", "prefix": "hello"},
			expected: true,
		},
		{
			name:     "hasPrefix false",
			function: "hasPrefix",
			input:    map[string]interface{}{"s": "hello world", "prefix": "world"},
			expected: false,
		},
		{
			name:     "hasSuffix true",
			function: "hasSuffix",
			input:    map[string]interface{}{"s": "hello world", "suffix": "world"},
			expected: true,
		},
		{
			name:     "hasSuffix false",
			function: "hasSuffix",
			input:    map[string]interface{}{"s": "hello world", "suffix": "hello"},
			expected: false,
		},
		{
			name:     "replace",
			function: "replace",
			input:    map[string]interface{}{"s": "hello world", "old": "world", "new": "Go"},
			expected: "hello Go",
		},
		{
			name:     "split",
			function: "split",
			input:    map[string]interface{}{"s": "a,b,c", "sep": ","},
			expected: []string{"a", "b", "c"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := engine.funcMap[tt.function]
			if !exists {
				t.Fatalf("Function %s not found", tt.function)
			}
			
			switch tt.function {
			case "contains", "hasPrefix", "hasSuffix":
				args := tt.input.(map[string]interface{})
				var result bool
				switch tt.function {
				case "contains":
					result = fn.(func(string, string) bool)(args["s"].(string), args["substr"].(string))
				case "hasPrefix":
					result = fn.(func(string, string) bool)(args["s"].(string), args["prefix"].(string))
				case "hasSuffix":
					result = fn.(func(string, string) bool)(args["s"].(string), args["suffix"].(string))
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			case "replace":
				args := tt.input.(map[string]interface{})
				result := fn.(func(string, string, string) string)(args["s"].(string), args["old"].(string), args["new"].(string))
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			case "split":
				args := tt.input.(map[string]interface{})
				result := fn.(func(string, string) []string)(args["s"].(string), args["sep"].(string))
				expected := tt.expected.([]string)
				if len(result) != len(expected) {
					t.Errorf("Expected length %d, got %d", len(expected), len(result))
					return
				}
				for i, v := range result {
					if v != expected[i] {
						t.Errorf("Expected %v, got %v at index %d", expected[i], v, i)
					}
				}
			}
		})
	}
}

func TestDefaultFunction(t *testing.T) {
	engine := NewTemplateEngine(false)
	defaultFn := engine.funcMap["default"].(func(interface{}, interface{}) interface{})
	
	tests := []struct {
		name         string
		defaultValue interface{}
		value        interface{}
		expected     interface{}
	}{
		{"nil value", "default", nil, "default"},
		{"empty string", "default", "", "default"},
		{"valid string", "default", "actual", "actual"},
		{"zero int", "default", 0, 0}, // 0 is not considered empty
		{"valid int", "default", 42, 42},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultFn(tt.defaultValue, tt.value)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLoadTemplatesError(t *testing.T) {
	engine := NewTemplateEngine(false)
	
	// Test with non-existent directory - this should not error as glob returns empty slice
	_ = engine.LoadTemplates("/non/existent/path")
	// Note: filepath.Glob doesn't return error for non-existent paths, just empty results
	// So we expect no error here, but no templates loaded
	if len(engine.templates) > 0 {
		t.Error("Expected no templates to be loaded from non-existent directory")
	}
	t.Logf("Templates loaded: %d", len(engine.templates))
}

func TestRenderNonExistentTemplate(t *testing.T) {
	engine := NewTemplateEngine(false)
	
	_, err := engine.Render("non-existent", nil)
	if err == nil {
		t.Error("Expected error when rendering non-existent template")
	}
	
	if !strings.Contains(err.Error(), "template non-existent not found") {
		t.Errorf("Expected 'template not found' error, got: %v", err)
	}
}

// Helper function to create temporary template files for testing
func createTempTemplateDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "templates_test")
	if err != nil {
		t.Fatal(err)
	}
	
	// Create directory structure
	layoutsDir := filepath.Join(tempDir, "layouts")
	componentsDir := filepath.Join(tempDir, "components")
	pagesDir := filepath.Join(tempDir, "pages")
	
	os.MkdirAll(layoutsDir, 0755)
	os.MkdirAll(componentsDir, 0755)
	os.MkdirAll(pagesDir, 0755)
	
	// Create base layout
	baseLayout := `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body>{{block "content" .}}{{end}}</body>
</html>`
	
	err = os.WriteFile(filepath.Join(layoutsDir, "base.html"), []byte(baseLayout), 0644)
	if err != nil {
		t.Fatal(err)
	}
	
	// Create test page that uses the base template
	testPage := `<h1>{{.Title}}</h1>
<p>{{.Content}}</p>`
	
	err = os.WriteFile(filepath.Join(pagesDir, "test.html"), []byte(testPage), 0644)
	if err != nil {
		t.Fatal(err)
	}
	
	return tempDir
}

func TestLoadAndRenderTemplate(t *testing.T) {
	tempDir := createTempTemplateDir(t)
	defer os.RemoveAll(tempDir)
	
	engine := NewTemplateEngine(false)
	err := engine.LoadTemplates(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	
	data := struct {
		Title   string
		Content string
	}{
		Title:   "Test Page",
		Content: "This is test content",
	}
	
	// The template name should match the page file name
	result, err := engine.Render("test", data)
	if err != nil {
		t.Fatal(err)
	}
	
	if !strings.Contains(result, "Test Page") {
		t.Error("Expected rendered template to contain title")
	}
	
	if !strings.Contains(result, "This is test content") {
		t.Error("Expected rendered template to contain content")
	}
}

func TestDevModeReloading(t *testing.T) {
	tempDir := createTempTemplateDir(t)
	defer os.RemoveAll(tempDir)
	
	engine := NewTemplateEngine(true) // Dev mode enabled
	err := engine.LoadTemplates(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	
	data := struct {
		Title   string
		Content string
	}{
		Title:   "Test Page",
		Content: "Original content",
	}
	
	// First render
	result1, err := engine.Render("test", data)
	if err != nil {
		t.Fatal(err)
	}
	
	if !strings.Contains(result1, "Original content") {
		t.Error("Expected first render to contain original content")
	}
	
	// Modify template file
	newTestPage := `<h1>{{.Title}}</h1>
<p>Modified: {{.Content}}</p>`
	
	err = os.WriteFile(filepath.Join(tempDir, "pages", "test.html"), []byte(newTestPage), 0644)
	if err != nil {
		t.Fatal(err)
	}
	
	// Second render (should reload in dev mode)
	result2, err := engine.Render("test", data)
	if err != nil {
		t.Fatal(err)
	}
	
	if !strings.Contains(result2, "Modified: Original content") {
		t.Error("Expected second render to contain modified content")
	}
}