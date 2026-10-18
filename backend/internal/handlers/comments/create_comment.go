package comments

import (
	"errors"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/pkg/httpx"
	"social-network/internal/pkg/imageupload"
	"social-network/internal/pkg/response"
)

// Create a comment to a post the user can view.
func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := middleware.UserIDFromRequest(h.DB, r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	request, imageFile, ok := readCommentRequest(w, r)
	if !ok {
		return
	}
	if imageFile != nil {
		defer imageFile.Close()
	}

	request.Content = strings.TrimSpace(request.Content)
	if request.PostID <= 0 {
		http.Error(w, "Invalid post id", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(request.Content) > 2000 {
		http.Error(w, "Comment must be at most 2000 characters", http.StatusBadRequest)
		return
	}
	if request.Content == "" && imageFile == nil {
		http.Error(w, "Comment content or image is required", http.StatusBadRequest)
		return
	}

	allowed, err := canViewPost(h.DB, userID, request.PostID)
	if err != nil {
		http.Error(w, "Failed to check post access", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var imageDiskPath, imagePath string
	keepImage := false
	defer func() {
		if imageDiskPath != "" && !keepImage {
			_ = os.Remove(imageDiskPath)
		}
	}()
	if imageFile != nil {
		imageDiskPath, imagePath, err = imageupload.Save(imageFile, "uploads/posts", "/uploads/posts")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	result, err := h.DB.Exec(
		`INSERT INTO comments (post_id, user_id, content, image_path) VALUES (?, ?, ?, ?)`,
		request.PostID, userID, request.Content, imagePath)
	if err != nil {
		http.Error(w, "Failed to create comment", http.StatusInternalServerError)
		return
	}
	commentID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Failed to create comment", http.StatusInternalServerError)
		return
	}
	comment := models.Comment{
		ID:        commentID,
		PostID:    request.PostID,
		UserID:    userID,
		Content:   request.Content,
		ImagePath: imagePath,
	}
	if err := h.DB.QueryRow(
		`SELECT c.created_at, u.id, u.first_name, u.last_name,
			COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM comments c JOIN users u ON u.id = c.user_id
		WHERE c.id = ?`, commentID,
	).Scan(&comment.CreatedAt, &comment.Author.ID, &comment.Author.FirstName,
		&comment.Author.LastName, &comment.Author.Nickname, &comment.Author.Avatar); err != nil {
		http.Error(w, "Failed to load created comment", http.StatusInternalServerError)
		return
	}
	keepImage = true
	response.WriteJSON(w, http.StatusCreated, comment)
}

func readCommentRequest(w http.ResponseWriter, r *http.Request) (commentRequest, multipart.File, bool) {
	var request commentRequest
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := httpx.DecodeJSON(w, r, 16*1024, &request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return request, nil, false
		}
		return request, nil, true
	}

	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		http.Error(w, "Invalid upload or request too large", http.StatusBadRequest)
		return request, nil, false
	}
	request.Content = r.FormValue("content")
	request.PostID, _ = strconv.ParseInt(r.FormValue("postId"), 10, 64)
	file, _, err := r.FormFile("image")
	if errors.Is(err, http.ErrMissingFile) {
		return request, nil, true
	}
	if err != nil {
		http.Error(w, "Invalid image upload", http.StatusBadRequest)
		return request, nil, false
	}
	return request, file, true
}
