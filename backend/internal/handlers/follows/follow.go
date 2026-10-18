package follows

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"social-network/internal/models"
	"social-network/internal/pkg/repo"
	"social-network/internal/pkg/response"
	"social-network/internal/websocket"
)

func (h Handler) Follow(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPut && r.Method != http.MethodDelete {
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if strconv.FormatInt(id, 10) == userID {
		response.WriteError(w, "You cannot follow yourself", http.StatusBadRequest)
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	status := "none"
	var notification *models.Notification
	var err error
	if r.Method == http.MethodPut {
		tx, txErr := h.DB.BeginTx(r.Context(), nil)
		if txErr != nil {
			response.WriteError(w, "Could not update follow", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		var existingStatus string
		err = tx.QueryRowContext(r.Context(), `SELECT status FROM follows WHERE follower_id = ? AND following_id = ?`, userID, id).Scan(&existingStatus)
		if err == nil {
			status = existingStatus
		} else if errors.Is(err, sql.ErrNoRows) {
			var followID int64
			err = tx.QueryRowContext(r.Context(), `INSERT INTO follows (follower_id, following_id, status)
				SELECT ?, id, CASE WHEN is_public = 1 THEN 'accepted' ELSE 'pending' END
				FROM users WHERE id = ? RETURNING id, status`, userID, id).Scan(&followID, &status)
			if err == nil {
				actorID, _ := strconv.ParseInt(userID, 10, 64)
				created := models.Notification{
					UserID: id, ActorID: actorID, Type: models.NotificationFollowRequest,
					EntityID: followID, EntityType: "follow", Content: "Someone wants to follow you.",
				}
				if status == "accepted" {
					created.Type = models.NotificationFollow
					created.EntityID = 0
					created.EntityType = "profile"
					created.Content = "Someone started following you."
				}
				err = repo.CreateNotification(tx, &created)
				if err == nil {
					notification = &created
				}
			}
		}
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteError(w, "Profile not found", http.StatusNotFound)
			return
		}
		if err == nil {
			err = tx.Commit()
		}
	} else {
		var removedID int64
		err = h.DB.QueryRowContext(r.Context(), `DELETE FROM follows
			WHERE follower_id = ? AND following_id = ? AND status = 'accepted'
			RETURNING id`, userID, id).Scan(&removedID)
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteError(w, "Only accepted follows can be removed", http.StatusConflict)
			return
		}
	}
	if err != nil {
		log.Println("Error updating follow:", err)
		response.WriteError(w, "Could not update follow", http.StatusInternalServerError)
		return
	}
	if notification != nil && h.WebSocket != nil {
		websocket.SendNotification(h.WebSocket, *notification)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "followStatus": status})
}
