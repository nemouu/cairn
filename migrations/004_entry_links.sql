-- Create a table to track bidirectional links between entries.
-- Ensures each link is unique and prevents self-referential links.

CREATE TABLE entry_links (
    source_id UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    PRIMARY KEY (source_id, target_id),
    CHECK (source_id <> target_id)
);
