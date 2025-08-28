package services

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"high-performance-news-website/internal/models"
)

// TrieNode represents a node in the Trie data structure
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
	tag      *models.Tag // Associated tag for this keyword
	keyword  string      // The complete keyword
}

// Trie represents a Trie data structure for efficient keyword matching
type Trie struct {
	root *TrieNode
}

// NewTrie creates a new Trie instance
func NewTrie() *Trie {
	return &Trie{
		root: &TrieNode{
			children: make(map[rune]*TrieNode),
		},
	}
}

// Insert adds a keyword to the Trie with its associated tag
func (t *Trie) Insert(keyword string, tag *models.Tag) {
	node := t.root
	normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
	
	for _, char := range normalizedKeyword {
		if node.children[char] == nil {
			node.children[char] = &TrieNode{
				children: make(map[rune]*TrieNode),
			}
		}
		node = node.children[char]
	}
	
	node.isEnd = true
	node.tag = tag
	node.keyword = keyword // Store original keyword for display
}

// KeywordMatch represents a matched keyword with its position and tag
type KeywordMatch struct {
	Keyword   string
	Tag       *models.Tag
	StartPos  int
	EndPos    int
	Length    int
}

// FindLongestMatches finds all longest keyword matches in the given text
func (t *Trie) FindLongestMatches(text string) []KeywordMatch {
	var matches []KeywordMatch
	normalizedText := strings.ToLower(text)
	textRunes := []rune(normalizedText)
	
	for i := 0; i < len(textRunes); i++ {
		// Check if we're at a word boundary (start of word)
		if i > 0 && isWordChar(textRunes[i-1]) {
			continue
		}
		
		node := t.root
		longestMatch := KeywordMatch{}
		
		for j := i; j < len(textRunes); j++ {
			char := textRunes[j]
			
			if node.children[char] == nil {
				break
			}
			
			node = node.children[char]
			
			if node.isEnd {
				// Check if this is a complete word (word boundary after)
				if j+1 >= len(textRunes) || !isWordChar(textRunes[j+1]) {
					longestMatch = KeywordMatch{
						Keyword:  node.keyword,
						Tag:      node.tag,
						StartPos: i,
						EndPos:   j + 1,
						Length:   j - i + 1,
					}
				}
			}
		}
		
		// Add the longest match found from this position
		if longestMatch.Length > 0 {
			matches = append(matches, longestMatch)
			// Skip ahead to avoid overlapping matches
			i = longestMatch.EndPos - 1
		}
	}
	
	return matches
}

// isWordChar checks if a character is part of a word
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '-'
}

// TagRepositoryInterface defines the interface for tag repository operations
type TagRepositoryInterface interface {
	GetAllWithKeywords(ctx context.Context) ([]models.Tag, error)
}

// AutoLinkingService handles automatic internal linking based on keyword banks
type AutoLinkingService struct {
	tagRepo TagRepositoryInterface
	trie    *Trie
}

// NewAutoLinkingService creates a new auto-linking service
func NewAutoLinkingService(tagRepo TagRepositoryInterface) *AutoLinkingService {
	return &AutoLinkingService{
		tagRepo: tagRepo,
		trie:    NewTrie(),
	}
}

// LoadKeywords loads all tag keywords into the Trie for efficient matching
func (s *AutoLinkingService) LoadKeywords(ctx context.Context) error {
	// Get all tags with their keywords
	tags, err := s.tagRepo.GetAllWithKeywords(ctx)
	if err != nil {
		return fmt.Errorf("failed to load tags with keywords: %w", err)
	}
	
	// Rebuild the Trie
	s.trie = NewTrie()
	
	// Insert all keywords into the Trie
	for i := range tags {
		tag := &tags[i] // Get pointer to the actual tag, not the loop variable
		for _, keyword := range tag.Keywords {
			if strings.TrimSpace(keyword) != "" {
				s.trie.Insert(keyword, tag)
			}
		}
	}
	
	return nil
}

// ProcessArticleLinks processes article content and adds internal links based on keyword matching
func (s *AutoLinkingService) ProcessArticleLinks(ctx context.Context, article *models.Article) (string, error) {
	// Check if auto-linking is disabled for this article
	if !article.AutoLinking {
		return article.Content, nil
	}
	
	// Load keywords if Trie is empty
	if s.trie.root == nil || len(s.trie.root.children) == 0 {
		if err := s.LoadKeywords(ctx); err != nil {
			return article.Content, fmt.Errorf("failed to load keywords: %w", err)
		}
	}
	
	// Find all keyword matches in the content
	matches := s.trie.FindLongestMatches(article.Content)
	
	// Sort matches by position (descending) to avoid position shifts during replacement
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].StartPos > matches[j].StartPos
	})
	
	// Track used keywords to ensure one link per keyword per article
	usedKeywords := make(map[string]bool)
	
	// Process matches and create links
	content := article.Content
	contentRunes := []rune(content)
	
	for _, match := range matches {
		// Skip if this keyword has already been linked
		normalizedKeyword := strings.ToLower(match.Keyword)
		if usedKeywords[normalizedKeyword] {
			continue
		}
		
		// Extract the original text to preserve case
		originalText := string(contentRunes[match.StartPos:match.EndPos])
		
		// Create the link
		link := fmt.Sprintf(`<a href="/tags/%s" title="%s">%s</a>`,
			match.Tag.Slug,
			html.EscapeString(match.Tag.Description),
			html.EscapeString(originalText))
		
		// Replace the text with the link
		before := string(contentRunes[:match.StartPos])
		after := string(contentRunes[match.EndPos:])
		content = before + link + after
		
		// Update contentRunes for next iteration
		contentRunes = []rune(content)
		
		// Mark this keyword as used
		usedKeywords[normalizedKeyword] = true
	}
	
	return content, nil
}

