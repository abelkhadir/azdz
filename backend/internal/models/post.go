package models

type Post struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"userId"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	ImagePath string `json:"imagePath"`
	Privacy   string `json:"privacy"`
	CreatedAt string `json:"createdAt,omitempty"`
	Author    Author `json:"author"`
}

type Comment struct {
	ID        int64  `json:"id"`
	PostID    int64  `json:"postId"`
	UserID    int64  `json:"userId"`
	Content   string `json:"content"`
	ImagePath string `json:"imagePath"`
	CreatedAt string `json:"createdAt,omitempty"`
	Author    Author `json:"author"`
}

// Author is the public identity displayed alongside a post or comment.
// It intentionally excludes account fields such as email and date of birth.
type Author struct {
	ID        int64  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
}

type PostAllowedUser struct {
	PostID int64 `json:"postId"`
	UserID int64 `json:"userId"`
}
