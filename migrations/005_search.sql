-- Add full-text search support for entries using PostgreSQL's tsvector.
-- Creates a search_vector column, indexes it, and backfills existing entries.

ALTER TABLE entries ADD COLUMN search_vector TSVECTOR;

CREATE INDEX idx_entries_search ON entries USING GIN (search_vector);

-- Populate search_vector for existing notes (title + body)
UPDATE entries e
SET search_vector = to_tsvector('english', e.title || ' ' || COALESCE(n.body, ''))
FROM notes n
WHERE n.entry_id = e.id;

-- Populate search_vector for existing bookmarks (title + URLs)
UPDATE entries e
SET search_vector = to_tsvector('english',
    e.title || ' ' || COALESCE(
        (SELECT string_agg(
            regexp_replace(regexp_replace(url, '^https?://', ''), '[^a-zA-Z0-9]+', ' ', 'g'),
            ' '
        ) FROM bookmark_items bi WHERE bi.entry_id = e.id),
        ''
    ))
WHERE e.entry_type = 'bookmark';

-- Populate search_vector for existing todos (title + item text)
UPDATE entries e
SET search_vector = to_tsvector('english',
    e.title || ' ' || COALESCE(
        (SELECT string_agg(body, ' ') FROM todo_items ti WHERE ti.entry_id = e.id),
        ''
    ))
WHERE e.entry_type = 'todo';
