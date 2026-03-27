package bookmarks

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemouu/cairn/internal/entries"
)

// BookmarkItem represents a single URL associated with a bookmark entry in Cairn.
// It includes metadata for health checks (status, last checked time, content hash)
// and display purposes (title, position).
// All fields are safe for concurrent use by multiple goroutines.
type BookmarkItem struct {
	ID            string     // Unique identifier for the bookmark item.
	EntryID       string     // ID of the parent bookmark entry.
	URL           string     // URL of the bookmarked resource.
	Title         *string    // Optional title for the bookmark item.
	LastStatus    *int       // HTTP status code from the last health check (nil if never checked).
	LastCheckedAt *time.Time // Timestamp of the last health check (nil if never checked).
	ContentHash   *string    // SHA-256 hash of the response body from the last check (nil if never checked or unreachable).
	Position      int        // Position of the item in the bookmark list (used for ordering).
	CreatedAt     time.Time  // Timestamp when the bookmark item was created.
}

// Create inserts a new bookmark entry and its associated URLs into the database.
// It returns the ID of the created entry or an error.
// URLs are added as bookmark items with sequential positions.
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

	// Insert each URL as a bookmark item (skips empty URLs)
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

// GetByID retrieves a bookmark entry and its associated items by ID.
// Returns the entry, a slice of bookmark items, or an error.
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

// UpdateTitle updates the title of a bookmark entry and refreshes its search vector.
// The search vector includes the title and all associated URLs for full-text search.
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

	// Update search vector to include the new title and all URLs
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

// Delete removes a bookmark entry and all its associated items from the database.
// Uses CASCADE to delete related rows in bookmark_items.
func Delete(ctx context.Context, pool *pgxpool.Pool, id string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM entries WHERE id = $1`, id,
	)
	return err
}

// UpdateCheckResult updates the health check status, timestamp, and content hash for a bookmark item.
func UpdateCheckResult(ctx context.Context, pool *pgxpool.Pool, itemID string, status int, contentHash *string) error {
	_, err := pool.Exec(ctx,
		`UPDATE bookmark_items
         SET last_status = $1, last_checked_at = now(), content_hash = $2
         WHERE id = $3`, status, contentHash, itemID,
	)
	return err
}

// AddItem appends a new URL to a bookmark entry, placing it at the end of the list.
// It also updates the entry's search vector to include the new URL.
func AddItem(ctx context.Context, pool *pgxpool.Pool, entryID, url string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get current max position to place the new item at the end
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

	// Update search vector to include the new URL
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

// DeleteItem removes a URL from a bookmark entry and refreshes the search vector.
// The search vector is updated to reflect the remaining URLs.
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

	// Update search vector to exclude the deleted URL
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
