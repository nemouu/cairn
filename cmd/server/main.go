// Cairn is a self-hosted personal knowledge management application.
//
// It provides three core entry types so far:
//   - Notes: Simple text notes with full-text search
//   - Bookmarks: Collections of URLs with link checking
//   - Todos: Task lists with checkable items
//
// Additions for the entry types so far are:
//   - Learning Cards: Collection of learning cards with two sides
//
// All entries support tagging for organization and can be searched using
// PostgreSQL's full-text search capabilities.
//
// The application uses Go with the standard library HTTP server, PostgreSQL
// for storage, and server-side rendered HTML templates with minimal JavaScript.
//
// Usage:
//
//	docker-compose up
//
// Note that the first time might take a bit longer since the containers are
// installed. The application will be available at http://localhost:8080
//
// Structure and how to add entry types:
//
// The internal folder contains general files for the database and entries
// and also subfolders for each entry type. This is where you can add a new
// entry type together with everything it needs to function. The migrations
// folder contains SQL migrations for the database and you will also have to
// add a migration for any entry type you want to add. Finally static and
// templates contain the general .css and .html files Cairn uses which also
// need adjusting in case you want to add an entry type.
package main

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"github.com/nemouu/cairn/internal/bookmarks"
	"github.com/nemouu/cairn/internal/database"
	"github.com/nemouu/cairn/internal/entries"
	"github.com/nemouu/cairn/internal/notes"
	"github.com/nemouu/cairn/internal/todos"
)

// main initializes the application, sets up the database, registers routes,
// and starts the HTTP server. It handles the dashboard and tag-filtered views,
// and delegates note/todo/bookmark routes to their respective packages.
//
// When adding an entry type make sure to add the register the route of the added
// package along with the other ones.
func main() {
	ctx := context.Background()

	// Connect to database and run migrations
	pool, err := database.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := database.RunMigrations(ctx, pool, "migrations"); err != nil {
		log.Fatal(err)
	}

	// Set up routes
	mux := http.NewServeMux()

	// Static files (CSS)
	mux.Handle("GET /static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	// Dashboard handler: lists all entries or filters by search query
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q") // Get search query from URL

		var entryList []entries.Entry
		var err error

		// Search mode or normal mode
		if query != "" {
			entryList, err = entries.Search(r.Context(), pool, query)
		} else {
			entryList, err = entries.ListAll(r.Context(), pool)
		}

		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		// Collect entry IDs for tag lookup
		var ids []string
		for _, e := range entryList {
			ids = append(ids, e.ID)
		}

		// Fetch tags for all entries in the current view
		entryTags, err := entries.GetTagsForEntries(r.Context(), pool, ids)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		// Render template with data or return an error
		var tmpl *template.Template
		if r.Header.Get("HX-Request") == "true" {
			tmpl, err = template.ParseFiles("templates/partials/entry_list.html")
			if err != nil {
				http.Error(w, "template error", http.StatusInternalServerError)
				return
			}
			data := map[string]any{
				"Entries":   entryList,
				"EntryTags": entryTags,
				"Query":     query,
			}
			if err := tmpl.ExecuteTemplate(w, "entry-list", data); err != nil {
				log.Println("template render error:", err)
			}
			return
		}

		tmpl, err = template.ParseFiles("templates/layout.html", "templates/home.html", "templates/partials/entry_list.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Title":     "Dashboard",
			"Entries":   entryList,
			"EntryTags": entryTags,
			"Query":     query,
		}

		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Println("template render error:", err)
		}
	})

	// Tag handler: lists entries filtered by a specific tag
	mux.HandleFunc("GET /tags/{name}", func(w http.ResponseWriter, r *http.Request) {
		tagName := r.PathValue("name")

		entriesByTagsList, err := entries.ListByTag(r.Context(), pool, tagName)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		// Collect entry IDs for tag lookup
		var ids []string
		for _, e := range entriesByTagsList {
			ids = append(ids, e.ID)
		}

		// Fetch tags for all entries in the current view
		entryTags, err := entries.GetTagsForEntries(r.Context(), pool, ids)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		// Render template with data or return an error
		var tmpl *template.Template
		if r.Header.Get("HX-Request") == "true" {
			tmpl, err = template.ParseFiles("templates/partials/entry_list.html")
			if err != nil {
				http.Error(w, "template error", http.StatusInternalServerError)
				return
			}
			data := map[string]any{
				"Entries":   entriesByTagsList,
				"EntryTags": entryTags,
			}
			if err := tmpl.ExecuteTemplate(w, "entry-list", data); err != nil {
				log.Println("template render error:", err)
			}
			return
		}

		tmpl, err = template.ParseFiles("templates/layout.html", "templates/home.html", "templates/partials/entry_list.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Title":     "Tag: " + tagName,
			"Entries":   entriesByTagsList,
			"EntryTags": entryTags,
		}

		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Println("template render error:", err)
		}
	})

	// Register type-specific routes. This is where you add a new entry type!
	notes.RegisterRoutes(mux, pool)
	todos.RegisterRoutes(mux, pool)
	bookmarks.RegisterRoutes(mux, pool)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
