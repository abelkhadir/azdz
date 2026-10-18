package middleware

import (
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"

	"social-network/internal/pkg/response"
)

// IsAllowedOrigin is shared by HTTP and WebSocket requests. In production,
// APP_ORIGIN must be the browser-facing origin, for example https://app.example.com.
func IsAllowedOrigin(origin string) bool {
	if configured := strings.TrimRight(strings.TrimSpace(os.Getenv("APP_ORIGIN")), "/"); configured != "" {
		parsed, err := url.Parse(configured)
		return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") &&
			parsed.Host != "" && parsed.User == nil && parsed.Path == "" &&
			parsed.RawQuery == "" && parsed.Fragment == "" && origin == configured
	}
	return origin == "http://localhost:3000" || origin == "http://localhost:3001" ||
		origin == "http://127.0.0.1:3000"
}

func trustedMutation(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		return IsAllowedOrigin(origin)
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		parsed, err := url.Parse(referer)
		return err == nil && parsed.User == nil && IsAllowedOrigin(parsed.Scheme+"://"+parsed.Host)
	}
	// Non-browser JSON clients do not send Origin. Browsers cannot send this
	// content type cross-origin without a successful CORS preflight. Forms with
	// neither Origin nor Referer are rejected, including multipart registration.
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && contentType == "application/json"
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		w.Header().Add("Vary", "Origin")
		if IsAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead && !trustedMutation(r) {
			response.WriteError(w, "Untrusted request origin", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
