CREATE TABLE groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    creator_id INTEGER NOT NULL,

    title TEXT NOT NULL,
    description TEXT NOT NULL,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (creator_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_groups_creator_id
ON groups(creator_id);