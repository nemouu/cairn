package todos

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemouu/cairn/internal/entries"
)

// RegisterRoutes registers all todo-related HTTP routes with the provided ServeMux.
// It maps URLs to their respective handlers for CRUD operations and item management.
func RegisterRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /todos/new", handleForm(pool, false))
	mux.HandleFunc("POST /todos", handleCreate(pool))
	mux.HandleFunc("GET /todos/{id}", handleView(pool))
	mux.HandleFunc("GET /todos/{id}/edit", handleForm(pool, true))
	mux.HandleFunc("POST /todos/{id}", handleUpdate(pool))
	mux.HandleFunc("POST /todos/{id}/delete", handleDelete(pool))
	mux.HandleFunc("POST /todos/{id}/items", handleAddItem(pool))
	mux.HandleFunc("POST /todos/{id}/items/{itemID}/toggle", handleToggleItem(pool))
	mux.HandleFunc("POST /todos/{id}/items/{itemID}/delete", handleDeleteItem(pool))
}

// handleForm renders the form template for creating or editing a todo list.
// If isEdit is true, it loads the existing todo and its tags for editing.
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
			tags, err := entries.GetTags(r.Context(), pool, id)
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			data["Title"] = "Edit – " + entry.Title
			data["Entry"] = entry
			data["Tags"] = tags
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

// handleCreate processes the form submission for creating a new todo list.
// It validates the title, creates the todo, sets tags, and redirects to the view page.
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

		tagStr := r.FormValue("tags")
		tagNames := strings.Split(tagStr, ",")
		if err := entries.SetTags(r.Context(), pool, id, tagNames); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/todos/"+id, http.StatusSeeOther)
	}
}

// handleView renders the view template for a todo list, including its items and tags.
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

		tags, err := entries.GetTags(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Title": entry.Title,
			"Entry": entry,
			"Todo":  todoItems,
			"Tags":  tags,
		}

		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Println("template render error:", err)
		}
	}
}

// handleUpdate processes the form submission for updating a todo list's title and tags.
// It validates the title, updates the todo, sets tags, and redirects to the view page.
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

		tagStr := r.FormValue("tags")
		tagNames := strings.Split(tagStr, ",")
		if err := entries.SetTags(r.Context(), pool, id, tagNames); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/todos/"+id, http.StatusSeeOther)
	}
}

// handleDelete removes a todo list and redirects to the dashboard.
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

// handleAddItem processes the form submission for adding a new item to a todo list.
// It validates the item body, adds the item, and redirects to the view page.
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

// handleToggleItem toggles the completion status of a todo item.
// It updates the item's IsDone status and redirects to the view page.
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

// handleDeleteItem removes a todo item from the list and redirects to the view page.
func handleDeleteItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		itemID := r.PathValue("itemID")

		if err := DeleteItem(r.Context(), pool, id, itemID); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/todos/"+id, http.StatusSeeOther)
	}
}
