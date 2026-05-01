package templates

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"time"

	"high-performance-news-website/internal/services"
)

// ResponsiveImageData holds all data needed for responsive image rendering
// This mirrors the struct in static_generator.go for template compatibility
type ResponsiveImageData struct {
	ThumbnailWebP string
	ThumbnailJPEG string
	SmallWebP     string
	SmallJPEG     string
	MediumWebP    string
	MediumJPEG    string
	LargeWebP     string
	LargeJPEG     string
	Width         int
	Height        int
	LQIP          string
	AltText       string
	HasVariants   bool
}

// TemplateEngine handles template rendering with layout inheritance
type TemplateEngine struct {
	templates         map[string]*template.Template
	funcMap           template.FuncMap
	devMode           bool
	templateDir       string
	seoService        *services.SEOService
	breadcrumbService *services.BreadcrumbService
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(devMode bool) *TemplateEngine {
	engine := &TemplateEngine{
		templates: make(map[string]*template.Template),
		devMode:   devMode,
	}
	engine.funcMap = engine.createFuncMap()
	return engine
}

// SetSEOServices sets the SEO-related services for template functions
func (te *TemplateEngine) SetSEOServices(seoService *services.SEOService, breadcrumbService *services.BreadcrumbService) {
	te.seoService = seoService
	te.breadcrumbService = breadcrumbService
	// Recreate function map with SEO functions
	te.funcMap = te.createFuncMap()
}

// LoadTemplates loads all templates from the templates directory
func (te *TemplateEngine) LoadTemplates(templatesDir string) error {
	te.templateDir = templatesDir
	// Load layout templates first
	layoutsPattern := filepath.Join(templatesDir, "layouts", "*.html")
	layouts, err := filepath.Glob(layoutsPattern)
	if err != nil {
		return fmt.Errorf("failed to load layouts: %w", err)
	}

	// Load component templates
	componentsPattern := filepath.Join(templatesDir, "components", "*.html")
	components, err := filepath.Glob(componentsPattern)
	if err != nil {
		return fmt.Errorf("failed to load components: %w", err)
	}

	// Load page templates
	pagesPattern := filepath.Join(templatesDir, "pages", "*.html")
	pages, err := filepath.Glob(pagesPattern)
	if err != nil {
		return fmt.Errorf("failed to load pages: %w", err)
	}

	// Create templates for each page with layouts and components
	for _, page := range pages {
		pageName := strings.TrimSuffix(filepath.Base(page), ".html")
		
		// Combine all template files
		allFiles := append([]string{page}, layouts...)
		allFiles = append(allFiles, components...)
		
		tmpl, err := template.New(pageName).Funcs(te.funcMap).ParseFiles(allFiles...)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", pageName, err)
		}
		
		te.templates[pageName] = tmpl
	}

	return nil
}

// Render renders a template with the given data
func (te *TemplateEngine) Render(templateName string, data interface{}) (string, error) {
	// In development mode, reload templates on each render
	if te.devMode {
		// Use the same directory that was originally loaded
		// For tests, we need to track the template directory
		if err := te.LoadTemplates(te.templateDir); err != nil {
			return "", fmt.Errorf("failed to reload templates: %w", err)
		}
	}

	tmpl, exists := te.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var buf strings.Builder
	// Try to execute the template with the template name + .html first
	templateDefName := templateName + ".html"
	if err := tmpl.ExecuteTemplate(&buf, templateDefName, data); err != nil {
		// If that fails, try with just the template name
		if err2 := tmpl.ExecuteTemplate(&buf, templateName, data); err2 != nil {
			// If both fail, try executing the main template
			if err3 := tmpl.Execute(&buf, data); err3 != nil {
				return "", fmt.Errorf("failed to execute template %s: primary error: %w, secondary error: %v, tertiary error: %v", templateName, err, err2, err3)
			}
		}
	}

	return buf.String(), nil
}

