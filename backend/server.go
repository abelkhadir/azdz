package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"social-network/internal/handlers/auth"
	"social-network/internal/handlers/comments"
	"social-network/internal/handlers/follows"
	"social-network/internal/handlers/groups"
	"social-network/internal/handlers/messages"
	notificationshandler "social-network/internal/handlers/notifications"
	"social-network/internal/handlers/posts"
	"social-network/internal/handlers/users"
	"social-network/internal/middleware"
	"social-network/internal/pkg/db"
	"social-network/internal/websocket"
)

func main() {
	dbFile := os.Getenv("DATABASE_PATH")
	if dbFile == "" {
		dbFile = "social-network.db"
	}

	dbPath, err := filepath.Abs(dbFile)
	if err != nil {
		log.Fatal(err)
	}
	migrationsPath, err := filepath.Abs("internal/pkg/db/migrations/sqlite")
	if err != nil {
		log.Fatal(err)
	}

	if err := db.RunMigrations(dbPath, migrationsPath); err != nil {
		log.Fatal(err)
	}

	database, err := db.OpenDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server started at http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, newRouter(database)); err != nil {
		log.Fatal(err)
	}
}

func newRouter(database *sql.DB) http.Handler {
	wsManager := websocket.NewManager()
	authHandler := auth.Handler{DB: database, CloseSession: func(token string) {
		websocket.CloseSession(wsManager, token)
	}}
	messageHandler := messages.Handler{DB: database, WebSocket: wsManager}
	notificationHandler := notificationshandler.Handler{DB: database, WebSocket: wsManager}
	postHandler := posts.Handler{DB: database}
	commentHandler := comments.Handler{DB: database}
	userHandler := users.Handler{DB: database}
	followHandler := follows.Handler{DB: database, WebSocket: wsManager}
	groupHandler := groups.Handler{DB: database, WebSocket: wsManager}

	mux := http.NewServeMux()
	mux.HandleFunc("/uploads/avatars/{name}", middleware.RequireSession(database, userHandler.Avatar))
	mux.HandleFunc("/uploads/posts/{name}", middleware.RequireSession(database, postHandler.Image))
	mux.HandleFunc("/api/auth/register", authHandler.Register)
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/logout", authHandler.Logout)
	mux.HandleFunc("/api/auth/me", middleware.RequireSession(database, userHandler.Me))
	mux.HandleFunc("/api/users/search", middleware.RequireSession(database, userHandler.Search))
	mux.HandleFunc("/profile/{id}", middleware.RequireSession(database, userHandler.Profile))
	mux.HandleFunc("/api/users/{id}/follow", middleware.RequireSession(database, followHandler.Follow))
	mux.HandleFunc("/api/follow-requests", middleware.RequireSession(database, followHandler.Requests))
	mux.HandleFunc("/api/follow-requests/{id}", middleware.RequireSession(database, followHandler.DecideRequest))
	mux.HandleFunc("/api/users/{id}/followers", middleware.RequireSession(database, followHandler.Followers))
	mux.HandleFunc("/api/users/{id}/following", middleware.RequireSession(database, followHandler.Following))
	mux.Handle("/api/groups", middleware.RequireSession(database, groupHandler.Collection))
	mux.Handle("/api/groups/{id}", middleware.RequireSession(database, groupHandler.Group))
	mux.Handle("/api/groups/{id}/members", middleware.RequireSession(database, groupHandler.Members))
	mux.Handle("/api/groups/{id}/membership", middleware.RequireSession(database, groupHandler.Membership))
	mux.Handle("/api/groups/{id}/events", middleware.RequireSession(database, groupHandler.Events))
	mux.Handle("/api/groups/{id}/posts", middleware.RequireSession(database, groupHandler.Posts))
	mux.Handle("/api/group-comments", middleware.RequireSession(database, groupHandler.Comments))
	mux.Handle("/api/events/{id}/response", middleware.RequireSession(database, groupHandler.RespondToEvent))
	mux.Handle("/api/groups/{id}/requests", middleware.RequireSession(database, groupHandler.CreateRequest))
	mux.Handle("/api/group-requests", middleware.RequireSession(database, groupHandler.Requests))
	mux.Handle("/api/group-requests/{id}", middleware.RequireSession(database, groupHandler.DecideRequest))
	mux.HandleFunc("/api/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			postHandler.List(w, r)
		case http.MethodPost:
			postHandler.Create(w, r)
		case http.MethodDelete:
			postHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/comments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			commentHandler.List(w, r)
		case http.MethodPost:
			commentHandler.Create(w, r)
		case http.MethodDelete:
			commentHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/messages/history", messageHandler.History)
	mux.HandleFunc("/api/messages/conversations", messageHandler.Conversations)
	mux.HandleFunc("/api/group-messages/history", messageHandler.GroupHistory)
	mux.HandleFunc("/api/notifications", notificationHandler.ListNotifications)
	mux.HandleFunc("/api/notifications/read", notificationHandler.MarkNotificationsRead)
	mux.HandleFunc("/api/notifications/chat", notificationHandler.ListChatNotifications)
	mux.HandleFunc("/api/notifications/chat/read", notificationHandler.MarkChatNotificationsRead)
	mux.HandleFunc("/api/notifications/groups", notificationHandler.ListGroupNotifications)
	mux.HandleFunc("/api/notifications/groups/read", notificationHandler.MarkGroupNotificationsRead)
	mux.HandleFunc("/ws", messageHandler.Connect)

	// CORS handles browser preflight before authentication; it never calls a route for OPTIONS.
	return middleware.CORS(middleware.Authenticate(database, mux))
}
