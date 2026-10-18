package auth

import (
	"database/sql"
	"errors"
	"log"
	"mime"
	"net/http"
	"strings"

	"social-network/internal/handlers/auth/validation"
	"social-network/internal/models"
	"social-network/internal/pkg/httpx"
	"social-network/internal/pkg/response"
	authservice "social-network/internal/services/auth"
)

func (h Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		response.WriteError(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	if h.checkExistingSession(w, r) {
		return
	}
	var req models.LoginRequest
	if err := httpx.DecodeJSON(w, r, formDataSize, &req); err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			response.WriteError(w, "Login request is too large", http.StatusRequestEntityTooLarge)
			return
		}
		response.WriteError(w, "invalid JSON", http.StatusBadRequest)
		log.Println("Error decoding JSON:", err)
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if errors := validation.Login(req); len(errors) > 0 {
		validation.WriteError(w, errors)
		log.Println("Validation errors:", errors)
		return
	}
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	var userID, passwordHash string
	err = h.DB.QueryRow(
		`SELECT id, password_hash FROM users WHERE email = ?`, req.Email,
	).Scan(&userID, &passwordHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Println("Error loading login user:", err)
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if errors.Is(err, sql.ErrNoRows) || authservice.ComparePassword(passwordHash, req.Password) != nil {
		response.WriteError(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}
	cookie, err := createSession(h.DB, userID)
	if err != nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, cookie)
	response.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Login successful",
		"userId":  userID,
	})
}
