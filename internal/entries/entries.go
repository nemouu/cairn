package entries

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Entry struct {
	ID        string
	EntryType string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Entry, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, entry_type, title, created_at, updated_at
         FROM entries
         ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		err := rows.Scan(&e.ID, &e.EntryType, &e.Title, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
