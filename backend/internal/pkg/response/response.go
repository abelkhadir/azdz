package response

import (
	"encoding/json"
	"net/http"
)

func WriteError(w http.ResponseWriter, message string, status int) {
	WriteJSON(w, status, map[string]any{
		"success": false,
		"message": message,
	})
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
