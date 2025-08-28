package services

import (
	"encoding/json"
	"fmt"
	"html/template"

	"high-performance-news-website/internal/models"
)

// BreadcrumbService handles breadcrumb navigation generation
type BreadcrumbService struct {
	baseURL  string
	siteName string
}

// NewBreadcrumbService creates a new breadcrumb service instance
func NewBreadcrumbService(baseURL, siteName string) *BreadcrumbService {
	return &BreadcrumbService{
		baseURL:  baseURL,
		siteName: siteName,
	}
}

// BreadcrumbItem represents a single breadcrumb item
type BreadcrumbItem struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Active   bool   `json:"active"`
}

// BreadcrumbList represents a complete breadcrumb navigation
type BreadcrumbList struct {
	Items  []BreadcrumbItem `json:"items"`
	Schema template.HTML    `json:"schema"`
}

// BreadcrumbListLD represents structured data for breadcrumbs
type BreadcrumbListLD struct {
	Context         string                    `json:"@context"`
	Type            string                    `json:"@type"`
	ItemListElement []BreadcrumbItemElement  `json:"itemListElement"`
}

// BreadcrumbItemElement represents a breadcrumb item in structured data
type BreadcrumbItemElement struct {
	Type     string `json:"@type"`
	Position int    `json:"position"`
	Name     string `json:"name"`
	Item     string `json:"item"`
}

// GenerateArticleBreadcrumbs creates breadcrumbs for article pages
func (bs *BreadcrumbService) GenerateArticleBreadcrumbs(article *models.Article, category *models.Category) (*BreadcrumbList, error) {
	if article == nil {
		return nil, fmt.Errorf("article cannot be nil")
	}

	items := []BreadcrumbItem{
		{
			Name:     "Home",
			URL:      bs.baseURL,
			Position: 1,
			Active:   false,
		},
	}

	position := 2

	// Add category if available
	if category != nil {
		// Add categories index page
		items = append(items, BreadcrumbItem{
			Name:     "Categories",
			URL:      bs.baseURL + "/categories",
			Position: position,
			Active:   false,
		})
		position++

		// Add parent categories if hierarchical
		if category.Parent != nil {
			parentItems := bs.buildCategoryHierarchy(category.Parent, position)
			items = append(items, parentItems...)
			position += len(parentItems)
		}

		// Add current category
		items = append(items, BreadcrumbItem{
			Name:     category.Name,
			URL:      bs.GetCategoryURL(category.Slug),
			Position: position,
			Active:   false,
		})
		position++
	}

	// Add current article (always last and active)
	items = append(items, BreadcrumbItem{
		Name:     article.Title,
		URL:      bs.GetArticleURL(article.Slug),
		Position: position,
		Active:   true,
	})

	// Generate structured data
	schema, err := bs.generateStructuredData(items)
	if err != nil {
		return nil, fmt.Errorf("failed to generate breadcrumb schema: %w", err)
	}

	return &BreadcrumbList{
		Items:  items,
		Schema: schema,
	}, nil
}

// GenerateCategoryBreadcrumbs creates breadcrumbs for category pages
func (bs *BreadcrumbService) GenerateCategoryBreadcrumbs(category *models.Category) (*BreadcrumbList, error) {
	if category == nil {
		return nil, fmt.Errorf("category cannot be nil")
	}

	items := []BreadcrumbItem{
		{
			Name:     "Home",
			URL:      bs.baseURL,
			Position: 1,
			Active:   false,
		},
		{
			Name:     "Categories",
			URL:      bs.baseURL + "/categories",
			Position: 2,
			Active:   false,
		},
	}

	position := 3

	// Add parent categories if hierarchical
	if category.Parent != nil {
		parentItems := bs.buildCategoryHierarchy(category.Parent, position)
		items = append(items, parentItems...)
		position += len(parentItems)
	}

	// Add current category (active)
	items = append(items, BreadcrumbItem{
		Name:     category.Name,
		URL:      bs.GetCategoryURL(category.Slug),
		Position: position,
		Active:   true,
	})

	// Generate structured data
	schema, err := bs.generateStructuredData(items)
	if err != nil {
		return nil, fmt.Errorf("failed to generate breadcrumb schema: %w", err)
	}

	return &BreadcrumbList{
		Items:  items,
		Schema: schema,
	}, nil
}

