package posts

import (
	"net/http"
	"path/filepath"
	"strconv"

	"social-network/internal/pkg/response"
)

// Image serves an upload only when the signed-in user can view its post.
func (h Handler) Image(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	viewerID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil || viewerID <= 0 {
		response.WriteError(w, "Not logged in", http.StatusUnauthorized)
		return
	}
	name := r.PathValue("name")
	if name == "" || filepath.Base(name) != name {
		http.NotFound(w, r)
		return
	}
	imagePath := "/uploads/posts/" + name
	var allowed bool
	err = h.DB.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM posts p WHERE (p.image_path = ? OR EXISTS (
			SELECT 1 FROM comments c WHERE c.post_id = p.id AND c.image_path = ?
		)) AND (
			p.user_id = ? OR p.privacy = 'public' OR
			(p.privacy = 'followers' AND EXISTS (
				SELECT 1 FROM follows f WHERE f.follower_id = ?
				AND f.following_id = p.user_id AND f.status = 'accepted'
			)) OR
			(p.privacy = 'private' AND EXISTS (
				SELECT 1 FROM post_allowed_users pau WHERE pau.post_id = p.id AND pau.user_id = ?
			) AND EXISTS (
				SELECT 1 FROM follows f WHERE f.follower_id = ?
				AND f.following_id = p.user_id AND f.status = 'accepted'
			))
		)
	)`, imagePath, imagePath, viewerID, viewerID, viewerID, viewerID).Scan(&allowed)
	if err != nil {
		response.WriteError(w, "Could not load image", http.StatusInternalServerError)
		return
	}
	if !allowed {
		response.WriteError(w, "Forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filepath.Join("uploads", "posts", name))
}
