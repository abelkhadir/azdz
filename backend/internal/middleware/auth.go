package middleware

import (
	"context"
	"database/sql"
	"net/http"

	"social-network/internal/pkg/response"
)

// Authenticate protects every route except the two public authentication routes.
func Authenticate(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/register" {
			next.ServeHTTP(w, r)
			return
		}

		userID, err := UserIDFromRequest(db, r)
		if err != nil {
			response.WriteError(w, "Not logged in", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
