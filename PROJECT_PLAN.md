# Cairn — Personal Knowledge Hub

A self-hosted personal journaling application built with Go, PostgreSQL, and server-side rendering. Multiple note types with different behaviors: plain notes, todo lists, and bookmarks with link-health monitoring.

---

## Tech Stack

- **Backend:** Go (standard library for HTTP routing and templates)
- **Database:** PostgreSQL 17 (hand-written SQL, no ORM)
- **Driver:** pgx/v5 (connection pooling via pgxpool)
- **Infrastructure:** Docker Compose (PostgreSQL in container, app runs on host during dev)
- **Templates:** Go `html/template` with layout composition
- **Styling:** Single CSS file, minimal and clean
- **Migrations:** Hand-rolled runner tracking applied files in a `schema_migrations` table

---

## Architecture Overview

### Database Design (Option B: Shared Base Table + Type Tables)

All entry types share a common `entries` table for identity, timestamps, and type. Each type has its own table with a foreign key back to `entries`. Cross-cutting features (tags, search, cross-references) operate on the shared `entries` table, so they work for all types automatically — existing and future.

```
entries (shared)
├── notes (entry_id FK → entries)
├── bookmarks (entry_id FK → entries)
├── todo_items (entry_id FK → entries)
├── tags / entry_tags (shared tagging)
└── entry_links (cross-references between any entries)
```

Adding a new note type in the future = one new migration + one new Go package. No changes to existing code.

### Go Project Structure

```
cairn/
├── cmd/server/main.go            -- entry point
├── internal/
│   ├── database/database.go      -- connection + migration runner
│   ├── entries/entries.go        -- shared queries (list all, delete, tags)
│   ├── notes/
│   │   ├── handlers.go
│   │   ├── queries.go
│   │   └── templates/
│   ├── bookmarks/
│   │   ├── handlers.go
│   │   ├── queries.go
│   │   ├── checker.go            -- HTTP health check logic
│   │   └── templates/
│   └── todos/
│       ├── handlers.go
│       ├── queries.go
│       └── templates/
├── templates/
│   ├── layout.html               -- shared HTML shell
│   ├── home.html                 -- dashboard
│   └── components/               -- reusable fragments
├── static/style.css
├── migrations/
│   ├── 001_entries.sql
│   ├── 002_initial_types.sql
│   ├── 003_tags.sql
│   └── 004_entry_links.sql
├── docker-compose.yml
├── .gitignore
├── go.mod
└── README.md
```

### Data Storage

PostgreSQL data lives in a **Docker named volume** (`pgdata`), managed by Docker on the host
machine. The data is NOT in the project directory — there are no database files to gitignore.

The volume persists across `docker-compose down` and `docker-compose up`. To destroy data
intentionally, use `docker-compose down -v` (the `-v` flag removes volumes). Be careful with this.

### .gitignore

```gitignore
# Go binary
/cairn
/cmd/server/server

# IDE
.vscode/
.idea/

# OS
.DS_Store

# Environment overrides
.env
docker-compose.override.yml
```

### UI Flow

The dashboard is the single main page. All entries are listed here, sorted by most recently updated.

```
Dashboard (GET /)
│
├── [+] button (top of page)
│   └── Clicking reveals a dropdown/popover with entry type choices:
│       ├── Note       → navigates to /notes/new
│       ├── Bookmark   → navigates to /bookmarks/new
│       └── Todo List  → navigates to /todos/new
│
├── Entry list
│   ├── Each entry shows: type badge, title, date
│   ├── Clicking an entry → navigates to its view page (/notes/{id}, etc.)
│   └── View page has Edit and Delete actions
│
└── Empty state when no entries exist
```

The [+] button type selector is a small HTML element that toggles visibility with minimal
JavaScript (a single click handler toggling a CSS class). No separate page needed. This is the
only JavaScript in the weekend build — everything else is plain HTML forms and links.

Implementation: a `<div>` containing the three type links, hidden by default with CSS
(`display: none`), toggled visible when the [+] button is clicked. Keyboard accessible
because the choices are `<a>` tags.

### URL Routes

```
GET  /                              Dashboard (all entries, sorted by updated_at)

GET  /notes/new                     New note form
POST /notes                         Create note
GET  /notes/{id}                    View note
GET  /notes/{id}/edit               Edit note form
POST /notes/{id}                    Update note
POST /notes/{id}/delete             Delete note

GET  /bookmarks/new                 New bookmark form
POST /bookmarks                     Create bookmark
GET  /bookmarks/{id}                View bookmark (shows check history)
GET  /bookmarks/{id}/edit           Edit bookmark form
POST /bookmarks/{id}                Update bookmark
POST /bookmarks/{id}/delete         Delete bookmark
POST /bookmarks/{id}/check          Trigger manual link check

GET  /todos/new                     New todo list form
POST /todos                         Create todo list
GET  /todos/{id}                    View todo list
GET  /todos/{id}/edit               Edit todo list title
POST /todos/{id}                    Update todo list title
POST /todos/{id}/delete             Delete todo list
POST /todos/{id}/items              Add item to list
POST /todos/{id}/items/{itemID}/toggle   Toggle item done/undone
POST /todos/{id}/items/{itemID}/delete   Remove item
```

