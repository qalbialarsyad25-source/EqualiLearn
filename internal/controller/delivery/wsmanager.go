package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

type WSManager struct {
	clients map[string]*websocket.Conn
	mu      sync.RWMutex
}

func NewWSManager() *WSManager {
	return &WSManager{
		clients: make(map[string]*websocket.Conn),
	}
}

func (m *WSManager) AddClient(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[userID] = conn
}

func (m *WSManager) RemoveClient(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, userID)
}

func (m *WSManager) SendToUser(userID string, msg interface{}) {
	m.mu.RLock()
	conn, ok := m.clients[userID]
	m.mu.RUnlock()

	if ok {
		conn.WriteJSON(msg)
	}
}
