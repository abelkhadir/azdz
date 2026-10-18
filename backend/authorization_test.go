package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/gorilla/websocket"

	"social-network/internal/models"
	appdb "social-network/internal/pkg/db"
)

const testOrigin = "http://localhost:3000"

type testApp struct {
	t      *testing.T
	db     *sql.DB
	server *httptest.Server
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	migrationsPath := filepath.Join("internal", "pkg", "db", "migrations", "sqlite")
	if err := appdb.RunMigrations(dbPath, migrationsPath); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	database, err := appdb.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	// SQLite settings such as foreign_keys apply per connection.
	database.SetMaxOpenConns(1)

	app := &testApp{t: t, db: database}
	app.server = httptest.NewServer(newRouter(database))
	t.Cleanup(func() {
		app.server.Close()
		database.Close()
	})
	return app
}

func (a *testApp) createUser(name string, public bool) int64 {
	a.t.Helper()

	result, err := a.db.Exec(
		`INSERT INTO users
			(email, password_hash, first_name, last_name, date_of_birth, is_public)
		 VALUES (?, 'test-hash', ?, 'User', '2000-01-01', ?)`,
		name+"@example.com", name, public,
	)
	if err != nil {
		a.t.Fatalf("create user %q: %v", name, err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		a.t.Fatalf("read user id: %v", err)
	}
	return userID
}

func (a *testApp) createSession(userID int64) string {
	a.t.Helper()

	token := uuid.Must(uuid.NewV4()).String()
	if _, err := a.db.Exec(
		`INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)`,
		userID, token, time.Now().Add(time.Hour),
	); err != nil {
		a.t.Fatalf("create session: %v", err)
	}
	return token
}

func (a *testApp) request(method, path, token string, body any) *http.Response {
	a.t.Helper()

	var requestBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
			a.t.Fatalf("encode request: %v", err)
		}
	}

	request, err := http.NewRequest(method, a.server.URL+path, &requestBody)
	if err != nil {
		a.t.Fatalf("create request: %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet && method != http.MethodHead {
		request.Header.Set("Origin", testOrigin)
	}
	if token != "" {
		request.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	}

	response, err := a.server.Client().Do(request)
	if err != nil {
		a.t.Fatalf("send request: %v", err)
	}
	a.t.Cleanup(func() { response.Body.Close() })
	return response
}

func (a *testApp) multipartRequest(path, token string, fields map[string]string) *http.Response {
	a.t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			a.t.Fatalf("write multipart field: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		a.t.Fatalf("close multipart body: %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, a.server.URL+path, &body)
	if err != nil {
		a.t.Fatalf("create multipart request: %v", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Origin", testOrigin)
	request.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	response, err := a.server.Client().Do(request)
	if err != nil {
		a.t.Fatalf("send multipart request: %v", err)
	}
	a.t.Cleanup(func() { response.Body.Close() })
	return response
}

func assertStatus(t *testing.T, response *http.Response, expected int) {
	t.Helper()
	if response.StatusCode != expected {
		t.Fatalf("expected status %d, got %d", expected, response.StatusCode)
	}
}

func TestProtectedRoutesRequireSession(t *testing.T) {
	app := newTestApp(t)

	tests := []struct {
		name string
		path string
	}{
		{name: "posts", path: "/api/posts"},
		{name: "comments", path: "/api/comments?postId=1"},
		{name: "private chat", path: "/api/messages/history?withUserId=1"},
		{name: "group chat", path: "/api/group-messages/history?groupId=1"},
		{name: "notifications", path: "/api/notifications"},
		{name: "profile", path: "/profile/1"},
		{name: "uploaded post image", path: "/uploads/posts/missing.png"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := app.request(http.MethodGet, test.path, "", nil)
			assertStatus(t, response, http.StatusUnauthorized)
		})
	}

	wsURL := "ws" + strings.TrimPrefix(app.server.URL, "http") + "/ws"
	connection, response, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Origin": []string{testOrigin},
	})
	if connection != nil {
		connection.Close()
	}
	if err == nil {
		t.Fatal("expected unauthenticated WebSocket connection to fail")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected WebSocket status %d, got %v", http.StatusUnauthorized, response)
	}
	response.Body.Close()
}

