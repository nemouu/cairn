package bookmarks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Check initiates asynchronous health checks for all bookmarks associated with the given entry.
// It queries the database for bookmark items, then launches a goroutine to check each URL.
// The checks continue even if the HTTP request context is cancelled.
func Check(ctx context.Context, pool *pgxpool.Pool, entryID string) error {
	rows, err := pool.Query(ctx,
		`SELECT id, url FROM bookmark_items WHERE entry_id = $1`,
		entryID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id  string
		url string
	}

	var items []item
	for rows.Next() {
		var i item
		if err := rows.Scan(&i.id, &i.url); err != nil {
			continue
		}
		items = append(items, i)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Launch checks in the background using a separate context.
	// This ensures checks complete even if the original HTTP request is cancelled.
	go func() {
		checkCtx := context.Background()
		for _, i := range items {
			checkItem(checkCtx, pool, i.id, i.url)
		}
	}()

	return nil
}

// checkItem performs a health check for a single bookmark URL.
// It records the HTTP status code and a SHA-256 hash of the response body (up to 1MB)
// in the database via UpdateCheckResult.
func checkItem(ctx context.Context, pool *pgxpool.Pool, itemID, url string) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)

	var status int
	var contentHash *string
	if err != nil {
		status = 0 // 0 indicates an unreachable URL
	} else {
		defer resp.Body.Close()
		status = resp.StatusCode
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // Limit to 1MB for memory efficiency and performance
		hash := sha256.Sum256(body)
		hashStr := hex.EncodeToString(hash[:])
		contentHash = &hashStr
	}

	UpdateCheckResult(ctx, pool, itemID, status, contentHash)
}
