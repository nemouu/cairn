package main

import (
	"context"
	"log"
	"net/http"

	"github.com/nemouu/cairn/internal/database"
)

func main() {
	ctx := context.Background()

	// Connect to database
	pool, err := database.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Run migrations
	if err := database.RunMigrations(ctx, pool, "migrations"); err != nil {
		log.Fatal(err)
	}

	// Set up routes
	mux := http.NewServeMux()

	// Static files (CSS)
	mux.Handle("GET /static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	// Dashboard
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Cairn is running!"))
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
