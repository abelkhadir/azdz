package groups

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"social-network/internal/models"
	"social-network/internal/pkg/httpx"
	"social-network/internal/pkg/repo"
	"social-network/internal/pkg/response"
	"social-network/internal/websocket"
)

type groupRequestInput struct {
	Type   string `json:"type"`
	UserID int64  `json:"userId"`
}

func (h Handler) Requests(w http.ResponseWriter, r *http.Request, userID string) {
	currentUserID, ok := parseUserID(w, userID)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	rows, err := h.DB.QueryContext(r.Context(), `SELECT gr.id, gr.group_id, gr.user_id,
		COALESCE(gr.invited_by, 0), gr.type, gr.status, g.title,
		COALESCE(NULLIF(u.nickname, ''), trim(u.first_name || ' ' || u.last_name), 'Member')
		FROM group_requests gr
		JOIN groups g ON g.id = gr.group_id
		JOIN users u ON u.id = CASE WHEN gr.type = 'invite' THEN gr.invited_by ELSE gr.user_id END
		WHERE gr.status = 'pending' AND (
			(gr.type = 'invite' AND gr.user_id = ?) OR
			(gr.type = 'join_request' AND g.creator_id = ?)
		)
		ORDER BY gr.created_at, gr.id`, currentUserID, currentUserID)
	if err != nil {
		log.Println("Error loading group requests:", err)
		response.WriteError(w, "Could not load group requests", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	requests := make([]models.GroupRequest, 0)
	for rows.Next() {
		var request models.GroupRequest
		if err := rows.Scan(&request.ID, &request.GroupID, &request.UserID, &request.InvitedBy,
			&request.Type, &request.Status, &request.GroupTitle, &request.ActorName); err != nil {
			log.Println("Error reading group request:", err)
			response.WriteError(w, "Could not load group requests", http.StatusInternalServerError)
			return
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		log.Println("Error reading group requests:", err)
		response.WriteError(w, "Could not load group requests", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"requests": requests})
}

func (h Handler) CreateRequest(w http.ResponseWriter, r *http.Request, userID string) {
	currentUserID, ok := parseUserID(w, userID)
	if !ok {
		return
	}
	groupID, ok := parseGroupID(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input groupRequestInput
	if err := httpx.DecodeJSON(w, r, 8*1024, &input); err != nil {
		response.WriteError(w, "Invalid group request", http.StatusBadRequest)
		return
	}
	if input.Type != "invite" && input.Type != "join_request" {
		response.WriteError(w, "Request type must be invite or join_request", http.StatusBadRequest)
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var creatorID int64
	var groupTitle string
	if err := h.DB.QueryRowContext(r.Context(), `SELECT creator_id, title FROM groups WHERE id = ?`, groupID).Scan(&creatorID, &groupTitle); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteError(w, "Group not found", http.StatusNotFound)
			return
		}
		log.Println("Error loading group for request:", err)
		response.WriteError(w, "Could not create group request", http.StatusInternalServerError)
		return
	}
	targetUserID := currentUserID
	var invitedBy any
	if input.Type == "invite" {
		if input.UserID <= 0 {
			response.WriteError(w, "Choose a username from the search results", http.StatusBadRequest)
			return
		}
		var isMember bool
		if err := h.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)`, groupID, currentUserID).Scan(&isMember); err != nil {
			response.WriteError(w, "Could not check group permissions", http.StatusInternalServerError)
			return
		}
		if !isMember {
			response.WriteError(w, "Only group members can invite people", http.StatusForbidden)
			return
		}
		err := h.DB.QueryRowContext(r.Context(), `SELECT id FROM users WHERE id = ?`, input.UserID).Scan(&targetUserID)
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteError(w, "No account found for that username", http.StatusNotFound)
			return
		}
		if err != nil {
			response.WriteError(w, "Could not find that account", http.StatusInternalServerError)
			return
		}
		if targetUserID == currentUserID {
			response.WriteError(w, "You are already a member of this group", http.StatusConflict)
			return
		}
		invitedBy = currentUserID
	} else if creatorID == currentUserID {
		response.WriteError(w, "You already own this group", http.StatusConflict)
		return
	}

	var isMember bool
	if err := h.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)`, groupID, targetUserID).Scan(&isMember); err != nil {
		response.WriteError(w, "Could not check group membership", http.StatusInternalServerError)
		return
	}
	if isMember {
		response.WriteError(w, "That user is already a group member", http.StatusConflict)
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		response.WriteError(w, "Could not create group request", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	var pending bool
	if err := tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM group_requests WHERE group_id = ? AND user_id = ? AND status = 'pending')`, groupID, targetUserID).Scan(&pending); err != nil {
		response.WriteError(w, "Could not check existing requests", http.StatusInternalServerError)
		return
	}
	if pending {
		response.WriteError(w, "A pending request already exists", http.StatusConflict)
		return
	}
	result, err := tx.ExecContext(r.Context(), `INSERT INTO group_requests (group_id, user_id, invited_by, type) VALUES (?, ?, ?, ?)`, groupID, targetUserID, invitedBy, input.Type)
	if err != nil {
		log.Println("Error creating group request:", err)
		response.WriteError(w, "Could not create group request", http.StatusInternalServerError)
		return
	}
	requestID, err := result.LastInsertId()
	if err != nil {
		response.WriteError(w, "Could not create group request", http.StatusInternalServerError)
		return
	}
	notificationUserID := targetUserID
	notificationType := models.NotificationGroupInvite
	if input.Type == "join_request" {
		notificationUserID = creatorID
		notificationType = models.NotificationGroupJoinRequest
	}
	notification := models.Notification{
		UserID: notificationUserID, ActorID: currentUserID, Type: notificationType,
		EntityID: requestID, EntityType: "group_request", Content: "You were invited to join " + groupTitle + ".",
	}
	if input.Type == "join_request" {
		notification.Content = "Someone requested to join " + groupTitle + "."
	}
	if err = repo.CreateNotification(tx, &notification); err != nil {
		log.Println("Error saving group request notification:", err)
		response.WriteError(w, "Could not notify the user about this request", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Println("Error committing group request:", err)
		response.WriteError(w, "Could not create group request", http.StatusInternalServerError)
		return
	}
	if h.WebSocket != nil {
		websocket.SendNotification(h.WebSocket, notification)
	}
	invitedByID := int64(0)
	if input.Type == "invite" {
		invitedByID = currentUserID
	}
	response.WriteJSON(w, http.StatusCreated, map[string]any{"request": models.GroupRequest{
		ID: requestID, GroupID: groupID, UserID: targetUserID, InvitedBy: invitedByID, Type: input.Type, Status: "pending",
	}})
}

func (h Handler) DecideRequest(w http.ResponseWriter, r *http.Request, userID string) {
	currentUserID, ok := parseUserID(w, userID)
	if !ok {
		return
	}
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", "PATCH")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	requestID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || requestID <= 0 {
		response.WriteError(w, "Invalid group request ID", http.StatusBadRequest)
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, 4*1024, &input); err != nil || (input.Status != "accepted" && input.Status != "declined") {
		response.WriteError(w, "Status must be accepted or declined", http.StatusBadRequest)
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		response.WriteError(w, "Could not update group request", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	var request models.GroupRequest
	var creatorID int64
	err = tx.QueryRowContext(r.Context(), `SELECT gr.id, gr.group_id, gr.user_id, COALESCE(gr.invited_by, 0), gr.type, gr.status, g.creator_id
		FROM group_requests gr JOIN groups g ON g.id = gr.group_id WHERE gr.id = ?`, requestID).
		Scan(&request.ID, &request.GroupID, &request.UserID, &request.InvitedBy, &request.Type, &request.Status, &creatorID)
	if errors.Is(err, sql.ErrNoRows) {
		response.WriteError(w, "Group request not found", http.StatusNotFound)
		return
	}
	if err != nil {
		response.WriteError(w, "Could not load group request", http.StatusInternalServerError)
		return
	}
	allowed := request.Type == "invite" && request.UserID == currentUserID || request.Type == "join_request" && creatorID == currentUserID
	if !allowed {
		response.WriteError(w, "You cannot decide this request", http.StatusForbidden)
		return
	}
	if request.Status != "pending" {
		response.WriteError(w, "This request has already been decided", http.StatusConflict)
		return
	}
	if input.Status == "accepted" {
		if _, err := tx.ExecContext(r.Context(), `INSERT OR IGNORE INTO group_members (group_id, user_id) VALUES (?, ?)`, request.GroupID, request.UserID); err != nil {
			log.Println("Error accepting group request:", err)
			response.WriteError(w, "Could not accept group request", http.StatusInternalServerError)
			return
		}
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE group_requests SET status = ? WHERE id = ? AND status = 'pending'`, input.Status, requestID); err != nil {
		response.WriteError(w, "Could not update group request", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM notifications WHERE user_id = ? AND entity_id = ? AND type IN (?, ?)`, currentUserID, requestID, models.NotificationGroupInvite, models.NotificationGroupJoinRequest); err != nil {
		response.WriteError(w, "Could not update request notification", http.StatusInternalServerError)
		return
	}
	var notification models.Notification
	if input.Status == "accepted" && request.Type == "join_request" {
		notification = models.Notification{
			UserID: request.UserID, ActorID: currentUserID, Type: models.NotificationGroupAccepted,
			EntityID: request.GroupID, EntityType: "group", Content: "Your request to join the group was accepted.",
		}
		if err := repo.CreateNotification(tx, &notification); err != nil {
			response.WriteError(w, "Could not notify the new member", http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		log.Println("Error committing group request decision:", err)
		response.WriteError(w, "Could not update group request", http.StatusInternalServerError)
		return
	}
	request.Status = input.Status
	if h.WebSocket != nil {
		if notification.ID != 0 {
			websocket.SendNotification(h.WebSocket, notification)
		}
		websocket.SendEventToUser(h.WebSocket, currentUserID, "notifications_read", nil)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"request": request})
}
