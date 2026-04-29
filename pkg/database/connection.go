package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"high-performance-news-website/internal/config"
)

type DB struct {
	*sql.DB
	preparedStmts map[string]*sql.Stmt
}

// Prepared statement names
const (
	StmtGetArticle     = "get_article"
	StmtGetHomepage    = "get_homepage"
	StmtGetCategory    = "get_category"
	StmtInsertView     = "insert_view"
	StmtInsertArticle  = "insert_article"
	StmtUpdateArticle  = "update_article"
	StmtGetTrending    = "get_trending"
	StmtGetPopular     = "get_popular"
)

func NewConnection(cfg *config.DatabaseConfig) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool for high performance (optimized for 50K+ articles/day)
	db.SetMaxOpenConns(cfg.MaxConns)     // 150 connections (leave 50 for maintenance)
	db.SetMaxIdleConns(cfg.MinConns)     // 40 idle connections
	db.SetConnMaxLifetime(time.Hour)     // Connection lifetime
	db.SetConnMaxIdleTime(10 * time.Minute) // Idle timeout

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	dbWrapper := &DB{
		DB:            db,
		preparedStmts: make(map[string]*sql.Stmt),
	}

	// Initialize prepared statements for performance
	if err := dbWrapper.initPreparedStatements(); err != nil {
		return nil, fmt.Errorf("failed to initialize prepared statements: %w", err)
	}

	return dbWrapper, nil
}

// NewPgBouncerConnection creates a connection through PgBouncer for production use
func NewPgBouncerConnection(cfg *config.DatabaseConfig) (*DB, error) {
	// Use PgBouncer port (6432) instead of direct PostgreSQL connection
	dsn := fmt.Sprintf("host=%s port=6432 user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open PgBouncer connection: %w", err)
	}

	// Optimized settings for PgBouncer transaction mode
	db.SetMaxOpenConns(200)              // Match PgBouncer max_client_conn
	db.SetMaxIdleConns(50)               // Match PgBouncer default_pool_size
	db.SetConnMaxLifetime(30 * time.Minute) // Shorter lifetime for transaction mode
	db.SetConnMaxIdleTime(5 * time.Minute)  // Shorter idle timeout

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PgBouncer: %w", err)
	}

	dbWrapper := &DB{
		DB:            db,
		preparedStmts: make(map[string]*sql.Stmt),
	}

	// Initialize prepared statements for performance
	if err := dbWrapper.initPreparedStatements(); err != nil {
		return nil, fmt.Errorf("failed to initialize prepared statements: %w", err)
	}

	return dbWrapper, nil
}

func (db *DB) initPreparedStatements() error {
	statements := map[string]string{
		StmtGetArticle: `
			SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id,
				   a.published_at, a.view_count, a.like_count, a.dislike_count,
				   a.meta_title, a.meta_description, a.canonical_url, a.schema_type,
				   a.featured_image_id, a.auto_linking,
				   CASE WHEN i.original_url LIKE '/uploads/%' THEN i.original_url ELSE NULL END as featured_image
			FROM articles a
			LEFT JOIN images i ON a.featured_image_id = i.id
			WHERE a.slug = $1 AND a.status = 'published'
			AND a.published_at IS NOT NULL`,

		StmtGetHomepage: `
			SELECT a.id, a.title, a.slug, a.excerpt, a.author_id, a.category_id, a.published_at, a.view_count,
				   CASE WHEN i.original_url LIKE '/uploads/%' THEN i.original_url ELSE NULL END
			FROM articles a
			LEFT JOIN images i ON a.featured_image_id = i.id
			WHERE a.status = 'published' AND a.published_at IS NOT NULL
			ORDER BY a.published_at DESC 
			LIMIT $1`,

		StmtGetCategory: `
			SELECT a.id, a.title, a.slug, a.excerpt, a.author_id, a.published_at, a.view_count,
				   CASE WHEN i.original_url LIKE '/uploads/%' THEN i.original_url ELSE NULL END
			FROM articles a
			LEFT JOIN images i ON a.featured_image_id = i.id
			WHERE a.status = 'published' AND a.published_at IS NOT NULL
			AND a.category_id = $1
			ORDER BY a.published_at DESC 
			LIMIT $2 OFFSET $3`,

		StmtInsertView: `
			INSERT INTO article_views (article_id, ip_address, user_agent, referer, created_at)
			VALUES ($1, $2, $3, $4, NOW())`,

		StmtInsertArticle: `
			INSERT INTO articles (title, slug, content, excerpt, author_id, category_id, 
								 status, published_at, meta_title, meta_description, 
								 canonical_url, schema_type, featured_image_id, auto_linking,
								 language_code, focus_keyword, moderation_status, last_moderated_by,
								 created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW(), NOW())
			RETURNING id, created_at`,

		StmtUpdateArticle: `
			UPDATE articles 
			SET title = $2, slug = $3, content = $4, excerpt = $5, 
				category_id = $6, status = $7, published_at = $8,
				meta_title = $9, meta_description = $10, canonical_url = $11,
				featured_image_id = $12, auto_linking = $13, updated_at = NOW()
			WHERE id = $1`,

		StmtGetTrending: `
			SELECT a.id, a.title, a.slug, a.excerpt, a.author_id, a.published_at, a.view_count,
				   (a.view_count * 0.7 + a.like_count * 0.2 + (EXTRACT(EPOCH FROM NOW() - a.published_at) / 3600)::int * -0.1) as trending_score
			FROM articles a
			WHERE a.status = 'published' 
			  AND a.published_at IS NOT NULL
			  AND a.published_at > NOW() - INTERVAL '24 hours'
			ORDER BY trending_score DESC
			LIMIT $1`,

		StmtGetPopular: `
			SELECT a.id, a.title, a.slug, a.excerpt, a.author_id, a.published_at, a.view_count
			FROM articles a
			WHERE a.status = 'published' 
			  AND a.published_at IS NOT NULL
			ORDER BY a.view_count DESC, a.published_at DESC
			LIMIT $1`,
	}

	for name, query := range statements {
		stmt, err := db.Prepare(query)
		if err != nil {
			return fmt.Errorf("failed to prepare statement %s: %w", name, err)
		}
		db.preparedStmts[name] = stmt
	}

	return nil
}

func (db *DB) GetPreparedStatement(name string) (*sql.Stmt, error) {
	stmt, exists := db.preparedStmts[name]
	if !exists {
		return nil, fmt.Errorf("prepared statement %s not found", name)
	}
	return stmt, nil
}

func (db *DB) Close() error {
	// Close all prepared statements first
	for name, stmt := range db.preparedStmts {
		if err := stmt.Close(); err != nil {
			log.Printf("Error closing prepared statement %s: %v", name, err)
		}
	}

	// Close the database connection
	return db.DB.Close()
}

func (db *DB) Health() error {
	return db.Ping()
}

// GetStats returns database connection statistics
func (db *DB) GetStats() sql.DBStats {
	return db.Stats()
}