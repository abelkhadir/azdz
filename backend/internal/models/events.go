package models

type Event struct {
	ID            int64  `json:"id"`
	GroupID       int64  `json:"groupId"`
	CreatorID     int64  `json:"creatorId"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	DateTime      string `json:"dateTime"`
	GoingCount    int    `json:"goingCount"`
	NotGoingCount int    `json:"notGoingCount"`
	Response      string `json:"response,omitempty"`
}

type EventResponse struct {
	EventID  int64  `json:"eventId"`
	UserID   int64  `json:"userId"`
	Response string `json:"response"`
}
