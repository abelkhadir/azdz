package auth

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"social-network/internal/middleware"
	"social-network/internal/pkg/response"

	"github.com/gofrs/uuid/v5"
)

const sessionLifetime = 7 * 24 * time.Hour

func (h Handler) checkExistingSession(w http.ResponseWriter, r *http.Request) bool {
	if h.DB == nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return true
	}
	cookie, err := r.Cookie("session_token")
	if errors.Is(err, http.ErrNoCookie) {
		return false
	}
	if err != nil {
		response.WriteError(w, "Invalid session cookie", http.StatusBadRequest)
		return true
	}
	_, err = middleware.CheckSession(h.DB, cookie.Value)
	if errors.Is(err, middleware.ErrInvalidSession) {
		return false
	}
	if err != nil {
		log.Println("Error checking session:", err)
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return true
	}
	response.WriteError(w, "Already logged in", http.StatusConflict)
	return true
}

func createSession(db *sql.DB, userID string) (*http.Cookie, error) {
	if db == nil {
		return nil, errors.New("auth database is not initialized")
	}
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(sessionLifetime)
	if _, err := db.Exec(`INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)`, userID, id.String(), expiresAt); err != nil {
		return nil, err
	}
	return &http.Cookie{
		Name: "session_token", Value: id.String(), Path: "/", HttpOnly: true,
		Secure: strings.HasPrefix(os.Getenv("APP_ORIGIN"), "https://"),
		SameSite: http.SameSiteLaxMode, Expires: expiresAt, MaxAge: int(sessionLifetime.Seconds()),
	}, nil
}
