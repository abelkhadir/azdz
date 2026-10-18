package groups

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"social-network/internal/models"
	"social-network/internal/pkg/httpx"
	"social-network/internal/pkg/repo"
	"social-network/internal/pkg/response"
	"social-network/internal/websocket"
)

type eventInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DateTime    string `json:"dateTime"`
}

// Events creates a group event and notifies every other group member.
func (h Handler) Events(w http.ResponseWriter, r *http.Request, userID string) {
	currentUserID, ok := parseUserID(w, userID)
	if !ok {
		return
	}
	groupID, ok := parseGroupID(w, r)
	if !ok {
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.listEvents(w, r, currentUserID, groupID)
	case http.MethodPost:
		h.createEvent(w, r, currentUserID, groupID)
	default:
		w.Header().Set("Allow", "GET, POST")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h Handler) createEvent(w http.ResponseWriter, r *http.Request, currentUserID, groupID int64) {
	var input eventInput
	if err := httpx.DecodeJSON(w, r, 16*1024, &input); err != nil {
		response.WriteError(w, "Invalid event", http.StatusBadRequest)
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Title == "" || utf8.RuneCountInString(input.Title) > maxTitleLength || input.Description == "" || utf8.RuneCountInString(input.Description) > maxDescriptionLength {
		response.WriteError(w, "Event title or description is invalid", http.StatusBadRequest)
		return
	}
	eventTime, err := time.Parse(time.RFC3339, input.DateTime)
	if err != nil {
		response.WriteError(w, "dateTime must be an RFC 3339 timestamp", http.StatusBadRequest)
		return
	}
	if !eventTime.After(time.Now()) {
		response.WriteError(w, "Event date must be in the future", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		response.WriteError(w, "Could not create event", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	var isMember bool
	if err := tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)`, groupID, currentUserID).Scan(&isMember); err != nil {
		response.WriteError(w, "Could not verify group membership", http.StatusInternalServerError)
		return
	}
	if !isMember {
		response.WriteError(w, "Only group members can create events", http.StatusForbidden)
		return
	}

	result, err := tx.ExecContext(r.Context(), `INSERT INTO events (group_id, creator_id, title, description, event_datetime) VALUES (?, ?, ?, ?, ?)`, groupID, currentUserID, input.Title, input.Description, input.DateTime)
	if err != nil {
		log.Println("Error creating event:", err)
		response.WriteError(w, "Could not create event", http.StatusInternalServerError)
		return
	}
	eventID, err := result.LastInsertId()
	if err != nil {
		response.WriteError(w, "Could not create event", http.StatusInternalServerError)
		return
	}

	rows, err := tx.QueryContext(r.Context(), `SELECT user_id FROM group_members WHERE group_id = ? AND user_id != ?`, groupID, currentUserID)
	if err != nil {
		response.WriteError(w, "Could not notify group members", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	notifications := make([]models.Notification, 0)
	for rows.Next() {
		var memberID int64
		if err := rows.Scan(&memberID); err != nil {
			response.WriteError(w, "Could not notify group members", http.StatusInternalServerError)
			return
		}
		notification := models.Notification{
			UserID: memberID, ActorID: currentUserID, Type: models.NotificationGroupEvent,
			EntityID: groupID, EntityType: "group", Content: input.Title,
		}
		if err := repo.CreateNotification(tx, &notification); err != nil {
			response.WriteError(w, "Could not notify group members", http.StatusInternalServerError)
			return
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		response.WriteError(w, "Could not notify group members", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		response.WriteError(w, "Could not create event", http.StatusInternalServerError)
		return
	}
	if h.WebSocket != nil {
		for _, notification := range notifications {
			websocket.SendNotification(h.WebSocket, notification)
		}
	}
	response.WriteJSON(w, http.StatusCreated, map[string]any{"event": models.Event{ID: eventID, GroupID: groupID, CreatorID: currentUserID, Title: input.Title, Description: input.Description, DateTime: input.DateTime}})
}

func (h Handler) listEvents(w http.ResponseWriter, r *http.Request, currentUserID, groupID int64) {
	var isMember bool
	if err := h.DB.QueryRowContext(r.Context(), `SELECT EXISTS(
		SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)`, groupID, currentUserID).Scan(&isMember); err != nil {
		response.WriteError(w, "Could not verify group membership", http.StatusInternalServerError)
		return
	}
	if !isMember {
		response.WriteError(w, "Only group members can view events", http.StatusForbidden)
		return
	}
	limit, offset, ok := parseListPage(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.QueryContext(r.Context(), `SELECT e.id, e.group_id, e.creator_id, e.title,
		e.description, e.event_datetime,
		(SELECT COUNT(*) FROM event_responses er WHERE er.event_id = e.id AND er.response = 'going'),
		(SELECT COUNT(*) FROM event_responses er WHERE er.event_id = e.id AND er.response = 'not_going'),
		COALESCE((SELECT response FROM event_responses er WHERE er.event_id = e.id AND er.user_id = ?), '')
		FROM events e WHERE e.group_id = ? ORDER BY e.event_datetime, e.id LIMIT ? OFFSET ?`, currentUserID, groupID, limit, offset)
	if err != nil {
		response.WriteError(w, "Could not load events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	events := make([]models.Event, 0)
	for rows.Next() {
		var event models.Event
		if err := rows.Scan(&event.ID, &event.GroupID, &event.CreatorID, &event.Title,
			&event.Description, &event.DateTime, &event.GoingCount, &event.NotGoingCount, &event.Response); err != nil {
			response.WriteError(w, "Could not load events", http.StatusInternalServerError)
			return
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		response.WriteError(w, "Could not load events", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h Handler) RespondToEvent(w http.ResponseWriter, r *http.Request, userID string) {
	currentUserID, ok := parseUserID(w, userID)
	if !ok {
		return
	}
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", "PUT")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	eventID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || eventID <= 0 {
		response.WriteError(w, "Invalid event ID", http.StatusBadRequest)
		return
	}
	var input struct {
		Response string `json:"response"`
	}
	if err := httpx.DecodeJSON(w, r, 4*1024, &input); err != nil || (input.Response != "going" && input.Response != "not_going") {
		response.WriteError(w, "Response must be going or not_going", http.StatusBadRequest)
		return
	}
	var groupID int64
	err = h.DB.QueryRowContext(r.Context(), `SELECT e.group_id FROM events e
		JOIN group_members gm ON gm.group_id = e.group_id AND gm.user_id = ?
		WHERE e.id = ?`, currentUserID, eventID).Scan(&groupID)
	if errors.Is(err, sql.ErrNoRows) {
		response.WriteError(w, "Event not found", http.StatusNotFound)
		return
	}
	if err != nil {
		response.WriteError(w, "Could not verify event access", http.StatusInternalServerError)
		return
	}
	_, err = h.DB.ExecContext(r.Context(), `INSERT INTO event_responses (event_id, user_id, response)
		VALUES (?, ?, ?) ON CONFLICT(event_id, user_id) DO UPDATE SET response = excluded.response,
		responded_at = CURRENT_TIMESTAMP`, eventID, currentUserID, input.Response)
	if err != nil {
		response.WriteError(w, "Could not save event response", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"response": models.EventResponse{EventID: eventID, UserID: currentUserID, Response: input.Response},
		"groupId":  groupID,
	})
}
