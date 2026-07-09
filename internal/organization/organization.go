package organization

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Organization represents a row from the organizations table.
type Organization struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Type        *string    `json:"type,omitempty"`    // company, government, ngo, academic, media, multilateral
	Description *string    `json:"description,omitempty"`
	CountryCode *string    `json:"country_code,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListAll returns all organizations, ordered by name.
func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Organization, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, type, description, country_code, is_active, created_at, updated_at
		FROM organizations
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(
			&o.ID, &o.Code, &o.Name, &o.Type, &o.Description,
			&o.CountryCode, &o.IsActive, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

// ListActive returns active organizations.
func ListActive(ctx context.Context, pool *pgxpool.Pool) ([]Organization, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, type, description, country_code, is_active, created_at, updated_at
		FROM organizations
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active organizations: %w", err)
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(
			&o.ID, &o.Code, &o.Name, &o.Type, &o.Description,
			&o.CountryCode, &o.IsActive, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

// GetByCode returns a single organization by its code.
func GetByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Organization, error) {
	var o Organization
	err := pool.QueryRow(ctx, `
		SELECT id, code, name, type, description, country_code, is_active, created_at, updated_at
		FROM organizations
		WHERE code = $1
	`, code).Scan(
		&o.ID, &o.Code, &o.Name, &o.Type, &o.Description,
		&o.CountryCode, &o.IsActive, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get organization %s: %w", code, err)
	}
	return &o, nil
}

// ListByType returns organizations of a given type.
func ListByType(ctx context.Context, pool *pgxpool.Pool, orgType string) ([]Organization, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, type, description, country_code, is_active, created_at, updated_at
		FROM organizations
		WHERE type = $1
		ORDER BY name
	`, orgType)
	if err != nil {
		return nil, fmt.Errorf("list organizations by type %s: %w", orgType, err)
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(
			&o.ID, &o.Code, &o.Name, &o.Type, &o.Description,
			&o.CountryCode, &o.IsActive, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}