// ProcessArticleLinksWithExclusions processes article content with keyword exclusions
func (s *AutoLinkingService) ProcessArticleLinksWithExclusions(ctx context.Context, article *models.Article, excludeKeywords []string) (string, error) {
	// Check if auto-linking is disabled for this article
	if !article.AutoLinking {
		return article.Content, nil
	}
	
	// Load keywords if Trie is empty
	if s.trie.root == nil || len(s.trie.root.children) == 0 {
		if err := s.LoadKeywords(ctx); err != nil {
			return article.Content, fmt.Errorf("failed to load keywords: %w", err)
		}
	}
	
	// Create exclusion map for fast lookup
	exclusions := make(map[string]bool)
	for _, keyword := range excludeKeywords {
		exclusions[strings.ToLower(strings.TrimSpace(keyword))] = true
	}
	
	// Find all keyword matches in the content
	matches := s.trie.FindLongestMatches(article.Content)
	
	// Filter out excluded keywords
	var filteredMatches []KeywordMatch
	for _, match := range matches {
		if !exclusions[strings.ToLower(match.Keyword)] {
			filteredMatches = append(filteredMatches, match)
		}
	}
	
	// Sort matches by position (descending) to avoid position shifts during replacement
	sort.Slice(filteredMatches, func(i, j int) bool {
		return filteredMatches[i].StartPos > filteredMatches[j].StartPos
	})
	
	// Track used keywords to ensure one link per keyword per article
	usedKeywords := make(map[string]bool)
	
	// Process matches and create links
	content := article.Content
	contentRunes := []rune(content)
	
	for _, match := range filteredMatches {
		// Skip if this keyword has already been linked
		normalizedKeyword := strings.ToLower(match.Keyword)
		if usedKeywords[normalizedKeyword] {
			continue
		}
		
		// Extract the original text to preserve case
		originalText := string(contentRunes[match.StartPos:match.EndPos])
		
		// Create the link
		link := fmt.Sprintf(`<a href="/tags/%s" title="%s">%s</a>`,
			match.Tag.Slug,
			html.EscapeString(match.Tag.Description),
			html.EscapeString(originalText))
		
		// Replace the text with the link
		before := string(contentRunes[:match.StartPos])
		after := string(contentRunes[match.EndPos:])
		content = before + link + after
		
		// Update contentRunes for next iteration
		contentRunes = []rune(content)
		
		// Mark this keyword as used
		usedKeywords[normalizedKeyword] = true
	}
	
	return content, nil
}

// GetKeywordMatches returns all keyword matches in the given text without creating links
func (s *AutoLinkingService) GetKeywordMatches(ctx context.Context, text string) ([]KeywordMatch, error) {
	// Load keywords if Trie is empty
	if s.trie.root == nil || len(s.trie.root.children) == 0 {
		if err := s.LoadKeywords(ctx); err != nil {
			return nil, fmt.Errorf("failed to load keywords: %w", err)
		}
	}
	
	return s.trie.FindLongestMatches(text), nil
}

// RefreshKeywords reloads all keywords from the database
func (s *AutoLinkingService) RefreshKeywords(ctx context.Context) error {
	return s.LoadKeywords(ctx)
}

// ValidateKeywordConflicts checks for keyword conflicts across tags
func (s *AutoLinkingService) ValidateKeywordConflicts(ctx context.Context) ([]string, error) {
	tags, err := s.tagRepo.GetAllWithKeywords(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load tags: %w", err)
	}
	
	keywordToTags := make(map[string][]string)
	
	// Build keyword to tags mapping
	for _, tag := range tags {
		for _, keyword := range tag.Keywords {
			normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
			if normalizedKeyword != "" {
				keywordToTags[normalizedKeyword] = append(keywordToTags[normalizedKeyword], tag.Name)
			}
		}
	}
	
	// Find conflicts
	var conflicts []string
	for keyword, tagNames := range keywordToTags {
		if len(tagNames) > 1 {
			conflicts = append(conflicts, fmt.Sprintf("Keyword '%s' is used in tags: %s", 
				keyword, strings.Join(tagNames, ", ")))
		}
	}
	
	return conflicts, nil
}

