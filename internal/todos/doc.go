// Package todos implements the core functionality for managing todo list entries in the Cairn application.
// The package follows the typical structure that entry types follow in this application: handlers.go for HTTP
// route handling, queries.go for database operations, and a templates/ directory for HTML forms and views.
//
// # Overview
//   - HTTP handlers for todo-related routes (creation, viewing, editing, deletion, toggling items).
//   - Database queries for todo CRUD operations and item management.
//   - Templates for rendering todo list forms and views.
//
// # Features
//   - Create, read, update, and delete todo lists.
//   - Add, toggle, and delete individual todo items.
//   - Render todo-specific HTML templates for forms and views.
//   - Integrate with the shared `entries` package for tagging and listing.
package todos
