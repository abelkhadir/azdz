package posts

import (
	"database/sql"
)

// limits posts to the supported visibility modes.
func validPostPrivacy(privacy string) bool {
	return privacy == "public" || privacy == "followers" || privacy == "private"
}

func isFollowerTx(tx *sql.Tx, creatorID, followerID int64) (bool, error) {
	var exists int
	err := tx.QueryRow(
		`SELECT EXISTS (
			SELECT 1 FROM follows
			WHERE follower_id = ? AND following_id = ? AND status = 'accepted'
		)`,
		followerID,
		creatorID,
	).Scan(&exists)
	return exists == 1, err
}
