package delivery

import (
	"sync"
	"time"

	"EquiliLearn/internal/model"

	"github.com/gorilla/websocket"
)

// ChatHub manages real-time WebSocket connections and broadcasts messages to group members.
type ChatHub struct {
	userClients        map[string]map[*websocket.Conn]bool
	groupSubscriptions map[string]map[string]bool // groupID -> set of userIDs currently in/viewing this room
	mu                 sync.RWMutex
}

func NewChatHub() *ChatHub {
	return &ChatHub{
		userClients:        make(map[string]map[*websocket.Conn]bool),
		groupSubscriptions: make(map[string]map[string]bool),
	}
}

func (h *ChatHub) AddClient(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.userClients[userID]; !ok {
		h.userClients[userID] = make(map[*websocket.Conn]bool)
	}
	h.userClients[userID][conn] = true
}

func (h *ChatHub) RemoveClient(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.userClients[userID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.userClients, userID)
			// Remove from all group subscriptions
			for _, users := range h.groupSubscriptions {
				delete(users, userID)
			}
		}
	}
}

func (h *ChatHub) SubscribeGroup(userID string, groupID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.groupSubscriptions[groupID]; !ok {
		h.groupSubscriptions[groupID] = make(map[string]bool)
	}
	h.groupSubscriptions[groupID][userID] = true
}

func (h *ChatHub) UnsubscribeGroup(userID string, groupID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if users, ok := h.groupSubscriptions[groupID]; ok {
		delete(users, userID)
		if len(users) == 0 {
			delete(h.groupSubscriptions, groupID)
		}
	}
}

// BroadcastToGroup sends a WebSocket message to all active connections belonging to users in the group.
func (h *ChatHub) BroadcastToGroup(groupID string, msg model.WSOutgoingMessage) {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	h.mu.RLock()
	usersInGroup, ok := h.groupSubscriptions[groupID]
	if !ok {
		h.mu.RUnlock()
		return
	}

	var conns []*websocket.Conn
	for uid := range usersInGroup {
		if userConns, exists := h.userClients[uid]; exists {
			for conn := range userConns {
				conns = append(conns, conn)
			}
		}
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		_ = conn.WriteJSON(msg)
	}
}

// BroadcastToUser sends a message directly to a specific user's open WebSocket connections.
func (h *ChatHub) BroadcastToUser(userID string, msg model.WSOutgoingMessage) {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	h.mu.RLock()
	userConns, ok := h.userClients[userID]
	if !ok {
		h.mu.RUnlock()
		return
	}

	conns := make([]*websocket.Conn, 0, len(userConns))
	for conn := range userConns {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		_ = conn.WriteJSON(msg)
	}
}

func (h *ChatHub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conns, ok := h.userClients[userID]
	return ok && len(conns) > 0
}
