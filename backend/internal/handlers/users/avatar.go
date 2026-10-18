package users

import (
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"social-network/internal/pkg/response"
)

// Avatar applies the same visibility rules as profile information.
func (h Handler) Avatar(w http.ResponseWriter, r *http.Request, viewerID string) {
	if r.Method != http.MethodGet {
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.PathValue("name")
	if name == "" || filepath.Base(name) != name {
		response.WriteError(w, "Avatar not found", http.StatusNotFound)
		return
	}
	var visible bool
	err := h.DB.QueryRowContext(r.Context(), `SELECT is_public = 1 OR id = ? OR EXISTS (
 SELECT 1 FROM follows WHERE follower_id = ? AND following_id = users.id AND status = 'accepted')
 FROM users WHERE avatar_path = ?`, viewerID, viewerID, "/uploads/avatars/"+name).Scan(&visible)
	if errors.Is(err, sql.ErrNoRows) {
		response.WriteError(w, "Avatar not found", http.StatusNotFound)
		return
	}
	if err != nil {
		response.WriteError(w, "Could not load avatar", http.StatusInternalServerError)
		return
	}
	if !visible {
		response.WriteError(w, "This profile is private", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, filepath.Join("uploads", "avatars", name))
}