// createFuncMap creates template functions
func (te *TemplateEngine) createFuncMap() template.FuncMap {
	return template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},
		"safeJS": func(s string) template.JS {
			return template.JS(s)
		},
		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"formatDate": func(t time.Time) string {
			return t.Format("January 2, 2006")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("January 2, 2006 at 3:04 PM")
		},
		"formatTime": func(t time.Time) string {
			return t.Format("3:04 PM")
		},
		"timeAgo": func(t time.Time) string {
			duration := time.Since(t)
			if duration < time.Minute {
				return "just now"
			} else if duration < time.Hour {
				minutes := int(duration.Minutes())
				return fmt.Sprintf("%d minute%s ago", minutes, pluralize(minutes))
			} else if duration < 24*time.Hour {
				hours := int(duration.Hours())
				return fmt.Sprintf("%d hour%s ago", hours, pluralize(hours))
			} else {
				days := int(duration.Hours() / 24)
				return fmt.Sprintf("%d day%s ago", days, pluralize(days))
			}
		},
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"multiply": func(a, b int) int {
			return a * b
		},
		"divide": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"le": func(a, b int) bool {
			return a <= b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"ge": func(a, b int) bool {
			return a >= b
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"containsSlice": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},
		"iterate": func(count int) []int {
			result := make([]int, count)
			for i := 0; i < count; i++ {
				result[i] = i
			}
			return result
		},
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(s, suffix)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"title": func(s string) string {
			return strings.Title(s)
		},
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
		"default": func(defaultValue, value interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
		"seq": func(start, end int) []int {
			if start > end {
				return []int{}
			}
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},
		"len": func(v interface{}) int {
			switch s := v.(type) {
			case []interface{}:
				return len(s)
			case []string:
				return len(s)
			case string:
				return len(s)
			default:
				return 0
			}
		},
		// imageVariant generates a URL for a specific image size/format variant
		// Usage: {{imageVariant .ImageURL "640" "webp"}} or {{imageVariant .ImageURL "1024" ""}}
		"imageVariant": func(src string, width string, format string) string {
			if src == "" {
				return ""
			}
			// If the image is already a variant URL or external, return as-is
			if strings.Contains(src, "?") || strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
				return src
			}
			// Extract file extension
			ext := filepath.Ext(src)
			basePath := strings.TrimSuffix(src, ext)
			
			// Determine output format
			outputExt := ext
			if format != "" {
				outputExt = "." + format
			}
			
			// Generate variant URL: /uploads/images/article-w640.webp
			return fmt.Sprintf("%s-w%s%s", basePath, width, outputExt)
		},
		// dict creates a map from key-value pairs for template use
		// Usage: {{template "responsive-image" (dict "src" .ImageURL "alt" .Title "class" "hero-img")}}
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		// hasResponsiveImage checks if the imageData has valid responsive variants
		"hasResponsiveImage": func(imageData interface{}) bool {
			if imageData == nil {
				return false
			}
			switch v := imageData.(type) {
			case *ResponsiveImageData:
				return v != nil && v.HasVariants
			case *services.ResponsiveImageData:
				return v != nil && v.HasVariants
			default:
				return false
			}
		},
		// Responsive image helper - generates <picture> element with WebP and JPEG sources
		"responsiveImage": func(imageData interface{}, lazyLoad bool, cssClass string) template.HTML {
			// Handle both *ResponsiveImageData and *services.ResponsiveImageData
			var data *ResponsiveImageData
			switch v := imageData.(type) {
			case *ResponsiveImageData:
				data = v
			case *services.ResponsiveImageData:
				if v != nil {
					data = &ResponsiveImageData{
						ThumbnailWebP: v.ThumbnailWebP,
						ThumbnailJPEG: v.ThumbnailJPEG,
						SmallWebP:     v.SmallWebP,
						SmallJPEG:     v.SmallJPEG,
						MediumWebP:    v.MediumWebP,
						MediumJPEG:    v.MediumJPEG,
						LargeWebP:     v.LargeWebP,
						LargeJPEG:     v.LargeJPEG,
						Width:         v.Width,
						Height:        v.Height,
						LQIP:          v.LQIP,
						AltText:       v.AltText,
						HasVariants:   v.HasVariants,
					}
				}
			default:
				return template.HTML("")
			}
			
			if data == nil || !data.HasVariants {
				return template.HTML("")
			}
			
			var html strings.Builder
			
			// Wrapper div for LQIP effect (only for lazy loaded images)
			if lazyLoad && data.LQIP != "" {
				html.WriteString(`<div class="responsive-image-wrapper" style="position: relative; overflow: hidden;">`)
				html.WriteString(fmt.Sprintf(`<div class="lqip-placeholder" style="position: absolute; inset: 0; background-image: url('%s'); background-size: cover; filter: blur(20px); transform: scale(1.1); transition: opacity 0.3s;"></div>`, data.LQIP))
			}
			
			html.WriteString("<picture>")
			
			// WebP sources (modern browsers)
			webpSrcset := []string{}
			if data.SmallWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 300w", data.SmallWebP))
			}
			if data.MediumWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 600w", data.MediumWebP))
			}
			if data.LargeWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 1200w", data.LargeWebP))
			}
			if len(webpSrcset) > 0 {
				html.WriteString(fmt.Sprintf(`<source type="image/webp" srcset="%s" sizes="(max-width: 300px) 300px, (max-width: 600px) 600px, (max-width: 1200px) 1200px, 100vw">`, strings.Join(webpSrcset, ", ")))
			}
			
			// JPEG sources (fallback)
			jpegSrcset := []string{}
			if data.SmallJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 300w", data.SmallJPEG))
			}
			if data.MediumJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 600w", data.MediumJPEG))
			}
			if data.LargeJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 1200w", data.LargeJPEG))
			}
			if len(jpegSrcset) > 0 {
				html.WriteString(fmt.Sprintf(`<source type="image/jpeg" srcset="%s" sizes="(max-width: 300px) 300px, (max-width: 600px) 600px, (max-width: 1200px) 1200px, 100vw">`, strings.Join(jpegSrcset, ", ")))
			}
			
			// Fallback img element
			fallbackSrc := data.MediumJPEG
			if fallbackSrc == "" {
				fallbackSrc = data.LargeJPEG
			}
			
			imgAttrs := []string{
				fmt.Sprintf(`src="%s"`, fallbackSrc),
				fmt.Sprintf(`alt="%s"`, data.AltText),
				`decoding="async"`,
				`sizes="(max-width: 300px) 300px, (max-width: 600px) 600px, (max-width: 1200px) 1200px, 100vw"`,
			}
			
			if data.Width > 0 {
				imgAttrs = append(imgAttrs, fmt.Sprintf(`width="%d"`, data.Width))
			}
			if data.Height > 0 {
				imgAttrs = append(imgAttrs, fmt.Sprintf(`height="%d"`, data.Height))
			}
			if cssClass != "" {
				imgAttrs = append(imgAttrs, fmt.Sprintf(`class="%s"`, cssClass))
			}
			
			if lazyLoad {
				imgAttrs = append(imgAttrs, `loading="lazy"`)
				if data.LQIP != "" {
					imgAttrs = append(imgAttrs, `onload="this.parentElement.parentElement.querySelector('.lqip-placeholder')?.remove()"`)
					imgAttrs = append(imgAttrs, `style="position: relative; z-index: 1;"`)
				}
			} else {
				imgAttrs = append(imgAttrs, `loading="eager"`, `fetchpriority="high"`)
			}
			
			html.WriteString(fmt.Sprintf("<img %s>", strings.Join(imgAttrs, " ")))
			html.WriteString("</picture>")
			
			// Close wrapper div
			if lazyLoad && data.LQIP != "" {
				html.WriteString("</div>")
			}
			
			return template.HTML(html.String())
		},
		// Simple responsive image without LQIP wrapper (for thumbnails/cards)
		"responsiveImageSimple": func(imageData interface{}, lazyLoad bool, cssClass string) template.HTML {
			// Handle both *ResponsiveImageData and *services.ResponsiveImageData
			var data *ResponsiveImageData
			switch v := imageData.(type) {
			case *ResponsiveImageData:
				data = v
			case *services.ResponsiveImageData:
				if v != nil {
					data = &ResponsiveImageData{
						ThumbnailWebP: v.ThumbnailWebP,
						ThumbnailJPEG: v.ThumbnailJPEG,
						SmallWebP:     v.SmallWebP,
						SmallJPEG:     v.SmallJPEG,
						MediumWebP:    v.MediumWebP,
						MediumJPEG:    v.MediumJPEG,
						LargeWebP:     v.LargeWebP,
						LargeJPEG:     v.LargeJPEG,
						Width:         v.Width,
						Height:        v.Height,
						LQIP:          v.LQIP,
						AltText:       v.AltText,
						HasVariants:   v.HasVariants,
					}
				}
			default:
				return template.HTML("")
			}
			
			if data == nil || !data.HasVariants {
				return template.HTML("")
			}
			
			var html strings.Builder
			html.WriteString("<picture>")
			
			// WebP sources
			webpSrcset := []string{}
			if data.ThumbnailWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 150w", data.ThumbnailWebP))
			}
			if data.SmallWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 300w", data.SmallWebP))
			}
			if data.MediumWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 600w", data.MediumWebP))
			}
			if len(webpSrcset) > 0 {
				html.WriteString(fmt.Sprintf(`<source type="image/webp" srcset="%s" sizes="(max-width: 150px) 150px, (max-width: 300px) 300px, 600px">`, strings.Join(webpSrcset, ", ")))
			}
			
			// JPEG sources
			jpegSrcset := []string{}
			if data.ThumbnailJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 150w", data.ThumbnailJPEG))
			}
			if data.SmallJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 300w", data.SmallJPEG))
			}
			if data.MediumJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 600w", data.MediumJPEG))
			}
			if len(jpegSrcset) > 0 {
				html.WriteString(fmt.Sprintf(`<source type="image/jpeg" srcset="%s" sizes="(max-width: 150px) 150px, (max-width: 300px) 300px, 600px">`, strings.Join(jpegSrcset, ", ")))
			}
			
			// Fallback img
			fallbackSrc := data.SmallJPEG
			if fallbackSrc == "" {
				fallbackSrc = data.MediumJPEG
			}
			
			loadingAttr := "lazy"
			if !lazyLoad {
				loadingAttr = "eager"
			}
			
			html.WriteString(fmt.Sprintf(`<img src="%s" alt="%s" class="%s" loading="%s" decoding="async">`, 
				fallbackSrc, data.AltText, cssClass, loadingAttr))
			html.WriteString("</picture>")
			
			return template.HTML(html.String())
		},
		// SEO-related template functions
		"renderMetaTags": te.renderMetaTags,
		"renderSchema": te.renderSchema,
		"renderBreadcrumbs": te.renderBreadcrumbs,
		"generateCanonicalURL": te.generateCanonicalURL,
		"generateOGTags": te.generateOGTags,
		"generateTwitterTags": te.generateTwitterTags,
		// Translation function for multilingual support
		"t": func(lang string, key string) string {
			return Translate(lang, key)
		},
	}
}

