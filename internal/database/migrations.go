package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations applies all pending goose migrations.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("goose version: %w", err)
	}
	log.Printf("migrations applied — schema version %d", version)

	return nil
}

// Ensure *sql.DB is linked (not used directly, but goose needs the import).
var _ *sql.DB