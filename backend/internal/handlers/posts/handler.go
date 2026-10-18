package posts

import "database/sql"

// provides normal post endpoints.
type Handler struct {
	DB *sql.DB
}

type postRequest struct {
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	ImagePath      string  `json:"imagePath"`
	Privacy        string  `json:"privacy"`
	AllowedUserIDs []int64 `json:"allowedUserIds"`
}
