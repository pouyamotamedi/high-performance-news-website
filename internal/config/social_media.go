package config

import (
	"fmt"
	"time"
)

// SocialMediaConfig holds social media integration configuration
type SocialMediaConfig struct {
	// General settings
	Enabled                bool          `mapstructure:"enabled" json:"enabled"`
	DefaultRetryAttempts   int           `mapstructure:"default_retry_attempts" json:"default_retry_attempts"`
	RetryBackoffMultiplier float64       `mapstructure:"retry_backoff_multiplier" json:"retry_backoff_multiplier"`
	WebhookTimeout         time.Duration `mapstructure:"webhook_timeout" json:"webhook_timeout"`
	
	// Platform-specific settings
	Facebook FacebookConfig `mapstructure:"facebook" json:"facebook"`
	Telegram TelegramConfig `mapstructure:"telegram" json:"telegram"`
	Twitter  TwitterConfig  `mapstructure:"twitter" json:"twitter"`
	
	// Content generation settings
	ContentGeneration ContentGenerationConfig `mapstructure:"content_generation" json:"content_generation"`
	
	// Rate limiting
	RateLimit RateLimitConfig `mapstructure:"rate_limit" json:"rate_limit"`
}

// FacebookConfig holds Facebook-specific configuration
type FacebookConfig struct {
	Enabled              bool          `mapstructure:"enabled" json:"enabled"`
	APIVersion           string        `mapstructure:"api_version" json:"api_version"`
	InstantArticles      bool          `mapstructure:"instant_articles" json:"instant_articles"`
	PostDelay            time.Duration `mapstructure:"post_delay" json:"post_delay"`
	WebhookVerifyToken   string        `mapstructure:"webhook_verify_token" json:"webhook_verify_token"`
	MaxPostLength        int           `mapstructure:"max_post_length" json:"max_post_length"`
	DefaultHashtagCount  int           `mapstructure:"default_hashtag_count" json:"default_hashtag_count"`
}

// TelegramConfig holds Telegram-specific configuration
type TelegramConfig struct {
	Enabled         bool          `mapstructure:"enabled" json:"enabled"`
	ParseMode       string        `mapstructure:"parse_mode" json:"parse_mode"`
	PostDelay       time.Duration `mapstructure:"post_delay" json:"post_delay"`
	MaxMessageLength int          `mapstructure:"max_message_length" json:"max_message_length"`
	DisablePreview  bool          `mapstructure:"disable_preview" json:"disable_preview"`
}

// TwitterConfig holds Twitter-specific configuration
type TwitterConfig struct {
	Enabled           bool          `mapstructure:"enabled" json:"enabled"`
	APIVersion        string        `mapstructure:"api_version" json:"api_version"`
	PostDelay         time.Duration `mapstructure:"post_delay" json:"post_delay"`
	MaxTweetLength    int           `mapstructure:"max_tweet_length" json:"max_tweet_length"`
	MaxHashtagCount   int           `mapstructure:"max_hashtag_count" json:"max_hashtag_count"`
	ThreadSupport     bool          `mapstructure:"thread_support" json:"thread_support"`
}

