-- Create tables for the initial entry types: notes, bookmarks, and todos.
-- Each table stores type-specific data and references the parent entries table.

CREATE TABLE notes (
    entry_id UUID PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
    body     TEXT NOT NULL DEFAULT ''
);

CREATE TABLE bookmark_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id        UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    title           TEXT,
    last_status     INTEGER,
    last_checked_at TIMESTAMPTZ,
    content_hash    TEXT,
    position        INTEGER NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_bookmark_items_entry ON bookmark_items (entry_id, position);

CREATE TABLE todo_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id   UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    is_done    BOOLEAN NOT NULL DEFAULT false,
    position   INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_todo_items_entry ON todo_items (entry_id, position);
