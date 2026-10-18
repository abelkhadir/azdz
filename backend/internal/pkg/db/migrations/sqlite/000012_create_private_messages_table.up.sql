CREATE TABLE private_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    sender_id INTEGER NOT NULL,
    receiver_id INTEGER NOT NULL,

    content TEXT NOT NULL,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (sender_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    FOREIGN KEY (receiver_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CHECK (sender_id != receiver_id)
);

CREATE INDEX idx_private_messages_sender
ON private_messages(sender_id);

CREATE INDEX idx_private_messages_receiver
ON private_messages(receiver_id);