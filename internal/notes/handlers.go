package notes

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /notes/new", handleForm(pool, false))
	mux.HandleFunc("POST /notes", handleCreate(pool))
	mux.HandleFunc("GET /notes/{id}", handleView(pool))
	mux.HandleFunc("GET /notes/{id}/edit", handleForm(pool, true))
	mux.HandleFunc("POST /notes/{id}", handleUpdate(pool))
	mux.HandleFunc("POST /notes/{id}/delete", handleDelete(pool))
}

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
			data["Title"] = "Edit – " + entry.Title
			data["Entry"] = entry
			data["Note"] = note
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

		http.Redirect(w, r, "/notes/"+id, http.StatusSeeOther)
	}
}

func handleView(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		entry, note, err := GetByID(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		tmpl, err := template.ParseFiles("templates/layout.html", "internal/notes/templates/view.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Title": entry.Title,
			"Entry": entry,
			"Note":  note,
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
		body := r.FormValue("body")

		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		if err := Update(r.Context(), pool, id, title, body); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/notes/"+id, http.StatusSeeOther)
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
