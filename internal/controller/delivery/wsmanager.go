package delivery

import (
	"sync"

	"github.com/gorilla/websocket"
)

type WSManager struct {
	clients map[string]map[*websocket.Conn]bool
	mu      sync.RWMutex
}

func NewWSManager() *WSManager {
	return &WSManager{
		clients: make(map[string]map[*websocket.Conn]bool),
	}
}

func (m *WSManager) AddClient(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[userID]; !ok {
		m.clients[userID] = make(map[*websocket.Conn]bool)
	}
	m.clients[userID][conn] = true
}

func (m *WSManager) RemoveClient(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if userConns, ok := m.clients[userID]; ok {
		delete(userConns, conn)
		if len(userConns) == 0 {
			delete(m.clients, userID)
		}
	}
}

func (m *WSManager) SendToUser(userID string, msg interface{}) {
	m.mu.RLock()
	userConns, ok := m.clients[userID]
	if !ok {
		m.mu.RUnlock()
		return
	}

	// Copy pointers under read lock to minimize lock contention
	conns := make([]*websocket.Conn, 0, len(userConns))
	for conn := range userConns {
		conns = append(conns, conn)
	}
	m.mu.RUnlock()

	for _, conn := range conns {
		_ = conn.WriteJSON(msg)
	}
}
