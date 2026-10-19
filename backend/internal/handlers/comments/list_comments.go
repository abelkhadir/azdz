package comments

import (
	"net/http"

	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/pkg/httpx"
	"social-network/internal/pkg/repo"
	"social-network/internal/pkg/response"
)

// returns comments for a post the user can view.
func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, err := middleware.UserIDFromRequest(h.DB, r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	postID, err := httpx.QueryID(r, "postId")
	if err != nil {
		http.Error(w, "Invalid post id", http.StatusBadRequest)
		return
	}

	allowed, err := repo.CanViewPost(h.DB, userID, postID)
	if err != nil {
		http.Error(w, "Failed to check post access", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	limit, err := httpx.QueryLimit(r, 20, 50)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}
	afterID, err := httpx.OptionalQueryID(r, "afterId")
	if err != nil {
		http.Error(w, "Invalid after ID", http.StatusBadRequest)
		return
	}
	rows, err := h.DB.Query(
		`SELECT c.id, c.post_id, c.user_id, COALESCE(c.content, ''), COALESCE(c.image_path, ''), c.created_at,
			u.id, u.first_name, u.last_name, COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM comments c JOIN users u ON u.id = c.user_id
		WHERE c.post_id = ? AND (? = 0 OR c.id > ?)
		ORDER BY c.id ASC
		LIMIT CASE WHEN ? > 0 THEN ? ELSE -1 END`,
		postID, afterID, afterID, limit, limit,
	)
	if err != nil {
		http.Error(w, "Failed to load comments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	comments := []models.Comment{}
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.ImagePath, &comment.CreatedAt,
			&comment.Author.ID, &comment.Author.FirstName, &comment.Author.LastName, &comment.Author.Nickname, &comment.Author.Avatar); err != nil {
			http.Error(w, "Failed to load comments", http.StatusInternalServerError)
			return
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to load comments", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, comments)
}
