package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"social-network/internal/models"
)

type Manager struct {
	clients map[int64]map[*Client]struct{}
	mu      sync.RWMutex
}

type ChatView struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
}

func NewManager() *Manager {
	return &Manager{clients: make(map[int64]map[*Client]struct{})}
}

func addClient(manager *Manager, client *Client) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.clients[client.UserID] == nil {
		manager.clients[client.UserID] = make(map[*Client]struct{})
	}
	manager.clients[client.UserID][client] = struct{}{}
}

func removeClient(manager *Manager, client *Client) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if _, exists := manager.clients[client.UserID][client]; !exists {
		return
	}
	delete(manager.clients[client.UserID], client)
	if len(manager.clients[client.UserID]) == 0 {
		delete(manager.clients, client.UserID)
	}
	close(client.send)
}

func CloseSession(manager *Manager, token string) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	for _, clients := range manager.clients {
		for client := range clients {
			if client.SessionToken == token {
				client.connection.Close()
			}
		}
	}
}

func SendEventToUser(manager *Manager, userID int64, eventType string, data any) bool {
	message, err := encodeEvent(eventType, data)
	if err != nil {
		log.Println("Failed to encode WebSocket event:", err)
		return false
	}

	manager.mu.RLock()
	defer manager.mu.RUnlock()
	sent := false
	for client := range manager.clients[userID] {
		select {
		case client.send <- message:
			sent = true
		default:
			client.connection.Close()
		}
	}
	return sent
}

func SendEventToUsers(manager *Manager, userIDs []int64, eventType string, data any) {
	for _, userID := range userIDs {
		SendEventToUser(manager, userID, eventType, data)
	}
}

func SendNotification(manager *Manager, notification models.Notification) bool {
	return SendEventToUser(manager, notification.UserID, models.WebSocketNotification, notification)
}

func SetChatView(manager *Manager, client *Client, view ChatView) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	client.view = view
}

func IsViewingChat(manager *Manager, userID int64, kind string, roomID int64) bool {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	for client := range manager.clients[userID] {
		if client.view.Kind == kind && client.view.ID == roomID {
			return true
		}
	}
	return false
}

func encodeEvent(eventType string, data any) ([]byte, error) {
	return json.Marshal(models.WebSocketEvent{Type: eventType, Data: data})
}