// GenerateTagBreadcrumbs creates breadcrumbs for tag pages
func (bs *BreadcrumbService) GenerateTagBreadcrumbs(tag *models.Tag) (*BreadcrumbList, error) {
	if tag == nil {
		return nil, fmt.Errorf("tag cannot be nil")
	}

	items := []BreadcrumbItem{
		{
			Name:     "Home",
			URL:      bs.baseURL,
			Position: 1,
			Active:   false,
		},
		{
			Name:     "Tags",
			URL:      bs.baseURL + "/tags",
			Position: 2,
			Active:   false,
		},
		{
			Name:     tag.Name,
			URL:      bs.GetTagURL(tag.Slug),
			Position: 3,
			Active:   true,
		},
	}

	// Generate structured data
	schema, err := bs.generateStructuredData(items)
	if err != nil {
		return nil, fmt.Errorf("failed to generate breadcrumb schema: %w", err)
	}

	return &BreadcrumbList{
		Items:  items,
		Schema: schema,
	}, nil
}

// GenerateSearchBreadcrumbs creates breadcrumbs for search pages
func (bs *BreadcrumbService) GenerateSearchBreadcrumbs(query string) (*BreadcrumbList, error) {
	items := []BreadcrumbItem{
		{
			Name:     "Home",
			URL:      bs.baseURL,
			Position: 1,
			Active:   false,
		},
	}

	searchTitle := "Search Results"
	if query != "" {
		searchTitle = fmt.Sprintf("Search: %s", query)
	}

	items = append(items, BreadcrumbItem{
		Name:     searchTitle,
		URL:      bs.baseURL + "/search",
		Position: 2,
		Active:   true,
	})

	// Generate structured data
	schema, err := bs.generateStructuredData(items)
	if err != nil {
		return nil, fmt.Errorf("failed to generate breadcrumb schema: %w", err)
	}

	return &BreadcrumbList{
		Items:  items,
		Schema: schema,
	}, nil
}

// GenerateCustomBreadcrumbs creates breadcrumbs for custom pages
func (bs *BreadcrumbService) GenerateCustomBreadcrumbs(pageName, pageURL string, parentItems ...BreadcrumbItem) (*BreadcrumbList, error) {
	items := []BreadcrumbItem{
		{
			Name:     "Home",
			URL:      bs.baseURL,
			Position: 1,
			Active:   false,
		},
	}

	position := 2

	// Add parent items if provided
	for _, parent := range parentItems {
		parent.Position = position
		parent.Active = false
		items = append(items, parent)
		position++
	}

	// Add current page
	items = append(items, BreadcrumbItem{
		Name:     pageName,
		URL:      pageURL,
		Position: position,
		Active:   true,
	})

	// Generate structured data
	schema, err := bs.generateStructuredData(items)
	if err != nil {
		return nil, fmt.Errorf("failed to generate breadcrumb schema: %w", err)
	}

	return &BreadcrumbList{
		Items:  items,
		Schema: schema,
	}, nil
}

// buildCategoryHierarchy recursively builds breadcrumb items for category hierarchy
func (bs *BreadcrumbService) buildCategoryHierarchy(category *models.Category, startPosition int) []BreadcrumbItem {
	var items []BreadcrumbItem

	if category.Parent != nil {
		// Recursively add parent categories first
		parentItems := bs.buildCategoryHierarchy(category.Parent, startPosition)
		items = append(items, parentItems...)
		startPosition += len(parentItems)
	}

	// Add current category
	items = append(items, BreadcrumbItem{
		Name:     category.Name,
		URL:      bs.GetCategoryURL(category.Slug),
		Position: startPosition,
		Active:   false,
	})

	return items
}

