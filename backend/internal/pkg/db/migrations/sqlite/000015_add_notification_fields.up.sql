ALTER TABLE notifications ADD COLUMN actor_id INTEGER;
ALTER TABLE notifications ADD COLUMN entity_id INTEGER;
ALTER TABLE notifications ADD COLUMN entity_type TEXT;
ALTER TABLE notifications ADD COLUMN content TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_notifications_private_unread
ON notifications(user_id, type, actor_id, is_read);

CREATE INDEX idx_notifications_group_unread
ON notifications(user_id, type, entity_id, is_read);
