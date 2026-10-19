package groups

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-sqlite3"

	"social-network/internal/models"
	"social-network/internal/pkg/httpx"
	"social-network/internal/pkg/response"
	"social-network/internal/websocket"
)

const (
	maxTitleLength       = 100
	maxDescriptionLength = 2000
)

type Handler struct {
	DB        *sql.DB
	WebSocket *websocket.Manager
}

type groupInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (h Handler) Collection(w http.ResponseWriter, r *http.Request, currentUserID int64) {
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.list(w, r, currentUserID)
	case http.MethodPost:
		h.create(w, r, currentUserID)
	default:
		w.Header().Set("Allow", "GET, POST")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h Handler) Group(w http.ResponseWriter, r *http.Request, currentUserID int64) {
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
		group, err := loadGroup(r, h.DB, groupID, currentUserID)
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteError(w, "Group not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Println("Error loading group:", err)
			response.WriteError(w, "Could not load group", http.StatusInternalServerError)
			return
		}
		response.WriteJSON(w, http.StatusOK, map[string]any{"group": group})
	case http.MethodPatch:
		h.update(w, r, groupID, currentUserID)
	default:
		w.Header().Set("Allow", "GET, PATCH")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h Handler) Members(w http.ResponseWriter, r *http.Request, _ int64) {
	groupID, ok := parseGroupID(w, r)
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
	var exists bool
	if err := h.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM groups WHERE id = ?)`, groupID).Scan(&exists); err != nil {
		log.Println("Error checking group:", err)
		response.WriteError(w, "Could not load group members", http.StatusInternalServerError)
		return
	}
	if !exists {
		response.WriteError(w, "Group not found", http.StatusNotFound)
		return
	}
	limit, offset, ok := parseListPage(w, r)
	if !ok {
		return
	}

	rows, err := h.DB.QueryContext(r.Context(), `SELECT CAST(u.id AS TEXT), u.first_name, u.last_name,
		COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM group_members gm JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = ? ORDER BY gm.joined_at, u.id LIMIT ? OFFSET ?`, groupID, limit, offset)
	if err != nil {
		log.Println("Error loading group members:", err)
		response.WriteError(w, "Could not load group members", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	members := make([]models.GroupMemberProfile, 0)
	for rows.Next() {
		var member models.GroupMemberProfile
		if err := rows.Scan(&member.ID, &member.FirstName, &member.LastName, &member.Nickname, &member.Avatar); err != nil {
			log.Println("Error reading group member:", err)
			response.WriteError(w, "Could not load group members", http.StatusInternalServerError)
			return
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		log.Println("Error reading group members:", err)
		response.WriteError(w, "Could not load group members", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"members": members})
}

func (h Handler) Membership(w http.ResponseWriter, r *http.Request, currentUserID int64) {
	groupID, ok := parseGroupID(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", "DELETE")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		log.Println("Error starting group leave:", err)
		response.WriteError(w, "Could not leave group", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var creatorID int64
	if err := tx.QueryRowContext(r.Context(), `SELECT creator_id FROM groups WHERE id = ?`, groupID).Scan(&creatorID); errors.Is(err, sql.ErrNoRows) {
		response.WriteError(w, "Group not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("Error checking group owner:", err)
		response.WriteError(w, "Could not leave group", http.StatusInternalServerError)
		return
	}

	if creatorID == currentUserID {
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM notifications
			WHERE (type IN ('group_message', 'group_join_accepted', 'group_event') AND entity_id = ?)
			   OR (type IN ('group_invite', 'group_join_request') AND entity_id IN (
				SELECT id FROM group_requests WHERE group_id = ?
			))`, groupID, groupID); err != nil {
			log.Println("Error deleting group notifications:", err)
			response.WriteError(w, "Could not delete group", http.StatusInternalServerError)
			return
		}
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM groups WHERE id = ? AND creator_id = ?`, groupID, currentUserID); err != nil {
			log.Println("Error deleting group:", err)
			response.WriteError(w, "Could not delete group", http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			log.Println("Error saving group deletion:", err)
			response.WriteError(w, "Could not delete group", http.StatusInternalServerError)
			return
		}
		response.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "groupDeleted": true})
		return
	}

	result, err := tx.ExecContext(r.Context(), `DELETE FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, currentUserID)
	if err != nil {
		log.Println("Error leaving group:", err)
		response.WriteError(w, "Could not leave group", http.StatusInternalServerError)
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		log.Println("Error checking group leave result:", err)
		response.WriteError(w, "Could not leave group", http.StatusInternalServerError)
		return
	}
	if changed == 0 {
		response.WriteError(w, "You are not a member of this group", http.StatusConflict)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Println("Error saving group leave:", err)
		response.WriteError(w, "Could not leave group", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "groupDeleted": false})
}

