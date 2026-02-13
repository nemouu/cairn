CREATE TABLE entries (
      id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      entry_type  TEXT NOT NULL CHECK (entry_type IN ('note', 'bookmark', 'todo')),
      title       TEXT NOT NULL,
      created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
  );

CREATE INDEX idx_entries_type ON entries (entry_type);
  
CREATE INDEX idx_entries_created_at ON entries (created_at DESC);