---

## Weekend Plan

### Friday Evening — Foundation (2-3 hours)

The goal is to have the boring infrastructure done so Saturday and Sunday are pure feature work.

#### 1. Project scaffolding

- [X] Create project directory and `go mod init`
- [X] Install pgx: `go get github.com/jackc/pgx/v5`
- [X] Create the directory structure as outlined above
- [X] Create `.gitignore`
- [X] Create `static/style.css` with basic reset and typography

#### 2. Docker Compose for PostgreSQL

- [X] Write `docker-compose.yml` with just the `db` service:
  ```yaml
  services:
    db:
      image: postgres:17
      environment:
        POSTGRES_DB: cairn
        POSTGRES_USER: cairn
        POSTGRES_PASSWORD: cairn
      ports:
        - "5432:5432"
      volumes:
        - pgdata:/var/lib/postgresql/data
  volumes:
    pgdata:
  ```
- [X] Run `docker-compose up db` and verify you can connect with `psql`

#### 3. Database connection and migration runner

- [X] Write `internal/database/database.go`:
  - `Connect(ctx) (*pgxpool.Pool, error)` — reads `DATABASE_URL` env var, falls back to localhost default
  - `RunMigrations(ctx, pool, dir) error` — creates `schema_migrations` table, reads `.sql` files in order, skips already-applied ones
- [X] Test it by running the app and seeing "applied migration" log output

#### 4. Write migration SQL files

- [X] `migrations/001_entries.sql` — the shared `entries` table
  ```sql
  CREATE TABLE entries (
      id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      entry_type  TEXT NOT NULL CHECK (entry_type IN ('note', 'bookmark', 'todo')),
      title       TEXT NOT NULL,
      created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  CREATE INDEX idx_entries_type ON entries (entry_type);
  CREATE INDEX idx_entries_created_at ON entries (created_at DESC);
  ```
- [X] `migrations/002_types.sql` — type-specific tables
  ```sql
  CREATE TABLE notes (
      entry_id UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
      body     TEXT NOT NULL DEFAULT ''
  );

  CREATE TABLE bookmarks (
      entry_id        UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
      url             TEXT NOT NULL,
      last_status     INTEGER,
      last_checked_at TIMESTAMPTZ,
      content_hash    TEXT
  );

  CREATE TABLE todo_items (
      id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      entry_id   UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
      body       TEXT NOT NULL,
      is_done    BOOLEAN NOT NULL DEFAULT false,
      position   INTEGER NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  CREATE INDEX idx_todo_items_entry ON todo_items (entry_id, position);
  ```
- [X] `migrations/003_tags.sql` — shared tagging
  ```sql
  CREATE TABLE tags (
      id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      name TEXT NOT NULL UNIQUE
  );

  CREATE TABLE entry_tags (
      entry_id UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
      tag_id   UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
      PRIMARY KEY (entry_id, tag_id)
  );
  ```
- [X] `migrations/004_entry_links.sql` — cross-references (schema only, no UI yet)
  ```sql
  CREATE TABLE entry_links (
      source_id UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
      target_id UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
      PRIMARY KEY (source_id, target_id),
      CHECK (source_id <> target_id)
  );
  ```

#### 5. Minimal main.go and layout template

- [X] Write `cmd/server/main.go`:
  - Connect to database
  - Run migrations
  - Set up static file serving
  - Register a single `GET /` route that renders "hello world" through the layout template
  - Listen on `:8080`
- [X] Write `templates/layout.html` with the HTML shell:
  - Header with app name
  - The [+] button with hidden type-selector dropdown
  - `{{template "content" .}}` block for page content
  - Minimal inline `<script>` for the [+] button toggle (the only JS in the app):
    ```html
    <button onclick="document.getElementById('type-menu').classList.toggle('hidden')">+</button>
    <div id="type-menu" class="hidden">
        <a href="/notes/new">Note</a>
        <a href="/bookmarks/new">Bookmark</a>
        <a href="/todos/new">Todo List</a>
    </div>
    ```
- [X] Add `.hidden { display: none; }` to `static/style.css`
- [X] Verify: `docker-compose up db`, then `go run ./cmd/server` → browser shows the layout with the [+] button working

**Friday evening checkpoint:** the app starts, connects to PostgreSQL, applies migrations, and renders a page through the template layout. Everything from here is features. DONE!

