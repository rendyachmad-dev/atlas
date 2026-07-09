package country

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Country represents a row from the countries table (ISO-3166).
type Country struct {
	ID           string    `json:"id"`
	Code         string    `json:"code"`         // ISO-3166 alpha-2 (e.g., "US")
	Alpha3       string    `json:"alpha3"`       // ISO-3166 alpha-3 (e.g., "USA")
	NumericCode  *string   `json:"numeric_code,omitempty"`
	Name         string    `json:"name"`
	OfficialName *string   `json:"official_name,omitempty"`
	Region       *string   `json:"region,omitempty"`
	Subregion    *string   `json:"subregion,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListAll returns all countries, ordered by name.
func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Country, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, alpha3, numeric_code, name, official_name, region, subregion, is_active, created_at
		FROM countries
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list countries: %w", err)
	}
	defer rows.Close()

	var countries []Country
	for rows.Next() {
		var c Country
		if err := rows.Scan(
			&c.ID, &c.Code, &c.Alpha3, &c.NumericCode,
			&c.Name, &c.OfficialName, &c.Region, &c.Subregion,
			&c.IsActive, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan country: %w", err)
		}
		countries = append(countries, c)
	}
	return countries, rows.Err()
}

// ListActive returns active countries.
func ListActive(ctx context.Context, pool *pgxpool.Pool) ([]Country, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, alpha3, numeric_code, name, official_name, region, subregion, is_active, created_at
		FROM countries
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active countries: %w", err)
	}
	defer rows.Close()

	var countries []Country
	for rows.Next() {
		var c Country
		if err := rows.Scan(
			&c.ID, &c.Code, &c.Alpha3, &c.NumericCode,
			&c.Name, &c.OfficialName, &c.Region, &c.Subregion,
			&c.IsActive, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan country: %w", err)
		}
		countries = append(countries, c)
	}
	return countries, rows.Err()
}

// GetByCode returns a single country by its ISO-3166 alpha-2 code.
func GetByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Country, error) {
	var c Country
	err := pool.QueryRow(ctx, `
		SELECT id, code, alpha3, numeric_code, name, official_name, region, subregion, is_active, created_at
		FROM countries
		WHERE code = $1
	`, code).Scan(
		&c.ID, &c.Code, &c.Alpha3, &c.NumericCode,
		&c.Name, &c.OfficialName, &c.Region, &c.Subregion,
		&c.IsActive, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get country %s: %w", code, err)
	}
	return &c, nil
}

// GetByAlpha3 returns a single country by its ISO-3166 alpha-3 code.
func GetByAlpha3(ctx context.Context, pool *pgxpool.Pool, code string) (*Country, error) {
	var c Country
	err := pool.QueryRow(ctx, `
		SELECT id, code, alpha3, numeric_code, name, official_name, region, subregion, is_active, created_at
		FROM countries
		WHERE alpha3 = $1
	`, code).Scan(
		&c.ID, &c.Code, &c.Alpha3, &c.NumericCode,
		&c.Name, &c.OfficialName, &c.Region, &c.Subregion,
		&c.IsActive, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get country alpha3 %s: %w", code, err)
	}
	return &c, nil
}

// ListByRegion returns countries in a given region.
func ListByRegion(ctx context.Context, pool *pgxpool.Pool, region string) ([]Country, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, alpha3, numeric_code, name, official_name, region, subregion, is_active, created_at
		FROM countries
		WHERE region = $1
		ORDER BY name
	`, region)
	if err != nil {
		return nil, fmt.Errorf("list countries by region %s: %w", region, err)
	}
	defer rows.Close()

	var countries []Country
	for rows.Next() {
		var c Country
		if err := rows.Scan(
			&c.ID, &c.Code, &c.Alpha3, &c.NumericCode,
			&c.Name, &c.OfficialName, &c.Region, &c.Subregion,
			&c.IsActive, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan country: %w", err)
		}
		countries = append(countries, c)
	}
	return countries, rows.Err()
}