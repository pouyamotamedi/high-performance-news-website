package services

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// SlugService handles slug generation with transliteration for SEO-friendly URLs
type SlugService struct{}

// NewSlugService creates a new SlugService
func NewSlugService() *SlugService {
	return &SlugService{}
}

// Arabic to Latin transliteration map
var arabicToLatin = map[rune]string{
	'ا': "a", 'أ': "a", 'إ': "e", 'آ': "a",
	'ب': "b", 'ت': "t", 'ث': "th",
	'ج': "j", 'ح': "h", 'خ': "kh",
	'د': "d", 'ذ': "dh",
	'ر': "r", 'ز': "z",
	'س': "s", 'ش': "sh",
	'ص': "s", 'ض': "d",
	'ط': "t", 'ظ': "z",
	'ع': "a", 'غ': "gh",
	'ف': "f", 'ق': "q",
	'ك': "k", 'ک': "k",
	'ل': "l", 'م': "m", 'ن': "n",
	'ه': "h", 'ة': "h",
	'و': "w", 'ؤ': "w",
	'ي': "y", 'ى': "y", 'ئ': "y",
	'ء': "",
	// Arabic numbers
	'٠': "0", '١': "1", '٢': "2", '٣': "3", '٤': "4",
	'٥': "5", '٦': "6", '٧': "7", '٨': "8", '٩': "9",
	// Common Arabic words - short vowels are usually omitted
	'َ': "a", 'ُ': "u", 'ِ': "i", 'ً': "", 'ٌ': "", 'ٍ': "",
	'ّ': "", 'ْ': "",
}

// German special characters
var germanToLatin = map[rune]string{
	'ä': "ae", 'Ä': "ae",
	'ö': "oe", 'Ö': "oe",
	'ü': "ue", 'Ü': "ue",
	'ß': "ss",
}

// French special characters
var frenchToLatin = map[rune]string{
	'à': "a", 'â': "a", 'æ': "ae",
	'ç': "c",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e",
	'î': "i", 'ï': "i",
	'ô': "o", 'œ': "oe",
	'ù': "u", 'û': "u", 'ü': "u",
	'ÿ': "y",
}

// Spanish special characters
var spanishToLatin = map[rune]string{
	'á': "a", 'é': "e", 'í': "i", 'ó': "o", 'ú': "u",
	'ñ': "n", 'Ñ': "n",
	'ü': "u",
}

// GenerateSlug creates an SEO-friendly slug from any text
// It handles Arabic, German, French, Spanish and other languages
func (s *SlugService) GenerateSlug(text string, languageCode string) string {
	if text == "" {
		return ""
	}

	// Normalize Unicode (NFC form)
	text = norm.NFC.String(text)

	// Convert to lowercase
	text = strings.ToLower(text)

	// Apply language-specific transliteration first
	var result strings.Builder
	for _, r := range text {
		// Check Arabic
		if latin, ok := arabicToLatin[r]; ok {
			result.WriteString(latin)
			continue
		}
		// Check German
		if latin, ok := germanToLatin[r]; ok {
			result.WriteString(latin)
			continue
		}
		// Check French
		if latin, ok := frenchToLatin[r]; ok {
			result.WriteString(latin)
			continue
		}
		// Check Spanish
		if latin, ok := spanishToLatin[r]; ok {
			result.WriteString(latin)
			continue
		}
		result.WriteRune(r)
	}
	text = result.String()

	// Remove diacritics from remaining characters using Unicode normalization
	text = removeDiacritics(text)

	// Replace spaces and underscores with hyphens
	text = strings.ReplaceAll(text, " ", "-")
	text = strings.ReplaceAll(text, "_", "-")

	// Remove any character that's not a-z, 0-9, or hyphen
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	text = reg.ReplaceAllString(text, "")

	// Replace multiple hyphens with single hyphen
	reg = regexp.MustCompile(`-+`)
	text = reg.ReplaceAllString(text, "-")

	// Trim hyphens from start and end
	text = strings.Trim(text, "-")

	// Limit length to 100 characters for SEO
	if len(text) > 100 {
		text = text[:100]
		// Don't end with a hyphen
		text = strings.TrimRight(text, "-")
	}

	return text
}

// removeDiacritics removes accent marks from characters
func removeDiacritics(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

// IsValidASCIISlug checks if a slug contains only ASCII characters
func (s *SlugService) IsValidASCIISlug(slug string) bool {
	if slug == "" {
		return false
	}

	// Check for valid slug pattern: lowercase letters, numbers, hyphens
	// No consecutive hyphens, no leading/trailing hyphens
	slugRegex := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	return slugRegex.MatchString(slug)
}

// SuggestSlug generates a slug suggestion for non-English text
// If the input is already a valid ASCII slug, returns it as-is
func (s *SlugService) SuggestSlug(text string, languageCode string) string {
	// If already a valid ASCII slug, return as-is
	if s.IsValidASCIISlug(text) {
		return text
	}

	// Generate transliterated slug
	return s.GenerateSlug(text, languageCode)
}
