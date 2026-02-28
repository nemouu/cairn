package entries

import (
	"context"
	"strings"
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

type Tag struct {
	ID   string
	Name string
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

func ListByTag(ctx context.Context, pool *pgxpool.Pool, name string) ([]Entry, error) {
	rows, err := pool.Query(ctx,
		`SELECT e.id, e.entry_type, e.title, e.created_at, e.updated_at
		 FROM entries e
		 JOIN entry_tags et ON et.entry_id = e.id
		 JOIN tags t ON t.id = et.tag_id
		 WHERE t.name = $1
		 ORDER BY e.updated_at DESC`, name)
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

func GetTags(ctx context.Context, pool *pgxpool.Pool, entryID string) ([]Tag, error) {
	rows, err := pool.Query(ctx,
		`SELECT t.id, t.name
		 FROM tags t
		 JOIN entry_tags et ON et.tag_id = t.id
		 WHERE et.entry_id = $1`, entryID)
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

func GetTagsForEntries(ctx context.Context, pool *pgxpool.Pool, entryIDs []string) (map[string][]Tag, error) {
	if len(entryIDs) == 0 {
		return make(map[string][]Tag), nil
	}

	rows, err := pool.Query(ctx,
		`SELECT et.entry_id, t.id, t.name
         FROM entry_tags et
         JOIN tags t ON t.id = et.tag_id
         WHERE et.entry_id = ANY($1)`,
		entryIDs)
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

func SetTags(ctx context.Context, pool *pgxpool.Pool, entryID string, tagNames []string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Step 1: Remove all existing tags for this entry
	_, err = tx.Exec(ctx,
		`DELETE FROM entry_tags WHERE entry_id = $1`,
		entryID)
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
             RETURNING id`, name).Scan(&tagID)
		if err != nil {
			return err
		}

		// Step 2b: Link the tag to the entry
		_, err = tx.Exec(ctx,
			`INSERT INTO entry_tags (entry_id, tag_id) VALUES ($1, $2)`,
			entryID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
