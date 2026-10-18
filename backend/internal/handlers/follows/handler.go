package follows

import (
	"database/sql"
	"net/http"
	"strconv"

	"social-network/internal/pkg/response"
	"social-network/internal/websocket"
)

type Handler struct {
	DB        *sql.DB
	WebSocket *websocket.Manager
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.WriteError(w, "Invalid ID", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}
