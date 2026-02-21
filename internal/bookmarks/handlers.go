package bookmarks

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /bookmarks/new", handleForm(pool, false))
	mux.HandleFunc("POST /bookmarks", handleCreate(pool))
	mux.HandleFunc("GET /bookmarks/{id}", handleView(pool))
	mux.HandleFunc("GET /bookmarks/{id}/edit", handleForm(pool, true))
	mux.HandleFunc("POST /bookmarks/{id}", handleUpdate(pool))
	mux.HandleFunc("POST /bookmarks/{id}/delete", handleDelete(pool))
	mux.HandleFunc("POST /bookmarks/{id}/check", handleCheck(pool))
}

func handleForm(pool *pgxpool.Pool, isEdit bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Title":  "New Bookmark",
			"IsEdit": isEdit,
		}

		if isEdit {
			id := r.PathValue("id")
			entry, bookmark, err := GetByID(r.Context(), pool, id)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			data["Title"] = "Edit – " + entry.Title
			data["Entry"] = entry
			data["Bookmark"] = bookmark
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
		url := r.FormValue("url")

		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		if url == "" {
			http.Error(w, "url is required", http.StatusBadRequest)
			return
		}

		id, err := Create(r.Context(), pool, title, url)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/bookmarks/"+id, http.StatusSeeOther)
	}
}

func handleView(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		entry, bookmark, err := GetByID(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		tmpl, err := template.ParseFiles("templates/layout.html", "internal/bookmarks/templates/view.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Title":    entry.Title,
			"Entry":    entry,
			"Bookmark": bookmark,
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
		url := r.FormValue("url")

		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		if url == "" {
			http.Error(w, "url is required", http.StatusBadRequest)
			return
		}

		if err := Update(r.Context(), pool, id, title, url); err != nil {
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