func (h Handler) list(w http.ResponseWriter, r *http.Request, currentUserID int64) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(query) > maxDescriptionLength {
		response.WriteError(w, "Search is too long", http.StatusBadRequest)
		return
	}
	limit, offset, ok := parseListPage(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.QueryContext(r.Context(), `SELECT g.id, g.creator_id, g.title, g.description,
		(SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.id),
		EXISTS(SELECT 1 FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ?),
		COALESCE((SELECT gr.status FROM group_requests gr WHERE gr.group_id = g.id AND gr.user_id = ? AND gr.status = 'pending' ORDER BY gr.id DESC LIMIT 1), '')
		FROM groups g
		WHERE ? = '' OR instr(lower(g.title || ' ' || g.description), lower(?)) > 0
		ORDER BY lower(g.title), g.id LIMIT ? OFFSET ?`, currentUserID, currentUserID, query, query, limit, offset)
	if err != nil {
		log.Println("Error listing groups:", err)
		response.WriteError(w, "Could not load groups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	groups := make([]models.Group, 0)
	for rows.Next() {
		var group models.Group
		if err := rows.Scan(&group.ID, &group.CreatorID, &group.Title, &group.Description, &group.MemberCount, &group.IsMember, &group.RequestStatus); err != nil {
			log.Println("Error reading group:", err)
			response.WriteError(w, "Could not load groups", http.StatusInternalServerError)
			return
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		log.Println("Error reading groups:", err)
		response.WriteError(w, "Could not load groups", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"groups": groups})
}

func (h Handler) create(w http.ResponseWriter, r *http.Request, currentUserID int64) {
	input, ok := decodeGroupInput(w, r)
	if !ok {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if !validInput(w, input) {
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		log.Println("Error starting group creation:", err)
		response.WriteError(w, "Could not create group", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(r.Context(), `INSERT INTO groups (creator_id, title, description) VALUES (?, ?, ?)`, currentUserID, input.Title, input.Description)
	if err != nil {
		if isDuplicateGroupTitle(err) {
			response.WriteError(w, "A group with this name already exists", http.StatusConflict)
			return
		}
		log.Println("Error creating group:", err)
		response.WriteError(w, "Could not create group", http.StatusInternalServerError)
		return
	}
	groupID, err := result.LastInsertId()
	if err != nil {
		log.Println("Error reading new group ID:", err)
		response.WriteError(w, "Could not create group", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO group_members (group_id, user_id) VALUES (?, ?)`, groupID, currentUserID); err != nil {
		log.Println("Error adding group creator as member:", err)
		response.WriteError(w, "Could not create group", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Println("Error saving group:", err)
		response.WriteError(w, "Could not create group", http.StatusInternalServerError)
		return
	}

	group := models.Group{ID: groupID, CreatorID: currentUserID, Title: input.Title, Description: input.Description, MemberCount: 1, IsMember: true}
	response.WriteJSON(w, http.StatusCreated, map[string]any{"group": group})
}

func (h Handler) update(w http.ResponseWriter, r *http.Request, groupID, currentUserID int64) {
	input, ok := decodeGroupInput(w, r)
	if !ok {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if !validInput(w, input) {
		return
	}

	result, err := h.DB.ExecContext(r.Context(), `UPDATE groups SET title = ?, description = ? WHERE id = ? AND creator_id = ?`, input.Title, input.Description, groupID, currentUserID)
	if err != nil {
		if isDuplicateGroupTitle(err) {
			response.WriteError(w, "A group with this name already exists", http.StatusConflict)
			return
		}
		log.Println("Error updating group:", err)
		response.WriteError(w, "Could not update group", http.StatusInternalServerError)
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		log.Println("Error checking group update:", err)
		response.WriteError(w, "Could not update group", http.StatusInternalServerError)
		return
	}
	if changed == 0 {
		var exists bool
		if err := h.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM groups WHERE id = ?)`, groupID).Scan(&exists); err != nil {
			log.Println("Error checking group:", err)
			response.WriteError(w, "Could not update group", http.StatusInternalServerError)
			return
		}
		if !exists {
			response.WriteError(w, "Group not found", http.StatusNotFound)
			return
		}
		response.WriteError(w, "Only the group creator can edit this group", http.StatusForbidden)
		return
	}

	group, err := loadGroup(r, h.DB, groupID, currentUserID)
	if err != nil {
		log.Println("Error reloading updated group:", err)
		response.WriteError(w, "Could not update group", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"group": group})
}

func decodeGroupInput(w http.ResponseWriter, r *http.Request) (groupInput, bool) {
	var input groupInput
	if err := httpx.DecodeJSON(w, r, 32*1024, &input); err != nil {
		response.WriteError(w, "Send a valid group title and description", http.StatusBadRequest)
		return groupInput{}, false
	}
	return input, true
}

func validInput(w http.ResponseWriter, input groupInput) bool {
	if input.Title == "" {
		response.WriteError(w, "Group title is required", http.StatusBadRequest)
		return false
	}
	if utf8.RuneCountInString(input.Title) > maxTitleLength {
		response.WriteError(w, "Group title must be 100 characters or fewer", http.StatusBadRequest)
		return false
	}
	if utf8.RuneCountInString(input.Description) > maxDescriptionLength {
		response.WriteError(w, "Group description must be 2000 characters or fewer", http.StatusBadRequest)
		return false
	}
	return true
}

func isDuplicateGroupTitle(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) &&
		(sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique || sqliteErr.ExtendedCode == sqlite3.ErrConstraintTrigger)
}

func loadGroup(r *http.Request, db *sql.DB, groupID, currentUserID int64) (models.Group, error) {
	var group models.Group
	err := db.QueryRowContext(r.Context(), `SELECT g.id, g.creator_id, g.title, g.description,
		(SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.id),
		EXISTS(SELECT 1 FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ?),
		COALESCE((SELECT gr.status FROM group_requests gr WHERE gr.group_id = g.id AND gr.user_id = ? AND gr.status = 'pending' ORDER BY gr.id DESC LIMIT 1), '')
		FROM groups g WHERE g.id = ?`, currentUserID, currentUserID, groupID).
		Scan(&group.ID, &group.CreatorID, &group.Title, &group.Description, &group.MemberCount, &group.IsMember, &group.RequestStatus)
	return group, err
}

func parseGroupID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.WriteError(w, "Invalid group ID", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func parseListPage(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	limit := 20
	offset := 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit <= 0 || limit > 50 {
			response.WriteError(w, "Invalid limit", http.StatusBadRequest)
			return 0, 0, false
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			response.WriteError(w, "Invalid offset", http.StatusBadRequest)
			return 0, 0, false
		}
	}
	return limit, offset, true
}
