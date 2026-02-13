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

func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://cairn:cairn@localhost:5432/cairn?sslmode=disable"
	}
	return pgxpool.New(ctx, url)
}

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
