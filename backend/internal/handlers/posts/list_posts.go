package posts

import (
	"net/http"

	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/pkg/httpx"
	"social-network/internal/pkg/repo"
	"social-network/internal/pkg/response"
)

// returns posts visible to the requesting user.
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

	ownerID, err := httpx.OptionalQueryID(r, "ownerId")
	if err != nil || (r.URL.Query().Has("ownerId") && ownerID <= 0) {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		return
	}
	limit, err := httpx.QueryLimit(r, 20, 50)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}
	beforeID, err := httpx.OptionalQueryID(r, "beforeId")
	if err != nil {
		http.Error(w, "Invalid before ID", http.StatusBadRequest)
		return
	}
	args := []any{ownerID, ownerID, beforeID, beforeID}
	args = append(args, repo.PostVisibilityArgs(userID)...)
	args = append(args, limit, limit)
	rows, err := h.DB.Query(
		`SELECT p.id, p.user_id, COALESCE(p.title, ''), COALESCE(p.content, ''), COALESCE(p.image_path, ''), p.privacy, p.created_at,
			u.id, u.first_name, u.last_name, COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM posts p JOIN users u ON u.id = p.user_id
		WHERE
			(? = 0 OR p.user_id = ?)
			AND (? = 0 OR p.id < ?)
			AND `+repo.PostVisibilitySQL+`
		ORDER BY p.id DESC
		LIMIT CASE WHEN ? > 0 THEN ? ELSE -1 END`,
		args...,
	)
	if err != nil {
		http.Error(w, "Failed to load posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	posts := []models.Post{}
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.ImagePath, &post.Privacy, &post.CreatedAt,
			&post.Author.ID, &post.Author.FirstName, &post.Author.LastName, &post.Author.Nickname, &post.Author.Avatar); err != nil {
			http.Error(w, "Failed to load posts", http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	response.WriteJSON(w, http.StatusOK, posts)
}
