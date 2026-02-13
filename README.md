# Cairn

Cairn is a self-hosted personal knowledge hub where different entry types have different behaviors like plain notes for freeform writing, todo lists with checkable items, and bookmarks that monitor themselves for link rot. Built as an introduction to Go web development and PostgreSQL, coming from a background in embedded systems and mobile development.

## Run

Start the database:

```
docker-compose up db
```

Start the app:

```
export DATABASE_URL=postgres://cairn:cairn@localhost:5432/cairn?sslmode=disable
go run ./cmd/server
```

Open [localhost:8080](http://localhost:8080).

Or run everything in Docker (coming soon):

```
docker-compose up
```

## Stack

- **Go** — standard library for HTTP routing and templates, pgx for PostgreSQL
- **PostgreSQL**
- **Docker Compose**

## Design

All entry types share a common `entries` table. Each type has its own table with type-specific columns and a foreign key back to `entries`. Shared features like tagging and search operate on the base table and work across all types automatically.

Adding a new entry type = one SQL migration + one Go package. No changes to existing code.

## Future Ideas

- Background scheduler for automatic bookmark health checks
- Content drift detection (page returns 200 but content has changed)
- Bookmark check history with status over time
- Reading notes with source tracking (URL, DOI, ISBN)
- Code snippets with server-side syntax highlighting
- Decision logs for recording reasoning behind choices
- Cross-references between entries
- Full-text search via PostgreSQL `tsvector`
- Turbo/Hotwire for snappy partial page updates
- Browser bookmark import

## License

MIT — see [LICENSE](LICENSE) for details.
