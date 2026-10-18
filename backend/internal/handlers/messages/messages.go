package messages

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"social-network/internal/middleware"
	"social-network/internal/pkg/repo"
	"social-network/internal/websocket"
)

type Handler struct {
	DB        *sql.DB
	WebSocket *websocket.Manager
}

func (h Handler) History(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := middleware.UserIDFromRequest(h.DB, r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	withUserID, err := parseIntQuery(r, "withUserId")
	if err != nil {
		http.Error(w, "Invalid withUserId", http.StatusBadRequest)
		return
	}

	allowed, err := repo.CanUsersChat(h.DB, userID, withUserID)
	if err != nil {
		http.Error(w, "Failed to check chat permission", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	limit, beforeID, err := paginationQuery(r)
	if err != nil {
		http.Error(w, "Invalid pagination", http.StatusBadRequest)
		return
	}

	history, err := repo.GetPrivateMessagesPage(h.DB, userID, withUserID, beforeID, limit)
	if err != nil {
		http.Error(w, "Failed to load messages", http.StatusInternalServerError)
		return
	}

	writeJSON(w, history)
}

func (h Handler) Conversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := middleware.UserIDFromRequest(h.DB, r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversations, err := repo.GetConversations(h.DB, userID)
	if err != nil {
		http.Error(w, "Failed to load conversations", http.StatusInternalServerError)
		return
	}

	writeJSON(w, conversations)
}

func (h Handler) GroupHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := middleware.UserIDFromRequest(h.DB, r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groupID, err := parseIntQuery(r, "groupId")
	if err != nil {
		http.Error(w, "Invalid groupId", http.StatusBadRequest)
		return
	}

	limit, beforeID, err := paginationQuery(r)
	if err != nil {
		http.Error(w, "Invalid pagination", http.StatusBadRequest)
		return
	}

	messages, err := repo.GetGroupMessagesPage(h.DB, userID, groupID, beforeID, limit)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "Failed to load group messages", http.StatusInternalServerError)
		return
	}

	writeJSON(w, messages)
}

func parseIntQuery(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.URL.Query().Get(name), 10, 64)
}

func paginationQuery(r *http.Request) (int, int64, error) {
	limit := 30
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return 0, 0, strconv.ErrSyntax
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}
	var beforeID int64
	if raw := r.URL.Query().Get("beforeId"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return 0, 0, strconv.ErrSyntax
		}
		beforeID = parsed
	}
	return limit, beforeID, nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}
