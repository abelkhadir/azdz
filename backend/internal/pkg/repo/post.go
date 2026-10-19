package repo

import "database/sql"

// PostVisibilitySQL applies all post privacy modes to the posts alias p.
// Use PostVisibilityArgs for its four viewer placeholders.
const PostVisibilitySQL = `(
	p.user_id = ? OR p.privacy = 'public' OR
	(p.privacy = 'followers' AND EXISTS (
		SELECT 1 FROM follows f WHERE f.follower_id = ?
		AND f.following_id = p.user_id AND f.status = 'accepted'
	)) OR
	(p.privacy = 'private' AND EXISTS (
		SELECT 1 FROM post_allowed_users pau WHERE pau.post_id = p.id AND pau.user_id = ?
	) AND EXISTS (
		SELECT 1 FROM follows f WHERE f.follower_id = ?
		AND f.following_id = p.user_id AND f.status = 'accepted'
	))
)`

func PostVisibilityArgs(viewerID int64) []any {
	return []any{viewerID, viewerID, viewerID, viewerID}
}

func CanViewPost(db *sql.DB, viewerID, postID int64) (bool, error) {
	args := append([]any{postID}, PostVisibilityArgs(viewerID)...)
	var visible bool
	err := db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM posts p WHERE p.id = ? AND `+PostVisibilitySQL+`
	)`, args...).Scan(&visible)
	return visible, err
}
