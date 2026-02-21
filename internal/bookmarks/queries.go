package bookmarks

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemouu/cairn/internal/entries"
)

type Bookmark struct {
	EntryID       string
	URL           string
	LastStatus    *int
	LastCheckedAt *time.Time
	ContentHash   *string
}

func Create(ctx context.Context, pool *pgxpool.Pool, title, url string) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO entries (entry_type, title) VALUES ('bookmark', $1) RETURNING id`,
		title,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO notes (entry_id, url) VALUES ($1, $2)`,
		id, url,
	)
	if err != nil {
		return "", err
	}

	return id, tx.Commit(ctx)
}

func GetByID(ctx context.Context, pool *pgxpool.Pool, id string) (entries.Entry, Bookmark, error) {
	var e entries.Entry
	var b Bookmark

	err := pool.QueryRow(ctx,
		`SELECT e.id, e.entry_type, e.title, e.created_at, e.updated_at,
            b.url, b.last_status, b.last_checked_at, b.content_hash
     FROM entries e
     JOIN bookmarks b ON b.entry_id = e.id
     WHERE e.id = $1`,
		id,
	).Scan(&e.ID, &e.EntryType, &e.Title, &e.CreatedAt, &e.UpdatedAt,
		&b.URL, &b.LastStatus, &b.LastCheckedAt, &b.ContentHash)

	b.EntryID = e.ID
	return e, b, err
}

func Update(ctx context.Context, pool *pgxpool.Pool, id, title, url string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE entries SET title = $1, updated_at = now() WHERE id = $2`,
		title, id,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE bookmarks SET url = $1 WHERE entry_id = $2`,
		url, id,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func Delete(ctx context.Context, pool *pgxpool.Pool, id string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM entries WHERE id = $1`,
		id,
	)
	return err
}

func UpdateCheckResult(ctx context.Context, pool *pgxpool.Pool, id string, status int, contentHash *string) error {
	_, err := pool.Exec(ctx,
		`UPDATE bookmarks
         SET last_status = $1, last_checked_at = now(), content_hash = $2
         WHERE entry_id = $3`,
		status, contentHash, id,
	)
	return err
}
