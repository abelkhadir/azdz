package groups

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"social-network/internal/models"
	"social-network/internal/pkg/httpx"
	"social-network/internal/pkg/response"
)

const maxGroupPostLength = 5000

type groupContentInput struct {
	Content string `json:"content"`
}

func (h Handler) Posts(w http.ResponseWriter, r *http.Request, userID string) {
	memberID, ok := parseUserID(w, userID)
	if !ok {
		return
	}
	groupID, ok := parseGroupID(w, r)
	if !ok || !requireMember(w, r, h.DB, groupID, memberID) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listPosts(w, r, groupID)
	case http.MethodPost:
		h.createPost(w, r, groupID, memberID)
	default:
		w.Header().Set("Allow", "GET, POST")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h Handler) listPosts(w http.ResponseWriter, r *http.Request, groupID int64) {
	limit, err := httpx.QueryLimit(r, 10, 50)
	if err != nil {
		response.WriteError(w, "Invalid limit", http.StatusBadRequest)
		return
	}
	beforeID, err := httpx.OptionalQueryID(r, "beforeId")
	if err != nil {
		response.WriteError(w, "Invalid beforeId", http.StatusBadRequest)
		return
	}
	rows, err := h.DB.QueryContext(r.Context(), `SELECT gp.id, gp.group_id, gp.user_id,
		COALESCE(gp.content, ''), COALESCE(gp.image_path, ''), gp.created_at,
		u.id, u.first_name, u.last_name, COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM group_posts gp JOIN users u ON u.id = gp.user_id
		WHERE gp.group_id = ? AND (? = 0 OR gp.id < ?)
		ORDER BY gp.id DESC LIMIT ?`, groupID, beforeID, beforeID, limit)
	if err != nil {
		response.WriteError(w, "Could not load group posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	posts := make([]models.GroupPost, 0)
	for rows.Next() {
		var post models.GroupPost
		if err := rows.Scan(&post.ID, &post.GroupID, &post.UserID, &post.Content, &post.ImagePath,
			&post.CreatedAt, &post.Author.ID, &post.Author.FirstName, &post.Author.LastName,
			&post.Author.Nickname, &post.Author.Avatar); err != nil {
			response.WriteError(w, "Could not load group posts", http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}
	if rows.Err() != nil {
		response.WriteError(w, "Could not load group posts", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"posts": posts})
}

func (h Handler) createPost(w http.ResponseWriter, r *http.Request, groupID, userID int64) {
	input, ok := decodeGroupContent(w, r, maxGroupPostLength)
	if !ok {
		return
	}
	result, err := h.DB.ExecContext(r.Context(),
		`INSERT INTO group_posts (group_id, user_id, content) VALUES (?, ?, ?)`, groupID, userID, input.Content)
	if err != nil {
		response.WriteError(w, "Could not create group post", http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	post, err := loadGroupPost(h.DB, id)
	if err != nil {
		response.WriteError(w, "Could not load group post", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusCreated, map[string]any{"post": post})
}

func (h Handler) Comments(w http.ResponseWriter, r *http.Request, userID string) {
	memberID, ok := parseUserID(w, userID)
	if !ok {
		return
	}
	postID, err := httpx.QueryID(r, "postId")
	if err != nil {
		response.WriteError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}
	var groupID int64
	if err := h.DB.QueryRowContext(r.Context(), `SELECT group_id FROM group_posts WHERE id = ?`, postID).Scan(&groupID); errors.Is(err, sql.ErrNoRows) {
		response.WriteError(w, "Group post not found", http.StatusNotFound)
		return
	} else if err != nil {
		response.WriteError(w, "Could not load group post", http.StatusInternalServerError)
		return
	}
	if !requireMember(w, r, h.DB, groupID, memberID) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listComments(w, r, postID)
	case http.MethodPost:
		h.createComment(w, r, postID, memberID)
	default:
		w.Header().Set("Allow", "GET, POST")
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h Handler) listComments(w http.ResponseWriter, r *http.Request, postID int64) {
	limit, err := httpx.QueryLimit(r, 10, 50)
	if err != nil {
		response.WriteError(w, "Invalid limit", http.StatusBadRequest)
		return
	}
	afterID, err := httpx.OptionalQueryID(r, "afterId")
	if err != nil {
		response.WriteError(w, "Invalid afterId", http.StatusBadRequest)
		return
	}
	rows, err := h.DB.QueryContext(r.Context(), `SELECT gc.id, gc.post_id, gc.user_id,
		COALESCE(gc.content, ''), COALESCE(gc.image_path, ''), gc.created_at,
		u.id, u.first_name, u.last_name, COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM group_comments gc JOIN users u ON u.id = gc.user_id
		WHERE gc.post_id = ? AND (? = 0 OR gc.id > ?) ORDER BY gc.id LIMIT ?`, postID, afterID, afterID, limit)
	if err != nil {
		response.WriteError(w, "Could not load comments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	comments := make([]models.GroupComment, 0)
	for rows.Next() {
		var comment models.GroupComment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Content,
			&comment.ImagePath, &comment.CreatedAt, &comment.Author.ID, &comment.Author.FirstName,
			&comment.Author.LastName, &comment.Author.Nickname, &comment.Author.Avatar); err != nil {
			response.WriteError(w, "Could not load comments", http.StatusInternalServerError)
			return
		}
		comments = append(comments, comment)
	}
	if rows.Err() != nil {
		response.WriteError(w, "Could not load comments", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"comments": comments})
}

func (h Handler) createComment(w http.ResponseWriter, r *http.Request, postID, userID int64) {
	input, ok := decodeGroupContent(w, r, 2000)
	if !ok {
		return
	}
	result, err := h.DB.ExecContext(r.Context(),
		`INSERT INTO group_comments (post_id, user_id, content) VALUES (?, ?, ?)`, postID, userID, input.Content)
	if err != nil {
		response.WriteError(w, "Could not create comment", http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	var comment models.GroupComment
	err = h.DB.QueryRowContext(r.Context(), `SELECT gc.id, gc.post_id, gc.user_id,
		COALESCE(gc.content, ''), COALESCE(gc.image_path, ''), gc.created_at,
		u.id, u.first_name, u.last_name, COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM group_comments gc JOIN users u ON u.id = gc.user_id WHERE gc.id = ?`, id).
		Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.ImagePath,
			&comment.CreatedAt, &comment.Author.ID, &comment.Author.FirstName, &comment.Author.LastName,
			&comment.Author.Nickname, &comment.Author.Avatar)
	if err != nil {
		response.WriteError(w, "Could not load comment", http.StatusInternalServerError)
		return
	}
	response.WriteJSON(w, http.StatusCreated, map[string]any{"comment": comment})
}

func decodeGroupContent(w http.ResponseWriter, r *http.Request, maximum int) (groupContentInput, bool) {
	var input groupContentInput
	if httpx.DecodeJSON(w, r, 16*1024, &input) != nil {
		response.WriteError(w, "Invalid content", http.StatusBadRequest)
		return input, false
	}
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" || utf8.RuneCountInString(input.Content) > maximum {
		response.WriteError(w, "Content is required and is too long", http.StatusBadRequest)
		return input, false
	}
	return input, true
}

func requireMember(w http.ResponseWriter, r *http.Request, db *sql.DB, groupID, userID int64) bool {
	var member bool
	if err := db.QueryRowContext(r.Context(), `SELECT EXISTS(
		SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)`, groupID, userID).Scan(&member); err != nil {
		response.WriteError(w, "Could not verify group membership", http.StatusInternalServerError)
		return false
	}
	if !member {
		response.WriteError(w, "Group membership required", http.StatusForbidden)
		return false
	}
	return true
}

func loadGroupPost(db *sql.DB, postID int64) (models.GroupPost, error) {
	var post models.GroupPost
	err := db.QueryRow(`SELECT gp.id, gp.group_id, gp.user_id, COALESCE(gp.content, ''),
		COALESCE(gp.image_path, ''), gp.created_at, u.id, u.first_name, u.last_name,
		COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM group_posts gp JOIN users u ON u.id = gp.user_id WHERE gp.id = ?`, postID).
		Scan(&post.ID, &post.GroupID, &post.UserID, &post.Content, &post.ImagePath,
			&post.CreatedAt, &post.Author.ID, &post.Author.FirstName, &post.Author.LastName,
			&post.Author.Nickname, &post.Author.Avatar)
	return post, err
}
