package repo

import (
	"database/sql"
	"errors"
	"social-network/internal/models"
)

var ErrMessageForbidden = errors.New("message sender is not allowed to use this conversation")

// CanUsersChat requires an accepted follow in either direction. Profile
// visibility alone never grants private-message access.
func CanUsersChat(db *sql.DB, user1ID, user2ID int64) (bool, error) {
	var exists int

	query := `
		SELECT EXISTS (
			SELECT 1 FROM follows WHERE status = 'accepted' AND (
				(follower_id = ? AND following_id = ?)
				OR (follower_id = ? AND following_id = ?)
			)
		)
	`

	err := db.QueryRow(
		query,
		user1ID,
		user2ID,
		user2ID,
		user1ID,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists == 1, nil
}

// SavePrivateMessage saves a private message in the database.
func SavePrivateMessage(db *sql.DB, message *models.PrivateMessage) error {
	query := `
		INSERT INTO private_messages (
			sender_id,
			receiver_id,
			content
		)
		SELECT ?, ?, ?
		WHERE EXISTS (
			SELECT 1 FROM follows WHERE status = 'accepted' AND (
				(follower_id = ? AND following_id = ?)
				OR (follower_id = ? AND following_id = ?)
			)
		)
	`

	result, err := db.Exec(
		query,
		message.SenderID,
		message.ReceiverID,
		message.Content,
		message.SenderID,
		message.ReceiverID,
		message.ReceiverID,
		message.SenderID,
	)

	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted != 1 {
		return ErrMessageForbidden
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	message.ID = id

	return db.QueryRow(
		`SELECT created_at FROM private_messages WHERE id = ?`,
		message.ID,
	).Scan(&message.CreatedAt)
}

// GetPrivateMessagesPage returns the newest page in chronological order.
// beforeID loads the next older page without being affected by new messages.
func GetPrivateMessagesPage(
	db *sql.DB,
	user1ID,
	user2ID,
	beforeID int64,
	limit int,
) ([]models.PrivateMessage, error) {

	query := `
		SELECT
			id,
			sender_id,
			receiver_id,
			content,
			created_at
		FROM (
			SELECT id, sender_id, receiver_id, content, created_at
			FROM private_messages
			WHERE ((sender_id = ? AND receiver_id = ?)
				OR (sender_id = ? AND receiver_id = ?))
				AND (? = 0 OR id < ?)
			ORDER BY id DESC
			LIMIT CASE WHEN ? > 0 THEN ? ELSE -1 END
		)
		ORDER BY id ASC
	`

	rows, err := db.Query(
		query,
		user1ID,
		user2ID,
		user2ID,
		user1ID,
		beforeID,
		beforeID,
		limit,
		limit,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []models.PrivateMessage{}

	for rows.Next() {
		var message models.PrivateMessage

		err := rows.Scan(
			&message.ID,
			&message.SenderID,
			&message.ReceiverID,
			&message.Content,
			&message.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// GetConversations returns the latest private message for each conversation.
func GetConversations(db *sql.DB, userID int64) ([]models.Conversation, error) {
	query := `
		SELECT
			u.id,
			u.first_name,
			u.last_name,
			COALESCE(u.nickname, ''),
			COALESCE(u.avatar_path, ''),
			pm.content,
			pm.created_at
		FROM private_messages pm
		JOIN users u
			ON u.id = CASE
				WHEN pm.sender_id = ? THEN pm.receiver_id
				ELSE pm.sender_id
			END
		WHERE (pm.sender_id = ? OR pm.receiver_id = ?)
		AND EXISTS (
			SELECT 1 FROM follows f WHERE f.status = 'accepted' AND (
				(f.follower_id = ? AND f.following_id = u.id)
				OR (f.follower_id = u.id AND f.following_id = ?)
			)
		)
		AND pm.id = (
			SELECT MAX(pm2.id)
			FROM private_messages pm2
			WHERE
				(pm2.sender_id = ? AND pm2.receiver_id = u.id)
				OR
				(pm2.sender_id = u.id AND pm2.receiver_id = ?)
		)
		ORDER BY pm.created_at DESC
	`

	rows, err := db.Query(
		query,
		userID,
		userID,
		userID,
		userID,
		userID,
		userID,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := []models.Conversation{}

	for rows.Next() {
		var conversation models.Conversation

		err := rows.Scan(
			&conversation.UserID,
			&conversation.FirstName,
			&conversation.LastName,
			&conversation.Nickname,
			&conversation.AvatarPath,
			&conversation.LastMessage,
			&conversation.LastMessageAt,
		)
		if err != nil {
			return nil, err
		}

		conversations = append(conversations, conversation)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return conversations, nil
}

func CanUserAccessGroupChat(db *sql.DB, userID, groupID int64) (bool, error) {
	var exists int

	err := db.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM group_members
			WHERE user_id = ? AND group_id = ?
		)`,
		userID,
		groupID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists == 1, nil
}

func SaveGroupMessage(db *sql.DB, message *models.GroupMessage) error {
	result, err := db.Exec(
		`INSERT INTO group_messages (group_id, sender_id, content)
		SELECT ?, ?, ?
		WHERE EXISTS (
			SELECT 1 FROM group_members
			WHERE group_id = ? AND user_id = ?
		)`,
		message.GroupID,
		message.SenderID,
		message.Content,
		message.GroupID,
		message.SenderID,
	)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted != 1 {
		return ErrMessageForbidden
	}

	message.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}

	return db.QueryRow(
		`SELECT gm.created_at,
			COALESCE(NULLIF(u.nickname, ''), NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), 'Member')
		FROM group_messages gm
		JOIN users u ON u.id = gm.sender_id
		WHERE gm.id = ?`,
		message.ID,
	).Scan(&message.CreatedAt, &message.SenderName)
}

func GetGroupMessagesPage(db *sql.DB, userID, groupID, beforeID int64, limit int) ([]models.GroupMessage, error) {
	allowed, err := CanUserAccessGroupChat(db, userID, groupID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, sql.ErrNoRows
	}

	rows, err := db.Query(
		`SELECT gm.id, gm.group_id, gm.sender_id,
			COALESCE(NULLIF(u.nickname, ''), NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), 'Member'),
			gm.content, gm.created_at
		FROM (
			SELECT id, group_id, sender_id, content, created_at
			FROM group_messages
			WHERE group_id = ? AND (? = 0 OR id < ?)
			ORDER BY id DESC
			LIMIT CASE WHEN ? > 0 THEN ? ELSE -1 END
		) gm
		JOIN users u ON u.id = gm.sender_id
		ORDER BY gm.id ASC`,
		groupID, beforeID, beforeID, limit, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []models.GroupMessage{}

	for rows.Next() {
		var message models.GroupMessage

		err := rows.Scan(
			&message.ID,
			&message.GroupID,
			&message.SenderID,
			&message.SenderName,
			&message.Content,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func GetGroupMemberIDs(db *sql.DB, groupID int64) ([]int64, error) {
	rows, err := db.Query(
		`SELECT user_id FROM group_members WHERE group_id = ?`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userIDs := []int64{}

	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}

		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userIDs, nil
}
