package repo

import (
	"database/sql"

	"social-network/internal/models"
)

// notificationStore lets the same helpers work with a database or transaction.
type notificationStore interface {
	Exec(string, ...any) (sql.Result, error)
	Query(string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
}

func CreateNotification(db notificationStore, notification *models.Notification) error {
	result, err := db.Exec(`INSERT INTO notifications
		(user_id, actor_id, type, entity_id, entity_type, content, is_read)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		notification.UserID, nullableID(notification.ActorID), notification.Type,
		nullableID(notification.EntityID), nullable(notification.EntityType), notification.Content,
		boolToInt(notification.IsRead),
	)
	if err != nil {
		return err
	}
	notification.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}
	return db.QueryRow(`SELECT created_at FROM notifications WHERE id = ?`, notification.ID).
		Scan(&notification.CreatedAt)
}

func ListNotifications(db notificationStore, userID, beforeID int64, limit int) ([]models.Notification, error) {
	rows, err := db.Query(`SELECT id, user_id, COALESCE(actor_id, 0), type,
		COALESCE(entity_id, 0), COALESCE(entity_type, ''), content, is_read, created_at
		FROM notifications
		WHERE user_id = ? AND (? = 0 OR id < ?)
		  AND (actor_id IS NOT NULL OR entity_id IS NOT NULL OR content != '')
		ORDER BY id DESC LIMIT ?`, userID, beforeID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.Notification, 0)
	for rows.Next() {
		var item models.Notification
		var isRead int
		if err := rows.Scan(&item.ID, &item.UserID, &item.ActorID, &item.Type,
			&item.EntityID, &item.EntityType, &item.Content, &isRead, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.IsRead = isRead == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func MarkAllNotificationsRead(db notificationStore, userID int64) error {
	_, err := db.Exec(`UPDATE notifications SET is_read = 1 WHERE user_id = ?`, userID)
	return err
}

func DeleteNotification(db notificationStore, userID, notificationID int64) error {
	_, err := db.Exec(`DELETE FROM notifications WHERE user_id = ? AND id = ?`, userID, notificationID)
	return err
}

func UnreadPrivateMessageCounts(db notificationStore, userID int64) ([]models.ChatUnreadCounter, error) {
	rows, err := db.Query(`SELECT actor_id, COUNT(*) FROM notifications
		WHERE user_id = ? AND type = ? AND is_read = 0 AND actor_id IS NOT NULL
		GROUP BY actor_id ORDER BY MAX(created_at) DESC`, userID, models.NotificationPrivateMessage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.ChatUnreadCounter, 0)
	for rows.Next() {
		var item models.ChatUnreadCounter
		if err := rows.Scan(&item.ActorID, &item.UnreadCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func MarkPrivateThreadRead(db notificationStore, userID, actorID int64) error {
	_, err := db.Exec(`DELETE FROM notifications WHERE user_id = ? AND type = ? AND actor_id = ?`,
		userID, models.NotificationPrivateMessage, actorID)
	return err
}

func UnreadGroupMessageCounts(db notificationStore, userID int64) ([]models.GroupUnreadCounter, error) {
	rows, err := db.Query(`SELECT entity_id, COUNT(*) FROM notifications
		WHERE user_id = ? AND type = ? AND is_read = 0 AND entity_id IS NOT NULL
		GROUP BY entity_id ORDER BY MAX(created_at) DESC`, userID, models.NotificationGroupMessage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.GroupUnreadCounter, 0)
	for rows.Next() {
		var item models.GroupUnreadCounter
		if err := rows.Scan(&item.GroupID, &item.UnreadCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func MarkGroupThreadRead(db notificationStore, userID, groupID int64) error {
	_, err := db.Exec(`DELETE FROM notifications WHERE user_id = ? AND type = ? AND entity_id = ?`,
		userID, models.NotificationGroupMessage, groupID)
	return err
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableID(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}
