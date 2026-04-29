package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Migrator struct {
	db       *sql.DB
	migrator *migrate.Migrate
}

func NewMigrator(db *sql.DB, migrationsPath string) (*Migrator, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return &Migrator{
		db:       db,
		migrator: m,
	}, nil
}

// Up runs all available migrations
func (m *Migrator) Up() error {
	if err := m.migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations up: %w", err)
	}
	log.Println("Migrations completed successfully")
	return nil
}

// Down rolls back all migrations
func (m *Migrator) Down() error {
	if err := m.migrator.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations down: %w", err)
	}
	log.Println("Migrations rolled back successfully")
	return nil
}

// Steps runs n migrations up (positive) or down (negative)
func (m *Migrator) Steps(n int) error {
	if err := m.migrator.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run %d migration steps: %w", n, err)
	}
	log.Printf("Migration steps (%d) completed successfully", n)
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrator.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(version int) error {
	if err := m.migrator.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version %d: %w", version, err)
	}
	log.Printf("Migration version forced to %d", version)
	return nil
}

// Drop drops the entire database schema
func (m *Migrator) Drop() error {
	if err := m.migrator.Drop(); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}
	log.Println("Database schema dropped successfully")
	return nil
}

// Close closes the migrator
func (m *Migrator) Close() error {
	sourceErr, dbErr := m.migrator.Close()
	if sourceErr != nil {
		return fmt.Errorf("failed to close migration source: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close migration database: %w", dbErr)
	}
	return nil
}

// CreatePartitions creates initial partitions for the articles table
func (m *Migrator) CreatePartitions() error {
	partitionSQL := `
		-- Create partition manager function
		CREATE OR REPLACE FUNCTION create_monthly_partitions()
		RETURNS void AS $$
		DECLARE
			start_date date;
			end_date date;
			partition_name text;
		BEGIN
			-- Create partitions for current month and next 3 months
			FOR i IN 0..3 LOOP
				start_date := date_trunc('month', CURRENT_DATE + (i || ' months')::interval);
				end_date := start_date + interval '1 month';
				partition_name := 'articles_' || to_char(start_date, 'YYYY_MM');
				
				-- Check if partition already exists
				IF NOT EXISTS (
					SELECT 1 FROM pg_class WHERE relname = partition_name
				) THEN
					EXECUTE format('CREATE TABLE %I PARTITION OF articles FOR VALUES FROM (%L) TO (%L)',
						partition_name, start_date, end_date);
					
					-- Create indexes on the partition
					EXECUTE format('CREATE INDEX idx_%I_published_at ON %I (published_at DESC) WHERE status = ''published''',
						partition_name, partition_name);
					EXECUTE format('CREATE INDEX idx_%I_category ON %I (category_id, published_at DESC)',
						partition_name, partition_name);
					EXECUTE format('CREATE INDEX idx_%I_author ON %I (author_id, published_at DESC)',
						partition_name, partition_name);
					EXECUTE format('CREATE INDEX idx_%I_slug ON %I (slug) WHERE status = ''published''',
						partition_name, partition_name);
					EXECUTE format('CREATE INDEX idx_%I_search ON %I USING gin(to_tsvector(''english'', title || '' '' || content))',
						partition_name, partition_name);
					
					-- Create BRIN index for time-series performance
					EXECUTE format('CREATE INDEX idx_%I_published_brin ON %I USING BRIN (published_at) WITH (pages_per_range = 128)',
						partition_name, partition_name);
						
					RAISE NOTICE 'Created partition: %', partition_name;
				END IF;
			END LOOP;
		END;
		$$ LANGUAGE plpgsql;

		-- Create article_tags partitions
		CREATE OR REPLACE FUNCTION create_article_tags_partitions()
		RETURNS void AS $$
		DECLARE
			start_date date;
			end_date date;
			partition_name text;
		BEGIN
			-- Create partitions for current month and next 3 months
			FOR i IN 0..3 LOOP
				start_date := date_trunc('month', CURRENT_DATE + (i || ' months')::interval);
				end_date := start_date + interval '1 month';
				partition_name := 'article_tags_' || to_char(start_date, 'YYYY_MM');
				
				-- Check if partition already exists
				IF NOT EXISTS (
					SELECT 1 FROM pg_class WHERE relname = partition_name
				) THEN
					EXECUTE format('CREATE TABLE %I PARTITION OF article_tags FOR VALUES FROM (%L) TO (%L)',
						partition_name, start_date, end_date);
					
					-- Create indexes on the partition
					EXECUTE format('CREATE INDEX idx_%I_article ON %I (article_id)',
						partition_name, partition_name);
					EXECUTE format('CREATE INDEX idx_%I_tag ON %I (tag_id)',
						partition_name, partition_name);
						
					RAISE NOTICE 'Created article_tags partition: %', partition_name;
				END IF;
			END LOOP;
		END;
		$$ LANGUAGE plpgsql;

		-- Create article_views partitions
		CREATE OR REPLACE FUNCTION create_article_views_partitions()
		RETURNS void AS $$
		DECLARE
			start_date date;
			end_date date;
			partition_name text;
		BEGIN
			-- Create partitions for current month and next 3 months
			FOR i IN 0..3 LOOP
				start_date := date_trunc('month', CURRENT_DATE + (i || ' months')::interval);
				end_date := start_date + interval '1 month';
				partition_name := 'article_views_' || to_char(start_date, 'YYYY_MM');
				
				-- Check if partition already exists
				IF NOT EXISTS (
					SELECT 1 FROM pg_class WHERE relname = partition_name
				) THEN
					EXECUTE format('CREATE TABLE %I PARTITION OF article_views FOR VALUES FROM (%L) TO (%L)',
						partition_name, start_date, end_date);
					
					-- Create BRIN index for high-volume analytics data
					EXECUTE format('CREATE INDEX idx_%I_created_brin ON %I USING BRIN (created_at) WITH (pages_per_range = 64)',
						partition_name, partition_name);
					EXECUTE format('CREATE INDEX idx_%I_article ON %I (article_id, created_at)',
						partition_name, partition_name);
						
					RAISE NOTICE 'Created article_views partition: %', partition_name;
				END IF;
			END LOOP;
		END;
		$$ LANGUAGE plpgsql;

		-- Execute partition creation functions
		SELECT create_monthly_partitions();
		SELECT create_article_tags_partitions();
		SELECT create_article_views_partitions();
	`

	if _, err := m.db.Exec(partitionSQL); err != nil {
		return fmt.Errorf("failed to create partitions: %w", err)
	}

	log.Println("Database partitions created successfully")
	return nil
}