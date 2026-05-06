package notes

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemouu/cairn/internal/entries"
)

// RegisterRoutes registers all note-related HTTP routes with the provided ServeMux.
// It maps URLs to their respective handlers for CRUD operations.
func RegisterRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /notes/new", handleForm(pool, false))
	mux.HandleFunc("POST /notes", handleCreate(pool))
	mux.HandleFunc("GET /notes/{id}", handleView(pool))
	mux.HandleFunc("GET /notes/{id}/edit", handleForm(pool, true))
	mux.HandleFunc("POST /notes/{id}", handleUpdate(pool))
	mux.HandleFunc("POST /notes/{id}/delete", handleDelete(pool))
	mux.HandleFunc("GET /notes/{id}/body", handleGetBody(pool))
	mux.HandleFunc("GET /notes/{id}/body/edit", handleEditBody(pool))
	mux.HandleFunc("POST /notes/{id}/body", handleUpdateBody(pool))
}

// handleForm renders the form template for creating or editing a note.
// If isEdit is true, it loads the existing note and its tags for editing.
func handleForm(pool *pgxpool.Pool, isEdit bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Title":  "New Note",
			"IsEdit": isEdit,
		}

		if isEdit {
			id := r.PathValue("id")
			entry, note, err := GetByID(r.Context(), pool, id)
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
			data["Note"] = note
			data["Tags"] = tags
		}

		tmpl, err := template.ParseFiles("templates/layout.html", "internal/notes/templates/form.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Println("template render error:", err)
		}
	}
}

// handleCreate processes the form submission for creating a new note.
// It validates the title, creates the note, sets tags, and redirects to the view page.
func handleCreate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		body := r.FormValue("body")

		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		id, err := Create(r.Context(), pool, title, body)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		tagStr := r.FormValue("tags")
		tagNames := strings.Split(tagStr, ",")
		if err := entries.SetTags(r.Context(), pool, id, tagNames); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/notes/"+id, http.StatusSeeOther)
	}
}

// handleView renders the view template for a note, including its content and tags.
func handleView(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		entry, note, err := GetByID(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		tmpl, err := template.ParseFiles("templates/layout.html", "internal/notes/templates/view.html", "internal/notes/templates/partials/body.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		tags, err := entries.GetTags(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Title": entry.Title,
			"Entry": entry,
			"Note":  note,
			"Tags":  tags,
		}

		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Println("template render error:", err)
		}
	}
}

// handleUpdate processes the form submission for updating a note's title, body, and tags.
// It validates the title, updates the note, sets tags, and redirects to the view page.
func handleUpdate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		body := r.FormValue("body")

		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		if err := Update(r.Context(), pool, id, title, body); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		tagStr := r.FormValue("tags")
		tagNames := strings.Split(tagStr, ",")
		if err := entries.SetTags(r.Context(), pool, id, tagNames); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/notes/"+id, http.StatusSeeOther)
	}
}

// handleDelete removes a note and redirects to the dashboard.
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

// handleGetBody returns the rendered body of a note (used for canceling edit).
func handleGetBody(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		entry, note, err := GetByID(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		tmpl, err := template.ParseFiles("internal/notes/templates/partials/body.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Entry": entry,
			"Note":  note,
		}

		if err := tmpl.ExecuteTemplate(w, "note-body", data); err != nil {
			log.Println("template render error:", err)
		}
	}
}

// handleEditBody returns the edit form for a note body.
func handleEditBody(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		entry, note, err := GetByID(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		tmpl, err := template.ParseFiles("internal/notes/templates/partials/body.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Entry": entry,
			"Note":  note,
		}

		if err := tmpl.ExecuteTemplate(w, "note-body-edit", data); err != nil {
			log.Println("template render error:", err)
		}
	}
}

// handleUpdateBody updates the note body and returns the rendered body.
func handleUpdateBody(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		body := r.FormValue("body")

		if err := UpdateBody(r.Context(), pool, id, body); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		entry, note, err := GetByID(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		tmpl, err := template.ParseFiles("internal/notes/templates/partials/body.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Entry": entry,
			"Note":  note,
		}

		if err := tmpl.ExecuteTemplate(w, "note-body", data); err != nil {
			log.Println("template render error:", err)
		}
	}
}
