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