// GetTrieStats returns statistics about the loaded Trie
func (s *AutoLinkingService) GetTrieStats() map[string]int {
	if s.trie == nil || s.trie.root == nil || len(s.trie.root.children) == 0 {
		return map[string]int{
			"total_nodes":    0,
			"total_keywords": 0,
		}
	}
	
	totalNodes := 0
	totalKeywords := 0
	
	var countNodes func(*TrieNode)
	countNodes = func(node *TrieNode) {
		totalNodes++
		if node.isEnd {
			totalKeywords++
		}
		for _, child := range node.children {
			countNodes(child)
		}
	}
	
	countNodes(s.trie.root)
	
	return map[string]int{
		"total_nodes":    totalNodes,
		"total_keywords": totalKeywords,
	}
}

// ProcessHTMLContent processes HTML content while preserving existing links and HTML tags
func (s *AutoLinkingService) ProcessHTMLContent(ctx context.Context, article *models.Article) (string, error) {
	// Check if auto-linking is disabled for this article
	if !article.AutoLinking {
		return article.Content, nil
	}
	
	// Load keywords if Trie is empty
	if s.trie.root == nil || len(s.trie.root.children) == 0 {
		if err := s.LoadKeywords(ctx); err != nil {
			return article.Content, fmt.Errorf("failed to load keywords: %w", err)
		}
	}
	
	content := article.Content
	
	// Find all HTML tags and existing links to avoid processing them
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	linkRegex := regexp.MustCompile(`(?i)<a[^>]*>.*?</a>`)
	
	htmlTags := htmlTagRegex.FindAllStringIndex(content, -1)
	existingLinks := linkRegex.FindAllStringIndex(content, -1)
	
	// Combine and sort all protected ranges
	var protectedRanges [][2]int
	for _, tag := range htmlTags {
		protectedRanges = append(protectedRanges, [2]int{tag[0], tag[1]})
	}
	for _, link := range existingLinks {
		protectedRanges = append(protectedRanges, [2]int{link[0], link[1]})
	}
	
	sort.Slice(protectedRanges, func(i, j int) bool {
		return protectedRanges[i][0] < protectedRanges[j][0]
	})
	
	// Merge overlapping ranges
	var mergedRanges [][2]int
	for _, currentRange := range protectedRanges {
		if len(mergedRanges) == 0 {
			mergedRanges = append(mergedRanges, currentRange)
		} else {
			lastRange := &mergedRanges[len(mergedRanges)-1]
			if currentRange[0] <= lastRange[1] {
				// Overlapping ranges, merge them
				if currentRange[1] > lastRange[1] {
					lastRange[1] = currentRange[1]
				}
			} else {
				// Non-overlapping range, add it
				mergedRanges = append(mergedRanges, currentRange)
			}
		}
	}
	
	protectedRanges = mergedRanges
	
	// Process text segments that are not protected
	var result strings.Builder
	lastEnd := 0
	
	for _, protectedRange := range protectedRanges {
		start, end := protectedRange[0], protectedRange[1]
		
		// Process text before this protected range
		if start > lastEnd {
			textSegment := content[lastEnd:start]
			processedSegment := s.processTextSegment(textSegment)
			result.WriteString(processedSegment)
		}
		
		// Add the protected range as-is
		result.WriteString(content[start:end])
		lastEnd = end
	}
	
	// Process remaining text after the last protected range
	if lastEnd < len(content) {
		textSegment := content[lastEnd:]
		processedSegment := s.processTextSegment(textSegment)
		result.WriteString(processedSegment)
	}
	
	return result.String(), nil
}

// processTextSegment processes a text segment for keyword matching
func (s *AutoLinkingService) processTextSegment(text string) string {
	matches := s.trie.FindLongestMatches(text)
	
	// Sort matches by position (descending) to avoid position shifts during replacement
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].StartPos > matches[j].StartPos
	})
	
	// Track used keywords to ensure one link per keyword per segment
	usedKeywords := make(map[string]bool)
	
	// Process matches and create links
	content := text
	contentRunes := []rune(content)
	
	for _, match := range matches {
		// Skip if this keyword has already been linked
		normalizedKeyword := strings.ToLower(match.Keyword)
		if usedKeywords[normalizedKeyword] {
			continue
		}
		
		// Extract the original text to preserve case
		originalText := string(contentRunes[match.StartPos:match.EndPos])
		
		// Create the link
		link := fmt.Sprintf(`<a href="/tags/%s" title="%s">%s</a>`,
			match.Tag.Slug,
			html.EscapeString(match.Tag.Description),
			html.EscapeString(originalText))
		
		// Replace the text with the link
		before := string(contentRunes[:match.StartPos])
		after := string(contentRunes[match.EndPos:])
		content = before + link + after
		
		// Update contentRunes for next iteration
		contentRunes = []rune(content)
		
		// Mark this keyword as used
		usedKeywords[normalizedKeyword] = true
	}
	
	return content
}