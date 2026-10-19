package avatar

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"social-network/internal/pkg/imageupload"
	"social-network/internal/pkg/response"
)

const avatarDir = "uploads/avatars"

// Save returns the image URL and whether registration can continue.
func Save(w http.ResponseWriter, r *http.Request) (string, bool) {
	file, header, err := r.FormFile("avatar")
	if err == http.ErrMissingFile {
		return "", true // Avatar is optional.
	}
	if err != nil {
		response.WriteError(w, "Invalid avatar upload", http.StatusBadRequest)
		return "", false
	}
	defer file.Close()

	if header.Size > 5<<20 {
		response.WriteError(w, "Avatar must be at most 5 MB", http.StatusBadRequest)
		return "", false
	}

	_, url, err := imageupload.Save(file, avatarDir, "/uploads/avatars")
	if err != nil {
		if errors.Is(err, imageupload.ErrInvalidImage) {
			response.WriteError(w, err.Error(), http.StatusBadRequest)
		} else {
			response.WriteError(w, "Something went wrong with avatar upload", http.StatusInternalServerError)
		}
		return "", false
	}
	return url, true
}

// Remove deletes a saved avatar using its image URL.
func Remove(url string) error {
	return os.Remove(filepath.Join(avatarDir, filepath.Base(url)))
}
