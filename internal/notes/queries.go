package notes

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemouu/cairn/internal/entries"
)

type Note struct {
	EntryID string
	Body    string
}

func Create(ctx context.Context, pool *pgxpool.Pool, title, body string) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO entries (entry_type, title) VALUES ('note', $1) RETURNING id`,
		title,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO notes (entry_id, body) VALUES ($1, $2)`,
		id, body,
	)
	if err != nil {
		return "", err
	}

	return id, tx.Commit(ctx)
}

func GetByID(ctx context.Context, pool *pgxpool.Pool, id string) (entries.Entry, Note, error) {
	var e entries.Entry
	var n Note

	err := pool.QueryRow(ctx,
		`SELECT e.id, e.entry_type, e.title, e.created_at, e.updated_at, n.body
         FROM entries e
         JOIN notes n ON n.entry_id = e.id
         WHERE e.id = $1`,
		id,
	).Scan(&e.ID, &e.EntryType, &e.Title, &e.CreatedAt, &e.UpdatedAt, &n.Body)

	n.EntryID = e.ID
	return e, n, err
}

func Update(ctx context.Context, pool *pgxpool.Pool, id, title, body string) error {
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
		`UPDATE notes SET body = $1 WHERE entry_id = $2`,
		body, id,
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
