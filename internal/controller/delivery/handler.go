package websocket

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	manager *WSManager
}

func NewWSHandler(manager *WSManager) *WSHandler {
	return &WSHandler{manager}
}

func (h *WSHandler) HandleWS(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	h.manager.AddClient(userID, conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			h.manager.RemoveClient(userID)
			break
		}
	}
}
