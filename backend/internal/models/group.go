package models

type Group struct {
	ID            int64  `json:"id"`
	CreatorID     int64  `json:"creatorId"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	MemberCount   int64  `json:"memberCount"`
	IsMember      bool   `json:"isMember"`
	RequestStatus string `json:"requestStatus,omitempty"`
}

type GroupMember struct {
	GroupID int64 `json:"groupId"`
	UserID  int64 `json:"userId"`
}

type GroupMemberProfile struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
}

type GroupRequest struct {
	ID         int64  `json:"id"`
	GroupID    int64  `json:"groupId"`
	UserID     int64  `json:"userId"`
	InvitedBy  int64  `json:"invitedBy"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	GroupTitle string `json:"groupTitle"`
	ActorName  string `json:"actorName"`
}

type GroupPost struct {
	ID        int64          `json:"id"`
	GroupID   int64          `json:"groupId"`
	UserID    int64          `json:"userId"`
	Content   string         `json:"content"`
	ImagePath string         `json:"imagePath"`
	CreatedAt string         `json:"createdAt"`
	Author    Author         `json:"author"`
	Comments  []GroupComment `json:"comments,omitempty"`
}

type GroupComment struct {
	ID        int64  `json:"id"`
	PostID    int64  `json:"postId"`
	UserID    int64  `json:"userId"`
	Content   string `json:"content"`
	ImagePath string `json:"imagePath"`
	CreatedAt string `json:"createdAt"`
	Author    Author `json:"author"`
}
