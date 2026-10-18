package avatar

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"

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

	// Check the file content.
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		response.WriteError(w, "Failed to read avatar", http.StatusBadRequest)
		return "", false
	}
	var ext string
	switch http.DetectContentType(buf[:n]) {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	default:
		response.WriteError(w, "Only JPEG, PNG and GIF images are allowed", http.StatusBadRequest)
		return "", false
	}

	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		response.WriteError(w, "Something went wrong with avatar upload", http.StatusInternalServerError)
		return "", false
	}
	// CreateTemp generates a unique name and never overwrites an existing file.
	dst, err := os.CreateTemp(avatarDir, "avatar-*"+ext)
	if err != nil {
		response.WriteError(w, "Something went wrong with avatar upload", http.StatusInternalServerError)
		return "", false
	}
	reader := io.MultiReader(bytes.NewReader(buf[:n]), file)
	_, copyErr := io.Copy(dst, reader)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(dst.Name())
		response.WriteError(w, "Something went wrong with avatar upload", http.StatusInternalServerError)
		return "", false
	}
	return "/uploads/avatars/" + filepath.Base(dst.Name()), true
}

// Remove deletes a saved avatar using its image URL.
func Remove(url string) error {
	return os.Remove(filepath.Join(avatarDir, filepath.Base(url)))
}