// generateStructuredData creates JSON-LD structured data for breadcrumbs
func (bs *BreadcrumbService) generateStructuredData(items []BreadcrumbItem) (template.HTML, error) {
	var elements []BreadcrumbItemElement

	for _, item := range items {
		elements = append(elements, BreadcrumbItemElement{
			Type:     "ListItem",
			Position: item.Position,
			Name:     item.Name,
			Item:     item.URL,
		})
	}

	breadcrumbLD := BreadcrumbListLD{
		Context:         "https://schema.org",
		Type:            "BreadcrumbList",
		ItemListElement: elements,
	}

	jsonBytes, err := json.MarshalIndent(breadcrumbLD, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal breadcrumb schema: %w", err)
	}

	jsonLD := fmt.Sprintf(`<script type="application/ld+json">
%s
</script>`, string(jsonBytes))

	return template.HTML(jsonLD), nil
}

// RenderBreadcrumbHTML generates HTML for breadcrumb navigation with enhanced accessibility
func (bs *BreadcrumbService) RenderBreadcrumbHTML(breadcrumbs *BreadcrumbList, cssClass string) template.HTML {
	if breadcrumbs == nil || len(breadcrumbs.Items) == 0 {
		return ""
	}

	if cssClass == "" {
		cssClass = "breadcrumb"
	}

	html := fmt.Sprintf(`<nav class="%s" aria-label="Breadcrumb navigation" role="navigation">
  <ol class="breadcrumb-list" itemscope itemtype="https://schema.org/BreadcrumbList">`, cssClass)

	for i, item := range breadcrumbs.Items {
		itemClass := "breadcrumb-item"
		if item.Active {
			itemClass += " active"
		}

		if item.Active {
			// Active item (current page) - no link but with microdata
			html += fmt.Sprintf(`
    <li class="%s" aria-current="page" itemprop="itemListElement" itemscope itemtype="https://schema.org/ListItem">
      <span itemprop="name">%s</span>
      <meta itemprop="position" content="%d">
    </li>`, itemClass, template.HTMLEscapeString(item.Name), item.Position)
		} else {
			// Regular item with link and microdata
			html += fmt.Sprintf(`
    <li class="%s" itemprop="itemListElement" itemscope itemtype="https://schema.org/ListItem">
      <a href="%s" itemprop="item">
        <span itemprop="name">%s</span>
      </a>
      <meta itemprop="position" content="%d">
    </li>`, itemClass, template.HTMLEscapeString(item.URL), template.HTMLEscapeString(item.Name), item.Position)
		}

		// Add separator (except for last item)
		if i < len(breadcrumbs.Items)-1 {
			html += `
    <li class="breadcrumb-separator" aria-hidden="true">
      <span>/</span>
    </li>`
		}
	}

	html += `
  </ol>
</nav>`

	return template.HTML(html)
}

// RenderBreadcrumbJSON generates JSON-LD structured data for breadcrumbs
func (bs *BreadcrumbService) RenderBreadcrumbJSON(breadcrumbs *BreadcrumbList) (template.HTML, error) {
	if breadcrumbs == nil || len(breadcrumbs.Items) == 0 {
		return "", nil
	}

	var elements []BreadcrumbItemElement
	for _, item := range breadcrumbs.Items {
		elements = append(elements, BreadcrumbItemElement{
			Type:     "ListItem",
			Position: item.Position,
			Name:     item.Name,
			Item:     item.URL,
		})
	}

	breadcrumbLD := BreadcrumbListLD{
		Context:         "https://schema.org",
		Type:            "BreadcrumbList",
		ItemListElement: elements,
	}

	jsonBytes, err := json.MarshalIndent(breadcrumbLD, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal breadcrumb schema: %w", err)
	}

	jsonLD := fmt.Sprintf(`<script type="application/ld+json">
%s
</script>`, string(jsonBytes))

	return template.HTML(jsonLD), nil
}

