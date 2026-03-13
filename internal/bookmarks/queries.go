package bookmarks

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemouu/cairn/internal/entries"
)

type BookmarkItem struct {
	ID            string
	EntryID       string
	URL           string
	Title         *string
	LastStatus    *int
	LastCheckedAt *time.Time
	ContentHash   *string
	Position      int
	CreatedAt     time.Time
}

func Create(ctx context.Context, pool *pgxpool.Pool, title string, urls []string) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO entries (entry_type, title) VALUES ('bookmark', $1) RETURNING id`, title,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	// Insert each URL as a bookmark item (will be empty on initial create)
	for i, url := range urls {
		if url == "" {
			continue
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO bookmark_items (entry_id, url, position) VALUES ($1, $2, $3)`, id, url, i,
		)
		if err != nil {
			return "", err
		}
	}

	return id, tx.Commit(ctx)
}

func GetByID(ctx context.Context, pool *pgxpool.Pool, id string) (entries.Entry, []BookmarkItem, error) {
	var e entries.Entry

	err := pool.QueryRow(ctx,
		`SELECT id, entry_type, title, created_at, updated_at
         FROM entries WHERE id = $1`, id,
	).Scan(&e.ID, &e.EntryType, &e.Title, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return e, nil, err
	}

	rows, err := pool.Query(ctx,
		`SELECT id, entry_id, url, title, last_status, last_checked_at, content_hash, position, created_at
         FROM bookmark_items
         WHERE entry_id = $1
         ORDER BY position`, id,
	)
	if err != nil {
		return e, nil, err
	}
	defer rows.Close()

	var items []BookmarkItem
	for rows.Next() {
		var item BookmarkItem
		err := rows.Scan(&item.ID, &item.EntryID, &item.URL, &item.Title,
			&item.LastStatus, &item.LastCheckedAt, &item.ContentHash,
			&item.Position, &item.CreatedAt)
		if err != nil {
			return e, nil, err
		}
		items = append(items, item)
	}

	return e, items, rows.Err()
}

func UpdateTitle(ctx context.Context, pool *pgxpool.Pool, id, title string) error {
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
		`UPDATE entries SET search_vector = to_tsvector('english',
	        (SELECT e.title || ' ' || COALESCE(
	            string_agg(
	                regexp_replace(regexp_replace(bi.url, '^https?://', ''), '[^a-zA-Z0-9]+', ' ', 'g'),
	                ' '
	            ), '')
	         FROM entries e
	         LEFT JOIN bookmark_items bi ON bi.entry_id = e.id
	         WHERE e.id = $1
	         GROUP BY e.id, e.title))
	     WHERE id = $1`, id,
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

func UpdateCheckResult(ctx context.Context, pool *pgxpool.Pool, itemID string, status int, contentHash *string) error {
	_, err := pool.Exec(ctx,
		`UPDATE bookmark_items
         SET last_status = $1, last_checked_at = now(), content_hash = $2
         WHERE id = $3`, status, contentHash, itemID,
	)
	return err
}

func AddItem(ctx context.Context, pool *pgxpool.Pool, entryID, url string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get current max position
	var maxPos int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), -1) FROM bookmark_items WHERE entry_id = $1`, entryID,
	).Scan(&maxPos)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO bookmark_items (entry_id, url, position) VALUES ($1, $2, $3)`, entryID, url, maxPos+1,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE entries SET search_vector = to_tsvector('english',
	        (SELECT e.title || ' ' || COALESCE(
	            string_agg(
	                regexp_replace(regexp_replace(bi.url, '^https?://', ''), '[^a-zA-Z0-9]+', ' ', 'g'),
	                ' '
	            ), '')
	         FROM entries e
	         LEFT JOIN bookmark_items bi ON bi.entry_id = e.id
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
		`DELETE FROM bookmark_items WHERE id = $1`, itemID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE entries SET search_vector = to_tsvector('english',
			(SELECT e.title || ' ' || COALESCE(
				string_agg(
					regexp_replace(regexp_replace(bi.url, '^https?://', ''), '[^a-zA-Z0-9]+', ' ', 'g'),
					' '
				), '')
			 FROM entries e
			 LEFT JOIN bookmark_items bi ON bi.entry_id = e.id
			 WHERE e.id = $1
			 GROUP BY e.id, e.title))
		 WHERE id = $1`, entryID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
