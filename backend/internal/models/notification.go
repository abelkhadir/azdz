package models

const (
	NotificationFollowRequest    = "follow_request"
	NotificationFollow           = "follow"
	NotificationFollowAccepted   = "follow_accepted"
	NotificationPrivateMessage   = "private_message"
	NotificationGroupMessage     = "group_message"
	NotificationGroupAccepted    = "group_join_accepted"
	NotificationGroupInvite      = "group_invite"
	NotificationGroupJoinRequest = "group_join_request"
	NotificationGroupEvent       = "group_event"
)

type Notification struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"userId"`
	ActorID    int64  `json:"actorId,omitempty"`
	Type       string `json:"type"`
	EntityID   int64  `json:"entityId,omitempty"`
	EntityType string `json:"entityType,omitempty"`
	Content    string `json:"content"`
	IsRead     bool   `json:"isRead"`
	CreatedAt  string `json:"createdAt,omitempty"`
}

type ChatUnreadCounter struct {
	ActorID     int64 `json:"actorId"`
	UnreadCount int   `json:"unreadCount"`
}

type GroupUnreadCounter struct {
	GroupID     int64 `json:"groupId"`
	UnreadCount int   `json:"unreadCount"`
}
