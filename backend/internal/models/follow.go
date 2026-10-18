package models

import "time"

type Follow struct {
	ID          int64     `json:"id"`
	FollowerID  int64     `json:"followerId"`
	FollowingID int64     `json:"followingId"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}