---

### Saturday — Core Features (6-8 hours)

#### 6. Shared entry listing (dashboard)

- [ ] Write `internal/entries/entries.go`:
  - `ListAll(ctx, pool) ([]Entry, error)` — queries all entries ordered by `updated_at DESC`
  - Define an `Entry` struct with the shared fields
- [ ] Write `templates/home.html` — the main dashboard:
  - Entry list where each entry shows:
    - Type badge (small colored label: Note, Bookmark, Todo)
    - Title (entire entry row is a clickable link to the view page)
    - Last updated date
  - Empty state message when no entries exist ("No entries yet. Click + to create one.")
- [ ] Wire up the `GET /` handler to use this

#### 7. Plain notes — full CRUD

This is the simplest type and will establish the pattern for the others.

- [ ] Write `internal/notes/queries.go`:
  - `Create(ctx, pool, title, body) (string, error)` — INSERT into both `entries` and `notes` in a transaction
  - `GetByID(ctx, pool, id) (Entry, Note, error)` — JOIN query
  - `Update(ctx, pool, id, title, body) error` — UPDATE both tables in a transaction, touch `updated_at`
  - `Delete(ctx, pool, id) error` — DELETE from `entries` (CASCADE handles `notes`)
- [ ] Write `internal/notes/handlers.go`:
  - `HandleForm` — render the new/edit form
  - `HandleCreate` — parse form, validate title not empty, call Create, redirect to view
  - `HandleView` — fetch and render
  - `HandleUpdate` — parse form, validate, call Update, redirect to view
  - `HandleDelete` — call Delete, redirect to dashboard
- [ ] Write note templates:
  - `templates/notes/form.html` — title input + body textarea (reused for new and edit)
  - `templates/notes/view.html` — rendered note with edit/delete actions
- [ ] Register routes in main.go
- [ ] Test the full flow: create → view → edit → view → delete → dashboard

#### 8. Todo lists

- [ ] Write `internal/todos/queries.go`:
  - `Create(ctx, pool, title) (string, error)` — creates entry only (items added separately)
  - `GetByID(ctx, pool, id) (Entry, []TodoItem, error)` — entry + all items ordered by position
  - `AddItem(ctx, pool, entryID, body) error` — INSERT item with position = max+1
  - `ToggleItem(ctx, pool, itemID) error` — flip `is_done`
  - `DeleteItem(ctx, pool, itemID) error`
  - `Delete(ctx, pool, id) error` — delete entry (CASCADE handles items)
- [ ] Write handlers following same pattern as notes
- [ ] Write todo templates:
  - `templates/todos/form.html` — just a title input for the list
  - `templates/todos/view.html` — the list with checkboxes, an "add item" form at the bottom, and delete buttons per item
- [ ] Register routes
- [ ] Test: create list → add items → toggle items → delete items → delete list

**Saturday checkpoint:** you have a working dashboard showing all entries, full CRUD for notes, and functional todo lists with item management. The core pattern is established.

---

### Sunday — Bookmarks, Polish, and Packaging (6-8 hours)

#### 9. Bookmarks with link checking

- [ ] Write `internal/bookmarks/queries.go`:
  - `Create(ctx, pool, title, url) (string, error)`
  - `GetByID(ctx, pool, id) (Entry, Bookmark, error)`
  - `Update(ctx, pool, id, title, url) error`
  - `Delete(ctx, pool, id) error`
  - `UpdateCheckResult(ctx, pool, id, status, contentHash) error`
- [ ] Write `internal/bookmarks/checker.go`:
  - `Check(ctx, pool, entryID) error` — HTTP GET with 10s timeout, record status code, hash response body (limit to 1MB with `io.LimitReader`), update bookmark row
  - Handle unreachable URLs gracefully (status = 0)
- [ ] Write handlers:
  - Standard CRUD handlers like notes
  - `HandleCheck` — calls the checker, redirects back to bookmark view
- [ ] Write bookmark templates:
  - `templates/bookmarks/form.html` — title + URL inputs
  - `templates/bookmarks/view.html` — shows URL (clickable), status badge (green/amber/red based on last_status), last checked time, "Check Now" button
- [ ] Register routes
- [ ] Test with known-good URLs, known-dead URLs, and slow/timeout URLs

#### 10. Tagging (shared across all types)

- [ ] Add tag input to all creation/edit forms (comma-separated text input)
- [ ] Write tag queries in `internal/entries/`:
  - `SetTags(ctx, pool, entryID, tagNames []string) error` — upsert tags, sync `entry_tags`
  - `GetTags(ctx, pool, entryID) ([]Tag, error)`
  - `ListByTag(ctx, pool, tagName) ([]Entry, error)`