// pluralize adds 's' for plural forms
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// SEO template functions

// renderMetaTags generates HTML meta tags for SEO
func (te *TemplateEngine) renderMetaTags(pageType string, data interface{}) template.HTML {
	if te.seoService == nil {
		return ""
	}

	meta, err := te.seoService.GenerateMetaTags(pageType, data)
	if err != nil {
		return template.HTML("<!-- Error generating meta tags -->")
	}

	html := fmt.Sprintf(`<title>%s</title>
<meta name="description" content="%s">`, 
		template.HTMLEscapeString(meta.Title),
		template.HTMLEscapeString(meta.Description))

	if meta.Keywords != "" {
		html += fmt.Sprintf(`
<meta name="keywords" content="%s">`, template.HTMLEscapeString(meta.Keywords))
	}

	if meta.Author != "" {
		html += fmt.Sprintf(`
<meta name="author" content="%s">`, template.HTMLEscapeString(meta.Author))
	}

	if meta.CanonicalURL != "" {
		html += fmt.Sprintf(`
<link rel="canonical" href="%s">`, template.HTMLEscapeString(meta.CanonicalURL))
	}

	html += fmt.Sprintf(`
<meta name="robots" content="%s">
<meta name="language" content="%s">`, meta.Robots, meta.Language)

	return template.HTML(html)
}

