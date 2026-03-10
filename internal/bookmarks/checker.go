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

// Check all bookmark items for a given entry
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

	// Launch checks in background - don't wait
	// Use a separate context so checks continue even if HTTP request is cancelled
	go func() {
		checkCtx := context.Background()
		for _, i := range items {
			checkItem(checkCtx, pool, i.id, i.url)
		}
	}()

	return nil
}

func checkItem(ctx context.Context, pool *pgxpool.Pool, itemID, url string) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)

	var status int
	var contentHash *string
	if err != nil {
		status = 0
	} else {
		defer resp.Body.Close()
		status = resp.StatusCode
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		hash := sha256.Sum256(body)
		hashStr := hex.EncodeToString(hash[:])
		contentHash = &hashStr
	}

	UpdateCheckResult(ctx, pool, itemID, status, contentHash)
}
