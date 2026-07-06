package region

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Region represents a row from the regions table.
type Region struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	ParentID    *string    `json:"parent_id,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListAll returns all regions, ordered by name.
func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Region, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM regions
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list regions: %w", err)
	}
	defer rows.Close()

	var regions []Region
	for rows.Next() {
		var r Region
		if err := rows.Scan(
			&r.ID, &r.Code, &r.Name, &r.Description,
			&r.ParentID, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan region: %w", err)
		}
		regions = append(regions, r)
	}
	return regions, rows.Err()
}

// ListActive returns active regions.
func ListActive(ctx context.Context, pool *pgxpool.Pool) ([]Region, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM regions
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active regions: %w", err)
	}
	defer rows.Close()

	var regions []Region
	for rows.Next() {
		var r Region
		if err := rows.Scan(
			&r.ID, &r.Code, &r.Name, &r.Description,
			&r.ParentID, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan region: %w", err)
		}
		regions = append(regions, r)
	}
	return regions, rows.Err()
}

// GetByCode returns a single region by its code.
func GetByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Region, error) {
	var r Region
	err := pool.QueryRow(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM regions
		WHERE code = $1
	`, code).Scan(
		&r.ID, &r.Code, &r.Name, &r.Description,
		&r.ParentID, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get region %s: %w", code, err)
	}
	return &r, nil
}