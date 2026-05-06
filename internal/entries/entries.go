// Package entries provides shared functionality for managing and querying entry records
// (e.g., bookmarks, notes, todos,...) and their associated tags.
// It includes utilities for listing, searching, and tagging entries across the application.
// This file only requires changes if functionality should be added that concerns all entry
// types.
package entries

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Entry represents a generic entry in the Cairn application.
// All entry types (bookmarks, notes, todos) share this base structure.
type Entry struct {
	ID        string    // Unique identifier for the entry.
	EntryType string    // Type of the entry (e.g., "bookmark", "note", "todo").
	Title     string    // Title or name of the entry.
	CreatedAt time.Time // Timestamp when the entry was created.
	UpdatedAt time.Time // Timestamp when the entry was last updated.
}

// Tag represents a user-defined label that can be associated with entries.
// Tags are shared across all entry types.
type Tag struct {
	ID   string // Unique identifier for the tag.
	Name string // Name of the tag (e.g., "work", "thesis").
}

// ListAll retrieves a slice of entries from the database with pagination support.
// Returns a slice of entries or an error.
func ListAll(ctx context.Context, pool *pgxpool.Pool, limit, offset int) ([]Entry, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, entry_type, title, created_at, updated_at
         FROM entries
         ORDER BY updated_at DESC
         LIMIT $1 OFFSET $2`, limit, offset,
	)
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

// ListByTag retrieves all entries associated with a specific tag with pagination support.
// Returns a slice of entries or an error.
func ListByTag(ctx context.Context, pool *pgxpool.Pool, name string, limit, offset int) ([]Entry, error) {
	rows, err := pool.Query(ctx,
		`SELECT e.id, e.entry_type, e.title, e.created_at, e.updated_at
		 FROM entries e
		 JOIN entry_tags et ON et.entry_id = e.id
		 JOIN tags t ON t.id = et.tag_id
		 WHERE t.name = $1
		 ORDER BY e.updated_at DESC
         LIMIT $2 OFFSET $3`, name, limit, offset,
	)
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

// GetTags retrieves all tags associated with a specific entry.
// Returns a slice of tags or an error.
func GetTags(ctx context.Context, pool *pgxpool.Pool, entryID string) ([]Tag, error) {
	rows, err := pool.Query(ctx,
		`SELECT t.id, t.name
		 FROM tags t
		 JOIN entry_tags et ON et.tag_id = t.id
		 WHERE et.entry_id = $1`, entryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		err := rows.Scan(&t.ID, &t.Name)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// GetTagsForEntries retrieves all tags for a list of entries.
// Returns a map of entry IDs to their associated tags, or an error.
// If entryIDs is empty, returns an empty map.
func GetTagsForEntries(ctx context.Context, pool *pgxpool.Pool, entryIDs []string) (map[string][]Tag, error) {
	if len(entryIDs) == 0 {
		return make(map[string][]Tag), nil
	}

	rows, err := pool.Query(ctx,
		`SELECT et.entry_id, t.id, t.name
         FROM entry_tags et
         JOIN tags t ON t.id = et.tag_id
         WHERE et.entry_id = ANY($1)`, entryIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]Tag)
	for rows.Next() {
		var entryID string
		var t Tag
		if err := rows.Scan(&entryID, &t.ID, &t.Name); err != nil {
			return nil, err
		}
		result[entryID] = append(result[entryID], t)
	}
	return result, rows.Err()
}

// SetTags replaces the tags for a specific entry with the provided tag names.
// It handles tag creation, deduplication, and linking in a single transaction.
// Returns an error if any step fails.
func SetTags(ctx context.Context, pool *pgxpool.Pool, entryID string, tagNames []string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Step 1: Remove all existing tags for this entry
	_, err = tx.Exec(ctx,
		`DELETE FROM entry_tags WHERE entry_id = $1`, entryID,
	)
	if err != nil {
		return err
	}

	// Step 2: For each tag name...
	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// Step 2a: Insert the tag if it doesn't exist, get its ID either way
		var tagID string
		err = tx.QueryRow(ctx,
			`INSERT INTO tags (name) VALUES ($1)
             ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
             RETURNING id`, name,
		).Scan(&tagID)
		if err != nil {
			return err
		}

		// Step 2b: Link the tag to the entry
		_, err = tx.Exec(ctx,
			`INSERT INTO entry_tags (entry_id, tag_id) VALUES ($1, $2)`, entryID, tagID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Search performs a full-text search across all entries with pagination support.
// Returns a slice of entries ranked by relevance, or an error.
func Search(ctx context.Context, pool *pgxpool.Pool, query string, limit, offset int) ([]Entry, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, entry_type, title, created_at, updated_at
		 FROM entries
		 WHERE search_vector @@ plainto_tsquery('english', $1)
		 ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC
         LIMIT $2 OFFSET $3`, query, limit, offset,
	)
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
