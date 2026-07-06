package industry

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Industry represents a row from the industries table.
type Industry struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	ParentID    *string    `json:"parent_id,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListAll returns all industries, ordered by name.
func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Industry, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM industries
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list industries: %w", err)
	}
	defer rows.Close()

	var industries []Industry
	for rows.Next() {
		var ind Industry
		if err := rows.Scan(
			&ind.ID, &ind.Code, &ind.Name, &ind.Description,
			&ind.ParentID, &ind.IsActive, &ind.CreatedAt, &ind.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan industry: %w", err)
		}
		industries = append(industries, ind)
	}
	return industries, rows.Err()
}

// ListActive returns active industries.
func ListActive(ctx context.Context, pool *pgxpool.Pool) ([]Industry, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM industries
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active industries: %w", err)
	}
	defer rows.Close()

	var industries []Industry
	for rows.Next() {
		var ind Industry
		if err := rows.Scan(
			&ind.ID, &ind.Code, &ind.Name, &ind.Description,
			&ind.ParentID, &ind.IsActive, &ind.CreatedAt, &ind.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan industry: %w", err)
		}
		industries = append(industries, ind)
	}
	return industries, rows.Err()
}

// GetByCode returns a single industry by its code.
func GetByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Industry, error) {
	var ind Industry
	err := pool.QueryRow(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM industries
		WHERE code = $1
	`, code).Scan(
		&ind.ID, &ind.Code, &ind.Name, &ind.Description,
		&ind.ParentID, &ind.IsActive, &ind.CreatedAt, &ind.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get industry %s: %w", code, err)
	}
	return &ind, nil
}