// GetBreadcrumbsForPath generates breadcrumbs based on URL path
func (bs *BreadcrumbService) GetBreadcrumbsForPath(path string, data interface{}) (*BreadcrumbList, error) {
	switch {
	case path == "/" || path == "":
		// Homepage - no breadcrumbs needed
		return &BreadcrumbList{Items: []BreadcrumbItem{}}, nil

	case path == "/categories":
		return bs.GenerateCustomBreadcrumbs("Categories", bs.baseURL+"/categories")

	case path == "/tags":
		return bs.GenerateCustomBreadcrumbs("Tags", bs.baseURL+"/tags")

	case path == "/trending":
		return bs.GenerateCustomBreadcrumbs("Trending", bs.baseURL+"/trending")

	case path == "/latest":
		return bs.GenerateCustomBreadcrumbs("Latest", bs.baseURL+"/latest")

	case path == "/popular":
		return bs.GenerateCustomBreadcrumbs("Popular", bs.baseURL+"/popular")

	case path == "/about":
		return bs.GenerateCustomBreadcrumbs("About", bs.baseURL+"/about")

	case path == "/contact":
		return bs.GenerateCustomBreadcrumbs("Contact", bs.baseURL+"/contact")

	case path == "/search":
		query := ""
		if queryData, ok := data.(map[string]interface{}); ok {
			if q, exists := queryData["query"]; exists {
				query = fmt.Sprintf("%v", q)
			}
		}
		return bs.GenerateSearchBreadcrumbs(query)

	default:
		// For dynamic pages, data should contain the necessary information
		return bs.GenerateCustomBreadcrumbs("Page", path)
	}
}

// URL generation helpers (matching other services)
func (bs *BreadcrumbService) GetArticleURL(slug string) string {
	return fmt.Sprintf("%s/article/%s", bs.baseURL, slug)
}

func (bs *BreadcrumbService) GetCategoryURL(slug string) string {
	return fmt.Sprintf("%s/category/%s", bs.baseURL, slug)
}

func (bs *BreadcrumbService) GetTagURL(slug string) string {
	return fmt.Sprintf("%s/tag/%s", bs.baseURL, slug)
}

// ValidateBreadcrumbs ensures breadcrumb data is valid
func (bs *BreadcrumbService) ValidateBreadcrumbs(breadcrumbs *BreadcrumbList) error {
	if breadcrumbs == nil {
		return fmt.Errorf("breadcrumbs cannot be nil")
	}

	if len(breadcrumbs.Items) == 0 {
		return nil // Empty breadcrumbs are valid
	}

	// Check that positions are sequential
	for i, item := range breadcrumbs.Items {
		expectedPosition := i + 1
		if item.Position != expectedPosition {
			return fmt.Errorf("breadcrumb item %d has position %d, expected %d", i, item.Position, expectedPosition)
		}

		// Check that names are not empty
		if item.Name == "" {
			return fmt.Errorf("breadcrumb item %d has empty name", i)
		}

		// Check that URLs are not empty (except for active items which might not need URLs)
		if item.URL == "" && !item.Active {
			return fmt.Errorf("breadcrumb item %d has empty URL", i)
		}
	}

	// Check that only the last item is active
	activeCount := 0
	lastActiveIndex := -1
	for i, item := range breadcrumbs.Items {
		if item.Active {
			activeCount++
			lastActiveIndex = i
		}
	}

	if activeCount > 1 {
		return fmt.Errorf("multiple active breadcrumb items found, only one allowed")
	}

	if activeCount == 1 && lastActiveIndex != len(breadcrumbs.Items)-1 {
		return fmt.Errorf("active breadcrumb item must be the last item")
	}

	return nil
}