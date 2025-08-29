package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Search   SearchConfig   `mapstructure:"search"`
	App      AppConfig      `mapstructure:"app"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Backup   BackupConfig   `mapstructure:"backup"`
}

type ServerConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Mode       string `mapstructure:"mode"` // debug, release, test
	StaticPath string `mapstructure:"static_path"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxConns     int    `mapstructure:"max_conns"`
	MinConns     int    `mapstructure:"min_conns"`
	UsePgBouncer bool   `mapstructure:"use_pgbouncer"`
	PgBouncerPort int   `mapstructure:"pgbouncer_port"`
}

type CacheConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	LogLevel    string `mapstructure:"log_level"`
	DevMode     bool   `mapstructure:"dev_mode"` // Enable development mode with mock services
}

type SearchConfig struct {
	MeiliSearchURL    string `mapstructure:"meilisearch_url"`
	MeiliSearchAPIKey string `mapstructure:"meilisearch_api_key"`
	IndexName         string `mapstructure:"index_name"`
	BatchSize         int    `mapstructure:"batch_size"`
	CacheTTLMinutes   int    `mapstructure:"cache_ttl_minutes"`
	Enabled           bool   `mapstructure:"enabled"`
}

type JWTConfig struct {
	Secret               string `mapstructure:"secret"`
	AccessTokenDuration  string `mapstructure:"access_token_duration"`
	RefreshTokenDuration string `mapstructure:"refresh_token_duration"`
}

func Load() (*Config, error) {
	// Set default values
	setDefaults()

	// Set config file name and paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/news-website")

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("NEWS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults and environment variables
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.static_path", "./web/static")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "news_website")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_conns", 150)
	viper.SetDefault("database.min_conns", 40)
	viper.SetDefault("database.use_pgbouncer", false)
	viper.SetDefault("database.pgbouncer_port", 6432)

	// Cache defaults (DragonflyDB)
	viper.SetDefault("cache.host", "localhost")
	viper.SetDefault("cache.port", 6379)
	viper.SetDefault("cache.password", "")
	viper.SetDefault("cache.db", 0)

	// App defaults
	viper.SetDefault("app.name", "High Performance News Website")
	viper.SetDefault("app.version", "1.0.0")
	viper.SetDefault("app.environment", "development")
	viper.SetDefault("app.log_level", "info")
	viper.SetDefault("app.dev_mode", false)

	// Search defaults
	viper.SetDefault("search.meilisearch_url", "http://localhost:7700")
	viper.SetDefault("search.meilisearch_api_key", "")
	viper.SetDefault("search.index_name", "articles")
	viper.SetDefault("search.batch_size", 1000)
	viper.SetDefault("search.cache_ttl_minutes", 5)
	viper.SetDefault("search.enabled", true)

	// JWT defaults
	viper.SetDefault("jwt.secret", "your-super-secret-jwt-key-change-this-in-production")
	viper.SetDefault("jwt.access_token_duration", "15m")
	viper.SetDefault("jwt.refresh_token_duration", "7d")

	// Backup defaults
	setBackupDefaults()
}

func setBackupDefaults() {
	// General backup defaults
	viper.SetDefault("backup.enabled", true)
	viper.SetDefault("backup.backup_dir", "/var/backups/news-website")
	viper.SetDefault("backup.retention_days", 30)
	viper.SetDefault("backup.compression_level", 6)
	viper.SetDefault("backup.encryption_enabled", true)
	viper.SetDefault("backup.encryption_key", "")
	
	// Backup scheduling defaults
	viper.SetDefault("backup.full_backup_interval", "24h")
	viper.SetDefault("backup.incremental_backup_interval", "1h")
	
	// Cross-region replication defaults
	viper.SetDefault("backup.cross_region_enabled", false)
	viper.SetDefault("backup.replication_targets", []interface{}{})
	
	// Point-in-time recovery defaults
	viper.SetDefault("backup.wal_archive_enabled", true)
	viper.SetDefault("backup.wal_archive_dir", "/var/backups/news-website/wal")
	
	// Disaster recovery testing defaults
	viper.SetDefault("backup.testing_enabled", true)
	viper.SetDefault("backup.testing_interval", "168h") // Weekly
	viper.SetDefault("backup.testing_retention", 5)
	
	// Notification defaults
	viper.SetDefault("backup.notification_enabled", true)
	viper.SetDefault("backup.notification_emails", []string{})
	viper.SetDefault("backup.slack_webhook_url", "")
}

func (c *Config) GetDatabaseDSN() string {
	port := c.Database.Port
	if c.Database.UsePgBouncer {
		port = c.Database.PgBouncerPort
	}
	
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func (c *Config) GetDirectDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}