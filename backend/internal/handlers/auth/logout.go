package auth

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"social-network/internal/pkg/response"
)

func (h Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cookie, err := r.Cookie("session_token")
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		response.WriteError(w, "Invalid session", http.StatusBadRequest)
		return
	}
	if cookie != nil {
		if h.DB == nil {
			response.WriteError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if _, err := h.DB.Exec(`DELETE FROM sessions WHERE token = ?`, cookie.Value); err != nil {
			log.Println("Error deleting logout session:", err)
			response.WriteError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if h.CloseSession != nil {
			h.CloseSession(cookie.Value)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name: "session_token", Value: "", Path: "/", HttpOnly: true,
		Secure: strings.HasPrefix(os.Getenv("APP_ORIGIN"), "https://"), SameSite: http.SameSiteLaxMode,
		Expires: time.Unix(1, 0), MaxAge: -1,
	})
	response.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "message": "Logged out successfully"})
}
