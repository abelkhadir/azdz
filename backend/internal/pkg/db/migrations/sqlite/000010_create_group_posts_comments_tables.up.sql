CREATE TABLE group_posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    group_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,

    content TEXT,
    image_path TEXT,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (group_id)
        REFERENCES groups(id)
        ON DELETE CASCADE,

    FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TABLE group_comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,

    content TEXT,
    image_path TEXT,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (post_id)
        REFERENCES group_posts(id)
        ON DELETE CASCADE,

    FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_group_posts_group_id
ON group_posts(group_id);

CREATE INDEX idx_group_comments_post_id
ON group_comments(post_id);