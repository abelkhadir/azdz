package comments

import (
	"database/sql"
)

// canViewPost applies post privacy rules before comment access.
func canViewPost(db *sql.DB, viewerID, postID int64) (bool, error) {
	var exists int
	err := db.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM posts p
			WHERE p.id = ?
			AND (
				p.user_id = ?
				OR p.privacy = 'public'
				OR (
					p.privacy = 'followers'
					AND EXISTS (
						SELECT 1 FROM follows f
						WHERE f.follower_id = ?
						AND f.following_id = p.user_id
						AND f.status = 'accepted'
					)
				)
				OR (
					p.privacy = 'private'
					AND EXISTS (
						SELECT 1 FROM post_allowed_users pau
						WHERE pau.post_id = p.id AND pau.user_id = ?
					)
					AND EXISTS (
						SELECT 1 FROM follows f
						WHERE f.follower_id = ? AND f.following_id = p.user_id AND f.status = 'accepted'
					)
				)
			)
		)`,
		postID, viewerID, viewerID, viewerID, viewerID).Scan(&exists)
	return exists == 1, err
}