// ContentGenerationConfig holds content generation settings
type ContentGenerationConfig struct {
	MaxHashtags          int      `mapstructure:"max_hashtags" json:"max_hashtags"`
	MinHashtagLength     int      `mapstructure:"min_hashtag_length" json:"min_hashtag_length"`
	ExcludeWords         []string `mapstructure:"exclude_words" json:"exclude_words"`
	IncludeArticleURL    bool     `mapstructure:"include_article_url" json:"include_article_url"`
	URLShortening        bool     `mapstructure:"url_shortening" json:"url_shortening"`
	CustomDomain         string   `mapstructure:"custom_domain" json:"custom_domain"`
	UTMParameters        bool     `mapstructure:"utm_parameters" json:"utm_parameters"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Facebook RateLimitPlatform `mapstructure:"facebook" json:"facebook"`
	Telegram RateLimitPlatform `mapstructure:"telegram" json:"telegram"`
	Twitter  RateLimitPlatform `mapstructure:"twitter" json:"twitter"`
}

// RateLimitPlatform holds platform-specific rate limits
type RateLimitPlatform struct {
	PostsPerHour   int           `mapstructure:"posts_per_hour" json:"posts_per_hour"`
	PostsPerDay    int           `mapstructure:"posts_per_day" json:"posts_per_day"`
	BurstLimit     int           `mapstructure:"burst_limit" json:"burst_limit"`
	CooldownPeriod time.Duration `mapstructure:"cooldown_period" json:"cooldown_period"`
}

// DefaultSocialMediaConfig returns default social media configuration
func DefaultSocialMediaConfig() SocialMediaConfig {
	return SocialMediaConfig{
		Enabled:                true,
		DefaultRetryAttempts:   3,
		RetryBackoffMultiplier: 2.0,
		WebhookTimeout:         30 * time.Second,
		
		Facebook: FacebookConfig{
			Enabled:             true,
			APIVersion:          "v18.0",
			InstantArticles:     true,
			PostDelay:           5 * time.Minute,
			MaxPostLength:       63206, // Facebook's character limit
			DefaultHashtagCount: 5,
		},
		
		Telegram: TelegramConfig{
			Enabled:          true,
			ParseMode:        "HTML",
			PostDelay:        2 * time.Minute,
			MaxMessageLength: 4096, // Telegram's character limit
			DisablePreview:   false,
		},
		
		Twitter: TwitterConfig{
			Enabled:         true,
			APIVersion:      "2",
			PostDelay:       3 * time.Minute,
			MaxTweetLength:  280, // Twitter's character limit
			MaxHashtagCount: 3,
			ThreadSupport:   true,
		},
		
		ContentGeneration: ContentGenerationConfig{
			MaxHashtags:       5,
			MinHashtagLength:  3,
			ExcludeWords:      []string{"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"},
			IncludeArticleURL: true,
			URLShortening:     false,
			UTMParameters:     true,
		},
		
		RateLimit: RateLimitConfig{
			Facebook: RateLimitPlatform{
				PostsPerHour:   10,
				PostsPerDay:    100,
				BurstLimit:     3,
				CooldownPeriod: 15 * time.Minute,
			},
			Telegram: RateLimitPlatform{
				PostsPerHour:   20,
				PostsPerDay:    200,
				BurstLimit:     5,
				CooldownPeriod: 10 * time.Minute,
			},
			Twitter: RateLimitPlatform{
				PostsPerHour:   15,
				PostsPerDay:    150,
				BurstLimit:     3,
				CooldownPeriod: 20 * time.Minute,
			},
		},
	}
}

// ValidateSocialMediaConfig validates the social media configuration
func ValidateSocialMediaConfig(config *SocialMediaConfig) error {
	if !config.Enabled {
		return nil // Skip validation if disabled
	}

	// Validate retry settings
	if config.DefaultRetryAttempts < 1 || config.DefaultRetryAttempts > 10 {
		return fmt.Errorf("default_retry_attempts must be between 1 and 10")
	}

	if config.RetryBackoffMultiplier < 1.0 || config.RetryBackoffMultiplier > 10.0 {
		return fmt.Errorf("retry_backoff_multiplier must be between 1.0 and 10.0")
	}

	// Validate webhook timeout
	if config.WebhookTimeout < time.Second || config.WebhookTimeout > 5*time.Minute {
		return fmt.Errorf("webhook_timeout must be between 1 second and 5 minutes")
	}

	// Validate Facebook config
	if config.Facebook.Enabled {
		if config.Facebook.APIVersion == "" {
			return fmt.Errorf("facebook api_version is required when Facebook is enabled")
		}
		if config.Facebook.MaxPostLength < 1 || config.Facebook.MaxPostLength > 100000 {
			return fmt.Errorf("facebook max_post_length must be between 1 and 100000")
		}
	}

	// Validate Telegram config
	if config.Telegram.Enabled {
		if config.Telegram.ParseMode != "" && 
		   config.Telegram.ParseMode != "HTML" && 
		   config.Telegram.ParseMode != "Markdown" &&
		   config.Telegram.ParseMode != "MarkdownV2" {
			return fmt.Errorf("telegram parse_mode must be HTML, Markdown, or MarkdownV2")
		}
		if config.Telegram.MaxMessageLength < 1 || config.Telegram.MaxMessageLength > 4096 {
			return fmt.Errorf("telegram max_message_length must be between 1 and 4096")
		}
	}

	// Validate Twitter config
	if config.Twitter.Enabled {
		if config.Twitter.MaxTweetLength < 1 || config.Twitter.MaxTweetLength > 280 {
			return fmt.Errorf("twitter max_tweet_length must be between 1 and 280")
		}
	}

	// Validate content generation config
	if config.ContentGeneration.MaxHashtags < 0 || config.ContentGeneration.MaxHashtags > 20 {
		return fmt.Errorf("content_generation max_hashtags must be between 0 and 20")
	}

	if config.ContentGeneration.MinHashtagLength < 1 || config.ContentGeneration.MinHashtagLength > 50 {
		return fmt.Errorf("content_generation min_hashtag_length must be between 1 and 50")
	}

	// Validate rate limits
	platforms := []RateLimitPlatform{
		config.RateLimit.Facebook,
		config.RateLimit.Telegram,
		config.RateLimit.Twitter,
	}

	for i, platform := range platforms {
		platformName := []string{"facebook", "telegram", "twitter"}[i]
		
		if platform.PostsPerHour < 0 || platform.PostsPerHour > 1000 {
			return fmt.Errorf("%s posts_per_hour must be between 0 and 1000", platformName)
		}
		
		if platform.PostsPerDay < 0 || platform.PostsPerDay > 10000 {
			return fmt.Errorf("%s posts_per_day must be between 0 and 10000", platformName)
		}
		
		if platform.BurstLimit < 0 || platform.BurstLimit > 100 {
			return fmt.Errorf("%s burst_limit must be between 0 and 100", platformName)
		}
		
		if platform.CooldownPeriod < 0 || platform.CooldownPeriod > 24*time.Hour {
			return fmt.Errorf("%s cooldown_period must be between 0 and 24 hours", platformName)
		}
	}

	return nil
}

// GetPlatformConfig returns configuration for a specific platform
func (c *SocialMediaConfig) GetPlatformConfig(platform string) interface{} {
	switch platform {
	case "facebook":
		return c.Facebook
	case "telegram":
		return c.Telegram
	case "twitter":
		return c.Twitter
	default:
		return nil
	}
}

// IsPlatformEnabled checks if a platform is enabled
func (c *SocialMediaConfig) IsPlatformEnabled(platform string) bool {
	if !c.Enabled {
		return false
	}
	
	switch platform {
	case "facebook":
		return c.Facebook.Enabled
	case "telegram":
		return c.Telegram.Enabled
	case "twitter":
		return c.Twitter.Enabled
	default:
		return false
	}
}

// GetPostDelay returns the post delay for a platform
func (c *SocialMediaConfig) GetPostDelay(platform string) time.Duration {
	switch platform {
	case "facebook":
		return c.Facebook.PostDelay
	case "telegram":
		return c.Telegram.PostDelay
	case "twitter":
		return c.Twitter.PostDelay
	default:
		return 5 * time.Minute // Default delay
	}
}

// GetRateLimit returns the rate limit configuration for a platform
func (c *SocialMediaConfig) GetRateLimit(platform string) *RateLimitPlatform {
	switch platform {
	case "facebook":
		return &c.RateLimit.Facebook
	case "telegram":
		return &c.RateLimit.Telegram
	case "twitter":
		return &c.RateLimit.Twitter
	default:
		return nil
	}
}