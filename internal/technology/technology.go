package technology

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Technology represents a row from the technologies table.
type Technology struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	ParentID    *string    `json:"parent_id,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListAll returns all technologies, ordered by name.
func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Technology, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM technologies
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list technologies: %w", err)
	}
	defer rows.Close()

	var techs []Technology
	for rows.Next() {
		var t Technology
		if err := rows.Scan(
			&t.ID, &t.Code, &t.Name, &t.Description,
			&t.ParentID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan technology: %w", err)
		}
		techs = append(techs, t)
	}
	return techs, rows.Err()
}

// ListActive returns active technologies.
func ListActive(ctx context.Context, pool *pgxpool.Pool) ([]Technology, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM technologies
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active technologies: %w", err)
	}
	defer rows.Close()

	var techs []Technology
	for rows.Next() {
		var t Technology
		if err := rows.Scan(
			&t.ID, &t.Code, &t.Name, &t.Description,
			&t.ParentID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan technology: %w", err)
		}
		techs = append(techs, t)
	}
	return techs, rows.Err()
}

// GetByCode returns a single technology by code.
func GetByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Technology, error) {
	var t Technology
	err := pool.QueryRow(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM technologies
		WHERE code = $1
	`, code).Scan(
		&t.ID, &t.Code, &t.Name, &t.Description,
		&t.ParentID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get technology %s: %w", code, err)
	}
	return &t, nil
}