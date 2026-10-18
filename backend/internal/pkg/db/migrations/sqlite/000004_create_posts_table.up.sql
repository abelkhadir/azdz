CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    user_id INTEGER NOT NULL,

    content TEXT,
    image_path TEXT,

    privacy TEXT NOT NULL DEFAULT 'public'
        CHECK (privacy IN ('public', 'followers', 'private')),

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_posts_user_id
ON posts(user_id);

CREATE INDEX idx_posts_created_at
ON posts(created_at);