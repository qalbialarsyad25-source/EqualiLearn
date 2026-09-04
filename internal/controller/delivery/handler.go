package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type WSHandler struct {
	manager *WSManager
}

func NewWSHandler(manager *WSManager) *WSHandler {
	return &WSHandler{manager: manager}
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
	defer conn.Close()

	h.manager.AddClient(userID, conn)
	defer h.manager.RemoveClient(userID, conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
