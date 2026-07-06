package stakeholder

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Stakeholder represents a row from the stakeholders table.
type Stakeholder struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListAll returns all stakeholders, ordered by name.
func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Stakeholder, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, is_active, created_at, updated_at
		FROM stakeholders
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list stakeholders: %w", err)
	}
	defer rows.Close()

	var stakeholders []Stakeholder
	for rows.Next() {
		var s Stakeholder
		if err := rows.Scan(
			&s.ID, &s.Code, &s.Name, &s.Description,
			&s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stakeholder: %w", err)
		}
		stakeholders = append(stakeholders, s)
	}
	return stakeholders, rows.Err()
}

// ListActive returns active stakeholders.
func ListActive(ctx context.Context, pool *pgxpool.Pool) ([]Stakeholder, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, is_active, created_at, updated_at
		FROM stakeholders
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active stakeholders: %w", err)
	}
	defer rows.Close()

	var stakeholders []Stakeholder
	for rows.Next() {
		var s Stakeholder
		if err := rows.Scan(
			&s.ID, &s.Code, &s.Name, &s.Description,
			&s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stakeholder: %w", err)
		}
		stakeholders = append(stakeholders, s)
	}
	return stakeholders, rows.Err()
}

// GetByCode returns a single stakeholder by its code.
func GetByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Stakeholder, error) {
	var s Stakeholder
	err := pool.QueryRow(ctx, `
		SELECT id, code, name, description, is_active, created_at, updated_at
		FROM stakeholders
		WHERE code = $1
	`, code).Scan(
		&s.ID, &s.Code, &s.Name, &s.Description,
		&s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get stakeholder %s: %w", code, err)
	}
	return &s, nil
}