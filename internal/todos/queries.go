package todos

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemouu/cairn/internal/entries"
)

type TodoItem struct {
	ID        string
	EntryID   string
	Body      string
	IsDone    bool
	Position  int
	CreatedAt time.Time
}

func Create(ctx context.Context, pool *pgxpool.Pool, title string) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO entries (entry_type, title) VALUES ('todo', $1) RETURNING id`, title,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx,
		`UPDATE entries SET search_vector = to_tsvector('english', $1)
		 WHERE id = $2`, title, id,
	)
	if err != nil {
		return "", err
	}

	return id, tx.Commit(ctx)
}

func GetByID(ctx context.Context, pool *pgxpool.Pool, id string) (entries.Entry, []TodoItem, error) {
	var e entries.Entry
	var t []TodoItem

	err := pool.QueryRow(ctx,
		`SELECT id, entry_type, title, created_at, updated_at
		 FROM entries
		 WHERE id = $1`, id,
	).Scan(&e.ID, &e.EntryType, &e.Title, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return e, nil, err
	}

	rows, err := pool.Query(ctx,
		`SELECT id, entry_id, body, is_done, position, created_at
   		 FROM todo_items
         WHERE entry_id = $1
         ORDER BY position`, id,
	)
	if err != nil {
		return e, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ti TodoItem
		err := rows.Scan(&ti.ID, &ti.EntryID, &ti.Body, &ti.IsDone, &ti.Position, &ti.CreatedAt)
		if err != nil {
			return e, nil, err
		}
		t = append(t, ti)
	}

	return e, t, err
}

func Update(ctx context.Context, pool *pgxpool.Pool, id, title string) error {
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
		`UPDATE entries SET search_vector = to_tsvector('english', $1)
     	 WHERE id = $2`, title, id,
	)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func AddItem(ctx context.Context, pool *pgxpool.Pool, entryID string, body string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO todo_items (entry_id, body, position)
         VALUES ($1, $2, COALESCE((SELECT MAX(position)
         FROM todo_items
         WHERE entry_id = $1), 0) + 1)`, entryID, body,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE entries SET search_vector = to_tsvector('english',
	        (SELECT e.title || ' ' || COALESCE(
	            string_agg(ti.body, ' '), '')
	         FROM entries e
	         LEFT JOIN todo_items ti ON ti.entry_id = e.id
	         WHERE e.id = $1
	         GROUP BY e.id, e.title))
	     WHERE id = $1`, entryID,
	)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func ToggleItem(ctx context.Context, pool *pgxpool.Pool, itemID string) error {
	_, err := pool.Exec(ctx,
		`UPDATE todo_items
		 SET is_done = NOT is_done
		 WHERE id = $1`, itemID,
	)
	return err
}

func UpdateItem(ctx context.Context, pool *pgxpool.Pool, entryID, itemID, body string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE todo_items SET body = $1 WHERE id = $2`, body, itemID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE entries SET search_vector = to_tsvector('english',
			(SELECT e.title || ' ' || COALESCE(
				string_agg(ti.body, ' '), '')
			 FROM entries e
			 LEFT JOIN todo_items ti ON ti.entry_id = e.id
			 WHERE e.id = $1
			 GROUP BY e.id, e.title))
		 WHERE id = $1`, entryID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func DeleteItem(ctx context.Context, pool *pgxpool.Pool, entryID, itemID string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`DELETE FROM todo_items WHERE id = $1`, itemID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE entries SET search_vector = to_tsvector('english',
			(SELECT e.title || ' ' || COALESCE(
				string_agg(ti.body, ' '), '')
			 FROM entries e
			 LEFT JOIN todo_items ti ON ti.entry_id = e.id
			 WHERE e.id = $1
			 GROUP BY e.id, e.title))
		 WHERE id = $1`, entryID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func Delete(ctx context.Context, pool *pgxpool.Pool, id string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM entries WHERE id = $1`, id,
	)
	return err
}
