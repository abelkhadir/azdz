package posts

import (
	"net/http"
	"path/filepath"

	"social-network/internal/pkg/repo"
	"social-network/internal/pkg/response"
)

// Image serves an upload only when the signed-in user can view its post.
func (h Handler) Image(w http.ResponseWriter, r *http.Request, viewerID int64) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.PathValue("name")
	if name == "" || filepath.Base(name) != name {
		http.NotFound(w, r)
		return
	}
	imagePath := "/uploads/posts/" + name
	args := append([]any{imagePath, imagePath}, repo.PostVisibilityArgs(viewerID)...)
	var allowed bool
	err := h.DB.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM posts p WHERE (p.image_path = ? OR EXISTS (
			SELECT 1 FROM comments c WHERE c.post_id = p.id AND c.image_path = ?
		)) AND `+repo.PostVisibilitySQL+`
	)`, args...).Scan(&allowed)
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
