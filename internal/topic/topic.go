package topic

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Topic represents a row from the topics table.
type Topic struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	ParentID    *string    `json:"parent_id,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListAll returns all topics, ordered by name.
func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Topic, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM topics
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}
	defer rows.Close()

	var topics []Topic
	for rows.Next() {
		var t Topic
		if err := rows.Scan(
			&t.ID, &t.Code, &t.Name, &t.Description,
			&t.ParentID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		topics = append(topics, t)
	}
	return topics, rows.Err()
}

// ListActive returns active topics.
func ListActive(ctx context.Context, pool *pgxpool.Pool) ([]Topic, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM topics
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active topics: %w", err)
	}
	defer rows.Close()

	var topics []Topic
	for rows.Next() {
		var t Topic
		if err := rows.Scan(
			&t.ID, &t.Code, &t.Name, &t.Description,
			&t.ParentID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		topics = append(topics, t)
	}
	return topics, rows.Err()
}

// GetByCode returns a single topic by its code.
func GetByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Topic, error) {
	var t Topic
	err := pool.QueryRow(ctx, `
		SELECT id, code, name, description, parent_id, is_active, created_at, updated_at
		FROM topics
		WHERE code = $1
	`, code).Scan(
		&t.ID, &t.Code, &t.Name, &t.Description,
		&t.ParentID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get topic %s: %w", code, err)
	}
	return &t, nil
}

// ListByParent returns topics under a given parent code.
func ListByParent(ctx context.Context, pool *pgxpool.Pool, parentCode string) ([]Topic, error) {
	rows, err := pool.Query(ctx, `
		SELECT t.id, t.code, t.name, t.description, t.parent_id, t.is_active, t.created_at, t.updated_at
		FROM topics t
		JOIN topics p ON p.id = t.parent_id
		WHERE p.code = $1
		ORDER BY t.name
	`, parentCode)
	if err != nil {
		return nil, fmt.Errorf("list topics by parent %s: %w", parentCode, err)
	}
	defer rows.Close()

	var topics []Topic
	for rows.Next() {
		var t Topic
		if err := rows.Scan(
			&t.ID, &t.Code, &t.Name, &t.Description,
			&t.ParentID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		topics = append(topics, t)
	}
	return topics, rows.Err()
}