- [ ] Add tag display to dashboard and individual entry views
- [ ] Add `GET /tags/{name}` route — filtered dashboard showing entries with that tag
- [ ] Tags should be clickable links to the filtered view

**If time permits — bonus features (pick any):**

#### 11. Full-text search (bonus)

- [ ] Add a `search_vector TSVECTOR` column to `entries`
- [ ] Write a trigger or application-level update that populates it from title + note body + bookmark URL
- [ ] Add a search box to the dashboard
- [ ] `GET /?q=searchterm` filters the entry list using `to_tsquery`
- [ ] Add a GIN index on the search vector column

#### 12. Dockerfile and full Docker Compose (bonus, but do this)

- [ ] Write multi-stage `Dockerfile`:
  - Build stage: `golang:1.24`, compile to static binary
  - Run stage: `alpine:latest`, copy binary + templates + migrations
- [ ] Add `app` service to `docker-compose.yml` with `depends_on: db`
- [ ] Verify `docker-compose up` brings up the entire system from scratch
- [ ] Document this in the README

#### 13. README (important — do this)

- [ ] Write a README that covers:
  - What the project is (one paragraph)
  - How to run it (`docker-compose up` or manual setup)
  - Screenshots or a brief description of what it looks like
  - Tech decisions: why Go stdlib router, why pgx without ORM, why Option B schema design
  - Schema overview (the shared base table pattern)
  - Future plans (briefly list expansion ideas to show you've thought ahead)
- [ ] Keep it concise — long READMEs don't get read

---

## Database Schema Reference

### Tables

| Table | Purpose |
|---|---|
| `entries` | Shared identity for all entry types (id, type, title, timestamps) |
| `notes` | Text body for plain notes (1:1 with entries) |
| `bookmarks` | URL + health check state (1:1 with entries) |
| `todo_items` | Checklist items belonging to a todo entry (many:1 with entries) |
| `tags` | Unique tag names |
| `entry_tags` | Many-to-many join between entries and tags |
| `entry_links` | Cross-references between any two entries (future) |
| `schema_migrations` | Tracks which migration files have been applied |

### Key Queries to Write

```sql
-- Dashboard: all entries, newest first
SELECT id, entry_type, title, created_at, updated_at
FROM entries ORDER BY updated_at DESC;

-- View a note (JOIN pattern used for all types)
SELECT e.id, e.title, e.created_at, e.updated_at, n.body
FROM entries e JOIN notes n ON n.entry_id = e.id
WHERE e.id = $1;

-- Todo items for a list
SELECT id, body, is_done, position
FROM todo_items WHERE entry_id = $1
ORDER BY position;

-- Entries filtered by tag
SELECT e.id, e.entry_type, e.title, e.created_at, e.updated_at
FROM entries e
JOIN entry_tags et ON et.entry_id = e.id
JOIN tags t ON t.id = et.tag_id
WHERE t.name = $1
ORDER BY e.updated_at DESC;

-- Full-text search (bonus)
SELECT id, entry_type, title, created_at, updated_at
FROM entries
WHERE search_vector @@ to_tsquery('english', $1)
ORDER BY ts_rank(search_vector, to_tsquery('english', $1)) DESC;
```

---

## Future Expansion Ideas

These are out of scope for the weekend but the schema and architecture support them:

- **Bookmark check history** — `bookmark_checks` table, track status over time, detect content drift
- **Background scheduler** — goroutine with `time.Ticker` that checks bookmarks periodically
- **Reading notes** — new type with source URL/DOI, structured metadata
- **Decision logs** — new type with options, reasoning, outcome tracking
- **Code snippets** — new type with language tag, server-side syntax highlighting via Chroma
- **Recurring checklists** — todo templates that can be instantiated multiple times
- **Cross-references UI** — leverage the `entry_links` table to connect related entries
- **Turbo/Hotwire** — add Turbo for snappy page transitions without full reloads (directly relevant to Klingit's stack)
- **Import/export** — browser bookmark import, markdown export
- **Full-text search** — PostgreSQL `tsvector`/`tsquery` across all entry types

---

## Key Patterns to Remember

### Transaction for multi-table inserts
Every entry creation touches two tables (`entries` + type table). Always use a transaction.

### Post/Redirect/Get
All POST handlers redirect on success (303 See Other). Never render a page directly from a POST — it causes duplicate submissions on refresh.

### Validate then act
Parse form → validate → database call → redirect. If validation fails, re-render the form with error messages and the user's input preserved.

### CASCADE deletes
Deleting from `entries` cascades to the type table, tags, links, and todo items. You only ever `DELETE FROM entries WHERE id = $1`.

### Environment-based configuration
Database URL comes from `DATABASE_URL` env var with a sensible localhost default. This is the only thing that changes between dev and Docker.
