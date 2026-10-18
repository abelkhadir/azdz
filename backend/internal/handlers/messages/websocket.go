package messages

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"social-network/internal/models"
	"social-network/internal/pkg/repo"
	"social-network/internal/websocket"
)

const maxMessageLength = 2000

func (h Handler) Connect(w http.ResponseWriter, r *http.Request) {
	websocket.Serve(h.WebSocket, h.DB, w, r, h.handleSocketEvent)
}

func (h Handler) handleSocketEvent(client *websocket.Client, event websocket.IncomingEvent) {
	switch event.Type {
	case "chat_view":
		h.handleChatView(client, event.Data)
	case models.WebSocketPrivateMessage:
		h.handlePrivateMessage(client, event.Data)
	case models.WebSocketGroupMessage:
		h.handleGroupMessage(client, event.Data)
	default:
		sendMessageError(client, "", "Unknown message type.")
	}
}

func (h Handler) handlePrivateMessage(client *websocket.Client, data []byte) {
	var request models.PrivateMessageRequest
	if websocket.Decode(data, &request) != nil {
		sendMessageError(client, "", "Invalid private message.")
		return
	}
	content := strings.TrimSpace(request.Content)
	if request.ReceiverID <= 0 || content == "" || utf8.RuneCountInString(content) > maxMessageLength {
		sendMessageError(client, request.ClientID, "Enter a message of up to 2000 characters.")
		return
	}

	allowed, err := repo.CanUsersChat(h.DB, client.UserID, request.ReceiverID)
	if err != nil {
		log.Println("Failed to check private chat permission:", err)
		sendMessageError(client, request.ClientID, "Could not verify this conversation.")
		return
	}
	if !allowed {
		sendMessageError(client, request.ClientID, "You can only message someone when one of you follows the other.")
		return
	}

	message := models.PrivateMessage{SenderID: client.UserID, ReceiverID: request.ReceiverID, Content: content}
	if err := repo.SavePrivateMessage(h.DB, &message); err != nil {
		if !errors.Is(err, repo.ErrMessageForbidden) {
			log.Println("Failed to save private message:", err)
		}
		sendMessageError(client, request.ClientID, "Your message was not sent.")
		return
	}

	websocket.SendEventToUser(h.WebSocket, request.ReceiverID, models.WebSocketPrivateMessage, message)
	h.notifyChat(request.ReceiverID, "private", client.UserID, client.UserID, message.Content)
	websocket.SendEventToUser(h.WebSocket, client.UserID, models.WebSocketPrivateMessage, message)
	websocket.SendEventToClient(client, models.WebSocketMessageSent, models.MessageStatus{ClientID: request.ClientID})
}

func (h Handler) handleGroupMessage(client *websocket.Client, data []byte) {
	var request models.GroupMessageRequest
	if websocket.Decode(data, &request) != nil {
		sendMessageError(client, "", "Invalid group message.")
		return
	}
	content := strings.TrimSpace(request.Content)
	if request.GroupID <= 0 || content == "" || utf8.RuneCountInString(content) > maxMessageLength {
		sendMessageError(client, request.ClientID, "Enter a message of up to 2000 characters.")
		return
	}

	allowed, err := repo.CanUserAccessGroupChat(h.DB, client.UserID, request.GroupID)
	if err != nil {
		log.Println("Failed to check group chat permission:", err)
		sendMessageError(client, request.ClientID, "Could not verify this group.")
		return
	}
	if !allowed {
		sendMessageError(client, request.ClientID, "Only current group members can send messages here.")
		return
	}

	message := models.GroupMessage{GroupID: request.GroupID, SenderID: client.UserID, Content: content}
	if err := repo.SaveGroupMessage(h.DB, &message); err != nil {
		if !errors.Is(err, repo.ErrMessageForbidden) {
			log.Println("Failed to save group message:", err)
		}
		sendMessageError(client, request.ClientID, "Your message was not sent.")
		return
	}
	members, err := repo.GetGroupMemberIDs(h.DB, request.GroupID)
	if err != nil {
		log.Println("Failed to load group members:", err)
		sendMessageError(client, request.ClientID, "Message saved, but delivery failed.")
		return
	}

	websocket.SendEventToUsers(h.WebSocket, members, models.WebSocketGroupMessage, message)
	for _, memberID := range members {
		if memberID != client.UserID {
			h.notifyChat(memberID, "group", request.GroupID, client.UserID, message.Content)
		}
	}
	websocket.SendEventToClient(client, models.WebSocketMessageSent, models.MessageStatus{ClientID: request.ClientID})
}

func (h Handler) handleChatView(client *websocket.Client, data []byte) {
	var view websocket.ChatView
	if websocket.Decode(data, &view) != nil {
		websocket.SetChatView(h.WebSocket, client, websocket.ChatView{})
		return
	}
	if view.Kind == "" && view.ID == 0 {
		websocket.SetChatView(h.WebSocket, client, view)
		return
	}

	var allowed bool
	var err error
	switch view.Kind {
	case "private":
		allowed, err = repo.CanUsersChat(h.DB, client.UserID, view.ID)
	case "group":
		allowed, err = repo.CanUserAccessGroupChat(h.DB, client.UserID, view.ID)
	}
	if err != nil || !allowed || view.ID <= 0 {
		websocket.SetChatView(h.WebSocket, client, websocket.ChatView{})
		return
	}
	websocket.SetChatView(h.WebSocket, client, view)

	if view.Kind == "group" {
		err = repo.MarkGroupThreadRead(h.DB, client.UserID, view.ID)
	} else {
		err = repo.MarkPrivateThreadRead(h.DB, client.UserID, view.ID)
	}
	if err != nil {
		log.Println("Failed to mark chat notifications read:", err)
		return
	}
	websocket.SendEventToUser(h.WebSocket, client.UserID, "notifications_read", view)
}

func (h Handler) notifyChat(userID int64, kind string, roomID, actorID int64, content string) {
	if websocket.IsViewingChat(h.WebSocket, userID, kind, roomID) {
		return
	}
	notificationType := models.NotificationPrivateMessage
	notification := models.Notification{
		UserID: userID, ActorID: actorID, Type: notificationType,
		EntityType: "profile", Content: content,
	}
	if kind == "group" {
		notification.Type = models.NotificationGroupMessage
		notification.EntityID = roomID
		notification.EntityType = "group"
	}
	if err := repo.CreateNotification(h.DB, &notification); err != nil {
		log.Println("Failed to save chat notification:", err)
		return
	}
	// The user may enter the chat while the notification is being saved.
	if websocket.IsViewingChat(h.WebSocket, userID, kind, roomID) {
		_ = repo.DeleteNotification(h.DB, userID, notification.ID)
		websocket.SendEventToUser(h.WebSocket, userID, "notifications_read", nil)
		return
	}
	websocket.SendNotification(h.WebSocket, notification)
}

func sendMessageError(client *websocket.Client, clientID, message string) {
	websocket.SendEventToClient(client, models.WebSocketMessageError, models.MessageStatus{
		ClientID: clientID, Message: message,
	})
}
