# Cairn

Cairn is a self-hosted personal knowledge hub for managing different entry types with specialized behaviors. It handles plain notes for writing, todo lists with progress tracking, and bookmarks with status monitoring. The project serves as an introduction to Go web development and PostgreSQL.

![Cairn Dashboard](screenshot.png)

## Run

Start the database:

```bash
docker-compose up db
```

Start the app:

```bash
export DATABASE_URL=postgres://cairn:cairn@localhost:5432/cairn?sslmode=disable
go run ./cmd/server/main.go
```

Open [localhost:8080](http://localhost:8080).

Or run the entire stack with Docker:

```bash
docker-compose up
```

## Stack

- Go: Standard library for routing and templates, pgx for PostgreSQL.
- PostgreSQL: Relational storage with full-text search.
- HTMX: Dynamic UI updates without page reloads.
- Alpine.js: Client-side interactivity and state management.
- Bulma: Modern CSS framework for a responsive, matte design.
- Docker Compose: Environment orchestration.

## Design

All entry types share a base entries table, while specific data lives in dedicated tables with foreign key links. This architecture allows shared features like tagging and search to work across all types automatically.

Recent updates have unified the design system using shared Go template partials for icons and UI components, centralized utility classes in CSS, and standardized JavaScript patterns for interactive elements like delete confirmations and theme switching.

## Future Plans

- Background scheduler for automatic bookmark health checks.
- Content drift detection for bookmarks.
- Code snippets with server-side syntax highlighting.
- Browser bookmark import.
- Index card entry type.
- Enhanced logging and monitoring.

## License

MIT — see [LICENSE](LICENSE) for details.
