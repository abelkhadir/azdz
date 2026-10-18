package models

type PrivateMessage struct {
	ID         int64  `json:"id"`
	SenderID   int64  `json:"senderId"`
	ReceiverID int64  `json:"receiverId"`
	Content    string `json:"content"`
	CreatedAt  string `json:"createdAt,omitempty"`
}

type PrivateMessageRequest struct {
	ReceiverID int64  `json:"receiverId"`
	Content    string `json:"content"`
	ClientID   string `json:"clientId"`
}

type WebSocketEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

const (
	WebSocketPrivateMessage = "private_message"
	WebSocketGroupMessage   = "group_message"
	WebSocketNotification   = "notification"
	WebSocketMessageSent    = "message_sent"
	WebSocketMessageError   = "message_error"
)

// MessageStatus is sent only to the submitting client, so the UI can retain a
// draft when a realtime message is rejected or cannot be stored.
type MessageStatus struct {
	ClientID string `json:"clientId"`
	Message  string `json:"message,omitempty"`
}

type Conversation struct {
	UserID        int64  `json:"userId"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	Nickname      string `json:"nickname"`
	AvatarPath    string `json:"avatarPath"`
	LastMessage   string `json:"lastMessage"`
	LastMessageAt string `json:"lastMessageAt"`
}

type GroupMessage struct {
	ID         int64  `json:"id"`
	GroupID    int64  `json:"groupId"`
	SenderID   int64  `json:"senderId"`
	SenderName string `json:"senderName"`
	Content    string `json:"content"`
	CreatedAt  string `json:"createdAt,omitempty"`
}

type GroupMessageRequest struct {
	GroupID  int64  `json:"groupId"`
	Content  string `json:"content"`
	ClientID string `json:"clientId"`
}