// renderSchema generates JSON-LD structured data
func (te *TemplateEngine) renderSchema(pageType string, data interface{}) template.HTML {
	if te.seoService == nil {
		return ""
	}

	var schema interface{}
	var err error

	switch pageType {
	case "article":
		if _, ok := data.(map[string]interface{}); ok {
			// Convert map to article struct - simplified for now
			schema, err = te.seoService.GenerateHomepageSchema() // Placeholder
		}
	case "homepage":
		schema, err = te.seoService.GenerateHomepageSchema()
	default:
		return template.HTML("<!-- Unknown page type for schema -->")
	}

	if err != nil {
		return template.HTML("<!-- Error generating schema -->")
	}

	jsonLD, err := te.seoService.RenderSchemaJSON(schema)
	if err != nil {
		return template.HTML("<!-- Error rendering schema JSON -->")
	}

	return jsonLD
}

// renderBreadcrumbs generates breadcrumb navigation HTML
func (te *TemplateEngine) renderBreadcrumbs(pageType string, data interface{}) template.HTML {
	if te.breadcrumbService == nil {
		return ""
	}

	var breadcrumbs *services.BreadcrumbList
	var err error

	switch pageType {
	case "search":
		query := ""
		if searchData, ok := data.(map[string]interface{}); ok {
			if q, exists := searchData["Query"]; exists {
				query = fmt.Sprintf("%v", q)
			}
		}
		breadcrumbs, err = te.breadcrumbService.GenerateSearchBreadcrumbs(query)
	default:
		breadcrumbs, err = te.breadcrumbService.GenerateCustomBreadcrumbs("Page", "/")
	}

	if err != nil {
		return template.HTML("<!-- Error generating breadcrumbs -->")
	}

	return te.breadcrumbService.RenderBreadcrumbHTML(breadcrumbs, "breadcrumb")
}

