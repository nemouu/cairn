// Package notes implements the core functionality for managing note entries in the Cairn application.
// The package follows the typical structure that entry types follow in this application: handlers.go for HTTP
// route handling, queries.go for database operations, and a templates/ directory for HTML forms and views.
//
// # Overview
//   - HTTP handlers for note-related routes (creation, viewing, editing, deletion).
//   - Database queries for note CRUD operations.
//   - Templates for rendering note forms and views.
//
// # Features
//   - Create, read, update, and delete notes.
//   - Render note-specific HTML templates for forms and views.
//   - Integrate with the shared `entries` package for tagging and listing.
package notes
