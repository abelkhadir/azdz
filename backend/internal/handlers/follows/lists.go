package follows

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"social-network/internal/pkg/response"
)

// List responses contain identity summaries, not private account fields.
type userSummary struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
}

func (h Handler) Followers(w http.ResponseWriter, r *http.Request, viewerID int64) {
	h.listRelationships(w, r, viewerID, true)
}

func (h Handler) Following(w http.ResponseWriter, r *http.Request, viewerID int64) {
	h.listRelationships(w, r, viewerID, false)
}

func (h Handler) listRelationships(w http.ResponseWriter, r *http.Request, viewerID int64, followers bool) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
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

	// Authorize and list within one read snapshot so a visibility change cannot
	// leave a stale authorization decision attached to a newer list.
	tx, err := h.DB.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		h.writeUsers(w, nil, err)
		return
	}
	defer tx.Rollback()
	var canView bool
	err = tx.QueryRowContext(r.Context(), `SELECT is_public = 1 OR id = ? OR EXISTS (
		SELECT 1 FROM follows WHERE follower_id = ? AND following_id = users.id AND status = 'accepted'
		) FROM users WHERE id = ?`, viewerID, viewerID, id).Scan(&canView)
	if errors.Is(err, sql.ErrNoRows) {
		response.WriteError(w, "Profile not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.writeUsers(w, nil, err)
		return
	}
	if !canView {
		response.WriteError(w, "This profile is private", http.StatusForbidden)
		return
	}

	query := `SELECT u.id, u.first_name, u.last_name, COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM follows f JOIN users u ON u.id = f.following_id
		WHERE f.follower_id = ? AND f.status = 'accepted' ORDER BY f.created_at DESC, f.id DESC`
	if followers {
		query = `SELECT u.id, u.first_name, u.last_name, COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
			FROM follows f JOIN users u ON u.id = f.follower_id
			WHERE f.following_id = ? AND f.status = 'accepted' ORDER BY f.created_at DESC, f.id DESC`
	}
	rows, err := tx.QueryContext(r.Context(), query, id)
	h.writeUsers(w, rows, err)
}

func (h Handler) writeUsers(w http.ResponseWriter, rows *sql.Rows, err error) {
	if err != nil {
		log.Println("Error loading follow list:", err)
		response.WriteError(w, "Could not load follow list", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	users := []userSummary{}
	for rows.Next() {
		var user userSummary
		if err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Nickname, &user.Avatar); err != nil {
			log.Println("Error reading follow list:", err)
			response.WriteError(w, "Could not load follow list", http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		log.Println("Error reading follow list:", err)
		response.WriteError(w, "Could not load follow list", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "users": users})
}
