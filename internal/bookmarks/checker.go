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

	for rows.Next() {
		var itemID, url string
		if err := rows.Scan(&itemID, &url); err != nil {
			continue
		}

		go checkItem(ctx, pool, itemID, url)
	}

	return rows.Err()
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
