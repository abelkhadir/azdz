package middleware

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"social-network/internal/pkg/response"
)

var ErrInvalidSession = errors.New("invalid session")

type userIDKey struct{}

// UserIDFromRequest returns the user authenticated by the session cookie.
// Client-provided IDs must never be used for authorization.
func UserIDFromRequest(db *sql.DB, r *http.Request) (int64, error) {
	if userID, ok := r.Context().Value(userIDKey{}).(int64); ok && userID > 0 {
		return userID, nil
	}

	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0, ErrInvalidSession
	}
	session, err := CheckSession(db, cookie.Value)
	return session.UserID, err
}

type Session struct {
	UserID    int64
	ExpiresAt time.Time
}

// CheckSession is the shared authentication check for HTTP and WebSocket actions.
// Always read the database: a connection must not cache permission to use a session.
func CheckSession(db *sql.DB, token string) (Session, error) {
	if db == nil {
		return Session{}, errors.New("session database is not initialized")
	}
	if _, err := uuid.FromString(token); err != nil {
		return Session{}, ErrInvalidSession
	}

	var session Session
	err := db.QueryRow(`SELECT sessions.user_id, sessions.expires_at
		FROM sessions JOIN users ON users.id = sessions.user_id
		WHERE sessions.token = ?`, token).Scan(&session.UserID, &session.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrInvalidSession
	}
	if err != nil {
		return Session{}, err
	}
	if session.UserID <= 0 || !session.ExpiresAt.After(time.Now()) {
		return Session{}, ErrInvalidSession
	}
	return session, nil
}

// RequireSession passes the verified user's ID to the protected handler.
func RequireSession(db *sql.DB, next func(http.ResponseWriter, *http.Request, int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := UserIDFromRequest(db, r)
		if err != nil {
			response.WriteError(w, "Not logged in", http.StatusUnauthorized)
			return
		}
		next(w, r, userID)
	}
}
