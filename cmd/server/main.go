package main

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"github.com/nemouu/cairn/internal/database"
	"github.com/nemouu/cairn/internal/entries"
	"github.com/nemouu/cairn/internal/notes"
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
		entryList, err := entries.ListAll(r.Context(), pool)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		tmpl, err := template.ParseFiles("templates/layout.html", "templates/home.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Title":   "Dashboard",
			"Entries": entryList,
		}
		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Println("template render error:", err)
		}
	})

	notes.RegisterRoutes(mux, pool)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
