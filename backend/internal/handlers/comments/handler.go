package comments

import "database/sql"

// Handler provides normal post comment endpoints.
type Handler struct {
	DB *sql.DB
}

type commentRequest struct {
	PostID  int64  `json:"postId"`
	Content string `json:"content"`
}
