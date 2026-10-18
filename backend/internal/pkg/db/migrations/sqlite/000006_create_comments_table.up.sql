CREATE TABLE comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,

    content TEXT,
    image_path TEXT,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (post_id)
        REFERENCES posts(id)
        ON DELETE CASCADE,

    FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_comments_post_id
ON comments(post_id);