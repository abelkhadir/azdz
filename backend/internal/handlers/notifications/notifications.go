package notificationshandler

import (
	"database/sql"
	"net/http"
	"strconv"

	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/pkg/httpx"
	"social-network/internal/pkg/repo"
	"social-network/internal/pkg/response"
	"social-network/internal/websocket"
)

type Handler struct {
	DB        *sql.DB
	WebSocket *websocket.Manager
}

func (h Handler) ListNotifications(res http.ResponseWriter, req *http.Request) {
	userID, ok := h.authenticate(res, req, http.MethodGet)
	if !ok {
		return
	}

	limit, err := httpx.QueryLimit(req, 50, 50)
	if err != nil {
		response.WriteError(res, "Invalid limit", http.StatusBadRequest)
		return
	}
	beforeID, err := httpx.OptionalQueryID(req, "beforeId")
	if err != nil {
		response.WriteError(res, "Invalid beforeId", http.StatusBadRequest)
		return
	}
	notifications, err := repo.ListNotifications(h.DB, userID, beforeID, int(limit))
	if err != nil {
		response.WriteError(res, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}
	var totalUnread int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM notifications
		WHERE user_id = ? AND is_read = 0
		AND (actor_id IS NOT NULL OR entity_id IS NOT NULL OR content != '')`, userID).Scan(&totalUnread); err != nil {
		response.WriteError(res, "Failed to count notifications", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(res, http.StatusOK, map[string]any{
		"message":       "notifications retrieved successfully",
		"notifications": notifications,
		"totalUnread":   totalUnread,
	})
}

func (h Handler) MarkNotificationsRead(res http.ResponseWriter, req *http.Request) {
	userID, ok := h.authenticate(res, req, http.MethodPost)
	if !ok {
		return
	}

	var payload *struct {
		ID  string `json:"id"`
		All bool   `json:"all"`
	}
	if !decodePayload(res, req, &payload) {
		return
	}
	if payload == nil {
		response.WriteError(res, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	var err error
	if payload.All || payload.ID == "" {
		err = repo.MarkAllNotificationsRead(h.DB, userID)
	} else {
		id, parseErr := positiveID(payload.ID)
		if parseErr != nil {
			response.WriteError(res, "Invalid notification id", http.StatusBadRequest)
			return
		}
		err = repo.DeleteNotification(h.DB, userID, id)
	}
	if err != nil {
		response.WriteError(res, "Failed to mark notifications", http.StatusInternalServerError)
		return
	}
	h.sendReadEvent(userID)
	response.WriteJSON(res, http.StatusOK, map[string]any{"message": "notifications updated"})
}

func (h Handler) ListChatNotifications(res http.ResponseWriter, req *http.Request) {
	h.listThreadNotifications(res, req, "private", "chat notifications retrieved successfully")
}

func (h Handler) ListGroupNotifications(res http.ResponseWriter, req *http.Request) {
	h.listThreadNotifications(res, req, "group", "group notifications retrieved successfully")
}

func (h Handler) listThreadNotifications(res http.ResponseWriter, req *http.Request, kind, message string) {
	userID, ok := h.authenticate(res, req, http.MethodGet)
	if !ok {
		return
	}

	counts := make(map[string]int)
	total := 0
	var err error
	if kind == "group" {
		var items []models.GroupUnreadCounter
		items, err = repo.UnreadGroupMessageCounts(h.DB, userID)
		for _, item := range items {
			counts[strconv.FormatInt(item.GroupID, 10)] = item.UnreadCount
			total += item.UnreadCount
		}
	} else {
		var items []models.ChatUnreadCounter
		items, err = repo.UnreadPrivateMessageCounts(h.DB, userID)
		for _, item := range items {
			counts[strconv.FormatInt(item.ActorID, 10)] = item.UnreadCount
			total += item.UnreadCount
		}
	}
	if err != nil {
		response.WriteError(res, "Failed to fetch chat notifications", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(res, http.StatusOK, map[string]any{
		"message":      message,
		"counts":       counts,
		"total_unread": total,
	})
}

func (h Handler) MarkChatNotificationsRead(res http.ResponseWriter, req *http.Request) {
	h.markThreadRead(res, req, "private", "chat notifications updated")
}

func (h Handler) MarkGroupNotificationsRead(res http.ResponseWriter, req *http.Request) {
	h.markThreadRead(res, req, "group", "group notifications updated")
}

func (h Handler) markThreadRead(res http.ResponseWriter, req *http.Request, kind, message string) {
	userID, ok := h.authenticate(res, req, http.MethodPost)
	if !ok {
		return
	}

	var payload *struct {
		ActorID string `json:"actor_id"`
		GroupID string `json:"group_id"`
	}
	if !decodePayload(res, req, &payload) {
		return
	}
	if payload == nil {
		response.WriteError(res, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	threadID := payload.ActorID
	field := "actor_id"
	if kind == "group" {
		threadID, field = payload.GroupID, "group_id"
	}
	id, err := positiveID(threadID)
	if err != nil {
		response.WriteError(res, "Missing or invalid "+field, http.StatusBadRequest)
		return
	}
	if kind == "group" {
		err = repo.MarkGroupThreadRead(h.DB, userID, id)
	} else {
		err = repo.MarkPrivateThreadRead(h.DB, userID, id)
	}
	if err != nil {
		response.WriteError(res, "Failed to mark chat notifications", http.StatusInternalServerError)
		return
	}
	h.sendReadEvent(userID)
	response.WriteJSON(res, http.StatusOK, map[string]any{"message": message})
}

func (h Handler) authenticate(res http.ResponseWriter, req *http.Request, method string) (int64, bool) {
	res.Header().Set("Cache-Control", "no-store")
	if req.Method != method {
		res.Header().Set("Allow", method)
		response.WriteError(res, "Method not allowed", http.StatusMethodNotAllowed)
		return 0, false
	}
	userID, err := middleware.UserIDFromRequest(h.DB, req)
	if err != nil {
		response.WriteError(res, "No active session", http.StatusUnauthorized)
		return 0, false
	}
	return userID, true
}

func (h Handler) sendReadEvent(userID int64) {
	if h.WebSocket != nil {
		websocket.SendEventToUser(h.WebSocket, userID, "notifications_read", nil)
	}
}

func decodePayload(res http.ResponseWriter, req *http.Request, payload any) bool {
	if err := httpx.DecodeJSON(res, req, 4*1024, payload); err != nil {
		response.WriteError(res, "Invalid JSON format", http.StatusBadRequest)
		return false
	}
	return true
}

func positiveID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