// generateCanonicalURL generates canonical URL for the current page
func (te *TemplateEngine) generateCanonicalURL(pageType, slug string) string {
	if te.seoService == nil {
		return ""
	}

	switch pageType {
	case "article":
		return te.seoService.GetArticleURL(slug)
	case "category":
		return te.seoService.GetCategoryURL(slug)
	case "tag":
		return te.seoService.GetTagURL(slug)
	default:
		return ""
	}
}

// generateOGTags generates Open Graph meta tags
func (te *TemplateEngine) generateOGTags(pageType string, data interface{}) template.HTML {
	if te.seoService == nil {
		return ""
	}

	meta, err := te.seoService.GenerateMetaTags(pageType, data)
	if err != nil {
		return template.HTML("<!-- Error generating OG tags -->")
	}

	html := fmt.Sprintf(`<meta property="og:title" content="%s">
<meta property="og:description" content="%s">
<meta property="og:type" content="%s">
<meta property="og:url" content="%s">`,
		template.HTMLEscapeString(meta.OGTitle),
		template.HTMLEscapeString(meta.OGDescription),
		template.HTMLEscapeString(meta.OGType),
		template.HTMLEscapeString(meta.OGURL))

	if meta.OGImage != "" {
		html += fmt.Sprintf(`
<meta property="og:image" content="%s">`, template.HTMLEscapeString(meta.OGImage))
	}

	return template.HTML(html)
}

// generateTwitterTags generates Twitter Card meta tags
func (te *TemplateEngine) generateTwitterTags(pageType string, data interface{}) template.HTML {
	if te.seoService == nil {
		return ""
	}

	meta, err := te.seoService.GenerateMetaTags(pageType, data)
	if err != nil {
		return template.HTML("<!-- Error generating Twitter tags -->")
	}

	html := fmt.Sprintf(`<meta name="twitter:card" content="%s">
<meta name="twitter:title" content="%s">
<meta name="twitter:description" content="%s">`,
		template.HTMLEscapeString(meta.TwitterCard),
		template.HTMLEscapeString(meta.TwitterTitle),
		template.HTMLEscapeString(meta.TwitterDescription))

	if meta.TwitterImage != "" {
		html += fmt.Sprintf(`
<meta name="twitter:image" content="%s">`, template.HTMLEscapeString(meta.TwitterImage))
	}

	return template.HTML(html)
}