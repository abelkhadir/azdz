package websocket

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"social-network/internal/middleware"
)

const maxMessageSize = 8 << 10

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
	return middleware.IsAllowedOrigin(r.Header.Get("Origin"))
}}

type IncomingEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type EventHandler func(*Client, IncomingEvent)

// Serve authenticates one socket and passes valid events to the message handler.
func Serve(manager *Manager, database *sql.DB, w http.ResponseWriter, r *http.Request, handle EventHandler) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	session, err := middleware.CheckSession(database, cookie.Value)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	connection.SetReadLimit(maxMessageSize)
	if err := connection.SetReadDeadline(session.ExpiresAt); err != nil {
		connection.Close()
		return
	}

	client := &Client{
		UserID: session.UserID, SessionToken: cookie.Value,
		connection: connection, send: make(chan []byte, 64),
	}
	addClient(manager, client)
	go writeMessages(client)

	defer func() {
		removeClient(manager, client)
		connection.Close()
	}()

	for {
		_, payload, err := connection.ReadMessage()
		if err != nil {
			return
		}
		// Logout or server-side session deletion takes effect before the next action.
		current, err := middleware.CheckSession(database, cookie.Value)
		if err != nil || current.UserID != client.UserID {
			return
		}

		var event IncomingEvent
		if err := Decode(payload, &event); err != nil || event.Type == "" {
			SendEventToClient(client, "message_error", map[string]string{"message": "Invalid message."})
			continue
		}
		handle(client, event)
	}
}

// Decode accepts exactly one JSON object and rejects unknown fields.
func Decode(payload []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("message must contain one JSON object")
	}
	return nil
}

func SendEventToClient(client *Client, eventType string, data any) {
	message, err := encodeEvent(eventType, data)
	if err != nil {
		log.Println("Failed to encode WebSocket event:", err)
		return
	}
	select {
	case client.send <- message:
	default:
		client.connection.Close()
	}
}
