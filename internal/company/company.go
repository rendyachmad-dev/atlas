package company

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Company represents a row from the companies table.
type Company struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	ParentID    *string    `json:"parent_id,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListAll returns all companies, ordered by name.
func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Company, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM companies
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()

	var companies []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(
			&c.ID, &c.Code, &c.Name, &c.Description,
			&c.ParentID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		companies = append(companies, c)
	}
	return companies, rows.Err()
}

// ListActive returns active companies.
func ListActive(ctx context.Context, pool *pgxpool.Pool) ([]Company, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM companies
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active companies: %w", err)
	}
	defer rows.Close()

	var companies []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(
			&c.ID, &c.Code, &c.Name, &c.Description,
			&c.ParentID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		companies = append(companies, c)
	}
	return companies, rows.Err()
}

// GetByCode returns a single company by code.
func GetByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Company, error) {
	var c Company
	err := pool.QueryRow(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM companies
		WHERE code = $1
	`, code).Scan(
		&c.ID, &c.Code, &c.Name, &c.Description,
		&c.ParentID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get company %s: %w", code, err)
	}
	return &c, nil
}