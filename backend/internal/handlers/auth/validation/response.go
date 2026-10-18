package validation

import (
	"net/http"

	"social-network/internal/pkg/response"
)

func WriteError(w http.ResponseWriter, errors map[string]string) {
	response.WriteJSON(w, http.StatusBadRequest, map[string]any{
		"success": false,
		"message": "Some registration fields are invalid",
		"errors":  errors,
	})
}
