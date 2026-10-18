package users

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"social-network/internal/models"
	"social-network/internal/pkg/response"
)

type profile struct {
	models.User
	IsPublic bool `json:"isPublic"`
}

func (h Handler) Profile(w http.ResponseWriter, r *http.Request, viewerID string) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.WriteError(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var user profile
	var followStatus string
	err = h.DB.QueryRowContext(r.Context(), `SELECT u.id, u.email, u.first_name, u.last_name,
		u.date_of_birth, COALESCE(u.avatar_path, ''), COALESCE(u.nickname, ''),
		COALESCE(u.about_me, ''), u.is_public,
		COALESCE((SELECT f.status FROM follows f WHERE f.follower_id = ?
			AND f.following_id = u.id AND f.status IN ('pending', 'accepted')), 'none')
		FROM users u WHERE u.id = ?`, viewerID, id).
		Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName,
			&user.DateOfBirth, &user.Avatar, &user.Nickname, &user.AboutMe,
			&user.IsPublic, &followStatus)
	if errors.Is(err, sql.ErrNoRows) {
		response.WriteError(w, "Profile not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("Error loading profile:", err)
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	isOwnProfile := user.ID == viewerID
	if !user.IsPublic && !isOwnProfile && followStatus != "accepted" {
		response.WriteJSON(w, http.StatusForbidden, map[string]any{
			"success": false, "message": "This profile is private",
			"followStatus": followStatus, "isOwnProfile": false,
		})
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true, "user": user, "followStatus": followStatus, "isOwnProfile": isOwnProfile,
	})
}
