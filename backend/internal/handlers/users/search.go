package users

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"social-network/internal/pkg/response"
)

type searchResult struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Nickname     string `json:"nickname"`
	Avatar       string `json:"avatar"`
	IsPublic     bool   `json:"isPublic"`
	FollowStatus string `json:"followStatus"`
}

// Search returns safe identity summaries for signed-in users. It deliberately
// never exposes email, birth date, or profile content for private accounts.
func (h Handler) Search(w http.ResponseWriter, r *http.Request, viewerID int64) {
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

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(query) > 100 {
		response.WriteError(w, "Search is too long", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("minChars") == "2" && utf8.RuneCountInString(query) < 2 {
		response.WriteError(w, "Enter at least 2 username characters", http.StatusBadRequest)
		return
	}
	pattern := "%" + escapeLike(strings.ToLower(query)) + "%"
	rows, err := h.DB.QueryContext(r.Context(), `SELECT u.id, u.first_name, u.last_name,
		COALESCE(u.nickname, ''), COALESCE(u.avatar_path, ''), u.is_public,
		COALESCE((SELECT f.status FROM follows f WHERE f.follower_id = ?
			AND f.following_id = u.id AND f.status IN ('pending', 'accepted')), 'none')
		FROM users u
		WHERE u.id != ? AND (? = '' OR (
			lower(COALESCE(u.nickname, '')) LIKE ? ESCAPE '\'
			OR lower(u.first_name) LIKE ? ESCAPE '\'
			OR lower(u.last_name) LIKE ? ESCAPE '\'
			OR lower(u.first_name || ' ' || u.last_name) LIKE ? ESCAPE '\'
		))
		ORDER BY lower(COALESCE(u.nickname, u.first_name || ' ' || u.last_name)), u.id
		LIMIT 20`, viewerID, viewerID, query, pattern, pattern, pattern, pattern)
	if err != nil {
		response.WriteError(w, "Could not search users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := make([]searchResult, 0)
	for rows.Next() {
		var user searchResult
		if err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Nickname, &user.Avatar, &user.IsPublic, &user.FollowStatus); err != nil {
			response.WriteError(w, "Could not search users", http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		response.WriteError(w, "Could not search users", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"users": users})
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "%", "\\%")
	return strings.ReplaceAll(value, "_", "\\_")
}
