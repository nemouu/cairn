ALTER TABLE entries ADD COLUMN search_vector TSVECTOR;

CREATE INDEX idx_entries_search ON entries USING GIN (search_vector);

-- Populate for existing notes
UPDATE entries e
SET search_vector = to_tsvector('english', e.title || ' ' || COALESCE(n.body, ''))
FROM notes n
WHERE n.entry_id = e.id;

-- Populate for existing bookmark_items (updated for new structure)
UPDATE entries e
SET search_vector = to_tsvector('english', e.title || ' ' || COALESCE(
    (SELECT string_agg(url, ' ') FROM bookmark_items bi WHERE bi.entry_id = e.id),
    ''
))
WHERE e.entry_type = 'bookmark';

-- Populate for todos
UPDATE entries e
SET search_vector = to_tsvector('english', e.title)
WHERE e.entry_type = 'todo';