func TestHTTPAuthorizationUsesSessionIdentity(t *testing.T) {
	app := newTestApp(t)
	ownerID := app.createUser("owner", true)
	attackerID := app.createUser("attacker", true)
	attackerToken := app.createSession(attackerID)

	result, err := app.db.Exec(
		`INSERT INTO posts (user_id, content, privacy) VALUES (?, 'private post', 'private')`,
		ownerID,
	)
	if err != nil {
		t.Fatalf("create private post: %v", err)
	}
	privatePostID, _ := result.LastInsertId()

	// A client-provided userId must not let someone comment as another user.
	response := app.request(http.MethodPost, "/api/comments", attackerToken, map[string]any{
		"postId":  privatePostID,
		"userId":  ownerID,
		"content": "spoofed comment",
	})
	assertStatus(t, response, http.StatusBadRequest)

	response = app.request(http.MethodPost, "/api/comments", attackerToken, map[string]any{
		"postId":  privatePostID,
		"content": "forbidden comment",
	})
	assertStatus(t, response, http.StatusForbidden)

	response = app.request(
		http.MethodGet,
		fmt.Sprintf("/api/comments?postId=%d", privatePostID),
		attackerToken,
		nil,
	)
	assertStatus(t, response, http.StatusForbidden)

	response = app.request(
		http.MethodDelete,
		fmt.Sprintf("/api/posts?id=%d", privatePostID),
		attackerToken,
		nil,
	)
	assertStatus(t, response, http.StatusNotFound)

	var remaining int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM posts WHERE id = ?`, privatePostID).Scan(&remaining); err != nil {
		t.Fatalf("check private post: %v", err)
	}
	if remaining != 1 {
		t.Fatal("another user was able to delete the private post")
	}

	result, err = app.db.Exec(
		`INSERT INTO posts (user_id, content, privacy) VALUES (?, 'public post', 'public')`,
		ownerID,
	)
	if err != nil {
		t.Fatalf("create public post: %v", err)
	}
	publicPostID, _ := result.LastInsertId()

	response = app.request(http.MethodPost, "/api/comments", attackerToken, map[string]any{
		"postId":  publicPostID,
		"content": "real author comes from session",
	})
	assertStatus(t, response, http.StatusCreated)

	var storedAuthorID int64
	if err := app.db.QueryRow(
		`SELECT user_id FROM comments WHERE post_id = ?`, publicPostID,
	).Scan(&storedAuthorID); err != nil {
		t.Fatalf("load created comment: %v", err)
	}
	if storedAuthorID != attackerID {
		t.Fatalf("comment author = %d, want authenticated user %d", storedAuthorID, attackerID)
	}
}

func TestChatAuthorizationRejectsUnrelatedUsersAndNonMembers(t *testing.T) {
	app := newTestApp(t)
	ownerID := app.createUser("group-owner", true)
	outsiderID := app.createUser("outsider", true)
	outsiderToken := app.createSession(outsiderID)

	result, err := app.db.Exec(
		`INSERT INTO groups (creator_id, title, description) VALUES (?, 'Private group', 'test')`,
		ownerID,
	)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	groupID, _ := result.LastInsertId()
	if _, err := app.db.Exec(
		`INSERT INTO group_members (group_id, user_id) VALUES (?, ?)`, groupID, ownerID,
	); err != nil {
		t.Fatalf("add group owner: %v", err)
	}

	response := app.request(
		http.MethodGet,
		fmt.Sprintf("/api/messages/history?withUserId=%d", ownerID),
		outsiderToken,
		nil,
	)
	assertStatus(t, response, http.StatusForbidden)

	response = app.request(
		http.MethodGet,
		fmt.Sprintf("/api/group-messages/history?groupId=%d", groupID),
		outsiderToken,
		nil,
	)
	assertStatus(t, response, http.StatusForbidden)

	connection := app.dialWebSocket(outsiderToken)
	defer connection.Close()

	app.writeWebSocketEvent(connection, models.WebSocketPrivateMessage, map[string]any{
		"receiverId": ownerID,
		"content":    "not allowed",
		"clientId":   "private-denied",
	})
	assertWebSocketError(t, connection, "private-denied")

	app.writeWebSocketEvent(connection, models.WebSocketGroupMessage, map[string]any{
		"groupId":  groupID,
		"content":  "not a member",
		"clientId": "group-denied",
	})
	assertWebSocketError(t, connection, "group-denied")

	var privateMessages, groupMessages int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM private_messages`).Scan(&privateMessages); err != nil {
		t.Fatalf("count private messages: %v", err)
	}
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM group_messages`).Scan(&groupMessages); err != nil {
		t.Fatalf("count group messages: %v", err)
	}
	if privateMessages != 0 || groupMessages != 0 {
		t.Fatalf("denied messages were stored: private=%d group=%d", privateMessages, groupMessages)
	}
}

func TestWebSocketMessageSenderComesFromSession(t *testing.T) {
	app := newTestApp(t)
	senderID := app.createUser("sender", true)
	receiverID := app.createUser("receiver", true)
	senderToken := app.createSession(senderID)

	// A single accepted follow in either direction permits chat.
	if _, err := app.db.Exec(
		`INSERT INTO follows (follower_id, following_id, status) VALUES (?, ?, 'accepted')`,
		senderID, receiverID,
	); err != nil {
		t.Fatalf("create accepted follow: %v", err)
	}

	connection := app.dialWebSocket(senderToken)
	defer connection.Close()

	app.writeWebSocketEvent(connection, models.WebSocketPrivateMessage, map[string]any{
		"receiverId": receiverID,
		"senderId":   receiverID,
		"content":    "spoofed",
		"clientId":   "private-spoofed",
	})
	var rejected models.MessageStatus
	readWebSocketEvent(t, connection, models.WebSocketMessageError, &rejected)

	var messageCount int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM private_messages`).Scan(&messageCount); err != nil {
		t.Fatalf("count rejected messages: %v", err)
	}
	if messageCount != 0 {
		t.Fatal("message containing a client-provided senderId was stored")
	}

	app.writeWebSocketEvent(connection, models.WebSocketPrivateMessage, map[string]any{
		"receiverId": receiverID,
		"content":    "hello",
		"clientId":   "private-allowed",
	})

	var received models.PrivateMessage
	readWebSocketEvent(t, connection, models.WebSocketPrivateMessage, &received)
	if received.SenderID != senderID {
		t.Fatalf("message sender = %d, want session user %d", received.SenderID, senderID)
	}
	if received.ReceiverID != receiverID {
		t.Fatalf("message receiver = %d, want %d", received.ReceiverID, receiverID)
	}

	var storedSenderID int64
	if err := app.db.QueryRow(
		`SELECT sender_id FROM private_messages WHERE id = ?`, received.ID,
	).Scan(&storedSenderID); err != nil {
		t.Fatalf("load stored message: %v", err)
	}
	if storedSenderID != senderID {
		t.Fatalf("stored sender = %d, want session user %d", storedSenderID, senderID)
	}
}

