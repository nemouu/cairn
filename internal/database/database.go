// Package database provides utilities for connecting to the PostgreSQL database
// and running schema migrations.
// It handles connection pooling and tracks applied migrations to ensure idempaticity.
package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect establishes a connection pool to the PostgreSQL database using the DATABASE_URL environment variable.
// If DATABASE_URL is not set, it defaults to a local development URL.
// Returns a *pgxpool.Pool or an error.
func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://cairn:cairn@localhost:5432/cairn?sslmode=disable"
	}
	return pgxpool.New(ctx, url)
}

// RunMigrations applies all pending SQL migrations from the specified directory.
// Migrations are applied in alphabetical order and tracked in the schema_migrations table.
// Returns an error if any migration fails.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	// Create tracking table if it doesn't exist
	_, err := pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            filename   TEXT PRIMARY KEY,
            applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
        )
    `)
	if err != nil {
		return err
	}

	// Read all .sql files from the migrations directory
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	// Apply each migration that hasn't been applied yet
	for _, f := range files {
		name := filepath.Base(f)

		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`,
			name,
		).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		sql, err := os.ReadFile(f)
		if err != nil {
			return err
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("migration %s failed: %w", name, err)
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO schema_migrations (filename) VALUES ($1)`, name,
		)
		if err != nil {
			return err
		}

		log.Printf("applied migration: %s", name)
	}
	return nil
}
