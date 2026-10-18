CREATE TABLE follows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    follower_id INTEGER NOT NULL,
    following_id INTEGER NOT NULL,

    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'declined')),

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (follower_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    FOREIGN KEY (following_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    UNIQUE (follower_id, following_id),

    CHECK (follower_id != following_id)
);

CREATE INDEX idx_follows_follower
ON follows(follower_id);

CREATE INDEX idx_follows_following
ON follows(following_id);