func TestGroupPostsAndCommentsRequireMembership(t *testing.T) {
	app := newTestApp(t)
	ownerID := app.createUser("group-post-owner", true)
	outsiderID := app.createUser("group-post-outsider", true)
	ownerToken := app.createSession(ownerID)
	outsiderToken := app.createSession(outsiderID)

	result, err := app.db.Exec(
		`INSERT INTO groups (creator_id, title, description) VALUES (?, 'Members only', 'test')`,
		ownerID,
	)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	groupID, _ := result.LastInsertId()
	if _, err := app.db.Exec(
		`INSERT INTO group_members (group_id, user_id) VALUES (?, ?)`, groupID, ownerID,
	); err != nil {
		t.Fatalf("add owner to group: %v", err)
	}

	postsPath := fmt.Sprintf("/api/groups/%d/posts", groupID)
	assertStatus(t, app.request(http.MethodGet, postsPath, outsiderToken, nil), http.StatusForbidden)
	assertStatus(t, app.request(http.MethodPost, postsPath, outsiderToken, map[string]any{
		"content": "outsider post",
	}), http.StatusForbidden)

	response := app.request(http.MethodPost, postsPath, ownerToken, map[string]any{
		"content": "member post",
	})
	assertStatus(t, response, http.StatusCreated)
	var created struct {
		Post models.GroupPost `json:"post"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode group post: %v", err)
	}

	commentsPath := fmt.Sprintf("/api/group-comments?postId=%d", created.Post.ID)
	assertStatus(t, app.request(http.MethodGet, commentsPath, outsiderToken, nil), http.StatusForbidden)
	assertStatus(t, app.request(http.MethodPost, commentsPath, outsiderToken, map[string]any{
		"content": "outsider comment",
	}), http.StatusForbidden)
	assertStatus(t, app.request(http.MethodPost, commentsPath, ownerToken, map[string]any{
		"content": "member comment",
	}), http.StatusCreated)

	var outsiderPosts, outsiderComments int
	if err := app.db.QueryRow(
		`SELECT COUNT(*) FROM group_posts WHERE group_id = ? AND user_id = ?`, groupID, outsiderID,
	).Scan(&outsiderPosts); err != nil {
		t.Fatalf("count outsider posts: %v", err)
	}
	if err := app.db.QueryRow(
		`SELECT COUNT(*) FROM group_comments WHERE post_id = ? AND user_id = ?`, created.Post.ID, outsiderID,
	).Scan(&outsiderComments); err != nil {
		t.Fatalf("count outsider comments: %v", err)
	}
	if outsiderPosts != 0 || outsiderComments != 0 {
		t.Fatalf("outsider content was stored: posts=%d comments=%d", outsiderPosts, outsiderComments)
	}
}

func TestPostTitleAndCommentImagePathValidation(t *testing.T) {
	app := newTestApp(t)
	userID := app.createUser("post-validation", true)
	token := app.createSession(userID)

	assertStatus(t, app.multipartRequest("/api/posts", token, map[string]string{
		"content": "missing title", "privacy": "public",
	}), http.StatusBadRequest)
	assertStatus(t, app.multipartRequest("/api/posts", token, map[string]string{
		"title": strings.Repeat("x", 101), "content": "too long", "privacy": "public",
	}), http.StatusBadRequest)

	response := app.multipartRequest("/api/posts", token, map[string]string{
		"title": "A valid title", "content": "post body", "privacy": "public",
	})
	assertStatus(t, response, http.StatusCreated)
	var post models.Post
	if err := json.NewDecoder(response.Body).Decode(&post); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	if post.Title != "A valid title" {
		t.Fatalf("post title = %q", post.Title)
	}

	// Clients may upload a comment image, but cannot choose a server path.
	response = app.request(http.MethodPost, "/api/comments", token, map[string]any{
		"postId": post.ID, "content": "forged", "imagePath": "/uploads/posts/other.png",
	})
	assertStatus(t, response, http.StatusBadRequest)
}

func TestPostPrivacyProtectsImagesAndKeepsPublicPostsVisible(t *testing.T) {
	app := newTestApp(t)
	ownerID := app.createUser("private-profile-owner", false)
	viewerID := app.createUser("post-viewer", true)
	ownerToken := app.createSession(ownerID)
	viewerToken := app.createSession(viewerID)

	filename := "privacy-test-" + uuid.Must(uuid.NewV4()).String() + ".png"
	diskPath := filepath.Join("uploads", "posts", filename)
	if err := os.MkdirAll(filepath.Dir(diskPath), 0755); err != nil {
		t.Fatalf("create upload directory: %v", err)
	}
	if err := os.WriteFile(diskPath, []byte("private image test"), 0644); err != nil {
		t.Fatalf("create test image: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(diskPath) })
	imagePath := "/uploads/posts/" + filename

	if _, err := app.db.Exec(
		`INSERT INTO posts (user_id, title, content, image_path, privacy)
		 VALUES (?, 'Private image', 'secret', ?, 'private')`, ownerID, imagePath,
	); err != nil {
		t.Fatalf("create private image post: %v", err)
	}
	assertStatus(t, app.request(http.MethodGet, imagePath, viewerToken, nil), http.StatusForbidden)
	assertStatus(t, app.request(http.MethodGet, imagePath, ownerToken, nil), http.StatusOK)

	result, err := app.db.Exec(
		`INSERT INTO posts (user_id, title, content, privacy)
		 VALUES (?, 'Public post', 'visible despite private profile', 'public')`, ownerID,
	)
	if err != nil {
		t.Fatalf("create public post: %v", err)
	}
	publicPostID, _ := result.LastInsertId()
	response := app.request(http.MethodGet, fmt.Sprintf("/api/posts?ownerId=%d", ownerID), viewerToken, nil)
	assertStatus(t, response, http.StatusOK)
	var posts []models.Post
	if err := json.NewDecoder(response.Body).Decode(&posts); err != nil {
		t.Fatalf("decode visible posts: %v", err)
	}
	if len(posts) != 1 || posts[0].ID != publicPostID {
		t.Fatalf("visible posts = %+v, want public post %d only", posts, publicPostID)
	}
}

func (a *testApp) dialWebSocket(token string) *websocket.Conn {
	a.t.Helper()

	wsURL := "ws" + strings.TrimPrefix(a.server.URL, "http") + "/ws"
	header := http.Header{"Origin": []string{testOrigin}}
	header.Add("Cookie", (&http.Cookie{Name: "session_token", Value: token}).String())
	connection, response, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		a.t.Fatalf("connect WebSocket: %v", err)
	}
	return connection
}

func (a *testApp) writeWebSocketEvent(connection *websocket.Conn, eventType string, data any) {
	a.t.Helper()
	if err := connection.WriteJSON(map[string]any{"type": eventType, "data": data}); err != nil {
		a.t.Fatalf("write WebSocket event: %v", err)
	}
}

func assertWebSocketError(t *testing.T, connection *websocket.Conn, clientID string) {
	t.Helper()
	var status models.MessageStatus
	readWebSocketEvent(t, connection, models.WebSocketMessageError, &status)
	if status.ClientID != clientID {
		t.Fatalf("message error clientId = %q, want %q", status.ClientID, clientID)
	}
}

func readWebSocketEvent(t *testing.T, connection *websocket.Conn, wantedType string, target any) {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set WebSocket deadline: %v", err)
	}
	for {
		var event struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := connection.ReadJSON(&event); err != nil {
			t.Fatalf("read WebSocket event %q: %v", wantedType, err)
		}
		if event.Type != wantedType {
			continue
		}
		if err := json.Unmarshal(event.Data, target); err != nil {
			t.Fatalf("decode WebSocket event %q: %v", wantedType, err)
		}
		return
	}
}
