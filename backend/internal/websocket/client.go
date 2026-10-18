package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

// Client is one browser tab. A user may have several connected clients.
type Client struct {
	UserID       int64
	SessionToken string
	connection   *websocket.Conn
	send         chan []byte
	view         ChatView
}

func writeMessages(client *Client) {
	for message := range client.send {
		_ = client.connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := client.connection.WriteMessage(websocket.TextMessage, message); err != nil {
			client.connection.Close()
			return
		}
	}
}
