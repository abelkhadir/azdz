DROP INDEX IF EXISTS idx_notifications_group_unread;
DROP INDEX IF EXISTS idx_notifications_private_unread;

ALTER TABLE notifications DROP COLUMN content;
ALTER TABLE notifications DROP COLUMN entity_type;
ALTER TABLE notifications DROP COLUMN entity_id;
ALTER TABLE notifications DROP COLUMN actor_id;
