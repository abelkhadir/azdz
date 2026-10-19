package follows

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"social-network/internal/models"
	"social-network/internal/pkg/repo"
	"social-network/internal/pkg/response"
	"social-network/internal/websocket"
)

// Requests lists only the pending requests addressed to the authenticated user.
func (h Handler) Requests(w http.ResponseWriter, r *http.Request, userID int64) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	rows, err := h.DB.QueryContext(r.Context(), `SELECT u.id, u.first_name, u.last_name,
		COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM follows f JOIN users u ON u.id = f.follower_id
		WHERE f.following_id = ? AND f.status = 'pending'
		ORDER BY f.created_at DESC, f.id DESC`, userID)
	h.writeUsers(w, rows, err)
}

// DecideRequest changes only a pending request belonging to the recipient.
func (h Handler) DecideRequest(w http.ResponseWriter, r *http.Request, userID int64) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPut && r.Method != http.MethodDelete {
		w.Header().Set("Allow", "PUT, DELETE")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	query := `UPDATE follows SET status = 'accepted'
		WHERE follower_id = ? AND following_id = ? AND status = 'pending' RETURNING id`
	status := "accepted"
	if r.Method == http.MethodDelete {
		query = `DELETE FROM follows
			WHERE follower_id = ? AND following_id = ? AND status = 'pending' RETURNING id`
		status = "none"
	}
	var relationshipID int64
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		response.WriteError(w, "Could not update follow request", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(r.Context(), query, id, userID).Scan(&relationshipID)
	if errors.Is(err, sql.ErrNoRows) {
		response.WriteError(w, "Pending follow request not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("Error deciding follow request:", err)
		response.WriteError(w, "Could not update follow request", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM notifications WHERE user_id = ? AND type = ? AND entity_id = ?`, userID, models.NotificationFollowRequest, relationshipID); err != nil {
		response.WriteError(w, "Could not update request notification", http.StatusInternalServerError)
		return
	}
	var notification models.Notification
	if status == "accepted" {
		notification = models.Notification{
			UserID: id, ActorID: userID, Type: models.NotificationFollowAccepted,
			EntityType: "profile", Content: "Your follow request was accepted.",
		}
		if err := repo.CreateNotification(tx, &notification); err != nil {
			response.WriteError(w, "Could not notify the follower", http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		response.WriteError(w, "Could not update follow request", http.StatusInternalServerError)
		return
	}
	if h.WebSocket != nil {
		if notification.ID != 0 {
			websocket.SendNotification(h.WebSocket, notification)
		}
		websocket.SendEventToUser(h.WebSocket, userID, "notifications_read", nil)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "followStatus": status})
}
