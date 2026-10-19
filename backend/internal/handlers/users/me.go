package users

import (
	"database/sql"
	"errors"
	"log"
	"mime"
	"net/http"

	"social-network/internal/pkg/httpx"
	"social-network/internal/pkg/response"
)

func (h Handler) Me(w http.ResponseWriter, r *http.Request, userID int64) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet && r.Method != http.MethodPatch {
		w.Header().Set("Allow", "GET, PATCH")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPatch && !h.updateVisibility(w, r, userID) {
		return
	}
	var user profile
	err := h.DB.QueryRowContext(r.Context(), `SELECT u.id, u.email, u.first_name, u.last_name,
		u.date_of_birth, COALESCE(u.avatar_path, ''), COALESCE(u.nickname, ''),
		COALESCE(u.about_me, ''), u.is_public
		FROM users u WHERE u.id = ?`, userID).
		Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName,
			&user.DateOfBirth, &user.Avatar, &user.Nickname, &user.AboutMe, &user.IsPublic)
	if errors.Is(err, sql.ErrNoRows) {
		response.WriteError(w, "Invalid session", http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Println("Error loading current user:", err)
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "user": user})
}

func (h Handler) updateVisibility(w http.ResponseWriter, r *http.Request, userID int64) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		response.WriteError(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return false
	}

	var req struct {
		IsPublic *bool `json:"isPublic"`
	}
	err = httpx.DecodeJSON(w, r, 1024, &req)
	if err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			response.WriteError(w, "Request is too large", http.StatusRequestEntityTooLarge)
		} else {
			response.WriteError(w, "Send a JSON object containing only isPublic as a boolean", http.StatusBadRequest)
		}
		return false
	}
	if req.IsPublic == nil {
		response.WriteError(w, "isPublic is required and must be a boolean", http.StatusBadRequest)
		return false
	}

	result, err := h.DB.ExecContext(r.Context(), `UPDATE users SET is_public = ? WHERE id = ?`, *req.IsPublic, userID)
	if err != nil {
		log.Println("Error updating profile visibility:", err)
		response.WriteError(w, "Could not update profile visibility", http.StatusInternalServerError)
		return false
	}
	count, err := result.RowsAffected()
	if err != nil {
		log.Println("Error checking profile visibility update:", err)
		response.WriteError(w, "Could not update profile visibility", http.StatusInternalServerError)
		return false
	}
	if count == 0 {
		response.WriteError(w, "Invalid session", http.StatusUnauthorized)
		return false
	}
	return true
}
