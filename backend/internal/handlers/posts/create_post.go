package posts

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/pkg/imageupload"
	"social-network/internal/pkg/response"
	"strings"
	"unicode/utf8"
)

// creates a normal post for the requesting user.
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

	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)

	if err := r.ParseMultipartForm(6 << 20); err != nil {
		http.Error(w, "Invalid upload or request too large", http.StatusBadRequest)
		return
	}

	var request postRequest

	request.Title = strings.TrimSpace(r.FormValue("title"))
	request.Content = strings.TrimSpace(r.FormValue("content"))
	request.Privacy = strings.TrimSpace(r.FormValue("privacy"))
	if request.Title == "" || utf8.RuneCountInString(request.Title) > 100 {
		http.Error(w, "Title is required and must be at most 100 characters", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(request.Content) > 5000 {
		http.Error(w, "Post content must be at most 5000 characters", http.StatusBadRequest)
		return
	}

	allowedUsersJSON := r.FormValue("allowedUserIds")
	if allowedUsersJSON != "" {
		if err := json.Unmarshal([]byte(allowedUsersJSON), &request.AllowedUserIDs); err != nil {
			http.Error(w, "Invalid allowed users", http.StatusBadRequest)
			return
		}
	}

	var imageDiskPath string
	keepImage := false

	defer func() {
		if imageDiskPath != "" && !keepImage {
			_ = os.Remove(imageDiskPath)
		}
	}()

	file, _, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		imageDiskPath, request.ImagePath, err = imageupload.Save(file, "uploads/posts", "/uploads/posts")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else if !errors.Is(err, http.ErrMissingFile) {
		http.Error(w, "Invalid image upload", http.StatusBadRequest)
		return
	}

	if request.Content == "" && request.ImagePath == "" {
		http.Error(w, "Post content or image is required", http.StatusBadRequest)
		return
	}

	if request.Privacy == "" {
		request.Privacy = "public"
	}

	if !validPostPrivacy(request.Privacy) {
		http.Error(w, "Invalid post privacy", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		`INSERT INTO posts (user_id, title, content, image_path, privacy) VALUES (?, ?, ?, ?, ?)`,
		userID,
		request.Title,
		request.Content,
		request.ImagePath,
		request.Privacy,
	)
	if err != nil {
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	postID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	if request.Privacy == "private" {
		for _, allowedUserID := range request.AllowedUserIDs {
			if allowedUserID <= 0 || allowedUserID == userID {
				continue
			}

			follows, err := isFollowerTx(tx, userID, allowedUserID)
			if err != nil {
				http.Error(w, "Failed to check followers", http.StatusInternalServerError)
				return
			}

			if !follows {
				http.Error(w, "Private posts can only include followers", http.StatusBadRequest)
				return
			}

			if _, err := tx.Exec(
				`INSERT OR IGNORE INTO post_allowed_users (post_id, user_id) VALUES (?, ?)`,
				postID,
				allowedUserID,
			); err != nil {
				http.Error(w, "Failed to save allowed users", http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	keepImage = true

	response.WriteJSON(w, http.StatusCreated, models.Post{
		ID:        postID,
		UserID:    userID,
		Title:     request.Title,
		Content:   request.Content,
		ImagePath: request.ImagePath,
		Privacy:   request.Privacy,
	})
}
