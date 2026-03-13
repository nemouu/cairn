ALTER TABLE entries ADD COLUMN search_vector TSVECTOR;

CREATE INDEX idx_entries_search ON entries USING GIN (search_vector);

-- Populate for existing note titles and notes
UPDATE entries e
SET search_vector = to_tsvector('english', e.title || ' ' || COALESCE(n.body, ''))
FROM notes n
WHERE n.entry_id = e.id;

-- Populate for existing bookmark titles and bookmark items
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

-- Populate for todos with their titles and item text
UPDATE entries e
SET search_vector = to_tsvector('english',
    e.title || ' ' || COALESCE(
        (SELECT string_agg(body, ' ') FROM todo_items ti WHERE ti.entry_id = e.id),
        ''
    ))
WHERE e.entry_type = 'todo';
