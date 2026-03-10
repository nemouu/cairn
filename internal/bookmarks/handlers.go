package bookmarks

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemouu/cairn/internal/entries"
)

func RegisterRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /bookmarks/new", handleForm(pool, false))
	mux.HandleFunc("POST /bookmarks", handleCreate(pool))
	mux.HandleFunc("GET /bookmarks/{id}", handleView(pool))
	mux.HandleFunc("GET /bookmarks/{id}/edit", handleForm(pool, true))
	mux.HandleFunc("POST /bookmarks/{id}", handleUpdate(pool))
	mux.HandleFunc("POST /bookmarks/{id}/delete", handleDelete(pool))
	mux.HandleFunc("POST /bookmarks/{id}/check", handleCheck(pool))

	// New routes for managing individual bookmark items
	mux.HandleFunc("POST /bookmarks/{id}/items", handleAddItem(pool))
	mux.HandleFunc("POST /bookmarks/{entryID}/items/{itemID}/delete", handleDeleteItem(pool))
}

func handleForm(pool *pgxpool.Pool, isEdit bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Title":  "New Bookmark",
			"IsEdit": isEdit,
		}

		if isEdit {
			id := r.PathValue("id")
			entry, _, err := GetByID(r.Context(), pool, id)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			tags, err := entries.GetTags(r.Context(), pool, id)
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			data["Title"] = "Edit – " + entry.Title
			data["Entry"] = entry
			data["Tags"] = tags
		}

		tmpl, err := template.ParseFiles("templates/layout.html", "internal/bookmarks/templates/form.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Println("template render error:", err)
		}
	}
}

func handleCreate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		// Create bookmark entry with no URLs initially (like creating a todo)
		id, err := Create(r.Context(), pool, title, []string{})
		if err != nil {
			log.Println("bookmark create error:", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		tagStr := r.FormValue("tags")
		tagNames := strings.Split(tagStr, ",")
		if err := entries.SetTags(r.Context(), pool, id, tagNames); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		// Redirect to view page where user can add URLs
		http.Redirect(w, r, "/bookmarks/"+id, http.StatusSeeOther)
	}
}

func handleView(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		entry, items, err := GetByID(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		tags, err := entries.GetTags(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		tmpl, err := template.ParseFiles("templates/layout.html", "internal/bookmarks/templates/view.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Title":         entry.Title,
			"Entry":         entry,
			"BookmarkItems": items,
			"Tags":          tags,
		}

		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Println("template render error:", err)
		}
	}
}

func handleUpdate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		// Only update title (URLs managed separately via add/delete item)
		if err := UpdateTitle(r.Context(), pool, id, title); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		tagStr := r.FormValue("tags")
		tagNames := strings.Split(tagStr, ",")
		if err := entries.SetTags(r.Context(), pool, id, tagNames); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/bookmarks/"+id, http.StatusSeeOther)
	}
}

func handleDelete(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if err := Delete(r.Context(), pool, id); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func handleCheck(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := Check(r.Context(), pool, id); err != nil {
			http.Error(w, "check failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/bookmarks/"+id, http.StatusSeeOther)
	}
}

func handleAddItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entryID := r.PathValue("id")

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		url := strings.TrimSpace(r.FormValue("url"))
		if url == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		if err := AddItem(r.Context(), pool, entryID, url); err != nil {
			log.Println("add bookmark item error:", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/bookmarks/"+entryID, http.StatusSeeOther)
	}
}

func handleDeleteItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entryID := r.PathValue("entryID")
		itemID := r.PathValue("itemID")

		if err := DeleteItem(r.Context(), pool, itemID); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/bookmarks/"+entryID, http.StatusSeeOther)
	}
}
