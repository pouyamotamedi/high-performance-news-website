package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Language represents a supported language in the system
type Language struct {
	Code       string    `json:"code" db:"code"`
	Name       string    `json:"name" db:"name"`
	NativeName string    `json:"native_name" db:"native_name"`
	Direction  string    `json:"direction" db:"direction"` // "ltr" or "rtl"
	IsActive   bool      `json:"is_active" db:"is_active"`
	SortOrder  int       `json:"sort_order" db:"sort_order"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// TranslationGroup represents a group of translated content
type TranslationGroup struct {
	ID        uint64    `json:"id" db:"id"`
	GroupType string    `json:"group_type" db:"group_type"` // "article", "category", "tag"
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Translation represents a single translation within a group
type Translation struct {
	ID                 uint64 `json:"id"`
	Title              string `json:"title,omitempty"`
	Name               string `json:"name,omitempty"` // For categories and tags
	Slug               string `json:"slug"`
	LanguageCode       string `json:"language_code"`
	LanguageName       string `json:"language_name"`
	LanguageNativeName string `json:"language_native_name"`
}

// Translations is a slice of Translation that implements database scanning
type Translations []Translation

// Scan implements the sql.Scanner interface for database scanning
func (t *Translations) Scan(value interface{}) error {
	if value == nil {
		*t = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, t)
	case string:
		return json.Unmarshal([]byte(v), t)
	default:
		return fmt.Errorf("cannot scan %T into Translations", value)
	}
}

// Value implements the driver.Valuer interface for database storage
func (t Translations) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil
	}
	return json.Marshal(t)
}

// MultilingualArticle extends Article with translation information
type MultilingualArticle struct {
	Article
	LanguageCode         string       `json:"language_code" db:"language_code"`
	TranslationGroupID   *uint64      `json:"translation_group_id" db:"translation_group_id"`
	LanguageName         string       `json:"language_name" db:"language_name"`
	LanguageNativeName   string       `json:"language_native_name" db:"language_native_name"`
	LanguageDirection    string       `json:"language_direction" db:"language_direction"`
	Translations         Translations `json:"translations" db:"translations"`
	IsFallback           bool         `json:"is_fallback" db:"is_fallback"`
}

// MultilingualCategory extends Category with translation information
type MultilingualCategory struct {
	Category
	LanguageCode         string       `json:"language_code" db:"language_code"`
	TranslationGroupID   *uint64      `json:"translation_group_id" db:"translation_group_id"`
	LanguageName         string       `json:"language_name" db:"language_name"`
	LanguageNativeName   string       `json:"language_native_name" db:"language_native_name"`
	LanguageDirection    string       `json:"language_direction" db:"language_direction"`
	Translations         Translations `json:"translations" db:"translations"`
}

// MultilingualTag extends Tag with translation information
type MultilingualTag struct {
	Tag
	LanguageCode         string       `json:"language_code" db:"language_code"`
	TranslationGroupID   *uint64      `json:"translation_group_id" db:"translation_group_id"`
	LanguageName         string       `json:"language_name" db:"language_name"`
	LanguageNativeName   string       `json:"language_native_name" db:"language_native_name"`
	LanguageDirection    string       `json:"language_direction" db:"language_direction"`
	Translations         Translations `json:"translations" db:"translations"`
}

// LanguageConfig holds configuration for language-aware operations
type LanguageConfig struct {
	DefaultLanguage  string   `json:"default_language"`
	FallbackLanguage string   `json:"fallback_language"`
	ActiveLanguages  []string `json:"active_languages"`
	RTLLanguages     []string `json:"rtl_languages"`
}

// IsRTL returns true if the language is right-to-left
func (lc *LanguageConfig) IsRTL(languageCode string) bool {
	for _, rtl := range lc.RTLLanguages {
		if rtl == languageCode {
			return true
		}
	}
	return false
}

// IsActive returns true if the language is active
func (lc *LanguageConfig) IsActive(languageCode string) bool {
	for _, active := range lc.ActiveLanguages {
		if active == languageCode {
			return true
		}
	}
	return false
}

// GetFallbackLanguage returns the fallback language or default if not set
func (lc *LanguageConfig) GetFallbackLanguage() string {
	if lc.FallbackLanguage != "" {
		return lc.FallbackLanguage
	}
	return lc.DefaultLanguage
}

// TranslationRequest represents a request to create or update translations
type TranslationRequest struct {
	GroupType   string            `json:"group_type" validate:"required,oneof=article category tag"`
	ContentIDs  []uint64          `json:"content_ids" validate:"required,min=2"`
	Translations map[string]string `json:"translations,omitempty"` // language_code -> content_id mapping
}

// LanguageRouteInfo contains information for language-aware URL routing
type LanguageRouteInfo struct {
	LanguageCode      string `json:"language_code"`
	IsDefault         bool   `json:"is_default"`
	URLPrefix         string `json:"url_prefix"`         // e.g., "/en", "/ar", "" for default
	Direction         string `json:"direction"`          // "ltr" or "rtl"
	AlternateURLs     map[string]string `json:"alternate_urls"` // language_code -> URL mapping
}

// ContentWithTranslations interface for content that supports translations
type ContentWithTranslations interface {
	GetID() uint64
	GetLanguageCode() string
	GetTranslationGroupID() *uint64
	GetTranslations() Translations
	SetTranslations(Translations)
}

// Implement ContentWithTranslations for MultilingualArticle
func (ma *MultilingualArticle) GetID() uint64 {
	return ma.ID
}

func (ma *MultilingualArticle) GetLanguageCode() string {
	return ma.LanguageCode
}

func (ma *MultilingualArticle) GetTranslationGroupID() *uint64 {
	return ma.TranslationGroupID
}

func (ma *MultilingualArticle) GetTranslations() Translations {
	return ma.Translations
}

func (ma *MultilingualArticle) SetTranslations(translations Translations) {
	ma.Translations = translations
}

// Implement ContentWithTranslations for MultilingualCategory
func (mc *MultilingualCategory) GetID() uint64 {
	return mc.ID
}

func (mc *MultilingualCategory) GetLanguageCode() string {
	return mc.LanguageCode
}

func (mc *MultilingualCategory) GetTranslationGroupID() *uint64 {
	return mc.TranslationGroupID
}

func (mc *MultilingualCategory) GetTranslations() Translations {
	return mc.Translations
}

func (mc *MultilingualCategory) SetTranslations(translations Translations) {
	mc.Translations = translations
}

// Implement ContentWithTranslations for MultilingualTag
func (mt *MultilingualTag) GetID() uint64 {
	return mt.ID
}

func (mt *MultilingualTag) GetLanguageCode() string {
	return mt.LanguageCode
}

func (mt *MultilingualTag) GetTranslationGroupID() *uint64 {
	return mt.TranslationGroupID
}

func (mt *MultilingualTag) GetTranslations() Translations {
	return mt.Translations
}

func (mt *MultilingualTag) SetTranslations(translations Translations) {
	mt.Translations = translations
}