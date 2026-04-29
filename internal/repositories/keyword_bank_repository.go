package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// KeywordBank represents a custom keyword bank
type KeywordBank struct {
	ID          uint64   `json:"id" db:"id"`
	Name        string   `json:"name" db:"name"`
	URL         string   `json:"url" db:"url"`
	Keywords    []string `json:"keywords" db:"keywords"`
	Description string   `json:"description" db:"description"`
	IsActive    bool     `json:"is_active" db:"is_active"`
}

// KeywordBankRepository handles keyword bank operations
type KeywordBankRepository struct {
	db *sql.DB
}

// NewKeywordBankRepository creates a new repository
func NewKeywordBankRepository(db *sql.DB) *KeywordBankRepository {
	return &KeywordBankRepository{db: db}
}

// GetAll retrieves all keyword banks
func (r *KeywordBankRepository) GetAll(ctx context.Context) ([]KeywordBank, error) {
	query := `SELECT id, name, url, keywords, description, is_active FROM keyword_banks ORDER BY name`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query keyword banks: %w", err)
	}
	defer rows.Close()
	
	var banks []KeywordBank
	for rows.Next() {
		var bank KeywordBank
		var keywordsJSON []byte
		var description sql.NullString
		
		err := rows.Scan(&bank.ID, &bank.Name, &bank.URL, &keywordsJSON, &description, &bank.IsActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan keyword bank: %w", err)
		}
		
		if description.Valid {
			bank.Description = description.String
		}
		
		if err := json.Unmarshal(keywordsJSON, &bank.Keywords); err != nil {
			return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
		}
		
		banks = append(banks, bank)
	}
	
	return banks, nil
}

// GetActive retrieves only active keyword banks
func (r *KeywordBankRepository) GetActive(ctx context.Context) ([]KeywordBank, error) {
	query := `SELECT id, name, url, keywords, description, is_active FROM keyword_banks WHERE is_active = true ORDER BY name`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active keyword banks: %w", err)
	}
	defer rows.Close()
	
	var banks []KeywordBank
	for rows.Next() {
		var bank KeywordBank
		var keywordsJSON []byte
		var description sql.NullString
		
		err := rows.Scan(&bank.ID, &bank.Name, &bank.URL, &keywordsJSON, &description, &bank.IsActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan keyword bank: %w", err)
		}
		
		if description.Valid {
			bank.Description = description.String
		}
		
		if err := json.Unmarshal(keywordsJSON, &bank.Keywords); err != nil {
			return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
		}
		
		banks = append(banks, bank)
	}
	
	return banks, nil
}

// Create creates a new keyword bank
func (r *KeywordBankRepository) Create(ctx context.Context, bank *KeywordBank) (*KeywordBank, error) {
	keywordsJSON, err := json.Marshal(bank.Keywords)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keywords: %w", err)
	}
	
	query := `
		INSERT INTO keyword_banks (name, url, keywords, description, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	
	err = r.db.QueryRowContext(ctx, query, bank.Name, bank.URL, keywordsJSON, bank.Description, bank.IsActive).Scan(&bank.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyword bank: %w", err)
	}
	
	return bank, nil
}

// Update updates an existing keyword bank
func (r *KeywordBankRepository) Update(ctx context.Context, bank *KeywordBank) error {
	keywordsJSON, err := json.Marshal(bank.Keywords)
	if err != nil {
		return fmt.Errorf("failed to marshal keywords: %w", err)
	}
	
	query := `
		UPDATE keyword_banks 
		SET name = $1, url = $2, keywords = $3, description = $4, is_active = $5, updated_at = NOW()
		WHERE id = $6
	`
	
	_, err = r.db.ExecContext(ctx, query, bank.Name, bank.URL, keywordsJSON, bank.Description, bank.IsActive, bank.ID)
	if err != nil {
		return fmt.Errorf("failed to update keyword bank: %w", err)
	}
	
	return nil
}

// Delete deletes a keyword bank
func (r *KeywordBankRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM keyword_banks WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete keyword bank: %w", err)
	}
	
	return nil
}

// GetByID retrieves a keyword bank by ID
func (r *KeywordBankRepository) GetByID(ctx context.Context, id uint64) (*KeywordBank, error) {
	query := `SELECT id, name, url, keywords, description, is_active FROM keyword_banks WHERE id = $1`
	
	var bank KeywordBank
	var keywordsJSON []byte
	var description sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(&bank.ID, &bank.Name, &bank.URL, &keywordsJSON, &description, &bank.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("keyword bank not found")
		}
		return nil, fmt.Errorf("failed to query keyword bank: %w", err)
	}
	
	if description.Valid {
		bank.Description = description.String
	}
	
	if err := json.Unmarshal(keywordsJSON, &bank.Keywords); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
	}
	
	return &bank, nil
}
