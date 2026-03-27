// Package bookmarks implements the core functionality for managing bookmark entries in the Cairn application.
// The package follows the typical structure that entry types follow in this application: handlers.go for HTTP
// route handling, queries.go for database operations and a templates/ directory for HTML forms and views.
// Additionally there is a type-specific logic file here called checker.go which handles the link rot checks.
//
// # Overview
//   - HTTP handlers for bookmark-related routes (creation, viewing, editing, deletion, URL health checks).
//   - Database queries for bookmark CRUD operations and health check results.
//   - URL health checking logic to monitor bookmarked links.
//   - Templates for rendering bookmark forms and views.
//
// # Features
//   - Create, read, update, and delete bookmarks.
//   - Check the health of bookmarked URLs (HTTP status, content hash).
//   - Render bookmark-specific HTML templates for forms and views.
//   - Integrate with the shared `entries` package for tagging and listing.
package bookmarks
