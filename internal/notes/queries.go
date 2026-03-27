package notes

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemouu/cairn/internal/entries"
)

// Note represents the content of a note entry.
// It is associated with an Entry record for metadata (title, tags, timestamps).
type Note struct {
	EntryID string // ID of the parent entry record.
	Body    string // Markdown or plain text content of the note.
}

// Create inserts a new note entry and its content into the database.
// It creates both the entry record and the note content, then updates the search vector.
// Returns the ID of the created entry or an error.
func Create(ctx context.Context, pool *pgxpool.Pool, title, body string) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO entries (entry_type, title) VALUES ('note', $1) RETURNING id`, title,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO notes (entry_id, body) VALUES ($1, $2)`, id, body,
	)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx,
		`UPDATE entries SET search_vector = to_tsvector('english', $1 || ' ' || $2)
     	 WHERE id = $3`, title, body, id,
	)
	if err != nil {
		return "", err
	}

	return id, tx.Commit(ctx)
}

// GetByID retrieves a note entry and its content by ID.
// Returns the entry metadata, note content, or an error.
func GetByID(ctx context.Context, pool *pgxpool.Pool, id string) (entries.Entry, Note, error) {
	var e entries.Entry
	var n Note

	err := pool.QueryRow(ctx,
		`SELECT e.id, e.entry_type, e.title, e.created_at, e.updated_at, n.body
         FROM entries e
         JOIN notes n ON n.entry_id = e.id
         WHERE e.id = $1`, id,
	).Scan(&e.ID, &e.EntryType, &e.Title, &e.CreatedAt, &e.UpdatedAt, &n.Body)

	n.EntryID = e.ID
	return e, n, err
}

// Update modifies the title and body of an existing note.
// It updates both the entry record and the note content, then refreshes the search vector.
// Returns an error if any step fails.
func Update(ctx context.Context, pool *pgxpool.Pool, id, title, body string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE entries SET title = $1, updated_at = now() WHERE id = $2`, title, id,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE notes SET body = $1 WHERE entry_id = $2`, body, id,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE entries SET search_vector = to_tsvector('english', $1 || ' ' || $2)
     	 WHERE id = $3`, title, body, id,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Delete removes a note entry and its associated content from the database.
// Uses CASCADE to delete related rows in the notes table.
func Delete(ctx context.Context, pool *pgxpool.Pool, id string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM entries WHERE id = $1`, id,
	)
	return err
}
