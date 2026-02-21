package todos

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /todos/new", handleForm(pool, false))
	mux.HandleFunc("POST /todos", handleCreate(pool))
	mux.HandleFunc("GET /todos/{id}", handleView(pool))
	mux.HandleFunc("GET /todos/{id}/edit", handleForm(pool, true))
	mux.HandleFunc("POST /todos/{id}", handleUpdate(pool))
	mux.HandleFunc("POST /todos/{id}/delete", handleDelete(pool))
	mux.HandleFunc("POST /todos/{id}/items", handleAddItem(pool))
	mux.HandleFunc("POST /todos/{id}/items/{itemID}/update", handleUpdateItem(pool))
	mux.HandleFunc("POST /todos/{id}/items/{itemID}/toggle", handleToggleItem(pool))
	mux.HandleFunc("POST /todos/{id}/items/{itemID}/delete", handleDeleteItem(pool))
}

// handleForm
func handleForm(pool *pgxpool.Pool, isEdit bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Title":  "New Todo",
			"IsEdit": isEdit,
		}

		if isEdit {
			id := r.PathValue("id")
			entry, _, err := GetByID(r.Context(), pool, id)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			data["Title"] = "Edit – " + entry.Title
			data["Entry"] = entry
		}

		tmpl, err := template.ParseFiles("templates/layout.html", "internal/todos/templates/form.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Println("template render error:", err)
		}
	}
}

// handleCreate
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

		id, err := Create(r.Context(), pool, title)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/todos/"+id, http.StatusSeeOther)
	}
}

// handleView
func handleView(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		entry, todoItems, err := GetByID(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		tmpl, err := template.ParseFiles("templates/layout.html", "internal/todos/templates/view.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Title": entry.Title,
			"Entry": entry,
			"Todo":  todoItems,
		}
		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Println("template render error:", err)
		}
	}
}

// handleUpdate
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

		if err := Update(r.Context(), pool, id, title); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/todos/"+id, http.StatusSeeOther)
	}
}

// handleDelete
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

// handleAddItem
func handleAddItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		body := r.FormValue("body")

		if body == "" {
			http.Error(w, "body is required", http.StatusBadRequest)
			return
		}

		if err := AddItem(r.Context(), pool, id, body); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/todos/"+id, http.StatusSeeOther)
	}
}

// handleUpdateItem
func handleUpdateItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		itemID := r.PathValue("itemID")

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		body := r.FormValue("body")

		if body == "" {
			http.Error(w, "body is required", http.StatusBadRequest)
			return
		}

		if err := UpdateItem(r.Context(), pool, itemID, body); err != nil {
			log.Println("update item error:", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/todos/"+id, http.StatusSeeOther)
	}
}

// handleToggleItem
func handleToggleItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		itemID := r.PathValue("itemID")

		if err := ToggleItem(r.Context(), pool, itemID); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/todos/"+id, http.StatusSeeOther)
	}
}

// handleDeleteItem
func handleDeleteItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		itemID := r.PathValue("itemID")

		if err := DeleteItem(r.Context(), pool, itemID); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/todos/"+id, http.StatusSeeOther)
	}
}
