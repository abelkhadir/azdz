CREATE TABLE group_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    group_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    invited_by INTEGER,

    type TEXT NOT NULL
        CHECK (type IN ('invite', 'join_request')),

    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'declined')),

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (group_id)
        REFERENCES groups(id)
        ON DELETE CASCADE,

    FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    FOREIGN KEY (invited_by)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_group_requests_group_id
ON group_requests(group_id);

CREATE INDEX idx_group_requests_user_id
ON group_requests(user_id);