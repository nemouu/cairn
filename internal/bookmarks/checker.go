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

func Check(ctx context.Context, pool *pgxpool.Pool, entryID string) error {
	var url string
	err := pool.QueryRow(ctx,
		`SELECT url FROM bookmarks WHERE entry_id = $1`,
		entryID,
	).Scan(&url)
	if err != nil {
		return err
	}

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

	return UpdateCheckResult(ctx, pool, entryID, status, contentHash)
}
